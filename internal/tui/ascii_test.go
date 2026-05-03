package tui_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gastonz/atelier/internal/tui"
)

// TestDragonArtDimensions enforces the dimensional contract for DragonArt:
// - Line count: between 20 and 50 inclusive
// - Every line: width between 60 and 100 inclusive
// - All lines same width (uniform padding)
// - All bytes < 128 (strict 7-bit ASCII)
// - No backtick characters
func TestDragonArtDimensions(t *testing.T) {
	lines := strings.Split(tui.DragonArt, "\n")

	// Line count check
	lineCount := len(lines)
	if lineCount < 20 || lineCount > 50 {
		t.Errorf("DragonArt line count = %d, want between 20 and 50 inclusive", lineCount)
	}

	// Per-line checks
	var firstWidth int
	for i, line := range lines {
		w := utf8.RuneCountInString(line)
		if i == 0 {
			firstWidth = w
		}

		// Width bounds
		if w < 60 || w > 100 {
			t.Errorf("line %d: width = %d, want between 60 and 100 inclusive", i, w)
		}

		// Uniform width
		if w != firstWidth {
			t.Errorf("line %d: width = %d, want %d (uniform width required)", i, w, firstWidth)
		}

		// 7-bit ASCII and no backtick
		for j, r := range line {
			if r >= 128 {
				t.Errorf("line %d, byte %d: rune %d >= 128 (non-ASCII)", i, j, r)
			}
			if r == '`' {
				t.Errorf("line %d, byte %d: backtick character found (forbidden in raw string literals)", i, j)
			}
		}
	}
}

// TestDragonDimensionConstants verifies that DragonRows and DragonCols
// match the actual dimensions of the DragonArt constant.
func TestDragonDimensionConstants(t *testing.T) {
	lines := strings.Split(tui.DragonArt, "\n")
	expectedRows := len(lines)
	if tui.DragonRows != expectedRows {
		t.Errorf("DragonRows = %d, want %d (actual line count)", tui.DragonRows, expectedRows)
	}

	if len(lines) == 0 {
		t.Fatal("DragonArt has no lines")
	}
	expectedCols := utf8.RuneCountInString(lines[0])
	if tui.DragonCols != expectedCols {
		t.Errorf("DragonCols = %d, want %d (width of first line)", tui.DragonCols, expectedCols)
	}
}
