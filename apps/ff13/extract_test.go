package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestExtractIndexEntriesGlobal confere que um índice global grava em
// data/partN/<caminho interno>.
func TestExtractIndexEntriesGlobal(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data", "part1")
	payload := []byte("GLOBAL-CONTENT")
	vpath := "txtres/resident/book/txtres_us.ztr"

	bin, coll := buildWhiteBin(t, vpath, payload)
	idx := &filelistIndex{Container: bin, Coll: coll}

	res := extractIndexEntriesToData(dataDir, "", idx, []indexEntry{{Path: vpath}}, false)
	if len(res.Errors) > 0 {
		t.Fatalf("erros: %v", res.Errors)
	}
	if res.Written != 1 || res.Requested != 1 {
		t.Fatalf("resultado: %+v", res)
	}

	got, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(vpath)))
	if err != nil {
		t.Fatalf("leitura: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("conteúdo: %q, quer %q", got, payload)
	}
}

// TestExtractIndexEntriesZone confere que um índice de zona grava em
// data/partN/zone/<z>/<caminho relativo> — o caminho canônico do app.
func TestExtractIndexEntriesZone(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data", "part1")
	payload := []byte("ZONE-CONTENT")
	vpath := "txtres/resident/book/txtres_us.ztr"

	bin, coll := buildWhiteBin(t, vpath, payload)
	idx := &filelistIndex{Container: bin, Coll: coll}

	res := extractIndexEntriesToData(dataDir, "z000", idx, []indexEntry{{Path: vpath}}, false)
	if len(res.Errors) > 0 {
		t.Fatalf("erros: %v", res.Errors)
	}
	if res.Written != 1 {
		t.Fatalf("resultado: %+v", res)
	}

	got, err := os.ReadFile(filepath.Join(dataDir, "zone", "z000", filepath.FromSlash(vpath)))
	if err != nil {
		t.Fatalf("leitura: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("conteúdo: %q, quer %q", got, payload)
	}
}

// TestExtractIndexEntriesSkipsExisting confere o force=false/true.
func TestExtractIndexEntriesSkipsExisting(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data", "part1")
	payload := []byte("ONE")
	vpath := "txtres/x_us.ztr"
	bin, coll := buildWhiteBin(t, vpath, payload)
	idx := &filelistIndex{Container: bin, Coll: coll}
	entries := []indexEntry{{Path: vpath}}

	first := extractIndexEntriesToData(dataDir, "", idx, entries, false)
	if first.Written != 1 || first.Skipped != 0 {
		t.Fatalf("primeira extração: %+v", first)
	}
	second := extractIndexEntriesToData(dataDir, "", idx, entries, false)
	if second.Written != 0 || second.Skipped != 1 {
		t.Fatalf("segunda extração: %+v", second)
	}
	forced := extractIndexEntriesToData(dataDir, "", idx, entries, true)
	if forced.Written != 1 || forced.Skipped != 0 {
		t.Fatalf("extração forçada: %+v", forced)
	}
}

// TestSelectExtractEntries confere a seleção por arquivo e por diretório.
func TestSelectExtractEntries(t *testing.T) {
	entries := []indexEntry{
		{Path: "txtres/a/x_us.ztr"},
		{Path: "txtres/a/y_us.ztr"},
		{Path: "txtres/b/z_us.ztr"},
	}

	byFile := selectExtractEntries(entries, []string{"txtres/a/x_us.ztr"})
	if len(byFile) != 1 || byFile[0].Path != "txtres/a/x_us.ztr" {
		t.Fatalf("seleção por arquivo: %+v", byFile)
	}

	byDir := selectExtractEntries(entries, []string{"txtres/a"})
	if len(byDir) != 2 {
		t.Fatalf("seleção por diretório: %+v", byDir)
	}

	if got := selectExtractEntries(entries, []string{"txtres/b/z_us.ztr"}); len(got) != 1 {
		t.Fatalf("seleção com barras invertidas: %+v", got)
	}
}
