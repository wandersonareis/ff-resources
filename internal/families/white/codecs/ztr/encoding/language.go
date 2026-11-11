package encoding

import "strings"

// DefaultLanguage é o idioma do slot editável (o "_us").
const DefaultLanguage = "us"

// A ordem é a do catálogo de idiomas da trilogia (us primeiro). O alemão é
// "gr" nos arquivos do jogo (e na wiki); "de" fica só como alias de chamadas
// antigas, não entra na lista canônica.
var languageOrder = []string{"us", "jp", "fr", "gr", "it", "sp", "ch", "kr"}

var languageCodepages = map[string]Codepage{
	"us": LatinJapanese,
	"jp": LatinJapanese,
	"fr": LatinJapanese,
	"gr": LatinJapanese,
	"de": LatinJapanese, // alias legado
	"it": LatinJapanese,
	"sp": LatinJapanese,
	"ch": Chinese,
	"kr": Korean,
}

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
