// Package migrations carries the .sql files into the binary. The files stay in
// a top level directory because that is where an operator looks for them, and
// go:embed cannot reach a parent directory from internal/database.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
