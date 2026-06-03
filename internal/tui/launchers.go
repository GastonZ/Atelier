package tui

// launchers.go — in-TUI manager for the configurable agent launchers
// (ScreenLaunchers list + ScreenLauncherForm add/edit). Edits mutate
// m.atelierCfg.Launchers and are persisted to config.yaml immediately, so a user
// who only ever downloaded the binary can manage launchers without ever touching
// a config file by hand.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gastonz/atelier/internal/config"
)

// --- navigation / state helpers ---

// enterLaunchers opens the launcher manager list.
func (m Model) enterLaunchers() Model {
	m.Screen = ScreenLaunchers
	if m.LauncherCursor >= len(m.atelierCfg.Launchers) {
		m.LauncherCursor = 0
	}
	m.ActionFlash = ""
	return m
}

// enterLauncherForm opens the add (editIndex < 0) or edit (editIndex >= 0) form.
func (m Model) enterLauncherForm(editIndex int) Model {
	m.launcherEditIndex = editIndex
	m.launcherErr = ""

	label, command, args := "", "", ""
	if editIndex >= 0 && editIndex < len(m.atelierCfg.Launchers) {
		l := m.atelierCfg.Launchers[editIndex]
		label, command, args = l.Label, l.Command, strings.Join(l.Args, " ")
	}
	m.launcherLabelInput.SetValue(label)
	m.launcherCmdInput.SetValue(command)
	m.launcherArgsInput.SetValue(args)

	m = m.focusLauncherField(0)
	m.Screen = ScreenLauncherForm
	return m
}

// focusLauncherField focuses form field i (0=label, 1=command, 2=args).
func (m Model) focusLauncherField(i int) Model {
	m.launcherFocus = i
	m.launcherLabelInput.Blur()
	m.launcherCmdInput.Blur()
	m.launcherArgsInput.Blur()
	switch i {
	case 0:
		m.launcherLabelInput.Focus()
	case 1:
		m.launcherCmdInput.Focus()
	default:
		m.launcherArgsInput.Focus()
	}
	return m
}

// persistLaunchers writes the current config to disk. On failure it surfaces the
// error via the flash line (callers set the success flash beforehand).
func (m Model) persistLaunchers() Model {
	if err := config.SaveAtelierConfig(m.configPath, m.atelierCfg); err != nil {
		m.ActionFlash = "No se pudo guardar la config: " + err.Error()
	}
	return m
}

// --- key handlers ---

// handleLauncherKeys handles the launcher manager list (ScreenLaunchers).
func (m Model) handleLauncherKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.atelierCfg.Launchers)
	switch {
	case msg.Type == tea.KeyEsc || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		m.Screen = ScreenWelcome
		return m, m.maybeAnimTick()

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "j":
		fallthrough
	case msg.Type == tea.KeyDown:
		if m.LauncherCursor < n-1 {
			m.LauncherCursor++
		}
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "k":
		fallthrough
	case msg.Type == tea.KeyUp:
		if m.LauncherCursor > 0 {
			m.LauncherCursor--
		}
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "a":
		return m.enterLauncherForm(-1), nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "e":
		if n == 0 {
			return m, nil
		}
		return m.enterLauncherForm(m.LauncherCursor), nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "d":
		if n == 0 {
			return m, nil
		}
		i := m.LauncherCursor
		ls := append([]config.Launcher{}, m.atelierCfg.Launchers[:i]...)
		m.atelierCfg.Launchers = append(ls, m.atelierCfg.Launchers[i+1:]...)
		if m.LauncherCursor >= len(m.atelierCfg.Launchers) && m.LauncherCursor > 0 {
			m.LauncherCursor--
		}
		m.ActionFlash = "Lanzador eliminado"
		return m.persistLaunchers(), nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "J":
		i := m.LauncherCursor
		if i < n-1 {
			ls := append([]config.Launcher{}, m.atelierCfg.Launchers...)
			ls[i], ls[i+1] = ls[i+1], ls[i]
			m.atelierCfg.Launchers = ls
			m.LauncherCursor++
			m = m.persistLaunchers()
		}
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "K":
		i := m.LauncherCursor
		if i > 0 {
			ls := append([]config.Launcher{}, m.atelierCfg.Launchers...)
			ls[i], ls[i-1] = ls[i-1], ls[i]
			m.atelierCfg.Launchers = ls
			m.LauncherCursor--
			m = m.persistLaunchers()
		}
		return m, nil
	}
	return m, nil
}

