package service

import (
	"path/filepath"
	"testing"
)

// TestListDataEntriesEmptyReturnsEmptySlice fixa o contrato do binding: sem
// nada em data/mods a lista é [] (nunca null), senão o frontend quebra ao
// filtrar/mapar a árvore.
func TestListDataEntriesEmptyReturnsEmptySlice(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	var svc Service
	entries, err := svc.ListDataEntries("part1")
	if err != nil {
		t.Fatalf("ListDataEntries: %v", err)
	}
	if entries == nil {
		t.Fatal("entries = nil; o binding deve devolver [] quando vazio")
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %d, quer 0", len(entries))
	}
}
