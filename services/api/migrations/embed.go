package migrations

import "embed"

// Files contains SQL migrations embedded for API startup.
//
//go:embed *.sql
var Files embed.FS
