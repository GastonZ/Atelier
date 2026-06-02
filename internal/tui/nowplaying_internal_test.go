package tui

import (
	"strings"
	"testing"

	"github.com/gastonz/atelier/internal/nowplaying"
)

func TestRenderBars(t *testing.T) {
	tests := []struct {
		name   string
		levels []float64
		want   string
	}{
		{"empty", nil, ""},
		{"min level", []float64{0}, "▁"},
		{"max level", []float64{1}, "█"},
		{"mid level", []float64{0.5}, "▅"},
		{"clamps below zero", []float64{-1}, "▁"},
		{"clamps above one", []float64{2}, "█"},
		{"sequence length preserved", []float64{0, 0.5, 1}, "▁▅█"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderBars(tt.levels); got != tt.want {
				t.Errorf("renderBars(%v) = %q, want %q", tt.levels, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this is way too long", 10, "this is w…"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := truncate(tt.s, tt.n); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func TestRenderNowPlaying(t *testing.T) {
	t.Run("absent track renders nothing", func(t *testing.T) {
		if got := renderNowPlaying(nowplaying.Track{Present: false}, nil); got != "" {
			t.Errorf("renderNowPlaying(absent) = %q, want empty", got)
		}
	})

	t.Run("playing track shows title, source and playing status", func(t *testing.T) {
		got := renderNowPlaying(nowplaying.Track{
			Title:   "Strobe",
			Artist:  "deadmau5",
			Source:  "Spotify",
			Playing: true,
			Present: true,
		}, nil)
		for _, want := range []string{"Strobe", "deadmau5", "via Spotify", "▶ playing"} {
			if !strings.Contains(got, want) {
				t.Errorf("renderNowPlaying() missing %q in:\n%s", want, got)
			}
		}
	})

	t.Run("paused track shows paused status", func(t *testing.T) {
		got := renderNowPlaying(nowplaying.Track{
			Title:   "Some Video",
			Source:  "Chrome",
			Playing: false,
			Present: true,
		}, nil)
		if !strings.Contains(got, "⏸ paused") {
			t.Errorf("renderNowPlaying(paused) missing paused marker in:\n%s", got)
		}
	})

	t.Run("playing track with live signal uses live levels", func(t *testing.T) {
		live := []float64{0, 0, 1.0, 0, 0} // one full-scale band → full block present
		got := renderNowPlaying(nowplaying.Track{
			Title: "Strobe", Source: "Spotify", Playing: true, Present: true,
		}, live)
		if !strings.Contains(got, "█") {
			t.Errorf("renderNowPlaying() with live signal should render a full block:\n%s", got)
		}
	})
}
