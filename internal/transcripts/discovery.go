package transcripts

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session represents one Claude Code session file (root or sub-agent).
type Session struct {
	ID            string    // session UUID (derived from filename without .jsonl)
	RootPath      string    // absolute path to the .jsonl file
	Cwd           string    // last seen cwd from parsed events (authoritative project path)
	ProjectID     string    // registry.Project.ID; "" if unmatched
	ProjectName   string    // registry.Project.Name or "" if unmatched
	LastEventTime time.Time // mtime of the .jsonl file (refined by latest event ts)
	AccumulatedUSD float64
	Events        []Event   // empty from ListActive/ListAll; populated by LoadEvents
	SubSessions   []Session // direct children under {id}/subagents/
}

// String returns a human-readable representation useful for debug output.
func (s Session) String() string {
	return fmt.Sprintf("Session{ID:%s, Cwd:%s, LastEvent:%s, SubSessions:%d}",
		s.ID, s.Cwd, s.LastEventTime.Format(time.RFC3339), len(s.SubSessions))
}

// Clock is a seam for time.Now, enabling deterministic tests.
type Clock interface {
	Now() time.Time
}

// realClock wraps time.Now for production use.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// RealClock is the exported production Clock implementation.
// Use RealClock{} when constructing a Scanner or any other dependency that
// needs a real wall-clock (i.e. in main.go / composition root).
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// Scanner discovers and loads Claude Code transcript sessions from the
// ~/.claude/projects/ directory hierarchy.
type Scanner interface {
	// ListActive returns root sessions whose .jsonl mtime is within window of
	// Clock.Now(). Each active root session includes its sub-sessions.
	ListActive(window time.Duration) ([]Session, error)

	// ListAll returns ALL root sessions regardless of mtime. Sub-sessions are
	// included for each root.
	ListAll() ([]Session, error)

	// LoadEvents reads the .jsonl for the given sessionID and returns all
	// non-skipped parsed events. sessionID must match a session discoverable
	// under the root.
	LoadEvents(sessionID string) ([]Event, error)
}

// fileScanner is the production Scanner implementation.
type fileScanner struct {
	root   string // absolute path to the projects root (e.g. ~/.claude/projects/)
	clock  Clock
	prices PriceTable // optional: when non-nil, discovery backfills AccumulatedUSD
}

// NewFileScanner returns a Scanner rooted at root using the provided clock.
// In production pass the result of ClaudeProjectsDir() for root, RealClock{},
// and DefaultPriceTable() (or an override) for prices. A nil prices skips cost
// backfill — sessions return with AccumulatedUSD=0 and rely on live events.
func NewFileScanner(root string, clock Clock, prices PriceTable) Scanner {
	return &fileScanner{root: root, clock: clock, prices: prices}
}

// ListActive discovers all root sessions whose .jsonl mtime falls within
// [now-window, now] and returns them with sub-sessions populated.
func (s *fileScanner) ListActive(window time.Duration) ([]Session, error) {
	all, err := s.discover()
	if err != nil {
		return nil, err
	}

	cutoff := s.clock.Now().Add(-window)
	var active []Session
	for _, sess := range all {
		info, err := os.Stat(sess.RootPath)
		if err != nil {
			continue // file disappeared between discovery and stat; skip
		}
		if !info.ModTime().Before(cutoff) {
			active = append(active, sess)
		}
	}
	if active == nil {
		active = []Session{}
	}
	return active, nil
}

// ListAll discovers all root sessions regardless of mtime.
func (s *fileScanner) ListAll() ([]Session, error) {
	sessions, err := s.discover()
	if err != nil {
		return nil, err
	}
	if sessions == nil {
		return []Session{}, nil
	}
	return sessions, nil
}

// LoadEvents reads and parses the .jsonl for the given sessionID.
// It searches for the file under the root directory tree.
func (s *fileScanner) LoadEvents(sessionID string) ([]Event, error) {
	path, err := s.findSessionPath(sessionID)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("transcripts: LoadEvents: open %q: %w", path, err)
	}
	defer f.Close()

	var events []Event
	if err := ParseStream(f, func(e Event) {
		events = append(events, e)
	}); err != nil {
		return nil, fmt.Errorf("transcripts: LoadEvents: parse %q: %w", path, err)
	}
	return events, nil
}

// ---------------------------------------------------------------------------
// Internal discovery helpers
// ---------------------------------------------------------------------------

