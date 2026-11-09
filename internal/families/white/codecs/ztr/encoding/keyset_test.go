package encoding

import "testing"

// Monta os 9 conjuntos (3 games x 3 codepages) e confere amostras de lookup
// contra os fontes C# (KeyDicts*), incluindo a composição comum+variantes.
func TestPrepareKeySet(t *testing.T) {
	ClearKeySets()
	for _, g := range []GameCode{FF131, FF132, FF133} {
		for _, cp := range []Codepage{LatinJapanese, Chinese, Korean} {
			if err := PrepareKeySet(g, cp); err != nil {
				t.Fatalf("PrepareKeySet(%s, %d): %v", g, cp, err)
			}
			if err := EnsureKeySet(g, cp); err != nil {
				t.Fatalf("EnsureKeySet(%s, %d): %v", g, cp, err)
			}
		}
	}

	// Fail-fast: configuração inexistente.
	if err := EnsureKeySet("ff139", LatinJapanese); err == nil {
		t.Fatal("EnsureKeySet com game inválido deveria falhar")
	}

	ks, err := KeySetFor(FF131, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := ks.DecodeByte(0x00); !ok || v != "{End}" {
		t.Errorf("DecodeByte(0x00) = %q, %v; queria {End}", v, ok)
	}
	if b, ok := ks.EncodeByte("{End}"); !ok || b != 0x00 {
		t.Errorf("EncodeByte({End}) = %v, %v; queria 0", b, ok)
	}

	// ColorKeys ff131 lj: (0xF9,0x32) -> "{Color Ex00}".
	tag, r, isRune, ok := ks.resolveDecodePair(0xF9, 0x32)
	if !ok || isRune || tag != "{Color Ex00}" {
		t.Errorf("resolveDecodePair(F9,32) = %q,%v,%v,%v; queria {Color Ex00}", tag, r, isRune, ok)
	}
	if b1, b2, ok := ks.resolveEncodeTag("{Color Ex00}"); !ok || b1 != 0xF9 || b2 != 0x32 {
		t.Errorf("resolveEncodeTag({Color Ex00}) = %02X %02X, %v; queria F9 32", b1, b2, ok)
	}

	// Chara (novo formato, sem chaves): par 0x85 0x40 -> '€'; e o reverse
	// devolve a PRIMEIRA ocorrência (mesma runa em duas entradas não
	// ambigua porque o índice guarda a primeira).
	if tag, r, isRune, ok := ks.resolveDecodePair(0x85, 0x40); !ok || !isRune || r != '€' {
		t.Errorf("resolveDecodePair(85,40) = %q,%U,%v,%v; queria runa € (U+20AC)", tag, r, isRune, ok)
	}
	if b1, b2, ok := ks.CharaPair('€'); !ok || b1 != 0x85 || b2 != 0x40 {
		t.Errorf("CharaPair(€) = %02X %02X, %v; queria 85 40", b1, b2, ok)
	}
	// "—" é o chara 0x85 0x57 (U+2014), não o 0x81 0x5C do charset.
	if b1, b2, ok := ks.CharaPair('—'); !ok || b1 != 0x85 || b2 != 0x57 {
		t.Errorf("CharaPair(—) = %02X %02X, %v; queria 85 57", b1, b2, ok)
	}
	// '-' ASCII NÃO resolve como chara (vai como 0x2D no encoder); o chara
	// hífen (0x85 0x56) decodifica como o duplo tipográfico U+2010.
	if _, _, ok := ks.CharaPair('-'); ok {
		t.Error("CharaPair('-') deveria falhar; '-' é ASCII e vira 0x2D")
	}
	if b1, b2, ok := ks.CharaPair('\u2010'); !ok || b1 != 0x85 || b2 != 0x56 {
		t.Errorf("CharaPair(U+2010) = %02X %02X, %v; queria 85 56", b1, b2, ok)
	}
	if tag, r, isRune, ok := ks.resolveDecodePair(0x85, 0x56); !ok || !isRune || r != '\u2010' {
		t.Errorf("resolveDecodePair(85,56) = %q,%U,%v,%v; queria runa U+2010", tag, r, isRune, ok)
	}

	// Área user-defined (PUA) do Shift-JIS: (0xF0,0x61)->U+E021,
	// (0xF7,0x43)->U+E527 — depois dos dicts de tags.
	if r, ok := ks.userDefinedPair(0xF0, 0x61); !ok || r != 0xE021 {
		t.Errorf("userDefinedPair(F0,61) = %U, %v; queria U+E021", r, ok)
	}
	if b1, b2, ok := ks.userDefinedRune(0xE021); !ok || b1 != 0xF0 || b2 != 0x61 {
		t.Errorf("userDefinedRune(U+E021) = %02X %02X, %v; queria F0 61", b1, b2, ok)
	}
	if r, ok := ks.userDefinedPair(0xF7, 0x43); !ok || r != 0xE527 {
		t.Errorf("userDefinedPair(F7,43) = %U, %v; queria U+E527", r, ok)
	}
	if r, ok := ks.userDefinedPair(0xF0, 0x7F); ok {
		t.Errorf("userDefinedPair(F0,7F) = %U; 0x7F não é trail", r)
	}

	// Composição do special: base comum + sufixo LJ.
	// (0x40,0x70) -> {TEXT_NEW_PAGE}; (0xF4,0x40) -> {Entity 1}.
	if tag, _, _, ok := ks.resolveDecodePair(0x40, 0x70); !ok || tag != "{TEXT_NEW_PAGE}" {
		t.Errorf("resolveDecodePair(40,70) = %q, %v; queria {TEXT_NEW_PAGE}", tag, ok)
	}
	if tag, _, _, ok := ks.resolveDecodePair(0xF4, 0x40); !ok || tag != "{Entity 1}" {
		t.Errorf("resolveDecodePair(F4,40) = %q, %v; queria {Entity 1}", tag, ok)
	}

	// Contagens da composição (análise dos fontes C#).
	if len(ks.Chara) != 118 {
		t.Errorf("len(Chara) = %d; queria 118", len(ks.Chara))
	}
	if len(ks.Color) != 43 || len(ks.Icon) != 54 || len(ks.Btn) != 33 {
		t.Errorf("ff131 lj: color=%d icon=%d btn=%d; queria 43/54/33",
			len(ks.Color), len(ks.Icon), len(ks.Btn))
	}
	if len(ks.Special) != 11 {
		t.Errorf("len(Special lj) = %d; queria 11", len(ks.Special))
	}

	// Ch não usa Unk2Keys (nulo no KeySet — igual ao C#).
	ksCh, err := KeySetFor(FF131, Chinese)
	if err != nil {
		t.Fatal(err)
	}
	if ksCh.Unk2 != nil {
		t.Error("KeySet ch deveria ter Unk2 nil")
	}
	if len(ksCh.Special) != 10 {
		t.Errorf("len(Special ch) = %d; queria 10", len(ksCh.Special))
	}
	// Chara no Big5: lead 0x80 + trails comuns.
	if b1, b2, ok := ksCh.CharaPair('€'); !ok || b1 != 0x80 || b2 != 0x40 {
		t.Errorf("CharaPair(€) ch = %02X %02X, %v; queria 80 40", b1, b2, ok)
	}

	// Kr: lead 0x80 também; color KR tem trails próprios (0x93+ no xiii).
	ksKr, err := KeySetFor(FF131, Korean)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range ksKr.Color {
		if e.B1 == 0xAC && e.B2 == 0x93 && e.Value == "{Color Ex00}" {
			found = true
		}
	}
	if !found {
		t.Error("color KR ff131 deveria ter (AC,93) -> {Color Ex00}")
	}
}

// TestDecodeEncodeLine exercita a máquina de estados com casos que cobrem
// todas as ramificações do decoder e do encoder.
func TestDecodeEncodeLine(t *testing.T) {
	ks, err := KeySetFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		raw  []byte
		text string
	}{
		// ASCII puro + terminador.
		{[]byte("Hello world\x00\x00"), "Hello world"},
		// terminador isolado: consome 00 00 em silêncio.
		{[]byte("\x00\x00"), ""},
		// {End}: 00 isolado seguido de byte não-zero.
		{[]byte("A\x00B\x00\x00"), "A{End}B"},
		// tag single {TEXT_ITALIC}.
		{[]byte("\x02italico\x00\x00"), "{TEXT_ITALIC}italico"},
		// color + chara + texto.
		{[]byte{0xF9, 0x40, 'X', 0x85, 0x57, 'Y', 0x00, 0x00},
			"{Color White}X—Y"},
		// {FF}.
		{[]byte{0xFF, 'a', 0x00, 0x00}, "{FF}a"},
		// katakana/fullwidth (multibyte do charset): 0x82 0x60 = 'Ａ'.
		{[]byte{0x82, 0x60, 0x00, 0x00}, "Ａ"},
		// multibyte cujo trail é 0x40..0x7e (não pode virar tag).
		{[]byte{0x82, 0x4F, 0x00, 0x00}, "０"},
	}

	for _, c := range cases {
		got, err := ks.DecodeLine(c.raw)
		if err != nil {
			t.Fatalf("DecodeLine(% x) erro: %v", c.raw, err)
		}
		if got != c.text {
			t.Errorf("DecodeLine(% x) = %q; queria %q", c.raw, got, c.text)
		}
		back, err := ks.EncodeLine(got)
		if err != nil {
			t.Fatalf("EncodeLine(%q) erro: %v", got, err)
		}
		if string(back) != string(c.raw) {
			t.Errorf("EncodeLine(%q) = % x; queria % x (round-trip)", got, back, c.raw)
		}
	}
}
