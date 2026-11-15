package formats

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"ffresources/internal/families/white/codecs/imgb/core"
	"ffresources/internal/families/white/codecs/wpd"
)

// LEITURA de containers de imagem (WPD/TRB) e conversão para DDS.

// ImageDataURL monta um data URL base64 para o frontend.
func ImageDataURL(mime string, b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

// IsImageRecordExt diz se a extensão de um registro é de imagem.
func IsImageRecordExt(ext string) bool {
	switch strings.ToLower(ext) {
	case "txbh", "txb", "vtex":
		return true
	}
	return false
}

// IsContainerExt diz se a extensão é de um container de imagem.
func IsContainerExt(name string) bool {
	switch strings.ToLower(FilepathExt(name)) {
	case ".xgr", ".xfv", ".trb", ".bin", ".wpd", ".wdb", ".xwb", ".xwp", ".grs":
		return true
	}
	return false
}

// FilepathExt devolve a extensão em minúsculas (aceita "\" e "/").
func FilepathExt(name string) string {
	name = strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i:]
	}
	return ""
}

// ContainerKind detecta pelo magic (WPD ou TRB). A extensão serve de dica.
func ContainerKind(name string, raw []byte) string {
	if len(raw) >= 4 && string(raw[:4]) == "WPD\x00" {
		return "wpd"
	}
	if len(raw) >= 8 && string(raw[:8]) == "SEDBRES " {
		return "trb"
	}
	if len(raw) >= 4 && string(raw[:4]) == "SEDB" {
		return "trb"
	}
	if strings.EqualFold(FilepathExt(name), ".trb") {
		return "trb"
	}
	if IsContainerExt(name) {
		return "wpd"
	}
	return ""
}

// ImageRecordSection extrai o bloco de header (SectionHeader + GTEX) de um
// registro WPD/TRB.
func ImageRecordSection(kind string, raw []byte, entry string) ([]byte, error) {
	switch kind {
	case "wpd":
		f, err := wpd.Read(raw)
		if err != nil {
			return nil, err
		}
		name, ext := SplitImageRecordKey(entry)
		for _, r := range f.Header.Records {
			if !strings.EqualFold(r.Name, name) || !strings.EqualFold(r.Extension, ext) {
				continue
			}
			data, ok := f.ExtractRecord(r.Name, r.Extension)
			if !ok {
				return nil, fmt.Errorf("registro %s não extraído", entry)
			}
			return data, nil
		}
		return nil, fmt.Errorf("registro %s não encontrado", entry)
	case "trb":
		h, _, err := imgbcore.ParseTRB(raw)
		if err != nil {
			return nil, err
		}
		idx := -1
		fmt.Sscanf(strings.TrimPrefix(entry, "trb:"), "%d", &idx)
		if idx < 0 || idx >= len(h.Entries) {
			return nil, fmt.Errorf("entrada TRB %d inválida", idx)
		}
		return h.SectionData(raw, h.Entries[idx])
	}
	return nil, fmt.Errorf("container desconhecido")
}

// SplitImageRecordKey separa "rec:nome.ext" em (nome, ext).
func SplitImageRecordKey(key string) (name, ext string) {
	key = strings.TrimPrefix(key, "rec:")
	if i := strings.LastIndex(key, "."); i >= 0 {
		return key[:i], key[i+1:]
	}
	return key, ""
}

// ImageSection resolve a seção da imagem e o arquivo de conteúdo.
func ImageSection(part string, r ImageRef) ([]byte, ImageFile, error) {
	table, err := ResolveImageFile(part, r)
	if err != nil {
		return nil, ImageFile{}, err
	}
	raw, err := table.Read()
	if err != nil {
		return nil, ImageFile{}, err
	}
	kind := ContainerKind(r.Path, raw)
	if kind == "" {
		return nil, ImageFile{}, fmt.Errorf("imagem: container desconhecido (%s)", r.Path)
	}
	sec, err := ImageRecordSection(kind, raw, r.Entry)
	if err != nil {
		return nil, ImageFile{}, err
	}
	return sec, table, nil
}

// DecodeImageDDS devolve o DDS completo/prévia e os metadados.
func DecodeImageDDS(part string, r ImageRef, allMips bool) ([]byte, imgbcore.ImageInfo, error) {
	var info imgbcore.ImageInfo
	sec, table, err := ImageSection(part, r)
	if err != nil {
		return nil, info, err
	}
	payload, err := ResolveImagePayload(part, r, table.Rel)
	if err != nil {
		return nil, info, err
	}
	content, err := payload.Read()
	if err != nil {
		return nil, info, err
	}
	platform := imgbcore.PlatformWin32
	switch strings.ToLower(r.Platform) {
	case "ps3":
		platform = imgbcore.PlatformPS3
	case "x360":
		platform = imgbcore.PlatformX360
	}
	return imgbcore.BuildDDS(sec, bytes.NewReader(content), platform, allMips)
}

// Diagnostics devolve formato/dimensões sem decodificar a imagem. É usado
// pela UI para decidir se o nó é imagem.
func Diagnostics(part string, r ImageRef) (imgbcore.ImageInfo, error) {
	sec, _, err := ImageSection(part, r)
	if err != nil {
		return imgbcore.ImageInfo{}, err
	}
	g, err := imgbcore.SectionGTEX(sec)
	if err != nil {
		return imgbcore.ImageInfo{}, err
	}
	return imgbcore.ImageInfo{
		Format:      imgbcore.FormatName(g.Format),
		Width:       uint32(g.Width),
		Height:      uint32(g.Height),
		MipmapCount: uint32(g.MipCount),
		Layers:      g.LayerCount(),
	}, nil
}
