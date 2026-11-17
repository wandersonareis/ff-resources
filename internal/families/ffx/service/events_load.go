package services

import (
	"errors"
	"fmt"
	"sync"

	"ffresources/internal/families/ffx/codecs/event"
	"ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/shared/logging"
)

// Estado de carga dos events por ÁRVORE (bucket), não por versão: ffx e
// eternalcalm dividem a árvore do FFX; ffx2 e lastmiss, a do FFX-2. É o mesmo
// bucket do datastore (ver profile/datastore/registry.go), então duas versões
// nunca carregam o mesmo store em paralelo.
//
// Por que existe:
//   - a descoberta (só ReadDir) é RÁPIDA e roda uma vez por árvore; o
//     universo de ids é publicado nela, então a lista já mostra TUDO com
//     status "loading" enquanto o conteúdo é lido;
//   - a leitura roda UMA vez em background (single-flight). Antes, cada
//     chamador que via HasEvents()==false disparava a própria carga: o store
//     ficava parcial e as leituras derivadas (view dedupada, índice de busca)
//     congelavam esse parcial - "aparece na lista mas não abre";
//   - quem precisa do conteúdo (GetEntry/GetCollection/export) ESPERA a carga
//     terminar, em vez de receber um store parcial.
var (
	eventsStateMu sync.Mutex
	eventsStates  = map[string]*eventsLoadState{}
)

// errEventsTreeChanged aborta uma carga cuja árvore mudou no meio (troca de
// gamefiles / toggle de mods). A próxima chamada redescobre.
var errEventsTreeChanged = fmt.Errorf("árvore de gamefiles mudou durante a carga de events")

// eventsLoadRun é UMA execução da leitura. O resultado vive nela (não no
// estado compartilhado), então um waiter nunca lê o erro de uma execução
// posterior.
type eventsLoadRun struct {
	finished chan struct{}
	err      error
}

type eventsLoadState struct {
	// mu protege os campos abaixo; NÃO é segurado durante I/O.
	mu          sync.Mutex
	done        bool
	startedOnce bool     // esta árvore já disparou uma carga (não bypassa)
	known       []string // universo descoberto (ordenado; vazio = árvore sem events)
	gen         int64    // treeGeneration da descoberta
	storeGen    uint64   // geração do store de events na descoberta
	root        string   // GameFilesRoot da descoberta (testes trocam o root)
	file        *event.EventsBinaryFile
	run         *eventsLoadRun // execução em voo (nil = nenhuma)
}

// eventsTreeKey é o bucket normalizado da árvore ("ffx" | "ffx2").
func eventsTreeKey(version common.GameVersion) string {
	return common.VersionPathName(version)
}

func eventsStateFor(version common.GameVersion) *eventsLoadState {
	key := eventsTreeKey(version)
	eventsStateMu.Lock()
	defer eventsStateMu.Unlock()
	st := eventsStates[key]
	if st == nil {
		st = &eventsLoadState{}
		eventsStates[key] = st
	}
	return st
}

// resetEventsLoadStates descarta toda a descoberta/carga (troca de árvore).
// Chamado por InvalidateViewCaches, que já segura a escrita do viewLoadMu.
func resetEventsLoadStates() {
	eventsStateMu.Lock()
	eventsStates = map[string]*eventsLoadState{}
	eventsStateMu.Unlock()
}

