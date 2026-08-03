package main

import (
	_ "embed"
	"encoding/json"
)

// frontendPackageJSON is embedded at compile time so the Go binary and the
// frontend share a single source of truth for the launcher's own version
// (frontend/package.json), instead of keeping two version strings in sync
// by hand.
//
//go:embed frontend/package.json
var frontendPackageJSON []byte

// launcherVersion holds the launcher's own version, read once at package
// init time from the embedded frontend/package.json.
var launcherVersion = parseLauncherVersion()

// parseLauncherVersion extracts the "version" field from the embedded
// frontend/package.json, returning an empty string if it can't be parsed.
func parseLauncherVersion() string {
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(frontendPackageJSON, &pkg); err != nil {
		return ""
	}
	return pkg.Version
}

// GetLauncherVersion returns the launcher's own version (see launcherVersion).
func (a *App) GetLauncherVersion() string {
	return launcherVersion
}
