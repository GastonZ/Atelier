package transcripts_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gastonz/atelier/internal/transcripts"
)

// testdataPath returns the absolute path to the testdata directory relative to
// the file calling this function. This works regardless of the working directory
// at test time.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(currentFile), "testdata", name)
}

// ---- ParseLine tests --------------------------------------------------------

func TestParseLine_UserEvent(t *testing.T) {
	line := []byte(`{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":"Hello"},"uuid":"uuid-u-001","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-001","cwd":"C:\\Projects\\Atelier","version":"2.1.126","gitBranch":"master"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event for user type")
	}
	if event.Kind() != transcripts.KindUser {
		t.Errorf("expected KindUser, got %v", event.Kind())
	}

	ue, ok := event.(*transcripts.UserEvent)
	if !ok {
		t.Fatalf("expected *UserEvent, got %T", event)
	}
	if ue.SessionIDValue != "sess-001" {
		t.Errorf("expected SessionID=sess-001, got %q", ue.SessionIDValue)
	}
	if ue.UUIDValue != "uuid-u-001" {
		t.Errorf("expected UUID=uuid-u-001, got %q", ue.UUIDValue)
	}
	if ue.Cwd != "C:\\Projects\\Atelier" {
		t.Errorf("expected cwd, got %q", ue.Cwd)
	}
	if ue.GitBranch != "master" {
		t.Errorf("expected branch master, got %q", ue.GitBranch)
	}
	if ue.Text != "Hello" {
		t.Errorf("expected text Hello, got %q", ue.Text)
	}
	// IsSidechain false
	if ue.IsSidechainFlag {
		t.Error("expected IsSidechain=false")
	}
}

func TestParseLine_AssistantEvent(t *testing.T) {
	line := []byte(`{"parentUuid":"uuid-u-001","isSidechain":false,"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","role":"assistant","content":[{"type":"text","text":"Hi there!"}],"stop_reason":"end_turn","usage":{"input_tokens":100,"cache_creation_input_tokens":200,"cache_read_input_tokens":50,"output_tokens":30}},"requestId":"req_001","uuid":"uuid-a-001","timestamp":"2026-01-01T10:00:02.000Z","sessionId":"sess-001"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Kind() != transcripts.KindAssistant {
		t.Errorf("expected KindAssistant, got %v", event.Kind())
	}

	ae, ok := event.(*transcripts.AssistantEvent)
	if !ok {
		t.Fatalf("expected *AssistantEvent, got %T", event)
	}
	if ae.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model claude-sonnet-4-6, got %q", ae.Model)
	}
	if ae.Text != "Hi there!" {
		t.Errorf("expected text 'Hi there!', got %q", ae.Text)
	}
	if ae.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %q", ae.StopReason)
	}
	if ae.Usage.InputTokens != 100 {
		t.Errorf("expected input_tokens=100, got %d", ae.Usage.InputTokens)
	}
	if ae.Usage.CacheCreationTokens != 200 {
		t.Errorf("expected cache_creation=200, got %d", ae.Usage.CacheCreationTokens)
	}
	if ae.Usage.CacheReadTokens != 50 {
		t.Errorf("expected cache_read=50, got %d", ae.Usage.CacheReadTokens)
	}
	if ae.Usage.OutputTokens != 30 {
		t.Errorf("expected output_tokens=30, got %d", ae.Usage.OutputTokens)
	}
}