// prepareEvents garante a DESCOBERTA da árvore (rápida) e dispara a leitura
// em background, sem esperar por ela. Devolve o estado e o universo de ids.
func (s *MetadataService) prepareEvents(version common.GameVersion) (*eventsLoadState, []string, error) {
	viewLoadMu.RLock()
	defer viewLoadMu.RUnlock()

	st := eventsStateFor(version)
	gen := treeGeneration.Load()
	storeGen := event.StoreGeneration()
	root := common.GameFilesRoot

	st.mu.Lock()
	// Árvore, store ou gamefiles mudaram: zera o estado. Uma carga em voo da
	// árvore anterior é abortada pelo próprio loader (compara a geração).
	if st.gen != gen || st.storeGen != storeGen || st.root != root {
		st.done = false
		st.storeGen = storeGen
		if st.gen != gen || st.root != root {
			st.known = nil
			st.file = nil
			st.gen = gen
			st.root = root
			st.startedOnce = false
		}
	}
	if st.known != nil {
		ids := st.known
		st.mu.Unlock()
		st.kick(s, version, gen)
		return st, ids, nil
	}
	st.mu.Unlock()

	// Descoberta: só ReadDir - não lê um único evento.
	b := event.NewEventsBinaryFile(version)
	b.OnLoaded = func(id string, done, total int) {
		s.reportEventsProgress(version, done, total)
	}
	if err := b.Discover(); err != nil {
		if errors.Is(err, event.ErrEventsTreeMissing) {
			// Sem event/obj_ps3 nesta árvore: universo vazio. O store (se
			// houver conteúdo publicado) continua servindo - ver ListEntries.
			st.mu.Lock()
			st.known = []string{}
			st.file = nil
			st.gen = gen
			st.storeGen = storeGen
			st.mu.Unlock()
			return st, []string{}, nil
		}
		return nil, nil, err
	}
	ids := append([]string(nil), b.Discovered...)

	st.mu.Lock()
	if st.gen != gen {
		// Outra goroutine redescobriu para uma árvore mais nova no meio.
		st.mu.Unlock()
		return st, ids, nil
	}
	if st.known == nil {
		st.known = ids
		st.file = b
		st.storeGen = storeGen
	}
	ids = st.known
	st.mu.Unlock()

	st.kick(s, version, gen)
	return st, ids, nil
}

// kick inicia a leitura em background uma única vez por geração.
func (st *eventsLoadState) kick(s *MetadataService, version common.GameVersion, gen int64) {
	st.mu.Lock()
	if st.done || st.run != nil || st.file == nil || st.gen != gen {
		st.mu.Unlock()
		return
	}
	// Store já povoado SEM esta árvore ter disparado carga (conteúdo
	// sintético de teste ou evento aplicado antes da 1ª listagem): não
	// recarrega por cima - a releitura do disco descartaria edições em
	// memória. Depois que ESTA árvore começou a carregar, o store só é
	// completo ao fim da carga; aí a regra não se aplica.
	if !st.startedOnce && len(event.GetAllEventIDs(version)) > 0 {
		st.done = true
		st.mu.Unlock()
		return
	}
	st.startedOnce = true
	run := &eventsLoadRun{finished: make(chan struct{})}
	st.run = run
	file := st.file
	st.mu.Unlock()

	go s.runEventsLoad(version, st, run, file, gen)
}

// runEventsLoad lê e publica os eventos sob o RLock (a carga não pode
// atravessar um ClearAllEvents). Se a árvore mudou antes do lock, aborta sem
// publicar nada: a próxima chamada redescobre.
func (s *MetadataService) runEventsLoad(version common.GameVersion, st *eventsLoadState, run *eventsLoadRun, b *event.EventsBinaryFile, gen int64) {
	viewLoadMu.RLock()
	defer viewLoadMu.RUnlock()

	if treeGeneration.Load() != gen {
		finishEventsRun(st, run, errEventsTreeChanged)
		return
	}

	if err := b.LoadDiscovered(); err != nil {
		finishEventsRun(st, run, err)
		return
	}

	s.reportEventSkips(b.Skipped)
	// Cache limpo ANTES de liberar os waiters: quem acordar já constrói a view
	// com o store completo (nada de reaproveitar a fração anterior).
	clearDedupViewCache()
	clearTextSearchCache()
	finishEventsRun(st, run, nil)
	EmitEventsLoaded(version, len(b.Discovered))
}

