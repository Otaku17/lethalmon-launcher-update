//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// applyLauncherUpdate runs inside a freshly downloaded launcher build,
// invoked by the old build's installLauncherUpdate with updateApplyFlag. It
// waits for targetExePath (the old build, currently exiting) to become
// replaceable — Windows keeps a running .exe locked until the process has
// actually exited — then moves itself into that location and hands off by
// launching the newly-installed exe.
func applyLauncherUpdate(targetExePath string) {
	selfPath, err := os.Executable()
	if err != nil {
		return
	}
	selfPath, err = filepath.EvalSymlinks(selfPath)
	if err != nil {
		return
	}

	const maxAttempts = 30
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := replaceExecutable(selfPath, targetExePath); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	cmd := exec.Command(targetExePath)
	_ = cmd.Start()
}

// replaceExecutable moves src onto dst. os.Rename alone would fail across
// drives (e.g. a %TEMP% download landing on a different volume than the
// install directory), so it falls back to a copy + remove in that case.
func replaceExecutable(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		return err
	}
	os.Remove(src)
	return nil
}
