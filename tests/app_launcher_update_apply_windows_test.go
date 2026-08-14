//go:build windows

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"LethalmonLauncher/backend"
)

// TestReplaceExecutable covers the last step of a self-update: the freshly
// downloaded build moving itself over the installed one. If this silently
// half-succeeded, the player would be left with a launcher that no longer
// starts, and no way back short of a manual reinstall.
func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "new.exe")
	dst := filepath.Join(dir, "installed.exe")

	const newBuild = "the freshly downloaded build"
	if err := os.WriteFile(src, []byte(newBuild), 0o755); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(dst, []byte("the old build"), 0o755); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	if err := backend.ReplaceExecutable(src, dst); err != nil {
		t.Fatalf("backend.ReplaceExecutable() error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != newBuild {
		t.Errorf("installed build = %q, want %q", got, newBuild)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("temporary download still present after the swap, stat error = %v", err)
	}
}

// TestReplaceExecutableWhenDestinationMissing covers an install directory that
// lost its exe (an antivirus quarantine, a partial uninstall): the update
// should still land rather than error out.
func TestReplaceExecutableWhenDestinationMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "new.exe")
	dst := filepath.Join(dir, "installed.exe")

	if err := os.WriteFile(src, []byte("build"), 0o755); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := backend.ReplaceExecutable(src, dst); err != nil {
		t.Fatalf("backend.ReplaceExecutable() error: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Errorf("destination missing after backend.ReplaceExecutable(): %v", err)
	}
}

func TestReplaceExecutableMissingSource(t *testing.T) {
	dir := t.TempDir()

	err := backend.ReplaceExecutable(filepath.Join(dir, "absent.exe"), filepath.Join(dir, "installed.exe"))
	if err == nil {
		t.Fatal("backend.ReplaceExecutable() returned nil for a source that doesn't exist")
	}
}

// TestUpdateApplyFlagIsDistinctive guards the flag main() dispatches on: it
// has to be something no shell or shortcut would pass by accident, since
// seeing it makes the launcher skip its GUI entirely and overwrite an
// executable.
func TestUpdateApplyFlagIsDistinctive(t *testing.T) {
	if backend.UpdateApplyFlag != "--lethalmon-update-apply" {
		t.Errorf("backend.UpdateApplyFlag = %q; changing it breaks self-updates from any already-installed build", backend.UpdateApplyFlag)
	}
}
