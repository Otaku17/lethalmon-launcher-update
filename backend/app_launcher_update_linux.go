//go:build linux

package backend

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
//
// That only holds for an actual rename(2), though. UpdateLauncher downloads
// the new build into the OS temp directory (see os.CreateTemp in
// app_launcher_update.go), which is commonly a different filesystem from
// wherever the AppImage itself lives (e.g. tmpfs vs. ~/Downloads) —
// os.Rename can't cross that boundary, so ReplaceExecutable's fallback takes
// over and opens currentPath for writing directly. currentPath is this
// process's own executable, still mapped and running, and Linux refuses
// that open() with ETXTBSY ("text file busy"). StageOnSameFilesystem below
// avoids the fallback entirely by making sure the rename source and
// destination always share a filesystem.
func installLauncherUpdate(newExePath string) error {
	// AppImage's own runtime sets this to the absolute path of the .AppImage
	// file it mounted itself from — the one thing on disk that actually
	// needs replacing, as opposed to wherever this process's code happens to
	// be running from (a squashfs mount under /tmp, not a real file).
	currentPath := os.Getenv("APPIMAGE")
	if currentPath == "" {
		return fmt.Errorf("not running from an AppImage (APPIMAGE environment variable is unset) — self-update isn't available outside one")
	}

	staged, err := StageOnSameFilesystem(newExePath, filepath.Dir(currentPath))
	if err != nil {
		return err
	}
	defer os.Remove(staged)

	if err := ReplaceExecutable(staged, currentPath); err != nil {
		return err
	}

	cmd := exec.Command(currentPath)
	return cmd.Start()
}

// StageOnSameFilesystem copies src into a new file inside dir — fsyncing it
// before returning — so that a subsequent os.Rename from the returned path
// is guaranteed to land on the same filesystem as dir and therefore complete
// as a single atomic rename(2), never the read+write+remove fallback in
// ReplaceExecutable that can't touch a file currently being executed.
// Exported for tests (see tests/app_launcher_update_linux_test.go).
func StageOnSameFilesystem(src, dir string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.CreateTemp(dir, ".lethalmon-launcher-update-*.AppImage")
	if err != nil {
		return "", err
	}
	stagedPath := out.Name()

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(stagedPath)
		return "", err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(stagedPath)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(stagedPath)
		return "", err
	}
	if err := os.Chmod(stagedPath, 0o755); err != nil {
		os.Remove(stagedPath)
		return "", err
	}

	return stagedPath, nil
}
