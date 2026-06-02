// Package disk provides disk size utilities and filesystem helpers.
// Uses stdlib only: filepath.WalkDir, os.Stat, os.UserHomeDir.
// No third-party libraries.
package disk

import (
	"io/fs"
	"path/filepath"
)

// WalkSize returns the total byte size of all regular files under path.
// Unreadable entries are skipped silently (no error returned for per-entry failures).
// Returns an error only if the root path cannot be walked at all.
func WalkSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries silently.
			return nil
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				// Skip if stat fails.
				return nil
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}
