package service

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
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
		if got := FilelistDataPath(c.zone, c.in); got != c.want {
			t.Errorf("FilelistDataPath(%q, %q) = %q, quer %q", c.zone, c.in, got, c.want)
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
		// ck/uk/ru entram no catálogo exclusivo do FF13: o arquivo é uma
		// referência da mesma unidade (não uma unidade própria).
		{"txtres/resident/book/txtres_ck.ztr", "txtres/resident/book/txtres", "ck", true},
		{"txtres/resident/book/txtres_uk.ztr", "txtres/resident/book/txtres", "uk", true},
		{"txtres/resident/book/txtres_ru.ztr", "txtres/resident/book/txtres", "ru", true},
		{"gui/resident/system.win32.imgb", "", "", false},
		{"db/ai/npc/pack/auto_c01.clb", "", "", false},
	}
	for _, c := range cases {
		id, lang, ok := FilelistUnitID(c.path)
		if ok != c.ok || id != c.id || lang != c.lang {
			t.Errorf("FilelistUnitID(%q) = (%q, %q, %v), quer (%q, %q, %v)",
				c.path, id, lang, ok, c.id, c.lang, c.ok)
		}
	}
}

// TestListDirNodesText confere que a listagem marca as folhas de TEXTO com o
// id da unidade (e o tipo do container), sem ler nada do white bin.
func TestListDirNodesText(t *testing.T) {
	entries := []IndexEntry{
		{Path: "db/ai/npc/pack/auto_c01_us.bin"},
		{Path: "txtres/resident/book/txtres_us.ztr"},
		{Path: "gui/resident/system.win32.imgb"},
	}
	nodes := ListDirNodes(parts.Get("part1"), BuildIndexTree(entries), "db/ai/npc/pack", "")
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
	ZoneNodes := ListDirNodes(parts.Get("part1"), BuildIndexTree(entries), "txtres/resident/book", "z000")
	if len(ZoneNodes) != 1 {
		t.Fatalf("nós da zona: %d, quer 1 (%+v)", len(ZoneNodes), ZoneNodes)
	}
	if ZoneNodes[0].UnitID != "zone/z000/txtres/resident/book/txtres" {
		t.Errorf("unitId da zona: %q", ZoneNodes[0].UnitID)
	}
	if ZoneNodes[0].Container != "ztr" {
		t.Errorf("container da zona: %q", ZoneNodes[0].Container)
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

	src := &SourceIndex{Units: map[string]map[string]UnitSource{
		"db/ai/npc/pack/auto_yus": {
			"us": {Rel: vpath, Bin: bin, Coll: coll, VPath: vpath},
		},
	}}

	unit, err := ResolveUnitWithSources(dataDir, parts.Get("part3"), src, "db/ai/npc/pack/auto_yus")
	if err != nil {
		t.Fatalf("ResolveUnitWithSources: %v", err)
	}
	if unit.Primary != vpath || unit.Container != "wpd" {
		t.Fatalf("unidade: primary=%q container=%q", unit.Primary, unit.Container)
	}
	// O container bruto vem do white bin...
	raw, err := ReadSource(unit.Sources[unit.PrimaryLang].Source)
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}
	f, err := wpd.Read(raw)
	if err != nil {
		t.Fatalf("wpd.Read: %v", err)
	}
	if len(f.Header.Records) != 2 {
		t.Fatalf("registros: %d, quer 2", len(f.Header.Records))
	}
	// ...e o texto é o registro .ztr dentro dele.
	text, err := ReadUnitFile(unit, unit.PrimaryLang)
	if err != nil {
		t.Fatalf("ReadUnitFile: %v", err)
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
	t.Setenv("FF_RESOURCES_DATA", filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods"))

	p := parts.Get("part3")
	dataDir, err := config.PartRoot("data", "part3")
	if err != nil {
		t.Fatalf("partRoot: %v", err)
	}
	src := PartSources(p)
	if len(src.Units) == 0 {
		t.Skip("sem white bin: nenhum índice com container")
	}

	// Acha unidades de container WPD que NÃO foram extraídas para data/ e
	// confere que o texto sai do white bin.
	var candidates []string
	for id, langs := range src.Units {
		s, ok := langs["us"]
		if !ok || !strings.HasSuffix(s.Rel, ".bin") {
			continue
		}
		if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(s.Rel))); err == nil {
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
		unit, err := ResolveUnit(dataDir, p, id)
		if err != nil {
			t.Fatalf("ResolveUnit(%s): %v", id, err)
		}
		if unit.Primary == "" || unit.Container != "wpd" {
			t.Fatalf("unidade %s: primary=%q container=%q", id, unit.Primary, unit.Container)
		}
		text, err := ReadUnitFile(unit, unit.PrimaryLang)
		if err != nil {
			// Um .bin que não é container WPD (o "filelist_sound_pack") não
			// tem texto: é o caso "fora do escopo", não uma falha.
			if strings.Contains(err.Error(), "sem registro .ztr") {
				notText++
				continue
			}
			t.Fatalf("ReadUnitFile(%s): %v", id, err)
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

// TestUnitFromWhiteBinRoundTrip confere a unidade vinda do white bin de ponta
// a ponta no nível de FORMATO: o pristine sai do container, a leitura alinha
// as linhas e o salvar reempacota o container com os registros não-texto
// intactos.
func TestUnitFromWhiteBinRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	vpath := "db/ai/npc/pack/auto_yus_us.bin"
	ids := []string{"id1", "id2"}
	wdb := []byte("WDB-BYTES-ORIGINAIS")
	container := testWpdBytesWith(t, ids, wdb)
	bin, coll := buildWhiteBin(t, vpath, container)

	src := &SourceIndex{Units: map[string]map[string]UnitSource{
		"db/ai/npc/pack/auto_yus": {
			"us": {Rel: vpath, Bin: bin, Coll: coll, VPath: vpath},
		},
	}}

	dataDir := filepath.Join(root, "data", "part1")
	unit, err := ResolveUnitWithSources(dataDir, parts.Get("part1"), src, "db/ai/npc/pack/auto_yus")
	if err != nil {
		t.Fatalf("ResolveUnitWithSources: %v", err)
	}
	content, err := ReadUnitContent("part1", unit)
	if err != nil {
		t.Fatalf("ReadUnitContent: %v", err)
	}
	if len(content.IDs) != 2 {
		t.Fatalf("rows: %d, quer 2", len(content.IDs))
	}
	if got := content.Orig["id1"]; got != "Hello" {
		t.Errorf("original[id1]: %q, quer %q (o pristine do white bin)", got, "Hello")
	}

	modsDir := filepath.Join(root, "mods", "part1")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ks, err := TextCodec("part1")
	if err != nil {
		t.Fatalf("TextCodec: %v", err)
	}
	blob, err := PackWpdText(modsDir, unit, ks, map[string]string{"id1": "Ola", "id2": "Mundo"})
	if err != nil {
		t.Fatalf("PackWpdText: %v", err)
	}

	f, err := wpd.Read(blob)
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
func injectSources(t *testing.T, part string, src *SourceIndex) {
	t.Helper()
	key := part + "|" + config.GameRoot(part)
	sourcesMu.Lock()
	sourcesCache[key] = src
	sourcesMu.Unlock()
	t.Cleanup(func() {
		sourcesMu.Lock()
		delete(sourcesCache, key)
		sourcesMu.Unlock()
	})
}
