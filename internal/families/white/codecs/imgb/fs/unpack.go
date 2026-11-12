// Package imgb implementa unpacking de arquivos IMGB.
package imgb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	imgbcore "ffresources/internal/families/white/codecs/imgb/core"
)

// UnpackIMGB descompacta um arquivo IMGB emparelhado com um arquivo de header GTEX.
// Equivalente a IMGBUnpack.UnpackIMGB.
func UnpackIMGB(headerBlockFile, imgbFile, outputDir string, platform imgbcore.Platform, showLog bool) error {
	// Ler GTEX do arquivo de header
	gtex, err := imgbcore.ReadGTEX(headerBlockFile)
	if err != nil {
		return fmt.Errorf("lendo GTEX de %s: %w", headerBlockFile, err)
	}

	if !gtex.IsValid {
		return fmt.Errorf("GTEX inválido em %s", headerBlockFile)
	}

	if showLog {
		fmt.Printf("Unpacking IMGB: %s -> %s\n", imgbFile, outputDir)
		fmt.Printf("  GTEX: %s %dx%d mips=%d type=%d fmt=%d\n",
			gtex.ImageName, gtex.Width, gtex.Height, gtex.MipCount, gtex.Type, gtex.Format)
	}

	// Criar diretório de saída
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Abrir arquivo IMGB
	imgbData, err := os.ReadFile(imgbFile)
	if err != nil {
		return fmt.Errorf("lendo IMGB %s: %w", imgbFile, err)
	}

	// Detectar formato do IMGB
	_, err = imgbcore.DetectFormat(imgbFile)
	if err != nil {
		return err
	}

	var imgbStream io.ReaderAt
	imgbStream = NewByteReaderAt(imgbData)

	// Processar baseado no tipo de textura (inclui tipos de console 4/5)
	switch {
	case gtex.Type.IsClassic():
		return unpackClassic(headerBlockFile, imgbStream, outputDir, gtex, platform, showLog)
	case gtex.Type.IsCubemap():
		return unpackCubemap(headerBlockFile, imgbStream, outputDir, gtex, platform, showLog)
	case gtex.Type.IsVolumemap():
		return unpackVolumemap(headerBlockFile, imgbStream, outputDir, gtex, platform, showLog)
	default:
		return fmt.Errorf("tipo GTEX desconhecido: %d", gtex.Type)
	}
}

// ByteReaderAt implementa io.ReaderAt para []byte.
type ByteReaderAt []byte

