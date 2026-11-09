// Package imgbcore fornece os tipos fundamentais do formato IMGB.
package imgbcore

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Platform representa a plataforma do arquivo IMGB.
type Platform int

const (
	PlatformWin32 Platform = iota
	PlatformPS3
	PlatformX360
)

func (p Platform) String() string {
	switch p {
	case PlatformWin32:
		return "win32"
	case PlatformPS3:
		return "ps3"
	case PlatformX360:
		return "x360"
	default:
		return fmt.Sprintf("unknown(%d)", p)
	}
}

// ParsePlatform converte uma string em Platform.
func ParsePlatform(s string) (Platform, error) {
	switch strings.ToLower(s) {
	case "win32":
		return PlatformWin32, nil
	case "ps3":
		return PlatformPS3, nil
	case "x360":
		return PlatformX360, nil
	default:
		return 0, fmt.Errorf("plataforma inválida: %s (esperado: win32, ps3, x360)", s)
	}
}

// FileExtension representa as extensões de arquivo suportadas pelo IMGB.
type FileExtension int

const (
	ExtTXB FileExtension = iota
	ExtTXBH
	ExtVTEX
)

func ParseFileExtension(s string) (FileExtension, error) {
	switch s {
	case "txb":
		return ExtTXB, nil
	case "txbh":
		return ExtTXBH, nil
	case "vtex":
		return ExtVTEX, nil
	}
	return 0, fmt.Errorf("extensão IMGB inválida: %s (esperado: txb, txbh, vtex)", s)
}

func (e FileExtension) String() string {
	switch e {
	case ExtTXB:
		return "txb"
	case ExtTXBH:
		return "txbh"
	case ExtVTEX:
		return "vtex"
	default:
		return fmt.Sprintf("unknown(%d)", e)
	}
}

// GTEXType representa o tipo de textura GTEX.
//
// Os valores 0/1/2 são usados nas versões de PC (win32); 4/5 são as variantes
// de console (PS3/X360), conforme IMGBlibrary (IMGBUnpack.cs):
// 4 = classic console, 5 = cubemap console.
type GTEXType byte

const (
	GTEXTypeClassic        GTEXType = 0
	GTEXTypeCubemap        GTEXType = 1
	GTEXTypeVolumemap      GTEXType = 2
	GTEXTypeClassicConsole GTEXType = 4
	GTEXTypeCubemapConsole GTEXType = 5
)

