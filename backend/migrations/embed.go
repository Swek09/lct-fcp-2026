package migrations

import "embed"

// Files contains the versioned SQL schema shipped with the API binary.
//
//go:embed *.sql
var Files embed.FS
