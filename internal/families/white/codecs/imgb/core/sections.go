package imgbcore

import (
	"encoding/binary"
	"fmt"
	"io"
)

// SectionHeader é o header SEDB que precede cada seção TXB/VTEX (48 bytes).
type SectionHeader struct {
	Magic         uint32
	Type          uint32
	Version       uint32
	Unknown       uint32
	SectionLength uint32
	Junk          [28]byte
}

const (
	// SectionMagic é "SEDB" em little-endian.
	SectionMagic uint32 = 0x42444553
	// SectionTypeTxb é "txb\0".
	SectionTypeTxb uint32 = 0x00627874
	// SectionTypeTxbh é "txbh" (aparece em WPD .xgr).
	SectionTypeTxbh uint32 = 0x68627874
	// SectionTypeVtex é "vtex".
	SectionTypeVtex uint32 = 0x78657476
)

// TypeName devolve o nome curto do tipo de seção ("txb", "vtex"...).
func (s SectionHeader) TypeName() string {
	switch s.Type {
	case SectionTypeTxb:
		return "txb"
	case SectionTypeTxbh:
		return "txbh"
	case SectionTypeVtex:
		return "vtex"
	}
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, s.Type)
	for i := range b {
		if b[i] == 0 {
			b = b[:i]
			break
		}
	}
	return string(b)
}

// ParseSectionHeader lê o header SEDB de uma seção.
func ParseSectionHeader(data []byte) (SectionHeader, error) {
	var h SectionHeader
	if len(data) < 48 {
		return h, fmt.Errorf("seção muito pequena: %d bytes", len(data))
	}
	h.Magic = binary.LittleEndian.Uint32(data[0:4])
	if h.Magic != SectionMagic {
		return h, fmt.Errorf("magic de seção inválido: 0x%08X", h.Magic)
	}
	h.Type = binary.LittleEndian.Uint32(data[4:8])
	h.Version = binary.BigEndian.Uint32(data[8:12])
	h.Unknown = binary.BigEndian.Uint32(data[12:16])
	h.SectionLength = binary.BigEndian.Uint32(data[16:20])
	copy(h.Junk[:], data[20:48])
	return h, nil
}

// TextureHeader é o header de 16 bytes antes do GTEX nos registros TXBH.
type TextureHeader struct {
	Unknown1 uint32
	Unknown2 uint32
	Unknown3 uint32
	Unknown4 uint32
}

// ParseTextureHeader lê o TextureHeader big-endian.
func ParseTextureHeader(data []byte) (TextureHeader, error) {
	var h TextureHeader
	if len(data) < 16 {
		return h, fmt.Errorf("texture header muito pequeno: %d bytes", len(data))
	}
	h.Unknown1 = binary.BigEndian.Uint32(data[0:4])
	h.Unknown2 = binary.BigEndian.Uint32(data[4:8])
	h.Unknown3 = binary.BigEndian.Uint32(data[8:12])
	h.Unknown4 = binary.BigEndian.Uint32(data[12:16])
	return h, nil
}

// VtexHeader é o header dos registros VTEX (0x64 bytes, conforme o Pulse).
type VtexHeader struct {
	Unknown1       int32
	DataLength     int32
	Unknown2       int16
	Unknown3       int16
	Unknown4       int16
	CounterOrType  int16
	Unknown6       []int32
	Name           string
	Extension      string
	Unknown17      int32
	GtexOffset     int32 // relativo ao fim do SectionHeader
	Unknown18      int32
	Unknown19      int32
}

// VtexHeaderSize é o tamanho fixo do VtexHeader.
const VtexHeaderSize = 0x64