func (t GTEXType) String() string {
	switch t {
	case GTEXTypeClassic:
		return "classic"
	case GTEXTypeCubemap:
		return "cubemap"
	case GTEXTypeVolumemap:
		return "volumemap"
	case GTEXTypeClassicConsole:
		return "classic-console"
	case GTEXTypeCubemapConsole:
		return "cubemap-console"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

// IsClassic indica textura 2D clássica (tipo 0 no PC, 4 no console).
func (t GTEXType) IsClassic() bool {
	return t == GTEXTypeClassic || t == GTEXTypeClassicConsole
}

// IsCubemap indica cubemap (tipo 1 no PC, 5 no console).
func (t GTEXType) IsCubemap() bool {
	return t == GTEXTypeCubemap || t == GTEXTypeCubemapConsole
}

// IsVolumemap indica volume map (tipo 2).
func (t GTEXType) IsVolumemap() bool {
	return t == GTEXTypeVolumemap
}

// GTEXTypeValues são os tipos aceitos (equivalente a SharedMethods.GTEXTypeValues).
var GTEXTypeValues = []GTEXType{
	GTEXTypeClassic, GTEXTypeClassicConsole,
	GTEXTypeCubemap, GTEXTypeCubemapConsole,
	GTEXTypeVolumemap,
}

// IsKnownGTEXType informa se o tipo é um dos reconhecidos pelo IMGBlibrary.
func (t GTEXType) IsKnownGTEXType() bool {
	for _, v := range GTEXTypeValues {
		if t == v {
			return true
		}
	}
	return false
}

// GTEXFormat representa o formato de pixel da textura.
type GTEXFormat byte

// Formatos GTEX relevantes para o processamento de imagem.
//
// Conforme DDSHelpers.cs, 3 e 4 são R8G8B8A8 não comprimido. O IMGBBlibrary
// usa ainda 24/25/26 para DXT1/DXT3/DXT5, mas este port adota 0/1/2 para os
// formatos DXT (ver GTEXFormatToDDS).
const (
	GTEXFormatRGB  GTEXFormat = 3 // R8G8B8A8 (com mipmaps)
	GTEXFormatRGBA GTEXFormat = 4 // R8G8B8A8
)

// GTEX é o header de informação de textura encontrado nos arquivos de bloco de header
// (ex: .ztr, .txb, .txbh, .vtex extraídos do WPD).
// Corresponde à struct IMGBlibrary.Support.GTEX.
type GTEX struct {
	ImageName          string     // Nome da imagem
	IsValid            bool       // Se o header GTEX é válido
	GTEXOffset         uint32     // Offset do GTEX no arquivo
	Version            byte       // Versão do formato
	UnkFlag            byte       // Flag desconhecida
	Format             GTEXFormat // Formato de pixel
	MipCount           byte       // Número de mipmaps
	UnkFlag2           byte       // Flag desconhecida 2
	Type               GTEXType   // Tipo de textura (classic/cubemap/volumemap)
	Width              uint16     // Largura da textura
	Height             uint16     // Altura da textura
	Depth              uint16     // Profundidade (para volume maps)
	MipInfoTableOffset uint32     // Offset da tabela de info de mipmaps
}

// MipInfo representa informação de um nível de mipmap.
type MipInfo struct {
	Offset uint32
	Size   uint32
}

// Magic numbers para identificação de formatos
const (
	// IMGB magic para veffs-style (hash table)
	IMGBMagicVeffs1 = 0x9248001A
	IMGBMagicVeffs2 = 0x24924924
	IMGBMagicVeffs3 = 0x04FD051C
	IMGBMagicVeffs4 = 0xAAAAAAAA

	// TRB/SEDBRES magic
	TRBMagic = 0x2053455242444553 // "SEDBRES " little-endian
)

// IMGBHeaderVeffs é o header do formato IMGB estilo veffs (hash table com 8191 entradas).
type IMGBHeaderVeffs struct {
	Magic1  uint32
	Magic2  uint32
	Magic3  uint32
	Magic4  uint32
	Entries []IMGBEntryVeffs
}

// IMGBEntryVeffs é uma entrada na hash table do IMGB veffs (16 bytes).
type IMGBEntryVeffs struct {
	Val1   uint32
	Val2   uint32
	Offset uint32
	Size   int32 // Signed, -1 = unused
}

// TRBHeader é o header do formato TRB (SEDBRES).
type TRBHeader struct {
	Magic       uint64 // "SEDBRES "
	Val1        uint32 // 4000
	Val2        uint32 // 0x400200
	FileSize    uint32 // Tamanho total do arquivo
	Reserved1   uint32
	Reserved2   uint32
	Reserved3   uint32
	FirstOffset uint32  // Offset do primeiro dado (geralmente 0x2F = 47)
	DataOffset  uint32  // Offset dos dados (0x198000)
	_           uint32  // Padding
	Name        [8]byte // "brt" + zeros
}

// TRBEntry é uma entrada na tabela do TRB (16 bytes).
type TRBEntry struct {
	Index    uint32 // Índice sequencial
	Offset1  uint32 // Offset 1
	Offset2  uint32 // Offset 2
	NameHash uint32 // Hash do nome (ou 0x02000000 para entradas genéricas)
}

// IMGBFormat identifica o formato do arquivo IMGB.
type IMGBFormat string

const (
	FormatUnknown   IMGBFormat = "unknown"
	FormatTRB       IMGBFormat = "trb"
	FormatIMGBVeffs IMGBFormat = "imgb_veffs"
	FormatIMGBc001  IMGBFormat = "imgb_c001"
)

// ReadGTEX lê um header GTEX de um arquivo (ex: .vtex, .txb, .txbh extraído do WPD).
func ReadGTEX(path string) (*GTEX, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	g, err := ParseGTEX(data)
	if err != nil {
		return nil, err
	}
	// Definir ImageName a partir do nome do arquivo (sem extensão)
	g.ImageName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return g, nil
}

// GTEXMagic é o magic "GTEX" em little-endian.
const GTEXMagic = 0x58455447

// ParseGTEX interpreta os bytes de um header GTEX.
func ParseGTEX(data []byte) (*GTEX, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("dados GTEX muito pequenos: %d bytes (mínimo 32)", len(data))
	}

	// Primeiro, procurar pelo magic "GTEX" (0x58455447)
	for offset := 0; offset+4 <= len(data); offset++ {
		magic := binary.LittleEndian.Uint32(data[offset : offset+4])
		if magic == GTEXMagic {
			// Encontrado magic GTEX, o header começa 4 bytes depois
			headerOffset := offset + 4
			if headerOffset+16 <= len(data) {
				g := &GTEX{}
				g.Version = data[headerOffset+0]
				g.UnkFlag = data[headerOffset+1]
				g.Format = GTEXFormat(data[headerOffset+2])
				g.MipCount = data[headerOffset+3]
				g.UnkFlag2 = data[headerOffset+4]
				g.Type = GTEXType(data[headerOffset+5])
				g.Width = binary.LittleEndian.Uint16(data[headerOffset+6 : headerOffset+8])
				g.Height = binary.LittleEndian.Uint16(data[headerOffset+8 : headerOffset+10])
				g.Depth = binary.LittleEndian.Uint16(data[headerOffset+10 : headerOffset+12])
				g.MipInfoTableOffset = binary.LittleEndian.Uint32(data[headerOffset+12 : headerOffset+16])

				// Validação básica (sem verificar MipInfoTableOffset já que pode estar em outro arquivo)
				if g.Version > 0 && g.Version <= 10 &&
					g.MipCount > 0 && g.MipCount <= 16 &&
					g.Width > 0 && g.Width <= 4096 &&
					g.Height > 0 && g.Height <= 4096 &&
					g.Type.IsKnownGTEXType() {
					g.GTEXOffset = uint32(headerOffset)
					g.IsValid = true
					return g, nil
				}
			}
		}
	}

	// Fallback: procurar pelo padrão de header GTEX válido (sem magic)
	for offset := 0; offset+32 <= len(data); offset += 4 {
		g := &GTEX{}
		g.Version = data[offset+0]
		g.UnkFlag = data[offset+1]
		g.Format = GTEXFormat(data[offset+2])
		g.MipCount = data[offset+3]
		g.UnkFlag2 = data[offset+4]
		g.Type = GTEXType(data[offset+5])
		g.Width = binary.LittleEndian.Uint16(data[offset+6 : offset+8])
		g.Height = binary.LittleEndian.Uint16(data[offset+8 : offset+10])
		g.Depth = binary.LittleEndian.Uint16(data[offset+10 : offset+12])
		g.MipInfoTableOffset = binary.LittleEndian.Uint32(data[offset+12 : offset+16])

		// Validação básica
		if g.IsValidGTEX() {
			g.GTEXOffset = uint32(offset)
			g.IsValid = true
			return g, nil
		}
	}

	return nil, fmt.Errorf("header GTEX válido não encontrado no arquivo")
}

// IsValidGTEX verifica se os valores do GTEX parecem válidos.
func (g *GTEX) IsValidGTEX() bool {
	if g.Version == 0 || g.Version > 10 {
		return false
	}
	if g.MipCount == 0 || g.MipCount > 16 {
		return false
	}
	if g.Width == 0 || g.Width > 4096 {
		return false
	}
	if g.Height == 0 || g.Height > 4096 {
		return false
	}
	if !g.Type.IsKnownGTEXType() {
		return false
	}
	return true
}

// ReadTRB lê um arquivo TRB (SEDBRES).
func ReadTRB(path string) (*TRBHeader, []TRBEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return ParseTRB(data)
}

// ParseTRB interpreta um arquivo TRB.
func ParseTRB(data []byte) (*TRBHeader, []TRBEntry, error) {
	if len(data) < 48 {
		return nil, nil, fmt.Errorf("arquivo TRB muito pequeno: %d bytes", len(data))
	}

	magic := binary.LittleEndian.Uint64(data[0:8])
	if magic != TRBMagic {
		return nil, nil, fmt.Errorf("magic TRB inválido: 0x%016X (esperado 0x%016X)", magic, TRBMagic)
	}

	h := &TRBHeader{
		Magic:       magic,
		Val1:        binary.LittleEndian.Uint32(data[8:12]),
		Val2:        binary.LittleEndian.Uint32(data[12:16]),
		FileSize:    binary.LittleEndian.Uint32(data[16:20]),
		Reserved1:   binary.LittleEndian.Uint32(data[20:24]),
		Reserved2:   binary.LittleEndian.Uint32(data[24:28]),
		Reserved3:   binary.LittleEndian.Uint32(data[28:32]),
		FirstOffset: binary.LittleEndian.Uint32(data[32:36]),
		DataOffset:  binary.LittleEndian.Uint32(data[36:40]),
	}
	copy(h.Name[:], data[40:48])

	// Tabela de entradas começa no offset 0x30 (48)
	tableStart := 48
	var entries []TRBEntry

	for i := tableStart; i+16 <= len(data); i += 16 {
		e := TRBEntry{
			Index:    binary.LittleEndian.Uint32(data[i : i+4]),
			Offset1:  binary.LittleEndian.Uint32(data[i+4 : i+8]),
			Offset2:  binary.LittleEndian.Uint32(data[i+8 : i+12]),
			NameHash: binary.LittleEndian.Uint32(data[i+12 : i+16]),
		}
		entries = append(entries, e)

		// Parar se encontrar entrada zerada (fim da tabela)
		if e.Index == 0 && e.Offset1 == 0 && e.Offset2 == 0 && e.NameHash == 0 {
			break
		}
	}

	return h, entries, nil
}

// ReadIMGBVeffs lê um arquivo IMGB no formato veffs (hash table).
func ReadIMGBVeffs(path string) (*IMGBHeaderVeffs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseIMGBVeffs(data)
}

// ParseIMGBVeffs interpreta um arquivo IMGB estilo veffs.
func ParseIMGBVeffs(data []byte) (*IMGBHeaderVeffs, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("arquivo IMGB veffs muito pequeno: %d bytes", len(data))
	}

	h := &IMGBHeaderVeffs{
		Magic1: binary.LittleEndian.Uint32(data[0:4]),
		Magic2: binary.LittleEndian.Uint32(data[4:8]),
		Magic3: binary.LittleEndian.Uint32(data[8:12]),
		Magic4: binary.LittleEndian.Uint32(data[12:16]),
	}

	// Verificar magic
	if h.Magic1 != IMGBMagicVeffs1 || h.Magic2 != IMGBMagicVeffs2 {
		return nil, fmt.Errorf("magic IMGB veffs inválido: 0x%08X 0x%08X (esperado 0x%08X 0x%08X)",
			h.Magic1, h.Magic2, IMGBMagicVeffs1, IMGBMagicVeffs2)
	}

	// Entradas começam no offset 16, cada uma 16 bytes
	entriesCount := (len(data) - 16) / 16
	h.Entries = make([]IMGBEntryVeffs, 0, entriesCount)

	for i := 16; i+16 <= len(data); i += 16 {
		e := IMGBEntryVeffs{
			Val1:   binary.LittleEndian.Uint32(data[i : i+4]),
			Val2:   binary.LittleEndian.Uint32(data[i+4 : i+8]),
			Offset: binary.LittleEndian.Uint32(data[i+8 : i+12]),
			Size:   int32(binary.LittleEndian.Uint32(data[i+12 : i+16])),
		}
		h.Entries = append(h.Entries, e)
	}

	return h, nil
}

