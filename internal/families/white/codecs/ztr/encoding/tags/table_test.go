package tags

import "testing"

func TestForPerLanguage(t *testing.T) {
	for _, g := range []GameCode{FF131, FF132, FF133} {
		for _, cp := range []Codepage{LatinJapanese, Chinese, Korean} {
			table, err := For(g, cp)
			if err != nil {
				t.Fatalf("For(%s, %d): %v", g, cp, err)
			}
			if len(table.Colors) == 0 || len(table.Icons) == 0 || len(table.Buttons) == 0 {
				t.Errorf("%s/%d: tabela vazia (colors=%d icons=%d buttons=%d)",
					g, cp, len(table.Colors), len(table.Icons), len(table.Buttons))
			}
			if len(table.Controls) != 6 || len(table.Glyphs) != 147 {
				t.Errorf("%s/%d: controls=%d glyphs=%d; queria 6/147",
					g, cp, len(table.Controls), len(table.Glyphs))
			}
		}
	}
}

func TestColorBytesPerLanguage(t *testing.T) {
	want := map[Codepage][2]byte{
		LatinJapanese: {0xF9, 0x40},
		Chinese:       {0xFE, 0x40},
		Korean:        {0xAC, 0xA1},
	}
	for cp, pair := range want {
		table, err := For(FF132, cp)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, c := range table.Colors {
			if c.Name == "White" {
				found = true
				if c.B1 != pair[0] || c.B2 != pair[1] {
					t.Errorf("White em %d = %02X %02X; queria %02X %02X",
						cp, c.B1, c.B2, pair[0], pair[1])
				}
				if c.Text() != "{CLR:"+hex(c.B1)+":"+hex(c.B2)+":White}" {
					t.Errorf("Text(White) = %q", c.Text())
				}
			}
		}
		if !found {
			t.Errorf("White ausente em %d", cp)
		}
	}
}

func TestFF133SharesFF132Palette(t *testing.T) {
	t2, err := For(FF132, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	t3, err := For(FF133, LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	if len(t2.Colors) != len(t3.Colors) {
		t.Fatalf("ff132=%d ff133=%d", len(t2.Colors), len(t3.Colors))
	}
	for i := range t2.Colors {
		if t2.Colors[i].Text() != t3.Colors[i].Text() {
			t.Errorf("cor %d difere: %q vs %q", i, t2.Colors[i].Text(), t3.Colors[i].Text())
		}
	}
}

func TestChineseHasNoRawExtra(t *testing.T) {
	table, err := For(FF131, Chinese)
	if err != nil {
		t.Fatal(err)
	}
	if table.RawExtra != nil {
		t.Error("chinês deveria ter RawExtra nil")
	}
}

func TestUnknown(t *testing.T) {
	if _, err := For("ff139", LatinJapanese); err == nil {
		t.Error("game desconhecido deveria falhar")
	}
	if _, err := For(FF131, Codepage(1)); err == nil {
		t.Error("codepage desconhecida deveria falhar")
	}
}

func hex(b byte) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[b>>4], digits[b&0x0F]})
}
