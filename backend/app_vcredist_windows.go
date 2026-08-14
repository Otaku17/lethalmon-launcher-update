//go:build windows

package backend

import (
	"fmt"
	"os"
	"os/exec"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

// vcRedistSignerOrg is the organization the Visual C++ Redistributable
// installer's Authenticode certificate must be issued to. Anything else —
// including a perfectly valid signature from some other publisher — is
// refused before the installer can run elevated.
const vcRedistSignerOrg = "Microsoft Corporation"

// vcRedistInstallerURL is Microsoft's permanent redirect to the latest x64
// Visual C++ Redistributable installer (2015-2022 line) — what the game's
// bundled Ruby runtime needs (VCRUNTIME140.dll, msvcp140.dll). Using the
// redirect instead of vendoring the installer means players always get
// whatever Microsoft currently considers safe to ship.
const vcRedistInstallerURL = "https://aka.ms/vs/17/release/vc_redist.x64.exe"

// vcRedistRegistryKey is where the redistributable's own installer records
// itself, so presence can be checked without running anything.
const vcRedistRegistryKey = `SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\X64`

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
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := a.downloadVCRedist(tmpPath); err != nil {
		return err
	}

	// The redirect target is Microsoft's own CDN over HTTPS, but TLS only
	// proves the connection wasn't tampered with in transit — it says
	// nothing about the file itself if that CDN or aka.ms redirect were
	// ever compromised. Checking the Authenticode signature confirms
	// what's about to run silently as an installer is genuinely
	// Microsoft's, not just "downloaded over a secure connection".
	if err := VerifyMicrosoftSignature(tmpPath); err != nil {
		return err
	}

	wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{Stage: "installing"})

	// The installer's manifest requires administrator privileges, and this
	// app normally runs unelevated: a plain exec.Command here would fail
	// immediately with ERROR_ELEVATION_REQUIRED instead of prompting for
	// consent, silently leaving the runtime uninstalled every time.
	if err := runElevated(tmpPath, "/install", "/quiet", "/norestart"); err != nil && !IsVCRedistSuccessExit(err) {
		return fmt.Errorf("vc_redist install failed: %w", err)
	}

	wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{Stage: "done", Percent: 100})

	return nil
}

// VerifyMicrosoftSignature checks that path carries a valid Authenticode
// signature issued to Microsoft Corporation, by calling Windows'
// WinVerifyTrust directly (see authenticode_windows.go). Guards against
// running a tampered or substituted installer if aka.ms's redirect target or
// Microsoft's CDN were ever compromised — HTTPS alone only proves the
// download wasn't altered in transit, not that it's genuinely Microsoft's
// file.
func VerifyMicrosoftSignature(path string) error {
	if err := VerifyAuthenticodeOrg(path, vcRedistSignerOrg); err != nil {
		return fmt.Errorf("vc_redist installer failed Microsoft signature verification: %w", err)
	}
	return nil
}

// IsVCRedistSuccessExit reports whether err from running the installer
// actually indicates success: 3010 means it succeeded but wants a reboot
// (irrelevant here — the DLLs it just registered are usable immediately),
// and 1638 means a newer version is already installed. Both are wins, not
// failures, but exec.Cmd.Run only returns nil for exit code 0.
func IsVCRedistSuccessExit(err error) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	code := exitErr.ExitCode()
	return code == 3010 || code == 1638
}

// downloadVCRedist downloads the installer into the file at destPath,
// emitting a vcRedistProgressEvent after every chunk read so the frontend
// can render a progress bar. See DownloadFile (app_download.go) for the
// retry/resume/idle-timeout behavior, shared with the game and launcher
// downloaders.
func (a *App) downloadVCRedist(destPath string) error {
	return DownloadFile(vcRedistInstallerURL, destPath, func(percent int) {
		wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{
			Stage:   "downloading",
			Percent: percent,
		})
	})
}
