//go:build !server

package appupdater

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	loggingService "ffresources/internal/shared/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// windowHTML é a janela de update em pt-BR (cópia adaptada do template
// assets/window.html do próprio Wails v3, MIT). Textos e cores mudam; o JS,
// os ids e o atributo data-state são contrato com o updater e ficam intactos.
//
//go:embed window.html
var windowHTML string

// startupDelay dá tempo da janela principal nascer e do frontend assinar os
// eventos updater:* antes do primeiro check silencioso.
const startupDelay = 5 * time.Second

// checkTimeout limita o check de startup (a API do GitHub pode engasgar e o
// usuário não deve esperar por isso).
const checkTimeout = 30 * time.Second

// themeToken é o placeholder de tema no window.html: o HTML nasce com
// `data-theme="__APP_THEME__"` e o renderTheme troca por "dark"/"light".
const themeToken = "__APP_THEME__"

// builtinWindow é o ponteiro entregue ao updater no Init. Mutá-lo aqui vale
// para a PRÓXIMA abertura: o Config do updater guarda a interface com o
// mesmo *BuiltinWindow.
var builtinWindow = &updater.BuiltinWindow{
	HTML: windowHTML,
	Options: updater.WindowOptions{
		Title: "Atualização de software",
	},
}

// renderTheme materializa o tema atual no HTML da janela.
func renderTheme() {
	builtinWindow.HTML = strings.ReplaceAll(windowHTML, themeToken, CurrentTheme())
}

// emitTheme avisa TODAS as janelas (main e a de update, se aberta) sobre a
// troca de tema. A de update escuta `app:theme` e só troca o atributo
// data-theme.
func emitTheme(theme string) {
	if app := application.Get(); app != nil {
		app.Event.Emit("app:theme", theme)
	}
}

// Init configura o app.Updater com o provider de releases do GitHub e a
// janela de update própria. Deve rodar depois de application.New e antes de
// app.Run().
//
// version é a versão do RELEASE TRAIN (tag do GitHub), usada na comparação;
// appVersion é a versão exibida do app (semver própria). Quando o app é
// rebuildado, as duas vêm por -ldflags.
func Init(app *application.App, version, appVersion string, cfg Config) error {
	setCurrent(version)
	setAppVersion(appVersion)

	if cfg.Repository == "" {
		return errors.New("updater: Repository é obrigatório")
	}
	if cfg.ChecksumAsset == "" {
		cfg.ChecksumAsset = checksumAsset
	}
	if cfg.WindowTitle != "" {
		builtinWindow.Options.Title = cfg.WindowTitle
	}

	var matcher github.AssetMatcher
	if cfg.AssetPrefix != "" {
		matcher = prefixAssetMatcher(cfg.AssetPrefix)
	}
	gh, err := github.New(github.Config{
		Repository:    cfg.Repository,
		ChecksumAsset: cfg.ChecksumAsset,
		AssetMatcher:  matcher,
	})
	if err != nil {
		return fmt.Errorf("updater: provider github: %w", err)
	}

	if cfg.Theme != nil {
		SetTheme(cfg.Theme())
	}
	renderTheme()

	upCfg := updater.Config{
		// Sem "v" na frente: o provider corta o prefixo das tags (v2.1.1).
		CurrentVersion: version,
		Providers:      []updater.Provider{noAssetIsUpToDate{inner: gh}},
		Window:         builtinWindow,
	}
	if err := app.Updater.Init(upCfg); err != nil {
		return fmt.Errorf("updater: init: %w", err)
	}

	// Guarda o alvo quando a atualização entra em stage: o diálogo de saída
	// nomeia a versão que será aplicada no reinício.
	app.Event.On(updater.EventUpdateReady, func(e *application.CustomEvent) {
		if e == nil {
			return
		}
		switch v := e.Data.(type) {
		case *updater.Release:
			setReadyVersion(v.Version)
		case updater.Release:
			setReadyVersion(v.Version)
		}
	})

	return nil
}

// StartupCheck faz UM check silencioso (sem janela) alguns segundos após o
// app subir. No-op em build de desenvolvimento ("dev").
func StartupCheck(app *application.App) {
	if v := CurrentVersion(); v == "" || v == "dev" {
		return
	}

	go func() {
		time.Sleep(startupDelay)

		ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
		rel, err := app.Updater.Check(ctx)
		cancel()
		if err != nil {
			loggingService.Debug("updater: check de startup ignorado: %s", err)
			return
		}
		if rel == nil {
			return
		}

		loggingService.Info("updater: %s disponível (atual %s) — abrindo a janela de atualização", rel.Version, CurrentVersion())
		renderTheme()
		if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
			loggingService.Error("updater: atualização do startup falhou: %s", err)
		}
	}()
}

// CheckNow abre a janela nativa de atualização e percorre o fluxo completo.
func CheckNow() error {
	app := application.Get()
	if app == nil {
		return errors.New("updater: aplicativo ainda não inicializado")
	}
	renderTheme()
	return app.Updater.CheckAndInstall(context.Background())
}

// Restart aplica a atualização em stage: sobe o helper, pede o Quit e, quando
// o processo sai, o helper troca o binário e relança a nova versão.
func Restart() error {
	app := application.Get()
	if app == nil {
		return errors.New("updater: aplicativo ainda não inicializado")
	}
	return app.Updater.Restart(context.Background())
}

// prefixAssetMatcher aceita só o zip do app (prefixo) e da plataforma/arch
// correntes. Sem isso, um release com spira+pulse poderia baixar o zip do
// outro app (os dois contêm windows-amd64 no nome).
func prefixAssetMatcher(prefix string) github.AssetMatcher {
	prefix = strings.ToLower(strings.TrimSpace(prefix)) + "-"
	return func(req updater.CheckRequest, assets []github.ReleaseAsset) int {
		wantOS := strings.ToLower(req.Platform)
		wantArch := strings.ToLower(req.Arch)
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			if wantOS != "" && !strings.Contains(name, wantOS) {
				continue
			}
			if wantArch != "" && !strings.Contains(name, wantArch) &&
				!(wantArch == "amd64" && strings.Contains(name, "x64")) {
				continue
			}
			return i
		}
		return -1
	}
}

// noAssetIsUpToDate trata "release sem asset do meu app" como "sem
// atualização": releases mistos contêm só os apps que mudaram.
type noAssetIsUpToDate struct {
	inner updater.Provider
}

func (p noAssetIsUpToDate) Name() string { return p.inner.Name() }

func (p noAssetIsUpToDate) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	rel, err := p.inner.Check(ctx, req)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no asset for") {
		return nil, nil
	}
	return rel, err
}

func (p noAssetIsUpToDate) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	return p.inner.Download(ctx, r, dst, onProgress)
}
