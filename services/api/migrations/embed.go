package migrations

import "embed"

// Files contains goose SQL migrations embedded for API startup.
//
//go:embed *.sql
var Files embed.FS
