package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"
)

// fixedTime is a deterministic time used across all tests.
var fixedTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

// newRegForTest constructs a fileRegistry with fully injected seams.
// home is a t.TempDir() path; now is a fixed time; ids is a list of UUIDs to hand out in order.
func newRegForTest(t *testing.T, home string, now time.Time, ids []string) *fileRegistry {
	t.Helper()
	var i int
	return newFileRegistryForTest(
		func() (string, error) { return home, nil },
		func() time.Time { return now },
		func() string {
			id := ids[i]
			i++
			return id
		},
	)
}

// TestFileRegistry_AddPersistsAndAssignsUUID verifies S1.1: add writes to disk,
// assigns a UUID, and sets the correct name/path fields.
func TestFileRegistry_AddPersistsAndAssignsUUID(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"a1b2c3d4-0000-0000-0000-000000000001"})

	proj, err := reg.Add("atelier", "/some/path")
	if err != nil {
		t.Fatalf("Add() unexpected error: %v", err)
	}

	// UUID format
	uuidRE := regexp.MustCompile(`^[0-9a-f-]{36}$`)
	if !uuidRE.MatchString(proj.ID) {
		t.Errorf("Add() id = %q, want UUID v4 format", proj.ID)
	}
	if proj.Name != "atelier" {
		t.Errorf("Add() name = %q, want %q", proj.Name, "atelier")
	}
	if proj.Path != "/some/path" {
		t.Errorf("Add() path = %q, want %q", proj.Path, "/some/path")
	}
	if proj.LastOpenedAt != nil {
		t.Error("Add() last_opened_at should be nil")
	}

	// Verify file exists on disk
	jsonPath := filepath.Join(home, ".atelier", "projects.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("projects.json not found: %v", err)
	}

	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		t.Fatalf("projects.json unmarshal error: %v", err)
	}
	if store.SchemaVersion != currentSchemaVersion {
		t.Errorf("schema_version = %d, want %d", store.SchemaVersion, currentSchemaVersion)
	}
	if len(store.Projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(store.Projects))
	}
	if store.Projects[0].Name != "atelier" {
		t.Errorf("stored name = %q, want %q", store.Projects[0].Name, "atelier")
	}
	// last_opened_at should be absent from JSON
	if raw := string(data); containsSubstring(raw, "last_opened_at") {
		t.Error("last_opened_at should be omitted from JSON when nil")
	}
}

// TestFileRegistry_AddRequiresNonEmptyName verifies that Add returns an error for empty names.
func TestFileRegistry_AddRequiresNonEmptyName(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			reg := newRegForTest(t, home, fixedTime, []string{"any-id"})
			_, err := reg.Add(tt.arg, "/path")
			if err == nil {
				t.Error("Add() should return error for empty name, got nil")
			}
		})
	}
}

// TestFileRegistry_ListReturnsInsertionOrder verifies S1.2: List preserves add order.
func TestFileRegistry_ListReturnsInsertionOrder(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{
		"id-0001-0000-0000-0000-000000000001",
		"id-0002-0000-0000-0000-000000000002",
		"id-0003-0000-0000-0000-000000000003",
	})

	names := []string{"A", "B", "C"}
	for i, n := range names {
		if _, err := reg.Add(n, "/path/"+n); err != nil {
			t.Fatalf("Add(%q) error at index %d: %v", n, i, err)
		}
	}

	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("len(List()) = %d, want 3", len(projects))
	}
	for i, n := range names {
		if projects[i].Name != n {
			t.Errorf("List()[%d].Name = %q, want %q", i, projects[i].Name, n)
		}
	}
}

// TestFileRegistry_ListReturnsEmptyWhenFileMissing verifies S1.6: missing file → empty list, no error.
func TestFileRegistry_ListReturnsEmptyWhenFileMissing(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{})

	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() error on missing file: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("len(List()) = %d, want 0", len(projects))
	}
}

// TestFileRegistry_DeleteRemovesByID verifies S1.3: delete removes project and persists.
func TestFileRegistry_DeleteRemovesByID(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{
		"id-0001-0000-0000-0000-000000000001",
		"id-0002-0000-0000-0000-000000000002",
	})

	p1, _ := reg.Add("P1", "/path/p1")
	_, _ = reg.Add("P2", "/path/p2")

	if err := reg.Delete(p1.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() after delete error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("len(List()) after delete = %d, want 1", len(projects))
	}
	if projects[0].Name != "P2" {
		t.Errorf("remaining project = %q, want P2", projects[0].Name)
	}
}

