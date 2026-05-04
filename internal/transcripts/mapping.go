package transcripts

import (
	"strings"

	"github.com/gastonz/atelier/internal/registry"
)

// UnmatchedLabel is the locked Spanish copy (§10) used as the bucket label when
// a session's cwd does not match any registered registry.Project.Path.
const UnmatchedLabel = "Sin tomo registrado"

// MatchProject performs a case-insensitive equality comparison between cwd
// and each registry.Project.Path to determine which project owns the session.
//
// Returns (project, true) on the first match.
// Returns (zero, false) when cwd is empty, projects is empty, or no path matches.
//
// R6.1: match is case-insensitive (Windows paths may vary in case).
// R6.2: no match → caller should use UnmatchedLabel as the tile label.
// R6.3: the first match is returned; callers display each matching session
//
//	as a separate tile.
func MatchProject(cwd string, projects []registry.Project) (registry.Project, bool) {
	if cwd == "" {
		return registry.Project{}, false
	}

	cwdLower := strings.ToLower(cwd)
	for _, p := range projects {
		if strings.ToLower(p.Path) == cwdLower {
			return p, true
		}
	}
	return registry.Project{}, false
}
