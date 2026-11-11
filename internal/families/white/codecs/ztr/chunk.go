package ztr

import (
	"encoding/binary"
	"fmt"
)

// pageTable é o "dictionary chunk" do formato: [u32 dictSize][triplos
// (pageIdx, a, b)][strings]. Um byte das strings que tem entrada na tabela é
// um PONTEIRO (expande para a concatenação resolvida de a e b, que também
// podem ser ponteiros — sempre para páginas já definidas); caso contrário é
// literal.
type pageTable struct {
	resolved map[byte][]byte // pageIdx -> bytes expandidos
	pages    int
}

// parseChunk interpreta um chunk a partir de data[pos:] e devolve a tabela,
// o offset das strings e o fim do chunk (posição pós-strings; para o
// unpacker de linhas o fim é dado pelos offsets do header).
func parseChunk(data []byte, pos int) (*pageTable, int, error) {
	if pos+4 > len(data) {
		return nil, 0, fmt.Errorf("ztr: chunk truncado em %d", pos)
	}
	dictSize := int(binary.BigEndian.Uint32(data[pos : pos+4]))
	if dictSize%3 != 0 {
		return nil, 0, fmt.Errorf("ztr: dict size %d não divisível por 3", dictSize)
	}
	base := pos + 4
	if base+dictSize > len(data) {
		return nil, 0, fmt.Errorf("ztr: dict size %d excede o chunk", dictSize)
	}
	n := dictSize / 3
	pt := &pageTable{resolved: make(map[byte][]byte, n), pages: n}
	type triple struct{ idx, a, b byte }
	triples := make([]triple, 0, n)
	seen := make(map[byte]bool, n)
	for i := 0; i < n; i++ {
		t := triple{data[base+3*i], data[base+3*i+1], data[base+3*i+2]}
		if seen[t.idx] {
			return nil, 0, fmt.Errorf("ztr: índice de página duplicado %d", t.idx)
		}
		seen[t.idx] = true
		triples = append(triples, t)
	}
	for _, t := range triples {
		// Slice SEMPRE novo: dar append num valor do map poderia escrever
		// no backing array de outra entrada e corromper expansões.
		a := pt.resolve(t.a)
		b := pt.resolve(t.b)
		nv := make([]byte, 0, len(a)+len(b))
		nv = append(nv, a...)
		nv = append(nv, b...)
		pt.resolved[t.idx] = nv
	}
	return pt, base + dictSize, nil
}

// resolve devolve a expansão de um item; item literal devolve o próprio byte.
func (pt *pageTable) resolve(v byte) []byte {
	if exp, ok := pt.resolved[v]; ok {
		return exp
	}
	return []byte{v}
}

// expand devolve a expansão do byte b ou nil se for literal.
func (pt *pageTable) expand(b byte) []byte {
	return pt.resolved[b]
}

// splitGroups divide size em grupos de até maxChunkData bytes — a mesma
// regra do DictionaryHelpers.GetItemsGroupCount (e do split_groups do
// codec de referência).
func splitGroups(size int) []int {
	var groups []int
	for left := size; left > 0; {
		g := left
		if g > maxChunkData {
			g = maxChunkData
		}
		groups = append(groups, g)
		left -= g
	}
	return groups
}

// pageIndexSkip é o byte 8: a implementação de referência nunca o usa como
// índice de página.
const pageIndexSkip = 8

// freePageNumbers devolve os índices 0..255 ausentes dos dados, exceto o 8 —
// apenas eles podem ser usados como ponteiros sem tornar o parse ambíguo.
func freePageNumbers(data []byte) []byte {
	present := [256]bool{}
	for _, b := range data {
		present[b] = true
	}
	var out []byte
	for i := 0; i < 256; i++ {
		if i != pageIndexSkip && !present[i] {
			out = append(out, byte(i))
		}
	}
	return out
}

// TieRule escolhe o desempate entre pares com a mesma contagem no
// CompressChunk. Empiricamente os arquivos txtres originais do jogo seguem
// TieMinLast (gerador original), mas há arquivos — ex.: auto_yus_us —
// comprimidos com a regra TieFirst de referência.
type TieRule int

const (
	// TieFirst: a PRIMEIRA ocorrência (na ordem dos dados) com a contagem
	// máxima vence — comportamento do modo -c2 de referência.
	TieFirst TieRule = iota
	// TieMinLast: maior contagem; empate resolvido pelo MENOR offset da
	// última ocorrência do par — o do gerador original (validado byte a
	// byte nos txtres originais).
	TieMinLast
)

// largestOccurringBytes porta o GetLargestOccuringBytes: para cada par (na
// sua PRIMEIRA ocorrência) conta as ocorrências não sobrepostas seguintes a
// partir de i+2; o desempate segue a TieRule.
func largestOccurringBytes(data []byte, tie TieRule) (b1, b2 byte, count int) {
	n := len(data)
	if n < 2 {
		return 0, 0, 0
	}
	type occInfo struct {
		positions []int
	}
	occ := make(map[uint16]*occInfo)
	order := make([]uint16, 0, n)
	for i := 0; i < n-1; i++ {
		k := uint16(data[i])<<8 | uint16(data[i+1])
		if o, ok := occ[k]; ok {
			o.positions = append(o.positions, i)
		} else {
			occ[k] = &occInfo{positions: []int{i}}
			order = append(order, k)
		}
	}

	var bestK uint16
	best := 0
	bestLast := 1 << 30
	for _, k := range order {
		o := occ[k]
		num := 1
		next := o.positions[0] + 2
		for _, p := range o.positions[1:] {
			if p >= next {
				num++
				next = p + 2
			}
		}
		last := o.positions[len(o.positions)-1]
		switch tie {
		case TieFirst:
			if num > best {
				best = num
				bestK = k
			}
		default: // TieMinLast: maior contagem; empate -> menor última ocorrência
			if num > best || (num == best && last < bestLast) {
				best = num
				bestK = k
				bestLast = last
			}
		}
	}
	if best == 0 {
		return 0, 0, 0
	}
	return byte(bestK >> 8), byte(bestK), best
}

// compressChunk porta o CompressionHelpers.CompressChunk: repetidamente
// encontra o par mais frequente (desempate pela TieRule), atribui-lhe uma
// página livre e substitui as ocorrências, até que o par mais frequente
// apareça menos de 4 vezes ou faltem páginas. Saída:
// [u32 3*npages][triplos][dados restantes].
func compressChunk(data []byte, tie TieRule) []byte {
	pages := freePageNumbers(data)
	var triples []byte
	cur := data
	nPages := 0
	for {
		b1, b2, cnt := largestOccurringBytes(cur, tie)
		if cnt < 4 || nPages == len(pages) {
			break
		}
		item := pages[nPages]
		triples = append(triples, item, b1, b2)
		out := make([]byte, 0, len(cur))
		for i := 0; i < len(cur); i++ {
			if i == len(cur)-1 {
				out = append(out, cur[i])
			} else if cur[i] == b1 && cur[i+1] == b2 {
				out = append(out, item)
				i++
			} else {
				out = append(out, cur[i])
			}
		}
		cur = out
		nPages++
	}
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(triples)))
	out := make([]byte, 0, 4+len(triples)+len(cur))
	out = append(out, size[:]...)
	out = append(out, triples...)
	out = append(out, cur...)
	return out
}
