package main

import (
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

// gameProcessNames lists every process image name that can end up running
// the game: Lethalmon.exe is a small stub that spawns the Ruby runtime
// (ruby.exe/rubyw.exe) and then exits itself, so tracking only the stub
// would report the game as closed while it's actually still running.
var gameProcessNames = []string{gameExeName, "ruby.exe", "rubyw.exe"}

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
func (a *App) GetInstallDir() (string, error) {
	if cfg := loadConfig(); cfg.InstallDir != "" {
		if err := os.MkdirAll(cfg.InstallDir, 0o755); err != nil {
			return "", err
		}
		return cfg.InstallDir, nil
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

	// Best-effort: a failed check/install (offline, blocked download) isn't
	// fatal here — it just means the launch attempt below may fail exactly
	// as it would have without this call. Whereas skipping it entirely
	// left every player missing the runtime stuck on a cryptic
	// "VCRUNTIME140.dll is missing" system dialog with no fix in sight.
	_ = a.ensureVCRedist()

	cmd := exec.Command(exePath)
	cmd.Dir = installDir
	if err := cmd.Start(); err != nil {
		return err
	}

	go a.watchGameExit(installDir)

	return nil
}

// watchGameExit polls until none of the game's processes are running
// anymore, then notifies the frontend. A short initial delay lets the stub
// executable finish spawning the real Ruby runtime process.
func (a *App) watchGameExit(installDir string) {
	time.Sleep(1500 * time.Millisecond)

	for {
		running, err := anyProcessRunning(gameProcessNames, installDir)
		if err != nil || !running {
			break
		}
		time.Sleep(2 * time.Second)
	}

	wailsruntime.EventsEmit(a.ctx, gameExitedEvent)
}

// StopGame force-kills every running game process.
func (a *App) StopGame() error {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}
	return killProcesses(gameProcessNames, installDir)
}

// IsGameRunning reports whether any game process is currently running.
func (a *App) IsGameRunning() (bool, error) {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return false, err
	}
	return anyProcessRunning(gameProcessNames, installDir)
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

// SelectInstallFolder opens a native folder picker so the user can choose
// where to install the game, defaulting to the current install directory.
func (a *App) SelectInstallFolder() (string, error) {
	defaultDir, err := a.GetInstallDir()
	if err != nil {
		return "", err
	}

	selected, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Choisir le dossier d'installation",
		DefaultDirectory: defaultDir,
	})
	if err != nil {
		return "", err
	}

	return selected, nil
}
