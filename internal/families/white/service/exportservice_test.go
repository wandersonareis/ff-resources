package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ffxcommon "ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/families/white/codecs/wpd"
	fmtstrings "ffresources/internal/shared/artifact/strings"
	"ffresources/internal/shared/hash"
)

// TestExportRowsSourceFromDataTargetFromMods confere o contrato do export
// (o mesmo do FFX): Text/Source vêm SEMPRE de data/ — todos os idiomas da
// unidade vão como idiomas de /data, o default ("us") incluso — e o Target
// vem de mods/. Duas unidades com o MESMO original e traduções DIFERENTES
// mantêm targets distintos no artefato: só o texto repetido no slot
// default colapsa em ref "$hash" (a tradução de cada uma continua na sua
// linha do idioma alvo).
func TestExportRowsSourceFromDataTargetFromMods(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	// Duas unidades com o mesmo pristine e traduções diferentes.
	writeFile(t, filepath.Join(root, "data", "part1", "a", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Shared original"}))
	writeFile(t, filepath.Join(root, "data", "part1", "b", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Shared original"}))
	writeFile(t, filepath.Join(root, "mods", "part1", "a", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Traducao A"}))
	writeFile(t, filepath.Join(root, "mods", "part1", "b", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Traducao B"}))

	coll, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor: %v", err)
	}
	unitA, err := collectionUnit(coll, "part1", "a/txtres")
	if err != nil {
		t.Fatalf("collectionUnit(a): %v", err)
	}
	unitB, err := collectionUnit(coll, "part1", "b/txtres")
	if err != nil {
		t.Fatalf("collectionUnit(b): %v", err)
	}
	entryA, err := exportEntryRows("part1", unitA)
	if err != nil {
		t.Fatalf("exportEntryRows(a): %v", err)
	}
	entryB, err := exportEntryRows("part1", unitB)
	if err != nil {
		t.Fatalf("exportEntryRows(b): %v", err)
	}

	rowA := entryA.Rows[0]
	rowB := entryB.Rows[0]
	// Text["us"] é o pristine de data/ nos dois — a tradução NÃO entra no slot.
	if rowA.Text["us"] != "Shared original" || rowB.Text["us"] != "Shared original" {
		t.Fatalf("text[us]: %q / %q, quer o pristine de data/ nos dois", rowA.Text["us"], rowB.Text["us"])
	}
	// Source é a CÓPIA do idioma default.
	if rowA.Source != "Shared original" || rowB.Source != "Shared original" {
		t.Fatalf("source: %q / %q, quer a cópia do idioma default nos dois", rowA.Source, rowB.Source)
	}
	// Target é a tradução de mods/ de cada unidade — e elas divergem.
	if rowA.Target != "Traducao A" || rowB.Target != "Traducao B" {
		t.Fatalf("target: %q / %q, quer a tradução de mods/ de cada uma", rowA.Target, rowB.Target)
	}
	// Hash do texto exportado (o pristine de data/, como os demais idiomas).
	wantHash := hash.Sum64Hex("Shared original")
	if rowA.Hash["us"] != wantHash || rowB.Hash["us"] != wantHash {
		t.Errorf("hash[us]: %q / %q, quer %q (hash do texto exportado)", rowA.Hash["us"], rowB.Hash["us"], wantHash)
	}
}

// TestExportEntryRowsKeepsReferences confere que TODOS os idiomas da
// unidade entram em Text como idiomas de /data (o default "us" incluso, com
// o pristine — não a tradução), que Source é a cópia do default, que Target
// vem de mods/ e que Original NÃO é preenchido (é exclusivo de exibição).
func TestExportEntryRowsKeepsReferences(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "txtres_fr.ztr"),
		testZtrBlob(t, ids, []string{"Bonjour"}))
	writeFile(t, filepath.Join(root, "mods", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Ola"}))

	coll, err := collectionFor("part1")
	if err != nil {
		t.Fatalf("collectionFor: %v", err)
	}
	unit, err := collectionUnit(coll, "part1", "txtres/txtres")
	if err != nil {
		t.Fatalf("collectionUnit: %v", err)
	}
	entry, err := exportEntryRows("part1", unit)
	if err != nil {
		t.Fatalf("exportEntryRows: %v", err)
	}
	row := entry.Rows[0]
	if row.Text["us"] != "Hello" {
		t.Errorf("text[us]: %q, quer %q (o default vai como idioma de /data, não como tradução)", row.Text["us"], "Hello")
	}
	if row.Text["fr"] != "Bonjour" {
		t.Errorf("text[fr]: %q, quer %q (a referência entra no artefato)", row.Text["fr"], "Bonjour")
	}
	if row.Source != "Hello" {
		t.Errorf("source: %q, quer %q (cópia do idioma default)", row.Source, "Hello")
	}
	if row.Target != "Ola" {
		t.Errorf("target: %q, quer %q (tradução de mods/)", row.Target, "Ola")
	}
	if len(row.Original) != 0 {
		t.Errorf("original: %v, quer vazio (exclusivo de exibição)", row.Original)
	}
	if entry.Metadata.Key != "part1/txtres/txtres" || entry.Metadata.ID != "txtres/txtres" {
		t.Errorf("metadata: key=%q id=%q", entry.Metadata.Key, entry.Metadata.ID)
	}
}

// TestExportArtifactPath confere o NOME do artefato: o escopo entra no nome
// (um subconjunto nunca sobrescreve o artefato completo do tradutor) e a
// "/" do id vira "_".
func TestExportArtifactPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	cases := []struct {
		kind string
		ids  []string
		want string
	}{
		// O kind legado "ztr" (todos os kinds) mantém o nome antigo.
		{"ztr", nil, "text_all_localizations_part1.json"},
		{"", nil, "text_all_localizations_part1.json"},
		// Cada kind tem o SEU artefato: nunca colidem no mesmo arquivo.
		{"resident", nil, "resident_all_localizations_part1.json"},
		{"event", nil, "event_all_localizations_part1.json"},
		{"scenario", nil, "scenario_all_localizations_part1.json"},
		// Uma entrada: o kind prefixa e o "/" do id vira "_".
		{"resident", []string{"txtres/resident/book/txtres"}, "resident_txtres_resident_book_txtres_all_localizations_part1.json"},
		{"ztr", []string{"txtres/resident/book/txtres"}, "text_txtres_resident_book_txtres_all_localizations_part1.json"},
		// Subconjunto: escopo entra no nome.
		{"resident", []string{"a/one", "b/two"}, "resident_sel_2_a_one_all_localizations_part1.json"},
	}
	for _, c := range cases {
		got, err := exportArtifactPath("part1", c.kind, c.ids)
		if err != nil {
			t.Fatalf("exportArtifactPath: %v", err)
		}
		if filepath.Base(got) != c.want {
			t.Errorf("exportArtifactPath(%q, %v) = %q, quer %q", c.kind, c.ids, filepath.Base(got), c.want)
		}
		wantDir := filepath.Join(root, "mods", "edits")
		if filepath.Dir(got) != wantDir {
			t.Errorf("pasta do artefato: %q, quer %q", filepath.Dir(got), wantDir)
		}
	}
}

// TestExportArtifactBatchWritesFile confere o lote: o arquivo é gravado em
// mods/edits com o nome do escopo e o conteúdo traz o pristine de data/
// (text/source) e a tradução de mods/ (target) — o export lê mods/ direto,
// independente da flag enableMods.
func TestExportArtifactBatchWritesFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))
	writeFile(t, filepath.Join(root, "mods", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Ola"}))

	var svc Service
	paths, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ztr"}}, "part1", nil, "json")
	if err != nil {
		t.Fatalf("ExportArtifactBatch: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths: %v, quer 1 arquivo", paths)
	}
	want := filepath.Join(root, "mods", "edits", "text_all_localizations_part1.json")
	if paths[0] != want {
		t.Fatalf("caminho: %q, quer %q", paths[0], want)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatalf("le %s: %v", paths[0], err)
	}
	if !strings.Contains(string(raw), "Ola") {
		t.Errorf("o artefato não traz a tradução (target): %s", raw)
	}
	if !strings.Contains(string(raw), "Hello") {
		t.Errorf("o artefato não traz o pristine (text/source): %s", raw)
	}
}

// TestExportArtifactBatchKindScopesUnidades confere que o LOTE exporta só o
// kind pedido e grava o artefato com o nome da família: um subconjunto nunca
// sobrescreve o artefato de outra família.
func TestExportArtifactBatchKindScopesUnidades(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	// Uma unidade `resident` e uma `event` na mesma árvore.
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "resident", "book", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Residente"}))
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "event", "ev_a", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Evento"}))

	var svc Service

	// Só o kind `event`: o artefato tem o nome do kind e só a unidade event.
	paths, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "event"}}, "part1", nil, "json")
	if err != nil {
		t.Fatalf("ExportArtifactBatch(event): %v", err)
	}
	want := filepath.Join(root, "mods", "edits", "event_all_localizations_part1.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("caminho: %v, quer %q", paths, want)
	}
	raw, rerr := os.ReadFile(want)
	if rerr != nil {
		t.Fatalf("leitura: %v", rerr)
	}
	if !strings.Contains(string(raw), "Evento") {
		t.Errorf("o artefato de event deveria trazer o texto de event: %s", raw)
	}
	if strings.Contains(string(raw), "Residente") {
		t.Errorf("o artefato de event NÃO deveria trazer o texto de resident: %s", raw)
	}

	// Só o kind `resident`: artefato irmão, sem colidir.
	paths, err = svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "resident"}}, "part1", nil, "json")
	if err != nil {
		t.Fatalf("ExportArtifactBatch(resident): %v", err)
	}
	want = filepath.Join(root, "mods", "edits", "resident_all_localizations_part1.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("caminho: %v, quer %q", paths, want)
	}
	raw, rerr = os.ReadFile(want)
	if rerr != nil {
		t.Fatalf("leitura: %v", rerr)
	}
	if !strings.Contains(string(raw), "Residente") || strings.Contains(string(raw), "Evento") {
		t.Errorf("o artefato de resident deveria trazer só o resident: %s", raw)
	}

	// O kind legado "ztr" continua exportando TUDO (compatibilidade).
	paths, err = svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ztr"}}, "part1", nil, "json")
	if err != nil {
		t.Fatalf("ExportArtifactBatch(ztr): %v", err)
	}
	want = filepath.Join(root, "mods", "edits", "text_all_localizations_part1.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("caminho: %v, quer %q", paths, want)
	}
	raw, rerr = os.ReadFile(want)
	if rerr != nil {
		t.Fatalf("leitura: %v", rerr)
	}
	if !strings.Contains(string(raw), "Evento") || !strings.Contains(string(raw), "Residente") {
		t.Errorf("o artefato completo deveria trazer os dois: %s", raw)
	}
}

