// Package nowplaying reports the media track currently playing on the system.
//
// On Windows it reads the System Media Transport Controls (SMTC / GSMTC) — the
// same source that powers the OS media flyout — so Spotify, browser audio
// (YouTube et al.), and any SMTC-aware app are reported uniformly: title,
// artist, source app, and play/pause state.
//
// The package exposes a Provider interface so the TUI depends on an abstraction
// (nil-safe at the call site) and tests inject fakes — no OS calls in unit tests.
package nowplaying

import (
	"encoding/json"
	"strings"
)

// Track is a snapshot of the current media session.
//
// Present is the gate the UI uses: when false, there is nothing worth drawing.
// Playing distinguishes an actively-playing track from a paused one.
type Track struct {
	Title   string // track / video title
	Artist  string // artist or channel; may be empty
	Source  string // friendly app name: "Spotify", "Chrome", "Edge", …
	Playing bool   // true when PlaybackStatus == Playing
	Present bool   // true when a media session with a non-empty title exists
}

// Provider returns the current now-playing track.
//
// Implementations must never block the UI for long; the TUI always calls them
// from a tea.Cmd goroutine. A nil Provider is valid — callers guard for it.
type Provider interface {
	// Current returns the current Track. A returned error means the lookup
	// failed (e.g. the platform helper errored); Track is then zero-valued.
	// "Nothing is playing" is NOT an error — it returns Track{Present: false}, nil.
	Current() (Track, error)
}

// smtcPayload mirrors the JSON emitted by the Windows SMTC helper script.
type smtcPayload struct {
	Title  string `json:"Title"`
	Artist string `json:"Artist"`
	App    string `json:"App"`
	Status string `json:"Status"`
}

// sentinelNoSession is emitted by the helper when the SMTC API is reachable
// but no app currently owns a media session.
const sentinelNoSession = "NO_SESSION"

// errorPrefix is emitted by the helper when the SMTC call itself failed.
const errorPrefix = "ERROR:"

// parseTrack converts raw helper output into a Track.
//
// It is the pure core of the package — every branch is unit-tested. It accepts:
//   - "" / "NO_SESSION…"        → Track{Present: false}, nil
//   - "ERROR: <msg>"            → Track{}, error
//   - {"Title":…,"Status":…}    → populated Track
func parseTrack(raw string) (Track, error) {
	s := strings.TrimSpace(raw)
	if s == "" || strings.HasPrefix(s, sentinelNoSession) {
		return Track{Present: false}, nil
	}
	if strings.HasPrefix(s, errorPrefix) {
		return Track{}, &HelperError{Message: strings.TrimSpace(strings.TrimPrefix(s, errorPrefix))}
	}

	var p smtcPayload
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		// Unrecognised output is treated as "nothing playing" rather than a hard
		// error — the UI should degrade quietly, never crash on a stray line.
		return Track{Present: false}, nil
	}

	title := strings.TrimSpace(p.Title)
	if title == "" {
		return Track{Present: false}, nil
	}

	return Track{
		Title:   title,
		Artist:  strings.TrimSpace(p.Artist),
		Source:  friendlySource(p.App),
		Playing: strings.EqualFold(strings.TrimSpace(p.Status), "Playing"),
		Present: true,
	}, nil
}

// friendlySource maps a raw SMTC SourceAppUserModelId to a human label.
//
// SMTC reports the owning *application*, not the website — so browser audio
// (a YouTube video, for instance) is honestly labelled by its browser while the
// video name lives in Track.Title. Unknown apps fall back to their trimmed id.
func friendlySource(appID string) string {
	id := strings.ToLower(strings.TrimSpace(appID))
	if id == "" {
		return ""
	}
	switch {
	case strings.Contains(id, "spotify"):
		return "Spotify"
	case strings.Contains(id, "msedge"), strings.Contains(id, "microsoft.edge"):
		return "Edge"
	case strings.Contains(id, "chrome"):
		return "Chrome"
	case strings.Contains(id, "firefox"):
		return "Firefox"
	case strings.Contains(id, "brave"):
		return "Brave"
	case strings.Contains(id, "opera"):
		return "Opera"
	case strings.Contains(id, "zen"):
		return "Zen"
	case strings.Contains(id, "vlc"):
		return "VLC"
	case strings.Contains(id, "mpv"):
		return "mpv"
	case strings.Contains(id, "foobar"):
		return "foobar2000"
	case strings.Contains(id, "apple") && strings.Contains(id, "music"):
		return "Apple Music"
	case strings.Contains(id, "itunes"):
		return "iTunes"
	}
	// Fall back to the raw id, stripped of a trailing ".exe" for readability.
	trimmed := strings.TrimSpace(appID)
	return strings.TrimSuffix(trimmed, ".exe")
}

// HelperError reports a failure inside the platform now-playing helper.
type HelperError struct {
	Message string
}

func (e *HelperError) Error() string {
	if e.Message == "" {
		return "nowplaying: helper failed"
	}
	return "nowplaying: " + e.Message
}
