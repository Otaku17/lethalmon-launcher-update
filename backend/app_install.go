package backend

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const installMigrationEvent = "install:migration-progress"

type migrationProgress struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Percent int    `json:"percent"`
	File    string `json:"file"`
}

// MoveInstallDir moves the game's current installation (if any) to newPath
// and persists it as the new install location. Progress is reported via the
// "install:migration-progress" event as files are migrated.
func (a *App) MoveInstallDir(newPath string) error {
	oldPath, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	newPath = filepath.Clean(newPath)
	oldPath = filepath.Clean(oldPath)

	if newPath == oldPath {
		return nil
	}

	if running, _ := anyProcessRunning(GameProcessNames, oldPath); running {
		return errors.New("the game is currently running, close it before moving the installation")
	}

	if err := os.MkdirAll(newPath, 0o755); err != nil {
		return err
	}

	var files []string
	err = filepath.WalkDir(oldPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	total := len(files)
	for i, src := range files {
		rel, err := filepath.Rel(oldPath, src)
		if err != nil {
			return err
		}
		dst := filepath.Join(newPath, rel)

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := MoveFile(src, dst); err != nil {
			return err
		}

		wailsruntime.EventsEmit(a.ctx, installMigrationEvent, migrationProgress{
			Current: i + 1,
			Total:   total,
			Percent: PercentOf(i+1, total),
			File:    rel,
		})
	}

	os.RemoveAll(oldPath)

	cfg := LoadConfig()
	cfg.InstallDir = newPath
	return SaveConfig(cfg)
}

// UninstallKeepList lists the save-related files/folders that UninstallGame
// preserves, so removing the game doesn't wipe the player's progress.
var UninstallKeepList = map[string]bool{
	"Saves":                true,
	"GlobalHallOfFame.dat": true,
	"GlobalPokedex.dat":    true,
}

// UninstallGame removes every file from the game's install directory except
// save data (see UninstallKeepList), leaving the (now near-empty) folder in
// place.
func (a *App) UninstallGame() error {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	if running, _ := anyProcessRunning(GameProcessNames, installDir); running {
		return errors.New("the game is currently running, close it before uninstalling")
	}

	entries, err := os.ReadDir(installDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if UninstallKeepList[entry.Name()] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(installDir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

// PercentOf returns current as a percentage of total, treating a total of
// zero as already complete rather than dividing by zero.
func PercentOf(current, total int) int {
	if total == 0 {
		return 100
	}
	return current * 100 / total
}

// MoveFile relocates a single file, falling back to a copy+delete when a
// plain rename isn't possible (e.g. moving across drives, or the source
// being briefly locked). Failing to delete the now-copied source file is
// not treated as fatal: the file was successfully migrated either way, and
// the leftover original gets a final cleanup pass via os.RemoveAll(oldPath).
func MoveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	os.Remove(src)
	return nil
}