// TestExportArtifactBatchRejectsKindOfOtherPart confere que um kind que a
// part NÃO serve é recusado: o part3 não tem `ac`, o part1 não tem
// `scenario`.
func TestExportArtifactBatchRejectsKindOfOtherPart(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	// Uma unidade por kind do part3, para o lote ter o que exportar.
	blob := testZtrBlob(t, []string{"id1"}, []string{"texto"})
	for _, rel := range []string{
		"txtres/resident/book/txtres_us.ztr",
		"txtres/event/ev_a/txtres_us.ztr",
		"txtres/zone/z0001/txtres_us.ztr",
	} {
		writeFile(t, filepath.Join(root, "data", "part3", filepath.FromSlash(rel)), blob)
	}
	// O `scenario` é o ÚNICO kind cujo texto mora num container .bin (WPD):
	// o .ztr fica DENTRO do container.
	writeWpd(t, filepath.Join(root, "data", "part3", "db", "ai", "npc", "pack", "auto_c01_us.bin"),
		[]wpd.Part{
			{Name: "auto_c01", Extension: "wdb", Data: []byte("WDB")},
			{Name: "auto_c01_us", Extension: "ztr", Data: blob},
		})

	var svc Service
	if _, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ac"}}, "part3", nil, "json"); err == nil {
		t.Error("part3 não tem `ac`: deveria falhar")
	}
	if _, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "scenario"}}, "part1", nil, "json"); err == nil {
		t.Error("part1 não tem `scenario`: deveria falhar")
	}
	// Os kinds que a part serve passam.
	for _, kind := range []string{"resident", "zone", "event", "scenario"} {
		if _, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: kind}}, "part3", nil, "json"); err != nil {
			t.Errorf("part3 deveria servir o kind %q: %v", kind, err)
		}
	}
}

