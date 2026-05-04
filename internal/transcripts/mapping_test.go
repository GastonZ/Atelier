package transcripts_test

import (
	"testing"
	"time"

	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/transcripts"
)

// testProjects returns a fixed list of registry.Project values for mapping tests.
func testProjects() []registry.Project {
	return []registry.Project{
		{
			ID:        "proj-001",
			Name:      "Atelier",
			Path:      `C:\Users\Usuario\Desktop\Atelier`,
			CreatedAt: time.Time{},
		},
		{
			ID:        "proj-002",
			Name:      "BackAgencia",
			Path:      `C:\Users\Usuario\Desktop\GZBackAgenciaV2`,
			CreatedAt: time.Time{},
		},
		{
			ID:        "proj-003",
			Name:      "Portfolio",
			Path:      `D:\Proyectos\Gaston\portfolio`,
			CreatedAt: time.Time{},
		},
	}
}

// ---- MatchProject tests ------------------------------------------------------

func TestMatchProject_ExactMatch(t *testing.T) {
	// R6.1: exact path match returns the correct project
	projects := testProjects()

	proj, ok := transcripts.MatchProject(`C:\Users\Usuario\Desktop\Atelier`, projects)
	if !ok {
		t.Fatal("expected match for exact Atelier path")
	}
	if proj.ID != "proj-001" {
		t.Errorf("expected proj-001, got %q", proj.ID)
	}
	if proj.Name != "Atelier" {
		t.Errorf("expected Atelier, got %q", proj.Name)
	}
}

func TestMatchProject_CaseInsensitiveMatch(t *testing.T) {
	// R6.1, S6.1: case-insensitive comparison on absolute paths
	projects := testProjects()

	// Lowercase version of the path
	proj, ok := transcripts.MatchProject(`c:\users\usuario\desktop\atelier`, projects)
	if !ok {
		t.Fatal("expected case-insensitive match for lowercase Atelier path")
	}
	if proj.ID != "proj-001" {
		t.Errorf("expected proj-001, got %q", proj.ID)
	}
}

func TestMatchProject_NoMatch_ReturnsFalse(t *testing.T) {
	// R6.2, S6.2: unrecognized cwd returns (_, false)
	projects := testProjects()

	_, ok := transcripts.MatchProject(`D:\SomeOtherProject\Unknown`, projects)
	if ok {
		t.Error("expected no match for unknown path, got ok=true")
	}
}

func TestMatchProject_MultipleProjects_CorrectOneReturned(t *testing.T) {
	// R6.3: multiple projects — the correct one is returned
	projects := testProjects()

	// Match for the Portfolio project on drive D:
	proj, ok := transcripts.MatchProject(`D:\Proyectos\Gaston\portfolio`, projects)
	if !ok {
		t.Fatal("expected match for Portfolio path")
	}
	if proj.ID != "proj-003" {
		t.Errorf("expected proj-003, got %q", proj.ID)
	}
}

func TestMatchProject_EmptyProjectsList_ReturnsFalse(t *testing.T) {
	// Edge case: empty project list always returns false
	_, ok := transcripts.MatchProject(`C:\Users\Usuario\Desktop\Atelier`, nil)
	if ok {
		t.Error("expected no match for empty project list, got ok=true")
	}
}

func TestMatchProject_EmptyCwd_ReturnsFalse(t *testing.T) {
	// Edge case: empty cwd returns false without panic
	projects := testProjects()
	_, ok := transcripts.MatchProject("", projects)
	if ok {
		t.Error("expected no match for empty cwd, got ok=true")
	}
}

func TestMatchProject_UnmatchedLabel_Constant(t *testing.T) {
	// R6.2: UnmatchedLabel constant must be "Sin tomo registrado" (locked copy §10)
	if transcripts.UnmatchedLabel != "Sin tomo registrado" {
		t.Errorf("UnmatchedLabel must be %q, got %q",
			"Sin tomo registrado", transcripts.UnmatchedLabel)
	}
}
