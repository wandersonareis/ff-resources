// Package family define os contratos compartilhados entre as familias de
// formato (FFX, White/FF13). Nao conhece nenhum jogo especifico: apenas as
// interfaces que os apps e o launcher usam.
//
// Fase de scaffold: Family/Profile/LanguageProfile existem para registrar os
// titulos; o Workspace (Tree/GetCollection/ApplyCollection/Import/Export) e um
// milestone posterior e sera adicionado quando o FF13 (part1) ganhar a trilha
// de texto.
package family

type Family interface {
	ID() string
	Profiles() []Profile
}

type Profile interface {
	ID() string
	DisplayName() string
	GameCode() string
	Languages() LanguageProfile
}

type LanguageProfile struct {
	Codes             []string          // ex.: us, jp, fr, de, it, sp, ch, kr
	ReferenceLanguage string            // idioma de referencia (fonte), ex.: "us"
	TargetLanguage    string            // slot alvo da traducao, ex.: "us"
	FilePattern       string            // ex.: ^(?P<base>.+)_(?P<lang>[a-z]{2})\.ztr$
	CharsetByLang     map[string]string // codepage por idioma (vazio = UTF-8)
}
