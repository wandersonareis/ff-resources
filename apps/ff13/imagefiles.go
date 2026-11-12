package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"ffresources/apps/ff13/parts"
	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
)

// imageRef é a identidade estável de uma imagem/container. É serializada em
// base64url para o frontend devolver ao backend sem conhecer o layout.
type imageRef struct {
	Source   string `json:"s"`           // data | filelist
	Root     string `json:"r,omitempty"` // raiz do filelist
	Path     string `json:"p"`           // tabela/arquivo (caminho interno ou relativo a data/partN)
	Entry    string `json:"e,omitempty"` // registro no container ("" = container/arquivo direto)
	Payload  string `json:"b,omitempty"` // .imgb pareado, quando conhecido
	Platform string `json:"f,omitempty"` // win32 | ps3 | x360
}

func encodeImageRef(r imageRef) string {
	b, _ := json.Marshal(r)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeImageRef(s string) (imageRef, error) {
	var r imageRef
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return r, fmt.Errorf("imagem: referência inválida")
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, fmt.Errorf("imagem: referência inválida")
	}
	if r.Path == "" {
		return r, fmt.Errorf("imagem: referência sem caminho")
	}
	if r.Source == "" {
		r.Source = "data"
	}
	return r, nil
}

var languageSuffix = regexp.MustCompile(`_(us|jp|uk|it|gr|fr|sp|ru|kr|ck|ch|cn|de|es)$`)

// imagePairPath devolve o .imgb pareado com uma tabela. Cobre:
//
//	title0.win32.xgr        -> title0.win32.imgb
//	veffs.us.win32.xfv      -> veffs.us.win32.imgb
//	c001.win32.trb          -> c001.win32.imgb
//	b00_us.white.win32.bin  -> b00.white.win32.imgb  (o _us de idioma sai)
func imagePairPath(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	ext := filepath.Ext(p)
	if ext == "" {
		return p + ".imgb"
	}
	base := strings.TrimSuffix(p, ext)
	if i := strings.Index(strings.ToLower(base), ".white."); i >= 0 {
		before := base[:i]
		after := base[i:]
		before = languageSuffix.ReplaceAllString(before, "")
		base = before + after
	}
	return base + ".imgb"
}

// imagePlatformOf extrai a plataforma do nome (win32/ps3/x360).
func imagePlatformOf(p string) string {
	n := strings.ToLower(p)
	switch {
	case strings.Contains(n, ".ps3."):
		return "ps3"
	case strings.Contains(n, ".x360."):
		return "x360"
	case strings.Contains(n, ".win32."):
		return "win32"
	default:
		return "win32"
	}
}

// imageFile é uma fonte de bytes: disco ou white bin.
type imageFile struct {
	Part  string
	Rel   string
	Disk  string
	Bin   string
	Coll  *fl.Collection
	VPath string
}

func (f imageFile) exists() bool {
	if f.Disk != "" {
		if st, err := os.Stat(f.Disk); err == nil && !st.IsDir() {
			return true
		}
	}
	return f.Bin != "" && f.Coll != nil
}

func (f imageFile) read() ([]byte, error) {
	if f.Disk != "" {
		return os.ReadFile(f.Disk)
	}
	if f.Bin != "" && f.Coll != nil {
		return whitebin.ExtractBytes(f.Bin, f.VPath, f.Coll)
	}
	return nil, fmt.Errorf("imagem: fonte vazia (%s)", f.Rel)
}

// imageFileIndex mapeia o caminho relativo a data/partN -> fonte (white bin).
// É construído uma vez por part e reutilizado nas prévias/expansões.
var (
	imageFileIndexMu    sync.Mutex
	imageFileIndexCache = map[string]map[string]imageFile{}
)

