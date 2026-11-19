package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/dto"
)

// testFilelistData monta o binário de um filelist V1 com as entradas dadas.
func testFilelistData(t *testing.T, entries ...fl.Entry) []byte {
	t.Helper()
	coll := &fl.Collection{
		GameCode: fl.GameFF131,
		Chunks:   []fl.Chunk{{ID: 0, Entries: entries}},
	}
	data, err := coll.Build()
	if err != nil {
		t.Fatalf("Build do filelist de teste: %v", err)
	}
	return data
}

func testEntry(path string, posSectors uint64, size uint32) fl.Entry {
	return fl.Entry{
		FileCode: 1,
		FileInfo: fl.FormatFileInfo(fl.FileLocation{
			PosSectors: posSectors,
			UncmpSize:  size,
			CmpSize:    size,
			Path:       path,
		}),
	}
}

// setupInsertEnv monta uma instalação sintética (jogo + mods temporários) e
// devolve as raízes.
func setupInsertEnv(t *testing.T) (gameRoot, modsRoot string) {
	t.Helper()
	base := t.TempDir()
	gameRoot = filepath.Join(base, "game")
	modsRoot = filepath.Join(base, "mods")
	dataRoot := filepath.Join(base, "data")
	for _, dir := range []string{
		filepath.Join(gameRoot, "white_data", "sys"),
		filepath.Join(gameRoot, "white_data", "zone"),
		filepath.Join(modsRoot, "part1"),
		filepath.Join(dataRoot, "part1"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FF_RESOURCES_GAME_ROOT", gameRoot)
	t.Setenv("FF_RESOURCES_MODS", modsRoot)
	t.Setenv("FF_RESOURCES_DATA", dataRoot)
	invalidateAfterFilelistWrite("part1")
	t.Cleanup(func() { invalidateAfterFilelistWrite("part1") })
	return gameRoot, modsRoot
}

// TestInsertBlobRepointsAndRestores cobre o fluxo completo de um índex em
// disco: append no white bin, reapontamento, manifesto e reverter.
func TestInsertBlobRepointsAndRestores(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)

	filelistPath := filepath.Join(gameRoot, "white_data", "sys", "filelistu.win32.bin")
	binPath := filepath.Join(gameRoot, "white_data", "sys", "white_imgu.win32.bin")

	originalList := testFilelistData(t, testEntry("txtres/test_us.ztr", 0, 4))
	if err := os.WriteFile(filelistPath, originalList, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := make([]byte, 4096)
	copy(bin, []byte("orig"))
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	svc := &Service{}
	outs, err := svc.InsertBlobs("part1", []InsertItem{{
		RelPath: "txtres/test_us.ztr",
		Data:    []byte("novo"),
	}})
	if err != nil {
		t.Fatalf("InsertBlobs: %v", err)
	}
	if len(outs) != 1 || !outs[0].Inserted {
		t.Fatalf("outcome = %+v, esperado inserted", outs)
	}

	// O white bin cresceu no fim, alinhado a 2048.
	st, err := os.Stat(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 4096+4 {
		t.Fatalf("tamanho do bin = %d, esperado 4100", st.Size())
	}

	// O filelist em disco aponta para o setor novo.
	raw, err := os.ReadFile(filelistPath)
	if err != nil {
		t.Fatal(err)
	}
	coll, err := fl.Parse(raw, fl.GameFF131)
	if err != nil {
		t.Fatal(err)
	}
	refs := coll.FindByPath("txtres/test_us.ztr")
	if len(refs) != 1 {
		t.Fatalf("refs = %d, esperado 1", len(refs))
	}
	loc, err := fl.ParseFileInfo(coll.Chunks[refs[0].Chunk].Entries[refs[0].Entry].FileInfo)
	if err != nil {
		t.Fatal(err)
	}
	if loc.PosSectors != 2 || loc.UncmpSize != 4 || loc.CmpSize != 4 {
		t.Fatalf("entrada = %+v, esperado pos 2 / 4 / 4", loc)
	}

	// E os bytes são os inseridos.
	got, err := whitebin.ExtractAt(binPath, loc.PosSectors, loc.UncmpSize, loc.CmpSize)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("novo")) {
		t.Fatalf("bytes = %q, esperado novo", got)
	}

	// Reverter volta o bin ao tamanho original e o filelist ao backup.
	res, err := svc.RestoreFilelistInserts("part1")
	if err != nil {
		t.Fatalf("RestoreFilelistInserts: %v", err)
	}
	if res.Bins != 1 || res.Filelists != 1 {
		t.Fatalf("restore = %+v, esperado 1 bin e 1 filelist", res)
	}
	st, err = os.Stat(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 4096 {
		t.Fatalf("tamanho após restore = %d, esperado 4096", st.Size())
	}
	raw, err = os.ReadFile(filelistPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, originalList) {
		t.Fatal("filelist não voltou ao conteúdo original")
	}
}

// TestInsertBlobDuplicatedPath garante que TODAS as ocorrências do mesmo
// caminho (endereços diferentes) são reapontadas.
func TestInsertBlobDuplicatedPath(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	filelistPath := filepath.Join(gameRoot, "white_data", "sys", "filelistu.win32.bin")
	binPath := filepath.Join(gameRoot, "white_data", "sys", "white_imgu.win32.bin")

	list := testFilelistData(t,
		testEntry("txtres/dup_us.ztr", 0, 4),
		testEntry("txtres/dup_us.ztr", 1, 4),
	)
	if err := os.WriteFile(filelistPath, list, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := make([]byte, 4096)
	copy(bin, []byte("orig"))
	copy(bin[2048:], []byte("orig"))
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	outs, err := (&Service{}).InsertBlobs("part1", []InsertItem{{
		RelPath: "txtres/dup_us.ztr",
		Data:    []byte("novo"),
	}})
	if err != nil {
		t.Fatalf("InsertBlobs: %v", err)
	}
	if outs[0].Entries != 2 {
		t.Fatalf("entries = %d, esperado 2", outs[0].Entries)
	}

	raw, _ := os.ReadFile(filelistPath)
	coll, err := fl.Parse(raw, fl.GameFF131)
	if err != nil {
		t.Fatal(err)
	}
	refs := coll.FindByPath("txtres/dup_us.ztr")
	if len(refs) != 2 {
		t.Fatalf("refs = %d, esperado 2", len(refs))
	}
	for _, ref := range refs {
		loc, err := fl.ParseFileInfo(coll.Chunks[ref.Chunk].Entries[ref.Entry].FileInfo)
		if err != nil {
			t.Fatal(err)
		}
		if loc.PosSectors != 2 {
			t.Fatalf("ocorrência %+v não foi reapontada", loc)
		}
	}
}

// TestInsertBlobNotIndexed devolve motivo sem erro (o chamador transforma em
// aviso/erro).
func TestInsertBlobNotIndexed(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	filelistPath := filepath.Join(gameRoot, "white_data", "sys", "filelistu.win32.bin")
	binPath := filepath.Join(gameRoot, "white_data", "sys", "white_imgu.win32.bin")
	if err := os.WriteFile(filelistPath, testFilelistData(t, testEntry("txtres/test_us.ztr", 0, 4)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	out, err := (&Service{}).InsertBlobs("part1", []InsertItem{{
		RelPath: "txtres/nao_existe_us.ztr",
		Data:    []byte("x"),
	}})
	if err != nil {
		t.Fatalf("InsertBlobs: %v", err)
	}
	if len(out) != 1 || out[0].Inserted || out[0].Reason == "" {
		t.Fatalf("outcome = %+v, esperado não indexado com motivo", out)
	}
}

// TestFilelistImageReadIsSourceAware garante o contrato do navegador: uma ref
// de FILELIST lê do white bin do índice, não da cópia (possivelmente velha)
// de data/. É o que faz o arquivo inserido aparecer no filelist.
func TestFilelistImageReadIsSourceAware(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	sysDir := filepath.Join(gameRoot, "white_data", "sys")
	filelistPath := filepath.Join(sysDir, "filelistu.win32.bin")
	binPath := filepath.Join(sysDir, "white_imgu.win32.bin")

	list := testFilelistData(t, testEntry("gui/test.win32.imgb", 0, 4))
	if err := os.WriteFile(filelistPath, list, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := make([]byte, 2048)
	copy(bin, []byte("bin!"))
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	// Cópia de data/ DIFERENTE: a ref de filelist não pode servi-la.
	dataCopy := filepath.Join(os.Getenv("FF_RESOURCES_DATA"), "part1", "gui", "test.win32.imgb")
	if err := os.MkdirAll(filepath.Dir(dataCopy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataCopy, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	f, err := ResolveImageFile("part1", ImageRef{
		Source:   "filelist",
		Root:     filelistPath,
		Path:     "gui/test.win32.imgb",
		Platform: "win32",
	})
	if err != nil {
		t.Fatalf("ResolveImageFile: %v", err)
	}
	got, err := f.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(got, []byte("bin!")) {
		t.Fatalf("ref de filelist leu %q; esperado bin! (não a cópia de data/)", got)
	}
}

// TestImageDeliverInsertOnly fecha o fluxo do import com "Inserir" apenas:
// o binário vai para o filelist, nada é gravado em mods/ e a leitura do
// filelist passa a devolver o conteúdo inserido.
func TestImageDeliverInsertOnly(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)
	sysDir := filepath.Join(gameRoot, "white_data", "sys")
	filelistPath := filepath.Join(sysDir, "filelistu.win32.bin")
	binPath := filepath.Join(sysDir, "white_imgu.win32.bin")

	if err := os.WriteFile(filelistPath,
		testFilelistData(t, testEntry("gui/test.win32.imgb", 0, 4)), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := make([]byte, 2048)
	copy(bin, []byte("orig"))
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	acc := newImageImportBatch()
	if err := acc.addDirect("gui/test.win32.imgb", []byte("novo"), "gui/test.win32.imgb.dds"); err != nil {
		t.Fatal(err)
	}
	updated, failed, warnings := acc.deliver(&Service{}, "part1", dto.ImageImportOptions{
		WriteMods:      false,
		InsertFilelist: true,
	})
	if len(failed) > 0 || len(warnings) > 0 {
		t.Fatalf("failed=%v warnings=%v", failed, warnings)
	}
	if len(updated) != 1 || updated[0] != "gui/test.win32.imgb.dds" {
		t.Fatalf("updated = %v", updated)
	}

	// Inserir sem Comprimir: nada em mods/.
	modsCopy := filepath.Join(os.Getenv("FF_RESOURCES_MODS"), "part1", "gui", "test.win32.imgb")
	if _, err := os.Stat(modsCopy); !os.IsNotExist(err) {
		t.Fatalf("mods/ não deveria ter o binário no modo insert-only (err=%v)", err)
	}

	// A leitura do filelist devolve o conteúdo inserido.
	invalidateAfterFilelistWrite("part1")
	f, err := ResolveImageFile("part1", ImageRef{
		Source:   "filelist",
		Root:     filelistPath,
		Path:     "gui/test.win32.imgb",
		Platform: "win32",
	})
	if err != nil {
		t.Fatalf("ResolveImageFile: %v", err)
	}
	got, err := f.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(got, []byte("novo")) {
		t.Fatalf("filelist leu %q, esperado novo", got)
	}
}

// TestInsertEmbeddedChain cobre a cadeia do part1: o filelist de zona mora
// dentro do white bin global e indexa um bin de zona separado.
func TestInsertEmbeddedChain(t *testing.T) {
	gameRoot, _ := setupInsertEnv(t)

	sysDir := filepath.Join(gameRoot, "white_data", "sys")
	zoneDir := filepath.Join(gameRoot, "white_data", "zone")
	parentListPath := filepath.Join(sysDir, "filelistu.win32.bin")
	parentBinPath := filepath.Join(sysDir, "white_imgu.win32.bin")
	zoneBinPath := filepath.Join(zoneDir, "white_z000u_img.win32.bin")

	// Índice de zona: uma entrada que aponta para o bin de zona.
	childBytes := testFilelistData(t, testEntry("txtres/zone_us.ztr", 0, 4))

	// Bin do pai: o arquivo do índice de zona no setor 0.
	parentBin := make([]byte, 4096)
	copy(parentBin, childBytes)
	if err := os.WriteFile(parentBinPath, parentBin, 0o644); err != nil {
		t.Fatal(err)
	}

	// Índice global declara o filelist de zona.
	parentList := testFilelistData(t,
		testEntry("zone/filelist_z000u.win32.bin", 0, uint32(len(childBytes))),
	)
	if err := os.WriteFile(parentListPath, parentList, 0o644); err != nil {
		t.Fatal(err)
	}

	// Bin da zona: dado original no setor 0.
	zoneBin := make([]byte, 4096)
	copy(zoneBin, []byte("orig"))
	if err := os.WriteFile(zoneBinPath, zoneBin, 0o644); err != nil {
		t.Fatal(err)
	}
	invalidateAfterFilelistWrite("part1")

	outs, err := (&Service{}).InsertBlobs("part1", []InsertItem{{
		RelPath: "zone/z000/txtres/zone_us.ztr",
		Data:    []byte("novo"),
	}})
	if err != nil {
		t.Fatalf("InsertBlobs (cadeia): %v", err)
	}
	if len(outs) != 1 || !outs[0].Inserted {
		t.Fatalf("outcome = %+v, esperado inserted", outs)
	}

	// O bin de zona cresceu.
	st, err := os.Stat(zoneBinPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 4100 {
		t.Fatalf("bin de zona = %d, esperado 4100", st.Size())
	}
	// O bin do pai ganhou a nova cópia do índice de zona.
	st, err = os.Stat(parentBinPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() <= 4096 {
		t.Fatalf("bin do pai = %d, esperado > 4096", st.Size())
	}

	// Redescobre a cadeia: o índice embutido aponta para os bytes novos.
	invalidateAfterFilelistWrite("part1")
	disc := DiscoverPart(parts.Get("part1"))
	var child *FilelistIndex
	for _, ix := range disc.Indexes {
		if ix.Virtual == "zone/filelist_z000u.win32.bin" {
			child = ix
			break
		}
	}
	if child == nil {
		t.Fatal("índice de zona não redescoberto")
	}
	got, err := whitebin.ExtractBytes(zoneBinPath, "txtres/zone_us.ztr", child.Coll)
	if err != nil {
		t.Fatalf("extract do dado inserido: %v", err)
	}
	if !bytes.Equal(got, []byte("novo")) {
		t.Fatalf("dado da zona = %q, esperado novo", got)
	}

	// Reverter volta os dois bins ao tamanho original.
	if _, err := (&Service{}).RestoreFilelistInserts("part1"); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for _, p := range []string{zoneBinPath, parentBinPath} {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if st.Size() != 4096 {
			t.Fatalf("%s após restore = %d, esperado 4096", filepath.Base(p), st.Size())
		}
	}
	invalidateAfterFilelistWrite("part1")
	disc = DiscoverPart(parts.Get("part1"))
	for _, ix := range disc.Indexes {
		if ix.Virtual == "zone/filelist_z000u.win32.bin" {
			child = ix
			break
		}
	}
	if child == nil {
		t.Fatal("índice de zona não redescoberto após restore")
	}
	got, err = whitebin.ExtractBytes(zoneBinPath, "txtres/zone_us.ztr", child.Coll)
	if err != nil {
		t.Fatalf("extract após restore: %v", err)
	}
	if !bytes.Equal(got, []byte("orig")) {
		t.Fatalf("dado da zona após restore = %q, esperado orig", got)
	}
}
