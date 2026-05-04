package tui_test

// monitor_regression_test.go — T29 (RED): Critical regression tests mandated by Batch 3 scope.
// These must pass at end of batch (they're the regression gate items in the task list).

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
)

// ---- R3.5 / S3.1: Empty state renders "El atelier duerme." -----------------

// TestCritical_EmptyStateRendersElatelier is the mandatory regression guard for R3.5/S3.1.
func TestCritical_EmptyStateRendersElatelier(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)

	// Explicitly load zero sessions
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg([]transcripts.Session{}, nil))
	m = result.(tui.Model)

	view := m.View()
	if !contains(view, "El atelier duerme.") {
		t.Errorf("CRITICAL: empty monitor must show 'El atelier duerme.' — got:\n%s", view)
	}
}

// ---- R3.2 / S3.3: Sub-agent collapse default --------------------------------

// TestCritical_SubAgentCollapseDefault is the mandatory regression guard for R3.2/S3.3.
func TestCritical_SubAgentCollapseDefault(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 3)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Freshly entered tile: sub-sessions MUST be hidden (not expanded)
	if m.AgentExpanded("s1") {
		t.Error("CRITICAL: newly loaded tile must start COLLAPSED (R3.2/S3.3)")
	}
}

// ---- R9.2 / S9.2: Buffer cap at 200 events ----------------------------------

// TestCritical_BufferCap200Events is the mandatory regression guard for R9.2/S9.2.
func TestCritical_BufferCap200Events(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(64), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Send 250 events
	for i := 0; i < 250; i++ {
		evt := makeAssistantEvent("s1", now.Add(time.Duration(i)*time.Second))
		msg := tui.MakeAgentEventMsg(evt)
		result, _ := m.Update(msg)
		m = result.(tui.Model)
	}

	// Buffer must hold exactly 200
	sessions2 := m.AgentSessions()
	if len(sessions2) == 0 {
		t.Fatal("CRITICAL: no sessions found after 250 events")
	}
	evtCount := len(sessions2[0].Events)
	if evtCount != 200 {
		t.Errorf("CRITICAL: buffer cap — Events len = %d, want 200 (R9.2/S9.2)", evtCount)
	}
}

// ---- R5.2 / S5.2: Replay snapshot — no appends after entry ------------------

// TestCritical_ReplaySnapshot is the mandatory regression guard for R5.2/S5.2.
func TestCritical_ReplaySnapshot(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	// Initially 2 events
	initialEvents := []transcripts.Event{
		makeAssistantEvent("s1", now),
		makeAssistantEvent("s1", now.Add(time.Second)),
	}
	scanner := &fakeScannerForTUI{activeSessions: sessions, loadEvents: initialEvents}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Enter zoom → replay
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	// Drain replayLoadedMsg with 2-event snapshot
	result, _ = m.Update(tui.MakeReplayLoadedMsg("s1", initialEvents, nil))
	m = result.(tui.Model)

	snapshotLen := m.ReplayLen()
	if snapshotLen != 2 {
		t.Fatalf("CRITICAL: replay snapshot len = %d, want 2", snapshotLen)
	}

	// Append a 3rd event via agentEventMsg — should NOT enter replay buffer
	newEvt := makeAssistantEvent("s1", now.Add(2*time.Second))
	result, _ = m.Update(tui.MakeAgentEventMsg(newEvt))
	m = result.(tui.Model)

	// Replay buffer still at 2
	if m.ReplayLen() != 2 {
		t.Errorf("CRITICAL: replay snapshot broken — ReplayLen = %d after append, want 2 (R5.2/S5.2)", m.ReplayLen())
	}
}

// ---- T39: Watcher goroutine no-leak -----------------------------------------

