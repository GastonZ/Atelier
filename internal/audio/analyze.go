package audio

import (
	"math"
	"math/cmplx"
)

// applyHann applies a Hann window to samples in place. Windowing reduces
// spectral leakage so a steady tone shows up as a clean peak instead of smear.
func applyHann(samples []float64) {
	n := len(samples)
	if n < 2 {
		return
	}
	for i := range samples {
		w := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
		samples[i] *= w
	}
}

// magnitudes returns the magnitude spectrum (the first len/2 bins) of a real
// signal. len(samples) must be a power of two.
func magnitudes(samples []float64) []float64 {
	n := len(samples)
	if n == 0 {
		return nil
	}
	buf := make([]complex128, n)
	for i, s := range samples {
		buf[i] = complex(s, 0)
	}
	fft(buf)

	half := n / 2
	if half == 0 {
		half = 1
	}
	mags := make([]float64, half)
	for i := 0; i < half; i++ {
		mags[i] = cmplx.Abs(buf[i])
	}
	return mags
}

// bands buckets a magnitude spectrum into n log-spaced frequency bands and
// returns a normalized 0..1 level per band.
//
// Log spacing matches how we perceive pitch (each octave gets similar visual
// weight). Each band's energy is log-compressed (audio is highly dynamic) and
// scaled by a fixed reference so quiet passages stay short and loud ones fill
// the bar — without per-frame auto-gain, which would make everything pump.
func bands(mags []float64, n int) []float64 {
	out := make([]float64, n)
	if n <= 0 || len(mags) == 0 {
		return out
	}

	// Skip bin 0 (DC offset) — it carries no musical information.
	lo := 1
	hi := len(mags)
	if hi <= lo {
		hi = lo + 1
	}

	logLo := math.Log(float64(lo))
	logHi := math.Log(float64(hi))
	span := logHi - logLo

	for b := 0; b < n; b++ {
		start := lo + int(math.Exp(logLo+span*float64(b)/float64(n))) - 1
		end := lo + int(math.Exp(logLo+span*float64(b+1)/float64(n))) - 1
		if end <= start {
			end = start + 1
		}
		if start < lo {
			start = lo
		}
		if end > len(mags) {
			end = len(mags)
		}

		var sum float64
		for i := start; i < end; i++ {
			sum += mags[i]
		}
		count := end - start
		if count > 0 {
			sum /= float64(count)
		}

		out[b] = compress(sum)
	}
	return out
}

// referenceLevel maps raw averaged magnitude to the top of the bar. Tuned
// empirically against live loopback output so loud music peaks reach ~0.8-0.95
// with headroom, while quieter passages still show clear movement.
const referenceLevel = 18.0

// compress log-scales a raw magnitude into 0..1.
func compress(mag float64) float64 {
	if mag <= 0 {
		return 0
	}
	v := math.Log10(1+mag) / math.Log10(1+referenceLevel)
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}

// resample maps src to exactly n values using nearest-neighbour sampling.
// Used to fit the capture's internal band count to whatever the UI requests.
func resample(src []float64, n int) []float64 {
	out := make([]float64, max(n, 0))
	if len(src) == 0 || n <= 0 {
		return out
	}
	for i := 0; i < n; i++ {
		idx := int(float64(i) * float64(len(src)) / float64(n))
		if idx >= len(src) {
			idx = len(src) - 1
		}
		out[i] = src[idx]
	}
	return out
}

// smooth blends new levels into prev with an asymmetric attack/decay so bars
// rise quickly to a transient but fall back gently — the classic VU feel.
// attack and decay are in 0..1 (fraction of the gap closed per frame).
func smooth(prev, next []float64, attack, decay float64) []float64 {
	out := make([]float64, len(next))
	for i := range next {
		var p float64
		if i < len(prev) {
			p = prev[i]
		}
		if next[i] >= p {
			out[i] = p + (next[i]-p)*attack
		} else {
			out[i] = p + (next[i]-p)*decay
		}
	}
	return out
}
