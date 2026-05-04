package transcripts

// validSpeeds is the ordered set of valid replay speeds (R5.4).
// The only valid values are 0.5, 1, 2, and 4 (wrapping in both directions).
var validSpeeds = []float64{0.5, 1.0, 2.0, 4.0}

// Replay is a cursor over a frozen snapshot of a session's event slice.
// It supports stepping forward/backward, pause/resume, and speed control.
//
// Replay is NOT goroutine-safe. The caller (tui.Update) is the sole owner.
type Replay struct {
	events  []Event  // snapshot (copy of caller's slice; R5.2)
	cursor  int      // current position in events (0-indexed)
	paused  bool     // true when auto-advance is suspended
	speed   float64  // current playback speed (one of validSpeeds)
}

// NewReplay creates a new Replay from the given event slice.
// The events are copied (snapshot semantics per R5.2) — mutations to the
// original slice after this call are NOT reflected in the replay.
func NewReplay(events []Event) *Replay {
	snapshot := make([]Event, len(events))
	copy(snapshot, events)
	return &Replay{
		events: snapshot,
		cursor: 0,
		paused: false,
		speed:  1.0,
	}
}

// Next advances the cursor by one event.
// Returns true if the cursor moved; false when already at the last event.
func (r *Replay) Next() bool {
	if len(r.events) == 0 || r.cursor >= len(r.events)-1 {
		return false
	}
	r.cursor++
	return true
}

// Prev moves the cursor back one event.
// Returns true if the cursor moved; false when already at the first event.
func (r *Replay) Prev() bool {
	if r.cursor <= 0 {
		return false
	}
	r.cursor--
	return true
}

// Pause suspends auto-advance. Idempotent.
func (r *Replay) Pause() {
	r.paused = true
}

// Resume resumes auto-advance. Idempotent.
func (r *Replay) Resume() {
	r.paused = false
}

// Paused reports whether auto-advance is currently suspended.
func (r *Replay) Paused() bool {
	return r.paused
}

// SetSpeed sets the playback speed. Clamps to the nearest valid value in
// {0.5, 1, 2, 4} rather than panicking on out-of-range input.
func (r *Replay) SetSpeed(s float64) {
	r.speed = clampToValidSpeed(s)
}

// Speed returns the current playback speed.
func (r *Replay) Speed() float64 {
	return r.speed
}

// Cursor returns the current zero-based cursor position.
func (r *Replay) Cursor() int {
	return r.cursor
}

// Len returns the total number of events in the replay snapshot.
func (r *Replay) Len() int {
	return len(r.events)
}

// Current returns the Event at the current cursor position.
// Returns nil when Len() == 0.
func (r *Replay) Current() Event {
	if len(r.events) == 0 {
		return nil
	}
	return r.events[r.cursor]
}

// ---------------------------------------------------------------------------
// Speed helpers
// ---------------------------------------------------------------------------

// clampToValidSpeed returns the valid speed value nearest to s.
// For exact matches, returns s directly. For values out of range, returns
// the minimum (0.5) or maximum (4.0) as appropriate. For values between
// valid entries, returns the nearest entry.
func clampToValidSpeed(s float64) float64 {
	if s <= validSpeeds[0] {
		return validSpeeds[0]
	}
	if s >= validSpeeds[len(validSpeeds)-1] {
		return validSpeeds[len(validSpeeds)-1]
	}

	best := validSpeeds[0]
	bestDist := abs64(s - best)
	for _, v := range validSpeeds[1:] {
		d := abs64(s - v)
		if d < bestDist {
			bestDist = d
			best = v
		}
	}
	return best
}

// abs64 returns the absolute value of a float64.
func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
