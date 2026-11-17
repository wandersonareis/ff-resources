package event

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"ffresources/internal/families/ffx/codecs/components"
	ffxencoding "ffresources/internal/families/ffx/codecs/encoding"
	"ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/families/ffx/profile/datastore"
	"ffresources/internal/families/ffx/profile/interactions"
	"ffresources/internal/families/ffx/profile/models"
)

// EventsBinaryFile carrega/salva os eventos de localization no binário.
// Este pacote desconhece JSON/DTO: exportação de texto vive em
// backend/builders (monta dto.Collection) + backend/formatters/json
// (serializa e escreve o arquivo). Em SaveToBinary o parâmetro string
// é usado como: "" = tudo, ou EventID.
type EventsBinaryFile struct {
	Version common.GameVersion
	Infos   components.IList[*models.EventFileInfo]
	Objects components.IList[datastore.IGlobalLocalizedTextObject]
	// Skipped carrega os eventos pulados na carga (falha de leitura ou sem
	// conteúdo): o loader não conhece progresso/toast — quem chama decide
	// o que fazer com a lista (o services reporta cada um).
	Skipped []EventSkip

	// Discovered é o universo descoberto (ids ordenados), publicado no fim
	// da descoberta - antes de qualquer leitura. É o que permite listar a
	// árvore inteira enquanto o conteúdo ainda está sendo lido.
	Discovered []string
	// OnDiscovered é chamado ao fim da descoberta (universo conhecido). A
	// listagem usa isso para servir a árvore completa com status "loading".
	OnDiscovered func(ids []string)
	// OnLoaded é chamado a cada evento publicado no store (progresso por
	// arquivo). done/total permitem a barra "carregando X de Y".
	OnLoaded func(id string, done, total int)
}

// EventSkip é um evento pulado na carga: quem reporta (Issue/toast) usa o
// motivo como mensagem.
type EventSkip struct {
	ID     string
	Reason string
}

func NewEventsBinaryFile(version common.GameVersion) *EventsBinaryFile {
	return &EventsBinaryFile{
		Version: version,
		Objects: components.NewEmptyList[datastore.IGlobalLocalizedTextObject](),
	}
}

func currentVersionOrDefault() common.GameVersion {
	if svc := interactions.NewInteractionService(); svc != nil {
		return svc.FFXAppConfig().GetGameVersion()
	}
	return common.GameVersionFFX
}

// eventsFolder resolve o diretório de eventos para uma localização.
// Descoberta na árvore ORIGINAL: com mods habilitado, a pasta
// mods/<loc>/event/obj_ps3 existe após o primeiro import com apenas alguns
// eventos — enumerar nela carregaria só o que está em mods. O fluxo correto:
// enumerar os eventos em gamefiles e, por arquivo, preferir o binário de mods
// (ReadLocalizedStringFiles via NewFileAccessor). Fallback: sem árvore
// original, usa o caminho resolvido normal (mods).
func (b *EventsBinaryFile) eventsFolder(localization string) (common.FileAccessor, error) {
	version := b.Version
	if version.String() == "" {
		version = common.GameVersionFFX
	}
	folder, err := common.NewRealFileAccessor(filepath.Join(
		common.GetPathRootForVersion(version),
		"new_"+localization+"pc",
		"event",
		"obj_ps3",
	))
	if err != nil {
		return common.FileAccessor{}, err
	}
	if folder.Exists {
		return folder, nil
	}
	return common.NewFileAccessor(filepath.Join(
		common.GetLocalizationRootForVersion(version, localization),
		"event",
		"obj_ps3",
	))
}

// ErrEventsTreeMissing indica que a árvore de gamefiles não tem
// event/obj_ps3 (gamefiles não configurado ou versão sem events). Não é
// falha de carga: o store (se houver conteúdo já publicado) continua válido.
var ErrEventsTreeMissing = fmt.Errorf("events directory missing")

// Discover resolve a pasta de eventos da árvore original e enumera o universo
// de ids. É rápido (só ReadDir) e idempotente: publica b.Infos e b.Discovered.
func (b *EventsBinaryFile) Discover() error {
	if b.Version.String() == "" {
		b.Version = currentVersionOrDefault()
	}
	eventsFolder, err := b.eventsFolder(common.DefaultLocalization)
	if err != nil {
		return fmt.Errorf("failed to resolve events directory: %w", err)
	}
	if !eventsFolder.Exists {
		b.Infos = components.NewList[*models.EventFileInfo](0)
		b.Discovered = nil
		if b.OnDiscovered != nil {
			b.OnDiscovered(nil)
		}
		return ErrEventsTreeMissing
	}
	if err := ffxencoding.EnsureAllCharsetsLoaded(b.Version); err != nil {
		return fmt.Errorf("charset maps not loaded: %w", err)
	}

	// Uma varredura só: as pastas são globais (não por localização).
	// A versão anterior varria a MESMA pasta uma vez por idioma (~10×).
	discovered, err := discoverEventFiles(eventsFolder, b.Version, false)
	if err != nil {
		return fmt.Errorf("failed to discover event files: %w", err)
	}

	infos := components.NewList[*models.EventFileInfo](len(discovered))
	ids := make([]string, 0, len(discovered))
	for _, info := range discovered {
		infos.Add(info)
		ids = append(ids, info.EventID)
	}
	b.Infos = infos
	b.Discovered = ids
	if b.OnDiscovered != nil {
		b.OnDiscovered(ids)
	}
	return nil
}

