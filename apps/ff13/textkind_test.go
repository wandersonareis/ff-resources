package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ffresources/apps/ff13/parts"
)

// TestTextKindFromRealData confere o MAPEAMENTO kind←diretório sobre a árvore
// real extraída em bin/data/partN: cada família tem arquivos e nenhuma
// unidade cai no fallback "text" quando não deveria.
//
// O teste é SKIPPADO quando a árvore não está presente (a extração é manual).
func TestTextKindFromRealData(t *testing.T) {
	for _, tc := range []struct {
		part  string
		kinds []string
	}{
		{"part1", []string{parts.KindAC, parts.KindResident, parts.KindZone, parts.KindEvent}},
		{"part2", []string{parts.KindAC, parts.KindResident, parts.KindZone, parts.KindEvent}},
		{"part3", []string{parts.KindResident, parts.KindZone, parts.KindEvent}},
	} {
		t.Run(tc.part, func(t *testing.T) {
			dataDir := partDataRoot(t, tc.part)
			p := parts.Get(tc.part)

			counts := map[string]int{}
			err := filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || filepath.Ext(path) != ".ztr" {
					return nil
				}
				rel, rerr := filepath.Rel(dataDir, path)
				if rerr != nil {
					return nil
				}
				rel = filepath.ToSlash(rel)
				// A unidade é o id "<dir>/<base>": o kind do diretório da
				// unidade é o mesmo do arquivo.
				kind := p.TextKind(rel)
				counts[kind]++
				return nil
			})
			if err != nil {
				t.Fatalf("varredura: %v", err)
			}

			for _, k := range tc.kinds {
				if counts[k] == 0 {
					t.Errorf("%s: kind %q sem nenhum arquivo em %s (mapeamento errado?)", tc.part, k, dataDir)
				}
				t.Logf("%s %-9s %d arquivos", tc.part, k, counts[k])
			}
			// O fallback "text" só existe por segurança: nada no txtres pode
			// cair nele.
			if counts[parts.KindText] != 0 {
				t.Errorf("%s: %d arquivos caíram no fallback %q", tc.part, counts[parts.KindText], parts.KindText)
			}

			// O `scenario` só existe no part3 — e é o ÚNICO cujo texto mora
			// num container .bin (WPD), não num .ztr solto.
			if tc.part != "part3" && counts[parts.KindScenario] != 0 {
				t.Errorf("%s: kind scenario não deveria existir", tc.part)
			}
		})
	}
}

// TestTextKindScenarioContainer confere que o `scenario` do part3 é o único
// kind cujo arquivo editável é um CONTAINER (WPD) com o .ztr dentro — a
// razão pela qual o importador não precisa de caminho especial por kind.
func TestTextKindScenarioContainer(t *testing.T) {
	p := parts.Get("part3")

	if got := p.TextKind("db/ai/npc/pack/auto_c01_us.bin"); got != parts.KindScenario {
		t.Errorf("TextKind(db/ai/npc/pack/auto_c01_us.bin) = %q, quer %q", got, parts.KindScenario)
	}
	if got := p.TextKind("db/ai/npc/pack/quest1008_us.bin"); got != parts.KindScenario {
		t.Errorf("TextKind(quest) = %q, quer %q", got, parts.KindScenario)
	}
	// Fora do db/ai/npc/pack não é scenario.
	if got := p.TextKind("txtres/event/ev_comn_020/txtres_us.ztr"); got != parts.KindEvent {
		t.Errorf("TextKind(txtres/event) = %q, quer %q", got, parts.KindEvent)
	}
	if got := p.TextKind("db/ext/z0161pack.bin"); got != parts.KindScenario {
		// db/ext NÃO é scenario: é outro WPD sem texto.
		if got == parts.KindScenario {
			t.Errorf("db/ext não deveria ser scenario")
		}
	}
}

// TestTextKindUnitIDsConfirma que o kind sai do ID DA UNIDADE (não do nome do
// arquivo): "zone/z004/txtres/ac/ac_bodm_230/txtres" é `ac` mesmo estando
// sob o caminho de zona do app.
func TestTextKindUnitIDs(t *testing.T) {
	p := parts.Get("part1")
	cases := []struct{ id, want string }{
		{"txtres/event/ev_comn_000/txtres", parts.KindEvent},
		{"txtres/resident/book/txtres", parts.KindResident},
		{"txtres/zone/z000/txtres", parts.KindZone},
		{"zone/z004/txtres/ac/ac_bodm_230/txtres", parts.KindAC},
		{"zone/z004/txtres/event/ev_bodm_200/txtres", parts.KindEvent},
		{"zone/z004/txtres/zone/z004/txtres", parts.KindZone},
	}
	for _, c := range cases {
		if got := p.TextKind(c.id); got != c.want {
			t.Errorf("TextKind(%q) = %q, quer %q", c.id, got, c.want)
		}
	}
	// Ordem estável para os relatórios.
	got := parts.Kinds("part1")
	sort.Strings(got)
	want := []string{parts.KindAC, parts.KindEvent, parts.KindResident, parts.KindZone}
	if len(got) != len(want) {
		t.Fatalf("Kinds(part1) = %v, quer %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Kinds(part1)[%d] = %q, quer %q", i, got[i], want[i])
		}
	}
}
