package transcripts

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// maxFsnotifyWatchers is the hard cap on concurrent fsnotify watchers (R9.1).
// Paths beyond this limit are monitored by polling only.
const maxFsnotifyWatchers = 32

// defaultChannelBufferSize is the default event channel buffer (R2.3, R2.5).
const defaultChannelBufferSize = 64

// Watcher delivers live-tail events for an explicit set of .jsonl paths.
// The primary mechanism is fsnotify; an always-on polling fallback ensures
// events are not missed when fsnotify fires unreliably (R2.2).
type Watcher interface {
	// Watch starts monitoring the given paths and returns a channel of events.
	// The caller must call Close() when done. Each path should be a .jsonl file.
	Watch(sessionPaths []string) (<-chan Event, error)

	// Close stops all goroutines and closes the event channel.
	Close() error
}

// fsnotifyWatcher is the production Watcher implementation.
//
// Architecture:
//   - fsnotify watches the PARENT DIRECTORIES of each tracked file
//     (ReadDirectoryChangesW on Windows requires directory-level watching).
//   - A polling goroutine runs concurrently (always-on, R2.2) and detects
//     size changes via periodic stat calls.
//   - Both fsnotify events and polling detections are translated into Event
//     values via ParseStream of the new bytes since last known offset.
//   - The output channel is buffered; full channel → new events are dropped
//     (R2.5), goroutine never blocks.
type fsnotifyWatcher struct {
	pollInterval time.Duration
	bufferSize   int

	mu        sync.Mutex
	fsnWatcher *fsnotify.Watcher
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	ch         chan Event

	// fileState tracks last seen size for each watched path (for polling diff).
	fileState map[string]int64
	// watchedPaths is the explicit set passed to Watch (R9.1).
	watchedPaths []string
}

// NewFsnotifyWatcher returns a Watcher with the given polling interval and
// the default channel buffer size.
func NewFsnotifyWatcher(pollInterval time.Duration) Watcher {
	return NewFsnotifyWatcherWithBufferSize(pollInterval, defaultChannelBufferSize)
}

// NewFsnotifyWatcherWithBufferSize returns a Watcher with a configurable
// channel buffer. Useful in tests to easily fill the channel.
func NewFsnotifyWatcherWithBufferSize(pollInterval time.Duration, bufSize int) Watcher {
	return &fsnotifyWatcher{
		pollInterval: pollInterval,
		bufferSize:   bufSize,
	}
}

// Watch starts watching the given explicit set of paths.
// Only the first maxFsnotifyWatchers paths get fsnotify coverage; the rest
// are monitored via polling only (R9.1, S9.1).
func (w *fsnotifyWatcher) Watch(sessionPaths []string) (<-chan Event, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan Event, w.bufferSize)
	w.ch = ch
	w.watchedPaths = sessionPaths
	w.fileState = make(map[string]int64, len(sessionPaths))

	// Record initial file sizes.
	for _, p := range sessionPaths {
		info, err := os.Stat(p)
		if err == nil {
			w.fileState[p] = info.Size()
		}
	}

	// Create fsnotify watcher.
	fsnW, err := fsnotify.NewWatcher()
	if err != nil {
		// If fsnotify fails to init, fall back to polling only (R2.2).
		fsnW = nil
	}
	w.fsnWatcher = fsnW

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	// Register directories with fsnotify (cap at maxFsnotifyWatchers).
	if fsnW != nil {
		registered := 0
		seenDirs := make(map[string]bool)
		for _, p := range sessionPaths {
			if registered >= maxFsnotifyWatchers {
				break
			}
			dir := filepath.Dir(p)
			if !seenDirs[dir] {
				seenDirs[dir] = true
				_ = fsnW.Add(dir) // best-effort; errors are non-fatal
				registered++
			}
		}
	}

	// Start the goroutines.
	w.wg.Add(2)

	go w.fsnotifyLoop(ctx)
	go w.pollingLoop(ctx)

	return ch, nil
}

// Close stops all goroutines, closes the fsnotify watcher, and closes the
// event channel. It is safe to call multiple times (idempotent via cancel).
func (w *fsnotifyWatcher) Close() error {
	w.mu.Lock()
	cancel := w.cancel
	fsnW := w.fsnWatcher
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	// Wait for goroutines to exit before closing the channel.
	w.wg.Wait()

	if fsnW != nil {
		fsnW.Close()
	}

	w.mu.Lock()
	if w.ch != nil {
		close(w.ch)
		w.ch = nil
	}
	w.mu.Unlock()

	return nil
}

// ---------------------------------------------------------------------------
// Internal goroutines
// ---------------------------------------------------------------------------

// fsnotifyLoop listens for fsnotify events and translates file-write events
// into Event values sent to the output channel.
func (w *fsnotifyWatcher) fsnotifyLoop(ctx context.Context) {
	defer w.wg.Done()

	if w.fsnWatcher == nil {
		return // fsnotify unavailable; polling handles everything
	}

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.fsnWatcher.Events:
			if !ok {
				return
			}
			// Only handle WRITE and CREATE events for tracked .jsonl files.
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if isTracked(w.watchedPaths, event.Name) {
					w.drainNewBytes(event.Name)
				}
			}

		case _, ok := <-w.fsnWatcher.Errors:
			if !ok {
				return
			}
			// Watcher error — polling fallback continues regardless.
		}
	}
}

// pollingLoop periodically stats all watched paths and emits events for any
// files whose size has grown since the last poll.
// This is the always-on fallback (R2.2) — it runs even when fsnotify is healthy.
func (w *fsnotifyWatcher) pollingLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.mu.Lock()
			paths := make([]string, len(w.watchedPaths))
			copy(paths, w.watchedPaths)
			w.mu.Unlock()

			for _, p := range paths {
				info, err := os.Stat(p)
				if err != nil {
					continue
				}
				w.mu.Lock()
				lastSize := w.fileState[p]
				w.mu.Unlock()

				if info.Size() > lastSize {
					w.drainNewBytes(p)
				}
			}
		}
	}
}

// drainNewBytes reads new content from the file since the last known offset,
// parses events, and sends non-nil events to the output channel.
// Full channel → new events are dropped (R2.5).
func (w *fsnotifyWatcher) drainNewBytes(path string) {
	w.mu.Lock()
	lastSize := w.fileState[path]
	w.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return
	}
	currentSize := info.Size()
	if currentSize <= lastSize {
		return
	}

	// Seek to the position after the last known byte.
	if lastSize > 0 {
		if _, err := f.Seek(lastSize, 0); err != nil {
			return
		}
	}

	// Update last known size before parsing (prevents duplicate sends on
	// concurrent fsnotify + polling triggers).
	w.mu.Lock()
	w.fileState[path] = currentSize
	w.mu.Unlock()

	// Parse new events and send them to the output channel.
	w.mu.Lock()
	ch := w.ch
	w.mu.Unlock()

	if ch == nil {
		return
	}

	_ = ParseStream(f, func(e Event) {
		sendOrDrop(ch, e)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isTracked returns true if path (normalized) matches one of the watchedPaths.
// Uses filepath.Clean for normalization.
func isTracked(watchedPaths []string, path string) bool {
	clean := filepath.Clean(path)
	for _, p := range watchedPaths {
		if filepath.Clean(p) == clean {
			return true
		}
	}
	return false
}

// sendOrDrop sends event to ch if there is capacity; otherwise drops it.
// Never blocks (R2.5).
func sendOrDrop(ch chan Event, e Event) {
	select {
	case ch <- e:
	default:
		// Channel full — drop the event.
	}
}
