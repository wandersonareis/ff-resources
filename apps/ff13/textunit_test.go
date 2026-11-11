package main

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/apps/ff13/parts"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
)

// TestParseUnitFileName confere o corte do sufixo de idioma: so conta quando
// o codigo e do catalogo da trilogia.
func TestParseUnitFileName(t *testing.T) {
	cases := []struct {
		name string
		base string
		lang string
	}{
		{"txtres_us.ztr", "txtres", "us"},
		{"auto_yus_us.bin", "auto_yus", "us"},
		{"txtres_fr.ztr", "txtres", "fr"},
		{"txtres_gr.ztr", "txtres", "gr"},
		{"txtres_jp.ztr", "txtres", "jp"},
		{"data_backup.bin", "data_backup", ""},
		{"txtres_ck.ztr", "txtres_ck", ""},
		{"book.ztr", "book", ""},
		{"_us.ztr", "_us", ""},
		{"us.ztr", "us", ""},
	}
	for _, c := range cases {
		base, lang := parseUnitFileName(c.name)
		if base != c.base || lang != c.lang {
			t.Errorf("parseUnitFileName(%q) = (%q, %q), esperado (%q, %q)", c.name, base, lang, c.base, c.lang)
		}
	}
}

// testZtrBlob monta um .ztr com os IDs/linhas pedidos (charset do part1).
func testZtrBlob(t *testing.T, ids []string, texts []string) []byte {
	t.Helper()
	ks, err := textCodec("part1")
	if err != nil {
		t.Fatalf("textCodec: %v", err)
	}
	lines, err := ks.EncodeLines(texts)
	if err != nil {
		t.Fatalf("EncodeLines: %v", err)
	}
	blob, err := ztr.WriteTie(&ztr.File{IDs: ids, Lines: lines}, ztr.Compressed, ztr.TieMinLast)
	if err != nil {
		t.Fatalf("WriteTie: %v", err)
	}
	return blob
}

// writeFile grava um arquivo (cria a pasta).
func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// writeWpd grava um container WPD com os registros pedidos.
func writeWpd(t *testing.T, path string, parts []wpd.Part) {
	t.Helper()
	blob, err := wpd.WriteFromParts(parts)
	if err != nil {
		t.Fatalf("WriteFromParts %s: %v", path, err)
	}
	writeFile(t, path, blob)
}

