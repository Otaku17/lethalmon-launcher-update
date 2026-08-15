//go:build linux

package tests

import (
	"testing"

	"LethalmonLauncher/backend"
)

// TestCleanGPUName covers the exact case a player reported: lspci -nn's
// "Radeon RX 7900 XT/7900 XTX/7900 GRE/7900M" card was showing up in
// Settings as the full vendor/codename/PCI-ID/revision line instead of just
// the card's marketing name.
func TestCleanGPUName(t *testing.T) {
	cases := map[string]string{
		"Advanced Micro Devices, Inc. [AMD/ATI] Navi 31 [Radeon RX 7900 XT/7900 XTX/7900 GRE/7900M] [1002:744c] (rev c4)": "Radeon RX 7900 XT/7900 XTX/7900 GRE/7900M",
		"NVIDIA Corporation GA104 [GeForce RTX 3070] [10de:2484] (rev a1)":                                                "GeForce RTX 3070",
		"Intel Corporation TigerLake-LP GT2 [Iris Xe Graphics] [8086:9a49] (rev 01)":                                      "Iris Xe Graphics",
		// No bracketed marketing name to fall back on: the cleaned (but
		// otherwise untouched) line is returned as-is.
		"Matrox Electronics Systems Ltd. MGA G200EH": "Matrox Electronics Systems Ltd. MGA G200EH",
	}

	for raw, want := range cases {
		if got := backend.CleanGPUName(raw); got != want {
			t.Errorf("backend.CleanGPUName(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestGpuPreferenceRoundTrip covers that the "force discrete GPU" toggle
// actually persists on Linux: previously GetGpuPreference read back
// os.Getenv("DRI_PRIME") of the launcher's own process, which SetGpuPreference
// never set, so the toggle silently didn't stick across a restart.
func TestGpuPreferenceRoundTrip(t *testing.T) {
	useTempConfigDir(t)

	a := &backend.App{}

	if got, err := a.GetGpuPreference(); err != nil || got {
		t.Fatalf("a.GetGpuPreference() = %v, %v, want false, nil before it's ever been set", got, err)
	}

	if err := a.SetGpuPreference(true); err != nil {
		t.Fatalf("a.SetGpuPreference(true) error: %v", err)
	}
	if got, err := a.GetGpuPreference(); err != nil || !got {
		t.Errorf("a.GetGpuPreference() = %v, %v, want true, nil after SetGpuPreference(true)", got, err)
	}

	if err := a.SetGpuPreference(false); err != nil {
		t.Fatalf("a.SetGpuPreference(false) error: %v", err)
	}
	if got, err := a.GetGpuPreference(); err != nil || got {
		t.Errorf("a.GetGpuPreference() = %v, %v, want false, nil after SetGpuPreference(false)", got, err)
	}
}
