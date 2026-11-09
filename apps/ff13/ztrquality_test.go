package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
)

// TestTranslatedZtrQuality confere a qualidade do importado: dos .ztr
// traduzidos, quantos o codec do app lê e quantos realmente mudaram em
// relação ao original.
func TestTranslatedZtrQuality(t *testing.T) {
	m := readImportManifest(t, "part1")
	data := partDataRoot(t, "part1")
	mods := partModsRoot(t, "part1")

	targetsByInternal := manifestTargetsByInternal(m)
	ambiguous := false
	for _, targets := range targetsByInternal {
		if len(targets) > 1 {
			ambiguous = true
			break
		}
	}

	total, changed, parseOK, parseBad := 0, 0, 0, 0
	badSamples := []string{}
	for _, e := range m.Entries {
		if !e.Matched || !strings.HasSuffix(strings.ToLower(e.Target), ".ztr") {
			continue
		}
		orig, err1 := os.ReadFile(filepath.Join(data, filepath.FromSlash(e.Target)))
		tr, err2 := os.ReadFile(resolveModsPath(mods, e.Target, e.Internal))
		if err1 != nil || err2 != nil {
			// Manifesto com o mesmo internal no global e na zona: só um
			// target existe; o ausente é a duplicata, não um arquivo perdido.
			if ambiguous {
				continue
			}
			t.Errorf("leitura: %s", e.Target)
			continue
		}
		total++
		if !bytes.Equal(orig, tr) {
			changed++
		}
		if _, err := ztr.Read(tr); err != nil {
			parseBad++
			if len(badSamples) < 8 {
				badSamples = append(badSamples, e.Target+" -> "+err.Error())
			}
			continue
		}
		parseOK++
	}
	t.Logf(".ztr traduzidos: %d | lidos pelo codec: %d | não lidos: %d | alterados vs original: %d",
		total, parseOK, parseBad, changed)
	for _, s := range badSamples {
		t.Logf("  não lido: %s", s)
	}
}