// TestScanTextUnits varre uma arvore SINTETICA e confere o agrupamento em
// unidades: os idiomas da mesma pasta/base viram UMA unidade, o container WPD
// conta, e o que nao e texto (sem magic WPD, imgb, xgr) fica de fora.
func TestScanTextUnits(t *testing.T) {
	dir := t.TempDir()
	ztrBlob := testZtrBlob(t, []string{"id1", "id2"}, []string{"Hello", "World"})

	// Mesma pasta, mesmo base: uma unidade com dois idiomas.
	writeFile(t, filepath.Join(dir, "txtres", "resident", "book", "txtres_us.ztr"), ztrBlob)
	writeFile(t, filepath.Join(dir, "txtres", "resident", "book", "txtres_fr.ztr"), ztrBlob)
	// Container WPD: uma unidade a parte.
	writeWpd(t, filepath.Join(dir, "db", "ai", "npc", "pack", "auto_yus_us.bin"), []wpd.Part{
		{Name: "auto_yus", Extension: "wdb", Data: []byte("WDB-BYTES")},
		{Name: "auto_yus_us", Extension: "ztr", Data: ztrBlob},
	})
	// .bin sem magic WPD: ignorado.
	writeFile(t, filepath.Join(dir, "db", "ai", "npc", "pack", "plain_us.bin"), []byte("NOT-A-WPD"))
	// Fora do escopo de texto: ignorado.
	writeFile(t, filepath.Join(dir, "db", "ai", "npc", "pack", "pic_us.imgb"), []byte("IMGB"))
	writeFile(t, filepath.Join(dir, "db", "ai", "npc", "pack", "xgr_us.xgr"), []byte("XGR"))
	// Pasta com dois bases: duas unidades.
	writeFile(t, filepath.Join(dir, "misc", "a_us.ztr"), ztrBlob)
	writeFile(t, filepath.Join(dir, "misc", "b_us.ztr"), ztrBlob)
	// Sem "_us": unidade descartada.
	writeFile(t, filepath.Join(dir, "misc", "orphan_fr.ztr"), ztrBlob)
	// A pasta de indices nao e conteudo de /data.
	writeFile(t, filepath.Join(dir, "filelist", "index_us.ztr"), ztrBlob)

	units, err := scanTextUnits(dir, parts.Get("part1"))
	if err != nil {
		t.Fatalf("scanTextUnits: %v", err)
	}
	byID := map[string]int{}
	for i, u := range units {
		byID[u.ID] = i
	}
	want := []string{
		"db/ai/npc/pack/auto_yus",
		"misc/a",
		"misc/b",
		"txtres/resident/book/txtres",
	}
	if len(units) != len(want) {
		var ids []string
		for _, u := range units {
			ids = append(ids, u.ID)
		}
		t.Fatalf("unidades: %v, esperado %v", ids, want)
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Errorf("unidade ausente: %s", id)
		}
	}

	// Os idiomas da mesma pasta/base ficam na MESMA unidade.
	book := units[byID["txtres/resident/book/txtres"]]
	if book.Label != "txtres" {
		t.Errorf("rótulo: %q, esperado %q (sem o sufixo de idioma)", book.Label, "txtres")
	}
	if len(book.Langs) != 2 || book.Langs[0] != "us" || book.Langs[1] != "fr" {
		t.Errorf("langs: %v, esperado [us fr]", book.Langs)
	}
	if book.Primary != "txtres/resident/book/txtres_us.ztr" {
		t.Errorf("primary: %q", book.Primary)
	}
	if book.Container != "ztr" {
		t.Errorf("container: %q, esperado ztr", book.Container)
	}
	if book.Key != "part1/"+book.ID {
		t.Errorf("key: %q", book.Key)
	}

	// A unidade WPD: container "wpd" e o .bin no primary.
	wpdUnit := units[byID["db/ai/npc/pack/auto_yus"]]
	if wpdUnit.Container != "wpd" {
		t.Errorf("container: %q, esperado wpd", wpdUnit.Container)
	}
	if wpdUnit.Primary != "db/ai/npc/pack/auto_yus_us.bin" {
		t.Errorf("primary: %q", wpdUnit.Primary)
	}
	if wpdUnit.Label != "auto_yus" {
		t.Errorf("rótulo: %q, esperado auto_yus", wpdUnit.Label)
	}
}

// TestResolveTextUnit confere a resolucao de uma unidade pelo id (sem varrer
// a arvore inteira) e o desempate do .ztr solto sobre o .bin do mesmo idioma.
func TestResolveTextUnit(t *testing.T) {
	dir := t.TempDir()
	ztrBlob := testZtrBlob(t, []string{"id1"}, []string{"Hello"})
	writeFile(t, filepath.Join(dir, "txtres", "resident", "book", "txtres_us.ztr"), ztrBlob)
	writeFile(t, filepath.Join(dir, "txtres", "resident", "book", "txtres_fr.ztr"), ztrBlob)
	writeWpd(t, filepath.Join(dir, "txtres", "resident", "book", "txtres_us.bin"), []wpd.Part{
		{Name: "txtres_us", Extension: "ztr", Data: ztrBlob},
	})

	unit, err := resolveTextUnit(dir, parts.Get("part1"), "txtres/resident/book/txtres")
	if err != nil {
		t.Fatalf("resolveTextUnit: %v", err)
	}
	if unit.ID != "txtres/resident/book/txtres" {
		t.Errorf("id: %q", unit.ID)
	}
	if unit.Container != "ztr" {
		t.Errorf("container: %q, esperado ztr (o .ztr solto ganha)", unit.Container)
	}
	if unit.Primary != "txtres/resident/book/txtres_us.ztr" {
		t.Errorf("primary: %q", unit.Primary)
	}
	if unit.Files["fr"] != "txtres/resident/book/txtres_fr.ztr" {
		t.Errorf("files[fr]: %q", unit.Files["fr"])
	}

	// Unidade inexistente.
	if _, err := resolveTextUnit(dir, parts.Get("part1"), "txtres/resident/book/nope"); err == nil {
		t.Error("resolveTextUnit de unidade inexistente devolveu erro nil")
	}
}
