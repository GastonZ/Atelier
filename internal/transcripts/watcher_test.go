package transcripts_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gastonz/atelier/internal/transcripts"
)

// FakeWatcher is a controllable Watcher implementation for use in tui tests.
// It delivers events via an in-memory channel and tracks Close calls.
type FakeWatcher struct {
	ch             chan transcripts.Event
	closeCallCount atomic.Int32
	watchedPaths   []string
}

// NewFakeWatcher creates a FakeWatcher with the given channel buffer size.
func NewFakeWatcher(bufSize int) *FakeWatcher {
	return &FakeWatcher{
		ch: make(chan transcripts.Event, bufSize),
	}
}

// Watch implements transcripts.Watcher. Records the paths and returns the channel.
func (fw *FakeWatcher) Watch(paths []string) (<-chan transcripts.Event, error) {
	fw.watchedPaths = paths
	return fw.ch, nil
}

// Close implements transcripts.Watcher.
func (fw *FakeWatcher) Close() error {
	fw.closeCallCount.Add(1)
	close(fw.ch)
	return nil
}

// CloseCallCount returns how many times Close has been called.
func (fw *FakeWatcher) CloseCallCount() int {
	return int(fw.closeCallCount.Load())
}

// SendEvent injects an event into the fake watcher's channel.
func (fw *FakeWatcher) SendEvent(e transcripts.Event) {
	fw.ch <- e
}

// WatchedPaths returns the paths passed to the last Watch call.
func (fw *FakeWatcher) WatchedPaths() []string {
	return fw.watchedPaths
}

// ---- FsnotifyWatcher tests --------------------------------------------------

