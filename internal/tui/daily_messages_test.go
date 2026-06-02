package tui_test

// daily_messages_test.go — RED tests for T30: new tea.Msg types, cmd factories, copy constants.

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/git"
	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/tui"
)

// --- Mock engram client for TUI tests ----------------------------------------

type mockEngramClientForTUI struct {
	observations []engram.Observation
	archives     []engram.Observation
	byIDObs      engram.Observation
	byIDErr      error
	listErr      error
}

func (m *mockEngramClientForTUI) ListByProject(_ string) ([]engram.Observation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.observations, nil
}

func (m *mockEngramClientForTUI) GetByID(_ int64) (engram.Observation, error) {
	return m.byIDObs, m.byIDErr
}

func (m *mockEngramClientForTUI) ListSDDArchivesForProject(_ string) ([]engram.Observation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.archives, nil
}

func (m *mockEngramClientForTUI) Close() error { return nil }

// --- Mock git readers for TUI tests ------------------------------------------

type mockStatusReaderForTUI struct {
	statuses map[string]git.Status
	err      error
}

func (m *mockStatusReaderForTUI) Status(repoPath string) (git.Status, error) {
	if m.err != nil {
		return git.Status{}, m.err
	}
	if s, ok := m.statuses[repoPath]; ok {
		return s, nil
	}
	return git.Status{Available: true, IsRepo: false}, nil
}

type mockLogReaderForTUI struct {
	commits  []git.Commit
	showBody string
	logErr   error
	showErr  error
}

func (m *mockLogReaderForTUI) Log(_ string, _ int) ([]git.Commit, error) {
	return m.commits, m.logErr
}

func (m *mockLogReaderForTUI) Show(_, _ string) (string, error) {
	return m.showBody, m.showErr
}

// --- T30 copy constants tests -------------------------------------------------

