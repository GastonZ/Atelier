package tui_test

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gastonz/atelier/internal/nowplaying"
	"github.com/gastonz/atelier/internal/tui"
)

// fakeAnalyzer is a deterministic audio.Analyzer for tests.
type fakeAnalyzer struct{ levels []float64 }

func (f fakeAnalyzer) Levels(n int) []float64 { return f.levels }
func (f fakeAnalyzer) Close() error           { return nil }

// updateModel feeds a message through Update and returns the resulting Model.
func updateModel(t *testing.T, m tui.Model, msg tea.Msg) tui.Model {
	t.Helper()
	result, _ := m.Update(msg)
	got, ok := result.(tui.Model)
	if !ok {
		t.Fatalf("Update() returned %T, want tui.Model", result)
	}
	return got
}

func TestNowPlayingLoadedUpdatesTrack(t *testing.T) {
	m := tui.New(nil, nil, nil)

	track := nowplaying.Track{Title: "Strobe", Source: "Spotify", Playing: true, Present: true}
	m = updateModel(t, m, tui.MakeNowPlayingLoadedMsg(track, nil))

	if got := m.CurrentTrack(); got != track {
		t.Errorf("CurrentTrack() = %+v, want %+v", got, track)
	}
}

func TestNowPlayingErrorKeepsLastTrack(t *testing.T) {
	m := tui.New(nil, nil, nil)

	good := nowplaying.Track{Title: "Strobe", Source: "Spotify", Playing: true, Present: true}
	m = tui.SetCurrentTrackForTest(m, good)

	// An errored snapshot must NOT blank the card.
	m = updateModel(t, m, tui.MakeNowPlayingLoadedMsg(nowplaying.Track{}, errors.New("helper hiccup")))

	if got := m.CurrentTrack(); got != good {
		t.Errorf("CurrentTrack() after error = %+v, want last good %+v", got, good)
	}
}

func TestNowPlayingTickWithoutProviderIsNoop(t *testing.T) {
	m := tui.New(nil, nil, nil) // no provider wired
	result, cmd := m.Update(tui.MakeNowPlayingTickMsg())
	if _, ok := result.(tui.Model); !ok {
		t.Fatalf("Update() returned %T, want tui.Model", result)
	}
	if cmd != nil {
		t.Error("nowPlayingTick with no provider should not schedule a command")
	}
}

func TestWelcomeShowsCardWhenPlaying(t *testing.T) {
	m := tui.New(nil, nil, nil)
	m.Screen = tui.ScreenWelcome
	m.Width = 120
	m.Height = 50
	m = tui.SetCurrentTrackForTest(m, nowplaying.Track{
		Title:   "Lo-fi beats",
		Source:  "Spotify",
		Playing: true,
		Present: true,
	})

	view := m.View()
	for _, want := range []string{"Lo-fi beats", "via Spotify", "▶ playing"} {
		if !strings.Contains(view, want) {
			t.Errorf("welcome view missing %q", want)
		}
	}
}

func TestAnimTickReadsLevelsOnWelcomePlaying(t *testing.T) {
	m := tui.New(nil, nil, nil)
	m.Screen = tui.ScreenWelcome
	m = tui.InjectAudio(m, fakeAnalyzer{levels: []float64{0.1, 0.9, 0.5}})
	m = tui.SetCurrentTrackForTest(m, nowplaying.Track{Title: "x", Present: true, Playing: true})

	result, cmd := m.Update(tui.MakeAnimTickMsg())
	got := result.(tui.Model)

	if len(got.BarLevels()) == 0 {
		t.Error("expected bar levels to be populated on welcome while playing")
	}
	if cmd == nil {
		t.Error("expected anim tick to re-queue while on welcome")
	}
}

func TestAnimTickStopsOffWelcome(t *testing.T) {
	m := tui.New(nil, nil, nil)
	m.Screen = tui.ScreenProjects
	m = tui.InjectAudio(m, fakeAnalyzer{levels: []float64{0.5}})

	_, cmd := m.Update(tui.MakeAnimTickMsg())
	if cmd != nil {
		t.Error("anim tick should not re-queue when off the welcome screen")
	}
}

func TestWelcomeHidesCardWhenAbsent(t *testing.T) {
	m := tui.New(nil, nil, nil)
	m.Screen = tui.ScreenWelcome
	m.Width = 120
	m.Height = 50
	// No track set → Present is false → card must not appear.
	if strings.Contains(m.View(), "via ") {
		t.Error("welcome view should not contain a now-playing card when nothing plays")
	}
}
