package main

import (
	"path/filepath"
	"testing"

	"ffresources/apps/ff13/common"
	"ffresources/apps/ff13/parts"
)

// TestEmbeddedFilelistDiscovery procura filelists DENTRO dos indices disponiveis
// (a cadeia do FF13: indice global -> indice de zona -> ...) e confere se eles
// casam com os white bins que estavam sem par.
func TestEmbeddedFilelistDiscovery(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	filelists, bins := scanPartFiles(p)
	t.Logf("em disco: %d indices, %d white bins", len(filelists), len(bins))
	t.Logf("roots: %d", len(roots))
	for _, r := range roots {
		t.Logf("  root %s -> par %q", filepath.Base(r.Path), r.WhiteBin)
	}

	indexes := discoverFilelists(p, roots, bins)
	levels := map[int]int{}
	for _, idx := range indexes {
		levels[idx.Level]++
	}
	t.Logf("indices descobertos por nivel: %v", levels)

	embedded := 0
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		embedded++
		if embedded <= 5 {
			t.Logf("nivel %d: %s (container %s, %d entradas)",
				idx.Level, idx.Virtual, filepath.Base(idx.Container), idx.Coll.FileCount())
		}
	}
	if embedded == 0 {
		t.Fatal("nenhum filelist embutido descoberto")
	}

	// Casamento: todo white bin sem par em disco precisa aparecer como índice
	// embutido (ou em disco).
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
	unpaired := []string{}
	for key, bin := range bins {
		if !covered[key] {
			unpaired = append(unpaired, filepath.Base(bin))
		}
	}
	t.Logf("white bins ainda sem par: %d", len(unpaired))
	for i, u := range unpaired {
		if i >= 5 {
			break
		}
		t.Logf("  sem par: %s", u)
	}
}

// TestEmbeddedFilelistExpansion abre um índice EMBUTIDO pela raiz
// "pai#caminho/virtual" e confere que ele lista o conteúdo do bin externo.
func TestEmbeddedFilelistExpansion(t *testing.T) {
	testGameRoot(t, "part1")

	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	for _, root := range roots {
		if root.Group == common.GroupZone {
			continue
		}
		nodes, lerr := svc.ListFilelistDir("part1", root.Path, "zone")
		if lerr != nil {
			continue
		}
		for _, n := range nodes {
			if n.Kind != common.KindIndex {
				continue
			}
			children, cerr := svc.ListFilelistDir("part1", root.Path+"#"+n.Path, "")
			if cerr != nil {
				t.Fatalf("ListFilelistDir(%s): %v", n.Path, cerr)
			}
			if len(children) == 0 {
				t.Fatalf("indice embutido %s sem filhos", n.Path)
			}
			t.Logf("%s -> %d filhos (ex.: %s)", n.Path, len(children), children[0].Name)
			return
		}
	}
	t.Skip("nenhum indice embutido encontrado nos indices da part1")
}
