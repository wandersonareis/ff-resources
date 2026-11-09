package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
)

// TestZtrParseFailures varre os .ztr de data/part1 e conta quais NÃO são
// lidos pelo codec: o app precisa saber o que é texto e o que é outro formato.
func TestZtrParseFailures(t *testing.T) {
	data := partDataRoot(t, "part1")
	bad := map[string]int{}
	badSample := map[string]string{}
	total := 0
	_ = filepath.WalkDir(data, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".ztr") {
			return nil
		}
		total++
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if _, perr := ztr.Read(raw); perr != nil {
			key := perr.Error()
			bad[key]++
			rel, _ := filepath.Rel(data, path)
			rel = filepath.ToSlash(rel)
			badSample[rel] = key
		}
		return nil
	})
	t.Logf("total .ztr = %d", total)
	byLang := map[string]int{}
	for rel := range badSample {
		lang := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(rel), "txtres_"), ".ztr")
		byLang[lang]++
	}
	t.Logf("falhas por idioma:")
	for lang, c := range byLang {
		t.Logf("  %-4s %d", lang, c)
	}
	t.Logf("total de falhas: %d", len(badSample))
	for k, n := range bad {
		t.Logf("%4d x %s", n, k)
	}
}
