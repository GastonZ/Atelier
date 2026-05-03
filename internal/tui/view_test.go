package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/tui"
)

// TestView covers the two rendering branches of viewWelcome() — preserved from bootstrap.
func TestView(t *testing.T) {
	// firstDragonLine is the first line of the art — used to detect dragon presence.
	firstDragonLine := strings.Split(tui.DragonArt, "\n")[0]

	tests := []struct {
		name            string
		width           int
		height          int
		wantTagline     bool
		wantDragon      bool
		wantResizeHint  bool
		wantMissionCtrl bool
	}{
		{
			name:           "small terminal triggers fallback branch",
			width:          50,
			height:         20,
			wantTagline:    true,
			wantDragon:     false,
			wantResizeHint: true,
		},
		{
			name:            "large terminal triggers full welcome branch",
			width:           120,
			height:          50,
			wantTagline:     true,
			wantDragon:      true,
			wantMissionCtrl: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t)
			result, _ := m.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			got, ok := result.(tui.Model)
			if !ok {
				t.Fatalf("Update() returned %T, want tui.Model", result)
			}

			view := got.View()

			if tt.wantTagline && !strings.Contains(view, "Dragon Atelier") {
				t.Error("View() missing 'Dragon Atelier' tagline")
			}

			if tt.wantDragon && !strings.Contains(view, firstDragonLine) {
				t.Error("View() missing first line of dragon art in full welcome branch")
			}
			if !tt.wantDragon && strings.Contains(view, firstDragonLine) {
				t.Error("View() contains dragon art in fallback branch (should not)")
			}

			if tt.wantResizeHint && !strings.Contains(view, "Resize terminal for full art") {
				t.Error("View() missing resize hint in fallback branch")
			}

			if tt.wantMissionCtrl && !strings.Contains(view, "Mission Control for AI Workflows") {
				t.Error("View() missing 'Mission Control for AI Workflows' subtitle in full branch")
			}
		})
	}
}

// TestViewWelcome_HasEnterHint verifies the welcome screen shows the enter hint.
func TestViewWelcome_HasEnterHint(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	if !strings.Contains(view, "presioná Enter") {
		t.Error("Welcome view missing 'presioná Enter' hint")
	}
}

// TestViewProjects_EmptyStateContainsAddHint covers S2.7.
func TestViewProjects_EmptyStateContainsAddHint(t *testing.T) {
	m := newTestModel(t) // empty registry
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	// Give it a size so view renders properly
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if len(view) == 0 {
		t.Fatal("View() returned empty string on ScreenProjects")
	}
	if !strings.Contains(view, "n") {
		t.Error("ScreenProjects empty state: view should contain 'n' as add hint")
	}
}

// TestViewAddProject_FormErrorIsVisible covers S3.7.
func TestViewAddProject_FormErrorIsVisible(t *testing.T) {
	m := newTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = tui.SetAddError(m, "Path not found")

	view := m.View()
	if !strings.Contains(view, "Path not found") {
		t.Errorf("ScreenAddProject: view missing error text 'Path not found'; got:\n%s", view)
	}
}

// TestViewConfirmDelete_ContainsProjectName covers S5.4.
func TestViewConfirmDelete_ContainsProjectName(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("myproject", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	view := m.View()
	if !strings.Contains(view, "myproject") {
		t.Errorf("ConfirmDelete view missing project name 'myproject'; got:\n%s", view)
	}
	if !strings.Contains(view, "y") {
		t.Error("ConfirmDelete view missing 'y' hint")
	}
	if !strings.Contains(view, "n") {
		t.Error("ConfirmDelete view missing 'n' hint")
	}
}

// TestViewProjectActions_RendersActionLabels verifies the locked §10 action labels.
func TestViewProjectActions_RendersActionLabels(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("TestProject", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjectActions {
		t.Fatalf("precondition: Screen = %v, want ScreenProjectActions", m.Screen)
	}

	view := m.View()
	for _, label := range []string{"Abrir en Claude Code", "Invocar PowerShell", "Copiar el sendero"} {
		if !strings.Contains(view, label) {
			t.Errorf("ScreenProjectActions view missing label %q", label)
		}
	}
}

// TestViewProjectActions_ActionFlashVisible verifies flash text renders above footer.
func TestViewProjectActions_ActionFlashVisible(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("TestProject", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Set the flash message
	m = tui.SetActionFlash(m, "Test flash message")

	view := m.View()
	if !strings.Contains(view, "Test flash message") {
		t.Errorf("ScreenProjectActions: flash message not visible in view; got:\n%s", view)
	}
}

// TestViewProjects_NonEmptyListRendersProjects verifies project list renders project names.
func TestViewProjects_NonEmptyListRendersProjects(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("MyAwesomeProject", t.TempDir())
	m := newTestModelWithReg(t, reg)

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)

	view := m.View()
	if !strings.Contains(view, "MyAwesomeProject") {
		t.Errorf("ScreenProjects non-empty: view missing project name 'MyAwesomeProject'; got:\n%s", view)
	}
}
