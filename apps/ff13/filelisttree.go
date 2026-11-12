package main

import (
	"path"
	"sort"
	"strings"
	"sync"

	"ffresources/apps/ff13/common"
	"ffresources/apps/ff13/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
	logging "ffresources/internal/shared/logging"
)

// Camada de ÁRVORE do navegador de filelist: transforma a coleção crua do
// índice (fl.Collection) nos nós do DTO (common.FilelistNode).
//
// O backend só entrega FATOS (caminho, nome, tamanho, kind): a fusão de
// diretórios com filho único (a regra do Pulse), a ordem e os rótulos são
// do frontend.

// indexEntry é uma entrada indexada: o caminho cru (relativo à raiz do
// índice) e o tamanho descomprimido.
type indexEntry struct {
	Path string
	Size uint32
}

// indexEntries percorre a coleção do índice e devolve as entradas ÚNICAS (o
// índice repete o mesmo caminho com endereços diferentes: vale a primeira
// ocorrência), já com a regra da part aplicada (KeepZoneEntry).
//
// keep (quando não é nil) filtra o caminho ANTES do parse do tamanho: a
// união da zona só precisa das entradas "zone/<z>/...".
func indexEntries(p parts.Part, idx *filelistIndex, keep func(path string) bool) []indexEntry {
	coll := idx.Coll
	if coll == nil {
		return nil
	}
	seen := make(map[string]bool, coll.FileCount())
	out := make([]indexEntry, 0, coll.FileCount())
	for ci := range coll.Chunks {
		for ei := range coll.Chunks[ci].Entries {
			info := coll.Chunks[ci].Entries[ei].FileInfo
			entryPath := fl.EntryPath(info)
			if seen[entryPath] {
				continue
			}
			seen[entryPath] = true
			// Regra da part (ex.: part1 descarta "bg/loc" de outra zona).
			if !p.KeepZoneEntry(idx.Virtual, entryPath) {
				continue
			}
			if keep != nil && !keep(entryPath) {
				continue
			}
			var size uint32
			if loc, lerr := fl.ParseFileInfo(info); lerr == nil {
				size = loc.UncmpSize
			}
			out = append(out, indexEntry{Path: entryPath, Size: size})
		}
	}
	return out
}

// treeNode é um nó da árvore de UM índice (diretório ou arquivo).
type treeNode struct {
	// Name é o nome cru do nó (com o casing original).
	Name string
	// Path é o caminho do nó dentro do índice (sempre com "/").
	Path string
	// Size é o tamanho descomprimido dos arquivos.
	Size uint32
	// IsDir marca os diretórios.
	IsDir bool
	// Children são os filhos imediatos, indexados pelo nome em minúsculas.
	Children map[string]*treeNode
}

// buildIndexTree monta a árvore de um índice a partir das entradas cruas.
// O caminho de um diretório é a concatenação dos segmentos com "/".
func buildIndexTree(entries []indexEntry) *treeNode {
	root := &treeNode{IsDir: true, Children: map[string]*treeNode{}}
	for _, e := range entries {
		segments := splitIndexPath(e.Path)
		if len(segments) == 0 {
			continue
		}
		cur := root
		for i := 0; i < len(segments)-1; i++ {
			name := segments[i]
			key := strings.ToLower(name)
			next, ok := cur.Children[key]
			if !ok || !next.IsDir {
				next = &treeNode{
					Name:     name,
					Path:     joinIndexPath(cur.Path, name),
					IsDir:    true,
					Children: map[string]*treeNode{},
				}
				cur.Children[key] = next
			}
			cur = next
		}
		last := segments[len(segments)-1]
		key := strings.ToLower(last)
		if existing, ok := cur.Children[key]; ok {
			// O mesmo nome já existe como diretório: o arquivo some dentro
			// dele (o índice é inconsistente, mas a árvore continua válida).
			if !existing.IsDir {
				existing.Size = e.Size
			}
			continue
		}
		cur.Children[key] = &treeNode{
			Name:  last,
			Path:  joinIndexPath(cur.Path, last),
			Size:  e.Size,
			IsDir: false,
		}
	}
	return root
}

