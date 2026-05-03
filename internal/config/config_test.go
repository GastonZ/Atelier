package config_test

import (
	"errors"
	"testing"

	"github.com/gastonz/atelier/internal/config"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		wantCfg bool
		wantErr error
	}{
		{
			name:    "returns ErrNotImplemented",
			wantCfg: false,
			wantErr: config.ErrNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load()

			if tt.wantCfg && cfg == nil {
				t.Error("Load() returned nil Config, want non-nil")
			}
			if !tt.wantCfg && cfg != nil {
				t.Errorf("Load() returned non-nil Config %v, want nil", cfg)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Load() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
