//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// hideWindow prevents a console window from flashing open when launching a
// console subprocess (e.g. powershell) from this GUI app on Windows.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// gameProcessFilter builds a PowerShell Where-Object expression matching
// processes by image name AND install directory (via ExecutablePath), so a
// same-named process running elsewhere on the machine (e.g. an unrelated
// ruby.exe) isn't mistaken for the game.
func gameProcessFilter(names []string, installDir string) string {
	quotedNames := make([]string, len(names))
	for i, n := range names {
		quotedNames[i] = "'" + strings.ReplaceAll(n, "'", "''") + "'"
	}
	safeDir := strings.ReplaceAll(installDir, "'", "''")

	return fmt.Sprintf(
		"Get-CimInstance Win32_Process | Where-Object { @(%s) -contains $_.Name -and $_.ExecutablePath -like '%s*' }",
		strings.Join(quotedNames, ","), safeDir,
	)
}

// anyProcessRunning reports whether any process with one of the given image
// names is currently running from within installDir.
func anyProcessRunning(names []string, installDir string) (bool, error) {
	if installDir == "" {
		return false, nil
	}

	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		gameProcessFilter(names, installDir)+" | Select-Object -ExpandProperty ProcessId")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(string(out)) != "", nil
}

// killProcesses force-kills every process matching any of the given image
// names running from within installDir.
func killProcesses(names []string, installDir string) error {
	if installDir == "" {
		return nil
	}

	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		gameProcessFilter(names, installDir)+" | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }")
	hideWindow(cmd)
	return cmd.Run()
}
