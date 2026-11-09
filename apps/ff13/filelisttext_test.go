package main

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ffresources/apps/ff13/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/shared/dto"
)

// TestFilelistDataPath confere o mapeamento do caminho INDEXADO para o
// caminho do layout de dados (data/partN): o índice global usa o caminho
// como é e o índice de zona usa o caminho relativo à zona.
func TestFilelistDataPath(t *testing.T) {
	cases := []struct {
		zone string
		in   string
		want string
	}{
		{"", "db/ai/npc/pack/auto_c01_us.bin", "db/ai/npc/pack/auto_c01_us.bin"},
		{"", "TXTRES/Resident/Book/TXTRES_US.ZTR", "txtres/resident/book/txtres_us.ztr"},
		{"z000", "txtres/resident/book/txtres_us.ztr", "zone/z000/txtres/resident/book/txtres_us.ztr"},
		{"z000", "zone/z000/txtres/resident/book/txtres_us.ztr", "zone/z000/txtres/resident/book/txtres_us.ztr"},
		{"z000", "\\txtres\\resident\\book\\txtres_us.ztr", "zone/z000/txtres/resident/book/txtres_us.ztr"},
		{"", "zone/z000/scr000.clb", "zone/z000/scr000.clb"},
	}
	for _, c := range cases {
		if got := filelistDataPath(c.zone, c.in); got != c.want {
			t.Errorf("filelistDataPath(%q, %q) = %q, quer %q", c.zone, c.in, got, c.want)
		}
	}
}

// TestFilelistUnitID confere o id da unidade (e o idioma) de uma entrada
// indexada.
func TestFilelistUnitID(t *testing.T) {
	cases := []struct {
		path string
		id   string
		lang string
		ok   bool
	}{
		{"db/ai/npc/pack/auto_c01_us.bin", "db/ai/npc/pack/auto_c01", "us", true},
		{"txtres/resident/book/txtres_us.ztr", "txtres/resident/book/txtres", "us", true},
		{"zone/z000/txtres/x_us.ztr", "zone/z000/txtres/x", "us", true},
		// "ck" está fora do catálogo de idiomas: o arquivo não tem idioma e
		// a unidade não tem primário (não é editável).
		{"txtres/resident/book/txtres_ck.ztr", "txtres/resident/book/txtres_ck", "", true},
		{"gui/resident/system.win32.imgb", "", "", false},
		{"db/ai/npc/pack/auto_c01.clb", "", "", false},
	}
	for _, c := range cases {
		id, lang, ok := filelistUnitID(c.path)
		if ok != c.ok || id != c.id || lang != c.lang {
			t.Errorf("filelistUnitID(%q) = (%q, %q, %v), quer (%q, %q, %v)",
				c.path, id, lang, ok, c.id, c.lang, c.ok)
		}
	}
}

// TestListDirNodesText confere que a listagem marca as folhas de TEXTO com o
// id da unidade (e o tipo do container), sem ler nada do white bin.
func TestListDirNodesText(t *testing.T) {
	entries := []indexEntry{
		{Path: "db/ai/npc/pack/auto_c01_us.bin"},
		{Path: "txtres/resident/book/txtres_us.ztr"},
		{Path: "gui/resident/system.win32.imgb"},
	}
	nodes := listDirNodes(parts.Get("part1"), buildIndexTree(entries), "db/ai/npc/pack", "")
	if len(nodes) != 1 {
		t.Fatalf("nós: %d, quer 1 (%+v)", len(nodes), nodes)
	}
	if nodes[0].UnitID != "db/ai/npc/pack/auto_c01" {
		t.Errorf("unitId: %q", nodes[0].UnitID)
	}
	if !nodes[0].Text || nodes[0].Container != "wpd" {
		t.Errorf("text=%v container=%q, quer true/wpd", nodes[0].Text, nodes[0].Container)
	}

	// O índice de ZONA põe o prefixo "zone/<z>/" no caminho de dados.
	zoneNodes := listDirNodes(parts.Get("part1"), buildIndexTree(entries), "txtres/resident/book", "z000")
	if len(zoneNodes) != 1 {
		t.Fatalf("nós da zona: %d, quer 1 (%+v)", len(zoneNodes), zoneNodes)
	}
	if zoneNodes[0].UnitID != "zone/z000/txtres/resident/book/txtres" {
		t.Errorf("unitId da zona: %q", zoneNodes[0].UnitID)
	}
	if zoneNodes[0].Container != "ztr" {
		t.Errorf("container da zona: %q", zoneNodes[0].Container)
	}
}

