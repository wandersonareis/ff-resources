package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"ffresources/apps/ff13/common"
	"ffresources/apps/ff13/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
	logging "ffresources/internal/shared/logging"
)

var (
	filelistMu    sync.Mutex
	filelistCache = map[string]*fl.Collection{}
)

// partDiscovery é o resultado da descoberta dos índices de uma part: as
// raízes (com o desempate já aplicado) e a cadeia de índices (disco +
// embutidos).
type partDiscovery struct {
	roots   []common.FilelistRoot
	indexes []*filelistIndex
}

// discoverPart descobre os índices da part: os que estão em DISCO (a pasta do
// jogo e a cópia extraída em data/partN/filelist) e os EMBUTIDOS dentro dos
// white bins (a cadeia do FF13: índice global → índice de zona → ...).
//
// Quando o mesmo índice existe em mais de um lugar, fica UM só (D4): vence a
// cópia da pasta de dados da part, depois a que tem white bin par, depois o
// caminho mais curto (empate: ordem alfabética).
func discoverPart(p parts.Part) partDiscovery {
	disk := listDiskRoots(p)
	var indexes []*filelistIndex
	if gameRoot(p.ID()) != "" {
		_, bins := scanPartFiles(p)
		indexes = discoverFilelists(p, disk, bins)
	}

	byName := map[string]common.FilelistRoot{}
	put := func(r common.FilelistRoot) {
		key := strings.ToLower(r.Name)
		if old, ok := byName[key]; ok {
			if betterRoot(r, old, p) {
				byName[key] = r
			}
			return
		}
		byName[key] = r
	}
	for _, r := range disk {
		put(r)
	}
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		name := filepath.Base(filepath.ToSlash(idx.Virtual))
		put(common.FilelistRoot{
			Name:      name,
			Path:      idx.Path,
			Part:      p.ID(),
			SourceDir: p.DataDirName(),
			Source:    common.SourceOf(name),
			Entries:   idx.Coll.FileCount(),
			WhiteBin:  idx.Container,
			Group:     common.GroupZone,
			Zone:      common.ZoneIDOf(name),
			Language:  p.LanguageOf(name),
			Kind:      common.KindEmbedded,
		})
	}

	out := make([]common.FilelistRoot, 0, len(byName))
	for _, r := range byName {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return partDiscovery{roots: out, indexes: indexes}
}

// betterRoot decide qual das DUAS cópias do mesmo índice vence (D4):
//
//  1. a pasta de dados da part (white_data/alba_data/weiss_data) vence a
//     cópia extraída pelo app (data/partN);
//  2. o índice com white bin par vence o sem par;
//  3. caminho mais curto; no empate, caminho em ordem alfabética.
func betterRoot(candidate, current common.FilelistRoot, p parts.Part) bool {
	inDataDir := func(r common.FilelistRoot) bool { return r.SourceDir == p.DataDirName() }
	if inDataDir(candidate) != inDataDir(current) {
		return inDataDir(candidate)
	}
	if (candidate.WhiteBin != "") != (current.WhiteBin != "") {
		return candidate.WhiteBin != ""
	}
	if len(candidate.Path) != len(current.Path) {
		return len(candidate.Path) < len(current.Path)
	}
	return strings.ToLower(candidate.Path) < strings.ToLower(current.Path)
}

// ListFilelistRoots descobre os filelists da PART informada. Onde procurar é
// decisão da part (part1: white_data/sys; part2: alba_data/sys+udp; part3:
// weiss_data/sys+dlc). Os índices EMBUTIDOS (os de zona, que moram dentro do
// bin global) entram como raízes do grupo "zone".
//
// O que sai daqui é DADO CRU: o nome/caminho do índice como estão no disco
// (sem rótulo, sem normalização), a plataforma e o idioma deduzidos do NOME
// do índice, a pasta onde a cópia vencedora mora e a contagem de entradas. A
// árvore e os nomes exibidos são do frontend.
//
// Também valida o par filelist <-> white bin e registra como ERRO todo
// arquivo que ficou sem par.
func (FF13Service) ListFilelistRoots(part string) ([]common.FilelistRoot, error) {
	p := parts.Get(part)
	disc := discoverPart(p)
	checkFilelistPairs(p, disc.indexes)
	return disc.roots, nil
}