func fixedString(b []byte) string {
	for i, v := range b {
		if v == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// ParseVtexHeader lê o VtexHeader de uma seção VTEX.
func ParseVtexHeader(data []byte) (VtexHeader, error) {
	var h VtexHeader
	if len(data) < 16 {
		return h, fmt.Errorf("vtex header muito pequeno: %d bytes", len(data))
	}
	le := binary.LittleEndian
	h.Unknown1 = int32(le.Uint32(data[0:4]))
	h.DataLength = int32(le.Uint32(data[4:8]))
	h.Unknown2 = int16(le.Uint16(data[8:10]))
	h.Unknown3 = int16(le.Uint16(data[10:12]))
	h.Unknown4 = int16(le.Uint16(data[12:14]))
	h.CounterOrType = int16(le.Uint16(data[14:16]))

	count := 0
	switch h.CounterOrType {
	case 0:
		count = 8
	case 2:
		count = 11
	default:
		count = 8
	}
	need := 16 + count*4 + 16 + 8 + 16
	if len(data) < need {
		return h, fmt.Errorf("vtex header truncado: %d bytes (esperado %d)", len(data), need)
	}
	off := 16
	h.Unknown6 = make([]int32, count)
	for i := 0; i < count; i++ {
		h.Unknown6[i] = int32(le.Uint32(data[off : off+4]))
		off += 4
	}
	h.Name = fixedString(data[off : off+16])
	off += 16
	h.Extension = fixedString(data[off : off+8])
	off += 8
	h.Unknown17 = int32(le.Uint32(data[off : off+4]))
	h.GtexOffset = int32(le.Uint32(data[off+4 : off+8]))
	h.Unknown18 = int32(le.Uint32(data[off+8 : off+12]))
	h.Unknown19 = int32(le.Uint32(data[off+12 : off+16]))
	return h, nil
}

// SectionGTEX localiza e interpreta o GTEX dentro de uma seção TXB/VTEX.
// O GTEX é procurado pelo magic, então funciona para os dois layouts.
func SectionGTEX(section []byte) (*GTEX, error) {
	if len(section) < 48 {
		return nil, fmt.Errorf("seção muito pequena: %d bytes", len(section))
	}
	h, err := ParseSectionHeader(section)
	if err == nil {
		switch h.Type {
		case SectionTypeVtex:
			if vh, verr := ParseVtexHeader(section[48:]); verr == nil {
				// GtexOffset é relativo ao fim do SectionHeader.
				base := 48 + int(vh.GtexOffset)
				if base >= 48 && base+20 <= len(section) {
					if g, ok := parseGTEXAt(section, base); ok {
						return g, nil
					}
				}
			}
		}
	}
	return ParseGTEX(section)
}

// ImageInfo é o retrato de uma textura decodificada.
type ImageInfo struct {
	Format      string
	Width       uint32
	Height      uint32
	MipmapCount uint32
	Layers      int
}

// FormatName devolve o nome legível do formato GTEX.
func FormatName(f GTEXFormat) string {
	switch f {
	case GTEXFormatDXT1, GTEXFormatLegacyDXT1:
		return "DXT1"
	case GTEXFormatDXT3, GTEXFormatLegacyDXT3:
		return "DXT3"
	case GTEXFormatDXT5, GTEXFormatLegacyDXT5:
		return "DXT5"
	case GTEXFormatRGB:
		return "X8R8G8B8"
	case GTEXFormatRGBA:
		return "A8R8G8B8"
	default:
		return fmt.Sprintf("GTEX(%d)", byte(f))
	}
}

// BuildDDS monta um DDS em memória a partir de uma seção TXB/VTEX e do
// conteúdo (.imgb). allMips=false usa só o mip 0 (prévia), como o
// DxTextureReader do Pulse.
func BuildDDS(section []byte, content io.ReaderAt, platform Platform, allMips bool) ([]byte, ImageInfo, error) {
	var info ImageInfo
	gtex, err := SectionGTEX(section)
	if err != nil {
		return nil, info, err
	}
	mips, err := ParseMipInfoTable(section, gtex)
	if err != nil {
		return nil, info, err
	}
	if len(mips) == 0 {
		return nil, info, fmt.Errorf("textura sem mips")
	}

	mipCount := 1
	if allMips {
		mipCount = len(mips)
	}
	if gtex.Type.IsCubemap() {
		mipCount = len(mips)
		if !allMips {
			mipCount = 1
		}
	}
	header, err := GetDDSHeaderMips(gtex, mipCount, 0)
	if err != nil {
		return nil, info, err
	}

	out := make([]byte, 0, len(header)+len(mips)*1024)
	out = append(out, header...)
	for i := 0; i < mipCount && i < len(mips); i++ {
		m := mips[i]
		if m.Size == 0 {
			continue
		}
		buf := make([]byte, m.Size)
		n, rerr := content.ReadAt(buf, int64(m.Offset))
		if rerr != nil && rerr != io.EOF {
			return nil, info, fmt.Errorf("lendo mip %d: %w", i, rerr)
		}
		if n != int(m.Size) {
			return nil, info, fmt.Errorf("mip %d: lidos %d de %d bytes", i, n, m.Size)
		}
		if platform == PlatformPS3 {
			buf = ProcessPS3ImageData(gtex, buf)
		}
		out = append(out, buf...)
	}

	info = ImageInfo{
		Format:      FormatName(gtex.Format),
		Width:       uint32(gtex.Width),
		Height:      uint32(gtex.Height),
		MipmapCount: uint32(gtex.MipCount),
		Layers:      gtex.LayerCount(),
	}
	return out, info, nil
}