// DetectFormat detecta o formato do arquivo baseado no magic.
func DetectFormat(path string) (IMGBFormat, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FormatUnknown, err
	}
	if len(data) < 8 {
		return FormatUnknown, nil
	}

	// Check TRB magic
	if len(data) >= 8 && binary.LittleEndian.Uint64(data[0:8]) == TRBMagic {
		return FormatTRB, nil
	}

	// Check IMGB veffs magic
	if len(data) >= 16 {
		m1 := binary.LittleEndian.Uint32(data[0:4])
		m2 := binary.LittleEndian.Uint32(data[4:8])
		if m1 == IMGBMagicVeffs1 && m2 == IMGBMagicVeffs2 {
			return FormatIMGBVeffs, nil
		}
	}

	// Check c001 IMGB pattern (01 00 00 00 55 55 55 55)
	if len(data) >= 8 {
		v1 := binary.LittleEndian.Uint32(data[0:4])
		v2 := binary.LittleEndian.Uint32(data[4:8])
		if v1 == 1 && v2 == 0x55555555 {
			return FormatIMGBc001, nil
		}
	}

	return FormatUnknown, nil
}

// CopyStreamTo copia dados de um stream para outro (equivalente a StreamHelpers.CopyStreamTo).
func CopyStreamTo(dst io.Writer, src io.Reader, size int64) (int64, error) {
	return io.CopyN(dst, src, size)
}

