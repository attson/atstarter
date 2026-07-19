package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// Version is stamped by the build via -ldflags "-X main.Version=…" during
// release builds; local `wails dev` builds keep the "dev" default.
var Version = "dev"

func main() {
	// Create an instance of the app structure
	app := NewApp()

	title := "atstarter"
	if Version != "dev" {
		title = "atstarter " + Version
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  title,
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
