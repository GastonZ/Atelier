// Package registry provides persistent storage for the user's project list.
// Projects are stored as JSON at ~/.atelier/projects.json.
// All writes are atomic (write-to-tmp then rename).
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// currentSchemaVersion is the only version this implementation reads and writes.
const currentSchemaVersion = 1

// Project is one entry in the user's project registry.
type Project struct {
	ID           string     `json:"id"`                        // UUID v4
	Name         string     `json:"name"`                      // user-supplied, free-form
	Path         string     `json:"path"`                      // native separators, as entered
	CreatedAt    time.Time  `json:"created_at"`                // RFC3339 on disk
	LastOpenedAt *time.Time `json:"last_opened_at,omitempty"` // nullable; absent in JSON when nil
}

// Store is the on-disk JSON envelope.
type Store struct {
	SchemaVersion int       `json:"schema_version"` // must equal currentSchemaVersion
	Projects      []Project `json:"projects"`
}

// Registry is the persistence boundary.
// The TUI depends on this interface; tests inject a mock or a file-backed instance.
type Registry interface {
	List() ([]Project, error)
	Add(name, path string) (Project, error)
	Delete(id string) error
	Touch(id string) error // sets LastOpenedAt to nowFn()
}

// fileRegistry is the JSON-file-backed Registry implementation.
// Constructor injects deterministic seams so tests never touch real $HOME.
type fileRegistry struct {
	homeDirFn func() (string, error)
	nowFn     func() time.Time
	newIDFn   func() string
}

// NewFileRegistry returns a production Registry bound to $HOME/.atelier/projects.json.
func NewFileRegistry() Registry {
	return &fileRegistry{
		homeDirFn: os.UserHomeDir,
		nowFn:     time.Now,
		newIDFn:   func() string { return uuid.NewString() },
	}
}

// newFileRegistryForTest is unexported; tests in this package use it via newRegForTest.
func newFileRegistryForTest(
	home func() (string, error),
	now func() time.Time,
	id func() string,
) *fileRegistry {
	return &fileRegistry{homeDirFn: home, nowFn: now, newIDFn: id}
}

// NewFileRegistryForTest is exported for use by the tui package tests.
// It exposes the same test-seam constructor without requiring tui to import test internals.
func NewFileRegistryForTest(
	home func() (string, error),
	now func() time.Time,
	id func() string,
) Registry {
	return newFileRegistryForTest(home, now, id)
}

// List returns all projects in insertion order.
// If the file does not exist, returns an empty slice and nil error.
func (r *fileRegistry) List() ([]Project, error) {
	s, err := r.load()
	if err != nil {
		return nil, err
	}
	if s.Projects == nil {
		return []Project{}, nil
	}
	return s.Projects, nil
}

// Add appends a new project to the registry and persists it atomically.
// Returns an error if name is empty after trimming whitespace.
func (r *fileRegistry) Add(name, path string) (Project, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Project{}, fmt.Errorf("registry: name required")
	}

	s, err := r.load()
	if err != nil {
		return Project{}, err
	}

	p := Project{
		ID:        r.newIDFn(),
		Name:      trimmed,
		Path:      path,
		CreatedAt: r.nowFn().UTC(),
	}
	s.Projects = append(s.Projects, p)

	if err := r.save(s); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Delete removes the project with the given id and persists the result.
// Returns ErrNotFound if no project with that id exists.
func (r *fileRegistry) Delete(id string) error {
	s, err := r.load()
	if err != nil {
		return err
	}

	idx := -1
	for i, p := range s.Projects {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}

	// Remove by index — immutable: build new slice
	newProjects := make([]Project, 0, len(s.Projects)-1)
	newProjects = append(newProjects, s.Projects[:idx]...)
	newProjects = append(newProjects, s.Projects[idx+1:]...)
	s.Projects = newProjects

	return r.save(s)
}

// Touch sets LastOpenedAt on the matching project to the current time and persists.
// Returns ErrNotFound if no project with that id exists.
func (r *fileRegistry) Touch(id string) error {
	s, err := r.load()
	if err != nil {
		return err
	}

	now := r.nowFn().UTC()
	found := false
	for i, p := range s.Projects {
		if p.ID == id {
			s.Projects[i].LastOpenedAt = &now
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}

	return r.save(s)
}

// load reads and deserializes the registry from disk.
// If the file is absent, returns a fresh empty Store.
// If schema_version != currentSchemaVersion, returns ErrSchemaMismatch.
func (r *fileRegistry) load() (Store, error) {
	_, file, _, err := r.paths()
	if err != nil {
		return Store{}, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Store{SchemaVersion: currentSchemaVersion, Projects: nil}, nil
		}
		return Store{}, fmt.Errorf("registry: load: %w", err)
	}

	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return Store{}, fmt.Errorf("registry: load: json: %w", err)
	}
	if s.SchemaVersion != currentSchemaVersion {
		return Store{}, fmt.Errorf("registry: unsupported schema version %d: %w", s.SchemaVersion, ErrSchemaMismatch)
	}

	return s, nil
}

// save serializes the store and atomically writes it to disk.
// Creates ~/.atelier/ (mode 0700) if absent.
// Writes to *.tmp then renames to avoid partial-write corruption.
func (r *fileRegistry) save(s Store) error {
	dir, file, tmp, err := r.paths()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("registry: save: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("registry: save: marshal: %w", err)
	}

	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("registry: save: write tmp: %w", err)
	}

	if err := os.Rename(tmp, file); err != nil {
		return fmt.Errorf("registry: save: rename: %w", err)
	}

	return nil
}

// paths derives the .atelier directory, projects.json, and projects.json.tmp paths.
func (r *fileRegistry) paths() (dir, file, tmp string, err error) {
	home, err := r.homeDirFn()
	if err != nil {
		return "", "", "", fmt.Errorf("registry: home dir: %w", err)
	}
	dir = home + string(os.PathSeparator) + ".atelier"
	file = dir + string(os.PathSeparator) + "projects.json"
	tmp = file + ".tmp"
	return dir, file, tmp, nil
}
