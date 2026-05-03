package actions

import "os/exec"

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

// OpenInClaudeCode spawns the claude CLI in a new console at projectPath, detached.
// Uses cmd.exe /c start "" claude so atelier's process tree does not own the new window.
func (w *windowsOpener) OpenInClaudeCode(projectPath string) error {
	cmd := execCommand("cmd.exe", "/c", "start", "", "claude")
	cmd.Dir = projectPath
	return cmd.Start()
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

// WindowsOpenerSeam is exported solely for testing so the external actions_test package
// can swap execCommand and call the methods via type assertion.
// Do NOT use this interface in production code — use Opener instead.
type WindowsOpenerSeam interface {
	OpenInClaudeCodeViaSeam(projectPath string) error
	SpawnPowerShellViaSeam(projectPath string) error
}
