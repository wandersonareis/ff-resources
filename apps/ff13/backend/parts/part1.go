package parts

import (
	"strings"

	"ffresources/apps/ff13/backend/common"
	fl "ffresources/internal/families/white/codecs/filelist"
)

// part1 = FINAL FANTASY XIII (Steam). Layout V1, sem criptografia.
//
// Particularidades:
//   - dados em `white_data`;
//   - filelists de disco em `white_data/sys`;
//   - o indice principal (filelistu/filelistc) pareia com white_imgu/white_imgc;
//   - os filelists de zona (`filelist_z000u`) NAO existem soltos no disco:
//     ficam dentro de white_imgu e indexam o bin externo white_z000u_img.
type part1 struct{}

func (part1) ID() string            { return "part1" }
func (part1) GameCode() fl.GameCode { return fl.GameFF131 }
func (part1) DataDir(root string) string {
	return dataDir(root, "white_data")
}
func (part1) DataDirName() string { return common.DataDirName("part1") }

// ZoneAsTopLevel: no part1 os indices de ZONA viram o no "Zone" no topo da
// arvore (o conteudo da zona vive so ali).
func (part1) ZoneAsTopLevel() bool   { return true }
func (part1) FilelistDirs() []string { return []string{"sys"} }

func (part1) PairKey(fileName string) (string, bool) {
	n := stemName(fileName)
	switch {
	case strings.HasPrefix(n, "filelist"):
		n = cutPrefix(n, "filelist")
	case strings.HasPrefix(n, "white"):
		n = cutPrefix(n, "white")
		// white_z000u_img -> z000u (o filelist interno e filelist_z000u).
		n = strings.TrimSuffix(n, "_img2")
		n = strings.TrimSuffix(n, "_img")
		// white_imgu/white_imgc -> u/c (o filelist irmao e filelistu/filelistc).
		if strings.HasPrefix(n, "img") {
			n = strings.TrimPrefix(n, "img")
		}
	default:
		return "", false
	}
	if n == "" {
		return "", false
	}
	return n, true
}

func (part1) EntryType(rel string) string { return entryTypeByPath(rel) }

// TextKind: o part1 tem ac/resident/zone/event. O `ac` so existe DENTRO das
// zonas ("zone/z004/txtres/ac/...") — mas a regra do `txtres` pega os dois
// casos (global e por zona).
func (part1) TextKind(rel string) string { return textKindByPath(rel) }

// LanguageOf: no part1 o idioma do indice vem do sufixo do nome ("u" = ingles,
// "c" = japones).
func (part1) LanguageOf(fileName string) string { return common.LanguageOf(fileName) }

// KeepZoneEntry reproduz a regra do Pulse (Albeoris): num indice de zona do
// part1, as entradas "bg/locXXX/..." que pertencem a OUTRA zona sao ruido do
// dump original e nao devem aparecer na arvore.
func (part1) KeepZoneEntry(listingName, entryPath string) bool {
	const zonePrefix = "zone/filelist_z"
	const bgLocPrefix = "bg/loc"
	name := strings.ToLower(listingName)
	path := strings.ToLower(entryPath)
	if !strings.HasPrefix(name, zonePrefix) || !strings.HasPrefix(path, bgLocPrefix) {
		return true
	}
	if len(name) < len(zonePrefix)+3 || len(path) < len(bgLocPrefix)+3 {
		return true
	}
	return name[len(zonePrefix):len(zonePrefix)+3] == path[len(bgLocPrefix):len(bgLocPrefix)+3]
}
