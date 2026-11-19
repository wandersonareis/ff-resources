package service

import (
	"os"
	"path/filepath"
	"testing"

	formatters "ffresources/internal/shared/artifact"
	"ffresources/internal/shared/dto"

	"ffresources/internal/families/white/codecs/ztr"
)

// writeTextArtifact grava a Collection no MESMO perfil do export do FF13.
func writeTextArtifact(t *testing.T, path string, c dto.Collection) {
	t.Helper()
	f, ok := formatters.Get("json")
	if !ok {
		t.Fatal("formatter json não registrado")
	}
	if _, err := formatters.NewWriter(f, c, path, textArtifactOptions()).Write(); err != nil {
		t.Fatalf("gravando artefato: %v", err)
	}
}

func textUnitCollection(unitID, part string, rows ...dto.TextRow) dto.Collection {
	return dto.Collection{
		unitID: {
			Metadata: dto.Metadata{Key: part + "/" + unitID, ID: unitID},
			Rows:     rows,
		},
	}
}

func decodeUnitText(t *testing.T, blob []byte) []string {
	t.Helper()
	zf, err := ztr.Read(blob)
	if err != nil {
		t.Fatalf("ztr.Read: %v", err)
	}
	ks, err := TextCodec("part1")
	if err != nil {
		t.Fatalf("TextCodec: %v", err)
	}
	texts, err := ks.DecodeLines(zf.Lines)
	if err != nil {
		t.Fatalf("DecodeLines: %v", err)
	}
	return texts
}

// TestTextImportModsPartial cobre o fluxo "Comprimir": import parcial por
// chave (só as rows com Target mudam), gravação em mods/ e preview.
func TestTextImportModsPartial(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	// Sem isso o appconfig real apontaria para a instalação do jogo.
	t.Setenv("FF_RESOURCES_GAME_ROOT", "")
	t.Setenv("FF_RESOURCES_CONFIG", filepath.Join(root, "sem-config.json"))

	unitID := "txtres/resident/book/txtres"
	writeFile(t, filepath.Join(root, "data", "part1", unitID+"_us.ztr"),
		testZtrBlob(t, []string{"id1", "id2"}, []string{"Hello", "World"}))

	artifact := filepath.Join(TranslatedTextRoot("part1"), "resident.json")
	writeTextArtifact(t, artifact, textUnitCollection(
		unitID, "part1",
		dto.TextRow{Index: 0, Name: "id1", Source: "Hello", Target: "Ola"},
		// id2 fica de fora do artefato (ou com target vazio): preserva o atual.
		dto.TextRow{Index: 1, Name: "id2", Source: "World"},
	))

	svc := &Service{}
	preview, err := svc.PreviewImportTranslated("part1", []string{"resident"})
	if err != nil {
		t.Fatalf("PreviewImportTranslated: %v", err)
	}
	if preview.EntryCount != 1 || preview.ChangedTexts != 1 {
		t.Fatalf("preview = %+v, esperado 1 unidade / 1 texto", preview)
	}

	batch, err := svc.ImportTranslated("part1", []string{"resident"},
		dto.TextImportOptions{WriteMods: true})
	if err != nil {
		t.Fatalf("ImportTranslated: %v", err)
	}
	if batch.ChangedTexts != 1 {
		t.Fatalf("changed = %d, esperado 1", batch.ChangedTexts)
	}

	mods := filepath.Join(root, "mods", "part1", unitID+"_us.ztr")
	raw, err := os.ReadFile(mods)
	if err != nil {
		t.Fatalf("mods não gravado: %v", err)
	}
	if got := decodeUnitText(t, raw); len(got) != 2 || got[0] != "Ola" || got[1] != "World" {
		t.Fatalf("textos em mods = %v, esperado [Ola World]", got)
	}
}

