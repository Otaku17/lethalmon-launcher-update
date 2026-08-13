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

	"LethalmonLauncher/internal/updatekey"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// verifyLauncherUpdateSignature downloads the signature file at
// signatureURL (see tools/updatesign) and checks it against data using the
// public key shared with the signing tool (see internal/updatekey).
func verifyLauncherUpdateSignature(signatureURL string, data []byte) error {
	pubKey, err := updatekey.Public()
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
	// companion asset (see tools/updatesign). Such a release cannot be
	// installed: UpdateLauncher refuses to run an update it can't verify.
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

	// Signature verification is mandatory, and checked before spending a
	// download on an artifact that would be rejected anyway. Treating a
	// missing ".sig" as "nothing to verify" would have made the whole
	// scheme optional in practice: anyone able to serve a modified release
	// asset could simply omit the signature to skip the check. The release
	// workflow always publishes a signature alongside the exe (see
	// .github/workflows/release.yml), so a release without one is a broken
	// release, not a normal case.
	if check.SignatureURL == "" {
		return fmt.Errorf("launcher update is not signed — refusing to install")
	}

	tmpFile, err := os.CreateTemp("", "lethalmon-launcher-update-*.exe")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	if err := a.downloadLauncherUpdate(check.DownloadURL, tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	tmpData, err := os.ReadFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := verifyLauncherUpdateSignature(check.SignatureURL, tmpData); err != nil {
		os.Remove(tmpPath)
		return err
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

// downloadLauncherUpdate downloads downloadURL into the file at destPath,
// emitting a launcherUpdateProgressEvent after every chunk read so the
// frontend can render a progress bar during the download stage. See
// downloadFile (app_download.go) for the retry/resume/idle-timeout behavior,
// shared with the game's own downloader.
func (a *App) downloadLauncherUpdate(downloadURL, destPath string) error {
	return downloadFile(downloadURL, destPath, func(percent int) {
		wailsruntime.EventsEmit(a.ctx, launcherUpdateProgressEvent, launcherUpdateProgress{
			Stage:   "downloading",
			Percent: percent,
		})
	})
}
