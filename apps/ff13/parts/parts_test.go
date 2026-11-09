package parts

import "testing"

// TestPairKeys garante que cada jogo pareia filelist <-> white bin com a sua
// propria convencao (part1 normaliza _img/imgu; part3 mantem _a/_v).
func TestPairKeys(t *testing.T) {
	cases := []struct {
		part string
		a    string
		b    string
	}{
		{"part1", "filelistu.win32.bin", "white_imgu.win32.bin"},
		{"part1", "filelistc.win32.bin", "white_imgc.win32.bin"},
		{"part1", "filelist_scru.win32.bin", "white_scru.win32.bin"},
		{"part1", "zone/filelist_z000u.win32.bin", "zone/white_z000u_img.win32.bin"},
		{"part2", "filelist_patch_c.win32.bin", "white_patch_c.win32.bin"},
		{"part2", "filelistu.win32.bin", "white_imgu.win32.bin"},
		{"part3", "filelist_p0000000img_a.win32.bin", "white_p0000000img_a.win32.bin"},
	}
	for _, c := range cases {
		p := Get(c.part)
		ka, okA := p.PairKey(c.a)
		kb, okB := p.PairKey(c.b)
		if !okA || !okB {
			t.Fatalf("%s: PairKey falhou em %q/%q", c.part, c.a, c.b)
		}
		if ka != kb {
			t.Errorf("%s: %q=%q != %q=%q", c.part, c.a, ka, c.b, kb)
		}
	}
}

// TestLanguageOf documenta o idioma do ÍNDICE por part: part1/part2 usam o
// sufixo "u"/"c"; o part3 usa o par "a"/"v" (o idioma vem da chave do par).
func TestLanguageOf(t *testing.T) {
	cases := []struct {
		part string
		file string
		want string
	}{
		{"part1", "filelistu.win32.bin", "u"},
		{"part1", "filelistc.win32.bin", "c"},
		{"part1", "filelist_scru.win32.bin", "u"},
		{"part2", "filelist_patch_c.win32.bin", "c"},
		{"part3", "filelist_p0000000img_a.win32.bin", "u"},
		{"part3", "filelist_p0000000img_v.win32.bin", "c"},
		{"part1", "filelist_scrap.win32.bin", ""},
	}
	for _, c := range cases {
		if got := Get(c.part).LanguageOf(c.file); got != c.want {
			t.Errorf("%s: LanguageOf(%q) = %q, quer %q", c.part, c.file, got, c.want)
		}
	}
}

// TestNormalizedPart confere o id canonico das parts.
func TestNormalizedPart(t *testing.T) {
	cases := map[string]string{
		"part1":   "part1",
		"part2":   "part2",
		"part3":   "part3",
		" PART3 ": "part3",
		"lixo":    "part1",
		"":        "part1",
	}
	for in, want := range cases {
		if got := NormalizedPart(in); got != want {
			t.Errorf("NormalizedPart(%q) = %q, quer %q", in, got, want)
		}
	}
}

// TestKeepZoneEntry reproduz a regra do Pulse: no part1, "bg/loc" de outra
// zona nao aparece no indice de zona.
func TestKeepZoneEntry(t *testing.T) {
	p1 := Get("part1")
	if p1.KeepZoneEntry("zone/filelist_z000c.win32.bin", "bg/loc005/m000/bin/block000.win32.imgb") {
		t.Error("part1 deveria descartar bg/loc de outra zona")
	}
	if !p1.KeepZoneEntry("zone/filelist_z000c.win32.bin", "bg/loc000/m000/bin/block000.win32.imgb") {
		t.Error("part1 deveria manter bg/loc da propria zona")
	}
	if !p1.KeepZoneEntry("zone/filelist_z000c.win32.bin", "ac/ability.bin") {
		t.Error("part1 deveria manter entradas normais")
	}
	if !Get("part2").KeepZoneEntry("zone/filelist_z0000c.win32.bin", "bg/loc005/m000/x.imgb") {
		t.Error("part2 nao filtra bg/loc")
	}
}

// TestPartLayouts documenta as pastas que cada part usa: mudar isso e uma
// decisao por jogo, nao um if global.
func TestPartLayouts(t *testing.T) {
	const root = `C:\game`
	for _, id := range []string{"part1", "part2", "part3"} {
		p := Get(id)
		if p.DataDir(root) == "" || len(p.FilelistDirs()) == 0 {
			t.Fatalf("%s: layout incompleto", id)
		}
	}
	if got := Get("part1").DataDir(root); got != `C:\game\white_data` {
		t.Errorf("part1 DataDir=%q", got)
	}
	if got := Get("part2").DataDir(root); got != `C:\game\alba_data` {
		t.Errorf("part2 DataDir=%q", got)
	}
	if got := Get("part3").DataDir(root); got != `C:\game\weiss_data` {
		t.Errorf("part3 DataDir=%q", got)
	}
}
