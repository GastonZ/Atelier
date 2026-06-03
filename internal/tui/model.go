package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/audio"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/git"
	"github.com/gastonz/atelier/internal/nowplaying"
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
	// ScreenMemoryBrowser shows engram observations for the selected project.
	ScreenMemoryBrowser
	// ScreenProjectHistory shows unified git+SDD history for the selected project.
	ScreenProjectHistory
	// ScreenDiskUsage shows disk usage breakdown (engram DB + claude projects).
	ScreenDiskUsage
)

// HistoryEntry is one item in the unified git+SDD history list.
// Source is "git" or "sdd"; Title is commit subject or archive title; Ref is hash or observation ID string.
type HistoryEntry struct {
	Source string    // "git" | "sdd"
	Date   time.Time // git: %ad parsed; sdd: observation.Timestamp
	Title  string    // git: commit subject; sdd: archive title
	Ref    string    // git: hash; sdd: strconv.FormatInt(id, 10)
}

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

	// --- now-playing (welcome screen ambient widget) ---
	nowPlaying   nowplaying.Provider // injected; nil = card never shown
	currentTrack nowplaying.Track    // latest snapshot, refreshed by the ticker

	// --- audio visualizer ---
	audio     audio.Analyzer // injected; nil = static bars only
	barLevels []float64      // latest live spectrum, refreshed by the anim ticker

	// --- list section (ScreenProjects) ---
	projects   []registry.Project // cached snapshot, refreshed on screen entry
	list       list.Model         // bubbles/list, lazy-init on first WindowSizeMsg
	ListInited bool               // exported for tests — true once list has been sized & seeded

	// --- selection (ScreenProjectActions, ScreenConfirmDelete) ---
	SelectedID   string // exported for tests — UUID of project the action/delete targets
	// ActionCursor range: 0-7.
	// 0=Claude Code, 1=VS Code, 2=PowerShell, 3=Copy Path, 4=Memory, 5=History, 6=Disk, 7=Borrar
	ActionCursor int

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

	// launcherAvailable reports whether a launcher command resolves on PATH
	// (display-only greying). Defaults to actions.CommandAvailable in New;
	// tests inject a deterministic stub.
	launcherAvailable func(string) bool

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

	// --- daily driver pack dependencies (injected via NewWithDailyPack) ---
	engramClient    engram.Client    // injected; nil-safe: ops flash error when nil
	gitStatusReader git.StatusReader // injected; nil-safe
	gitLogReader    git.LogReader    // injected; nil-safe

	// --- ScreenMemoryBrowser state ---
	memoryEntries    []engram.Observation // loaded list for current tomo
	memoryCursor     int                  // index for non-filtered nav
	memoryViewing    *engram.Observation  // non-nil = detail viewport active
	memoryFilterText string               // mirror of memoryList filter input
	memoryLoading    bool                 // spinner while loadMemoryCmd in flight
	memoryError      string               // error flash for memory ops
	memoryList       list.Model           // bubbles/list with filtering enabled
	memoryViewport   viewport.Model       // detail body scroll

	// --- ScreenProjects extension ---
	projectFilterText string              // mirror of ScreenProjects filter input
	gitStatusCache    map[string]git.Status // keyed by tomo Path; no TTL; refresh on 'r'
	gitStatusLoading  bool                  // while fan-out in flight

	// --- ScreenProjectHistory state ---
	historyEntries      []HistoryEntry  // merged + sorted (date desc; sdd below git on same date)
	historyCursor       int             // index in entries when in list mode
	historyViewingRef   string          // "" = list mode; commit hash or sdd id = detail mode
	historyDetailBody   string          // git show output OR archive content for viewport
	historyLoading      bool            // spinner while load in flight
	historyDetailLoading bool           // spinner while detail load in flight
	historyError        string          // error flash
	historyList         list.Model      // bubbles/list for entries
	historyViewport     viewport.Model  // detail scroll

	// --- ScreenDiskUsage state ---
	diskEngramBytes        int64             // ~/.engram/*.db* total
	diskClaudeProjectsTotal int64            // ~/.claude/projects/ total
	diskPerTomo            map[string]int64  // keyed by tomo ID; 0 = "Sin crónica"
	diskCursor             int               // selected disk row
	diskLoading            bool              // spinner while WalkDir in flight
	diskError              string            // error flash
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
		Screen:            ScreenWelcome,
		registry:          reg,
		opener:            op,
		clipboard:         cb,
		nameInput:         name,
		pathInput:         path,
		AddFocus:          0,
		atelierCfg:        config.DefaultAtelierConfig(),
		launcherAvailable: actions.CommandAvailable,
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
// When a now-playing provider is wired, it kicks off the ambient polling loop:
// an immediate fetch plus a recurring tick. With no provider it returns nil.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.nowPlaying != nil {
		cmds = append(cmds, loadNowPlayingCmd(m.nowPlaying), nowPlayingTickCmd())
	}
	if m.audio != nil {
		cmds = append(cmds, animTickCmd())
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// newProjectList constructs a bubbles/list.Model sized for the current terminal.
// CRITICAL: KeyMap.Quit and KeyMap.ForceQuit are neutered to prevent bubbles/list
// from intercepting q and ctrl+c — our Update handler owns those keys.
func newProjectList(width, height int, items []list.Item) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	// T31: Enable filtering for '/' project search (R6.1).
	// The filter-precedence guard in handleProjectsKeys ensures our esc/enter
	// handler is bypassed when the list is in filter mode.
	l.SetFilteringEnabled(true)
	// NEUTERED — our handler owns q/esc. This prevents bubbles/list quit-hijack.
	l.KeyMap.Quit = key.NewBinding()
	l.KeyMap.ForceQuit = key.NewBinding()
	return l
}

