package tests

import (
	"os"
	"path/filepath"
	"testing"

	"LethalmonLauncher/backend"
)

// useTempConfigDir points os.UserConfigDir at a scratch directory so config
// tests never read or clobber the real launcher settings.
func useTempConfigDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("AppData", dir)         // Windows
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux, so the suite runs off-Windows too

	return dir
}

func TestConfigRoundTrip(t *testing.T) {
	useTempConfigDir(t)

	want := backend.Config{InstallDir: filepath.Join("D:", "Games", "Lethalmon")}
	if err := backend.SaveConfig(want); err != nil {
		t.Fatalf("backend.SaveConfig() error: %v", err)
	}

	if got := backend.LoadConfig(); got.InstallDir != want.InstallDir {
		t.Errorf("backend.LoadConfig().InstallDir = %q, want %q", got.InstallDir, want.InstallDir)
	}
}

func TestLoadConfigWithoutFile(t *testing.T) {
	useTempConfigDir(t)

	if got := backend.LoadConfig(); got.InstallDir != "" {
		t.Errorf("backend.LoadConfig() on first run = %+v, want the zero value", got)
	}
}

// TestLoadConfigWithCorruptFile pins the deliberate choice to swallow a
// corrupt config rather than surface it: a truncated or hand-edited
// config.json puts the launcher back to first-run defaults, where
// GetInstallDir can recover, instead of failing every call that needs the
// install directory.
func TestLoadConfigWithCorruptFile(t *testing.T) {
	dir := useTempConfigDir(t)

	path := filepath.Join(dir, "LethalmonLauncher", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for _, corrupt := range []string{"", "{", "not json at all", `{"installDir":`} {
		if err := os.WriteFile(path, []byte(corrupt), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		if got := backend.LoadConfig(); got.InstallDir != "" {
			t.Errorf("backend.LoadConfig() with %q = %+v, want the zero value", corrupt, got)
		}
	}
}

func TestConfigFilePathCreatesParentDir(t *testing.T) {
	useTempConfigDir(t)

	path, err := backend.ConfigFilePath()
	if err != nil {
		t.Fatalf("backend.ConfigFilePath() error: %v", err)
	}

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("backend.ConfigFilePath() did not create its parent directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("%s is not a directory", filepath.Dir(path))
	}
}