// TestExportArtifactBatchMultiKindUmStepPorKind confere que um lote com
// VÁRIOS kinds abre um step por família (cada um com o seu rótulo) e grava
// um artefato por kind.
func TestExportArtifactBatchMultiKindUmStepPorKind(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	blob := testZtrBlob(t, []string{"id1"}, []string{"texto"})
	writeFile(t, filepath.Join(root, "data", "part3", "txtres", "resident", "book", "txtres_us.ztr"), blob)
	writeFile(t, filepath.Join(root, "data", "part3", "txtres", "event", "ev_a", "txtres_us.ztr"), blob)
	writeWpd(t, filepath.Join(root, "data", "part3", "db", "ai", "npc", "pack", "auto_c01_us.bin"),
		[]wpd.Part{
			{Name: "auto_c01", Extension: "wdb", Data: []byte("WDB")},
			{Name: "auto_c01_us", Extension: "ztr", Data: blob},
		})

	var svc Service
	paths, err := svc.ExportArtifactBatch([]ExportPlanItem{
		{Kind: "resident"},
		{Kind: "event"},
		{Kind: "scenario"},
	}, "part3", nil, "json")
	if err != nil {
		t.Fatalf("ExportArtifactBatch: %v", err)
	}
	// Um arquivo por kind — nenhum colide.
	want := []string{
		filepath.Join(root, "mods", "edits", "event_all_localizations_part3.json"),
		filepath.Join(root, "mods", "edits", "resident_all_localizations_part3.json"),
		filepath.Join(root, "mods", "edits", "scenario_all_localizations_part3.json"),
	}
	if len(paths) != len(want) {
		t.Fatalf("arquivos: %v, quer %d", paths, len(want))
	}
	for _, w := range want {
		if _, serr := os.Stat(w); serr != nil {
			t.Errorf("artefato ausente %q: %v", filepath.Base(w), serr)
		}
	}
}

