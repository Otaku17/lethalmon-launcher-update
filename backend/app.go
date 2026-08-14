// Package backend holds everything the launcher does: downloading and
// installing the game, reading and writing its options, detecting the GPU,
// checking for and applying launcher self-updates, and installing the Visual
// C++ Redistributable.
//
// It is a separate package from main so that main.go is only what Wails needs
// to open a window, and so the behaviour below can be tested on its own (see
// the tests/ directory). App's exported methods are the launcher's API: Wails
// binds them and generates the frontend's TypeScript bindings from them, in
// frontend/wailsjs/go/backend.
//
// Files suffixed _windows.go / _other.go are build-tag pairs: the Windows
// implementation and its no-op stub elsewhere.
package backend

import "context"

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}
