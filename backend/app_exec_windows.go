//go:build windows

package backend

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

// runElevated launches exePath with args under UAC elevation (the "runas"
// verb) and waits for it to exit, exposing its exit code the same way a
// plain exec.Cmd.Run would (nil on 0, *exec.ExitError otherwise). Needed for
// installers like vc_redist.x64.exe that are manifested
// requireAdministrator: launching those via a plain CreateProcess (what
// exec.Command does) fails immediately with ERROR_ELEVATION_REQUIRED when
// this app itself isn't running elevated, instead of prompting the user for
// consent the way double-clicking the installer would.
func runElevated(exePath string, args ...string) error {
	quotedArgs := make([]string, len(args))
	for i, arg := range args {
		quotedArgs[i] = "'" + strings.ReplaceAll(arg, "'", "''") + "'"
	}
	argList := ""
	if len(args) > 0 {
		argList = " -ArgumentList @(" + strings.Join(quotedArgs, ",") + ")"
	}

	script := fmt.Sprintf(
		"$p = Start-Process -FilePath '%s'%s -Verb RunAs -WindowStyle Hidden -Wait -PassThru; exit $p.ExitCode",
		strings.ReplaceAll(exePath, "'", "''"), argList,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	hideWindow(cmd)
	return cmd.Run()
}

// GameProcessFilter builds a PowerShell Where-Object expression matching
// processes by image name AND install directory (via ExecutablePath), so a
// same-named process running elsewhere on the machine (e.g. an unrelated
// ruby.exe) isn't mistaken for the game.
func GameProcessFilter(names []string, installDir string) string {
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
		GameProcessFilter(names, installDir)+" | Select-Object -ExpandProperty ProcessId")
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
		GameProcessFilter(names, installDir)+" | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }")
	hideWindow(cmd)
	return cmd.Run()
}