// listDirNodes devolve os filhos imediatos de um diretório do índice (um
// nível). Os diretórios vêm ANTES dos arquivos, cada grupo em ordem alfabética
// (pelo nome em minúsculas).
//
// zone é a zona da raiz ("z000" ou ""): ela define o caminho de DADOS do
// arquivo indexado (o índice de zona guarda o caminho relativo à zona) e,
// com ele, o id da unidade de texto.
//
// p é a part: o idioma do índice embutido ("u"/"c" no part1/2, "a"/"v" no
// part3) é uma convenção de cada jogo.
func listDirNodes(p parts.Part, root *treeNode, dirPath string, zone string, rootRefs ...string) []common.FilelistNode {
	rootRef := ""
	if len(rootRefs) > 0 {
		rootRef = rootRefs[0]
	}
	dir := findDir(root, dirPath)
	if dir == nil {
		return nil
	}
	children := make([]*treeNode, 0, len(dir.Children))
	for _, child := range dir.Children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir != children[j].IsDir {
			return children[i].IsDir
		}
		li, lj := strings.ToLower(children[i].Name), strings.ToLower(children[j].Name)
		if li != lj {
			return li < lj
		}
		return children[i].Name < children[j].Name
	})

	out := make([]common.FilelistNode, 0, len(children))
	for _, child := range children {
		if child.IsDir {
			chain, finalPath := mergeChain(child)
			node := common.FilelistNode{
				Name:  child.Name,
				Path:  finalPath,
				IsDir: true,
				Kind:  common.KindDir,
			}
			if len(chain) > 1 {
				node.MergeChain = chain
			}
			out = append(out, node)
			continue
		}
		kind := common.KindFile
		if isFilelistName(child.Name) {
			kind = common.KindIndex
		}
		node := common.FilelistNode{
			Name:     child.Name,
			Path:     child.Path,
			Size:     child.Size,
			Kind:     kind,
			Language: p.LanguageOf(child.Name),
		}
		if kind == common.KindFile {
			// Arquivo de TEXTO (.ztr solto ou .bin container WPD): a folha
			// abre a unidade de texto (a mesma da árvore /data). A
			// verificação do container (o registro .ztr dentro do WPD)
			// acontece no clique — a listagem não lê nada.
			dataPath := filelistDataPath(zone, child.Path)
			if unitID, lang, ok := filelistUnitID(dataPath); ok {
				node.UnitID = unitID
				node.UnitLang = lang
				node.Text = true
				node.Container = containerOfExt(child.Name)
			}
			// Face do GUI: .imgb sem tabela externa (DXT1 128x256 implícito).
			if isResidentFaceImage(dataPath) {
				if child.Size > 8 {
					node.Image = true
					node.ImageRef = encodeImageRef(imageRef{
						Source:   "filelist",
						Root:     rootRef,
						Path:     child.Path,
						Platform: imagePlatformOf(child.Name),
					})
				}
			} else {
				// Imagem direta (.dds/.png) ou container de imagem. A tabela
				// (.xgr/.xfv/.bin/.trb) é expansível; o .imgb é a metade
				// binária e continua folha comum.
				ext := strings.ToLower(path.Ext(child.Name))
				switch ext {
				case ".dds", ".png":
					node.Image = true
					node.ImageRef = encodeImageRef(imageRef{
						Source:   "filelist",
						Root:     rootRef,
						Path:     child.Path,
						Platform: imagePlatformOf(child.Name),
					})
				case ".xgr", ".xfv", ".bin", ".trb", ".wpd", ".wdb", ".xwb", ".xwp", ".grs":
					node.Image = true
					node.ImageContainer = true
					if ext == ".trb" {
						node.Container = "trb"
					} else {
						node.Container = "wpd"
					}
					node.Pair = imagePairPath(child.Path)
					node.ImageRef = encodeImageRef(imageRef{
						Source:   "filelist",
						Root:     rootRef,
						Path:     child.Path,
						Payload:  imagePairPath(filelistDataPath(zone, child.Path)),
						Platform: imagePlatformOf(child.Name),
					})
				}
			}
		}
		out = append(out, node)
	}
	return out
}

// mergeChain devolve a cadeia de diretórios com FILHO ÚNICO que começa em dir
// e o caminho final da cadeia (o diretório cujos filhos são pedidos na
// expansão).
//
// É a regra do Pulse (AbsorbSingleChildContainer): um diretório que só tem um
// filho diretório vira um só nó ("empty/empty"). A fusão só acontece com
// diretório: um diretório com um único ARQUIVO continua como está.
func mergeChain(dir *treeNode) ([]string, string) {
	chain := []string{dir.Name}
	cur := dir
	for len(cur.Children) == 1 {
		var only *treeNode
		for _, child := range cur.Children {
			only = child
		}
		if only == nil || !only.IsDir {
			break
		}
		cur = only
		chain = append(chain, cur.Name)
	}
	return chain, cur.Path
}

