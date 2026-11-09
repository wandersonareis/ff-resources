// Command ff13import importa uma tradução PRONTA (7z ou árvore extraída) para
// o layout do app FF13: extrai os ORIGINAIS que faltam do filelist/white bin
// para data/partN e grava os arquivos traduzidos em mods/partN.
//
// Layout de destino (decidido com o usuário):
//
//	data/part1/
//	  gui/ txtres/ vfx/            conteúdo GLOBAL (por caminho interno)
//	  zone/<zona>/txtres/...       conteúdo de ZONA (uma pasta por zona, sem
//	                               região c/u; todos os idiomas)
//	mods/part1/                     espelho com os arquivos TRADUZIDOS
//
// O nome do índice filelist e o white bin par saem do modelo de parts (o
// mesmo do app), então isto não é um "if part==" espalhado.
//
// Uso:
//
//	go run ./scripts/ff13import -part part1 ^
//	  -game "D:\Steam\steamapps\common\FINAL FANTASY XIII" ^
//	  -7z "D:\MEGA\FF13-1\FF13-1.7z" ^
//	  -data "<bin>\data\part1" -mods "<bin>\data\mods\part1"
//
// A tradução é externa: entra só pela linha de comando do utilitário. O app
// nunca lê de fora de data/, mods/ e da pasta do jogo.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"ffresources/apps/ff13/parts"
	"ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
)

// extForeign são extensões EXTERNAS ao filelist: nunca entram.
var extForeign = map[string]bool{
	".exe": true, ".cmd": true, ".bat": true, ".dll": true,
	".png": true, ".txt": true, ".psd": true, ".dds": true, ".strings": true,
}

// index é um filelist carregado + o white bin par e o mapa caminho->caminho.
type index struct {
	name    string
	region  string // "u"/"c"/"" (global)
	bin     string
	raw     []byte // bytes do índice, para copiar para data/partN/filelist
	coll    *filelist.Collection
	entries map[string]string // lower(path) -> path
}

type zone struct {
	id      string // "z004"
	byPath  map[string]*index
	primary *index // região usada para extrair os originais (u quando existe)
}

type manifest struct {
	Part        string        `json:"part"`
	Translation string        `json:"translation"`
	GameRoot    string        `json:"gameRoot"`
	DataDir     string        `json:"dataDir"`
	ModsDir     string        `json:"modsDir"`
	DataFiles   int           `json:"dataFiles"`
	ModFiles    int           `json:"modFiles"`
	Content     int           `json:"content"`
	Skipped     []string      `json:"skipped,omitempty"`
	Errors      []string      `json:"errors,omitempty"`
	Entries     []manifestRow `json:"entries"`
}

type manifestRow struct {
	Source   string `json:"source"`
	Internal string `json:"internal,omitempty"`
	Zone     string `json:"zone,omitempty"`
	Target   string `json:"target,omitempty"`
	Matched  bool   `json:"matched"`
	Error    string `json:"error,omitempty"`
}

func main() {
	var (
		part        = flag.String("part", "", "part1, part2 ou part3")
		gameRoot    = flag.String("game", "", "raiz da instalação do jogo")
		dataDir     = flag.String("data", "", "diretório data/partN (originais)")
		modsDir     = flag.String("mods", "", "diretório mods/partN (traduzidos)")
		translation = flag.String("translation", "", "árvore da tradução JÁ extraída")
		sevenZip    = flag.String("7z", "", "arquivo .7z da tradução (extrai sozinho)")
		force       = flag.Bool("force", false, "reextrair mesmo se o arquivo já existir")
	)
	flag.Parse()
	if *part == "" || *gameRoot == "" || *dataDir == "" || *modsDir == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *translation == "" && *sevenZip == "" {
		fmt.Fprintln(os.Stderr, "ff13import: informe -translation <dir> ou -7z <arquivo>")
		os.Exit(2)
	}
	if err := run(*part, *gameRoot, *dataDir, *modsDir, *translation, *sevenZip, *force); err != nil {
		fmt.Fprintln(os.Stderr, "ff13import:", err)
		os.Exit(1)
	}
}

