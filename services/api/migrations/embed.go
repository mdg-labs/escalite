package migrations

import "embed"

// Files contains SQL migrations embedded for API startup.
//
//go:embed *.sql atlas.sum
var Files embed.FS
