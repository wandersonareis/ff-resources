package ztr

import (
	"encoding/binary"
	"fmt"
)

// Format identifica o empacotamento de um .ztr na GRAVAÇÃO. A leitura é
// automática (Read detecta o formato pelo primeiro u32).
//
// Os nomes espelham o ZtrFileType do Pulse:
//   - big-endian comprimido: o formato original do jogo (magic 1);
//   - little-endian: par (1 entrada) ou dicionário (N entradas), sem chunks.
type Format int

const (
	// FormatBigEndianCompressed é o formato do jogo: header big-endian,
	// chunks de dicionário e line info (o WriteTie atual).
	FormatBigEndianCompressed Format = iota
	// FormatLittleEndian grava o dicionário little-endian não comprimido
	// usado pelos arquivos de tradução (par quando há 1 entrada).
	FormatLittleEndian
)

// WriteFormat grava f no formato pedido. O BE continua sendo o padrão do
// app; o LE fica disponível para uma escolha posterior (config/UI).
func WriteFormat(f *File, format Format, mode Mode, tie TieRule) ([]byte, error) {
	switch format {
	case FormatLittleEndian:
		return WriteLE(f)
	case FormatBigEndianCompressed:
		return WriteTie(f, mode, tie)
	default:
		return nil, fmt.Errorf("ztr: formato de gravação desconhecido: %d", format)
	}
}

// WriteLE grava o dicionário little-endian do Pulse. Com uma entrada usa o
// par (ZtrFileType.LittleEndianUncompressedPair); com mais de uma usa o
// dicionário (LittleEndianUncompressedDictionary).
func WriteLE(f *File) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("ztr: file nil")
	}
	if len(f.IDs) != len(f.Lines) {
		return nil, fmt.Errorf("ztr: %d IDs vs %d linhas", len(f.IDs), len(f.Lines))
	}
	if len(f.IDs) == 0 {
		return nil, fmt.Errorf("ztr: arquivo sem linhas")
	}
	if len(f.IDs) == 1 {
		return writeLittleEndianPair(f.IDs[0], f.Lines[0])
	}
	return writeLittleEndianDictionary(f)
}

// writeLittleEndianPair espelha ZtrFilePacker.PackLittleEndianUncompressedPair:
//
//	[u32 LE 1][u32 LE keyOff=12][u32 LE valueOff=12+len(key)+1]
//	[key][0][value][00 00]
func writeLittleEndianPair(id string, line []byte) ([]byte, error) {
	key := []byte(id)
	value := ensureLineTerminator(line)

	keyOff := uint32(12)
	valueOff := keyOff + uint32(len(key)) + 1

	out := make([]byte, 0, int(valueOff)+len(value))
	var b4 [4]byte
	binary.LittleEndian.PutUint32(b4[:], 1)
	out = append(out, b4[:]...)
	binary.LittleEndian.PutUint32(b4[:], keyOff)
	out = append(out, b4[:]...)
	binary.LittleEndian.PutUint32(b4[:], valueOff)
	out = append(out, b4[:]...)
	out = append(out, key...)
	out = append(out, 0)
	out = append(out, value...)
	return out, nil
}

// writeLittleEndianDictionary espelha
// ZtrFilePacker.PackLittleEndianUncompressedDictionary:
//
//	[u32 LE count][count*2 offsets LE][chave \0 valor 00 00]...
func writeLittleEndianDictionary(f *File) ([]byte, error) {
	count := len(f.IDs)
	offsets := make([]uint32, count*2)
	pos := uint32(4 + count*2*4)

	for i := range f.IDs {
		key := []byte(f.IDs[i])
		value := ensureLineTerminator(f.Lines[i])
		offsets[i*2] = pos
		pos += uint32(len(key)) + 1
		offsets[i*2+1] = pos
		pos += uint32(len(value))
	}

	out := make([]byte, 0, int(pos))
	var b4 [4]byte
	binary.LittleEndian.PutUint32(b4[:], uint32(count))
	out = append(out, b4[:]...)
	for _, off := range offsets {
		binary.LittleEndian.PutUint32(b4[:], off)
		out = append(out, b4[:]...)
	}
	for i := range f.IDs {
		out = append(out, []byte(f.IDs[i])...)
		out = append(out, 0)
		out = append(out, ensureLineTerminator(f.Lines[i])...)
	}
	return out, nil
}

