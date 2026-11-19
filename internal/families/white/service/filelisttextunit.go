package service

// LEITURA DE UMA UNIDADE DE TEXTO A PARTIR DO FILELIST.
//
// A árvore Filelist e a árvore /data são FONTES diferentes: o filelist é o
// dado validado do jogo (índice + white bin); /data é a extração, que pode
// estar velha, incompleta ou corrompida. Quando a folha veio do filelist, a
// unidade é montada com as entradas do PRÓPRIO índice — nunca com o que está
// em data/partN.

import (
	"fmt"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/profile/parts"
)

// ResolveFilelistTextUnit monta a unidade a partir do índice `root` (caminho
// em disco, "pai#virtual" ou "zone:z000"). `unitID` é o id da unidade
// ("<dir>/<base>"); `indexPath` é o caminho clicado e só serve de preferência
// quando o índice repete o mesmo idioma em mais de uma entrada.
func ResolveFilelistTextUnit(part, root, indexPath, unitID string) (*TextUnit, error) {
	p := parts.Get(part)
	idx, zone, err := ResolveFilelistIndex(p, root)
	if err != nil {
		return nil, err
	}
	if idx == nil || idx.Coll == nil {
		return nil, fmt.Errorf("filelist: índice não carregado (%s)", root)
	}
	if idx.Container == "" {
		return nil, fmt.Errorf("filelist: índice sem white bin (%s)", root)
	}
	dir, base, err := UnitScope(unitID)
	if err != nil {
		return nil, err
	}

	files := map[string]UnitFile{}
	preferred := normalizeInsertRel(indexPath)
	for ci := range idx.Coll.Chunks {
		for ei := range idx.Coll.Chunks[ci].Entries {
			entryPath := fl.EntryPath(idx.Coll.Chunks[ci].Entries[ei].FileInfo)
			dataPath := FilelistDataPath(zone, entryPath)
			id, lang, ok := FilelistUnitID(dataPath)
			if !ok || id != unitID {
				continue
			}
			// O primeiro de cada idioma vence; o caminho clicado tem
			// precedência quando o índice repete a mesma língua.
			if _, exists := files[lang]; exists &&
				(preferred == "" || normalizeInsertRel(entryPath) != preferred) {
				continue
			}
			files[lang] = UnitFile{
				Rel:       dataPath,
				Container: ContainerOfExt(entryPath),
				Source: UnitSource{
					Rel:   dataPath,
					Bin:   idx.Container,
					Coll:  idx.Coll,
					VPath: entryPath,
				},
			}
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("filelist: unidade %s não encontrada em %s", unitID, root)
	}
	u := buildUnit(dir, base, files, "", p.TextKind(unitID))
	if u.Primary == "" {
		return nil, fmt.Errorf("filelist: unidade %s sem arquivo _us", unitID)
	}
	return u, nil
}
