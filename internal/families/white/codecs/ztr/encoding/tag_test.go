package encoding

import "testing"

// prepareAll monta os 9 conjuntos (3 games x 3 codepages) — as tabelas são
// publicadas em PrepareKeySet; sem isso KeySetFor falha.
func prepareAll(t *testing.T) {
	t.Helper()
	ClearKeySets()
	for _, g := range []GameCode{FF131, FF132, FF133} {
		for _, cp := range []Codepage{LatinJapanese, Chinese, Korean} {
			if err := PrepareKeySet(g, cp); err != nil {
				t.Fatalf("PrepareKeySet(%s, %d): %v", g, cp, err)
			}
		}
	}
}

// TestTagText cobre a FORMATAÇÃO de cada classe: COLOR só o nome, ICON/BTN
// hex + nome, as demais hex puro (sem nome).
func TestTagText(t *testing.T) {
	cases := []struct {
		tag  Tag
		want string
	}{
		{Tag{ClassColor, 0xF9, 0x40, "White"}, "{COLOR:White}"},
		{Tag{ClassColor, 0xAC, 0xA1, "White"}, "{COLOR:White}"},
		{Tag{ClassIcon, 0xF0, 0x40, "Clock"}, "{ICON:F0:40:Clock}"},
		{Tag{ClassBtn, 0xF1, 0x40, "A"}, "{BTN:F1:40:A}"},
		{Tag{ClassVar, 0xF4, 0x40, ""}, "{VAR:F4:40}"},
		{Tag{ClassCtr, 0xF7, 0x40, ""}, "{CTR:F7:40}"},
		{Tag{ClassKey, 0xF6, 0x40, ""}, "{KEY:F6:40}"},
		{Tag{ClassPage, 0x40, 0x70, ""}, "{PAGE:40:70}"},
		{Tag{ClassNL, 0x40, 0x72, ""}, "{NL:40:72}"},
		{Tag{ClassTab, 0x85, 0x60, ""}, "{TAB:85:60}"},
		{Tag{ClassRaw, 0x81, 0x40, ""}, "{RAW:81:40}"},
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

// TestEncodeTagByClass fecha a gramática do encode: o hex (quando existe) é
// AUTORITATIVO — o nome nem é lido.
func TestEncodeTagByClass(t *testing.T) {
	prepareAll(t)
	ks, err := KeySetFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	ok := []struct {
		inner string
		want  []byte
	}{
		// COLOR: nome resolve o byte.
		{"COLOR:White", []byte{0xF9, 0x40}},
		{"COLOR:Gold", []byte{0xF9, 0x42}},
		{"color:gold", []byte{0xF9, 0x42}}, // case-insensitive
		// ICON/BTN: hex autoritativo; o nome é ignorado (inclusive errado).
		{"ICON:F2:40:Sword", []byte{0xF2, 0x40}},
		{"ICON:F2:40:NomeQueNaoExiste", []byte{0xF2, 0x40}},
		{"ICON:F2:40", []byte{0xF2, 0x40}}, // sem nome, aceita
		{"BTN:F1:40:A", []byte{0xF1, 0x40}},
		// VAR/CTR/KEY: hex puro, sem nome — ex-{Entity 1}.
		{"VAR:F4:40", []byte{0xF4, 0x40}},
		{"VAR:F4:43", []byte{0xF4, 0x43}},
		{"CTR:F7:41", []byte{0xF7, 0x41}},
		{"KEY:F6:40", []byte{0xF6, 0x40}},
		// Estruturais.
		{"PAGE:40:70", []byte{0x40, 0x70}},
		{"NL:40:72", []byte{0x40, 0x72}},
		{"TAB:85:60", []byte{0x85, 0x60}},
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
		got, found := ks.encodeTag(c.inner)
		if !found {
			t.Errorf("encodeTag(%q) não casou", c.inner)
			continue
		}
		if string(got) != string(c.want) {
			t.Errorf("encodeTag(%q) = % X; queria % X", c.inner, got, c.want)
		}
	}

	// Desconhecidas: não casam (o chamador faz o rewind do '{').
	for _, inner := range []string{
		"COLOR:NoSuchColor", "ICON:ZZ:40:X", "ICON:F2", "VAR:F4",
		"ENTITY:1", "Color White", "PC:01", "FANTASMA",
	} {
		if got, found := ks.encodeTag(inner); found {
			t.Errorf("encodeTag(%q) casou = % X; queria false", inner, got)
		}
	}
}

// TestEncodeTagColorSharedProvaQueNome não é identidade para as classes de
// hex: nos ícones o MESMO nome existe em bytes diferentes por jogo, e o hex
// manda.
func TestIconNameDiffersAcrossGames(t *testing.T) {
	prepareAll(t)
	ks2, err := KeySetFor(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	ks3, err := KeySetFor(FF133, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}

	// "Shield" é F2:63 no ff132 e F2:5C no ff133: o hex do texto decide.
	got2, _ := ks2.encodeTag("ICON:F2:63:Shield")
	got3, _ := ks3.encodeTag("ICON:F2:63:Shield")
	if string(got2) != string(got3) || string(got2) != string([]byte{0xF2, 0x63}) {
		t.Errorf("hex autoritativo falhou: ff132=% X ff133=% X", got2, got3)
	}

	// O decode do par do ff133 devolve o nome dele (não o do ff132).
	tag, _, _, ok := ks3.resolveDecodePair(0xF2, 0x5C)
	if !ok || tag.Text() != "{ICON:F2:5C:Shield}" {
		t.Errorf("ff133 resolveDecodePair(F2,5C) = %+v, %v", tag, ok)
	}
}

// TestColorSharedAcrossCodepages prova o compartilhamento: um nome, três
// pares — a cópia de uma tradução entre linguagens continua correta.
func TestColorSharedAcrossCodepages(t *testing.T) {
	prepareAll(t)
	want := map[Codepage][2]byte{
		LatinJapanese: {0xF9, 0x40},
		Chinese:       {0xFE, 0x40},
		Korean:        {0xAC, 0xA1},
	}
	for cp, pair := range want {
		ks, err := KeySetFor(FF132, cp)
		if err != nil {
			t.Fatal(err)
		}
		got, found := ks.encodeTag("COLOR:White")
		if !found || got[0] != pair[0] || got[1] != pair[1] {
			t.Errorf("COLOR:White em %s = % X, %v; queria % X", cpName(cp), got, found, pair)
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
