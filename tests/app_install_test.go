package tests

import (
	"os"
	"path/filepath"
	"testing"

	"LethalmonLauncher/backend"
)

func TestPercentOf(t *testing.T) {
	cases := []struct {
		current, total, want int
	}{
		{0, 100, 0},
		{50, 100, 50},
		{100, 100, 100},
		{1, 3, 33},
		{2, 3, 66},
		// An empty archive or an empty migration is finished, not a division
		// by zero — this is what keeps a progress bar from crashing the app on
		// a zero-file operation.
		{0, 0, 100},
		{5, 0, 100},
	}

	for _, c := range cases {
		if got := backend.PercentOf(c.current, c.total); got != c.want {
			t.Errorf("backend.PercentOf(%d, %d) = %d, want %d", c.current, c.total, got, c.want)
		}
	}
}

func TestMoveFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.dat")
	dst := filepath.Join(dir, "moved", "dst.dat")

	const content = "save data that must survive the move"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir destination: %v", err)
	}

	if err := backend.MoveFile(src, dst); err != nil {
		t.Fatalf("backend.MoveFile() error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != content {
		t.Errorf("destination = %q, want %q", got, content)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source still exists after backend.MoveFile(), stat error = %v", err)
	}
}

// TestMoveFileOverwritesDestination covers a re-run of a migration that was
// interrupted: files already copied must be replaced, not refused.
func TestMoveFileOverwritesDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.dat")
	dst := filepath.Join(dir, "dst.dat")

	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(dst, []byte("stale leftover"), 0o644); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	if err := backend.MoveFile(src, dst); err != nil {
		t.Fatalf("backend.MoveFile() error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("destination = %q, want %q", got, "new")
	}
}

// TestUninstallKeepList states in code what UninstallGame promises in the UI:
// removing the game leaves the player's progress behind.
func TestUninstallKeepList(t *testing.T) {
	for _, name := range []string{"Saves", "GlobalHallOfFame.dat", "GlobalPokedex.dat"} {
		if !backend.UninstallKeepList[name] {
			t.Errorf("backend.UninstallKeepList is missing %q — uninstalling would delete player progress", name)
		}
	}

	for _, name := range []string{"Lethalmon.exe", "Data", "version.txt", ".gameopts"} {
		if backend.UninstallKeepList[name] {
			t.Errorf("backend.UninstallKeepList keeps %q, which should be removed on uninstall", name)
		}
	}
}
