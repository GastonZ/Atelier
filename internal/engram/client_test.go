package engram

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// buildTestDB creates a temporary SQLite database seeded from testdata/seed.sql.
// Returns the path to the created DB file in a t.TempDir().
func buildTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_engram.db")

	seedSQL, err := os.ReadFile(filepath.Join("testdata", "seed.sql"))
	if err != nil {
		t.Fatalf("buildTestDB: read seed.sql: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("buildTestDB: open DB: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(string(seedSQL)); err != nil {
		t.Fatalf("buildTestDB: exec seed.sql: %v", err)
	}
	return dbPath
}

// ============================================================================
// Client interface tests
// ============================================================================

// TestNewClient_OpensMissingDB verifies error (not panic) when DB does not exist.
func TestNewClient_OpensMissingDB(t *testing.T) {
	c, err := NewClient("/nonexistent/path/engram.db")
	if err == nil {
		_ = c.Close()
		t.Fatal("NewClient() expected error for missing DB, got nil")
	}
	// Must not panic — error returned cleanly.
}

// TestNewClient_OpensValidDB verifies the client opens a seeded DB without error.
func TestNewClient_OpensValidDB(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	defer func() { _ = c.Close() }()
}

// TestListByProject_ReturnsCorrectCount verifies S8.1: 3 observations for "atelier", ignores deleted.
func TestListByProject_ReturnsCorrectCount(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListByProject("atelier")
	if err != nil {
		t.Fatalf("ListByProject() error = %v, want nil", err)
	}
	// Seed has 3 live + 1 deleted + 2 archive-reports = 6 rows for atelier, but
	// seed.sql: ids 1,2,3 = regular live; 6 = deleted; 7,8 = archive-reports; 9 = long.
	// All non-deleted = 1,2,3,7,8,9 → 6. But wait, archives have deleted_at=NULL too.
	// Design says ListByProject returns all non-deleted for the project.
	// Let's count: ids 1,2,3,7,8,9 all have project='atelier' and deleted_at IS NULL → 6.
	if len(obs) != 6 {
		t.Errorf("ListByProject(atelier) returned %d observations, want 6", len(obs))
	}
}

// TestListByProject_IgnoresSoftDeleted verifies soft-deleted rows are excluded.
func TestListByProject_IgnoresSoftDeleted(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListByProject("atelier")
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	for _, o := range obs {
		if o.ID == 6 {
			t.Error("ListByProject returned deleted observation (id=6)")
		}
	}
}

// TestListByProject_OrderedCreatedAtDesc verifies S8.1: newest first.
func TestListByProject_OrderedCreatedAtDesc(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListByProject("atelier")
	if err != nil {
		t.Fatalf("ListByProject() error: %v", err)
	}
	if len(obs) < 2 {
		t.Skip("not enough observations to check order")
	}
	for i := 1; i < len(obs); i++ {
		if obs[i-1].Timestamp.Before(obs[i].Timestamp) {
			t.Errorf("observations not sorted desc: obs[%d].Timestamp=%v < obs[%d].Timestamp=%v",
				i-1, obs[i-1].Timestamp, i, obs[i].Timestamp)
		}
	}
}

// TestListByProject_UnknownProject verifies S8.2: empty slice, nil error for unknown project.
func TestListByProject_UnknownProject(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListByProject("nonexistent-project")
	if err != nil {
		t.Fatalf("ListByProject(unknown) error = %v, want nil", err)
	}
	if len(obs) != 0 {
		t.Errorf("ListByProject(unknown) returned %d obs, want 0", len(obs))
	}
}

// TestGetByID_ReturnsFullContent verifies S8.4: 5000-char content not truncated.
func TestGetByID_ReturnsFullContent(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.GetByID(9)
	if err != nil {
		t.Fatalf("GetByID(9) error = %v, want nil", err)
	}
	if len(obs.Content) != 5000 {
		t.Errorf("GetByID(9) content length = %d, want 5000", len(obs.Content))
	}
}

// TestGetByID_Fields verifies all fields are correctly populated.
func TestGetByID_Fields(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID(1) error: %v", err)
	}
	if obs.ID != 1 {
		t.Errorf("ID = %d, want 1", obs.ID)
	}
	if obs.Project != "atelier" {
		t.Errorf("Project = %q, want atelier", obs.Project)
	}
	if obs.Type != "architecture" {
		t.Errorf("Type = %q, want architecture", obs.Type)
	}
	if obs.Title != "Bootstrap decision" {
		t.Errorf("Title = %q, want Bootstrap decision", obs.Title)
	}
	if obs.TopicKey != "sdd/atelier-bootstrap/design" {
		t.Errorf("TopicKey = %q, want sdd/atelier-bootstrap/design", obs.TopicKey)
	}
	wantTS := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	if !obs.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", obs.Timestamp, wantTS)
	}
}