// findDir acha o diretório de um caminho interno do índice ("" = a raiz).
func findDir(root *treeNode, dirPath string) *treeNode {
	segments := splitIndexPath(dirPath)
	cur := root
	for _, segment := range segments {
		next, ok := cur.Children[strings.ToLower(segment)]
		if !ok || !next.IsDir {
			return nil
		}
		cur = next
	}
	return cur
}

// splitIndexPath quebra um caminho interno de índice nos seus segmentos.
func splitIndexPath(p string) []string {
	normalized := strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	out := make([]string, 0, 4)
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." {
			continue
		}
		out = append(out, segment)
	}
	return out
}

// joinIndexPath junta um caminho de diretório com o nome do filho.
func joinIndexPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return path.Join(dir, name)
}

// zoneNodes monta a listagem de UMA ZONA: a união do conteúdo dos índices de
// zona da zona (u+c, que têm precedência) com as entradas "zone/<z>/..." dos
// demais índices da part (ex.: o índice de scripts tem "zone/z000/scr000.clb").
//
// O prefixo "zone/<z>/" é removido: a listagem tem a mesma forma de
// ListFilelistDir (um nível + MergeChain + tamanho). dir é o diretório dentro
// da zona ("" = o topo dela).
func zoneNodes(p parts.Part, disc partDiscovery, zone, dir string) []common.FilelistNode {
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone == "" {
		return nil
	}
	prefix := "zone/" + zone + "/"
	seen := map[string]bool{}
	entries := make([]indexEntry, 0, 1024)

	add := func(entryPath string, size uint32) {
		key := strings.ToLower(entryPath)
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, indexEntry{Path: entryPath, Size: size})
	}

	// 1. Os índices de ZONA da zona: o conteúdo inteiro, com o prefixo
	// "zone/<z>/" removido quando ele existe.
	for _, idx := range disc.indexes {
		if !isZoneIndexOf(idx, zone) {
			continue
		}
		for _, e := range indexEntries(p, idx, nil) {
			entryPath := e.Path
			if len(entryPath) >= len(prefix) && strings.EqualFold(entryPath[:len(prefix)], prefix) {
				entryPath = entryPath[len(prefix):]
			}
			add(entryPath, e.Size)
		}
	}
	// 2. Os demais índices da part: só as entradas "zone/<z>/...".
	for _, idx := range disc.indexes {
		if isZoneIndexOf(idx, zone) {
			continue
		}
		keep := func(entryPath string) bool {
			return strings.HasPrefix(strings.ToLower(entryPath), prefix)
		}
		for _, e := range indexEntries(p, idx, keep) {
			add(e.Path[len(prefix):], e.Size)
		}
	}

	return listDirNodes(p, buildIndexTree(entries), dir, zone, zoneRootPrefix+zone)
}

// zoneRootPrefix marca a raiz virtual de uma ZONA no protocolo do frontend:
// ListFilelistDir(part, "zone:z000", "txtres") lista um diretório da zona.
const zoneRootPrefix = "zone:"

// zoneRootOf devolve a zona de uma raiz virtual ("zone:z000" → "z000").
func zoneRootOf(root string) (string, bool) {
	if !strings.HasPrefix(strings.ToLower(root), zoneRootPrefix) {
		return "", false
	}
	zone := strings.TrimSpace(root[len(zoneRootPrefix):])
	if zone == "" {
		return "", false
	}
	return zone, true
}

// isZoneIndexOf diz se o índice é o índice de ZONA da zona pedida.
func isZoneIndexOf(idx *filelistIndex, zone string) bool {
	name := idx.Virtual
	if name == "" {
		name = idx.Path
	}
	return common.ZoneIDOf(name) == zone
}

// warnedSources guarda os índices que já geraram aviso de plataforma (o
// aviso sai uma vez por índice, não a cada expansão).
var warnedSources sync.Map

// checkSource confere a consistência da PLATAFORMA (D6): a plataforma é uma
// propriedade do índice INTEIRO (tirada do nome dele). Uma entrada com
// extensão de outra plataforma é dado inconsistente: o backend registra um
// aviso e MANTÉM a plataforma do índice.
func checkSource(idx *filelistIndex, source string, entries []indexEntry) {
	if source == "" {
		return
	}
	for _, e := range entries {
		other := common.SourceOf(baseName(e.Path))
		if other == "" || other == source {
			continue
		}
		if _, loaded := warnedSources.LoadOrStore(idx.Path, true); loaded {
			return
		}
		logging.Warn("filelist: indice %s mantem a plataforma %s, mas a entrada %s e de outra plataforma (%s)",
			idx.Path, source, baseName(e.Path), other)
		return
	}
}
