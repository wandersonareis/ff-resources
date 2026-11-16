package main

import (
	"embed"

	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	ff13service "ffresources/apps/ff13/backend/service"
	"ffresources/internal/platform/appupdater"
	logging "ffresources/internal/shared/logging"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/out
var assets embed.FS

// currentVersion é a versão do RELEASE TRAIN (tag do GitHub), injetada por
// -ldflags "-X main.currentVersion=<VERSION>". "dev" desliga o check de
// startup do updater.
var currentVersion = "dev"

// appVersion é a versão exibida do app (semver própria), injetada por
// -ldflags "-X main.appVersion=<apps/ff13/VERSION>".
var appVersion = "dev"

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	logging.SetAppName("ff13")
	logging.Init()

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "FF13 Resources Editor",
		Description: "FINAL FANTASY XIII Trilogy Resources Editor",
		Services: []application.Service{
			application.NewService(ff13service.New()),
			application.NewService(&ff13service.WindowService{}),
			application.NewService(&ff13service.UpdateService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Auto-update: releases do repo do monorepo; o asset deste app é pulse-*.
	if err := appupdater.Init(app, currentVersion, appVersion, appupdater.Config{
		Repository:  "wandersonareis/ff-resources",
		AssetPrefix: "pulse",
		WindowTitle: "Atualização — FINAL FANTASY XIII Trilogy",
	}); err != nil {
		log.Fatalf("updater: %v", err)
	}

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
		// Window sized to the golden ratio (1280 / 800 ≈ 1.6).
		Width:  1280,
		Height: 800,
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

	// Check silencioso de atualização (~5s depois).
	appupdater.StartupCheck(app)

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
