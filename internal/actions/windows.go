//go:build windows

package actions

import "os/exec"

// newPlatformOpener returns the Windows-backed Opener. Selected at compile time
// via the windows build constraint; see opener_other.go for the non-Windows stub.
func newPlatformOpener() Opener {
	return &windowsOpener{}
}

// execCommand is the testability seam for os/exec.Command.
// Tests swap it via SetExecCommand to capture arguments without spawning real processes.
var execCommand = exec.Command

// SetExecCommand replaces the exec.Command function used by windowsOpener.
// Call defer SetExecCommand(exec.Command) in tests to restore the original.
func SetExecCommand(fn func(name string, arg ...string) *exec.Cmd) {
	execCommand = fn
}

// windowsOpener implements Opener using cmd.exe /c start for detached process launch.
type windowsOpener struct{}

// LaunchInDir spawns command (with args) in a new console at projectPath, detached.
// Uses cmd.exe /c start "" <command> [args...] so atelier's process tree does not
// own the new window. This is the generic primitive behind every agent launcher.
func (w *windowsOpener) LaunchInDir(projectPath, command string, args ...string) error {
	cmdArgs := append([]string{"/c", "start", "", command}, args...)
	cmd := execCommand("cmd.exe", cmdArgs...)
	cmd.Dir = projectPath
	return cmd.Start()
}

// OpenInClaudeCode launches the claude CLI at projectPath. Kept for the Opener
// contract and existing callers; delegates to the generic LaunchInDir.
func (w *windowsOpener) OpenInClaudeCode(projectPath string) error {
	return w.LaunchInDir(projectPath, "claude")
}

// SpawnPowerShell opens a new PowerShell window at projectPath, detached.
func (w *windowsOpener) SpawnPowerShell(projectPath string) error {
	cmd := execCommand("cmd.exe", "/c", "start", "", "powershell.exe", "-NoLogo", "-NoExit")
	cmd.Dir = projectPath
	return cmd.Start()
}

// OpenInClaudeCodeViaSeam is the testability entry point for OpenInClaudeCode.
// Exposed so external tests can call it after type-asserting to WindowsOpenerSeam.
func (w *windowsOpener) OpenInClaudeCodeViaSeam(projectPath string) error {
	return w.OpenInClaudeCode(projectPath)
}

// SpawnPowerShellViaSeam is the testability entry point for SpawnPowerShell.
func (w *windowsOpener) SpawnPowerShellViaSeam(projectPath string) error {
	return w.SpawnPowerShell(projectPath)
}

// OpenInVSCode opens the project in VS Code. Delegates to the vscode.go detection chain.
func (w *windowsOpener) OpenInVSCode(projectPath string) error {
	return openInVSCodeImpl(projectPath)
}

// OpenInVSCodeViaSeam is the testability entry point for OpenInVSCode.
func (w *windowsOpener) OpenInVSCodeViaSeam(projectPath string) error {
	return w.OpenInVSCode(projectPath)
}

// WindowsOpenerSeam is exported solely for testing so the external actions_test package
// can swap execCommand and call the methods via type assertion.
// Do NOT use this interface in production code — use Opener instead.
type WindowsOpenerSeam interface {
	OpenInClaudeCodeViaSeam(projectPath string) error
	SpawnPowerShellViaSeam(projectPath string) error
	OpenInVSCodeViaSeam(projectPath string) error // NEW
}
