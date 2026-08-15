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
	for _, name := range []string{"Saves", "GlobalHallOfFame.dat", "GlobalPokedex.dat", "GlobalPokedex.dat.bak"} {
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

// TestGameInstallEntries pins the exact top-level entries a released game
// ships (see the screenshot in the incident this list was added for),
// keeping GameInstallEntries and UninstallKeepList honest about which one a
// given name belongs to.
func TestGameInstallEntries(t *testing.T) {
	for _, name := range []string{
		"audio", "Data", "Fonts", "graphics", "lib", "psdk", "ruby_builtin_dlls",
		".gameopts", "Game.rb", "Game.yarb",
		"Lethalmon.exe", "msvcrt-ruby300.dll", "ruby.exe", "rubyw.exe", "version.txt",
	} {
		if !backend.GameInstallEntries[name] {
			t.Errorf("backend.GameInstallEntries is missing %q — UninstallGame would leave it behind instead of removing it", name)
		}
	}

	// Save data belongs in UninstallKeepList, not here: GameInstallEntries is
	// only consulted for what to delete, and these must never be deleted.
	for _, name := range []string{"Saves", "GlobalHallOfFame.dat", "GlobalPokedex.dat", "GlobalPokedex.dat.bak"} {
		if backend.GameInstallEntries[name] {
			t.Errorf("backend.GameInstallEntries lists %q, which belongs in UninstallKeepList instead", name)
		}
	}
}

// TestMoveInstallFilesIntoOwnSubfolder is the regression test for a second
// incident: a pre-existing custom install (bare, no GameInstallSubfolder)
// "moved" to the very same base folder it already lived in — which is
// exactly what happens when ResolveInstallDir applies the isolation
// subfolder to it (see MoveInstallDir, which resolves newPath this way
// before calling this function). newPath ends up inside oldPath, and the
// old unconditional os.RemoveAll(oldPath) at the end deleted the
// destination right along with the source it had just been moved out of,
// losing every file. MoveInstallFiles must survive newPath being nested
// inside oldPath.
func TestMoveInstallFilesIntoOwnSubfolder(t *testing.T) {
	oldPath := t.TempDir()
	newPath := backend.ResolveInstallDir(oldPath)

	const content = "game data that must survive the move"
	if err := os.WriteFile(filepath.Join(oldPath, "Lethalmon.exe"), []byte(content), 0o644); err != nil {
		t.Fatalf("write game file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(oldPath, "Saves"), 0o755); err != nil {
		t.Fatalf("mkdir Saves: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldPath, "Saves", "save1.dat"), []byte("progress"), 0o644); err != nil {
		t.Fatalf("write save file: %v", err)
	}

	if err := backend.MoveInstallFiles(oldPath, newPath, nil); err != nil {
		t.Fatalf("backend.MoveInstallFiles() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(newPath, "Lethalmon.exe"))
	if err != nil {
		t.Fatalf("Lethalmon.exe did not survive the move: %v", err)
	}
	if string(got) != content {
		t.Errorf("Lethalmon.exe content = %q, want %q", got, content)
	}

	if _, err := os.Stat(filepath.Join(newPath, "Saves", "save1.dat")); err != nil {
		t.Errorf("Saves/save1.dat did not survive the move: %v", err)
	}
}

// TestMoveInstallFilesToElsewhereStillCleansUpOldPath covers the common
// case (newPath genuinely elsewhere, not nested inside oldPath) still fully
// removes the old directory, exactly as before this function had to start
// handling the nested case too.
func TestMoveInstallFilesToElsewhereStillCleansUpOldPath(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old")
	newPath := filepath.Join(root, "new")
	if err := os.MkdirAll(oldPath, 0o755); err != nil {
		t.Fatalf("mkdir oldPath: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldPath, "Lethalmon.exe"), []byte("game"), 0o644); err != nil {
		t.Fatalf("write game file: %v", err)
	}

	if err := backend.MoveInstallFiles(oldPath, newPath, nil); err != nil {
		t.Fatalf("backend.MoveInstallFiles() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(newPath, "Lethalmon.exe")); err != nil {
		t.Errorf("Lethalmon.exe did not survive the move: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("oldPath still exists after the move, stat error = %v", err)
	}
}

// TestUninstallGameLeavesForeignFilesAlone is the regression test for the
// actual incident: a player's install directory wasn't isolated in its own
// GameInstallSubfolder — an install from before ResolveInstallDir existed,
// or one that's simply never been moved since — and UninstallGame used to
// delete every entry in it except a short save-data keep list, including
// whatever unrelated files the player already had stored there.
func TestUninstallGameLeavesForeignFilesAlone(t *testing.T) {
	useTempConfigDir(t)
	installDir := t.TempDir()

	if err := backend.SaveConfig(backend.Config{InstallDir: installDir}); err != nil {
		t.Fatalf("backend.SaveConfig() error: %v", err)
	}

	foreign := filepath.Join(installDir, "my_vacation_photos.zip")
	if err := os.WriteFile(foreign, []byte("precious"), 0o644); err != nil {
		t.Fatalf("write foreign file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "Lethalmon.exe"), []byte("game"), 0o644); err != nil {
		t.Fatalf("write game file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(installDir, "Data"), 0o755); err != nil {
		t.Fatalf("mkdir Data: %v", err)
	}

	a := &backend.App{}
	if err := a.UninstallGame(); err != nil {
		t.Fatalf("a.UninstallGame() error: %v", err)
	}

	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("foreign file was removed by UninstallGame: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installDir, "Lethalmon.exe")); !os.IsNotExist(err) {
		t.Errorf("game file Lethalmon.exe was not removed by UninstallGame")
	}
	if _, err := os.Stat(filepath.Join(installDir, "Data")); !os.IsNotExist(err) {
		t.Errorf("game folder Data was not removed by UninstallGame")
	}
}
