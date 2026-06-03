// Package actions provides interfaces and implementations for launching external
// processes and interacting with the system clipboard on behalf of Atelier.
package actions

// Opener launches external processes for a project path.
// The TUI depends on this interface; tests inject a MockOpener.
type Opener interface {
	OpenInClaudeCode(projectPath string) error
	SpawnPowerShell(projectPath string) error
	// OpenInVSCode opens the project in VS Code (or VS Code Insiders).
	// Resolves the binary via: code → code-insiders → %LOCALAPPDATA% fallback.
	// Returns an informative error if no VS Code installation is found.
	OpenInVSCode(projectPath string) error
}

// Clipboard is the system clipboard write boundary.
// The TUI depends on this interface; tests inject a MockClipboard.
type Clipboard interface {
	WriteAll(text string) error
}

// NewOpener returns the platform-appropriate Opener. The concrete type is
// resolved at compile time by newPlatformOpener: a real launcher on Windows
// (windows.go) and a "not supported yet" stub elsewhere (opener_other.go).
// TODO(cross-platform): real Linux/macOS launchers land in a follow-up change.
func NewOpener() Opener {
	return newPlatformOpener()
}

// NewClipboard returns the production clipboard backed by github.com/atotto/clipboard.
func NewClipboard() Clipboard {
	return &atottoClipboard{}
}
