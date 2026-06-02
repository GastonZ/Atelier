// Package engram provides a read-only SQLite client over ~/.engram/engram.db.
// It exposes the Observation type and a Client interface for TUI consumption.
// No write operations (delete, prune, save) — read-only v1 scope.
package engram

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Observation mirrors a row from the engram.db observations table.
// Fields are mapped by column NAME (not position) for schema-drift safety.
// Timestamp maps to the created_at column.
type Observation struct {
	ID        int64
	Project   string
	Scope     string
	Type      string
	Title     string
	Content   string
	TopicKey  string
	Timestamp time.Time // created_at
}

// PreviewContent returns the preview text for the list entry.
// Rule (design §10): take the first line of Content, cap at 100 runes.
// If the first line is blank (empty or whitespace-only), fall back to Title.
func PreviewContent(obs Observation) string {
	if obs.Content == "" {
		return obs.Title
	}

	// Take first line only.
	firstLine := obs.Content
	if idx := strings.Index(obs.Content, "\n"); idx >= 0 {
		firstLine = obs.Content[:idx]
	}

	// Blank first line → fallback to title.
	if strings.TrimSpace(firstLine) == "" {
		return obs.Title
	}

	// Cap at 100 runes (UTF-8-safe).
	if utf8.RuneCountInString(firstLine) <= 100 {
		return firstLine
	}

	// Truncate to exactly 100 runes.
	runes := []rune(firstLine)
	return string(runes[:100])
}
