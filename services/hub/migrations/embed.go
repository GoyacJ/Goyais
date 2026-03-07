package migrations

import "embed"

// Files exposes the versioned SQL migrations for stage-0/1 persistence setup.
//
//go:embed *.sql
var Files embed.FS
