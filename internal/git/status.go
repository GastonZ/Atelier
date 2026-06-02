package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Status is the parsed output of `git status --porcelain=v1 -b`.
type Status struct {
	Branch    string
	Modified  int  // count of all changed lines (M/A/D/R/C/?? etc.)
	Ahead     int  // commits ahead of remote
	Behind    int  // commits behind remote
	IsRepo    bool // false when the path is not a git repo
	Available bool // false when git binary is not on PATH
}

// StatusReader is the boundary for git status reads.
// The TUI depends on this interface; tests inject fakes.
type StatusReader interface {
	Status(repoPath string) (Status, error)
}

// execStatusReader is the concrete StatusReader using os/exec.
type execStatusReader struct{}

// NewStatusReader returns a StatusReader backed by the git CLI.
func NewStatusReader() StatusReader {
	return &execStatusReader{}
}

// Status runs `git status --porcelain=v1 -b` with a 500ms timeout.
// Returns Status{Available:false} if git is not on PATH.
// Returns Status{Available:true, IsRepo:false} on non-zero exit or timeout.
func (r *execStatusReader) Status(repoPath string) (Status, error) {
	if !IsAvailable() {
		return Status{Available: false}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := execCommand("git", "status", "--porcelain=v1", "-b")
	cmd.Dir = repoPath

	// Wrap in context timeout via goroutine + channel pattern.
	type result struct {
		out []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := cmd.Output()
		ch <- result{out, err}
	}()

	select {
	case <-ctx.Done():
		// Timeout — kill the process if possible.
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return Status{Available: true, IsRepo: false}, nil
	case r := <-ch:
		if r.err != nil {
			// Non-zero exit = not a git repo or other error.
			return Status{Available: true, IsRepo: false}, nil
		}
		s := parsePorcelain(string(r.out))
		s.Available = true
		return s, nil
	}
}

// parsePorcelain parses the output of `git status --porcelain=v1 -b`.
// It is an unexported pure function, easily testable.
func parsePorcelain(stdout string) Status {
	lines := strings.Split(stdout, "\n")
	if len(lines) == 0 || stdout == "" {
		return Status{Available: true, IsRepo: false}
	}

	s := Status{Available: true}
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			// Branch header: "## branch...remote [ahead N, behind M]"
			s.IsRepo = true
			s.Branch, s.Ahead, s.Behind = parseBranchHeader(line[3:])
		} else if len(line) >= 2 {
			// Status line: XY path — any non-space XY means a change.
			// Count as modified if any character in first 2 cols is not space.
			xy := line[:2]
			if strings.TrimSpace(xy) != "" {
				s.Modified++
			}
		}
	}
	return s
}

// parseBranchHeader parses the branch-tracking portion of the ## header line.
// Input: "main...origin/main [ahead 2, behind 1]"
func parseBranchHeader(header string) (branch string, ahead, behind int) {
	// Extract ahead/behind counts from the bracket suffix if present.
	if idx := strings.Index(header, "["); idx >= 0 {
		brackets := header[idx+1:]
		if end := strings.Index(brackets, "]"); end >= 0 {
			brackets = brackets[:end]
		}
		ahead = extractCount(brackets, "ahead")
		behind = extractCount(brackets, "behind")
		header = strings.TrimSpace(header[:idx])
	}
	// Extract branch name (before "...").
	if idx := strings.Index(header, "..."); idx >= 0 {
		branch = header[:idx]
	} else {
		branch = header
	}
	return branch, ahead, behind
}

// extractCount extracts a numeric value after a keyword like "ahead" or "behind".
// Input example: "ahead 2, behind 1".
func extractCount(s, keyword string) int {
	idx := strings.Index(s, keyword)
	if idx < 0 {
		return 0
	}
	rest := strings.TrimSpace(s[idx+len(keyword):])
	// rest may be "2" or "2, behind 1" — grab the first token.
	parts := strings.FieldsFunc(rest, func(r rune) bool {
		return r == ',' || r == ' '
	})
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

// FormatGitIndicator formats a Status as a compact indicator string.
// Design §10 format rules:
//   - Status{Available:false} or {IsRepo:false} → "?"
//   - All-zero → "✓"
//   - Otherwise: concatenate "↑<Ahead>" "↓<Behind>" "M<Modified>" (only non-zero parts).
func FormatGitIndicator(s Status) string {
	if !s.Available || !s.IsRepo {
		return "?"
	}
	if s.Modified == 0 && s.Ahead == 0 && s.Behind == 0 {
		return "✓"
	}
	var b strings.Builder
	if s.Ahead > 0 {
		fmt.Fprintf(&b, "↑%d", s.Ahead)
	}
	if s.Behind > 0 {
		fmt.Fprintf(&b, "↓%d", s.Behind)
	}
	if s.Modified > 0 {
		fmt.Fprintf(&b, "M%d", s.Modified)
	}
	return b.String()
}
