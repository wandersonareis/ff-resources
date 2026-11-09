package main

import (
	"testing"
)

// TestFilelistRootsAllParts valida a descoberta POR PART em cada jogo
// instalado. Os caminhos vem do arquivo de config local (ff13.config.json);
// o jogo ausente faz o teste ser pulado.
func TestFilelistRootsAllParts(t *testing.T) {
	var svc FF13Service
	for _, part := range []string{"part1", "part2", "part3"} {
		part := part
		t.Run(part, func(t *testing.T) {
			testGameRoot(t, part)
			roots, err := svc.ListFilelistRoots(part)
			if err != nil {
				t.Fatalf("ListFilelistRoots: %v", err)
			}
			if len(roots) == 0 {
				t.Fatal("nenhum filelist encontrado")
			}
			for _, r := range roots {
				t.Logf("%s: %s (%d entradas, par=%q)", part, r.Name, r.Entries, r.WhiteBin)
				nodes, lerr := svc.ListFilelistDir(part, r.Path, "")
				if lerr != nil {
					t.Errorf("%s: ListFilelistDir(%s): %v", part, r.Name, lerr)
					continue
				}
				if len(nodes) == 0 {
					t.Errorf("%s: %s sem entradas", part, r.Name)
				}
			}
		})
	}
}
