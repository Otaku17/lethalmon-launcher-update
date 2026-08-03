package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const gameOptsFileName = ".gameopts"

// defaultGameOpts are injected for any key missing from the current
// .gameopts file (e.g. right after a fresh install, which only ships with
// --lang set) — same defaults as the launcher's own Settings page.
var defaultGameOpts = map[string]string{
	"lang":          "fr",
	"master_volume": "50",
	"auto_save":     "false",
	"scale":         "2",
}

// gameOptsPath returns the full path to the game's .gameopts file inside
// installDir.
func gameOptsPath(installDir string) string {
	return filepath.Join(installDir, gameOptsFileName)
}

// readGameOpts parses the .gameopts file into a key->value map (without the
// leading "--"). A missing file returns an empty map, not an error.
func readGameOpts(installDir string) (map[string]string, error) {
	data, err := os.ReadFile(gameOptsPath(installDir))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	opts := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "--") {
			continue
		}

		line = strings.TrimPrefix(line, "--")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		opts[parts[0]] = parts[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return opts, nil
}

// writeGameOpts persists opts to the .gameopts file, one "--key=value" per
// line (sorted by key for stable, diff-friendly output).
func writeGameOpts(installDir string, opts map[string]string) error {
	keys := make([]string, 0, len(opts))
	for k := range opts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("--%s=%s\n", k, opts[k]))
	}

	return os.WriteFile(gameOptsPath(installDir), []byte(sb.String()), 0o644)
}

// ensureGameOptsDefaults injects any default key missing from the current
// .gameopts (e.g. a fresh install only ships with --lang set), without
// touching keys that are already present.
func ensureGameOptsDefaults(installDir string) error {
	opts, err := readGameOpts(installDir)
	if err != nil {
		return err
	}

	changed := false
	for key, value := range defaultGameOpts {
		if _, exists := opts[key]; !exists {
			opts[key] = value
			changed = true
		}
	}

	if !changed {
		return nil
	}

	return writeGameOpts(installDir, opts)
}

// GetGameOpts returns the game's current options (from .gameopts), backing
// the launcher's Game settings so they read the real values instead of a
// separate local copy. Any key missing from the file (e.g. an install that
// predates a launcher setting) is backfilled with its default and persisted.
func (a *App) GetGameOpts() (map[string]string, error) {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return nil, err
	}

	if err := ensureGameOptsDefaults(installDir); err != nil {
		return nil, err
	}

	return readGameOpts(installDir)
}

// SetGameOpt updates a single game option in .gameopts, leaving every other
// existing option untouched.
func (a *App) SetGameOpt(key, value string) error {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return err
	}

	opts, err := readGameOpts(installDir)
	if err != nil {
		return err
	}

	opts[key] = value

	return writeGameOpts(installDir, opts)
}

// ResetGameOptsDefaults resets every known game option back to its default
// value (see defaultGameOpts), overwriting whatever is currently set —
// unlike ensureGameOptsDefaults, which only backfills missing keys. Returns
// the full options map afterwards so the frontend can refresh its fields.
func (a *App) ResetGameOptsDefaults() (map[string]string, error) {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return nil, err
	}

	opts, err := readGameOpts(installDir)
	if err != nil {
		return nil, err
	}

	for key, value := range defaultGameOpts {
		opts[key] = value
	}

	if err := writeGameOpts(installDir, opts); err != nil {
		return nil, err
	}

	return opts, nil
}
