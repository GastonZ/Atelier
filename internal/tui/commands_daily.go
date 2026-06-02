package tui

// commands_daily.go — async tea.Cmd factories for the daily-driver-pack screens.
// Keeps commands.go small per coding-standards rule.
// All external IO (engram query, git exec, disk WalkDir) goes through these factories.
// UI never blocks — every factory returns a tea.Cmd that emits a typed tea.Msg.

import (
	"sort"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/disk"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/git"
	"github.com/gastonz/atelier/internal/registry"
)

// loadMemoryCmd queries engram for all observations of a project.
func loadMemoryCmd(c engram.Client, project string) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return memoryLoadedMsg{entries: nil, err: nil}
		}
		obs, err := c.ListByProject(project)
		return memoryLoadedMsg{entries: obs, err: err}
	}
}

// LoadMemoryCmdForTest exposes loadMemoryCmd for white-box testing.
func LoadMemoryCmdForTest(c engram.Client, project string) tea.Cmd {
	return loadMemoryCmd(c, project)
}

// loadMemoryDetailCmd fetches the full content of one observation by ID.
func loadMemoryDetailCmd(c engram.Client, id int64) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return memoryDetailLoadedMsg{}
		}
		obs, err := c.GetByID(id)
		return memoryDetailLoadedMsg{obs: obs, err: err}
	}
}

// LoadMemoryDetailCmdForTest exposes loadMemoryDetailCmd for testing.
func LoadMemoryDetailCmdForTest(c engram.Client, id int64) tea.Cmd {
	return loadMemoryDetailCmd(c, id)
}

// loadHistoryCmd merges and sorts git commits + SDD archives for a project.
// Sort order: date desc; ties: git before sdd (locked answer #9).
func loadHistoryCmd(c engram.Client, lr git.LogReader, project, repoPath string) tea.Cmd {
	return func() tea.Msg {
		entries := make([]HistoryEntry, 0, 35)

		// 1. Git commits
		if lr != nil {
			commits, _ := lr.Log(repoPath, 30)
			for _, cm := range commits {
				entries = append(entries, HistoryEntry{
					Source: "git",
					Date:   cm.Date,
					Title:  cm.Subject,
					Ref:    cm.Hash,
				})
			}
		}

		// 2. SDD archives
		if c != nil {
			archives, _ := c.ListSDDArchivesForProject(project)
			for _, a := range archives {
				entries = append(entries, HistoryEntry{
					Source: "sdd",
					Date:   a.Timestamp,
					Title:  a.Title,
					Ref:    strconv.FormatInt(a.ID, 10),
				})
			}
		}

		// 3. Sort: date desc, tie-break: git above sdd on same date.
		sort.SliceStable(entries, func(i, j int) bool {
			di := entries[i].Date.Format("2006-01-02")
			dj := entries[j].Date.Format("2006-01-02")
			if di != dj {
				return di > dj // date desc
			}
			// Same date: git comes before sdd.
			if entries[i].Source == "git" && entries[j].Source == "sdd" {
				return true
			}
			if entries[i].Source == "sdd" && entries[j].Source == "git" {
				return false
			}
			return false // both same source: preserve relative order
		})

		return historyLoadedMsg{entries: entries, err: nil}
	}
}

// LoadHistoryCmdForTest exposes loadHistoryCmd for testing.
func LoadHistoryCmdForTest(c engram.Client, lr git.LogReader, project, repoPath string) tea.Cmd {
	return loadHistoryCmd(c, lr, project, repoPath)
}

// loadHistoryDetailGitCmd fetches the output of `git show --stat <hash>`.
func loadHistoryDetailGitCmd(lr git.LogReader, repoPath, hash string) tea.Cmd {
	return func() tea.Msg {
		if lr == nil {
			return historyDetailLoadedMsg{}
		}
		body, err := lr.Show(repoPath, hash)
		return historyDetailLoadedMsg{body: body, err: err}
	}
}

