package audio

// Analyzer exposes the live audio spectrum as a small set of normalized levels.
//
// Implementations run a background capture loop; Levels is safe to call from the
// UI goroutine at any cadence. A nil Analyzer is valid — callers guard for it.
type Analyzer interface {
	// Levels returns n band levels in 0..1, low frequencies first. When no audio
	// is being captured (silence, or capture unavailable) it returns all zeros,
	// letting the caller fall back to a static visual.
	Levels(n int) []float64
	// Close stops the capture loop and releases resources.
	Close() error
}

// internalBands is the spectral resolution the capture loop maintains; Levels
// resamples this down to whatever the UI asks for.
const internalBands = 28

// windowSize is the FFT window length (power of two). ~46ms at 44.1kHz — a good
// balance between frequency resolution and responsiveness.
const windowSize = 2048

// NewAnalyzer returns the platform audio analyzer (WASAPI loopback on Windows,
// a no-op elsewhere). It never returns nil.
func NewAnalyzer() Analyzer {
	return newPlatformAnalyzer()
}

// noopAnalyzer is used on unsupported platforms or as a safe fallback.
type noopAnalyzer struct{}

func (noopAnalyzer) Levels(n int) []float64 {
	if n < 0 {
		n = 0
	}
	return make([]float64, n)
}

func (noopAnalyzer) Close() error { return nil }
