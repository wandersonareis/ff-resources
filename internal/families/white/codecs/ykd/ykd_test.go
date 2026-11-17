package ykd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/wpd"
)

// TestParseRealTitle0 lê o container title0 do FF13 quando os dados de
// trabalho existem no repositório. Sem os dados (CI), vira skip.
func TestParseRealTitle0(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "apps", "ff13", "bin", "data",
		"part1", "gui", "resident", "title0.win32.xgr")
	if _, err := os.Stat(path); err != nil {
		t.Skip("dados do FF13 ausentes")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	container, err := wpd.Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, record := range container.Header.Records {
		if !strings.EqualFold(record.Extension, "ykd") {
			continue
		}
		names = append(names, record.Name+"."+record.Extension)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("container sem ykd")
	}

	for _, record := range container.Header.Records {
		if !strings.EqualFold(record.Extension, "ykd") {
			continue
		}
		data, ok := container.ExtractRecord(record.Name, record.Extension)
		if !ok {
			t.Fatalf("registro %s não extraído", record.Name)
		}
		file, err := Parse(data)
		if err != nil {
			t.Fatalf("%s: %v", record.Name, err)
		}
		if len(file.Blocks) == 0 || len(file.Resources) == 0 {
			t.Fatalf("%s: estrutura vazia (blocks=%d resources=%d)", record.Name, len(file.Blocks), len(file.Resources))
		}
		var refs, text int
		for _, block := range file.Blocks {
			refs += len(block.Tail.ResourceRefs)
			if block.Tail.Text != nil {
				text++
			}
		}
		t.Logf("%s: blocks=%d resources=%d refs=%d text=%d", record.Name,
			len(file.Blocks), len(file.Resources), refs, text)
	}
}
