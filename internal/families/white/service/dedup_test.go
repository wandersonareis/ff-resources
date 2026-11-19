package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/shared/dto"
)

// TestExportArtifactDedupAndMetadata confere que o FF13 usa o MESMO motor de
// serialização do shared: metadata por unidade (key/id/row_count/files) e
// dedup de texto repetido em ref "$hash", com a identidade por chave.
func TestExportArtifactDedupAndMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	texts := []string{"Repetido no jogo inteiro"}
	writeFile(t, filepath.Join(root, "data", "part1", "a", "txtres_us.ztr"), testZtrBlob(t, ids, texts))
	writeFile(t, filepath.Join(root, "data", "part1", "b", "txtres_us.ztr"), testZtrBlob(t, ids, texts))

	var svc Service
	paths, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ztr"}}, "part1", nil, "json", dto.ExportOptions{})
	if err != nil {
		t.Fatalf("ExportArtifactBatch: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths: %v", paths)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]struct {
		Metadata struct {
			Key      string            `json:"key"`
			ID       string            `json:"id"`
			RowCount int               `json:"row_count"`
			Files    map[string]string `json:"files"`
		} `json:"metadata"`
		Rows []struct {
			Key  string            `json:"key"`
			Text map[string]string `json:"text"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(doc) != 2 {
		t.Fatalf("entradas: %d, quer 2 (%s)", len(doc), raw)
	}

	var literal, refs int
	for id, entry := range doc {
		if entry.Metadata.Key == "" || entry.Metadata.ID == "" {
			t.Errorf("%s: metadata incompleta: %+v", id, entry.Metadata)
		}
		if entry.Metadata.RowCount != 1 {
			t.Errorf("%s: row_count=%d, quer 1", id, entry.Metadata.RowCount)
		}
		if len(entry.Metadata.Files) == 0 {
			t.Errorf("%s: metadata.files vazio", id)
		}
		if len(entry.Rows) != 1 {
			t.Fatalf("%s: rows=%d", id, len(entry.Rows))
		}
		if entry.Rows[0].Key == "" {
			t.Errorf("%s: row sem Key (identidade por chave)", id)
		}
		text := entry.Rows[0].Text["us"]
		if text == "Repetido no jogo inteiro" {
			literal++
		} else if len(text) > 0 && text[0] == '$' {
			refs++
		} else {
			t.Errorf("%s: texto inesperado %q", id, text)
		}
	}
	if literal != 1 || refs != 1 {
		t.Fatalf("dedup: literal=%d refs=%d, quer 1/1 (%s)", literal, refs, raw)
	}
}
