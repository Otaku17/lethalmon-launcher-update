//go:build linux

package backend

import (
	"os"
	"os/exec"
	"strings"
)

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

	SortGPUsByLikelyDiscrete(gpus)

	return gpus, nil
}

// GetGpuPreference reports whether the launcher is configured
// to use the high-performance GPU.
//
// On Linux this is represented by the DRI_PRIME setting used
// when launching the game.
func (a *App) GetGpuPreference() (bool, error) {
	return os.Getenv("DRI_PRIME") == "1", nil
}

// SetGpuPreference cannot permanently modify the environment of
// the launcher process. The setting should be applied when the
// game process is launched.
//
// This method currently only reports success. The actual DRI_PRIME
// environment variable should be applied to the game command.
func (a *App) SetGpuPreference(enabled bool) error {
	return nil
}