// TestExportArtifactBatchRejectsUnknownFormat confere a validação ANTES do
// batch (o erro vira toast no frontend).
func TestExportArtifactBatchRejectsUnknownFormat(t *testing.T) {
	var svc Service
	if _, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ztr"}}, "part1", nil, "xml"); err == nil {
		t.Fatal("formato desconhecido deveria falhar")
	}
	if _, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "events"}}, "part1", nil, "json"); err == nil {
		t.Fatal("kind desconhecido deveria falhar")
	}
	if _, err := svc.ExportArtifactBatch(nil, "part1", nil, "json"); err == nil {
		t.Fatal("lote vazio deveria falhar")
	}
}

// TestExportEntryWritesPair confere o export de UMA entrada: o par json +
// strings, no nome da entrada.
func TestExportEntryWritesPair(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))

	var svc Service
	paths, err := svc.ExportEntry("part1", "txtres/txtres", nil)
	if err != nil {
		t.Fatalf("ExportEntry: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths: %v, quer o par json + strings", paths)
	}
	wantJSON := filepath.Join(root, "mods", "edits", "text_txtres_txtres_all_localizations_part1.json")
	wantStrings := filepath.Join(root, "mods", "edits", "text_txtres_txtres_all_localizations_part1.strings")
	if paths[0] != wantJSON || paths[1] != wantStrings {
		t.Fatalf("caminhos: %v, quer %v e %v", paths, wantJSON, wantStrings)
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artefato ausente %s: %v", p, err)
		}
	}
}

