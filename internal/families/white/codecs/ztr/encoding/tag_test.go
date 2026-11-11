package encoding

import "testing"

// prepareAll monta os 9 conjuntos (3 games x 3 codepages).
func prepareAll(t *testing.T) {
	t.Helper()
	ClearTextCodecs()
	for _, g := range []GameCode{FF131, FF132, FF133} {
		for _, cp := range []Codepage{LatinJapanese, Chinese, Korean} {
			if err := PrepareTextCodec(g, cp); err != nil {
				t.Fatalf("PrepareTextCodec(%s, %d): %v", g, cp, err)
			}
		}
	}
}

// TestTagText cobre a FORMATAÇÃO: o formato é o mesmo para todas as classes
// de par; o nome só aparece quando existe.
func TestTagText(t *testing.T) {
	cases := []struct {
		tag  Tag
		want string
	}{
		{Tag{Class: ClassColor, B1: 0xF9, B2: 0x40, Name: "White"}, "{CLR:F9:40:White}"},
		{Tag{Class: ClassColor, B1: 0xAC, B2: 0xA1, Name: "White"}, "{CLR:AC:A1:White}"},
		{Tag{Class: ClassIcon, B1: 0xF0, B2: 0x40, Name: "Clock"}, "{ICON:F0:40:Clock}"},
		{Tag{Class: ClassBtn, B1: 0xF1, B2: 0x40, Name: "A"}, "{BTN:F1:40:A}"},
		{Tag{Class: ClassSpace, B1: 0x85, B2: 0x60}, "{SPACE:85:60}"},
		{Tag{Class: ClassVar, B1: 0xF4, B2: 0x40}, "{VAR:F4:40}"},
		{Tag{Class: ClassCtr, B1: 0xF7, B2: 0x40}, "{CTR:F7:40}"},
		{Tag{Class: ClassKey, B1: 0xF6, B2: 0x40}, "{KEY:F6:40}"},
		{Tag{Class: ClassPage, B1: 0x40, B2: 0x70}, "{PAGE:40:70}"},
		{Tag{Class: ClassNL, B1: 0x40, B2: 0x72}, "{NL:40:72}"},
		{Tag{Class: ClassRaw, B1: 0x81, B2: 0x40}, "{RAW:81:40}"},
	}
	for _, c := range cases {
		if got := c.tag.Text(); got != c.want {
			t.Errorf("Text(%+v) = %q; queria %q", c.tag, got, c.want)
		}
		if got := string(c.tag.AppendText(nil)); got != c.want {
			t.Errorf("AppendText(%+v) = %q; queria %q", c.tag, got, c.want)
		}
	}
}

