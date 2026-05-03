package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/registry"
)

// Screen identifies the active TUI screen.
// New screens are added as additional iota constants — never remove or reorder existing ones.
// Values are not serialized but ordering is treated as stable to prevent future regressions.
type Screen int

const (
	// ScreenWelcome is the initial welcome / mission-control screen.
	ScreenWelcome Screen = iota
	// ScreenProjects shows the user's project list.
	ScreenProjects
	// ScreenAddProject shows the form to add a new project.
	ScreenAddProject
	// ScreenProjectActions shows the action menu for a selected project.
	ScreenProjectActions
	// ScreenConfirmDelete shows the deletion confirmation prompt.
	ScreenConfirmDelete
)

// Model holds ALL application state. No sub-models.
// Every Update branch returns a NEW Model (value semantics — immutability).
type Model struct {
	// --- existing fields (unchanged from bootstrap) ---
	Screen     Screen
	PrevScreen Screen
	Cursor     int
	Width      int
	Height     int
	Quitting   bool

	// --- dependencies (injected in New) ---
	registry  registry.Registry
	opener    actions.Opener
	clipboard actions.Clipboard

	// --- list section (ScreenProjects) ---
	projects   []registry.Project // cached snapshot, refreshed on screen entry
	list       list.Model         // bubbles/list, lazy-init on first WindowSizeMsg
	ListInited bool               // exported for tests — true once list has been sized & seeded

	// --- selection (ScreenProjectActions, ScreenConfirmDelete) ---
	SelectedID   string // exported for tests — UUID of project the action/delete targets
	ActionCursor int    // exported for tests — 0=Claude Code, 1=PowerShell, 2=Copy Path

	// --- add form (ScreenAddProject) ---
	nameInput textinput.Model
	pathInput textinput.Model
	AddFocus  int    // exported for tests — 0=name, 1=path
	AddError  string // exported for tests — empty=no error; non-empty=inline error

	// --- transient feedback ---
	ActionFlash string // exported for tests — last action result, cleared on next nav
}

// New returns a Model wired with the production-or-mock dependencies.
// main.go calls tui.New(registry.NewFileRegistry(), actions.NewOpener(), actions.NewClipboard()).
// Tests inject mocks.
func New(reg registry.Registry, op actions.Opener, cb actions.Clipboard) Model {
	name := textinput.New()
	name.Placeholder = "Nombre del tomo"
	name.Prompt = "> "
	name.CharLimit = 64
	name.Focus()

	path := textinput.New()
	path.Placeholder = `C:\Sendero\hacia\el\tomo`
	path.Prompt = "> "
	path.CharLimit = 1024

	return Model{
		Screen:    ScreenWelcome,
		registry:  reg,
		opener:    op,
		clipboard: cb,
		nameInput: name,
		pathInput: path,
		AddFocus:  0,
	}
}

// Init is called by Bubble Tea on program start.
// Returns nil — no initial commands needed.
func (m Model) Init() tea.Cmd {
	return nil
}

// newProjectList constructs a bubbles/list.Model sized for the current terminal.
// CRITICAL: KeyMap.Quit and KeyMap.ForceQuit are neutered to prevent bubbles/list
// from intercepting q and ctrl+c — our Update handler owns those keys.
func newProjectList(width, height int, items []list.Item) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	// NEUTERED — our handler owns q/esc. This prevents bubbles/list quit-hijack.
	l.KeyMap.Quit = key.NewBinding()
	l.KeyMap.ForceQuit = key.NewBinding()
	return l
}

// projectItem wraps registry.Project to satisfy the list.Item interface.
type projectItem struct {
	project registry.Project
}

func (i projectItem) FilterValue() string { return i.project.Name }
func (i projectItem) Title() string       { return i.project.Name }
func (i projectItem) Description() string { return i.project.Path }

// projectsToItems converts a project slice to list.Item slice.
func projectsToItems(projects []registry.Project) []list.Item {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p}
	}
	return items
}

// findProject returns the project with the given id, or nil if not found.
func (m Model) findProject(id string) *registry.Project {
	for i, p := range m.projects {
		if p.ID == id {
			return &m.projects[i]
		}
	}
	return nil
}

// resetAddForm clears the add form to a clean state.
func (m Model) resetAddForm() Model {
	m.nameInput.SetValue("")
	m.pathInput.SetValue("")
	m.nameInput.Focus()
	m.pathInput.Blur()
	m.AddFocus = 0
	m.AddError = ""
	return m
}

// --- Test helper exports ---
// These exported functions expose internal state for white-box testing.
// They are NOT part of the public API and must only be used from _test.go files.

// NameInputValue returns the current value of the name input field.
func (m Model) NameInputValue() string { return m.nameInput.Value() }

// PathInputValue returns the current value of the path input field.
func (m Model) PathInputValue() string { return m.pathInput.Value() }

// SetNameInput returns a new Model with the name input set to the given value.
func SetNameInput(m Model, v string) Model {
	m.nameInput.SetValue(v)
	return m
}

// SetPathInput returns a new Model with the path input set to the given value.
func SetPathInput(m Model, v string) Model {
	m.pathInput.SetValue(v)
	return m
}

// SetAddError returns a new Model with AddError set to the given message.
func SetAddError(m Model, msg string) Model {
	m.AddError = msg
	return m
}

// SetActionFlash returns a new Model with ActionFlash set to the given message.
func SetActionFlash(m Model, msg string) Model {
	m.ActionFlash = msg
	return m
}

// DrainProjectsLoaded drives the loadProjectsCmd and feeds the result back into Update,
// ensuring m.projects is populated before test assertions run.
func DrainProjectsLoaded(t interface{ Helper(); Fatalf(string, ...interface{}) }, m Model) Model {
	t.Helper()
	cmd := loadProjectsCmd(m.registry)
	if cmd == nil {
		return m
	}
	msg := cmd()
	result, _ := m.Update(msg)
	got, ok := result.(Model)
	if !ok {
		t.Fatalf("DrainProjectsLoaded: Update() returned %T, want Model", result)
	}
	return got
}
