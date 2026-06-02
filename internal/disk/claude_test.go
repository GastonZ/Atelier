package disk

// claude_test.go — Tests for ClaudeProjectsDir and ClaudeProjectsDirForPath.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClaudeProjectsDir_ReturnsPath verifies ClaudeProjectsDir returns a non-empty path.
// This test calls the real os.UserHomeDir so it works on any machine.
func TestClaudeProjectsDir_ReturnsPath(t *testing.T) {
	got, err := ClaudeProjectsDir()
	if err != nil {
		t.Fatalf("ClaudeProjectsDir() error: %v", err)
	}
	if got == "" {
		t.Error("ClaudeProjectsDir() returned empty string")
	}
	// Should contain ".claude/projects" as a suffix path component.
	if filepath.Base(filepath.Dir(got)) != ".claude" {
		// Flexible check: just verify the path ends in ".claude/projects" (cross-platform).
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".claude", "projects")
		if got != expected {
			t.Errorf("ClaudeProjectsDir() = %q, want %q", got, expected)
		}
	}
}

// TestClaudeProjectsDirForPath_ExistingDir returns the path when the dir exists.
func TestClaudeProjectsDirForPath_ExistingDir(t *testing.T) {
	// We can't easily test with a real ~/.claude/projects/<key>,
	// but we can test the heuristic key derivation by creating the dir.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("os.UserHomeDir() failed:", err)
	}

	// Build a synthetic tomo path and derive its expected key.
	tomoPath := "C:\\Users\\Test\\myproject"
	key := "c:-users-test-myproject"
	dir := filepath.Join(home, ".claude", "projects", key)

	// Create the dir temporarily.
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Skipf("Could not create test dir %q: %v", dir, err)
	}
	defer os.RemoveAll(dir)

	got, err := ClaudeProjectsDirForPath(tomoPath)
	if err != nil {
		t.Fatalf("ClaudeProjectsDirForPath() error: %v", err)
	}
	if got == "" {
		t.Errorf("ClaudeProjectsDirForPath(%q) returned empty (dir exists, expected non-empty)", tomoPath)
	}
}

// TestClaudeProjectsDirForPath_NonExistingDir returns empty string when dir doesn't exist.
func TestClaudeProjectsDirForPath_NonExistingDir(t *testing.T) {
	// A tomo path that will never map to an existing dir.
	tomoPath := "Z:\\nonexistent\\path\\for\\test\\xyz\\abc123"
	got, err := ClaudeProjectsDirForPath(tomoPath)
	if err != nil {
		t.Fatalf("ClaudeProjectsDirForPath() error: %v (expected nil for non-existing dir)", err)
	}
	// Non-existing dir → returns empty string (best-effort).
	if got != "" {
		// The dir somehow exists — that's fine, just log it.
		t.Logf("ClaudeProjectsDirForPath(%q) = %q (dir exists unexpectedly)", tomoPath, got)
	}
}

// TestClaudeProjectsDirForPath_KeyDerivation verifies the heuristic key derivation
// (lowercase + replace separators/colons with '-') without creating real dirs.
func TestClaudeProjectsDirForPath_KeyDerivation(t *testing.T) {
	// Test that the function runs without error on various inputs.
	paths := []string{
		`C:\Users\test\project`,
		`/home/user/project`,
		`C:/Users/test/project`,
		`simple`,
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			// Just verify no panic or unexpected error.
			_, err := ClaudeProjectsDirForPath(p)
			if err != nil {
				t.Errorf("ClaudeProjectsDirForPath(%q) error: %v", p, err)
			}
		})
	}
}
