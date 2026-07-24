package migrations

import "embed"

// Files contains Atlas SQL migrations embedded for API startup.
//
//go:embed *.sql atlas.sum
var Files embed.FS
