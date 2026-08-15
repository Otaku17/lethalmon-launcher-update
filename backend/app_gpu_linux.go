//go:build linux

package backend

import (
	"os/exec"
	"regexp"
	"strings"
)

// pciIDPattern matches the PCI vendor:device ID lspci -nn appends in
// brackets, e.g. "[1002:744c]" — an identifier, not part of the card's name.
var pciIDPattern = regexp.MustCompile(`\s*\[[0-9a-fA-F]{4}:[0-9a-fA-F]{4}\]`)

// revisionPattern matches the silicon revision lspci appends, e.g. "(rev c4)".
var revisionPattern = regexp.MustCompile(`\s*\(rev [0-9a-fA-F]+\)`)

// lastBracketPattern captures the last "[...]" group on an lspci line, which
// is the card's marketing name (e.g. "Radeon RX 7900 XT/7900 XTX/7900
// GRE/7900M") as opposed to the vendor name/chip codename lspci reports
// alongside it (e.g. "Advanced Micro Devices, Inc. [AMD/ATI] Navi 31").
var lastBracketPattern = regexp.MustCompile(`\[([^\[\]]+)\]\s*$`)

// CleanGPUName reduces a raw lspci line down to just the card's marketing
// name, so Settings shows "Radeon RX 7900 XT/7900 XTX/7900 GRE/7900M"
// instead of "Advanced Micro Devices, Inc. [AMD/ATI] Navi 31 [Radeon RX
// 7900 XT/7900 XTX/7900 GRE/7900M] [1002:744c] (rev c4)".
func CleanGPUName(name string) string {
	cleaned := revisionPattern.ReplaceAllString(name, "")
	cleaned = pciIDPattern.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)

	if match := lastBracketPattern.FindStringSubmatch(cleaned); match != nil {
		return strings.TrimSpace(match[1])
	}

	return cleaned
}

// GetGPUs returns the graphics cards detected on the PCI bus.
func (a *App) GetGPUs() ([]string, error) {
	cmd := exec.Command(
		"lspci",
		"-nn",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus []string

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		upper := strings.ToUpper(line)

		// Only keep actual graphics adapters.
		if !strings.Contains(upper, "VGA COMPATIBLE CONTROLLER") &&
			!strings.Contains(upper, "3D CONTROLLER") &&
			!strings.Contains(upper, "DISPLAY CONTROLLER") {
			continue
		}

		// lspci example:
		//
		// 03:00.0 VGA compatible controller [0300]:
		// Advanced Micro Devices, Inc. [AMD/ATI] Navi 31 [Radeon RX 7900 GRE] [1002:744c]
		//
		// We want everything after the ": ".
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[1])

		if name != "" {
			gpus = append(gpus, name)
		}
	}

	// Sorted on the raw lspci names first: GPUPriority looks for vendor
	// markers like "NVIDIA" that CleanGPUName strips out to produce the
	// display name, so cleaning before sorting would break the heuristic.
	SortGPUsByLikelyDiscrete(gpus)

	cleaned := make([]string, len(gpus))
	for i, name := range gpus {
		cleaned[i] = CleanGPUName(name)
	}

	return cleaned, nil
}

// GetGpuPreference reports whether the launcher is configured to force the
// game onto the discrete GPU. Persisted in the launcher's own config (see
// Config.ForceDiscreteGpu) since, unlike Windows, there's no OS-level
// per-app setting to read it back from.
func (a *App) GetGpuPreference() (bool, error) {
	return LoadConfig().ForceDiscreteGpu, nil
}

// SetGpuPreference persists whether the game should be forced onto the
// discrete GPU. It can't be applied to the launcher's own environment right
// now — the game runs as a separate wine process, so it's actually applied
// by newGameCommand (app_exec_linux.go) setting DRI_PRIME on that process's
// environment the next time the game is launched.
func (a *App) SetGpuPreference(enabled bool) error {
	cfg := LoadConfig()
	cfg.ForceDiscreteGpu = enabled
	return SaveConfig(cfg)
}
