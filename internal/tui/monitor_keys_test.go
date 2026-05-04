package tui_test

// monitor_keys_test.go — T29 (RED): Key handler tests for agent monitor screens.
// Covers §5 keymap tables from the design.

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
)

// ---- S7.1: Navigation entry keys --------------------------------------------

// TestWelcome_AKeyNavigatesToMonitor covers R7.1.
func TestWelcome_AKeyNavigatesToMonitor(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("a on Welcome: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}
	if m.PrevScreen != tui.ScreenWelcome {
		t.Errorf("a on Welcome: PrevScreen = %v, want ScreenWelcome", m.PrevScreen)
	}
}

// TestProjects_MKeyNavigatesToMonitor covers R7.2.
func TestProjects_MKeyNavigatesToMonitor(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	// Navigate to Projects first
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjects {
		t.Fatalf("Enter on Welcome: Screen = %v, want ScreenProjects", m.Screen)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("m on Projects: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}
	if m.PrevScreen != tui.ScreenProjects {
		t.Errorf("m on Projects: PrevScreen = %v, want ScreenProjects", m.PrevScreen)
	}
}

// ---- ScreenAgentMonitor: j/k navigation -------------------------------------

// TestMonitor_JKMovesCursor covers R7.3 (j/k).
func TestMonitor_JKMovesCursor(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{
		makeSession("s1", now.Add(-1*time.Minute), 0),
		makeSession("s2", now, 0),
		makeSession("s3", now.Add(-2*time.Minute), 0),
	}
	scanner := &fakeScannerForTUI{activeSessions: sessions}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	m, _ = navigateToMonitor(t, m)
	// Seed sessions
	msg := tui.MakeAgentSessionsLoadedMsg(sessions, nil)
	result, _ := m.Update(msg)
	m = result.(tui.Model)

	if m.AgentTileCursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.AgentTileCursor)
	}

	// j moves down
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.AgentTileCursor != 1 {
		t.Errorf("j: AgentTileCursor = %d, want 1", m.AgentTileCursor)
	}

	// k moves up
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.AgentTileCursor != 0 {
		t.Errorf("k: AgentTileCursor = %d, want 0", m.AgentTileCursor)
	}
}

// TestMonitor_JDoesNotGoBelow covers cursor clamping at bottom.
func TestMonitor_JDoesNotGoBelow(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0), makeSession("s2", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Move to last item
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	// Try to go past end
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.AgentTileCursor != 1 {
		t.Errorf("j at bottom: AgentTileCursor = %d, want 1 (clamped)", m.AgentTileCursor)
	}
}

// TestMonitor_KDoesNotGoAbove covers cursor clamping at top.
func TestMonitor_KDoesNotGoAbove(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Try to go above first item
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.AgentTileCursor != 0 {
		t.Errorf("k at top: AgentTileCursor = %d, want 0 (clamped)", m.AgentTileCursor)
	}
}

// ---- ScreenAgentMonitor: number key jump ------------------------------------

// TestMonitor_NumberKeyJumpsToTile covers S7.2 (R7.3 1-9 keys).
func TestMonitor_NumberKeyJumpsToTile(t *testing.T) {
	now := time.Now()
	sessions := make([]transcripts.Session, 4)
	for i := range sessions {
		sessions[i] = makeSession(fmt.Sprintf("s%d", i+1), now.Add(time.Duration(-i)*time.Minute), 0)
	}

	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Press '3' → 0-based index 2
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if m.AgentTileCursor != 2 {
		t.Errorf("'3' key: AgentTileCursor = %d, want 2", m.AgentTileCursor)
	}

	// Press '1' → 0-based index 0
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if m.AgentTileCursor != 0 {
		t.Errorf("'1' key: AgentTileCursor = %d, want 0", m.AgentTileCursor)
	}
}

// TestMonitor_NumberKeyOutOfRange stays at current cursor.
func TestMonitor_NumberKeyOutOfRange(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Press '9' with only 1 session — should be no-op (index 8 > len-1)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	if m.AgentTileCursor != 0 {
		t.Errorf("'9' out of range: AgentTileCursor = %d, want 0", m.AgentTileCursor)
	}
}

// ---- ScreenAgentMonitor: o/c sub-agent expand/collapse ----------------------

// TestMonitor_OExpandsSubAgents covers S3.4, R7.5.
func TestMonitor_OExpandsSubAgents(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 3)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Default: collapsed
	if m.AgentExpanded("s1") {
		t.Fatal("precondition: s1 should start collapsed")
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if !m.AgentExpanded("s1") {
		t.Error("o: s1 sub-agents should now be expanded")
	}
}

// TestMonitor_CCollapsesSubAgents covers R7.5 collapse.
func TestMonitor_CCollapsesSubAgents(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 2)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Expand first
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if !m.AgentExpanded("s1") {
		t.Fatal("precondition: s1 should be expanded after o")
	}

	// Then collapse
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if m.AgentExpanded("s1") {
		t.Error("c: s1 sub-agents should now be collapsed")
	}
}

