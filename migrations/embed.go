package migrations

import "embed"

// Files embeds SQL migrations for sqlite/postgres.
//
//go:embed sqlite/*.sql postgres/*.sql
var Files embed.FS
