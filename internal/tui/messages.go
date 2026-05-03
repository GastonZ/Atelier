package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/registry"
)

// projectsLoadedMsg is dispatched after registry.List() completes asynchronously.
type projectsLoadedMsg struct {
	projects []registry.Project
	err      error
}

// loadProjectsCmd reads the registry asynchronously and returns a tea.Cmd.
// It is exported for test helpers (DrainProjectsLoaded) but remains lowercase-callable.
func loadProjectsCmd(reg registry.Registry) tea.Cmd {
	return func() tea.Msg {
		ps, err := reg.List()
		return projectsLoadedMsg{projects: ps, err: err}
	}
}

// actionDoneMsg is dispatched after an Opener or Clipboard action completes.
type actionDoneMsg struct {
	flash string // user-facing fantasy-themed result copy
	err   error
}

// runOpenClaudeCmd executes OpenInClaudeCode and Touch(id) asynchronously.
func runOpenClaudeCmd(op Openers, reg registry.Registry, id, path string) tea.Cmd {
	return func() tea.Msg {
		err := op.OpenInClaudeCode(path)
		if err != nil {
			return actionDoneMsg{flash: "Un dragón rugió: " + err.Error(), err: err}
		}
		touchErr := reg.Touch(id)
		if touchErr != nil {
			return actionDoneMsg{flash: "Tomo abierto (no se pudo registrar la lectura)", err: nil}
		}
		return actionDoneMsg{flash: "Tomo abierto en Claude Code"}
	}
}

// runPowerShellCmd executes SpawnPowerShell asynchronously (does NOT touch registry).
func runPowerShellCmd(op Openers, path string) tea.Cmd {
	return func() tea.Msg {
		err := op.SpawnPowerShell(path)
		if err != nil {
			return actionDoneMsg{flash: "Un dragón rugió: " + err.Error(), err: err}
		}
		return actionDoneMsg{flash: "PowerShell invocado en el tomo"}
	}
}

// runCopyPathCmd writes path to the clipboard asynchronously.
func runCopyPathCmd(cb Clipboards, path string) tea.Cmd {
	return func() tea.Msg {
		err := cb.WriteAll(path)
		if err != nil {
			return actionDoneMsg{flash: "Un dragón rugió: " + err.Error(), err: err}
		}
		return actionDoneMsg{flash: "Sendero copiado al pergamino"}
	}
}

// Openers is a local alias to avoid circular naming; mirrors actions.Opener.
type Openers interface {
	OpenInClaudeCode(projectPath string) error
	SpawnPowerShell(projectPath string) error
}

// Clipboards is a local alias; mirrors actions.Clipboard.
type Clipboards interface {
	WriteAll(text string) error
}