// TestFileRegistry_DeleteReturnsNotFoundForUnknownID verifies S1.4.
func TestFileRegistry_DeleteReturnsNotFoundForUnknownID(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{})

	err := reg.Delete("nonexistent-id")
	if err == nil {
		t.Fatal("Delete() should return error for unknown id, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

// TestFileRegistry_TouchUpdatesLastOpenedAt verifies S1.10.
func TestFileRegistry_TouchUpdatesLastOpenedAt(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"id-0001-0000-0000-0000-000000000001"})

	proj, _ := reg.Add("P1", "/path/p1")
	if proj.LastOpenedAt != nil {
		t.Fatal("LastOpenedAt should be nil after Add")
	}

	if err := reg.Touch(proj.ID); err != nil {
		t.Fatalf("Touch() error: %v", err)
	}

	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() after Touch error: %v", err)
	}
	if projects[0].LastOpenedAt == nil {
		t.Fatal("LastOpenedAt should be set after Touch")
	}
	if !projects[0].LastOpenedAt.Equal(fixedTime) {
		t.Errorf("LastOpenedAt = %v, want %v", *projects[0].LastOpenedAt, fixedTime)
	}

	// Verify persisted to disk
	jsonPath := filepath.Join(home, ".atelier", "projects.json")
	data, _ := os.ReadFile(jsonPath)
	if !containsSubstring(string(data), "last_opened_at") {
		t.Error("last_opened_at should be present in JSON after Touch")
	}
}

// TestFileRegistry_TouchReturnsNotFoundForUnknownID verifies ErrNotFound sentinel on Touch.
func TestFileRegistry_TouchReturnsNotFoundForUnknownID(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{})

	err := reg.Touch("nonexistent-id")
	if err == nil {
		t.Fatal("Touch() should return error for unknown id, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Touch() error = %v, want ErrNotFound", err)
	}
}

// TestFileRegistry_RoundTripPreservesAllFields verifies that all fields survive a save/load cycle.
func TestFileRegistry_RoundTripPreservesAllFields(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"round-trip-id-00000000000001"})

	added, _ := reg.Add("RoundTrip", "/round/trip")
	_ = reg.Touch(added.ID)

	// Load fresh from disk via a new registry instance pointing to same home
	reg2 := newRegForTest(t, home, fixedTime, []string{})
	projects, err := reg2.List()
	if err != nil {
		t.Fatalf("List() round-trip error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("round-trip: len(projects) = %d, want 1", len(projects))
	}
	p := projects[0]
	if p.Name != "RoundTrip" {
		t.Errorf("Name = %q, want %q", p.Name, "RoundTrip")
	}
	if p.Path != "/round/trip" {
		t.Errorf("Path = %q, want %q", p.Path, "/round/trip")
	}
	if p.LastOpenedAt == nil {
		t.Fatal("LastOpenedAt should be non-nil after round-trip")
	}
}

// TestFileRegistry_LoadReturnsSchemaMismatchOnFutureVersion verifies S1.7.
func TestFileRegistry_LoadReturnsSchemaMismatchOnFutureVersion(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".atelier")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"schema_version":2,"projects":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "projects.json"), content, 0600); err != nil {
		t.Fatal(err)
	}

	reg := newRegForTest(t, home, fixedTime, []string{})
	_, err := reg.List()
	if err == nil {
		t.Fatal("List() should return error on schema mismatch, got nil")
	}
	if !errors.Is(err, ErrSchemaMismatch) {
		t.Errorf("List() error = %v, want to wrap ErrSchemaMismatch", err)
	}
}

// TestFileRegistry_AtomicWriteUsesTmpAndRename verifies S1.5: no .tmp left after write.
func TestFileRegistry_AtomicWriteUsesTmpAndRename(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"atomic-id-00000000000000001"})

	if _, err := reg.Add("x", "/p"); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	dir := filepath.Join(home, ".atelier")
	tmpPath := filepath.Join(dir, "projects.json.tmp")
	jsonPath := filepath.Join(dir, "projects.json")

	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("projects.json.tmp should not exist after Add completes")
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Errorf("projects.json should exist: %v", err)
	}
}

// TestFileRegistry_DirectoryCreatedWith0700 verifies S1.8 (skipped on Windows).
func TestFileRegistry_DirectoryCreatedWith0700(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits not applicable on Windows")
	}
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"dir-perm-id-00000000000001"})

	if _, err := reg.Add("x", "/p"); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	dir := filepath.Join(home, ".atelier")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("dir perm = %04o, want 0700", perm)
	}
}

