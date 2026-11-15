package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffresources/internal/families/white/codecs/whitebin"
	"ffresources/internal/families/white/profile/parts"
)

// EXTRAÇÃO do filelist para data/.
//
// O filelist é o inventário completo do jogo (todos os arquivos e todos os
// idiomas); data/partN guarda só o que já foi extraído. Esta função lê os
// bytes do white bin e grava em data/partN no caminho CANÔNICO do app:
//
//	índice global: data/partN/<caminho do índice>
//	índice de zona: data/partN/zone/<z>/<caminho relativo à zona>
//
// É o mesmo mapeamento de FilelistDataPath (filelisttext.go), então os
// arquivos de zona caem direto em zone/<z>/... — sem mover nada depois.
// O mods/ não é tocado: ele é criado pelo import, no caminho target correto.

// FilelistExtractResult é o retrato de uma extração.
type FilelistExtractResult struct {
	Root      string   `json:"root"`
	Zone      string   `json:"zone,omitempty"`
	Requested int      `json:"requested"`
	Written   int      `json:"written"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
}

// ExtractIndexEntriesToData é o miolo testável: recebe o índice já resolvido
// e as entradas selecionadas e grava em data/ no caminho canônico.
func ExtractIndexEntriesToData(dataDir, zone string, idx *FilelistIndex, selected []IndexEntry, force bool) FilelistExtractResult {
	result := FilelistExtractResult{
		Zone:      zone,
		Requested: len(selected),
	}
	for _, e := range selected {
		dataPath := FilelistDataPath(zone, e.Path)
		dst := filepath.Join(dataDir, filepath.FromSlash(dataPath))

		if !force {
			if _, serr := os.Stat(dst); serr == nil {
				result.Skipped++
				continue
			}
		}

		raw, xerr := whitebin.ExtractBytes(idx.Container, e.Path, idx.Coll)
		if xerr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", e.Path, xerr))
			continue
		}
		if merr := os.MkdirAll(filepath.Dir(dst), 0o755); merr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", dataPath, merr))
			continue
		}
		if werr := os.WriteFile(dst, raw, 0o644); werr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", dataPath, werr))
			continue
		}
		result.Written++
	}
	return result
}

// ResolveFilelistIndex resolve a raiz do filelist para o índice e a zona.
func ResolveFilelistIndex(p parts.Part, root string) (*FilelistIndex, string, error) {
	if zone, ok := ZoneRootOf(root); ok {
		for _, idx := range DiscoverPart(p).Indexes {
			if IsZoneIndexOf(idx, zone) {
				return idx, zone, nil
			}
		}
		return nil, "", fmt.Errorf("filelist: índice da zona %s não encontrado", zone)
	}

	idx, err := EmbeddedRoot(p, root)
	if err != nil {
		return nil, "", err
	}
	return idx, ZoneOfIndex(idx), nil
}

// SelectExtractEntries aplica a seleção de paths sobre as entradas do índice.
// Um path de diretório inclui todos os filhos (prefixo + "/").
func SelectExtractEntries(entries []IndexEntry, paths []string) []IndexEntry {
	if len(paths) == 0 {
		return entries
	}

	wanted := make([]string, 0, len(paths))
	for _, raw := range paths {
		if p := NormalizeIndexPath(raw); p != "" {
			wanted = append(wanted, p)
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	selected := make([]IndexEntry, 0, len(entries))
	for _, e := range entries {
		entryPath := NormalizeIndexPath(e.Path)
		for _, w := range wanted {
			if entryPath == w || strings.HasPrefix(entryPath, w+"/") {
				selected = append(selected, e)
				break
			}
		}
	}
	return selected
}

// NormalizeIndexPath normaliza um caminho interno para comparação: barras,
// minúsculas e sem barra inicial/final.
func NormalizeIndexPath(p string) string {
	p = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"))
	return strings.Trim(p, "/")
}
