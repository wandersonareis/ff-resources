package main

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/shared/dto"
)

// TestTextServiceSmoke valida o caminho <bin>/data/partN -> FF13Service -> DTO.
// A árvore fica RELATIVA AO EXECUTÁVEL (bin/data), como no projeto FFX antigo;
// no teste o executável é temporário, então apontamos as envs para o bin/data
// do app (cwd = apps/ff13).
func TestTextServiceSmoke(t *testing.T) {
	data := filepath.Join("bin", "data")
	t.Setenv("FF_RESOURCES_DATA", data)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(data, "mods"))

	var svc FF13Service
	for _, part := range []string{"part1", "part2", "part3"} {
		part := part
		t.Run(part, func(t *testing.T) {
			units, err := svc.ListTextUnits(part)
			if err != nil {
				t.Fatalf("ListTextUnits(%s): %v", part, err)
			}
			if len(units) == 0 {
				t.Fatalf("nenhuma unidade em bin/data/%s", part)
			}
			// Toda unidade listada tem um "_us" editável.
			for _, u := range units {
				if u.Primary == "" {
					t.Errorf("unidade sem primário: %s", u.ID)
				}
			}
			// Abre a primeira unidade que o codec consegue ler (parte dos
			// .ztr do jogo ainda esta no formato little-endian, fora do
			// alcance do codec).
			var opened int
			for _, unit := range units {
				entry, err := svc.GetTextEntry(part, unit.ID, "")
				if err != nil {
					continue
				}
				// Arquivo de texto VAZIO (o jogo tem vários: o idioma não
				// existe naquela entrada) não serve para o smoke; tenta o
				// próximo.
				if len(entry.Rows) == 0 {
					continue
				}
				opened++
				if entry.Rows[0].Name == "" {
					t.Fatalf("%s: primeira row sem chave", part)
				}
				break
			}
			if opened == 0 {
				t.Fatalf("nenhuma unidade de %s pôde ser aberta", part)
			}
		})
	}
}

// TestGetTextEntryWithReferences confere a unidade de pasta: o "_us" e o
// texto editavel (mods quando existe, data pristine), e os OUTROS idiomas
// da mesma unidade entram como referencia em Original[lang].
func TestGetTextEntryWithReferences(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1", "id2"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello", "World"}))
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_fr.ztr"),
		testZtrBlob(t, ids, []string{"Bonjour", "Monde"}))
	// Traducao em mods: e o que a tabela mostra.
	writeFile(t, filepath.Join(root, "mods", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hi", "World"}))

	var svc FF13Service
	entry, err := svc.GetTextEntry("part1", "txtres/resident/book/txtres", "")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if len(entry.Rows) != 2 {
		t.Fatalf("rows: %d, esperado 2", len(entry.Rows))
	}
	if got := entry.Rows[0].Text["us"]; got != "Hi" {
		t.Errorf("text[us]: %q, esperado %q (o de mods)", got, "Hi")
	}
	if got := entry.Rows[0].Original["us"]; got != "Hello" {
		t.Errorf("original[us]: %q, esperado %q (pristine)", got, "Hello")
	}
	if got := entry.Rows[0].Original["fr"]; got != "Bonjour" {
		t.Errorf("original[fr]: %q, esperado %q (referencia)", got, "Bonjour")
	}
	if got := entry.Rows[1].Original["fr"]; got != "Monde" {
		t.Errorf("original[fr] da 2a row: %q, esperado %q", got, "Monde")
	}
	if entry.Metadata.ID != "txtres/resident/book/txtres" {
		t.Errorf("metadata.id: %q", entry.Metadata.ID)
	}
	if entry.Metadata.Key != "part1/txtres/resident/book/txtres" {
		t.Errorf("metadata.key: %q", entry.Metadata.Key)
	}
	if entry.Metadata.Files["fr"] != "txtres/resident/book/txtres_fr.ztr" {
		t.Errorf("metadata.files[fr]: %q", entry.Metadata.Files["fr"])
	}
}

// TestGetTextEntryWithLittleEndianTranslation confere a leitura da tradução
// no formato LE do Pulse (dicionário não comprimido): o data/ continua BE e o
// mods/ pode ser LE — o ztr.Read detecta e decodifica.
func TestGetTextEntryWithLittleEndianTranslation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"$id1", "$id2"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello", "World"}))

	ks, err := textCodec("part1")
	if err != nil {
		t.Fatalf("textCodec: %v", err)
	}
	lines, err := ks.EncodeLines([]string{"Ola", "Mundo"})
	if err != nil {
		t.Fatalf("EncodeLines: %v", err)
	}
	le, err := ztr.WriteLE(&ztr.File{IDs: ids, Lines: lines})
	if err != nil {
		t.Fatalf("WriteLE: %v", err)
	}
	writeFile(t, filepath.Join(root, "mods", "part1", "txtres", "resident", "book", "txtres_us.ztr"), le)

	var svc FF13Service
	entry, err := svc.GetTextEntry("part1", "txtres/resident/book/txtres", "")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if got := entry.Rows[0].Text["us"]; got != "Ola" {
		t.Errorf("text[us]: %q, esperado %q (tradução LE de mods)", got, "Ola")
	}
	if got := entry.Rows[1].Text["us"]; got != "Mundo" {
		t.Errorf("text[us] da 2a row: %q, esperado %q", got, "Mundo")
	}
	if got := entry.Rows[0].Original["us"]; got != "Hello" {
		t.Errorf("original[us]: %q, esperado %q (pristine de data)", got, "Hello")
	}
}

