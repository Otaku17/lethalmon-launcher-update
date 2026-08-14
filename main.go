package main

import (
	"embed"
	"os"

	"LethalmonLauncher/backend"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// frontendPackageJSON is embedded here rather than in package backend because
// //go:embed can only reach files inside the declaring file's own directory —
// backend/ cannot see ../frontend/package.json. main() parses it and injects
// the result, so the Go binary and the frontend still share a single source of
// truth for the launcher's version.
//
//go:embed frontend/package.json
var frontendPackageJSON []byte

// main configures and starts the Wails window, binding backend.App's exported
// methods so the frontend can call them directly (see wailsjs/go/backend/App).
//
// A self-update in progress briefly re-invokes this same exe with
// backend.UpdateApplyFlag instead of starting the GUI — see
// backend/app_launcher_update_windows.go and backend.ApplyLauncherUpdate.
func main() {
	if len(os.Args) >= 3 && os.Args[1] == backend.UpdateApplyFlag {
		backend.ApplyLauncherUpdate(os.Args[2])
		return
	}

	backend.SetLauncherVersion(backend.ParseLauncherVersion(frontendPackageJSON))

	// Create an instance of the app structure
	app := backend.NewApp()

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
		OnStartup:        app.Startup,
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
