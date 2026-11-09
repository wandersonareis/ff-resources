package filelist

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// rawEntry é uma entrada como está na tabela binária, antes de resolver o
// caminho no chunk.
type rawEntry struct {
	fileCode    uint32
	chunkID     uint16
	fileInfoPos uint16
	fileTypeID  byte
}

// Parse lê um filelist (possivelmente criptografado, no caso do ff132) e
// devolve uma Collection neutra. A descriptografia acontece em memória.
//
// Funciona igualmente para filelists de PC (win32), PS3 e Xbox 360, pois o
// layout e a criptografia são os mesmos nas três plataformas (ver package doc).
func Parse(data []byte, g GameCode) (*Collection, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("filelist muito pequeno: %d bytes", len(data))
	}

	work, base, crypt, err := decryptFilelist(data, g)
	if err != nil {
		return nil, err
	}
	coll := &Collection{GameCode: g, Crypt: crypt}

	if len(work) < base+12 {
		return nil, fmt.Errorf("filelist truncado no cabeçalho")
	}

	chunkInfoTableOffset := binary.LittleEndian.Uint32(work[base:])
	chunkDataStartOffset := binary.LittleEndian.Uint32(work[base+4:])
	fileCount := binary.LittleEndian.Uint32(work[base+8:])

	entries := make([]rawEntry, 0, fileCount)
	pos := base + 12
	for i := uint32(0); i < fileCount; i++ {
		if pos+8 > len(work) {
			return nil, fmt.Errorf("filelist truncado na tabela de entradas (entrada %d)", i)
		}
		if g.UsesV2Entries() {
			fileCode := binary.LittleEndian.Uint32(work[pos:])
			fileInfoPosRaw := binary.LittleEndian.Uint16(work[pos+4:])
			chunkByte := work[pos+6]
			fileTypeID := work[pos+7]
			chunkID := uint16(chunkByte)<<1 | fileInfoPosRaw>>15
			entries = append(entries, rawEntry{
				fileCode:    fileCode,
				chunkID:     chunkID,
				fileInfoPos: fileInfoPosRaw & 0x7FFF,
				fileTypeID:  fileTypeID,
			})
		} else {
			entries = append(entries, rawEntry{
				fileCode:    binary.LittleEndian.Uint32(work[pos:]),
				chunkID:     binary.LittleEndian.Uint16(work[pos+4:]),
				fileInfoPos: binary.LittleEndian.Uint16(work[pos+6:]),
			})
		}
		pos += 8
	}

	// O WhiteBinTools usa o chunkID da última entrada + 1.
	chunkCount := 1
	if len(entries) > 0 {
		chunkCount = int(entries[len(entries)-1].chunkID) + 1
	}

	chunkData := make([][]byte, chunkCount)
	for i := 0; i < chunkCount; i++ {
		p := base + int(chunkInfoTableOffset) + i*12
		if p+12 > len(work) {
			return nil, fmt.Errorf("filelist truncado na tabela de chunks (chunk %d)", i)
		}
		cmpSize := binary.LittleEndian.Uint32(work[p+4:])
		chunkPos := binary.LittleEndian.Uint32(work[p+8:])
		start := base + int(chunkDataStartOffset) + int(chunkPos)
		if start+int(cmpSize) > len(work) {
			return nil, fmt.Errorf("dados do chunk %d fora do arquivo", i)
		}
		dec, err := zlibDecompress(work[start : start+int(cmpSize)])
		if err != nil {
			return nil, fmt.Errorf("decomprimindo chunk %d: %w", i, err)
		}
		chunkData[i] = dec
	}

	coll.Chunks = make([]Chunk, chunkCount)
	for i := range coll.Chunks {
		coll.Chunks[i].ID = uint16(i)
	}
	for _, re := range entries {
		if int(re.chunkID) >= chunkCount {
			return nil, fmt.Errorf("entrada referencia chunk inexistente: %d", re.chunkID)
		}
		info := readCString(chunkData[re.chunkID], int(re.fileInfoPos))
		coll.Chunks[re.chunkID].Entries = append(coll.Chunks[re.chunkID].Entries, Entry{
			FileCode:   re.fileCode,
			FileTypeID: re.fileTypeID,
			FileInfo:   info,
		})
	}

	return coll, nil
}

// decryptFilelist detecta e remove a criptografia do filelist (ff132),
// devolvendo o buffer lógico, o offset base dos dados e as informações de
// criptografia. Para arquivos sem cabeçalho de criptografia, retorna os dados
// originais e base 0.
func decryptFilelist(data []byte, g GameCode) (work []byte, base int, crypt CryptInfo, err error) {
	hasCrypt := len(data) >= 24 && binary.LittleEndian.Uint32(data[20:24]) == EncryptionTagConstant
	if hasCrypt && g != GameFF132 {
		return nil, 0, crypt, fmt.Errorf("detectado filelist criptografado; use o game code 'ff132'")
	}

	work = data
	base = 0
	if !hasCrypt {
		return work, base, crypt, nil
	}

	base = 32
	var md5 [16]byte
	copy(md5[:], data[0:16])
	crypt.Encrypted = true
	crypt.SeedA = binary.LittleEndian.Uint64(md5[0:8])
	crypt.SeedB = binary.LittleEndian.Uint64(md5[8:16])
	crypt.EncryptionTag = binary.LittleEndian.Uint32(data[20:24])

	bodySize := binary.BigEndian.Uint32(data[16:20])
	cryptBodySize := bodySize + 8
	if cryptBodySize%8 != 0 {
		return nil, 0, crypt, fmt.Errorf("tamanho do corpo a decifrar não é válido (não múltiplo de 8)")
	}

	isDecrypted := false
	if int(32+cryptBodySize) <= len(data) {
		if binary.LittleEndian.Uint32(data[32+cryptBodySize-8:]) == cryptBodySize-8 {
			isDecrypted = true
		}
	}
	if isDecrypted {
		return work, base, crypt, nil
	}

	xorTable := generateXORTable(seedFromMD5(md5))
	work = make([]byte, len(data))
	copy(work, data)
	cryptBody(xorTable, work[32:], cryptBodySize/8, false)

	filelistDataSize := binary.BigEndian.Uint32(work[16:20])
	hashOffset := 32 + int(filelistDataSize) + 4
	if hashOffset+4 <= len(work) {
		stored := binary.LittleEndian.Uint32(work[hashOffset:])
		computed := computeCheckSum(work, filelistDataSize/4, 32)
		if stored != computed {
			return nil, 0, crypt, fmt.Errorf("filelist não foi decifrado corretamente (checksum inválido)")
		}
	}
	return work, base, crypt, nil
}

// Decrypted devolve o filelist com o corpo já decifrado (mantendo o cabeçalho
// de criptografia de 32 bytes). Para arquivos não criptografados, devolve os
// próprios dados. Útil para inspeção/cruzamento com outras ferramentas.
func Decrypted(data []byte, g GameCode) ([]byte, error) {
	work, _, _, err := decryptFilelist(data, g)
	return work, err
}

// readCString lê bytes ASCII a partir de start até um '\0' (ou o fim).
func readCString(b []byte, start int) string {
	if start < 0 || start >= len(b) {
		return ""
	}
	end := start
	for end < len(b) && b[end] != 0 {
		end++
	}
	return string(b[start:end])
}

func zlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func zlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
