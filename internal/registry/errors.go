package registry

import "errors"

// ErrNotFound is returned by Delete and Touch when the project id is not in the registry.
var ErrNotFound = errors.New("registry: project not found")

// ErrSchemaMismatch is returned by load when the on-disk schema_version != currentSchemaVersion.
var ErrSchemaMismatch = errors.New("registry: schema version mismatch")
