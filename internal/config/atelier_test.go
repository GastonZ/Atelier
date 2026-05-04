package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gastonz/atelier/internal/config"
)

// --- helpers ---

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

// --- DefaultAtelierConfig ---

func TestDefaultAtelierConfig(t *testing.T) {
	cfg := config.DefaultAtelierConfig()

	if cfg.ActiveWindowMinutes != 15 {
		t.Errorf("ActiveWindowMinutes = %d, want 15", cfg.ActiveWindowMinutes)
	}
	if cfg.PollingIntervalMs != 500 {
		t.Errorf("PollingIntervalMs = %d, want 500", cfg.PollingIntervalMs)
	}
}

// --- LoadAtelierConfig ---

func TestLoadAtelierConfig_MissingFile_UsesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml") // file does not exist

	cfg, err := config.LoadAtelierConfig(path)

	if err != nil {
		t.Errorf("LoadAtelierConfig(missing) error = %v, want nil", err)
	}
	if cfg.ActiveWindowMinutes != 15 {
		t.Errorf("ActiveWindowMinutes = %d, want 15", cfg.ActiveWindowMinutes)
	}
	if cfg.PollingIntervalMs != 500 {
		t.Errorf("PollingIntervalMs = %d, want 500", cfg.PollingIntervalMs)
	}
}

func TestLoadAtelierConfig_ValidFile_ParsedCorrectly(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yaml", `
active_window_minutes: 30
polling_interval_ms: 1000
`)

	cfg, err := config.LoadAtelierConfig(path)

	if err != nil {
		t.Errorf("LoadAtelierConfig(valid) error = %v, want nil", err)
	}
	if cfg.ActiveWindowMinutes != 30 {
		t.Errorf("ActiveWindowMinutes = %d, want 30", cfg.ActiveWindowMinutes)
	}
	if cfg.PollingIntervalMs != 1000 {
		t.Errorf("PollingIntervalMs = %d, want 1000", cfg.PollingIntervalMs)
	}
}

func TestLoadAtelierConfig_MalformedFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yaml", `
active_window_minutes: [this is: not: valid: yaml
`)

	cfg, err := config.LoadAtelierConfig(path)

	if err == nil {
		t.Error("LoadAtelierConfig(malformed) error = nil, want non-nil error")
	}
	// Even on error, returned struct must carry defaults (caller may choose to use them)
	if cfg.ActiveWindowMinutes != 15 {
		t.Errorf("ActiveWindowMinutes on error = %d, want default 15", cfg.ActiveWindowMinutes)
	}
	if cfg.PollingIntervalMs != 500 {
		t.Errorf("PollingIntervalMs on error = %d, want default 500", cfg.PollingIntervalMs)
	}
}

// --- Table-driven triangulation ---

func TestLoadAtelierConfig_PartialFile_MissingFieldsUseDefaults(t *testing.T) {
	cases := []struct {
		name           string
		yaml           string
		wantWindowMins int
		wantPollingMs  int
	}{
		{
			name:           "only active_window_minutes set",
			yaml:           "active_window_minutes: 45\n",
			wantWindowMins: 45,
			wantPollingMs:  500, // default
		},
		{
			name:           "only polling_interval_ms set",
			yaml:           "polling_interval_ms: 250\n",
			wantWindowMins: 15,  // default
			wantPollingMs:  250,
		},
		{
			name:           "empty file uses all defaults",
			yaml:           "",
			wantWindowMins: 15,
			wantPollingMs:  500,
		},
		{
			name:           "zero values treated as explicit zeros (not defaults)",
			yaml:           "active_window_minutes: 0\npolling_interval_ms: 0\n",
			wantWindowMins: 0,
			wantPollingMs:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "config.yaml", tc.yaml)

			cfg, err := config.LoadAtelierConfig(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.ActiveWindowMinutes != tc.wantWindowMins {
				t.Errorf("ActiveWindowMinutes = %d, want %d", cfg.ActiveWindowMinutes, tc.wantWindowMins)
			}
			if cfg.PollingIntervalMs != tc.wantPollingMs {
				t.Errorf("PollingIntervalMs = %d, want %d", cfg.PollingIntervalMs, tc.wantPollingMs)
			}
		})
	}
}

// --- ActiveWindow and PollingInterval helpers ---

func TestAtelierConfig_ActiveWindow(t *testing.T) {
	cfg := config.AtelierConfig{ActiveWindowMinutes: 30}
	got := cfg.ActiveWindow()
	const wantNs = int64(30) * 60 * 1_000_000_000 // 30 minutes in nanoseconds
	if int64(got) != wantNs {
		t.Errorf("ActiveWindow() = %v, want %v ns", got, wantNs)
	}
}

func TestAtelierConfig_PollingInterval(t *testing.T) {
	cfg := config.AtelierConfig{PollingIntervalMs: 500}
	got := cfg.PollingInterval()
	const wantNs = int64(500) * 1_000_000 // 500ms in nanoseconds
	if int64(got) != wantNs {
		t.Errorf("PollingInterval() = %v, want %v ns", got, wantNs)
	}
}

// --- LoadAtelierConfig OS error (path is a directory, not a file) ---

func TestLoadAtelierConfig_PathIsDirectory_ReturnsError(t *testing.T) {
	dir := t.TempDir() // pass the directory itself — ReadFile on a dir returns an error

	cfg, err := config.LoadAtelierConfig(dir)

	if err == nil {
		t.Error("LoadAtelierConfig(dir) error = nil, want non-nil error")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Error("error should NOT be ErrNotExist for a directory path")
	}
	// Defaults must still be returned
	if cfg.ActiveWindowMinutes != 15 {
		t.Errorf("ActiveWindowMinutes = %d, want default 15", cfg.ActiveWindowMinutes)
	}
	if cfg.PollingIntervalMs != 500 {
		t.Errorf("PollingIntervalMs = %d, want default 500", cfg.PollingIntervalMs)
	}
}

// --- DefaultAtelierConfigPath ---

func TestDefaultAtelierConfigPath_ContainsAtelier(t *testing.T) {
	path := config.DefaultAtelierConfigPath()
	if path == "" {
		t.Error("DefaultAtelierConfigPath() = \"\", want non-empty path")
	}
	// Must contain .atelier in the path
	base := filepath.Base(filepath.Dir(path))
	if base != ".atelier" {
		t.Errorf("parent dir = %q, want .atelier", base)
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("filename = %q, want config.yaml", filepath.Base(path))
	}
}
