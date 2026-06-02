package tui_test

// golden_test.go — T41: Golden view tests for agent monitor screens.
// Tests render a model to a fixed-width string and compare against golden files.
// To update golden files: go test ./internal/tui/... -update
//
// Golden files live in testdata/*.golden (checked into git).
// Five scenarios:
//   - agent_monitor_empty.golden
//   - agent_monitor_one_tile.golden
//   - agent_monitor_one_tile_subagents_expanded.golden
//   - agent_zoom.golden
//   - agent_replay_paused.golden

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/transcripts"
	"github.com/gastonz/atelier/internal/tui"
)

// updateGolden is set via -update flag to regenerate golden files.
var updateGolden = flag.Bool("update", false, "update golden files")

const goldenDir = "testdata"

// goldenTest holds the input model state and expected golden file name.
type goldenTest struct {
	name  string
	build func(t *testing.T) tui.Model
}

// TestGoldenViews runs all golden view scenarios.
func TestGoldenViews(t *testing.T) {
	// Fixed time for deterministic output.
	fixedNow := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	// Pin relativeTime's clock to fixedNow so session timestamps render as
	// "ahora mismo" regardless of the real date the suite runs on. Without this,
	// relativeTime compares against the real wall clock and the golden output
	// drifts ("29d", "31d", …) as time passes — a latent non-determinism bug.
	defer tui.SetNowFuncForTest(func() time.Time { return fixedNow })()

	tests := []goldenTest{
		{
			name: "agent_monitor_empty",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
				m, _ = navigateToMonitor(t, m)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg([]transcripts.Session{}, nil))
				return result.(tui.Model)
			},
		},
		{
			name: "agent_monitor_one_tile",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				sessions := []transcripts.Session{makeSessionAt("s1", fixedNow, 0, "MiProyecto")}
				m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
				m, _ = navigateToMonitor(t, m)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
				return result.(tui.Model)
			},
		},
		{
			name: "agent_monitor_one_tile_subagents_expanded",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				sessions := []transcripts.Session{makeSessionAt("s1", fixedNow, 2, "MiProyecto")}
				m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
				m, _ = navigateToMonitor(t, m)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
				m = result.(tui.Model)
				// Expand sub-agents
				m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
				return m
			},
		},
		{
			name: "agent_zoom",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				sessions := []transcripts.Session{makeSessionAt("s1", fixedNow, 0, "MiProyecto")}
				m := newMonitorModel(t, &fakeScannerForTUI{}, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
				m, _ = navigateToMonitor(t, m)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
				m = result.(tui.Model)
				m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
				return m
			},
		},
		{
			name: "agent_replay_paused",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				sessions := []transcripts.Session{makeSessionAt("s1", fixedNow, 0, "MiProyecto")}
				events := []transcripts.Event{
					&transcripts.AssistantEvent{
						UUIDValue:      "evt1",
						SessionIDValue: "s1",
						TimestampValue: fixedNow,
						Model:          "claude-sonnet-4-6",
						Text:           "Hola, estoy listo para ayudarte.",
					},
				}
				scanner := &fakeScannerForTUI{activeSessions: sessions, loadEvents: events}
				m := newMonitorModel(t, scanner, newFakeWatcherForTUI(4), &fakePriceTableForTUI{known: true}, config.DefaultAtelierConfig())
				m, _ = navigateToMonitor(t, m)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				result, _ = m.Update(tui.MakeAgentSessionsLoadedMsg(sessions, nil))
				m = result.(tui.Model)
				m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
				m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
				result, _ = m.Update(tui.MakeReplayLoadedMsg("s1", events, nil))
				m = result.(tui.Model)
				// Pause
				m, _ = dispatchKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
				return m
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.build(t)
			got := normalizeView(m.View())

			goldenPath := filepath.Join(goldenDir, tt.name+".golden")

			if *updateGolden {
				if err := os.MkdirAll(goldenDir, 0o755); err != nil {
					t.Fatalf("mkdir testdata: %v", err)
				}
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				t.Logf("updated golden: %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				// First run: create golden file automatically
				if err := os.MkdirAll(goldenDir, 0o755); err != nil {
					t.Fatalf("mkdir testdata: %v", err)
				}
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				t.Logf("created golden: %s", goldenPath)
				return
			}
			if err != nil {
				t.Fatalf("read golden %s: %v", goldenPath, err)
			}

			wantStr := normalizeView(string(want))
			if got != wantStr {
				t.Errorf("view mismatch for %s.\nGot:\n%s\n\nWant:\n%s", tt.name, got, wantStr)
			}
		})
	}
}

// normalizeView strips trailing whitespace from each line and trailing blank
// lines so golden files are stable across platforms.
func normalizeView(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	// Trim trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// makeSessionAt creates a session with a fixed time for golden tests.
func makeSessionAt(id string, ts time.Time, numSubs int, projectName string) transcripts.Session {
	s := makeSession(id, ts, numSubs)
	s.ProjectName = projectName
	for i := range s.SubSessions {
		s.SubSessions[i].ProjectName = "sub-" + projectName
	}
	return s
}
