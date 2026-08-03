//go:build !windows

package main

// applyLauncherUpdate: self-update is only implemented for Windows (see
// app_launcher_update_other.go) — this mode is unreachable on other
// platforms since installLauncherUpdate there never spawns it.
func applyLauncherUpdate(targetExePath string) {}
