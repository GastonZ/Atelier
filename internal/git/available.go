// Package git provides wrappers over the git CLI via os/exec.
// Each public function uses its own execCommand / lookPath seam
// so tests can inject canned output without spawning real processes.
package git

import (
	"os/exec"
	"sync"
)

// lookPath is the testability seam for exec.LookPath.
// Tests swap it to control git binary availability.
var lookPath = exec.LookPath

// availableOnce ensures the availability check runs at most once per process.
var availableOnce sync.Once

// availableResult caches the result of the first IsAvailable call.
var availableResult bool

// IsAvailable returns true iff the git binary is on PATH.
// The result is cached after the first call — subsequent calls are O(1).
// To reset the cache (tests only), call resetAvailableCache().
func IsAvailable() bool {
	availableOnce.Do(func() {
		_, err := lookPath("git")
		availableResult = err == nil
	})
	return availableResult
}

// resetAvailableCache resets the sync.Once and cached result.
// ONLY for use in tests — not exported to production callers.
func resetAvailableCache() {
	availableOnce = sync.Once{}
	availableResult = false
}

// execCommand is the testability seam for os/exec.Command.
// Tests swap this via SetExecCommand to capture args without spawning real processes.
// This seam is SEPARATE from internal/actions/windows.go by design — per-package independence.
var execCommand = exec.Command

// SetExecCommand replaces the exec.Command seam used by git status/log/show readers.
// Call defer SetExecCommand(exec.Command) in tests to restore.
func SetExecCommand(fn func(name string, arg ...string) *exec.Cmd) {
	execCommand = fn
}
