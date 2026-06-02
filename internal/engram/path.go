package engram

import (
	"os"
	"path/filepath"
)

// DefaultDBPath returns the path to the engram database: ~/.engram/engram.db.
// Uses os.UserHomeDir for cross-platform home resolution.
// Returns an error if the home directory cannot be determined.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".engram", "engram.db"), nil
}
