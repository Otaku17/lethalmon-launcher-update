//go:build linux

package backend

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// hideWindow is a no-op on Linux (no console flash issue there).
func hideWindow(cmd *exec.Cmd) {}

// ErrWineNotFound is returned by newGameCommand when wine isn't installed.
// The frontend matches on this exact message to guide the player through
// installing it themselves, instead of failing the launch with an opaque
// "exec format error".
var ErrWineNotFound = errors.New("wine_not_found")

// newGameCommand builds the command used to launch the game's Windows
// executable under wine — the game itself only ships a Windows build (see
// GameProcessNames), wine is what makes running it on Linux possible at all.
//
// Unlike the Visual C++ Redistributable on Windows (see ensureVCRedist),
// wine isn't fetched and installed automatically: it's a system-level
// package tied to the distro's package manager rather than a redistributable
// DLL, so the player is expected to install it themselves. ErrWineNotFound
// lets the frontend guide them through that instead of failing silently.
func newGameCommand(exePath, installDir string) (*exec.Cmd, error) {
	winePath, err := exec.LookPath("wine")
	if err != nil {
		return nil, ErrWineNotFound
	}

	cmd := exec.Command(winePath, exePath)
	cmd.Dir = installDir

	if LoadConfig().ForceDiscreteGpu {
		// DRI_PRIME covers Mesa (AMD/Intel) PRIME render offload; the other
		// two cover NVIDIA's proprietary-driver equivalent. Setting all
		// three is harmless on setups where they don't apply — the ones
		// that don't match the installed driver are simply ignored.
		cmd.Env = append(os.Environ(),
			"DRI_PRIME=1",
			"__NV_PRIME_RENDER_OFFLOAD=1",
			"__GLX_VENDOR_LIBRARY_NAME=nvidia",
		)
	}

	return cmd, nil
}

// commPIDs returns the PIDs of every running process whose kernel-reported
// name (/proc/<pid>/comm) matches one of names, case-insensitively. Wine
// preserves the original Windows executable's name here — running
// "Lethalmon.exe" under wine still shows up with comm "Lethalmon.exe", the
// same way a native Linux binary would — so no wine-specific detection is
// needed to track the game's processes.
func commPIDs(names []string) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[strings.ToLower(n)] = true
	}

	var pids []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			// Gone since ReadDir, or not ours to read — either way, not a
			// match.
			continue
		}

		if wanted[strings.ToLower(strings.TrimSpace(string(comm)))] {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}

// anyProcessRunning reports whether any process with one of the given image
// names is currently running.
//
// installDir is unused on Linux: unlike Windows' Win32_Process, there's no
// fixed format to recover a wine process's original filesystem path from
// (its cmdline reflects wine's own path translation, which varies). Since
// this launcher only ever runs one game, matching on process name alone is
// an acceptable trade-off.
func anyProcessRunning(names []string, installDir string) (bool, error) {
	pids, err := commPIDs(names)
	if err != nil {
		return false, err
	}
	return len(pids) > 0, nil
}

// killProcesses force-kills every running process matching any of the given
// image names. See anyProcessRunning for why installDir isn't used to scope
// the match on Linux.
func killProcesses(names []string, installDir string) error {
	pids, err := commPIDs(names)
	if err != nil {
		return err
	}

	for _, pid := range pids {
		// Best-effort: a process that already exited between commPIDs and
		// here isn't an error condition.
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	return nil
}
