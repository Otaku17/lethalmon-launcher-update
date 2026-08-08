//go:build windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

// vcRedistInstallerURL is Microsoft's permanent redirect to the latest x64
// Visual C++ Redistributable installer (2015-2022 line) — what the game's
// bundled Ruby runtime needs (VCRUNTIME140.dll, msvcp140.dll). Using the
// redirect instead of vendoring the installer means players always get
// whatever Microsoft currently considers safe to ship.
const vcRedistInstallerURL = "https://aka.ms/vs/17/release/vc_redist.x64.exe"

// vcRedistRegistryKey is where the redistributable's own installer records
// itself, so presence can be checked without running anything.
const vcRedistRegistryKey = `SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\X64`

// vcRedistProgressEvent reports "downloading" (with percent tracked as
// bytes arrive) then "installing" (percent unknown — the silent installer
// gives no progress feedback) while ensureVCRedist fetches and runs the
// installer. Not emitted at all when the redistributable is already
// present, which is the common case on every launch after the first.
const vcRedistProgressEvent = "game:vcredist-progress"

type vcRedistProgress struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
}

// vcRedistInstalled reports whether the x64 Visual C++ Redistributable is
// already present, via the registry key its installer writes.
func vcRedistInstalled() bool {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, vcRedistRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	installed, _, err := key.GetIntegerValue("Installed")
	return err == nil && installed == 1
}

// ensureVCRedist silently installs the Visual C++ Redistributable if it
// isn't present yet. Without it, players see a bare "VCRUNTIME140.dll is
// missing" system dialog instead of the game starting, with no indication
// of what actually went wrong or how to fix it.
func (a *App) ensureVCRedist() error {
	if vcRedistInstalled() {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "vc_redist-*.exe")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := a.downloadVCRedist(tmpFile); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{Stage: "installing"})

	cmd := exec.Command(tmpPath, "/install", "/quiet", "/norestart")
	hideWindow(cmd)
	if err := cmd.Run(); err != nil && !isVCRedistSuccessExit(err) {
		return fmt.Errorf("vc_redist install failed: %w", err)
	}

	wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{Stage: "done", Percent: 100})

	return nil
}

// isVCRedistSuccessExit reports whether err from running the installer
// actually indicates success: 3010 means it succeeded but wants a reboot
// (irrelevant here — the DLLs it just registered are usable immediately),
// and 1638 means a newer version is already installed. Both are wins, not
// failures, but exec.Cmd.Run only returns nil for exit code 0.
func isVCRedistSuccessExit(err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	code := exitErr.ExitCode()
	return code == 3010 || code == 1638
}

// downloadVCRedist streams the installer into dst, emitting a
// vcRedistProgressEvent after every chunk read so the frontend can render a
// progress bar (mirrors downloadWithProgress in app_download.go).
func (a *App) downloadVCRedist(dst *os.File) error {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(vcRedistInstallerURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 64*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}

			written += int64(n)
			percent := 0
			if total > 0 {
				percent = int(written * 100 / total)
			}

			wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{
				Stage:   "downloading",
				Percent: percent,
			})
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}
