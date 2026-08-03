//go:build !windows

package main

import "fmt"

// installLauncherUpdate: self-update is only implemented for Windows (the
// launcher only ships a Windows build for now).
func installLauncherUpdate(newExePath string) error {
	return fmt.Errorf("launcher self-update is only supported on Windows")
}
