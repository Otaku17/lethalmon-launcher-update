//go:build windows

package tests

import (
	"testing"

	"LethalmonLauncher/backend"
)

func TestGPUPriority(t *testing.T) {
	cases := map[string]int{
		"NVIDIA GeForce RTX 4070":      2,
		"nvidia geforce gtx 1650":      2,
		"Intel(R) UHD Graphics 630":    0,
		"Intel(R) Iris(R) Xe Graphics": 0,
		"AMD Radeon(TM) Graphics":      0,
		// A discrete Radeon has a model name rather than the bare "Graphics"
		// an APU reports, so it doesn't fall into the integrated bucket.
		"AMD Radeon RX 6700 XT": 1,
	}

	for name, want := range cases {
		if got := backend.GPUPriority(name); got != want {
			t.Errorf("backend.GPUPriority(%q) = %d, want %d", name, got, want)
		}
	}
}

// TestSortGPUsByLikelyDiscrete covers what callers actually depend on: gpus[0]
// is the card the game should run on, on a laptop that reports both.
func TestSortGPUsByLikelyDiscrete(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{
			name: "integrated listed first",
			in:   []string{"Intel(R) UHD Graphics 630", "NVIDIA GeForce RTX 4070"},
			want: "NVIDIA GeForce RTX 4070",
		},
		{
			name: "discrete already first",
			in:   []string{"NVIDIA GeForce RTX 4070", "Intel(R) UHD Graphics 630"},
			want: "NVIDIA GeForce RTX 4070",
		},
		{
			name: "AMD APU alongside a discrete Radeon",
			in:   []string{"AMD Radeon(TM) Graphics", "AMD Radeon RX 6700 XT"},
			want: "AMD Radeon RX 6700 XT",
		},
		{
			name: "single adapter",
			in:   []string{"Intel(R) UHD Graphics 630"},
			want: "Intel(R) UHD Graphics 630",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gpus := append([]string{}, c.in...)
			backend.SortGPUsByLikelyDiscrete(gpus)

			if gpus[0] != c.want {
				t.Errorf("first GPU = %q, want %q (got %v)", gpus[0], c.want, gpus)
			}
		})
	}
}

// TestSortGPUsIsStable pins the stable sort: two adapters the heuristic ranks
// equally must keep the order Windows reported, so the "detected GPU" shown in
// Settings doesn't change between launches on the same machine.
func TestSortGPUsIsStable(t *testing.T) {
	gpus := []string{"AMD Radeon RX 6700 XT", "AMD Radeon RX 7800 XT"}
	backend.SortGPUsByLikelyDiscrete(gpus)

	if gpus[0] != "AMD Radeon RX 6700 XT" || gpus[1] != "AMD Radeon RX 7800 XT" {
		t.Errorf("equally ranked adapters were reordered: %v", gpus)
	}
}

func TestSortGPUsEmpty(t *testing.T) {
	var gpus []string
	backend.SortGPUsByLikelyDiscrete(gpus) // must not panic
	if len(gpus) != 0 {
		t.Errorf("sorting an empty list produced %v", gpus)
	}
}
