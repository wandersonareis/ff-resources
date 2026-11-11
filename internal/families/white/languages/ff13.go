// Package languages centraliza o catálogo de idiomas da família White
// (FINAL FANTASY XIII / XIII-2 / Lightning Returns).
//
// O catálogo é EXCLUSIVO da trilogia: o FFX usa outros códigos (por exemplo
// "de" para alemão), então não dá para reaproveitar common.SupportedLanguages.
// Aqui o alemão é "gr", como nos arquivos do jogo, e os códigos de traduções
// comunitárias (ck/uk/ru) entram como idiomas de referência.
package languages

// Language é um idioma do catálogo: Code é o sufixo usado nos arquivos
// (txtres_<code>.ztr) e Name é o rótulo exibido na UI.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// FF13 é o catálogo canônico da trilogia, na ordem de exibição: o inglês
// (slot editável) primeiro e os demais na ordem histórica, com os códigos das
// traduções comunitárias ao final.
var FF13 = []Language{
	{Code: "us", Name: "English"},
	{Code: "jp", Name: "Japanese"},
	{Code: "fr", Name: "French"},
	{Code: "gr", Name: "German"},
	{Code: "it", Name: "Italian"},
	{Code: "sp", Name: "Spanish"},
	{Code: "ch", Name: "Chinese"},
	{Code: "kr", Name: "Korean"},
	{Code: "ck", Name: "Czech"},
	{Code: "uk", Name: "Ukrainian"},
	{Code: "ru", Name: "Russian"},
}

// FF13Codes devolve só os códigos, na ordem canônica. É a lista que alimenta a
// unitLangs (sufixo de arquivo), o codec/paleta e o profile da família.
func FF13Codes() []string {
	out := make([]string, len(FF13))
	for i, l := range FF13 {
		out[i] = l.Code
	}
	return out
}
