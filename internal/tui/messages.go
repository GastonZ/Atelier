package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/git"
	"github.com/gastonz/atelier/internal/nowplaying"
	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/transcripts"
)

// projectsLoadedMsg is dispatched after registry.List() completes asynchronously.
type projectsLoadedMsg struct {
	projects []registry.Project
	err      error
}

// loadProjectsCmd reads the registry asynchronously and returns a tea.Cmd.
// It is exported for test helpers (DrainProjectsLoaded) but remains lowercase-callable.
func loadProjectsCmd(reg registry.Registry) tea.Cmd {
	return func() tea.Msg {
		ps, err := reg.List()
		return projectsLoadedMsg{projects: ps, err: err}
	}
}

// actionDoneMsg is dispatched after an Opener or Clipboard action completes.
type actionDoneMsg struct {
	flash string // user-facing fantasy-themed result copy
	err   error
}

// runOpenClaudeCmd executes OpenInClaudeCode and Touch(id) asynchronously.
func runOpenClaudeCmd(op Openers, reg registry.Registry, id, path string) tea.Cmd {
	return func() tea.Msg {
		err := op.OpenInClaudeCode(path)
		if err != nil {
			return actionDoneMsg{flash: "Un dragón rugió: " + err.Error(), err: err}
		}
		touchErr := reg.Touch(id)
		if touchErr != nil {
			return actionDoneMsg{flash: "Tomo abierto (no se pudo registrar la lectura)", err: nil}
		}
		return actionDoneMsg{flash: "Tomo abierto en Claude Code"}
	}
}

// runPowerShellCmd executes SpawnPowerShell asynchronously (does NOT touch registry).
func runPowerShellCmd(op Openers, path string) tea.Cmd {
	return func() tea.Msg {
		err := op.SpawnPowerShell(path)
		if err != nil {
			return actionDoneMsg{flash: "Un dragón rugió: " + err.Error(), err: err}
		}
		return actionDoneMsg{flash: "PowerShell invocado en el tomo"}
	}
}

// runCopyPathCmd writes path to the clipboard asynchronously.
func runCopyPathCmd(cb Clipboards, path string) tea.Cmd {
	return func() tea.Msg {
		err := cb.WriteAll(path)
		if err != nil {
			return actionDoneMsg{flash: "Un dragón rugió: " + err.Error(), err: err}
		}
		return actionDoneMsg{flash: "Sendero copiado al pergamino"}
	}
}

// Openers is a local alias to avoid circular naming; mirrors actions.Opener.
type Openers interface {
	OpenInClaudeCode(projectPath string) error
	SpawnPowerShell(projectPath string) error
	OpenInVSCode(projectPath string) error
}

// Clipboards is a local alias; mirrors actions.Clipboard.
type Clipboards interface {
	WriteAll(text string) error
}

// ============================================================================
// Agent monitor messages
// ============================================================================

// agentSessionsLoadedMsg is dispatched after Scanner.ListActive() completes.
type agentSessionsLoadedMsg struct {
	sessions []transcripts.Session
	err      error
}

// MakeAgentSessionsLoadedMsg constructs an agentSessionsLoadedMsg for tests.
func MakeAgentSessionsLoadedMsg(sessions []transcripts.Session, err error) tea.Msg {
	return agentSessionsLoadedMsg{sessions: sessions, err: err}
}

// agentEventMsg is dispatched when the watcher delivers a new event.
type agentEventMsg struct {
	event transcripts.Event
}

// MakeAgentEventMsg constructs an agentEventMsg for tests.
func MakeAgentEventMsg(event transcripts.Event) tea.Msg {
	return agentEventMsg{event: event}
}

// agentWatcherErrMsg is dispatched when the watcher encounters an error.
type agentWatcherErrMsg struct {
	err error
}

// replayLoadedMsg is dispatched after LoadEvents() completes for a replay entry.
type replayLoadedMsg struct {
	sessionID string
	events    []transcripts.Event
	err       error
}

// MakeReplayLoadedMsg constructs a replayLoadedMsg for tests.
func MakeReplayLoadedMsg(sessionID string, events []transcripts.Event, err error) tea.Msg {
	return replayLoadedMsg{sessionID: sessionID, events: events, err: err}
}

// replayTickMsg is dispatched by the replay ticker on each tick.
type replayTickMsg struct{}

// configLoadedMsg is dispatched after LoadAtelierConfig() completes.
type configLoadedMsg struct {
	cfg config.AtelierConfig
	err error
}

