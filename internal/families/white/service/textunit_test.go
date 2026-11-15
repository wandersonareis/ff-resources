package service

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/families/white/profile/parts"
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
		{"txtres_ck.ztr", "txtres", "ck"},
		{"txtres_uk.ztr", "txtres", "uk"},
		{"txtres_ru.ztr", "txtres", "ru"},
		{"data_backup.bin", "data_backup", ""},
		{"txtres_zz.ztr", "txtres_zz", ""},
		{"book.ztr", "book", ""},
		{"_us.ztr", "_us", ""},
		{"us.ztr", "us", ""},
	}
	for _, c := range cases {
		base, lang := ParseUnitFileName(c.name)
		if base != c.base || lang != c.lang {
			t.Errorf("ParseUnitFileName(%q) = (%q, %q), esperado (%q, %q)", c.name, base, lang, c.base, c.lang)
		}
	}
}

// TestTxtresSuffixesInCatalog trava a regra nos DADOS REAIS: todo sufixo
// txtres_<code>.ztr presente em data/partN precisa estar no catálogo do FF13.
// Sem isso, ck/uk/ru (traduções comunitárias) voltavam a ser unidades próprias
// em vez de referências da unidade.
func TestTxtresSuffixesInCatalog(t *testing.T) {
	for _, part := range []string{"part1", "part2", "part3"} {
		root := partDataRoot(t, part)
		seen := map[string]bool{}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			base := filepath.Base(path)
			if !strings.HasPrefix(base, "txtres_") || !strings.HasSuffix(base, ".ztr") {
				return nil
			}
			code := strings.TrimSuffix(strings.TrimPrefix(base, "txtres_"), ".ztr")
			if len(code) != 2 {
				return nil
			}
			seen[code] = true
			if !IsUnitLang(code) {
				t.Errorf("%s: sufixo %q fora do catálogo (arquivo %s)", part, code, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: varrendo data: %v", part, err)
		}
		codes := make([]string, 0, len(seen))
		for code := range seen {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		t.Logf("%s: sufixos encontrados: %v", part, codes)
	}
}

// testZtrBlob monta um .ztr com os IDs/linhas pedidos (charset do part1).
func testZtrBlob(t *testing.T, ids []string, texts []string) []byte {
	t.Helper()
	ks, err := TextCodec("part1")
	if err != nil {
		t.Fatalf("TextCodec: %v", err)
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

	units, err := ScanTextUnits(dir, parts.Get("part1"))
	if err != nil {
		t.Fatalf("ScanTextUnits: %v", err)
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

	unit, err := ResolveTextUnit(dir, parts.Get("part1"), "txtres/resident/book/txtres")
	if err != nil {
		t.Fatalf("ResolveTextUnit: %v", err)
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
	if _, err := ResolveTextUnit(dir, parts.Get("part1"), "txtres/resident/book/nope"); err == nil {
		t.Error("ResolveTextUnit de unidade inexistente devolveu erro nil")
	}
}
