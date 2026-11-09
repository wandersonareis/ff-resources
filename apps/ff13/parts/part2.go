package parts

import (
	"strings"

	"ffresources/apps/ff13/common"
	fl "ffresources/internal/families/white/codecs/filelist"
)

// part2 = FINAL FANTASY XIII-2. Layout V2, filelist criptografado.
//
// Particularidades:
//   - dados em `alba_data`;
//   - filelists de disco em `alba_data/sys` e nos patches em `alba_data/udp`;
//   - o par e sempre filelist_X <-> white_X (inclui filelist_patch_c e
//     filelist_patch_scr_c); filelistu/filelistc seguem pareando com
//     white_imgu/white_imgc.
type part2 struct{}

func (part2) ID() string            { return "part2" }
func (part2) GameCode() fl.GameCode { return fl.GameFF132 }
func (part2) DataDir(root string) string {
	return dataDir(root, "alba_data")
}
func (part2) DataDirName() string { return common.DataDirName("part2") }

// ZoneAsTopLevel: no part2 os indices de ZONA ficam no grupo "zone" (o
// conteudo da zona nao vira no de topo).
func (part2) ZoneAsTopLevel() bool   { return false }
func (part2) FilelistDirs() []string { return []string{"sys", "udp"} }

func (part2) PairKey(fileName string) (string, bool) {
	n := stemName(fileName)
	switch {
	case strings.HasPrefix(n, "filelist"):
		n = cutPrefix(n, "filelist")
	case strings.HasPrefix(n, "white"):
		n = cutPrefix(n, "white")
		// white_img2a/white_imga -> 2a/a; white_z0093u_img -> z0093u.
		if strings.HasPrefix(n, "img") {
			n = strings.TrimPrefix(n, "img")
		}
		n = strings.TrimSuffix(n, "_img2")
		n = strings.TrimSuffix(n, "_img")
	default:
		return "", false
	}
	if n == "" {
		return "", false
	}
	return n, true
}

func (part2) EntryType(rel string) string { return entryTypeByPath(rel) }

// TextKind: o part2 tem ac/resident/zone/event (todos globais, sob txtres).
func (part2) TextKind(rel string) string { return textKindByPath(rel) }

// LanguageOf: no part2 o idioma do indice vem do sufixo do nome ("u" = ingles,
// "c" = japones).
func (part2) LanguageOf(fileName string) string { return common.LanguageOf(fileName) }

// KeepZoneEntry: no part2 nao ha o ruido de "bg/loc" que o Pulse filtra no
// part1 — todas as entradas valem.
func (part2) KeepZoneEntry(string, string) bool { return true }
