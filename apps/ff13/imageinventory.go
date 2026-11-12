package main

import (
	"os"
	"path/filepath"
	"strings"

	"ffresources/apps/ff13/parts"
)

func lowerRel(p string) string {
	p = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"))
	return strings.TrimPrefix(p, "/")
}

func imageDataKind(name string) (kind string, container bool) {
	switch filepathExt(name) {
	case ".dds", ".png":
		return "file", false
	case ".trb":
		return "trb", true
	case ".xgr", ".xfv", ".bin", ".wpd", ".wdb", ".xwb", ".xwp", ".grs":
		return "wpd", true
	}
	return "", false
}

// imageDataEntries classifica os arquivos extraídos que são imagens ou
// containers de imagem. O .imgb pareado é a metade binária do container e não
// vira folha própria.
func imageDataEntries(part string, all map[string]bool) []DataEntry {
	p := parts.Get(part)
	hasPair := func(pair string) bool {
		return all[lowerRel(pair)]
	}

	out := make([]DataEntry, 0, 64)
	dataDir, _ := partRoot("data", part)
	for rel := range all {
		rel = lowerRel(rel)
		// Face do GUI: .imgb sem tabela externa, DXT1 128x256 implícito.
		if isResidentFaceImage(rel) {
			disk := filepath.Join(dataDir, filepath.FromSlash(rel))
			if st, serr := os.Stat(disk); serr == nil && st.Size() > 8 {
				out = append(out, DataEntry{
					ID:       rel,
					Label:    filepath.Base(rel),
					Type:     p.EntryType(rel),
					Image:    true,
					ImageRef: encodeImageRef(imageRef{Source: "data", Path: rel, Platform: imagePlatformOf(rel)}),
				})
			}
			continue
		}
		kind, container := imageDataKind(rel)
		if kind == "" {
			continue
		}
		if !container {
			out = append(out, DataEntry{
				ID:       rel,
				Label:    filepath.Base(rel),
				Type:     p.EntryType(rel),
				Image:    true,
				ImageRef: encodeImageRef(imageRef{Source: "data", Path: rel, Platform: imagePlatformOf(rel)}),
			})
			continue
		}
		pair := imagePairPath(rel)
		// .bin só entra como imagem quando tem o .imgb par (os .bin de texto
		// do part3 não têm). .xgr/.xfv/.trb entram mesmo sem par: a expansão
		// mostra os registros e a prévia avisa se o payload falta.
		if !hasPair(pair) {
			switch filepathExt(rel) {
			case ".xgr", ".xfv", ".trb":
			default:
				continue
			}
		}
		out = append(out, DataEntry{
			ID:             rel,
			Label:          filepath.Base(rel),
			Type:           p.EntryType(rel),
			Container:      kind,
			Image:          true,
			ImageContainer: true,
			ImageRef:       encodeImageRef(imageRef{Source: "data", Path: rel, Payload: pair, Platform: imagePlatformOf(rel)}),
			Pair:           pair,
		})
	}
	return out
}

// imageFileExistsOnDisk é usado nos testes/plano de delete.
func imageFileExistsOnDisk(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
