package main

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/shared/dto"
)

// TestDataCollectionAggregates confere o INVENTÁRIO da coleção: as unidades
// de texto (Text=true) e os arquivos que o app não decodifica (Text=false)
// convivem na mesma lista.
func TestDataCollectionAggregates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1", "id2"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello", "World"}))
	writeFile(t, filepath.Join(root, "data", "part1", "gui", "resident", "system.win32.imgb"),
		[]byte("IMGB"))

	var svc FF13Service
	entries, err := svc.ListDataEntries("part1")
	if err != nil {
		t.Fatalf("ListDataEntries: %v", err)
	}
	var text, other []DataEntry
	for _, e := range entries {
		if e.Text {
			text = append(text, e)
		} else {
			other = append(other, e)
		}
	}
	if len(text) != 1 {
		t.Fatalf("unidades de texto: %d, quer 1 (%+v)", len(text), text)
	}
	if text[0].ID != "txtres/resident/book/txtres" {
		t.Errorf("id da unidade: %q", text[0].ID)
	}
	if text[0].Primary != "txtres/resident/book/txtres_us.ztr" {
		t.Errorf("primario: %q", text[0].Primary)
	}
	if text[0].Modded {
		t.Error("unidade sem tradução marcada como modded")
	}
	if len(other) != 1 {
		t.Fatalf("arquivos fora do escopo: %d, quer 1 (%+v)", len(other), other)
	}
	if other[0].ID != "gui/resident/system.win32.imgb" {
		t.Errorf("id fora do escopo: %q", other[0].ID)
	}
}

// TestDataCollectionCache confere o CACHE da coleção: a segunda leitura
// devolve a MESMA coleção (a varredura do disco não paga de novo) e a
// invalidação descarta a cache (a próxima leitura refaz a varredura).
func TestDataCollectionCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))

	first, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor: %v", err)
	}
	second, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor (2a): %v", err)
	}
	if first != second {
		t.Error("a coleção não foi reaproveitada: a varredura do disco pagou de novo")
	}

	// Um arquivo novo só aparece depois da invalidação.
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_fr.ztr"),
		testZtrBlob(t, ids, []string{"Bonjour"}))
	invalidateCollection("part1")

	third, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor (após invalidar): %v", err)
	}
	if third == first {
		t.Error("a invalidação não descartou a coleção")
	}
	unit, ok := third.units["txtres/resident/book/txtres"]
	if !ok {
		t.Fatal("unidade ausente depois da invalidação")
	}
	if _, ok := unit.Files["fr"]; !ok {
		t.Error("o idioma novo não entrou na coleção")
	}
}

// TestDataCollectionKeyIncludesRoots confere que a chave da coleção carrega
// as raízes: coleções de árvores diferentes não podem se misturar.
func TestDataCollectionKeyIncludesRoots(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	key := collectionKey("part1")
	if want := "part1|" + filepath.Join(root, "data") + "|" + filepath.Join(root, "mods"); key != want {
		t.Errorf("collectionKey: %q, quer %q", key, want)
	}
}

// TestInvalidateCollectionKeepsOtherParts confere que a invalidação de uma
// part NÃO derruba a coleção das outras.
func TestInvalidateCollectionKeepsOtherParts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "a", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))
	writeFile(t, filepath.Join(root, "data", "part2", "txtres", "b", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hallo"}))

	first1, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor(part1): %v", err)
	}
	first2, err := collectionFor("part2")
	if err != nil {
		t.Fatalf("collectionFor(part2): %v", err)
	}
	invalidateCollection("part1")
	second2, err := collectionFor("part2")
	if err != nil {
		t.Fatalf("collectionFor(part2) após invalidar part1: %v", err)
	}
	if second2 != first2 {
		t.Error("a coleção do part2 foi derrubada pela invalidação do part1")
	}
	second1, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor(part1) após invalidar: %v", err)
	}
	if second1 == first1 {
		t.Error("a coleção do part1 não foi descartada")
	}
}

// TestCollectionContentCache confere o cache do CONTEÚDO: a segunda leitura
// da mesma unidade não decodifica de novo.
func TestCollectionContentCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))

	coll, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor: %v", err)
	}
	unit, err := collectionUnit(coll, "part1", "txtres/resident/book/txtres")
	if err != nil {
		t.Fatalf("collectionUnit: %v", err)
	}
	first, err := unitContentOf(coll, "part1", unit)
	if err != nil {
		t.Fatalf("unitContentOf: %v", err)
	}
	second, err := unitContentOf(coll, "part1", unit)
	if err != nil {
		t.Fatalf("unitContentOf (2a): %v", err)
	}
	if first != second {
		t.Error("o conteúdo não foi reaproveitado do cache")
	}
	if first.orig["id1"] != "Hello" {
		t.Errorf("pristine: %q, quer %q", first.orig["id1"], "Hello")
	}
}

// TestListDataEntriesAfterSave confere o caminho completo: o save grava em
// mods/ e a coleção é invalidada (a próxima leitura devolve a tradução).
func TestListDataEntriesAfterSave(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	t.Setenv("FF_RESOURCES_CONFIG", filepath.Join(root, "ff13.config.json"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))

	var svc FF13Service
	if _, err := svc.ListDataEntries("part1"); err != nil {
		t.Fatalf("ListDataEntries: %v", err)
	}
	c := dto.Collection{
		"txtres/resident/book/txtres": {
			Metadata: dto.Metadata{Key: "part1/txtres/resident/book/txtres", ID: "txtres/resident/book/txtres"},
			Rows: []dto.TextRow{
				{Index: 0, Name: "id1", Text: map[string]string{"us": "Ola"}, Target: "Ola"},
			},
		},
	}
	if err := svc.ApplyTextCollection("part1", c); err != nil {
		t.Fatalf("ApplyTextCollection: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "mods", "part1", "txtres", "resident", "book", "txtres_us.ztr")); err != nil {
		t.Fatalf("tradução ausente em mods: %v", err)
	}
	// A coleção invalidada é reconstruída e a unidade agora está `modded`.
	entries, err := svc.ListDataEntries("part1")
	if err != nil {
		t.Fatalf("ListDataEntries (após salvar): %v", err)
	}
	found := false
	for _, e := range entries {
		if e.ID != "txtres/resident/book/txtres" {
			continue
		}
		found = true
		if !e.Modded {
			t.Error("unidade traduzida não marcada como modded")
		}
	}
	if !found {
		t.Fatal("unidade ausente após o save")
	}
	entry, err := svc.GetTextEntry("part1", "txtres/resident/book/txtres", "")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if got := entry.Rows[0].Text["us"]; got != "Ola" {
		t.Errorf("text[us]: %q, quer %q (a tradução recém-salva)", got, "Ola")
	}
}
