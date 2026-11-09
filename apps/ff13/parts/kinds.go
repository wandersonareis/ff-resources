package parts

import (
	"path/filepath"
	"strings"
)

// KINDS DE TEXTO do FF13.
//
// O texto do FF13 tem UM formato só (o .ztr), então dividi-lo por tipo de
// ARQUIVO não separa nada. A régua que separa é o DIRETÓRIO: o `txtres` é o
// pai de todos os subdiretórios de texto, e o nome da pasta logo abaixo dele
// é a família do conteúdo.
//
//	ac        → txtres/ac        (habilidades / descrições deCharacteristic)
//	resident  → txtres/resident  (livro, missões, sistema, review, fragment)
//	zone      → txtres/zone      (textos de zona)
//	event     → txtres/event     (scripts de evento)
//	scenario  → db/ai/npc/pack   (LR: falas de IA/quest — container WPD)
//
// O `scenario` é o ÚNICO kind cujo texto NÃO é um .ztr solto: o arquivo é um
// container .bin (WPD) que carrega o .ztr dentro. Todos os outros são .ztr
// avulso — por isso o importador não precisa de caminho especial por kind.
const (
	// KindAC é o kind de txtres/ac.
	KindAC = "ac"
	// KindResident é o kind de txtres/resident.
	KindResident = "resident"
	// KindZone é o kind de txtres/zone.
	KindZone = "zone"
	// KindEvent é o kind de txtres/event.
	KindEvent = "event"
	// KindScenario é o kind de db/ai/npc/pack (só no part3).
	KindScenario = "scenario"
	// KindText é o fallback: texto que não cai em nenhuma pasta conhecida.
	KindText = "text"
)

// txtresKinds são as famílias que vivem DENTRO de txtres, indexadas pelo nome
// da pasta. O mesmo nome é a chave e o valor do kind.
var txtresKinds = map[string]string{
	"ac":       KindAC,
	"resident": KindResident,
	"zone":     KindZone,
	"event":    KindEvent,
}

// scenarioDirs são os prefixos de caminho que carregam o kind `scenario`.
// Cada part declara os seus (só o part3 tem).
var scenarioDirs = []string{
	"db/ai/npc/pack",
}

// textKindByPath classifica o TEXTO pelo caminho interno do jogo.
//
// A regra é mecânica: procura o segmento `txtres` no caminho e devolve o
// kind da pasta logo abaixo dele. Isso cobre o `txtres` global
// ("txtres/event/ev_comn_000/txtres_us.ztr") E o `txtres` de zona
// ("zone/z004/txtres/ac/ac_bodm_230/txtres_us.ztr") — no part1 o `ac` só
// existe dentro das zonas.
//
// O prefixo `zone/<z>/` é o prefixo que o APP usa no layout de data/; o
// caminho interno do jogo começa em `txtres`. Nos dois casos o segmento
// `txtres` aparece, então a mesma regra serve para os dois.
func textKindByPath(rel string) string {
	p := strings.ToLower(filepath.ToSlash(rel))
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if s != "txtres" {
			continue
		}
		if i+1 < len(segs) {
			if kind, ok := txtresKinds[segs[i+1]]; ok {
				return kind
			}
		}
		return KindText
	}
	return KindText
}

// scenarioKindByPath devolve o kind `scenario` quando o caminho está numa das
// pastas de scenario declaradas.
func scenarioKindByPath(rel string, dirs []string) string {
	p := strings.ToLower(filepath.ToSlash(rel))
	for _, d := range dirs {
		if strings.HasPrefix(p, d+"/") || p == d {
			return KindScenario
		}
	}
	return ""
}
