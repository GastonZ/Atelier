// Package tui — Catppuccin Mocha palette constants and lipgloss style declarations.
//
// Catppuccin Mocha palette
// Source: https://github.com/catppuccin/catppuccin/blob/main/docs/style-guide.md
// Palette JSON: https://github.com/catppuccin/palette/blob/main/palette.json
// These constants are the canonical Catppuccin Mocha hex values.
// Prefix: c* to distinguish from brand accent constants below.
package tui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha — canonical hex palette (14 tokens).
const (
	cBase     = "#1e1e2e"
	cCrust    = "#11111b"
	cSurface0 = "#313244"
	cSurface2 = "#585b70"
	cText     = "#cdd6f4"
	cSubtext0 = "#a6adc8"
	cMauve    = "#cba6f7"
	cLavender = "#b4befe"
	cSapphire = "#74c7ec"
	cPink     = "#f5c2e7"
	cRed      = "#f38ba8"
	cGreen    = "#a6e3a1"
	cYellow   = "#f9e2af"
	cPeach    = "#fab387"
)

// --- Dragon Brand Accents ---
// NOT Catppuccin Mocha. Brand identity only — see design §4.1.
// Dragon accents are explicitly separate from canonical Catppuccin Mocha to prevent
// contamination of UI styling. The dragon is identity, not a UI component.
// Naming convention: dragon* prefix for all brand accent constants.
const (
	dragonRed   = "#8B0000"
	dragonEmber = "#a83232"
)

// Dragon brand lipgloss styles.
var (
	DragonRedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(dragonRed))
	DragonEmberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(dragonEmber))
)

// Brand text styles — use canonical Catppuccin Mocha colors.
var (
	// TaglineStyle renders the "Dragon Atelier" tagline in Catppuccin Lavender, bold.
	TaglineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cLavender)).Bold(true)
	// SubtitleStyle renders the "Mission Control for AI Workflows" line in Catppuccin Subtext0.
	SubtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cSubtext0))
	// HintStyle renders "press q to quit" and similar hints in Catppuccin Surface2, italic.
	HintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cSurface2)).Italic(true)
)

// Screen UI styles — used for list, form, and action views.
var (
	// TitleBarStyle renders screen titles: bold, Dragon Red, padded.
	TitleBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(dragonRed)).
			Bold(true).
			PaddingBottom(1)

	// FooterHintStyle renders the keyboard hint line at the bottom of each screen.
	FooterHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cSurface2)).
			Italic(true)

	// ErrorInlineStyle renders inline form validation errors in Catppuccin Red.
	ErrorInlineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cRed))

	// FlashSuccessStyle renders success flash messages in Catppuccin Green, italic.
	FlashSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cGreen)).
				Italic(true)

	// SelectedRowStyle renders the selected action row in ScreenProjectActions.
	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(cSurface0)).
				Foreground(lipgloss.Color(cLavender))
)

// --- Agent monitor tile styles -----------------------------------------------

var (
	// TileActiveStyle renders the border of the currently-selected tile.
	TileActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(cLavender)).
			Padding(0, 1)

	// TileInactiveStyle renders the border of unselected tiles.
	TileInactiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(cSurface2)).
				Padding(0, 1)

	// TileCostStyle renders the cost line inside a tile.
	TileCostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cGreen))

	// TileLastEventStyle renders the last-event preview header.
	TileLastEventHeaderStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color(cSubtext0)).
					Italic(true)

	// SubAgentIndentStyle renders nested sub-agent tiles with a dimmed indent.
	SubAgentIndentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cSurface2)).
				PaddingLeft(2)

	// ReplayHeaderStyle renders the "Crónica del taller" header.
	ReplayHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(cMauve)).
				Bold(true)

	// AgentFlashStyle renders transient error/warning messages.
	AgentFlashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(cYellow)).
			Italic(true)
)

// The following blank identifiers prevent "declared and not used" errors for
// Catppuccin Mocha palette tokens not yet referenced by active style declarations.
// They are retained for completeness of the canonical palette.
var (
	_ = lipgloss.Color(cBase)
	_ = lipgloss.Color(cCrust)
	_ = lipgloss.Color(cText)
	_ = lipgloss.Color(cSapphire)
	_ = lipgloss.Color(cPink)
)
