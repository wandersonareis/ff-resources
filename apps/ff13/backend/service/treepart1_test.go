package service

import (
	"sort"
	"testing"

	"ffresources/apps/ff13/backend/common"
)

// TestPart1TreeShape percorre a árvore do part1 como a UI a monta:
//
//	Filelist
//	  ├── <globais>          (u, c, scru, scrc)
//	  └── Zone
//	        ├── z000, z002, …
//	Data
//	  ├── <conteúdo global por caminho interno>
//	  └── zone/<z>/<caminho interno>
//
// O teste confere o ESQUELETO (não o conteúdo inteiro): quantos índices,
// como se agrupam e o que aparece no primeiro nível de cada raiz.
func TestPart1TreeShape(t *testing.T) {
	testGameRoot(t, "part1")

	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}

	globals := []string{}
	zones := []string{}
	for _, r := range roots {
		if r.Group == common.GroupZone {
			zones = append(zones, r.Zone)
			continue
		}
		globals = append(globals, r.Name)
	}
	sort.Strings(globals)
	sort.Strings(zones)

	if len(globals) != 4 {
		t.Errorf("esperados 4 índices globais, obtidos %d: %v", len(globals), globals)
	}
	if len(zones) != 46 {
		t.Errorf("esperados 46 índices de zona, obtidos %d", len(zones))
	}
	t.Logf("globais: %v", globals)
	t.Logf("zonas: %d (%v…%v)", len(zones), zones[0], zones[len(zones)-1])

	// Primeiro nível de cada raiz global: as pastas de topo do conteúdo.
	for _, r := range roots {
		if r.Group == common.GroupZone {
			continue
		}
		nodes, derr := svc.ListFilelistDir("part1", r.Path, "")
		if derr != nil {
			t.Errorf("ListFilelistDir(%s): %v", r.Name, derr)
			continue
		}
		names := make([]string, 0, len(nodes))
		for _, n := range nodes {
			names = append(names, n.Name)
		}
		sort.Strings(names)
		t.Logf("%-18s %d nós no topo: %v", r.Name, len(nodes), names)
		if len(nodes) == 0 {
			t.Errorf("%s: índice sem entradas", r.Name)
		}
	}

	// Uma zona: a listagem é a UNIÃO do índice de zona (u+c) com as entradas
	// "zone/<z>/..." dos demais índices da part.
	zoneNodes, err := svc.ListFilelistZone("part1", "z000")
	if err != nil {
		t.Fatalf("ListFilelistZone(z000): %v", err)
	}
	names := make([]string, 0, len(zoneNodes))
	for _, n := range zoneNodes {
		names = append(names, n.Name)
	}
	sort.Strings(names)
	t.Logf("zone/z000: %d nós no topo: %v", len(zoneNodes), names)
	if len(zoneNodes) == 0 {
		t.Error("z000: zona sem conteúdo")
	}
}
