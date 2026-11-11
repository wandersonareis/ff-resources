package encoding

import (
	"strings"

	loggingService "ffresources/internal/shared/logging"
)

// CODEC das tags: texto entre '{' e '}' -> bytes. É o único lugar que
// conhece a GRAMÁTICA da tag — a máquina de estados (encode.go) só pergunta
// "esta tag casou?".
//
// Gramática (a identidade é o PAR DE BYTES; o nome é display):
//
//	{CLR:F9:40:White}      cor: os bytes do idioma mandam
//	{ICON:F0:40:Clock}
//	{BTN:F1:40:A}
//	{SPACE:85:60}          espaço sem quebra (U+00A0; U+202F no francês)
//	{VAR:F4:40} {CTR:F7:40} {KEY:F6:40}
//	{PAGE:40:70} {NL:40:72}
//	{RAW:81:40}
//	{B:FF}                 0xFF isolado
//	{END} {ESC} {TEXT_ITALIC} {LINE} {ART} {ART2}
//
// Tag desconhecida devolve false e o chamador faz o rewind do '{'. Em
// particular, {COLOR:Nome} e {TAB:...} NÃO são aceitos: o formato é sempre
// {CLASSE:B1:B2[:NOME]} com os dois bytes.

// encodeTag devolve os bytes de uma tag (inner = conteúdo entre '{' e '}',
// sem as chaves).
func (tc *TextCodec) encodeTag(inner string) ([]byte, bool) {
	head, rest, hasColon := strings.Cut(inner, ":")
	upper := strings.ToUpper(strings.TrimSpace(head))

	// Byte único isolado: {B:FF}.
	if upper == "B" {
		v, ok := parseHexByte(strings.TrimSpace(rest))
		return []byte{v}, ok
	}

	// Classe de par: o hex é AUTORITATIVO; o nome (se houver) é ignorado.
	if _, ok := tagClassOf(upper); ok {
		b1, b2, ok := hexPair(rest)
		if !ok {
			return nil, false
		}
		if _, known := tc.tagByPair(b1, b2); !known {
			// Par fora das tabelas: aceita (hex é autoritativo) mas avisa —
			// o decode seguinte emite {RAW:..} e a degradação fica visível.
			loggingService.Warn("tag %s: par %02X %02X não está nas tabelas do jogo",
				inner, b1, b2)
		}
		return []byte{b1, b2}, true
	}

	// Sem classe: single nomeado ({END}, {TEXT_ITALIC}...).
	if !hasColon {
		if b, ok := tc.EncodeControl("{" + upper + "}"); ok {
			return []byte{b}, true
		}
	}
	return nil, false
}

// EncodeTag é a face pública de encodeTag: os bytes de uma tag (sem as
// chaves). Serve para validar uma tag digitada/sugerida sem passar pela
// máquina de estados inteira.
func (tc *TextCodec) EncodeTag(inner string) ([]byte, bool) { return tc.encodeTag(inner) }

// closeBrace encontra o '}' de uma tag aberta em `open`, limitando o
// conteúdo interno a 48 runas. Devolve o índice do '}' ou false quando não
// há fechamento no limite — o '{' vira literal e o conteúdo é reprocessado
// como texto.
func closeBrace(rs []rune, open int) (int, bool) {
	limit := open + 48
	if limit >= len(rs) {
		limit = len(rs) - 1
	}
	for i := open + 1; i <= limit; i++ {
		if rs[i] == '}' {
			return i, true
		}
	}
	return 0, false
}

// hexPair converte os dois primeiros segmentos hex de um payload
// ("F9:40:White" -> 0xF9, 0x40). O resto (o nome) é ignorado.
func hexPair(s string) (b1, b2 byte, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	b1, ok1 := parseHexByte(parts[0])
	b2, ok2 := parseHexByte(parts[1])
	return b1, b2, ok1 && ok2
}
