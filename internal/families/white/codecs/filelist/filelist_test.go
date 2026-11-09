package filelist

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// goldenCase descreve um JSON de referência gerado pelo WhiteBinTools.exe e o
// game code correspondente.
type goldenCase struct {
	file string
	game GameCode
}

var goldenCases = []goldenCase{
	{"ff131_scru.bin.json", GameFF131},
	{"ff132_lr_a.bin.json", GameFF132},
}

// TestGoldenJSON valida o parser, o escritor JSON, o builder binário e a
// criptografia contra arquivos gerados pelo WhiteBinTools original.
func TestGoldenJSON(t *testing.T) {
	for _, tc := range goldenCases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			golden, err := os.ReadFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatalf("lendo golden: %v", err)
			}

			coll, err := ReadJSON(golden, tc.game)
			if err != nil {
				t.Fatalf("ReadJSON: %v", err)
			}

			// 1) A conversão Collection -> JSON deve reproduzir o golden byte a byte.
			var got bytes.Buffer
			if err := coll.WriteJSON(&got); err != nil {
				t.Fatalf("WriteJSON: %v", err)
			}
			if !bytes.Equal(got.Bytes(), golden) {
				t.Fatalf("JSON gerado difere do golden (%d vs %d bytes)", got.Len(), len(golden))
			}

			// 2) Collection -> binário -> Collection deve preservar o conteúdo.
			bin, err := coll.Build()
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			coll2, err := Parse(bin, tc.game)
			if err != nil {
				t.Fatalf("Parse(Build): %v", err)
			}
			if !reflect.DeepEqual(coll, coll2) {
				t.Fatalf("round-trip binário alterou a coleção")
			}

			// 3) O JSON de coll2 também deve ser igual ao golden.
			var got2 bytes.Buffer
			if err := coll2.WriteJSON(&got2); err != nil {
				t.Fatalf("WriteJSON(2): %v", err)
			}
			if !bytes.Equal(got2.Bytes(), golden) {
				t.Fatalf("JSON após round-trip difere do golden")
			}
		})
	}
}

// TestRealFilelists testa filelists reais de PC (Steam), PS3 (disco) e
// Xbox 360 (RGH), se estiverem presentes. O mesmo parser e serializador é
// usado para as três plataformas.
func TestRealFilelists(t *testing.T) {
	cases := []struct {
		name string
		path string
		game GameCode
		want int
	}{
		{"ff13-1", `D:\Steam\steamapps\common\FINAL FANTASY XIII\white_data\sys\filelistu.win32.bin`, GameFF131, 14107},
		{"ff13-2", `D:\Steam\steamapps\common\FINAL FANTASY XIII-2\alba_data\sys\filelistu.win32.bin`, GameFF132, 14381},
		{"ff13-lr", `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\filelista.win32.bin`, GameFF132, 6280},
		// PS3 (FF13-2, do disco): mesmo formato/criptografia do PC.
		{"ff13-2-ps3", `H:\PS3_GAME\USRDIR\alba_data\sys\filelistu.ps3.bin`, GameFF132, 26332},
		{"ff13-2-ps3-scr", `H:\PS3_GAME\USRDIR\alba_data\sys\filelist_scru.ps3.bin`, GameFF132, 340},
		// Xbox 360 (FF13-2, RGH): idem.
		{"ff13-2-x360", `D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\fileliste.x360.bin`, GameFF132, 20040},
		{"ff13-2-x360-scr", `D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\filelist_scre.x360.bin`, GameFF132, 7047},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.path)
			if err != nil {
				t.Skipf("filelist Steam não encontrado: %v", err)
			}

			coll, err := Parse(data, tc.game)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if coll.FileCount() != tc.want {
				t.Fatalf("fileCount = %d, esperado %d", coll.FileCount(), tc.want)
			}

			// binário -> binário
			bin, err := coll.Build()
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			coll2, err := Parse(bin, tc.game)
			if err != nil {
				t.Fatalf("Parse(Build): %v", err)
			}
			if !reflect.DeepEqual(coll, coll2) {
				t.Fatalf("round-trip binário alterou a coleção")
			}

			// JSON -> JSON
			var j1, j2 bytes.Buffer
			if err := coll.WriteJSON(&j1); err != nil {
				t.Fatal(err)
			}
			coll3, err := ReadJSON(j1.Bytes(), tc.game)
			if err != nil {
				t.Fatalf("ReadJSON: %v", err)
			}
			if err := coll3.WriteJSON(&j2); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(j1.Bytes(), j2.Bytes()) {
				t.Fatalf("round-trip JSON alterou o conteúdo")
			}
		})
	}
}

