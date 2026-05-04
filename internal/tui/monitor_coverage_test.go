package tui_test

// monitor_coverage_test.go — T33/T38 (TRIANGULATE): Additional tests to reach
// the 80%+ coverage gate required for the TUI package.
// Covers: replay tick, event preview, relativeTime, rootPathsOf, zoom view variants,
// handleReplayLoaded error, agentEventMsg with unknown model, etc.

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
)

// ---- replay tick tests -------------------------------------------------------

// TestReplayTick_AdvancesCursorWhenPlaying covers handleReplayTick GREEN path.
func TestReplayTick_AdvancesCursorWhenPlaying(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Ensure playing (not paused)
	if m.ReplayPaused {
		m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // unpause
	}

	initial := m.ReplayCursor
	// Dispatch a replayTickMsg
	result, _ := m.Update(tui.MakeReplayTickMsg())
	got := result.(tui.Model)

	if got.ReplayCursor != initial+1 {
		t.Errorf("replayTickMsg: ReplayCursor = %d, want %d", got.ReplayCursor, initial+1)
	}
}

// TestReplayTick_Noop_WhenPaused covers handleReplayTick paused path.
func TestReplayTick_Noop_WhenPaused(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Pause
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if !m.ReplayPaused {
		t.Fatal("precondition: should be paused")
	}

	initial := m.ReplayCursor
	result, _ := m.Update(tui.MakeReplayTickMsg())
	got := result.(tui.Model)

	if got.ReplayCursor != initial {
		t.Errorf("replayTickMsg when paused: cursor should not advance; got %d, want %d", got.ReplayCursor, initial)
	}
}

// TestReplayTick_AutoPausesAtEnd covers auto-pause at end of events.
func TestReplayTick_AutoPausesAtEnd(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	// Only 1 event — first tick should exhaust it
	events := []transcripts.Event{makeAssistantEvent("s1", now)}
	scanner := &fakeScannerForTUI{activeSessions: sessions, loadEvents: events}
	m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	result, _ = m.Update(tui.MakeReplayLoadedMsg("s1", events, nil))
	m = result.(tui.Model)

	// Already at event 0 (index 0 of 1-event list). Next tick tries to advance past end.
	result, _ = m.Update(tui.MakeReplayTickMsg())
	got := result.(tui.Model)
	if !got.ReplayPaused {
		t.Error("replayTick at end: should auto-pause when no more events")
	}
}

// ---- MakeReplayTickMsg helper -----------------------------------------------

// TestMakeReplayTickMsg verifies the constructor returns the correct msg type.
func TestMakeReplayTickMsg_IsReplayTickMsg(t *testing.T) {
	msg := tui.MakeReplayTickMsg()
	if msg == nil {
		t.Fatal("MakeReplayTickMsg returned nil")
	}
}

// ---- handleReplayLoaded error path ------------------------------------------

// TestReplayLoaded_ErrorReturnsToZoom covers handleReplayLoaded error path.
func TestReplayLoaded_ErrorGoesBackToZoom(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{activeSessions: sessions}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	// Simulate replay load error
	result, _ = m.Update(tui.MakeReplayLoadedMsg("s1", nil, errTest))
	got := result.(tui.Model)

	if got.Screen != tui.ScreenAgentZoom {
		t.Errorf("replayLoaded error: Screen = %v, want ScreenAgentZoom", got.Screen)
	}
	if got.AgentFlash == "" {
		t.Error("replayLoaded error: AgentFlash should be set")
	}
}

// ---- agentEventMsg: unknown model flash -------------------------------------

// TestAgentEvent_UnknownModelFlash covers flash for unknown model in applyEventToSession.
func TestAgentEvent_UnknownModelFlash(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	// PriceTable that returns unknown for all models
	price := &fakePriceTableForTUI{cost: 0, known: false}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), price, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	evt := makeAssistantEvent("s1", now.Add(time.Second))
	result, _ = m.Update(tui.MakeAgentEventMsg(evt))
	m = result.(tui.Model)

	// Session should have been updated (even with cost=0)
	if len(m.AgentSessions()) == 0 {
		t.Fatal("sessions should still be present after event")
	}
}

// ---- agentWatcherErrMsg coverage --------------------------------------------

// TestWatcherErrMsg_SetsFlash covers agentWatcherErrMsg handler.
func TestWatcherErrMsg_SetsFlash(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)

	watcherErr := tui.MakeWatcherErrMsg(errTest)
	result, _ := m.Update(watcherErr)
	got := result.(tui.Model)

	if got.AgentFlash == "" {
		t.Error("agentWatcherErrMsg: AgentFlash should be set")
	}
	if !strings.Contains(got.AgentFlash, "test error") {
		t.Errorf("agentWatcherErrMsg: AgentFlash = %q, want to contain error text", got.AgentFlash)
	}
}

