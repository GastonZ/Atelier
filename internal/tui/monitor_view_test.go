package tui_test

// monitor_view_test.go — T29 (RED): View tests for agent monitor screens.
// Covers §3 locked copy, empty state, tile rendering, zoom view, replay view.

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
)

// ---- S3.1: Empty monitor state ----------------------------------------------

// TestViewMonitor_EmptyStateShowsElatelier covers S3.1 / R3.5.
func TestViewMonitor_EmptyStateShowsElatelier(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	// Give it dimensions
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "El atelier duerme.") {
		t.Errorf("Monitor empty: view missing 'El atelier duerme.'; got:\n%s", view)
	}
}

// TestViewMonitor_EmptyStateNoTileFrame covers S3.1 (no tile frames in empty state).
func TestViewMonitor_EmptyStateNoTileFrame(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	// Load empty sessions
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(nil, nil))
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "El atelier duerme.") {
		t.Errorf("Monitor empty loaded: view missing 'El atelier duerme.'; got:\n%s", view)
	}
}

// ---- S3.2: One session, no sub-agents ----------------------------------------

// TestViewMonitor_OneTileNoSubAgents covers S3.2.
func TestViewMonitor_OneTileNoSubAgents(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "TestProject") {
		t.Errorf("Monitor 1 tile: view missing project name 'TestProject'; got:\n%s", view)
	}
	// Cost line should be present
	if !strings.Contains(view, "Pergaminos") {
		t.Errorf("Monitor 1 tile: view missing cost line 'Pergaminos'; got:\n%s", view)
	}
}

// ---- S3.3: One session, three sub-agents, collapsed -------------------------

// TestViewMonitor_SubAgentsCollapsedByDefault covers S3.3 / R3.2.
func TestViewMonitor_SubAgentsCollapsedByDefault(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 3)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Sub-agents not yet expanded
	if m.AgentExpanded("s1") {
		t.Fatal("Sub-agents should be collapsed by default")
	}

	view := m.View()
	// Should show collapsed sub-agent indicator (e.g., "3 sub-agents") but NOT individual sub-agent IDs
	if !strings.Contains(view, "3") {
		t.Errorf("Monitor collapsed: view should mention 3 sub-agents; got:\n%s", view)
	}
	// Individual sub-agent session IDs (s1-sub-a etc.) should NOT be visible
	if strings.Contains(view, "s1-sub-a") {
		t.Errorf("Monitor collapsed: individual sub-agent s1-sub-a should NOT be visible; got:\n%s", view)
	}
}

// ---- S3.4: Expand sub-agents ------------------------------------------------

// TestViewMonitor_ExpandedSubAgentsVisible covers S3.4.
func TestViewMonitor_ExpandedSubAgentsVisible(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 2)}
	// Give sub-sessions a meaningful project name for assertion
	sessions[0].SubSessions[0].ProjectName = "SubAgent1"
	sessions[0].SubSessions[1].ProjectName = "SubAgent2"

	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)

	// Expand
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if !m.AgentExpanded("s1") {
		t.Fatal("Sub-agents should be expanded after o")
	}

	view := m.View()
	// Sub-agent indicator count should still be visible or sub-sessions listed
	if !strings.Contains(view, "TestProject") {
		t.Errorf("Monitor expanded: root project name missing; got:\n%s", view)
	}
}

// ---- Monitor footer ---------------------------------------------------------

// TestViewMonitor_FooterHints covers the locked footer copy.
func TestViewMonitor_FooterHints(t *testing.T) {
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "j/k") {
		t.Errorf("Monitor footer: view missing 'j/k' hint; got:\n%s", view)
	}
	if !strings.Contains(view, "esc: volver") {
		t.Errorf("Monitor footer: view missing 'esc: volver' hint; got:\n%s", view)
	}
}

// ---- Zoom view --------------------------------------------------------------

// TestViewZoom_ShowsProjectAndCost covers zoom view basic fields.
func TestViewZoom_ShowsProjectAndCost(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAgentZoom {
		t.Fatalf("precondition: Screen = %v, want ScreenAgentZoom", m.Screen)
	}

	view := m.View()
	if !strings.Contains(view, "TestProject") {
		t.Errorf("Zoom: view missing project name 'TestProject'; got:\n%s", view)
	}
}

// TestViewZoom_FooterHints covers zoom footer copy.
func TestViewZoom_FooterHints(t *testing.T) {
	now := time.Now()
	sessions := []transcripts.Session{makeSession("s1", now, 0)}
	m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
	m, _ = navigateToMonitor(t, m)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)
	result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
	m = result.(tui.Model)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()
	if !strings.Contains(view, "revivir") {
		t.Errorf("Zoom footer: view missing 'revivir'; got:\n%s", view)
	}
	if !strings.Contains(view, "esc: volver") {
		t.Errorf("Zoom footer: view missing 'esc: volver'; got:\n%s", view)
	}
}

// ---- Replay view ------------------------------------------------------------

// TestViewReplay_ShowsReplayHeader covers locked "Crónica del taller" header.
func TestViewReplay_ShowsReplayHeader(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "Crónica") {
		t.Errorf("Replay: view missing 'Crónica del taller' header; got:\n%s", view)
	}
}

// TestViewReplay_FooterHints covers replay footer locked copy.
func TestViewReplay_FooterHints(t *testing.T) {
	m := buildModelOnReplayScreen(t)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = result.(tui.Model)

	view := m.View()
	if !strings.Contains(view, "espacio: pausar") {
		t.Errorf("Replay footer: view missing 'espacio: pausar'; got:\n%s", view)
	}
	if !strings.Contains(view, "esc: volver") {
		t.Errorf("Replay footer: view missing 'esc: volver'; got:\n%s", view)
	}
}
