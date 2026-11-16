package service

import (
	"ffresources/internal/platform/appupdater"
)

// UpdateService expõe as AÇÕES do auto-update do FF13 ao frontend (banner de
// atualização). O estado do fluxo vem dos eventos `wails:updater:*`; aqui
// ficam só as chamadas de volta que exigem bindings.
type UpdateService struct{}

// CheckNow abre a janela de atualização e roda o fluxo completo (check →
// download → verificação → stage). Bloqueia até o fim; o binding roda em
// goroutine própria e a UI não trava.
func (u *UpdateService) CheckNow() error {
	return appupdater.CheckNow()
}

// Restart aplica a atualização em stage: sobe o helper, encerra o app e,
// quando o processo sai, o binário é trocado e a nova versão relança.
func (u *UpdateService) Restart() error {
	return appupdater.Restart()
}

// CurrentVersion devolve a versão do RELEASE TRAIN em execução — o número que
// o updater compara com a tag do GitHub.
func (u *UpdateService) CurrentVersion() string {
	return appupdater.CurrentVersion()
}

// AppVersion devolve a versão exibida do app (semver própria), que pode ser
// diferente do release train.
func (u *UpdateService) AppVersion() string {
	return appupdater.AppVersion()
}
