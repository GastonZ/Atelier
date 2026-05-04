package tui_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/tui"
)

// isCmdQuit checks whether a tea.Cmd produces a tea.QuitMsg.
func isCmdQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	return ok
}

// fixedTime is a deterministic time used in registry helpers.
var fixedTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

// newTestRegistry creates a file registry backed by a temp dir for unit tests.
func newTestRegistry(t *testing.T) registry.Registry {
	t.Helper()
	tmpHome := t.TempDir()
	var i int
	ids := []string{
		"test-id-00000000000000001",
		"test-id-00000000000000002",
		"test-id-00000000000000003",
		"test-id-00000000000000004",
	}
	return registry.NewFileRegistryForTest(
		func() (string, error) { return tmpHome, nil },
		func() time.Time { return fixedTime },
		func() string {
			if i >= len(ids) {
				return "test-id-overflow-00000001"
			}
			id := ids[i]
			i++
			return id
		},
	)
}

// newTestModel creates a Model wired with test doubles.
func newTestModel(t *testing.T) tui.Model {
	t.Helper()
	reg := newTestRegistry(t)
	return tui.New(reg, &MockOpener{}, &MockClipboard{})
}

// newTestModelWithReg creates a Model wired with a specific registry and test doubles.
func newTestModelWithReg(t *testing.T, reg registry.Registry) tui.Model {
	t.Helper()
	return tui.New(reg, &MockOpener{}, &MockClipboard{})
}

// dispatchKey sends a key message to the model and returns the new model.
func dispatchKey(t *testing.T, m tui.Model, msg tea.Msg) (tui.Model, tea.Cmd) {
	t.Helper()
	result, cmd := m.Update(msg)
	got, ok := result.(tui.Model)
	if !ok {
		t.Fatalf("Update() returned %T, want tui.Model", result)
	}
	return got, cmd
}

// TestUpdate covers the existing Update() branches — preserved from bootstrap.
func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		wantWidth   int
		wantHeight  int
		wantQuitCmd bool
	}{
		{
			name:        "WindowSizeMsg sets Width and Height",
			msg:         tea.WindowSizeMsg{Width: 100, Height: 40},
			wantWidth:   100,
			wantHeight:  40,
			wantQuitCmd: false,
		},
		{
			name:        "key q returns tea.Quit on ScreenWelcome",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			wantWidth:   0,
			wantHeight:  0,
			wantQuitCmd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t)
			result, cmd := m.Update(tt.msg)

			got, ok := result.(tui.Model)
			if !ok {
				t.Fatalf("Update() returned %T, want tui.Model", result)
			}

			if got.Width != tt.wantWidth {
				t.Errorf("Width = %d, want %d", got.Width, tt.wantWidth)
			}
			if got.Height != tt.wantHeight {
				t.Errorf("Height = %d, want %d", got.Height, tt.wantHeight)
			}

			isQuit := cmd != nil && isCmdQuit(cmd)
			if isQuit != tt.wantQuitCmd {
				t.Errorf("isQuitCmd = %v, want %v", isQuit, tt.wantQuitCmd)
			}
		})
	}
}

// --- S2.x: ScreenProjects key tests ---

// TestScreenProjects_EnterTransitionsToProjectActions covers S2.1.
func TestScreenProjects_EnterTransitionsToProjectActions(t *testing.T) {
	reg := newTestRegistry(t)
	p, _ := reg.Add("TestProject", t.TempDir())
	m := newTestModelWithReg(t, reg)

	// Navigate to ScreenProjects via Enter on Welcome
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjects {
		t.Fatalf("after Enter on Welcome, Screen = %v, want ScreenProjects", m.Screen)
	}

	// Initialize list with WindowSizeMsg
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Dispatch projectsLoadedMsg to seed the model
	m = tui.DrainProjectsLoaded(t, m)

	// Dispatch Enter to open actions for first project
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjectActions {
		t.Errorf("Enter on Projects: Screen = %v, want ScreenProjectActions", m.Screen)
	}
	if m.SelectedID != p.ID {
		t.Errorf("SelectedID = %q, want %q", m.SelectedID, p.ID)
	}
}

