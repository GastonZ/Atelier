//go:build !windows

package audio

// newPlatformAnalyzer returns a no-op analyzer on non-Windows platforms, where
// WASAPI loopback is unavailable. The TUI then shows the static waveform.
func newPlatformAnalyzer() Analyzer {
	return noopAnalyzer{}
}
