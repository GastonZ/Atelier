package tui_test

// memory_link_test.go — engram link picker (ScreenMemoryLink).

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/tui"
)

func TestMemoryLink_PickAndPersist(t *testing.T) {
	reg := newTestRegistry(t)
	proj, _ := reg.Add("Agencia Back", t.TempDir()) // friendly name ≠ engram key
	mockEng := &mockEngramClientForTUI{
		projects: []engram.ProjectStat{
			{Key: "GZBackAgenciaV2", Count: 341},
			{Key: "other", Count: 5},
		},
	}

	m := tui.New(reg, &MockOpener{}, &MockClipboard{})
	m = tui.InjectDailyPackDeps(m, mockEng, nil, nil)

	// Welcome → Projects → select the project → ProjectActions.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjectActions {
		t.Fatalf("precondition: Screen = %v, want ScreenProjectActions", m.Screen)
	}

	// Jump to the "Vincular memoria" entry: launchers (base) + VSCode, PowerShell,
	// Copy, Memory → MemoryLink at base+4.
	base := len(config.DefaultLaunchers())
	for i := 0; i < base+4; i++ {
		m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}

	// Enter → opens picker (loading) and returns the load command.
	m, cmd := dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenMemoryLink {
		t.Fatalf("after Enter on Vincular: Screen = %v, want ScreenMemoryLink", m.Screen)
	}
	if cmd == nil {
		t.Fatal("expected loadEngramProjectsCmd, got nil")
	}
	res, _ := m.Update(cmd())
	m = res.(tui.Model)

	// Cursor on first key (GZBackAgenciaV2) → Enter links it.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != tui.ScreenProjectActions {
		t.Errorf("after link: Screen = %v, want ScreenProjectActions", m.Screen)
	}

	// Persisted to the registry.
	projects, _ := reg.List()
	if len(projects) != 1 || projects[0].EngramProject != "GZBackAgenciaV2" {
		t.Errorf("EngramProject = %q, want GZBackAgenciaV2", projects[0].EngramProject)
	}
	_ = proj
}

func TestMemoryLink_NoEngramShowsFlash(t *testing.T) {
	reg := newTestRegistry(t)
	_, _ = reg.Add("P", t.TempDir())

	// No daily-pack injection → engramClient is nil.
	m := tui.New(reg, &MockOpener{}, &MockClipboard{})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = dispatchKey(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = tui.DrainProjectsLoaded(t, m)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	base := len(config.DefaultLaunchers())
	for i := 0; i < base+4; i++ {
		m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Without engram it must NOT navigate to the picker.
	if m.Screen == tui.ScreenMemoryLink {
		t.Error("opened picker without engram client; expected to stay and flash")
	}
}
