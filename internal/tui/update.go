package tui

import tea "github.com/charmbracelet/bubbletea"

// Update is the Bubble Tea message handler.
// It processes all incoming messages and returns a new Model (immutable — value semantics).
// tea.WindowSizeMsg is handled unconditionally first to ensure terminal dimensions are
// always current for the size-check fork in View().
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// Run creates and starts the Bubble Tea program with alt-screen mode.
// Alt-screen is mandatory for Windows Terminal / PowerShell rendering (see design §3.2).
// Called by cmd/atelier/main.go — main does not import bubbletea directly.
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
