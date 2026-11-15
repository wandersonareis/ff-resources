// Package white implementa a familia White Engine (FFXIII, LRFFXIII e titulos
// futuros que compartilham filelist + ZTR + imagens, como Dirge of Cerberus).
//
// A separacao do plano: "formatos" (codecs/) nao sabem qual jogo e; "titulos"
// (titles/) sao profiles com gameCode, idiomas, layouts e limites. Este pacote
// raiz so registra os profiles conhecidos.
package white

import (
	"sort"

	"ffresources/internal/families/white/family"
	"ffresources/internal/families/white/titles/ff13"
)

type Family struct{}

func New() *Family { return &Family{} }

func (*Family) ID() string { return "white" }

func (*Family) Profiles() []family.Profile {
	ps := []family.Profile{
		ff13.Part1(),
		ff13.Part2(),
		ff13.Part3(),
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID() < ps[j].ID() })
	return ps
}

// ProfilesByID devolve os profiles indexados por ID (part1/part2/part3).
func (*Family) ProfilesByID() map[string]family.Profile {
	out := map[string]family.Profile{}
	for _, p := range New().Profiles() {
		out[p.ID()] = p
	}
	return out
}
