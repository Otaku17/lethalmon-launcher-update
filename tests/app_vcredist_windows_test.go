//go:build windows

package tests

import (
	"testing"

	"LethalmonLauncher/backend"
)

// TestVCRedistInstalledDoesNotPanic is a smoke test for the detection logic
// itself: whatever the actual state of the machine running the suite,
// VCRedistInstalled must return a plain bool rather than panicking, since
// it's called on every LaunchGame regardless of environment.
func TestVCRedistInstalledDoesNotPanic(t *testing.T) {
	_ = backend.VCRedistInstalled()
}

// TestErrVCRedistNotFoundMessage locks down the exact error string the
// frontend matches on (see HomePage.tsx's handleLaunch) to show the "install
// the Visual C++ Redistributable yourself" modal, the same way wine_not_found
// is matched for Wine on Linux.
func TestErrVCRedistNotFoundMessage(t *testing.T) {
	const want = "vcredist_not_found"
	if got := backend.ErrVCRedistNotFound.Error(); got != want {
		t.Errorf("backend.ErrVCRedistNotFound.Error() = %q, want %q", got, want)
	}
}
