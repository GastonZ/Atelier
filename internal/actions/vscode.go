//go:build windows

package actions

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// vsCodeExecCommand is the testability seam for the VS Code spawn command.
var vsCodeExecCommand = exec.Command

// vsCodeLookPath is the testability seam for exec.LookPath.
var vsCodeLookPath = exec.LookPath

// vsCodeStatCheck is the testability seam for path existence checks.
// When nil, defaults to the real os.Stat check.
var vsCodeStatCheck func(path string) bool

// SetVSCodeExecCommand replaces the exec.Command seam for VS Code spawn.
func SetVSCodeExecCommand(fn func(name string, arg ...string) *exec.Cmd) {
	vsCodeExecCommand = fn
}

// SetLookPath replaces the exec.LookPath seam for VS Code binary detection.
func SetLookPath(fn func(file string) (string, error)) {
	vsCodeLookPath = fn
}

// SetVSCodeStatCheck replaces the stat-check seam for path existence checks.
// Pass nil to restore the real os.Stat check.
func SetVSCodeStatCheck(fn func(path string) bool) {
	vsCodeStatCheck = fn
}

func localAppDataBase() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, "AppData", "Local")
	}
	return base
}

// localAppDataCodePath returns the hard-coded fallback path for code.cmd.
func localAppDataCodePath() string {
	return filepath.Join(localAppDataBase(), "Programs", "Microsoft VS Code", "bin", "code.cmd")
}

// localAppDataCodeExePath returns the hard-coded path for Code.exe.
func localAppDataCodeExePath() string {
	return filepath.Join(localAppDataBase(), "Programs", "Microsoft VS Code", "Code.exe")
}

func pathExists(path string) bool {
	if vsCodeStatCheck != nil {
		return vsCodeStatCheck(path)
	}
	_, err := os.Stat(path)
	return err == nil
}

// fallbackExists is an alias for pathExists, kept for backward compatibility.
func fallbackExists(path string) bool { return pathExists(path) }

// codeExeFromCmdPath derives Code.exe from a code.cmd path.
// code.cmd lives at <install>/bin/code.cmd; Code.exe lives at <install>/Code.exe.
func codeExeFromCmdPath(cmdPath string) string {
	exe := filepath.Join(filepath.Dir(filepath.Dir(cmdPath)), "Code.exe")
	if pathExists(exe) {
		return exe
	}
	return ""
}

// openInVSCodeImpl implements the VS Code detection chain.
// Prefers spawning Code.exe directly (no console window) over the cmd.exe wrapper.
func openInVSCodeImpl(projectPath string) error {
	// Step 1: code on PATH → derive Code.exe from the code.cmd location.
	if cmdPath, err := vsCodeLookPath("code"); err == nil {
		if exePath := codeExeFromCmdPath(cmdPath); exePath != "" {
			return spawnExe(exePath, projectPath)
		}
		return spawnViaCmd("code", projectPath)
	}

	// Step 2: code-insiders on PATH.
	if cmdPath, err := vsCodeLookPath("code-insiders"); err == nil {
		if exePath := codeExeFromCmdPath(cmdPath); exePath != "" {
			return spawnExe(exePath, projectPath)
		}
		return spawnViaCmd("code-insiders", projectPath)
	}

	// Step 3: Code.exe at the known user-install location.
	if exeFallback := localAppDataCodeExePath(); pathExists(exeFallback) {
		return spawnExe(exeFallback, projectPath)
	}

	// Step 4: code.cmd at the known user-install location (last resort).
	if cmdFallback := localAppDataCodePath(); pathExists(cmdFallback) {
		return spawnViaCmd(cmdFallback, projectPath)
	}

	return errors.New("VS Code no encontrado. Instalalo o agregálo al PATH.")
}

// spawnExe spawns a GUI executable directly — no cmd.exe wrapper, no console window.
func spawnExe(exePath, projectPath string) error {
	cmd := vsCodeExecCommand(exePath, projectPath)
	return cmd.Start()
}

// spawnViaCmd is the fallback for batch-file launchers.
// HideWindow suppresses the cmd.exe console flash.
func spawnViaCmd(binary, projectPath string) error {
	cmd := vsCodeExecCommand("cmd.exe", "/c", "start", "", binary, projectPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
