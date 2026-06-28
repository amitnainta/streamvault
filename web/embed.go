// Package web embeds the compiled React frontend (web/dist) into the binary.
// During development, web/dist may be empty — the Vite dev server at :5174 is used instead.
package web

import "embed"

//go:embed dist
var DistFS embed.FS
