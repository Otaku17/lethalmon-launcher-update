package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const downloadProgressEvent = "install:download-progress"

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

	if running, _ := anyProcessRunning(gameProcessNames, installDir); running {
		return fmt.Errorf("the game is currently running, close it before (re)installing")
	}

	tmpFile, err := os.CreateTemp("", "lethalmon-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := a.downloadWithProgress(check.DownloadURL, tmpFile); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

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
	if err := ensureGameOptsDefaults(installDir); err != nil {
		return err
	}

	wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{Stage: "done", Percent: 100})

	return nil
}

// downloadWithProgress streams url into dst, emitting a downloadProgressEvent
// after every chunk read so the frontend can render a progress bar.
func (a *App) downloadWithProgress(url string, dst *os.File) error {
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Get(url)
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

			wailsruntime.EventsEmit(a.ctx, downloadProgressEvent, downloadProgress{
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
		targetPath := filepath.Join(cleanDest, file.Name)

		if targetPath != cleanDest && !strings.HasPrefix(targetPath, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", file.Name)
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
			Percent: percentOf(i+1, total),
		})
	}

	return nil
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
