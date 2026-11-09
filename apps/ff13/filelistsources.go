package main

import (
	"sort"
	"sync"

	"ffresources/apps/ff13/common"
	"ffresources/apps/ff13/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
)

// FONTES dos bytes de uma unidade de texto.
//
// O app guarda em data/partN só o que foi extraído (os .ztr e os arquivos
// traduzidos). O FILELIST, porém, tem TODOS os arquivos do jogo: cada um
// deles mora num WHITE BIN, endereçado pelo índice que o descreve. Quando o
// arquivo não está em data/partN, os bytes saem do white bin — lidos sob
// demanda, sem gravar nada em disco.

// unitSource diz de onde vêm os bytes de um arquivo de uma unidade.
type unitSource struct {
	// root: quando não é vazio, o arquivo está em <root>/<rel> (data/partN).
	root string
	// rel: caminho do arquivo relativo a data/partN — é o mesmo caminho do
	// mod em mods/partN.
	rel string
	// bin: quando não é vazio, o arquivo está no white bin (coll + vpath).
	bin   string
	coll  *fl.Collection
	vpath string
}

// isDisk diz se a fonte é um arquivo em disco (data/partN).
func (s unitSource) isDisk() bool { return s.root != "" }

// sourceIndex é o mapa unidade -> idioma -> fonte, montado a partir dos
// índices da part.
type sourceIndex struct {
	units map[string]map[string]unitSource
}

var (
	sourcesMu    sync.Mutex
	sourcesCache = map[string]*sourceIndex{}
)

// partSources devolve (com cache por part) o índice das fontes do white bin.
// Sem o jogo instalado o índice é VAZIO: o app segue funcionando só com
// data/partN.
func partSources(p parts.Part) *sourceIndex {
	key := p.ID() + "|" + gameRoot(p.ID())
	sourcesMu.Lock()
	defer sourcesMu.Unlock()
	if s, ok := sourcesCache[key]; ok {
		return s
	}
	s := buildSourceIndex(p)
	sourcesCache[key] = s
	return s
}

// buildSourceIndex varre os índices da part e mapeia cada arquivo indexado
// para a sua fonte (o white bin que o declara).
//
// A ordem dos índices decide quem ganha quando dois declaram o mesmo
// caminho: os de ZONA vêm antes dos globais (a mesma regra da união de zona
// em zoneNodes) e, dentro da zona, o índice "u" (inglês) vem antes do "c".
func buildSourceIndex(p parts.Part) *sourceIndex {
	idx := &sourceIndex{units: map[string]map[string]unitSource{}}
	if gameRoot(p.ID()) == "" {
		return idx
	}
	disc := discoverPart(p)
	indexes := append([]*filelistIndex(nil), disc.indexes...)
	sort.SliceStable(indexes, func(i, j int) bool {
		zi, zj := zoneOfIndex(indexes[i]) != "", zoneOfIndex(indexes[j]) != ""
		if zi != zj {
			return zi
		}
		return indexLanguageRank(p, indexes[i]) < indexLanguageRank(p, indexes[j])
	})

	for _, ix := range indexes {
		if ix.Container == "" {
			continue
		}
		zone := zoneOfIndex(ix)
		for _, e := range indexEntries(p, ix, nil) {
			dataPath := filelistDataPath(zone, e.Path)
			unitID, lang, ok := filelistUnitID(dataPath)
			if !ok {
				continue
			}
			langs := idx.units[unitID]
			if langs == nil {
				langs = map[string]unitSource{}
				idx.units[unitID] = langs
			}
			if _, exists := langs[lang]; exists {
				continue
			}
			langs[lang] = unitSource{
				rel:   dataPath,
				bin:   ix.Container,
				coll:  ix.Coll,
				vpath: e.Path,
			}
		}
	}
	return idx
}

// unitsOf devolve as fontes de uma unidade (idioma -> fonte).
func (s *sourceIndex) unitsOf(id string) map[string]unitSource {
	if s == nil {
		return nil
	}
	return s.units[id]
}

// zoneOfIndex devolve a zona de um índice (pelo nome do arquivo: "z000"),
// ou "" quando o índice é global.
func zoneOfIndex(idx *filelistIndex) string {
	if idx == nil {
		return ""
	}
	return common.ZoneIDOf(indexNameOf(idx))
}

// indexLanguageRank ordena os índices pela região: "u" (inglês) primeiro,
// depois "c" (japonês), depois o resto.
//
// p é a part: a convenção do par muda (part1/2 = "u"/"c", part3 = "a"/"v").
func indexLanguageRank(p parts.Part, idx *filelistIndex) int {
	switch p.LanguageOf(indexNameOf(idx)) {
	case common.LangU:
		return 0
	case common.LangC:
		return 1
	default:
		return 2
	}
}
