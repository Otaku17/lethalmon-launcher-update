package backend

import "encoding/json"

// launcherVersion holds the launcher's own version. It is injected by main()
// rather than read here, because the version lives in frontend/package.json —
// one source of truth shared with the frontend, instead of two strings kept in
// sync by hand — and //go:embed cannot reach outside the directory of the file
// that declares it. The embed therefore stays in package main at the
// repository root and hands the parsed value over through SetLauncherVersion.
var launcherVersion string

// SetLauncherVersion records the running launcher's version. Call it once at
// startup, before anything can reach GetLauncherVersionCheck: left empty, the
// version check treats the launcher as having no version at all and reports an
// update as available on every run.
func SetLauncherVersion(version string) {
	launcherVersion = version
}

// ParseLauncherVersion extracts the "version" field from the contents of
// frontend/package.json, returning an empty string if it can't be parsed.
func ParseLauncherVersion(packageJSON []byte) string {
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageJSON, &pkg); err != nil {
		return ""
	}
	return pkg.Version
}

// GetLauncherVersion returns the launcher's own version (see launcherVersion).
func (a *App) GetLauncherVersion() string {
	return launcherVersion
}
