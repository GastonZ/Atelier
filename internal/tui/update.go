package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/registry"
)

// Update is the Bubble Tea message handler.
// It processes all incoming messages and returns a new Model (immutable — value semantics).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Lazy list init: construct the list on first WindowSizeMsg while on ScreenProjects.
		// The list requires non-zero dimensions — we never init it before receiving a size.
		if m.Screen == ScreenProjects && !m.ListInited && m.Width > 0 && m.Height > 0 {
			reservedRows := 4 // title + footer + flash + spacer
			listHeight := m.Height - reservedRows
			if listHeight < 1 {
				listHeight = 1
			}
			items := projectsToItems(m.projects)
			m.list = newProjectList(m.Width, listHeight, items)
			m.ListInited = true
		}
		return m, nil

	case projectsLoadedMsg:
		if msg.err != nil {
			m.ActionFlash = "Los tomos están sellados: " + msg.err.Error()
			return m, nil
		}
		m.projects = msg.projects
		if m.ListInited {
			items := projectsToItems(m.projects)
			m.list.SetItems(items)
		}
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.ActionFlash = "Un dragón rugió: " + msg.err.Error()
		} else {
			m.ActionFlash = msg.flash
		}
		if m.Screen == ScreenProjectActions {
			m.Screen = ScreenProjects
		}
		return m, loadProjectsCmd(m.registry)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// Run creates and starts the Bubble Tea program with alt-screen mode.
// Alt-screen is mandatory for Windows Terminal / PowerShell rendering.
// Called by cmd/atelier/main.go after wiring the registry and actions.
func Run(reg registry.Registry, op actions.Opener, cb actions.Clipboard) error {
	m := New(reg, op, cb)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// listSelectedProject returns the currently highlighted project from the bubbles/list.
// Returns nil if the list is empty or no item is selected.
func (m Model) listSelectedProject() *list.Item {
	if len(m.projects) == 0 {
		return nil
	}
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	return &item
}
