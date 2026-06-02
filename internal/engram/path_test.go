package engram

import (
	"os"
	"strings"
	"testing"
)

// TestDefaultDBPath_ReturnsNonEmpty verifies the path is non-empty and ends in engram.db.
func TestDefaultDBPath_ReturnsNonEmpty(t *testing.T) {
	path, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath() error = %v, want nil", err)
	}
	if path == "" {
		t.Fatal("DefaultDBPath() returned empty string")
	}
	if !strings.HasSuffix(path, "engram.db") {
		t.Errorf("DefaultDBPath() = %q, want suffix engram.db", path)
	}
}

// TestDefaultDBPath_ContainsHomeDir verifies the path contains the home directory.
func TestDefaultDBPath_ContainsHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("os.UserHomeDir() not available:", err)
	}
	path, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath() error = %v", err)
	}
	if !strings.HasPrefix(path, home) {
		t.Errorf("DefaultDBPath() = %q, want prefix %q", path, home)
	}
}

// TestDefaultDBPath_UsesEngramSubdir verifies the path includes .engram subdir.
func TestDefaultDBPath_UsesEngramSubdir(t *testing.T) {
	path, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath() error = %v", err)
	}
	// Path must contain .engram directory component.
	if !strings.Contains(path, ".engram") {
		t.Errorf("DefaultDBPath() = %q, want to contain .engram", path)
	}
}
