package main

import (
	"testing"
)

// encodeTagForTest codifica uma tag com o keyset da part (LatinJapanese, o
// dos arquivos "_us") — espelho do que o editor faz ao validar um chip.
func encodeTagForTest(part, inner string) ([]byte, bool) {
	ks, err := keySet(part)
	if err != nil {
		return nil, false
	}
	return ks.EncodeTag(inner)
}

// TestTagCatalogShape confere o catálogo servido ao editor: as classes do
// FF13, os valores por part e que todo valor sugerido codifica (o editor não
// pode oferecer tag que o encode não aceite — viraria texto literal).
func TestTagCatalogShape(t *testing.T) {
	want := map[string]map[string]int{
		"part1": {"COLOR": 43, "ICON": 54, "BTN": 33, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "TAB": 1},
		"part2": {"COLOR": 31, "ICON": 88, "BTN": 33, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "TAB": 1},
		"part3": {"COLOR": 31, "ICON": 90, "BTN": 35, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "TAB": 1},
	}

	var svc FF13Service
	for _, part := range []string{"part1", "part2", "part3"} {
		cat, err := svc.GetTagCatalog(part)
		if err != nil {
			t.Fatalf("%s: GetTagCatalog: %v", part, err)
		}
		if cat.Version != part {
			t.Errorf("%s: version = %q", part, cat.Version)
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
				t.Errorf("%s/%s: sem hint (o popup do estágio 1 mostraria vazio)", part, e.Tag)
			}
			for _, v := range e.Values {
				inner := v.Tag[1 : len(v.Tag)-1]
				if b, ok := encodeTagForTest(part, inner); !ok {
					t.Errorf("%s/%s: valor sugerido não codifica: %s", part, e.Tag, v.Tag)
				} else if len(b) == 0 {
					t.Errorf("%s/%s: zero bytes: %s", part, e.Tag, v.Tag)
				}
			}
		}
		for cls, n := range want[part] {
			if got[cls] != n {
				t.Errorf("%s/%s: %d valores, queria %d", part, cls, got[cls], n)
			}
		}

		// RAW e B NÃO entram: são bytes crus de slots desconhecidos.
		if _, ok := entries["RAW"]; ok {
			t.Error("RAW não deveria ser sugerido (byte cru de slot desconhecido)")
		}
		if _, ok := entries["B"]; ok {
			t.Error("B não deveria ser sugerido")
		}
	}
}

// TestTagCatalogPartDependentValues marca o que muda por jogo: o MESMO nome
// de ícone em bytes diferentes (Icon Shield = F2:63 no ff132, F2:5C no ff133)
// e a paleta (ff131 tem os EXTRA que o ZTRtool deixou sem nome).
func TestTagCatalogPartDependentValues(t *testing.T) {
	var svc FF13Service

	cat1, _ := svc.GetTagCatalog("part1")
	cat3, _ := svc.GetTagCatalog("part3")

	icon1 := valuesOf(cat1, "ICON")
	icon3 := valuesOf(cat3, "ICON")
	if hasTag(icon1, "{ICON:F2:63:Shield}") && !hasTag(icon3, "{ICON:F2:63:Shield}") {
		t.Log("ff131 tem Shield em F2:63? (conferir composição)")
	}
	if !hasTag(icon3, "{ICON:F2:5C:Shield}") {
		t.Error("ff133 deveria sugerir {ICON:F2:5C:Shield}")
	}
	if hasTag(icon1, "{ICON:F2:5C:Shield}") {
		t.Error("ff131 não deveria sugerir {ICON:F2:5C:Shield} (par do ff133)")
	}

	color1 := valuesOf(cat1, "COLOR")
	if !hasTag(color1, "{COLOR:EXTRA00}") {
		t.Error("ff131 deveria expor os EXTRA (slots sem nome no ZTRtool)")
	}
	if hasTag(valuesOf(cat3, "COLOR"), "{COLOR:EXTRA00}") {
		t.Error("ff133 não deveria expor EXTRA00 (paleta de 31)")
	}

	// VAR: o par muda por codepage, mas o catálogo serve o LJ (o do "_us").
	if !hasTag(valuesOf(cat1, "VAR"), "{VAR:F4:40}") {
		t.Error("ff131 deveria sugerir {VAR:F4:40}")
	}
}

func valuesOf(cat tagCatalog, tag string) []tagSuggestion {
	for _, e := range cat.Tags {
		if e.Tag == tag {
			return e.Values
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
