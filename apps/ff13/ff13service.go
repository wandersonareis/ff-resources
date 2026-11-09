package main

import (
	"ffresources/internal/families/white"
	"ffresources/internal/shared/progress"
)

// FF13Service e a ponte Wails do app ff13: expõe os perfis da trilogia para o
// frontend. Registrado no main (application.NewService).
//
// progress é a ponte das operações LONGAS (o export): criada no startup e
// injetada — nil = sem progresso (testes/CLI), e o batch vira no-op.
type FF13Service struct {
	progress *progress.Emitter
}

func (FF13Service) Parts() []white.PartInfo { return white.FF13Service{}.ListParts() }

// LocalConfig devolve o caminho da config local (raízes dos jogos e binários):
// é de onde a interface tira o caminho de instalação.
func (FF13Service) LocalConfig() ConfigSnapshot { return configSnapshot() }
