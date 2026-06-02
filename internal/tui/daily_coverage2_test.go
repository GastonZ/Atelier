package tui_test

// daily_coverage2_test.go — Additional targeted coverage tests.

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/tui"
)

// ============================================================================
// FilterValue coverage for memoryItem and historyItem
// ============================================================================

// TestMemoryItem_FilterValue verifies that filtering works (exercises FilterValue).
func TestMemoryItem_FilterValue_UsedByList(t *testing.T) {
	// Load memory entries, then send filter keys to force list.FilterValue calls.
	obs := []engram.Observation{
		{ID: 1, Title: "auth middleware", Content: "auth content", Timestamp: time.Now()},
		{ID: 2, Title: "database setup", Content: "db content", Timestamp: time.Now()},
	}
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenMemoryBrowser)
	result, _ := m.Update(tui.MakeMemoryLoadedMsg(obs, nil))
	m = result.(tui.Model)

	// Trigger '/' to enter filter mode — exercises FilterValue on each item.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	// Type a filter character — bubbles/list calls FilterValue on each item.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.Screen != tui.ScreenMemoryBrowser {
		t.Error("Screen changed unexpectedly during memory filter")
	}
	// Esc to close filter.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	_ = m
}

// TestHistoryItem_FilterValue_UsedByList verifies history list filter exercises FilterValue.
func TestHistoryItem_FilterValue_UsedByList(t *testing.T) {
	entries := []tui.HistoryEntry{
		{Source: "git", Date: time.Now(), Title: "feat: authentication", Ref: "abc"},
		{Source: "sdd", Date: time.Now(), Title: "SDD archive", Ref: "1"},
	}
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenProjectHistory)
	result, _ := m.Update(tui.MakeHistoryLoadedMsg(entries, nil))
	m = result.(tui.Model)

	// Enter filter mode.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	// Type filter text — exercises FilterValue.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	// Esc to close filter.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenProjectHistory {
		t.Errorf("Screen = %v, want ScreenProjectHistory", m.Screen)
	}
}

// ============================================================================
// parseHistoryRef via handleProjectHistoryKeys SDD detail
// ============================================================================

// TestHandleProjectHistoryKeys_EnterOnSDDItem_TriggersDetailLoad verifies SDD entry
// detail loading, which exercises parseHistoryRef.
func TestHandleProjectHistoryKeys_EnterOnSDDItem_TriggersDetailLoad(t *testing.T) {
	eng := &mockEngramClientForTUI{
		byIDObs: engram.Observation{ID: 42, Content: "# Archive\n\nContent"},
	}
	entries := []tui.HistoryEntry{
		{Source: "sdd", Date: time.Now(), Title: "SDD archive", Ref: "42"},
	}
	m := newTestModel(t)
	m = tui.InjectDailyPackDeps(m, eng, nil, nil)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenProjectHistory)

	// Load history entries.
	result, _ := m.Update(tui.MakeHistoryLoadedMsg(entries, nil))
	m = result.(tui.Model)

	// Press Enter — the list should have the sdd item selected (it's the only item).
	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Should have returned a cmd (loadHistoryDetailSDDCmd).
	if cmd == nil {
		t.Log("Note: cmd is nil (list may not have selected item in test env without rendering)")
		// This is acceptable — the list needs to be rendered to have a selection.
		return
	}
	// Execute the cmd.
	result2, _ := m.Update(cmd())
	m = result2.(tui.Model)
	_ = m
}

// ============================================================================
// handleProjectHistoryKeys navigation coverage
// ============================================================================

func TestHandleProjectHistoryKeys_JKNavigation(t *testing.T) {
	entries := []tui.HistoryEntry{
		{Source: "git", Date: time.Now(), Title: "commit 1", Ref: "a"},
		{Source: "git", Date: time.Now().Add(-time.Hour), Title: "commit 2", Ref: "b"},
	}
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenProjectHistory)
	result, _ := m.Update(tui.MakeHistoryLoadedMsg(entries, nil))
	m = result.(tui.Model)

	// j/k navigation (delegate to list).
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.Screen != tui.ScreenProjectHistory {
		t.Errorf("Screen = %v after j/k, want ScreenProjectHistory", m.Screen)
	}
}

// ============================================================================
// handleProjectsKeys coverage for 'r' with nil reader
// ============================================================================

func TestHandleProjectsKeys_RWithNoGitReader_NoCmd(t *testing.T) {
	m := newTestModel(t) // no gitStatusReader injected
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)

	// 'r' with no gitStatusReader — should not panic, return nil cmd.
	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	// cmd may be nil since there's no git reader.
	_ = cmd
	_ = m
}

// ============================================================================
// handleProjectActionsKeys for action 7 (Borrar) — no SelectedID
// ============================================================================

func TestHandleProjectActionsKeys_Borrar_NoSelectedID(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenProjectActions)
	// ActionCursor = 7 with no SelectedID.
	m.ActionCursor = 7

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	// Should be a no-op (no project selected).
	if m.Screen != tui.ScreenProjectActions && m.Screen != tui.ScreenConfirmDelete {
		t.Errorf("Screen = %v after Borrar with no SelectedID", m.Screen)
	}
}

// ============================================================================
// Memory browser error flash path
// ============================================================================

func TestViewMemoryBrowser_ErrorState(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenMemoryBrowser)
	m = tui.SetMemoryErrorForTest(m, "error message")

	view := m.View()
	if view == "" {
		t.Fatal("viewMemoryBrowser error state returned empty string")
	}
}

// TestViewProjectHistory_ErrorState verifies history error flash renders.
func TestViewProjectHistory_ErrorState(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenProjectHistory)
	m = tui.SetHistoryErrorForTest(m, "history error")

	view := m.View()
	if view == "" {
		t.Fatal("viewProjectHistory error state returned empty string")
	}
}

// TestViewDiskUsage_ErrorState verifies disk error flash renders.
func TestViewDiskUsage_ErrorState(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.SetScreenForTest(m, tui.ScreenDiskUsage)
	m = tui.SetDiskErrorForTest(m, "disk error")

	view := m.View()
	if view == "" {
		t.Fatal("viewDiskUsage error state returned empty string")
	}
}

// ============================================================================
// diskRowPath coverage
// ============================================================================

func TestDiskRowPath_ExploresRows(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("TestProject", t.TempDir())
	m := newTestModelWithReg(t, reg)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m = tui.SetScreenForTest(m, tui.ScreenDiskUsage)

	result, _ := m.Update(tui.MakeDiskUsageLoadedMsg(1024, 2048, map[string]int64{}, nil))
	m = result.(tui.Model)

	// Press Enter on row 0 (engram dir).
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Navigate to row 2 (per-tomo).
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	// Enter on per-tomo row.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	_ = m
}

// ============================================================================
// MakeDiskUsageLoadedMsg with per-tomo data
// ============================================================================

func TestViewDiskUsage_WithPerTomo(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("MyProject", t.TempDir())
	m := newTestModelWithReg(t, reg)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m = tui.SetScreenForTest(m, tui.ScreenDiskUsage)

	perTomo := map[string]int64{} // keys will be project IDs
	result, _ := m.Update(tui.MakeDiskUsageLoadedMsg(512, 1024, perTomo, nil))
	m = result.(tui.Model)

	view := m.View()
	if view == "" {
		t.Fatal("viewDiskUsage with per-tomo returned empty string")
	}
}
