package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// launcherUpdatePublicKeyB64 is the public half of the ed25519 keypair used
// to sign launcher update artifacts (see tools/updatesign). It only lets
// this binary VERIFY that a downloaded update was signed by whoever holds
// the matching private key — it can't be used to sign anything, so
// embedding it here is safe. This protects against a compromised GitHub
// account or a tampered release asset serving a modified executable; it's
// independent of (and doesn't replace) Authenticode code signing, which is
// about OS-level trust (SmartScreen/Defender), not update integrity.
const launcherUpdatePublicKeyB64 = "E1rqwTfPYmMI0R0bMMR3NuMegdfDkUpoWxO6xZsMSS0="

// launcherUpdatePublicKey decodes launcherUpdatePublicKeyB64 into a usable
// ed25519 key, validating its length.
func launcherUpdatePublicKey() (ed25519.PublicKey, error) {
	key, err := base64.StdEncoding.DecodeString(launcherUpdatePublicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid embedded public key: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid embedded public key size: %d", len(key))
	}
	return ed25519.PublicKey(key), nil
}

// verifyLauncherUpdateSignature downloads the signature file at
// signatureURL (see tools/updatesign) and checks it against data using the
// embedded public key.
func verifyLauncherUpdateSignature(signatureURL string, data []byte) error {
	pubKey, err := launcherUpdatePublicKey()
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(signatureURL)
	if err != nil {
		return fmt.Errorf("fetch update signature: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch update signature: HTTP %d", resp.StatusCode)
	}

	sigB64, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read update signature: %w", err)
	}

	signature, err := base64.StdEncoding.DecodeString(string(sigB64))
	if err != nil {
		return fmt.Errorf("invalid update signature encoding: %w", err)
	}

	if !ed25519.Verify(pubKey, data, signature) {
		return fmt.Errorf("update signature verification failed — refusing to install")
	}

	return nil
}

// LauncherVersionCheck reports whether a newer launcher build than the
// currently running one is available for download.
type LauncherVersionCheck struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl,omitempty"`
	// SignatureURL is empty when the latest GitHub release has no ".sig"
	// companion asset (see tools/updatesign) — signature verification is
	// then simply skipped, not treated as an error.
	SignatureURL string `json:"signatureUrl,omitempty"`
}

// GetLauncherVersionCheck compares the running launcher's own version
// against the latest release in launcherReleasesRepo, so the launcher can
// prompt the user to update itself — the exact same GitHub-releases-based
// approach GetGameVersionCheck uses for the game.
func (a *App) GetLauncherVersionCheck() (LauncherVersionCheck, error) {
	current := launcherVersion

	releases, err := fetchReleases(launcherReleasesRepo)
	if err != nil {
		return LauncherVersionCheck{}, err
	}
	if len(releases) == 0 {
		return LauncherVersionCheck{CurrentVersion: current}, nil
	}

	latestRelease := releases[0]
	latest := trimVersion(latestRelease.TagName)

	var downloadURL, signatureURL string
	for _, asset := range latestRelease.Assets {
		switch {
		case strings.HasSuffix(asset.Name, ".exe.sig"):
			signatureURL = asset.BrowserDownloadURL
		case strings.HasSuffix(asset.Name, ".exe"):
			downloadURL = asset.BrowserDownloadURL
		}
	}

	return LauncherVersionCheck{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: current == "" || compareVersions(latest, current) > 0,
		DownloadURL:     downloadURL,
		SignatureURL:    signatureURL,
	}, nil
}

const launcherUpdateProgressEvent = "launcher:update-progress"

type launcherUpdateProgress struct {
	// Stage is one of "downloading" or "installing".
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
}

// updateApplyFlag, when passed as os.Args[1] (see main.go), switches this
// binary into a small headless "apply update" mode instead of starting the
// GUI: see applyLauncherUpdate. This is how the self-update installs itself
// without any .bat/cmd.exe helper — the freshly downloaded build briefly
// runs itself in this mode to wait out the old process's exit, move itself
// into the install location, and hand off to the newly-installed exe.
const updateApplyFlag = "--lethalmon-update-apply"

// UpdateLauncher downloads the latest launcher build and installs it in
// place of the currently running executable, then relaunches and quits this
// process. The actual file swap happens inside the freshly downloaded build
// itself, briefly running headlessly (see installLauncherUpdate and
// applyLauncherUpdate) because the OS keeps the running executable locked —
// it can only be replaced once this process has exited.
func (a *App) UpdateLauncher() error {
	check, err := a.GetLauncherVersionCheck()
	if err != nil {
		return err
	}
	if check.DownloadURL == "" {
		return fmt.Errorf("no downloadable launcher update found")
	}

	tmpFile, err := os.CreateTemp("", "lethalmon-launcher-update-*.exe")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if err := a.downloadLauncherUpdate(check.DownloadURL, tmpFile); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	tmpFile.Close()

	// Signature verification is opportunistic: the release only has a
	// ".sig" asset when one was uploaded alongside the exe (see
	// tools/updatesign). If present, a bad signature aborts the update; if
	// absent, we simply skip the check rather than blocking installs on
	// optional metadata.
	if check.SignatureURL != "" {
		tmpData, err := os.ReadFile(tmpPath)
		if err != nil {
			os.Remove(tmpPath)
			return err
		}

		if err := verifyLauncherUpdateSignature(check.SignatureURL, tmpData); err != nil {
			os.Remove(tmpPath)
			return err
		}
	}

	wailsruntime.EventsEmit(a.ctx, launcherUpdateProgressEvent, launcherUpdateProgress{
		Stage:   "installing",
		Percent: 100,
	})

	if err := installLauncherUpdate(tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	wailsruntime.Quit(a.ctx)
	return nil
}

// downloadLauncherUpdate streams downloadURL into dst, emitting a
// launcherUpdateProgressEvent after every chunk read so the frontend can
// render a progress bar during the download stage.
func (a *App) downloadLauncherUpdate(downloadURL string, dst *os.File) error {
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Get(downloadURL)
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

			wailsruntime.EventsEmit(a.ctx, launcherUpdateProgressEvent, launcherUpdateProgress{
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