func (b ByteReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(b)) {
		return 0, io.EOF
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// NewByteReaderAt cria um novo ByteReaderAt a partir de []byte.
func NewByteReaderAt(data []byte) ByteReaderAt {
	return ByteReaderAt(data)
}

// unpackClassic descompacta textura clássica (2D).
func unpackClassic(headerBlockFile string, imgbStream io.ReaderAt, outputDir string, gtex *imgbcore.GTEX, platform imgbcore.Platform, showLog bool) error {
	// Ler tabela de mipmaps do arquivo de header
	mipTable, err := readMipInfoTable(headerBlockFile, gtex)
	if err != nil {
		return fmt.Errorf("lendo tabela de mipmaps: %w", err)
	}

	// Gerar header DDS
	ddsHeader, err := imgbcore.GetDDSHeader(gtex)
	if err != nil {
		return err
	}

	// Arquivo de saída
	baseName := strings.TrimSuffix(filepath.Base(headerBlockFile), filepath.Ext(headerBlockFile))
	outputFile := filepath.Join(outputDir, baseName+".dds")

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Escrever header DDS
	if _, err := outFile.Write(ddsHeader); err != nil {
		return err
	}

	// Copiar dados de cada mipmap
	for i, mip := range mipTable {
		if mip.Size == 0 {
			if showLog {
				fmt.Printf("  Mip %d: vazio (offset=0x%X)\n", i, mip.Offset)
			}
			continue
		}

		if showLog {
			fmt.Printf("  Mip %d: offset=0x%X size=%d\n", i, mip.Offset, mip.Size)
		}

		// Ler dados do IMGB
		mipData := make([]byte, mip.Size)
		n, err := imgbStream.ReadAt(mipData, int64(mip.Offset))
		if err != nil && err != io.EOF {
			return fmt.Errorf("lendo mip %d: %w", i, err)
		}
		if n != int(mip.Size) {
			return fmt.Errorf("mip %d: tamanho inesperado (lidos %d, esperado %d)", i, n, mip.Size)
		}

		// Processar dados baseado na plataforma
		if platform == imgbcore.PlatformPS3 {
			// PS3: des-swizzle (se aplicável) e correção de cor ARGB -> BGRA
			mipData = imgbcore.ProcessPS3ImageData(gtex, mipData)
		}

		if _, err := outFile.Write(mipData); err != nil {
			return err
		}
	}

	if showLog {
		fmt.Printf("  DDS salvo: %s\n", outputFile)
	}

	return nil
}

// unpackCubemap descompacta cubemap (6 faces).
func unpackCubemap(headerBlockFile string, imgbStream io.ReaderAt, outputDir string, gtex *imgbcore.GTEX, platform imgbcore.Platform, showLog bool) error {
	// Para cubemaps, cada face tem seus próprios mipmaps
	// O GTEX contém offset para a tabela de mipmaps que inclui todas as faces
	mipTable, err := readMipInfoTable(headerBlockFile, gtex)
	if err != nil {
		return err
	}

	ddsHeader, err := imgbcore.GetDDSHeader(gtex)
	if err != nil {
		return err
	}

	baseName := strings.TrimSuffix(filepath.Base(headerBlockFile), filepath.Ext(headerBlockFile))
	outputFile := filepath.Join(outputDir, baseName+".dds")

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if _, err := outFile.Write(ddsHeader); err != nil {
		return err
	}

	// Cubemap tem 6 faces * MipCount mipmaps
	mipsPerFace := len(mipTable) / 6
	if len(mipTable)%6 != 0 {
		return fmt.Errorf("tabela de mipmaps inválida para cubemap: %d entradas", len(mipTable))
	}

	for face := 0; face < 6; face++ {
		for mip := 0; mip < mipsPerFace; mip++ {
			idx := face*mipsPerFace + mip
			mipInfo := mipTable[idx]

			if mipInfo.Size == 0 {
				continue
			}

			mipData := make([]byte, mipInfo.Size)
			n, err := imgbStream.ReadAt(mipData, int64(mipInfo.Offset))
			if err != nil && err != io.EOF {
				return err
			}
			if n != int(mipInfo.Size) {
				return fmt.Errorf("face %d mip %d: tamanho inesperado", face, mip)
			}

			if platform == imgbcore.PlatformPS3 {
				mipData = imgbcore.ProcessPS3ImageData(gtex, mipData)
			}

			if _, err := outFile.Write(mipData); err != nil {
				return err
			}
		}
	}

	if showLog {
		fmt.Printf("  Cubemap DDS salvo: %s\n", outputFile)
	}

	return nil
}

// unpackVolumemap descompacta volume map (3D texture).
func unpackVolumemap(headerBlockFile string, imgbStream io.ReaderAt, outputDir string, gtex *imgbcore.GTEX, platform imgbcore.Platform, showLog bool) error {
	// Similar ao classic mas com depth > 1
	return unpackClassic(headerBlockFile, imgbStream, outputDir, gtex, platform, showLog)
}

// readMipInfoTable lê a tabela de informação de mipmaps do arquivo de header.
func readMipInfoTable(headerBlockFile string, gtex *imgbcore.GTEX) ([]imgbcore.MipInfo, error) {
	data, err := os.ReadFile(headerBlockFile)
	if err != nil {
		return nil, err
	}
	return imgbcore.ParseMipInfoTable(data, gtex)
}

// GetGTEXChunkOffset obtém o offset do chunk GTEX no arquivo (equivalente a SharedMethods.GetGTEXChunkOffset).
func GetGTEXChunkOffset(path string) (int, error) {
	return imgbcore.GetGTEXChunkOffset(path)
}
