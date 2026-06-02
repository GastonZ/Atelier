package tui_test

// daily_golden_test.go — T47-T49: Golden view tests for the daily-driver-pack screens.
//
// Tests render a model to a fixed-width string and compare against golden files.
// To update golden files: go test ./internal/tui/... -update
//
// Golden files live in testdata/*.golden (checked into git).
// Four scenarios:
//   - memory_browser_empty.golden
//   - memory_browser_populated.golden
//   - project_history_mixed.golden
//   - disk_usage_standard.golden

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/tui"
)

// TestDailyGoldenViews runs all golden view scenarios for the daily-driver-pack screens.
func TestDailyGoldenViews(t *testing.T) {
	fixedNow := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)

	tests := []goldenTest{
		{
			// T47a: Memory Browser — empty state (project with no observations).
			name: "memory_browser_empty",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				eng := &mockEngramClientForTUI{
					observations: []engram.Observation{}, // empty
				}
				m := newTestModel(t)
				m = tui.InjectDailyPackDeps(m, eng, nil, nil)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				m = tui.SetScreenForTest(m, tui.ScreenMemoryBrowser)
				// Dispatch memoryLoadedMsg with empty entries.
				result, _ = m.Update(tui.MakeMemoryLoadedMsg([]engram.Observation{}, nil))
				return result.(tui.Model)
			},
		},
		{
			// T47b: Memory Browser — populated state (3 observations).
			name: "memory_browser_populated",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				obs := []engram.Observation{
					{
						ID:        1,
						Project:   "atelier",
						Scope:     "project",
						Type:      "architecture",
						Title:     "JWT auth middleware",
						Content:   "**What**: Switched from sessions to JWT\n**Why**: Scalability",
						Timestamp: fixedNow.Add(-2 * 24 * time.Hour),
					},
					{
						ID:        2,
						Project:   "atelier",
						Scope:     "project",
						Type:      "bugfix",
						Title:     "Fixed N+1 query",
						Content:   "**What**: Added index to users table\n**Why**: Slow queries",
						Timestamp: fixedNow.Add(-24 * time.Hour),
					},
					{
						ID:        3,
						Project:   "atelier",
						Scope:     "project",
						Type:      "decision",
						Title:     "Use Bubble Tea",
						Content:   "**What**: Chose Bubble Tea for TUI framework",
						Timestamp: fixedNow,
					},
				}
				eng := &mockEngramClientForTUI{observations: obs}
				m := newTestModel(t)
				m = tui.InjectDailyPackDeps(m, eng, nil, nil)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				m = tui.SetScreenForTest(m, tui.ScreenMemoryBrowser)
				result, _ = m.Update(tui.MakeMemoryLoadedMsg(obs, nil))
				return result.(tui.Model)
			},
		},
		{
			// T48: Project History — mixed git + SDD entries.
			name: "project_history_mixed",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				entries := []tui.HistoryEntry{
					{
						Source: "git",
						Date:   fixedNow,
						Title:  "feat: add daily driver pack screens",
						Ref:    "abc1234",
					},
					{
						Source: "sdd",
						Date:   fixedNow,
						Title:  "sdd/atelier-daily-driver-pack/archive-report",
						Ref:    "42",
					},
					{
						Source: "git",
						Date:   fixedNow.Add(-24 * time.Hour),
						Title:  "fix: memory browser filter precedence",
						Ref:    "def5678",
					},
					{
						Source: "git",
						Date:   fixedNow.Add(-48 * time.Hour),
						Title:  "refactor: extract commands_daily",
						Ref:    "ghi9012",
					},
				}
				m := newTestModel(t)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				m = tui.SetScreenForTest(m, tui.ScreenProjectHistory)
				result, _ = m.Update(tui.MakeHistoryLoadedMsg(entries, nil))
				return result.(tui.Model)
			},
		},
		{
			// T49: Disk Usage — standard state with data loaded.
			name: "disk_usage_standard",
			build: func(t *testing.T) tui.Model {
				t.Helper()
				reg := newTestRegistry(t)
				projPath := t.TempDir()
				proj, _ := reg.Add("Atelier", projPath)
				m := newTestModelWithReg(t, reg)
				result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				m = result.(tui.Model)
				m = tui.DrainProjectsLoaded(t, m)
				m = tui.SetScreenForTest(m, tui.ScreenDiskUsage)

				// Per-tomo: keyed by project ID (UUID from test registry).
				perTomo := map[string]int64{
					proj.ID: 1024 * 1024 * 3, // 3 MB
				}
				result, _ = m.Update(tui.MakeDiskUsageLoadedMsg(
					1024*1024*45, // 45 MB engram
					1024*1024*120, // 120 MB claude projects
					perTomo,
					nil,
				))
				return result.(tui.Model)
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
				// First run: create golden file automatically.
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
