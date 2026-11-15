// Package service é a ponte Wails do app FF13: expõe ao frontend os métodos
// que orquestram os formatos (texto, filelist, imagem), a coleção de dados e o
// export. O conhecimento dos formatos mora em backend/formats; a configuração
// de execução, em backend/config.
//
// Tudo aqui é chamado pelo frontend (bindings) ou pelo main do app.
package service

import (
	"ffresources/apps/ff13/backend/config"
	"ffresources/internal/families/white"
	"ffresources/internal/shared/progress"
)

// FF13Service e a ponte Wails do app ff13: expõe os perfis da trilogia para o
// frontend. Registrado no main (application.NewService).
//
// progress é a ponte das operações LONGAS (o export): criada em New() e
// injetada — nil = sem progresso (testes/CLI), e o batch vira no-op.
type FF13Service struct {
	progress *progress.Emitter
}

// New monta o serviço com a ponte de progresso ligada aos eventos do Wails.
func New() *FF13Service {
	return &FF13Service{progress: progress.NewEmitter(notifyWarn)}
}

func (FF13Service) Parts() []white.PartInfo { return white.FF13Service{}.ListParts() }

// LocalConfig devolve o caminho da config local (raízes dos jogos e binários):
// é de onde a interface tira o caminho de instalação.
func (FF13Service) LocalConfig() config.ConfigSnapshot { return config.Snapshot() }