// listDiskRoots acha os índices EM DISCO: os da pasta do jogo (sys, udp,
// dlc...) e a cópia em data/partN/filelist (app autossuficiente). O grupo
// "zone" vem da pasta onde o índice mora.
func listDiskRoots(p parts.Part) []common.FilelistRoot {
	dirs := make([]string, 0, 4)
	if root := gameRoot(p.ID()); root != "" {
		base := p.DataDir(root)
		for _, sub := range p.FilelistDirs() {
			dirs = append(dirs, filepath.Join(base, filepath.FromSlash(sub)))
		}
	}
	if data, err := partRoot("data", p.ID()); err == nil {
		dirs = append(dirs, data)
	}

	seen := map[string]bool{}
	out := make([]common.FilelistRoot, 0)
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			name := strings.ToLower(d.Name())
			if !isFilelistName(name) {
				return nil
			}
			clean := filepath.Clean(path)
			if seen[clean] {
				return nil
			}
			seen[clean] = true
			coll, cerr := loadFilelist(clean, p.ID())
			if cerr != nil {
				return nil
			}
			group := filelistGroup(clean)
			root := common.FilelistRoot{
				Name:      d.Name(),
				Path:      clean,
				Part:      p.ID(),
				SourceDir: sourceDirOf(p, clean),
				Source:    common.SourceOf(d.Name()),
				Entries:   coll.FileCount(),
				WhiteBin:  pairOnDisk(p, clean),
				Group:     group,
				Language:  p.LanguageOf(d.Name()),
				Kind:      common.KindDisk,
			}
			if group == common.GroupZone {
				root.Zone = common.ZoneIDOf(d.Name())
			}
			out = append(out, root)
			return nil
		})
	}
	return out
}

// sourceDirOf diz onde a cópia do índice mora: a pasta de dados da part
// (white_data/alba_data/weiss_data) ou a cópia extraída pelo app
// (data/partN). É o "sourceDir" do DTO.
func sourceDirOf(p parts.Part, path string) string {
	if root := gameRoot(p.ID()); root != "" {
		base := p.DataDir(root)
		if base != "" {
			if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
				return p.DataDirName()
			}
		}
	}
	return common.AppDataDir(p.ID())
}

// filelistGroup classifica o índice pela pasta onde mora: "zone" para os de
// zona (filelist/zone/...), vazio para os globais.
func filelistGroup(path string) string {
	if strings.EqualFold(filepath.Base(filepath.Dir(path)), "zone") {
		return common.GroupZone
	}
	return ""
}

// ListFilelistDir devolve os FILHOS IMEDIATOS de um diretório de um índice.
//
// root pode ser "indice.bin" (índice em disco), "indice.bin#zone/filelist_x"
// (índice EMBUTIDO dentro de um white bin) ou "zone:z000" (a raiz virtual de
// uma zona): nesse caso o índice embutido é extraído do container e lido, e
// dir é o caminho DENTRO dele.
//
// O que sai daqui é DADO CRU: um nível da árvore, com o caminho de cada nó
// (para pedir os filhos), o tamanho descomprimido e o kind (dir/file/index).
// Diretórios com filho único já vêm FUNDIDOS (MergeChain), como no Pulse.
func (FF13Service) ListFilelistDir(part, root, dir string) ([]common.FilelistNode, error) {
	p := parts.Get(part)
	if zone, ok := zoneRootOf(root); ok {
		disc := discoverPart(p)
		return zoneNodes(p, disc, zone, dir), nil
	}
	idx, err := embeddedRoot(p, root)
	if err != nil {
		return nil, err
	}
	entries := indexEntries(p, idx, nil)
	checkSource(idx, common.SourceOf(indexNameOf(idx)), entries)
	return listDirNodes(p, buildIndexTree(entries), dir, zoneOfIndex(idx)), nil
}

// ListFilelistZone devolve a UNIÃO do conteúdo de uma zona: o índice de zona
// (u+c, embutido) e as entradas "zone/<z>/..." dos demais índices da part
// (ex.: o índice de scripts tem "zone/z000/scr000.clb"). O prefixo
// "zone/<z>/" é removido e o dedupe é pelo caminho em minúsculas (o índice de
// zona tem precedência).
func (FF13Service) ListFilelistZone(part, zone string) ([]common.FilelistNode, error) {
	p := parts.Get(part)
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone == "" {
		return nil, fmt.Errorf("filelist: zona vazia")
	}
	disc := discoverPart(p)
	return zoneNodes(p, disc, zone, ""), nil
}

// indexNameOf devolve o nome do arquivo do índice (o embutido usa o caminho
// virtual dentro do container do pai).
func indexNameOf(idx *filelistIndex) string {
	if idx.Virtual != "" {
		return filepath.Base(filepath.ToSlash(idx.Virtual))
	}
	return filepath.Base(filepath.ToSlash(idx.Path))
}

