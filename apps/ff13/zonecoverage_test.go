package main

import (
	"path/filepath"
	"strings"
	"testing"

	"ffresources/apps/ff13/parts"
)

// TestZoneCoverage cruza as pastas de zona INDEXADAS com os white bins de
// zona em disco: as zonas que existem no índice mas não têm filelist são as
// que só podem ser descobertas dentro do próprio bin da zona.
func TestZoneCoverage(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	filelists, bins := scanPartFiles(p)
	indexes := discoverFilelists(p, roots, bins)

	// Pastas de zona citadas por QUALQUER índice (inclusive embutidos).
	zones := map[string]bool{}
	for _, idx := range indexes {
		for _, ep := range idx.Coll.Paths() {
			lower := strings.ToLower(ep)
			if !strings.Contains(lower, "zone") {
				continue
			}
			for _, seg := range strings.Split(lower, "/") {
				if strings.HasPrefix(seg, "z") && len(seg) == 4 {
					zones[seg] = true
				}
			}
		}
	}
	t.Logf("zonas citadas nos indices: %d", len(zones))

	covered := map[string]bool{}
	for key := range filelists {
		covered[key] = true
	}
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		if key, ok := p.PairKey(filepath.Base(idx.Virtual)); ok {
			covered[key] = true
		}
	}
	for key, bin := range bins {
		if covered[key] {
			continue
		}
		zone := key
		if strings.HasPrefix(zone, "z") {
			zone = zone[:4]
		}
		t.Logf("sem par: %-34s zona=%-6s zona citada no indice=%v", filepath.Base(bin), zone, zones[zone])
	}
}