// LoadDiscovered lê os eventos descobertos e os publica no store. A leitura
// roda em paralelo (I/O-bound, um worker por CPU até 8) e a publicação é
// feita em ORDEM canônica: o store só recebe evento completo e a ordem dos
// avisos/carimbos não depende do escalonamento.
func (b *EventsBinaryFile) LoadDiscovered() error {
	if b.Infos == nil {
		if err := b.Discover(); err != nil && !errors.Is(err, ErrEventsTreeMissing) {
			return err
		}
	}
	infos := b.Infos.Items()
	if len(infos) == 0 {
		common.LogInfo("events loaded: 0")
		return nil
	}

	files := make([]*EventFile, len(infos))
	errs := make([]error, len(infos))
	parallelReadEvents(infos, files, errs)

	// Sem barra de progresso modal: a carga é implícita (boot/carga de
	// árvore) - o progresso vai por OnLoaded para a árvore. Falhas por
	// evento são PULADAS e ficam em b.Skipped; quem chama (services)
	// reporta cada uma e a carga segue com o que deu certo.
	for i, info := range infos {
		err := errs[i]
		if err != nil {
			b.Skipped = append(b.Skipped, EventSkip{
				ID:     info.EventID,
				Reason: fmt.Sprintf("evento pulado (falha de leitura): %s: %v", info.EventID, err),
			})
			common.LogWarning("failed to read event file %s: %v", info.EventID, err)
			continue
		}
		eventFile := files[i]
		if eventFile == nil {
			b.Skipped = append(b.Skipped, EventSkip{
				ID:     info.EventID,
				Reason: fmt.Sprintf("evento pulado (sem conteúdo): %s", info.EventID),
			})
			continue
		}
		SetEvent(info.Version, info.EventID, eventFile)
		if b.Objects == nil {
			b.Objects = components.NewEmptyList[datastore.IGlobalLocalizedTextObject]()
		}
		b.Objects.Add(NewEventKeyedStringFile(info, eventFile.Strings))
		if b.OnLoaded != nil {
			b.OnLoaded(info.EventID, i+1, len(infos))
		}
	}

	// Vigia de frescura: carimbo físico de cada evento carregado — binário
	// modificado no disco (tradução copiada em mods/) recarrega o evento
	// na próxima leitura (EnsureEventFresh).
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		ids = append(ids, info.EventID)
	}
	stampAllEvents(b.Version, ids)
	common.LogInfo("events loaded: %d", len(infos))
	return nil
}

// parallelReadEvents lê os eventos em paralelo, mantendo o resultado no
// índice de origem (ordem determinística para a publicação).
func parallelReadEvents(infos []*models.EventFileInfo, files []*EventFile, errs []error) {
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers > len(infos) {
		workers = len(infos)
	}
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				files[i], errs[i] = readEventFile(infos[i])
			}
		}()
	}
	for i := range infos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}

// LoadFromBinary é o caminho síncrono completo (descoberta + leitura), usado
// por appcore/CLI. A árvore usa Discover/LoadDiscovered para servir a lista
// antes do conteúdo.
func (b *EventsBinaryFile) LoadFromBinary() error {
	if err := b.Discover(); err != nil {
		return err
	}
	return b.LoadDiscovered()
}

func (b *EventsBinaryFile) eventIDs(filePath string) ([]string, error) {
	if strings.TrimSpace(filePath) == "" {
		return GetAllEventIDs(b.Version), nil
	}
	eventID := eventIDFromParam(filePath)
	if eventID == "" {
		return nil, fmt.Errorf("invalid event selector: %s", filePath)
	}
	if GetEvent(b.Version, eventID) == nil {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}
	return []string{eventID}, nil
}

func eventIDFromParam(filePath string) string {
	base := filepath.Base(strings.TrimSpace(filePath))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.Split(base, "_")
	if len(parts) > 1 {
		last := parts[len(parts)-1]
		if len(last) >= 2 {
			return last
		}
	}
	if len(base) >= 2 {
		return base
	}
	return ""
}

func (b *EventsBinaryFile) SaveToBinary(filePath string) error {
	eventIDs, err := b.eventIDs(filePath)
	if err != nil {
		return err
	}
	for _, eventID := range eventIDs {
		if err := ExportEventStringsToLocalizations(b.Version, eventID); err != nil {
			return err
		}
	}
	return nil
}

func (b *EventsBinaryFile) GetObjects() components.IList[datastore.IGlobalLocalizedTextObject] {
	if b.Objects == nil {
		b.Objects = components.NewEmptyList[datastore.IGlobalLocalizedTextObject]()
	}
	return b.Objects
}
