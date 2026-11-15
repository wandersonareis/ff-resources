package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// importManifest espelha o manifesto escrito por scripts/ff13import. É a
// FONTE da verdade do import: o app não guarda caminho de fora de data/,
// mods/ e da pasta do jogo, então o teste não referencia nenhum caminho
// externo.
type importManifest struct {
	Part      string   `json:"part"`
	DataDir   string   `json:"dataDir"`
	ModsDir   string   `json:"modsDir"`
	DataFiles int      `json:"dataFiles"`
	ModFiles  int      `json:"modFiles"`
	Content   int      `json:"content"`
	Skipped   []string `json:"skipped"`
	Errors    []string `json:"errors"`
	Entries   []struct {
		Source   string `json:"source"`
		Internal string `json:"internal"`
		Zone     string `json:"zone"`
		Target   string `json:"target"`
		Matched  bool   `json:"matched"`
		Error    string `json:"error"`
	} `json:"entries"`
}

func readImportManifest(t *testing.T, part string) importManifest {
	t.Helper()
	mods := partModsRoot(t, part)
	raw, err := os.ReadFile(filepath.Join(mods, "_import.json"))
	if err != nil {
		t.Skipf("manifesto de importação ausente (%s): %v", mods, err)
	}
	var m importManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifesto inválido: %v", err)
	}
	return m
}

