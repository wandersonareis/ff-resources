package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestZtrVariantSurvey classifica os .ztr little-endian de data/ e mods/:
//   - tipo 1 = par único (key+value);
//   - tipo n = dicionário com 2n offsets, que pode ter key e value em
//     BLOCOS separados ou intercalados (key e value juntos).
func TestZtrVariantSurvey(t *testing.T) {
	data := partDataRoot(t, "part1")
	mods := partModsRoot(t, "part1")

	survey := func(root string) {
		counts := map[string]int{}
		samples := map[string]string{}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".ztr") {
				return nil
			}
			raw, rerr := os.ReadFile(path)
			if rerr != nil || len(raw) < 12 {
				return nil
			}
			tp := binary.LittleEndian.Uint32(raw[0:4])
			if tp == 0 {
				counts["big(dicionario BE)"]++
				return nil
			}
			key := "?"
			if tp == 1 {
				key = "little:par"
			} else {
				n := int(tp)
				need := 4 + n*8
				if len(raw) < need {
					key = "little:truncado"
				} else {
					o0 := binary.LittleEndian.Uint32(raw[4:8])
					o1 := binary.LittleEndian.Uint32(raw[8:12])
					s0 := peekString(raw, o0)
					s1 := peekString(raw, o1)
					switch {
					case strings.HasPrefix(s0, "$") && strings.HasPrefix(s1, "$"):
						key = "little:blocos"
					case strings.HasPrefix(s0, "$"):
						key = "little:intercalado"
					default:
						key = "little:?"
					}
				}
			}
			counts[key]++
			if _, ok := samples[key]; !ok {
				rel, _ := filepath.Rel(root, path)
				samples[key] = filepath.ToSlash(rel)
			}
			return nil
		})
		t.Logf("== %s ==", root)
		for k, c := range counts {
			t.Logf("  %-22s %5d   (ex.: %s)", k, c, samples[k])
		}
	}
	survey(data)
	survey(mods)
}

// peekString devolve os primeiros 24 bytes legíveis do texto em off.
func peekString(raw []byte, off uint32) string {
	if int(off) >= len(raw) {
		return ""
	}
	end := int(off) + 24
	if end > len(raw) {
		end = len(raw)
	}
	b := raw[off:end]
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
