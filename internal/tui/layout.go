package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// overlayTopLeft composites panel onto base, anchoring the panel's top-left
// corner at (top, left) in base's coordinate space.
//
// It is deliberately conservative: a panel line is only written where the base
// row's target columns are blank (spaces). If any non-space rune sits under the
// panel on a given row, that row is left untouched rather than risk splicing
// through an ANSI-styled run and corrupting colors. In practice the welcome
// canvas has a blank top-left region (the dragon art is centered/right), so the
// card lands cleanly.
func overlayTopLeft(base, panel string, top, left int) string {
	if panel == "" {
		return base
	}
	baseLines := strings.Split(base, "\n")
	panelLines := strings.Split(panel, "\n")

	for i, pl := range panelLines {
		row := top + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLines[row] = spliceLeft(baseLines[row], pl, left)
	}
	return strings.Join(baseLines, "\n")
}

// spliceLeft writes panel into base starting at visible column `left`, keeping
// the remainder of base (with its ANSI styling) intact. It only proceeds when
// the first `left + width(panel)` runes of base are spaces — guaranteeing the
// region is unstyled so rune indices equal visible columns. Otherwise it
// returns base unchanged.
func spliceLeft(base, panel string, left int) string {
	pw := lipgloss.Width(panel)
	cut := left + pw

	r := []rune(base)
	if cut >= len(r) {
		// Base is shorter than the splice point: the whole region is (logically)
		// blank padding, so left-pad the panel and drop the rest.
		return strings.Repeat(" ", left) + panel
	}

	for i := 0; i < cut; i++ {
		if r[i] != ' ' {
			return base // styled content under the panel — skip this row.
		}
	}

	return strings.Repeat(" ", left) + panel + string(r[cut:])
}
