package service

import (
	"fmt"

	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
)

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
func (Service) ExtractFilelistToData(part, root string, paths []string, force bool) (FilelistExtractResult, error) {
	part = config.NormalizePart(part)
	result := FilelistExtractResult{Root: root}

	dataDir, err := config.PartRoot("data", part)
	if err != nil {
		return result, err
	}

	p := parts.Get(part)
	idx, zone, err := ResolveFilelistIndex(p, root)
	if err != nil {
		return result, err
	}
	result.Zone = zone
	if idx.Container == "" {
		return result, fmt.Errorf("filelist: índice sem white bin par: %s", root)
	}

	selected := SelectExtractEntries(IndexEntries(p, idx, nil), paths)
	result.Requested = len(selected)
	if len(selected) == 0 {
		return result, fmt.Errorf("filelist: nenhum arquivo encontrado em %s", root)
	}

	extracted := ExtractIndexEntriesToData(dataDir, zone, idx, selected, force)
	extracted.Root = result.Root
	extracted.Zone = result.Zone
	return extracted, nil
}