// ---- ScreenAgentMonitor: enter zooms ----------------------------------------

// TestMonitor_EnterNavigatesToZoom covers R7.3 (Enter → ScreenAgentZoom).
func TestMonitor_EnterNavigatesToZoom(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAgentZoom {
		t.Errorf("Enter on Monitor: Screen = %v, want ScreenAgentZoom", m.Screen)
	}
	if m.AgentZoomedID != "s1" {
		t.Errorf("Enter on Monitor: AgentZoomedID = %q, want %q", m.AgentZoomedID, "s1")
	}
}

// TestMonitor_EnterNoopWhenEmpty covers Enter on empty monitor.
func TestMonitor_EnterNoopWhenEmpty(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	// No sessions loaded → enter should be no-op
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("Enter on empty Monitor: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}
}

// ---- ScreenAgentMonitor: esc returns to PrevScreen --------------------------

// TestMonitor_EscReturnsToPrevScreen covers R7.3 (Esc).
func TestMonitor_EscReturnsToPrevScreen(t *testing.T) {
	fw := newFakeWatcherForTUI(4)
	m := newMonitorModel(t, &fakeScannerForTUI{}, fw, &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())

	// Come from Welcome
	m, _ = navigateToMonitor(t, m)
	if m.PrevScreen != tui.ScreenWelcome {
		t.Fatalf("precondition PrevScreen = %v, want ScreenWelcome", m.PrevScreen)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("Esc on Monitor: Screen = %v, want ScreenWelcome", m.Screen)
	}
}

// TestMonitor_EscCallsWatcherCancel covers the watcher goroutine cancel on exit (T39).
func TestMonitor_EscCallsWatcherCancel(t *testing.T) {
	fw := newFakeWatcherForTUI(4)
	m := newMonitorModel(t, &fakeScannerForTUI{}, fw, &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)

	// Watcher not yet closed
	if fw.closeCount != 0 {
		t.Fatalf("precondition: closeCount = %d, want 0", fw.closeCount)
	}

	// We need to drain the startWatcherCmd that was returned
	// Just escape — watcher cancel should be called
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// The watcher cancel should have been registered and called
	if !m.WatcherCancelCalled() {
		t.Error("Esc on Monitor: watcher cancel was NOT called (goroutine leak risk)")
	}
}

// ---- ScreenAgentZoom: r enters replay, esc returns to monitor ---------------

// TestZoom_REntersReplay covers R7.4.
func TestZoom_REntersReplay(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	events := []transcripts.Event{makeAssistantEvent("s1", now)}
	scanner := &fakeScannerForTUI{activeSessions: sessions, loadEvents: events}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Enter zoom
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAgentZoom {
		t.Fatalf("precondition: Screen = %v, want ScreenAgentZoom", m.Screen)
	}

	// Press r → enters replay
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.Screen != tui.ScreenAgentReplay {
		t.Errorf("r on Zoom: Screen = %v, want ScreenAgentReplay", m.Screen)
	}
}

// TestZoom_EscReturnsToMonitor covers R7.4 (esc from zoom).
func TestZoom_EscReturnsToMonitor(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{activeSessions: sessions}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAgentZoom {
		t.Fatalf("precondition: Screen = %v, want ScreenAgentZoom", m.Screen)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenAgentMonitor {
		t.Errorf("Esc on Zoom: Screen = %v, want ScreenAgentMonitor", m.Screen)
	}
}

// ---- ScreenAgentReplay: key handlers ----------------------------------------

// TestReplay_SpaceTogglesPause covers R5.3.
func TestReplay_SpaceTogglesPause(t *testing.T) {
	m := buildModelOnReplayScreen(t)

	// Initially playing (not paused)
	if m.ReplayPaused {
		t.Fatal("precondition: replay should start unpaused")
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if !m.ReplayPaused {
		t.Error("space: replay should be paused after first press")
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if m.ReplayPaused {
		t.Error("space: replay should be unpaused after second press")
	}
}

// TestReplay_PlusIncreasesSpeed covers R5.4.
func TestReplay_PlusIncreasesSpeed(t *testing.T) {
	m := buildModelOnReplayScreen(t)

	// Default speed 1x
	if m.ReplaySpeed != 1.0 {
		t.Fatalf("precondition: ReplaySpeed = %f, want 1.0", m.ReplaySpeed)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if m.ReplaySpeed != 2.0 {
		t.Errorf("+: ReplaySpeed = %f, want 2.0", m.ReplaySpeed)
	}
}

// TestReplay_MinusDecreasesSpeed covers R5.4 (backward).
func TestReplay_MinusDecreasesSpeed(t *testing.T) {
	m := buildModelOnReplayScreen(t)

	// Default speed 1x → - should go to 0.5
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if m.ReplaySpeed != 0.5 {
		t.Errorf("-: ReplaySpeed = %f, want 0.5", m.ReplaySpeed)
	}
}

// TestReplay_SpeedCyclesForwardAndWraps covers 0.5→1→2→4→0.5 cycle (R5.4).
func TestReplay_SpeedCyclesForwardAndWraps(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Start at 1x; expected cycle: 1 → 2 → 4 → 0.5 → 1
	expected := []float64{2.0, 4.0, 0.5, 1.0}
	for i, want := range expected {
		m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
		if m.ReplaySpeed != want {
			t.Errorf("step %d +: ReplaySpeed = %f, want %f", i, m.ReplaySpeed, want)
		}
	}
}

// TestReplay_StepForward covers R5.3 (>).
func TestReplay_StepForward(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Pause first to prevent auto-advance
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})

	initial := m.ReplayCursor
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	if m.ReplayCursor != initial+1 {
		t.Errorf(">: ReplayCursor = %d, want %d", m.ReplayCursor, initial+1)
	}
}

// TestReplay_StepBackward covers R5.5 / S5.5.
func TestReplay_StepBackward(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Pause and advance to event 1
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	if m.ReplayCursor != 1 {
		t.Fatalf("precondition: ReplayCursor = %d, want 1", m.ReplayCursor)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	if m.ReplayCursor != 0 {
		t.Errorf("<: ReplayCursor = %d, want 0", m.ReplayCursor)
	}
}

// TestReplay_EscReturnsToZoom covers S5.6.
func TestReplay_EscReturnsToZoom(t *testing.T) {
	m := buildModelOnReplayScreen(t)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenAgentZoom {
		t.Errorf("Esc on Replay: Screen = %v, want ScreenAgentZoom", m.Screen)
	}
}

// buildModelOnReplayScreen constructs a Model already in ScreenAgentReplay
// with 3 events loaded, paused.
func buildModelOnReplayScreen(t *testing.T) tui.Model {
	t.Helper()
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	events := []transcripts.Event{
		makeAssistantEvent("s1", now),
		makeAssistantEvent("s1", now.Add(time.Second)),
		makeAssistantEvent("s1", now.Add(2*time.Second)),
	}
	scanner := &fakeScannerForTUI{activeSessions: sessions, loadEvents: events}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Zoom
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	// Enter replay: press r
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	// Drain the replayLoadedMsg
	msg := tui.MakeReplayLoadedMsg("s1", events, nil)
	result, _ = m.Update(msg)
	m = result.(tui.Model)

	if m.Screen != tui.ScreenAgentReplay {
		t.Fatalf("buildModelOnReplayScreen: Screen = %v, want ScreenAgentReplay", m.Screen)
	}
	return m
}
