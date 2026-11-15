package service

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"ffresources/apps/ff13/backend/config"
	"ffresources/apps/ff13/backend/formats"
	"ffresources/apps/ff13/backend/parts"
	loggingService "ffresources/internal/shared/logging"
)

// COLEÇÃO de data/partN: o INVENTÁRIO do que foi extraído, cacheado por part.
//
// A coleção tem duas camadas:
//
//   - o INVENTÁRIO (as entradas: unidades de texto e arquivos fora do
//     escopo) — a varredura do disco, feita UMA vez por part;
//   - o CONTEÚDO (a visão default decodificada de cada unidade) — o que a
//     tabela mostra.
//
// A coleção existe para que a troca de aba não pague a varredura: a aba ativa
// pede a coleção (construída na hora se ainda não existir) e as outras duas
// entram em background pelo PreloadVersions.
//
// A árvore /data mostra só as UNIDADES DE TEXTO (Text=true): os arquivos
// fora do escopo ficam no inventário do backend (são o que o filelist
// mostra) e não viram folha.

// dataCollection é a coleção de UMA part.
type dataCollection struct {
	// entries é o inventário (unidades + arquivos fora do escopo).
	entries []formats.DataEntry
	// units indexa as unidades de texto por id.
	units map[string]*formats.TextUnit
	// content guarda a visão default (o slot "us") de cada unidade já
	// decodificada.
	content map[string]*formats.UnitContent
	// mu protege units e content (a coleção é lida de várias goroutines:
	// a aba ativa e o preload das outras parts).
	mu sync.Mutex
}

// dataBuild é uma construção de coleção EM CURSO: quem pedir a coleção
// enquanto ela é montada espera o mesmo resultado (a varredura do disco
// acontece uma vez só).
type dataBuild struct {
	done chan struct{}
	coll *dataCollection
	err  error
}

var (
	dataMu       sync.Mutex
	dataCache    = map[string]*dataCollection{}
	dataBuilders = map[string]*dataBuild{}
)

// collectionKey identifica a coleção: a PART e as raízes de data/mods.
//
// A raiz entra na chave porque o mesmo app pode apontar para árvores
// diferentes (o env FF_RESOURCES_DATA/FF_RESOURCES_MODS muda entre
// execuções) — coleções de raízes diferentes não podem se misturar.
func collectionKey(part string) string {
	data, _ := config.DataRoot("data")
	mods, _ := config.DataRoot("mods")
	return config.NormalizePart(part) + "|" + data + "|" + mods
}

// collectionFor devolve a coleção da part, construindo se preciso.
//
// A construção é única por part: a demanda (ListDataEntries) e o preload
// podem chegar juntos (o usuário troca de aba enquanto o preload roda) e
// quem pedir durante a construção espera o MESMO resultado.
func collectionFor(part string) (*dataCollection, error) {
	part = config.NormalizePart(part)
	key := collectionKey(part)
	dataMu.Lock()
	if c, ok := dataCache[key]; ok {
		dataMu.Unlock()
		return c, nil
	}
	if b, ok := dataBuilders[key]; ok {
		dataMu.Unlock()
		<-b.done
		return b.coll, b.err
	}
	b := &dataBuild{done: make(chan struct{})}
	dataBuilders[key] = b
	dataMu.Unlock()

	coll, err := buildDataCollection(part)

	dataMu.Lock()
	delete(dataBuilders, key)
	if err == nil {
		dataCache[key] = coll
	}
	dataMu.Unlock()

	b.coll, b.err = coll, err
	close(b.done)
	return coll, err
}

// buildDataCollection monta a coleção da part: varre data/partN uma vez e
// separa as unidades de texto dos arquivos fora do escopo.
func buildDataCollection(part string) (*dataCollection, error) {
	p := parts.Get(part)
	dataDir, err := config.PartRoot("data", part)
	if err != nil {
		return nil, err
	}
	units, others, err := formats.ScanDataDir(dataDir, p)
	if err != nil {
		return nil, err
	}
	coll := &dataCollection{
		units:   make(map[string]*formats.TextUnit, len(units)),
		content: map[string]*formats.UnitContent{},
	}
	// mods/partN é o espelho do data/partN: a unidade cujo primário já
	// existe lá está traduzida. A árvore usa isso para preferir a cópia
	// certa quando o mesmo caminho visual aparece no global e na zona.
	modsDir, _ := config.PartRoot("mods", part)
	for _, u := range units {
		coll.units[u.ID] = u
		entry := formats.DataEntry{
			ID:        u.ID,
			Label:     u.Label,
			Type:      u.Type,
			Kind:      u.Kind,
			Text:      true,
			Langs:     u.Langs,
			Primary:   u.Primary,
			Files:     u.Files,
			Container: u.Container,
		}
		if modsDir != "" && u.Primary != "" {
			if _, serr := os.Stat(filepath.Join(modsDir, filepath.FromSlash(u.Primary))); serr == nil {
				entry.Modded = true
			}
		}
		coll.entries = append(coll.entries, entry)
	}
	for _, rel := range others {
		coll.entries = append(coll.entries, formats.DataEntry{
			ID:    rel,
			Label: path.Base(rel),
			Type:  p.EntryType(rel),
			Text:  false,
		})
	}
	// Imagens/containers de imagem do inventário. O conjunto inclui os
	// arquivos das unidades de texto para o pareamento .xgr/.bin + .imgb.
	all := make(map[string]bool, len(others)+64)
	for _, rel := range others {
		all[formats.LowerRel(rel)] = true
	}
	for _, u := range units {
		for _, f := range u.Files {
			all[formats.LowerRel(f)] = true
		}
	}
	coll.entries = append(coll.entries, formats.ImageDataEntries(part, all)...)
	sort.Slice(coll.entries, func(i, j int) bool { return coll.entries[i].ID < coll.entries[j].ID })
	return coll, nil
}