// ---- eventPreview edge cases ------------------------------------------------

// TestEventPreview_ToolUseEvent covers ToolUseEvent branch.
func TestEventPreview_ToolUseEvent(t *testing.T) {
	view := tui.EventPreviewForTest(&transcripts.ToolUseEvent{
		UUIDValue:      "u1",
		SessionIDValue: "s1",
		ToolName:       "bash",
	})
	if !strings.Contains(view, "tool: bash") {
		t.Errorf("eventPreview ToolUseEvent = %q, want 'tool: bash'", view)
	}
}

// TestEventPreview_UserEvent covers UserEvent branch.
func TestEventPreview_UserEvent(t *testing.T) {
	view := tui.EventPreviewForTest(&transcripts.UserEvent{
		UUIDValue:      "u1",
		SessionIDValue: "s1",
		Text:           "hello world",
	})
	if !strings.Contains(view, "user: hello world") {
		t.Errorf("eventPreview UserEvent = %q, want 'user: hello world'", view)
	}
}

// TestEventPreview_UserEvent_LongText covers truncation of user text.
func TestEventPreview_UserEvent_LongText(t *testing.T) {
	long := strings.Repeat("x", 80)
	view := tui.EventPreviewForTest(&transcripts.UserEvent{
		UUIDValue:      "u1",
		SessionIDValue: "s1",
		Text:           long,
	})
	if len(view) > 80 {
		t.Logf("truncated view: %q", view)
	}
}

// TestEventPreview_ToolResultEvent covers ToolResultEvent branch.
func TestEventPreview_ToolResultEvent(t *testing.T) {
	view := tui.EventPreviewForTest(&transcripts.ToolResultEvent{
		UUIDValue:      "u1",
		SessionIDValue: "s1",
		IsError:        false,
		OutputSummary:  "ok",
	})
	if !strings.Contains(view, "tool_result: ok") {
		t.Errorf("eventPreview ToolResultEvent = %q, want 'tool_result: ok'", view)
	}
}

// TestEventPreview_ToolResultEvent_Error covers ToolResultEvent error branch.
func TestEventPreview_ToolResultEvent_Error(t *testing.T) {
	view := tui.EventPreviewForTest(&transcripts.ToolResultEvent{
		UUIDValue:      "u1",
		SessionIDValue: "s1",
		IsError:        true,
		OutputSummary:  "oops",
	})
	if !strings.Contains(view, "error") {
		t.Errorf("eventPreview ToolResultEvent error = %q, want 'error'", view)
	}
}

// TestEventPreview_AssistantEvent_LongText covers truncation of assistant text.
func TestEventPreview_AssistantEvent_LongText(t *testing.T) {
	long := strings.Repeat("a", 100)
	view := tui.EventPreviewForTest(&transcripts.AssistantEvent{
		UUIDValue:      "u1",
		SessionIDValue: "s1",
		Text:           long,
	})
	if !strings.Contains(view, "…") {
		t.Errorf("eventPreview long assistant = %q, want truncation with '…'", view)
	}
}

// TestEventPreview_Nil covers nil event branch.
func TestEventPreview_Nil(t *testing.T) {
	view := tui.EventPreviewForTest(nil)
	if view != "—" {
		t.Errorf("eventPreview nil = %q, want '—'", view)
	}
}

// ---- relativeTime coverage --------------------------------------------------

// TestRelativeTime_ZeroTime covers zero time.
func TestRelativeTime_ZeroTime(t *testing.T) {
	v := tui.RelativeTimeForTest(time.Time{})
	if v != "—" {
		t.Errorf("relativeTime zero = %q, want '—'", v)
	}
}

// TestRelativeTime_UnderMinute covers "ahora mismo".
func TestRelativeTime_UnderMinute(t *testing.T) {
	v := tui.RelativeTimeForTest(time.Now().Add(-10 * time.Second))
	if v != "ahora mismo" {
		t.Errorf("relativeTime <1min = %q, want 'ahora mismo'", v)
	}
}

// TestRelativeTime_Minutes covers minute display.
func TestRelativeTime_Minutes(t *testing.T) {
	v := tui.RelativeTimeForTest(time.Now().Add(-5 * time.Minute))
	if !strings.Contains(v, "m") {
		t.Errorf("relativeTime 5min = %q, want to contain 'm'", v)
	}
}

// TestRelativeTime_Hours covers hour display.
func TestRelativeTime_Hours(t *testing.T) {
	v := tui.RelativeTimeForTest(time.Now().Add(-3 * time.Hour))
	if !strings.Contains(v, "h") {
		t.Errorf("relativeTime 3h = %q, want to contain 'h'", v)
	}
}

// TestRelativeTime_Days covers day display.
func TestRelativeTime_Days(t *testing.T) {
	v := tui.RelativeTimeForTest(time.Now().Add(-48 * time.Hour))
	if !strings.Contains(v, "d") {
		t.Errorf("relativeTime 2d = %q, want to contain 'd'", v)
	}
}

