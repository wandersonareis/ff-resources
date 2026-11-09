package encoding

// Área "user-defined" do Shift-JIS/Windows-31J: 0xF040–0xF9FC mapeada
// linearmente para U+E000–U+E757 (Private Use Area). O jogo usa esses pares
// para glifos da fonte própria (ícones custom); o .NET Framework decodifica
// assim e o ZTRtool, consequentemente, também — os txt oráculos carregam
// essas runas. golang.org/x/text NÃO conhece a área, então o mapeamento é
// feito aqui, na máquina de estados.
//
// Os dicionários de tags têm prioridade: (0xF0,0x61) é {Icon Objective} no
// ff132 (dict do jogo) e "apenas" PUA no ff131 — a consulta aos dicts vem
// antes, como no C#.
//
// Big5/EUC-KR também têm áreas user-defined no .NET, mas sem arquivos de
// referência para validar deixamos sem mapeamento (o par segue cru).
const (
	sjisUserLeadLo  = 0xF0
	sjisUserLeadHi  = 0xF9
	sjisUserTrailLo = 0x40
	sjisUserTrailHi = 0xFC
	sjisUserRuneLo  = 0xE000
	sjisUserRuneHi  = 0xE757
	sjisUserWidth   = 188 // 0x40..0x7E + 0x80..0xFC (0x7F excluído)
)

// userDefinedPair devolve a runa PUA do par (b1,b2) na área user-defined
// do Shift-JIS, se houver.
func (ks *KeySet) userDefinedPair(b1, b2 byte) (rune, bool) {
	if ks.Codepage != LatinJapanese {
		return 0, false
	}
	if b1 < sjisUserLeadLo || b1 > sjisUserLeadHi || b2 == 0x7F {
		return 0, false
	}
	if b2 < sjisUserTrailLo || b2 > sjisUserTrailHi {
		return 0, false
	}
	off := int(b2) - sjisUserTrailLo
	if b2 > 0x7F {
		off-- // pula o 0x7F
	}
	return rune(sjisUserRuneLo + int(b1-sjisUserLeadLo)*sjisUserWidth + off), true
}

// userDefinedRune devolve o par user-defined de uma runa PUA.
func (ks *KeySet) userDefinedRune(r rune) (byte, byte, bool) {
	if ks.Codepage != LatinJapanese {
		return 0, 0, false
	}
	if r < sjisUserRuneLo || r > sjisUserRuneHi {
		return 0, 0, false
	}
	idx := int(r - sjisUserRuneLo)
	b1 := byte(sjisUserLeadLo + idx/sjisUserWidth)
	off := idx % sjisUserWidth
	b2 := byte(sjisUserTrailLo + off)
	if off >= 0x3F { // 0x7F não é trail: desloca a partir daqui
		b2++
	}
	return b1, b2, true
}
