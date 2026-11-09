package common

const (
	OriginalsFolder     = "jppc/"
	DefaultLocalization = "us"
)

// TranslationLanguage é o idioma alvo das traduções (o PO declara no header
// `Language:`). O idioma fonte é o DefaultLocalization ("us").
const TranslationLanguage = "pt-BR"

// DefaultTargetLocalization é o código do idioma ALVO como LINHA do formato
// .strings (v2): a tradução de mods/ vira uma linha própria `<index>:br` em
// vez do campo `target` na linha do idioma fonte. Dois letras, como todos os
// idiomas de linha do formato — `pt-BR` acima é o nome do header PO, não um
// código de linha.
const DefaultTargetLocalization = "br"
