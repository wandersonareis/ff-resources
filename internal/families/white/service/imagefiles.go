package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
)

// ImageRef é a identidade estável de uma imagem/container. É serializada em
// base64url para o frontend devolver ao backend sem conhecer o layout.
type ImageRef struct {
	Source   string `json:"s"`           // data | filelist
	Root     string `json:"r,omitempty"` // raiz do filelist
	Path     string `json:"p"`           // tabela/arquivo (caminho interno ou relativo a data/partN)
	Entry    string `json:"e,omitempty"` // registro no container ("" = container/arquivo direto)
	Payload  string `json:"b,omitempty"` // .imgb pareado, quando conhecido
	Platform string `json:"f,omitempty"` // win32 | ps3 | x360
}

func EncodeImageRef(r ImageRef) string {
	b, _ := json.Marshal(r)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeImageRef(s string) (ImageRef, error) {
	var r ImageRef
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

// ImagePairPath devolve o .imgb pareado com uma tabela. Cobre:
//
//	title0.win32.xgr        -> title0.win32.imgb
//	veffs.us.win32.xfv      -> veffs.us.win32.imgb
//	c001.win32.trb          -> c001.win32.imgb
//	b00_us.white.win32.bin  -> b00.white.win32.imgb  (o _us de idioma sai)
func ImagePairPath(p string) string {
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

// ImagePlatformOf extrai a plataforma do nome (win32/ps3/x360).
func ImagePlatformOf(p string) string {
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

// ImageFile é uma fonte de bytes: disco ou white bin.
type ImageFile struct {
	Part  string
	Rel   string
	Disk  string
	Bin   string
	Coll  *fl.Collection
	VPath string
}

func (f ImageFile) Exists() bool {
	if f.Disk != "" {
		if st, err := os.Stat(f.Disk); err == nil && !st.IsDir() {
			return true
		}
	}
	return f.Bin != "" && f.Coll != nil
}

func (f ImageFile) Read() ([]byte, error) {
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
	imageFileIndexCache = map[string]map[string]ImageFile{}
)

func ImageFileIndexFor(p parts.Part) map[string]ImageFile {
	key := p.ID() + "|" + config.GameRoot(p.ID())
	imageFileIndexMu.Lock()
	if m, ok := imageFileIndexCache[key]; ok {
		imageFileIndexMu.Unlock()
		return m
	}
	imageFileIndexMu.Unlock()

	m := map[string]ImageFile{}
	for _, idx := range DiscoverPart(p).Indexes {
		if idx.Container == "" {
			continue
		}
		zone := ZoneOfIndex(idx)
		for _, e := range IndexEntries(p, idx, nil) {
			rel := strings.ToLower(FilelistDataPath(zone, e.Path))
			if _, exists := m[rel]; exists {
				continue
			}
			m[rel] = ImageFile{
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

func InvalidateImageFileIndex(part string) {
	part = config.NormalizePart(part)
	imageFileIndexMu.Lock()
	for k := range imageFileIndexCache {
		if strings.HasPrefix(k, part+"|") {
			delete(imageFileIndexCache, k)
		}
	}
	imageFileIndexMu.Unlock()
}

// ResolveRelFile procura um caminho relativo a data/partN no disco e, por
// fim, no white bin. A preferência MODS/DATA segue a flag enableMods (ligada =
// tradução primeiro): é ela que faz uma imagem importada em mods/ ser servida.
func ResolveRelFile(part, rel string) (ImageFile, bool) {
	part = config.NormalizePart(part)
	rel = normalizeImageRel(rel)
	if rel == "" {
		return ImageFile{}, false
	}
	modsFirst := config.EnableModsEnabled()
	if modsFirst {
		if f, ok := resolveDiskRel(part, "mods", rel); ok {
			return f, true
		}
	}
	if f, ok := resolveDiskRel(part, "data", rel); ok {
		return f, true
	}
	if !modsFirst {
		if f, ok := resolveDiskRel(part, "mods", rel); ok {
			return f, true
		}
	}
	if f, ok := ImageFileIndexFor(parts.Get(part))[rel]; ok && f.Exists() {
		return f, true
	}
	return ImageFile{}, false
}

// resolveRelFilePristine resolve SÓ data/ (e o white bin): é o lado
// "original" usado para decidir se uma cópia em mods/ realmente difere.
func resolveRelFilePristine(part, rel string) (ImageFile, bool) {
	part = config.NormalizePart(part)
	rel = normalizeImageRel(rel)
	if rel == "" {
		return ImageFile{}, false
	}
	if f, ok := resolveDiskRel(part, "data", rel); ok {
		return f, true
	}
	if f, ok := ImageFileIndexFor(parts.Get(part))[rel]; ok && f.Exists() {
		return f, true
	}
	return ImageFile{}, false
}

// resolveDiskRel resolve um caminho interno dentro de um escopo (data/mods).
func resolveDiskRel(part, scope, rel string) (ImageFile, bool) {
	dir, err := config.PartRoot(scope, part)
	if err != nil {
		return ImageFile{}, false
	}
	disk := filepath.Join(dir, filepath.FromSlash(rel))
	if st, serr := os.Stat(disk); serr == nil && !st.IsDir() {
		return ImageFile{Part: part, Rel: rel, Disk: disk}, true
	}
	return ImageFile{}, false
}

// normalizeImageRel padroniza um caminho interno para lookup (minúsculas,
// barras "/", sem barra inicial).
func normalizeImageRel(rel string) string {
	rel = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/"))
	return strings.TrimPrefix(rel, "/")
}

// ResolveImageFile resolve a tabela/arquivo da ref. Para filelist usa o
// índice indicado (e cai para data/mods/white bin quando preciso).
func ResolveImageFile(part string, r ImageRef) (ImageFile, error) {
	part = config.NormalizePart(part)
	if r.Source == "filelist" && r.Root != "" {
		p := parts.Get(part)
		idx, zone, err := ResolveFilelistIndex(p, r.Root)
		if err == nil {
			rel := FilelistDataPath(zone, r.Path)
			// mods-first/data-first + white bin, a mesma régua de ResolveRelFile.
			if f, ok := ResolveRelFile(part, rel); ok {
				return f, nil
			}
			if idx.Container != "" {
				return ImageFile{
					Part:  part,
					Rel:   rel,
					Bin:   idx.Container,
					Coll:  idx.Coll,
					VPath: r.Path,
				}, nil
			}
			return ImageFile{}, fmt.Errorf("imagem: arquivo não encontrado no índice (%s)", r.Path)
		}
	}
	if f, ok := ResolveRelFile(part, r.Path); ok {
		return f, nil
	}
	return ImageFile{}, fmt.Errorf("imagem: arquivo não encontrado: %s", r.Path)
}

// ResolveImagePayload resolve o .imgb pareado com a tabela.
func ResolveImagePayload(part string, r ImageRef, tableRel string) (ImageFile, error) {
	pairRel := r.Payload
	if pairRel == "" {
		pairRel = ImagePairPath(tableRel)
	}
	if f, ok := ResolveRelFile(part, pairRel); ok {
		return f, nil
	}
	return ImageFile{}, fmt.Errorf("imagem: binário .imgb não encontrado (%s)", pairRel)
}
