package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// keyMap defines the key bindings for the application.
// Using bubbles/key provides built-in help rendering and consistent key handling.
type keyMap struct {
	Quit key.Binding
}

// keys holds the global key bindings.
var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// handleKey dispatches key messages to the appropriate per-screen handler.
// One handler function per screen — adding a screen means adding a case here.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Screen {
	case ScreenWelcome:
		return m.handleWelcomeKeys(msg)
	}
	return m, nil
}

// handleWelcomeKeys handles key events on the welcome screen.
// Bindings: q and ctrl+c quit the program.
func (m Model) handleWelcomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.Quitting = true
		return m, tea.Quit
	}
	return m, nil
}
