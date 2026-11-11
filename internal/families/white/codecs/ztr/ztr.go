package ztr

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// File é o conteúdo lógico de um .ztr: pares de ID de linha (strings ASCII
// terminadas em \0 no blob de IDs) e linhas cruas (bytes da codepage com
// terminador 00 00 — a camada de texto do core/encoding cuida do resto).
type File struct {
	IDs   []string
	Lines [][]byte
}

// Mode seleciona a forma de empacotamento, espelhando os switches do
// implementação de referência: -c (sem compressão: chunks com dict vazio) e
// -c2 (comprimido).
type Mode int

const (
	Compressed   Mode = iota // -c2: CompressChunk por grupo de 4096 bytes
	Uncompressed             // -c: [u32 0] + dados crus por grupo
)

// Read interpreta um .ztr completo e devolve o conteúdo. O formato é
// detectado pelo primeiro u32 lido em little-endian, como no Pulse:
//
//	0     -> big-endian comprimido (dicionário de chunks)
//	1     -> little-endian par não comprimido
//	> 1   -> little-endian dicionário não comprimido
//
// Um .ztr VAZIO (o jogo tem alguns: só o header, sem linhas — o conteúdo
// daquele idioma não existe) não é erro: devolve um File sem IDs e sem
// linhas. É o que a árvore precisa para abrir o arquivo sem reclamar.
func Read(data []byte) (*File, error) {
	if len(data) >= 4 {
		typ := binary.LittleEndian.Uint32(data[:4])
		switch {
		case typ == 0:
			return readBigEndian(data)
		case typ == 1:
			return readLittleEndianPair(data)
		default:
			return readLittleEndianDictionary(data, int(typ))
		}
	}
	return readBigEndian(data)
}

// readBigEndian é o parser do formato original do jogo (magic 1 em
// big-endian). Fica separado para o dispatch de Read.
func readBigEndian(data []byte) (*File, error) {
	h, idsPos, err := parseHeader(data)
	if err != nil {
		if isEmptyFile(data) {
			return &File{}, nil
		}
		return nil, err
	}

	// --- chunks de IDs (consecutivos, sem tabela de offsets) ---
	idsBlob := make([]byte, 0, h.DcmpIDsSize)
	pos := idsPos
	for _, g := range splitGroups(int(h.DcmpIDsSize)) {
		pt, sp, err := parseChunk(data, pos)
		if err != nil {
			return nil, err
		}
		d := 0
		for d < g {
			if sp >= len(data) {
				return nil, fmt.Errorf("ztr: id chunk truncado")
			}
			b := data[sp]
			sp++
			if exp := pt.expand(b); exp != nil {
				idsBlob = append(idsBlob, exp...)
				d += len(exp)
			} else {
				idsBlob = append(idsBlob, b)
				d++
			}
		}
		if d != g {
			return nil, fmt.Errorf("ztr: id group %d bytes, esperado %d", d, g)
		}
		pos = sp
	}

	// --- chunks de linha (offsets relativos à região) ---
	regionPos := pos
	var blob []byte
	bases := make([]int, 0, len(h.ChunkOffsets)-1)
	maps := make([][]int, 0, len(h.ChunkOffsets)-1)
	for i := 0; i+1 < len(h.ChunkOffsets); i++ {
		start := regionPos + int(h.ChunkOffsets[i])
		end := regionPos + int(h.ChunkOffsets[i+1])
		if end > len(data) || start > end {
			return nil, fmt.Errorf("ztr: chunk %d fora do arquivo", i)
		}
		pt, sp, err := parseChunk(data, start)
		if err != nil {
			return nil, err
		}
		base := len(blob)
		m := make([]int, 0, end-sp)
		for c := sp; c < end; c++ {
			m = append(m, len(blob)-base)
			if exp := pt.expand(data[c]); exp != nil {
				blob = append(blob, exp...)
			} else {
				blob = append(blob, data[c])
			}
		}
		bases = append(bases, base)
		maps = append(maps, m)
	}

	// --- início de cada linha via line info ---
	starts := make([]int, 0, len(h.LineInfos))
	for i, li := range h.LineInfos {
		cid := int(li.ChunkID)
		if cid >= len(maps) {
			return nil, fmt.Errorf("ztr: line info %d aponta para chunk inexistente %d", i, cid)
		}
		m := maps[cid]
		if int(li.Pos) >= len(m) {
			return nil, fmt.Errorf("ztr: line info %d pos %d fora do chunk %d", i, li.Pos, cid)
		}
		s := bases[cid] + m[li.Pos] + int(li.CharaStart)
		starts = append(starts, s)
	}

	// --- separação das linhas ---
	lines := make([][]byte, 0, len(starts))
	for i, s := range starts {
		e := len(blob)
		if i+1 < len(starts) {
			e = starts[i+1]
		} else {
			t := bytes.Index(blob[s:], []byte{0x00, 0x00})
			if t < 0 {
				return nil, fmt.Errorf("ztr: última linha sem terminador")
			}
			e = s + t + 2
			if e != len(blob) {
				return nil, fmt.Errorf("ztr: sobram %d bytes depois da última linha", len(blob)-e)
			}
		}
		line := blob[s:e]
		if !bytes.HasSuffix(line, []byte{0x00, 0x00}) {
			return nil, fmt.Errorf("ztr: linha %d não termina com 00 00", i)
		}
		lines = append(lines, line)
	}
	if len(lines) != int(h.LineCount) {
		return nil, fmt.Errorf("ztr: %d linhas decodificadas, header diz %d", len(lines), h.LineCount)
	}

	// --- IDs: blob terminado por \0, split ---
	var outIDs []string
	if len(idsBlob) > 0 {
		parts := bytes.Split(bytes.TrimSuffix(idsBlob, []byte{0}), []byte{0})
		outIDs = make([]string, len(parts))
		for i, id := range parts {
			outIDs[i] = string(id)
		}
	}
	return &File{IDs: outIDs, Lines: lines}, nil
}

