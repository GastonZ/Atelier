package tui

// project_actions.go — single source of truth for the per-project actions menu
// (ScreenProjectActions). Both the renderer (view.go) and the key dispatcher
// (keys.go) derive the ordered item list from buildProjectActions, so labels and
// dispatch indices can never drift apart. Configurable agent launchers come
// first; the fixed actions follow, with the destructive "Borrar" last.

// actionKind tags how a menu item is dispatched on Enter.
type actionKind int

const (
	actionLauncher actionKind = iota // spawn a configured agent CLI via LaunchInDir
	actionVSCode
	actionPowerShell
	actionCopyPath
	actionMemory
	actionHistory
	actionDisk
	actionDelete
)

// projectAction is one row in the project actions menu.
type projectAction struct {
	label string
	kind  actionKind

	// launcher fields (only meaningful when kind == actionLauncher)
	name      string // bare launcher label, e.g. "Claude Code" (for flash text)
	command   string
	args      []string
	available bool // CLI resolves on PATH (display-only; dispatch still attempts)
}

// buildProjectActions returns the ordered actions menu for the current project:
// the configured launchers (each "Abrir en <label>") followed by the fixed
// actions. It is pure over the model's config, so view and keys stay in sync.
func (m Model) buildProjectActions() []projectAction {
	items := make([]projectAction, 0, len(m.atelierCfg.Launchers)+7)

	for _, l := range m.atelierCfg.Launchers {
		available := true
		if m.launcherAvailable != nil {
			available = m.launcherAvailable(l.Command)
		}
		items = append(items, projectAction{
			label:     "Abrir en " + l.Label,
			kind:      actionLauncher,
			name:      l.Label,
			command:   l.Command,
			args:      l.Args,
			available: available,
		})
	}

	items = append(items,
		projectAction{label: "Abrir en VS Code", kind: actionVSCode},
		projectAction{label: "Invocar PowerShell", kind: actionPowerShell},
		projectAction{label: "Copiar el sendero", kind: actionCopyPath},
		projectAction{label: "Ver memoria", kind: actionMemory},
		projectAction{label: "Ver historial", kind: actionHistory},
		projectAction{label: "Ver disco", kind: actionDisk},
		projectAction{label: "Borrar", kind: actionDelete},
	)
	return items
}
