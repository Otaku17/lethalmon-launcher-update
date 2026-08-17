package backend

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"LethalmonLauncher/internal/updatekey"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// VerifyLauncherUpdateSignature downloads the signature file at
// signatureURL (see tools/updatesign) and checks it against data using the
// public key shared with the signing tool (see internal/updatekey).
func VerifyLauncherUpdateSignature(signatureURL string, data []byte) error {
	pubKey, err := updatekey.Public()
	if err != nil {
		return err
	}

	sigB64, err := FetchUpdateSignature(signatureURL)
	if err != nil {
		return err
	}

	return VerifyUpdateSignature(pubKey, data, sigB64)
}

// FetchUpdateSignature downloads the base64-encoded signature published
// alongside a launcher release.
func FetchUpdateSignature(signatureURL string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(signatureURL)
	if err != nil {
		return nil, fmt.Errorf("fetch update signature: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch update signature: HTTP %d", resp.StatusCode)
	}

	sigB64, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read update signature: %w", err)
	}

	return sigB64, nil
}

// VerifyUpdateSignature checks data against a base64-encoded ed25519
// signature. It's split out from the download above so the decision itself —
// the one step standing between a downloaded .exe and it replacing the
// running launcher — can be exercised by tests without a network round trip.
//
// Surrounding whitespace is tolerated because the signature travels as a text
// file: a stray trailing newline (from an editor, a CI step, a text-mode
// transfer) would otherwise fail base64 decoding and surface as "signature
// verification failed", pointing at a compromised artifact when the real
// problem is formatting.
func VerifyUpdateSignature(pubKey ed25519.PublicKey, data, sigB64 []byte) error {
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
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
	latestRelease, ok := LatestStableRelease(releases)
	if !ok {
		return LauncherVersionCheck{CurrentVersion: current}, nil
	}

	latest := TrimVersion(latestRelease.TagName)

	downloadURL, signatureURL := PickLauncherAssets(latestRelease.Assets)

	return LauncherVersionCheck{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: current == "" || CompareVersions(latest, current) > 0,
		DownloadURL:     downloadURL,
		SignatureURL:    signatureURL,
	}, nil
}

// launcherAssetSuffix is the file extension release.yml publishes this
// platform's launcher build under: ".exe" on Windows, ".AppImage" on Linux —
// matching whatever installLauncherUpdate on this platform actually knows
// how to apply (see app_launcher_update_windows.go / _linux.go).
func launcherAssetSuffix() string {
	if goruntime.GOOS == "linux" {
		return ".AppImage"
	}
	return ".exe"
}

// PickLauncherAssets picks this platform's launcher build and its detached
// signature out of a release's assets (see launcherAssetSuffix).
func PickLauncherAssets(assets []ReleaseAsset) (downloadURL, signatureURL string) {
	return PickLauncherAssetsForSuffix(assets, launcherAssetSuffix())
}

// PickLauncherAssetsForSuffix is PickLauncherAssets with the file extension
// as a parameter, so both platforms' matching behavior can be covered by
// tests regardless of which platform actually runs them.
//
// The "<suffix>.sig" case is matched first and the two are never allowed to
// collapse into one another: mistaking the signature file for the executable
// (or the reverse) wouldn't fail loudly, it would silently verify an artifact
// against itself and hand a bogus "update" to installLauncherUpdate. Returning
// an empty downloadURL or signatureURL instead makes UpdateLauncher refuse the
// release, which is the outcome an incomplete release should have.
func PickLauncherAssetsForSuffix(assets []ReleaseAsset, suffix string) (downloadURL, signatureURL string) {
	for _, asset := range assets {
		switch {
		case strings.HasSuffix(asset.Name, suffix+".sig"):
			signatureURL = asset.BrowserDownloadURL
		case strings.HasSuffix(asset.Name, suffix):
			downloadURL = asset.BrowserDownloadURL
		}
	}
	return downloadURL, signatureURL
}

const launcherUpdateProgressEvent = "launcher:update-progress"

type launcherUpdateProgress struct {
	// Stage is one of "downloading" or "installing".
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
}

// UpdateApplyFlag, when passed as os.Args[1] (see main.go), switches this
// binary into a small headless "apply update" mode instead of starting the
// GUI: see ApplyLauncherUpdate. This is how the self-update installs itself
// without any .bat/cmd.exe helper — the freshly downloaded build briefly
// runs itself in this mode to wait out the old process's exit, move itself
// into the install location, and hand off to the newly-installed exe.
const UpdateApplyFlag = "--lethalmon-update-apply"

// UpdateLauncher downloads the latest launcher build and installs it in
// place of the currently running executable, then relaunches and quits this
// process. The actual file swap happens inside the freshly downloaded build
// itself, briefly running headlessly (see installLauncherUpdate and
// ApplyLauncherUpdate) because the OS keeps the running executable locked —
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

	tmpFile, err := os.CreateTemp("", "lethalmon-launcher-update-*"+launcherAssetSuffix())
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

	if err := VerifyLauncherUpdateSignature(check.SignatureURL, tmpData); err != nil {
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

// ReplaceExecutable moves src onto dst. os.Rename alone would fail across
// drives (e.g. a %TEMP% download landing on a different volume than the
// install directory), so it falls back to a copy + remove in that case.
//
// Shared between app_launcher_update_apply_windows.go (run from a freshly
// downloaded build, after the old one has actually exited and released its
// file lock) and app_launcher_update_linux.go (run directly against the
// still-running process's own executable, which Linux — unlike Windows —
// allows replacing on disk without waiting for it to exit first).
func ReplaceExecutable(src, dst string) error {
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

// downloadLauncherUpdate downloads downloadURL into the file at destPath,
// emitting a launcherUpdateProgressEvent after every chunk read so the
// frontend can render a progress bar during the download stage. See
// DownloadFile (app_download.go) for the retry/resume/idle-timeout behavior,
// shared with the game's own downloader.
func (a *App) downloadLauncherUpdate(downloadURL, destPath string) error {
	return DownloadFile(downloadURL, destPath, func(percent int) {
		wailsruntime.EventsEmit(a.ctx, launcherUpdateProgressEvent, launcherUpdateProgress{
			Stage:   "downloading",
			Percent: percent,
		})
	})
}
