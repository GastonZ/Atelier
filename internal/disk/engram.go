package disk

import (
	"os"
	"path/filepath"
	"strings"
)

// EngramDBSize returns the total byte size of the engram database files:
// ~/.engram/engram.db + engram.db-wal + engram.db-shm.
// Missing files are silently treated as 0 bytes (partial WAL is normal).
// Returns (0, nil) if the ~/.engram/ directory does not exist.
func EngramDBSize() (int64, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	return engramDBSizeFromDir(filepath.Join(home, ".engram"))
}

// engramDBSizeFromDir is the testable implementation that takes a dir parameter.
func engramDBSizeFromDir(dir string) (int64, error) {
	files := []string{
		filepath.Join(dir, "engram.db"),
		filepath.Join(dir, "engram.db-wal"),
		filepath.Join(dir, "engram.db-shm"),
	}
	var total int64
	for _, f := range files {
		info, err := os.Stat(f)
		if os.IsNotExist(err) {
			continue // missing WAL/SHM files are normal
		}
		if err != nil {
			// Other stat errors: skip, don't fail
			continue
		}
		total += info.Size()
	}
	return total, nil
}

// ClaudeProjectsDir returns the path to ~/.claude/projects/.
// Returns an error if os.UserHomeDir() fails.
func ClaudeProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// ClaudeProjectsDirForPath returns the ~/.claude/projects/<key>/ directory for a tomo path.
// The key is derived by lowercasing the path and replacing path separators and colons with "-".
// Returns ("", nil) when the derived directory does not exist (best-effort heuristic).
func ClaudeProjectsDirForPath(tomoPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Heuristic: lowercase, replace \, /, : with "-".
	key := strings.ToLower(tomoPath)
	key = strings.ReplaceAll(key, "\\", "-")
	key = strings.ReplaceAll(key, "/", "-")
	key = strings.ReplaceAll(key, ":", "-")

	dir := filepath.Join(home, ".claude", "projects", key)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", nil // best-effort: directory doesn't exist → return empty, no error
	}
	return dir, nil
}
