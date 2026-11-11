package encoding

import "ffresources/internal/families/white/codecs/ztr/encoding/tags"

// buildTextCodec compõe o codec do par (game, codepage) a partir das tabelas
// do pacote tags. O idioma entra como codepage: cada arquivo carrega a
// tabela inteira do seu idioma (inclusive a paleta de cores).
func buildTextCodec(game GameCode, cp Codepage) (*TextCodec, error) {
	charset, err := CharsetFor(cp)
	if err != nil {
		return nil, err
	}
	table, err := tags.For(game, cp)
	if err != nil {
		return nil, err
	}
	tc := &TextCodec{
		Game:      game,
		Codepage:  cp,
		Charset:   charset,
		Controls:  table.Controls,
		Colors:    table.Colors,
		Icons:     table.Icons,
		Buttons:   table.Buttons,
		Specials:  table.Specials,
		Raw:       table.Raw,
		RawExtra:  table.RawExtra,
		Glyphs:    table.Glyphs,
		glyphLead: table.GlyphLead,
	}
	buildIndexes(tc)
	return tc, nil
}
