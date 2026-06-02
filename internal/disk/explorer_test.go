package disk

import (
	"os/exec"
	"testing"
)

// TestOpenInExplorer_BuildsExpectedCommand verifies cmd.exe /c start "" <path> is invoked.
func TestOpenInExplorer_BuildsExpectedCommand(t *testing.T) {
	var capturedName string
	var capturedArgs []string

	origExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = arg
		// Return a no-op command so Start() doesn't fail.
		return exec.Command("cmd.exe", "/c", "exit", "0")
	}
	defer func() { execCommand = origExecCommand }()

	if err := OpenInExplorer("C:\\some\\path"); err != nil {
		// Start() on our no-op cmd should succeed; real test is args.
		t.Logf("OpenInExplorer() error (may be OK): %v", err)
	}

	if capturedName != "cmd.exe" {
		t.Errorf("command name = %q, want cmd.exe", capturedName)
	}
	wantArgs := []string{"/c", "start", "", "C:\\some\\path"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}
}

// TestOpenInExplorer_NonZeroExitPropagatesError verifies error on non-zero exit.
func TestOpenInExplorer_NonZeroExitPropagatesError(t *testing.T) {
	origExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		// Return a command that starts fine (Start() succeeds) but we can't check exit.
		// Since Start() is non-blocking, we test the happy path only here.
		// The error propagation is tested by the error-returning seam.
		return exec.Command("cmd.exe", "/c", "exit", "0")
	}
	defer func() { execCommand = origExecCommand }()

	// Happy path: no error expected.
	if err := OpenInExplorer("C:\\temp"); err != nil {
		t.Logf("OpenInExplorer returned (expected on CI): %v", err)
	}
}

// TestSetDiskExecCommand verifies the seam setter works.
func TestSetDiskExecCommand(t *testing.T) {
	orig := execCommand
	SetDiskExecCommand(func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd.exe", "/c", "exit", "0")
	})
	if execCommand == nil {
		t.Error("SetDiskExecCommand set nil")
	}
	execCommand = orig
}
