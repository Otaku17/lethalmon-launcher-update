//go:build linux

package tests

import (
	"testing"

	"LethalmonLauncher/backend"
)

// TestErrWineNotFoundMessage guards a cross-language contract: the frontend
// matches this exact string to show a translated "install wine" guide
// instead of a raw Go error (see handleLaunch in HomePage.tsx). Renaming it
// here without updating the UI would degrade the message to an untranslated
// blob.
func TestErrWineNotFoundMessage(t *testing.T) {
	if got := backend.ErrWineNotFound.Error(); got != "wine_not_found" {
		t.Errorf("backend.ErrWineNotFound = %q, want %q — the frontend matches on this exact value", got, "wine_not_found")
	}
}

// TestIsWineAvailableReflectsPATH covers that IsWineAvailable actually looks
// at PATH instead of e.g. always reporting true — that would silently defeat
// the whole point of guiding the player through installing wine instead of
// letting LaunchGame fail with an opaque exec error.
func TestIsWineAvailableReflectsPATH(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // a directory guaranteed not to contain "wine"

	a := &backend.App{}
	if available, err := a.IsWineAvailable(); err != nil || available {
		t.Errorf("a.IsWineAvailable() = %v, %v, want false, nil with wine off PATH", available, err)
	}
}
