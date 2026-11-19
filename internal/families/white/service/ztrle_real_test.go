package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
)

// TestRealPart1TranslationLoads confere o caminho completo do part1 depois
// que a árvore de mods passou a usar o caminho canônico zone/<z>/...: o data
// é BE e a tradução é LE; o serviço deve mostrar o texto de mods.
func TestRealPart1TranslationLoads(t *testing.T) {
	data := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data")
	unit := "zone/z004/txtres/ac/ac_bodm_230/txtres"
	if _, err := os.Stat(filepath.Join(data, "part1", filepath.FromSlash(unit)+"_us.ztr")); err != nil {
		t.Skipf("sem fixture real do part1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "mods", "part1", filepath.FromSlash(unit)+"_us.ztr")); err != nil {
		t.Skipf("sem tradução real do part1 em mods/: %v", err)
	}
	t.Setenv("FF_RESOURCES_DATA", mustAbs(t, data))
	t.Setenv("FF_RESOURCES_MODS", mustAbs(t, filepath.Join(data, "mods")))

	var svc Service
	entry, err := svc.GetTextEntry("part1", unit, "", "", "")
	if err != nil {
		t.Fatalf("GetTextEntry(part1): %v", err)
	}
	if len(entry.Rows) == 0 {
		t.Fatal("sem rows")
	}
	if entry.Rows[0].Text["us"] == entry.Rows[0].Original["us"] {
		t.Errorf("part1 não mostrou a tradução: text=%q original=%q",
			entry.Rows[0].Text["us"], entry.Rows[0].Original["us"])
	}
}

// TestRealPart2TranslationLoads confere o caminho completo: part2 tem os
// caminhos de mods iguais aos de data, e a tradução é LE. O serviço deve
// mostrar o texto de mods (não o pristine).
func TestRealPart2TranslationLoads(t *testing.T) {
	data := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data")
	if _, err := os.Stat(filepath.Join(data, "part2", "txtres", "ac", "ac_aaea_007", "txtres_us.ztr")); err != nil {
		t.Skipf("sem fixture real do part2: %v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "mods", "part2", "txtres", "ac", "ac_aaea_007", "txtres_us.ztr")); err != nil {
		t.Skipf("sem tradução real do part2 em mods/: %v", err)
	}
	t.Setenv("FF_RESOURCES_DATA", mustAbs(t, data))
	t.Setenv("FF_RESOURCES_MODS", mustAbs(t, filepath.Join(data, "mods")))

	var svc Service
	entry, err := svc.GetTextEntry("part2", "txtres/ac/ac_aaea_007/txtres", "", "", "")
	if err != nil {
		t.Fatalf("GetTextEntry(part2): %v", err)
	}
	if len(entry.Rows) == 0 {
		t.Fatal("sem rows")
	}
	if entry.Rows[0].Text["us"] == entry.Rows[0].Original["us"] {
		t.Errorf("part2 não mostrou a tradução: text=%q original=%q",
			entry.Rows[0].Text["us"], entry.Rows[0].Original["us"])
	}
}

// TestRealTranslationZtrFormats confere que os .ztr reais da tradução
// (little-endian no part1/part2) são lidos e decodificados. Skippa quando a
// árvore bin/data não está presente.
func TestRealTranslationZtrFormats(t *testing.T) {
	cases := []struct {
		part string
		path string
	}{
		{"part1", filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods", "part1", "zone", "z004", "txtres", "ac", "ac_bodm_230", "txtres_us.ztr")},
		{"part2", filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods", "part2", "txtres", "ac", "ac_aaea_007", "txtres_us.ztr")},
	}
	for _, c := range cases {
		c := c
		t.Run(c.part, func(t *testing.T) {
			raw, err := os.ReadFile(c.path)
			if err != nil {
				t.Skipf("sem fixture real (%s): %v", c.path, err)
			}
			f, err := ztr.Read(raw)
			if err != nil {
				t.Fatalf("ztr.Read(%s): %v", c.path, err)
			}
			if len(f.IDs) == 0 || len(f.Lines) == 0 {
				t.Fatalf("%s: arquivo vazio (%d IDs, %d linhas)", c.path, len(f.IDs), len(f.Lines))
			}
			if !strings.HasPrefix(f.IDs[0], "$") {
				t.Errorf("%s: ID inesperado: %q", c.path, f.IDs[0])
			}
			ks, err := TextCodec(c.part)
			if err != nil {
				t.Fatalf("TextCodec(%s): %v", c.part, err)
			}
			texts, err := ks.DecodeLines(f.Lines)
			if err != nil {
				t.Fatalf("DecodeLines(%s): %v", c.path, err)
			}
			for i, text := range texts {
				if strings.TrimSpace(text) == "" && i == 0 {
					t.Errorf("%s: primeira linha vazia", c.path)
				}
			}
		})
	}
}
