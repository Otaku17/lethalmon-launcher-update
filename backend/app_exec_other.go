//go:build !windows

package backend

import "os/exec"

// hideWindow is a no-op on non-Windows platforms (no console flash issue there).
func hideWindow(cmd *exec.Cmd) {}

// anyProcessRunning / killProcesses: process-name based tracking is only
// implemented for Windows (the game only ships a .exe for now).
func anyProcessRunning(names []string, installDir string) (bool, error) { return false, nil }
func killProcesses(names []string, installDir string) error             { return nil }
