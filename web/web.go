// Package web holds the single HTML page and its stylesheet, embedded into the
// binary so the server and its tests run correctly from any working directory.
package web

import "embed"

//go:embed index.html.tmpl style.css
var FS embed.FS
