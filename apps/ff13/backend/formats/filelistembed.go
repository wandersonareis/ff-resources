package formats

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ffresources/apps/ff13/backend/common"
	"ffresources/apps/ff13/backend/config"
	"ffresources/apps/ff13/backend/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
	logging "ffresources/internal/shared/logging"
)

// FilelistIndex é um índice (filelist) conhecido: em disco ou EMBUTIDO dentro
// de um white bin.
//
// A cadeia real do FF13 é:
//
//	white_imgu  <-- filelistu (índice em disco, no sys/)
//	white_z000u_img <-- filelist_z000u (índice EMBUTIDO no white_imgu)
//	white_z104u_img <-- filelist_z104u (índice embutido no white_z000u_img?)
//
// Ou seja: o índice de um white bin vive DENTRO de outro white bin. Para ler
// um índice embutido é preciso extrair o arquivo do container (whitebin) e
// parsear o resultado como filelist.
type FilelistIndex struct {
	// Path: caminho absoluto do índice em disco (nível 0) ou
	// "pai#caminho/virtual" para os embutidos.
	Path string
	// Parent: índice que contém este (vazio nos de disco).
	Parent string
	// Virtual: caminho virtual do índice dentro do container do pai.
	Virtual string
	// Container: white bin onde o ARQUIVO do índice mora.
	Container string
	// Level: 0 = índice em disco, 1+ = índice embutido.
	Level int
	// Coll: a coleção já parseada.
	Coll *fl.Collection
}

var (
	embeddedMu    sync.Mutex
	embeddedCache = map[string]*fl.Collection{}
)

// ReadEmbeddedIndex extrai o índice embutido (virtual) do container do pai e
// o parseia. O cache evita reextrair o mesmo índice a cada consulta.
func ReadEmbeddedIndex(parent *FilelistIndex, virtual string, game fl.GameCode) (*fl.Collection, error) {
	if parent == nil || parent.Coll == nil {
		return nil, fmt.Errorf("indice pai invalido")
	}
	if parent.Container == "" {
		return nil, fmt.Errorf("container desconhecido para %s", parent.Path)
	}
	key := parent.Path + "#" + virtual
	embeddedMu.Lock()
	if c, ok := embeddedCache[key]; ok {
		embeddedMu.Unlock()
		return c, nil
	}
	embeddedMu.Unlock()

	tmp, err := os.MkdirTemp("", "ff13-filelist")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	results, err := whitebin.ExtractFile(parent.Container, tmp, virtual, parent.Coll)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("indice embutido nao encontrado: %s", virtual)
	}
	raw, err := os.ReadFile(results[0].OutFile)
	if err != nil {
		return nil, err
	}
	coll, err := fl.Parse(raw, game)
	if err != nil {
		return nil, fmt.Errorf("parse do indice embutido %s: %w", virtual, err)
	}
	embeddedMu.Lock()
	embeddedCache[key] = coll
	embeddedMu.Unlock()
	return coll, nil
}

// DiscoverFilelists varre os índices em disco e, dentro deles, procura OUTROS
// filelists (entradas cujo nome é filelist*.bin). Cada um desses é extraído do
// white bin que o contém e lido — é assim que a cadeia de índices do FF13 é
// descoberta. bins mapeia a chave de par -> white bin em disco.
//
// Devolve todos os índices encontrados (nível 0 e embutidos).
func DiscoverFilelists(p parts.Part, roots []common.FilelistRoot, bins map[string]string) []*FilelistIndex {
	queue := make([]*FilelistIndex, 0, len(roots))
	out := make([]*FilelistIndex, 0, len(roots))
	seen := map[string]bool{}
	for _, r := range roots {
		// Sem white bin par não há de onde extrair índice embutido (é o
		// caso da cópia do índice em data/partN/filelist).
		if r.WhiteBin == "" {
			continue
		}
		coll, err := LoadFilelist(r.Path, p.ID())
		if err != nil {
			continue
		}
		idx := &FilelistIndex{Path: r.Path, Container: r.WhiteBin, Level: 0, Coll: coll}
		queue = append(queue, idx)
		out = append(out, idx)
		seen[r.Path] = true
	}

	// Limite de profundidade: a cadeia é curta (índice global -> índice de
	// zona -> índice do dlc), mas um índice malformado não pode gerar laço.
	const maxLevel = 4
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.Level >= maxLevel {
			continue
		}
		for _, ep := range cur.Coll.Paths() {
			base := filepath.Base(ep)
			if !IsFilelistName(base) {
				continue
			}
			key, ok := p.PairKey(base)
			if !ok {
				continue
			}
			container := bins[key]
			if container == "" {
				// Sem white bin par em disco: nada a extrair deste índice.
				continue
			}
			id := cur.Path + "#" + ep
			if seen[id] {
				continue
			}
			coll, err := ReadEmbeddedIndex(cur, ep, p.GameCode())
			if err != nil {
				logging.Warn("filelist: indice embutido ilegivel (%s em %s): %v", ep, cur.Path, err)
				continue
			}
			seen[id] = true
			child := &FilelistIndex{
				Path:      id,
				Parent:    cur.Path,
				Virtual:   ep,
				Container: container,
				Level:     cur.Level + 1,
				Coll:      coll,
			}
			queue = append(queue, child)
			out = append(out, child)
		}
	}
	return out
}

// EmbeddedRoot resolve a raiz pedida pela UI: "caminho.do.indice" (índice em
// disco) ou "pai#caminho/virtual" (índice embutido). O pai pode ser, ele
// mesmo, um índice embutido — por isso o corte é no ÚLTIMO '#'.
func EmbeddedRoot(p parts.Part, root string) (*FilelistIndex, error) {
	i := strings.LastIndex(root, "#")
	if i < 0 {
		coll, err := LoadFilelist(root, p.ID())
		if err != nil {
			return nil, err
		}
		return &FilelistIndex{Path: root, Container: PairOnDisk(p, root), Level: 0, Coll: coll}, nil
	}
	parentPath, virtual := root[:i], root[i+1:]
	parentIndex, err := EmbeddedRoot(p, parentPath)
	if err != nil {
		return nil, err
	}
	embedded, err := ReadEmbeddedIndex(parentIndex, virtual, p.GameCode())
	if err != nil {
		return nil, err
	}
	// Container do índice embutido = white bin que ele INDEXA (não onde o
	// arquivo dele mora: esse é o container do pai).
	container := ""
	if key, ok := p.PairKey(filepath.Base(virtual)); ok {
		container = PartBins(p)[key]
	}
	return &FilelistIndex{
		Path:      root,
		Parent:    parentPath,
		Virtual:   virtual,
		Container: container,
		Level:     parentIndex.Level + 1,
		Coll:      embedded,
	}, nil
}

// PartBins devolve o mapa chave -> white bin da part, com cache por processo
// (os arquivos do jogo não mudam durante a sessão).
func PartBins(p parts.Part) map[string]string {
	key := p.ID() + "|" + config.GameRoot(p.ID())
	binsMu.Lock()
	defer binsMu.Unlock()
	if m, ok := binsCache[key]; ok {
		return m
	}
	_, bins := ScanPartFiles(p)
	binsCache[key] = bins
	return bins
}

var (
	binsMu    sync.Mutex
	binsCache = map[string]map[string]string{}
)
