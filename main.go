package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"Postwoman/internal/database"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	appOptions := &options.App{
		Title:     "Postwoman",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			ProgramName: "Postwoman",
		},
	}
	// Portable builds keep the WebView2 cache beside the executable so a USB
	// copy carries no state in the host user profile.
	if webviewData := database.PortableWebviewDataPath(); webviewData != "" {
		_ = os.MkdirAll(webviewData, 0o755)
		appOptions.Windows = &windows.Options{WebviewUserDataPath: webviewData}
	}

	err := wails.Run(appOptions)

	if err != nil {
		println("Error:", err.Error())
	}
}