// pollingTickMsg is dispatched by the polling ticker on each tick.
type pollingTickMsg struct{}

// nowPlayingInterval is how often the welcome screen refreshes the now-playing card.
const nowPlayingInterval = 3 * time.Second

// nowPlayingTickMsg is dispatched by the now-playing ticker on each tick.
type nowPlayingTickMsg struct{}

// nowPlayingLoadedMsg carries the latest now-playing snapshot.
type nowPlayingLoadedMsg struct {
	track nowplaying.Track
	err   error
}

// MakeNowPlayingLoadedMsg constructs a nowPlayingLoadedMsg for tests.
func MakeNowPlayingLoadedMsg(track nowplaying.Track, err error) tea.Msg {
	return nowPlayingLoadedMsg{track: track, err: err}
}

// MakeNowPlayingTickMsg constructs a nowPlayingTickMsg for tests.
func MakeNowPlayingTickMsg() tea.Msg {
	return nowPlayingTickMsg{}
}

// nowPlayingTickCmd schedules a single now-playing tick.
func nowPlayingTickCmd() tea.Cmd {
	return tea.Tick(nowPlayingInterval, func(_ time.Time) tea.Msg {
		return nowPlayingTickMsg{}
	})
}

// animInterval drives the waveform animation (~15 fps).
const animInterval = 66 * time.Millisecond

// animTickMsg is dispatched by the waveform animation ticker on each frame.
type animTickMsg struct{}

// MakeAnimTickMsg constructs an animTickMsg for tests.
func MakeAnimTickMsg() tea.Msg { return animTickMsg{} }

// animTickCmd schedules a single animation frame tick.
func animTickCmd() tea.Cmd {
	return tea.Tick(animInterval, func(_ time.Time) tea.Msg {
		return animTickMsg{}
	})
}

// loadNowPlayingCmd queries the provider for the current track.
// A nil provider yields an empty (not-present) track — never an error.
func loadNowPlayingCmd(p nowplaying.Provider) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return nowPlayingLoadedMsg{track: nowplaying.Track{Present: false}}
		}
		t, err := p.Current()
		return nowPlayingLoadedMsg{track: t, err: err}
	}
}

// MakePollingTickMsg constructs a pollingTickMsg for tests.
func MakePollingTickMsg() tea.Msg {
	return pollingTickMsg{}
}

// MakeReplayTickMsg constructs a replayTickMsg for tests.
func MakeReplayTickMsg() tea.Msg {
	return replayTickMsg{}
}

// ============================================================================
// Agent monitor commands
// ============================================================================

// loadAgentSessionsCmdWithCfg calls scanner.ListActive and dispatches agentSessionsLoadedMsg.
func loadAgentSessionsCmdWithCfg(scanner transcripts.Scanner, cfg config.AtelierConfig) tea.Cmd {
	return func() tea.Msg {
		sessions, err := scanner.ListActive(cfg.ActiveWindow())
		return agentSessionsLoadedMsg{sessions: sessions, err: err}
	}
}

// startWatcherCmdFn starts the watcher for the given session root paths.
// It returns (drainCmd, cancel, ch):
//   - drainCmd: the first drain command (blocks on ch and emits agentEventMsg)
//   - cancel: closes the watcher (call on screen exit)
//   - ch: the event channel; store on Model so handleAgentEvent can re-drain
//     directly without calling Watch again (prevents goroutine leaks)
func startWatcherCmdFn(watcher transcripts.Watcher, paths []string) (tea.Cmd, func(), <-chan transcripts.Event) {
	ch, err := watcher.Watch(paths)
	if err != nil {
		// Return a cmd that delivers the error; cancel is a no-op.
		return func() tea.Msg {
			return agentWatcherErrMsg{err: err}
		}, func() { _ = watcher.Close() }, nil
	}

	// drainCmd blocks on the channel and emits agentEventMsg.
	// After each event the handler re-queues a new drainAgentChannelCmd(ch)
	// using the stored channel — Watch is NEVER called again after setup.
	drainCmd := drainAgentChannelCmd(ch)
	cancel := func() { _ = watcher.Close() }
	return drainCmd, cancel, ch
}

// drainAgentChannelCmd returns a tea.Cmd that blocks on ch and emits agentEventMsg.
// The handler stores ch on the Model and calls this directly (not via Watch).
func drainAgentChannelCmd(ch <-chan transcripts.Event) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return nil
		}
		return agentEventMsg{event: evt}
	}
}

