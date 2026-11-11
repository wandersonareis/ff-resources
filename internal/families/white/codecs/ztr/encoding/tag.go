package encoding

import (
	"strconv"

	"ffresources/internal/families/white/codecs/ztr/encoding/tags"
)

// MODELO das tags: a identidade é o PAR DE BYTES + a classe. O nome é só
// display. O formato é comum a todas as classes de par:
//
//	{CLR:F9:40:White}     cor (bytes do idioma; nome/swatch display)
//	{ICON:F0:40:Clock}    ícone
//	{BTN:F1:40:A}         botão
//	{SPACE:85:60}         espaço sem quebra (U+00A0; U+202F no francês)
//	{VAR:F4:40}           variável do jogo
//	{CTR:F7:40}           contador
//	{KEY:F6:40}           item-chave
//	{PAGE:40:70} {NL:40:72} {RAW:81:40} {B:FF}
//	{END} {ESC} {TEXT_ITALIC} {LINE} {ART} {ART2}
type Tag = tags.Tag

// TagClass é a família semântica da tag. É o primeiro segmento do texto.
type TagClass = tags.Class

const (
	ClassColor = tags.ClassColor
	ClassIcon  = tags.ClassIcon
	ClassBtn   = tags.ClassBtn
	ClassSpace = tags.ClassSpace
	ClassVar   = tags.ClassVar
	ClassCtr   = tags.ClassCtr
	ClassKey   = tags.ClassKey
	ClassPage  = tags.ClassPage
	ClassNL    = tags.ClassNL
	ClassRaw   = tags.ClassRaw
)

// tagClassOf resolve o primeiro segmento da tag na classe. false = não é
// classe de par (pode ser um single: {END}, {TEXT_ITALIC}...).
func tagClassOf(upper string) (TagClass, bool) {
	switch upper {
	case "CLR":
		return ClassColor, true
	case "ICON":
		return ClassIcon, true
	case "BTN":
		return ClassBtn, true
	case "SPACE":
		return ClassSpace, true
	case "VAR":
		return ClassVar, true
	case "CTR":
		return ClassCtr, true
	case "KEY":
		return ClassKey, true
	case "PAGE":
		return ClassPage, true
	case "NL":
		return ClassNL, true
	case "RAW":
		return ClassRaw, true
	}
	return "", false
}

// parseHexByte converte UM componente hex de token (1 ou 2 dígitos).
// Segundo retorno falso = token inválido.
func parseHexByte(s string) (byte, bool) {
	if len(s) == 0 || len(s) > 2 {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, false
	}
	return byte(v), true
}
