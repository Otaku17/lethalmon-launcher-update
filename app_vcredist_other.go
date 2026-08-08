//go:build !windows

package main

// ensureVCRedist is a no-op outside Windows: the game itself only ships a
// Windows build (see app_exec_other.go), so there is no runtime to check for.
func (a *App) ensureVCRedist() error {
	return nil
}
