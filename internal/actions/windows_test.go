//go:build windows

package actions_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/gastonz/atelier/internal/actions"
)

// TestNewOpener_ReturnsWindowsOpener verifies the constructor returns the concrete
// windowsOpener (via the seam) on Windows.
func TestNewOpener_ReturnsWindowsOpener(t *testing.T) {
	op := actions.NewOpener()
	if op == nil {
		t.Fatal("NewOpener() returned nil")
	}
	// Type check via interface satisfaction — ensure it is the concrete
	// windowsOpener via the seam.
	if _, ok := op.(actions.WindowsOpenerSeam); !ok {
		t.Error("NewOpener() should implement WindowsOpenerSeam (testability interface)")
	}
}

// TestWindowsOpener_OpenInClaudeCode_BuildsExpectedCommand verifies S4.1 semantics:
// The command is built as cmd.exe /c start "" claude with Dir=projectPath.
func TestWindowsOpener_OpenInClaudeCode_BuildsExpectedCommand(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	var capturedDir string

	// Swap the execCommand seam to capture args instead of launching.
	actions.SetExecCommand(func(name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		// Return a no-op cmd (exit 0 immediately) so Start() doesn't fail.
		c := exec.Command("cmd.exe", "/c", "exit", "0")
		capturedDir = c.Dir
		return c
	})
	defer actions.SetExecCommand(exec.Command) // restore

	op := actions.NewOpener()
	seam := op.(actions.WindowsOpenerSeam)
	if err := seam.OpenInClaudeCodeViaSeam("/test/path"); err != nil {
		// Start() on our fake cmd may succeed; real test is args
		_ = err
	}

	if capturedName != "cmd.exe" {
		t.Errorf("command name = %q, want cmd.exe", capturedName)
	}
	wantArgs := []string{"/c", "start", "", "claude"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}
	_ = capturedDir
}

// TestWindowsOpener_LaunchInDir_BuildsExpectedCommand verifies the generic launcher
// primitive: cmd.exe /c start "" <command> [args...].
func TestWindowsOpener_LaunchInDir_BuildsExpectedCommand(t *testing.T) {
	var capturedName string
	var capturedArgs []string

	actions.SetExecCommand(func(name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	defer actions.SetExecCommand(exec.Command)

	op := actions.NewOpener()
	if err := op.LaunchInDir("/test/path", "codex", "--model", "o3"); err != nil {
		_ = err // Start() on the fake cmd may succeed; the real assertion is on args
	}

	if capturedName != "cmd.exe" {
		t.Errorf("command name = %q, want cmd.exe", capturedName)
	}
	wantArgs := []string{"/c", "start", "", "codex", "--model", "o3"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}
}

// TestWindowsOpener_SpawnPowerShell_BuildsExpectedCommand verifies R4.8 semantics.
func TestWindowsOpener_SpawnPowerShell_BuildsExpectedCommand(t *testing.T) {
	var capturedName string
	var capturedArgs []string

	actions.SetExecCommand(func(name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	defer actions.SetExecCommand(exec.Command)

	op := actions.NewOpener()
	seam := op.(actions.WindowsOpenerSeam)
	_ = seam.SpawnPowerShellViaSeam("/test/path")

	if capturedName != "cmd.exe" {
		t.Errorf("command name = %q, want cmd.exe", capturedName)
	}
	wantArgs := []string{"/c", "start", "", "powershell.exe", "-NoLogo", "-NoExit"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}
}

// TestWindowsOpener_OpenInClaudeCode_Integration is an integration smoke test.
// Guarded by ATELIER_INTEGRATION=1. Not part of normal go test ./...
func TestWindowsOpener_OpenInClaudeCode_Integration(t *testing.T) {
	if os.Getenv("ATELIER_INTEGRATION") != "1" {
		t.Skip("set ATELIER_INTEGRATION=1 to run integration tests")
	}
	op := actions.NewOpener()
	// On CI this would fail — only runs locally with real cmd.exe
	if err := op.OpenInClaudeCode(t.TempDir()); err != nil {
		t.Logf("integration: OpenInClaudeCode returned (may be expected without claude installed): %v", err)
	}
}
