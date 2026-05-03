package tui

import "github.com/charmbracelet/lipgloss"

// View renders the current screen state as a string.
// Bubble Tea calls this after every model update to refresh the display.
func (m Model) View() string {
	if m.Quitting {
		return "See you later.\n"
	}
	switch m.Screen {
	case ScreenWelcome:
		return m.viewWelcome()
	}
	return ""
}

// viewWelcome renders the welcome/mission-control screen.
// Forks on terminal size: full welcome with dragon art when the terminal is large enough,
// or a minimal text-only fallback when the terminal is too small.
// Threshold: Width >= DragonCols && Height >= DragonRows + 6.
// The +6 accounts for: 1 blank line + 1 tagline + 1 subtitle + 2 blank lines + 1 hint.
func (m Model) viewWelcome() string {
	if m.Width >= DragonCols && m.Height >= DragonRows+6 {
		return m.renderFullWelcome()
	}
	return m.renderSmallFallback()
}

// renderFullWelcome composes the complete welcome screen:
// dragon art (in dragonRed) + tagline "Dragon Atelier" (Lavender, bold)
// + subtitle "Mission Control for AI Workflows" (Subtext0)
// + hint "press q to quit" (Surface2, italic).
// All content is vertically and horizontally centered via lipgloss.
func (m Model) renderFullWelcome() string {
	art := RenderDragon(DragonRedStyle)
	tagline := TaglineStyle.Render("Dragon Atelier")
	subtitle := SubtitleStyle.Render("Mission Control for AI Workflows")
	hint := HintStyle.Render("press q to quit")

	block := lipgloss.JoinVertical(
		lipgloss.Center,
		art,
		"",
		tagline,
		subtitle,
		"",
		"",
		hint,
	)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, block)
}

// renderSmallFallback renders a minimal text-only welcome for terminals too small
// to display the full dragon art without visual corruption.
// Contains "Dragon Atelier" (always present for consistent test assertions)
// and a resize hint. No dragon art is rendered.
func (m Model) renderSmallFallback() string {
	tagline := TaglineStyle.Render("Dragon Atelier")
	note := SubtitleStyle.Render("Resize terminal for full art")
	hint := HintStyle.Render("press q to quit")

	block := lipgloss.JoinVertical(
		lipgloss.Center,
		tagline,
		note,
		"",
		hint,
	)
	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, block)
	}
	return block
}
