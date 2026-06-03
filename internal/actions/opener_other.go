//go:build !windows

package actions

import (
	"fmt"
	"runtime"
)

// otherOpener is the non-Windows fallback Opener. Atelier's launchers currently
// target Windows only (cmd.exe / PowerShell / %LOCALAPPDATA% VS Code lookup), so
// on other platforms every action returns a clear "not supported yet" error
// instead of failing to compile. The TUI surfaces this as a flash message.
//
// This mirrors the build-tag split already used by internal/audio
// (capture_windows.go / capture_other.go) and internal/nowplaying.
type otherOpener struct{}

// newPlatformOpener returns the non-Windows stub Opener. Selected at compile
// time via the !windows build constraint; see windows.go for the real launcher.
func newPlatformOpener() Opener {
	return &otherOpener{}
}

func unsupported(action string) error {
	return fmt.Errorf("%s is not supported on %s yet (Windows only for now)", action, runtime.GOOS)
}

func (o *otherOpener) OpenInClaudeCode(string) error { return unsupported("opening Claude Code") }

func (o *otherOpener) SpawnPowerShell(string) error { return unsupported("spawning a shell") }

func (o *otherOpener) OpenInVSCode(string) error { return unsupported("opening VS Code") }
