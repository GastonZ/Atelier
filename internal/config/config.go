package config

import "errors"

// ErrNotImplemented is returned by Load() until config support is added in a future change.
var ErrNotImplemented = errors.New("config: not yet implemented")

// Config holds atelier runtime configuration. Currently a placeholder.
type Config struct{}

// Load reads the atelier config from disk. Currently a stub.
func Load() (*Config, error) {
	return nil, ErrNotImplemented
}