// TestFileRegistry_FileWrittenWith0600 verifies S1.9 (skipped on Windows).
func TestFileRegistry_FileWrittenWith0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits not applicable on Windows")
	}
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"file-perm-id-0000000000001"})

	if _, err := reg.Add("x", "/p"); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	jsonPath := filepath.Join(home, ".atelier", "projects.json")
	info, err := os.Stat(jsonPath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file perm = %04o, want 0600", perm)
	}
}

// --- Triangulation: additional edge case tests ---

// TestFileRegistry_AddTrimsNameWhitespace verifies that names are stored trimmed.
func TestFileRegistry_AddTrimsNameWhitespace(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"trim-id-0000000000000001"})

	proj, err := reg.Add("  myproject  ", "/path")
	if err != nil {
		t.Fatalf("Add() unexpected error: %v", err)
	}
	if proj.Name != "myproject" {
		t.Errorf("Add() name = %q, want %q (trimmed)", proj.Name, "myproject")
	}
}

// TestFileRegistry_PathsWithSpaces verifies paths containing spaces are preserved correctly.
func TestFileRegistry_PathsWithSpaces(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{"space-id-000000000000001"})

	proj, err := reg.Add("spaced", "/path with spaces/my project")
	if err != nil {
		t.Fatalf("Add() unexpected error: %v", err)
	}
	if proj.Path != "/path with spaces/my project" {
		t.Errorf("Add() path = %q, want path with spaces preserved", proj.Path)
	}

	// Verify round-trip
	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if projects[0].Path != "/path with spaces/my project" {
		t.Errorf("List() path = %q, want path with spaces preserved", projects[0].Path)
	}
}

// TestFileRegistry_DuplicateNamesAllowed verifies that two projects with the same name can coexist.
func TestFileRegistry_DuplicateNamesAllowed(t *testing.T) {
	home := t.TempDir()
	reg := newRegForTest(t, home, fixedTime, []string{
		"dup-id-0000000000000001",
		"dup-id-0000000000000002",
	})

	_, err1 := reg.Add("samename", "/path/1")
	_, err2 := reg.Add("samename", "/path/2")
	if err1 != nil || err2 != nil {
		t.Fatalf("Add() duplicate names should be allowed; err1=%v err2=%v", err1, err2)
	}

	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("len(List()) = %d, want 2 (duplicate names allowed)", len(projects))
	}
}

// TestFileRegistry_LoadCorruptedJSON verifies that invalid JSON returns a wrapped error.
func TestFileRegistry_LoadCorruptedJSON(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".atelier")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "projects.json"), []byte("not-json{{{"), 0600); err != nil {
		t.Fatal(err)
	}

	reg := newRegForTest(t, home, fixedTime, []string{})
	_, err := reg.List()
	if err == nil {
		t.Fatal("List() should return error for corrupted JSON, got nil")
	}
}

// TestFileRegistry_HomeDirError verifies that a homeDirFn error is propagated.
func TestFileRegistry_HomeDirError(t *testing.T) {
	reg := newFileRegistryForTest(
		func() (string, error) { return "", fmt.Errorf("home dir unavailable") },
		func() time.Time { return fixedTime },
		func() string { return "any-id" },
	)

	_, err := reg.List()
	if err == nil {
		t.Fatal("List() should propagate homeDirFn error, got nil")
	}

	_, err = reg.Add("name", "/path")
	if err == nil {
		t.Fatal("Add() should propagate homeDirFn error, got nil")
	}

	err = reg.Delete("any-id")
	if err == nil {
		t.Fatal("Delete() should propagate homeDirFn error, got nil")
	}

	err = reg.Touch("any-id")
	if err == nil {
		t.Fatal("Touch() should propagate homeDirFn error, got nil")
	}
}

// TestNewFileRegistry_Smoke verifies that NewFileRegistry returns a non-nil Registry.
// We do NOT call List/Add against real $HOME — just verify the type.
func TestNewFileRegistry_Smoke(t *testing.T) {
	reg := NewFileRegistry()
	if reg == nil {
		t.Error("NewFileRegistry() returned nil")
	}
}

// TestNewFileRegistryForTest_Smoke verifies the exported test constructor works.
func TestNewFileRegistryForTest_Smoke(t *testing.T) {
	home := t.TempDir()
	reg := NewFileRegistryForTest(
		func() (string, error) { return home, nil },
		func() time.Time { return fixedTime },
		func() string { return "smoke-test-id-0000000001" },
	)
	if reg == nil {
		t.Fatal("NewFileRegistryForTest() returned nil")
	}
	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected empty registry, got %d projects", len(projects))
	}
}

// containsSubstring is a helper to check substring presence without importing strings.
func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findInString(s, sub))
}

func findInString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
