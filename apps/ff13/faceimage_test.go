package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResidentFaceImage(t *testing.T) {
	root := t.TempDir()
	rel := "gui/resident/face/face001.win32.imgb"
	disk := filepath.Join(root, "part2", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(disk), 0o755); err != nil {
		t.Fatal(err)
	}
	// DXT1 128x256: bloco 4x4 = 8 bytes. Preenche todos os blocos com preto.
	raw := make([]byte, residentFaceRawSize)
	if err := os.WriteFile(disk, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FF_RESOURCES_DATA", root)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ref := encodeImageRef(imageRef{Source: "data", Path: rel, Platform: "win32"})
	entry, err := (FF13Service{}).GetImageEntry("part2", ref)
	if err != nil {
		t.Fatalf("GetImageEntry(face): %v", err)
	}
	if entry.Width != residentFaceWidth || entry.Height != residentFaceHeight || entry.PNGData == "" {
		t.Fatalf("face: %dx%d png=%d", entry.Width, entry.Height, len(entry.PNGData))
	}
	if entry.Format != "DXT1" {
		t.Fatalf("formato=%q, quer DXT1", entry.Format)
	}

	entries, err := (FF13Service{}).ListDataEntries("part2")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.ID == rel && e.Image {
			if e.ImageContainer {
				t.Fatalf("entrada face não deveria ser container")
			}
			return
		}
	}
	t.Fatalf("entrada face não apareceu no inventário como imagem")
}
