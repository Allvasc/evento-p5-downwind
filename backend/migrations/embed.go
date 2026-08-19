// Package migrations embeds the SQL migration files into the compiled binary, so the
// API can apply them itself on startup (see internal/db.Migrate) instead of requiring
// a separate `goose` invocation with direct network access to the database — which
// managed platforms (EasyPanel, etc.) often don't expose outside their internal
// network. Idempotent: goose tracks applied versions in `goose_db_version`, so running
// this on every boot is safe.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
