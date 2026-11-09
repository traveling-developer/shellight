// Package frontend embeds the built dashboard frontend (npm run build).
package frontend

import (
	"embed"
	"io/fs"
)

// dist keeps a .gitkeep, so the package builds before the frontend is
// built; the dashboard is then empty.
//
//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend, rooted at dist.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // "dist" is a valid path
	}
	return sub
}