// TestEncodeTagByClass fecha a gramática do encode: o hex é AUTORITATIVO e
// o nome é ignorado. Cor sem os DOIS bytes não codifica.
func TestEncodeTagByClass(t *testing.T) {
	prepareAll(t)
	tc, err := TextCodecFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	ok := []struct {
		inner string
		want  []byte
	}{
		// CLR: os bytes do idioma mandam; o nome é display.
		{"CLR:F9:40:White", []byte{0xF9, 0x40}},
		{"CLR:F9:42:Gold", []byte{0xF9, 0x42}},
		{"clr:f9:42:gold", []byte{0xF9, 0x42}}, // case-insensitive
		{"CLR:F9:42", []byte{0xF9, 0x42}},      // sem nome, aceita
		// ICON/BTN: hex autoritativo; o nome é ignorado (inclusive errado).
		{"ICON:F2:40:Sword", []byte{0xF2, 0x40}},
		{"ICON:F2:40:NomeQueNaoExiste", []byte{0xF2, 0x40}},
		{"ICON:F2:40", []byte{0xF2, 0x40}},
		{"BTN:F1:40:A", []byte{0xF1, 0x40}},
		// VAR/CTR/KEY: hex puro, sem nome.
		{"VAR:F4:40", []byte{0xF4, 0x40}},
		{"VAR:F4:43", []byte{0xF4, 0x43}},
		{"CTR:F7:41", []byte{0xF7, 0x41}},
		{"KEY:F6:40", []byte{0xF6, 0x40}},
		// Estruturais. SPACE é espaço sem quebra, não tabulação.
		{"PAGE:40:70", []byte{0x40, 0x70}},
		{"NL:40:72", []byte{0x40, 0x72}},
		{"SPACE:85:60", []byte{0x85, 0x60}},
		// Bytes crus.
		{"RAW:81:40", []byte{0x81, 0x40}},
		{"RAW:FF:82", []byte{0xFF, 0x82}},
		{"B:FF", []byte{0xFF}},
		// Singles.
		{"END", []byte{0x00}},
		{"ESC", []byte{0x01}},
		{"TEXT_ITALIC", []byte{0x02}},
		{"LINE", []byte{0x03}},
		{"ART", []byte{0x04}},
		{"ART2", []byte{0x05}},
	}
	for _, c := range ok {
		got, found := tc.encodeTag(c.inner)
		if !found {
			t.Errorf("encodeTag(%q) não casou", c.inner)
			continue
		}
		if string(got) != string(c.want) {
			t.Errorf("encodeTag(%q) = % X; queria % X", c.inner, got, c.want)
		}
	}

	// Desconhecidas/legado: não casam (o chamador faz o rewind do '{').
	for _, inner := range []string{
		"CLR:White", "CLR:NoSuchColor", "COLOR:White", "TAB:85:60",
		"ICON:ZZ:40:X", "ICON:F2", "VAR:F4", "ENTITY:1", "Color White",
		"PC:01", "FANTASMA",
	} {
		if got, found := tc.encodeTag(inner); found {
			t.Errorf("encodeTag(%q) casou = % X; queria false", inner, got)
		}
	}
}

// TestColorBytesAreIdentity prova que o nome não é identidade da cor: o hex
// decide, e o mesmo nome tem bytes diferentes por codepage.
func TestColorBytesAreIdentity(t *testing.T) {
	prepareAll(t)
	tc, err := TextCodecFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tc.encodeTag("CLR:F9:7F:QualquerNome")
	if string(got) != string([]byte{0xF9, 0x7F}) {
		t.Errorf("hex autoritativo falhou: % X", got)
	}
}

// TestIconNameDiffersAcrossGames prova que o hex do texto decide o byte,
// mesmo quando o nome do mesmo par é outro no jogo.
func TestIconNameDiffersAcrossGames(t *testing.T) {
	prepareAll(t)
	tc2, err := TextCodecFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	tc3, err := TextCodecFor(FF133, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	got2, _ := tc2.encodeTag("ICON:F2:63:Shield")
	got3, _ := tc3.encodeTag("ICON:F2:63:Shield")
	if string(got2) != string(got3) || string(got2) != string([]byte{0xF2, 0x63}) {
		t.Errorf("hex autoritativo falhou: ff132=% X ff133=% X", got2, got3)
	}

	tag, _, _, ok := tc3.resolveDecodePair(0xF2, 0x5C)
	if !ok || tag.Text() != "{ICON:F2:5C:Shield}" {
		t.Errorf("ff133 resolveDecodePair(F2,5C) = %+v, %v", tag, ok)
	}
}

// TestColorSharedAcrossCodepages prova que cada idioma tem o SEU par para o
// mesmo nome.
func TestColorSharedAcrossCodepages(t *testing.T) {
	prepareAll(t)
	want := map[Codepage][2]byte{
		LatinJapanese: {0xF9, 0x40},
		Chinese:       {0xFE, 0x40},
		Korean:        {0xAC, 0xA1},
	}
	for cp, pair := range want {
		tc, err := TextCodecFor(FF132, cp)
		if err != nil {
			t.Fatal(err)
		}
		got, found := tc.encodeTag("CLR:" + hexByte(pair[0]) + ":" + hexByte(pair[1]) + ":White")
		if !found || got[0] != pair[0] || got[1] != pair[1] {
			t.Errorf("CLR White em %s = % X, %v; queria % X", cpName(cp), got, found, pair)
		}
	}
}

func cpName(cp Codepage) string {
	switch cp {
	case LatinJapanese:
		return "LatinJapanese"
	case Chinese:
		return "Chinese"
	case Korean:
		return "Korean"
	}
	return "?"
}

func hexByte(b byte) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[b>>4], digits[b&0x0F]})
}