func imageFileIndexFor(p parts.Part) map[string]imageFile {
	key := p.ID() + "|" + gameRoot(p.ID())
	imageFileIndexMu.Lock()
	if m, ok := imageFileIndexCache[key]; ok {
		imageFileIndexMu.Unlock()
		return m
	}
	imageFileIndexMu.Unlock()

	m := map[string]imageFile{}
	for _, idx := range discoverPart(p).indexes {
		if idx.Container == "" {
			continue
		}
		zone := zoneOfIndex(idx)
		for _, e := range indexEntries(p, idx, nil) {
			rel := strings.ToLower(filelistDataPath(zone, e.Path))
			if _, exists := m[rel]; exists {
				continue
			}
			m[rel] = imageFile{
				Part:  p.ID(),
				Rel:   rel,
				Bin:   idx.Container,
				Coll:  idx.Coll,
				VPath: e.Path,
			}
		}
	}

	imageFileIndexMu.Lock()
	imageFileIndexCache[key] = m
	imageFileIndexMu.Unlock()
	return m
}

func invalidateImageFileIndex(part string) {
	part = normalizePart(part)
	imageFileIndexMu.Lock()
	for k := range imageFileIndexCache {
		if strings.HasPrefix(k, part+"|") {
			delete(imageFileIndexCache, k)
		}
	}
	imageFileIndexMu.Unlock()
}

// resolveRelFile procura um caminho relativo a data/partN no disco (data,
// depois mods) e, por fim, no white bin.
func resolveRelFile(part, rel string) (imageFile, bool) {
	part = normalizePart(part)
	rel = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/"))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return imageFile{}, false
	}
	if dataDir, err := partRoot("data", part); err == nil {
		disk := filepath.Join(dataDir, filepath.FromSlash(rel))
		if st, serr := os.Stat(disk); serr == nil && !st.IsDir() {
			return imageFile{Part: part, Rel: rel, Disk: disk}, true
		}
	}
	if enableModsEnabled() {
		if modsDir, err := partRoot("mods", part); err == nil {
			disk := filepath.Join(modsDir, filepath.FromSlash(rel))
			if st, serr := os.Stat(disk); serr == nil && !st.IsDir() {
				return imageFile{Part: part, Rel: rel, Disk: disk}, true
			}
		}
	}
	if f, ok := imageFileIndexFor(parts.Get(part))[rel]; ok && f.exists() {
		return f, true
	}
	return imageFile{}, false
}

// resolveImageFile resolve a tabela/arquivo da ref. Para filelist usa o
// índice indicado (e cai para data/mods/white bin quando preciso).
func resolveImageFile(part string, r imageRef) (imageFile, error) {
	part = normalizePart(part)
	if r.Source == "filelist" && r.Root != "" {
		p := parts.Get(part)
		idx, zone, err := resolveFilelistIndex(p, r.Root)
		if err == nil {
			rel := filelistDataPath(zone, r.Path)
			if dataDir, derr := partRoot("data", part); derr == nil {
				disk := filepath.Join(dataDir, filepath.FromSlash(rel))
				if st, serr := os.Stat(disk); serr == nil && !st.IsDir() {
					return imageFile{Part: part, Rel: rel, Disk: disk}, nil
				}
			}
			if enableModsEnabled() {
				if modsDir, merr := partRoot("mods", part); merr == nil {
					disk := filepath.Join(modsDir, filepath.FromSlash(rel))
					if st, serr := os.Stat(disk); serr == nil && !st.IsDir() {
						return imageFile{Part: part, Rel: rel, Disk: disk}, nil
					}
				}
			}
			if idx.Container != "" {
				return imageFile{
					Part:  part,
					Rel:   rel,
					Bin:   idx.Container,
					Coll:  idx.Coll,
					VPath: r.Path,
				}, nil
			}
			if f, ok := resolveRelFile(part, rel); ok {
				return f, nil
			}
			return imageFile{}, fmt.Errorf("imagem: arquivo não encontrado no índice (%s)", r.Path)
		}
	}
	if f, ok := resolveRelFile(part, r.Path); ok {
		return f, nil
	}
	return imageFile{}, fmt.Errorf("imagem: arquivo não encontrado: %s", r.Path)
}

// resolveImagePayload resolve o .imgb pareado com a tabela.
func resolveImagePayload(part string, r imageRef, tableRel string) (imageFile, error) {
	pairRel := r.Payload
	if pairRel == "" {
		pairRel = imagePairPath(tableRel)
	}
	if f, ok := resolveRelFile(part, pairRel); ok {
		return f, nil
	}
	return imageFile{}, fmt.Errorf("imagem: binário .imgb não encontrado (%s)", pairRel)
}