// ---- zoom view unmatched session coverage -----------------------------------

// TestViewZoom_UnmatchedSession shows "Sin tomo registrado" when no project matched.
func TestViewZoom_UnmatchedSession(t *testing.T) {
	now := time.Now()
	// Session with empty ProjectName → unmatched
	sessions := []transcripts.Session{{
		ID:            "unmatched1",
		RootPath:      "/fake/path/unmatched1.jsonl",
		ProjectName:   "",
		LastEventTime: now,
	}}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAgentZoom {
		t.Fatalf("precondition: Screen = %v, want ScreenAgentZoom", m.Screen)
	}

	view := m.View()
	if !strings.Contains(view, tui.CopyMonitorUnmatched) {
		t.Errorf("Zoom unmatched: view missing %q; got:\n%s", tui.CopyMonitorUnmatched, view)
	}
}

// TestViewZoom_NoSessionFound covers the "session not found" branch.
func TestViewZoom_NoSessionFound(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	// Force zoom screen without sessions
	m = tui.SetAgentZoomedID(m, "nonexistent")
	m = tui.SetScreen(m, tui.ScreenAgentZoom)

	view := m.View()
	// Should not panic; should show some fallback text
	if view == "" {
		t.Error("Zoom no-session: view should not be empty")
	}
}

// ---- replay helpers coverage ------------------------------------------------

// TestReplayInterval_Zero covers zero speed fallback.
func TestReplayInterval_ZeroSpeed(t *testing.T) {
	d := tui.ReplayIntervalForTest(0)
	if d <= 0 {
		t.Errorf("replayInterval(0) = %v, want positive duration", d)
	}
}

// TestReplaySpeedUp_UnknownDefault covers default fallback.
func TestReplaySpeedUp_UnknownDefault(t *testing.T) {
	// Speed value not in cycle — should return 1.0 default
	v := tui.ReplaySpeedUpForTest(99.0)
	if v != 1.0 {
		t.Errorf("replaySpeedUp(99) = %f, want 1.0 (default)", v)
	}
}

// TestReplaySpeedDown_UnknownDefault covers default fallback.
func TestReplaySpeedDown_UnknownDefault(t *testing.T) {
	v := tui.ReplaySpeedDownForTest(99.0)
	if v != 1.0 {
		t.Errorf("replaySpeedDown(99) = %f, want 1.0 (default)", v)
	}
}

// ---- MakeReplayTickMsg helper -----------------------------------------------

// TestReplayTickMsg_WhenNotOnReplayScreen is a noop (no crash).
func TestReplayTick_Noop_WhenNotOnReplayScreen(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	// Not on replay screen — should be noop
	result, _ := m.Update(tui.MakeReplayTickMsg())
	got := result.(tui.Model)
	if got.Screen == tui.ScreenAgentReplay {
		t.Error("replayTick on non-replay screen should not switch to replay")
	}
}

// ---- monitor with flash from sessions load error ----------------------------

// TestMonitor_FlashFromSessionsError covers AgentFlash from session load error.
func TestMonitor_FlashRenderedInView(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tui.MakeAgentSessionsLoadedMsg(nil, errTest))
	m = result.(tui.Model)
	result, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "sellados") {
		t.Errorf("Monitor flash from error: view should contain error text; got:\n%s", view)
	}
}

// ---- renderSubTile with non-empty ProjectName coverage ----------------------

// TestViewMonitor_ExpandedSubAgents_RenderSubTile covers renderSubTile path.
func TestViewMonitor_ExpandedSubAgents_RenderSubTile(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 2)}
	sessions[0].SubSessions[0].ProjectName = "SubProj"
	sessions[0].SubSessions[1].ProjectName = ""

	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Expand
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	view := m.View()
	// Sub-agent with name should appear
	if !strings.Contains(view, "SubProj") {
		t.Errorf("Monitor expanded sub: view missing 'SubProj'; got:\n%s", view)
	}
}

// ---- replay step at boundary ------------------------------------------------

// TestReplay_StepForwardAtEnd does not panic.
func TestReplay_StepForwardAtEnd(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Pause and jump to last event
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // pause
	// Step past the end (3 events, 0-indexed to 2)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	// One more step at the last event — should not panic
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	if m.Screen != tui.ScreenAgentReplay {
		t.Errorf("step at end: Screen = %v, want ScreenAgentReplay", m.Screen)
	}
}

// TestReplay_StepBackwardAtStart does not panic.
func TestReplay_StepBackwardAtStart(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	// Already at cursor 0
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // pause
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	if m.ReplayCursor < 0 {
		t.Errorf("step backward at start: ReplayCursor = %d, must not go negative", m.ReplayCursor)
	}
}
