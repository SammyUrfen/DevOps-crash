// Package web carries the view into the binary, so the compiled server is the
// whole deployment unit. No sidecar, no volume mount, no static file server.
package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html
var files embed.FS

// FS serves the page at the root path.
func FS() fs.FS { return files }