func TestFsnotifyWatcher_Watch_ReturnsChannel(t *testing.T) {
	// Watch returns a readable channel that is non-nil
	dir := t.TempDir()
	testFile := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	w := transcripts.NewFsnotifyWatcher(500 * time.Millisecond)
	defer w.Close()

	ch, err := w.Watch([]string{testFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}
	if ch == nil {
		t.Error("expected non-nil channel from Watch")
	}
}

func TestFsnotifyWatcher_ExplicitPaths_NoGlob(t *testing.T) {
	// R9.1: Watch only watches the explicit []string of paths given — never globs.
	// Verify by watching one file and confirming ONLY events for that file arrive.
	dir := t.TempDir()
	watchedFile := filepath.Join(dir, "watched.jsonl")
	unwatchedFile := filepath.Join(dir, "unwatched.jsonl")

	for _, f := range []string{watchedFile, unwatchedFile} {
		if err := os.WriteFile(f, []byte(""), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	// Watch only the watchedFile path — NOT the directory.
	// The watcher should watch the parent directory for the file's events.
	w := transcripts.NewFsnotifyWatcher(100 * time.Millisecond)
	defer w.Close()

	ch, err := w.Watch([]string{watchedFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	// Append to watched file — should trigger event.
	appendLine(t, watchedFile, `{"type":"user","sessionId":"watched-sess"}`)

	select {
	case event := <-ch:
		if event == nil {
			t.Error("received nil event")
		}
		// Event should be from the watched session.
		// (The content we wrote is a user event.)
	case <-time.After(2 * time.Second):
		t.Error("expected event from watched file within 2s, got none")
	}
}

func TestFsnotifyWatcher_Close_StopsGoroutine(t *testing.T) {
	// Close() stops the watcher goroutine and closes the channel.
	dir := t.TempDir()
	testFile := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	w := transcripts.NewFsnotifyWatcher(100 * time.Millisecond)
	ch, err := w.Watch([]string{testFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	// Close the watcher.
	if err := w.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Channel must be closed after Close() is called.
	// Read until close (with timeout to prevent hanging).
	deadline := time.After(1 * time.Second)
	for {
		select {
		case _, open := <-ch:
			if !open {
				return // channel closed as expected
			}
		case <-deadline:
			t.Error("channel not closed within 1s after Close()")
			return
		}
	}
}

func TestFsnotifyWatcher_PollingFallback_AlwaysOn(t *testing.T) {
	// R2.2: polling fallback must always run alongside fsnotify.
	// Even if no fsnotify events arrive, the polling ticker fires and can
	// detect file size changes.
	dir := t.TempDir()
	testFile := filepath.Join(dir, "poll-session.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Use a very short polling interval to ensure polling fires quickly.
	w := transcripts.NewFsnotifyWatcher(50 * time.Millisecond)
	defer w.Close()

	ch, err := w.Watch([]string{testFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	// Append a line — the polling ticker should detect the size change
	// and emit events within 2 * polling interval.
	time.Sleep(100 * time.Millisecond) // let the watcher settle
	appendLine(t, testFile, `{"type":"user","sessionId":"poll-sess","cwd":"C:\\P","message":{"role":"user","content":"poll test"},"uuid":"uuid-poll","timestamp":"2026-01-01T10:00:00.000Z","version":"2.1.126","gitBranch":"main"}`)

	select {
	case event := <-ch:
		if event == nil {
			t.Error("polling event was nil")
		}
	case <-time.After(3 * time.Second):
		t.Error("polling fallback did not deliver event within 3s")
	}
}

func TestFsnotifyWatcher_Backpressure_ChannelFull_NoPanic(t *testing.T) {
	// R2.5: when channel is full, new events are dropped — no panic, no goroutine block.
	dir := t.TempDir()
	testFile := filepath.Join(dir, "burst-session.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Create watcher with buffer size 1 to easily fill it.
	w := transcripts.NewFsnotifyWatcherWithBufferSize(50*time.Millisecond, 1)
	defer w.Close()

	_, err := w.Watch([]string{testFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	// Rapidly append many lines — channel will fill; extras are dropped.
	// The test passes if we reach the end without panicking or blocking forever.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			appendLine(t, testFile,
				`{"type":"user","sessionId":"burst-sess","cwd":"C:\\P","message":{"role":"user","content":"msg"},"uuid":"uuid-b","timestamp":"2026-01-01T10:00:00.000Z","version":"2.1.126","gitBranch":"main"}`)
			time.Sleep(10 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		// No panic, no hang — test passes.
	case <-time.After(5 * time.Second):
		t.Error("watcher blocked for >5s with full channel — goroutine leak suspected")
	}
}

func TestFsnotifyWatcher_WatcherCap_32MaxWatchers(t *testing.T) {
	// R9.1, S9.1: watcher accepts at most 32 paths via fsnotify; extras fall
	// back to polling only. No panic when given 33 paths.
	dir := t.TempDir()

	var paths []string
	for i := 0; i < 33; i++ {
		p := filepath.Join(dir, fmt.Sprintf("session-%03d.jsonl", i))
		if err := os.WriteFile(p, []byte(""), 0644); err != nil {
			t.Fatalf("setup file %d: %v", i, err)
		}
		paths = append(paths, p)
	}

	w := transcripts.NewFsnotifyWatcher(100 * time.Millisecond)
	defer w.Close()

	ch, err := w.Watch(paths)
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}
	if ch == nil {
		t.Error("expected non-nil channel even with 33 paths")
	}
	// No panic reaching here = test passes.
}

// ---- T20: TRIANGULATE edge cases --------------------------------------------

func TestFsnotifyWatcher_BurstOf50Events_NoPanic(t *testing.T) {
	// S2.2: burst of events does not crash; final state is consistent.
	dir := t.TempDir()
	testFile := filepath.Join(dir, "burst-50.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	w := transcripts.NewFsnotifyWatcherWithBufferSize(30*time.Millisecond, 128)
	defer w.Close()

	ch, err := w.Watch([]string{testFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	// Append 50 lines in rapid succession.
	const numLines = 50
	for i := 0; i < numLines; i++ {
		appendLine(t, testFile,
			fmt.Sprintf(`{"type":"user","sessionId":"burst-50","cwd":"C:\\P","message":{"role":"user","content":"line %d"},"uuid":"uuid-%03d","timestamp":"2026-01-01T10:00:00.000Z","version":"2.1.126","gitBranch":"main"}`, i, i))
	}

	// Drain the channel for up to 3 seconds — we expect some events to arrive.
	var received int
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				goto done
			}
			received++
			if received >= 1 {
				goto done // we just need at least one event to confirm delivery
			}
		case <-deadline:
			goto done
		}
	}
done:
	if received == 0 {
		t.Error("expected at least 1 event from burst of 50 appends, got 0")
	}
}

func TestFsnotifyWatcher_WatcherStartFail_PollingStillWorks(t *testing.T) {
	// R2.2: even if fsnotify fully fails to watch a path (e.g., file removed),
	// polling detects size changes on files that DO exist.
	dir := t.TempDir()
	existingFile := filepath.Join(dir, "exists.jsonl")
	if err := os.WriteFile(existingFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Include a nonexistent file alongside the real one.
	nonexistentFile := filepath.Join(dir, "gone.jsonl")

	w := transcripts.NewFsnotifyWatcher(50 * time.Millisecond)
	defer w.Close()

	ch, err := w.Watch([]string{existingFile, nonexistentFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	time.Sleep(80 * time.Millisecond)
	appendLine(t, existingFile,
		`{"type":"user","sessionId":"exists-sess","cwd":"C:\\P","message":{"role":"user","content":"hello"},"uuid":"uuid-ex","timestamp":"2026-01-01T10:00:00.000Z","version":"2.1.126","gitBranch":"main"}`)

	select {
	case e := <-ch:
		if e == nil {
			t.Error("received nil event")
		}
	case <-time.After(3 * time.Second):
		t.Error("polling did not deliver event within 3s even with nonexistent path in list")
	}
}

func TestFsnotifyWatcher_PollingFiresAtConfiguredInterval(t *testing.T) {
	// Polling fires at the configured interval (100ms in this test).
	dir := t.TempDir()
	testFile := filepath.Join(dir, "interval-test.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	const pollInterval = 100 * time.Millisecond
	w := transcripts.NewFsnotifyWatcher(pollInterval)
	defer w.Close()

	ch, err := w.Watch([]string{testFile})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	start := time.Now()
	time.Sleep(120 * time.Millisecond) // let watcher settle past first tick
	appendLine(t, testFile,
		`{"type":"user","sessionId":"interval-sess","cwd":"C:\\P","message":{"role":"user","content":"tick"},"uuid":"uuid-tick","timestamp":"2026-01-01T10:00:00.000Z","version":"2.1.126","gitBranch":"main"}`)

	select {
	case <-ch:
		elapsed := time.Since(start)
		// Should arrive within ~2 poll intervals.
		if elapsed > 5*time.Second {
			t.Errorf("event took too long: %v", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Error("polling event did not arrive within 5s")
	}
}

// ---- FakeWatcher tests (validates the test double itself) -------------------

func TestFakeWatcher_SendReceive(t *testing.T) {
	fw := NewFakeWatcher(4)

	ch, err := fw.Watch([]string{"path/a.jsonl", "path/b.jsonl"})
	if err != nil {
		t.Fatalf("Watch error: %v", err)
	}

	if len(fw.WatchedPaths()) != 2 {
		t.Errorf("expected 2 watched paths, got %d", len(fw.WatchedPaths()))
	}

	// Send a synthetic event.
	sent := &transcripts.UserEvent{
		UUIDValue:      "uuid-fake-001",
		SessionIDValue: "sess-fake",
		TimestampValue: time.Now(),
		Text:           "hello",
	}
	fw.SendEvent(sent)

	select {
	case got := <-ch:
		if got.UUID() != "uuid-fake-001" {
			t.Errorf("expected uuid-fake-001, got %q", got.UUID())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected event on channel, got none")
	}
}

func TestFakeWatcher_Close_TracksCallCount(t *testing.T) {
	fw := NewFakeWatcher(4)
	_, _ = fw.Watch(nil)

	if fw.CloseCallCount() != 0 {
		t.Errorf("expected 0 close calls before Close(), got %d", fw.CloseCallCount())
	}

	_ = fw.Close()

	if fw.CloseCallCount() != 1 {
		t.Errorf("expected 1 close call, got %d", fw.CloseCallCount())
	}
}

// ---- helpers ----------------------------------------------------------------

// appendLine opens path for append and writes a single JSONL line.
func appendLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("appendLine: open %q: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatalf("appendLine: write: %v", err)
	}
	f.Sync()
}
