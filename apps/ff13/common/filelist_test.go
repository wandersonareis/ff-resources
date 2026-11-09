package common_test

import (
	"encoding/json"
	"testing"

	"ffresources/apps/ff13/common"
)

// TestFilelistRootJSONFields trava o CONTRATO do DTO de raiz: os nomes dos
// campos no JSON são a interface com o frontend e não podem mudar por
// casualidade.
func TestFilelistRootJSONFields(t *testing.T) {
	root := common.FilelistRoot{
		Name:      "filelistu.win32.bin",
		Path:      "D:/jogo/sys/filelistu.win32.bin",
		Part:      "part1",
		SourceDir: "white_data",
		Entries:   42,
		WhiteBin:  "D:/jogo/sys/white_imgu.win32.bin",
		Group:     common.GroupZone,
		Zone:      "z000",
		Language:  common.LangU,
		Source:    common.SourceWin32,
		Kind:      common.KindDisk,
	}
	raw, err := json.Marshal(root)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"name":"filelistu.win32.bin","path":"D:/jogo/sys/filelistu.win32.bin",` +
		`"part":"part1","sourceDir":"white_data","entries":42,` +
		`"whiteBin":"D:/jogo/sys/white_imgu.win32.bin","group":"zone","zone":"z000",` +
		`"language":"u","source":"win32","kind":"disk"}`
	if string(raw) != want {
		t.Fatalf("json:\n got %s\nwant %s", raw, want)
	}
}

// TestFilelistRootOptionalFields omite os campos opcionais quando vazios (o
// frontend usa a ausência para distinguir "sem grupo" de "sem par").
func TestFilelistRootOptionalFields(t *testing.T) {
	raw, err := json.Marshal(common.FilelistRoot{
		Name:      "filelistu.win32.bin",
		Path:      "p",
		Part:      "part1",
		SourceDir: "data/part1",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"name":"filelistu.win32.bin","path":"p","part":"part1","sourceDir":"data/part1","entries":0}`
	if string(raw) != want {
		t.Fatalf("json:\n got %s\nwant %s", raw, want)
	}
}

// TestFilelistNodeJSONFields trava o contrato do DTO de nó: name/path crus,
// isDir sempre presente, size/mergeChain/kind/language opcionais.
func TestFilelistNodeJSONFields(t *testing.T) {
	raw, err := json.Marshal(common.FilelistNode{
		Name:  "empty",
		Path:  "empty/empty",
		IsDir: true,
		Kind:  common.KindDir,
		MergeChain: []string{
			"empty", "empty",
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"name":"empty","path":"empty/empty","mergeChain":["empty","empty"],"isDir":true,"kind":"dir"}`
	if string(raw) != want {
		t.Fatalf("json:\n got %s\nwant %s", raw, want)
	}

	raw, err = json.Marshal(common.FilelistNode{
		Name:     "filelist_z000u.win32.bin",
		Path:     "zone/filelist_z000u.win32.bin",
		Size:     1234,
		Kind:     common.KindIndex,
		Language: common.LangU,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want = `{"name":"filelist_z000u.win32.bin","path":"zone/filelist_z000u.win32.bin","isDir":false,"size":1234,"kind":"index","language":"u"}`
	if string(raw) != want {
		t.Fatalf("json:\n got %s\nwant %s", raw, want)
	}
}

// TestSourceOf confere a PLATAFORMA do índice pelo nome do arquivo
// (win32/x360/ps3). Sem plataforma no nome: fonte desconhecida (vazia),
// nunca inventada.
func TestSourceOf(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"filelistu.win32.bin", common.SourceWin32},
		{"filelist_scrc.x360.bin", common.SourceX360},
		{"filelist_patch.PS3.BIN", common.SourcePS3},
		{"filelist_z000u.win32.bin", common.SourceWin32},
		{"FILELISTU.WIN32.BIN", common.SourceWin32},
		{"zone/filelist_z000u.win32.bin", common.SourceWin32},
		{"filelistu.bin", ""},
		{"filelistu.foo.bin", ""},
		{"filelistu", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := common.SourceOf(c.name); got != c.want {
			t.Errorf("SourceOf(%q) = %q, quer %q", c.name, got, c.want)
		}
	}
}

// TestLanguageOf confere o IDIOMA do índice: o nome SEM EXTENSÃO que termina
// em "u" é inglês e em "c" é japonês (regra mecânica).
func TestLanguageOf(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"filelistu.win32.bin", common.LangU},
		{"filelistc.win32.bin", common.LangC},
		{"filelist_scru.win32.bin", common.LangU},
		{"filelist_scrc.win32.bin", common.LangC},
		{"filelist_z000u.win32.bin", common.LangU},
		{"filelist_patch_c.win32.bin", common.LangC},
		{"filelist_patch.win32.bin", ""},
		{"filelistu.bin", common.LangU},
		{"", ""},
	}
	for _, c := range cases {
		if got := common.LanguageOf(c.name); got != c.want {
			t.Errorf("LanguageOf(%q) = %q, quer %q", c.name, got, c.want)
		}
	}
}

// TestZoneIDOf confere o ID da ZONA no nome do índice.
func TestZoneIDOf(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"filelist_z000u.win32.bin", "z000"},
		{"filelist_z106c.win32.bin", "z106"},
		{"zone/filelist_z000u.win32.bin", "z000"},
		{"filelistu.win32.bin", ""},
		{"filelist_scru.win32.bin", ""},
	}
	for _, c := range cases {
		if got := common.ZoneIDOf(c.name); got != c.want {
			t.Errorf("ZoneIDOf(%q) = %q, quer %q", c.name, got, c.want)
		}
	}
}

// TestDataDirName confere a pasta de dados de cada part (e a pasta do app
// onde mora a cópia extraída do índice).
func TestDataDirName(t *testing.T) {
	cases := map[string]string{
		"part1": "white_data",
		"part2": "alba_data",
		"part3": "weiss_data",
		"":      "white_data",
	}
	for part, want := range cases {
		if got := common.DataDirName(part); got != want {
			t.Errorf("DataDirName(%q) = %q, quer %q", part, got, want)
		}
	}
	if got := common.AppDataDir("part2"); got != "data/part2" {
		t.Errorf("AppDataDir(part2) = %q, quer data/part2", got)
	}
}
