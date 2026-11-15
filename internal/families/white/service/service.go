// Package service concentra a ORQUESTRAÇÃO do jogo White (FFXIII): unidades
// de texto (ZTR/WPD), filelists (descoberta, índice, árvore, zona e extração),
// imagens (WPD/TRB/faces), coleção de data/partN, export e catálogo de tags.
//
// Os CODECS ficam em white/codecs; a configuração de execução e as
// particularidades de partN, em white/profile. Este pacote não conhece o
// app Wails: o app expõe uma casca fina que delega para cá.
package service

import (
	"ffresources/internal/shared/progress"
)

// Service é o núcleo do jogo White: mantém a ponte de progresso das operações
// longas (o export) e o estado de execução (part ativa, preferência de mods).
//
// progress nil = sem progresso (testes/CLI), e o batch vira no-op.
type Service struct {
	progress *progress.Emitter
}

// New monta o serviço com a ponte de progresso ligada aos eventos do Wails.
func New() *Service {
	return &Service{progress: progress.NewEmitter(notifyWarn)}
}