// LoadHistoryDetailGitCmdForTest exposes loadHistoryDetailGitCmd for testing.
func LoadHistoryDetailGitCmdForTest(lr git.LogReader, repoPath, hash string) tea.Cmd {
	return loadHistoryDetailGitCmd(lr, repoPath, hash)
}

// loadHistoryDetailSDDCmd fetches the full content of an SDD archive observation.
func loadHistoryDetailSDDCmd(c engram.Client, id int64) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return historyDetailLoadedMsg{}
		}
		obs, err := c.GetByID(id)
		if err != nil {
			return historyDetailLoadedMsg{err: err}
		}
		return historyDetailLoadedMsg{body: obs.Content, err: nil}
	}
}

// LoadHistoryDetailSDDCmdForTest exposes loadHistoryDetailSDDCmd for testing.
func LoadHistoryDetailSDDCmdForTest(c engram.Client, id int64) tea.Cmd {
	return loadHistoryDetailSDDCmd(c, id)
}

// loadGitStatusCmd fans out git status reads for all projects in parallel.
// Each per-tomo read has a 500ms timeout (enforced by execStatusReader.Status).
// Partial results are returned on timeout — per-tomo errors → Status{Available:true,IsRepo:false}.
func loadGitStatusCmd(sr git.StatusReader, projects []registry.Project) tea.Cmd {
	return func() tea.Msg {
		if sr == nil {
			return gitStatusLoadedMsg{statuses: map[string]git.Status{}}
		}

		result := make(map[string]git.Status, len(projects))
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, p := range projects {
			p := p // capture loop var
			wg.Add(1)
			go func() {
				defer wg.Done()
				s, err := sr.Status(p.Path)
				if err != nil {
					s = git.Status{Available: true, IsRepo: false}
				}
				mu.Lock()
				result[p.Path] = s
				mu.Unlock()
			}()
		}

		// Outer ceiling: 5s for the whole fan-out.
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// all done
		case <-time.After(5 * time.Second):
			// partial results — whatever we collected is returned.
		}

		mu.Lock()
		defer mu.Unlock()
		return gitStatusLoadedMsg{statuses: result, err: nil}
	}
}

// LoadGitStatusCmdForTest exposes loadGitStatusCmd for testing.
func LoadGitStatusCmdForTest(sr git.StatusReader, projects []registry.Project) tea.Cmd {
	return loadGitStatusCmd(sr, projects)
}

// loadDiskUsageCmd computes disk usage for engram DB, claude projects, and per-tomo dirs.
// Sequential WalkDir for predictability. Runs in tea.Cmd — UI shows CopyDiskLoading.
func loadDiskUsageCmd(projects []registry.Project) tea.Cmd {
	return func() tea.Msg {
		var engramBytes int64
		var claudeBytes int64
		perTomo := make(map[string]int64, len(projects))

		// 1. Engram DB size.
		engramBytes, _ = disk.EngramDBSize()

		// 2. Total ~/.claude/projects/ size.
		claudeProjectsDir, err := disk.ClaudeProjectsDir()
		if err == nil && claudeProjectsDir != "" {
			claudeBytes, _ = disk.WalkSize(claudeProjectsDir)
		}

		// 3. Per-tomo claude-projects size.
		for _, p := range projects {
			tomoDir, err := disk.ClaudeProjectsDirForPath(p.Path)
			if err != nil || tomoDir == "" {
				perTomo[p.ID] = 0
				continue
			}
			size, _ := disk.WalkSize(tomoDir)
			perTomo[p.ID] = size
		}

		return diskUsageLoadedMsg{
			engramBytes: engramBytes,
			claudeBytes: claudeBytes,
			perTomo:     perTomo,
			err:         nil,
		}
	}
}

// LoadDiskUsageCmdForTest exposes loadDiskUsageCmd for testing.
func LoadDiskUsageCmdForTest(projects []registry.Project) tea.Cmd {
	return loadDiskUsageCmd(projects)
}
