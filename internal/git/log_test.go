package git

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// fakeOutputCmd returns a *exec.Cmd that outputs the given content via cmd.exe /c type.
// This is Windows-compatible and avoids the Unix `cat` command.
func fakeOutputCmd(t *testing.T, content string) *exec.Cmd {
	t.Helper()
	f := t.TempDir() + "\\out.txt"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatalf("fakeOutputCmd: write: %v", err)
	}
	return exec.Command("cmd.exe", "/c", "type", f)
}

// ============================================================================
// parseLogLine unit tests (pure function — no seam needed)
// ============================================================================

// TestParseLogLine_ValidLine verifies happy-path parsing.
func TestParseLogLine_ValidLine(t *testing.T) {
	commit, ok := parseLogLine("abc1234|2026-05-04|Add feature")
	if !ok {
		t.Fatal("parseLogLine() ok = false, want true")
	}
	if commit.Hash != "abc1234" {
		t.Errorf("Hash = %q, want abc1234", commit.Hash)
	}
	want := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	if !commit.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", commit.Date, want)
	}
	if commit.Subject != "Add feature" {
		t.Errorf("Subject = %q, want 'Add feature'", commit.Subject)
	}
}

// TestParseLogLine_MalformedLine verifies malformed lines are skipped.
func TestParseLogLine_MalformedLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only one field", "abc1234"},
		{"only two fields", "abc1234|2026-05-04"},
		{"bad date", "abc1234|not-a-date|subject"},
		{"leading pipe no hash", "|2026-05-04|subject"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseLogLine(tt.input)
			if ok {
				t.Errorf("parseLogLine(%q) ok = true, want false", tt.input)
			}
		})
	}
}

// TestParseLogLine_SubjectWithPipes verifies subjects containing | are preserved.
func TestParseLogLine_SubjectWithPipes(t *testing.T) {
	commit, ok := parseLogLine("abc1234|2026-05-04|Subject with | pipes")
	if !ok {
		t.Fatal("parseLogLine() ok = false")
	}
	if commit.Subject != "Subject with | pipes" {
		t.Errorf("Subject = %q, want 'Subject with | pipes'", commit.Subject)
	}
}

// ============================================================================
// parseLogOutput unit tests (pure function)
// ============================================================================

// TestParseLogOutput_ThirtyLines verifies 30 valid lines → 30 commits.
func TestParseLogOutput_ThirtyLines(t *testing.T) {
	var lines string
	for i := 1; i <= 30; i++ {
		lines += fmt.Sprintf("abc%04d|2026-05-04|Commit %d\n", i, i)
	}
	commits := parseLogOutput(lines)
	if len(commits) != 30 {
		t.Errorf("parseLogOutput returned %d commits, want 30", len(commits))
	}
}

// TestParseLogOutput_EmptyOutput verifies empty string → empty slice.
func TestParseLogOutput_EmptyOutput(t *testing.T) {
	commits := parseLogOutput("")
	if len(commits) != 0 {
		t.Errorf("parseLogOutput('') returned %d, want 0", len(commits))
	}
}

// TestParseLogOutput_MalformedLinesSkipped verifies malformed lines are skipped silently.
func TestParseLogOutput_MalformedLinesSkipped(t *testing.T) {
	stdout := "abc1234|2026-05-04|Good\nnot-valid\n|bad|\nabc5678|2026-05-05|Also good\n"
	commits := parseLogOutput(stdout)
	if len(commits) != 2 {
		t.Errorf("parseLogOutput returned %d commits, want 2", len(commits))
	}
}

// ============================================================================
// Log / Show integration tests using execCommand seam
// ============================================================================

// TestLog_ParsesThroughExecSeam verifies Log parses commits via the seam.
func TestLog_ParsesThroughExecSeam(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) { return "git.exe", nil }
	defer func() { lookPath = origLP; resetAvailableCache() }()

	var lines string
	for i := 1; i <= 5; i++ {
		lines += fmt.Sprintf("abc%04d|2026-05-04|Commit %d\n", i, i)
	}

	SetExecCommand(func(name string, arg ...string) *exec.Cmd {
		cmd := fakeOutputCmd(t, lines)
		return cmd
	})
	defer SetExecCommand(exec.Command)

	lr := NewLogReader()
	commits, err := lr.Log("", 5)
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(commits) != 5 {
		t.Errorf("Log() returned %d commits, want 5", len(commits))
	}
}

// TestLog_EmptyOutputViaSeam verifies empty exec output → empty slice.
func TestLog_EmptyOutputViaSeam(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) { return "git.exe", nil }
	defer func() { lookPath = origLP; resetAvailableCache() }()

	SetExecCommand(func(name string, arg ...string) *exec.Cmd {
		return fakeOutputCmd(t, "")
	})
	defer SetExecCommand(exec.Command)

	lr := NewLogReader()
	commits, err := lr.Log("", 30)
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("Log() returned %d commits, want 0", len(commits))
	}
}

// TestShow_PassesThroughOutput verifies Show returns non-empty stdout.
func TestShow_PassesThroughOutput(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) { return "git.exe", nil }
	defer func() { lookPath = origLP; resetAvailableCache() }()

	content := "commit abc1234\nAuthor: Test\nDate: 2026-05-04\n"

	SetExecCommand(func(name string, arg ...string) *exec.Cmd {
		return fakeOutputCmd(t, content)
	})
	defer SetExecCommand(exec.Command)

	lr := NewLogReader()
	out, err := lr.Show("", "abc1234")
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if len(out) == 0 {
		t.Error("Show() returned empty output, want non-empty")
	}
}

// TestShow_NonZeroExit verifies error on non-zero exit.
func TestShow_NonZeroExit(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) { return "git.exe", nil }
	defer func() { lookPath = origLP; resetAvailableCache() }()

	SetExecCommand(func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd.exe", "/c", "exit", "1")
	})
	defer SetExecCommand(exec.Command)

	lr := NewLogReader()
	_, err := lr.Show("", "badhash")
	if err == nil {
		t.Fatal("Show() expected error on non-zero exit, got nil")
	}
}

// TestLog_ZeroCommitsGitMissing verifies empty result when git not on PATH.
func TestLog_ZeroCommitsGitMissing(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) {
		return "", &exec.Error{Name: "git", Err: exec.ErrNotFound}
	}
	defer func() { lookPath = origLP; resetAvailableCache() }()

	lr := NewLogReader()
	commits, err := lr.Log("", 30)
	if err != nil {
		t.Fatalf("Log() should not error when git missing, got: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("Log() = %d commits, want 0 when git unavailable", len(commits))
	}
}