// TestScreenProjects_NTransitionsToAddProject covers S2.2.
func TestScreenProjects_NTransitionsToAddProject(t *testing.T) {
	m := newTestModel(t)
	// Navigate to ScreenProjects
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.Screen != tui.ScreenAddProject {
		t.Errorf("n on Projects: Screen = %v, want ScreenAddProject", m.Screen)
	}
	if m.NameInputValue() != "" {
		t.Errorf("name input = %q, want empty on add screen entry", m.NameInputValue())
	}
	if m.PathInputValue() != "" {
		t.Errorf("path input = %q, want empty on add screen entry", m.PathInputValue())
	}
	if m.AddFocus != 0 {
		t.Errorf("AddFocus = %d, want 0 (name field focused)", m.AddFocus)
	}
}

// TestScreenProjects_DTransitionsToConfirmDelete covers S2.3.
func TestScreenProjects_DTransitionsToConfirmDelete(t *testing.T) {
	reg := newTestRegistry(t)
	p, _ := reg.Add("TestProject", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.Screen != tui.ScreenConfirmDelete {
		t.Errorf("d on Projects: Screen = %v, want ScreenConfirmDelete", m.Screen)
	}
	if m.SelectedID != p.ID {
		t.Errorf("SelectedID = %q, want %q", m.SelectedID, p.ID)
	}
}

// TestScreenProjects_QReturnsToWelcomeNotQuit covers S2.4 / S6.2.
func TestScreenProjects_QReturnsToWelcomeNotQuit(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjects {
		t.Fatalf("precondition: Screen = %v, want ScreenProjects", m.Screen)
	}

	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("q on Projects: Screen = %v, want ScreenWelcome", m.Screen)
	}
	if cmd != nil {
		t.Errorf("q on Projects: cmd should be nil (not tea.Quit), got non-nil cmd")
	}
}

// TestScreenProjects_EscReturnsToWelcome covers S2.5 / S6.3.
func TestScreenProjects_EscReturnsToWelcome(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("esc on Projects: Screen = %v, want ScreenWelcome", m.Screen)
	}
}

// TestScreenProjects_WindowSizeMsgLazyInit covers S2.8.
func TestScreenProjects_WindowSizeMsgLazyInit(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Before WindowSizeMsg, list should not be initialized
	if m.ListInited {
		t.Error("list should not be initialized before WindowSizeMsg")
	}

	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	// After WindowSizeMsg, list should be initialized (no panic, non-zero dimensions)
	if !m.ListInited {
		t.Error("list should be initialized after WindowSizeMsg while on ScreenProjects")
	}
}

// TestRealFlow_WindowSizeOnWelcomeThenProjectsLoadedInitsList is a regression test
// for the bug where WindowSizeMsg fires once at startup (while on ScreenWelcome),
// then the user navigates into ScreenProjects and projectsLoadedMsg arrives — but
// ListInited stayed false because the original WindowSizeMsg handler only
// initialized the list while Screen == ScreenProjects. With multiple registered
// projects this manifested as: view stuck on "Invocando los tomos...", Enter
// silently picked projects[0] (an invisible item).
//
// The fix: initOrUpdateProjectList runs from BOTH WindowSizeMsg and
// projectsLoadedMsg, gated only by non-zero dimensions. After the projects
// arrive on ScreenProjects, the list is built (or refreshed) so the user sees it.
func TestRealFlow_WindowSizeOnWelcomeThenProjectsLoadedInitsList(t *testing.T) {
	reg := newTestRegistry(t)
	if _, err := reg.Add("Tomo 1", t.TempDir()); err != nil {
		t.Fatalf("seed Add 1: %v", err)
	}
	if _, err := reg.Add("Tomo 2", t.TempDir()); err != nil {
		t.Fatalf("seed Add 2: %v", err)
	}

	m := newTestModelWithReg(t, reg)

	// WindowSizeMsg fires at startup while on ScreenWelcome (the real Bubble Tea flow).
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.Screen != tui.ScreenWelcome {
		t.Fatalf("precondition: Screen = %v, want ScreenWelcome", m.Screen)
	}

	// User presses Enter on Welcome → Screen = ScreenProjects, loadProjectsCmd fires.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjects {
		t.Fatalf("after Enter: Screen = %v, want ScreenProjects", m.Screen)
	}

	// Drain the loadProjectsCmd to deliver projectsLoadedMsg synchronously.
	m = tui.DrainProjectsLoaded(t, m)

	// Regression assertion: list MUST be initialized after the projects arrive,
	// even though WindowSizeMsg fired earlier on ScreenWelcome.
	if !m.ListInited {
		t.Fatal("ListInited should be true after WindowSizeMsg(Welcome) + projectsLoadedMsg(Projects)")
	}

	// View must NOT show the loading placeholder once the list is initialized.
	view := m.View()
	if strings.Contains(view, "Invocando los tomos") {
		t.Errorf("view should not contain loading placeholder once list is built; got:\n%s", view)
	}
}

