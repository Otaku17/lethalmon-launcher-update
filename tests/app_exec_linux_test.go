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