// TestExportArtifactBatchStringsV2 confere o artefato .strings do FF13 no
// formato v2: o par CAT são LINHAS próprias (idioma-fonte com o source no
// valor, idioma-alvo com a tradução), o mapa vive no header e nenhum campo
// source/target aparece na linha. A releitura pelo parser do formato
// devolve Source (data/) e Target (mods/) — o ciclo do tradutor.
func TestExportArtifactBatchStringsV2(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FF_RESOURCES_DATA", filepath.Join(root, "data"))
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))

	ids := []string{"id1"}
	writeFile(t, filepath.Join(root, "data", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))
	writeFile(t, filepath.Join(root, "mods", "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Ola"}))

	var svc Service
	paths, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ztr"}}, "part1", nil, "strings")
	if err != nil {
		t.Fatalf("ExportArtifactBatch(strings): %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths: %v, quer 1 arquivo", paths)
	}
	want := filepath.Join(root, "mods", "edits", "text_all_localizations_part1.strings")
	if paths[0] != want {
		t.Fatalf("caminho: %q, quer %q", paths[0], want)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatalf("lendo artefato: %v", err)
	}
	out := string(raw)
	// O par CAT no header: versão + mapa fonte/alvo.
	if !strings.Contains(out, "row_count=1 version=2 source=us target=br*/") {
		t.Errorf("header sem o mapa do par CAT: %s", out)
	}
	// Linha do idioma-fonte (source no próprio valor) + linha do idioma-alvo.
	// O name da row é a chave do ZTR ("id1"), como nos demais formatos.
	usHash := hash.Sum64Hex("Hello")
	brHash := hash.Sum64Hex("Ola")
	if !strings.Contains(out, "txtres/txtres:id1:0:us║$"+usHash+" = Hello") {
		t.Errorf("linha do idioma-fonte ausente: %s", out)
	}
	if !strings.Contains(out, "txtres/txtres:id1:0:br║$"+brHash+" = Ola") {
		t.Errorf("linha do idioma-alvo ausente: %s", out)
	}
	// Nenhum campo source/target na linha: cada linha tem UM ║ (o da chave).
	if n := countUnescapedSep(out); n != 2 {
		t.Errorf("║ não escapados: %d, quer 2 (um por linha):\n%s", n, out)
	}
	// O ciclo: relendo o artefato, source (data/) e target (mods/) voltam.
	c, err := fmtstrings.Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	entry, ok := c["txtres/txtres"]
	if !ok || len(entry.Rows) != 1 {
		t.Fatalf("entrada txtres/txtres ausente: %+v", c)
	}
	row := entry.Rows[0]
	if row.Text["us"] != "Hello" || row.Source != "Hello" {
		t.Errorf("fonte: %+v, quer o pristine de data/", row)
	}
	if row.Target != "Ola" {
		t.Errorf("target: %q, quer %q (tradução de mods/)", row.Target, "Ola")
	}
}

// countUnescapedSep conta os ║ NÃO escapados: um por linha (o separador da
// chave). Um campo `║source = `/`║target = ` de escape apareceria como um
// segundo ║ na mesma linha.
func countUnescapedSep(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if !strings.HasPrefix(s[i:], "║") {
			continue
		}
		backslashes := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			n++
		}
		i += len("║") - 1
	}
	return n
}

