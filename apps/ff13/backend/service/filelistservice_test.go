package service

import (
	"strings"
	"testing"

	"ffresources/apps/ff13/backend/common"
)

// TestFilelistTreeSmoke valida a descoberta + a listagem do filelist do FF13.
// O caminho do jogo vem do arquivo de config local (ff13.config.json).
func TestFilelistTreeSmoke(t *testing.T) {
	testGameRoot(t, "part1")

	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	if len(roots) == 0 {
		t.Fatal("nenhum filelist encontrado")
	}
	for _, r := range roots {
		t.Logf("root: %s (path=%s part=%s source=%s lang=%s group=%q zone=%q kind=%s dir=%s, %d entradas)",
			r.Name, r.Path, r.Part, r.Source, r.Language, r.Group, r.Zone, r.Kind, r.SourceDir, r.Entries)
		if r.Part != "part1" {
			t.Errorf("root %s: part %q, quer part1", r.Name, r.Part)
		}
		if r.Source == "" {
			t.Errorf("root %s: source vazia", r.Name)
		}
		if r.SourceDir == "" {
			t.Errorf("root %s: sourceDir vazia", r.Name)
		}
		if r.Kind != common.KindDisk && r.Kind != common.KindEmbedded {
			t.Errorf("root %s: kind %q inesperado", r.Name, r.Kind)
		}
	}

	// Um dos filelists (o de conteúdo, filelistu) tem txtres na raiz; os de
	// script (filelist_scru) têm btscene/chr/sys/vfx/zone.
	var root common.FilelistRoot
	found := false
	for _, r := range roots {
		nodes, lerr := svc.ListFilelistDir("part1", r.Path, "")
		if lerr != nil {
			continue
		}
		for _, n := range nodes {
			if n.Name == "txtres" && n.IsDir {
				root = r
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatalf("nenhum filelist com txtres na raiz: %+v", roots)
	}
	t.Logf("usando %s", root.Name)

	nodes, err := svc.ListFilelistDir("part1", root.Path, "txtres")
	if err != nil {
		t.Fatalf("ListFilelistDir: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("txtres sem filhos")
	}
	for _, n := range nodes {
		t.Logf("%s (kind=%s dir=%v size=%d)", n.Path, n.Kind, n.IsDir, n.Size)
	}

	// Os índices de ZONA moram dentro do índice global: a pasta "zone" traz
	// os índices embutidos (kind=index).
	zone, nested := 0, 0
	zoneNodes, zerr := svc.ListFilelistDir("part1", root.Path, "zone")
	if zerr != nil {
		t.Fatalf("ListFilelistDir(zone): %v", zerr)
	}
	for _, n := range zoneNodes {
		if !strings.HasPrefix(n.Path, "zone/") {
			continue
		}
		zone++
		if n.Kind == common.KindIndex {
			nested++
			t.Logf("zone filelist: %s (%d bytes)", n.Path, n.Size)
		}
	}
	t.Logf("zone: %d entradas, %d filelists embutidos", zone, nested)
}

// TestFilelistZoneListing confere a listagem da ZONA: a união do índice de
// zona com as entradas "zone/<z>/..." dos demais índices da part.
func TestFilelistZoneListing(t *testing.T) {
	testGameRoot(t, "part1")

	var svc FF13Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	zones := map[string]bool{}
	for _, r := range roots {
		if r.Group == common.GroupZone && r.Zone != "" {
			zones[r.Zone] = true
		}
	}
	if len(zones) == 0 {
		t.Skip("nenhum índice de zona descoberto")
	}
	for zone := range zones {
		nodes, nerr := svc.ListFilelistZone("part1", zone)
		if nerr != nil {
			t.Fatalf("ListFilelistZone(%s): %v", zone, nerr)
		}
		if len(nodes) == 0 {
			t.Errorf("zona %s sem conteúdo", zone)
			continue
		}
		names := make([]string, 0, len(nodes))
		for _, n := range nodes {
			names = append(names, n.Name)
		}
		t.Logf("zona %s: %d nós no topo: %v", zone, len(nodes), names)
		break
	}
}
