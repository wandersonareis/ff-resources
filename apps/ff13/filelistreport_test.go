package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ffresources/apps/ff13/parts"
)

// TestUnpairedWhiteBins lista os white bins que ficaram SEM par (nenhum
// filelist no disco nem índice embutido que os cite). É o relatório do que a
// varredura de índices não alcançou.
func TestUnpairedWhiteBins(t *testing.T) {
	for _, part := range []string{"part1", "part2", "part3"} {
		part := part
		t.Run(part, func(t *testing.T) {
			testGameRoot(t, part)
			p := parts.Get(part)
			var svc FF13Service
			roots, err := svc.ListFilelistRoots(part)
			if err != nil {
				t.Fatalf("ListFilelistRoots: %v", err)
			}
			filelists, bins := scanPartFiles(p)
			covered := map[string]bool{}
			for key := range filelists {
				covered[key] = true
			}
			for _, idx := range discoverFilelists(p, roots, bins) {
				if idx.Level == 0 {
					continue
				}
				if key, ok := p.PairKey(filepath.Base(idx.Virtual)); ok {
					covered[key] = true
				}
			}
			total := 0
			mapped := 0
			type miss struct {
				name string
				key  string
				mb   float64
			}
			misses := []miss{}
			for key, bin := range bins {
				if covered[key] {
					mapped++
					continue
				}
				total++
				var size int64
				if st, serr := os.Stat(bin); serr == nil {
					size = st.Size()
				}
				rel, _ := filepath.Rel(p.DataDir(gameRoot(part)), bin)
				misses = append(misses, miss{filepath.Base(rel), key, float64(size) / (1024 * 1024)})
			}
			sort.Slice(misses, func(i, j int) bool { return misses[i].name < misses[j].name })
			for _, m := range misses {
				t.Logf("SEM PAR: %-34s chave=%-10s %8.1f MB", m.name, m.key, m.mb)
			}
			t.Logf("%s: %d white bins SEM par, %d mapeados (total %d)", part, total, mapped, len(bins))
		})
	}
}
