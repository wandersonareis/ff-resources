package service

import (
	"fmt"
	"strings"

	"ffresources/apps/ff13/backend/common"
	"ffresources/apps/ff13/backend/formats"
	"ffresources/apps/ff13/backend/parts"
)

// NAVEGAÇÃO do filelist: as raízes (índices) da part, os filhos de um
// diretório e a união de uma zona. A descoberta e a árvore moram em
// backend/formats; aqui só a ponte com o frontend.

// ListFilelistRoots descobre os filelists da PART informada. Onde procurar é
// decisão da part (part1: white_data/sys; part2: alba_data/sys+udp; part3:
// weiss_data/sys+dlc). Os índices EMBUTIDOS (os de zona, que moram dentro do
// bin global) entram como raízes do grupo "zone".
//
// O que sai daqui é DADO CRU: o nome/caminho do índice como estão no disco
// (sem rótulo, sem normalização), a plataforma e o idioma deduzidos do NOME
// do índice, a pasta onde a cópia vencedora mora e a contagem de entradas. A
// árvore e os nomes exibidos são do frontend.
//
// Também valida o par filelist <-> white bin e registra como ERRO todo
// arquivo que ficou sem par.
func (FF13Service) ListFilelistRoots(part string) ([]common.FilelistRoot, error) {
	p := parts.Get(part)
	disc := formats.DiscoverPart(p)
	formats.CheckFilelistPairs(p, disc.Indexes)
	return formats.PruneKeyOnlyRoots(disc), nil
}

// ListFilelistDir devolve os FILHOS IMEDIATOS de um diretório de um índice.
//
// root pode ser "indice.bin" (índice em disco), "indice.bin#zone/filelist_x"
// (índice EMBUTIDO dentro de um white bin) ou "zone:z000" (a raiz virtual de
// uma zona): nesse caso o índice embutido é extraído do container e lido, e
// dir é o caminho DENTRO dele.
//
// O que sai daqui é DADO CRU: um nível da árvore, com o caminho de cada nó
// (para pedir os filhos), o tamanho descomprimido e o kind (dir/file/index).
// Diretórios com filho único já vêm FUNDIDOS (MergeChain), como no Pulse.
func (FF13Service) ListFilelistDir(part, root, dir string) ([]common.FilelistNode, error) {
	p := parts.Get(part)
	if zone, ok := formats.ZoneRootOf(root); ok {
		disc := formats.DiscoverPart(p)
		return formats.ZoneNodes(p, disc, zone, dir), nil
	}
	idx, err := formats.EmbeddedRoot(p, root)
	if err != nil {
		return nil, err
	}
	entries := formats.IndexEntries(p, idx, nil)
	formats.CheckSource(idx, common.SourceOf(formats.IndexNameOf(idx)), entries)
	return formats.ListDirNodes(p, formats.BuildIndexTree(entries), dir, formats.ZoneOfIndex(idx), root), nil
}

// ListFilelistZone devolve a UNIÃO do conteúdo de uma zona: o índice de zona
// (u+c, embutido) e as entradas "zone/<z>/..." dos demais índices da part
// (ex.: o índice de scripts tem "zone/z000/scr000.clb"). O prefixo
// "zone/<z>/" é removido e o dedupe é pelo caminho em minúsculas (o índice de
// zona tem precedência).
func (FF13Service) ListFilelistZone(part, zone string) ([]common.FilelistNode, error) {
	p := parts.Get(part)
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone == "" {
		return nil, fmt.Errorf("filelist: zona vazia")
	}
	disc := formats.DiscoverPart(p)
	return formats.ZoneNodes(p, disc, zone, ""), nil
}
