//go:build linux

package backend

import (
	"fmt"
	"os"
	"os/exec"
)

// installLauncherUpdate replaces the running AppImage with the freshly
// downloaded one and relaunches it, entirely within this call.
//
// Unlike Windows (see app_launcher_update_windows.go / _apply_windows.go),
// this needs none of the "spawn a headless instance of the new build to
// wait out this process's exit, then have that instance move itself into
// place" dance: Linux doesn't lock a running executable's file on disk — the
// kernel keeps serving the old file's pages to this already-running process
// even after its directory entry is repointed by rename — so the swap can
// happen directly here, in the process that's still running from the file
// being replaced.
func installLauncherUpdate(newExePath string) error {
	// AppImage's own runtime sets this to the absolute path of the .AppImage
	// file it mounted itself from — the one thing on disk that actually
	// needs replacing, as opposed to wherever this process's code happens to
	// be running from (a squashfs mount under /tmp, not a real file).
	currentPath := os.Getenv("APPIMAGE")
	if currentPath == "" {
		return fmt.Errorf("not running from an AppImage (APPIMAGE environment variable is unset) — self-update isn't available outside one")
	}

	if err := os.Chmod(newExePath, 0o755); err != nil {
		return err
	}

	if err := ReplaceExecutable(newExePath, currentPath); err != nil {
		return err
	}

	cmd := exec.Command(currentPath)
	return cmd.Start()
}
