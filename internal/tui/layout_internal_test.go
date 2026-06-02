package tui

import (
	"strings"
	"testing"
)

func TestSpliceLeft(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		panel string
		left  int
		want  string
	}{
		{
			name:  "writes into blank region",
			base:  "          X", // 10 spaces then X at col 10
			panel: "ABC",
			left:  2,
			want:  "  ABC     X", // 2 spaces, ABC (cols 2-4), then base from col 5
		},
		{
			name:  "skips row when content sits under panel",
			base:  "X         ",
			panel: "ABC",
			left:  2,
			want:  "X         ", // X at col 0 is within cut → untouched
		},
		{
			name:  "base shorter than splice point pads and replaces",
			base:  "   ",
			panel: "ABC",
			left:  2,
			want:  "  ABC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := spliceLeft(tt.base, tt.panel, tt.left); got != tt.want {
				t.Errorf("spliceLeft() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOverlayTopLeft(t *testing.T) {
	// 4 rows of 12 blank columns.
	base := strings.Join([]string{
		strings.Repeat(" ", 12),
		strings.Repeat(" ", 12),
		strings.Repeat(" ", 12),
		strings.Repeat(" ", 12),
	}, "\n")
	panel := "AAA\nBBB"

	got := overlayTopLeft(base, panel, 1, 2)
	lines := strings.Split(got, "\n")

	if !strings.HasPrefix(lines[1], "  AAA") {
		t.Errorf("row 1 = %q, want it to start with '  AAA'", lines[1])
	}
	if !strings.HasPrefix(lines[2], "  BBB") {
		t.Errorf("row 2 = %q, want it to start with '  BBB'", lines[2])
	}
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("row 0 should remain blank, got %q", lines[0])
	}
	if strings.TrimSpace(lines[3]) != "" {
		t.Errorf("row 3 should remain blank, got %q", lines[3])
	}
}

func TestOverlayTopLeftEmptyPanelIsNoop(t *testing.T) {
	base := "hello\nworld"
	if got := overlayTopLeft(base, "", 1, 2); got != base {
		t.Errorf("overlayTopLeft with empty panel = %q, want unchanged", got)
	}
}