// --- S3.x: ScreenAddProject tests ---

// TestScreenAddProject_TabMovesFocusToPath covers S3.1.
func TestScreenAddProject_TabMovesFocusToPath(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if m.AddFocus != 0 {
		t.Fatalf("precondition: AddFocus = %d, want 0", m.AddFocus)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.AddFocus != 1 {
		t.Errorf("Tab: AddFocus = %d, want 1", m.AddFocus)
	}
}

// TestScreenAddProject_ShiftTabMovesFocusToName covers S3.2.
func TestScreenAddProject_ShiftTabMovesFocusToName(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyTab}) // go to path
	if m.AddFocus != 1 {
		t.Fatalf("precondition: AddFocus = %d, want 1", m.AddFocus)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.AddFocus != 0 {
		t.Errorf("ShiftTab: AddFocus = %d, want 0", m.AddFocus)
	}
}

// TestScreenAddProject_EnterOnPathWithInvalidPathSetsError covers S3.3.
func TestScreenAddProject_EnterOnPathWithInvalidPathSetsError(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Set a valid name first, then move focus to path
	m = tui.SetNameInput(m, "myproject")
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyTab}) // focus path

	// Set path to a non-existent dir
	m = tui.SetPathInput(m, "/nonexistent/path/xyz/abc123")

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenAddProject {
		t.Errorf("Enter with invalid path: Screen = %v, want ScreenAddProject", m.Screen)
	}
	if m.AddError == "" {
		t.Error("Enter with invalid path: AddError should be non-empty")
	}
	if m.AddFocus != 1 {
		t.Errorf("Enter with invalid path: AddFocus = %d, want 1", m.AddFocus)
	}
}

// TestScreenAddProject_EnterOnPathWithValidPathSavesAndTransitions covers S3.4.
func TestScreenAddProject_EnterOnPathWithValidPathSavesAndTransitions(t *testing.T) {
	reg := newTestRegistry(t)
	m := newTestModelWithReg(t, reg)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	validDir := t.TempDir()
	m = tui.SetNameInput(m, "myproject")
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m = tui.SetPathInput(m, validDir)

	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Process the loadProjectsCmd
	if cmd != nil {
		result, _ := m.Update(cmd())
		m = result.(tui.Model)
	}

	if m.Screen != tui.ScreenProjects {
		t.Errorf("Enter with valid path: Screen = %v, want ScreenProjects", m.Screen)
	}
	if m.AddError != "" {
		t.Errorf("Enter with valid path: AddError = %q, want empty", m.AddError)
	}
}

// TestScreenAddProject_EscCancelsWithoutSaving covers S3.5.
func TestScreenAddProject_EscCancelsWithoutSaving(t *testing.T) {
	reg := newTestRegistry(t)
	m := newTestModelWithReg(t, reg)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	m = tui.SetNameInput(m, "myproject")
	m = tui.SetPathInput(m, "/some/path")

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenProjects {
		t.Errorf("Esc on Add: Screen = %v, want ScreenProjects", m.Screen)
	}
	if m.AddError != "" {
		t.Errorf("Esc on Add: AddError = %q, want empty", m.AddError)
	}

	projects, _ := reg.List()
	if len(projects) != 0 {
		t.Errorf("Esc on Add: len(projects) = %d, want 0 (nothing saved)", len(projects))
	}
}

