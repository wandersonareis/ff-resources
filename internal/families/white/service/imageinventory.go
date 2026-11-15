package service

import (
	"os"
	"path/filepath"
	"strings"

	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
)

func LowerRel(p string) string {
	p = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"))
	return strings.TrimPrefix(p, "/")
}

func ImageDataKind(name string) (kind string, container bool) {
	switch FilepathExt(name) {
	case ".dds", ".png":
		return "file", false
	case ".trb":
		return "trb", true
	case ".xgr", ".xfv", ".bin", ".wpd", ".wdb", ".xwb", ".xwp", ".grs":
		return "wpd", true
	}
	return "", false
}

// ImageDataEntries classifica os arquivos extraídos que são imagens ou
// containers de imagem. O .imgb pareado é a metade binária do container e não
// vira folha própria.
func ImageDataEntries(part string, all map[string]bool) []DataEntry {
	p := parts.Get(part)
	hasPair := func(pair string) bool {
		return all[LowerRel(pair)]
	}

	out := make([]DataEntry, 0, 64)
	dataDir, _ := config.PartRoot("data", part)
	for rel := range all {
		rel = LowerRel(rel)
		// Face do GUI: .imgb sem tabela externa, DXT1 128x256 implícito.
		if IsResidentFaceImage(rel) {
			disk := filepath.Join(dataDir, filepath.FromSlash(rel))
			if st, serr := os.Stat(disk); serr == nil && st.Size() > 8 {
				out = append(out, DataEntry{
					ID:       rel,
					Label:    filepath.Base(rel),
					Type:     p.EntryType(rel),
					Image:    true,
					ImageRef: EncodeImageRef(ImageRef{Source: "data", Path: rel, Platform: ImagePlatformOf(rel)}),
				})
			}
			continue
		}
		kind, container := ImageDataKind(rel)
		if kind == "" {
			continue
		}
		if !container {
			out = append(out, DataEntry{
				ID:       rel,
				Label:    filepath.Base(rel),
				Type:     p.EntryType(rel),
				Image:    true,
				ImageRef: EncodeImageRef(ImageRef{Source: "data", Path: rel, Platform: ImagePlatformOf(rel)}),
			})
			continue
		}
		pair := ImagePairPath(rel)
		// .bin só entra como imagem quando tem o .imgb par (os .bin de texto
		// do part3 não têm). .xgr/.xfv/.trb entram mesmo sem par: a expansão
		// mostra os registros e a prévia avisa se o payload falta.
		if !hasPair(pair) {
			switch FilepathExt(rel) {
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
			ImageRef:       EncodeImageRef(ImageRef{Source: "data", Path: rel, Payload: pair, Platform: ImagePlatformOf(rel)}),
			Pair:           pair,
		})
	}
	return out
}

// ImageFileExistsOnDisk é usado nos testes/plano de delete.
func ImageFileExistsOnDisk(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
