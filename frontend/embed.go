package frontend

import (
	"embed"
	"io/fs"
)

// Embed the frontend dist files into the binary
// Use all:dist to include hidden files and handle empty directories
//
//go:embed all:dist
var distFS embed.FS

// GetDistFS returns the embedded filesystem containing the frontend files
func GetDistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
