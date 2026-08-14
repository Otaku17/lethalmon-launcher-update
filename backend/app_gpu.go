package backend

import (
	"sort"
	"strings"
)

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

	case strings.Contains(upper, "RX "),
		strings.Contains(upper, "RADEON PRO"),
		strings.Contains(upper, "ARC "):
		return 2

	case strings.Contains(upper, "UHD"),
		strings.Contains(upper, "IRIS"),
		strings.Contains(upper, "RADEON GRAPHICS"):
		return 0

	default:
		return 1
	}
}
