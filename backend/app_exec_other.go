//go:build !windows && !linux

package backend

import "os/exec"

// hideWindow is a no-op on non-Windows platforms (no console flash issue there).
func hideWindow(cmd *exec.Cmd) {}

// anyProcessRunning / killProcesses: process-name based tracking isn't
// implemented on this platform (see app_exec_linux.go for Linux's).
func anyProcessRunning(names []string, installDir string) (bool, error) { return false, nil }
func killProcesses(names []string, installDir string) error             { return nil }

// newGameCommand builds the command used to launch the game's executable.
// The game only ships a Windows build, so this platform has no wine-style
// runtime wired in yet — it's kept only so app_game.go's LaunchGame compiles
// here too.
func newGameCommand(exePath, installDir string) (*exec.Cmd, error) {
	cmd := exec.Command(exePath)
	cmd.Dir = installDir
	return cmd, nil
}

// wineAvailable: not applicable outside Linux (see app_exec_linux.go).
func wineAvailable() bool { return true }
