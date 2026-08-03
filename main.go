package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// main configures and starts the Wails window, binding App's exported
// methods so the frontend can call them directly (see wailsjs/go/main/App).
//
// A self-update in progress briefly re-invokes this same exe with
// updateApplyFlag instead of starting the GUI — see
// app_launcher_update_windows.go and applyLauncherUpdate.
func main() {
	if len(os.Args) >= 3 && os.Args[1] == updateApplyFlag {
		applyLauncherUpdate(os.Args[2])
		return
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "Lethalmon Launcher",
		Width:         1280,
		Height:        720,
		MinWidth:      1280,
		MinHeight:     720,
		MaxWidth:      1280,
		MaxHeight:     720,
		DisableResize: true,
		Frameless:     true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			DisableWindowIcon:    false,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
