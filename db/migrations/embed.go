package migrations

import "embed"

// Files contains the complete, versioned migration history used by the server.
// Embedding it makes startup independent from the current working directory and
// from an installed Goose CLI binary.
//
//go:embed *.sql
var Files embed.FS
