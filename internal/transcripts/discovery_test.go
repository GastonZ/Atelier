package transcripts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gastonz/atelier/internal/transcripts"
)

// FakeClock is a test-only Clock implementation with a controllable time.
type FakeClock struct {
	CurrentTime time.Time
}

// Now returns the configured CurrentTime.
func (c *FakeClock) Now() time.Time {
	return c.CurrentTime
}

// setupDiscoveryFixtureRoot creates a temporary directory tree that mirrors
// ~/.claude/projects/ with controlled mtime values.
//
// Layout:
//
//	<root>/
//	  <projectKey>/
//	    <sessionID>.jsonl            (mtime = modTime)
//	    <sessionID>/subagents/       (optional sub-agents)
//
// Returns the root path and a cleanup function.
func setupDiscoveryFixtureRoot(t *testing.T, sessions []testSession) string {
	t.Helper()
	root := t.TempDir()

	for _, s := range sessions {
		dir := filepath.Join(root, s.projectKey)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}

		sessionFile := filepath.Join(dir, s.sessionID+".jsonl")

		// Write synthetic content (at least one user event with cwd).
		// JSON requires backslashes to be escaped, so use forward slashes or
		// replace backslashes with double-backslashes in the JSON string.
		jsonCwd := strings.ReplaceAll(s.cwd, `\`, `\\`)
		content := `{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":"test"},"uuid":"uuid-disc-001","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"` + s.sessionID + `","cwd":"` + jsonCwd + `","version":"2.1.126","gitBranch":"master"}` + "\n"
		if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
			t.Fatalf("write session file: %v", err)
		}
		// Set mtime to the requested value.
		if err := os.Chtimes(sessionFile, s.modTime, s.modTime); err != nil {
			t.Fatalf("chtimes %q: %v", sessionFile, err)
		}

		// Create sub-agent files if requested.
		for i, sub := range s.subAgents {
			subDir := filepath.Join(dir, s.sessionID, "subagents")
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("mkdir subagents: %v", err)
			}
			subFile := filepath.Join(subDir, sub)
			subContent := `{"type":"ai-title","aiTitle":"sub","sessionId":"sub-` + s.sessionID + `-` + string(rune('0'+i)) + `"}` + "\n"
			if err := os.WriteFile(subFile, []byte(subContent), 0644); err != nil {
				t.Fatalf("write sub-agent file: %v", err)
			}
		}
	}

	return root
}

type testSession struct {
	projectKey string
	sessionID  string
	cwd        string
	modTime    time.Time
	subAgents  []string // sub-agent filenames under subagents/
}

// ---- ListActive tests -------------------------------------------------------

func TestScanner_ListActive_RecentSession(t *testing.T) {
	// S1.2: session with mtime within active window appears in active set
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Projects--Atelier",
			sessionID:  "session-active-001",
			cwd:        `C:\Projects\Atelier`,
			modTime:    now.Add(-5 * time.Minute), // 5 min ago — within 15 min window
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 active session, got %d", len(sessions))
	}
	if sessions[0].ID != "session-active-001" {
		t.Errorf("expected session-active-001, got %q", sessions[0].ID)
	}
}

func TestScanner_ListActive_EmptyDirectory(t *testing.T) {
	// S1.1: empty projects directory returns empty slice (not error)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	root := t.TempDir() // empty — no project subdirs

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)
	if err != nil {
		t.Fatalf("ListActive error for empty dir: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for empty directory, got %d", len(sessions))
	}
}

func TestScanner_ListActive_OldSessionExcluded(t *testing.T) {
	// S1.2: sessions older than active window are not in the active set
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Projects--Atelier",
			sessionID:  "session-old-001",
			cwd:        `C:\Projects\Atelier`,
			modTime:    now.Add(-30 * time.Minute), // 30 min ago — outside 15 min window
		},
		{
			projectKey: "C--Projects--Atelier",
			sessionID:  "session-recent-001",
			cwd:        `C:\Projects\Atelier`,
			modTime:    now.Add(-5 * time.Minute), // 5 min ago — within window
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 active session, got %d", len(sessions))
	}
	if sessions[0].ID != "session-recent-001" {
		t.Errorf("expected session-recent-001 in active set, got %q", sessions[0].ID)
	}
}

func TestScanner_ListActive_WithSubAgents(t *testing.T) {
	// S1.3: active root session with sub-agents includes them in SubSessions
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Projects--Atelier",
			sessionID:  "session-with-subs",
			cwd:        `C:\Projects\Atelier`,
			modTime:    now.Add(-2 * time.Minute),
			subAgents:  []string{"agent-001.jsonl", "agent-002.jsonl"},
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(sessions))
	}

	if len(sessions[0].SubSessions) != 2 {
		t.Errorf("expected 2 sub-sessions, got %d", len(sessions[0].SubSessions))
	}
}

// ---- ListAll tests ----------------------------------------------------------

func TestScanner_ListAll_ReturnsAllSessions(t *testing.T) {
	// R1.1, R1.2: ListAll returns ALL sessions regardless of mtime
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Projects--A",
			sessionID:  "old-session",
			cwd:        `C:\Projects\A`,
			modTime:    now.Add(-48 * time.Hour), // 2 days ago
		},
		{
			projectKey: "C--Projects--B",
			sessionID:  "recent-session",
			cwd:        `C:\Projects\B`,
			modTime:    now.Add(-1 * time.Minute),
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListAll()
	if err != nil {
		t.Fatalf("ListAll error: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions from ListAll, got %d", len(sessions))
	}
}

// ---- LoadEvents tests -------------------------------------------------------

func TestScanner_LoadEvents_SimpleSession(t *testing.T) {
	// LoadEvents reads the session's .jsonl and returns the parsed events
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	// Build a minimal fixture directly.
	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Projects--Atelier",
			sessionID:  "session-for-load",
			cwd:        `C:\Projects\Atelier`,
			modTime:    now.Add(-3 * time.Minute),
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	events, err := scanner.LoadEvents("session-for-load")
	if err != nil {
		t.Fatalf("LoadEvents error: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected at least one event from LoadEvents")
	}
}

// ---- T13: TRIANGULATE edge cases -------------------------------------------

func TestScanner_ListActive_MissingDirectory_ReturnsError(t *testing.T) {
	// S1.5: missing .claude/projects/ directory returns error, not panic
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	scanner := transcripts.NewFileScanner("/nonexistent/path/that/does/not/exist", clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)

	if err == nil {
		t.Error("expected error for missing directory, got nil")
	}
	if sessions != nil {
		t.Errorf("expected nil sessions on error, got %v", sessions)
	}
}

func TestScanner_ListActive_ExpiredParent_SubAgentsNotIncluded(t *testing.T) {
	// R1.4: if parent session is expired, sub-agents are NOT in active set
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Projects--Atelier",
			sessionID:  "session-expired-parent",
			cwd:        `C:\Projects\Atelier`,
			modTime:    now.Add(-60 * time.Minute), // 60 min old — outside 15 min window
			subAgents:  []string{"agent-001.jsonl", "agent-002.jsonl"},
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	// The expired parent should not appear in active set
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions (parent expired), got %d", len(sessions))
	}
}

func TestScanner_CwdExtracted_CaseInsensitiveAvailableForCaller(t *testing.T) {
	// R1.5: cwd field from JSONL events is extracted (not the folder name encoding)
	// The cwd should be the raw path from the file content.
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}

	expectedCwd := `C:\Users\Usuario\Desktop\Atelier`
	root := setupDiscoveryFixtureRoot(t, []testSession{
		{
			projectKey: "C--Users-Usuario-Desktop-Atelier",
			sessionID:  "session-cwd-test",
			cwd:        expectedCwd,
			modTime:    now.Add(-2 * time.Minute),
		},
	})

	scanner := transcripts.NewFileScanner(root, clock, nil)
	sessions, err := scanner.ListActive(15 * time.Minute)
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	if sessions[0].Cwd != expectedCwd {
		t.Errorf("expected cwd=%q, got %q", expectedCwd, sessions[0].Cwd)
	}
}

func TestScanner_LoadEvents_UnknownSessionID_ReturnsError(t *testing.T) {
	// LoadEvents on a non-existent session ID returns an error, not a panic.
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{CurrentTime: now}
	root := t.TempDir()

	scanner := transcripts.NewFileScanner(root, clock, nil)
	_, err := scanner.LoadEvents("nonexistent-session-id")
	if err == nil {
		t.Error("expected error for nonexistent session ID, got nil")
	}
}
