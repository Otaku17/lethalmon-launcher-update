package main

// vcRedistProgressEvent reports "downloading" (with percent tracked as
// bytes arrive) then "installing" (percent unknown — the silent installer
// gives no progress feedback) while ensureVCRedist fetches and runs the
// installer, or "failed" if it didn't succeed. Not emitted at all when the
// redistributable is already present, which is the common case on every
// launch after the first.
const vcRedistProgressEvent = "game:vcredist-progress"

type vcRedistProgress struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	// Error is only set when Stage is "failed", so the frontend can show the
	// player why the runtime install didn't happen (e.g. the UAC prompt was
	// declined) instead of leaving them to guess after the game crashes with
	// a bare "VCRUNTIME140.dll is missing" system dialog.
	Error string `json:"error,omitempty"`
}
