//go:build linux

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"LethalmonLauncher/backend"
)

// TestStageOnSameFilesystem covers the fix for the "text file busy" crash
// reported during real-world Linux auto-update: UpdateLauncher downloads
// into the OS temp directory, which is often a different filesystem from
// wherever the installed AppImage lives, so a plain os.Rename between the
// two fails and ReplaceExecutable's fallback tries to write straight into
// the running AppImage's own backing file — something the kernel refuses
// while it's still mapped and executing. Staging the download next to the
// destination first keeps the final rename on one filesystem.
func TestStageOnSameFilesystem(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	src := filepath.Join(srcDir, "downloaded.AppImage")
	const contents = "the freshly downloaded AppImage"
	if err := os.WriteFile(src, []byte(contents), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	staged, err := backend.StageOnSameFilesystem(src, dstDir)
	if err != nil {
		t.Fatalf("backend.StageOnSameFilesystem() error: %v", err)
	}
	defer os.Remove(staged)

	if filepath.Dir(staged) != dstDir {
		t.Errorf("staged file is in %q, want it inside %q so the later rename can't cross filesystems", filepath.Dir(staged), dstDir)
	}

	got, err := os.ReadFile(staged)
	if err != nil {
		t.Fatalf("read staged file: %v", err)
	}
	if string(got) != contents {
		t.Errorf("staged file contents = %q, want %q", got, contents)
	}

	info, err := os.Stat(staged)
	if err != nil {
		t.Fatalf("stat staged file: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("staged file mode = %v, want it executable so the relaunch after rename can run it", info.Mode())
	}

	// The whole point: the staged file and the destination directory now
	// share a filesystem (both under t.TempDir()'s parent), so the rename
	// ReplaceExecutable performs is a same-filesystem, atomic one.
	dst := filepath.Join(dstDir, "installed.AppImage")
	if err := backend.ReplaceExecutable(staged, dst); err != nil {
		t.Fatalf("backend.ReplaceExecutable() error: %v", err)
	}
	got, err = os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != contents {
		t.Errorf("installed contents = %q, want %q", got, contents)
	}
}

func TestStageOnSameFilesystemMissingSource(t *testing.T) {
	dir := t.TempDir()

	if _, err := backend.StageOnSameFilesystem(filepath.Join(dir, "absent.AppImage"), dir); err == nil {
		t.Fatal("backend.StageOnSameFilesystem() returned nil for a source that doesn't exist")
	}
}
