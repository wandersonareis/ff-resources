package main

import (
	"embed"

	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	ffxcommon "ffresources/internal/families/ffx/profile/common"
	logging "ffresources/internal/shared/logging"
	"ffresources/internal/shared/progress"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/out
var assets embed.FS

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
}

// initWriteGuard aponta o GameFilesRoot da trava de escrita herdada do FFX
// (CheckWritablePath) para a raiz de data/ do FF13. Sem isso, GameFilesRoot
// fica no diretório do executável e o caminho do export — <raiz>/data/mods/edits/…
// — é lido como "data\mods\…" (fora de mods/) e recusado no FINAL do batch:
// o progresso mostra todas as unidades processadas, mas nenhum arquivo sai.
// Com GameFilesRoot = <raiz>/data, o destino vira "mods/edits/…" e é aceito;
// data/partN/ continua bloqueado (proteção preservada).
func initWriteGuard() {
	root, err := dataRoot("data")
	if err != nil {
		return
	}
	ffxcommon.SetGameFilesRoot(root)
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	logging.SetAppName("ff13")
	logging.Init()
	initWriteGuard()

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "FF13 Resources Editor",
		Description: "FINAL FANTASY XIII Trilogy Resources Editor",
		Services: []application.Service{
			application.NewService(&GreetService{}),
			application.NewService(&FF13Service{
				// Ponte de progresso: o export publica os snapshots nos
				// eventos do wails (ProgressStart/Update/End) — sem global,
				// sem canal de pacote.
				progress: progress.NewEmitter(notifyWarn),
			}),
			application.NewService(&WindowService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		// Name identifica a principal no registry (appwindows): as janelas
		// filhas (config, logs) nascem ligadas a ela e fecham junto.
		Name:  "main",
		Title: "FINAL FANTASY XIII Trilogy Resources Editor",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:  1000,
		Height: 618,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
