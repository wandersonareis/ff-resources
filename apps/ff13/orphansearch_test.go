package main

import (
	"path/filepath"
	"strings"
	"testing"

	"ffresources/apps/ff13/parts"
)

// TestOrphanKeySearch procura, em TODOS os índices (globais + embutidos),
// qualquer entrada que mencione a chave de uma zona órfã — com QUALQUER
// nome de arquivo, não só os que começam com "filelist".
func TestOrphanKeySearch(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	filelists, bins := scanPartFiles(p)
	indexes := discoverFilelists(p, roots, bins)

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

	orphans := []string{}
	for key := range bins {
		if !covered[key] {
			orphans = append(orphans, key)
		}
	}

	for _, key := range orphans {
		hits := []string{}
		for _, idx := range indexes {
			for _, ep := range idx.Coll.Paths() {
				lp := strings.ToLower(ep)
				if strings.Contains(lp, key) {
					hits = append(hits, filepath.Base(idx.Path)+" -> "+ep)
					break
				}
			}
			if len(hits) > 0 {
				break
			}
		}
		if len(hits) == 0 {
			t.Logf("%-8s NENHUMA mencao em %d indices", key, len(indexes))
			continue
		}
		t.Logf("%-8s -> %s", key, hits[0])
	}
}
