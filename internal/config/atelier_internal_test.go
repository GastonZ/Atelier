package config

// White-box tests for unexported helpers in atelier.go.
// These live in the `config` package (not `config_test`) to access unexported functions.

import (
	"errors"
	"testing"
)

func TestAtelierConfigPath_HomeDirError_ReturnsEmpty(t *testing.T) {
	errFake := errors.New("no home dir")
	got := atelierConfigPath(func() (string, error) {
		return "", errFake
	})
	if got != "" {
		t.Errorf("atelierConfigPath(err) = %q, want empty string", got)
	}
}

func TestAtelierConfigPath_ValidHomeDir(t *testing.T) {
	got := atelierConfigPath(func() (string, error) {
		return "/home/testuser", nil
	})
	if got == "" {
		t.Error("atelierConfigPath(ok) = \"\", want non-empty path")
	}
	// Must end with config.yaml inside .atelier
	wantSuffix := ".atelier"
	if base := dirName(got); base != wantSuffix {
		t.Errorf("parent dir = %q, want %q", base, wantSuffix)
	}
}

// dirName is filepath.Base(filepath.Dir(p)) — avoids importing path/filepath in test.
func dirName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			// found last separator, now look for the previous one
			for j := i - 1; j >= 0; j-- {
				if p[j] == '/' || p[j] == '\\' {
					return p[j+1 : i]
				}
			}
			return p[:i]
		}
	}
	return ""
}
