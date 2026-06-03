package tui_test

// MockOpener captures call arguments and returns canned errors.
// Mirrors internal/actions/mocks_test.go for use in tui package tests.
type MockOpener struct {
	LaunchInDirCalls      []LaunchCall
	OpenInClaudeCodeCalls []string
	SpawnPowerShellCalls  []string
	OpenInVSCodeCalls     []string

	LaunchInDirErr      error
	OpenInClaudeCodeErr error
	SpawnPowerShellErr  error
	OpenInVSCodeErr     error
}

// LaunchCall records one LaunchInDir invocation for assertions.
type LaunchCall struct {
	Path    string
	Command string
	Args    []string
}

func (m *MockOpener) LaunchInDir(projectPath, command string, args ...string) error {
	m.LaunchInDirCalls = append(m.LaunchInDirCalls, LaunchCall{Path: projectPath, Command: command, Args: args})
	return m.LaunchInDirErr
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
// Mirrors internal/actions/mocks_test.go for use in tui package tests.
type MockClipboard struct {
	Writes   []string
	WriteErr error
}

func (m *MockClipboard) WriteAll(text string) error {
	m.Writes = append(m.Writes, text)
	return m.WriteErr
}