// projectItem wraps registry.Project to satisfy the list.Item interface.
type projectItem struct {
	project   registry.Project
	indicator string // git status indicator (e.g. "✓", "M3", "?") — empty if not loaded
}

func (i projectItem) FilterValue() string { return i.project.Name }
func (i projectItem) Title() string {
	if i.indicator != "" {
		return i.project.Name + "  " + i.indicator
	}
	return i.project.Name
}
func (i projectItem) Description() string { return i.project.Path }

// projectsToItems converts a project slice to list.Item slice (no git indicators).
func projectsToItems(projects []registry.Project) []list.Item {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p}
	}
	return items
}

// projectsToItemsWithStatus converts a project slice to list.Item slice with git indicators.
func projectsToItemsWithStatus(projects []registry.Project, cache map[string]git.Status) []list.Item {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		indicator := ""
		if cache != nil {
			if s, ok := cache[p.Path]; ok {
				indicator = git.FormatGitIndicator(s)
			}
		}
		items[i] = projectItem{project: p, indicator: indicator}
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

// --- Daily driver pack accessor/helpers (exported for tests) -----------------

// GitStatusCache returns the current git status cache (exported for tests).
func (m Model) GitStatusCache() map[string]git.Status { return m.gitStatusCache }

// DiskLoaded returns true if the disk usage data has been loaded (exported for tests).
func (m Model) DiskLoaded() bool { return m.diskLoading == false && m.diskEngramBytes > 0 || m.diskPerTomo != nil }

// MemoryLoading returns true if memory is currently loading.
func (m Model) MemoryLoading() bool { return m.memoryLoading }

// MemoryEntries returns the loaded memory entries (exported for tests).
func (m Model) MemoryEntries() []engram.Observation { return m.memoryEntries }

// MemoryViewing returns the currently viewed observation, or nil.
func (m Model) MemoryViewing() *engram.Observation { return m.memoryViewing }

// HistoryEntries returns the loaded history entries (exported for tests).
func (m Model) HistoryEntries() []HistoryEntry { return m.historyEntries }

// HistoryViewingRef returns the current detail ref ("" = list mode).
func (m Model) HistoryViewingRef() string { return m.historyViewingRef }

// DiskEngramBytes returns the total engram DB bytes (exported for tests).
func (m Model) DiskEngramBytes() int64 { return m.diskEngramBytes }

// DiskCursor returns the disk usage screen cursor.
func (m Model) DiskCursor() int { return m.diskCursor }

// NewWithDailyPack returns a Model wired with the daily-driver-pack dependencies.
// Extends NewWithMonitor with engram.Client, git.StatusReader, git.LogReader.
func NewWithDailyPack(
	reg registry.Registry,
	op actions.Opener,
	cb actions.Clipboard,
	scanner transcripts.Scanner,
	watcher transcripts.Watcher,
	prices transcripts.PriceTable,
	cfg config.AtelierConfig,
	engramCl engram.Client,
	statusR git.StatusReader,
	logR git.LogReader,
) Model {
	m := NewWithMonitor(reg, op, cb, scanner, watcher, prices, cfg)
	m.engramClient = engramCl
	m.gitStatusReader = statusR
	m.gitLogReader = logR
	return m
}

// SetMemoryErrorForTest is a test helper to set memoryError.
func SetMemoryErrorForTest(m Model, err string) Model {
	m.memoryError = err
	return m
}

// SetHistoryErrorForTest is a test helper to set historyError.
func SetHistoryErrorForTest(m Model, err string) Model {
	m.historyError = err
	return m
}

// SetDiskErrorForTest is a test helper to set diskError.
func SetDiskErrorForTest(m Model, err string) Model {
	m.diskError = err
	return m
}

// SetDiskLoadingForTest is a test helper to set diskLoading state.
func SetDiskLoadingForTest(m Model, loading bool) Model {
	m.diskLoading = loading
	return m
}

// SetMemoryLoadingForTest is a test helper to set memoryLoading state.
func SetMemoryLoadingForTest(m Model, loading bool) Model {
	m.memoryLoading = loading
	return m
}

// SetHistoryLoadingForTest is a test helper to set historyLoading state.
func SetHistoryLoadingForTest(m Model, loading bool) Model {
	m.historyLoading = loading
	return m
}

// InjectDailyPackDeps is a test helper that injects daily-driver-pack dependencies
// into an existing Model without requiring NewWithDailyPack (for retrofitting test models).
func InjectDailyPackDeps(m Model, c engram.Client, sr git.StatusReader, lr git.LogReader) Model {
	m.engramClient = c
	m.gitStatusReader = sr
	m.gitLogReader = lr
	return m
}

// InjectNowPlaying returns a new Model with the now-playing provider set.
// Used by the production wiring (RunWithDailyPack) and by tests.
func InjectNowPlaying(m Model, p nowplaying.Provider) Model {
	m.nowPlaying = p
	return m
}

// CurrentTrack returns the latest now-playing snapshot (exported for tests).
func (m Model) CurrentTrack() nowplaying.Track { return m.currentTrack }

// SetCurrentTrackForTest sets the current track directly (test helper).
func SetCurrentTrackForTest(m Model, t nowplaying.Track) Model {
	m.currentTrack = t
	return m
}

// InjectAudio returns a new Model with the audio analyzer set.
func InjectAudio(m Model, a audio.Analyzer) Model {
	m.audio = a
	return m
}

// BarLevels returns the latest live spectrum levels (exported for tests).
func (m Model) BarLevels() []float64 { return m.barLevels }

// SetBarLevelsForTest sets the bar levels directly (test helper).
func SetBarLevelsForTest(m Model, levels []float64) Model {
	m.barLevels = levels
	return m
}

// SetScreenForTest is a test helper to set the screen without navigation flow.
func SetScreenForTest(m Model, s Screen) Model {
	m.Screen = s
	return m
}

// SetHistoryViewingRefForTest is a test helper to set historyViewingRef directly.
func SetHistoryViewingRefForTest(m Model, ref string) Model {
	m.historyViewingRef = ref
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
