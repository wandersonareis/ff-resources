package encoding

import (
	"strings"

	"ffresources/internal/families/white/languages"
)

// DefaultLanguage é o idioma do slot editável (o "_us").
const DefaultLanguage = "us"

// A ordem é a do catálogo EXCLUSIVO da trilogia (us primeiro). O alemão é
// "gr" nos arquivos do jogo (e na wiki); "de" fica só como alias de chamadas
// antigas, não entra na lista canônica.
var languageOrder = languages.FF13Codes()

// languageCodepages é o codepage FÍSICO de cada arquivo. ch/kr têm tabela
// própria; os demais (incluindo ck/uk/ru) seguem o LatinJapanese — o texto
// dessas traduções comunitárias é lido com a tabela do slot editável.
var languageCodepages = func() map[string]Codepage {
	out := make(map[string]Codepage, len(languages.FF13)+1)
	for _, l := range languages.FF13 {
		switch l.Code {
		case "ch":
			out[l.Code] = Chinese
		case "kr":
			out[l.Code] = Korean
		default:
			out[l.Code] = LatinJapanese
		}
	}
	out["de"] = LatinJapanese // alias legado
	return out
}()

// Languages devolve os códigos de idioma na ordem canônica.
func Languages() []string {
	out := make([]string, len(languageOrder))
	copy(out, languageOrder)
	return out
}

// NormalizeLanguage devolve o código em minúsculas; vazio vira DefaultLanguage.
func NormalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return DefaultLanguage
	}
	return lang
}

// CodepageForLanguage resolve o codepage FÍSICO do arquivo. A consulta é
// estrita: um código fora do catálogo devolve false (quem lê o arquivo decide
// se cai no DefaultLanguage; quem serve a paleta deve falhar).
func CodepageForLanguage(lang string) (Codepage, bool) {
	cp, ok := languageCodepages[NormalizeLanguage(lang)]
	return cp, ok
}

// IsFrench é o único idioma com exigência tipográfica de U+202F no espaço
// sem quebra.
func IsFrench(lang string) bool {
	return NormalizeLanguage(lang) == "fr"
}
