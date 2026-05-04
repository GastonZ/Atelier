package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/transcripts"
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
	// ScreenAgentMonitor shows live tiles for active Claude sessions.
	ScreenAgentMonitor
	// ScreenAgentZoom shows the detail view for a single session tile.
	ScreenAgentZoom
	// ScreenAgentReplay shows step-by-step replay of a session's events.
	ScreenAgentReplay
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

	// --- agent monitor dependencies (injected via NewWithMonitor) ---
	agentScanner transcripts.Scanner
	agentWatcher transcripts.Watcher
	priceTable   transcripts.PriceTable
	atelierCfg   config.AtelierConfig

	// --- agent monitor state ---
	agentSessions    []transcripts.Session // active sessions, sorted mtime-desc
	AgentTileCursor  int                   // exported for tests — index into agentSessions
	agentExpanded    map[string]bool        // sessionID → sub-agent group expanded?
	AgentZoomedID    string                // exported for tests — empty when not zoomed
	AgentFlash       string                // exported for tests — transient error/warning
	agentFlashUntil  time.Time             // auto-clear flash after N seconds
	watcherCancel    func()                // closes watcher when leaving ScreenAgentMonitor
	watcherCancelSet bool                  // tracks whether cancel was called (for tests)
	agentWatcherCh   <-chan transcripts.Event // stored channel returned by Watch; used for drain re-queuing

	// --- replay state ---
	replayEvents []transcripts.Event // snapshot loaded at replay entry
	ReplayCursor int                 // exported for tests — mirror of replayCtrl.Cursor()
	ReplayPaused bool                // exported for tests — mirror of replayCtrl.Paused()
	ReplaySpeed  float64             // exported for tests — mirror of replayCtrl.Speed()
	replayCtrl   *transcripts.Replay // authoritative replay state
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

// NewWithMonitor returns a Model wired with both the existing dependencies and the
// new agent-monitor dependencies. cmd/atelier/main.go will switch to this in Batch 4.
// Tests use this to inject fakes for the monitor screens.
func NewWithMonitor(
	reg registry.Registry,
	op actions.Opener,
	cb actions.Clipboard,
	scanner transcripts.Scanner,
	watcher transcripts.Watcher,
	prices transcripts.PriceTable,
	cfg config.AtelierConfig,
) Model {
	m := New(reg, op, cb)
	m.agentScanner = scanner
	m.agentWatcher = watcher
	m.priceTable = prices
	m.atelierCfg = cfg
	m.agentExpanded = make(map[string]bool)
	return m
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

// initOrUpdateProjectList builds the bubbles list when dimensions are known,
// or refreshes its items if already built. Idempotent and safe to call from
// any handler (WindowSizeMsg, projectsLoadedMsg) without screen-state checks.
//
// The original gate "only init while on ScreenProjects" caused a bug where
// WindowSizeMsg fires once at startup on ScreenWelcome — ListInited stayed
// false forever, so the list never rendered when navigating into Projects.
func (m Model) initOrUpdateProjectList() Model {
	if m.Width <= 0 || m.Height <= 0 {
		return m
	}
	items := projectsToItems(m.projects)
	if !m.ListInited {
		reservedRows := 4 // title + footer + flash + spacer
		listHeight := m.Height - reservedRows
		if listHeight < 1 {
			listHeight = 1
		}
		m.list = newProjectList(m.Width, listHeight, items)
		m.ListInited = true
	} else {
		m.list.SetItems(items)
	}
	return m
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

// --- Agent monitor accessor/helpers (exported for tests) --------------------

// AgentSessions returns the active session slice (exported for tests).
func (m Model) AgentSessions() []transcripts.Session { return m.agentSessions }

// AgentExpanded returns true if the given session's sub-agents are expanded.
func (m Model) AgentExpanded(sessionID string) bool { return m.agentExpanded[sessionID] }

// WatcherCancelCalled returns true if watcherCancel was called (goroutine-leak test gate).
func (m Model) WatcherCancelCalled() bool { return m.watcherCancelSet }

// ReplayLen returns the number of events in the current replay snapshot.
func (m Model) ReplayLen() int { return len(m.replayEvents) }

// initAgentExpanded ensures agentExpanded is non-nil (safe to mutate in Update).
func (m Model) initAgentExpanded() Model {
	if m.agentExpanded == nil {
		m.agentExpanded = make(map[string]bool)
	}
	return m
}

// callWatcherCancel calls watcherCancel exactly once and marks it called.
func (m Model) callWatcherCancel() Model {
	if m.watcherCancel != nil && !m.watcherCancelSet {
		m.watcherCancel()
	}
	m.watcherCancelSet = true
	m.watcherCancel = nil
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