// TestGetByID_NotFound verifies error (not panic) for missing ID.
func TestGetByID_NotFound(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.GetByID(9999)
	if err == nil {
		t.Fatal("GetByID(9999) expected error, got nil")
	}
}

// TestListSDDArchivesForProject_ReturnsArchiveRows verifies only archive-report topic_key rows returned.
func TestListSDDArchivesForProject_ReturnsArchiveRows(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListSDDArchivesForProject("atelier")
	if err != nil {
		t.Fatalf("ListSDDArchivesForProject() error: %v", err)
	}
	// Seed has 2 archive-report rows for atelier (ids 7, 8).
	if len(obs) != 2 {
		t.Errorf("ListSDDArchivesForProject returned %d, want 2", len(obs))
	}
	const suffix = "/archive-report"
	for _, o := range obs {
		if o.TopicKey == "" {
			t.Error("archive row has empty TopicKey")
		}
		if len(o.TopicKey) < len(suffix) || o.TopicKey[len(o.TopicKey)-len(suffix):] != suffix {
			t.Errorf("unexpected TopicKey: %q (want suffix %q)", o.TopicKey, suffix)
		}
	}
}

// TestListSDDArchivesForProject_OtherProject verifies archives are scoped to the project.
func TestListSDDArchivesForProject_OtherProject(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListSDDArchivesForProject("other")
	if err != nil {
		t.Fatalf("ListSDDArchivesForProject(other) error: %v", err)
	}
	if len(obs) != 0 {
		t.Errorf("ListSDDArchivesForProject(other) returned %d, want 0", len(obs))
	}
}

// TestClose_Idempotent verifies Close() can be called multiple times without error.
func TestClose_Idempotent(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close() first call error: %v", err)
	}
	// Second close should not panic (may return error for already-closed db, which is acceptable).
	_ = c.Close()
}

// ============================================================================
// T9: Edge case / triangulation tests
// ============================================================================

// TestNewClient_SchemaGuard_EmptyTable verifies schema guard returns error for missing columns.
func TestNewClient_SchemaGuard_EmptyTable(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/empty.db"

	// Create a DB with an observations table that lacks required columns.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE observations (fake_col TEXT)`)
	db.Close()
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = NewClient(dbPath)
	if err == nil {
		t.Fatal("NewClient should return error for schema mismatch, got nil")
	}
	if !contains(err.Error(), "schema mismatch") {
		t.Errorf("error = %q, want to contain 'schema mismatch'", err.Error())
	}
}

// TestNewClient_SchemaGuard_NoObservationsTable verifies error when table is missing.
func TestNewClient_SchemaGuard_NoObservationsTable(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/no_obs.db"

	// Create a DB with NO observations table at all.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE other_table (id INTEGER PRIMARY KEY)`)
	db.Close()
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = NewClient(dbPath)
	if err == nil {
		t.Fatal("NewClient should return error when observations table absent, got nil")
	}
}

// TestListByProject_MultipleProjects verifies project isolation.
func TestListByProject_MultipleProjects(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	other, err := c.ListByProject("other")
	if err != nil {
		t.Fatalf("ListByProject(other) error: %v", err)
	}
	// Seed has 2 live rows for "other" (ids 4, 5).
	if len(other) != 2 {
		t.Errorf("ListByProject(other) = %d, want 2", len(other))
	}
	for _, o := range other {
		if o.Project != "other" {
			t.Errorf("observation project = %q, want other", o.Project)
		}
	}
}

