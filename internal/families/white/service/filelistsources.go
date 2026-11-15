package service

import (
	"sort"
	"sync"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/profile/common"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
)

// FONTES dos bytes de uma unidade de texto.
//
// O app guarda em data/partN só o que foi extraído (os .ztr e os arquivos
// traduzidos). O FILELIST, porém, tem TODOS os arquivos do jogo: cada um
// deles mora num WHITE BIN, endereçado pelo índice que o descreve. Quando o
// arquivo não está em data/partN, os bytes saem do white bin — lidos sob
// demanda, sem gravar nada em disco.

// UnitSource diz de onde vêm os bytes de um arquivo de uma unidade.
type UnitSource struct {
	// Root: quando não é vazio, o arquivo está em <root>/<rel> (data/partN).
	Root string
	// Rel: caminho do arquivo relativo a data/partN — é o mesmo caminho do
	// mod em mods/partN.
	Rel string
	// Bin: quando não é vazio, o arquivo está no white bin (coll + vpath).
	Bin   string
	Coll  *fl.Collection
	VPath string
}

// IsDisk diz se a fonte é um arquivo em disco (data/partN).
func (s UnitSource) IsDisk() bool { return s.Root != "" }

// SourceIndex é o mapa unidade -> idioma -> fonte, montado a partir dos
// índices da part.
type SourceIndex struct {
	Units map[string]map[string]UnitSource
}

var (
	sourcesMu    sync.Mutex
	sourcesCache = map[string]*SourceIndex{}
)

// PartSources devolve (com cache por part) o índice das fontes do white bin.
// Sem o jogo instalado o índice é VAZIO: o app segue funcionando só com
// data/partN.
func PartSources(p parts.Part) *SourceIndex {
	key := p.ID() + "|" + config.GameRoot(p.ID())
	sourcesMu.Lock()
	defer sourcesMu.Unlock()
	if s, ok := sourcesCache[key]; ok {
		return s
	}
	s := BuildSourceIndex(p)
	sourcesCache[key] = s
	return s
}

// BuildSourceIndex varre os índices da part e mapeia cada arquivo indexado
// para a sua fonte (o white bin que o declara).
//
// A ordem dos índices decide quem ganha quando dois declaram o mesmo
// caminho: os de ZONA vêm antes dos globais (a mesma regra da união de zona
// em ZoneNodes) e, dentro da zona, o índice "u" (inglês) vem antes do "c".
func BuildSourceIndex(p parts.Part) *SourceIndex {
	idx := &SourceIndex{Units: map[string]map[string]UnitSource{}}
	if config.GameRoot(p.ID()) == "" {
		return idx
	}
	disc := DiscoverPart(p)
	indexes := append([]*FilelistIndex(nil), disc.Indexes...)
	sort.SliceStable(indexes, func(i, j int) bool {
		zi, zj := ZoneOfIndex(indexes[i]) != "", ZoneOfIndex(indexes[j]) != ""
		if zi != zj {
			return zi
		}
		return IndexLanguageRank(p, indexes[i]) < IndexLanguageRank(p, indexes[j])
	})

	for _, ix := range indexes {
		if ix.Container == "" {
			continue
		}
		zone := ZoneOfIndex(ix)
		for _, e := range IndexEntries(p, ix, nil) {
			dataPath := FilelistDataPath(zone, e.Path)
			unitID, lang, ok := FilelistUnitID(dataPath)
			if !ok {
				continue
			}
			langs := idx.Units[unitID]
			if langs == nil {
				langs = map[string]UnitSource{}
				idx.Units[unitID] = langs
			}
			if _, exists := langs[lang]; exists {
				continue
			}
			langs[lang] = UnitSource{
				Rel:   dataPath,
				Bin:   ix.Container,
				Coll:  ix.Coll,
				VPath: e.Path,
			}
		}
	}
	return idx
}

// UnitsOf devolve as fontes de uma unidade (idioma -> fonte).
func (s *SourceIndex) UnitsOf(id string) map[string]UnitSource {
	if s == nil {
		return nil
	}
	return s.Units[id]
}

// ZoneOfIndex devolve a zona de um índice (pelo nome do arquivo: "z000"),
// ou "" quando o índice é global.
func ZoneOfIndex(idx *FilelistIndex) string {
	if idx == nil {
		return ""
	}
	return common.ZoneIDOf(IndexNameOf(idx))
}

// IndexLanguageRank ordena os índices pela região: "u" (inglês) primeiro,
// depois "c" (japonês), depois o resto.
//
// p é a part: a convenção do par muda (part1/2 = "u"/"c", part3 = "a"/"v").
func IndexLanguageRank(p parts.Part, idx *FilelistIndex) int {
	switch p.LanguageOf(IndexNameOf(idx)) {
	case common.LangU:
		return 0
	case common.LangC:
		return 1
	default:
		return 2
	}
}
