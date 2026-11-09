package main

import (
	"testing"

	"ffresources/apps/ff13/common"
	"ffresources/apps/ff13/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
)

// syntheticCollection monta uma Collection de mentira a partir dos fileInfo
// ("pos:uncmp:cmp:caminho").
func syntheticCollection(infos ...string) *fl.Collection {
	coll := &fl.Collection{GameCode: fl.GameFF131}
	chunk := fl.Chunk{ID: 0}
	for i, info := range infos {
		chunk.Entries = append(chunk.Entries, fl.Entry{FileCode: uint32(i), FileInfo: info})
	}
	coll.Chunks = []fl.Chunk{chunk}
	return coll
}

// syntheticIndex monta um índice de mentira (o Virtual é o caminho do índice
// embutido dentro do container do pai).
func syntheticIndex(path, virtual string, infos ...string) *filelistIndex {
	return &filelistIndex{
		Path:    path,
		Virtual: virtual,
		Level:   1,
		Coll:    syntheticCollection(infos...),
	}
}

// TestIndexEntriesAreRaw confere que as entradas saem CRUAS: caminho indexado
// com o casing original e tamanho descomprimido (os tamanhos do fileInfo são
// HEXADECIMAIS).
func TestIndexEntriesAreRaw(t *testing.T) {
	coll := syntheticCollection(
		"10:1000:500:sys/Main.BIN",
		"20:2000:2000:sys/Sub/Deep.bin",
		"30:300:100:zone/filelist_z000u.win32.bin",
		"40:0:0:txtres/read.bin",
	)
	idx := &filelistIndex{Path: "root.bin", Level: 0, Coll: coll}

	got := indexEntries(parts.Get("part1"), idx, nil)
	if len(got) != 4 {
		t.Fatalf("entradas: %d, quer 4 (%+v)", len(got), got)
	}
	want := []indexEntry{
		{Path: "sys/Main.BIN", Size: 0x1000},
		{Path: "sys/Sub/Deep.bin", Size: 0x2000},
		{Path: "zone/filelist_z000u.win32.bin", Size: 0x300},
		{Path: "txtres/read.bin", Size: 0},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entrada %d: got %+v, quer %+v", i, got[i], want[i])
		}
	}
}

// TestIndexEntriesKeepZoneFilter confere que a regra da part (o filtro
// "bg/loc de outra zona" do part1) continua valendo sobre o dado CRU.
func TestIndexEntriesKeepZoneFilter(t *testing.T) {
	coll := syntheticCollection(
		"1:10:10:bg/loc000/m000/bin/block000.win32.imgb",
		"2:20:20:bg/loc005/m000/bin/block000.win32.imgb",
		"3:30:30:ac/ability.bin",
	)
	idx := &filelistIndex{
		Path:    "filelistu.win32.bin#zone/filelist_z000u.win32.bin",
		Virtual: "zone/filelist_z000u.win32.bin",
		Level:   1,
		Coll:    coll,
	}

	got := indexEntries(parts.Get("part1"), idx, nil)
	if len(got) != 2 {
		t.Fatalf("entradas: %d, quer 2 (a de outra zona deve sumir): %+v", len(got), got)
	}
	for _, e := range got {
		if e.Path == "bg/loc005/m000/bin/block000.win32.imgb" {
			t.Fatalf("entrada de outra zona entrou: %+v", e)
		}
	}

	// No part2 o filtro não existe: tudo entra.
	got2 := indexEntries(parts.Get("part2"), idx, nil)
	if len(got2) != 3 {
		t.Fatalf("part2: entradas %d, quer 3: %+v", len(got2), got2)
	}
}

// TestIndexEntriesDeduplicates confere que o mesmo caminho indexado
// (repetição do dump) vira UMA entrada na árvore.
func TestIndexEntriesDeduplicates(t *testing.T) {
	coll := syntheticCollection(
		"1:100:100:sys/dup.bin",
		"2:200:200:sys/dup.bin",
		"3:300:300:sys/other.bin",
	)
	idx := &filelistIndex{Path: "root.bin", Level: 0, Coll: coll}
	got := indexEntries(parts.Get("part1"), idx, nil)
	if len(got) != 2 {
		t.Fatalf("entradas: %d, quer 2: %+v", len(got), got)
	}
	if got[0].Path != "sys/dup.bin" || got[0].Size != 0x100 {
		t.Errorf("dup: got %+v, quer path=sys/dup.bin size=%d", got[0], 0x100)
	}
}

