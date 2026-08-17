//go:build windows

package backend

import (
	"errors"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// vcRedistRegistryKey is where the redistributable's own installer records
// itself, so presence can be checked without running anything.
const vcRedistRegistryKey = `SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\X64`

// vcRedistRequiredDLLs are the DLLs the game's bundled Ruby runtime actually
// loads at startup. Checked in addition to the registry key: a partially
// removed or corrupted redistributable can leave the "Installed" flag set in
// the registry while these are missing, mismatched in architecture, or
// otherwise unusable — a case the registry check alone would miss and report
// as "installed" right up until the game crashes with a bare
// "VCRUNTIME140.dll is missing" system dialog.
var vcRedistRequiredDLLs = []string{
	"vcruntime140.dll",
	"vcruntime140_1.dll",
	"msvcp140.dll",
}

// ErrVCRedistNotFound is returned by ensureVCRedist when the Visual C++
// Redistributable isn't installed, or is installed but broken. The frontend
// matches on this exact message to guide the player through installing it
// themselves — mirroring how wine is handled on Linux (see ErrWineNotFound):
// a system-level component the player installs, rather than one this app
// fetches and runs an elevated installer for on their behalf.
var ErrVCRedistNotFound = errors.New("vcredist_not_found")

// VCRedistInstalled reports whether the x64 Visual C++ Redistributable is
// genuinely present and usable. Two independent checks both have to pass:
// the registry key the redistributable's own installer writes, and the
// required DLLs actually being loadable. Relying on the registry key alone
// is what let players get stuck with the key present but the DLLs gone; the
// DLL check alone would miss a version mismatch the registry key does catch.
func VCRedistInstalled() bool {
	if !vcRedistRegistryInstalled() {
		return false
	}

	for _, name := range vcRedistRequiredDLLs {
		if !vcRedistDLLLoadable(name) {
			return false
		}
	}

	return true
}

func vcRedistRegistryInstalled() bool {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, vcRedistRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	installed, _, err := key.GetIntegerValue("Installed")
	return err == nil && installed == 1
}

// vcRedistDLLLoadable reports whether name can actually be loaded through the
// normal system DLL search order, rather than merely found by name: LoadDLL
// fails the same way on a present-but-wrong-architecture or corrupted file as
// it does on one that's entirely absent, which a plain file-existence check
// would miss.
func vcRedistDLLLoadable(name string) bool {
	dll, err := windows.LoadDLL(name)
	if err != nil {
		return false
	}
	dll.Release()
	return true
}

// ensureVCRedist reports ErrVCRedistNotFound if the Visual C++
// Redistributable isn't installed. Unlike earlier versions of the launcher,
// it no longer downloads or silently runs an elevated installer — like wine
// on Linux (see newGameCommand), the runtime is a system-level component the
// player is expected to install themselves. LaunchGame surfaces the error so
// the frontend can point them at the official installer.
func (a *App) ensureVCRedist() error {
	if VCRedistInstalled() {
		return nil
	}
	return ErrVCRedistNotFound
}
