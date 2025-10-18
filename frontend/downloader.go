package frontend

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"desktop-server/updater"
)

const (
	FrontendRepo    = "yhonda-ohishi-pub-dev/desktop-server-front"
	FrontendDistDir = "frontend/dist"
)

// GetFrontendVersion returns the frontend version, using desktop-server's version
func GetFrontendVersion() string {
	// Use desktop-server's version for frontend
	if updater.Version != "" && updater.Version != "dev" {
		return updater.Version
	}
	// Fallback to v1.0.0 for dev builds
	return "v1.0.0"
}

// DownloadLatestRelease downloads the latest frontend release from GitHub
func DownloadLatestRelease(forceUpdate bool) error {
	// Check if frontend already exists
	if !forceUpdate {
		if _, err := os.Stat(FrontendDistDir); err == nil {
			fmt.Println("Frontend already exists, skipping download")
			return nil
		}
	}

	version := GetFrontendVersion()
	fmt.Printf("Downloading frontend release version %s...\n", version)

	// Try versioned URL first (matching desktop-server version)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/desktop-server-frontend-%s.zip",
		FrontendRepo, version, version)

	// If versioned URL doesn't exist, try v1.2.0 (current latest)
	if !urlExists(downloadURL) {
		fmt.Printf("Version %s not found, trying v1.2.0...\n", version)
		downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/v1.2.0/desktop-server-frontend-v1.2.0.zip", FrontendRepo)

		// Final fallback to v1.0.0
		if !urlExists(downloadURL) {
			fmt.Println("v1.2.0 not found, falling back to v1.0.0...")
			downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/v1.0.0/desktop-server-frontend-v1.0.0.zip", FrontendRepo)
		}
	}

	// Download the zip file
	tmpFile := "frontend_tmp.zip"
	if err := downloadFile(tmpFile, downloadURL); err != nil {
		return fmt.Errorf("failed to download frontend: %w", err)
	}
	defer os.Remove(tmpFile)

	// Extract the zip file
	if err := unzip(tmpFile, FrontendDistDir); err != nil {
		return fmt.Errorf("failed to extract frontend: %w", err)
	}

	fmt.Println("Frontend downloaded successfully")
	return nil
}

// urlExists checks if a URL is accessible
func urlExists(url string) bool {
	resp, err := http.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// unzip extracts a zip archive to a destination directory
func unzip(src, dest string) error {
	// Remove existing directory
	if err := os.RemoveAll(dest); err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		err := extractFile(f, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

// extractFile extracts a single file from a zip archive
func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(dest, f.Name)

	// Check for ZipSlip vulnerability
	if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
