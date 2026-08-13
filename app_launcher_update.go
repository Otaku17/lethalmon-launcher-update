package main

import (
	"context"
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

// downloadLauncherUpdate downloads downloadURL into the file at destPath
// (which must already exist, see os.CreateTemp), emitting a
// launcherUpdateProgressEvent after every chunk read so the frontend can
// render a progress bar during the download stage.
//
// If an attempt stalls (see downloadIdleTimeout, shared with the game's own
// downloader in app_download.go) or otherwise fails, it is retried up to
// downloadMaxAttempts times, resuming from the bytes already written instead
// of starting over — see downloadLauncherUpdateAttempt.
func (a *App) downloadLauncherUpdate(downloadURL, destPath string) error {
	var lastErr error

	for attempt := 1; attempt <= downloadMaxAttempts; attempt++ {
		if err := a.downloadLauncherUpdateAttempt(downloadURL, destPath); err != nil {
			lastErr = err
			if attempt < downloadMaxAttempts {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("download failed after %d attempts: %w", downloadMaxAttempts, lastErr)
}

// downloadLauncherUpdateAttempt makes a single attempt at downloading
// downloadURL, appending to (or resuming) whatever is already at destPath
// via a Range request. The connection is aborted if no data arrives for
// downloadIdleTimeout, covering both an unresponsive server and a body read
// that stalls partway through — either way the caller retries rather than
// hanging indefinitely or being cut off by an arbitrary total-duration cap.
func (a *App) downloadLauncherUpdateAttempt(downloadURL, destPath string) error {
	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	offset, err := out.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	idleTimer := time.AfterFunc(downloadIdleTimeout, cancel)
	defer idleTimer.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("connection stalled (no response within %s)", downloadIdleTimeout)
		}
		return err
	}
	defer resp.Body.Close()
	idleTimer.Reset(downloadIdleTimeout)

	written := offset
	var total int64

	switch resp.StatusCode {
	case http.StatusOK:
		// The server sent the full body instead of honoring our Range
		// request, so any partial data we had is useless — start over.
		if offset > 0 {
			if err := out.Truncate(0); err != nil {
				return err
			}
			if _, err := out.Seek(0, io.SeekStart); err != nil {
				return err
			}
			written = 0
		}
		total = resp.ContentLength
	case http.StatusPartialContent:
		total = offset + resp.ContentLength
	default:
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			idleTimer.Reset(downloadIdleTimeout)

			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
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
			return nil
		}
		if readErr != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("connection stalled (no data for %s)", downloadIdleTimeout)
			}
			return readErr
		}
	}
}
