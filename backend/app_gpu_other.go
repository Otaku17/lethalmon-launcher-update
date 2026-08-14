//go:build !windows

package backend

// GetGpuPreference is only supported on Windows.
func (a *App) GetGpuPreference() (bool, error) {
	return false, nil
}

// SetGpuPreference is only supported on Windows.
func (a *App) SetGpuPreference(enabled bool) error {
	return nil
}

// GetGPUs is only supported on Windows.
func (a *App) GetGPUs() ([]string, error) {
	return []string{}, nil
}
