// Package migrations embeds the versioned SQL migrations so the binaries do not depend on the working directory.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
