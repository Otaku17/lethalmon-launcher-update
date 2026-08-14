package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the launcher's own persisted settings, stored as JSON at the
// path returned by ConfigFilePath — distinct from the game's own .gameopts
// file (see app_gameopts.go).
type Config struct {
	InstallDir string `json:"installDir"`
}

// ConfigFilePath returns where the launcher's config file lives, creating
// its parent directory if needed.
func ConfigFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(configDir, "LethalmonLauncher")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads the persisted launcher config, returning a zero-value
// config (not an error) if none exists yet.
func LoadConfig() Config {
	path, err := ConfigFilePath()
	if err != nil {
		return Config{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}

	return cfg
}

// SaveConfig persists cfg to disk as JSON, overwriting any previous config.
func SaveConfig(cfg Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
