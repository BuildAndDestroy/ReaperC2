// Package docs ships operator documentation as embedded Markdown (see *.md in this directory).
package docs

import "embed"

// Markdown contains all bundled documentation pages (*.md only; embed.go is excluded).
//
//go:embed *.md
var Markdown embed.FS