// PadStream preenche um stream com zeros até o alinhamento (equivalente a StreamHelpers.PadStream).
func PadStream(w io.Writer, currentPos, alignment int64) (int64, error) {
	if alignment <= 0 {
		return 0, nil
	}
	remainder := currentPos % alignment
	if remainder == 0 {
		return 0, nil
	}
	pad := alignment - remainder
	_, err := w.Write(make([]byte, pad))
	return pad, err
}

// GetGTEXChunkOffset obtém o offset do chunk GTEX no arquivo (equivalente a SharedMethods.GetGTEXChunkOffset).
func GetGTEXChunkOffset(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1, err
	}

	// Procurar por header GTEX válido
	for offset := 0; offset+32 <= len(data); offset += 4 {
		if offset+16 <= len(data) {
			width := binary.LittleEndian.Uint16(data[offset+6 : offset+8])
			height := binary.LittleEndian.Uint16(data[offset+8 : offset+10])
			mipCount := data[offset+3]
			if width > 0 && width <= 4096 && height > 0 && height <= 4096 && mipCount > 0 && mipCount <= 16 {
				return offset, nil
			}
		}
	}
	return -1, fmt.Errorf("GTEX não encontrado em %s", path)
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

// GTEXFormatToDDS mapeia formato GTEX para FourCC DDS.
var GTEXFormatToDDS = map[GTEXFormat]uint32{
	0: 0x31545844, // DXT1
	1: 0x33545844, // DXT3
	2: 0x35545844, // DXT5
	3: 0x00000000, // RGB
	4: 0x00000000, // RGBA
}

