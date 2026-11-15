package service

import (
	"ffresources/apps/ff13/backend/formats"
	"strings"
	"testing"
)

// encodeTagForTest codifica uma tag com o codec da part (LatinJapanese, o
// dos arquivos "_us") — espelho do que o editor faz ao validar um chip.
func encodeTagForTest(part, inner string) ([]byte, bool) {
	tc, err := formats.TextCodec(part)
	if err != nil {
		return nil, false
	}
	return tc.EncodeTag(inner)
}

// TestTagCatalogShape confere o catálogo servido ao editor: as classes do
// FF13, os valores por part e que todo valor sugerido codifica.
func TestTagCatalogShape(t *testing.T) {
	want := map[string]map[string]int{
		"part1": {"CLR": 43, "ICON": 54, "BTN": 33, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "SPACE": 1},
		"part2": {"CLR": 31, "ICON": 88, "BTN": 33, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "SPACE": 1},
		"part3": {"CLR": 31, "ICON": 90, "BTN": 35, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "SPACE": 1},
	}

	var svc FF13Service
	for _, part := range []string{"part1", "part2", "part3"} {
		cat, err := svc.GetTagCatalog(part, "us")
		if err != nil {
			t.Fatalf("%s: GetTagCatalog: %v", part, err)
		}
		if cat.Version != part || cat.Lang != "us" {
			t.Errorf("%s: version=%q lang=%q", part, cat.Version, cat.Lang)
		}

		got := map[string]int{}
		entries := map[string]tagCatalogEntry{}
		for _, e := range cat.Tags {
			got[e.Tag] = len(e.Values)
			entries[e.Tag] = e
			if e.Prefix != "{"+e.Tag+":" {
				t.Errorf("%s/%s: prefix = %q, queria %q", part, e.Tag, e.Prefix, "{"+e.Tag+":")
			}
			if e.Hint == "" {
				t.Errorf("%s/%s: sem hint", part, e.Tag)
			}
			for _, v := range e.Values {
				inner := v.Tag[1 : len(v.Tag)-1]
				if b, ok := encodeTagForTest(part, inner); !ok {
					t.Errorf("%s/%s: valor sugerido não codifica: %s", part, e.Tag, v.Tag)
				} else if len(b) == 0 {
					t.Errorf("%s/%s: zero bytes: %s", part, e.Tag, v.Tag)
				}
				if e.Tag == "CLR" && v.Hex == "" {
					t.Errorf("%s: cor sem swatch: %s", part, v.Tag)
				}
			}
		}
		for cls, n := range want[part] {
			if got[cls] != n {
				t.Errorf("%s/%s: %d valores, queria %d", part, cls, got[cls], n)
			}
		}

		if _, ok := entries["RAW"]; ok {
			t.Error("RAW não deveria ser sugerido")
		}
		if _, ok := entries["B"]; ok {
			t.Error("B não deveria ser sugerido")
		}
		if _, ok := entries["COLOR"]; ok {
			t.Error("COLOR não deveria existir; a classe é CLR")
		}
		if _, ok := entries["TAB"]; ok {
			t.Error("TAB não deveria existir; a classe é SPACE")
		}
		space := entries["SPACE"]
		if len(space.Values) != 1 || space.Values[0].Tag != "{SPACE:85:60}" {
			t.Errorf("%s: SPACE = %+v", part, space.Values)
		}
		// White é reset, não cor de paleta.
		white := false
		for _, v := range entries["CLR"].Values {
			if v.Label == "White" {
				white = v.Reset
			}
		}
		if !white {
			t.Errorf("%s: White deveria ter Reset=true", part)
		}
	}
}

