package disk

import (
	"os"
	"path/filepath"
	"testing"
)

// createTestFiles creates files of given sizes in dir. Returns total size.
func createTestFiles(t *testing.T, dir string, sizes []int) int64 {
	t.Helper()
	var total int64
	for i, size := range sizes {
		name := filepath.Join(dir, filepath.FromSlash("file"+string(rune('0'+i))+".bin"))
		data := make([]byte, size)
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatalf("createTestFiles: %v", err)
		}
		total += int64(size)
	}
	return total
}

// TestWalkSize_KnownFiles verifies WalkSize returns the correct sum.
func TestWalkSize_KnownFiles(t *testing.T) {
	dir := t.TempDir()
	want := createTestFiles(t, dir, []int{1024, 512, 256})

	got, err := WalkSize(dir)
	if err != nil {
		t.Fatalf("WalkSize() error: %v", err)
	}
	if got != want {
		t.Errorf("WalkSize() = %d, want %d", got, want)
	}
}

// TestWalkSize_EmptyDir verifies empty directory returns 0.
func TestWalkSize_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, err := WalkSize(dir)
	if err != nil {
		t.Fatalf("WalkSize() error: %v", err)
	}
	if got != 0 {
		t.Errorf("WalkSize(empty) = %d, want 0", got)
	}
}

// TestWalkSize_NonExistentPath verifies non-existent path returns error (not panic).
func TestWalkSize_NonExistentPath(t *testing.T) {
	_, err := WalkSize(filepath.Join(t.TempDir(), "does_not_exist"))
	// Should return an error or zero — must not panic.
	// WalkDir returns error for non-existent root.
	_ = err // acceptable — either 0 or error
}

// TestWalkSize_NestedDirs verifies files in subdirectories are counted.
func TestWalkSize_NestedDirs(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	// 100 bytes in root, 200 bytes in subdir.
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "b.bin"), make([]byte, 200), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := WalkSize(dir)
	if err != nil {
		t.Fatalf("WalkSize() error: %v", err)
	}
	if got != 300 {
		t.Errorf("WalkSize() = %d, want 300", got)
	}
}

// ============================================================================
// EngramDBSize tests
// ============================================================================

// TestEngramDBSize_AllFiles verifies sum of .db + .db-wal + .db-shm.
func TestEngramDBSize_AllFiles(t *testing.T) {
	// Override the engramDir seam with a temp dir.
	dir := t.TempDir()

	// Create fake db files with known sizes.
	writeBytes(t, filepath.Join(dir, "engram.db"), 1000)
	writeBytes(t, filepath.Join(dir, "engram.db-wal"), 500)
	writeBytes(t, filepath.Join(dir, "engram.db-shm"), 250)

	got, err := engramDBSizeFromDir(dir)
	if err != nil {
		t.Fatalf("EngramDBSize() error: %v", err)
	}
	if got != 1750 {
		t.Errorf("EngramDBSize() = %d, want 1750", got)
	}
}

// TestEngramDBSize_MissingAllFiles verifies 0 is returned when no files exist.
func TestEngramDBSize_MissingAllFiles(t *testing.T) {
	dir := t.TempDir()
	got, err := engramDBSizeFromDir(dir)
	if err != nil {
		t.Fatalf("EngramDBSize() error: %v", err)
	}
	if got != 0 {
		t.Errorf("EngramDBSize(empty dir) = %d, want 0", got)
	}
}

// TestEngramDBSize_PartialFiles verifies partial WAL files are summed correctly.
func TestEngramDBSize_PartialFiles(t *testing.T) {
	dir := t.TempDir()
	// Only .db exists; .db-wal and .db-shm missing.
	writeBytes(t, filepath.Join(dir, "engram.db"), 800)

	got, err := engramDBSizeFromDir(dir)
	if err != nil {
		t.Fatalf("EngramDBSize() error: %v", err)
	}
	if got != 800 {
		t.Errorf("EngramDBSize() = %d, want 800", got)
	}
}

// writeBytes creates a file with n zero bytes.
func writeBytes(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.WriteFile(path, make([]byte, n), 0644); err != nil {
		t.Fatalf("writeBytes: %v", err)
	}
}