// TestResolveUnitFromWhiteBin confere a unidade que NÃO está em data/partN:
// os bytes saem do white bin (índice de fontes injetado).
func TestResolveUnitFromWhiteBin(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data", "part1")
	vpath := "db/ai/npc/pack/auto_yus_us.bin"
	ids := []string{"id1", "id2"}
	container := testWpdBytes(t, ids)
	bin, coll := buildWhiteBin(t, vpath, container)

	src := &sourceIndex{units: map[string]map[string]unitSource{
		"db/ai/npc/pack/auto_yus": {
			"us": {rel: vpath, bin: bin, coll: coll, vpath: vpath},
		},
	}}

	unit, err := resolveUnitWithSources(dataDir, parts.Get("part3"), src, "db/ai/npc/pack/auto_yus")
	if err != nil {
		t.Fatalf("resolveUnitWithSources: %v", err)
	}
	if unit.Primary != vpath || unit.Container != "wpd" {
		t.Fatalf("unidade: primary=%q container=%q", unit.Primary, unit.Container)
	}
	// O container bruto vem do white bin...
	raw, err := readSource(unit.files[unit.PrimaryLang].src)
	if err != nil {
		t.Fatalf("readSource: %v", err)
	}
	f, err := wpd.Read(raw)
	if err != nil {
		t.Fatalf("wpd.Read: %v", err)
	}
	if len(f.Header.Records) != 2 {
		t.Fatalf("registros: %d, quer 2", len(f.Header.Records))
	}
	// ...e o texto é o registro .ztr dentro dele.
	text, err := readUnitFile(unit, unit.PrimaryLang)
	if err != nil {
		t.Fatalf("readUnitFile: %v", err)
	}
	if got := decodeTestZtr(t, text); got[0] != "Hello" || got[1] != "World" {
		t.Errorf("texto: %v, quer [Hello World]", got)
	}
}

// TestResolveUnitRealWhiteBin confere (com o jogo instalado) a unidade que
// NÃO está em data/partN: ela é resolvida pelo white bin e o texto sai de
// lá.
func TestResolveUnitRealWhiteBin(t *testing.T) {
	testGameRoot(t, "part3")
	t.Setenv("FF_RESOURCES_DATA", filepath.Join("bin", "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join("bin", "data", "mods"))

	p := parts.Get("part3")
	dataDir, err := partRoot("data", "part3")
	if err != nil {
		t.Fatalf("partRoot: %v", err)
	}
	src := partSources(p)
	if len(src.units) == 0 {
		t.Skip("sem white bin: nenhum índice com container")
	}

	// Acha unidades de container WPD que NÃO foram extraídas para data/ e
	// confere que o texto sai do white bin.
	var candidates []string
	for id, langs := range src.units {
		s, ok := langs["us"]
		if !ok || !strings.HasSuffix(s.rel, ".bin") {
			continue
		}
		if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(s.rel))); err == nil {
			continue
		}
		candidates = append(candidates, id)
	}
	if len(candidates) == 0 {
		t.Skip("nenhum container WPD ausente de data/part3")
	}
	sort.Strings(candidates)

	var read, empty, notText int
	for _, id := range candidates {
		unit, err := resolveUnit(dataDir, p, id)
		if err != nil {
			t.Fatalf("resolveUnit(%s): %v", id, err)
		}
		if unit.Primary == "" || unit.Container != "wpd" {
			t.Fatalf("unidade %s: primary=%q container=%q", id, unit.Primary, unit.Container)
		}
		text, err := readUnitFile(unit, unit.PrimaryLang)
		if err != nil {
			// Um .bin que não é container WPD (o "filelist_sound_pack") não
			// tem texto: é o caso "fora do escopo", não uma falha.
			if strings.Contains(err.Error(), "sem registro .ztr") {
				notText++
				continue
			}
			t.Fatalf("readUnitFile(%s): %v", id, err)
		}
		zf, err := ztr.Read(text)
		if err != nil {
			t.Fatalf("ztr.Read(%s): %v", id, err)
		}
		if len(zf.IDs) == 0 {
			empty++
			continue
		}
		read++
		t.Logf("unidade do white bin: %s (%d linhas)", id, len(zf.IDs))
	}
	if read == 0 {
		t.Skipf("nenhuma unidade com texto entre %d containers (%d vazios, %d sem texto)",
			len(candidates), empty, notText)
	}
}

