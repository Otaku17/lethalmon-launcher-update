//go:build !windows

package backend

// ApplyLauncherUpdate is unreachable outside Windows: it exists only to wait
// out a locked .exe (see app_launcher_update_apply_windows.go), a problem
// Linux doesn't have (installLauncherUpdate there replaces its own running
// executable directly, see app_launcher_update_linux.go) and darwin doesn't
// support self-update at all yet (app_launcher_update_other.go).
func ApplyLauncherUpdate(targetExePath string) {}
