package parts

import (
	"path/filepath"
	"strings"

	"ffresources/apps/ff13/backend/common"
	fl "ffresources/internal/families/white/codecs/filelist"
)

// part3 = LIGHTNING RETURNS: FINAL FANTASY XIII. Layout V2 (mesmo codec do
// XIII-2), mas com outra distribuicao de pastas.
//
// Particularidades:
//   - dados em `weiss_data`;
//   - filelists de disco em `weiss_data/sys` e em cada DLC (`weiss_data/dlc`);
//   - o par e filelist_<chave> <-> white_<chave> SEM normalizar sufixos:
//     filelist_p0000000img_a pareia com white_p0000000img_a (o _a/_v faz
//     parte da chave).
type part3 struct{}

func (part3) ID() string            { return "part3" }
func (part3) GameCode() fl.GameCode { return fl.GameFF132 }
func (part3) DataDir(root string) string {
	return dataDir(root, "weiss_data")
}
func (part3) DataDirName() string { return common.DataDirName("part3") }

// ZoneAsTopLevel: no part3 os indices de ZONA ficam no grupo "zone" (o
// conteudo da zona nao vira no de topo).
func (part3) ZoneAsTopLevel() bool   { return false }
func (part3) FilelistDirs() []string { return []string{"sys", "dlc"} }

func (part3) PairKey(fileName string) (string, bool) {
	n := stemName(fileName)
	switch {
	case strings.HasPrefix(n, "filelist"):
		n = cutPrefix(n, "filelist")
	case strings.HasPrefix(n, "white"):
		n = cutPrefix(n, "white")
		// white_img2a/white_imga -> 2a/a (os indices sao filelist2a/filelista);
		// white_z0120a_img -> z0120a (filelist interno filelist_z0120a).
		if strings.HasPrefix(n, "img") {
			n = strings.TrimPrefix(n, "img")
		}
		n = strings.TrimSuffix(n, "_img2")
		n = strings.TrimSuffix(n, "_img")
	default:
		return "", false
	}
	n = strings.TrimSuffix(n, ".win32")
	if n == "" {
		return "", false
	}
	return filepath.Base(n), true
}

// KeepZoneEntry: no part3 todas as entradas valem.
func (part3) KeepZoneEntry(string, string) bool { return true }

// TextKind: o part3 tem resident/zone/event (sob txtres) e scenario (as
// falas de IA/quest em db/ai/npc/pack — container .bin com o .ztr dentro).
// Nao tem `ac`.
func (part3) TextKind(rel string) string {
	if k := scenarioKindByPath(rel, scenarioDirs); k != "" {
		return k
	}
	return textKindByPath(rel)
}

func (part3) EntryType(rel string) string {
	p := strings.ToLower(filepath.ToSlash(rel))
	switch {
	case strings.Contains(p, "/sys/"):
		return "system"
	case strings.Contains(p, "/dlc/"):
		return "dlc"
	case strings.Contains(p, "/zone/"):
		return "zone"
	case strings.Contains(p, "/event/"):
		return "event"
	case strings.Contains(p, "/mission/"):
		return "mission"
	default:
		return "text"
	}
}

// LanguageOf devolve o idioma do indice do part3 (LR).
//
// O LR nao usa o par "u"/"c" do part1/2: o par e "a" (ingles) e "v"
// (japones), e o marcador fica no FIM da chave do par ("filelista" ⇄
// "white_imga" -> "a"). A chave do par (PairKey) e a fonte: ela ja normaliza
// o nome (tira o prefixo "filelist"/"white", o "img" e o "_img"/"_img2"), de
// modo que o ultimo caractere e o idioma.
//
// Usar a chave do par (e nao "qualquer nome que termina em a/v") evita
// rotular como idioma um arquivo que so termine em "a".
func (part3) LanguageOf(fileName string) string {
	key, ok := part3{}.PairKey(fileName)
	if !ok || key == "" {
		return ""
	}
	switch key[len(key)-1] {
	case 'a':
		return common.LangU
	case 'v':
		return common.LangC
	}
	return ""
}