// TestTextImportInsertOnlyValidatedSource fecha o caso do filelist: o
// pristine vem do white bin (fonte validada), o /data tem OUTRO texto, e o
// Inserir apenas não grava em mods/.
func TestTextImportInsertOnlyValidatedSource(t *testing.T) {
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
	// /data tem OUTRO texto: se o import misturar fontes, o resultado sai
	// errado (o esperado é o texto do filelist).
	writeFile(t, filepath.Join(os.Getenv("FF_RESOURCES_DATA"), "part1", unitID+"_us.ztr"),
		testZtrBlob(t, []string{"id1", "id2"}, []string{"Data1", "Data2"}))
	invalidateAfterFilelistWrite("part1")

	artifact := filepath.Join(TranslatedTextRoot("part1"), "resident.json")
	writeTextArtifact(t, artifact, textUnitCollection(
		unitID, "part1",
		dto.TextRow{Index: 0, Name: "id1", Source: "File1", Target: "Novo1"},
		dto.TextRow{Index: 1, Name: "id2", Source: "File2", Target: "Novo2"},
	))

	svc := &Service{}
	if _, err := svc.ImportTranslated("part1", []string{"resident"}, dto.TextImportOptions{
		WriteMods:      false,
		InsertFilelist: true,
	}); err != nil {
		t.Fatalf("ImportTranslated (insert-only): %v", err)
	}

	// Inserir sem Comprimir: nada em mods/.
	mods := filepath.Join(os.Getenv("FF_RESOURCES_MODS"), "part1", unitID+"_us.ztr")
	if _, err := os.Stat(mods); !os.IsNotExist(err) {
		t.Fatalf("mods/ não deveria ter o binário no insert-only (err=%v)", err)
	}

	// A leitura pela ORIGEM filelist devolve o texto inserido.
	entry, err := svc.GetTextEntry("part1", unitID, "", filelistPath, unitID+"_us.ztr")
	if err != nil {
		t.Fatalf("GetTextEntry(filelist): %v", err)
	}
	if len(entry.Rows) != 2 {
		t.Fatalf("rows = %d, esperado 2", len(entry.Rows))
	}
	if got := entry.Rows[0].Text["us"]; got != "Novo1" {
		t.Fatalf("filelist text = %q, esperado Novo1", got)
	}
	if got := entry.Rows[0].Original["us"]; got != "Novo1" {
		t.Fatalf("filelist original = %q, esperado Novo1", got)
	}

	// A leitura pela origem /data continua mostrando o texto de data/.
	dataEntry, err := svc.GetTextEntry("part1", unitID, "", "", "")
	if err != nil {
		t.Fatalf("GetTextEntry(data): %v", err)
	}
	if got := dataEntry.Rows[0].Original["us"]; got != "Data1" {
		t.Fatalf("data original = %q, esperado Data1", got)
	}
}

// TestTextImportSourceAwareFilelist confirma que o unit resolvido pelo
// filelist não usa o cache da coleção de data/.
func TestTextImportSourceAwareFilelist(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	unitID := "txtres/resident/book/txtres"
	filelistPath := filepath.Join(gameRoot, "white_data", "sys", "filelistu.win32.bin")
	binPath := filepath.Join(gameRoot, "white_data", "sys", "white_imgu.win32.bin")

	pristine := testZtrBlob(t, []string{"id1"}, []string{"Filelist"})
	bin := make([]byte, 2048)
	copy(bin, pristine)
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filelistPath, testFilelistData(t,
		testEntry(unitID+"_us.ztr", 0, uint32(len(pristine)))), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(os.Getenv("FF_RESOURCES_DATA"), "part1", unitID+"_us.ztr"),
		testZtrBlob(t, []string{"id1"}, []string{"Data"}))
	invalidateAfterFilelistWrite("part1")

	unit, err := ResolveFilelistTextUnit("part1", filelistPath, unitID+"_us.ztr", unitID)
	if err != nil {
		t.Fatalf("ResolveFilelistTextUnit: %v", err)
	}
	content, err := ReadUnitPrimaryContent("part1", unit)
	if err != nil {
		t.Fatalf("ReadUnitPrimaryContent: %v", err)
	}
	if got := content.Orig["id1"]; got != "Filelist" {
		t.Fatalf("pristino = %q, esperado Filelist (não data/)", got)
	}
	if unit.Sources["us"].Source.IsDisk() {
		t.Fatalf("fonte validada deveria ser o white bin, veio %+v", unit.Sources["us"])
	}
}
