package main

import (
	"testing"
)

// TestSetActivePartFeedsSnapshot confere que a part ativa é estado de
// EXECUÇÃO do backend: a aba principal manda a part e o ConfigSnapshot a
// devolve (é o que a janela de config usa para mostrar a instalação certa).
func TestSetActivePartFeedsSnapshot(t *testing.T) {
	var svc FF13Service
	t.Cleanup(func() { svc.SetActivePart("part1") })

	svc.SetActivePart("part3")
	if got := svc.ConfigSnapshot().GameVersion; got != "part3" {
		t.Errorf("gameVersion: %q, quer %q", got, "part3")
	}

	// O id é normalizado: part desconhecida cai no part1.
	svc.SetActivePart("lixo")
	if got := svc.ConfigSnapshot().GameVersion; got != "part1" {
		t.Errorf("gameVersion (id inválido): %q, quer %q", got, "part1")
	}
}

// TestActivePartDefault confere o default: sem nenhum aviso da aba, a part
// ativa é o part1.
func TestActivePartDefault(t *testing.T) {
	activePart.Store("")
	if got := currentActivePart(); got != "part1" {
		t.Errorf("currentActivePart: %q, quer %q", got, "part1")
	}
}