// Write monta um .ztr completo no modo pedido, usando o desempate do
// gerador original (TieMinLast) — veja WriteTie para escolher a regra.
func Write(f *File, mode Mode) ([]byte, error) {
	return WriteTie(f, mode, TieMinLast)
}

// WriteTie monta um .ztr completo no modo e regra de desempate pedidos.
//
// O blob de linhas são os bytes crus concatenados (cada linha já com o
// terminador 00 00); o line info é calculado conforme o modo:
//
//   - Uncompressed: fiel ao PackUncmp — posições cruas nos chunks de 4096
//     bytes, CharaStart sempre 0 (incluindo o corner em que uma linha
//     termina exatamente no limite de 4096: o line info da linha seguinte
//     fica (chunk atual, 4096), como no C#);
//   - Compressed: fiel ao PackCmp — chunks comprimidos e line info
//     resolvido por busca binária nas expansões.
func WriteTie(f *File, mode Mode, tie TieRule) ([]byte, error) {
	if len(f.IDs) != len(f.Lines) {
		return nil, fmt.Errorf("ztr: %d IDs vs %d linhas", len(f.IDs), len(f.Lines))
	}
	if len(f.Lines) == 0 {
		return nil, fmt.Errorf("ztr: arquivo sem linhas")
	}

	// blobs
	var idsBlob []byte
	for _, id := range f.IDs {
		idsBlob = append(idsBlob, id...)
		idsBlob = append(idsBlob, 0)
	}
	linesBlob := bytes.Join(f.Lines, nil)

	// chunks de IDs
	var idsChunks [][]byte
	off := 0
	for _, g := range splitGroups(len(idsBlob)) {
		switch mode {
		case Uncompressed:
			idsChunks = append(idsChunks, append([]byte{0, 0, 0, 0}, idsBlob[off:off+g]...))
		default:
			idsChunks = append(idsChunks, compressChunk(idsBlob[off:off+g], tie))
		}
		off += g
	}

	// chunks de linhas + line info
	var lineChunks [][]byte
	var lineInfos []LineInfo
	switch mode {
	case Uncompressed:
		lineChunks, lineInfos = packUncompressed(linesBlob, len(f.Lines))
	default:
		var err error
		lineChunks, lineInfos, err = packCompressed(linesBlob, len(f.Lines), tie)
		if err != nil {
			return nil, err
		}
	}

	// offsets relativos à região de chunks de linha
	offsets := make([]uint32, 0, len(lineChunks)+1)
	var acc uint32
	offsets = append(offsets, 0)
	for _, ch := range lineChunks {
		acc += uint32(len(ch))
		offsets = append(offsets, acc)
	}

	h := &Header{
		Magic:        Magic,
		LineCount:    uint32(len(f.Lines)),
		DcmpIDsSize:  uint32(len(idsBlob)),
		ChunkOffsets: offsets,
		LineInfos:    lineInfos,
	}
	out := appendHeader(nil, h)
	for _, ch := range idsChunks {
		out = append(out, ch...)
	}
	for _, ch := range lineChunks {
		out = append(out, ch...)
	}
	return out, nil
}

