package actions_test

// MockOpener captures call arguments and returns canned errors.
// It implements the Opener interface.
type MockOpener struct {
	OpenInClaudeCodeCalls []string // projectPath per call
	SpawnPowerShellCalls  []string
	OpenInVSCodeCalls     []string

	OpenInClaudeCodeErr error
	SpawnPowerShellErr  error
	OpenInVSCodeErr     error
}

func (m *MockOpener) OpenInClaudeCode(projectPath string) error {
	m.OpenInClaudeCodeCalls = append(m.OpenInClaudeCodeCalls, projectPath)
	return m.OpenInClaudeCodeErr
}

func (m *MockOpener) SpawnPowerShell(projectPath string) error {
	m.SpawnPowerShellCalls = append(m.SpawnPowerShellCalls, projectPath)
	return m.SpawnPowerShellErr
}

func (m *MockOpener) OpenInVSCode(projectPath string) error {
	m.OpenInVSCodeCalls = append(m.OpenInVSCodeCalls, projectPath)
	return m.OpenInVSCodeErr
}

// MockClipboard captures writes and returns canned errors.
// It implements the Clipboard interface.
type MockClipboard struct {
	Writes   []string
	WriteErr error
}

func (m *MockClipboard) WriteAll(text string) error {
	m.Writes = append(m.Writes, text)
	return m.WriteErr
}