func finishEventsRun(st *eventsLoadState, run *eventsLoadRun, err error) {
	run.err = err
	st.mu.Lock()
	st.run = nil
	if err == nil {
		st.done = true
	}
	st.mu.Unlock()
	close(run.finished)
}

// ensureEventsLoaded espera a carga COMPLETA da árvore. É o portão dos
// caminhos que precisam de conteúdo (GetEntry/GetCollection/export/preload).
func (s *MetadataService) ensureEventsLoaded(version common.GameVersion) error {
	for attempt := 0; attempt < 2; attempt++ {
		st, _, err := s.prepareEvents(version)
		if err != nil {
			return fmt.Errorf("failed to load events for version %s: %w", version, err)
		}
		st.mu.Lock()
		run := st.run
		done := st.done
		st.mu.Unlock()
		if done {
			return nil
		}
		if run == nil {
			// Nada a carregar (árvore sem eventos / descoberta vazia).
			return nil
		}
		<-run.finished
		if run.err == errEventsTreeChanged {
			continue // redescobre na próxima volta
		}
		if run.err != nil {
			loggingService.Warn("carga de events %s falhou: %v", version, run.err)
			return fmt.Errorf("failed to load all events for version %s: %w", version, run.err)
		}
		// A troca de árvore pode ter limpado o store entre o fim da carga e
		// este retorno (o estado é substituído): confere no estado CORRENTE
		// antes de servir como completo.
		current := eventsStateFor(version)
		current.mu.Lock()
		stillDone := current.done
		current.mu.Unlock()
		if !stillDone {
			continue
		}
		return nil
	}
	return fmt.Errorf("events de %s: árvore de gamefiles mudou durante a carga", version)
}

// ensureAllEventsLoaded mantém o contrato do export bulk (ids vazios = tudo):
// espera a carga completa. Diferente do antigo, não recarrega o que já está
// completo - frescura é responsabilidade de EnsureAllEventsFresh.
func (s *MetadataService) ensureAllEventsLoaded(version common.GameVersion) error {
	return s.ensureEventsLoaded(version)
}

// eventsPendingStatus devolve, para cada id conhecido, se o conteúdo ainda
// não está no store. A árvore usa "loading" em vez de sumir com o arquivo.
func eventsPendingStatus(version common.GameVersion, ids []string) map[string]bool {
	loaded := make(map[string]bool, len(ids))
	for _, id := range event.GetAllEventIDs(version) {
		loaded[id] = true
	}
	pending := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !loaded[id] {
			pending[id] = true
		}
	}
	return pending
}

// EventsLoadProgress é o payload dos eventos Wails de progresso da carga.
// Version é a versão que iniciou a carga; Tree é o bucket compartilhado
// ("ffx" | "ffx2") - a aba de eternalcalm/lastmiss escuta o mesmo bucket.
type EventsLoadProgress struct {
	Version string `json:"version"`
	Tree    string `json:"tree"`
	Loaded  int    `json:"loaded"`
	Total   int    `json:"total"`
}

func (s *MetadataService) reportEventsProgress(version common.GameVersion, loaded, total int) {
	if total <= 0 {
		return
	}
	// Throttle: um aviso por arquivo é barulhento demais (530 eventos).
	if loaded != total && loaded%10 != 0 {
		return
	}
	EmitEvent("EventsLoadProgress", EventsLoadProgress{
		Version: version.String(),
		Tree:    eventsTreeKey(version),
		Loaded:  loaded,
		Total:   total,
	})
}

// EmitEventsLoaded avisa o frontend que o conteúdo da árvore está completo:
// a lista é atualizada (some o "loading" das folhas) e o índice de busca,
// descartado, é reconstruído na próxima consulta.
func EmitEventsLoaded(version common.GameVersion, total int) {
	EmitEvent("EventsLoaded", EventsLoadProgress{
		Version: version.String(),
		Tree:    eventsTreeKey(version),
		Loaded:  total,
		Total:   total,
	})
}
