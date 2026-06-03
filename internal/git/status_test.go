package git

import (
	"os/exec"
	"testing"
)

// ============================================================================
// parsePorcelain unit tests (direct function tests — pure, no seam needed)
// ============================================================================

// TestParsePorcelain_CleanRepo verifies clean output → zero counts, IsRepo=true.
func TestParsePorcelain_CleanRepo(t *testing.T) {
	// git status --porcelain=v1 -b for a clean repo on branch main.
	stdout := "## main...origin/main\n"
	s := parsePorcelain(stdout)
	if !s.IsRepo {
		t.Error("IsRepo = false, want true")
	}
	if s.Modified != 0 {
		t.Errorf("Modified = %d, want 0", s.Modified)
	}
	if s.Ahead != 0 {
		t.Errorf("Ahead = %d, want 0", s.Ahead)
	}
	if s.Behind != 0 {
		t.Errorf("Behind = %d, want 0", s.Behind)
	}
}

// TestParsePorcelain_ModifiedFiles verifies modified count from XY status lines.
func TestParsePorcelain_ModifiedFiles(t *testing.T) {
	// Three changed files: staged, unstaged, untracked.
	stdout := "## main...origin/main\n M foo.go\nM  bar.go\n?? new.go\n"
	s := parsePorcelain(stdout)
	if s.Modified != 3 {
		t.Errorf("Modified = %d, want 3", s.Modified)
	}
	if !s.IsRepo {
		t.Error("IsRepo = false, want true")
	}
}

// TestParsePorcelain_Ahead verifies ahead commit count from branch header.
func TestParsePorcelain_Ahead(t *testing.T) {
	stdout := "## main...origin/main [ahead 2]\n"
	s := parsePorcelain(stdout)
	if s.Ahead != 2 {
		t.Errorf("Ahead = %d, want 2", s.Ahead)
	}
	if s.Behind != 0 {
		t.Errorf("Behind = %d, want 0", s.Behind)
	}
}

// TestParsePorcelain_Behind verifies behind count.
func TestParsePorcelain_Behind(t *testing.T) {
	stdout := "## main...origin/main [behind 1]\n"
	s := parsePorcelain(stdout)
	if s.Behind != 1 {
		t.Errorf("Behind = %d, want 1", s.Behind)
	}
}

// TestParsePorcelain_AheadAndBehind verifies both counts parsed together.
func TestParsePorcelain_AheadAndBehind(t *testing.T) {
	stdout := "## feature...origin/feature [ahead 3, behind 2]\n"
	s := parsePorcelain(stdout)
	if s.Ahead != 3 {
		t.Errorf("Ahead = %d, want 3", s.Ahead)
	}
	if s.Behind != 2 {
		t.Errorf("Behind = %d, want 2", s.Behind)
	}
}

// TestParsePorcelain_NotARepo verifies non-branch header → IsRepo=false.
func TestParsePorcelain_NotARepo(t *testing.T) {
	// Non-zero exit from git status produces no ## line; we get empty / error output.
	s := parsePorcelain("")
	if s.IsRepo {
		t.Error("IsRepo = true, want false for empty output (non-repo)")
	}
}

// TestParsePorcelain_NoRemote verifies HEAD branch without remote tracking branch.
func TestParsePorcelain_NoRemote(t *testing.T) {
	// No ...origin/... means no remote tracking — ahead/behind both 0.
	stdout := "## main\n"
	s := parsePorcelain(stdout)
	if !s.IsRepo {
		t.Error("IsRepo = false, want true")
	}
	if s.Ahead != 0 || s.Behind != 0 {
		t.Errorf("Ahead=%d Behind=%d, want 0,0 for no remote", s.Ahead, s.Behind)
	}
}

// TestParsePorcelain_MultipleModifiedTypes verifies that D, R, C, A lines count as modified.
func TestParsePorcelain_MultipleModifiedTypes(t *testing.T) {
	// A=added, D=deleted, R=renamed, C=copied.
	stdout := "## main...origin/main\nA  added.go\nD  deleted.go\nR  old.go -> new.go\n"
	s := parsePorcelain(stdout)
	if s.Modified != 3 {
		t.Errorf("Modified = %d, want 3 (A+D+R)", s.Modified)
	}
}

// ============================================================================
// FormatGitIndicator unit tests (pure function — no seam needed)
// ============================================================================

// ============================================================================
// StatusReader integration tests via execCommand seam
// ============================================================================

// TestNewStatusReader_ReturnsNonNil verifies the constructor returns a non-nil reader.
func TestNewStatusReader_ReturnsNonNil(t *testing.T) {
	sr := NewStatusReader()
	if sr == nil {
		t.Fatal("NewStatusReader() returned nil")
	}
}