// discover walks the root and returns all root sessions with their sub-sessions.
// If the root directory does not exist, returns (nil, error) — caller decides
// how to surface (S1.5).
func (s *fileScanner) discover() ([]Session, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("transcripts: scanner: projects directory not found: %q: %w", s.root, err)
		}
		return nil, fmt.Errorf("transcripts: scanner: readdir %q: %w", s.root, err)
	}

	var sessions []Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(s.root, entry.Name())
		rootSessions, err := s.discoverProjectDir(projectDir)
		if err != nil {
			// Best-effort: skip unreadable project directories
			continue
		}
		sessions = append(sessions, rootSessions...)
	}
	return sessions, nil
}

// discoverProjectDir finds all root .jsonl files in a project directory and
// populates their sub-sessions.
func (s *fileScanner) discoverProjectDir(projectDir string) ([]Session, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, e := range entries {
		if e.IsDir() {
			continue // sub-agent dirs are handled below via the root session
		}
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(e.Name(), ".jsonl")
		sessionPath := filepath.Join(projectDir, e.Name())

		sess := Session{
			ID:       sessionID,
			RootPath: sessionPath,
		}

		// Populate mtime.
		info, err := e.Info()
		if err == nil {
			sess.LastEventTime = info.ModTime()
		}

		// Populate cwd + cost + last-event time in one file pass.
		cwd, cost, lastEvent := s.extractMetadata(sessionPath)
		sess.Cwd = cwd
		sess.AccumulatedUSD = cost
		if !lastEvent.IsZero() && lastEvent.After(sess.LastEventTime) {
			sess.LastEventTime = lastEvent
		}

		// Discover sub-agents.
		subAgentDir := filepath.Join(projectDir, sessionID, "subagents")
		subs, _ := s.discoverSubAgents(subAgentDir, sessionID)
		sess.SubSessions = subs

		sessions = append(sessions, sess)
	}
	return sessions, nil
}

// discoverSubAgents returns sub-sessions from the subagents/ directory of a
// root session. Missing directory is silently ignored (not all sessions have
// sub-agents).
func (s *fileScanner) discoverSubAgents(subAgentDir, parentID string) ([]Session, error) {
	entries, err := os.ReadDir(subAgentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var subs []Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		subPath := filepath.Join(subAgentDir, e.Name())
		subID := parentID + "/subagents/" + strings.TrimSuffix(e.Name(), ".jsonl")

		sub := Session{
			ID:       subID,
			RootPath: subPath,
		}
		if info, err := e.Info(); err == nil {
			sub.LastEventTime = info.ModTime()
		}
		cwd, cost, lastEvent := s.extractMetadata(subPath)
		sub.Cwd = cwd
		sub.AccumulatedUSD = cost
		if !lastEvent.IsZero() && lastEvent.After(sub.LastEventTime) {
			sub.LastEventTime = lastEvent
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// extractMetadata opens the .jsonl ONCE and returns:
//   - cwd: from the first UserEvent with a non-empty Cwd field
//   - cost: total USD across all AssistantEvents (uses fileScanner.prices; zero if prices is nil)
//   - lastEvent: latest non-zero event timestamp seen (refines mtime when events are newer)
//
// Single-pass scan keeps discovery O(file size) per session — important when
// every active session triggers a refresh on each polling tick.
func (s *fileScanner) extractMetadata(path string) (cwd string, cost float64, lastEvent time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, time.Time{}
	}
	defer f.Close()

	_ = ParseStream(f, func(e Event) {
		if cwd == "" {
			if ue, ok := e.(*UserEvent); ok && ue.Cwd != "" {
				cwd = ue.Cwd
			}
		}
		if t := e.Timestamp(); !t.IsZero() && t.After(lastEvent) {
			lastEvent = t
		}
		if s.prices != nil {
			if ae, ok := e.(*AssistantEvent); ok {
				c, _ := s.prices.Cost(ae.Model,
					ae.Usage.InputTokens,
					ae.Usage.OutputTokens,
					ae.Usage.CacheCreationTokens,
					ae.Usage.CacheReadTokens)
				cost += c
			}
		}
	})
	return cwd, cost, lastEvent
}

// findSessionPath locates the .jsonl file for a given session ID by walking
// the root directory tree.
func (s *fileScanner) findSessionPath(sessionID string) (string, error) {
	var found string
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			candidate := strings.TrimSuffix(d.Name(), ".jsonl")
			if candidate == sessionID {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("transcripts: session %q not found under %q", sessionID, s.root)
	}
	return found, nil
}

// ---------------------------------------------------------------------------
// Production path helper (unused by tests, available for main)
// ---------------------------------------------------------------------------

// ClaudeProjectsDir returns the platform-appropriate path to ~/.claude/projects/.
func ClaudeProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("transcripts: home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "projects"), nil
}
