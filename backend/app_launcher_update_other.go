//go:build !windows && !linux

package backend

import "fmt"

// installLauncherUpdate: self-update isn't implemented on this platform (see
// app_launcher_update_windows.go / _linux.go for the two that do).
func installLauncherUpdate(newExePath string) error {
	return fmt.Errorf("launcher self-update is not supported on this platform")
}
