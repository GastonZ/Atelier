package tui

// daily_helpers.go — helper types and functions for the daily-driver-pack screens.
// Bubbles list items, viewport constructors, and utility functions.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/gastonz/atelier/internal/engram"
)

// ============================================================================
// Memory Browser list item
// ============================================================================

// memoryItem wraps an engram.Observation to satisfy the list.Item interface.
type memoryItem struct {
	obs engram.Observation
}

// FilterValue returns the search string for bubbles/list filter.
func (i memoryItem) FilterValue() string {
	return i.obs.Title + " " + memoryPreview(i.obs)
}

// Title returns the display title line for the list item.
func (i memoryItem) Title() string {
	date := FormatHistoryDate(i.obs.Timestamp)
	return fmt.Sprintf("%s  [%s]  %s", date, i.obs.Type, i.obs.Title)
}

// Description returns the preview line for the list item.
func (i memoryItem) Description() string {
	return memoryPreview(i.obs)
}

// memoryPreview returns the first line of content capped at 100 runes,
// with fallback to Title if the first line is blank (design §10 locked rule).
func memoryPreview(obs engram.Observation) string {
	if obs.Content == "" {
		return obs.Title
	}
	lines := strings.SplitN(obs.Content, "\n", 2)
	first := strings.TrimSpace(lines[0])
	if first == "" {
		return obs.Title
	}
	// Cap at 100 runes (UTF-8 safe).
	runes := []rune(first)
	if len(runes) > 100 {
		return string(runes[:100])
	}
	return first
}

// memoryObsToItems converts a slice of observations to list.Item slice.
func memoryObsToItems(obs []engram.Observation) []list.Item {
	items := make([]list.Item, len(obs))
	for i, o := range obs {
		items[i] = memoryItem{obs: o}
	}
	return items
}

// newMemoryList constructs a bubbles/list.Model for ScreenMemoryBrowser with filtering enabled.
func newMemoryList(width, height int, items []list.Item) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	// Neuter quit to prevent hijack.
	l.KeyMap.Quit = key.NewBinding()
	l.KeyMap.ForceQuit = key.NewBinding()
	return l
}

// MemoryPreviewForTest exposes memoryPreview for testing (package-level white-box).
func MemoryPreviewForTest(obs engram.Observation) string { return memoryPreview(obs) }

// ============================================================================
// History list item
// ============================================================================

// historyItem wraps a HistoryEntry to satisfy the list.Item interface.
type historyItem struct {
	entry HistoryEntry
}

// FilterValue returns the search string.
func (i historyItem) FilterValue() string {
	return i.entry.Title
}

// Title returns the display line: "[git] 2026-05-04  subject" or "[sdd] 2026-05-04  title".
func (i historyItem) Title() string {
	marker := CopyHistoryGitMarker
	if i.entry.Source == "sdd" {
		marker = CopyHistorySDDMarker
	}
	return fmt.Sprintf("%s  %s  %s", marker, FormatHistoryDate(i.entry.Date), i.entry.Title)
}

// Description returns the source ref.
func (i historyItem) Description() string { return i.entry.Ref }

// historyEntriesToItems converts HistoryEntry slice to list.Item slice.
func historyEntriesToItems(entries []HistoryEntry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = historyItem{entry: e}
	}
	return items
}

// newHistoryList constructs a bubbles/list.Model for ScreenProjectHistory.
func newHistoryList(width, height int, items []list.Item) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.KeyMap.Quit = key.NewBinding()
	l.KeyMap.ForceQuit = key.NewBinding()
	return l
}

// ============================================================================
// Viewport constructor
// ============================================================================

// newDetailViewport creates a viewport sized for detail views.
func newDetailViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height-4)
	return vp
}

// ============================================================================
// ScreenDiskUsage helpers
// ============================================================================

// diskRowPath returns the filesystem path for the currently selected disk row.
// Row 0 = engram dir; row 1 = claude projects dir; row 2+ = per-tomo dirs.
func (m Model) diskRowPath() string {
	switch m.diskCursor {
	case 0:
		// ~/.engram/
		home := userHomeDirOrEmpty()
		if home == "" {
			return ""
		}
		return home + "/.engram"
	case 1:
		// ~/.claude/projects/
		home := userHomeDirOrEmpty()
		if home == "" {
			return ""
		}
		return home + "/.claude/projects"
	default:
		// Per-tomo dir (cursor 2 = first tomo, etc.)
		idx := m.diskCursor - 2
		if idx < 0 || idx >= len(m.projects) {
			return ""
		}
		proj := m.projects[idx]
		path, _ := tomoClaudePath(proj.Path)
		return path
	}
}

// tomoClaudePath returns the ~/.claude/projects/<key>/ path for a tomo path.
// Heuristic mirrors ClaudeProjectsDirForPath in internal/disk.
func tomoClaudePath(tomoPath string) (string, error) {
	home := userHomeDirOrEmpty()
	if home == "" {
		return "", fmt.Errorf("home dir unknown")
	}
	key := strings.ToLower(tomoPath)
	key = strings.ReplaceAll(key, "\\", "-")
	key = strings.ReplaceAll(key, "/", "-")
	key = strings.ReplaceAll(key, ":", "-")
	return home + "/.claude/projects/" + key, nil
}

// userHomeDirOrEmpty returns os.UserHomeDir() or empty string on error.
// Used by diskRowPath to resolve engram and claude-projects dirs for the Explorer action.
func userHomeDirOrEmpty() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// ============================================================================
// parseHistoryRef converts a string ref (e.g., "42") to int64.
// Used when opening SDD archive details.
// ============================================================================

func parseHistoryRef(ref string) (int64, error) {
	return strconv.ParseInt(ref, 10, 64)
}

// ============================================================================
// MemoryPreview rune-cap test helper
// ============================================================================

// MemoryPreviewRuneCapTest verifies the 100-rune cap is applied correctly.
// Used only in tests to confirm UTF-8 safety.
func MemoryPreviewRuneCapTest(s string) string {
	runes := []rune(s)
	if utf8.RuneCountInString(s) > 100 {
		return string(runes[:100])
	}
	return s
}
