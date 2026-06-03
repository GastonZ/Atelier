package tui

// memory_link.go — maps an Atelier project to an Engram project key.
//
// Engram keys observations by the project key the saver used (usually the folder
// basename), while the registry name is a free-form label — so filtering memory
// by the registry name showed little or nothing. Resolution order:
//   1. the explicit EngramProject override set via this picker, else
//   2. the path basename heuristic (matched case-insensitively by the queries).
// The picker lists the REAL engram keys with counts so users map to truth, not a
// guess.

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gastonz/atelier/internal/engram"
	"github.com/gastonz/atelier/internal/registry"
)

// engramKeyFor resolves the engram project key for a project: the explicit
// override when set, otherwise the path basename.
func engramKeyFor(p registry.Project) string {
	if strings.TrimSpace(p.EngramProject) != "" {
		return p.EngramProject
	}
	return filepath.Base(p.Path)
}

// currentEngramKeyForLink returns the engram key currently resolved for the
// project being linked (used to mark it in the picker).
func (m Model) currentEngramKeyForLink() string {
	if p := m.findProject(m.memoryLinkProjID); p != nil {
		return engramKeyFor(*p)
	}
	return ""
}

// --- command + message ---

type engramProjectsLoadedMsg struct {
	stats []engram.ProjectStat
	err   error
}

// loadEngramProjectsCmd fetches the distinct engram project keys + counts.
func loadEngramProjectsCmd(c engram.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := c.ListProjects()
		return engramProjectsLoadedMsg{stats: stats, err: err}
	}
}

// LoadEngramProjectsCmdForTest exposes loadEngramProjectsCmd for white-box tests.
func LoadEngramProjectsCmdForTest(c engram.Client) tea.Cmd { return loadEngramProjectsCmd(c) }

func (m Model) handleEngramProjectsLoaded(msg engramProjectsLoadedMsg) (tea.Model, tea.Cmd) {
	m.memoryLinkLoading = false
	if msg.err != nil {
		m.memoryLinkErr = "Engram: " + msg.err.Error()
		return m, nil
	}
	m.memoryLinkErr = ""
	m.memoryLinkStats = msg.stats

	// Land the cursor on the currently-linked key, if present.
	cur := m.currentEngramKeyForLink()
	m.MemoryLinkCursor = 0
	for i, s := range msg.stats {
		if strings.EqualFold(s.Key, cur) {
			m.MemoryLinkCursor = i
			break
		}
	}
	return m, nil
}

// --- key handler ---

// handleMemoryLinkKeys handles the engram link picker (ScreenMemoryLink).
func (m Model) handleMemoryLinkKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.memoryLinkStats)
	switch {
	case msg.Type == tea.KeyEsc || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
		m.Screen = ScreenProjectActions
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "j":
		fallthrough
	case msg.Type == tea.KeyDown:
		if m.MemoryLinkCursor < n-1 {
			m.MemoryLinkCursor++
		}
		return m, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "k":
		fallthrough
	case msg.Type == tea.KeyUp:
		if m.MemoryLinkCursor > 0 {
			m.MemoryLinkCursor--
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		if n == 0 {
			return m, nil
		}
		key := m.memoryLinkStats[m.MemoryLinkCursor].Key
		if err := m.registry.SetEngramProject(m.memoryLinkProjID, key); err != nil {
			m.ActionFlash = "No se pudo vincular: " + err.Error()
			m.Screen = ScreenProjectActions
			return m, nil
		}
		// Reflect the new mapping in the in-memory project list (immutable copy)
		// so the next "Ver memoria" uses it without a disk reload.
		projs := append([]registry.Project{}, m.projects...)
		for i := range projs {
			if projs[i].ID == m.memoryLinkProjID {
				projs[i].EngramProject = key
				break
			}
		}
		m.projects = projs
		m.ActionFlash = "Memoria vinculada a " + key
		m.Screen = ScreenProjectActions
		return m, nil
	}
	return m, nil
}

// --- view ---

// viewMemoryLink renders the engram link picker (ScreenMemoryLink).
func (m Model) viewMemoryLink() string {
	projName := m.memoryLinkProjID
	if p := m.findProject(m.memoryLinkProjID); p != nil {
		projName = p.Name
	}
	title := TitleBarStyle.Render("=== Vincular memoria: " + projName + " ===")
	sub := SubtitleStyle.Render(`  Elegí la clave de Engram que "Ver memoria" debe mostrar para este tomo`)

	var rows []string
	switch {
	case m.memoryLinkLoading:
		rows = append(rows, "", HintStyle.Render("  Consultando Engram..."), "")
	case m.memoryLinkErr != "":
		rows = append(rows, "", ErrorInlineStyle.Render("  "+m.memoryLinkErr), "")
	case len(m.memoryLinkStats) == 0:
		rows = append(rows, "", "  Engram no tiene proyectos.", "")
	default:
		cur := m.currentEngramKeyForLink()
		for i, s := range m.memoryLinkStats {
			marker := " "
			if strings.EqualFold(s.Key, cur) {
				marker = "•" // currently linked
			}
			row := fmt.Sprintf("%s %s  (%d)", marker, s.Key, s.Count)
			if i == m.MemoryLinkCursor {
				rows = append(rows, SelectedRowStyle.Render("  [*] "+row))
			} else {
				rows = append(rows, "  [ ] "+row)
			}
		}
	}

	parts := []string{title, sub, ""}
	parts = append(parts, rows...)
	parts = append(parts, "")
	footer := FooterHintStyle.Render("  j/k: navegar  ·  enter: vincular  ·  esc: volver")
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
