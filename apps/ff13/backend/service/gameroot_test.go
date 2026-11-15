package service

import (
	"testing"

	"ffresources/apps/ff13/backend/config"
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
	config.SetActivePart("")
	if got := config.ActivePart(); got != "part1" {
		t.Errorf("ActivePart: %q, quer %q", got, "part1")
	}
}
