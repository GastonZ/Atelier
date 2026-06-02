package disk

import (
	"math"
	"testing"
)

// TestHumanReadable_AllCases covers the spec requirement R5.5: B / KB / MB / GB formatting.
func TestHumanReadable_AllCases(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"1023 bytes (max B)", 1023, "1023 B"},
		{"exactly 1 KB", 1024, "1.00 KB"},
		{"1.5 KB", 1536, "1.50 KB"},
		{"exactly 1 MB", 1024 * 1024, "1.00 MB"},
		{"1.5 MB", int64(1.5 * 1024 * 1024), "1.50 MB"},
		{"2.3 GB", 2469606195, "2.30 GB"}, // 2.3 * 1024^3 = 2469606195.2
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.00 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HumanReadable(tt.bytes)
			if got != tt.want {
				t.Errorf("HumanReadable(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// TestHumanReadable_LargeValue verifies max int64-ish value doesn't panic.
func TestHumanReadable_LargeValue(t *testing.T) {
	// Should return a GB value without panicking.
	got := HumanReadable(math.MaxInt32)
	if got == "" {
		t.Error("HumanReadable(MaxInt32) returned empty string")
	}
	// Should be in GB range.
	if len(got) < 3 {
		t.Errorf("HumanReadable(large) = %q, expected GB range string", got)
	}
}

// TestHumanReadable_DecimalsOnlyAbove1KB verifies B range has no decimals.
func TestHumanReadable_DecimalsOnlyAbove1KB(t *testing.T) {
	got := HumanReadable(500)
	if got != "500 B" {
		t.Errorf("HumanReadable(500) = %q, want '500 B' (no decimals below 1KB)", got)
	}
}
