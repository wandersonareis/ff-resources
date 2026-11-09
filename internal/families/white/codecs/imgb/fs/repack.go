// Package imgb implementa repacking de arquivos IMGB.
package imgb

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	imgbcore "ffresources/internal/families/white/codecs/imgb/core"
)

// RepackIMGB reempacota um arquivo IMGB a partir de arquivos DDS extraídos.
// Equivalente a IMGBRepack1.RepackIMGBType1 e IMGBRepack2.RepackIMGBType2.
func RepackIMGB(headerBlockFile, outputImgbFile, extractedDir string, platform imgbcore.Platform) error {
	// Ler GTEX do arquivo de header
	gtex, err := imgbcore.ReadGTEX(headerBlockFile)
	if err != nil {
		return fmt.Errorf("lendo GTEX de %s: %w", headerBlockFile, err)
	}

	if !gtex.IsValid {
		return fmt.Errorf("GTEX inválido em %s", headerBlockFile)
	}

	fmt.Printf("Repacking IMGB: %s <- %s\n", outputImgbFile, extractedDir)
	fmt.Printf("  GTEX: %s %dx%d mips=%d type=%d fmt=%d\n",
		gtex.ImageName, gtex.Width, gtex.Height, gtex.MipCount, gtex.Type, gtex.Format)

	// Abrir arquivo DDS
	baseName := strings.TrimSuffix(filepath.Base(headerBlockFile), filepath.Ext(headerBlockFile))
	ddsFile := filepath.Join(extractedDir, baseName+".dds")

	ddsData, err := os.ReadFile(ddsFile)
	if err != nil {
		return fmt.Errorf("lendo DDS %s: %w", ddsFile, err)
	}

	// Validar DDS
	if len(ddsData) < 128 || string(ddsData[0:4]) != "DDS " {
		return fmt.Errorf("arquivo DDS inválido: %s", ddsFile)
	}

	// Parsear header DDS para obter info de mipmaps
	ddsMips, err := parseDDSMipOffsets(ddsData, gtex)
	if err != nil {
		return err
	}

	// Criar arquivo IMGB de saída
	outFile, err := os.Create(outputImgbFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Escrever baseado no formato detectado do IMGB original
	// Por enquanto, escrevemos no formato veffs (hash table)
	return writeIMGBVeffs(outFile, ddsData, ddsMips, gtex, platform)
}

// parseDDSMipOffsets extrai offsets e tamanhos dos mipmaps do header DDS.
func parseDDSMipOffsets(data []byte, gtex *imgbcore.GTEX) ([]imgbcore.MipInfo, error) {
	if len(data) < 148 {
		return nil, fmt.Errorf("DDS muito pequeno para header")
	}

	// Header DDS começa em offset 4 (após "DDS ")
	dwSize := binary.LittleEndian.Uint32(data[4:8])
	if dwSize != 124 {
		return nil, fmt.Errorf("DDS header size inesperado: %d", dwSize)
	}

	dwFlags := binary.LittleEndian.Uint32(data[8:12])
	dwMipMapCount := binary.LittleEndian.Uint32(data[28:32])
	_ = binary.LittleEndian.Uint32(data[20:24]) // dwPitchOrLinearSize - unused

	var mips []imgbcore.MipInfo
	dataOffset := 128 // Após header DDS (4 + 124)

	hasMipMap := (dwFlags & 0x00020000) != 0

	if hasMipMap && dwMipMapCount > 0 {
		for i := uint32(0); i < dwMipMapCount; i++ {
			// Calcular tamanho do mip baseado no formato
			var mipSize uint32
			if gtex.Format == 0 { // DXT1
				w := (gtex.Width + 3) >> 2
				h := (gtex.Height + 3) >> 2
				if w == 0 {
					w = 1
				}
				if h == 0 {
					h = 1
				}
				mipSize = uint32(w) * uint32(h) * 8
			} else if gtex.Format == 1 || gtex.Format == 2 { // DXT3/5
				w := (gtex.Width + 3) >> 2
				h := (gtex.Height + 3) >> 2
				if w == 0 {
					w = 1
				}
				if h == 0 {
					h = 1
				}
				mipSize = uint32(w) * uint32(h) * 16
			} else {
				w := gtex.Width >> i
				h := gtex.Height >> i
				if w == 0 {
					w = 1
				}
				if h == 0 {
					h = 1
				}
				mipSize = uint32(w * h * 4)
			}

			mips = append(mips, imgbcore.MipInfo{
				Offset: uint32(dataOffset),
				Size:   mipSize,
			})
			dataOffset += int(mipSize)
		}
	} else {
		// Single mip
		mips = append(mips, imgbcore.MipInfo{
			Offset: uint32(dataOffset),
			Size:   uint32(len(data)) - 128,
		})
	}

	return mips, nil
}

// writeIMGBVeffs escreve um arquivo IMGB no formato veffs (hash table).
func writeIMGBVeffs(w io.Writer, ddsData []byte, mips []imgbcore.MipInfo, gtex *imgbcore.GTEX, platform imgbcore.Platform) error {
	// Calcular tamanho necessário
	totalDataSize := 0
	for _, mip := range mips {
		totalDataSize += int(mip.Size)
	}

	// Header veffs: 16 bytes + 8191 entries * 16 bytes = 131072 bytes fixo?
	// O veffs.us.win32.imgb tem exatamente 131072 bytes
	const veffsTableSize = 8191 * 16
	const veffsHeaderSize = 16
	const veffsTotalSize = veffsHeaderSize + veffsTableSize

	// Alocar buffer
	buf := make([]byte, veffsTotalSize)

	// Escrever magic
	binary.LittleEndian.PutUint32(buf[0:4], imgbcore.IMGBMagicVeffs1)
	binary.LittleEndian.PutUint32(buf[4:8], imgbcore.IMGBMagicVeffs2)
	binary.LittleEndian.PutUint32(buf[8:12], 0x04FD051C)  // Magic3 do original
	binary.LittleEndian.PutUint32(buf[12:16], 0xAAAAAAAA) // Magic4 do original

	// Preencher tabela de hash (inicialmente tudo 0x55555555)
	for i := 0; i < 8191; i++ {
		off := 16 + i*16
		binary.LittleEndian.PutUint32(buf[off:off+4], 1)
		binary.LittleEndian.PutUint32(buf[off+4:off+8], 0x55555555)
		binary.LittleEndian.PutUint32(buf[off+8:off+12], 0)
		binary.LittleEndian.PutUint32(buf[off+12:off+16], 0xFFFFFFFF)
	}

	// Escrever dados dos mipmaps no final do arquivo (após a tabela)
	dataStart := veffsTotalSize

	// Para o repack, precisamos saber onde colocar os dados
	// O formato original parece ter os dados após a tabela
	// Vamos escrever os dados dos mipmaps sequencialmente

	currentOffset := uint32(dataStart)
	for _, mip := range mips {
		// Encontrar slot na hash table (simplificado: usar primeira entrada vazia)
		slot := findHashSlot(buf, uint32(currentOffset))
		if slot >= 0 {
			off := 16 + slot*16
			binary.LittleEndian.PutUint32(buf[off:off+4], 1)
			binary.LittleEndian.PutUint32(buf[off+4:off+8], 0)
			binary.LittleEndian.PutUint32(buf[off+8:off+12], currentOffset)
			binary.LittleEndian.PutUint32(buf[off+12:off+16], mip.Size)
		}
		currentOffset += mip.Size
	}

	// Escrever header + tabela
	if _, err := w.Write(buf); err != nil {
		return err
	}

	// Escrever dados dos mipmaps
	for _, mip := range mips {
		// Ler dados do DDS original
		start := 128 + int(mip.Offset)
		end := start + int(mip.Size)
		if end <= len(ddsData) {
			if _, err := w.Write(ddsData[start:end]); err != nil {
				return err
			}
		}
	}

	return nil
}

// findHashSlot encontra um slot disponível na hash table.
func findHashSlot(buf []byte, targetOffset uint32) int {
	// Hash simples baseado no offset
	hash := int(targetOffset % 8191)
	for i := 0; i < 8191; i++ {
		idx := (hash + i) % 8191
		off := 16 + idx*16
		val2 := binary.LittleEndian.Uint32(buf[off+4 : off+8])
		if val2 == 0x55555555 {
			return idx
		}
	}
	return -1
}

// RepackIMGBType2 reempacota usando o formato tipo 2 (para c001-style).
func RepackIMGBType2(headerBlockFile, outputImgbFile, extractedDir string, platform imgbcore.Platform) error {
	// Formato c001: 85 entradas de 8 bytes + dados
	// Pattern: val1=1, val2=0x55555555

	gtex, err := imgbcore.ReadGTEX(headerBlockFile)
	if err != nil {
		return err
	}

	baseName := strings.TrimSuffix(filepath.Base(headerBlockFile), filepath.Ext(headerBlockFile))
	ddsFile := filepath.Join(extractedDir, baseName+".dds")

	ddsData, err := os.ReadFile(ddsFile)
	if err != nil {
		return err
	}

	mips, err := parseDDSMipOffsets(ddsData, gtex)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputImgbFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Header: 85 * 8 = 680 bytes
	const headerEntries = 85
	headerSize := headerEntries * 8

	buf := make([]byte, headerSize)
	for i := 0; i < headerEntries; i++ {
		off := i * 8
		binary.LittleEndian.PutUint32(buf[off:off+4], 1)
		binary.LittleEndian.PutUint32(buf[off+4:off+8], 0x55555555)
	}

	if _, err := outFile.Write(buf); err != nil {
		return err
	}

	// Dados dos mipmaps
	for _, mip := range mips {
		start := 128 + int(mip.Offset)
		end := start + int(mip.Size)
		if end <= len(ddsData) {
			if _, err := outFile.Write(ddsData[start:end]); err != nil {
				return err
			}
		}
	}

	// Padding final com pattern 0x1F84FF7B 0xAAAAAAAA
	paddingPattern := []byte{0x1F, 0x84, 0xFF, 0x7B, 0xAA, 0xAA, 0xAA, 0xAA}
	for i := 0; i < 1000; i++ {
		if _, err := outFile.Write(paddingPattern); err != nil {
			return err
		}
	}

	return nil
}
