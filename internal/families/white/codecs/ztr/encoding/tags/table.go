package tags

import (
	"errors"
	"fmt"
)

// Sentinelas de configuração ausente. O pacote encoding as reexporta para
// manter os erros públicos do codec.
var (
	ErrUnknownGame     = errors.New("jogo desconhecido nas tabelas de tags")
	ErrUnknownCodepage = errors.New("codepage desconhecida nas tabelas de tags")
)

// Tables é a composição de UMA tabela (jogo, codepage). O idioma entra como
// codepage: cada arquivo carrega os valores do seu idioma.
type Tables struct {
	Controls  []ByteTag
	Colors    []Tag
	Icons     []Tag
	Buttons   []Tag
	Specials  []Tag
	Raw       []Tag
	RawExtra  []Tag
	Glyphs    []GlyphPair
	GlyphLead byte
}

// For compõe a tabela do par (jogo, codepage). O charset é responsabilidade
// do pacote encoding; aqui ficam só as tags e os glifos.
func For(game GameCode, cp Codepage) (*Tables, error) {
	colors, ok := colorsFor(game)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownGame, game)
	}
	colorList, ok := colors[cp]
	if !ok {
		return nil, fmt.Errorf("%w: %d (game=%s)", ErrUnknownCodepage, cp, game)
	}
	icons, ok := iconsFor(game)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownGame, game)
	}
	buttons, ok := buttonsFor(game)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownGame, game)
	}
	specials, ok := specialTagsByCodepage[cp]
	if !ok {
		return nil, fmt.Errorf("%w: %d (game=%s)", ErrUnknownCodepage, cp, game)
	}
	glyphLead, ok := glyphLeadByCodepage[cp]
	if !ok {
		return nil, fmt.Errorf("%w: %d (game=%s)", ErrUnknownCodepage, cp, game)
	}

	rawExtra := rawTagsExtra
	if cp == Chinese {
		// KeysDecoderCh/KeysEncoderCh não consultam Unk2Keys.
		rawExtra = nil
	}

	return &Tables{
		Controls:  controlTags,
		Colors:    colorList,
		Icons:     icons.expand(ClassIcon, cp),
		Buttons:   buttons.expand(ClassBtn, cp),
		Specials:  specials,
		Raw:       rawTags,
		RawExtra:  rawExtra,
		Glyphs:    glyphPairs,
		GlyphLead: glyphLead,
	}, nil
}

func colorsFor(game GameCode) (map[Codepage][]Tag, bool) {
	switch game {
	case FF131:
		return colorsFF13, true
	case FF132:
		return colorsFF13_2, true
	case FF133:
		return colorsFF13_3, true
	}
	return nil, false
}

func iconsFor(game GameCode) (SpanSet, bool) {
	switch game {
	case FF131:
		return iconsFF13, true
	case FF132:
		return iconsFF13_2, true
	case FF133:
		return iconsFF13_3, true
	}
	return SpanSet{}, false
}

func buttonsFor(game GameCode) (SpanSet, bool) {
	switch game {
	case FF131:
		return buttonsFF13, true
	case FF132:
		return buttonsFF13_2, true
	case FF133:
		return buttonsFF13_3, true
	}
	return SpanSet{}, false
}
