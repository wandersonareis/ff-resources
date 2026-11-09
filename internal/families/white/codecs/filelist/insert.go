package filelist

import (
	"fmt"
	"sort"
)

// O formato filelist não permite adicionar nem remover entradas: só é
// possível reapontar o endereço de um arquivo já listado. Além disso, o mesmo
// caminho pode aparecer em várias entradas (com endereços diferentes). Ao
// inserir/substituir um arquivo, TODAS as ocorrências daquele caminho devem
// ser reapontadas para o mesmo novo endereço.

// EntryPath devolve o caminho (4º campo) de um FileInfo. Se a string estiver
// malformada, devolve a própria string.
func EntryPath(fileInfo string) string {
	loc, err := ParseFileInfo(fileInfo)
	if err != nil {
		return fileInfo
	}
	return loc.Path
}

// FindByPath devolve todas as referências cujo caminho é igual a path.
func (c *Collection) FindByPath(path string) []EntryRef {
	var refs []EntryRef
	for ci := range c.Chunks {
		for ei := range c.Chunks[ci].Entries {
			if EntryPath(c.Chunks[ci].Entries[ei].FileInfo) == path {
				refs = append(refs, EntryRef{ci, ei})
			}
		}
	}
	return refs
}

// SetAddressByPath atualiza posição/tamanhos de todas as entradas com o mesmo
// caminho (as "repetições"), preservando o caminho. É o ponto que garante que
// as repetições sejam alteradas da mesma forma. Retorna quantas entradas
// foram atualizadas.
func (c *Collection) SetAddressByPath(path string, posSectors uint64, uncmpSize, cmpSize uint32) (int, error) {
	refs := c.FindByPath(path)
	if len(refs) == 0 {
		return 0, fmt.Errorf("caminho não encontrado no filelist: %s", path)
	}
	for _, ref := range refs {
		e := &c.Chunks[ref.Chunk].Entries[ref.Entry]
		loc, err := ParseFileInfo(e.FileInfo)
		if err != nil {
			return 0, fmt.Errorf("entrada com fileInfo inválido (%q): %w", e.FileInfo, err)
		}
		loc.PosSectors = posSectors
		loc.UncmpSize = uncmpSize
		loc.CmpSize = cmpSize
		e.FileInfo = FormatFileInfo(loc)
	}
	return len(refs), nil
}

// DuplicatedPaths devolve os caminhos que aparecem em mais de uma entrada,
// com a contagem de ocorrências. Útil para certificar que a inserção cobriu
// todas as repetições.
func (c *Collection) DuplicatedPaths() map[string]int {
	counts := make(map[string]int)
	for ci := range c.Chunks {
		for ei := range c.Chunks[ci].Entries {
			counts[EntryPath(c.Chunks[ci].Entries[ei].FileInfo)]++
		}
	}
	for p, n := range counts {
		if n <= 1 {
			delete(counts, p)
		}
	}
	return counts
}

// Paths devolve todos os caminhos, ordenados.
func (c *Collection) Paths() []string {
	seen := make(map[string]struct{})
	for ci := range c.Chunks {
		for ei := range c.Chunks[ci].Entries {
			seen[EntryPath(c.Chunks[ci].Entries[ei].FileInfo)] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// AlignUp2048 arredonda um offset para o próximo múltiplo de 2048.
func AlignUp2048(offset int64) int64 {
	if offset%2048 == 0 {
		return offset
	}
	return offset + (2048 - offset%2048)
}

// PlanAppend calcula onde um arquivo inserido no fim do container deve ficar:
// o offset real (alinhado a 2048) e a posição em setores gravada no filelist.
func PlanAppend(containerLen int64) (alignedOffset int64, posSectors uint64) {
	alignedOffset = AlignUp2048(containerLen)
	return alignedOffset, uint64(alignedOffset / 2048)
}

// InsertResult descreve o que uma inserção fez.
type InsertResult struct {
	// AlignedOffset é o offset (em bytes) onde o arquivo deve ser escrito.
	AlignedOffset int64
	// PosSectors é o valor de posição (setores) gravado no filelist.
	PosSectors uint64
	// UpdatedEntries é quantas entradas foram reapontadas (inclui repetições).
	UpdatedEntries int
	// DuplicateOccurrences é o número de entradas encontradas para o caminho.
	DuplicateOccurrences int
}

// AppendFile implementa a inserção do WhiteBinTools: o arquivo novo vai para o
// fim do container (alinhado a 2048) e todas as ocorrências do mesmo caminho
// no filelist passam a apontar para esse endereço.
//
// uncmpSize é o tamanho descomprimido; cmpSize é o tamanho efetivamente
// armazenado (igual a uncmpSize quando não comprimido).
func (c *Collection) AppendFile(path string, containerLen int64, uncmpSize, cmpSize uint32) (InsertResult, error) {
	aligned, posSectors := PlanAppend(containerLen)
	refs := c.FindByPath(path)
	if len(refs) == 0 {
		return InsertResult{}, fmt.Errorf("caminho não encontrado no filelist: %s", path)
	}
	updated, err := c.SetAddressByPath(path, posSectors, uncmpSize, cmpSize)
	if err != nil {
		return InsertResult{}, err
	}
	return InsertResult{
		AlignedOffset:        aligned,
		PosSectors:           posSectors,
		UpdatedEntries:       updated,
		DuplicateOccurrences: len(refs),
	}, nil
}
