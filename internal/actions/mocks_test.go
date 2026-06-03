package actions_test

// MockOpener captures call arguments and returns canned errors.
// It implements the Opener interface.
type MockOpener struct {
	LaunchInDirCalls      []string // "path|command|arg1 arg2" per call
	OpenInClaudeCodeCalls []string // projectPath per call
	SpawnPowerShellCalls  []string
	OpenInVSCodeCalls     []string

	LaunchInDirErr      error
	OpenInClaudeCodeErr error
	SpawnPowerShellErr  error
	OpenInVSCodeErr     error
}

func (m *MockOpener) LaunchInDir(projectPath, command string, args ...string) error {
	rec := projectPath + "|" + command
	for _, a := range args {
		rec += " " + a
	}
	m.LaunchInDirCalls = append(m.LaunchInDirCalls, rec)
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
// It implements the Clipboard interface.
type MockClipboard struct {
	Writes   []string
	WriteErr error
}

func (m *MockClipboard) WriteAll(text string) error {
	m.Writes = append(m.Writes, text)
	return m.WriteErr
}
