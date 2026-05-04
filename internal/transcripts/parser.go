// Package transcripts provides JSONL transcript parsing, file discovery,
// file watching, cost calculation, project mapping, and replay for Claude Code sessions.
package transcripts

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// EventKind is a typed integer tag for discriminating Event implementations.
type EventKind int

const (
	KindUser      EventKind = iota // user turn (plain message or tool result)
	KindAssistant                  // assistant model response
	KindToolUse                    // assistant turn containing a tool_use content block
	KindToolResult                 // user turn containing a tool_result content block
	KindMeta                       // metadata line (ai-title, permission-mode, last-prompt, etc.)
)

// Event is a sealed interface for all events emitted by the parser.
// The type-tag method Kind() enables exhaustive switching without relying on
// type assertions at every call site.
type Event interface {
	Kind() EventKind
	Timestamp() time.Time
	SessionID() string
	UUID() string
	IsSidechain() bool
}

// Usage holds token counts from an assistant message's usage field.
type Usage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
}

// UserEvent represents a user turn that contains plain text content.
type UserEvent struct {
	UUIDValue       string
	SessionIDValue  string
	TimestampValue  time.Time
	IsSidechainFlag bool
	Cwd             string
	GitBranch       string
	Text            string // flattened from message.content string or text blocks
}

func (e *UserEvent) Kind() EventKind     { return KindUser }
func (e *UserEvent) Timestamp() time.Time { return e.TimestampValue }
func (e *UserEvent) SessionID() string    { return e.SessionIDValue }
func (e *UserEvent) UUID() string         { return e.UUIDValue }
func (e *UserEvent) IsSidechain() bool    { return e.IsSidechainFlag }

// AssistantEvent represents an assistant model response (text or tool_use content).
type AssistantEvent struct {
	UUIDValue       string
	SessionIDValue  string
	TimestampValue  time.Time
	IsSidechainFlag bool
	Model           string
	Text            string // concatenated text blocks for display
	StopReason      string
	Usage           Usage
}

func (e *AssistantEvent) Kind() EventKind     { return KindAssistant }
func (e *AssistantEvent) Timestamp() time.Time { return e.TimestampValue }
func (e *AssistantEvent) SessionID() string    { return e.SessionIDValue }
func (e *AssistantEvent) UUID() string         { return e.UUIDValue }
func (e *AssistantEvent) IsSidechain() bool    { return e.IsSidechainFlag }

// ToolUseEvent represents a tool invocation extracted from an assistant turn's
// content blocks. Note: the parent line type is "assistant"; we emit a
// ToolUseEvent only when the first content block is type "tool_use".
// In practice we emit AssistantEvent for all assistant lines (including those
// with tool_use content) — ToolUseEvent is available for callers that want a
// dedicated type, but ParseLine emits AssistantEvent for assistant-type JSONL lines.
type ToolUseEvent struct {
	UUIDValue       string
	SessionIDValue  string
	TimestampValue  time.Time
	IsSidechainFlag bool
	ToolName        string
	ToolUseID       string
	InputSummary    string // first 80 chars of JSON-encoded input
}

func (e *ToolUseEvent) Kind() EventKind     { return KindToolUse }
func (e *ToolUseEvent) Timestamp() time.Time { return e.TimestampValue }
func (e *ToolUseEvent) SessionID() string    { return e.SessionIDValue }
func (e *ToolUseEvent) UUID() string         { return e.UUIDValue }
func (e *ToolUseEvent) IsSidechain() bool    { return e.IsSidechainFlag }

// ToolResultEvent represents a user turn that carries a tool_result content block.
type ToolResultEvent struct {
	UUIDValue       string
	SessionIDValue  string
	TimestampValue  time.Time
	IsSidechainFlag bool
	ToolUseID       string
	IsError         bool
	OutputSummary   string // first 200 chars of result content
}

func (e *ToolResultEvent) Kind() EventKind     { return KindToolResult }
func (e *ToolResultEvent) Timestamp() time.Time { return e.TimestampValue }
func (e *ToolResultEvent) SessionID() string    { return e.SessionIDValue }
func (e *ToolResultEvent) UUID() string         { return e.UUIDValue }
func (e *ToolResultEvent) IsSidechain() bool    { return e.IsSidechainFlag }