// DDSHeader representa o header de um arquivo DDS.
type DDSHeader struct {
	DwSize              uint32
	DwFlags             uint32
	DwHeight            uint32
	DwWidth             uint32
	DwPitchOrLinearSize uint32
	DwDepth             uint32
	DwMipMapCount       uint32
	DwReserved1         [11]uint32
	DwSize2             uint32
	DwFlags2            uint32
	DwFourCC            uint32
	DwRGBBitCount       uint32
	DwRBitMask          uint32
	DwGBitMask          uint32
	DwBBitMask          uint32
	DwABitMask          uint32
	DwCaps1             uint32
	DwCaps2             uint32
	DwCaps3             uint32
	DwCaps4             uint32
	DwReserved2         uint32
}

// GetDDSHeader gera um header DDS a partir de um GTEX (equivalente a DDSHelpers.GetDDSHeader).
func GetDDSHeader(g *GTEX) ([]byte, error) {
	fourCC := GTEXFormatToDDS[g.Format]
	isCompressed := fourCC != 0

	h := DDSHeader{
		DwSize:        124,
		DwFlags:       0x00021007, // DDSD_CAPS | DDSD_HEIGHT | DDSD_WIDTH | DDSD_PIXELFORMAT | DDSD_LINEARSIZE
		DwHeight:      uint32(g.Height),
		DwWidth:       uint32(g.Width),
		DwDepth:       0, // preenchido apenas para volume maps
		DwMipMapCount: uint32(g.MipCount),
		DwFlags2:      32,
		DwSize2:       32,
	}

	if isCompressed {
		h.DwFourCC = fourCC
		h.DwPitchOrLinearSize = ((uint32(g.Width) + 3) / 4) * ((uint32(g.Height) + 3) / 4) * 8
		if fourCC == 0x31545844 { // DXT1
			h.DwPitchOrLinearSize /= 2
		}
		h.DwCaps1 = 0x00001000 | 0x00400000 // DDSCAPS_TEXTURE | DDSCAPS_MIPMAP
	} else {
		h.DwRGBBitCount = 32
		h.DwRBitMask = 0x00FF0000
		h.DwGBitMask = 0x0000FF00
		h.DwBBitMask = 0x000000FF
		h.DwABitMask = 0xFF000000
		h.DwPitchOrLinearSize = uint32(g.Width) * 4
		h.DwCaps1 = 0x00001000
	}

	if g.MipCount > 1 {
		h.DwFlags |= 0x00020000 // DDSD_MIPMAPCOUNT
		h.DwCaps1 |= 0x00400000 // DDSCAPS_MIPMAP
	}

	if g.Type.IsCubemap() {
		h.DwCaps2 = 0x00000200 | 0x00000400 | 0x00000800 | 0x00001000 | 0x00002000 | 0x00004000 | 0x00008000 // All cube faces
		h.DwCaps1 |= 0x00000008                                                                              // DDSCAPS_COMPLEX
	}

	if g.Type.IsVolumemap() {
		h.DwFlags |= 0x00800000 // DDSD_DEPTH
		h.DwCaps2 = 0x00200000  // DDSCAPS2_VOLUME
		h.DwCaps1 |= 0x00000008 // DDSCAPS_COMPLEX
		h.DwDepth = uint32(g.Depth)
	}

	// Serializar
	buf := make([]byte, 148)
	binary.LittleEndian.PutUint32(buf[0:4], 0x20534444) // "DDS "
	binary.LittleEndian.PutUint32(buf[4:8], h.DwSize)
	binary.LittleEndian.PutUint32(buf[8:12], h.DwFlags)
	binary.LittleEndian.PutUint32(buf[12:16], h.DwHeight)
	binary.LittleEndian.PutUint32(buf[16:20], h.DwWidth)
	binary.LittleEndian.PutUint32(buf[20:24], h.DwPitchOrLinearSize)
	binary.LittleEndian.PutUint32(buf[24:28], h.DwDepth)
	binary.LittleEndian.PutUint32(buf[28:32], h.DwMipMapCount)
	// Reserved1 (44 bytes) already zero
	binary.LittleEndian.PutUint32(buf[76:80], h.DwSize2)
	binary.LittleEndian.PutUint32(buf[80:84], h.DwFlags2)
	binary.LittleEndian.PutUint32(buf[84:88], h.DwFourCC)
	binary.LittleEndian.PutUint32(buf[88:92], h.DwRGBBitCount)
	binary.LittleEndian.PutUint32(buf[92:96], h.DwRBitMask)
	binary.LittleEndian.PutUint32(buf[96:100], h.DwGBitMask)
	binary.LittleEndian.PutUint32(buf[100:104], h.DwBBitMask)
	binary.LittleEndian.PutUint32(buf[104:108], h.DwABitMask)
	binary.LittleEndian.PutUint32(buf[108:112], h.DwCaps1)
	binary.LittleEndian.PutUint32(buf[112:116], h.DwCaps2)
	binary.LittleEndian.PutUint32(buf[116:120], h.DwCaps3)
	binary.LittleEndian.PutUint32(buf[120:124], h.DwCaps4)
	binary.LittleEndian.PutUint32(buf[124:128], h.DwReserved2)

	return buf, nil
}