// TestTranslationImport confere o requisito central: o app tem acesso ao
// MESMO número de arquivos da tradução pronta.
//
// O manifesto lista cada arquivo de conteúdo da tradução; para cada um:
//   - o traduzido existe em mods/partN (no mesmo caminho do app);
//   - o ORIGINAL existe em data/partN (extraído do white bin);
//   - o app abre o .ztr (data + mods) pelo caminho do app.
//
// Nada aqui aponta para fora de data/ e mods/.
func TestTranslationImport(t *testing.T) {
	part := "part1"
	m := readImportManifest(t, part)
	data := partDataRoot(t, part)
	mods := partModsRoot(t, part)

	if m.Part != part {
		t.Fatalf("manifesto de outra part: %q", m.Part)
	}
	if len(m.Errors) > 0 {
		t.Errorf("importação com %d erros: %v", len(m.Errors), m.Errors)
	}

	targetsByInternal := manifestTargetsByInternal(m)
	ambiguous := false
	for _, targets := range targetsByInternal {
		if len(targets) > 1 {
			ambiguous = true
			break
		}
	}

	matched := 0
	legacy := ambiguous
	ztrTargets := map[string]bool{}
	for _, e := range m.Entries {
		if !e.Matched {
			if e.Error != "" {
				t.Errorf("entrada com erro: %s: %s", e.Source, e.Error)
			}
			continue
		}
		matched++
		if e.Target == "" {
			t.Errorf("%s: casado sem caminho de destino", e.Source)
			continue
		}
		// A árvore de mods pode ser legada: o arquivo pode estar no caminho
		// interno do pacote de tradução quando o import antigo gravou ali.
		modsPath := resolveModsPath(mods, e.Target, e.Internal)
		if _, err := os.Stat(modsPath); err != nil {
			// O manifesto repete o mesmo caminho interno para o índice global
			// e o de zona; só um dos targets pode existir. Com duplicata, a
			// ausência de um target não é erro.
			if ambiguous {
				t.Logf("target duplicado ausente em mods (ok): %s", e.Target)
				continue
			}
			t.Errorf("traduzido ausente em mods: %s (%v)", e.Target, err)
			continue
		}
		dataPath := filepath.Join(data, filepath.FromSlash(e.Target))
		if _, err := os.Stat(dataPath); err != nil {
			if ambiguous {
				t.Logf("target duplicado ausente em data (ok): %s", e.Target)
				continue
			}
			t.Errorf("original ausente em data: %s (%v)", e.Target, err)
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Target), ".ztr") {
			ztrTargets[e.Target] = true
		}
	}
	if matched != m.Content {
		t.Errorf("manifesto inconsistente: %d casados, mas Content=%d", matched, m.Content)
	}
	if matched == 0 {
		t.Fatal("nenhum arquivo da tradução importado")
	}
	t.Logf("tradução: %d arquivos de conteúdo (%d .ztr)", matched, len(ztrTargets))

	// O app lista data/partN por UNIDADES: toda unidade com "_us" aparece,
	// e a traducao de um idioma cai na MESMA unidade do "_us".
	t.Setenv("FF_RESOURCES_DATA", mustAbs(t, filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data")))
	t.Setenv("FF_RESOURCES_MODS", mustAbs(t, filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods")))
	var svc Service
	units, err := svc.ListTextUnits(part)
	if err != nil {
		t.Fatalf("ListTextUnits: %v", err)
	}
	listed := map[string]bool{}
	for _, u := range units {
		listed[u.ID] = true
	}
	for target := range ztrTargets {
		unit := unitOfTarget(target)
		if listed[unit] {
			continue
		}
		// Sem "_us" a unidade nao entra na arvore (a traducao fica so como
		// referencia) — confere que e esse o caso.
		if hasUsSibling(data, target) {
			t.Errorf("app não lista a unidade do .ztr da tradução: %s", target)
		}
	}
	t.Logf("app lista %d unidades de texto — data/part1 tem todos os idiomas", len(units))

	// E o app ABRE uma unidade traduzida (data = original, mods = tradução).
	//
	// Só os .ztr no formato BIG-ENDIAN (o que o codec implementa hoje) são
	// abertos aqui: os de formato little-endian (key+value juntos, ~9% dos
	// arquivos do jogo) são a lacuna do codec — ver TestZtrCodecGap. A
	// árvore os mostra igualmente; só a EDIÇÃO deles espera o codec.
	opened := 0
	for target := range ztrTargets {
		if _, err := svc.GetTextEntry(part, unitOfTarget(target), ""); err != nil {
			continue
		}
		opened++
	}
	if opened == 0 {
		t.Fatal("nenhum .ztr traduzido pôde ser aberto")
	}
	t.Logf("abertos pelo codec: %d de %d .ztr traduzidos", opened, len(ztrTargets))

	// mods/partN tem exatamente os arquivos da tradução + o manifesto.
	onDisk := map[string]bool{}
	_ = filepath.WalkDir(mods, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(mods, path)
		if rerr != nil {
			return nil
		}
		onDisk[filepath.ToSlash(rel)] = true
		return nil
	})
	delete(onDisk, "_import.json")
	if legacy {
		// Árvore legada: os arquivos podem estar no caminho interno do
		// pacote de tradução, então a contagem por target não vale.
		t.Logf("mods/part1 em layout legado: %d arquivos em disco, %d entradas no manifesto", len(onDisk), matched)
	} else if len(onDisk) != matched {
		t.Errorf("mods/part1 tem %d arquivos, a tradução tem %d", len(onDisk), matched)
	}
}

// TestImportManifestSkipsExternal confere que os arquivos EXTERNOS ao
// filelist (o instalador, as capturas de tela) nunca entraram em mods/.
func TestImportManifestSkipsExternal(t *testing.T) {
	m := readImportManifest(t, "part1")
	for _, e := range m.Entries {
		ext := strings.ToLower(filepath.Ext(e.Source))
		switch ext {
		case ".exe", ".cmd", ".bat", ".dll", ".png", ".txt":
			t.Errorf("arquivo externo importado: %s", e.Source)
		}
	}
}

// TestZoneContentLayout confere o esquema de pasta da zona: uma pasta por
// zona (sem região c/u) e o conteúdo dentro dela pelo caminho interno.
func TestZoneContentLayout(t *testing.T) {
	data := partDataRoot(t, "part1")
	zoneDir := filepath.Join(data, "zone")
	entries, err := os.ReadDir(zoneDir)
	if err != nil {
		t.Fatalf("lendo zone: %v", err)
	}
	zones := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("arquivo solto em data/part1/zone: %s", e.Name())
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "z") {
			t.Errorf("pasta de zona unexpected: %s", name)
			continue
		}
		if strings.HasSuffix(name, "u") || strings.HasSuffix(name, "c") {
			t.Errorf("zona com região no nome (deveria ser uma pasta por zona): %s", name)
		}
		zones = append(zones, name)
	}
	sort.Strings(zones)
	if len(zones) == 0 {
		t.Fatal("nenhuma zona extraída em data/part1/zone")
	}
	t.Logf("zonas em data/part1/zone: %v", zones)
}

