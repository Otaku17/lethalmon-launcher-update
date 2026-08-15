package backend

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// gameExeName is the game's executable filename, used to locate an existing
// install and to target the right process for GPU preference forcing.
const gameExeName = "Lethalmon.exe"

// GameProcessNames lists every process image name that can end up running
// the game: Lethalmon.exe is a small stub that spawns the Ruby runtime
// (ruby.exe/rubyw.exe) and then exits itself, so tracking only the stub
// would report the game as closed while it's actually still running.
var GameProcessNames = []string{gameExeName, "ruby.exe", "rubyw.exe"}

// gameExitedEvent is emitted once none of the game's processes are running
// anymore, whether the user closed the game or it was killed via StopGame.
const gameExitedEvent = "game:exited"

// legacyInstallDir returns the install location used by the previous
// (Electron-based) launcher, so existing installs are picked up automatically
// instead of asking the user to reinstall.
func legacyInstallDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "com.lethalmon-launcher.app", "game", "release"), nil
}

// GetInstallDir returns the directory where the game is installed,
// creating it if it doesn't exist yet. Honors a custom location persisted
// via MoveInstallDir; otherwise falls back to an existing install from the
// previous launcher if found, then to the default location.
//
// A persisted custom location that's no longer usable (e.g. it now points
// at a file instead of a folder, or its parent is no longer accessible —
// this has happened with a stray "D:\Config.Msi" value) is self-healed by
// clearing it from the config, rather than leaving the launcher permanently
// broken until someone manually edits/deletes config.json.
func (a *App) GetInstallDir() (string, error) {
	if cfg := LoadConfig(); cfg.InstallDir != "" {
		if info, statErr := os.Stat(cfg.InstallDir); statErr == nil && !info.IsDir() {
			cfg.InstallDir = ""
			SaveConfig(cfg)
		} else if err := os.MkdirAll(cfg.InstallDir, 0o755); err == nil {
			return cfg.InstallDir, nil
		} else {
			cfg.InstallDir = ""
			SaveConfig(cfg)
		}
	}

	if legacyDir, err := legacyInstallDir(); err == nil {
		if _, statErr := os.Stat(filepath.Join(legacyDir, gameExeName)); statErr == nil {
			return legacyDir, nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	installDir := filepath.Join(homeDir, "LethalmonLauncher", "Game")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return "", err
	}

	return installDir, nil
}

// ResetInstallPath clears any custom install location persisted via
// MoveInstallDir, so GetInstallDir falls back to the legacy or default
// location. Exposed as a manual escape hatch for a corrupted/inaccessible
// custom path, so recovering no longer requires editing or deleting
// config.json by hand.
func (a *App) ResetInstallPath() (string, error) {
	cfg := LoadConfig()
	cfg.InstallDir = ""
	if err := SaveConfig(cfg); err != nil {
		return "", err
	}
	return a.GetInstallDir()
}

// GetGameVersion reads the installed game's version from its version.txt
// file. Returns an empty string (no error) if the game isn't installed.
func (a *App) GetGameVersion() (string, error) {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(installDir, "version.txt"))
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(data)), nil
}

// LaunchGame starts a new instance of the installed game's executable.
func (a *App) LaunchGame() error {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	exePath := filepath.Join(installDir, gameExeName)
	if _, err := os.Stat(exePath); err != nil {
		return err
	}

	// Best-effort: a failed check/install (offline, blocked download, UAC
	// prompt declined) isn't fatal here — it just means the launch attempt
	// below may fail exactly as it would have without this call. But it's
	// reported via vcRedistProgressEvent (stage "failed") so the frontend
	// can at least tell the player why, instead of leaving them to guess
	// after the game crashes with a bare "VCRUNTIME140.dll is missing"
	// system dialog.
	if err := a.ensureVCRedist(); err != nil {
		wailsruntime.EventsEmit(a.ctx, vcRedistProgressEvent, vcRedistProgress{Stage: "failed", Error: err.Error()})
	}

	if err := StampLauncherEdition(installDir); err != nil {
		return err
	}

	cmd, err := newGameCommand(exePath, installDir)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	go a.watchGameExit(installDir)

	return nil
}

// gameStartupGrace bounds how long watchGameExit waits for one of
// GameProcessNames to first appear, so a launch that genuinely fails (crash,
// missing dependency) doesn't leave the launcher reporting the game as
// running forever. It's generous mainly for Linux's sake: a wine prefix
// that's never been used before bootstraps itself (wineboot, first-run
// device setup) before the game's actual process starts, which can take
// well over the couple of seconds a native Windows launch needs — long
// enough that a short, fixed delay mistook that bootstrap time for "the game
// already closed" and made the Stop button disappear a few seconds into the
// very first launch after installing wine.
const gameStartupGrace = 60 * time.Second

