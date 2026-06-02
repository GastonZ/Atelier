package disk

import "os/exec"

// execCommand is the testability seam for os/exec.Command.
// Tests swap via SetDiskExecCommand to capture arguments without spawning real processes.
var execCommand = exec.Command

// SetDiskExecCommand replaces the exec.Command seam used by OpenInExplorer.
// Call defer SetDiskExecCommand(exec.Command) in tests to restore.
func SetDiskExecCommand(fn func(name string, arg ...string) *exec.Cmd) {
	execCommand = fn
}

// OpenInExplorer opens Windows Explorer at the given path in a detached process.
// Uses cmd.exe /c start "" <path> — same pattern as internal/actions/windows.go.
// Non-blocking: calls Start() not Run().
func OpenInExplorer(path string) error {
	cmd := execCommand("cmd.exe", "/c", "start", "", path)
	return cmd.Start()
}
