// Package service é a CASCA Wails do app FF13. O núcleo (texto, filelist,
// imagens, coleção e export) mora em internal/families/white/service; aqui
// ficam só os métodos que dependem do app: registro do serviço, perfis,
// configuração local, seletores nativos e a trava de escrita.
//
// FF13Service EMBUTE o núcleo: os métodos de domínio são promovidos para o
// binding sem reescrever assinatura, e o app só acrescenta o que é dele.
package service

import (
	"ffresources/internal/families/white"
	"ffresources/internal/families/white/profile/config"
	whiteservice "ffresources/internal/families/white/service"
)

// FF13Service é a ponte Wails do app ff13: expõe os perfis da trilogia e o
// núcleo White ao frontend. Registrado no main (application.NewService).
type FF13Service struct {
	*whiteservice.Service
}

// New monta o serviço com o núcleo White e a ponte de progresso do Wails.
func New() *FF13Service {
	return &FF13Service{Service: whiteservice.New()}
}

// Parts devolve os perfis da trilogia (FFXIII/FFXIII-2/LRFFXIII).
func (FF13Service) Parts() []white.PartInfo { return white.FF13Service{}.ListParts() }

// LocalConfig devolve o caminho da config local (raízes dos jogos e binários):
// é de onde a interface tira o caminho de instalação.
func (FF13Service) LocalConfig() config.ConfigSnapshot { return config.Snapshot() }

// InitWriteGuard aponta o GameFilesRoot da trava de escrita herdada do FFX
// (CheckWritablePath) para a raiz de data/ do FF13.
func InitWriteGuard() {
	whiteservice.InitWriteGuard()
}
