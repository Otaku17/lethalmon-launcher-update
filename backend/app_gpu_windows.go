//go:build windows

package backend

import (
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const gpuPreferenceRegistryKey = `Software\Microsoft\DirectX\UserGpuPreferences`

// GetGPUs returns the names of the graphics cards detected on this machine.
// Filtered to adapters actually on the PCI bus, so virtual display
// adapters — VR runtimes (Oculus/Meta Quest Link, SteamVR, Windows Mixed
// Reality), Remote Desktop, "Microsoft Basic Render Driver" — don't get
// mistaken for a real GPU (they're registered as software/ROOT devices,
// never as PCI\... ones).
func (a *App) GetGPUs() ([]string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_VideoController | Where-Object { $_.PNPDeviceID -like 'PCI\\*' } | Select-Object -ExpandProperty Name")
	hideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	gpus := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			gpus = append(gpus, name)
		}
	}

	SortGPUsByLikelyDiscrete(gpus)

	return gpus, nil
}

// gpuPreferenceExePath returns the game executable's path, which is the key
// Windows' per-app GPU preference registry entries are keyed on.
func (a *App) gpuPreferenceExePath() (string, error) {
	installDir, err := a.GetInstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, gameExeName), nil
}

// GetGpuPreference reports whether the game is currently forced to run on
// the dedicated/high-performance GPU.
func (a *App) GetGpuPreference() (bool, error) {
	exePath, err := a.gpuPreferenceExePath()
	if err != nil {
		return false, err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, gpuPreferenceRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false, nil
	}
	defer key.Close()

	value, _, err := key.GetStringValue(exePath)
	if err != nil {
		return false, nil
	}

	return strings.Contains(value, "GpuPreference=2"), nil
}

// SetGpuPreference forces (or stops forcing) the game to run on the
// dedicated/high-performance GPU via Windows' per-app graphics settings.
func (a *App) SetGpuPreference(enabled bool) error {
	exePath, err := a.gpuPreferenceExePath()
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, gpuPreferenceRegistryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(exePath); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}

	return key.SetStringValue(exePath, "GpuPreference=2;")
}
