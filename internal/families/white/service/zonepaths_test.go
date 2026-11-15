package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestZoneIndexPaths mostra os caminhos de um índice de zona: precisamos
// saber se o "txtres/..." do translation bate com o que o índice declara.
func TestZoneIndexPaths(t *testing.T) {
	testGameRoot(t, "part1")

	zoneDir := filepath.Join(partDataRoot(t, "part1"), "zone")
	dirEntries, err := os.ReadDir(zoneDir)
	if err != nil {
		t.Fatalf("lendo zone: %v", err)
	}
	for _, de := range dirEntries {
		name := de.Name()
		if !strings.HasSuffix(name, ".bin") {
			continue
		}
		coll, err := LoadFilelist(filepath.Join(zoneDir, name), "part1")
		if err != nil {
			t.Logf("%s: erro %v", name, err)
			continue
		}
		paths := coll.Paths()
		txtres := 0
		prefixes := map[string]int{}
		samples := []string{}
		for _, p := range paths {
			lp := strings.ToLower(p)
			if strings.Contains(lp, "txtres") {
				txtres++
				if len(samples) < 5 {
					samples = append(samples, lp)
				}
			}
			top := strings.SplitN(lp, "/", 2)[0]
			prefixes[top]++
		}
		t.Logf("%s: %d entradas, %d com txtres, prefixos=%v", name, len(paths), txtres, prefixes)
		for _, s := range samples {
			t.Logf("    %s", s)
		}
		if strings.HasPrefix(name, "filelist_z000u") {
			break
		}
	}
}