// MetaEvent represents a metadata-only line such as "ai-title", "permission-mode",
// "last-prompt", "file-history-snapshot", etc.
type MetaEvent struct {
	UUIDValue       string
	SessionIDValue  string
	TimestampValue  time.Time
	IsSidechainFlag bool
	MetaKind        string
	Payload         map[string]string
}

func (e *MetaEvent) Kind() EventKind     { return KindMeta }
func (e *MetaEvent) Timestamp() time.Time { return e.TimestampValue }
func (e *MetaEvent) SessionID() string    { return e.SessionIDValue }
func (e *MetaEvent) UUID() string         { return e.UUIDValue }
func (e *MetaEvent) IsSidechain() bool    { return e.IsSidechainFlag }

// ---------------------------------------------------------------------------
// Internal wire types (unexported) — only used for JSON decoding.
// ---------------------------------------------------------------------------

// rawLine is a minimal discriminator struct. We decode the full line into
// specific structs only after we know the type.
type rawLine struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	SessionID string `json:"sessionId"`
	Timestamp string `json:"timestamp"`
	IsSidechain bool `json:"isSidechain"`
}

type rawUserMessage struct {
	Role    string            `json:"role"`
	Content json.RawMessage   `json:"content"`
}

type rawUserLine struct {
	rawLine
	ParentUUID string         `json:"parentUuid"`
	Message    rawUserMessage `json:"message"`
	Cwd        string         `json:"cwd"`
	GitBranch  string         `json:"gitBranch"`
}

type rawContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

type rawUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type rawAssistantMessage struct {
	Model      string            `json:"model"`
	ID         string            `json:"id"`
	Role       string            `json:"role"`
	Content    []rawContentBlock `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      rawUsage          `json:"usage"`
}

type rawAssistantLine struct {
	rawLine
	ParentUUID string              `json:"parentUuid"`
	Message    rawAssistantMessage `json:"message"`
}

// rawMetaLine is a catch-all for metadata lines. We store the raw JSON for
// flexible payload extraction.
type rawMetaLine struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	UUID      string          `json:"uuid"`
	Timestamp string          `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// ParseLine — public API
// ---------------------------------------------------------------------------

// ParseLine parses a single JSONL line and returns the corresponding Event.
//
// Returns (nil, nil) for skipped event types ("attachment", "progress").
// Returns (nil, nil) for empty or whitespace-only lines.
// Returns (nil, error) for malformed JSON.
func ParseLine(line []byte) (Event, error) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil, nil
	}

	// First pass: decode only the type field to discriminate.
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(trimmed, &disc); err != nil {
		return nil, fmt.Errorf("transcripts: parse: %w", err)
	}

	switch disc.Type {
	case "attachment", "progress":
		// R3.6: always skip these types
		return nil, nil

	case "user":
		return parseUserLine(trimmed)

	case "assistant":
		return parseAssistantLine(trimmed)

	default:
		// Meta / unknown — treated as MetaEvent
		return parseMetaLine(trimmed, disc.Type)
	}
}

// ParseStream reads all lines from r and calls emit for each non-nil Event.
// Empty lines and skipped types (attachment, progress) produce no calls.
// Returns the first unrecoverable error (malformed JSON does not stop the stream
// — callers that want strict mode should use ParseLine directly).
func ParseStream(r io.Reader, emit func(Event)) error {
	scanner := bufio.NewScanner(r)
	// Allow up to 10 MB per line to handle large attachment/progress lines
	// (they must be parsed enough to detect their type before being discarded).
	const maxTokenSize = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		event, err := ParseLine(line)
		if err != nil {
			// Non-fatal: skip malformed lines during streaming.
			continue
		}
		if event != nil {
			emit(event)
		}
	}
	return scanner.Err()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try without nanoseconds
		t, err = time.Parse("2006-01-02T15:04:05.000Z", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t.UTC()
}