func run(part, gameRoot, dataDir, modsDir, translation, sevenZip string, force bool) error {
	p := parts.Get(part)
	gameData := p.DataDir(gameRoot)
	if _, err := os.Stat(gameData); err != nil {
		return fmt.Errorf("pasta de dados do jogo: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		return err
	}

	// 1) Índices globais (em disco) + seus bins.
	bins := scanBins(p, gameData)
	globals := loadGlobals(p, gameData, bins)
	if len(globals) == 0 {
		return fmt.Errorf("nenhum filelist global carregado em %s", gameData)
	}

	// 2) Índices de zona embutidos no bin global.
	zones := discoverZones(p, bins, globals)

	// 2b) Copia os índices para data/partN/filelist (app autossuficiente).
	if err := exportIndexes(dataDir, globals, zones); err != nil {
		return err
	}

	// 3) Extrai os ORIGINAIS de texto (todos os idiomas) para data/.
	dataFiles := extractOriginals(p, globals, zones, dataDir, force)

	// 4) Tradução: resolve a árvore e importa o conteúdo para mods/.
	m := manifest{
		Part: part, Translation: translation, GameRoot: gameRoot,
		DataDir: dataDir, ModsDir: modsDir, DataFiles: dataFiles,
	}
	srcRoot := translation
	if srcRoot == "" {
		tmp, err := extract7z(sevenZip)
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		srcRoot = tmp
	}
	m.Translation = srcRoot
	if err := importTranslation(globals, zones, srcRoot, dataDir, modsDir, &m); err != nil {
		return err
	}
	m.ModFiles = countFiles(modsDir)
	sort.Slice(m.Entries, func(i, j int) bool { return m.Entries[i].Source < m.Entries[j].Source })

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(modsDir, "_import.json"), raw, 0o644); err != nil {
		return err
	}
	// O manifesto antigo (data/) não é mais a fonte da verdade.
	_ = os.Remove(filepath.Join(dataDir, "_import.json"))
	fmt.Printf("ff13import %s: %d originais em data, %d traduzidos em mods, %d sem par, %d erros\n",
		part, m.DataFiles, m.Content, len(m.Skipped), len(m.Errors))
	for _, e := range m.Errors {
		fmt.Fprintln(os.Stderr, "  erro:", e)
	}
	return nil
}

// ---- descoberta de índices ----

// scanBins indexa os white bins por chave de par, do mesmo jeito que o app.
func scanBins(p parts.Part, gameData string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(gameData, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "white") ||
			!strings.HasSuffix(lower, ".bin") ||
			strings.HasSuffix(lower, ".bin2") {
			return nil
		}
		// `white_zXXXu_img2.win32.bin` (2º arquivo) NÃO é o bin que o índice
		// `filelist_zXXXu` descreve, e o PairKey colapsa os dois na mesma
		// chave. Fica só o bin principal (`..._img.win32.bin`).
		if strings.Contains(lower, "_img2") {
			return nil
		}
		if key, ok := p.PairKey(name); ok {
			out[key] = path
		}
		return nil
	})
	return out
}