// baseName devolve o último segmento de um caminho indexado (aceita "/" e
// "\" como separadores, sem mexer no casing).
func baseName(path string) string {
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// pairOnDisk devolve o white bin irmão (mesma chave de par) numa pasta já
// conhecida, ou "" quando ele não existe no disco.
func pairOnDisk(p parts.Part, filelistPath string) string {
	key, ok := p.PairKey(filepath.Base(filelistPath))
	if !ok {
		return ""
	}
	entries, err := os.ReadDir(filepath.Dir(filelistPath))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		k, ok := p.PairKey(e.Name())
		if ok && k == key && isWhiteBinName(e.Name()) {
			return filepath.Join(filepath.Dir(filelistPath), e.Name())
		}
	}
	return ""
}

// isFilelistName aceita so os indices de verdade: `filelist...bin` (nao
// `.bin2`, nao `.json` e nao `filelist.info.bin`).
func isFilelistName(name string) bool {
	n := strings.ToLower(name)
	return strings.HasPrefix(n, "filelist") &&
		strings.HasSuffix(n, ".bin") &&
		!strings.HasSuffix(n, ".bin2") &&
		!strings.Contains(n, ".info.")
}

func isWhiteBinName(name string) bool {
	n := strings.ToLower(name)
	return strings.HasPrefix(n, "white") && strings.HasSuffix(n, ".bin") && !strings.HasSuffix(n, ".bin2")
}

// scanPartFiles varre a pasta de dados da part e devolve os arquivos em disco
// indexados pela CHAVE do par: filelists (filelist*.bin) e white bins.
func scanPartFiles(p parts.Part) (filelists map[string]string, bins map[string]string) {
	filelists = map[string]string{}
	bins = map[string]string{}
	root := gameRoot(p.ID())
	if root == "" {
		return filelists, bins
	}
	base := p.DataDir(root)
	if base == "" {
		return filelists, bins
	}
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".bin") || strings.HasSuffix(name, ".bin2") {
			return nil
		}
		key, ok := p.PairKey(name)
		if !ok {
			return nil
		}
		if isFilelistName(name) {
			filelists[key] = path
			return nil
		}
		if isWhiteBinName(name) {
			// `white_zXXXu_img2.win32.bin` (2º arquivo) colapsa na MESMA
			// chave do bin principal pelo PairKey; o bin que o índice
			// descreve é o `..._img.win32.bin`. Não deixar o `_img2`
			// sobrescrever o principal (senão a extração lê o bin errado).
			if _, exists := bins[key]; !exists || !strings.Contains(name, "_img2") {
				bins[key] = path
			}
		}
		return nil
	})
	return filelists, bins
}

// checkFilelistPairs varre os arquivos da part e registra como ERRO todo
// white bin que não tem filelist (nem no disco, nem embutido em outro índice)
// e todo filelist que não tem white bin par.
//
// A cadeia de índices do FF13 é: filelistu (disco) -> zone/filelist_z000u
// (embutido no white_imgu) -> ... Cada índice embutido é extraído do container
// e lido para descobrir o próximo nível.
func checkFilelistPairs(p parts.Part, indexes []*filelistIndex) {
	filelists, bins := scanPartFiles(p)
	if len(filelists) == 0 && len(bins) == 0 {
		return
	}

	// Chaves cobertas por algum índice: em disco + embutidos (descobertos).
	keys := map[string]bool{}
	for key := range filelists {
		keys[key] = true
	}
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		if key, ok := p.PairKey(filepath.Base(idx.Virtual)); ok {
			keys[key] = true
		}
	}

	for key, bin := range bins {
		if !keys[key] {
			logging.Error("filelist: white bin sem par (nao ha filelist no disco nem indice embutido): %s", bin)
		}
	}
	for key, path := range filelists {
		if _, ok := bins[key]; !ok {
			logging.Error("filelist: indice sem white bin par: %s", path)
		}
	}
}
func loadFilelist(path, part string) (*fl.Collection, error) {
	clean := filepath.Clean(path)
	filelistMu.Lock()
	defer filelistMu.Unlock()
	if c, ok := filelistCache[clean]; ok {
		return c, nil
	}
	raw, err := os.ReadFile(clean)
	if err != nil {
		return nil, err
	}
	c, err := fl.Parse(raw, parts.Get(part).GameCode())
	if err != nil {
		return nil, err
	}
	filelistCache[clean] = c
	return c, nil
}
