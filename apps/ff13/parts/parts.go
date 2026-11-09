// Package parts concentra as PARTICULARIDADES de cada jogo da trilogia
// FINAL FANTASY XIII (part1/part2/part3).
//
// O filelist e a arvore de /data nao sao iguais nos tres jogos: cada um tem
// sua pasta de dados (white_data/alba_data/weiss_data), seus diretorios de
// indice, seu game code e sua convencao de par filelist <-> white bin. Deixar
// isso em um arquivo por part evita "if part == ..." espalhado no servico.
package parts

import (
	"path/filepath"
	"strings"

	fl "ffresources/internal/families/white/codecs/filelist"
)

// Part descreve as particularidades de UM jogo da trilogia. O servico Wails
// so orquestra: pergunta a Part onde estao os arquivos e como parear.
type Part interface {
	ID() string
	GameCode() fl.GameCode
	// DataDir e a pasta que contem os white bins do jogo (white_data etc.).
	// Recebe a raiz do jogo (ou a propria pasta de dados) e devolve o caminho
	// absoluto, mesmo que ainda nao exista.
	DataDir(gameRoot string) string
	// DataDirName e o NOME da pasta de dados (white_data/alba_data/
	// weiss_data): e o "sourceDir" do DTO de filelist e o criterio do
	// desempate entre as copias do mesmo indice.
	DataDirName() string
	// ZoneAsTopLevel informa se os indices de ZONA viram um no "Zone" no
	// topo da arvore de filelist (part1) em vez de um grupo com os indices
	// dentro.
	ZoneAsTopLevel() bool
	// FilelistDirs sao subpastas de DataDir onde procurar filelists no disco.
	// A busca e recursiva dentro de cada uma delas.
	FilelistDirs() []string
	// PairKey normaliza o nome de um arquivo (filelist OU white bin) para a
	// chave do par. ok=false = arquivo que nao participa do pareamento.
	PairKey(fileName string) (key string, ok bool)
	// EntryType classifica o conteudo pelo caminho interno (system, event...).
	// O backend nao traduz: o frontend decide rotulo/icone.
	EntryType(rel string) string
	// TextKind classifica o TEXTO pelo diretorio (ac/resident/zone/event/
	// scenario). E o escopo de export/import: o FF13 tem um formato de
	// texto so (o .ztr), entao a pasta e o que separa os tipos.
	TextKind(rel string) string
	// HasTextKind diz se a part TEM o kind (o escopo de export/import so
	// oferece os kinds que a part serviu).
	HasTextKind(kind string) bool
	// KeepZoneEntry decide se uma entrada de um indice de ZONA deve aparecer.
	// listingName e o nome do indice que contem a entrada (ex.:
	// "zone/filelist_z000c.win32.bin"); entryPath e o caminho da entrada.
	// Cada part pode ter a sua regra (o Pulse so filtra no part1).
	KeepZoneEntry(listingName, entryPath string) bool
	// LanguageOf devolve o codigo de idioma do INDICE ("u"/"c") pelo nome do
	// arquivo do indice; "" quando o nome nao diz. Cada part tem a sua
	// convencao: part1/part2 usam o sufixo "u"/"c", o part3 usa o par "a"/"v"
	// (a chave do par carrega o idioma).
	LanguageOf(fileName string) string
}

// Get resolve a Part pelo id estavel (part1/part2/part3). Ids desconhecidos
// caem no part1, mantendo o comportamento antigo.
func Get(id string) Part {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "part2":
		return part2{}
	case "part3":
		return part3{}
	default:
		return part1{}
	}
}

// Kinds devolve os kinds de TEXTO que a part serviu, na ordem canonica. E o
// que o frontend mostra no escopo de export/import: um kind que a part nao
// tem NAO aparece (o `ac` do part3 e o `scenario` do part1/part2).
func Kinds(id string) []string {
	p := Get(id)
	all := []string{KindAC, KindResident, KindZone, KindEvent, KindScenario}
	out := make([]string, 0, len(all))
	for _, k := range all {
		if p.HasTextKind(k) {
			out = append(out, k)
		}
	}
	return out
}

// hasKinds é a implementação de HasTextKind: true quando o kind pedido está
// no conjunto servido. Fica em um helper para as parts não repetirem o
// switch — só o CONJUNTO muda entre elas.
func hasKinds(kind string, served ...string) bool {
	for _, k := range served {
		if kind == k {
			return true
		}
	}
	return false
}

// HasTextKind diz se a part TEM o kind (usado pela lista de kinds e pelo
// filtro de export). O `scenario` (db/ai/npc/pack) só existe no part3.
func (part1) HasTextKind(kind string) bool {
	return hasKinds(kind, KindAC, KindResident, KindZone, KindEvent)
}

// HasTextKind diz se a part TEM o kind (usado pela lista de kinds e pelo
// filtro de export). O `scenario` (db/ai/npc/pack) só existe no part3.
func (part2) HasTextKind(kind string) bool {
	return hasKinds(kind, KindAC, KindResident, KindZone, KindEvent)
}

// HasTextKind diz se a part TEM o kind (usado pela lista de kinds e pelo
// filtro de export). O part3 não tem `ac`.
func (part3) HasTextKind(kind string) bool {
	return hasKinds(kind, KindResident, KindZone, KindEvent, KindScenario)
}

// NormalizedPart devolve o id canonico usado no resto do app.
func NormalizedPart(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "part2":
		return "part2"
	case "part3":
		return "part3"
	default:
		return "part1"
	}
}

// dataDir resolve a pasta de dados aceitando tanto a raiz do jogo quanto a
// propria pasta (white_data/alba_data/weiss_data) ja apontada.
func dataDir(gameRoot, name string) string {
	root := strings.TrimSpace(gameRoot)
	if root == "" {
		return ""
	}
	if strings.EqualFold(filepath.Base(filepath.Clean(root)), name) {
		return filepath.Clean(root)
	}
	return filepath.Join(root, name)
}

// stemName devolve o nome em minusculas sem .bin2/.bin/.win32 — a base usada
// pelas regras de pareamento.
func stemName(name string) string {
	n := strings.ToLower(filepath.Base(name))
	n = strings.TrimSuffix(n, ".bin2")
	n = strings.TrimSuffix(n, ".bin")
	n = strings.TrimSuffix(n, ".win32")
	return n
}

// cutPrefix remove um prefixo e o "_" opcional seguinte.
func cutPrefix(s, prefix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimPrefix(s, "_")
}

// entryTypeByPath e a regua comum; cada Part pode ter a sua.
func entryTypeByPath(rel string) string {
	p := strings.ToLower(filepath.ToSlash(rel))
	switch {
	case strings.Contains(p, "/system/"):
		return "system"
	case strings.Contains(p, "/mission/"):
		return "mission"
	case strings.Contains(p, "/book"):
		return "book"
	case strings.Contains(p, "/review/"):
		return "review"
	case strings.Contains(p, "/event/"):
		return "event"
	case strings.Contains(p, "/zone/"):
		return "zone"
	case strings.Contains(p, "/ac/"):
		return "ac"
	default:
		return "text"
	}
}