// loadGlobals carrega os filelists em disco (sys/) e pareia com o white bin.
func loadGlobals(p parts.Part, gameData string, bins map[string]string) []*index {
	out := []*index{}
	_ = filepath.WalkDir(gameData, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !isFilelistName(name) {
			return nil
		}
		key, ok := p.PairKey(name)
		if !ok {
			return nil
		}
		bin := bins[key]
		if bin == "" {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		coll, cerr := filelist.Parse(raw, p.GameCode())
		if cerr != nil {
			return nil
		}
		out = append(out, newIndex(name, regionOf(key), bin, raw, coll))
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// discoverZones extrai os índices `zone/filelist_zXXX` de dentro dos bins
// globais, lê e pareia com o bin de zona `white_zXXX{c,u}_img`.
func discoverZones(p parts.Part, bins map[string]string, globals []*index) map[string]*zone {
	out := map[string]*zone{}
	for _, g := range globals {
		for _, ep := range g.coll.Paths() {
			base := filepath.Base(ep)
			if !isFilelistName(base) {
				continue
			}
			key, ok := p.PairKey(base)
			if !ok {
				continue
			}
			bin := bins[key]
			if bin == "" {
				continue
			}
			raw, coll, err := readEmbedded(g.bin, g.coll, ep, p.GameCode())
			if err != nil {
				continue
			}
			zid := zoneID(key)
			z := out[zid]
			if z == nil {
				z = &zone{id: zid, byPath: map[string]*index{}}
				out[zid] = z
			}
			idx := newIndex(base, regionOf(key), bin, raw, coll)
			z.byPath[regionOf(key)] = idx
			if z.primary == nil || regionOf(key) == "u" {
				z.primary = idx
			}
		}
	}
	return out
}

func newIndex(name, region, bin string, raw []byte, coll *filelist.Collection) *index {
	idx := &index{name: name, region: region, bin: bin, raw: raw, coll: coll, entries: map[string]string{}}
	_ = coll.EachEntry(func(_ filelist.EntryRef, e *filelist.Entry) error {
		path := strings.ReplaceAll(filelist.EntryPath(e.FileInfo), "\\", "/")
		path = strings.TrimSpace(path)
		if path == "" || path == " " {
			return nil
		}
		idx.entries[strings.ToLower(path)] = path
		return nil
	})
	return idx
}

// readEmbedded extrai um índice embutido do bin global, devolve os bytes e o
// parseia.
func readEmbedded(bin string, coll *filelist.Collection, virtual string, game filelist.GameCode) ([]byte, *filelist.Collection, error) {
	tmp, err := os.MkdirTemp("", "ff13idx")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(tmp)
	res, err := whitebin.ExtractFile(bin, tmp, virtual, coll)
	if err != nil {
		return nil, nil, err
	}
	if len(res) == 0 {
		return nil, nil, fmt.Errorf("índice embutido vazio: %s", virtual)
	}
	raw, err := os.ReadFile(res[0].OutFile)
	if err != nil {
		return nil, nil, err
	}
	parsed, err := filelist.Parse(raw, game)
	if err != nil {
		return nil, nil, err
	}
	return raw, parsed, nil
}

// exportIndexes grava os índices em data/partN/filelist (globais) e
// filelist/zone (zonas), para o app funcionar sem a pasta do jogo. Remove os
// índices soltos que a versão anterior deixava em data/partN/zone.
func exportIndexes(dataDir string, globals []*index, zones map[string]*zone) error {
	root := filepath.Join(dataDir, "filelist")
	if err := os.MkdirAll(filepath.Join(root, "zone"), 0o755); err != nil {
		return err
	}
	if old, _ := filepath.Glob(filepath.Join(dataDir, "zone", "filelist_*.bin")); len(old) > 0 {
		for _, f := range old {
			_ = os.Remove(f)
		}
	}
	for _, g := range globals {
		if len(g.raw) == 0 {
			continue
		}
		if err := os.WriteFile(filepath.Join(root, g.name), g.raw, 0o644); err != nil {
			return err
		}
	}
	for _, z := range zones {
		for _, idx := range z.byPath {
			if len(idx.raw) == 0 {
				continue
			}
			if err := os.WriteFile(filepath.Join(root, "zone", idx.name), idx.raw, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- extração dos originais ----

// extractOriginals extrai para data/ todo `.ztr` (todos os idiomas) dos
// globais e das zonas. Devolve quantos arquivos data/ tem no fim.
func extractOriginals(p parts.Part, globals []*index, zones map[string]*zone, dataDir string, force bool) int {
	for _, g := range globals {
		for key, path := range g.entries {
			if !isTextPath(path) {
				continue
			}
			_ = extract(g.bin, dataDir, path, g.coll, force)
			_ = key
		}
	}
	for _, z := range zones {
		if z.primary == nil {
			continue
		}
		out := filepath.Join(dataDir, "zone", z.id)
		for _, path := range z.primary.entries {
			if !isTextPath(path) {
				continue
			}
			_ = extract(z.primary.bin, out, path, z.primary.coll, force)
		}
	}
	return countFiles(dataDir)
}

// extract grava outDir/<virtual> (pula se já existe, salvo force).
func extract(bin, outDir, virtual string, coll *filelist.Collection, force bool) error {
	dst := filepath.Join(outDir, filepath.FromSlash(virtual))
	if !force {
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
	}
	_, err := whitebin.ExtractFile(bin, outDir, virtual, coll)
	return err
}

// ---- importação da tradução ----

func importTranslation(globals []*index, zones map[string]*zone, srcRoot, dataDir, modsDir string, m *manifest) error {
	globalByPath := map[string]*index{}
	for _, g := range globals {
		for k := range g.entries {
			if _, ok := globalByPath[k]; !ok {
				globalByPath[k] = g
			}
		}
	}
	// Mapa caminho interno -> zona (união de c/u de todas as zonas).
	zoneByInternal := map[string]string{}
	for id, z := range zones {
		for _, idx := range z.byPath {
			for k := range idx.entries {
				if _, ok := zoneByInternal[k]; !ok {
					zoneByInternal[k] = id
				}
			}
		}
	}

	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(srcRoot, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		base := filepath.Base(rel)

		if strings.HasPrefix(strings.ToLower(base), "filelist") {
			m.Skipped = append(m.Skipped, rel+" (índice)")
			return nil
		}
		if extForeign[strings.ToLower(filepath.Ext(base))] {
			m.Skipped = append(m.Skipped, rel+" (externo)")
			return nil
		}

		internal, zoneID, fromZone, ok := classify(rel, zoneByInternal)
		if !ok {
			m.Skipped = append(m.Skipped, rel+" (fora da árvore)")
			return nil
		}

		// Alvo no layout do app.
		target := internal
		origDir := dataDir
		if fromZone {
			if zoneID == "" {
				// Conteúdo de zona que nenhum índice declara: sem lugar no layout.
				m.Entries = append(m.Entries, manifestRow{Source: rel, Internal: internal})
				return nil
			}
			target = filepath.ToSlash(filepath.Join("zone", zoneID, internal))
			origDir = filepath.Join(dataDir, "zone", zoneID)
		}

		// Origem (white bin + coleção + caminho interno) e validação de par.
		var bin string
		var coll *filelist.Collection
		if fromZone {
			z := zones[zoneID]
			if z == nil || z.byPath["u"] == nil {
				m.Entries = append(m.Entries, manifestRow{Source: rel, Internal: internal, Zone: zoneID, Target: target})
				return nil
			}
			coll = z.byPath["u"].coll
			bin = z.byPath["u"].bin
			if _, ok := z.byPath["u"].entries[strings.ToLower(internal)]; !ok {
				// Pode estar só no c.
				if z.byPath["c"] != nil {
					coll = z.byPath["c"].coll
					bin = z.byPath["c"].bin
				}
			}
		} else {
			g, ok := globalByPath[strings.ToLower(internal)]
			if !ok {
				m.Entries = append(m.Entries, manifestRow{Source: rel, Internal: internal, Target: target})
				return nil
			}
			bin, coll = g.bin, g.coll
		}

		// Garante o ORIGINAL em data/ (mesmo layout do alvo).
		if err := extract(bin, origDir, internal, coll, false); err != nil {
			m.Errors = append(m.Errors, fmt.Sprintf("%s: %v", rel, err))
			m.Entries = append(m.Entries, manifestRow{Source: rel, Internal: internal, Zone: zoneID, Target: target, Error: err.Error()})
			return nil
		}

		// Copia o TRADUZIDO para mods/ no mesmo layout.
		dst := filepath.Join(modsDir, filepath.FromSlash(target))
		if err := copyFile(path, dst); err != nil {
			m.Errors = append(m.Errors, fmt.Sprintf("%s: %v", rel, err))
			m.Entries = append(m.Entries, manifestRow{Source: rel, Internal: internal, Zone: zoneID, Target: target, Error: err.Error()})
			return nil
		}
		m.Content++
		m.Entries = append(m.Entries, manifestRow{Source: rel, Internal: internal, Zone: zoneID, Target: target, Matched: true})
		return nil
	})
}

// classify separa o caminho da tradução em (interno, zona). Os prefixos são a
// raiz virtual que o instalador original usa: sys/FF13S (global) e
// zone/FF13Z (conteúdo de zona, injetado em todos os bins de zona).
func classify(rel string, zoneByInternal map[string]string) (internal, zoneID string, fromZone, ok bool) {
	slash := filepath.ToSlash(rel)
	for _, pre := range []string{"sys/FF13S/", "sys/ff13s/"} {
		if strings.HasPrefix(strings.ToLower(slash), strings.ToLower(pre)) {
			return slash[len(pre):], "", false, true
		}
	}
	for _, pre := range []string{"zone/FF13Z/", "zone/ff13z/"} {
		if strings.HasPrefix(strings.ToLower(slash), strings.ToLower(pre)) {
			internal = slash[len(pre):]
			zoneID = zoneByInternal[strings.ToLower(internal)]
			return internal, zoneID, true, true
		}
	}
	return "", "", false, false
}

// ---- utilitários ----

func isFilelistName(name string) bool {
	n := strings.ToLower(name)
	return strings.HasPrefix(n, "filelist") &&
		strings.HasSuffix(n, ".bin") &&
		!strings.HasSuffix(n, ".bin2") &&
		!strings.Contains(n, ".info.")
}

func isTextPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".ztr")
}

// regionOf devolve a região (u/c) da chave de par ("z004u" -> "u").
func regionOf(key string) string {
	k := strings.ToLower(key)
	if strings.HasSuffix(k, "u") || strings.HasSuffix(k, "c") {
		return k[len(k)-1:]
	}
	return ""
}

// zoneID remove a região da chave ("z004u" -> "z004").
func zoneID(key string) string {
	k := strings.ToLower(key)
	if strings.HasSuffix(k, "u") || strings.HasSuffix(k, "c") {
		return k[:len(k)-1]
	}
	return k
}

func copyFile(src, dst string) error {
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func countFiles(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		n++
		return nil
	})
	return n
}

// extract7z extrai o .7z para uma pasta temporária com o 7z do sistema.
func extract7z(archive string) (string, error) {
	tmp, err := os.MkdirTemp("", "ff13trans")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("7z", "x", archive, "-o"+tmp, "-y")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tmp)
		return "", fmt.Errorf("extraindo %s: %v (%s)", archive, err, strings.TrimSpace(string(out)))
	}
	return tmp, nil
}
