package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gastonz/atelier/internal/transcripts"
)

// View renders the current screen state as a string.
// Bubble Tea calls this after every model update to refresh the display.
func (m Model) View() string {
	if m.Quitting {
		return "See you later.\n"
	}
	switch m.Screen {
	case ScreenWelcome:
		return m.viewWelcome()
	case ScreenProjects:
		return m.viewProjects()
	case ScreenAddProject:
		return m.viewAddProject()
	case ScreenProjectActions:
		return m.viewProjectActions()
	case ScreenConfirmDelete:
		return m.viewConfirmDelete()
	case ScreenAgentMonitor:
		return m.viewAgentMonitor()
	case ScreenAgentZoom:
		return m.viewAgentZoom()
	case ScreenAgentReplay:
		return m.viewAgentReplay()
	}
	return ""
}

// viewWelcome renders the welcome/mission-control screen.
// Forks on terminal size: full welcome with dragon art when the terminal is large enough,
// or a minimal text-only fallback when the terminal is too small.
func (m Model) viewWelcome() string {
	if m.Width >= DragonCols && m.Height >= DragonRows+6 {
		return m.renderFullWelcome()
	}
	return m.renderSmallFallback()
}

// renderFullWelcome composes the complete welcome screen:
// dragon art (in dragonRed) + tagline + subtitle + hints.
func (m Model) renderFullWelcome() string {
	art := RenderDragon(DragonRedStyle)
	tagline := TaglineStyle.Render("Dragon Atelier")
	subtitle := SubtitleStyle.Render("Mission Control for AI Workflows")
	hint := HintStyle.Render("press q to quit")
	enterHint := HintStyle.Render("presioná Enter para abrir tus tomos")

	block := lipgloss.JoinVertical(
		lipgloss.Center,
		art,
		"",
		tagline,
		subtitle,
		"",
		enterHint,
		hint,
	)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, block)
}

// renderSmallFallback renders a minimal text-only welcome for small terminals.
func (m Model) renderSmallFallback() string {
	tagline := TaglineStyle.Render("Dragon Atelier")
	note := SubtitleStyle.Render("Resize terminal for full art")
	hint := HintStyle.Render("press q to quit")
	enterHint := HintStyle.Render("presioná Enter para abrir tus tomos")

	block := lipgloss.JoinVertical(
		lipgloss.Center,
		tagline,
		note,
		"",
		enterHint,
		hint,
	)
	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, block)
	}
	return block
}

