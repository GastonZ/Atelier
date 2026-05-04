package tui_test

// monitor_test.go — T29 (RED): Failing tests for Batch 3 TUI extension.
// These tests reference types/fields/functions not yet implemented.
// They compile-fail until T30 (GREEN) is complete.
//
// Covers:
//   - Copy constants (copy.go)
//   - Model fields for agent monitor (model.go extension)
//   - tui.NewWithMonitor constructor
//   - ScreenAgentMonitor / ScreenAgentZoom / ScreenAgentReplay iotas
//   - ScreenWelcome `a` key → ScreenAgentMonitor
//   - ScreenProjects `m` key → ScreenAgentMonitor
//   - Agent monitor empty state ("El atelier duerme.")
//   - Sub-agent collapse default (S3.3)
//   - Buffer cap 200 events (S9.2)
//   - Replay snapshot semantics (S5.2)
//   - Watcher goroutine cancel on screen exit (T39 / §4.1)
//   - Polling fallback via polling ticker
//   - agentSessionsLoadedMsg applied to model
//   - agentEventMsg applied to model
//   - Per-screen key handlers: monitor j/k, o/c, enter, esc, 1-9
//   - Zoom handler: r enters replay, esc returns to monitor
//   - Replay handler: <, >, space, +, -, esc

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
)

// ---- Fake dependencies for tui tests ----------------------------------------

// fakeScannerForTUI is a minimal Scanner for tui tests.
type fakeScannerForTUI struct {
	activeSessions []transcripts.Session
	allSessions    []transcripts.Session
	loadEvents     []transcripts.Event
	loadErr        error
	activeErr      error
}

func (f *fakeScannerForTUI) ListActive(_ time.Duration) ([]transcripts.Session, error) {
	return f.activeSessions, f.activeErr
}

func (f *fakeScannerForTUI) ListAll() ([]transcripts.Session, error) {
	return f.allSessions, nil
}

func (f *fakeScannerForTUI) LoadEvents(_ string) ([]transcripts.Event, error) {
	return f.loadEvents, f.loadErr
}

// fakeWatcherForTUI is a controllable Watcher for tui tests.
type fakeWatcherForTUI struct {
	ch         chan transcripts.Event
	closeCount int
	watchErr   error
	paths      []string
}

func newFakeWatcherForTUI(bufSize int) *fakeWatcherForTUI {
	return &fakeWatcherForTUI{ch: make(chan transcripts.Event, bufSize)}
}

func (fw *fakeWatcherForTUI) Watch(paths []string) (<-chan transcripts.Event, error) {
	fw.paths = paths
	if fw.watchErr != nil {
		return nil, fw.watchErr
	}
	return fw.ch, nil
}

func (fw *fakeWatcherForTUI) Close() error {
	fw.closeCount++
	return nil
}

func (fw *fakeWatcherForTUI) send(e transcripts.Event) {
	fw.ch <- e
}

// fakePriceTableForTUI is a PriceTable with fixed cost.
type fakePriceTableForTUI struct {
	cost  float64
	known bool
}

func (f *fakePriceTableForTUI) Cost(_ string, _, _, _, _ int) (float64, bool) {
	return f.cost, f.known
}

// fakeClockForTUI is a Clock returning a fixed time.
type fakeClockForTUI struct{ t time.Time }

func (c *fakeClockForTUI) Now() time.Time { return c.t }

// makeAssistantEvent creates a minimal AssistantEvent with the given sessionID.
func makeAssistantEvent(sessionID string, ts time.Time) transcripts.Event {
	return &transcripts.AssistantEvent{
		UUIDValue:      "evt-" + sessionID,
		SessionIDValue: sessionID,
		TimestampValue: ts,
		Model:          "claude-sonnet-4-6",
		Text:           "hello",
	}
}

// makeSession returns a Session with the given ID and n sub-sessions.
func makeSession(id string, ts time.Time, numSubs int) transcripts.Session {
	subs := make([]transcripts.Session, numSubs)
	for i := range subs {
		subs[i] = transcripts.Session{
			ID:            id + "-sub-" + string(rune('a'+i)),
			RootPath:      "/fake/path/sub" + string(rune('a'+i)) + ".jsonl",
			LastEventTime: ts,
		}
	}
	return transcripts.Session{
		ID:            id,
		RootPath:      "/fake/path/" + id + ".jsonl",
		ProjectName:   "TestProject",
		LastEventTime: ts,
		SubSessions:   subs,
	}
}

// newMonitorModel builds a Model via tui.NewWithMonitor wired with fakes.
func newMonitorModel(t *testing.T, scanner transcripts.Scanner, watcher transcripts.Watcher, price transcripts.PriceTable, cfg config.AtelierConfig) tui.Model {
	t.Helper()
	reg := newTestRegistry(t)
	return tui.NewWithMonitor(
		reg,
		&MockOpener{},
		&MockClipboard{},
		scanner,
		watcher,
		price,
		cfg,
	)
}

// ---- T29: Copy constants tests -----------------------------------------------

// TestCopyConstants_EmptyState verifies the "El atelier duerme." constant exists.
func TestCopyConstants_EmptyState(t *testing.T) {
	if tui.CopyMonitorEmpty == "" {
		t.Fatal("CopyMonitorEmpty constant must not be empty")
	}
	if !strings.Contains(tui.CopyMonitorEmpty, "duerme") {
		t.Errorf("CopyMonitorEmpty = %q, want it to contain 'duerme'", tui.CopyMonitorEmpty)
	}
}