// TestStatusReader_GitNotAvailable verifies Status returns Available=false when git missing.
func TestStatusReader_GitNotAvailable(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) {
		return "", &exec.Error{Name: "git", Err: exec.ErrNotFound}
	}
	defer func() { lookPath = origLP; resetAvailableCache() }()

	sr := NewStatusReader()
	status, err := sr.Status("")
	if err != nil {
		t.Fatalf("Status() error = %v, want nil", err)
	}
	if status.Available {
		t.Error("Available = true, want false when git not on PATH")
	}
}

// TestStatusReader_CleanRepo verifies Status parses clean repo output.
func TestStatusReader_CleanRepo(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) { return "git.exe", nil }
	defer func() { lookPath = origLP; resetAvailableCache() }()

	// Inject canned clean repo output.
	stdout := "## main...origin/main\n"
	SetExecCommand(func(name string, arg ...string) *exec.Cmd {
		return fakeOutputCmd(t, stdout)
	})
	defer SetExecCommand(exec.Command)

	sr := NewStatusReader()
	status, err := sr.Status("")
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if !status.Available {
		t.Error("Available = false, want true")
	}
	if !status.IsRepo {
		t.Error("IsRepo = false, want true for clean repo")
	}
	if status.Modified != 0 || status.Ahead != 0 || status.Behind != 0 {
		t.Errorf("clean repo should have all-zero counts, got M=%d A=%d B=%d",
			status.Modified, status.Ahead, status.Behind)
	}
}

// TestStatusReader_NonZeroExit verifies IsRepo=false on non-zero exit (not a git repo).
func TestStatusReader_NonZeroExit(t *testing.T) {
	resetAvailableCache()
	origLP := lookPath
	lookPath = func(file string) (string, error) { return "git.exe", nil }
	defer func() { lookPath = origLP; resetAvailableCache() }()

	SetExecCommand(func(name string, arg ...string) *exec.Cmd {
		return failCmd()
	})
	defer SetExecCommand(exec.Command)

	sr := NewStatusReader()
	status, err := sr.Status("")
	if err != nil {
		t.Fatalf("Status() error: %v (should return nil, not error)", err)
	}
	if !status.Available {
		t.Error("Available = false, want true (git is available, just not a repo)")
	}
	if status.IsRepo {
		t.Error("IsRepo = true, want false on non-zero exit")
	}
}

// TestExtractCount_MissingKeyword verifies 0 is returned when keyword absent.
func TestExtractCount_MissingKeyword(t *testing.T) {
	got := extractCount("ahead 3", "behind")
	if got != 0 {
		t.Errorf("extractCount(no 'behind') = %d, want 0", got)
	}
}

// TestExtractCount_KeywordAtEnd verifies count parsing at string end.
func TestExtractCount_KeywordAtEnd(t *testing.T) {
	got := extractCount("behind 5", "behind")
	if got != 5 {
		t.Errorf("extractCount('behind 5') = %d, want 5", got)
	}
}

// ============================================================================
// TestFormatGitIndicator_AllCases covers all indicator combinations from design §10.
func TestFormatGitIndicator_AllCases(t *testing.T) {
	tests := []struct {
		name string
		s    Status
		want string
	}{
		{
			name: "unknown — git not available",
			s:    Status{Available: false},
			want: "?",
		},
		{
			name: "unknown — not a repo",
			s:    Status{Available: true, IsRepo: false},
			want: "?",
		},
		{
			name: "clean",
			s:    Status{Available: true, IsRepo: true, Modified: 0, Ahead: 0, Behind: 0},
			want: "✓",
		},
		{
			name: "modified only",
			s:    Status{Available: true, IsRepo: true, Modified: 3},
			want: "M3",
		},
		{
			name: "ahead only",
			s:    Status{Available: true, IsRepo: true, Ahead: 2},
			want: "↑2",
		},
		{
			name: "behind only",
			s:    Status{Available: true, IsRepo: true, Behind: 1},
			want: "↓1",
		},
		{
			name: "ahead + modified",
			s:    Status{Available: true, IsRepo: true, Ahead: 2, Modified: 3},
			want: "↑2M3",
		},
		{
			name: "ahead + behind",
			s:    Status{Available: true, IsRepo: true, Ahead: 2, Behind: 1},
			want: "↑2↓1",
		},
		{
			name: "ahead + behind + modified",
			s:    Status{Available: true, IsRepo: true, Ahead: 2, Behind: 1, Modified: 3},
			want: "↑2↓1M3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatGitIndicator(tt.s)
			if got != tt.want {
				t.Errorf("FormatGitIndicator() = %q, want %q", got, tt.want)
			}
		})
	}
}
