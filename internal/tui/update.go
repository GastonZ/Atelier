package tui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/actions"
	"github.com/gastonz/atelier/internal/config"
	"github.com/gastonz/atelier/internal/registry"
	"github.com/gastonz/atelier/internal/transcripts"
)

// Update is the Bubble Tea message handler.
// It processes all incoming messages and returns a new Model (immutable — value semantics).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m = m.initOrUpdateProjectList()
		return m, nil

	case projectsLoadedMsg:
		if msg.err != nil {
			m.ActionFlash = "Los tomos están sellados: " + msg.err.Error()
			return m, nil
		}
		m.projects = msg.projects
		m = m.initOrUpdateProjectList()
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.ActionFlash = "Un dragón rugió: " + msg.err.Error()
		} else {
			m.ActionFlash = msg.flash
		}
		if m.Screen == ScreenProjectActions {
			m.Screen = ScreenProjects
		}
		return m, loadProjectsCmd(m.registry)

	case agentSessionsLoadedMsg:
		return m.handleAgentSessionsLoaded(msg)

	case agentEventMsg:
		return m.handleAgentEvent(msg)

	case agentWatcherErrMsg:
		m.AgentFlash = fmt.Sprintf(CopyWatcherError, msg.err.Error())
		return m, nil

	case replayLoadedMsg:
		return m.handleReplayLoaded(msg)

	case replayTickMsg:
		return m.handleReplayTick()

	case pollingTickMsg:
		return m.handlePollingTick()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleAgentSessionsLoaded processes the result of Scanner.ListActive().
// It enriches each session with ProjectID/ProjectName by matching its cwd
// against the registered projects (R6.1). Without this step, every session
// renders as "Sin tomo registrado" because the Scanner does not know about
// the registry — it only discovers files.
func (m Model) handleAgentSessionsLoaded(msg agentSessionsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.AgentFlash = "Los tomos están sellados: " + msg.err.Error()
		return m, nil
	}

	// Pull the registered projects for cwd matching. List() is a cheap disk
	// read; doing it on each scan avoids stale-cache issues when the user
	// adds/removes tomos while the monitor is open.
	var projects []registry.Project
	if m.registry != nil {
		if list, err := m.registry.List(); err == nil {
			projects = list
		}
	}

	sessions := make([]transcripts.Session, len(msg.sessions))
	for i, sess := range msg.sessions {
		sessions[i] = enrichSessionWithProject(sess, projects)
	}

	// Sort mtime descending (R3.4)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastEventTime.After(sessions[j].LastEventTime)
	})
	m.agentSessions = sessions
	// Clamp cursor to valid range
	if m.AgentTileCursor >= len(m.agentSessions) && len(m.agentSessions) > 0 {
		m.AgentTileCursor = len(m.agentSessions) - 1
	}
	return m, nil
}

// enrichSessionWithProject returns a copy of sess with ProjectID/ProjectName
// populated from a cwd match against projects. Sub-sessions are enriched
// independently because a sub-agent may inherit the parent's cwd or run with
// a different one (rare, but observed).
func enrichSessionWithProject(sess transcripts.Session, projects []registry.Project) transcripts.Session {
	if proj, ok := transcripts.MatchProject(sess.Cwd, projects); ok {
		sess.ProjectID = proj.ID
		sess.ProjectName = proj.Name
	}
	if len(sess.SubSessions) > 0 {
		subs := make([]transcripts.Session, len(sess.SubSessions))
		for i, sub := range sess.SubSessions {
			// Sub-agents that have no cwd of their own inherit the parent's.
			if sub.Cwd == "" {
				sub.Cwd = sess.Cwd
			}
			if proj, ok := transcripts.MatchProject(sub.Cwd, projects); ok {
				sub.ProjectID = proj.ID
				sub.ProjectName = proj.Name
			}
			subs[i] = sub
		}
		sess.SubSessions = subs
	}
	return sess
}

// handleAgentEvent applies a single live event to the appropriate session.
// Buffer cap: at most 200 events retained per session (R9.2).
func (m Model) handleAgentEvent(msg agentEventMsg) (tea.Model, tea.Cmd) {
	evt := msg.event
	if evt == nil {
		// Re-queue the drain command to wait for the next event
		if m.agentWatcher != nil {
			// drain is re-queued externally via waitForAgentEventCmd pattern
		}
		return m, nil
	}

	targetID := evt.SessionID()
	sessions := make([]transcripts.Session, len(m.agentSessions))
	copy(sessions, m.agentSessions)

	for i, s := range sessions {
		if s.ID == targetID {
			updated := applyEventToSession(s, evt, m.priceTable)
			sessions[i] = updated
			break
		}
	}

	// Re-sort by mtime desc
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastEventTime.After(sessions[j].LastEventTime)
	})
	m.agentSessions = sessions

	// Re-queue drain cmd using the stored channel (never calls Watch again).
	// Calling Watch(nil) on the production fsnotifyWatcher would create a new
	// channel, reset fileState, and start new goroutines — leaking the old ones.
	// Instead, agentWatcherCh is stored once at watcher-start and reused here.
	var drainCmd tea.Cmd
	if m.agentWatcherCh != nil {
		drainCmd = drainAgentChannelCmd(m.agentWatcherCh)
	}
	return m, drainCmd
}

