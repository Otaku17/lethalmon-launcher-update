package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"LethalmonLauncher/backend"
)

func TestReadGameOptsMissingFile(t *testing.T) {
	opts, err := backend.ReadGameOpts(t.TempDir())
	if err != nil {
		t.Fatalf("backend.ReadGameOpts() on a fresh directory errored: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("backend.ReadGameOpts() = %v, want an empty map", opts)
	}
}

func TestGameOptsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := map[string]string{
		"lang":          "it",
		"master_volume": "80",
		"auto_save":     "true",
		"scale":         "3",
	}

	if err := backend.WriteGameOpts(dir, want); err != nil {
		t.Fatalf("backend.WriteGameOpts() error: %v", err)
	}

	got, err := backend.ReadGameOpts(dir)
	if err != nil {
		t.Fatalf("backend.ReadGameOpts() error: %v", err)
	}

	for k, v := range want {
		if got[k] != v {
			t.Errorf("opts[%q] = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("read back %d options, wrote %d", len(got), len(want))
	}
}

// TestWriteGameOptsIsStable matters because .gameopts is a file the game reads
// and the player may open: writing it in a different order every time would
// make every save look like a change.
func TestWriteGameOptsIsStable(t *testing.T) {
	dir := t.TempDir()
	opts := map[string]string{"scale": "2", "lang": "fr", "auto_save": "false"}

	if err := backend.WriteGameOpts(dir, opts); err != nil {
		t.Fatalf("backend.WriteGameOpts() error: %v", err)
	}
	first, err := os.ReadFile(backend.GameOptsPath(dir))
	if err != nil {
		t.Fatalf("read .gameopts: %v", err)
	}

	if err := backend.WriteGameOpts(dir, opts); err != nil {
		t.Fatalf("backend.WriteGameOpts() second call error: %v", err)
	}
	second, err := os.ReadFile(backend.GameOptsPath(dir))
	if err != nil {
		t.Fatalf("read .gameopts: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("writing the same options twice produced different files:\n%q\n%q", first, second)
	}

	want := "--auto_save=false\n--lang=fr\n--scale=2\n"
	if string(first) != want {
		t.Errorf(".gameopts = %q, want %q", first, want)
	}
}

func TestReadGameOptsParsing(t *testing.T) {
	dir := t.TempDir()

	content := strings.Join([]string{
		"--lang=fr",
		"",
		"   --scale=2   ",
		"# a comment the game ignores",
		"not-a-flag",
		"--broken-without-value",
		// A value containing "=" must survive intact: SplitN(…, 2) exists
		// precisely so a path or an expression isn't cut at its first "=".
		"--custom=a=b=c",
		"--empty=",
	}, "\n")

	if err := os.WriteFile(backend.GameOptsPath(dir), []byte(content), 0o644); err != nil {
		t.Fatalf("write .gameopts: %v", err)
	}

	opts, err := backend.ReadGameOpts(dir)
	if err != nil {
		t.Fatalf("backend.ReadGameOpts() error: %v", err)
	}

	want := map[string]string{
		"lang":   "fr",
		"scale":  "2",
		"custom": "a=b=c",
		"empty":  "",
	}

	if len(opts) != len(want) {
		t.Errorf("parsed %d options (%v), want %d", len(opts), opts, len(want))
	}
	for k, v := range want {
		if opts[k] != v {
			t.Errorf("opts[%q] = %q, want %q", k, opts[k], v)
		}
	}
}

// TestEnsureGameOptsDefaultsPreservesExisting is the guard against a settings
// reset on update: backfilling defaults must never overwrite a choice the
// player already made.
func TestEnsureGameOptsDefaultsPreservesExisting(t *testing.T) {
	dir := t.TempDir()

	// What a fresh install ships with: only --lang.
	if err := backend.WriteGameOpts(dir, map[string]string{"lang": "it"}); err != nil {
		t.Fatalf("backend.WriteGameOpts() error: %v", err)
	}

	if err := backend.EnsureGameOptsDefaults(dir); err != nil {
		t.Fatalf("backend.EnsureGameOptsDefaults() error: %v", err)
	}

	opts, err := backend.ReadGameOpts(dir)
	if err != nil {
		t.Fatalf("backend.ReadGameOpts() error: %v", err)
	}

	if opts["lang"] != "it" {
		t.Errorf("lang = %q, want the player's existing %q", opts["lang"], "it")
	}
	for key, def := range backend.DefaultGameOpts {
		if key == "lang" {
			continue
		}
		if opts[key] != def {
			t.Errorf("opts[%q] = %q, want the default %q", key, opts[key], def)
		}
	}
}

func TestEnsureGameOptsDefaultsIsIdempotent(t *testing.T) {
	dir := t.TempDir()

	if err := backend.EnsureGameOptsDefaults(dir); err != nil {
		t.Fatalf("first backend.EnsureGameOptsDefaults() error: %v", err)
	}
	first, err := os.ReadFile(backend.GameOptsPath(dir))
	if err != nil {
		t.Fatalf("read .gameopts: %v", err)
	}

	if err := backend.EnsureGameOptsDefaults(dir); err != nil {
		t.Fatalf("second backend.EnsureGameOptsDefaults() error: %v", err)
	}
	second, err := os.ReadFile(backend.GameOptsPath(dir))
	if err != nil {
		t.Fatalf("read .gameopts: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("a second call rewrote the file:\n%q\n%q", first, second)
	}
}

// TestStampLauncherEditionOverwrites covers the opposite rule to the defaults
// above: launcher_edition is not a preference to preserve, it records which
// launcher started the game this time and must be rewritten on every launch.
func TestStampLauncherEditionOverwrites(t *testing.T) {
	dir := t.TempDir()

	if err := backend.WriteGameOpts(dir, map[string]string{
		"lang":                            "fr",
		backend.LauncherEditionGameOptKey: "false",
	}); err != nil {
		t.Fatalf("backend.WriteGameOpts() error: %v", err)
	}

	if err := backend.StampLauncherEdition(dir); err != nil {
		t.Fatalf("backend.StampLauncherEdition() error: %v", err)
	}

	opts, err := backend.ReadGameOpts(dir)
	if err != nil {
		t.Fatalf("backend.ReadGameOpts() error: %v", err)
	}

	if opts[backend.LauncherEditionGameOptKey] != "true" {
		t.Errorf("%s = %q, want %q", backend.LauncherEditionGameOptKey, opts[backend.LauncherEditionGameOptKey], "true")
	}
	if opts["lang"] != "fr" {
		t.Errorf("lang = %q, want the untouched %q", opts["lang"], "fr")
	}
}

func TestStampLauncherEditionOnFreshInstall(t *testing.T) {
	dir := t.TempDir()

	if err := backend.StampLauncherEdition(dir); err != nil {
		t.Fatalf("backend.StampLauncherEdition() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, backend.GameOptsFileName)); err != nil {
		t.Fatalf("backend.StampLauncherEdition() did not create .gameopts: %v", err)
	}

	opts, err := backend.ReadGameOpts(dir)
	if err != nil {
		t.Fatalf("backend.ReadGameOpts() error: %v", err)
	}
	if opts[backend.LauncherEditionGameOptKey] != "true" {
		t.Errorf("%s = %q, want %q", backend.LauncherEditionGameOptKey, opts[backend.LauncherEditionGameOptKey], "true")
	}
}
