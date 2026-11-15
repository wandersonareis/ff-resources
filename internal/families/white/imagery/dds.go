// Package imagery contém utilitários de imagem compartilhados entre famílias.
package imagery

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"

	"github.com/woozymasta/bcn"
)

const ddsHeaderLen = 128

const (
	ddpfAlphaPixels = 0x1
	ddpfAlpha       = 0x2
	ddpfFourCC      = 0x4
	ddpfRGB         = 0x40
	ddpfLuminance   = 0x20000

	ddsFlagPitch = 0x8

	fccDXT1 = 0x31545844
	fccDXT3 = 0x33545844
	fccDXT5 = 0x35545844
	fccBC5U = 0x55354342
	fccATI2 = 0x32495441
)

// DDSInfo é o resultado do parse de um header DDS.
type DDSInfo struct {
	Width  uint32
	Height uint32
	Format string
}

// DDSToPNG decodifica o mip 0 de um DDS em PNG. O payload do DDS é gravado de
// baixo para cima, então a imagem é espelhada na vertical.
func DDSToPNG(dds []byte) ([]byte, error) {
	if len(dds) < ddsHeaderLen {
		return nil, fmt.Errorf("dds: %d bytes, esperado ao menos %d", len(dds), ddsHeaderLen)
	}
	if string(dds[0:4]) != "DDS " {
		return nil, fmt.Errorf("dds: magia inválida %q", string(dds[0:4]))
	}
	width := binary.LittleEndian.Uint32(dds[16:20])
	height := binary.LittleEndian.Uint32(dds[12:16])
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("dds: dimensões inválidas %dx%d", width, height)
	}
	pfFlags := binary.LittleEndian.Uint32(dds[80:84])
	fourCC := binary.LittleEndian.Uint32(dds[84:88])
	flags := binary.LittleEndian.Uint32(dds[8:12])

	format := bcn.FormatUnknown
	switch {
	case pfFlags&ddpfFourCC != 0:
		switch fourCC {
		case fccDXT1:
			format = bcn.FormatBC1
		case fccDXT3:
			format = bcn.FormatBC2
		case fccDXT5:
			format = bcn.FormatBC3
		case fccBC5U, fccATI2:
			format = bcn.FormatBC5
		}
	default:
		switch {
		case pfFlags&ddpfRGB != 0 && flags&ddsFlagPitch != 0:
			format = bcn.FormatBGRA8
		case pfFlags&ddpfLuminance != 0:
			format = bcn.FormatR8
		case pfFlags&(ddpfAlphaPixels|ddpfAlpha) != 0 && pfFlags&ddpfRGB == 0:
			format = bcn.FormatA8
		}
	}
	if format == bcn.FormatUnknown {
		return nil, fmt.Errorf("dds: formato não reconhecido (pfFlags=0x%08X fourCC=0x%08X)", pfFlags, fourCC)
	}

	img, err := bcn.DecodeImage(dds[ddsHeaderLen:], int(width), int(height), format)
	if err != nil {
		return nil, fmt.Errorf("decodificando dds: %w", err)
	}
	// O conteúdo GTEX/IMGB do FF13 é top-down (diferente do payload Phyre do
	// FFX, que é bottom-up). Não espelhar aqui: a orientação do jogo é a
	// orientação servida; o Flip do painel é só visual.
	if format == bcn.FormatA8 {
		pix := img.Pix
		for i := 0; i+3 < len(pix); i += 4 {
			v := pix[i+3]
			pix[i], pix[i+1], pix[i+2], pix[i+3] = v, v, v, 255
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("gerando png: %w", err)
	}
	return buf.Bytes(), nil
}

// InfoDDS lê dimensões e formato de um DDS.
func InfoDDS(dds []byte) (DDSInfo, error) {
	var out DDSInfo
	if len(dds) < ddsHeaderLen || string(dds[0:4]) != "DDS " {
		return out, fmt.Errorf("dds inválido")
	}
	out.Width = binary.LittleEndian.Uint32(dds[16:20])
	out.Height = binary.LittleEndian.Uint32(dds[12:16])
	fourCC := binary.LittleEndian.Uint32(dds[84:88])
	switch fourCC {
	case fccDXT1:
		out.Format = "DXT1"
	case fccDXT3:
		out.Format = "DXT3"
	case fccDXT5:
		out.Format = "DXT5"
	case fccBC5U, fccATI2:
		out.Format = "BC5"
	default:
		out.Format = "RGBA8"
	}
	return out, nil
}