// handleLauncherFormKeys handles the add/edit launcher form (ScreenLauncherForm).
func (m Model) handleLauncherFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.launcherErr = ""
		m.Screen = ScreenLaunchers
		return m, nil

	case tea.KeyTab, tea.KeyDown:
		return m.focusLauncherField((m.launcherFocus + 1) % 3), nil

	case tea.KeyShiftTab, tea.KeyUp:
		return m.focusLauncherField((m.launcherFocus + 2) % 3), nil

	case tea.KeyEnter:
		// Enter advances through the fields, then submits on the last.
		if m.launcherFocus < 2 {
			return m.focusLauncherField(m.launcherFocus + 1), nil
		}
		return m.submitLauncherForm()
	}

	// Delegate typing to the focused input.
	var cmd tea.Cmd
	switch m.launcherFocus {
	case 0:
		m.launcherLabelInput, cmd = m.launcherLabelInput.Update(msg)
	case 1:
		m.launcherCmdInput, cmd = m.launcherCmdInput.Update(msg)
	default:
		m.launcherArgsInput, cmd = m.launcherArgsInput.Update(msg)
	}
	return m, cmd
}

// submitLauncherForm validates and upserts the launcher, persists, and returns to
// the list. Args are split on whitespace.
func (m Model) submitLauncherForm() (tea.Model, tea.Cmd) {
	label := strings.TrimSpace(m.launcherLabelInput.Value())
	command := strings.TrimSpace(m.launcherCmdInput.Value())
	args := strings.Fields(m.launcherArgsInput.Value())

	if label == "" {
		m.launcherErr = "El lanzador necesita un nombre"
		return m.focusLauncherField(0), nil
	}
	if command == "" {
		m.launcherErr = "El lanzador necesita un comando"
		return m.focusLauncherField(1), nil
	}

	l := config.Launcher{Label: label, Command: command, Args: args}
	if m.launcherEditIndex >= 0 && m.launcherEditIndex < len(m.atelierCfg.Launchers) {
		ls := append([]config.Launcher{}, m.atelierCfg.Launchers...)
		ls[m.launcherEditIndex] = l
		m.atelierCfg.Launchers = ls
		m.LauncherCursor = m.launcherEditIndex
		m.ActionFlash = "Lanzador actualizado"
	} else {
		m.atelierCfg.Launchers = append(append([]config.Launcher{}, m.atelierCfg.Launchers...), l)
		m.LauncherCursor = len(m.atelierCfg.Launchers) - 1
		m.ActionFlash = "Lanzador agregado"
	}
	m.launcherErr = ""
	m = m.persistLaunchers()
	m.Screen = ScreenLaunchers
	return m, nil
}

// --- views ---

// viewLaunchers renders the launcher manager list (ScreenLaunchers).
func (m Model) viewLaunchers() string {
	title := TitleBarStyle.Render("=== Lanzadores de agentes ===")
	sub := SubtitleStyle.Render("  Se abren en el directorio del proyecto · guardado en ~/.atelier/config.yaml")

	var rows []string
	if len(m.atelierCfg.Launchers) == 0 {
		rows = append(rows, "", "  No hay lanzadores. Presioná `a` para agregar uno.", "")
	} else {
		for i, l := range m.atelierCfg.Launchers {
			desc := l.Command
			if len(l.Args) > 0 {
				desc += " " + strings.Join(l.Args, " ")
			}
			row := l.Label + "  ·  " + desc
			if m.launcherAvailable != nil && !m.launcherAvailable(l.Command) {
				row += "  (no instalado)"
			}
			if i == m.LauncherCursor {
				rows = append(rows, SelectedRowStyle.Render("  [*] "+row))
			} else {
				rows = append(rows, "  [ ] "+row)
			}
		}
	}

	parts := []string{title, sub, ""}
	parts = append(parts, rows...)
	parts = append(parts, "")
	if m.ActionFlash != "" {
		parts = append(parts, FlashSuccessStyle.Render("  "+m.ActionFlash), "")
	}
	footer := FooterHintStyle.Render("  a: agregar  ·  e: editar  ·  d: borrar  ·  J/K: mover  ·  esc: volver")
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// viewLauncherForm renders the add/edit launcher form (ScreenLauncherForm).
func (m Model) viewLauncherForm() string {
	heading := "Nuevo lanzador"
	if m.launcherEditIndex >= 0 {
		heading = "Editar lanzador"
	}
	title := TitleBarStyle.Render("=== " + heading + " ===")

	labelLbl, cmdLbl, argsLbl := "  Nombre", "  Comando", "  Args"
	bold := func(s string) string { return lipgloss.NewStyle().Bold(true).Render(s) }
	switch m.launcherFocus {
	case 0:
		labelLbl = bold(labelLbl)
	case 1:
		cmdLbl = bold(cmdLbl)
	default:
		argsLbl = bold(argsLbl)
	}

	labelField := labelLbl + "\n  " + m.launcherLabelInput.View()
	cmdField := cmdLbl + "\n  " + m.launcherCmdInput.View()
	argsField := argsLbl + "\n  " + m.launcherArgsInput.View()

	parts := []string{title, "", labelField, "", cmdField, "", argsField}
	if m.launcherErr != "" {
		parts = append(parts, "", "  "+ErrorInlineStyle.Render(m.launcherErr))
	}
	footer := FooterHintStyle.Render("  tab: alternar  ·  enter: siguiente / confirmar  ·  esc: cancelar")
	parts = append(parts, "", footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
