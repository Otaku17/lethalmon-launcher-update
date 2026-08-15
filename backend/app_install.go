package backend

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	// ResolveInstallDir is idempotent (it recognizes a path that already
	// ends with GameInstallSubfolder and leaves it alone), so this is safe
	// to apply even when the frontend already resolved newPath itself (see
	// SelectInstallFolder)  this is the actual choke point that decides
	// what cfg.InstallDir (and therefore every other install/uninstall
	// operation) ends up pointing at, so it can't rely on every caller
	// having done the right thing first.
	newPath = ResolveInstallDir(newPath)
	oldPath = filepath.Clean(oldPath)

	if newPath == oldPath {
		return nil
	}

	if running, _ := anyProcessRunning(GameProcessNames, oldPath); running {
		return errors.New("the game is currently running, close it before moving the installation")
	}

	if err := MoveInstallFiles(oldPath, newPath, func(current, total int, file string) {
		wailsruntime.EventsEmit(a.ctx, installMigrationEvent, migrationProgress{
			Current: current,
			Total:   total,
			Percent: PercentOf(current, total),
			File:    file,
		})
	}); err != nil {
		return err
	}

	cfg := LoadConfig()
	cfg.InstallDir = newPath
	return SaveConfig(cfg)
}

// MoveInstallFiles moves every file from oldPath into newPath, mirroring the
// directory structure, then cleans up oldPath. Split out from MoveInstallDir
// so the actual file-migration logic — including the newPath-inside-oldPath
// case (see isWithin) that caused a previous version of this to delete the
// destination it had just moved files into — can be exercised by tests
// without a live Wails context: wailsruntime.EventsEmit (unlike this
// function's onProgress) calls log.Fatal, killing the whole process, when
// given one that wasn't set up by the actual running app (see a.ctx). Pass a
// nil onProgress to skip progress reporting entirely.
func MoveInstallFiles(oldPath, newPath string, onProgress func(current, total int, file string)) error {
	// Captured before newPath is created below: when newPath lands inside
	// oldPath (see isWithin's doc comment for when that happens), cleanup at
	// the end must only remove what was actually here before the move, never
	// the freshly created destination sitting inside it.
	originalEntries, err := os.ReadDir(oldPath)
	if err != nil {
		return err
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

		if onProgress != nil {
			onProgress(i+1, total, rel)
		}
	}

	if isWithin(newPath, oldPath) {
		// newPath lives inside oldPath itself — the common case for any
		// pre-ResolveInstallDir custom install, where "moving" just applies
		// GameInstallSubfolder to the same base folder the player was
		// already using. oldPath can't be removed as a whole without
		// deleting the destination right along with it, so only the entries
		// that existed there before the move (captured above, before newPath
		// was created) are removed.
		for _, entry := range originalEntries {
			os.RemoveAll(filepath.Join(oldPath, entry.Name()))
		}
	} else {
		os.RemoveAll(oldPath)
	}

	return nil
}

// isWithin reports whether path is target itself or a descendant of it —
// e.g. moving the install from "D:\Games" to "D:\Games\lethalmon_game"
// (ResolveInstallDir applying the isolation subfolder to a base folder the
// player already had files directly in) lands newPath inside oldPath.
func isWithin(path, target string) bool {
	path = filepath.Clean(path)
	target = filepath.Clean(target)
	return path == target || strings.HasPrefix(path, target+string(os.PathSeparator))
}

// UninstallKeepList lists the save-related files/folders that UninstallGame
// preserves, so removing the game doesn't wipe the player's progress.
var UninstallKeepList = map[string]bool{
	"Saves":                 true,
	"GlobalHallOfFame.dat":  true,
	"GlobalPokedex.dat":     true,
	"GlobalPokedex.dat.bak": true,
}

// GameInstallEntries lists every top-level file/folder the game itself ships
// (everything in its release .zip, i.e. GameProcessNames plus its data
// folders and support DLL). UninstallGame only ever deletes entries in this
// list  an allowlist rather than "not in UninstallKeepList"  so an install
// directory that predates GameInstallSubfolder (shared with files the player
// already had there before ResolveInstallDir started isolating new installs)
// still can't lose anything but the game's own files on uninstall. A game
// update that ships a new top-level file/folder not listed here just leaves
// it behind uninstalled rather than deleted  an occasional stray leftover
// is a far better failure mode than deleting a player's own files, which is
// exactly the incident this list exists to prevent from happening again.
var GameInstallEntries = map[string]bool{
	"audio":              true,
	"Data":               true,
	"Fonts":              true,
	"graphics":           true,
	"lib":                true,
	"psdk":               true,
	"ruby_builtin_dlls":  true,
	".gameopts":          true,
	"Game.rb":            true,
	"Game.yarb":          true,
	gameExeName:          true,
	"msvcrt-ruby300.dll": true,
	"ruby.exe":           true,
	"rubyw.exe":          true,
	"version.txt":        true,
}

// UninstallGame removes every game file from the install directory except
// save data (see UninstallKeepList), leaving the (now near-empty) folder in
// place. Only entries recognized as belonging to the game (see
// GameInstallEntries) are ever touched  anything else found alongside the
// game (a player's own files, if their install directory isn't an isolated
// GameInstallSubfolder) is left exactly where it was.
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
		if UninstallKeepList[entry.Name()] || !GameInstallEntries[entry.Name()] {
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
