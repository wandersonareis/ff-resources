// Package ztr implementa o container .ztr da trilogia Final Fantasy XIII
// (Zone Text Resource): header big-endian, chunks de dicionário com
// substituição recursiva de pares (a "compressão" do formato) e a tabela de
// line info que localiza cada linha dentro dos chunks.
//
// A organização espelha o Pulse.FS/ZTR (header/unpacker/packer separados,
// entradas key->value), mas o algoritmo de compressão é o REAL, portado do
// CompressionHelpers.cs do ZTRtool (validado byte a byte contra os arquivos
// originais do jogo, incluindo o desempate tie_min_last que reproduz o
// gerador original). Os dois modos de saída do ZTRtool estão disponíveis:
// sem compressão (-c) e comprimido (-c2).
package ztr

import (
	"encoding/binary"
	"fmt"
)

// Magic identificador do formato (UInt64 big-endian == 1).
const Magic uint64 = 1

// LineInfo localiza uma linha: chunk, deslocamento dentro da expansão da
// página do dicionário (CharaStartInDictPage) e posição nas strings
// comprimidas do chunk (LineStartPosInChunk). No modo comprimido a posição
// pode cair no meio da expansão de um byte; CharaStart dá o deslocamento.
type LineInfo struct {
	ChunkID    byte
	CharaStart byte
	Pos        uint16
}

// Header do arquivo .ztr (tudo big-endian):
//
//	[u64 magic][u32 lineCount][u32 dcmpIDsSize][u32 chunkCount]
//	[u32 offsets...][LineInfo...]
type Header struct {
	Magic        uint64
	LineCount    uint32
	DcmpIDsSize  uint32
	ChunkOffsets []uint32
	LineInfos    []LineInfo
}

// chunkTableSizeBytes é o tamanho dos cabeçalhos fixos do arquivo.
const (
	lineInfoSize = 4
	maxChunkData = 4096 // limite de dados (descomprimidos) por chunk
)

// parseHeader lê o header e devolve o offset do fim (início da região de
// chunks de IDs).
func parseHeader(data []byte) (*Header, int, error) {
	if len(data) < 20 {
		return nil, 0, fmt.Errorf("ztr: arquivo menor que o header (%d bytes)", len(data))
	}
	h := &Header{
		Magic:       binary.BigEndian.Uint64(data[0:8]),
		LineCount:   binary.BigEndian.Uint32(data[8:12]),
		DcmpIDsSize: binary.BigEndian.Uint32(data[12:16]),
	}
	n := binary.BigEndian.Uint32(data[16:20])
	// Um arquivo VAZIO tem 1 offset (sem linhas) — o resto tem 2 ou mais
	// (um chunk de IDs + os chunks de texto).
	if int(n) < 1 || 20+int(n)*4 > len(data) {
		return nil, 0, fmt.Errorf("ztr: chunk table inválida (%d offsets)", n)
	}
	h.ChunkOffsets = make([]uint32, n)
	for i := range h.ChunkOffsets {
		h.ChunkOffsets[i] = binary.BigEndian.Uint32(data[20+4*i : 24+4*i])
	}
	liPos := 20 + int(n)*4
	if liPos+int(h.LineCount)*lineInfoSize > len(data) {
		return nil, 0, fmt.Errorf("ztr: line info table excede o arquivo")
	}
	h.LineInfos = make([]LineInfo, h.LineCount)
	for i := range h.LineInfos {
		o := liPos + i*lineInfoSize
		h.LineInfos[i] = LineInfo{
			ChunkID:    data[o],
			CharaStart: data[o+1],
			Pos:        binary.BigEndian.Uint16(data[o+2 : o+4]),
		}
	}
	if h.Magic != Magic {
		return nil, 0, fmt.Errorf("ztr: magic %#x, queria %d", h.Magic, Magic)
	}
	return h, liPos + int(h.LineCount)*lineInfoSize, nil
}

// isEmptyFile diz se o blob é um .ztr sem conteúdo: o jogo tem arquivos
// assim (o idioma daquela entrada não existe) — são só zeros ou o header
// sem linhas. Não é corrupção: o app abre e mostra a tabela vazia.
func isEmptyFile(data []byte) bool {
	if len(data) >= 8 {
		magic := binary.BigEndian.Uint64(data[0:8])
		if magic == Magic {
			// Header válido: sem linhas (e sem bloco de IDs) = vazio.
			if len(data) >= 16 {
				lines := binary.BigEndian.Uint32(data[8:12])
				ids := binary.BigEndian.Uint32(data[12:16])
				return lines == 0 && ids == 0
			}
			return true
		}
	}
	// Arquivo truncado só com zeros (4 bytes no retail).
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// appendHeader serializa o header.
func appendHeader(out []byte, h *Header) []byte {
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], h.Magic)
	out = append(out, b8[:]...)
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], h.LineCount)
	out = append(out, b4[:]...)
	binary.BigEndian.PutUint32(b4[:], h.DcmpIDsSize)
	out = append(out, b4[:]...)
	binary.BigEndian.PutUint32(b4[:], uint32(len(h.ChunkOffsets)))
	out = append(out, b4[:]...)
	for _, o := range h.ChunkOffsets {
		binary.BigEndian.PutUint32(b4[:], o)
		out = append(out, b4[:]...)
	}
	for _, li := range h.LineInfos {
		out = append(out, li.ChunkID, li.CharaStart)
		binary.BigEndian.PutUint16(b4[:2], li.Pos)
		out = append(out, b4[:2]...)
	}
	return out
}