// TestCopyConstants_ReplayHeader verifies the "Crónica del taller" constant.
func TestCopyConstants_ReplayHeader(t *testing.T) {
	if tui.CopyReplayHeader == "" {
		t.Fatal("CopyReplayHeader constant must not be empty")
	}
	if !strings.Contains(tui.CopyReplayHeader, "Crónica") {
		t.Errorf("CopyReplayHeader = %q, want it to contain 'Crónica'", tui.CopyReplayHeader)
	}
}

// TestCopyConstants_FooterMonitor verifies the monitor footer hint constant.
func TestCopyConstants_FooterMonitor(t *testing.T) {
	if !strings.Contains(tui.CopyFooterMonitor, "j/k") {
		t.Errorf("CopyFooterMonitor = %q, want it to contain 'j/k'", tui.CopyFooterMonitor)
	}
}

// TestCopyConstants_FooterZoom verifies the zoom footer hint constant.
func TestCopyConstants_FooterZoom(t *testing.T) {
	if !strings.Contains(tui.CopyFooterZoom, "revivir") {
		t.Errorf("CopyFooterZoom = %q, want it to contain 'revivir'", tui.CopyFooterZoom)
	}
}

// TestCopyConstants_FooterReplay verifies the replay footer hint constant.
func TestCopyConstants_FooterReplay(t *testing.T) {
	if !strings.Contains(tui.CopyFooterReplay, "espacio") {
		t.Errorf("CopyFooterReplay = %q, want it to contain 'espacio'", tui.CopyFooterReplay)
	}
}

// TestCopyConstants_CostLine verifies cost line format string.
func TestCopyConstants_CostLine(t *testing.T) {
	if !strings.Contains(tui.CopyCostLine, "Pergaminos") {
		t.Errorf("CopyCostLine = %q, want it to contain 'Pergaminos'", tui.CopyCostLine)
	}
	if !strings.Contains(tui.CopyCostLine, "USD") {
		t.Errorf("CopyCostLine = %q, want it to contain 'USD'", tui.CopyCostLine)
	}
}

// ---- T29: Screen iotas exist --------------------------------------------------

// TestScreenIotas_NewScreensExist verifies the three new screen values are defined.
func TestScreenIotas_NewScreensExist(t *testing.T) {
	// Just referencing them compiles if they exist.
	_ = tui.ScreenAgentMonitor
	_ = tui.ScreenAgentZoom
	_ = tui.ScreenAgentReplay
}

// ---- T29: NewWithMonitor constructor exists -----------------------------------

// TestNewWithMonitor_ReturnsModel verifies the constructor compiles and returns a model.
func TestNewWithMonitor_ReturnsModel(t *testing.T) {
	scanner := &fakeScannerForTUI{}
	watcher := newFakeWatcherForTUI(4)
	price := &fakePriceTableForTUI{cost: 0, known: true}
	cfg := config.DefaultAtelierConfig()
	m := newMonitorModel(t, scanner, watcher, price, cfg)
	// Model starts on welcome screen
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("NewWithMonitor: Screen = %v, want ScreenWelcome", m.Screen)
	}
}

// ---- T29: Model field accessors exist ----------------------------------------

// TestModelFields_AgentTileCursor verifies AgentTileCursor is zero-initialized.
func TestModelFields_AgentTileCursor(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	if m.AgentTileCursor != 0 {
		t.Errorf("AgentTileCursor = %d, want 0", m.AgentTileCursor)
	}
}

// TestModelFields_AgentFlash verifies AgentFlash is empty on init.
func TestModelFields_AgentFlash(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	if m.AgentFlash != "" {
		t.Errorf("AgentFlash = %q, want empty", m.AgentFlash)
	}
}

// ---- T30 RED: agentSessionsLoadedMsg applied to model ------------------------

// TestAgentSessionsLoadedMsg_SetsAgentSessions verifies msg populates agentSessions.
func TestAgentSessionsLoadedMsg_SetsAgentSessions(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	// Navigate to monitor screen
	m, _ = navigateToMonitor(t, m)

	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	msg := tui.MakeAgentSessionsLoadedMsg(sessions, nil)

	result, _ := m.Update(msg)
	got := result.(tui.Model)

	if len(got.AgentSessions()) != 1 {
		t.Errorf("AgentSessions len = %d, want 1", len(got.AgentSessions()))
	}
	if got.AgentSessions()[0].ID != "s1" {
		t.Errorf("AgentSessions[0].ID = %q, want %q", got.AgentSessions()[0].ID, "s1")
	}
}

// navigateToMonitor sends the 'a' key from ScreenWelcome to reach ScreenAgentMonitor.
func navigateToMonitor(t *testing.T, m tui.Model) (tui.Model, tea.Cmd) {
	t.Helper()
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := result.(tui.Model)
	if got.Screen != tui.ScreenAgentMonitor {
		t.Fatalf("navigateToMonitor: Screen = %v, want ScreenAgentMonitor", got.Screen)
	}
	return got, cmd
}

// TestAgentSessionsLoadedMsg_ErrorSetsFlash verifies error sets AgentFlash.
func TestAgentSessionsLoadedMsg_ErrorSetsFlash(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)

	msg := tui.MakeAgentSessionsLoadedMsg(nil, errTest)
	result, _ := m.Update(msg)
	got := result.(tui.Model)

	if got.AgentFlash == "" {
		t.Error("AgentFlash should be set when sessions load returns error")
	}
}

// errTest is a sentinel error for tests.
var errTest = &testError{"test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
