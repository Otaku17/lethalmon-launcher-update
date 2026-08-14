package backend

import (
	"archive/zip"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const downloadProgressEvent = "install:download-progress"

// DownloadIdleTimeout bounds how long we'll wait without receiving any data
// (including the initial connection/response) before treating the attempt as
// stalled. It is deliberately not an overall timeout: a slow-but-steady
// connection is allowed to take as long as it needs to finish, only a
// connection that stops delivering bytes gets cut off.
//
// A variable rather than a constant so tests can shrink it — the stall and
// retry paths are the ones worth covering, and at production values a test
// exercising them would spend minutes asleep.
var DownloadIdleTimeout = 30 * time.Second

// DownloadMaxAttempts is how many times a stalled or failed download attempt
// is retried (resuming via a Range request, see downloadFileAttempt) before
// giving up entirely.
const DownloadMaxAttempts = 6

// DownloadRetryDelay is how long to wait before retrying attempt n, backing
// off further each time so a server having a bad minute isn't hammered.
// Variable for the same reason as DownloadIdleTimeout.
var DownloadRetryDelay = func(attempt int) time.Duration {
	return time.Duration(attempt) * 2 * time.Second
}

type downloadProgress struct {
	// Stage is one of "downloading", "extracting" or "done".
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
}

// DownloadGame fetches the game's latest GitHub release .zip and extracts it
// into the current install directory (see GetInstallDir), reporting
// progress via the "install:download-progress" event.
func (a *App) DownloadGame() error {
	check, err := a.GetGameVersionCheck()
	if err != nil {
		return err
	}
	if check.DownloadURL == "" {
		return fmt.Errorf("no downloadable release found for the game")
	}

	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	if running, _ := anyProcessRunning(GameProcessNames, installDir); running {
		return fmt.Errorf("the game is currently running, close it before (re)installing")
	}

	tmpFile, err := os.CreateTemp("", "lethalmon-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := a.downloadWithProgress(check.DownloadURL, tmpPath); err != nil {
		return err
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	if err := a.extractZipWithProgress(tmpPath, installDir); err != nil {
		return err
	}

	versionFile := filepath.Join(installDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(check.LatestVersion), 0o644); err != nil {
		return err
	}

	// A fresh install only ships --lang in .gameopts: fill in the rest
	// with the same defaults as the launcher's own Settings page.
	if err := EnsureGameOptsDefaults(installDir); err != nil {
		return err
	}

	wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{Stage: "done", Percent: 100})

	return nil
}

// RepairGame redownloads the game's latest release .zip and re-extracts only
// the files that are missing or whose contents don't match the archive
// (corrupted/manually edited files), leaving everything else untouched.
// Progress is reported via the same "install:download-progress" event as
// DownloadGame, with a "repairing" stage in place of "extracting".
func (a *App) RepairGame() error {
	check, err := a.GetGameVersionCheck()
	if err != nil {
		return err
	}
	if check.DownloadURL == "" {
		return fmt.Errorf("no downloadable release found for the game")
	}

	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	if running, _ := anyProcessRunning(GameProcessNames, installDir); running {
		return fmt.Errorf("the game is currently running, close it before repairing")
	}

	tmpFile, err := os.CreateTemp("", "lethalmon-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := a.downloadWithProgress(check.DownloadURL, tmpPath); err != nil {
		return err
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	if err := a.repairFromZip(tmpPath, installDir); err != nil {
		return err
	}

	versionFile := filepath.Join(installDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(check.LatestVersion), 0o644); err != nil {
		return err
	}

	wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{Stage: "done", Percent: 100})

	return nil
}

// repairFromZip extracts each entry of the zip at zipPath into destDir only
// if the file on disk is missing or its CRC32 doesn't match the archive's
// copy, emitting a downloadProgressEvent (stage "repairing") after every
// entry so the frontend can render progress across the whole archive.
func (a *App) repairFromZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	cleanDest := filepath.Clean(destDir)
	total := len(reader.File)

	for i, file := range reader.File {
		targetPath, err := SafeExtractPath(cleanDest, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		} else if !FileMatchesZipEntry(targetPath, file) {
			if err := extractZipFile(file, targetPath); err != nil {
				return err
			}
		}

		wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{
			Stage:   "repairing",
			Percent: PercentOf(i+1, total),
		})
	}

	return nil
}

// FileMatchesZipEntry reports whether the file at path already has the same
// size and CRC32 checksum as the given zip entry, meaning it doesn't need to
// be re-extracted.
func FileMatchesZipEntry(path string, entry *zip.File) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || uint64(info.Size()) != entry.UncompressedSize64 {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, f); err != nil {
		return false
	}

	return hash.Sum32() == entry.CRC32
}

