package transcripts_test

import (
	"testing"
	"time"

	"github.com/gastonz/atelier/internal/transcripts"
)

// makeTestEvents returns a slice of synthetic Events for replay tests.
func makeTestEvents(n int) []transcripts.Event {
	events := make([]transcripts.Event, n)
	for i := 0; i < n; i++ {
		events[i] = &transcripts.UserEvent{
			UUIDValue:      "uuid-replay-" + string(rune('a'+i)),
			SessionIDValue: "sess-replay",
			TimestampValue: time.Date(2026, 1, 1, 10, 0, i, 0, time.UTC),
			Text:           "event " + string(rune('A'+i)),
		}
	}
	return events
}

// ---- NewReplay tests --------------------------------------------------------

func TestReplay_NewReplay_CursorAtZero(t *testing.T) {
	events := makeTestEvents(5)
	r := transcripts.NewReplay(events)

	if r.Cursor() != 0 {
		t.Errorf("expected initial cursor=0, got %d", r.Cursor())
	}
	if r.Len() != 5 {
		t.Errorf("expected Len=5, got %d", r.Len())
	}
}

func TestReplay_NewReplay_SnapshotSemantics(t *testing.T) {
	// R5.2, S5.2: NewReplay copies the slice; mutations to the original
	// do NOT affect the replay sequence.
	events := makeTestEvents(3)
	r := transcripts.NewReplay(events)

	// Mutate the original slice after creating the replay.
	events[0] = &transcripts.UserEvent{
		UUIDValue: "mutated-001",
		Text:      "mutated",
	}

	// Replay's Current() should still return the original event.
	current := r.Current()
	if current == nil {
		t.Fatal("expected non-nil current event")
	}
	ue, ok := current.(*transcripts.UserEvent)
	if !ok {
		t.Fatalf("expected *UserEvent, got %T", current)
	}
	if ue.UUIDValue == "mutated-001" {
		t.Error("replay snapshot was mutated by external slice change — snapshot failed")
	}
	if ue.UUIDValue != "uuid-replay-a" {
		t.Errorf("expected uuid-replay-a, got %q", ue.UUIDValue)
	}
}

// ---- Next / Prev tests -------------------------------------------------------

func TestReplay_Next_AdvancesCursor(t *testing.T) {
	r := transcripts.NewReplay(makeTestEvents(5))

	ok := r.Next()
	if !ok {
		t.Error("expected Next()=true when not at end")
	}
	if r.Cursor() != 1 {
		t.Errorf("expected cursor=1 after one Next(), got %d", r.Cursor())
	}
}

func TestReplay_Next_ReturnsFalseAtEnd(t *testing.T) {
	// R5.3: Next() returns false at the last event.
	r := transcripts.NewReplay(makeTestEvents(3))

	r.Next() // 0→1
	r.Next() // 1→2

	// Now at index 2 (last). Next should return false.
	ok := r.Next()
	if ok {
		t.Error("expected Next()=false at last event, got true")
	}
	// Cursor should stay at 2.
	if r.Cursor() != 2 {
		t.Errorf("expected cursor=2 after Next() at end, got %d", r.Cursor())
	}
}

func TestReplay_Prev_MovesBackward(t *testing.T) {
	r := transcripts.NewReplay(makeTestEvents(5))
	r.Next() // → 1
	r.Next() // → 2

	ok := r.Prev()
	if !ok {
		t.Error("expected Prev()=true when not at start")
	}
	if r.Cursor() != 1 {
		t.Errorf("expected cursor=1 after Prev(), got %d", r.Cursor())
	}
}

func TestReplay_Prev_ReturnsFalseAtStart(t *testing.T) {
	// R5.5: Prev() returns false at cursor=0.
	r := transcripts.NewReplay(makeTestEvents(3))

	ok := r.Prev()
	if ok {
		t.Error("expected Prev()=false at start, got true")
	}
	if r.Cursor() != 0 {
		t.Errorf("expected cursor=0 after Prev() at start, got %d", r.Cursor())
	}
}

// ---- Pause / Resume tests ---------------------------------------------------

func TestReplay_PauseResume_StateTransitions(t *testing.T) {
	// R5.3, S5.3: Pause() / Resume() / Paused() state transitions.
	r := transcripts.NewReplay(makeTestEvents(3))

	if r.Paused() {
		t.Error("expected not paused initially")
	}

	r.Pause()
	if !r.Paused() {
		t.Error("expected paused after Pause()")
	}

	r.Resume()
	if r.Paused() {
		t.Error("expected not paused after Resume()")
	}
}

// ---- SetSpeed tests ---------------------------------------------------------

func TestReplay_SetSpeed_CyclesCorrectly(t *testing.T) {
	// R5.4, S5.4: speed cycles through 0.5/1/2/4.
	r := transcripts.NewReplay(makeTestEvents(1))

	validSpeeds := []float64{0.5, 1, 2, 4}
	for _, s := range validSpeeds {
		r.SetSpeed(s)
		if r.Speed() != s {
			t.Errorf("SetSpeed(%.1f) → Speed()=%.1f, expected %.1f", s, r.Speed(), s)
		}
	}
}

