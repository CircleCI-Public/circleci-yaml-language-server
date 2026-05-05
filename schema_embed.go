package schema

import _ "embed"

// EmbeddedSchemaJSON contains the built-in schema.json, embedded at compile time.
// This allows the binary to work without requiring a separate schema file.
//
//go:embed schema.json
var EmbeddedSchemaJSON []byte