// applyEventToSession returns a new Session with the event appended and
// the buffer capped at 200. Cost is incremented for AssistantEvents.
// This is a pure function (no mutation of the original Session).
func applyEventToSession(s transcripts.Session, evt transcripts.Event, prices transcripts.PriceTable) transcripts.Session {
	const maxEvents = 200

	// Update last event time
	if evt.Timestamp().After(s.LastEventTime) {
		s.LastEventTime = evt.Timestamp()
	}

	// Accumulate cost for assistant events
	if ae, ok := evt.(*transcripts.AssistantEvent); ok && prices != nil {
		cost, known := prices.Cost(ae.Model, ae.Usage.InputTokens, ae.Usage.OutputTokens,
			ae.Usage.CacheCreationTokens, ae.Usage.CacheReadTokens)
		s.AccumulatedUSD += cost
		if !known {
			// Flash is handled at the caller level (handleAgentEvent)
			_ = ae.Model
		}
	}

	// Append event and cap buffer
	events := make([]transcripts.Event, len(s.Events)+1)
	copy(events, s.Events)
	events[len(s.Events)] = evt
	if len(events) > maxEvents {
		events = events[len(events)-maxEvents:]
	}
	s.Events = events

	return s
}

// handlePollingTick refreshes sessions on each polling tick (always-on fallback R2.2).
func (m Model) handlePollingTick() (tea.Model, tea.Cmd) {
	if m.Screen != ScreenAgentMonitor {
		return m, nil
	}
	var loadCmd tea.Cmd
	if m.agentScanner != nil {
		loadCmd = loadAgentSessionsCmdWithCfg(m.agentScanner, m.atelierCfg)
	}
	// Re-queue the polling ticker
	return m, tea.Batch(loadCmd, pollingTickCmd(m.atelierCfg))
}

// handleReplayLoaded initialises the replay controller from a loaded event set.
func (m Model) handleReplayLoaded(msg replayLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.AgentFlash = "Error cargando la crónica: " + msg.err.Error()
		m.Screen = ScreenAgentZoom
		return m, nil
	}

	m.replayEvents = msg.events
	m.replayCtrl = transcripts.NewReplay(msg.events)
	m.ReplayCursor = 0
	m.ReplayPaused = false
	m.ReplaySpeed = 1.0
	if m.replayCtrl != nil {
		m.replayCtrl.SetSpeed(m.ReplaySpeed)
	}

	// Schedule first replay tick
	return m, replayTickCmd(m.ReplaySpeed)
}

// handleReplayTick advances the replay cursor by one event when not paused.
func (m Model) handleReplayTick() (tea.Model, tea.Cmd) {
	if m.Screen != ScreenAgentReplay || m.ReplayPaused {
		return m, nil
	}
	if m.replayCtrl != nil {
		advanced := m.replayCtrl.Next()
		m.ReplayCursor = m.replayCtrl.Cursor()
		if !advanced {
			// Reached end — auto-pause
			m.ReplayPaused = true
			m.replayCtrl.Pause()
			return m, nil
		}
	}
	// Re-queue next tick
	return m, replayTickCmd(m.ReplaySpeed)
}

// Run creates and starts the Bubble Tea program with alt-screen mode.
// Alt-screen is mandatory for Windows Terminal / PowerShell rendering.
// Called by cmd/atelier/main.go after wiring the registry and actions.
func Run(reg registry.Registry, op actions.Opener, cb actions.Clipboard) error {
	m := New(reg, op, cb)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunWithMonitor creates and starts the Bubble Tea program wired with all
// agent-monitor dependencies. This is the production entry point used by main.go.
// It mirrors Run but uses NewWithMonitor as the model constructor.
func RunWithMonitor(
	reg registry.Registry,
	op actions.Opener,
	cb actions.Clipboard,
	scanner transcripts.Scanner,
	watcher transcripts.Watcher,
	prices transcripts.PriceTable,
	cfg config.AtelierConfig,
) error {
	m := NewWithMonitor(reg, op, cb, scanner, watcher, prices, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// listSelectedProject returns the currently highlighted project from the bubbles/list.
// Returns nil if the list is empty or no item is selected.
func (m Model) listSelectedProject() *list.Item {
	if len(m.projects) == 0 {
		return nil
	}
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	return &item
}
