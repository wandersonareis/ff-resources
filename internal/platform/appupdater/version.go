// Package appupdater é o ÚNICO ponto do app que fala com o updater do
// Wails v3 (pkg/updater): configura a origem das releases no GitHub, cuida
// do check silencioso de startup e expõe as ações (verificar agora,
// reiniciar para aplicar) para o menu, para o UpdateService do frontend e
// para o diálogo de saída.
//
// O pacote é dividido por build tag porque o modo servidor (`-tags server`
// — binário HTTP puro, sem janela nativa) não tem o que atualizar da forma
// como o desktop atualiza:
//
//	appupdater.go          //go:build !server  — janela, check e fluxo reais
//	appupdater_stub.go     //go:build server   — no-ops e erro claro
//	version.go             (sem tag)           — estado compartilhado
//
// Origem das releases: o repositório do monorepo (configurado por app em
// appupdater.Init), com um zip por app ("spira-..."/"pulse-...") e o sidecar
// SHA256SUMS por release.
package appupdater

import (
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// checksumAsset é o sidecar SHA-256 padrão publicado junto de cada release.
const checksumAsset = "SHA256SUMS"

var (
	stateMu sync.RWMutex

	// current é a versão do RELEASE TRAIN em execução, gravada por Init e
	// injetada por -ldflags. É a versão que o updater compara com a tag.
	current string

	// appVersion é a versão EXIBIDA do app (semver própria), também injetada
	// por ldflags. Pode ficar estagnada enquanto o train avança.
	appVersion string

	// ready é a versão já baixada e em stage, esperando o restart.
	ready string

	// theme é o tema corrente ("dark"|"light"): espelho do config.json para
	// a janela de update (HTML estático, webview separada, sem acesso ao
	// localStorage do frontend).
	theme string
)

// CurrentVersion devolve a versão do release train em execução: "2.5.0" em
// build de release e "dev" em desenvolvimento. É a versão que o updater
// compara com a tag do GitHub.
func CurrentVersion() string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return current
}

// AppVersion devolve a versão exibida do app (semver própria). Pode ser
// diferente do release train: o app foi buildado num train 2.5.0 com versão
// 1.4.0, por exemplo.
func AppVersion() string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return appVersion
}

// setCurrent grava a versão em execução (chamado pelos dois Init).
func setCurrent(v string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	current = v
}

// setAppVersion grava a versão exibida do app (chamado pelos dois Init).
func setAppVersion(v string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	appVersion = v
}

// setReadyVersion registra o alvo da atualização em stage (evento
// EventUpdateReady) — o diálogo de saída usa para nomear a versão.
func setReadyVersion(v string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	ready = v
}

// ReadyVersion devolve a versão que o próximo restart vai aplicar
// ("" quando não há nada em stage).
func ReadyVersion() string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return ready
}

// CurrentTheme devolve o tema a renderizar na janela de update:
// o persistido via ConfigService.SetTheme ou o default "dark" (o mesmo do
// Providers do frontend) antes de qualquer sync.
func CurrentTheme() string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	if theme != "dark" && theme != "light" {
		return "dark"
	}
	return theme
}

// SetTheme grava o tema corrente (chamado pelo ConfigService quando o
// usuário alterna no frontend, e na carga do config).
func SetTheme(t string) {
	t = trimTheme(t)
	stateMu.Lock()
	theme = t
	stateMu.Unlock()
	if t != "" {
		emitTheme(t) // fora do lock: broadcast para as janelas (update aberta)
	}
}

// trimTheme normaliza para os dois valores que o CSS do window.html conhece.
func trimTheme(t string) string {
	t = strings.TrimSpace(t)
	if t != "dark" && t != "light" {
		return ""
	}
	return t
}

// ReadyToApply diz se já existe uma atualização baixada, verificada e em
// stage: o próximo restart troca o binário. É o sinal que o App.shouldQuit
// usa para trocar o diálogo normal de "alterações não salvas" por um que
// explica o reinício — sem isso o diálogo de saída cancelaria o restart do
// updater (o helper espera o pai sair e desiste após HelperReadyTimeout).
func ReadyToApply() bool {
	app := application.Get()
	if app == nil {
		return false
	}
	return app.Updater.State() == updater.StateReady
}
