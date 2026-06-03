package actions_test

import (
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