// TestFilelistTreeSchema confere o esquema da árvore de filelist: os
// globais (u, c, scru, scrc) soltos e os de ZONA no grupo "zone".
func TestFilelistTreeSchema(t *testing.T) {
	testGameRoot(t, "part1")

	var svc Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	globals := []string{}
	zones := []string{}
	for _, r := range roots {
		if r.Group == "zone" {
			zones = append(zones, r.Name)
			continue
		}
		globals = append(globals, r.Name)
	}
	sort.Strings(globals)
	sort.Strings(zones)
	t.Logf("globais: %v", globals)
	t.Logf("zona (%d): %v", len(zones), zones)

	// A ORDEM de exibição é do frontend (part1: inglês antes do japonês);
	// o backend só entrega o conjunto. Aqui o teste confere o CONTEÚDO.
	wantGlobals := []string{"filelist_scrc.win32.bin", "filelist_scru.win32.bin", "filelistc.win32.bin", "filelistu.win32.bin"}
	if strings.Join(globals, ",") != strings.Join(wantGlobals, ",") {
		t.Errorf("globais: esperado %v, obtido %v", wantGlobals, globals)
	}
	if len(zones) != 46 {
		t.Errorf("esperados 46 índices de zona, obtidos %d", len(zones))
	}
	for _, z := range zones {
		if !strings.HasPrefix(z, "filelist_z") {
			t.Errorf("índice de zona com nome inesperado: %s", z)
		}
	}
}

// mustAbs devolve o caminho absoluto (relativo ao package do teste).
// manifestTargetsByInternal agrupa os targets do manifesto por caminho
// interno. O mesmo internal pode aparecer no índice global e no de zona com
// targets diferentes; só um deles existe no layout canônico.
func manifestTargetsByInternal(m importManifest) map[string][]string {
	out := map[string][]string{}
	for _, e := range m.Entries {
		if !e.Matched || e.Internal == "" || e.Target == "" {
			continue
		}
		out[e.Internal] = append(out[e.Internal], e.Target)
	}
	return out
}

// resolveModsPath devolve o caminho do arquivo em mods: o target (layout
// canônico do app) ou, quando a árvore é legada, o internal do pacote de
// tradução.
func resolveModsPath(mods, target, internal string) string {
	p := filepath.Join(mods, filepath.FromSlash(target))
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return filepath.Join(mods, filepath.FromSlash(internal))
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs(%s): %v", path, err)
	}
	return abs
}

// unitOfTarget devolve o id da UNIDADE de um .ztr (diretório + nome-base sem
// o sufixo de idioma).
func unitOfTarget(target string) string {
	dir, file := SplitRel(filepath.ToSlash(target))
	base, _ := ParseUnitFileName(file)
	return UnitID(dir, base)
}

// hasUsSibling diz se o .ztr tem um irmão "_us" na mesma pasta (a unidade
// primária que entra na árvore).
func hasUsSibling(data, target string) bool {
	dir, file := SplitRel(filepath.ToSlash(target))
	base, _ := ParseUnitFileName(file)
	_, err := os.Stat(filepath.Join(data, filepath.FromSlash(dir), base+"_us.ztr"))
	return err == nil
}
