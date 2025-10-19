package frontend

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// GetDistFS returns the filesystem containing the frontend files
// This now loads from the runtime dist directory instead of embedded files
func GetDistFS() (fs.FS, error) {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build path to frontend/dist relative to executable
	distPath := filepath.Join(filepath.Dir(exePath), FrontendDistDir)

	// Check if dist directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("frontend dist directory not found at %s", distPath)
	}

	return os.DirFS(distPath), nil
}