// TestTagCatalogPartDependentValues marca o que muda por jogo: o MESMO nome
// de ícone em bytes diferentes e a paleta do ff131.
func TestTagCatalogPartDependentValues(t *testing.T) {
	var svc FF13Service
	cat1, _ := svc.GetTagCatalog("part1", "us")
	cat3, _ := svc.GetTagCatalog("part3", "us")

	icon1 := valuesOf(cat1, "ICON")
	icon3 := valuesOf(cat3, "ICON")
	if !hasTag(icon3, "{ICON:F2:5C:Shield}") {
		t.Error("ff133 deveria sugerir {ICON:F2:5C:Shield}")
	}
	if hasTag(icon1, "{ICON:F2:5C:Shield}") {
		t.Error("ff131 não deveria sugerir {ICON:F2:5C:Shield} (par do ff133)")
	}

	color1 := valuesOf(cat1, "CLR")
	if !hasTag(color1, "{CLR:F9:32:Ex00}") {
		t.Error("ff131 deveria expor os Ex (semi-transparentes)")
	}
	if hasTag(valuesOf(cat3, "CLR"), "{CLR:F9:32:Ex00}") {
		t.Error("ff133 não deveria expor Ex00 (paleta de 31)")
	}
	if !hasTag(color1, "{CLR:F9:40:White}") {
		t.Error("ff131 deveria sugerir {CLR:F9:40:White}")
	}
	// O grupo principal não inclui as Ex nem o White (reset).
	main := 0
	ex := 0
	reset := 0
	for _, v := range color1 {
		if v.Group == "ex" {
			ex++
			if !v.SemiTransparent {
				t.Errorf("%s deveria ser semitransparente", v.Tag)
			}
			continue
		}
		if v.Reset {
			reset++
			continue
		}
		main++
	}
	if main != 13 || ex != 29 || reset != 1 {
		t.Errorf("ff131 CLR: main=%d ex=%d reset=%d; queria 13/29/1", main, ex, reset)
	}

	if !hasTag(valuesOf(cat1, "VAR"), "{VAR:F4:40}") {
		t.Error("ff131 deveria sugerir {VAR:F4:40}")
	}
}

// TestTagCatalogLanguage fixa o espaço sem quebra por idioma: o mesmo par,
// mas o hint/label mudam entre us (U+00A0) e fr (U+202F).
func TestTagCatalogLanguage(t *testing.T) {
	var svc FF13Service
	us, err := svc.GetTagCatalog("part1", "us")
	if err != nil {
		t.Fatal(err)
	}
	fr, err := svc.GetTagCatalog("part1", "fr")
	if err != nil {
		t.Fatal(err)
	}
	usSpace := entryOf(us, "SPACE")
	frSpace := entryOf(fr, "SPACE")
	if !strings.Contains(usSpace.Hint, "U+00A0") {
		t.Errorf("us/SPACE hint = %q; queria U+00A0", usSpace.Hint)
	}
	if !strings.Contains(frSpace.Hint, "U+202F") {
		t.Errorf("fr/SPACE hint = %q; queria U+202F", frSpace.Hint)
	}
	if usSpace.Values[0].Tag != "{SPACE:85:60}" || frSpace.Values[0].Tag != "{SPACE:85:60}" {
		t.Errorf("SPACE deveria ter o mesmo par: us=%q fr=%q",
			usSpace.Values[0].Tag, frSpace.Values[0].Tag)
	}
	if usSpace.Values[0].Label != "NBSP" || frSpace.Values[0].Label != "NNBSP" {
		t.Errorf("label de SPACE: us=%q fr=%q", usSpace.Values[0].Label, frSpace.Values[0].Label)
	}

	// Cores: o mesmo nome, bytes do idioma (F9/FE/AC).
	for lang, want := range map[string]string{
		"us": "{CLR:F9:40:White}",
		"ch": "{CLR:FE:40:White}",
		"kr": "{CLR:AC:A1:White}",
		"gr": "{CLR:F9:40:White}",
	} {
		cat, err := svc.GetTagCatalog("part1", lang)
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if !hasTag(valuesOf(cat, "CLR"), want) {
			t.Errorf("%s: White = %s ausente", lang, want)
		}
	}
}

func TestTagCatalogPaletteError(t *testing.T) {
	var svc FF13Service
	if _, err := svc.GetTagCatalog("part1", "xx"); err == nil {
		t.Fatal("idioma desconhecido deveria falhar")
	} else if !strings.Contains(err.Error(), "paleta de cores indisponível para o idioma xx") {
		t.Errorf("erro = %q", err)
	}
}

func valuesOf(cat tagCatalog, tag string) []tagSuggestion {
	if e := entryOf(cat, tag); e != nil {
		return e.Values
	}
	return nil
}

func entryOf(cat tagCatalog, tag string) *tagCatalogEntry {
	for i := range cat.Tags {
		if cat.Tags[i].Tag == tag {
			return &cat.Tags[i]
		}
	}
	return nil
}

func hasTag(values []tagSuggestion, tag string) bool {
	for _, v := range values {
		if v.Tag == tag {
			return true
		}
	}
	return false
}