// TestGetTextEntryFromWhiteBin confere o serviço inteiro com a unidade vinda
// do white bin: a tabela abre (pristine do white bin) e o salvar grava o
// container reempacotado em mods/ — com os registros não-texto intactos.
func TestGetTextEntryFromWhiteBin(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	vpath := "db/ai/npc/pack/auto_yus_us.bin"
	ids := []string{"id1", "id2"}
	wdb := []byte("WDB-BYTES-ORIGINAIS")
	container := testWpdBytesWith(t, ids, wdb)
	bin, coll := buildWhiteBin(t, vpath, container)

	injectSources(t, "part1", &sourceIndex{units: map[string]map[string]unitSource{
		"db/ai/npc/pack/auto_yus": {
			"us": {rel: vpath, bin: bin, coll: coll, vpath: vpath},
		},
	}})

	var svc FF13Service
	entry, err := svc.GetTextEntry("part1", "db/ai/npc/pack/auto_yus", "")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if len(entry.Rows) != 2 {
		t.Fatalf("rows: %d, quer 2", len(entry.Rows))
	}
	if got := entry.Rows[0].Original["us"]; got != "Hello" {
		t.Errorf("original[us]: %q, quer %q (o pristine do white bin)", got, "Hello")
	}
	if entry.Metadata.ID != "db/ai/npc/pack/auto_yus" {
		t.Errorf("metadata.id: %q", entry.Metadata.ID)
	}

	c := dto.Collection{
		"db/ai/npc/pack/auto_yus": {
			Metadata: dto.Metadata{Key: "part1/db/ai/npc/pack/auto_yus", ID: "db/ai/npc/pack/auto_yus"},
			Rows: []dto.TextRow{
				{Index: 0, Name: "id1", Text: map[string]string{"us": "Ola"}, Target: "Ola"},
				{Index: 1, Name: "id2", Text: map[string]string{"us": "Mundo"}, Target: "Mundo"},
			},
		},
	}
	if err := svc.ApplyTextCollection("part1", c); err != nil {
		t.Fatalf("ApplyTextCollection: %v", err)
	}

	dst := filepath.Join(root, "mods", "part1", filepath.FromSlash(vpath))
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("le %s: %v", dst, err)
	}
	f, err := wpd.Read(raw)
	if err != nil {
		t.Fatalf("wpd.Read: %v", err)
	}
	gotWdb, ok := f.ExtractRecord("auto_yus", "wdb")
	if !ok {
		t.Fatal("registro .wdb ausente")
	}
	if !bytes.Equal(gotWdb, wdb) {
		t.Errorf("registro .wdb alterado: %q, quer %q", gotWdb, wdb)
	}
	gotZtr, ok := f.ExtractRecord("auto_yus_us", "ztr")
	if !ok {
		t.Fatal("registro .ztr ausente")
	}
	texts := decodeTestZtr(t, gotZtr)
	if texts[0] != "Ola" || texts[1] != "Mundo" {
		t.Errorf("texto gravado: %v, quer [Ola Mundo]", texts)
	}
}

// ---- auxiliares ----

// testWpdBytes monta um container WPD com um registro .wdb e um registro .ztr.
func testWpdBytes(t *testing.T, ids []string) []byte {
	t.Helper()
	return testWpdBytesWith(t, ids, []byte("WDB-BYTES-ORIGINAIS"))
}

// testWpdBytesWith monta um container WPD com o registro .wdb pedido e um
// registro .ztr com o texto de teste.
func testWpdBytesWith(t *testing.T, ids []string, wdb []byte) []byte {
	t.Helper()
	blob, err := wpd.WriteFromParts([]wpd.Part{
		{Name: "auto_yus", Extension: "wdb", Data: wdb},
		{Name: "auto_yus_us", Extension: "ztr", Data: testZtrBlob(t, ids, []string{"Hello", "World"})},
	})
	if err != nil {
		t.Fatalf("wpd.WriteFromParts: %v", err)
	}
	return blob
}

// decodeTestZtr decodifica as linhas de um .ztr de teste.
func decodeTestZtr(t *testing.T, raw []byte) []string {
	t.Helper()
	zf, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("ztr.Read: %v", err)
	}
	ks, err := keySet("part1")
	if err != nil {
		t.Fatalf("keySet: %v", err)
	}
	texts, err := ks.DecodeLines(zf.Lines)
	if err != nil {
		t.Fatalf("DecodeLines: %v", err)
	}
	return texts
}

// buildWhiteBin monta um WHITE BIN sintético: um arquivo comprimido (zlib)
// no setor 1 e o filelist que o indexa. Devolve o caminho do bin e a coleção
// já parseada.
func buildWhiteBin(t *testing.T, vpath string, payload []byte) (string, *fl.Collection) {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		t.Fatalf("zlib.Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib.Close: %v", err)
	}
	cmp := buf.Bytes()
	if len(cmp) == len(payload) {
		t.Fatal("payload comprimido do mesmo tamanho: o teste precisa de um arquivo comprimido")
	}
	bin := make([]byte, 2048)
	bin = append(bin, cmp...)

	loc := fl.FileLocation{
		PosSectors: 1,
		UncmpSize:  uint32(len(payload)),
		CmpSize:    uint32(len(cmp)),
		Path:       vpath,
	}
	coll := &fl.Collection{
		GameCode: fl.GameFF131,
		Chunks: []fl.Chunk{{
			ID: 0,
			Entries: []fl.Entry{
				{FileCode: 1, FileInfo: fl.FormatFileInfo(loc)},
			},
		}},
	}
	raw, err := coll.Build()
	if err != nil {
		t.Fatalf("Collection.Build: %v", err)
	}
	parsed, err := fl.Parse(raw, fl.GameFF131)
	if err != nil {
		t.Fatalf("filelist.Parse: %v", err)
	}
	path := filepath.Join(t.TempDir(), "white.bin")
	writeFile(t, path, bin)
	return path, parsed
}

// injectSources coloca um índice de fontes no cache da part (o teste não
// tem o jogo instalado: o cache real viria vazio).
func injectSources(t *testing.T, part string, src *sourceIndex) {
	t.Helper()
	key := part + "|" + gameRoot(part)
	sourcesMu.Lock()
	sourcesCache[key] = src
	sourcesMu.Unlock()
	t.Cleanup(func() {
		sourcesMu.Lock()
		delete(sourcesCache, key)
		sourcesMu.Unlock()
	})
}