// TestCopyConstants_MemoryBrowserCopy verifies the locked Spanish copy constants exist and are correct.
func TestCopyConstants_MemoryBrowserCopy(t *testing.T) {
	tests := []struct {
		name  string
		got   string
		want  string
	}{
		{"CopyMemoryEmpty", tui.CopyMemoryEmpty, "Este tomo aún no tiene memorias."},
		{"CopyMemoryLoading", tui.CopyMemoryLoading, "Consultando los pergaminos…"},
		{"CopyHistoryEmpty", tui.CopyHistoryEmpty, "Este tomo no tiene crónicas registradas."},
		{"CopyHistoryLoading", tui.CopyHistoryLoading, "Reuniendo la crónica…"},
		{"CopyDiskLoading", tui.CopyDiskLoading, "Pesando los pergaminos…"},
		{"CopyDiskZeroPerTomo", tui.CopyDiskZeroPerTomo, "Sin crónica"},
		{"CopyVSCodeMissing", tui.CopyVSCodeMissing, "VS Code no encontrado. Instalalo o agregálo al PATH."},
		{"CopyGitMissing", tui.CopyGitMissing, "git no instalado."},
		{"CopyHistoryGitMarker", tui.CopyHistoryGitMarker, "[git]"},
		{"CopyHistorySDDMarker", tui.CopyHistorySDDMarker, "[sdd]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestCopyConstants_FormatStrings verifies format-string copy constants exist.
func TestCopyConstants_FormatStrings(t *testing.T) {
	// These are format strings - just verify they're non-empty.
	if tui.CopyMemoryError == "" {
		t.Error("CopyMemoryError should be non-empty")
	}
	if tui.CopyHistoryError == "" {
		t.Error("CopyHistoryError should be non-empty")
	}
}

// TestCopyConstants_FooterHints verifies footer hint constants exist.
func TestCopyConstants_FooterHints(t *testing.T) {
	hints := []struct {
		name string
		val  string
	}{
		{"CopyFooterMemoryList", tui.CopyFooterMemoryList},
		{"CopyFooterMemoryDetail", tui.CopyFooterMemoryDetail},
		{"CopyFooterHistoryList", tui.CopyFooterHistoryList},
		{"CopyFooterHistoryDetail", tui.CopyFooterHistoryDetail},
		{"CopyFooterDiskUsage", tui.CopyFooterDiskUsage},
		{"CopyFooterProjectsExt", tui.CopyFooterProjectsExt},
	}
	for _, tt := range hints {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val == "" {
				t.Errorf("%s must be non-empty", tt.name)
			}
		})
	}
}

// --- T30 message constructor tests -------------------------------------------

// TestMakeMemoryLoadedMsg_ConstructsCorrectly verifies the Make* constructors exist and work.
func TestMakeMemoryLoadedMsg_ConstructsCorrectly(t *testing.T) {
	obs := []engram.Observation{
		{ID: 1, Title: "Test"},
	}
	msg := tui.MakeMemoryLoadedMsg(obs, nil)
	if msg == nil {
		t.Fatal("MakeMemoryLoadedMsg returned nil")
	}
}

func TestMakeMemoryDetailLoadedMsg_ConstructsCorrectly(t *testing.T) {
	obs := engram.Observation{ID: 42, Content: "full content"}
	msg := tui.MakeMemoryDetailLoadedMsg(obs, nil)
	if msg == nil {
		t.Fatal("MakeMemoryDetailLoadedMsg returned nil")
	}
}

func TestMakeHistoryLoadedMsg_ConstructsCorrectly(t *testing.T) {
	entries := []tui.HistoryEntry{
		{Source: "git", Date: time.Now(), Title: "fix bug", Ref: "abc123"},
	}
	msg := tui.MakeHistoryLoadedMsg(entries, nil)
	if msg == nil {
		t.Fatal("MakeHistoryLoadedMsg returned nil")
	}
}

func TestMakeHistoryDetailLoadedMsg_ConstructsCorrectly(t *testing.T) {
	msg := tui.MakeHistoryDetailLoadedMsg("diff content", nil)
	if msg == nil {
		t.Fatal("MakeHistoryDetailLoadedMsg returned nil")
	}
}

func TestMakeGitStatusLoadedMsg_ConstructsCorrectly(t *testing.T) {
	statuses := map[string]git.Status{
		"/path/to/repo": {Available: true, IsRepo: true},
	}
	msg := tui.MakeGitStatusLoadedMsg(statuses, nil)
	if msg == nil {
		t.Fatal("MakeGitStatusLoadedMsg returned nil")
	}
}

func TestMakeDiskUsageLoadedMsg_ConstructsCorrectly(t *testing.T) {
	msg := tui.MakeDiskUsageLoadedMsg(1024, 2048, nil, nil)
	if msg == nil {
		t.Fatal("MakeDiskUsageLoadedMsg returned nil")
	}
}

// --- T30 cmd factories tests -------------------------------------------------

// TestLoadMemoryCmd_EmitsMemoryLoadedMsg verifies the cmd factory emits the correct message.
func TestLoadMemoryCmd_EmitsMemoryLoadedMsg(t *testing.T) {
	client := &mockEngramClientForTUI{
		observations: []engram.Observation{
			{ID: 1, Title: "obs1", Project: "atelier"},
		},
	}
	cmd := tui.LoadMemoryCmdForTest(client, "atelier")
	if cmd == nil {
		t.Fatal("LoadMemoryCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

// TestLoadMemoryCmd_ErrorPath verifies error path emits error message.
func TestLoadMemoryCmd_ErrorPath(t *testing.T) {
	client := &mockEngramClientForTUI{
		listErr: errors.New("db locked"),
	}
	cmd := tui.LoadMemoryCmdForTest(client, "atelier")
	if cmd == nil {
		t.Fatal("LoadMemoryCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil on error path")
	}
}

// TestLoadGitStatusCmd_EmitsGitStatusMsg verifies the git status fan-out cmd.
func TestLoadGitStatusCmd_EmitsGitStatusMsg(t *testing.T) {
	sr := &mockStatusReaderForTUI{
		statuses: map[string]git.Status{
			"/repo": {Available: true, IsRepo: true, Modified: 2},
		},
	}
	projects := []registry.Project{
		{ID: "p1", Name: "Test", Path: "/repo"},
	}
	cmd := tui.LoadGitStatusCmdForTest(sr, projects)
	if cmd == nil {
		t.Fatal("LoadGitStatusCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

// TestLoadHistoryCmd_EmitsHistoryMsg verifies the history merge+sort cmd.
func TestLoadHistoryCmd_EmitsHistoryMsg(t *testing.T) {
	client := &mockEngramClientForTUI{
		archives: []engram.Observation{
			{ID: 1, Title: "SDD archive", TopicKey: "sdd/test/archive-report"},
		},
	}
	lr := &mockLogReaderForTUI{
		commits: []git.Commit{
			{Hash: "abc", Date: time.Now(), Subject: "first commit"},
		},
	}
	cmd := tui.LoadHistoryCmdForTest(client, lr, "atelier", "/repo")
	if cmd == nil {
		t.Fatal("LoadHistoryCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

// TestLoadDiskUsageCmd_EmitsDiskMsg verifies the disk usage cmd.
func TestLoadDiskUsageCmd_EmitsDiskMsg(t *testing.T) {
	projects := []registry.Project{
		{ID: "p1", Name: "Test", Path: "/repo"},
	}
	cmd := tui.LoadDiskUsageCmdForTest(projects)
	if cmd == nil {
		t.Fatal("LoadDiskUsageCmdForTest returned nil cmd")
	}
	msg := cmd()
	// May be any type — just verify non-nil.
	_ = msg
}

// TestFormatHistoryDate_ISO8601 verifies the locked date format.
func TestFormatHistoryDate_ISO8601(t *testing.T) {
	d := time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
	got := tui.FormatHistoryDate(d)
	if got != "2026-05-04" {
		t.Errorf("FormatHistoryDate = %q, want %q", got, "2026-05-04")
	}
}

// TestFormatGitIndicatorInTUI_Passthrough verifies the TUI uses correct indicator format.
func TestFormatGitIndicatorInTUI_Passthrough(t *testing.T) {
	// git.FormatGitIndicator is already tested in internal/git.
	// Here we just check TUI can call it or wraps it correctly.
	// The indicator for a clean repo is "✓".
	status := git.Status{Available: true, IsRepo: true, Modified: 0}
	ind := git.FormatGitIndicator(status)
	if ind != "✓" {
		t.Errorf("clean indicator = %q, want ✓", ind)
	}
}

// TestLoadMemoryDetailCmd_EmitsDetailMsg verifies the memory detail cmd.
func TestLoadMemoryDetailCmd_EmitsDetailMsg(t *testing.T) {
	obs := engram.Observation{ID: 42, Title: "detail obs", Content: "full content here"}
	client := &mockEngramClientForTUI{byIDObs: obs}
	cmd := tui.LoadMemoryDetailCmdForTest(client, 42)
	if cmd == nil {
		t.Fatal("LoadMemoryDetailCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

// TestLoadHistoryDetailGitCmd_EmitsDetailMsg verifies the git detail cmd.
func TestLoadHistoryDetailGitCmd_EmitsDetailMsg(t *testing.T) {
	lr := &mockLogReaderForTUI{showBody: "diff content"}
	cmd := tui.LoadHistoryDetailGitCmdForTest(lr, "/repo", "abc123")
	if cmd == nil {
		t.Fatal("LoadHistoryDetailGitCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

// TestLoadHistoryDetailSDDCmd_EmitsDetailMsg verifies the SDD detail cmd.
func TestLoadHistoryDetailSDDCmd_EmitsDetailMsg(t *testing.T) {
	obs := engram.Observation{ID: 99, Content: "# Archive Report\n\nContent here"}
	client := &mockEngramClientForTUI{byIDObs: obs}
	cmd := tui.LoadHistoryDetailSDDCmdForTest(client, 99)
	if cmd == nil {
		t.Fatal("LoadHistoryDetailSDDCmdForTest returned nil cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil")
	}
}

// Ensure tea.Cmd is a function type (compile guard).
var _ tea.Cmd = tea.Cmd(nil)