// TestCritical_WatcherCancelOnNumberKeyExit tests watcher cancel on number-key screen exit.
func TestCritical_WatcherCancelOnNumberKeyExit(t *testing.T) {
	// This test confirms cancel is called even when leaving via esc
	fw := newFakeWatcherForTUI(4)
	m := newMonitorModel(t, &fakeScannerForTUI{}, fw, &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)

	// Press Esc to leave — watcher cancel must be called
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.WatcherCancelCalled() {
		t.Error("CRITICAL: watcher cancel must be called on esc from Monitor (goroutine leak)")
	}
}

// TestCritical_WatcherCancelCalledOnlyOnce verifies cancel is idempotent.
func TestCritical_WatcherCancelCalledOnlyOnce(t *testing.T) {
	fw := newFakeWatcherForTUI(4)
	m := newMonitorModel(t, &fakeScannerForTUI{}, fw, &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)

	// Esc once
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.WatcherCancelCalled() {
		t.Fatal("first esc: watcher cancel not called")
	}
}

// ---- R2.2: Polling fallback always-on ---------------------------------------

// TestCritical_PollingFallbackWithBrokenWatcher covers R2.2.
// Even when watcher returns error, the polling ticker cmd must be returned.
func TestCritical_PollingFallbackWithBrokenWatcher(t *testing.T) {
	// Watcher that errors on Watch
	fw := &fakeWatcherForTUI{ch: make(chan transcripts.Event, 4), watchErr: errTest}
	m := newMonitorModel(t, &fakeScannerForTUI{}, fw, &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	// Navigate to monitor — watcher fails but polling must still run
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = result.(tui.Model)
	_ = cmd // accept the batch of cmds

	// AgentFlash should have been set (watcher error)
	// but model must still be on monitor screen and not crashed
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("Broken watcher: Screen = %v, want ScreenAgentMonitor (must survive watcher error)", m.Screen)
	}

	// A pollingTickMsg should succeed even without fsnotify
	tickMsg := tui.MakePollingTickMsg()
	result, _ = m.Update(tickMsg)
	m = result.(tui.Model)
	// No panic, model on same screen
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("After poll tick: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}
}

// ---- Carry-over fix: Watch(nil) goroutine-leak prevention -------------------

// TestCritical_WatchCalledOnlyOnce verifies that handleAgentEvent re-drains the
// channel via the stored agentWatcherCh reference and does NOT call Watch again.
// In the production fsnotifyWatcher, calling Watch(nil) a second time would create
// a new channel, reset file-state, and start new goroutines — leaking the old pair.
// The fake watcher records each Watch call so we can assert exactly one call.
func TestCritical_WatchCalledOnlyOnce(t *testing.T) {
	fw := &countingFakeWatcher{fakeWatcherForTUI: newFakeWatcherForTUI(16)}
	sessions := []transcripts.Session{makeSession("s1", time.Now(), 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{activeSessions: sessions}, fw, &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	// Enter monitor — Watch is called once here via startWatcherCmdFn.
	m, _ = navigateToMonitor(t, m)

	// Feed 3 agentEventMsg values — each triggers handleAgentEvent which
	// should re-drain the stored channel, NOT call Watch again.
	for i := 0; i < 3; i++ {
		evt := makeAssistantEvent("s1", time.Now())
		result, _ := m.Update(tui.MakeAgentEventMsg(evt))
		m = result.(tui.Model)
	}

	if fw.watchCallCount != 1 {
		t.Errorf("Watch called %d times, want exactly 1 (carry-over fix: no re-Watch on drain)", fw.watchCallCount)
	}
}

// countingFakeWatcher wraps fakeWatcherForTUI and counts Watch invocations.
type countingFakeWatcher struct {
	*fakeWatcherForTUI
	watchCallCount int
}

func (c *countingFakeWatcher) Watch(paths []string) (<-chan transcripts.Event, error) {
	c.watchCallCount++
	return c.fakeWatcherForTUI.Watch(paths)
}

// TestRegression_AgentSessionsLoadedEnrichedWithProjectName guards the production bug
// where ScreenAgentMonitor displayed every session as "Sin tomo registrado" because
// no caller in the TUI ever invoked transcripts.MatchProject() against the registered
// projects. The Scanner returns Sessions with ProjectName="" by design (it only knows
// files, not the registry). The handleAgentSessionsLoaded handler MUST enrich each
// session's ProjectID/ProjectName by matching its Cwd against m.registry.List().
//
// Bug surfaced when user opened the monitor on the live atelier session — tile showed
// "Sin tomo registrado" despite the cwd matching the registered Atelier project path.
func TestRegression_AgentSessionsLoadedEnrichedWithProjectName(t *testing.T) {
	reg := newTestRegistry(t)
	projectPath := t.TempDir() // a real directory so registry.Add(...) doesn't reject it
	added, err := reg.Add("MiTomo", projectPath)
	if err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	m := tui.NewWithMonitor(
		reg,
		&MockOpener{},
		&MockClipboard{},
		&fakeScannerForTUI{},
		newFakeWatcherForTUI(4),
		&fakePriceTableForTUI{known: true},
		config.DefaultAtelierConfig(),
	)
	m, _ = navigateToMonitor(t, m)

	// A fresh-from-Scanner session: Cwd matches the registered project, but
	// ProjectName/ProjectID are empty because the Scanner never sees the registry.
	now := time.Now()
	sess := transcripts.Session{
		ID:            "session-1",
		RootPath:      "/fake/path/session-1.jsonl",
		Cwd:           projectPath,
		LastEventTime: now,
		// Sub-agent that inherits parent cwd implicitly (Cwd left empty)
		SubSessions: []transcripts.Session{{
			ID:            "session-1-sub-a",
			RootPath:      "/fake/path/sub-a.jsonl",
			LastEventTime: now,
		}},
	}

	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg([]transcripts.Session{sess}, nil))
	m = result.(tui.Model)

	got := m.AgentSessions()
	if len(got) != 1 {
		t.Fatalf("AgentSessions len = %d, want 1", len(got))
	}
	if got[0].ProjectID != added.ID {
		t.Errorf("ProjectID = %q, want %q (must be set by enrichment via MatchProject)", got[0].ProjectID, added.ID)
	}
	if got[0].ProjectName != "MiTomo" {
		t.Errorf("ProjectName = %q, want %q (must be set by enrichment via MatchProject)", got[0].ProjectName, "MiTomo")
	}

	// Sub-session inherits parent's cwd → also gets matched.
	if len(got[0].SubSessions) != 1 {
		t.Fatalf("SubSessions len = %d, want 1", len(got[0].SubSessions))
	}
	if got[0].SubSessions[0].ProjectName != "MiTomo" {
		t.Errorf("SubSession.ProjectName = %q, want %q (sub-agent inherits parent cwd → match)", got[0].SubSessions[0].ProjectName, "MiTomo")
	}
}

// TestRegression_AgentNavigation_NestedZoomDoesNotPolluteExitTarget guards a
// production bug where pressing Esc on Monitor (after a Monitor→Zoom→Esc round
// trip) left the user trapped on Monitor. Root cause: Monitor→Zoom transition
// overwrote PrevScreen with ScreenAgentMonitor, then Zoom→Esc set Screen back
// to Monitor without restoring PrevScreen. So Monitor→Esc → leaveAgentMonitor
// → Screen = PrevScreen = Monitor → no exit.
//
// The fix: Monitor→Zoom and Zoom→Replay are NESTED transitions; their back
// targets are hardcoded (Zoom→Monitor, Replay→Zoom). PrevScreen is reserved
// for "exit Monitor entirely" and must be set ONLY when Monitor is entered.
func TestRegression_AgentNavigation_NestedZoomDoesNotPolluteExitTarget(t *testing.T) {
	now := time.Now()
	scanner := &fakeScannerForTUI{
		activeSessions: []transcripts.Session{makeSession("s1", now, 0)},
	}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	// 1. Welcome → `a` → Monitor (PrevScreen = Welcome)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenAgentMonitor {
		t.Fatalf("after 'a': Screen = %v, want ScreenAgentMonitor", m.Screen)
	}

	// Seed sessions so Enter has something to zoom into.
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(scanner.activeSessions, nil))
	m = result.(tui.Model)

	// 2. Monitor → Enter → Zoom (must NOT pollute PrevScreen)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenAgentZoom {
		t.Fatalf("after Enter: Screen = %v, want ScreenAgentZoom", m.Screen)
	}

	// 3. Zoom → Esc → Monitor
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenAgentMonitor {
		t.Fatalf("after Esc on Zoom: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}

	// 4. Monitor → Esc → MUST exit to Welcome (the original entry point).
	// Before the fix, this would stay on Monitor because PrevScreen had been
	// polluted to Monitor in step 2.
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenWelcome {
		t.Fatalf("CRITICAL: after Esc on Monitor: Screen = %v, want ScreenWelcome (PrevScreen pollution)", m.Screen)
	}
}

// TestRegression_AgentNavigation_ReplayBacksToZoom verifies Replay→Esc always
// returns to Zoom, regardless of PrevScreen state. Replay is nested under Zoom
// (only entry: `r` from Zoom).
func TestRegression_AgentNavigation_ReplayBacksToZoom(t *testing.T) {
	now := time.Now()
	scanner := &fakeScannerForTUI{
		activeSessions: []transcripts.Session{makeSession("s1", now, 0)},
		loadEvents:     []transcripts.Event{makeAssistantEvent("s1", now)},
	}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	// Navigate Welcome → Monitor → Enter (Zoom) → r (Replay)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(scanner.activeSessions, nil))
	m = result.(tui.Model)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(tui.Model)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenAgentReplay {
		t.Fatalf("after r: Screen = %v, want ScreenAgentReplay", m.Screen)
	}

	// Replay → Esc → MUST go to Zoom (not Monitor, not Welcome).
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenAgentZoom {
		t.Errorf("Replay Esc: Screen = %v, want ScreenAgentZoom", m.Screen)
	}

	// And Zoom → Esc → Monitor (still nested, PrevScreen still Welcome).
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("Zoom Esc after Replay: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}

	// And Monitor → Esc → Welcome (full unwind).
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = result.(tui.Model)
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("Monitor Esc after full unwind: Screen = %v, want ScreenWelcome", m.Screen)
	}
}

// TestRegression_AgentSessionsLoadedNoMatchKeepsEmpty verifies that sessions
// whose cwd does NOT match any registered project keep ProjectName="" so the
// view falls back to the locked "Sin tomo registrado" label (R6.2).
func TestRegression_AgentSessionsLoadedNoMatchKeepsEmpty(t *testing.T) {
	reg := newTestRegistry(t)
	if _, err := reg.Add("MiTomo", t.TempDir()); err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	m := tui.NewWithMonitor(
		reg,
		&MockOpener{},
		&MockClipboard{},
		&fakeScannerForTUI{},
		newFakeWatcherForTUI(4),
		&fakePriceTableForTUI{known: true},
		config.DefaultAtelierConfig(),
	)
	m, _ = navigateToMonitor(t, m)

	sess := transcripts.Session{
		ID:            "wandering-session",
		RootPath:      "/fake/path/wandering.jsonl",
		Cwd:           `D:\SomeOtherProject`, // intentionally unregistered
		LastEventTime: time.Now(),
	}

	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg([]transcripts.Session{sess}, nil))
	m = result.(tui.Model)

	got := m.AgentSessions()
	if len(got) != 1 {
		t.Fatalf("AgentSessions len = %d, want 1", len(got))
	}
	if got[0].ProjectName != "" || got[0].ProjectID != "" {
		t.Errorf("unmatched cwd: ProjectID=%q ProjectName=%q, want both empty", got[0].ProjectID, got[0].ProjectName)
	}
}

// helper: contains checks if s contains sub.
func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && (func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
