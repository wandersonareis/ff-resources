package service

import (
	ffxcommon "ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/families/white/profile/config"
)

// InitWriteGuard aponta o GameFilesRoot da trava de escrita herdada do FFX
// (CheckWritablePath) para a raiz de data/ do FF13. Sem isso, GameFilesRoot
// fica no diretório do executável e o caminho do export — <raiz>/data/mods/edits/…
// — é lido como "data\mods\…" (fora de mods/) e recusado no FINAL do batch:
// o progresso mostra todas as unidades processadas, mas nenhum arquivo sai.
// Com GameFilesRoot = <raiz>/data, o destino vira "mods/edits/…" e é aceito;
// data/partN/ continua bloqueado (proteção preservada).
func InitWriteGuard() {
	root, err := config.DataRoot("data")
	if err != nil {
		return
	}
	ffxcommon.SetGameFilesRoot(root)
}