// loadReplayEventsCmd loads events for a session to start replay.
func loadReplayEventsCmd(scanner transcripts.Scanner, sessionID string) tea.Cmd {
	return func() tea.Msg {
		events, err := scanner.LoadEvents(sessionID)
		return replayLoadedMsg{sessionID: sessionID, events: events, err: err}
	}
}

// pollingTickCmd schedules a single polling tick after cfg.PollingInterval().
func pollingTickCmd(cfg config.AtelierConfig) tea.Cmd {
	return tea.Tick(cfg.PollingInterval(), func(_ time.Time) tea.Msg {
		return pollingTickMsg{}
	})
}

// replayTickCmd schedules a single replay tick at the current speed.
func replayTickCmd(speed float64) tea.Cmd {
	interval := replayInterval(speed)
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return replayTickMsg{}
	})
}

// ============================================================================
// Daily driver pack messages
// ============================================================================

// memoryLoadedMsg is dispatched after loadMemoryCmd completes.
type memoryLoadedMsg struct {
	entries []engram.Observation
	err     error
}

// MakeMemoryLoadedMsg constructs a memoryLoadedMsg for tests.
func MakeMemoryLoadedMsg(entries []engram.Observation, err error) tea.Msg {
	return memoryLoadedMsg{entries: entries, err: err}
}

// memoryDetailLoadedMsg is dispatched after loadMemoryDetailCmd completes.
type memoryDetailLoadedMsg struct {
	obs engram.Observation
	err error
}

// MakeMemoryDetailLoadedMsg constructs a memoryDetailLoadedMsg for tests.
func MakeMemoryDetailLoadedMsg(obs engram.Observation, err error) tea.Msg {
	return memoryDetailLoadedMsg{obs: obs, err: err}
}

// historyLoadedMsg is dispatched after loadHistoryCmd completes.
type historyLoadedMsg struct {
	entries []HistoryEntry
	err     error
}

// MakeHistoryLoadedMsg constructs a historyLoadedMsg for tests.
func MakeHistoryLoadedMsg(entries []HistoryEntry, err error) tea.Msg {
	return historyLoadedMsg{entries: entries, err: err}
}

// historyDetailLoadedMsg is dispatched after a history detail cmd completes.
type historyDetailLoadedMsg struct {
	body string
	err  error
}

// MakeHistoryDetailLoadedMsg constructs a historyDetailLoadedMsg for tests.
func MakeHistoryDetailLoadedMsg(body string, err error) tea.Msg {
	return historyDetailLoadedMsg{body: body, err: err}
}

// gitStatusLoadedMsg is dispatched after the git status fan-out completes.
type gitStatusLoadedMsg struct {
	statuses map[string]git.Status // keyed by tomo.Path
	err      error
}

// MakeGitStatusLoadedMsg constructs a gitStatusLoadedMsg for tests.
func MakeGitStatusLoadedMsg(statuses map[string]git.Status, err error) tea.Msg {
	return gitStatusLoadedMsg{statuses: statuses, err: err}
}

// diskUsageLoadedMsg is dispatched after loadDiskUsageCmd completes.
type diskUsageLoadedMsg struct {
	engramBytes int64
	claudeBytes int64
	perTomo     map[string]int64
	err         error
}

// MakeDiskUsageLoadedMsg constructs a diskUsageLoadedMsg for tests.
func MakeDiskUsageLoadedMsg(engramBytes, claudeBytes int64, perTomo map[string]int64, err error) tea.Msg {
	return diskUsageLoadedMsg{engramBytes: engramBytes, claudeBytes: claudeBytes, perTomo: perTomo, err: err}
}

// runOpenVSCodeCmd executes OpenInVSCode and Touch(id) asynchronously.
func runOpenVSCodeCmd(op Openers, reg registry.Registry, id, path string) tea.Cmd {
	return func() tea.Msg {
		err := op.OpenInVSCode(path)
		if err != nil {
			return actionDoneMsg{flash: CopyVSCodeMissing, err: err}
		}
		touchErr := reg.Touch(id)
		if touchErr != nil {
			return actionDoneMsg{flash: "Tomo abierto en VS Code (no se pudo registrar la lectura)", err: nil}
		}
		return actionDoneMsg{flash: "Tomo abierto en VS Code"}
	}
}

// rootPathsOf extracts the root .jsonl paths from a slice of active sessions.
// Only root sessions (not sub-sessions) are included — the watcher monitors directories
// so sub-agent files are covered by their parent directory watch.
func rootPathsOf(sessions []transcripts.Session) []string {
	paths := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if s.RootPath != "" {
			paths = append(paths, s.RootPath)
		}
	}
	return paths
}
