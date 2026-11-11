// Package tags guarda as tabelas de tags do texto do FF13 (trilogia XIII).
//
// A identidade de uma tag de par é (classe, B1, B2). O nome e o swatch são
// apenas display: o MESMO nome pode existir em bytes diferentes entre jogos
// e idiomas. O formato textual é o comum às classes:
//
//	{CLASSE:B1:B2:NOME}   com nome (ex.: {CLR:F9:40:White})
//	{CLASSE:B1:B2}        sem nome (ex.: {VAR:F4:40})
//
// O hex é sempre autoritativo no encode; o nome é ignorado.
package tags

// Class é a família semântica da tag. É o primeiro segmento do texto.
type Class string

const (
	// ClassColor é a paleta de cor do texto. O par vem do idioma/codepage;
	// o nome e o swatch são representativos.
	ClassColor Class = "CLR"
	ClassIcon  Class = "ICON"
	ClassBtn   Class = "BTN"
	// ClassSpace é o espaço sem quebra (U+00A0; no francês, U+202F).
	// O nome antigo era {TEXT_TAB} — não é tabulação.
	ClassSpace Class = "SPACE"
	ClassVar   Class = "VAR"
	ClassCtr   Class = "CTR"
	ClassKey   Class = "KEY"
	ClassPage  Class = "PAGE"
	ClassNL    Class = "NL"
	ClassRaw   Class = "RAW"
)

// Tag é uma entrada de par: classe + bytes + display.
type Tag struct {
	Class Class
	B1    byte
	B2    byte
	Name  string
	// Swatch é a cor representativa do display (só ClassColor). Nunca
	// participa do encode nem da identidade.
	Swatch string
}

// Pair devolve a chave de consulta do par (a identidade).
func (t Tag) Pair() uint16 { return uint16(t.B1)<<8 | uint16(t.B2) }

// Text formata a tag no formato comum: {CLASSE:B1:B2:NOME}.
func (t Tag) Text() string { return string(t.AppendText(nil)) }

// AppendText formata a tag direto no buffer de saída (sem string
// intermediária) — é o caminho quente do decode.
func (t Tag) AppendText(dst []byte) []byte {
	dst = append(dst, '{')
	dst = append(dst, t.Class...)
	dst = append(dst, ':')
	dst = appendHexByte(dst, t.B1)
	dst = append(dst, ':')
	dst = appendHexByte(dst, t.B2)
	if t.Name != "" {
		dst = append(dst, ':')
		dst = append(dst, t.Name...)
	}
	return append(dst, '}')
}

// FormatPair monta a forma textual comum a partir dos bytes e do nome.
func FormatPair(class Class, b1, b2 byte, name string) string {
	return Tag{Class: class, B1: b1, B2: b2, Name: name}.Text()
}

// ByteTag é um mapa byte único -> tag (controles 0x00..0x05).
type ByteTag struct {
	Key   byte
	Value string
}

// GlyphPair é a "codepage estendida" do jogo: trail do par -> runa exibida.
// O lead varia por codepage e é aplicado na composição (glyphLeadByCodepage).
type GlyphPair struct {
	Trail byte
	Rune  rune
}

const hexDigits = "0123456789ABCDEF"

func appendHexByte(dst []byte, b byte) []byte {
	return append(dst, hexDigits[b>>4], hexDigits[b&0x0F])
}
