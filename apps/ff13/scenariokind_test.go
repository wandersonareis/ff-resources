package main

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/shared/dto"
)

// TestApplyTextCollectionScenarioContainer confere que o IMPORT não precisa
// de caminho especial por kind: o único kind cujo texto mora num CONTAINER
// (scenario, db/ai/npc/pack) já é reconhecido pelo container do primário, e
// o save reempacota o WPD preservando o .wdb — igual aos demais kinds.
func TestApplyTextCollectionScenarioContainer(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	t.Setenv("FF_RESOURCES_CONFIG", filepath.Join(root, "ff13.config.json"))

	ids := []string{"id1"}
	// O pristine do scenario é um WPD com o .ztr DENTRO.
	writeWpd(t, filepath.Join(root, "data", "part3", "db", "ai", "npc", "pack", "auto_c01_us.bin"),
		[]wpd.Part{
			{Name: "auto_c01", Extension: "wdb", Data: []byte("WDB-BINARIO")},
			{Name: "auto_c01_us", Extension: "ztr", Data: testZtrBlob(t, ids, []string{"Cenario"})},
		})

	var svc FF13Service
	// A árvore enxerga a unidade com o kind `scenario`.
	entries, err := svc.ListDataEntries("part3")
	if err != nil {
		t.Fatalf("ListDataEntries: %v", err)
	}
	var found *DataEntry
	for i := range entries {
		if entries[i].ID == "db/ai/npc/pack/auto_c01" {
			found = &entries[i]
		}
	}
	if found == nil {
		t.Fatalf("unidade scenario ausente da coleção: %+v", entries)
	}
	if found.Kind != "scenario" {
		t.Errorf("kind = %q, quer scenario", found.Kind)
	}
	if found.Container != "wpd" {
		t.Errorf("container = %q, quer wpd", found.Container)
	}

	// O import aplica a tradução pela MESMA rota dos .ztr soltos: a chave é o
	// id da unidade, e o save reempacota o container.
	if err := svc.ApplyTextCollection("part3", dto.Collection{
		"db/ai/npc/pack/auto_c01": {
			Metadata: dto.Metadata{Key: "part3/db/ai/npc/pack/auto_c01", ID: "db/ai/npc/pack/auto_c01"},
			Rows: []dto.TextRow{
				{Index: 0, Name: "id1", Text: map[string]string{"us": "Traduzido"}, Target: "Traduzido"},
			},
		},
	}); err != nil {
		t.Fatalf("ApplyTextCollection: %v", err)
	}

	dst := filepath.Join(root, "mods", "part3", "db", "ai", "npc", "pack", "auto_c01_us.bin")
	raw, rerr := os.ReadFile(dst)
	if rerr != nil {
		t.Fatalf("leitura do traduzido: %v", rerr)
	}
	// O container continua um WPD (os .wdb preservados + o .ztr reescrito).
	if len(raw) < 4 || string(raw[:4]) != "WPD\x00" {
		t.Fatalf("o traduzido deveria continuar um container WPD: %x", raw[:4])
	}
	f, werr := wpd.Read(raw)
	if werr != nil {
		t.Fatalf("lendo o traduzido: %v", werr)
	}
	ztrData, ok := f.ExtractRecord("auto_c01_us", "ztr")
	if !ok {
		t.Fatal("registro .ztr ausente no container reempacotado")
	}
	if !contains(ztrData, []byte("Traduzido")) {
		t.Errorf("o .ztr dentro do container deveria trazer a tradução")
	}
	if wdb, ok := f.ExtractRecord("auto_c01", "wdb"); !ok || string(wdb) != "WDB-BINARIO" {
		t.Errorf("o .wdb deveria ser preservado byte a byte, veio %q", string(wdb))
	}

	// E a leitura volta a tradução do slot "us".
	entry, err := svc.GetTextEntry("part3", "db/ai/npc/pack/auto_c01", "")
	if err != nil {
		t.Fatalf("GetTextEntry: %v", err)
	}
	if got := entry.Rows[0].Text["us"]; got != "Traduzido" {
		t.Errorf("text[us] = %q, quer Traduzido", got)
	}
}

// contains é um auxiliar local (o pacote de teste não importa bytes aqui).
func contains(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
