//go:build windows

package backend

import (
	"os/exec"
	"path/filepath"
	"sort"
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

// SortGPUsByLikelyDiscrete puts the most likely discrete card first. Both a
// discrete GPU and its motherboard/CPU's integrated one show up in the list
// (the PCI-bus filter in GetGPUs only excludes virtual adapters, not
// integrated ones), and callers use gpus[0] as "the" detected GPU.
//
// The sort is stable so adapters the heuristic can't tell apart keep the order
// Windows reported them in, instead of shuffling between calls.
func SortGPUsByLikelyDiscrete(gpus []string) {
	sort.SliceStable(gpus, func(i, j int) bool {
		return GPUPriority(gpus[i]) > GPUPriority(gpus[j])
	})
}

// GPUPriority ranks a GPU name by how likely it is to be a discrete card
// rather than an integrated one, since Win32_VideoController doesn't
// distinguish the two. NVIDIA doesn't make integrated GPUs for consumer
// AMD/Intel platforms, so any "NVIDIA" name ranks above generic
// "AMD Radeon(TM) Graphics" (APU) / "Intel(R) ... Graphics" (iGPU) names.
func GPUPriority(name string) int {
	upper := strings.ToUpper(name)

	switch {
	case strings.Contains(upper, "NVIDIA"):
		return 2
	case strings.Contains(upper, "GRAPHICS") || strings.Contains(upper, "INTEL") ||
		strings.Contains(upper, "UHD") || strings.Contains(upper, "IRIS"):
		return 0
	default:
		return 1
	}
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
