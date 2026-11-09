package filelist

import (
	"encoding/binary"
	"fmt"
)

// Build serializa a Collection no arquivo de filelist pronto para gravar.
// Para ff132 com Crypt.Encrypted=true, o resultado já sai criptografado com o
// cabeçalho e o checksum.
//
// O binário gerado serve para PC (win32), PS3 e Xbox 360: o layout é o mesmo,
// e a única diferença entre plataformas são os caminhos e o conteúdo do white
// bin, que já vêm na Collection.
func (c *Collection) Build() ([]byte, error) {
	plain, err := c.buildPlaintext()
	if err != nil {
		return nil, err
	}
	if !c.Crypt.Encrypted {
		return plain, nil
	}
	return c.encrypt(plain)
}

// buildPlaintext monta o layout lógico:
//
//	[crypt header 32, se houver][header 12][entries][chunkInfo][chunkData]
//
// Os offsets do header são relativos ao início dos dados após o crypt header.
func (c *Collection) buildPlaintext() ([]byte, error) {
	fileCount := c.FileCount()
	chunkCount := c.ChunkCount()
	if chunkCount == 0 {
		return nil, fmt.Errorf("collection sem chunks")
	}

	var out []byte
	if c.Crypt.Encrypted {
		md5 := c.Crypt.MD5HashBytes()
		out = append(out, md5[:]...)
		var b4 [4]byte
		binary.LittleEndian.PutUint32(b4[:], 0xFFFFFFFF)
		out = append(out, b4[:]...)
		binary.LittleEndian.PutUint32(b4[:], c.Crypt.EncryptionTag)
		out = append(out, b4[:]...)
		// O cabeçalho de criptografia tem 32 bytes: sobram 8 bytes zerados
		// (BuildCryptHeader escreve apenas 24 bytes num buffer de 32).
		out = append(out, make([]byte, 8)...)
	}

	// Header.
	chunkInfoTableOffset := uint32(12 + fileCount*8)
	chunkDataStartOffset := uint32(12 + fileCount*8 + chunkCount*12)
	header := make([]byte, 12)
	binary.LittleEndian.PutUint32(header[0:], chunkInfoTableOffset)
	binary.LittleEndian.PutUint32(header[4:], chunkDataStartOffset)
	binary.LittleEndian.PutUint32(header[8:], uint32(fileCount))
	out = append(out, header...)

	// Para cada chunk, monta os dados e a posição de cada fileInfo.
	type chunkBuild struct {
		data  []byte
		cmp   []byte
		infos []uint16 // posição de cada entrada dentro do chunk (na ordem)
	}
	builds := make([]chunkBuild, chunkCount)

	for i := 0; i < chunkCount; i++ {
		ch := &c.Chunks[i]
		var data []byte
		infos := make([]uint16, len(ch.Entries))
		for j := range ch.Entries {
			info := ch.Entries[j].FileInfo
			pos := len(data)
			if c.GameCode.UsesV2Entries() {
				if pos > 0x7FFF {
					return nil, fmt.Errorf("chunk %d: posição de fileInfo (%d) excede 15 bits do layout V2", i, pos)
				}
			} else if pos > 0xFFFF {
				return nil, fmt.Errorf("chunk %d: posição de fileInfo (%d) excede 16 bits", i, pos)
			}
			infos[j] = uint16(pos)
			data = append(data, info...)
			data = append(data, 0)
		}
		// O último chunk recebe o sentinela "end\0" após as strings reais.
		if i == chunkCount-1 {
			data = append(data, []byte("end")...)
			data = append(data, 0)
		}
		cmp, err := zlibCompress(data)
		if err != nil {
			return nil, fmt.Errorf("comprimindo chunk %d: %w", i, err)
		}
		builds[i] = chunkBuild{data: data, cmp: cmp, infos: infos}
	}

	// Tabela de entradas.
	entries := make([]byte, fileCount*8)
	ep := 0
	for i := 0; i < chunkCount; i++ {
		for j := range c.Chunks[i].Entries {
			e := &c.Chunks[i].Entries[j]
			binary.LittleEndian.PutUint32(entries[ep:], e.FileCode)
			if c.GameCode.UsesV2Entries() {
				infoPos := builds[i].infos[j]
				chunkID := uint16(i)
				if chunkID%2 != 0 {
					infoPos |= 0x8000
				}
				binary.LittleEndian.PutUint16(entries[ep+4:], infoPos)
				entries[ep+6] = byte(chunkID >> 1)
				entries[ep+7] = e.FileTypeID
			} else {
				binary.LittleEndian.PutUint16(entries[ep+4:], uint16(i))
				binary.LittleEndian.PutUint16(entries[ep+6:], builds[i].infos[j])
			}
			ep += 8
		}
	}
	out = append(out, entries...)

	// Tabela de informações dos chunks.
	chunkInfo := make([]byte, chunkCount*12)
	var chunkPos uint32
	for i := 0; i < chunkCount; i++ {
		b := builds[i]
		binary.LittleEndian.PutUint32(chunkInfo[i*12+0:], uint32(len(b.data)))
		binary.LittleEndian.PutUint32(chunkInfo[i*12+4:], uint32(len(b.cmp)))
		binary.LittleEndian.PutUint32(chunkInfo[i*12+8:], chunkPos)
		chunkPos += uint32(len(b.cmp))
	}
	out = append(out, chunkInfo...)

	// Dados comprimidos dos chunks.
	for i := 0; i < chunkCount; i++ {
		out = append(out, builds[i].cmp...)
	}

	return out, nil
}

// encrypt reproduz FilelistCrypto.EncryptProcess + CryptFilelist (encrypt).
func (c *Collection) encrypt(plain []byte) ([]byte, error) {
	if len(plain) < 32 {
		return nil, fmt.Errorf("filelist sem cabeçalho de criptografia")
	}

	filelistDataSize := uint32(len(plain) - 32)
	if filelistDataSize%8 != 0 {
		pad := 8 - (filelistDataSize % 8)
		plain = append(plain, make([]byte, pad)...)
		filelistDataSize += pad
	}
	plain = append(plain, make([]byte, 16)...)

	binary.BigEndian.PutUint32(plain[16:20], filelistDataSize)
	binary.LittleEndian.PutUint32(plain[len(plain)-16:], filelistDataSize)

	cryptBodySize := filelistDataSize + 8
	blockCount := cryptBodySize / 8

	// Preenche o checksum (o WhiteBinTools grava em 32+filelistDataSize+4).
	checksum := computeCheckSum(plain, (cryptBodySize-8)/4, 32)
	binary.LittleEndian.PutUint32(plain[32+int(filelistDataSize)+4:], checksum)

	md5 := c.Crypt.MD5HashBytes()
	xorTable := generateXORTable(seedFromMD5(md5))
	cryptBody(xorTable, plain[32:len(plain)-8], blockCount, true)

	return plain, nil
}
