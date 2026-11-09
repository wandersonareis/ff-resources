package whitebin_test

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
)

func loadCollection(t *testing.T, path string, g filelist.GameCode) *filelist.Collection {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("filelist não encontrado: %v", err)
	}
	coll, err := filelist.Parse(data, g)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return coll
}

// independentDecompress reproduz a leitura crua do white bin + zlib, sem usar
// o pacote whitebin, para conferir o resultado da extração.
func independentDecompress(t *testing.T, whiteBin string, coll *filelist.Collection, vpath string) []byte {
	t.Helper()
	refs := coll.FindByPath(vpath)
	if len(refs) == 0 {
		t.Fatalf("caminho ausente: %s", vpath)
	}
	e := coll.Chunks[refs[0].Chunk].Entries[refs[0].Entry]
	loc, err := filelist.ParseFileInfo(e.FileInfo)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(whiteBin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sr := io.NewSectionReader(f, int64(loc.PosSectors)*2048, int64(loc.CmpSize))
	if loc.IsCompressed() {
		zr, err := zlib.NewReader(sr)
		if err != nil {
			t.Fatal(err)
		}
		defer zr.Close()
		out, err := io.ReadAll(zr)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	out, err := io.ReadAll(sr)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestExtractSingleFileLR(t *testing.T) {
	const (
		fl    = `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\filelist_scra.win32.bin`
		white = `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\white_scra.win32.bin`
		vpath = "sys/script/fld/navi.clb"
	)
	coll := loadCollection(t, fl, filelist.GameFF132)
	if _, err := os.Stat(white); err != nil {
		t.Skipf("white bin ausente: %v", err)
	}

	dir := t.TempDir()
	results, err := whitebin.ExtractFile(white, dir, vpath, coll)
	if err != nil {
		t.Fatalf("ExtractFile: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, veio %d", len(results))
	}
	r := results[0]
	if !r.Compressed || r.State != whitebin.Decompressed {
		t.Fatalf("esperava Decompressed, veio %+v", r)
	}

	got, err := os.ReadFile(r.OutFile)
	if err != nil {
		t.Fatal(err)
	}
	if uint32(len(got)) != r.UncmpSize {
		t.Fatalf("tamanho %d != UncmpSize %d", len(got), r.UncmpSize)
	}
	want := independentDecompress(t, white, coll, vpath)
	if !bytes.Equal(got, want) {
		t.Fatalf("conteúdo extraído difere da leitura independente")
	}
}

func TestExtractSingleFilePS3(t *testing.T) {
	const (
		fl    = `H:\PS3_GAME\USRDIR\alba_data\sys\filelistu.ps3.bin`
		white = `H:\PS3_GAME\USRDIR\alba_data\sys\white_imgu.ps3.bin`
		vpath = "chr/fa/f006/bin/f006.ps3.trb"
	)
	coll := loadCollection(t, fl, filelist.GameFF132)
	if _, err := os.Stat(white); err != nil {
		t.Skipf("white bin PS3 ausente: %v", err)
	}

	dir := t.TempDir()
	results, err := whitebin.ExtractFile(white, dir, vpath, coll)
	if err != nil {
		t.Fatalf("ExtractFile: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, veio %d", len(results))
	}
	r := results[0]
	got, err := os.ReadFile(r.OutFile)
	if err != nil {
		t.Fatal(err)
	}
	if uint32(len(got)) != r.UncmpSize {
		t.Fatalf("tamanho %d != UncmpSize %d", len(got), r.UncmpSize)
	}
	if !bytes.HasPrefix(got, []byte("SEDBRES ")) {
		t.Fatalf("magic inesperado: % x", got[:8])
	}
	want := independentDecompress(t, white, coll, vpath)
	if !bytes.Equal(got, want) {
		t.Fatalf("conteúdo extraído difere da leitura independente")
	}
}

func TestExtractSingleFileXbox360(t *testing.T) {
	const (
		fl    = `D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\fileliste.x360.bin`
		white = `D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\white_imge.x360.bin`
		vpath = "chr/exte/e047/bin/e047.x360.trb"
	)
	coll := loadCollection(t, fl, filelist.GameFF132)
	if _, err := os.Stat(white); err != nil {
		t.Skipf("white bin Xbox 360 ausente: %v", err)
	}

	dir := t.TempDir()
	results, err := whitebin.ExtractFile(white, dir, vpath, coll)
	if err != nil {
		t.Fatalf("ExtractFile: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, veio %d", len(results))
	}
	r := results[0]
	got, err := os.ReadFile(r.OutFile)
	if err != nil {
		t.Fatal(err)
	}
	if uint32(len(got)) != r.UncmpSize {
		t.Fatalf("tamanho %d != UncmpSize %d", len(got), r.UncmpSize)
	}
	if !bytes.HasPrefix(got, []byte("SEDBRES ")) {
		t.Fatalf("magic inesperado: % x", got[:8])
	}
	want := independentDecompress(t, white, coll, vpath)
	if !bytes.Equal(got, want) {
		t.Fatalf("conteúdo extraído difere da leitura independente")
	}
}

func TestExtractUnknownPath(t *testing.T) {
	const fl = `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\filelist_scra.win32.bin`
	coll := loadCollection(t, fl, filelist.GameFF132)
	if _, err := whitebin.ExtractFile(fl, t.TempDir(), "nao/existe.bin", coll); err == nil {
		t.Fatal("esperava erro para caminho inexistente")
	}
}

func TestExtractWritesNestedDirs(t *testing.T) {
	const (
		fl    = `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\filelist_scra.win32.bin`
		white = `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\white_scra.win32.bin`
		vpath = "sys/script/fld/navi.clb"
	)
	coll := loadCollection(t, fl, filelist.GameFF132)
	if _, err := os.Stat(white); err != nil {
		t.Skipf("white bin ausente: %v", err)
	}
	dir := t.TempDir()
	results, err := whitebin.ExtractFile(white, dir, vpath, coll)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "sys", "script", "fld", "navi.clb")
	if results[0].OutFile != want {
		t.Fatalf("OutFile = %q, quer %q", results[0].OutFile, want)
	}
}

// syntheticWhiteBin monta um WHITE BIN sintético: cada arquivo é comprimido
// (zlib) e colocado no seu setor (2048 bytes). Devolve o caminho do bin e o
// filelist que o indexa.
func syntheticWhiteBin(t *testing.T, files map[string][]byte) (string, *filelist.Collection) {
	t.Helper()
	type item struct {
		vpath   string
		payload []byte
	}
	items := make([]item, 0, len(files))
	for vpath, payload := range files {
		items = append(items, item{vpath, payload})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].vpath < items[j].vpath })

	bin := make([]byte, 2048)
	entries := make([]filelist.Entry, 0, len(items))
	for i, it := range items {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write(it.payload); err != nil {
			t.Fatalf("zlib.Write: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("zlib.Close: %v", err)
		}
		cmp := buf.Bytes()
		if len(cmp) == len(it.payload) {
			t.Fatalf("payload de %s não comprimiu: o teste precisa de um arquivo comprimido", it.vpath)
		}
		// O setor 0 fica vazio: o primeiro arquivo começa no setor 1.
		for len(bin)%2048 != 0 {
			bin = append(bin, 0)
		}
		pos := uint64(len(bin) / 2048)
		bin = append(bin, cmp...)
		loc := filelist.FileLocation{
			PosSectors: pos,
			UncmpSize:  uint32(len(it.payload)),
			CmpSize:    uint32(len(cmp)),
			Path:       it.vpath,
		}
		entries = append(entries, filelist.Entry{FileCode: uint32(i + 1), FileInfo: filelist.FormatFileInfo(loc)})
	}

	coll := &filelist.Collection{
		GameCode: filelist.GameFF131,
		Chunks:   []filelist.Chunk{{ID: 0, Entries: entries}},
	}
	raw, err := coll.Build()
	if err != nil {
		t.Fatalf("Collection.Build: %v", err)
	}
	parsed, err := filelist.Parse(raw, filelist.GameFF131)
	if err != nil {
		t.Fatalf("filelist.Parse: %v", err)
	}
	path := filepath.Join(t.TempDir(), "white.bin")
	if err := os.WriteFile(path, bin, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, parsed
}

// TestExtractBytesSynthetic confere a leitura em memória: os bytes saem do
// white bin sem gravar nada em disco.
func TestExtractBytesSynthetic(t *testing.T) {
	payload := []byte("TEXTO DE TESTE DO WHITE BIN")
	bin, coll := syntheticWhiteBin(t, map[string][]byte{"txtres/x_us.ztr": payload})

	got, err := whitebin.ExtractBytes(bin, "txtres/x_us.ztr", coll)
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("bytes: %q, quer %q", got, payload)
	}
}

// TestExtractBytesUnknownPath confere o erro de caminho ausente.
func TestExtractBytesUnknownPath(t *testing.T) {
	bin, coll := syntheticWhiteBin(t, map[string][]byte{"txtres/x_us.ztr": []byte("TEXTO")})
	if _, err := whitebin.ExtractBytes(bin, "nao/existe.bin", coll); err == nil {
		t.Fatal("esperava erro para caminho inexistente")
	}
}

// TestExtractBytesMatchesFile confere que a leitura em memória devolve os
// MESMOS bytes que a extração para disco.
func TestExtractBytesMatchesFile(t *testing.T) {
	payload := []byte("TEXTO DE TESTE DO WHITE BIN, COM TAMANHO DIFERENTE DO COMPRIMIDO")
	bin, coll := syntheticWhiteBin(t, map[string][]byte{"txtres/x_us.ztr": payload})

	dir := t.TempDir()
	results, err := whitebin.ExtractFile(bin, dir, "txtres/x_us.ztr", coll)
	if err != nil {
		t.Fatalf("ExtractFile: %v", err)
	}
	onDisk, err := os.ReadFile(results[0].OutFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got, err := whitebin.ExtractBytes(bin, "txtres/x_us.ztr", coll)
	if err != nil {
		t.Fatalf("ExtractBytes: %v", err)
	}
	if !bytes.Equal(got, onDisk) {
		t.Fatal("ExtractBytes difere do ExtractFile")
	}
}
