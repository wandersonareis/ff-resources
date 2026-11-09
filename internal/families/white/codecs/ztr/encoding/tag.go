package encoding

import "strconv"

// MODELO das tags: a identidade é o PAR DE BYTES + a CLASSE. O nome é só
// display — o MESMO nome existe em bytes diferentes entre jogos (Icon Shield
// é f2:63 no ff132 e f2:5c no ff133), então nome não pode chavear nada.
//
// Formato no texto (o que o tradutor vê):
//
//	{COLOR:White}        cor: o NOME resolve o byte (paleta compartilhada)
//	{ICON:F0:40:Clock}   hex AUTORITATIVO + nome (só display)
//	{BTN:F1:40:A}        idem
//	{VAR:F4:40}          ex-{Entity 1}: hex somente, sem nome
//	{CTR:F7:40}          ex-{Counter Type 1}
//	{KEY:F6:40}          ex-{Key Entity}
//	{PAGE:40:70}         ex-{TEXT_NEW_PAGE}
//	{NL:40:72}           ex-{TEXT_NEW_LINE}
//	{TAB:85:60}          ex-{TEXT_TAB}
//	{RAW:81:40}          ex-{Unk81 40}/{ExChara85 80}: bytes crus
//	{B:FF}               0xFF isolado (ex-{FF})
//	{END} {ESC} {TEXT_ITALIC} {TEXT_NORMAL} {LINE} {ART} {ART2}
//	                      controles de byte único (SingleKeys)

// TagClass é a família semântica da tag. É o primeiro segmento do texto.
type TagClass string

const (
	// ClassColor é a única classe resolvida por NOME: a paleta é fechada e
	// a mesma cor muda de byte entre codepages/jogos.
	ClassColor TagClass = "COLOR"
	ClassIcon  TagClass = "ICON"
	ClassBtn   TagClass = "BTN"
	ClassVar   TagClass = "VAR"
	ClassCtr   TagClass = "CTR"
	ClassKey   TagClass = "KEY"
	ClassPage  TagClass = "PAGE"
	ClassNL    TagClass = "NL"
	ClassTab   TagClass = "TAB"
	ClassRaw   TagClass = "RAW"
)

// Tag é uma entrada de par: bytes + classe + nome de display.
type Tag struct {
	Class TagClass
	B1    byte
	B2    byte
	Name  string
}

// Pair devolve a chave de consulta do par (a identidade).
func (t Tag) Pair() uint16 { return uint16(t.B1)<<8 | uint16(t.B2) }

// named diz se a classe exibe o nome no texto (ICON/BTN). As demais classes
// de par são hex puro — o nome não é identidade e só ocuparia espaço.
func (t Tag) named() bool {
	return t.Class == ClassIcon || t.Class == ClassBtn
}

// Text formata a tag no formato novo:
//
//	COLOR    -> {COLOR:White}          (só o nome: ele resolve o byte)
//	ICON/BTN -> {ICON:F0:40:Clock}     (hex autoritativo + nome de display)
//	demais   -> {VAR:F4:40}            (hex puro, sem nome)
func (t Tag) Text() string {
	switch {
	case t.Class == ClassColor:
		return "{COLOR:" + t.Name + "}"
	case t.named():
		return "{" + string(t.Class) + ":" + hexByteUpper(t.B1) + ":" +
			hexByteUpper(t.B2) + ":" + t.Name + "}"
	default:
		return "{" + string(t.Class) + ":" + hexByteUpper(t.B1) + ":" + hexByteUpper(t.B2) + "}"
	}
}

// AppendText formata a tag direto no buffer de saída (sem string
// intermediária) — é o caminho quente do decode.
func (t Tag) AppendText(dst []byte) []byte {
	if t.Class == ClassColor {
		dst = append(dst, "{COLOR:"...)
		dst = append(dst, t.Name...)
		return append(dst, '}')
	}
	dst = append(dst, '{')
	dst = append(dst, t.Class...)
	dst = append(dst, ':')
	dst = appendHexByte(dst, t.B1)
	dst = append(dst, ':')
	dst = appendHexByte(dst, t.B2)
	if t.named() {
		dst = append(dst, ':')
		dst = append(dst, t.Name...)
	}
	return append(dst, '}')
}

// CharaLead devolve o byte lead do par chara desta codepage (0x85 no
// Shift-JIS, 0x80 no Big5/EUC-KR).
func (ks *KeySet) CharaLead() byte { return ks.charaLead }

// tagClassOf resolve o primeiro segmento da tag na classe. false = não é
// classe de par (pode ser um single: {END}, {TEXT_ITALIC}...).
func tagClassOf(upper string) (TagClass, bool) {
	switch upper {
	case "COLOR":
		return ClassColor, true
	case "ICON":
		return ClassIcon, true
	case "BTN":
		return ClassBtn, true
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
	case "TAB":
		return ClassTab, true
	case "RAW":
		return ClassRaw, true
	}
	return "", false
}

// hexByteUpper formata um byte em 2 dígitos hexadecimais maiúsculos.
func hexByteUpper(b byte) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[b>>4], digits[b&0x0F]})
}

// appendHexByte é o hexByteUpper sem alocação.
func appendHexByte(dst []byte, b byte) []byte {
	const digits = "0123456789ABCDEF"
	return append(dst, digits[b>>4], digits[b&0x0F])
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