// packUncompressed reproduz o PackUncmp: chunks [u32 0] + 4096 bytes e
// line info com posições cruas (CharaStart 0). O header do próximo chunk é
// escrito no evento de corte mesmo que o fluxo acabe ali (chunk vazio de 4
// bytes, como no C#), e a posição da linha seguinte é avaliada ANTES do
// corte — fiel inclusive ao corner de linha terminando exatamente no
// limite de 4096.
func packUncompressed(linesBlob []byte, lineCount int) ([][]byte, []LineInfo) {
	var chunks [][]byte
	infos := make([]LineInfo, lineCount)
	infos[0] = LineInfo{} // (0, 0, 0)

	cur := []byte{0, 0, 0, 0} // header do chunk atual (escrito antes do loop no C#)
	chunkID, copyCounter, linesWritten := 0, 0, 0
	prevByte := byte(0xFF)

	for _, b := range linesBlob {
		copyCounter++
		cur = append(cur, b)
		if b == 0 && prevByte == 0 {
			linesWritten++
			if linesWritten < lineCount {
				infos[linesWritten] = LineInfo{
					ChunkID: byte(chunkID),
					Pos:     uint16(copyCounter),
				}
				prevByte = 0xFF
			}
		} else {
			prevByte = b
		}
		if copyCounter == maxChunkData {
			copyCounter = 0
			chunkID++
			chunks = append(chunks, cur)
			cur = []byte{0, 0, 0, 0} // header do próximo (pode ficar vazio)
		}
	}
	chunks = append(chunks, cur)
	return chunks, infos
}

// packCompressed reproduz o PackCmp: chunks comprimidos + line info
// resolvido nas expansões (inverse exato do unpack), como o build_lineinfo
// do codec de referência.
func packCompressed(linesBlob []byte, lineCount int, tie TieRule) ([][]byte, []LineInfo, error) {
	var chunks [][]byte
	off := 0
	for _, g := range splitGroups(len(linesBlob)) {
		chunks = append(chunks, compressChunk(linesBlob[off:off+g], tie))
		off += g
	}

	bases := make([]int, len(chunks))
	maps := make([][]int, len(chunks))
	sizes := make([]int, len(chunks))
	total := 0
	for i, ch := range chunks {
		pt, sp, err := parseChunk(ch, 0)
		if err != nil {
			return nil, nil, err
		}
		m := make([]int, 0, len(ch)-sp)
		d := 0
		for c := sp; c < len(ch); c++ {
			m = append(m, d)
			if exp := pt.expand(ch[c]); exp != nil {
				d += len(exp)
			} else {
				d++
			}
		}
		bases[i], maps[i], sizes[i] = total, m, d
		total += d
	}
	if total != len(linesBlob) {
		return nil, nil, fmt.Errorf("ztr: chunks cobrem %d bytes, blob tem %d", total, len(linesBlob))
	}

	starts := make([]int, 0, lineCount)
	pos := 0
	for pos < len(linesBlob) {
		starts = append(starts, pos)
		t := bytes.Index(linesBlob[pos:], []byte{0x00, 0x00})
		if t < 0 {
			return nil, nil, fmt.Errorf("ztr: linha sem terminador em %d", pos)
		}
		pos += t + 2
	}
	if len(starts) != lineCount {
		return nil, nil, fmt.Errorf("ztr: %d linhas no blob, esperado %d", len(starts), lineCount)
	}

	infos := make([]LineInfo, 0, lineCount)
	for _, s := range starts {
		cid := bisectRight(bases, s) - 1
		if cid < 0 {
			return nil, nil, fmt.Errorf("ztr: sem chunk para o offset %d", s)
		}
		rel := s - bases[cid]
		m := maps[cid]
		c := bisectRight(m, rel) - 1
		if c < 0 {
			return nil, nil, fmt.Errorf("ztr: sem slot para o offset %d", s)
		}
		k := rel - m[c]
		slot := sizes[cid] - m[c]
		if c+1 < len(m) {
			slot = m[c+1] - m[c]
		}
		if k >= slot {
			return nil, nil, fmt.Errorf("ztr: offset %d cai fora do slot", s)
		}
		if cid > 0xFF || c > 0xFFFF {
			return nil, nil, fmt.Errorf("ztr: valores fora do formato (chunk=%d pos=%d)", cid, c)
		}
		infos = append(infos, LineInfo{ChunkID: byte(cid), CharaStart: byte(k), Pos: uint16(c)})
	}
	return chunks, infos, nil
}

// bisectRight devolve o primeiro índice cujo valor é > x (upper bound).
func bisectRight(a []int, x int) int {
	lo, hi := 0, len(a)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if a[mid] <= x {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