// TestMergeChain confere a fusão dos diretórios com FILHO ÚNICO (a regra do
// Pulse): "empty/empty/item" vira o nó "empty/empty" (que pede os filhos no
// diretório final da cadeia) contendo a folha "item".
func TestMergeChain(t *testing.T) {
	entries := []indexEntry{
		{Path: "empty/empty/item.bin"},
		{Path: "other/child.bin"},
		{Path: "many/one.bin"},
		{Path: "many/two.bin"},
	}
	tree := buildIndexTree(entries)
	nodes := listDirNodes(parts.Get("part1"), tree, "", "")

	if len(nodes) != 3 {
		t.Fatalf("nós: %d, quer 3 (%+v)", len(nodes), nodes)
	}
	merged := nodes[0]
	if merged.Name != "empty" || merged.Path != "empty/empty" {
		t.Errorf("nó fundido: got %+v, quer name=empty path=empty/empty", merged)
	}
	if len(merged.MergeChain) != 2 || merged.MergeChain[0] != "empty" || merged.MergeChain[1] != "empty" {
		t.Errorf("mergeChain: got %v, quer [empty empty]", merged.MergeChain)
	}
	if !merged.IsDir || merged.Kind != common.KindDir {
		t.Errorf("nó fundido: got kind=%q isDir=%v, quer dir", merged.Kind, merged.IsDir)
	}
	// Os filhos do nó fundido são os do diretório final da cadeia.
	children := listDirNodes(parts.Get("part1"), tree, merged.Path, "")
	if len(children) != 1 || children[0].Name != "item.bin" || children[0].IsDir {
		t.Errorf("filhos do nó fundido: %+v", children)
	}

	// "many" tem dois filhos: nada funde.
	if nodes[1].Name != "many" || len(nodes[1].MergeChain) != 0 {
		t.Errorf("nó com dois filhos: %+v", nodes[1])
	}
	// "other" tem um único filho ARQUIVO: não funde (a fusão é só com
	// diretório), mas o caminho pedido é o do diretório.
	if nodes[2].Name != "other" || nodes[2].Path != "other" || len(nodes[2].MergeChain) != 0 {
		t.Errorf("nó com arquivo único: %+v", nodes[2])
	}
}

// TestListDirNodesOrder confere a ordem da listagem: diretórios primeiro,
// arquivos depois, cada grupo em ordem alfabética (sem diferenciar maiúsculas
// de minúsculas).
func TestListDirNodesOrder(t *testing.T) {
	entries := []indexEntry{
		{Path: "Zoo/File.bin"},
		{Path: "Alpha/One.bin"},
		{Path: "beta.bin"},
		{Path: "Alpha.bin"},
	}
	nodes := listDirNodes(parts.Get("part1"), buildIndexTree(entries), "", "")
	got := make([]string, 0, len(nodes))
	for _, n := range nodes {
		got = append(got, n.Name)
	}
	want := []string{"Alpha", "Zoo", "Alpha.bin", "beta.bin"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("ordem: got %v, want %v", got, want)
		}
	}
}

