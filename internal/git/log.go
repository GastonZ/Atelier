package git

import (
	"strings"
	"time"
)

// Commit is one entry from `git log --pretty=%h|%ad|%s --date=short`.
type Commit struct {
	Hash    string
	Date    time.Time // parsed from YYYY-MM-DD
	Subject string
}

// LogReader is the boundary for git log + show reads.
// Tests inject fakes; production uses execLogReader.
type LogReader interface {
	// Log returns up to n commits from repoPath, newest first.
	// Returns empty slice (not error) when git is unavailable or repo has no commits.
	Log(repoPath string, n int) ([]Commit, error)
	// Show returns the output of `git show --stat <hash>` verbatim.
	Show(repoPath, hash string) (string, error)
}

// execLogReader is the concrete LogReader backed by the git CLI.
type execLogReader struct{}

// NewLogReader returns a LogReader backed by the git CLI.
func NewLogReader() LogReader {
	return &execLogReader{}
}

// Log runs `git log --oneline -n --pretty=%h|%ad|%s --date=short`.
// Malformed lines are silently skipped.
func (r *execLogReader) Log(repoPath string, n int) ([]Commit, error) {
	if !IsAvailable() {
		return nil, nil
	}

	format := "--pretty=%h|%ad|%s"
	cmd := execCommand("git", "log", format, "--date=short", "-n", strN(n))
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		// Non-zero exit (empty repo, not a repo, etc.) → return empty slice.
		return nil, nil
	}

	return parseLogOutput(string(out)), nil
}

// parseLogOutput parses the multi-line output of git log.
// Malformed lines are silently skipped per design.
func parseLogOutput(stdout string) []Commit {
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		c, ok := parseLogLine(line)
		if ok {
			commits = append(commits, c)
		}
	}
	return commits
}

// parseLogLine parses one pipe-delimited log line: "%h|%ad|%s".
// Returns (Commit, true) on success; (Commit{}, false) if the line is malformed.
func parseLogLine(line string) (Commit, bool) {
	if line == "" {
		return Commit{}, false
	}

	// Split on first two pipes only — subject may contain pipes.
	idx1 := strings.Index(line, "|")
	if idx1 < 0 {
		return Commit{}, false
	}
	hash := line[:idx1]
	if hash == "" {
		return Commit{}, false
	}

	rest := line[idx1+1:]
	idx2 := strings.Index(rest, "|")
	if idx2 < 0 {
		return Commit{}, false
	}
	dateStr := rest[:idx2]
	subject := rest[idx2+1:]

	date, err := time.ParseInLocation("2006-01-02", dateStr, time.UTC)
	if err != nil {
		return Commit{}, false
	}

	return Commit{Hash: hash, Date: date, Subject: subject}, true
}

// strN converts an int to a string for use as a command argument.
func strN(n int) string {
	if n <= 0 {
		return "0"
	}
	// Simple int-to-string without importing strconv.
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