// TestScreenAddProject_EnterOnNameFieldMovesFocus covers S3.6.
func TestScreenAddProject_EnterOnNameFieldMovesFocus(t *testing.T) {
	reg := newTestRegistry(t)
	m := newTestModelWithReg(t, reg)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.AddFocus != 0 {
		t.Fatalf("precondition: AddFocus = %d, want 0", m.AddFocus)
	}

	m = tui.SetNameInput(m, "myproject")
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.AddFocus != 1 {
		t.Errorf("Enter on name: AddFocus = %d, want 1", m.AddFocus)
	}
	if m.Screen != tui.ScreenAddProject {
		t.Errorf("Enter on name: Screen = %v, want ScreenAddProject", m.Screen)
	}
	projects, _ := reg.List()
	if len(projects) != 0 {
		t.Errorf("Enter on name: len(projects) = %d, want 0 (should not save)", len(projects))
	}
}

// --- S4.x: ScreenProjectActions tests ---

// TestScreenProjectActions_EscReturnsToProjects covers S4.8.
func TestScreenProjectActions_EscReturnsToProjects(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("TestProject", t.TempDir())
	mockOpener := &MockOpener{}
	m := tui.New(reg, mockOpener, &MockClipboard{})

	// Navigate to ScreenProjects
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)

	// Enter project actions
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjectActions {
		t.Fatalf("precondition: Screen = %v, want ScreenProjectActions", m.Screen)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenProjects {
		t.Errorf("Esc on Actions: Screen = %v, want ScreenProjects", m.Screen)
	}
	if len(mockOpener.OpenInClaudeCodeCalls)+len(mockOpener.SpawnPowerShellCalls) != 0 {
		t.Error("Esc on Actions: no opener calls expected")
	}
}

// TestScreenProjectActions_JKMovesActionCursor covers action cursor navigation.
func TestScreenProjectActions_JKMovesActionCursor(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("TestProject", t.TempDir())
	m := tui.New(reg, &MockOpener{}, &MockClipboard{})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjectActions {
		t.Fatalf("precondition: Screen = %v, want ScreenProjectActions", m.Screen)
	}
	if m.ActionCursor != 0 {
		t.Fatalf("precondition: ActionCursor = %d, want 0", m.ActionCursor)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ActionCursor != 1 {
		t.Errorf("j: ActionCursor = %d, want 1", m.ActionCursor)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.ActionCursor != 0 {
		t.Errorf("k: ActionCursor = %d, want 0", m.ActionCursor)
	}
}

// TestScreenProjectActions_EnterExecutesOpenInClaude covers S4.5.
func TestScreenProjectActions_EnterExecutesOpenInClaude(t *testing.T) {
	reg := newTestRegistry(t)
	p, _ := reg.Add("TestProject", t.TempDir())
	mockOpener := &MockOpener{}
	m := tui.New(reg, mockOpener, &MockClipboard{})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.ActionCursor != 0 {
		t.Fatalf("precondition: ActionCursor = %d, want 0 (OpenInClaude)", m.ActionCursor)
	}

	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	// Execute the returned cmd (the action cmd)
	if cmd != nil {
		result, _ := m.Update(cmd())
		m = result.(tui.Model)
	}

	if len(mockOpener.OpenInClaudeCodeCalls) == 0 {
		t.Fatal("OpenInClaudeCode was not called")
	}
	if mockOpener.OpenInClaudeCodeCalls[0] != p.Path {
		t.Errorf("OpenInClaudeCode path = %q, want %q", mockOpener.OpenInClaudeCodeCalls[0], p.Path)
	}
	if m.Screen != tui.ScreenProjects {
		t.Errorf("after OpenInClaude: Screen = %v, want ScreenProjects", m.Screen)
	}
}

// TestScreenProjectActions_EnterExecutesPowerShell covers S4.6.
func TestScreenProjectActions_EnterExecutesPowerShell(t *testing.T) {
	reg := newTestRegistry(t)
	p, _ := reg.Add("TestProject", t.TempDir())
	mockOpener := &MockOpener{}
	m := tui.New(reg, mockOpener, &MockClipboard{})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Move cursor to PowerShell (index 1)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ActionCursor != 1 {
		t.Fatalf("precondition: ActionCursor = %d, want 1", m.ActionCursor)
	}

	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		result, _ := m.Update(cmd())
		m = result.(tui.Model)
	}

	if len(mockOpener.SpawnPowerShellCalls) == 0 {
		t.Fatal("SpawnPowerShell was not called")
	}
	if mockOpener.SpawnPowerShellCalls[0] != p.Path {
		t.Errorf("SpawnPowerShell path = %q, want %q", mockOpener.SpawnPowerShellCalls[0], p.Path)
	}
	if m.Screen != tui.ScreenProjects {
		t.Errorf("after PowerShell: Screen = %v, want ScreenProjects", m.Screen)
	}
}

// TestScreenProjectActions_EnterExecutesCopyPath covers S4.7.
func TestScreenProjectActions_EnterExecutesCopyPath(t *testing.T) {
	reg := newTestRegistry(t)
	p, _ := reg.Add("TestProject", t.TempDir())
	mockClipboard := &MockClipboard{}
	m := tui.New(reg, &MockOpener{}, mockClipboard)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Move cursor to Copy Path (index 2)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.ActionCursor != 2 {
		t.Fatalf("precondition: ActionCursor = %d, want 2", m.ActionCursor)
	}

	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		result, _ := m.Update(cmd())
		m = result.(tui.Model)
	}

	if len(mockClipboard.Writes) == 0 {
		t.Fatal("clipboard.WriteAll was not called")
	}
	if mockClipboard.Writes[0] != p.Path {
		t.Errorf("clipboard wrote %q, want %q", mockClipboard.Writes[0], p.Path)
	}
	if m.ActionFlash == "" {
		t.Error("ActionFlash should be set after copy path")
	}
	if m.Screen != tui.ScreenProjects {
		t.Errorf("after CopyPath: Screen = %v, want ScreenProjects", m.Screen)
	}
}

