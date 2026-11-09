package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ffresources/apps/ff13/parts"
)

// TestZoneUnion calcula, para o part1, como os caminhos internos se
// distribuem entre as zonas: quantos txtres cada indice de zona declara, a
// uniao e a intersecao. Isso decide se o conteudo de zona pode ser extraido
// uma vez (dedupe) ou por zona.
func TestZoneUnion(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	_, bins := scanPartFiles(p)
	indexes := discoverFilelists(p, roots, bins)

	owner := map[string][]string{} // caminho -> zonas que o declaram
	perZone := map[string]int{}
	total := 0
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		base := filepath.Base(idx.Virtual)
		zone := strings.TrimSuffix(strings.TrimPrefix(base, "filelist_"), ".win32.bin")
		for _, ep := range idx.Coll.Paths() {
			lp := strings.ToLower(filepath.ToSlash(ep))
			if !strings.Contains(lp, "txtres") {
				continue
			}
			owner[lp] = append(owner[lp], zone)
			perZone[zone]++
			total++
		}
	}

	unique := make([]string, 0, len(owner))
	for k := range owner {
		unique = append(unique, k)
	}
	sort.Strings(unique)

	shared := 0
	for _, zs := range owner {
		if len(zs) > 1 {
			shared++
		}
	}
	t.Logf("zonas=%d, entradas txtres=%d, caminhos unicos=%d, em 2+ zonas=%d",
		len(perZone), total, len(unique), shared)
	for _, z := range sortedKeys(perZone) {
		t.Logf("  %-8s %d", z, perZone[z])
	}
	for i, u := range unique {
		if i >= 8 {
			break
		}
		t.Logf("  unico: %s  <- %v", u, owner[u])
	}
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
