package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// appConfig is the launcher's own persisted settings, stored as JSON at the
// path returned by configFilePath — distinct from the game's own .gameopts
// file (see app_gameopts.go).
type appConfig struct {
	InstallDir string `json:"installDir"`
}

// configFilePath returns where the launcher's config file lives, creating
// its parent directory if needed.
func configFilePath() (string, error) {
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

// loadConfig reads the persisted launcher config, returning a zero-value
// config (not an error) if none exists yet.
func loadConfig() appConfig {
	path, err := configFilePath()
	if err != nil {
		return appConfig{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return appConfig{}
	}

	var cfg appConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return appConfig{}
	}

	return cfg
}

// saveConfig persists cfg to disk as JSON, overwriting any previous config.
func saveConfig(cfg appConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
