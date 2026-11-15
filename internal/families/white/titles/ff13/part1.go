// Package ff13 concentra os titulos da trilogia FINAL FANTASY XIII como
// profiles da familia White. Os IDs internos sao estaveis (part1/part2/part3);
// os nomes de exibicao longos vivem aqui e no manifesto do launcher.
package ff13

import (
	"ffresources/internal/families/white/family"
	"ffresources/internal/families/white/languages"
)

type profile struct {
	id       string
	display  string
	gameCode string
	langs    family.LanguageProfile
}

func (p profile) ID() string                        { return p.id }
func (p profile) DisplayName() string               { return p.display }
func (p profile) GameCode() string                  { return p.gameCode }
func (p profile) Languages() family.LanguageProfile { return p.langs }

// base monta um profile da trilogia com o perfil de idiomas do FF13: arquivos
// irmaos no mesmo diretorio, mudando apenas o sufixo (txt_us.ztr, txt_jp.ztr).
func base(id, display, gameCode string) profile {
	return profile{
		id:       id,
		display:  display,
		gameCode: gameCode,
		langs: family.LanguageProfile{
			Codes:             languages.FF13Codes(),
			ReferenceLanguage: "us",
			TargetLanguage:    "us",
			FilePattern:       `^(?P<base>.+)_(?P<lang>[a-z]{2})\.ztr$`,
			CharsetByLang:     map[string]string{},
		},
	}
}

// Part1 -> FINAL FANTASY XIII
func Part1() profile { return base("part1", "FINAL FANTASY XIII", "ff131") }