// downloadWithProgress downloads url into the file at destPath, emitting a
// downloadProgressEvent after every chunk read so the frontend can render a
// progress bar. See DownloadFile for the retry/resume/idle-timeout behavior.
func (a *App) downloadWithProgress(url, destPath string) error {
	return DownloadFile(url, destPath, func(percent int) {
		wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{
			Stage:   "downloading",
			Percent: percent,
		})
	})
}

// DownloadFile downloads url into the file at destPath (which must already
// exist, see os.CreateTemp), calling onProgress after every chunk read so
// callers can render a progress bar. Shared between the game download (see
// downloadWithProgress) and the launcher self-update (see
// downloadLauncherUpdate in app_launcher_update.go).
//
// If an attempt stalls (see DownloadIdleTimeout) or otherwise fails, it is
// retried up to DownloadMaxAttempts times. Each retry resumes from the bytes
// already written instead of starting over, so a flaky connection only ever
// has to re-fetch the tail end it lost.
func DownloadFile(url, destPath string, onProgress func(percent int)) error {
	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	var lastErr error

	for attempt := 1; attempt <= DownloadMaxAttempts; attempt++ {
		err := downloadFileAttempt(url, out, onProgress)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < DownloadMaxAttempts {
			time.Sleep(DownloadRetryDelay(attempt))
		}
	}

	return fmt.Errorf("download failed after %d attempts: %w", DownloadMaxAttempts, lastErr)
}

// downloadFileAttempt makes a single attempt at downloading url, appending to
// (or resuming) whatever is already in out via a Range request. The
// connection is aborted if no data arrives for DownloadIdleTimeout, which
// covers both an unresponsive server and a body read that stalls partway
// through — either way the caller retries rather than hanging indefinitely
// or being cut off by an arbitrary total-duration cap.
func downloadFileAttempt(url string, out *os.File, onProgress func(percent int)) error {
	offset, err := out.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Reset on every successful read (and once more after headers arrive);
	// firing cancels ctx, which unblocks whatever read is currently in
	// flight — the connection attempt, the response headers, or the body.
	idleTimer := time.AfterFunc(DownloadIdleTimeout, cancel)
	defer idleTimer.Stop()

	// stallOr turns a request/read error into a clearer message when it was
	// actually caused by the idle timeout firing.
	stallOr := func(err error) error {
		if ctx.Err() != nil {
			return fmt.Errorf("connection stalled after %s of inactivity", DownloadIdleTimeout)
		}
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return stallOr(err)
	}
	defer resp.Body.Close()
	idleTimer.Reset(DownloadIdleTimeout)

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
			idleTimer.Reset(DownloadIdleTimeout)

			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}

			written += int64(n)
			percent := 0
			if total > 0 {
				percent = int(written * 100 / total)
			}
			onProgress(percent)
		}

		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return stallOr(readErr)
		}
	}
}

// extractZipWithProgress extracts every entry of the zip at zipPath into
// destDir, emitting a downloadProgressEvent after each file so the frontend
// can render extraction progress separately from the download itself.
func (a *App) extractZipWithProgress(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	cleanDest := filepath.Clean(destDir)
	total := len(reader.File)

	for i, file := range reader.File {
		targetPath, err := SafeExtractPath(cleanDest, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		} else {
			if err := extractZipFile(file, targetPath); err != nil {
				return err
			}
		}

		wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{
			Stage:   "extracting",
			Percent: PercentOf(i+1, total),
		})
	}

	return nil
}

// SafeExtractPath resolves a zip entry's name against destDir, rejecting any
// entry that would land outside it — the "zip slip" attack, where entry names
// like "../../evil" make a naive extractor overwrite files anywhere the
// process can write. The game archive is fetched from a GitHub release rather
// than from the player, but the whole point of verifying what gets installed
// is not assuming the artifact is what it should be.
//
// Shared by extractZipWithProgress and repairFromZip so the two extraction
// paths can't drift apart on the one check that keeps them contained.
func SafeExtractPath(destDir, entryName string) (string, error) {
	cleanDest := filepath.Clean(destDir)
	targetPath := filepath.Join(cleanDest, entryName)

	if targetPath != cleanDest && !strings.HasPrefix(targetPath, cleanDest+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid file path in archive: %s", entryName)
	}

	return targetPath, nil
}

// extractZipFile writes a single zip entry's contents to targetPath,
// creating any missing parent directories first.
func extractZipFile(file *zip.File, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