// ReadMipInfoTable lê a tabela de informação de mipmaps do arquivo de header.
func ReadMipInfoTable(headerBlockFile string, gtex *GTEX) ([]MipInfo, error) {
	data, err := os.ReadFile(headerBlockFile)
	if err != nil {
		return nil, err
	}

	offset := int(gtex.MipInfoTableOffset)
	if offset < 0 || offset+8 > len(data) {
		return nil, fmt.Errorf("offset de tabela de mipmaps inválido: 0x%X", gtex.MipInfoTableOffset)
	}

	var mipTable []MipInfo
	for i := 0; i < int(gtex.MipCount); i++ {
		entryOffset := offset + i*8
		if entryOffset+8 > len(data) {
			break
		}
		mipTable = append(mipTable, MipInfo{
			Offset: binary.LittleEndian.Uint32(data[entryOffset : entryOffset+4]),
			Size:   binary.LittleEndian.Uint32(data[entryOffset+4 : entryOffset+8]),
		})
	}

	return mipTable, nil
}

// ProcessPS3ImageData processa os dados de um mip de imagem PS3.
//
// Porte fiel de IMGBlibrary.Unpack.PS3UnpackHelpers.ProcessPS3ImageData.
//
// Para imagens PS3 nos formatos 3/4 (R8G8B8A8 não comprimido) a ordem dos
// canais é corrigida de ARGB para BGRA (o que o DDS espera). Quando a imagem
// está "swizzled" (formato 4 + tipo 4, isto é, classic console), os pixels
// são antes reorganizados da ordem Morton (Z-order) para a ordem linear.
// Qualquer outro formato (por exemplo DXT) é devolvido sem alterações.
func ProcessPS3ImageData(gtex *GTEX, data []byte) []byte {
	// Formato 4 + tipo 4: imagem swizzled → des-swizzle e correção de cor.
	if gtex.Format == GTEXFormatRGBA && gtex.Type == GTEXTypeClassicConsole {
		return ColorAsBGRA(MortonUnswizzle(gtex, data))
	}

	// Formato 3 ou 4: apenas correção de cor (DDS espera BGRA).
	if gtex.Format == GTEXFormatRGB || gtex.Format == GTEXFormatRGBA {
		return ColorAsBGRA(data)
	}

	return data
}

