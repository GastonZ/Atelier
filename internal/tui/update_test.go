package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/tui"
)

// TestUpdate covers the core Update() branches:
// 1. tea.WindowSizeMsg populates Width and Height on the returned model.
// 2. KeyMsg "q" returns tea.Quit command.
func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		wantWidth   int
		wantHeight  int
		wantQuitCmd bool
	}{
		{
			name:        "WindowSizeMsg sets Width and Height",
			msg:         tea.WindowSizeMsg{Width: 100, Height: 40},
			wantWidth:   100,
			wantHeight:  40,
			wantQuitCmd: false,
		},
		{
			name:        "key q returns tea.Quit",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			wantWidth:   0,
			wantHeight:  0,
			wantQuitCmd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tui.New()
			result, cmd := m.Update(tt.msg)

			got, ok := result.(tui.Model)
			if !ok {
				t.Fatalf("Update() returned %T, want tui.Model", result)
			}

			if got.Width != tt.wantWidth {
				t.Errorf("Width = %d, want %d", got.Width, tt.wantWidth)
			}
			if got.Height != tt.wantHeight {
				t.Errorf("Height = %d, want %d", got.Height, tt.wantHeight)
			}

			isQuit := cmd != nil && isCmdQuit(cmd)
			if isQuit != tt.wantQuitCmd {
				t.Errorf("isQuitCmd = %v, want %v", isQuit, tt.wantQuitCmd)
			}
		})
	}
}

// isCmdQuit checks whether a tea.Cmd produces a tea.QuitMsg.
func isCmdQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	return ok
}
