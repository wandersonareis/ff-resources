package service

import (
	"ffresources/apps/ff13/backend/formats"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/apps/ff13/backend/parts"
)

// TestZoneIndexDetails confere dois detalhes que o Pulse (Albeoris) trata:
//   - as entradas "zone/filelist_zXXX*.bin2" são índices VAZIOS (2º arquivo);
//   - dentro de um índice de zona, as entradas "bg/loc..." podem ser de outra
//     zona e são descartadas (IsUnexpectedEntry no Pulse).
func TestZoneIndexDetails(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	_, bins := formats.ScanPartFiles(p)
	indexes := formats.DiscoverFilelists(p, roots, bins)

	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		base := filepath.Base(idx.Virtual)
		zone := strings.TrimPrefix(base, "filelist_")
		zone = strings.TrimSuffix(zone, ".win32.bin")

		// bg/loc de outra zona?
		foreign := 0
		for _, ep := range idx.Coll.Paths() {
			lp := strings.ToLower(ep)
			if !strings.HasPrefix(lp, "bg/loc") {
				continue
			}
			rest := strings.TrimPrefix(lp, "bg/loc")
			if len(rest) >= 3 && rest[:3] != zone[:min(3, len(zone))] {
				foreign++
			}
		}
		t.Logf("%-28s entradas=%-6d bg/loc de outra zona=%d", base, idx.Coll.FileCount(), foreign)
		if foreign > 0 {
			for _, ep := range idx.Coll.Paths() {
				lp := strings.ToLower(ep)
				if strings.HasPrefix(lp, "bg/loc") {
					t.Logf("    ex.: %s", ep)
					break
				}
			}
		}
	}
}

// TestEmptySecondIndexes confere que os índices ".bin2" são vazios.
func TestEmptySecondIndexes(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	_, _ = formats.ScanPartFiles(p)

	for _, root := range roots {
		coll, lerr := formats.LoadFilelist(root.Path, "part1")
		if lerr != nil {
			continue
		}
		container := formats.PairOnDisk(p, root.Path)
		if container == "" {
			continue
		}
		parent := &formats.FilelistIndex{Path: root.Path, Container: container, Level: 0, Coll: coll}
		for _, ep := range coll.Paths() {
			base := filepath.Base(ep)
			if !strings.HasPrefix(strings.ToLower(base), "filelist") || !strings.HasSuffix(strings.ToLower(base), ".bin2") {
				continue
			}
			embedded, err := formats.ReadEmbeddedIndex(parent, ep, p.GameCode())
			if err != nil {
				t.Logf("%s: nao parseou (%v)", base, err)
				continue
			}
			t.Logf("%s: %d entradas", base, embedded.FileCount())
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
