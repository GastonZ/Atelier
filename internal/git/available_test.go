package git

import (
	"os/exec"
	"testing"
)

// TestIsAvailable_TrueWhenGitOnPath verifies IsAvailable returns true when git is found.
func TestIsAvailable_TrueWhenGitOnPath(t *testing.T) {
	// Reset cache for a clean test.
	resetAvailableCache()

	// Swap lookPath seam to simulate git being available.
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "git" {
			return "/usr/bin/git", nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
	defer func() {
		lookPath = origLookPath
		resetAvailableCache()
	}()

	got := IsAvailable()
	if !got {
		t.Error("IsAvailable() = false, want true when git is on PATH")
	}
}

// TestIsAvailable_FalseWhenGitMissing verifies IsAvailable returns false when git is not found.
func TestIsAvailable_FalseWhenGitMissing(t *testing.T) {
	resetAvailableCache()

	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
	defer func() {
		lookPath = origLookPath
		resetAvailableCache()
	}()

	got := IsAvailable()
	if got {
		t.Error("IsAvailable() = true, want false when git is not on PATH")
	}
}

// TestIsAvailable_CachedAfterFirstCall verifies the result is memoized.
func TestIsAvailable_CachedAfterFirstCall(t *testing.T) {
	resetAvailableCache()

	callCount := 0
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		callCount++
		return "/usr/bin/git", nil
	}
	defer func() {
		lookPath = origLookPath
		resetAvailableCache()
	}()

	// Call twice.
	IsAvailable()
	IsAvailable()

	if callCount != 1 {
		t.Errorf("lookPath called %d times, want exactly 1 (cached)", callCount)
	}
}
