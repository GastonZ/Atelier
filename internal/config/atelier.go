package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Launcher describes one configurable "open this project in <tool>" action shown
// in the per-project actions menu. Command is resolved on PATH and spawned
// detached in the project directory; Args are passed verbatim after it (some CLIs
// need a subcommand or flag). Example YAML:
//
//	launchers:
//	  - { label: "Claude Code", command: "claude" }
//	  - { label: "Codex",       command: "codex" }
//	  - { label: "Aider",       command: "aider", args: ["--no-auto-commits"] }
type Launcher struct {
	Label   string   `yaml:"label"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
}

// AtelierConfig holds runtime knobs read from ~/.atelier/config.yaml.
// Defaults: ActiveWindowMinutes=15, PollingIntervalMs=500, Launchers=DefaultLaunchers().
type AtelierConfig struct {
	// ActiveWindowMinutes controls how recent (in minutes) a session's last-event
	// mtime must be to appear in the active set. Default: 15.
	ActiveWindowMinutes int `yaml:"active_window_minutes"`

	// PollingIntervalMs is the fallback-polling tick period in milliseconds.
	// Default: 500.
	PollingIntervalMs int `yaml:"polling_interval_ms"`

	// Launchers are the configurable agent/editor entries at the top of the
	// per-project actions menu. Absent in the file → DefaultLaunchers(); present
	// (even empty) → used verbatim, so users can fully customise or clear them.
	Launchers []Launcher `yaml:"launchers"`
}

// DefaultLaunchers returns the agent launchers shipped out of the box: the
// best-known AI coding CLIs. Users override the whole list via config.yaml.
func DefaultLaunchers() []Launcher {
	return []Launcher{
		{Label: "Claude Code", Command: "claude"},
		{Label: "Codex", Command: "codex"},
		{Label: "Gemini", Command: "gemini"},
	}
}

// DefaultAtelierConfig returns an AtelierConfig populated with the documented
// defaults (active_window=15 min, polling=500ms, default launchers). Never errors.
func DefaultAtelierConfig() AtelierConfig {
	return AtelierConfig{
		ActiveWindowMinutes: 15,
		PollingIntervalMs:   500,
		Launchers:           DefaultLaunchers(),
	}
}

// ActiveWindow converts ActiveWindowMinutes to a time.Duration.
func (c AtelierConfig) ActiveWindow() time.Duration {
	return time.Duration(c.ActiveWindowMinutes) * time.Minute
}

// PollingInterval converts PollingIntervalMs to a time.Duration.
func (c AtelierConfig) PollingInterval() time.Duration {
	return time.Duration(c.PollingIntervalMs) * time.Millisecond
}

// DefaultAtelierConfigPath returns the canonical path to the user's atelier
// config file: ~/.atelier/config.yaml. It expands the home directory via
// os.UserHomeDir(). On error, it returns an empty string.
func DefaultAtelierConfigPath() string {
	return atelierConfigPath(os.UserHomeDir)
}

// atelierConfigPath is the testable core of DefaultAtelierConfigPath. It accepts
// a homeDir function so tests can inject a controlled value or an error.
func atelierConfigPath(homeDir func() (string, error)) string {
	home, err := homeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".atelier", "config.yaml")
}

// rawAtelierConfig mirrors AtelierConfig but uses pointer fields so that yaml.v3
// can distinguish between "field absent" and "field explicitly set to zero".
type rawAtelierConfig struct {
	ActiveWindowMinutes *int        `yaml:"active_window_minutes"`
	PollingIntervalMs   *int        `yaml:"polling_interval_ms"`
	Launchers           *[]Launcher `yaml:"launchers"`
}

// LoadAtelierConfig reads path and returns an AtelierConfig.
//
// Behaviour contract (matches spec G8 / R8.1–R8.3):
//   - File missing    → returns defaults, nil error (silent)
//   - File valid      → returns parsed values; absent fields receive defaults
//   - File malformed  → returns defaults, non-nil error (caller must surface warning)
func LoadAtelierConfig(path string) (AtelierConfig, error) {
	defaults := DefaultAtelierConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Missing file is not an error — use defaults silently.
			return defaults, nil
		}
		// Unexpected OS error (permissions, etc.) — treat like malformed.
		return defaults, err
	}

	var raw rawAtelierConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return defaults, err
	}

	result := defaults // start from defaults; override only fields that were present
	if raw.ActiveWindowMinutes != nil {
		result.ActiveWindowMinutes = *raw.ActiveWindowMinutes
	}
	if raw.PollingIntervalMs != nil {
		result.PollingIntervalMs = *raw.PollingIntervalMs
	}
	if raw.Launchers != nil {
		// Present (even as an empty list) → honor the user's choice verbatim.
		result.Launchers = *raw.Launchers
	}
	return result, nil
}

// SaveAtelierConfig writes cfg to path as YAML, atomically (temp file + rename),
// creating the parent directory if needed. This is what lets the TUI persist
// in-app edits (e.g. the launcher manager). The whole config is serialized, so
// any hand-written comments in an existing file are lost — an accepted trade-off
// for editing from the UI.
func SaveAtelierConfig(path string, cfg AtelierConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