// TestZoneUnionNodes confere a união da zona: o conteúdo do índice de ZONA
// (u+c) mais as entradas "zone/<z>/..." dos demais índices da part, com o
// prefixo removido e o dedupe pelo caminho em minúsculas (o índice de zona
// vence).
func TestZoneUnionNodes(t *testing.T) {
	zone := syntheticIndex(
		"filelistu.win32.bin#zone/filelist_z000u.win32.bin",
		"zone/filelist_z000u.win32.bin",
		"1:10:10:ac/schedule/ac_comn_000/data.bin",
		"2:20:20:zone/z000/z000.wdb",
		"3:30:30:bg/loc005/m000/bin/block000.win32.imgb",
	)
	zoneC := syntheticIndex(
		"filelistc.win32.bin#zone/filelist_z000c.win32.bin",
		"zone/filelist_z000c.win32.bin",
		"4:40:40:ac/schedule/ac_comn_001/data.bin",
	)
	scripts := syntheticIndex(
		"filelist_scru.win32.bin",
		"",
		"5:50:50:zone/z000/scr000.clb",
		"6:60:60:zone/z002/scr000.clb",
	)
	disc := partDiscovery{indexes: []*filelistIndex{zone, zoneC, scripts}}

	nodes := zoneNodes(parts.Get("part1"), disc, "z000", "")
	got := make([]string, 0, len(nodes))
	for _, n := range nodes {
		got = append(got, n.Name)
	}
	// O diretório "ac" funde com o filho único "schedule"; a entrada de outra
	// zona (bg/loc005) é descartada pela regra do part1.
	want := []string{"ac", "scr000.clb", "z000.wdb"}
	if len(got) != len(want) {
		t.Fatalf("nós da zona: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("nós da zona: got %v, want %v", got, want)
		}
	}
	if len(nodes[0].MergeChain) != 2 || nodes[0].MergeChain[1] != "schedule" {
		t.Errorf("cadeia fundida: %v, quer [ac schedule]", nodes[0].MergeChain)
	}
	if nodes[0].Path != "ac/schedule" {
		t.Errorf("caminho do nó fundido: %q", nodes[0].Path)
	}
	if nodes[1].Size != 0x50 {
		t.Errorf("tamanho do arquivo da zona: %d, quer %d", nodes[1].Size, 0x50)
	}
}

// TestZoneUnionDedupe confere que o mesmo caminho (em minúsculas) declarado
// pelo índice de zona e por outro índice vira UM nó só — o do índice de zona
// (que tem precedência).
func TestZoneUnionDedupe(t *testing.T) {
	zone := syntheticIndex(
		"filelistu.win32.bin#zone/filelist_z000u.win32.bin",
		"zone/filelist_z000u.win32.bin",
		"1:10:10:zone/z000/scr000.clb",
	)
	scripts := syntheticIndex(
		"filelist_scru.win32.bin",
		"",
		"2:99:99:zone/z000/scr000.clb",
	)
	disc := partDiscovery{indexes: []*filelistIndex{zone, scripts}}

	nodes := zoneNodes(parts.Get("part1"), disc, "z000", "")
	if len(nodes) != 1 {
		t.Fatalf("nós: %d, quer 1 (%+v)", len(nodes), nodes)
	}
	if nodes[0].Name != "scr000.clb" || nodes[0].Size != 0x10 {
		t.Errorf("nó: %+v, quer scr000.clb com tamanho %d", nodes[0], 0x10)
	}
}

// TestOneFilelistOnePlatform confere a regra D6: a plataforma é do índice
// INTEIRO (tirada do nome dele) e nenhuma entrada muda a plataforma. O
// backend só avisa quando uma entrada vem de outra plataforma.
func TestOneFilelistOnePlatform(t *testing.T) {
	name := "filelistu.win32.bin"
	if got := common.SourceOf(name); got != common.SourceWin32 {
		t.Fatalf("SourceOf(%q) = %q, quer %q", name, got, common.SourceWin32)
	}

	entries := []indexEntry{
		{Path: "mot/pc/sk_c001_lt/s1.white.win32.bin"},
		{Path: "mot/pc/sk_c001_lt/t1.white.ps3.bin"},
	}
	idx := &filelistIndex{Path: "filelistu.win32.bin", Level: 0}

	// A listagem não carrega plataforma por entrada: a do índice é a única.
	nodes := listDirNodes(parts.Get("part1"), buildIndexTree(entries), "mot/pc/sk_c001_lt", "")
	if len(nodes) != 2 {
		t.Fatalf("nós: %d, quer 2 (%+v)", len(nodes), nodes)
	}
	for _, n := range nodes {
		if n.Language != "" {
			t.Errorf("nó %q com idioma %q: a entrada não muda o índice", n.Name, n.Language)
		}
	}

	// A inconsistência (entrada de outra plataforma) é avisada, mas a
	// plataforma do índice é mantida.
	checkSource(idx, common.SourceOf(name), entries)
}
