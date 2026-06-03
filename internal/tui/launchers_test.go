package tui_test

// launchers_test.go — in-TUI launcher manager (ScreenLaunchers / ScreenLauncherForm).

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/tui"
)

func newLauncherTestModel(t *testing.T) (tui.Model, string) {
	t.Helper()
	reg := newTestRegistry(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	m := tui.New(reg, &MockOpener{}, &MockClipboard{})
	m = tui.WithConfigPath(m, cfgPath)
	return m, cfgPath
}

func TestLauncherManager_OpensFromWelcome(t *testing.T) {
	m, _ := newLauncherTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.Screen != tui.ScreenLaunchers {
		t.Fatalf("after 'l': Screen = %v, want ScreenLaunchers", m.Screen)
	}
	// Defaults are present out of the box.
	if m.LauncherCount() != len(config.DefaultLaunchers()) {
		t.Errorf("LauncherCount = %d, want %d", m.LauncherCount(), len(config.DefaultLaunchers()))
	}
}

func TestLauncherManager_AddPersists(t *testing.T) {
	m, cfgPath := newLauncherTestModel(t)
	start := m.LauncherCount()

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}) // → launchers
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}) // → form (new)
	if m.Screen != tui.ScreenLauncherForm {
		t.Fatalf("after 'a': Screen = %v, want ScreenLauncherForm", m.Screen)
	}

	// Type label, advance, type command, advance past args, submit.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Aider")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // → command field
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("aider")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // → args field
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // submit (args empty)

	if m.Screen != tui.ScreenLaunchers {
		t.Fatalf("after submit: Screen = %v, want ScreenLaunchers", m.Screen)
	}
	if m.LauncherCount() != start+1 {
		t.Errorf("LauncherCount = %d, want %d", m.LauncherCount(), start+1)
	}

	// Persisted to the injected config path.
	cfg, err := config.LoadAtelierConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadAtelierConfig error = %v", err)
	}
	if len(cfg.Launchers) != start+1 {
		t.Fatalf("persisted launchers = %d, want %d", len(cfg.Launchers), start+1)
	}
	last := cfg.Launchers[len(cfg.Launchers)-1]
	if last.Label != "Aider" || last.Command != "aider" {
		t.Errorf("persisted last launcher = %+v, want Aider/aider", last)
	}
}

func TestLauncherManager_DeletePersists(t *testing.T) {
	m, cfgPath := newLauncherTestModel(t)
	start := m.LauncherCount()
	if start == 0 {
		t.Skip("no default launchers to delete")
	}

	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}) // → launchers
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}) // delete cursor 0

	if m.LauncherCount() != start-1 {
		t.Errorf("LauncherCount = %d, want %d", m.LauncherCount(), start-1)
	}
	cfg, err := config.LoadAtelierConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadAtelierConfig error = %v", err)
	}
	if len(cfg.Launchers) != start-1 {
		t.Errorf("persisted launchers = %d, want %d", len(cfg.Launchers), start-1)
	}
}

func TestLauncherManager_ValidationRejectsEmptyCommand(t *testing.T) {
	m, _ := newLauncherTestModel(t)
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	// Label only, no command → submit attempt should stay on the form.
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Solo Nombre")})
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // → command
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // → args
	m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // submit attempt

	if m.Screen != tui.ScreenLauncherForm {
		t.Errorf("empty command: Screen = %v, want ScreenLauncherForm (rejected)", m.Screen)
	}
}
