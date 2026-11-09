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
//	{COLOR:White}         nome -> par (paleta compartilhada)
//	{ICON:F0:40:Clock}    par do hex; nome ignorado
//	{BTN:F1:40:A}
//	{VAR:F4:40} {CTR:F7:40} {KEY:F6:40}
//	{PAGE:40:70} {NL:40:72} {TAB:85:60}
//	{RAW:81:40}
//	{B:FF}                0xFF isolado
//	{END} {ESC} {TEXT_ITALIC} {TEXT_NORMAL} {LINE} {ART} {ART2}
//
// Tag desconhecida devolve false e o chamador faz o rewind do '{'.

// encodeTag devolve os bytes de uma tag (inner = conteúdo entre '{' e '}',
// sem as chaves).
func (ks *KeySet) encodeTag(inner string) ([]byte, bool) {
	head, rest, hasColon := strings.Cut(inner, ":")
	upper := strings.ToUpper(strings.TrimSpace(head))

	// Byte único isolado: {B:FF}.
	if upper == "B" {
		v, ok := parseHexByte(strings.TrimSpace(rest))
		return []byte{v}, ok
	}

	// Classe de par: o hex (quando existe) é AUTORITATIVO.
	if class, ok := tagClassOf(upper); ok {
		if class == ClassColor {
			// Única classe resolvida por nome.
			name := strings.TrimSpace(rest)
			tag, found := ks.tagByName(ClassColor, name)
			if !found {
				// Aceita qualquer caixa ("gold" -> "Gold").
				for _, t := range ks.Color {
					if strings.EqualFold(t.Name, name) {
						tag, found = t, true
						break
					}
				}
			}
			if !found {
				return nil, false
			}
			return []byte{tag.B1, tag.B2}, true
		}
		b1, b2, ok := hexPair(rest)
		if !ok {
			return nil, false
		}
		if _, known := ks.tagByPair(b1, b2); !known {
			// Par fora das tabelas: aceita (hex é autoritativo) mas avisa —
			// o decode seguinte emite {RAW:..} e a degradação fica visível.
			loggingService.Warn("tag %s: par %02X %02X não está nas tabelas do jogo",
				inner, b1, b2)
		}
		return []byte{b1, b2}, true
	}

	// Sem classe: single nomeado ({END}, {TEXT_ITALIC}...).
	if !hasColon {
		if b, ok := ks.EncodeByte("{" + upper + "}"); ok {
			return []byte{b}, true
		}
	}
	return nil, false
}

// EncodeTag é a face pública de encodeTag: os bytes de uma tag (sem as
// chaves). Serve para validar uma tag digitada/sugerida sem passar pela
// máquina de estados inteira — é o que o catálogo e o e2e usam para garantir
// que nada é sugerido sem codificar.
func (ks *KeySet) EncodeTag(inner string) ([]byte, bool) { return ks.encodeTag(inner) }

// closeBrace encontra o '}' de uma tag aberta em `open`, limitando o
// conteúdo interno a 32 runas (o mesmo limite do DeriveSymbolString do C#).
// Devolve o índice do '}' ou false quando não há fechamento no limite — o
// '{' vira literal e o conteúdo é reprocessado como texto.
func closeBrace(rs []rune, open int) (int, bool) {
	limit := open + 32
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
// ("F0:40:Clock" -> 0xF0, 0x40). O resto (o nome) é ignorado.
func hexPair(s string) (b1, b2 byte, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	b1, ok1 := parseHexByte(parts[0])
	b2, ok2 := parseHexByte(parts[1])
	return b1, b2, ok1 && ok2
}