// --- S5.x: ScreenConfirmDelete tests ---

// TestScreenConfirmDelete_YDeletesAndTransitions covers S5.1.
func TestScreenConfirmDelete_YDeletesAndTransitions(t *testing.T) {
	reg := newTestRegistry(t)
	p1, _ := reg.Add("P1", t.TempDir())
	_, _ = reg.Add("P2", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.Screen != tui.ScreenConfirmDelete {
		t.Fatalf("precondition: Screen = %v, want ScreenConfirmDelete", m.Screen)
	}
	if m.SelectedID != p1.ID {
		t.Fatalf("precondition: SelectedID = %q, want %q", m.SelectedID, p1.ID)
	}

	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd != nil {
		result, _ := m.Update(cmd())
		m = result.(tui.Model)
	}

	if m.Screen != tui.ScreenProjects {
		t.Errorf("y on Confirm: Screen = %v, want ScreenProjects", m.Screen)
	}
	if m.SelectedID != "" {
		t.Errorf("y on Confirm: SelectedID = %q, want empty", m.SelectedID)
	}

	projects, _ := reg.List()
	if len(projects) != 1 {
		t.Errorf("after delete: len(projects) = %d, want 1", len(projects))
	}
	if projects[0].Name != "P2" {
		t.Errorf("after delete: remaining project = %q, want P2", projects[0].Name)
	}
}

// TestScreenConfirmDelete_NCancels covers S5.2.
func TestScreenConfirmDelete_NCancels(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("P1", t.TempDir())
	_, _ = reg.Add("P2", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.Screen != tui.ScreenProjects {
		t.Errorf("n on Confirm: Screen = %v, want ScreenProjects", m.Screen)
	}

	projects, _ := reg.List()
	if len(projects) != 2 {
		t.Errorf("n on Confirm: len(projects) = %d, want 2 (nothing deleted)", len(projects))
	}
}

// TestScreenConfirmDelete_EscCancels covers S5.3.
func TestScreenConfirmDelete_EscCancels(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("P1", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenProjects {
		t.Errorf("esc on Confirm: Screen = %v, want ScreenProjects", m.Screen)
	}

	projects, _ := reg.List()
	if len(projects) != 1 {
		t.Errorf("esc on Confirm: len(projects) = %d, want 1 (nothing deleted)", len(projects))
	}
}

// --- S6.x: Quit key handling across screens ---

// TestQuitHandling_QOnWelcomeQuits covers S6.1.
func TestQuitHandling_QOnWelcomeQuits(t *testing.T) {
	m := newTestModel(t)
	_, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if !isCmdQuit(cmd) {
		t.Error("q on Welcome: should return tea.Quit cmd")
	}
}

// TestQuitHandling_CtrlCAlwaysQuits covers S6.7.
func TestQuitHandling_CtrlCAlwaysQuits(t *testing.T) {
	screens := []struct {
		name     string
		setup    func(tui.Model) tui.Model
	}{
		{"Welcome", func(m tui.Model) tui.Model { return m }},
		{"Projects", func(m tui.Model) tui.Model {
			m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
			return m
		}},
	}

	for _, s := range screens {
		t.Run(s.name, func(t *testing.T) {
			m := newTestModel(t)
			m = s.setup(m)
			_, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
			if !isCmdQuit(cmd) {
				t.Errorf("ctrl+c on %s: should return tea.Quit cmd", s.name)
			}
		})
	}
}

// TestQuitHandling_EscOnWelcomeIsNoop covers S6.8.
func TestQuitHandling_EscOnWelcomeIsNoop(t *testing.T) {
	m := newTestModel(t)
	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("esc on Welcome: Screen = %v, want ScreenWelcome", m.Screen)
	}
	if cmd != nil {
		t.Error("esc on Welcome: cmd should be nil (no-op)")
	}
}

// TestQuitHandling_EscOnAddProjectGoesToProjects covers S6.4.
func TestQuitHandling_EscOnAddProjectGoesToProjects(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.Screen != tui.ScreenAddProject {
		t.Fatalf("precondition: Screen = %v, want ScreenAddProject", m.Screen)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenProjects {
		t.Errorf("esc on Add: Screen = %v, want ScreenProjects", m.Screen)
	}
}

// TestQuitHandling_EscOnConfirmDeleteGoesToProjects covers S6.6.
func TestQuitHandling_EscOnConfirmDeleteGoesToProjects(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("P1", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.Screen != tui.ScreenConfirmDelete {
		t.Fatalf("precondition: Screen = %v, want ScreenConfirmDelete", m.Screen)
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != tui.ScreenProjects {
		t.Errorf("esc on ConfirmDelete: Screen = %v, want ScreenProjects", m.Screen)
	}
}

// --- T22: CRITICAL REGRESSION TEST for bubbles/list quit-hijack ---

// TestListQuitHijackRegression_QOnProjectsDoesNotQuit is the permanent regression guard.
// It explicitly tests that the bubbles/list KeyMap.Quit override holds:
// dispatching "q" while on ScreenProjects with a non-zero initialized list
// returns nil cmd (NOT tea.Quit) and navigates to ScreenWelcome.
func TestListQuitHijackRegression_QOnProjectsDoesNotQuit(t *testing.T) {
	m := newTestModel(t)

	// Navigate to ScreenProjects
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjects {
		t.Fatalf("precondition: Screen = %v, want ScreenProjects", m.Screen)
	}

	// Initialize the list (non-zero dimensions)
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m.ListInited {
		t.Fatalf("precondition: list not initialized after WindowSizeMsg")
	}

	// Dispatch q — the list's default quit binding should be NEUTERED
	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	// CRITICAL assertions:
	if cmd != nil {
		// Verify cmd is not tea.Quit
		if isCmdQuit(cmd) {
			t.Error("REGRESSION: q on ScreenProjects returned tea.Quit — list KeyMap.Quit override is broken!")
		}
	}
	if m.Screen != tui.ScreenWelcome {
		t.Errorf("q on ScreenProjects with initialized list: Screen = %v, want ScreenWelcome", m.Screen)
	}
}
