package services

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"ffresources/internal/families/ffx/codecs/event"
	"ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/shared/logging"
	"ffresources/testdata/ffx/tempdir"
)

// TestEventsListaCompletaAntesDoConteudo cobre a regra nova: a lista sai da
// DESCOBERTA (disco) e pode ser servida com o store vazio; o conteúdo chega
// depois, arquivo por arquivo. Abrir um evento nesse intervalo espera a carga
// e resolve - nunca "entry not found".
func TestEventsListaCompletaAntesDoConteudo(t *testing.T) {
	svc := seedDisplayRoot(t)
	version := common.GameVersionFFX2

	ids := []string{"zload001", "zload002", "zload003"}
	for _, id := range ids {
		seedEventsSynthetic(version, id, []string{"Frase do " + id})
		seedEventsData(t, version, id) // data/ + mods/
	}

	// Store vazio: a listagem NÃO pode depender do conteúdo já carregado.
	event.ClearEvents(version)
	resetEventsLoadStates()
	clearDedupViewCache()
	t.Cleanup(resetEventsLoadStates)
	t.Cleanup(clearDedupViewCache)

	entries, err := svc.ListEntries(KindEvents, version)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != len(ids) {
		t.Fatalf("lista com store vazio = %d entradas, quero %d", len(entries), len(ids))
	}
	for i, id := range ids {
		if entries[i].ID != id {
			t.Fatalf("entrada %d = %q, quero %q", i, entries[i].ID, id)
		}
	}

	// Abre todos em paralelo: uma única carga (single-flight) e nenhuma
	// entrada pode falhar por o store estar parcial no momento do clique.
	var wg sync.WaitGroup
	errs := make(chan error, len(ids))
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			entry, gerr := svc.GetEntry(KindEvents, id, version)
			if gerr != nil {
				errs <- gerr
				return
			}
			if len(entry.Rows) == 0 {
				errs <- os.ErrInvalid
			}
		}(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("GetEntry concorrente: %v", err)
	}

	// Depois da carga não sobra folha pendente.
	ready, err := svc.ListEntries(KindEvents, version)
	if err != nil {
		t.Fatalf("list pós-carga: %v", err)
	}
	for _, e := range ready {
		if e.Status == "loading" {
			t.Fatalf("%s continuou em loading depois da carga", e.ID)
		}
	}
}

// TestInventarioEventsIgnoraObjPSV: a árvore mods/ do ffx2 tem obj_ps3 E
// obj_psv com os MESMOS ids. O inventário só olha obj_ps3 - a cópia de PSV
// não é entrada de evento e não pode virar aviso "só em mods".
func TestInventarioEventsIgnoraObjPSV(t *testing.T) {
	svc := diagnosticsRoot(t)
	version := common.GameVersionFFX2

	seedEventsSynthetic(version, "zpv001", []string{"Frase do zpv001"})
	seedEventsData(t, version, "zpv001")

	lroot := common.GetLocalizationRootForVersion(version, common.DefaultLocalization)
	rel, err := event.EventRelPath("zpv001")
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(common.GameFilesRoot, common.ModsFolder, lroot, rel)
	dst := filepath.Join(common.GameFilesRoot, common.ModsFolder, lroot,
		strings.Replace(rel, "obj_ps3", "obj_psv", 1))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}

	resetLogFresh(t)
	if _, err := svc.ListEntries(KindEvents, version); err != nil {
		t.Fatalf("list: %v", err)
	}
	if content := logFileContent(t); strings.Contains(content, "regra 4") {
		t.Fatalf("inventário denunciou a cópia obj_psv:\n%s", content)
	}
}

// TestInventarioEventsIgnoraListaDeSkip: ids da lista estática de skip existem
// em data/ e em mods/, mas ficam fora do store (por decisão da árvore). O
// inventário compara ARQUIVOS em disco - não pode chamá-los de "só em mods".
func TestInventarioEventsIgnoraListaDeSkip(t *testing.T) {
	seedEventsTempRoot(t)
	version := common.GameVersionFFX
	event.ClearEvents(version)
	resetEventsLoadStates()
	t.Cleanup(func() {
		event.ClearEvents(version)
		resetEventsLoadStates()
	})
	if err := ensureVersionReady(version); err != nil {
		t.Fatalf("prepare version: %v", err)
	}
	loggingService.ResetForTest(tempdir.TempDir(t))
	svc := NewMetadataService(nil)

	// sysfp está na lista de skip do FFX e existe em data/ + mods/.
	seedEventsSynthetic(version, "sysfp", []string{"Frase do sysfp"})
	seedEventsData(t, version, "sysfp")
	lroot := common.GetLocalizationRootForVersion(version, common.DefaultLocalization)
	rel, err := event.EventRelPath("sysfp")
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(common.GameFilesRoot, common.ModsFolder, lroot, rel)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(common.GameFilesRoot, lroot, rel)
	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.ListEntries(KindEvents, version); err != nil {
		t.Fatalf("list: %v", err)
	}
	content := logFileContent(t)
	if strings.Contains(content, "regra 4") {
		t.Fatalf("inventário denunciou um arquivo da lista de skip:\n%s", content)
	}
}
