package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/tui"
)

// TestView covers the two rendering branches of viewWelcome():
// 1. Small terminal (below threshold) → fallback branch: "Dragon Atelier" present, no dragon art.
// 2. Large terminal (above threshold) → full welcome branch: dragon art + "Dragon Atelier" present.
func TestView(t *testing.T) {
	// firstDragonLine is the first line of the art — used to detect dragon presence.
	firstDragonLine := strings.Split(tui.DragonArt, "\n")[0]

	tests := []struct {
		name            string
		width           int
		height          int
		wantTagline     bool
		wantDragon      bool
		wantResizeHint  bool
		wantMissionCtrl bool
	}{
		{
			name:           "small terminal triggers fallback branch",
			width:          50,
			height:         20,
			wantTagline:    true,
			wantDragon:     false,
			wantResizeHint: true,
		},
		{
			name:            "large terminal triggers full welcome branch",
			width:           120,
			height:          50,
			wantTagline:     true,
			wantDragon:      true,
			wantMissionCtrl: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tui.New()
			result, _ := m.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			got, ok := result.(tui.Model)
			if !ok {
				t.Fatalf("Update() returned %T, want tui.Model", result)
			}

			view := got.View()

			if tt.wantTagline && !strings.Contains(view, "Dragon Atelier") {
				t.Error("View() missing 'Dragon Atelier' tagline")
			}

			if tt.wantDragon && !strings.Contains(view, firstDragonLine) {
				t.Error("View() missing first line of dragon art in full welcome branch")
			}
			if !tt.wantDragon && strings.Contains(view, firstDragonLine) {
				t.Error("View() contains dragon art in fallback branch (should not)")
			}

			if tt.wantResizeHint && !strings.Contains(view, "Resize terminal for full art") {
				t.Error("View() missing resize hint in fallback branch")
			}

			if tt.wantMissionCtrl && !strings.Contains(view, "Mission Control for AI Workflows") {
				t.Error("View() missing 'Mission Control for AI Workflows' subtitle in full branch")
			}
		})
	}
}
