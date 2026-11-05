// Package migrations embeds the SQL migrations, so the binary applies them without a working
// directory or an external goose.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
