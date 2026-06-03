package actions_test

import (
	"os/exec"
	"testing"

	"github.com/gastonz/atelier/internal/actions"
)

// TestNewOpener_NotNil verifies the constructor returns a non-nil Opener on every
// platform. The Windows-specific concrete-type (seam) assertions live in
// windows_test.go, which is gated behind the windows build constraint.
func TestNewOpener_NotNil(t *testing.T) {
	if actions.NewOpener() == nil {
		t.Fatal("NewOpener() returned nil")
	}
}

// TestCommandAvailable verifies PATH resolution via the injectable seam.
func TestCommandAvailable(t *testing.T) {
	defer actions.SetLauncherLookPath(exec.LookPath)

	// Empty command is never available (short-circuits before lookup).
	if actions.CommandAvailable("") {
		t.Error("CommandAvailable(\"\") = true, want false")
	}

	actions.SetLauncherLookPath(func(string) (string, error) { return "/usr/bin/claude", nil })
	if !actions.CommandAvailable("claude") {
		t.Error("CommandAvailable(found) = false, want true")
	}

	actions.SetLauncherLookPath(func(string) (string, error) { return "", exec.ErrNotFound })
	if actions.CommandAvailable("nope") {
		t.Error("CommandAvailable(not found) = true, want false")
	}
}

// TestNewClipboard_ReturnsAtottoBacked verifies the constructor returns a non-nil Clipboard.
func TestNewClipboard_ReturnsAtottoBacked(t *testing.T) {
	cb := actions.NewClipboard()
	if cb == nil {
		t.Fatal("NewClipboard() returned nil")
	}
}

// TestAtottoClipboard_WriteAll_DelegatesToMock verifies that the MockClipboard records calls.
func TestAtottoClipboard_WriteAll_DelegatesToMock(t *testing.T) {
	mock := &MockClipboard{}
	if err := mock.WriteAll("/my/path"); err != nil {
		t.Fatalf("MockClipboard.WriteAll() error: %v", err)
	}
	if len(mock.Writes) != 1 {
		t.Fatalf("len(Writes) = %d, want 1", len(mock.Writes))
	}
	if mock.Writes[0] != "/my/path" {
		t.Errorf("Writes[0] = %q, want %q", mock.Writes[0], "/my/path")
	}
}
