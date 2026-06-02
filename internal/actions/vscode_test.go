//go:build windows

package actions_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/gastonz/atelier/internal/actions"
)

// TestOpenInVSCode_CodeOnPath verifies S1.1: spawns via cmd.exe wrapper when Code.exe
// cannot be derived from the mocked PATH entry (the fake path has no real parent).
func TestOpenInVSCode_CodeOnPath(t *testing.T) {
	var capturedName string
	var capturedArgs []string

	actions.SetLookPath(func(file string) (string, error) {
		if file == "code" {
			return "C:\\code.exe", nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	})
	defer actions.SetLookPath(exec.LookPath)

	actions.SetVSCodeExecCommand(func(name string, arg ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = arg
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	defer actions.SetVSCodeExecCommand(exec.Command)

	op := actions.NewOpener()
	_ = op.OpenInVSCode("C:\\my\\project")

	if capturedName != "cmd.exe" {
		t.Errorf("command = %q, want cmd.exe", capturedName)
	}
	found := false
	for _, a := range capturedArgs {
		if a == "code" {
			found = true
		}
	}
	if !found {
		t.Errorf("args %v do not contain 'code'", capturedArgs)
	}
}

// TestOpenInVSCode_CodeOnPath_DerivesCodeExe verifies that when code.cmd is on PATH
// and Code.exe exists at the derived path, it is spawned directly (no cmd.exe).
func TestOpenInVSCode_CodeOnPath_DerivesCodeExe(t *testing.T) {
	var capturedBinary string

	actions.SetLookPath(func(file string) (string, error) {
		if file == "code" {
			return `C:\FakeInstall\bin\code.cmd`, nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	})
	defer actions.SetLookPath(exec.LookPath)

	actions.SetVSCodeStatCheck(func(path string) bool {
		return strings.HasSuffix(path, "Code.exe")
	})
	defer actions.SetVSCodeStatCheck(nil)

	actions.SetVSCodeExecCommand(func(name string, arg ...string) *exec.Cmd {
		capturedBinary = name
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	defer actions.SetVSCodeExecCommand(exec.Command)

	_ = actions.NewOpener().OpenInVSCode(`C:\my\project`)

	if !strings.HasSuffix(capturedBinary, "Code.exe") {
		t.Errorf("binary = %q, want path ending in Code.exe", capturedBinary)
	}
}

// TestOpenInVSCode_OnlyInsiders verifies S1.2: uses code-insiders when code is missing.
func TestOpenInVSCode_OnlyInsiders(t *testing.T) {
	var capturedArgs []string

	actions.SetLookPath(func(file string) (string, error) {
		if file == "code-insiders" {
			return "C:\\code-insiders.exe", nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	})
	defer actions.SetLookPath(exec.LookPath)

	actions.SetVSCodeExecCommand(func(name string, arg ...string) *exec.Cmd {
		capturedArgs = arg
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	defer actions.SetVSCodeExecCommand(exec.Command)

	_ = actions.NewOpener().OpenInVSCode("C:\\project")

	found := false
	for _, a := range capturedArgs {
		if a == "code-insiders" {
			found = true
		}
	}
	if !found {
		t.Errorf("args %v do not contain 'code-insiders'", capturedArgs)
	}
}

// TestOpenInVSCode_LocalAppDataFallback verifies S1.3: spawns Code.exe directly from
// the known %LOCALAPPDATA% location when code/code-insiders are not on PATH.
func TestOpenInVSCode_LocalAppDataFallback(t *testing.T) {
	var capturedBinary string
	var capturedArgs []string

	actions.SetLookPath(func(file string) (string, error) {
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	})
	defer actions.SetLookPath(exec.LookPath)

	actions.SetVSCodeStatCheck(func(path string) bool {
		return true
	})
	defer actions.SetVSCodeStatCheck(nil)

	actions.SetVSCodeExecCommand(func(name string, arg ...string) *exec.Cmd {
		capturedBinary = name
		capturedArgs = arg
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	defer actions.SetVSCodeExecCommand(exec.Command)

	_ = actions.NewOpener().OpenInVSCode("C:\\project")

	// Should spawn Code.exe directly, not through cmd.exe.
	if !strings.HasSuffix(capturedBinary, "Code.exe") {
		t.Errorf("binary = %q, want path ending in Code.exe", capturedBinary)
	}
	if len(capturedArgs) != 1 || capturedArgs[0] != "C:\\project" {
		t.Errorf("args = %v, want [C:\\project]", capturedArgs)
	}
}

// TestOpenInVSCode_NothingFound verifies S1.4: returns informative error when all fail.
func TestOpenInVSCode_NothingFound(t *testing.T) {
	actions.SetLookPath(func(file string) (string, error) {
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	})
	defer actions.SetLookPath(exec.LookPath)

	actions.SetVSCodeStatCheck(func(path string) bool {
		return false
	})
	defer actions.SetVSCodeStatCheck(nil)

	op := actions.NewOpener()
	err := op.OpenInVSCode("C:\\project")
	if err == nil {
		t.Fatal("OpenInVSCode() expected error when all fallbacks fail, got nil")
	}
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// TestNewOpener_ImplementsWindowsOpenerSeamWithVSCode verifies type assertion includes new method.
func TestNewOpener_ImplementsWindowsOpenerSeamWithVSCode(t *testing.T) {
	op := actions.NewOpener()
	if _, ok := op.(actions.WindowsOpenerSeam); !ok {
		t.Error("NewOpener() should implement WindowsOpenerSeam")
	}
}