// TestExportWritesIntoDataModsEdits reproduz o layout real do app: o
// executável fica na raiz e os dados em <raiz>/data, com as mods em
// <raiz>/data/mods. Nesse layout, a trava CheckWritablePath (GameFilesRoot
// = <raiz>, o padrão herdado do FFX) recusa "data\mods\edits\…" e o export
// falha NO FINAL — progresso cheio, zero arquivos. O startup chama
// InitWriteGuard (GameFilesRoot vira <raiz>/data) e o destino passa a ser
// "mods/edits/…": aceito, e data/partN/ continua bloqueado.
func TestExportWritesIntoDataModsEdits(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data")
	t.Setenv("FF_RESOURCES_DATA", data)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(data, "mods"))

	prev := ffxcommon.GameFilesRoot
	t.Cleanup(func() { ffxcommon.GameFilesRoot = prev })

	// Estado "antes do fix": GameFilesRoot = <raiz> (dir do executável).
	ffxcommon.GameFilesRoot = root
	target := filepath.Join(data, "mods", "edits", "text_all_localizations_part1.json")
	if err := ffxcommon.CheckWritablePath(target); err == nil {
		t.Fatal("sem InitWriteGuard, CheckWritablePath deveria recusar data/mods/edits")
	}

	// O startup do app: GameFilesRoot vira <raiz>/data.
	InitWriteGuard()
	if ffxcommon.GameFilesRoot != filepath.Clean(data) {
		t.Fatalf("GameFilesRoot: %q, quer %q", ffxcommon.GameFilesRoot, data)
	}
	if err := ffxcommon.CheckWritablePath(target); err != nil {
		t.Fatalf("após InitWriteGuard, mods/edits deve ser gravável: %v", err)
	}
	// A proteção continua: data/partN/ segue somente leitura.
	if err := ffxcommon.CheckWritablePath(filepath.Join(data, "part1", "x.ztr")); err == nil {
		t.Fatal("data/partN/ deveria continuar bloqueado")
	}

	ids := []string{"id1"}
	writeFile(t, filepath.Join(data, "part1", "txtres", "txtres_us.ztr"),
		testZtrBlob(t, ids, []string{"Hello"}))

	var svc Service
	paths, err := svc.ExportArtifactBatch([]ExportPlanItem{{Kind: "ztr"}}, "part1", nil, "json")
	if err != nil {
		t.Fatalf("ExportArtifactBatch: %v", err)
	}
	want := filepath.Join(data, "mods", "edits", "text_all_localizations_part1.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("paths: %v, quer [%s]", paths, want)
	}
	raw, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("artefato não gravado em %s: %v", want, err)
	}
	if !strings.Contains(string(raw), "Hello") {
		t.Errorf("artefato sem conteúdo: %s", raw)
	}
}

// TestExportRealDataSmoke roda o export de UMA unidade com os dados reais de
// bin/data (pula se ausente) no layout real do app, e confere que o artefato
// sai em bin/data/mods/edits. Remove o artefato no fim.
func TestExportRealDataSmoke(t *testing.T) {
	data := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data")
	if _, err := os.Stat(data); err != nil {
		t.Skip("bin/data ausente")
	}
	t.Setenv("FF_RESOURCES_DATA", data)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(data, "mods"))

	prev := ffxcommon.GameFilesRoot
	t.Cleanup(func() { ffxcommon.GameFilesRoot = prev })
	InitWriteGuard()

	var svc Service
	units, err := svc.ListTextUnits("part1")
	if err != nil || len(units) == 0 {
		t.Skipf("ListTextUnits(part1): %v (n=%d)", err, len(units))
	}
	u := units[0]
	paths, err := svc.ExportEntry("part1", u.ID, nil)
	if err != nil {
		t.Fatalf("ExportEntry(%s): %v", u.ID, err)
	}
	if len(paths) == 0 {
		t.Fatal("ExportEntry devolveu zero caminhos")
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artefato ausente %s: %v", p, err)
		} else {
			t.Logf("OK %s", p)
		}
		os.Remove(p)
	}
}