// MortonUnswizzle reorganiza pixels da ordem Morton (Z-order) para linear.
//
// Porte fiel de PS3UnpackHelpers.MortonUnswizzle. O buffer de saída tem
// Width*Height*4 bytes (as dimensões do mip 0, como no original); leituras
// além dos dados disponíveis são ignoradas para não causar pânico.
func MortonUnswizzle(gtex *GTEX, swizzled []byte) []byte {
	width := int(gtex.Width)
	height := int(gtex.Height)

	unswizzled := make([]byte, width*height*4)

	for m := 0; m < width*height; m++ {
		readPos := m * 4
		if readPos+4 > len(swizzled) {
			break
		}

		val1 := 0
		val2 := 0
		val3 := 1
		val4 := 1
		val5 := m
		val6 := width
		val7 := height

		for val6 > 1 || val7 > 1 {
			if val6 > 1 {
				val1 += val4 * (val5 & 1)
				val5 >>= 1
				val4 *= 2
				val6 >>= 1
			}
			if val7 > 1 {
				val2 += val3 * (val5 & 1)
				val5 >>= 1
				val3 *= 2
				val7 >>= 1
			}
		}

		pixelOffset := (val2*width + val1) * 4
		if pixelOffset+4 > len(unswizzled) {
			continue
		}

		copy(unswizzled[pixelOffset:pixelOffset+4], swizzled[readPos:readPos+4])
	}

	return unswizzled
}

// ColorAsBGRA converte pixels de ARGB para BGRA.
//
// Porte de PS3UnpackHelpers.ColorAsBGRA: cada pixel de 4 bytes (A,R,G,B) é
// reescrito como (B,G,R,A). O original itera byte a byte, mas o efeito final
// é exatamente esta troca por pixel.
func ColorAsBGRA(data []byte) []byte {
	corrected := make([]byte, len(data))
	for i := 0; i+4 <= len(data); i += 4 {
		corrected[i+0] = data[i+3] // B
		corrected[i+1] = data[i+2] // G
		corrected[i+2] = data[i+1] // R
		corrected[i+3] = data[i+0] // A
	}
	return corrected
}
