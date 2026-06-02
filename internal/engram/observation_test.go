package engram

import (
	"strings"
	"testing"
	"time"
)

// TestObservation_FieldsExist verifies Observation has the required exported fields.
// This is a compile-time guard — if any field is missing the test won't compile (RED).
func TestObservation_FieldsExist(t *testing.T) {
	obs := Observation{
		ID:        1,
		Project:   "atelier",
		Scope:     "project",
		Type:      "architecture",
		Title:     "Bootstrap decision",
		Content:   "Some content",
		TopicKey:  "sdd/atelier-bootstrap/design",
		Timestamp: time.Now(),
	}
	if obs.ID != 1 {
		t.Errorf("ID = %d, want 1", obs.ID)
	}
	if obs.Project != "atelier" {
		t.Errorf("Project = %q, want atelier", obs.Project)
	}
}

// TestPreviewContent_FirstLineCap100 verifies that preview is the first line, capped at 100 runes.
// Table-driven per design §10 memory entry preview rule.
func TestPreviewContent_FirstLineCap100(t *testing.T) {
	long200 := strings.Repeat("a", 200)
	exact100 := strings.Repeat("b", 100)
	short50 := strings.Repeat("c", 50)

	tests := []struct {
		name    string
		content string
		title   string
		want    string
	}{
		{
			name:    "long first line capped at 100",
			content: long200 + "\nSecond line",
			title:   "Fallback Title",
			want:    strings.Repeat("a", 100),
		},
		{
			name:    "exact 100-char first line — not truncated",
			content: exact100 + "\nMore",
			title:   "Fallback Title",
			want:    exact100,
		},
		{
			name:    "short first line — returned as-is",
			content: short50,
			title:   "Fallback Title",
			want:    short50,
		},
		{
			name:    "blank first line — fallback to title",
			content: "\nSecond line after blank",
			title:   "My Title",
			want:    "My Title",
		},
		{
			name:    "empty content — fallback to title",
			content: "",
			title:   "Empty Content Title",
			want:    "Empty Content Title",
		},
		{
			name:    "first line is exactly blank (space) — fallback to title",
			content: "   \nactual content",
			title:   "Space Title",
			want:    "Space Title",
		},
		{
			name:    "unicode content — rune-safe cap",
			content: strings.Repeat("á", 150),
			title:   "Unicode",
			want:    strings.Repeat("á", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := Observation{Title: tt.title, Content: tt.content}
			got := PreviewContent(obs)
			if got != tt.want {
				t.Errorf("PreviewContent() = %q, want %q", got, tt.want)
			}
		})
	}
}
