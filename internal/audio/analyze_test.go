package audio

import (
	"math"
	"testing"
)

func TestIsPowerOfTwo(t *testing.T) {
	cases := map[int]bool{0: false, 1: true, 2: true, 3: false, 4: true, 1024: true, 1000: false}
	for in, want := range cases {
		if got := isPowerOfTwo(in); got != want {
			t.Errorf("isPowerOfTwo(%d) = %v, want %v", in, got, want)
		}
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, 2: 2, 3: 4, 5: 8, 1024: 1024, 1025: 2048}
	for in, want := range cases {
		if got := nextPowerOfTwo(in); got != want {
			t.Errorf("nextPowerOfTwo(%d) = %d, want %d", in, got, want)
		}
	}
}

// TestFFTPeakAtExpectedBin verifies a pure cosine produces its energy peak at
// the matching frequency bin — the core correctness property of the FFT.
func TestFFTPeakAtExpectedBin(t *testing.T) {
	const n = 64
	const k = 5 // frequency bin
	samples := make([]float64, n)
	for i := range samples {
		samples[i] = math.Cos(2 * math.Pi * float64(k) * float64(i) / float64(n))
	}

	mags := magnitudes(samples)

	peak := 0
	for i := 1; i < len(mags); i++ {
		if mags[i] > mags[peak] {
			peak = i
		}
	}
	if peak != k {
		t.Errorf("FFT peak at bin %d, want %d", peak, k)
	}
}

func TestMagnitudesLengthIsHalf(t *testing.T) {
	mags := magnitudes(make([]float64, 32))
	if len(mags) != 16 {
		t.Errorf("magnitudes len = %d, want 16", len(mags))
	}
}

func TestBandsRangeAndLength(t *testing.T) {
	mags := make([]float64, 128)
	for i := range mags {
		mags[i] = float64(i) // ramp
	}
	const n = 14
	b := bands(mags, n)
	if len(b) != n {
		t.Fatalf("bands len = %d, want %d", len(b), n)
	}
	for i, v := range b {
		if v < 0 || v > 1 {
			t.Errorf("band[%d] = %f, out of 0..1", i, v)
		}
	}
}

func TestBandsEmptyInput(t *testing.T) {
	if b := bands(nil, 5); len(b) != 5 {
		t.Errorf("bands(nil,5) len = %d, want 5", len(b))
	}
}

func TestCompress(t *testing.T) {
	if compress(0) != 0 {
		t.Errorf("compress(0) = %f, want 0", compress(0))
	}
	if v := compress(1e9); v < 0.99 {
		t.Errorf("compress(huge) = %f, want ~1", v)
	}
	// Monotonic.
	if compress(10) >= compress(100) {
		t.Error("compress not monotonic increasing")
	}
}

func TestSmoothAttackAndDecay(t *testing.T) {
	prev := []float64{0.2}
	// Rising: should move up by attack fraction.
	up := smooth(prev, []float64{1.0}, 0.5, 0.1)
	if math.Abs(up[0]-0.6) > 1e-9 {
		t.Errorf("attack: got %f, want 0.6", up[0])
	}
	// Falling: should move down by decay fraction.
	down := smooth([]float64{1.0}, []float64{0.0}, 0.5, 0.1)
	if math.Abs(down[0]-0.9) > 1e-9 {
		t.Errorf("decay: got %f, want 0.9", down[0])
	}
}

func TestResample(t *testing.T) {
	tests := []struct {
		name string
		src  []float64
		n    int
		want []float64
	}{
		{"downsample", []float64{1, 2, 3, 4}, 2, []float64{1, 3}},
		{"upsample", []float64{1, 2}, 4, []float64{1, 1, 2, 2}},
		{"identity", []float64{5, 6, 7}, 3, []float64{5, 6, 7}},
		{"empty src", nil, 3, []float64{0, 0, 0}},
		{"zero n", []float64{1, 2}, 0, []float64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resample(tt.src, tt.n)
			if len(got) != len(tt.want) {
				t.Fatalf("resample len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("resample[%d] = %f, want %f", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestApplyHannZeroesEdges(t *testing.T) {
	s := []float64{1, 1, 1, 1, 1, 1, 1, 1}
	applyHann(s)
	if s[0] != 0 {
		t.Errorf("Hann first sample = %f, want 0", s[0])
	}
	if s[len(s)-1] != 0 {
		t.Errorf("Hann last sample = %f, want 0", s[len(s)-1])
	}
}