func TestBuildRepeatedFiles(t *testing.T) {
	coll := &Collection{
		GameCode: GameFF131,
		Chunks: []Chunk{
			{ID: 0, Entries: []Entry{
				{FileCode: 1, FileInfo: "10:100:90:foo/bar.bin"},
				{FileCode: 2, FileInfo: "20:200:180:foo/other.bin"},
			}},
			{ID: 1, Entries: []Entry{
				{FileCode: 1, FileInfo: "30:100:90:foo/bar.bin"},
				{FileCode: 3, FileInfo: "40:300:280:foo/bar.bin"},
			}},
		},
	}

	dups := coll.DuplicatedPaths()
	if dups["foo/bar.bin"] != 3 {
		t.Fatalf("esperava 3 ocorrências de foo/bar.bin, veio %v", dups)
	}

	res, err := coll.AppendFile("foo/bar.bin", 5000, 0x120, 0x100)
	if err != nil {
		t.Fatal(err)
	}
	if res.AlignedOffset != 6144 || res.PosSectors != 3 {
		t.Fatalf("append planejado errado: %+v", res)
	}
	if res.UpdatedEntries != 3 || res.DuplicateOccurrences != 3 {
		t.Fatalf("deveria atualizar as 3 repetições: %+v", res)
	}

	for _, ref := range coll.FindByPath("foo/bar.bin") {
		loc, err := ParseFileInfo(coll.Chunks[ref.Chunk].Entries[ref.Entry].FileInfo)
		if err != nil {
			t.Fatal(err)
		}
		if loc.PosSectors != 3 || loc.UncmpSize != 0x120 || loc.CmpSize != 0x100 || loc.Path != "foo/bar.bin" {
			t.Fatalf("repetição não atualizada: %+v", loc)
		}
	}

	// A entrada de outro caminho não pode ter sido tocada.
	loc, _ := ParseFileInfo(coll.Chunks[0].Entries[1].FileInfo)
	if loc.PosSectors != 0x20 || loc.UncmpSize != 0x200 || loc.CmpSize != 0x180 {
		t.Fatalf("entrada não relacionada foi alterada: %+v", loc)
	}

	if _, err := coll.SetAddressByPath("nao/existe.bin", 1, 1, 1); err == nil {
		t.Fatal("esperava erro para caminho inexistente")
	}
}

func TestAlignUp2048(t *testing.T) {
	cases := []struct{ in, want int64 }{
		{0, 0},
		{1, 2048},
		{2048, 2048},
		{2049, 4096},
		{5000, 6144},
	}
	for _, c := range cases {
		if got := AlignUp2048(c.in); got != c.want {
			t.Fatalf("AlignUp2048(%d) = %d, quer %d", c.in, got, c.want)
		}
	}
	if off, sec := PlanAppend(4096); off != 4096 || sec != 2 {
		t.Fatalf("PlanAppend(4096) = %d,%d", off, sec)
	}
}

func TestParseGameCode(t *testing.T) {
	for _, s := range []string{"ff131", "-ff131", "FF132", "-ff132", "dirge"} {
		if _, err := ParseGameCode(s); err != nil {
			t.Fatalf("ParseGameCode(%q): %v", s, err)
		}
	}
	if _, err := ParseGameCode("ff999"); err == nil {
		t.Fatal("esperava erro para game code inválido")
	}
}

func TestFileInfoRoundTrip(t *testing.T) {
	info := "1a2b3c:1000:800:txtres/zone/z0060/txtres_sp.ztr"
	loc, err := ParseFileInfo(info)
	if err != nil {
		t.Fatal(err)
	}
	if loc.PosSectors != 0x1a2b3c || loc.UncmpSize != 0x1000 || loc.CmpSize != 0x800 {
		t.Fatalf("parse errado: %+v", loc)
	}
	if got := FormatFileInfo(loc); got != info {
		t.Fatalf("FormatFileInfo = %q, quer %q", got, info)
	}
	if !strings.HasPrefix(EntryPath(info), "txtres/") {
		t.Fatalf("EntryPath errado: %q", EntryPath(info))
	}
}

func TestEncryptedRoundTripSynthetic(t *testing.T) {
	coll := &Collection{
		GameCode: GameFF132,
		Crypt: CryptInfo{
			Encrypted:     true,
			SeedA:         0x1122334455667788,
			SeedB:         0x99AABBCCDDEEFF00,
			EncryptionTag: EncryptionTagConstant,
		},
		Chunks: []Chunk{
			{ID: 0, Entries: []Entry{{FileCode: 7, FileTypeID: 16, FileInfo: "0:10:8:a.bin"}}},
			{ID: 1, Entries: []Entry{{FileCode: 8, FileTypeID: 32, FileInfo: "2:20:18:b.bin"}}},
			{ID: 2, Entries: []Entry{{FileCode: 9, FileTypeID: 0, FileInfo: "4:30:28:c.bin"}}},
		},
	}

	bin, err := coll.Build()
	if err != nil {
		t.Fatal(err)
	}
	// Deve ser detectado como criptografado.
	if len(bin) < 24 || !coll.Crypt.Encrypted {
		t.Fatal("estado inesperado")
	}
	got, err := Parse(bin, GameFF132)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(coll, got) {
		t.Fatalf("round-trip criptografado falhou")
	}
}
