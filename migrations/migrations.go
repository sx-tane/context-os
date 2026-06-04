// Package migrations exposes the embedded SQL migration files for use by the API startup.
package migrations

import "embed"

// Files contains all *.sql migration files in this directory.
//
//go:embed *.sql
var Files embed.FS