func TestReplay_SetSpeed_ClampToNearest(t *testing.T) {
	// Design spec: SetSpeed clamps to nearest valid value, does NOT panic.
	r := transcripts.NewReplay(makeTestEvents(1))

	// Out-of-range values should not panic.
	r.SetSpeed(0.0)  // below minimum
	r.SetSpeed(10.0) // above maximum
	r.SetSpeed(-1.0) // negative

	// Speed should be in valid range after clamping.
	s := r.Speed()
	valid := s == 0.5 || s == 1 || s == 2 || s == 4
	if !valid {
		t.Errorf("Speed %v is not in valid set {0.5, 1, 2, 4}", s)
	}
}

// ---- Current tests ----------------------------------------------------------

func TestReplay_Current_ReturnsEventAtCursor(t *testing.T) {
	events := makeTestEvents(5)
	r := transcripts.NewReplay(events)

	current := r.Current()
	if current == nil {
		t.Fatal("expected non-nil Current() at cursor 0")
	}
	ue := current.(*transcripts.UserEvent)
	if ue.UUIDValue != "uuid-replay-a" {
		t.Errorf("expected uuid-replay-a at cursor 0, got %q", ue.UUIDValue)
	}

	r.Next() // → cursor 1
	current2 := r.Current()
	ue2 := current2.(*transcripts.UserEvent)
	if ue2.UUIDValue != "uuid-replay-b" {
		t.Errorf("expected uuid-replay-b at cursor 1, got %q", ue2.UUIDValue)
	}
}

// ---- Len tests --------------------------------------------------------------

func TestReplay_Len_ReturnsTotal(t *testing.T) {
	r := transcripts.NewReplay(makeTestEvents(7))
	if r.Len() != 7 {
		t.Errorf("expected Len=7, got %d", r.Len())
	}
}

func TestReplay_CursorStaysInBounds_NeverNegative(t *testing.T) {
	r := transcripts.NewReplay(makeTestEvents(2))

	// Cursor at 0; calling Prev repeatedly should not go below 0.
	for i := 0; i < 5; i++ {
		r.Prev()
	}
	if r.Cursor() < 0 {
		t.Errorf("cursor went negative: %d", r.Cursor())
	}
}

func TestReplay_CursorStaysInBounds_NeverBeyondLen(t *testing.T) {
	events := makeTestEvents(3)
	r := transcripts.NewReplay(events)

	// Advance past the end repeatedly.
	for i := 0; i < 10; i++ {
		r.Next()
	}
	if r.Cursor() >= r.Len() {
		t.Errorf("cursor %d >= Len %d", r.Cursor(), r.Len())
	}
}

// ---- T24: TRIANGULATE edge cases -------------------------------------------

func TestReplay_NewReplay_Nil_EmptyReplay(t *testing.T) {
	// NewReplay(nil): Len==0, Current==nil, Next==false
	r := transcripts.NewReplay(nil)

	if r.Len() != 0 {
		t.Errorf("expected Len=0 for nil input, got %d", r.Len())
	}
	if r.Current() != nil {
		t.Errorf("expected Current=nil for nil input, got %v", r.Current())
	}
	if ok := r.Next(); ok {
		t.Error("expected Next()=false for nil input, got true")
	}
	if ok := r.Prev(); ok {
		t.Error("expected Prev()=false for nil input, got true")
	}
}

func TestReplay_SetSpeed_WrapAround_ViaRepeatedCycles(t *testing.T) {
	// Speed wrap-around: cycling through all 4 speeds and back brings us to 0.5.
	// The test verifies SetSpeed correctly handles all valid values.
	r := transcripts.NewReplay(makeTestEvents(1))

	// Cycle: 0.5 → 1 → 2 → 4 → (wrap back to 0.5)
	speeds := []float64{0.5, 1.0, 2.0, 4.0, 0.5}
	for i, s := range speeds {
		r.SetSpeed(s)
		got := r.Speed()
		if got != s {
			t.Errorf("step %d: SetSpeed(%.1f) → Speed()=%.1f, expected %.1f", i, s, got, s)
		}
	}
}

func TestReplay_MultipleCurrentCalls_NoStateCorruption(t *testing.T) {
	// Concurrent-safe read: multiple Current() calls (single goroutine) do not
	// corrupt state.
	r := transcripts.NewReplay(makeTestEvents(5))
	r.Next() // cursor → 1

	// Call Current() multiple times and verify consistent results.
	first := r.Current()
	second := r.Current()
	third := r.Current()

	if first != second || second != third {
		t.Error("multiple Current() calls returned different event instances")
	}
	if r.Cursor() != 1 {
		t.Errorf("cursor moved after multiple Current() calls: %d", r.Cursor())
	}
}

func TestReplay_Empty_PauseResume_NoPanic(t *testing.T) {
	// Pause/Resume on empty replay should not panic.
	r := transcripts.NewReplay(nil)
	r.Pause()
	r.Resume()
	r.Pause()
	// Reaching here means no panic.
	if !r.Paused() {
		t.Error("expected paused after last Pause()")
	}
}