// TestGetByID_DeletedRow verifies soft-deleted observations are not returned by GetByID.
func TestGetByID_DeletedRow(t *testing.T) {
	dbPath := buildTestDB(t)
	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	// id=6 is soft-deleted in the seed.
	_, err = c.GetByID(6)
	if err == nil {
		t.Fatal("GetByID(6) should return error for soft-deleted row, got nil")
	}
}

// TestParseTimestamp_Formats verifies multiple timestamp formats are handled.
func TestParseTimestamp_Formats(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"2026-05-04 10:00:00", time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)},
		{"2026-05-04T10:00:00Z", time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)},
		{"2026-05-04T10:00:00", time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)},
		{"2026-05-04", time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTimestamp(tt.input)
			if err != nil {
				t.Fatalf("parseTimestamp(%q) error: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseTimestamp(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseTimestamp_InvalidFormat verifies error on unrecognized format.
func TestParseTimestamp_InvalidFormat(t *testing.T) {
	_, err := parseTimestamp("not-a-date")
	if err == nil {
		t.Fatal("parseTimestamp(invalid) expected error, got nil")
	}
}

// TestIsLocked_DetectsLockedError verifies the lock detection helper.
func TestIsLocked_DetectsLockedError(t *testing.T) {
	tests := []struct {
		errStr string
		want   bool
	}{
		{"database is locked", true},
		{"SQLITE_BUSY: database is locked", true},
		{"no such table: foo", false},
		{"", false},
	}
	for _, tt := range tests {
		var err error
		if tt.errStr != "" {
			err = fmt.Errorf("%s", tt.errStr)
		}
		got := isLocked(err)
		if got != tt.want {
			t.Errorf("isLocked(%q) = %v, want %v", tt.errStr, got, tt.want)
		}
	}
}

// TestListSDDArchivesForProject_EmptyDB verifies empty result for DB with no archives.
func TestListSDDArchivesForProject_EmptyDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/minimal.db"

	// Create a minimal DB with the right schema but no data.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE observations (
		id INTEGER PRIMARY KEY,
		sync_id TEXT, session_id TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL DEFAULT '', title TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '', tool_name TEXT, project TEXT,
		scope TEXT NOT NULL DEFAULT 'project', topic_key TEXT,
		normalized_hash TEXT, revision_count INTEGER NOT NULL DEFAULT 1,
		duplicate_count INTEGER NOT NULL DEFAULT 1, last_seen_at TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')), deleted_at TEXT)`)
	db.Close()
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	c, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer func() { _ = c.Close() }()

	obs, err := c.ListSDDArchivesForProject("atelier")
	if err != nil {
		t.Fatalf("ListSDDArchivesForProject() error: %v", err)
	}
	if len(obs) != 0 {
		t.Errorf("expected 0 archives from empty DB, got %d", len(obs))
	}
}

// ============================================================================
// MockEngramClient — provided for TUI tests (R8.5).
// Placed here so it's accessible across the same package's test binaries.
// ============================================================================

// MockEngramClient is a test double implementing the Client interface.
// It records calls and returns canned responses.
type MockEngramClient struct {
	ListByProjectFn              func(name string) ([]Observation, error)
	GetByIDFn                    func(id int64) (Observation, error)
	ListSDDArchivesForProjectFn  func(name string) ([]Observation, error)
	CloseFn                      func() error

	// Recorded calls.
	ListByProjectCalls             []string
	GetByIDCalls                   []int64
	ListSDDArchivesForProjectCalls []string
}

func (m *MockEngramClient) ListByProject(name string) ([]Observation, error) {
	m.ListByProjectCalls = append(m.ListByProjectCalls, name)
	if m.ListByProjectFn != nil {
		return m.ListByProjectFn(name)
	}
	return nil, nil
}

func (m *MockEngramClient) GetByID(id int64) (Observation, error) {
	m.GetByIDCalls = append(m.GetByIDCalls, id)
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}
	return Observation{}, errors.New("not found")
}

func (m *MockEngramClient) ListSDDArchivesForProject(name string) ([]Observation, error) {
	m.ListSDDArchivesForProjectCalls = append(m.ListSDDArchivesForProjectCalls, name)
	if m.ListSDDArchivesForProjectFn != nil {
		return m.ListSDDArchivesForProjectFn(name)
	}
	return nil, nil
}

func (m *MockEngramClient) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}