func TestParseLine_ToolUseEvent(t *testing.T) {
	line := []byte(`{"parentUuid":"uuid-u-001","isSidechain":false,"type":"assistant","message":{"model":"claude-opus-4-7","id":"msg_02","role":"assistant","content":[{"type":"tool_use","id":"tool-001","name":"Bash","input":{"command":"echo hello"}}],"stop_reason":"tool_use","usage":{"input_tokens":80,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":20}},"requestId":"req_002","uuid":"uuid-a-002","timestamp":"2026-01-01T10:00:05.000Z","sessionId":"sess-001"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	// An assistant message with tool_use content still comes as AssistantEvent
	// (the tool_use is embedded in the content array)
	if event.Kind() != transcripts.KindAssistant {
		t.Errorf("expected KindAssistant for tool_use content, got %v", event.Kind())
	}
}

func TestParseLine_ToolResultEvent(t *testing.T) {
	line := []byte(`{"parentUuid":"uuid-a-002","isSidechain":false,"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-001","content":"hello\n"}]},"uuid":"uuid-u-002","timestamp":"2026-01-01T10:00:06.000Z","sessionId":"sess-001","cwd":"C:\\Projects","version":"2.1.126","gitBranch":"master"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	// Tool results come as user-type messages — Kind depends on message content detection
	// The parser should emit KindToolResult when message.content is a tool_result array
	if event.Kind() != transcripts.KindToolResult {
		t.Errorf("expected KindToolResult for tool_result user message, got %v", event.Kind())
	}

	tre, ok := event.(*transcripts.ToolResultEvent)
	if !ok {
		t.Fatalf("expected *ToolResultEvent, got %T", event)
	}
	if tre.ToolUseID != "tool-001" {
		t.Errorf("expected ToolUseID=tool-001, got %q", tre.ToolUseID)
	}
}

