package service

import (
	"testing"

	fl "ffresources/internal/families/white/codecs/filelist"
)

func TestKnownKeyOnlyFilelists(t *testing.T) {
	for _, name := range []string{
		"filelist_p0000000img_a.win32.bin",
		"filelist_p0000000img_v.win32.bin",
		"FILELIST_P0000000IMG_A.WIN32.BIN",
	} {
		if !IsKnownKeyOnlyFilelist(name) {
			t.Errorf("%q deveria ser conhecido", name)
		}
	}
	for _, name := range []string{
		"filelist_p0000001img_a.win32.bin",
		"filelista.win32.bin",
		"filelistu.win32.bin",
	} {
		if IsKnownKeyOnlyFilelist(name) {
			t.Errorf("%q NÃO deveria ser conhecido", name)
		}
	}
}

func TestKeyOnlyCollection(t *testing.T) {
	coll := &fl.Collection{Chunks: []fl.Chunk{{Entries: []fl.Entry{{
		FileInfo: "0:10:10:sys/dlc/key/key00000000.dat",
	}}}}}
	if !IsKeyOnlyCollection(coll) {
		t.Fatal("key00000000.dat sozinho deveria ser key-only")
	}

	withOther := &fl.Collection{Chunks: []fl.Chunk{{Entries: []fl.Entry{
		{FileInfo: "0:10:10:sys/dlc/key/key00000000.dat"},
		{FileInfo: "0:10:10:chr/pc/c001/bin/c001.win32.trb"},
	}}}}
	if IsKeyOnlyCollection(withOther) {
		t.Fatal("coleção com outro arquivo NÃO é key-only")
	}

	wrongName := &fl.Collection{Chunks: []fl.Chunk{{Entries: []fl.Entry{{
		FileInfo: "0:10:10:sys/dlc/key/other.dat",
	}}}}}
	if IsKeyOnlyCollection(wrongName) {
		t.Fatal("outro .dat não é key-only")
	}
}
