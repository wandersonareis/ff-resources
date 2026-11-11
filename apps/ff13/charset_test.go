package main

import (
	"path/filepath"
	"slices"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
	ztenc "ffresources/internal/families/white/codecs/ztr/encoding"
)

// TestEncodingLanguagesMatchUnitLangs trava o catálogo de idiomas dos dois
// lados: o sufixo de arquivo (_us/_jp/...) e a paleta do codec precisam
// listar os MESMOS códigos na mesma ordem.
func TestEncodingLanguagesMatchUnitLangs(t *testing.T) {
	if got := ztenc.Languages(); !slices.Equal(got, unitLangs) {
		t.Errorf("encoding.Languages() = %v; unitLangs = %v", got, unitLangs)
	}
}

// ztrBlobFor monta um .ztr no codepage pedido — o charset do ARQUIVO, que
// é o que o jogo grava em cada sufixo de idioma.
func ztrBlobFor(t *testing.T, cp ztenc.Codepage, ids []string, texts []string) []byte {
	t.Helper()
	ks, err := textCodecFor("part1", cp)
	if err != nil {
		t.Fatalf("textCodecFor(part1, %d): %v", cp, err)
	}
	lines, err := ks.EncodeLines(texts)
	if err != nil {
		t.Fatalf("EncodeLines(%d): %v", cp, err)
	}
	blob, err := ztr.WriteTie(&ztr.File{IDs: ids, Lines: lines}, ztr.Compressed, ztr.TieMinLast)
	if err != nil {
		t.Fatalf("WriteTie: %v", err)
	}
	return blob
}

// TestCodepageOf fixa o mapa idioma -> codepage do arquivo. ERRAR aqui é o
// bug do mojibake: o .ztr _ch é Big5 e o _kr é EUC-KR; ler os dois como
// Shift-JIS transforma cada byte 0xA1-0xDF em kana de meia largura e os
// pares em ideographas erradas ("ｾｵ ｼ� ...").
func TestCodepageOf(t *testing.T) {
	cases := map[string]ztenc.Codepage{
		"us": ztenc.LatinJapanese,
		"jp": ztenc.LatinJapanese,
		"ch": ztenc.Chinese,
		"kr": ztenc.Korean,
		// Fora do catálogo ou sem charset próprio: segue o do slot editável.
		"":   ztenc.LatinJapanese,
		"br": ztenc.LatinJapanese,
		"fr": ztenc.LatinJapanese,
		"gr": ztenc.LatinJapanese,
		"de": ztenc.LatinJapanese, // alias legado
		"it": ztenc.LatinJapanese,
		"sp": ztenc.LatinJapanese,
		"ck": ztenc.LatinJapanese,
		"uk": ztenc.LatinJapanese,
	}
	for lang, want := range cases {
		if got := codepageOf(lang); got != want {
			t.Errorf("codepageOf(%q) = %d; queria %d", lang, got, want)
		}
	}
}

// TestGetTextEntryCodepagePerLanguage é a trava do mojibake nas referências
// CJK: cada idioma é lido no charset do SEU arquivo. As três linhas são as
// da unidade real zone/z002/txtres/ac/ac_hgcr_051 (a mesma dos exports em
// mods/edits), cuja leitura errada produzia "ｾｵ ｼ� ﾀﾖﾀｻ" / "ｦ�Gﾁﾙｯ爭ﾎ｡K｡K".
func TestGetTextEntryCodepagePerLanguage(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"$ahgcr051_010lt", "$ahgcr081_010lt"}
	dir := filepath.Join(root, "data", "part1", "zone", "z002", "txtres", "ac", "ac_hgcr_051")
	writeFile(t, filepath.Join(dir, "txtres_us.ztr"), ztrBlobFor(t, ztenc.LatinJapanese, ids,
		[]string{"Looks operational.", "Dead end, huh?"}))
	writeFile(t, filepath.Join(dir, "txtres_jp.ztr"), ztrBlobFor(t, ztenc.LatinJapanese, ids,
		[]string{"使えそうだな", "迷えるかもしれない"}))
	writeFile(t, filepath.Join(dir, "txtres_kr.ztr"), ztrBlobFor(t, ztenc.Korean, ids,
		[]string{"쓸 수 있을 듯 하군", "막다른 길인가"}))
	writeFile(t, filepath.Join(dir, "txtres_ch.ztr"), ztrBlobFor(t, ztenc.Chinese, ids,
		[]string{"似乎還能用……", "死路嗎……"}))
	// Referência fora do escopo CJK: continua no LatinJapanese (ASCII).
	writeFile(t, filepath.Join(dir, "txtres_fr.ztr"), ztrBlobFor(t, ztenc.LatinJapanese, ids,
		[]string{"On dirait que ca marche.", "Un cul-de-sac, non ?"}))

	var svc FF13Service
	entry, err := svc.GetTextEntry("part1", "zone/z002/txtres/ac/ac_hgcr_051/txtres", "")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if len(entry.Rows) != len(ids) {
		t.Fatalf("rows: %d, esperado %d", len(entry.Rows), len(ids))
	}

	// (linha, idioma) -> texto que o charset do arquivo produz.
	cases := []struct {
		row  int
		lang string
		want string
	}{
		{0, "us", "Looks operational."},
		{0, "jp", "使えそうだな"},
		{0, "kr", "쓸 수 있을 듯 하군"},
		{0, "ch", "似乎還能用……"},
		{0, "fr", "On dirait que ca marche."},
		{1, "kr", "막다른 길인가"},
		{1, "ch", "死路嗎……"},
	}
	for _, c := range cases {
		if got := entry.Rows[c.row].Original[c.lang]; got != c.want {
			t.Errorf("row %d Original[%s] = %q; queria %q (charset do arquivo)",
				c.row, c.lang, got, c.want)
		}
	}
	// O slot editável vem do pristine do _us.
	if got := entry.Rows[0].Text["us"]; got != "Looks operational." {
		t.Errorf("text[us] = %q; queria o pristine do _us", got)
	}
}