func TestParseLine_MetaEvent(t *testing.T) {
	line := []byte(`{"type":"ai-title","aiTitle":"Dragon Atelier","sessionId":"sess-meta-001"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event for meta type")
	}
	if event.Kind() != transcripts.KindMeta {
		t.Errorf("expected KindMeta, got %v", event.Kind())
	}

	me, ok := event.(*transcripts.MetaEvent)
	if !ok {
		t.Fatalf("expected *MetaEvent, got %T", event)
	}
	if me.MetaKind != "ai-title" {
		t.Errorf("expected MetaKind=ai-title, got %q", me.MetaKind)
	}
}

func TestParseLine_AttachmentSkipped(t *testing.T) {
	// R3.6: attachment must return (nil, nil)
	line := []byte(`{"type":"attachment","attachment":{"type":"hook_success","hookName":"test","output":"OK"},"uuid":"uuid-att-001","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-001"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for attachment type, got %T", event)
	}
}

func TestParseLine_ProgressSkipped(t *testing.T) {
	// R3.6: progress must return (nil, nil)
	line := []byte(`{"type":"progress","data":{"type":"hook_progress","hookEvent":"PreToolUse","hookName":"check","command":"run"},"toolUseID":"t-001","parentToolUseID":"","uuid":"uuid-prog-001","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-001"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for progress type, got %T", event)
	}
}

func TestParseLine_MalformedJSON(t *testing.T) {
	line := []byte(`{not valid json}`)

	event, err := transcripts.ParseLine(line)
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
	if event != nil {
		t.Errorf("expected nil event for malformed JSON, got %T", event)
	}
}

func TestParseLine_EmptyLine(t *testing.T) {
	event, err := transcripts.ParseLine([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error for empty line: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for empty line, got %T", event)
	}
}

func TestParseLine_WhitespaceLine(t *testing.T) {
	event, err := transcripts.ParseLine([]byte("   \t  "))
	if err != nil {
		t.Fatalf("unexpected error for whitespace line: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for whitespace line, got %T", event)
	}
}

func TestParseLine_TimestampParsed(t *testing.T) {
	line := []byte(`{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":"test"},"uuid":"uuid-ts-001","timestamp":"2026-03-15T14:30:45.123Z","sessionId":"sess-ts","cwd":"C:\\Proj","version":"2.1.126","gitBranch":"main"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	ts := event.Timestamp()
	expected := time.Date(2026, 3, 15, 14, 30, 45, 123000000, time.UTC)
	if !ts.Equal(expected) {
		t.Errorf("expected timestamp %v, got %v", expected, ts)
	}
}

// ---- ParseStream tests -------------------------------------------------------

func TestParseStream_SimpleSession(t *testing.T) {
	path := testdataPath(t, "simple-session.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("could not open fixture: %v", err)
	}
	defer f.Close()

	var events []transcripts.Event
	err = transcripts.ParseStream(f, func(e transcripts.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("ParseStream error: %v", err)
	}

	// simple-session.jsonl has:
	// 1 MetaEvent (last-prompt)
	// 3 UserEvent (incl. tool result user messages)
	// 3 AssistantEvent
	// Total non-skipped: 7 events
	if len(events) == 0 {
		t.Fatal("expected events from simple-session.jsonl, got 0")
	}

	// Check first is meta
	if events[0].Kind() != transcripts.KindMeta {
		t.Errorf("expected first event to be KindMeta, got %v", events[0].Kind())
	}

	// Verify we have at least one assistant event with correct model
	hasAssistant := false
	for _, e := range events {
		if ae, ok := e.(*transcripts.AssistantEvent); ok {
			if ae.Model == "claude-sonnet-4-6" {
				hasAssistant = true
				break
			}
		}
	}
	if !hasAssistant {
		t.Error("expected at least one AssistantEvent with model=claude-sonnet-4-6")
	}
}

func TestParseStream_NoisyFile_ZeroEvents(t *testing.T) {
	// R3.6 regression: noisy.jsonl has ONLY attachment+progress lines → zero events
	path := testdataPath(t, "noisy.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("could not open fixture: %v", err)
	}
	defer f.Close()

	var events []transcripts.Event
	err = transcripts.ParseStream(f, func(e transcripts.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("ParseStream error: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events from noisy.jsonl (all attachment/progress), got %d", len(events))
	}
}

func TestParseStream_MetaOnlyFile(t *testing.T) {
	path := testdataPath(t, "meta-only.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("could not open fixture: %v", err)
	}
	defer f.Close()

	var events []transcripts.Event
	err = transcripts.ParseStream(f, func(e transcripts.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("ParseStream error: %v", err)
	}

	// meta-only.jsonl has 4 meta lines — all should be MetaEvents
	if len(events) == 0 {
		t.Fatal("expected MetaEvents from meta-only.jsonl")
	}
	for i, e := range events {
		if e.Kind() != transcripts.KindMeta {
			t.Errorf("event[%d]: expected KindMeta, got %v", i, e.Kind())
		}
	}
}

func TestParseStream_EmptyReader(t *testing.T) {
	var events []transcripts.Event
	err := transcripts.ParseStream(bytes.NewReader(nil), func(e transcripts.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("unexpected error for empty reader: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty reader, got %d", len(events))
	}
}

// ---- T25: Coverage gate — exercise interface methods directly ---------------

// TestEventInterface_Methods exercises all Event interface methods on each
// concrete type to hit the otherwise-uncovered getter lines.
func TestEventInterface_Methods(t *testing.T) {
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		event     transcripts.Event
		wantKind  transcripts.EventKind
		wantSess  string
		wantUUID  string
		wantSide  bool
	}{
		{
			name: "UserEvent",
			event: &transcripts.UserEvent{
				UUIDValue:       "u-uuid",
				SessionIDValue:  "u-sess",
				TimestampValue:  ts,
				IsSidechainFlag: false,
			},
			wantKind: transcripts.KindUser,
			wantSess: "u-sess",
			wantUUID: "u-uuid",
			wantSide: false,
		},
		{
			name: "AssistantEvent",
			event: &transcripts.AssistantEvent{
				UUIDValue:       "a-uuid",
				SessionIDValue:  "a-sess",
				TimestampValue:  ts,
				IsSidechainFlag: true,
			},
			wantKind: transcripts.KindAssistant,
			wantSess: "a-sess",
			wantUUID: "a-uuid",
			wantSide: true,
		},
		{
			name: "ToolUseEvent",
			event: &transcripts.ToolUseEvent{
				UUIDValue:       "tu-uuid",
				SessionIDValue:  "tu-sess",
				TimestampValue:  ts,
				IsSidechainFlag: false,
			},
			wantKind: transcripts.KindToolUse,
			wantSess: "tu-sess",
			wantUUID: "tu-uuid",
			wantSide: false,
		},
		{
			name: "ToolResultEvent",
			event: &transcripts.ToolResultEvent{
				UUIDValue:       "tr-uuid",
				SessionIDValue:  "tr-sess",
				TimestampValue:  ts,
				IsSidechainFlag: true,
			},
			wantKind: transcripts.KindToolResult,
			wantSess: "tr-sess",
			wantUUID: "tr-uuid",
			wantSide: true,
		},
		{
			name: "MetaEvent",
			event: &transcripts.MetaEvent{
				UUIDValue:       "me-uuid",
				SessionIDValue:  "me-sess",
				TimestampValue:  ts,
				IsSidechainFlag: false,
				MetaKind:        "ai-title",
			},
			wantKind: transcripts.KindMeta,
			wantSess: "me-sess",
			wantUUID: "me-uuid",
			wantSide: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.event.Kind() != tc.wantKind {
				t.Errorf("Kind: got %v, want %v", tc.event.Kind(), tc.wantKind)
			}
			if tc.event.SessionID() != tc.wantSess {
				t.Errorf("SessionID: got %q, want %q", tc.event.SessionID(), tc.wantSess)
			}
			if tc.event.UUID() != tc.wantUUID {
				t.Errorf("UUID: got %q, want %q", tc.event.UUID(), tc.wantUUID)
			}
			if tc.event.IsSidechain() != tc.wantSide {
				t.Errorf("IsSidechain: got %v, want %v", tc.event.IsSidechain(), tc.wantSide)
			}
			if !tc.event.Timestamp().Equal(ts) {
				t.Errorf("Timestamp: got %v, want %v", tc.event.Timestamp(), ts)
			}
		})
	}
}

// TestExtractUserText_ArrayOfTextBlocks exercises the array-of-blocks path
// in extractUserText (previously uncovered).
func TestParseLine_UserEvent_ArrayContentBlocks(t *testing.T) {
	// Content as array of text blocks (not a plain string)
	line := []byte(`{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":[{"type":"text","text":"Block A"},{"type":"text","text":" Block B"}]},"uuid":"uuid-blocks","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-blocks","cwd":"C:\\P","version":"2.1.126","gitBranch":"main"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	ue, ok := event.(*transcripts.UserEvent)
	if !ok {
		t.Fatalf("expected *UserEvent, got %T", event)
	}
	if ue.Text != "Block A Block B" {
		t.Errorf("expected concatenated text 'Block A Block B', got %q", ue.Text)
	}
}

// TestParseLine_AssistantEvent_ToolUseContent exercises the assistant line
// with tool_use content (text is empty, tool name visible in content).
func TestParseLine_AssistantEvent_MultipleContentBlocks(t *testing.T) {
	line := []byte(`{"parentUuid":"uuid-p","isSidechain":false,"type":"assistant","message":{"model":"claude-opus-4-7","id":"msg_multi","role":"assistant","content":[{"type":"text","text":"Thinking... "},{"type":"tool_use","id":"t-001","name":"Read","input":{"file_path":"/tmp/x"}}],"stop_reason":"tool_use","usage":{"input_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":25}},"requestId":"req_multi","uuid":"uuid-multi","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-multi"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ae, ok := event.(*transcripts.AssistantEvent)
	if !ok {
		t.Fatalf("expected *AssistantEvent, got %T", event)
	}
	// Text should contain only the text block, not the tool_use block.
	if ae.Text != "Thinking... " {
		t.Errorf("expected 'Thinking... ', got %q", ae.Text)
	}
}

// ---- T6: TRIANGULATE edge cases ---------------------------------------------

// Attachment mid-stream surrounded by real events yields only the real events.
func TestParseStream_AttachmentMidStream_OnlyRealEventsEmitted(t *testing.T) {
	input := []string{
		`{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":"before"},"uuid":"uuid-before","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-mid","cwd":"C:\\P","version":"2.1.126","gitBranch":"main"}`,
		`{"type":"attachment","attachment":{"type":"hook_success","hookName":"check","output":"ok"},"uuid":"uuid-att","timestamp":"2026-01-01T10:00:01.000Z","sessionId":"sess-mid"}`,
		`{"parentUuid":"uuid-before","isSidechain":false,"type":"user","message":{"role":"user","content":"after"},"uuid":"uuid-after","timestamp":"2026-01-01T10:00:02.000Z","sessionId":"sess-mid","cwd":"C:\\P","version":"2.1.126","gitBranch":"main"}`,
	}

	combined := bytes.Join(func() [][]byte {
		out := make([][]byte, len(input))
		for i, s := range input {
			out[i] = []byte(s)
		}
		return out
	}(), []byte("\n"))

	var events []transcripts.Event
	err := transcripts.ParseStream(bytes.NewReader(combined), func(e transcripts.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("ParseStream error: %v", err)
	}

	// Should have exactly 2 events (both UserEvents), not 3.
	if len(events) != 2 {
		t.Errorf("expected 2 events (attachment skipped), got %d", len(events))
	}
	for i, e := range events {
		if e.Kind() != transcripts.KindUser {
			t.Errorf("event[%d]: expected KindUser, got %v", i, e.Kind())
		}
	}
}

// AssistantEvent with missing usage field (no "usage" key) doesn't crash.
func TestParseLine_AssistantEvent_MissingUsage_NoCrash(t *testing.T) {
	line := []byte(`{"parentUuid":"uuid-u-001","isSidechain":false,"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_noUsage","role":"assistant","content":[{"type":"text","text":"No usage here."}],"stop_reason":"end_turn"},"requestId":"req_nou","uuid":"uuid-a-nou","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-nou"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error when usage is missing: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	ae, ok := event.(*transcripts.AssistantEvent)
	if !ok {
		t.Fatalf("expected *AssistantEvent, got %T", event)
	}
	// All token counts should be zero.
	if ae.Usage.InputTokens != 0 || ae.Usage.OutputTokens != 0 {
		t.Errorf("expected zero token counts when usage missing, got %+v", ae.Usage)
	}
}

// UTF-8 content in text fields is preserved correctly.
func TestParseLine_UTF8Content(t *testing.T) {
	text := "Hola! ¿Cómo estás? 你好 — こんにちは 🐉"
	line := []byte(`{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":"` +
		text + `"},"uuid":"uuid-utf8","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-utf8","cwd":"C:\\P","version":"2.1.126","gitBranch":"main"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	ue, ok := event.(*transcripts.UserEvent)
	if !ok {
		t.Fatalf("expected *UserEvent, got %T", event)
	}
	if ue.Text != text {
		t.Errorf("UTF-8 text not preserved: got %q", ue.Text)
	}
}

// Truncated last line (simulate append mid-write): malformed JSON on last line
// of a stream should NOT stop already-emitted events from being returned.
func TestParseStream_TruncatedLastLine_PriorEventsStillEmitted(t *testing.T) {
	good := `{"parentUuid":null,"isSidechain":false,"type":"user","message":{"role":"user","content":"complete line"},"uuid":"uuid-trunc-good","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-trunc","cwd":"C:\\P","version":"2.1.126","gitBranch":"main"}`
	truncated := `{"parentUuid":"uuid-trunc-good","type":"assistant","message":{"model":"cl` // truncated mid-write

	combined := []byte(good + "\n" + truncated)

	var events []transcripts.Event
	err := transcripts.ParseStream(bytes.NewReader(combined), func(e transcripts.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("unexpected stream-level error: %v", err)
	}

	// The good line should have been emitted; truncated line is silently skipped.
	if len(events) != 1 {
		t.Errorf("expected 1 event (good line), got %d", len(events))
	}
	if events[0].Kind() != transcripts.KindUser {
		t.Errorf("expected KindUser for good line, got %v", events[0].Kind())
	}
}

// IsSidechain flag is propagated for sidechain events.
func TestParseLine_IsSidechainPropagated(t *testing.T) {
	line := []byte(`{"parentUuid":"uuid-parent","isSidechain":true,"type":"user","message":{"role":"user","content":"sidechain msg"},"uuid":"uuid-side","timestamp":"2026-01-01T10:00:00.000Z","sessionId":"sess-side","cwd":"C:\\P","version":"2.1.126","gitBranch":"main"}`)

	event, err := transcripts.ParseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if !event.IsSidechain() {
		t.Error("expected IsSidechain=true, got false")
	}
}
