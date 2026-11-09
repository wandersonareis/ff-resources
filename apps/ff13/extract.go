package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffresources/apps/ff13/parts"
	"ffresources/internal/families/white/codecs/whitebin"
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
// É o mesmo mapeamento de filelistDataPath (filelisttext.go), então os
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

// ExtractFilelistToData extrai arquivos de um índice do filelist para
// data/partN.
//
// root aceita o mesmo protocolo da árvore:
//   - caminho de um índice em disco/embutido (ex.: "…/filelistu.win32.bin");
//   - "zone:z000" (a raiz virtual de uma zona).
//
// paths são os caminhos INTERNOS do índice (os mesmos que a árvore devolve);
// vazio = todas as entradas do índice. Um caminho de diretório seleciona
// todos os arquivos sob ele.
//
// force=false (padrão) pula o que já existe em data/; force=true sobrescreve.
func (FF13Service) ExtractFilelistToData(part, root string, paths []string, force bool) (FilelistExtractResult, error) {
	part = normalizePart(part)
	result := FilelistExtractResult{Root: root}

	dataDir, err := partRoot("data", part)
	if err != nil {
		return result, err
	}

	p := parts.Get(part)
	idx, zone, err := resolveFilelistIndex(p, root)
	if err != nil {
		return result, err
	}
	result.Zone = zone
	if idx.Container == "" {
		return result, fmt.Errorf("filelist: índice sem white bin par: %s", root)
	}

	selected := selectExtractEntries(indexEntries(p, idx, nil), paths)
	result.Requested = len(selected)
	if len(selected) == 0 {
		return result, fmt.Errorf("filelist: nenhum arquivo encontrado em %s", root)
	}

	extracted := extractIndexEntriesToData(dataDir, zone, idx, selected, force)
	extracted.Root = result.Root
	extracted.Zone = result.Zone
	return extracted, nil
}

// extractIndexEntriesToData é o miolo testável: recebe o índice já resolvido
// e as entradas selecionadas e grava em data/ no caminho canônico.
func extractIndexEntriesToData(dataDir, zone string, idx *filelistIndex, selected []indexEntry, force bool) FilelistExtractResult {
	result := FilelistExtractResult{
		Zone:      zone,
		Requested: len(selected),
	}
	for _, e := range selected {
		dataPath := filelistDataPath(zone, e.Path)
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

// resolveFilelistIndex resolve a raiz do filelist para o índice e a zona.
func resolveFilelistIndex(p parts.Part, root string) (*filelistIndex, string, error) {
	if zone, ok := zoneRootOf(root); ok {
		for _, idx := range discoverPart(p).indexes {
			if isZoneIndexOf(idx, zone) {
				return idx, zone, nil
			}
		}
		return nil, "", fmt.Errorf("filelist: índice da zona %s não encontrado", zone)
	}

	idx, err := embeddedRoot(p, root)
	if err != nil {
		return nil, "", err
	}
	return idx, zoneOfIndex(idx), nil
}

// selectExtractEntries aplica a seleção de paths sobre as entradas do índice.
// Um path de diretório inclui todos os filhos (prefixo + "/").
func selectExtractEntries(entries []indexEntry, paths []string) []indexEntry {
	if len(paths) == 0 {
		return entries
	}

	wanted := make([]string, 0, len(paths))
	for _, raw := range paths {
		if p := normalizeIndexPath(raw); p != "" {
			wanted = append(wanted, p)
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	selected := make([]indexEntry, 0, len(entries))
	for _, e := range entries {
		entryPath := normalizeIndexPath(e.Path)
		for _, w := range wanted {
			if entryPath == w || strings.HasPrefix(entryPath, w+"/") {
				selected = append(selected, e)
				break
			}
		}
	}
	return selected
}

// normalizeIndexPath normaliza um caminho interno para comparação: barras,
// minúsculas e sem barra inicial/final.
func normalizeIndexPath(p string) string {
	p = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"))
	return strings.Trim(p, "/")
}
