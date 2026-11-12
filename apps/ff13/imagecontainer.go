package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"ffresources/apps/ff13/common"
	"ffresources/internal/families/white/codecs/imgb/core"
	"ffresources/internal/families/white/codecs/wpd"
)

func imageDataURL(mime string, b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func isImageRecordExt(ext string) bool {
	switch strings.ToLower(ext) {
	case "txbh", "txb", "vtex":
		return true
	}
	return false
}

func isContainerExt(name string) bool {
	switch strings.ToLower(filepathExt(name)) {
	case ".xgr", ".xfv", ".trb", ".bin", ".wpd", ".wdb", ".xwb", ".xwp", ".grs":
		return true
	}
	return false
}

func filepathExt(name string) string {
	name = strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i:]
	}
	return ""
}

// containerKind detecta pelo magic (WPD ou TRB). A extensão serve de dica.
func containerKind(name string, raw []byte) string {
	if len(raw) >= 4 && string(raw[:4]) == "WPD\x00" {
		return "wpd"
	}
	if len(raw) >= 8 && string(raw[:8]) == "SEDBRES " {
		return "trb"
	}
	if len(raw) >= 4 && string(raw[:4]) == "SEDB" {
		return "trb"
	}
	if strings.EqualFold(filepathExt(name), ".trb") {
		return "trb"
	}
	if isContainerExt(name) {
		return "wpd"
	}
	return ""
}

// imageRecordSection extrai o bloco de header (SectionHeader + GTEX) de um
// registro WPD/TRB.
func imageRecordSection(kind string, raw []byte, entry string) ([]byte, error) {
	switch kind {
	case "wpd":
		f, err := wpd.Read(raw)
		if err != nil {
			return nil, err
		}
		name, ext := splitImageRecordKey(entry)
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

func splitImageRecordKey(key string) (name, ext string) {
	key = strings.TrimPrefix(key, "rec:")
	if i := strings.LastIndex(key, "."); i >= 0 {
		return key[:i], key[i+1:]
	}
	return key, ""
}

// ListImageContainer lista os filhos de uma tabela WPD/TRB. Imagens ganham
// ImageRef; os demais registros viram arquivos comuns.
func (FF13Service) ListImageContainer(part, ref string) ([]common.ImageNode, error) {
	r, err := decodeImageRef(ref)
	if err != nil {
		return nil, err
	}
	if r.Entry != "" {
		return nil, fmt.Errorf("imagem: nó não é um container")
	}
	f, err := resolveImageFile(normalizePart(part), r)
	if err != nil {
		return nil, err
	}
	raw, err := f.read()
	if err != nil {
		return nil, err
	}
	kind := containerKind(r.Path, raw)
	if kind == "" {
		return nil, fmt.Errorf("imagem: formato de container desconhecido (%s)", r.Path)
	}
	pairRel := r.Payload
	if pairRel == "" {
		pairRel = imagePairPath(f.Rel)
	}

	nodes := make([]common.ImageNode, 0, 32)
	switch kind {
	case "wpd":
		wf, err := wpd.Read(raw)
		if err != nil {
			return nil, err
		}
		for _, rec := range wf.Header.Records {
			entry := fmt.Sprintf("rec:%s.%s", rec.Name, rec.Extension)
			node := common.ImageNode{
				ID:   encodeImageRef(imageRef{Source: r.Source, Root: r.Root, Path: r.Path, Entry: entry, Payload: pairRel, Platform: imagePlatformOf(r.Path)}),
				Name: rec.Name + "." + rec.Extension,
				Path: entry,
				Size: rec.Size,
				Kind: "file",
				Pair: pairRel,
			}
			if isImageRecordExt(rec.Extension) {
				if sec, ok := wf.ExtractRecord(rec.Name, rec.Extension); ok {
					if _, gerr := imgbcore.SectionGTEX(sec); gerr == nil {
						node.Kind = "image"
						node.ImageRef = node.ID
						node.Container = "wpd"
					}
				}
			}
			nodes = append(nodes, node)
		}
	case "trb":
		h, entries, err := imgbcore.ParseTRB(raw)
		if err != nil {
			return nil, err
		}
		for i, e := range entries {
			sec, serr := h.SectionData(raw, e)
			name := fmt.Sprintf("%d", i)
			if serr == nil {
				if sh, herr := imgbcore.ParseSectionHeader(sec); herr == nil {
					name = fmt.Sprintf("%d.%s", i, sh.TypeName())
				}
			}
			entry := fmt.Sprintf("trb:%d", i)
			node := common.ImageNode{
				ID:   encodeImageRef(imageRef{Source: r.Source, Root: r.Root, Path: r.Path, Entry: entry, Payload: pairRel, Platform: imagePlatformOf(r.Path)}),
				Name: name,
				Path: entry,
				Size: e.Length,
				Kind: "file",
				Pair: pairRel,
			}
			if serr == nil {
				if _, gerr := imgbcore.SectionGTEX(sec); gerr == nil {
					node.Kind = "image"
					node.ImageRef = node.ID
					node.Container = "trb"
				}
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// imageSection resolve a seção da imagem e o arquivo de conteúdo.
func imageSection(part string, r imageRef) ([]byte, imageFile, error) {
	table, err := resolveImageFile(part, r)
	if err != nil {
		return nil, imageFile{}, err
	}
	raw, err := table.read()
	if err != nil {
		return nil, imageFile{}, err
	}
	kind := containerKind(r.Path, raw)
	if kind == "" {
		return nil, imageFile{}, fmt.Errorf("imagem: container desconhecido (%s)", r.Path)
	}
	sec, err := imageRecordSection(kind, raw, r.Entry)
	if err != nil {
		return nil, imageFile{}, err
	}
	return sec, table, nil
}

// decodeImageDDS devolve o DDS completo/prévia e os metadados.
func decodeImageDDS(part string, r imageRef, allMips bool) ([]byte, imgbcore.ImageInfo, error) {
	var info imgbcore.ImageInfo
	sec, table, err := imageSection(part, r)
	if err != nil {
		return nil, info, err
	}
	payload, err := resolveImagePayload(part, r, table.Rel)
	if err != nil {
		return nil, info, err
	}
	content, err := payload.read()
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

// imageDiagnostics é usado pela UI para decidir se o nó é imagem sem decodificar.
func imageDiagnostics(part string, r imageRef) (imgbcore.ImageInfo, error) {
	sec, _, err := imageSection(part, r)
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
