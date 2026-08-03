//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// installLauncherUpdate hands off to the freshly downloaded build at
// newExePath: it launches it with updateApplyFlag, which makes it briefly
// run headlessly (see applyLauncherUpdate) to wait out this process's exit,
// move itself into place over the current install, and then start
// normally.
//
// This avoids shelling out to a dropped .bat run through cmd.exe to do the
// file swap — a GUI app silently spawning a hidden shell that overwrites an
// executable is exactly the shape antivirus heuristics (including Windows
// Defender) flag as malware/ransomware-like behavior. Here the only thing
// spawned is another instance of the same launcher binary.
func installLauncherUpdate(newExePath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return err
	}

	cmd := exec.Command(newExePath, updateApplyFlag, currentExe)
	return cmd.Start()
}
