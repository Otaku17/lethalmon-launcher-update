//go:build windows

package tests

import (
	"strings"
	"testing"

	"LethalmonLauncher/backend"
)

// TestGameProcessFilterScopesToInstallDir covers why the filter exists at all:
// the game runs under ruby.exe, a name shared with any other Ruby install on
// the machine. Matching on the image name alone would let an unrelated
// ruby.exe make the launcher believe the game is running — and, worse, make
// StopGame kill it.
func TestGameProcessFilterScopesToInstallDir(t *testing.T) {
	filter := backend.GameProcessFilter(backend.GameProcessNames, `C:\Games\Lethalmon`)

	for _, name := range backend.GameProcessNames {
		if !strings.Contains(filter, "'"+name+"'") {
			t.Errorf("filter does not match %q:\n%s", name, filter)
		}
	}

	if !strings.Contains(filter, "ExecutablePath") {
		t.Errorf("filter does not constrain ExecutablePath, so it would match processes outside the install:\n%s", filter)
	}
	if !strings.Contains(filter, `C:\Games\Lethalmon`) {
		t.Errorf("filter does not mention the install directory:\n%s", filter)
	}
}

// TestGameProcessFilterEscapesQuotes covers install paths containing an
// apostrophe — a Windows username like "O'Brien" is enough. PowerShell single
// quotes are escaped by doubling; getting this wrong would break the
// expression, and anyProcessRunning treats a failed command as "not running",
// so the launcher would happily overwrite a running game's files.
func TestGameProcessFilterEscapesQuotes(t *testing.T) {
	filter := backend.GameProcessFilter([]string{"ruby.exe"}, `C:\Users\O'Brien\Games`)

	if !strings.Contains(filter, `C:\Users\O''Brien\Games`) {
		t.Errorf("apostrophe in the install path was not doubled:\n%s", filter)
	}

	if strings.Contains(filter, `'C:\Users\O'Brien\Games`) {
		t.Errorf("filter contains an unescaped apostrophe:\n%s", filter)
	}
}

func TestGameProcessFilterEscapesProcessNames(t *testing.T) {
	filter := backend.GameProcessFilter([]string{"weird'name.exe"}, `C:\Games`)

	if !strings.Contains(filter, `'weird''name.exe'`) {
		t.Errorf("apostrophe in a process name was not doubled:\n%s", filter)
	}
}