// viewProjects renders the project list screen (ScreenProjects).
func (m Model) viewProjects() string {
	title := TitleBarStyle.Render("=== Tus Tomos ===")

	var body string
	if len(m.projects) == 0 {
		// Empty state — S2.7
		empty := lipgloss.JoinVertical(lipgloss.Left,
			"",
			"  Las páginas están vacías.",
			"  Presioná `n` para inscribir tu primer tomo.",
			"",
		)
		body = empty
	} else if m.ListInited {
		body = m.list.View()
	} else {
		body = HintStyle.Render("  Invocando los tomos...")
	}

	var flash string
	if m.ActionFlash != "" {
		flash = FlashSuccessStyle.Render("  " + m.ActionFlash)
	}

	var footer string
	if len(m.projects) == 0 {
		footer = FooterHintStyle.Render("  n: inscribir nuevo  ·  esc: volver")
	} else {
		footer = FooterHintStyle.Render("  n: inscribir nuevo  ·  enter: abrir  ·  d: borrar  ·  esc: volver")
	}

	parts := []string{title, body}
	if flash != "" {
		parts = append(parts, flash)
	}
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// viewAddProject renders the add project form (ScreenAddProject).
func (m Model) viewAddProject() string {
	title := TitleBarStyle.Render("=== Inscribir nuevo tomo ===")

	nameLbl := "  Nombre"
	pathLbl := "  Sendero"
	if m.AddFocus == 0 {
		nameLbl = lipgloss.NewStyle().Bold(true).Render(nameLbl)
	} else {
		pathLbl = lipgloss.NewStyle().Bold(true).Render(pathLbl)
	}

	nameField := nameLbl + "\n  " + m.nameInput.View()
	pathField := pathLbl + "\n  " + m.pathInput.View()

	parts := []string{title, "", nameField, "", pathField}

	if m.AddError != "" {
		errLine := "  " + ErrorInlineStyle.Render(m.AddError)
		parts = append(parts, "", errLine)
	}

	footer := FooterHintStyle.Render("  tab: alternar  ·  enter: confirmar  ·  esc: cancelar")
	parts = append(parts, "", footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// viewProjectActions renders the project action menu (ScreenProjectActions).
func (m Model) viewProjectActions() string {
	// Find the selected project name/path for the title
	proj := m.findProject(m.SelectedID)
	projectName := m.SelectedID
	projectPath := ""
	if proj != nil {
		projectName = proj.Name
		projectPath = proj.Path
	}

	title := TitleBarStyle.Render("=== " + projectName + " ===")
	pathLine := SubtitleStyle.Render("  " + projectPath)

	actions := []string{
		"Abrir en Claude Code",
		"Invocar PowerShell",
		"Copiar el sendero",
	}

	var actionLines []string
	for i, a := range actions {
		if i == m.ActionCursor {
			actionLines = append(actionLines, SelectedRowStyle.Render("  [*] "+a))
		} else {
			actionLines = append(actionLines, "  [ ] "+a)
		}
	}

	parts := []string{title, pathLine, ""}
	parts = append(parts, actionLines...)
	parts = append(parts, "")

	if m.ActionFlash != "" {
		parts = append(parts, FlashSuccessStyle.Render("  "+m.ActionFlash), "")
	}

	footer := FooterHintStyle.Render("  j/k: navegar  ·  enter: ejecutar  ·  esc: volver")
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// viewConfirmDelete renders the deletion confirmation screen (ScreenConfirmDelete).
func (m Model) viewConfirmDelete() string {
	proj := m.findProject(m.SelectedID)
	projectName := m.SelectedID
	projectPath := ""
	if proj != nil {
		projectName = proj.Name
		projectPath = proj.Path
	}

	prompt := fmt.Sprintf("¿Borrar el tomo %q?", projectName)
	pathLine := SubtitleStyle.Render(projectPath)
	instruction := "(y / n)"
	footer := FooterHintStyle.Render("y: confirmar  ·  n / esc: cancelar")

	inner := lipgloss.JoinVertical(lipgloss.Center,
		TaglineStyle.Render(prompt),
		"",
		pathLine,
		"",
		HintStyle.Render(instruction),
		"",
		footer,
	)

	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, inner)
	}
	return inner
}

// ============================================================================
// Agent monitor views
// ============================================================================

// viewAgentMonitor renders ScreenAgentMonitor — live tile grid.
func (m Model) viewAgentMonitor() string {
	title := TitleBarStyle.Render("=== El Atelier ===")

	// Flash line (watcher errors, price warnings)
	var flash string
	if m.AgentFlash != "" {
		flash = AgentFlashStyle.Render("  " + m.AgentFlash)
	}

	// Body: empty state or tile list
	var body string
	if len(m.agentSessions) == 0 {
		emptyText := CopyMonitorEmpty
		if m.Width > 0 && m.Height > 0 {
			body = lipgloss.Place(m.Width, m.Height-4, lipgloss.Center, lipgloss.Center, emptyText)
		} else {
			body = emptyText
		}
	} else {
		body = m.renderTileList()
	}

	footer := FooterHintStyle.Render(CopyFooterMonitor)

	parts := []string{title}
	if flash != "" {
		parts = append(parts, flash)
	}
	parts = append(parts, body, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderTileList renders all session tiles stacked vertically.
func (m Model) renderTileList() string {
	var rows []string
	for i, s := range m.agentSessions {
		rows = append(rows, m.renderTile(i, s))
		// Expanded sub-agents
		if m.agentExpanded[s.ID] {
			for _, sub := range s.SubSessions {
				rows = append(rows, SubAgentIndentStyle.Render(m.renderSubTile(sub)))
			}
		}
	}
	return strings.Join(rows, "\n")
}

// renderTile renders a single root session tile.
func (m Model) renderTile(idx int, s transcripts.Session) string {
	selected := idx == m.AgentTileCursor

	name := s.ProjectName
	if name == "" {
		name = CopyMonitorUnmatched
	}

	// Last activity line
	since := relativeTime(s.LastEventTime)
	header := fmt.Sprintf("%s  ·  %s", name, since)

	// Cost line
	costLine := TileCostStyle.Render(fmt.Sprintf(CopyCostLine, s.AccumulatedUSD))

	// Sub-agent indicator
	var subLine string
	if len(s.SubSessions) > 0 {
		if m.agentExpanded[s.ID] {
			subLine = fmt.Sprintf("  %d sub-agentes (abiertos)", len(s.SubSessions))
		} else {
			subLine = fmt.Sprintf("  %d sub-agentes", len(s.SubSessions))
		}
	}

	// Last event preview
	var lastEvt string
	if len(s.Events) > 0 {
		lastEvt = TileLastEventHeaderStyle.Render(CopyLastEventHeader) + " " + eventPreview(s.Events[len(s.Events)-1])
	}

	inner := header + "\n" + costLine
	if subLine != "" {
		inner += "\n" + subLine
	}
	if lastEvt != "" {
		inner += "\n" + lastEvt
	}

	if selected {
		return TileActiveStyle.Render(inner)
	}
	return TileInactiveStyle.Render(inner)
}

// renderSubTile renders a sub-agent session as a compact nested tile.
func (m Model) renderSubTile(s transcripts.Session) string {
	name := s.ProjectName
	if name == "" {
		name = "sub-agente"
	}
	since := relativeTime(s.LastEventTime)
	return fmt.Sprintf("  · %s  %s  %s", name, since, fmt.Sprintf(CopyCostLine, s.AccumulatedUSD))
}

// viewAgentZoom renders ScreenAgentZoom — detail view for a single session.
func (m Model) viewAgentZoom() string {
	title := TitleBarStyle.Render("=== El Atelier — Detalle ===")

	// Find the zoomed session
	var s *transcripts.Session
	for i := range m.agentSessions {
		if m.agentSessions[i].ID == m.AgentZoomedID {
			s = &m.agentSessions[i]
			break
		}
	}

	var body string
	if s == nil {
		body = SubtitleStyle.Render("  Sesión no encontrada.")
	} else {
		name := s.ProjectName
		if name == "" {
			name = CopyMonitorUnmatched
		}
		since := relativeTime(s.LastEventTime)
		costLine := TileCostStyle.Render(fmt.Sprintf(CopyCostLine, s.AccumulatedUSD))
		lastEvt := ""
		if len(s.Events) > 0 {
			lastEvt = TileLastEventHeaderStyle.Render(CopyLastEventHeader) + "\n  " + eventPreview(s.Events[len(s.Events)-1])
		}

		lines := []string{
			SubtitleStyle.Render("  " + name + "  ·  " + since),
			"  " + costLine,
		}
		if lastEvt != "" {
			lines = append(lines, "  "+lastEvt)
		}
		body = strings.Join(lines, "\n")
	}

	// Flash
	var flash string
	if m.AgentFlash != "" {
		flash = AgentFlashStyle.Render("  " + m.AgentFlash)
	}

	footer := FooterHintStyle.Render(CopyFooterZoom)

	parts := []string{title, body}
	if flash != "" {
		parts = append(parts, flash)
	}
	parts = append(parts, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// viewAgentReplay renders ScreenAgentReplay — step-by-step event replay.
func (m Model) viewAgentReplay() string {
	title := ReplayHeaderStyle.Render(CopyReplayHeader)

	// Status line: cursor/total, speed, paused flag
	totalEvents := len(m.replayEvents)
	paused := ""
	if m.ReplayPaused {
		paused = "  [pausado]"
	}
	status := fmt.Sprintf("  evento %d / %d  ·  velocidad %.1fx%s",
		m.ReplayCursor+1, totalEvents, m.ReplaySpeed, paused)

	// Current event body
	var eventBody string
	if totalEvents > 0 && m.ReplayCursor < totalEvents {
		eventBody = "  " + eventPreview(m.replayEvents[m.ReplayCursor])
	}

	footer := FooterHintStyle.Render(CopyFooterReplay)

	parts := []string{title, status}
	if eventBody != "" {
		parts = append(parts, eventBody)
	}
	parts = append(parts, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// ============================================================================
// Shared view helpers
// ============================================================================

// relativeTime returns a human-readable relative time string ("2m ago", "just now").
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "ahora mismo"
	case diff < time.Hour:
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh", int(diff.Hours()))
	default:
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	}
}

// eventPreview returns a one-line summary of an event for display.
func eventPreview(evt transcripts.Event) string {
	if evt == nil {
		return "—"
	}
	switch e := evt.(type) {
	case *transcripts.AssistantEvent:
		preview := e.Text
		if len(preview) > 80 {
			preview = preview[:80] + "…"
		}
		return preview
	case *transcripts.ToolUseEvent:
		return "tool: " + e.ToolName
	case *transcripts.UserEvent:
		preview := e.Text
		if len(preview) > 60 {
			preview = preview[:60] + "…"
		}
		return "user: " + preview
	case *transcripts.ToolResultEvent:
		if e.IsError {
			return "tool_result: [error] " + e.OutputSummary
		}
		return "tool_result: " + e.OutputSummary
	default:
		return fmt.Sprintf("[%T]", evt)
	}
}
