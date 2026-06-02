package tui

// test_exports.go — exported wrappers for internal functions used in _test.go files.
// These are ONLY used from test files. They expose internal helpers for whitebox coverage.
// Not part of the public API.

import (
	"time"

	"github.com/gastonz/atelier/internal/transcripts"
)

// EventPreviewForTest exposes the internal eventPreview function for tests.
func EventPreviewForTest(evt transcripts.Event) string {
	return eventPreview(evt)
}

// RelativeTimeForTest exposes the internal relativeTime function for tests.
func RelativeTimeForTest(t time.Time) string {
	return relativeTime(t)
}

// SetNowFuncForTest overrides the clock used by relativeTime and returns a
// restore function. Tests call `defer tui.SetNowFuncForTest(fn)()` to pin the
// clock and undo it afterwards, keeping relative-time output deterministic.
func SetNowFuncForTest(f func() time.Time) func() {
	prev := nowFunc
	nowFunc = f
	return func() { nowFunc = prev }
}

// ReplayIntervalForTest exposes replayInterval for tests.
func ReplayIntervalForTest(speed float64) time.Duration {
	return replayInterval(speed)
}

// ReplaySpeedUpForTest exposes replaySpeedUp for tests.
func ReplaySpeedUpForTest(current float64) float64 {
	return replaySpeedUp(current)
}

// ReplaySpeedDownForTest exposes replaySpeedDown for tests.
func ReplaySpeedDownForTest(current float64) float64 {
	return replaySpeedDown(current)
}

// MakeWatcherErrMsg constructs an agentWatcherErrMsg for tests.
func MakeWatcherErrMsg(err error) agentWatcherErrMsg {
	return agentWatcherErrMsg{err: err}
}

// SetAgentZoomedID returns a new Model with AgentZoomedID set.
func SetAgentZoomedID(m Model, id string) Model {
	m.AgentZoomedID = id
	return m
}

// SetScreen returns a new Model with Screen set to the given value.
func SetScreen(m Model, s Screen) Model {
	m.Screen = s
	return m
}
