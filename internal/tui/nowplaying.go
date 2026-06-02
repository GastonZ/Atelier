package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gastonz/atelier/internal/nowplaying"
)

// barRunes maps a normalized level (0..1) to a vertical block glyph.
// Index 0 is the shortest bar; the last index is a full block.
var barRunes = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// renderBars converts a slice of normalized levels (each clamped to 0..1) into
// a string of block glyphs — one glyph per level.
//
// This is the shared visualizer renderer. Batch 1 feeds it a static pattern;
// Batch 2 will feed it live per-band magnitudes from the WASAPI/FFT pipeline.
// Keeping it a pure function of its input keeps both paths testable.
func renderBars(levels []float64) string {
	if len(levels) == 0 {
		return ""
	}
	var b strings.Builder
	for _, l := range levels {
		switch {
		case l < 0:
			l = 0
		case l > 1:
			l = 1
		}
		idx := int(l*float64(len(barRunes)-1) + 0.5)
		b.WriteRune(barRunes[idx])
	}
	return b.String()
}

// staticPlayingLevels is a fixed, pleasant-looking waveform used while a track
// is playing but the live analyzer reports silence (or is unavailable).
var staticPlayingLevels = []float64{0.2, 0.5, 0.8, 1.0, 0.7, 0.4, 0.6, 0.9, 0.5, 0.3, 0.6, 0.8, 0.4, 0.2}

// hasSignal reports whether live levels carry meaningful audio (any band above a
// small noise floor). Silence → fall back to the static pattern so the card
// still looks alive between tracks.
func hasSignal(levels []float64) bool {
	for _, l := range levels {
		if l > 0.04 {
			return true
		}
	}
	return false
}

// flatLevels renders a quiet, near-silent line for paused/stopped playback.
func flatLevels(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = 0.05
	}
	return out
}

// truncate shortens s to at most n runes, appending an ellipsis when cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// numBars is the width of the waveform visualizer, in glyphs.
const numBars = 14

// renderNowPlaying builds the "now playing" card shown beside the dragon on the
// welcome screen. It returns "" when nothing is playing, so the caller can fall
// back to the dragon-only layout.
//
// live holds real-time audio levels from the loopback analyzer; when present and
// the track is playing, the bars react to the actual sound. Otherwise they fall
// back to a static pattern (playing) or a flat line (paused).
func renderNowPlaying(t nowplaying.Track, live []float64) string {
	if !t.Present {
		return ""
	}

	const maxTitle = 30

	title := NowPlayingTitleStyle.Render("♪ " + truncate(t.Title, maxTitle))

	var meta string
	switch {
	case t.Artist != "" && t.Source != "":
		meta = truncate(t.Artist, maxTitle) + "  ·  via " + t.Source
	case t.Artist != "":
		meta = truncate(t.Artist, maxTitle)
	case t.Source != "":
		meta = "via " + t.Source
	}
	metaLine := NowPlayingMetaStyle.Render(meta)

	var status string
	var levels []float64
	switch {
	case t.Playing && hasSignal(live):
		status = NowPlayingStatusStyle.Render("▶ playing")
		levels = live
	case t.Playing:
		status = NowPlayingStatusStyle.Render("▶ playing")
		levels = staticPlayingLevels
	default:
		status = NowPlayingMetaStyle.Render("⏸ paused")
		levels = flatLevels(numBars)
	}
	bars := NowPlayingBarsStyle.Render(renderBars(levels))

	card := lipgloss.JoinVertical(lipgloss.Left,
		title,
		metaLine,
		"",
		bars,
		status,
	)
	return NowPlayingCardStyle.Render(card)
}