// watchGameExit polls until none of the game's processes are running
// anymore, then notifies the frontend. It doesn't report the game as exited
// just because none of GameProcessNames has shown up yet — only once one of
// them has been seen running (the normal case), or gameStartupGrace has
// passed without that ever happening (the launch failed outright).
func (a *App) watchGameExit(installDir string) {
	started := false
	deadline := time.Now().Add(gameStartupGrace)

	for {
		time.Sleep(1500 * time.Millisecond)

		running, err := anyProcessRunning(GameProcessNames, installDir)
		if err != nil {
			break
		}
		if running {
			started = true
		} else if started || time.Now().After(deadline) {
			break
		}
	}

	wailsruntime.EventsEmit(a.ctx, gameExitedEvent)
}

// StopGame force-kills every running game process.
func (a *App) StopGame() error {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}
	return killProcesses(GameProcessNames, installDir)
}

// IsGameRunning reports whether any game process is currently running.
func (a *App) IsGameRunning() (bool, error) {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return false, err
	}
	return anyProcessRunning(GameProcessNames, installDir)
}

// OpenInstallFolder opens the game's install directory in the OS file explorer.
func (a *App) OpenInstallFolder() error {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", installDir)
	case "darwin":
		cmd = exec.Command("open", installDir)
	default:
		cmd = exec.Command("xdg-open", installDir)
	}

	return cmd.Start()
}

// ErrOneDrivePath is returned by SelectInstallFolder when the user picks a
// folder synced by OneDrive. The frontend matches on this exact message to
// show a translated, user-friendly explanation instead of a raw Go error.
var ErrOneDrivePath = errors.New("onedrive_path_not_allowed")

// IsOneDrivePath reports whether path lives inside a OneDrive-synced folder.
// OneDrive locks/renames files mid-sync and can silently break the game's
// save files and updater, so installing there is not supported.
func IsOneDrivePath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = strings.ToLower(abs)

	// Primary check: compare against the OneDrive sync-root env vars Windows
	// sets for personal and work/school accounts.
	for _, envVar := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		root := os.Getenv(envVar)
		if root == "" {
			continue
		}
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			rootAbs = root
		}
		rootAbs = strings.ToLower(rootAbs)
		if abs == rootAbs || strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) {
			return true
		}
	}

	// Fallback: catch OneDrive folders on other drives/accounts (e.g.
	// "D:\OneDrive - CompanyName") that the env vars above don't cover.
	for _, part := range strings.Split(abs, string(filepath.Separator)) {
		if strings.HasPrefix(part, "onedrive") {
			return true
		}
	}

	return false
}

// folderPickerFallbackTitle stands in when the frontend passes an empty
// title to SelectInstallFolder. It's in English to match i18n's fallbackLng,
// and exists only so a missing translation key degrades to a readable dialog
// instead of an untitled one.
const folderPickerFallbackTitle = "Choose the installation folder"

// SelectInstallFolder opens a native folder picker so the user can choose
// where to install the game, defaulting to the current install directory.
//
// The title is supplied by the frontend rather than defined here: this dialog
// is the one piece of launcher UI drawn by the OS instead of by React, so a
// string hardcoded in Go would stay stuck in one language while the rest of
// the interface follows the user's choice.
func (a *App) SelectInstallFolder(title string) (string, error) {
	defaultDir, err := a.GetInstallDir()
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(title) == "" {
		title = folderPickerFallbackTitle
	}

	selected, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: defaultDir,
	})
	if err != nil {
		return "", err
	}
	if selected == "" {
		return "", nil
	}

	if IsOneDrivePath(selected) {
		return "", ErrOneDrivePath
	}

	return selected, nil
}

// LauncherEditionGameOptKey is the .gameopts key stamped on every launch to
// mark the install as started by this (Wails-based) launcher. The previous
// Electron-based launcher never wrote this key, so the game can read its
// presence (via PARGV.game_opt in Ruby, which re-reads .gameopts from disk
// on every call) to distinguish "launched by the new launcher" from a
// legacy-launcher or manual launch — e.g. to gate a launcher-exclusive
// Mystery Gift.
const LauncherEditionGameOptKey = "launcher_edition"

// StampLauncherEdition writes "true" into .gameopts under
// LauncherEditionGameOptKey, unconditionally overwriting any previous value.
// Unlike EnsureGameOptsDefaults, this isn't a user preference to preserve —
// it must always reflect which launcher actually started the game this time.
func StampLauncherEdition(installDir string) error {
	opts, err := ReadGameOpts(installDir)
	if err != nil {
		return err
	}

	opts[LauncherEditionGameOptKey] = "true"

	return WriteGameOpts(installDir, opts)
}
