// Package audio captures the system's output audio (WASAPI loopback on Windows)
// and turns it into a small set of normalized frequency-band levels suitable for
// driving a terminal waveform visualizer.
//
// The DSP core (FFT, windowing, band bucketing) is pure and unit-tested. The
// capture backend is platform-specific and injected behind the Analyzer
// interface, so the TUI and tests never touch real audio hardware.
package audio

import "math"

// fft computes the in-place radix-2 decimation-in-time FFT of x.
// len(x) MUST be a power of two; callers guarantee this.
func fft(x []complex128) {
	n := len(x)
	if n <= 1 {
		return
	}

	// Bit-reversal permutation.
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}

	// Butterfly stages.
	for length := 2; length <= n; length <<= 1 {
		ang := -2 * math.Pi / float64(length)
		wlen := complex(math.Cos(ang), math.Sin(ang))
		for i := 0; i < n; i += length {
			w := complex(1, 0)
			half := length >> 1
			for k := 0; k < half; k++ {
				u := x[i+k]
				v := x[i+k+half] * w
				x[i+k] = u + v
				x[i+k+half] = u - v
				w *= wlen
			}
		}
	}
}

// isPowerOfTwo reports whether n is a positive power of two.
func isPowerOfTwo(n int) bool {
	return n > 0 && n&(n-1) == 0
}

// nextPowerOfTwo returns the smallest power of two >= n (minimum 1).
func nextPowerOfTwo(n int) int {
	if n < 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}
