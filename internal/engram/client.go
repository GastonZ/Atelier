package engram

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Client is the read-only boundary the TUI depends on.
// Concrete implementation: sqliteClient.
// Test implementation: MockEngramClient (in client_test.go).
type Client interface {
	// ListByProject returns non-deleted observations for a project, sorted created_at DESC.
	ListByProject(name string) ([]Observation, error)
	// GetByID returns the full observation for the given id (no content truncation).
	// Returns an error if the observation is not found or is soft-deleted.
	GetByID(id int64) (Observation, error)
	// ListSDDArchivesForProject returns non-deleted observations whose topic_key
	// matches 'sdd/%/archive-report' for the given project, sorted created_at DESC.
	ListSDDArchivesForProject(name string) ([]Observation, error)
	// ListProjects returns the distinct project keys present in the store with
	// their non-deleted observation counts, most populous first. Used by the
	// project↔engram link picker so users map to real keys, not guesses.
	ListProjects() ([]ProjectStat, error)
	// Close releases the database connection.
	Close() error
}

// ProjectStat is one distinct engram project key and how many non-deleted
// observations it holds.
type ProjectStat struct {
	Key   string
	Count int
}

// requiredColumns lists the columns the client needs present in the observations table.
// Schema guard fails fast if any are missing.
var requiredColumns = []string{
	"id", "type", "title", "content", "project", "scope",
	"topic_key", "created_at", "deleted_at",
}

// sqliteClient is the read-only SQLite implementation of Client.
type sqliteClient struct {
	db *sql.DB
}

// NewClient opens the engram DB at dbPath in read-only WAL mode.
// Runs PRAGMA table_info to validate required columns exist.
// Returns a wrapped error on schema mismatch or open failure.
func NewClient(dbPath string) (Client, error) {
	// Open read-only with WAL journal mode hint.
	dsn := fmt.Sprintf("file:%s?mode=ro&_journal_mode=WAL", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("engram: open %s: %w", dbPath, err)
	}

	// Ping to confirm the file is readable (mode=ro fails fast if missing).
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("engram: ping %s: %w", dbPath, err)
	}

	// Schema guard: run PRAGMA table_info to confirm required columns.
	if err := validateSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &sqliteClient{db: db}, nil
}

// validateSchema confirms all required columns exist in the observations table.
func validateSchema(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(observations)")
	if err != nil {
		return fmt.Errorf("engram: schema check: %w", err)
	}
	defer rows.Close()

	present := make(map[string]bool)
	for rows.Next() {
		var cid, notnull, pk int
		var name, colType string
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("engram: schema scan: %w", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("engram: schema rows: %w", err)
	}

	for _, col := range requiredColumns {
		if !present[col] {
			return fmt.Errorf("engram: schema mismatch: missing column %s", col)
		}
	}
	return nil
}

// ListByProject returns non-deleted observations for the project, newest first.
func (c *sqliteClient) ListByProject(name string) ([]Observation, error) {
	const q = `
SELECT id, project, scope, type, title, content, topic_key, created_at
FROM observations
WHERE deleted_at IS NULL AND project = ? COLLATE NOCASE
ORDER BY created_at DESC`

	return c.queryObservations(q, name)
}

// ListProjects returns distinct project keys and their non-deleted counts,
// most populous first.
func (c *sqliteClient) ListProjects() ([]ProjectStat, error) {
	const q = `
SELECT project, COUNT(*) AS n
FROM observations
WHERE deleted_at IS NULL AND project IS NOT NULL AND project <> ''
GROUP BY project
ORDER BY n DESC, project ASC`

	rows, err := c.retryQuery(q)
	if err != nil {
		return nil, fmt.Errorf("engram: ListProjects query: %w", err)
	}
	defer rows.Close()

	var stats []ProjectStat
	for rows.Next() {
		var s ProjectStat
		if err := rows.Scan(&s.Key, &s.Count); err != nil {
			return nil, fmt.Errorf("engram: ListProjects scan: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetByID returns the full observation for id. Returns an error if not found.
func (c *sqliteClient) GetByID(id int64) (Observation, error) {
	const q = `
SELECT id, project, scope, type, title, content, topic_key, created_at
FROM observations
WHERE id = ? AND deleted_at IS NULL`

	rows, err := c.retryQuery(q, id)
	if err != nil {
		return Observation{}, fmt.Errorf("engram: GetByID query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return Observation{}, errors.New("engram: observation not found")
	}
	obs, err := scanObservation(rows)
	if err != nil {
		return Observation{}, err
	}
	return obs, rows.Err()
}

// ListSDDArchivesForProject returns non-deleted archive-report observations for the project.
func (c *sqliteClient) ListSDDArchivesForProject(name string) ([]Observation, error) {
	const q = `
SELECT id, project, scope, type, title, content, topic_key, created_at
FROM observations
WHERE deleted_at IS NULL
  AND project = ? COLLATE NOCASE
  AND topic_key LIKE 'sdd/%'
  AND topic_key LIKE '%/archive-report'
ORDER BY created_at DESC`

	return c.queryObservations(q, name)
}

// Close releases the database connection.
func (c *sqliteClient) Close() error {
	if c.db == nil {
		return nil
	}
	return c.db.Close()
}

// queryObservations is a helper that runs a SELECT query returning []Observation.
func (c *sqliteClient) queryObservations(query string, args ...interface{}) ([]Observation, error) {
	rows, err := c.retryQuery(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Observation
	for rows.Next() {
		obs, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, obs)
	}
	return out, rows.Err()
}

// retryQuery executes a query, retrying once after 100ms on "database is locked".
// This handles contention when the engram MCP writer holds a brief write lock.
func (c *sqliteClient) retryQuery(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := c.db.Query(query, args...)
	if err != nil && isLocked(err) {
		time.Sleep(100 * time.Millisecond)
		rows, err = c.db.Query(query, args...)
	}
	return rows, err
}

// isLocked returns true if the error indicates a SQLite locked database.
func isLocked(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "database is locked") || contains(s, "SQLITE_BUSY")
}

// contains is a simple substring check (avoids importing strings in this file).
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// scanObservation reads one observation row.
// Column order must match the SELECT list in all queries:
// id, project, scope, type, title, content, topic_key, created_at
func scanObservation(rows *sql.Rows) (Observation, error) {
	var obs Observation
	var topicKey sql.NullString
	var createdAtStr string

	if err := rows.Scan(
		&obs.ID,
		&obs.Project,
		&obs.Scope,
		&obs.Type,
		&obs.Title,
		&obs.Content,
		&topicKey,
		&createdAtStr,
	); err != nil {
		return Observation{}, fmt.Errorf("engram: scan observation: %w", err)
	}

	obs.TopicKey = topicKey.String

	// Parse created_at. Engram stores ISO datetime strings.
	// Try multiple formats for resilience.
	ts, err := parseTimestamp(createdAtStr)
	if err != nil {
		// Non-fatal — use zero time rather than failing the whole query.
		ts = time.Time{}
	}
	obs.Timestamp = ts

	return obs, nil
}

// parseTimestamp attempts several common SQLite datetime string formats.
var timestampFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseTimestamp(s string) (time.Time, error) {
	for _, f := range timestampFormats {
		if t, err := time.ParseInLocation(f, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("engram: cannot parse timestamp %q", s)
}
