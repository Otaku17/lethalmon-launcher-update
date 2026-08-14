package tests

import (
	"path/filepath"
	"testing"

	"LethalmonLauncher/backend"
)

// clearOneDriveEnv removes the OneDrive sync-root variables so each case
// controls exactly which of them is set.
func clearOneDriveEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		t.Setenv(v, "")
	}
}

// TestIsOneDrivePathViaEnv covers the primary check: the sync roots Windows
// itself advertises. Installing there lets OneDrive lock and re-upload game
// files mid-write, which is how saves get corrupted.
func TestIsOneDrivePathViaEnv(t *testing.T) {
	root := filepath.Join(t.TempDir(), "OneDrive")

	for _, envVar := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		t.Run(envVar, func(t *testing.T) {
			clearOneDriveEnv(t)
			t.Setenv(envVar, root)

			inside := filepath.Join(root, "Documents", "Lethalmon")
			if !backend.IsOneDrivePath(inside) {
				t.Errorf("backend.IsOneDrivePath(%q) = false, want true", inside)
			}
			if !backend.IsOneDrivePath(root) {
				t.Errorf("backend.IsOneDrivePath(%q) = false for the sync root itself", root)
			}
		})
	}
}

func TestIsOneDrivePathAllowsNormalFolders(t *testing.T) {
	clearOneDriveEnv(t)
	t.Setenv("OneDrive", filepath.Join(t.TempDir(), "OneDrive"))

	allowed := []string{
		filepath.Join("C:", "Games", "Lethalmon"),
		filepath.Join("D:", "Lethalmon"),
		// Contains "onedrive" but no path component starts with it, so the
		// heuristic must not fire.
		filepath.Join("C:", "Games", "MyOneDriveBackup", "Lethalmon"),
		filepath.Join("C:", "Users", "steve", "Documents", "Games"),
	}

	for _, path := range allowed {
		if backend.IsOneDrivePath(path) {
			t.Errorf("backend.IsOneDrivePath(%q) = true, want false", path)
		}
	}
}

// TestIsOneDrivePathFallback covers the case the env vars miss: a work or
// school sync root on another drive, which Windows names "OneDrive - Company"
// and doesn't always advertise.
func TestIsOneDrivePathFallback(t *testing.T) {
	clearOneDriveEnv(t)

	flagged := []string{
		filepath.Join("D:", "OneDrive - Contoso", "Games", "Lethalmon"),
		filepath.Join("E:", "onedrive", "Lethalmon"),
		filepath.Join("C:", "Users", "steve", "OneDrive - University of Somewhere", "L"),
	}

	for _, path := range flagged {
		if !backend.IsOneDrivePath(path) {
			t.Errorf("backend.IsOneDrivePath(%q) = false, want true", path)
		}
	}
}

// TestErrOneDrivePathMessage guards a cross-language contract: the frontend
// matches this exact string to show a translated explanation instead of a raw
// Go error (see handleBrowseInstallPath in HomePage.tsx / SettingsPage.tsx).
// Renaming it here without updating the UI would degrade the message to an
// untranslated blob.
func TestErrOneDrivePathMessage(t *testing.T) {
	if got := backend.ErrOneDrivePath.Error(); got != "onedrive_path_not_allowed" {
		t.Errorf("backend.ErrOneDrivePath = %q, want %q — the frontend matches on this exact value", got, "onedrive_path_not_allowed")
	}
}
