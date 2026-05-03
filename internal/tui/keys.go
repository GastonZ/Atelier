package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// keyMap defines the global key bindings for the application.
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
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ctrl+c always quits from any screen (S6.7)
	if msg.Type == tea.KeyCtrlC {
		m.Quitting = true
		return m, tea.Quit
	}

	switch m.Screen {
	case ScreenWelcome:
		return m.handleWelcomeKeys(msg)
	case ScreenProjects:
		return m.handleProjectsKeys(msg)
	case ScreenAddProject:
		return m.handleAddProjectKeys(msg)
	case ScreenProjectActions:
		return m.handleProjectActionsKeys(msg)
	case ScreenConfirmDelete:
		return m.handleConfirmDeleteKeys(msg)
	}
	return m, nil
}

// handleWelcomeKeys handles key events on the welcome screen.
// q quits; Enter navigates to ScreenProjects; esc is a no-op.
func (m Model) handleWelcomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.Quitting = true
		return m, tea.Quit
	case msg.Type == tea.KeyEnter:
		m.PrevScreen = ScreenWelcome
		m.Screen = ScreenProjects
		return m, loadProjectsCmd(m.registry)
	case msg.Type == tea.KeyEsc:
		// no-op on Welcome — S6.8
		return m, nil
	}
	return m, nil
}

// handleProjectsKeys handles key events on the project list screen.
// IMPORTANT: our q/esc handler runs BEFORE delegating to m.list.Update to ensure
// the (already-neutered) bubbles/list Quit binding never fires.
func (m Model) handleProjectsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		// q and esc both go to Welcome — NOT a quit (S2.4, S2.5, S6.2, S6.3)
		m.Screen = ScreenWelcome
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "n":
		// n opens the add project form (S2.2)
		m.PrevScreen = ScreenProjects
		m = m.resetAddForm()
		m.Screen = ScreenAddProject
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "d":
		// d opens confirm delete for selected project (S2.3)
		if len(m.projects) == 0 {
			return m, nil
		}
		if m.ListInited {
			item := m.list.SelectedItem()
			if item != nil {
				pi := item.(projectItem)
				m.SelectedID = pi.project.ID
			} else if len(m.projects) > 0 {
				m.SelectedID = m.projects[0].ID
			}
		} else if len(m.projects) > 0 {
			m.SelectedID = m.projects[0].ID
		}
		if m.SelectedID == "" {
			return m, nil
		}
		m.PrevScreen = ScreenProjects
		m.Screen = ScreenConfirmDelete
		return m, nil

	case msg.Type == tea.KeyEnter:
		// Enter opens actions for the selected project (S2.1)
		if len(m.projects) == 0 {
			return m, nil
		}
		if m.ListInited {
			item := m.list.SelectedItem()
			if item != nil {
				pi := item.(projectItem)
				m.SelectedID = pi.project.ID
			} else if len(m.projects) > 0 {
				m.SelectedID = m.projects[0].ID
			}
		} else if len(m.projects) > 0 {
			m.SelectedID = m.projects[0].ID
		}
		if m.SelectedID == "" {
			return m, nil
		}
		m.ActionCursor = 0
		m.PrevScreen = ScreenProjects
		m.Screen = ScreenProjectActions
		return m, nil
	}

	// Delegate navigation keys to the embedded list (j/k/arrows/pgdn/pgup)
	if m.ListInited {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleAddProjectKeys handles key events on the add project form.
func (m Model) handleAddProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel without saving (S3.5, S6.4)
		m.AddError = ""
		m.Screen = ScreenProjects
		return m, nil

	case tea.KeyTab:
		// Cycle focus forward (S3.1, S3.2)
		m.AddFocus = 1 - m.AddFocus
		if m.AddFocus == 0 {
			m.nameInput.Focus()
			m.pathInput.Blur()
		} else {
			m.nameInput.Blur()
			m.pathInput.Focus()
		}
		return m, nil

	case tea.KeyShiftTab:
		// Cycle focus backward (S3.2)
		m.AddFocus = 1 - m.AddFocus
		if m.AddFocus == 0 {
			m.nameInput.Focus()
			m.pathInput.Blur()
		} else {
			m.nameInput.Blur()
			m.pathInput.Focus()
		}
		return m, nil

	case tea.KeyEnter:
		if m.AddFocus == 0 {
			// Enter on name field: move focus to path (S3.6)
			m.AddFocus = 1
			m.nameInput.Blur()
			m.pathInput.Focus()
			return m, nil
		}
		// Enter on path field: validate and save (S3.3, S3.4)
		name := strings.TrimSpace(m.nameInput.Value())
		path := m.pathInput.Value()

		if name == "" {
			m.AddError = "El tomo necesita un nombre"
			m.AddFocus = 0
			m.nameInput.Focus()
			m.pathInput.Blur()
			return m, nil
		}

		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			m.AddError = fmt.Sprintf("El sendero indicado no existe: %s", path)
			m.AddFocus = 1
			m.nameInput.Blur()
			m.pathInput.Focus()
			return m, nil
		}

		if _, err := m.registry.Add(name, path); err != nil {
			m.AddError = "El pergamino no aceptó el tomo: " + err.Error()
			return m, nil
		}

		m.AddError = ""
		m.Screen = ScreenProjects
		return m, loadProjectsCmd(m.registry)
	}

	// Delegate typing to the focused textinput
	var cmd tea.Cmd
	if m.AddFocus == 0 {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.pathInput, cmd = m.pathInput.Update(msg)
	}
	return m, cmd
}

// handleProjectActionsKeys handles key events on the project actions screen.
func (m Model) handleProjectActionsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		// Esc/q returns to projects without executing any action (S4.8, S6.5)
		m.Screen = ScreenProjects
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "j":
		fallthrough
	case msg.Type == tea.KeyDown:
		if m.ActionCursor < 2 {
			m.ActionCursor++
		}
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "k":
		fallthrough
	case msg.Type == tea.KeyUp:
		if m.ActionCursor > 0 {
			m.ActionCursor--
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		proj := m.findProject(m.SelectedID)
		if proj == nil {
			return m, nil
		}
		switch m.ActionCursor {
		case 0: // Open in Claude Code
			return m, runOpenClaudeCmd(m.opener, m.registry, proj.ID, proj.Path)
		case 1: // Spawn PowerShell
			return m, runPowerShellCmd(m.opener, proj.Path)
		case 2: // Copy path
			return m, runCopyPathCmd(m.clipboard, proj.Path)
		}
	}
	return m, nil
}

// handleConfirmDeleteKeys handles key events on the confirm delete screen.
func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "y":
		// Confirm deletion (S5.1, S5.2)
		if err := m.registry.Delete(m.SelectedID); err != nil {
			m.ActionFlash = "Un dragón rugió: " + err.Error()
		}
		m.SelectedID = ""
		m.Screen = ScreenProjects
		return m, loadProjectsCmd(m.registry)

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "n":
		fallthrough
	case msg.Type == tea.KeyEsc:
		fallthrough
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "q":
		// Cancel — no deletion (S5.2, S5.3, S6.6)
		m.Screen = ScreenProjects
		return m, nil
	}
	return m, nil
}
