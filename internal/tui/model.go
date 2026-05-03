package tui

import tea "github.com/charmbracelet/bubbletea"

// Screen identifies the active TUI screen.
// New screens are added as additional iota constants — never remove or reorder existing ones.
type Screen int

const (
	// ScreenWelcome is the initial welcome / mission-control screen.
	ScreenWelcome Screen = iota
)

// Model holds ALL application state. No sub-models.
// Every Update branch returns a NEW Model (value semantics — immutability).
type Model struct {
	// Screen is the currently active screen.
	Screen Screen
	// PrevScreen holds the previous screen for back-navigation (seeded for future use).
	PrevScreen Screen
	// Cursor tracks the focused item index on list-like screens (seeded for future use).
	Cursor int
	// Width is the terminal width in columns, populated by tea.WindowSizeMsg.
	// Zero until the first WindowSizeMsg arrives.
	Width int
	// Height is the terminal height in rows, populated by tea.WindowSizeMsg.
	// Zero until the first WindowSizeMsg arrives.
	Height int
	// Quitting is true after the user presses q or ctrl+c.
	Quitting bool
}

// New returns a zero-value Model initialized to the welcome screen.
func New() Model {
	return Model{
		Screen: ScreenWelcome,
	}
}

// Init is called by Bubble Tea on program start.
// Returns nil — no initial commands needed.
func (m Model) Init() tea.Cmd {
	return nil
}
