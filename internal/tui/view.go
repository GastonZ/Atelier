package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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
