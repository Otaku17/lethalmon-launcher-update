//go:build windows

package tests

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"LethalmonLauncher/backend"
)

// exitErrorWithCode runs a trivial command that exits with the given code, to
// get a real *exec.ExitError rather than a hand-built stub.
func exitErrorWithCode(t *testing.T, code int) error {
	t.Helper()

	err := exec.Command("cmd", "/c", "exit", strconv.Itoa(code)).Run()
	if err == nil {
		t.Fatalf("cmd /c exit %d unexpectedly succeeded", code)
	}
	return err
}

// TestIsVCRedistSuccessExit covers the installer exit codes that mean success
// despite being non-zero. Treating them as failures would surface a scary
// "install failed" to players whose runtime is, in fact, installed — 3010 in
// particular is the common case on a machine that hasn't rebooted.
func TestIsVCRedistSuccessExit(t *testing.T) {
	t.Run("3010 reboot required", func(t *testing.T) {
		if !backend.IsVCRedistSuccessExit(exitErrorWithCode(t, 3010)) {
			t.Error("exit 3010 (success, reboot pending) was treated as a failure")
		}
	})

	t.Run("1638 newer version present", func(t *testing.T) {
		if !backend.IsVCRedistSuccessExit(exitErrorWithCode(t, 1638)) {
			t.Error("exit 1638 (a newer runtime is already installed) was treated as a failure")
		}
	})

	t.Run("a real failure", func(t *testing.T) {
		if backend.IsVCRedistSuccessExit(exitErrorWithCode(t, 1)) {
			t.Error("exit 1 was treated as success")
		}
	})

	t.Run("1602 user cancelled", func(t *testing.T) {
		if backend.IsVCRedistSuccessExit(exitErrorWithCode(t, 1602)) {
			t.Error("exit 1602 (cancelled) was treated as success")
		}
	})

	t.Run("not an exit error", func(t *testing.T) {
		if backend.IsVCRedistSuccessExit(errors.New("powershell could not be started")) {
			t.Error("a non-exit error was treated as success")
		}
	})
}

// TestVerifyMicrosoftSignatureRejectsUnsignedFile is the fail-closed check: an
// installer that carries no Authenticode signature must never be run elevated,
// whatever it claims to be.
func TestVerifyMicrosoftSignatureRejectsUnsignedFile(t *testing.T) {
	unsigned := filepath.Join(t.TempDir(), "not-really-an-installer.exe")
	if err := os.WriteFile(unsigned, []byte("MZ this is not a signed binary"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := backend.VerifyMicrosoftSignature(unsigned); err == nil {
		t.Fatal("backend.VerifyMicrosoftSignature() accepted an unsigned file")
	}
}

// TestVerifyMicrosoftSignatureAcceptsWindowsBinary exercises the success path
// against a file Windows itself ships and signs. It is the only way to confirm
// the WinVerifyTrust plumbing and the certificate-organization lookup actually
// agree on a genuine Microsoft binary.
func TestVerifyMicrosoftSignatureAcceptsWindowsBinary(t *testing.T) {
	// notepad.exe is signed by Microsoft on every supported Windows version.
	const signed = `C:\Windows\System32\notepad.exe`

	if _, err := os.Stat(signed); err != nil {
		t.Skipf("%s not present: %v", signed, err)
	}

	if err := backend.VerifyMicrosoftSignature(signed); err != nil {
		t.Errorf("backend.VerifyMicrosoftSignature(%s) rejected a genuine Microsoft binary: %v", signed, err)
	}
}

// TestVerifyAuthenticodeOrgRejectsWrongPublisher confirms the check is about
// *who* signed the file, not merely that it is signed: a validly signed
// Microsoft binary must still be refused when a different organization is
// expected.
func TestVerifyAuthenticodeOrgRejectsWrongPublisher(t *testing.T) {
	const signed = `C:\Windows\System32\notepad.exe`

	if _, err := os.Stat(signed); err != nil {
		t.Skipf("%s not present: %v", signed, err)
	}

	err := backend.VerifyAuthenticodeOrg(signed, "Definitely Not Microsoft Ltd")
	if err == nil {
		t.Fatal("backend.VerifyAuthenticodeOrg() accepted a file signed by a different organization")
	}
}
