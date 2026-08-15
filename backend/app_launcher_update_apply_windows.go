//go:build windows

package backend

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ApplyLauncherUpdate runs inside a freshly downloaded launcher build,
// invoked by the old build's installLauncherUpdate with UpdateApplyFlag. It
// waits for targetExePath (the old build, currently exiting) to become
// replaceable — Windows keeps a running .exe locked until the process has
// actually exited — then moves itself into that location and hands off by
// launching the newly-installed exe.
func ApplyLauncherUpdate(targetExePath string) {
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
		if err := ReplaceExecutable(selfPath, targetExePath); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	cmd := exec.Command(targetExePath)
	_ = cmd.Start()
}
