package service

import (
	"os"
	"path/filepath"
	"testing"

	formatters "ffresources/internal/shared/artifact"
	"ffresources/internal/shared/dto"

	"ffresources/internal/families/white/codecs/ztr"
)

// TestExportFilelistDecodeAndExtract fecha o fluxo do Exportar quando a fonte
// é o FILELIST:
//   - Descompactar gera o artefato a partir do texto do white bin (não de
//     data/);
//   - Extrair grava o binário REAL (descomprimido) em data/partN.
func TestExportFilelistDecodeAndExtract(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	unitID := "txtres/resident/book/txtres"
	filelistPath := filepath.Join(gameRoot, "white_data", "sys", "filelistu.win32.bin")
	binPath := filepath.Join(gameRoot, "white_data", "sys", "white_imgu.win32.bin")

	pristine := testZtrBlob(t, []string{"id1", "id2"}, []string{"File1", "File2"})
	bin := make([]byte, 4096)
	copy(bin, pristine)
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filelistPath, testFilelistData(t,
		testEntry(unitID+"_us.ztr", 0, uint32(len(pristine)))), 0o644); err != nil {
		t.Fatal(err)
	}
	// /data NÃO tem a unidade: o export precisa sair do white bin.
	invalidateAfterFilelistWrite("part1")

	svc := &Service{}
	paths, err := svc.ExportArtifactBatch(
		[]ExportPlanItem{{Kind: "resident", Source: "filelist"}},
		"part1", nil, "json",
		dto.ExportOptions{Decode: true, Extract: true},
	)
	if err != nil {
		t.Fatalf("ExportArtifactBatch: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v, esperado artefato + binário", paths)
	}

	// Descompactar: o artefato tem o texto do white bin.
	var artifact string
	for _, p := range paths {
		if filepath.Ext(p) == ".json" {
			artifact = p
		}
	}
	if artifact == "" {
		t.Fatalf("artefato JSON não encontrado em %v", paths)
	}
	f, ok := formatters.Get("json")
	if !ok {
		t.Fatal("formatter json não registrado")
	}
	c, err := formatters.NewReader(f, artifact, artifactOptions(nil)).Read()
	if err != nil {
		t.Fatalf("lendo artefato: %v", err)
	}
	entry, ok := c[unitID]
	if !ok {
		t.Fatalf("artefato sem a unidade %s: %v", unitID, c.SortedKeys())
	}
	if len(entry.Rows) != 2 || entry.Rows[0].Text["us"] != "File1" {
		t.Fatalf("row do artefato = %+v", entry.Rows)
	}

	// Extrair: o binário descomprimido caiu em data/partN.
	extracted := filepath.Join(os.Getenv("FF_RESOURCES_DATA"), "part1", filepath.FromSlash(unitID)+"_us.ztr")
	if _, err := os.Stat(extracted); err != nil {
		t.Fatalf("binário não extraído em data/: %v", err)
	}
	raw, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatal(err)
	}
	zf, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("ztr extraído inválido: %v", err)
	}
	ks, err := TextCodec("part1")
	if err != nil {
		t.Fatal(err)
	}
	texts, err := ks.DecodeLines(zf.Lines)
	if err != nil {
		t.Fatal(err)
	}
	if len(texts) != 2 || texts[0] != "File1" {
		t.Fatalf("texto extraído = %v", texts)
	}

	// Já existe: um segundo Extrair pula (não sobrescreve).
	paths2, err := svc.ExportArtifactBatch(
		[]ExportPlanItem{{Kind: "resident", Source: "filelist"}},
		"part1", nil, "json",
		dto.ExportOptions{Extract: true},
	)
	if err != nil {
		t.Fatalf("segundo Extract: %v", err)
	}
	for _, p := range paths2 {
		if p == extracted {
			t.Fatalf("binário existente foi reescrito: %s", p)
		}
	}
}

// TestExtractImageFilesFromFilelist confere o Extrair de imagem: o arquivo do
// nó vem do filelist; refs de data/ são recusadas.
func TestExtractImageFilesFromFilelist(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	filelistPath := filepath.Join(gameRoot, "white_data", "sys", "filelistu.win32.bin")
	binPath := filepath.Join(gameRoot, "white_data", "sys", "white_imgu.win32.bin")

	payload := []byte("IMAGEM-CRU-DO-FILELIST")
	bin := make([]byte, 4096)
	copy(bin, payload)
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filelistPath, testFilelistData(t,
		testEntry("gui/test.win32.dds", 0, uint32(len(payload)))), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	filelistRef := EncodeImageRef(ImageRef{
		Source:   "filelist",
		Root:     filelistPath,
		Path:     "gui/test.win32.dds",
		Platform: "win32",
	})
	dataRef := EncodeImageRef(ImageRef{
		Source: "data",
		Path:   "gui/test.win32.dds",
	})

	svc := &Service{}
	res, err := svc.ExtractImageFiles("part1", []string{filelistRef, dataRef}, "", false)
	if err != nil {
		t.Fatalf("ExtractImageFiles: %v", err)
	}
	if len(res.Done) != 1 {
		t.Fatalf("done = %v, esperado 1", res.Done)
	}
	if len(res.Failed) != 1 {
		t.Fatalf("failed = %v, esperado 1 (ref de data/)", res.Failed)
	}
	got, err := os.ReadFile(res.Done[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("bytes extraídos = %q", got)
	}
}
