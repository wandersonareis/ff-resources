package encoding

// IsMultibyteSecondByte responde se (b1,b2) é um par que forma um caractere
// multibyte do codepage — equivalente às funções *CharaCheck legadas (faixas
// em characheck_tables.go). O decoder usa isso para passar o 2º byte adiante
// sem tentar interpretá-lo como chave de controle.
func (tc *TextCodec) IsMultibyteSecondByte(b1, b2 byte) bool {
	var ranges [][4]byte
	switch tc.Codepage {
	case Chinese:
		ranges = Big5Ranges
	case Korean:
		ranges = EucKrRanges
	default:
		ranges = ShiftJISRanges
	}
	for _, r := range ranges {
		if b1 >= r[0] && b1 <= r[1] && b2 >= r[2] && b2 <= r[3] {
			return true
		}
	}
	return false
}
