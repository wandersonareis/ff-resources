package encoding

import "ffresources/internal/families/white/codecs/ztr/encoding/tags"

// TextCodec é a composição de dicionários de um par (jogo, codepage). Cada
// arquivo de texto usa o codec do SEU idioma; não existe codec "compartilhado"
// entre codepages.
//
// Os pares são Tag (classe + bytes + display): a identidade é o PAR, e a
// consulta é por CLASSE numa ordem única (ver indexes.go). Os glifos ficam no
// meio da ordem.
type TextCodec struct {
	Game     GameCode
	Codepage Codepage
	Charset  Charset // codepage física (texto comum)

	Controls []tags.ByteTag // 0x00..0x05
	Colors   []Tag
	Icons    []Tag
	Buttons  []Tag
	Specials []Tag // PAGE/NL/SPACE + VAR/CTR/KEY
	Raw      []Tag
	RawExtra []Tag // nil no chinês, como no decoder legado
	Glyphs   []tags.GlyphPair

	glyphLead byte

	// Índices por categoria (montados por buildIndexes; imutáveis depois).
	controlFwd map[byte]string
	controlRev map[string]byte
	pairFwd    map[TagClass]map[uint16]Tag
	nameFwd    map[TagClass]map[string]Tag
	glyphFwd   map[uint16]rune
	glyphRev   map[rune]uint16
}

// GlyphLead devolve o byte lead dos pares de glifos desta codepage (0x85 no
// Shift-JIS, 0x80 no Big5/EUC-KR).
func (tc *TextCodec) GlyphLead() byte { return tc.glyphLead }

// Ordem de consulta dos pares, única para decode e encode. Nenhum par pode
// existir em duas classes; o checker falha se isso quebrar.
var (
	classesBeforeGlyphs = []TagClass{ClassColor, ClassIcon, ClassBtn}
	classesAfterGlyphs  = []TagClass{
		ClassPage, ClassNL, ClassSpace,
		ClassVar, ClassCtr, ClassKey,
		ClassRaw,
	}
)