// truncateString returns the first max bytes of s (rune-safe truncation is
// not required for summaries — we truncate at byte boundaries for speed).
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func parseUserLine(line []byte) (Event, error) {
	var raw rawUserLine
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("transcripts: parse user: %w", err)
	}

	// Detect if the content is a tool_result array.
	// Content may be: a plain string, or a JSON array of content blocks.
	if isToolResultContent(raw.Message.Content) {
		return parseToolResultLine(&raw), nil
	}

	// Plain user text.
	text := extractUserText(raw.Message.Content)
	return &UserEvent{
		UUIDValue:       raw.UUID,
		SessionIDValue:  raw.SessionID,
		TimestampValue:  parseTimestamp(raw.Timestamp),
		IsSidechainFlag: raw.IsSidechain,
		Cwd:             raw.Cwd,
		GitBranch:       raw.GitBranch,
		Text:            text,
	}, nil
}

// isToolResultContent returns true when the raw content JSON begins with a
// tool_result content-block array.
func isToolResultContent(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return false
	}
	var blocks []rawContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}
	if len(blocks) > 0 && blocks[0].Type == "tool_result" {
		return true
	}
	return false
}

// parseToolResultLine extracts a ToolResultEvent from a user line whose content
// is a tool_result array.
func parseToolResultLine(raw *rawUserLine) *ToolResultEvent {
	var blocks []rawContentBlock
	_ = json.Unmarshal(raw.Message.Content, &blocks)

	toolUseID := ""
	isError := false
	outputSummary := ""

	if len(blocks) > 0 {
		b := blocks[0]
		toolUseID = b.ToolUseID
		isError = b.IsError
		// Content may be a string or an array; capture its raw JSON as summary.
		if len(b.Content) > 0 {
			// Attempt string decode first.
			var s string
			if err := json.Unmarshal(b.Content, &s); err == nil {
				outputSummary = truncateString(s, 200)
			} else {
				outputSummary = truncateString(string(b.Content), 200)
			}
		}
	}

	return &ToolResultEvent{
		UUIDValue:       raw.UUID,
		SessionIDValue:  raw.SessionID,
		TimestampValue:  parseTimestamp(raw.Timestamp),
		IsSidechainFlag: raw.IsSidechain,
		ToolUseID:       toolUseID,
		IsError:         isError,
		OutputSummary:   outputSummary,
	}
}

// extractUserText flattens a user message's content into a plain string.
// Content may be a JSON string or a JSON array of text-block objects.
func extractUserText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}

	// Try plain string first.
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}

	// Try array of content blocks.
	if trimmed[0] == '[' {
		var blocks []rawContentBlock
		if err := json.Unmarshal(raw, &blocks); err == nil {
			var sb bytes.Buffer
			for _, b := range blocks {
				if b.Type == "text" {
					sb.WriteString(b.Text)
				}
			}
			return sb.String()
		}
	}

	return string(raw)
}

func parseAssistantLine(line []byte) (Event, error) {
	var raw rawAssistantLine
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("transcripts: parse assistant: %w", err)
	}

	// Concatenate text blocks for display.
	var sb bytes.Buffer
	for _, block := range raw.Message.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	return &AssistantEvent{
		UUIDValue:       raw.UUID,
		SessionIDValue:  raw.SessionID,
		TimestampValue:  parseTimestamp(raw.Timestamp),
		IsSidechainFlag: raw.IsSidechain,
		Model:           raw.Message.Model,
		Text:            sb.String(),
		StopReason:      raw.Message.StopReason,
		Usage: Usage{
			InputTokens:         raw.Message.Usage.InputTokens,
			OutputTokens:        raw.Message.Usage.OutputTokens,
			CacheCreationTokens: raw.Message.Usage.CacheCreationInputTokens,
			CacheReadTokens:     raw.Message.Usage.CacheReadInputTokens,
		},
	}, nil
}

// parseMetaLine handles any line type not matched by "user" or "assistant".
// It extracts known string fields into the Payload map.
func parseMetaLine(line []byte, metaKind string) (Event, error) {
	// Decode into a generic map to capture all top-level string fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("transcripts: parse meta: %w", err)
	}

	payload := make(map[string]string)
	for k, v := range raw {
		if k == "type" {
			continue
		}
		// Store string values directly; arrays/objects stored as JSON.
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			payload[k] = s
		} else {
			payload[k] = string(v)
		}
	}

	sessionID := payload["sessionId"]
	uuid := payload["uuid"]
	ts := parseTimestamp(payload["timestamp"])

	return &MetaEvent{
		UUIDValue:      uuid,
		SessionIDValue: sessionID,
		TimestampValue: ts,
		MetaKind:       metaKind,
		Payload:        payload,
	}, nil
}