// ListDataEntries devolve o INVENTÁRIO de data/partN: as unidades de texto e
// os arquivos fora do escopo.
//
// É a carga da aba: a coleção é cacheada por part, então a troca de aba não
// paga a varredura de novo.
func (FF13Service) ListDataEntries(part string) ([]formats.DataEntry, error) {
	coll, err := collectionFor(config.NormalizePart(part))
	if err != nil {
		return nil, err
	}
	return coll.entries, nil
}

// PreloadVersions monta as coleções das parts informadas em BACKGROUND: é o
// que faz a troca de aba ser instantânea (a aba ativa pede a própria
// coleção; as outras duas ficam prontas).
//
// Idempotente: part já cacheada é pulada.
func (FF13Service) PreloadVersions(partIDs []string) {
	for _, id := range partIDs {
		part := config.NormalizePart(id)
		key := collectionKey(part)
		dataMu.Lock()
		_, cached := dataCache[key]
		_, building := dataBuilders[key]
		dataMu.Unlock()
		if cached || building {
			continue
		}
		go func(part string) {
			if _, err := collectionFor(part); err != nil {
				loggingService.Warn("preload da coleção %s falhou: %v", part, err)
			}
		}(part)
	}
}

// InvalidateDataCollection limpa o cache da part: a próxima leitura refaz a
// varredura (e o conteúdo decodificado). É o que acontece quando o mods/
// muda (um save) ou quando o app recarrega a aba a frio.
func (FF13Service) InvalidateDataCollection(part string) {
	invalidateCollection(config.NormalizePart(part))
}

// invalidateCollection tira a coleção da part do cache.
func invalidateCollection(part string) {
	prefix := config.NormalizePart(part) + "|"
	dataMu.Lock()
	for k := range dataCache {
		if strings.HasPrefix(k, prefix) {
			delete(dataCache, k)
		}
	}
	dataMu.Unlock()
}

// collectionUnit devolve a unidade de texto da coleção (a do data/partN) e,
// se ela não estiver lá, resolve pelo white bin (o arquivo pode não ter sido
// extraído) — a unidade resolvida entra no cache da coleção.
func collectionUnit(coll *dataCollection, part, id string) (*formats.TextUnit, error) {
	coll.mu.Lock()
	unit, ok := coll.units[id]
	coll.mu.Unlock()
	if ok {
		return unit, nil
	}
	dataDir, err := config.PartRoot("data", part)
	if err != nil {
		return nil, err
	}
	unit, err = formats.ResolveUnit(dataDir, parts.Get(part), id)
	if err != nil {
		return nil, err
	}
	coll.mu.Lock()
	coll.units[id] = unit
	coll.mu.Unlock()
	return unit, nil
}

// unitContentOf devolve a visão default (o slot "us") da unidade, com cache
// na coleção: o pristine, o texto atual (mods-first) e as referências dos
// demais idiomas.
func unitContentOf(coll *dataCollection, part string, unit *formats.TextUnit) (*formats.UnitContent, error) {
	coll.mu.Lock()
	content, ok := coll.content[unit.ID]
	coll.mu.Unlock()
	if ok {
		return content, nil
	}
	content, err := formats.ReadUnitContent(part, unit)
	if err != nil {
		return nil, err
	}
	coll.mu.Lock()
	coll.content[unit.ID] = content
	coll.mu.Unlock()
	return content, nil
}

// normalizeLang devolve o código de idioma normalizado ("us", "jp"...); ""
// quando o código não é do catálogo.
func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return ""
	}
	if !formats.IsUnitLang(lang) {
		return ""
	}
	return lang
}
