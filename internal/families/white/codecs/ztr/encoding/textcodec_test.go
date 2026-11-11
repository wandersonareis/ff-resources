package encoding

import "testing"

// Monta os 9 conjuntos (3 games x 3 codepages) e confere amostras de lookup
// contra as tabelas de referência, incluindo a composição por idioma.
func TestPrepareTextCodec(t *testing.T) {
	ClearTextCodecs()
	for _, g := range []GameCode{FF131, FF132, FF133} {
		for _, cp := range []Codepage{LatinJapanese, Chinese, Korean} {
			if err := PrepareTextCodec(g, cp); err != nil {
				t.Fatalf("PrepareTextCodec(%s, %d): %v", g, cp, err)
			}
			if err := EnsureTextCodec(g, cp); err != nil {
				t.Fatalf("EnsureTextCodec(%s, %d): %v", g, cp, err)
			}
		}
	}

	// Fail-fast: configuração inexistente.
	if err := EnsureTextCodec("ff139", LatinJapanese); err == nil {
		t.Fatal("EnsureTextCodec com game inválido deveria falhar")
	}

	tc, err := TextCodecFor(FF131, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := tc.DecodeControl(0x00); !ok || v != "{END}" {
		t.Errorf("DecodeControl(0x00) = %q, %v; queria {END}", v, ok)
	}
	if b, ok := tc.EncodeControl("{END}"); !ok || b != 0x00 {
		t.Errorf("EncodeControl({END}) = %v, %v; queria 0", b, ok)
	}

	// Cor ff131 lj: (0xF9,0x32) -> {CLR:F9:32:Ex00}.
	tag, r, isRune, ok := tc.resolveDecodePair(0xF9, 0x32)
	if !ok || isRune || tag.Class != ClassColor || tag.Name != "Ex00" ||
		tag.Text() != "{CLR:F9:32:Ex00}" {
		t.Errorf("resolveDecodePair(F9,32) = %+v,%v,%v,%v; queria CLR:F9:32:Ex00", tag, r, isRune, ok)
	}
	if b, ok := tc.encodeTag("CLR:F9:32:Ex00"); !ok || b[0] != 0xF9 || b[1] != 0x32 {
		t.Errorf("encodeTag(CLR:F9:32:Ex00) = % X, %v; queria F9 32", b, ok)
	}
	// O NOME não resolve cor: só os dois bytes.
	if b, ok := tc.encodeTag("CLR:White"); ok {
		t.Errorf("encodeTag(CLR:White) casou = % X; queria false (faltam os bytes)", b)
	}
	if b, ok := tc.encodeTag("CLR:F9:40:White"); !ok || b[0] != 0xF9 || b[1] != 0x40 {
		t.Errorf("encodeTag(CLR:F9:40:White) = % X, %v; queria F9 40", b, ok)
	}

	// Glifos (novo formato, sem chaves): par 0x85 0x40 -> '€'; e o reverse
	// devolve a PRIMEIRA ocorrência.
	if tag, r, isRune, ok := tc.resolveDecodePair(0x85, 0x40); !ok || !isRune || r != '€' {
		t.Errorf("resolveDecodePair(85,40) = %q,%U,%v,%v; queria runa € (U+20AC)", tag, r, isRune, ok)
	}
	if b1, b2, ok := tc.GlyphBytes('€'); !ok || b1 != 0x85 || b2 != 0x40 {
		t.Errorf("GlyphBytes(€) = %02X %02X, %v; queria 85 40", b1, b2, ok)
	}
	// "—" é o glifo 0x85 0x57 (U+2014), não o 0x81 0x5C do charset.
	if b1, b2, ok := tc.GlyphBytes('—'); !ok || b1 != 0x85 || b2 != 0x57 {
		t.Errorf("GlyphBytes(—) = %02X %02X, %v; queria 85 57", b1, b2, ok)
	}
	// '-' ASCII NÃO resolve como glifo (vai como 0x2D no encoder).
	if _, _, ok := tc.GlyphBytes('-'); ok {
		t.Error("GlyphBytes('-') deveria falhar; '-' é ASCII e vira 0x2D")
	}
	if b1, b2, ok := tc.GlyphBytes('\u2010'); !ok || b1 != 0x85 || b2 != 0x56 {
		t.Errorf("GlyphBytes(U+2010) = %02X %02X, %v; queria 85 56", b1, b2, ok)
	}
	if tag, r, isRune, ok := tc.resolveDecodePair(0x85, 0x56); !ok || !isRune || r != '\u2010' {
		t.Errorf("resolveDecodePair(85,56) = %q,%U,%v,%v; queria runa U+2010", tag, r, isRune, ok)
	}

	// Área user-defined (PUA) do Shift-JIS.
	if r, ok := tc.userDefinedPair(0xF0, 0x61); !ok || r != 0xE021 {
		t.Errorf("userDefinedPair(F0,61) = %U, %v; queria U+E021", r, ok)
	}
	if b1, b2, ok := tc.userDefinedRune(0xE021); !ok || b1 != 0xF0 || b2 != 0x61 {
		t.Errorf("userDefinedRune(U+E021) = %02X %02X, %v; queria F0 61", b1, b2, ok)
	}
	if r, ok := tc.userDefinedPair(0xF7, 0x43); !ok || r != 0xE527 {
		t.Errorf("userDefinedPair(F7,43) = %U, %v; queria U+E527", r, ok)
	}
	if r, ok := tc.userDefinedPair(0xF0, 0x7F); ok {
		t.Errorf("userDefinedPair(F0,7F) = %U; 0x7F não é trail", r)
	}

	// Composição do special.
	if tag, _, _, ok := tc.resolveDecodePair(0x40, 0x70); !ok || tag.Text() != "{PAGE:40:70}" {
		t.Errorf("resolveDecodePair(40,70) = %+v, %v; queria {PAGE:40:70}", tag, ok)
	}
	if tag, _, _, ok := tc.resolveDecodePair(0xF4, 0x40); !ok || tag.Text() != "{VAR:F4:40}" {
		t.Errorf("resolveDecodePair(F4,40) = %+v, %v; queria {VAR:F4:40}", tag, ok)
	}
	if b, ok := tc.encodeTag("VAR:F4:40"); !ok || b[0] != 0xF4 || b[1] != 0x40 {
		t.Errorf("encodeTag(VAR:F4:40) = % X, %v; queria F4 40", b, ok)
	}
	// O par do espaço sem quebra não é tabulação.
	if tag, _, _, ok := tc.resolveDecodePair(0x85, 0x60); !ok || tag.Text() != "{SPACE:85:60}" {
		t.Errorf("resolveDecodePair(85,60) = %+v, %v; queria {SPACE:85:60}", tag, ok)
	}

	// O MESMO nome de ícone existe em bytes diferentes por jogo: o hex é
	// autoritativo (Icon Shield = F2:63 no ff132 e F2:5C no ff133).
	tc3, err := TextCodecFor(FF133, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	if t3, _, _, ok := tc3.resolveDecodePair(0xF2, 0x5C); !ok || t3.Text() != "{ICON:F2:5C:Shield}" {
		t.Errorf("resolveDecodePair(F2,5C) ff133 = %+v, %v; queria {ICON:F2:5C:Shield}", t3, ok)
	}

	// Contagens da composição.
	if len(tc.Glyphs) != 118 {
		t.Errorf("len(Glyphs) = %d; queria 118", len(tc.Glyphs))
	}
	if len(tc.Colors) != 43 || len(tc.Icons) != 54 || len(tc.Buttons) != 33 {
		t.Errorf("ff131 lj: colors=%d icons=%d buttons=%d; queria 43/54/33",
			len(tc.Colors), len(tc.Icons), len(tc.Buttons))
	}
	if len(tc.Specials) != 11 {
		t.Errorf("len(Specials lj) = %d; queria 11", len(tc.Specials))
	}
	if len(tc.Raw) != 60 || len(tc.RawExtra) != 10 {
		t.Errorf("ff131 lj: raw=%d rawExtra=%d; queria 60/10", len(tc.Raw), len(tc.RawExtra))
	}

	// Ch não consulta o segundo conjunto de pares crus.
	tcCh, err := TextCodecFor(FF131, Chinese)
	if err != nil {
		t.Fatal(err)
	}
	if tcCh.RawExtra != nil {
		t.Error("TextCodec ch deveria ter RawExtra nil")
	}
	if len(tcCh.Specials) != 10 {
		t.Errorf("len(Specials ch) = %d; queria 10", len(tcCh.Specials))
	}
	if b1, b2, ok := tcCh.GlyphBytes('€'); !ok || b1 != 0x80 || b2 != 0x40 {
		t.Errorf("GlyphBytes(€) ch = %02X %02X, %v; queria 80 40", b1, b2, ok)
	}

	// Kr: lead 0x80 também; a paleta KR tem trails próprios.
	tcKr, err := TextCodecFor(FF131, Korean)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range tcKr.Colors {
		if e.B1 == 0xAC && e.B2 == 0x93 && e.Class == ClassColor && e.Name == "Ex00" {
			found = true
		}
	}
	if !found {
		t.Error("cor KR ff131 deveria ter (AC,93) -> CLR:AC:93:Ex00")
	}
}

// TestDecodeEncodeLine exercita a máquina de estados com casos que cobrem
// todas as ramificações do decoder e do encoder.
func TestDecodeEncodeLine(t *testing.T) {
	tc, err := TextCodecFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		raw  []byte
		text string
	}{
		{[]byte("Hello world\x00\x00"), "Hello world"},
		{[]byte("\x00\x00"), ""},
		{[]byte("A\x00B\x00\x00"), "A{END}B"},
		{[]byte("\x02italico\x00\x00"), "{TEXT_ITALIC}italico"},
		{[]byte{0xF9, 0x40, 'X', 0x85, 0x57, 'Y', 0x00, 0x00},
			"{CLR:F9:40:White}X—Y"},
		{[]byte{0x85, 0x60, 'X', 0x00, 0x00}, "{SPACE:85:60}X"},
		{[]byte{0xFF, 'a', 0x00, 0x00}, "{B:FF}a"},
		{[]byte{0x82, 0x60, 0x00, 0x00}, "Ａ"},
		{[]byte{0x82, 0x4F, 0x00, 0x00}, "０"},
	}

	for _, c := range cases {
		got, err := tc.DecodeLine(c.raw)
		if err != nil {
			t.Fatalf("DecodeLine(% x) erro: %v", c.raw, err)
		}
		if got != c.text {
			t.Errorf("DecodeLine(% x) = %q; queria %q", c.raw, got, c.text)
		}
		back, err := tc.EncodeLine(got)
		if err != nil {
			t.Fatalf("EncodeLine(%q) erro: %v", got, err)
		}
		if string(back) != string(c.raw) {
			t.Errorf("EncodeLine(%q) = % x; queria % x (round-trip)", got, back, c.raw)
		}
	}
}
