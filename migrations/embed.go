// Package migrations embeds the SQL migration files into the binary so the
// service can bring the schema up to date on start-up.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