// TestGetTextEntryOtherLanguage confere a visão de OUTRO idioma: o slot
// editável ("us") continua sendo o texto atual (a base do rascunho) e o
// idioma pedido entra em Text[lang] com o PRISTINE dele (referência
// somente-leitura).
func TestGetTextEntryOtherLanguage(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	t.Setenv("FF_RESOURCES_CONFIG", filepath.Join(root, "ff13.config.json"))

	ids := []string{"id1", "id2"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello", "World"}))
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_fr.ztr"),
		testZtrBlob(t, ids, []string{"Bonjour", "Monde"}))
	writeFile(t, filepath.Join(root, "mods", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hi", "World"}))

	var svc FF13Service
	entry, err := svc.GetTextEntry("part1", "txtres/resident/book/txtres", "fr")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if len(entry.Rows) != 2 {
		t.Fatalf("rows: %d, esperado 2", len(entry.Rows))
	}
	if got := entry.Rows[0].Text["us"]; got != "Hi" {
		t.Errorf("text[us]: %q, esperado %q (o slot editável continua o de mods)", got, "Hi")
	}
	if got := entry.Rows[0].Text["fr"]; got != "Bonjour" {
		t.Errorf("text[fr]: %q, esperado %q (o pristine do idioma pedido)", got, "Bonjour")
	}
	if got := entry.Rows[0].Original["fr"]; got != "Bonjour" {
		t.Errorf("original[fr]: %q, esperado %q", got, "Bonjour")
	}
	// O par de export é sempre o do idioma default.
	if entry.Rows[0].Source != "Hello" || entry.Rows[0].Target != "Hi" {
		t.Errorf("source/target: %q / %q, esperado %q / %q",
			entry.Rows[0].Source, entry.Rows[0].Target, "Hello", "Hi")
	}
}

// TestApplyTextCollectionZtr confere que a edicao da unidade de pasta grava
// o .ztr em mods/ (o original de data/ nao e tocado).
func TestApplyTextCollectionZtr(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1", "id2"}
	orig := filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr")
	writeFile(t, orig, testZtrBlob(t, ids, []string{"Hello", "World"}))

	var svc FF13Service
	c := dto.Collection{
		"txtres/resident/book/txtres": {
			Metadata: dto.Metadata{Key: "part1/txtres/resident/book/txtres", ID: "txtres/resident/book/txtres"},
			Rows: []dto.TextRow{
				{Index: 0, Name: "id1", Text: map[string]string{"us": "Ola"}, Target: "Ola"},
				{Index: 1, Name: "id2", Text: map[string]string{"us": "Mundo"}, Target: "Mundo"},
			},
		},
	}
	if err := svc.ApplyTextCollection("part1", c); err != nil {
		t.Fatalf("ApplyTextCollection: %v", err)
	}

	dst := filepath.Join(root, "mods", "part1", "txtres", "resident", "book", "txtres_us.ztr")
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("le %s: %v", dst, err)
	}
	zf, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("ztr.Read: %v", err)
	}
	ks, err := textCodec("part1")
	if err != nil {
		t.Fatalf("textCodec: %v", err)
	}
	texts, err := ks.DecodeLines(zf.Lines)
	if err != nil {
		t.Fatalf("DecodeLines: %v", err)
	}
	if texts[0] != "Ola" || texts[1] != "Mundo" {
		t.Errorf("gravado: %v, esperado [Ola Mundo]", texts)
	}

	// O original de data/ continua intacto.
	before, err := os.ReadFile(orig)
	if err != nil {
		t.Fatalf("le original: %v", err)
	}
	origZf, err := ztr.Read(before)
	if err != nil {
		t.Fatalf("ztr.Read original: %v", err)
	}
	origTexts, err := ks.DecodeLines(origZf.Lines)
	if err != nil {
		t.Fatalf("DecodeLines original: %v", err)
	}
	if origTexts[0] != "Hello" {
		t.Errorf("original alterado: %q", origTexts[0])
	}
}

