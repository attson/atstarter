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

// UpdateVerifyPublicKey is the base64-encoded Ed25519 public key used to
// verify SHA256SUMS.sig on downloaded update artifacts. Injected via
// -ldflags "-X main.UpdateVerifyPublicKey=…" from the CI secret. Empty in
// dev builds — the updater downgrades to check-only (no install) in that case.
var UpdateVerifyPublicKey = ""

func main() {
	// Create an instance of the app structure
	app := NewApp()

	title := "AT Starter"
	if Version != "dev" {
		title = "AT Starter " + Version
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
