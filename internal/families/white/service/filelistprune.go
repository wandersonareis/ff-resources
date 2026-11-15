package service

import (
	"regexp"
	"strings"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/profile/common"
)

// keyOnlyFilelists são os índices de DLC que só carregam a chave/liberação
// do pacote. O LR não muda mais; manter a lista evita abrir e varrer esses
// filelists. O fallback abaixo cobre um eventual índice novo.
var keyOnlyFilelists = []string{
	"filelist_p0000000img_a.win32.bin",
	"filelist_p0000000img_v.win32.bin",
}

// keyDatPattern casa o único arquivo dos índices "só chave":
// sys/dlc/key/key00000000.dat.
var keyDatPattern = regexp.MustCompile(`^sys/dlc/key/key\d{8}\.dat$`)

func IsKnownKeyOnlyFilelist(name string) bool {
	n := strings.ToLower(strings.TrimSpace(BaseName(name)))
	for _, known := range keyOnlyFilelists {
		if n == known {
			return true
		}
	}
	return false
}

func IsKeyDatPath(p string) bool {
	p = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"))
	return keyDatPattern.MatchString(p)
}

// IsKeyOnlyCollection devolve true quando o índice tem exatamente UM arquivo
// e ele é o key*.dat do DLC.
func IsKeyOnlyCollection(coll *fl.Collection) bool {
	if coll == nil {
		return false
	}
	paths := coll.Paths()
	if len(paths) != 1 {
		return false
	}
	return IsKeyDatPath(paths[0])
}

// PruneKeyOnlyRoots remove da árvore os filelists que só têm a chave. Os
// nomes conhecidos já foram descartados antes de abrir o arquivo; os demais
// passam pela varredura de conteúdo.
func PruneKeyOnlyRoots(disc PartDiscovery) []common.FilelistRoot {
	byPath := make(map[string]*fl.Collection, len(disc.Indexes))
	for _, idx := range disc.Indexes {
		byPath[idx.Path] = idx.Coll
	}
	out := make([]common.FilelistRoot, 0, len(disc.Roots))
	for _, root := range disc.Roots {
		if IsKnownKeyOnlyFilelist(root.Name) {
			continue
		}
		if coll := byPath[root.Path]; IsKeyOnlyCollection(coll) {
			continue
		}
		out = append(out, root)
	}
	return out
}
