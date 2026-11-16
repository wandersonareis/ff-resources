package service

import (
	"ffresources/internal/platform/appupdater"
)

// UpdateService expõe as AÇÕES do auto-update ao frontend (banner de
// atualização em frontend/src/components/update-banner.tsx). O estado do
// fluxo (disponível, baixando, pronta) a banner lê direto dos eventos
// updater:* publicados no bus do app — aqui ficam só as chamadas de volta
// que exigem bindings: abrir a janela nativa, reiniciar para aplicar e
// consultar a versão em execução.
//
// No modo servidor os métodos devolvem erro claro (stub de backend/appupdater):
// não há janela nativa nem troca de binário para fazer.
type UpdateService struct{}

// CheckNow abre a janela de atualização e roda o fluxo completo (check →
// download → verificação → stage). Bloqueia até o fim, mas o binding JS roda
// em goroutine própria — a UI não trava.
func (u *UpdateService) CheckNow() error {
	return appupdater.CheckNow()
}

// Restart aplica a atualização em stage: sobe o helper (o próprio binário),
// encerra o app e, quando o processo sai, o binário é trocado e a nova
// versão relança. Erro se não há nada em stage.
func (u *UpdateService) Restart() error {
	return appupdater.Restart()
}

// CurrentVersion devolve a versão do RELEASE TRAIN em execução ("2.5.0" em
// build de release, "dev" em desenvolvimento — sem -ldflags). É o número que
// o updater compara com a tag do GitHub.
func (u *UpdateService) CurrentVersion() string {
	return appupdater.CurrentVersion()
}

// AppVersion devolve a versão exibida do app (semver própria), que pode ser
// diferente do release train.
func (u *UpdateService) AppVersion() string {
	return appupdater.AppVersion()
}