// TestApplyTextCollectionWpd confere o reempacotamento do container WPD: o
// .bin vai para mods/ com o registro .ztr reescrito e os demais registros
// byte a byte.
func TestApplyTextCollectionWpd(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1", "id2"}
	wdb := []byte("WDB-BYTES-ORIGINAIS")
	container := filepath.Join(root, "data", "part1", "db", "ai", "npc", "pack", "auto_yus_us.bin")
	writeWpd(t, container, []wpd.Part{
		{Name: "auto_yus", Extension: "wdb", Data: wdb},
		{Name: "auto_yus_us", Extension: "ztr", Data: testZtrBlob(t, ids, []string{"Hello", "World"})},
	})

	var svc FF13Service
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

	dst := filepath.Join(root, "mods", "part1", "db", "ai", "npc", "pack", "auto_yus_us.bin")
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("le %s: %v", dst, err)
	}
	f, err := wpd.Read(raw)
	if err != nil {
		t.Fatalf("wpd.Read: %v", err)
	}
	if len(f.Header.Records) != 2 {
		t.Fatalf("registros: %d, esperado 2", len(f.Header.Records))
	}
	gotWdb, ok := f.ExtractRecord("auto_yus", "wdb")
	if !ok {
		t.Fatal("registro .wdb ausente")
	}
	if string(gotWdb) != string(wdb) {
		t.Errorf("registro .wdb alterado: %q, esperado %q", gotWdb, wdb)
	}
	gotZtr, ok := f.ExtractRecord("auto_yus_us", "ztr")
	if !ok {
		t.Fatal("registro .ztr ausente")
	}
	zf, err := ztr.Read(gotZtr)
	if err != nil {
		t.Fatalf("ztr.Read: %v", err)
	}
	ks, err := textCodec("part1")
	if err != nil {
		t.Fatalf("textCodec: %v", err)
	}
	texts, err := ks.DecodeLines(zf.Lines)
	if err != nil {
		t.Fatalf("DecodeLines: %v", err)
	}
	if texts[0] != "Ola" || texts[1] != "Mundo" {
		t.Errorf("texto gravado: %v, esperado [Ola Mundo]", texts)
	}
}