// readLittleEndianPair lê o par não comprimido:
//
//	[u32 LE 1][u32 LE keyOff][u32 LE valueOff][chave \0][valor 00 00]
func readLittleEndianPair(data []byte) (*File, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("ztr LE par: header curto (%d bytes)", len(data))
	}
	keyOff := int(binary.LittleEndian.Uint32(data[4:8]))
	valueOff := int(binary.LittleEndian.Uint32(data[8:12]))

	key, err := readBoundedKey(data, keyOff, valueOff)
	if err != nil {
		return nil, fmt.Errorf("ztr LE par: chave: %w", err)
	}
	value, err := readBoundedValue(data, valueOff, len(data))
	if err != nil {
		return nil, fmt.Errorf("ztr LE par: valor: %w", err)
	}
	return &File{
		IDs:   []string{string(key)},
		Lines: [][]byte{ensureLineTerminator(value)},
	}, nil
}

// readLittleEndianDictionary lê o dicionário não comprimido:
//
//	[u32 LE count][count*2 offsets LE][chave \0 valor 00 00]...
func readLittleEndianDictionary(data []byte, count int) (*File, error) {
	if count < 0 || count > 102400 {
		return nil, fmt.Errorf("ztr LE dicionário: contagem fora do limite: %d", count)
	}
	if count == 0 {
		return nil, fmt.Errorf("ztr LE dicionário: contagem zero")
	}
	need := 4 + count*2*4
	if need > len(data) {
		return nil, fmt.Errorf("ztr LE dicionário: tabela de offsets excede o arquivo")
	}

	ids := make([]string, count)
	lines := make([][]byte, count)
	for i := 0; i < count; i++ {
		keyOff := int(binary.LittleEndian.Uint32(data[4+i*8 : 8+i*8]))
		valueOff := int(binary.LittleEndian.Uint32(data[8+i*8 : 12+i*8]))

		// O valor termina onde começa a próxima chave (ou no fim do
		// arquivo). Usar o offset vizinho — em vez de procurar 00 00 solto
		// — aceita os arquivos de tradução que terminam o valor com um
		// único zero (alguns .ztr antigos do part1 são assim).
		valueEnd := len(data)
		if i+1 < count {
			valueEnd = int(binary.LittleEndian.Uint32(data[4+(i+1)*8 : 8+(i+1)*8]))
		}

		key, err := readBoundedKey(data, keyOff, valueOff)
		if err != nil {
			return nil, fmt.Errorf("ztr LE dicionário: chave %d: %w", i, err)
		}
		value, err := readBoundedValue(data, valueOff, valueEnd)
		if err != nil {
			return nil, fmt.Errorf("ztr LE dicionário: valor %d: %w", i, err)
		}
		ids[i] = string(key)
		lines[i] = ensureLineTerminator(value)
	}
	return &File{IDs: ids, Lines: lines}, nil
}

// readBoundedKey lê a chave até o primeiro zero dentro de [off, end).
func readBoundedKey(data []byte, off, end int) ([]byte, error) {
	if off < 0 || off >= len(data) {
		return nil, fmt.Errorf("offset %d fora do arquivo (%d bytes)", off, len(data))
	}
	if end <= off || end > len(data) {
		end = len(data)
	}
	for i := off; i < end; i++ {
		if data[i] == 0 {
			return data[off:i], nil
		}
	}
	return nil, fmt.Errorf("chave sem terminador a partir de %d", off)
}

// readBoundedValue lê o valor em [off, end) e remove os zeros finais do
// terminador. O Pulse usa 00 00, mas alguns .ztr de tradução do part1 usam
// um único zero; limitar pelo offset vizinho aceita os dois.
func readBoundedValue(data []byte, off, end int) ([]byte, error) {
	if off < 0 || off > len(data) {
		return nil, fmt.Errorf("offset %d fora do arquivo (%d bytes)", off, len(data))
	}
	if end <= off || end > len(data) {
		end = len(data)
	}
	value := data[off:end]
	for len(value) > 0 && value[len(value)-1] == 0 {
		value = value[:len(value)-1]
	}
	return value, nil
}

// ensureLineTerminator devolve a linha com o terminador 00 00 que o
// DecodeLines espera (as linhas lidas do BE já o incluem).
func ensureLineTerminator(line []byte) []byte {
	if len(line) >= 2 && line[len(line)-2] == 0 && line[len(line)-1] == 0 {
		return line
	}
	out := make([]byte, 0, len(line)+2)
	out = append(out, line...)
	out = append(out, 0, 0)
	return out
}
