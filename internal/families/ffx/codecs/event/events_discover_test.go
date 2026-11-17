package event_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	ffxencoding "ffresources/internal/families/ffx/codecs/encoding"
	"ffresources/internal/families/ffx/codecs/event"
	"ffresources/internal/families/ffx/profile/common"
	"ffresources/testdata/ffx/tempdir"
)

// seedEventOnDisk grava um evento VÁLIDO nas duas árvores (mods/ via export,
// data/ por cópia do binário da localização padrão) - o suficiente para a
// descoberta (nome) e a leitura (conteúdo).
func seedEventOnDisk(t *testing.T, version common.GameVersion, id string, texts []string) {
	t.Helper()
	charset := ffxencoding.GetCharsetForLanguage(common.DefaultLocalization)
	objs := make([]*event.LocalizedFieldStringObject, 0, len(texts))
	for _, text := range texts {
		fs := event.NewEmptyFieldString(charset, version)
		fs.SetRegularString(text)
		objs = append(objs, event.NewLocalizedFieldStringObjectWithContent(common.DefaultLocalization, fs))
	}
	event.SetEvent(version, id, &event.EventFile{ID: id, Version: version, Strings: objs})
	if err := event.ExportEventStringsToLocalizations(version, id); err != nil {
		t.Fatalf("export %s: %v", id, err)
	}
	rel, err := event.EventRelPath(id)
	if err != nil {
		t.Fatalf("relpath %s: %v", id, err)
	}
	lroot := common.GetLocalizationRootForVersion(version, common.DefaultLocalization)
	src := filepath.Join(common.GameFilesRoot, common.ModsFolder, lroot, rel)
	dst := filepath.Join(common.GameFilesRoot, lroot, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ler binário exportado: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func prepareEventTestRoot(t *testing.T) {
	t.Helper()
	prevRoot, prevMods := common.GameFilesRoot, common.DisableMods
	common.GameFilesRoot = tempdir.TempDir(t)
	common.DisableMods = false
	t.Cleanup(func() {
		common.GameFilesRoot = prevRoot
		common.DisableMods = prevMods
		event.ClearEvents(common.GameVersionFFX)
	})
	if err := ffxencoding.EnsureAllCharsetsLoaded(common.GameVersionFFX); err != nil {
		// Charsets vêm de dados do jogo (não das fixtures): prepara pelo mapa
		// estático quando o ambiente de teste ainda não carregou.
		for _, cs := range common.Charsets {
			if perr := ffxencoding.PrepareCharset(common.GameVersionFFX, cs); perr != nil {
				t.Fatal(perr)
			}
		}
	}
}

// TestDiscoverSeparaUniversoDeConteudo: a descoberta publica o universo de ids
// (só ReadDir) sem ler nenhum evento - é o que permite listar a árvore
// completa enquanto o conteúdo carrega. Pasta sem <id>.bin não é evento e a
// lista estática de skip continua fora do universo da árvore.
func TestDiscoverSeparaUniversoDeConteudo(t *testing.T) {
	prepareEventTestRoot(t)
	version := common.GameVersionFFX
	event.ClearEvents(version)

	seedEventOnDisk(t, version, "zev001", []string{"Frase do zev001"})
	seedEventOnDisk(t, version, "zev002", []string{"Frase do zev002"})

	// Os arquivos já estão no disco: zera o store para provar que a
	// descoberta lista o universo sem depender de conteúdo carregado.
	event.ClearEvents(version)

	objRoot := filepath.Join(common.GameFilesRoot,
		common.GetLocalizationRootForVersion(version, common.DefaultLocalization),
		"event", "obj_ps3")

	// Pasta de evento SEM o binário homônimo: não é evento.
	if err := os.MkdirAll(filepath.Join(objRoot, "ze", "zev003"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Id da lista estática de skip (existe em disco, não na árvore).
	skipDir := filepath.Join(objRoot, "sy", "sysfp")
	if err := os.MkdirAll(skipDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skipDir, "sysfp.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := event.NewEventsBinaryFile(version)
	if err := b.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if want := []string{"zev001", "zev002"}; !reflect.DeepEqual(b.Discovered, want) {
		t.Fatalf("universo descoberto = %v, quero %v", b.Discovered, want)
	}
	// Descoberta NÃO publica conteúdo: a lista pode ser servida antes da carga.
	if got := event.GetAllEventIDs(version); len(got) != 0 {
		t.Fatalf("Discover publicou conteúdo no store: %v", got)
	}

	if err := b.LoadDiscovered(); err != nil {
		t.Fatalf("LoadDiscovered: %v", err)
	}
	got := event.GetAllEventIDs(version)
	if len(got) != 2 {
		t.Fatalf("store após a carga = %v, quero 2 eventos", got)
	}

	// O inventário de mods usa a descoberta de DISCO (inclui a skip list):
	// sysfp existe em data/ e não pode ser denunciado como "só em mods".
	dataIDs, err := event.DiscoverEventIDsFromData(version)
	if err != nil {
		t.Fatalf("DiscoverEventIDsFromData: %v", err)
	}
	has := func(ids []string, id string) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}
	if !has(dataIDs, "sysfp") {
		t.Fatal("a descoberta de disco deve incluir os ids da lista de skip")
	}
	if has(dataIDs, "zev003") {
		t.Fatal("pasta sem binário não pode entrar na descoberta de disco")
	}
}
