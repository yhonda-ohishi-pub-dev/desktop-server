package frontend

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	FrontendRepo    = "yhonda-ohishi-pub-dev/desktop-server-front"
	FrontendDistDir = "frontend/dist"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// GetLatestFrontendVersion fetches the latest release tag from GitHub
func GetLatestFrontendVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", FrontendRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return release.TagName, nil
}

// GetFrontendVersion returns the frontend version to download
func GetFrontendVersion() string {
	// Try to get the latest version from GitHub
	if version, err := GetLatestFrontendVersion(); err == nil {
		return version
	}
	// Fallback to a known version
	return "v1.4.0"
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

	fmt.Println("Downloading latest frontend release...")

	// Get the latest version from GitHub
	latestVersion, err := GetLatestFrontendVersion()
	if err != nil {
		fmt.Printf("Warning: Failed to fetch latest version: %v\n", err)
		latestVersion = "v1.4.0" // Fallback version
	}

	fmt.Printf("Fetching frontend version: %s\n", latestVersion)

	// Try the latest version first
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/desktop-server-frontend-%s.zip", FrontendRepo, latestVersion, latestVersion)
	fmt.Printf("Checking URL: %s\n", downloadURL)

	// Fallback to v1.2.0 if latest doesn't exist
	if !urlExists(downloadURL) {
		fmt.Printf("%s not found, falling back to v1.2.0...\n", latestVersion)
		downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/v1.2.0/desktop-server-frontend-v1.2.0.zip", FrontendRepo)
		fmt.Printf("Fallback URL: %s\n", downloadURL)
	} else {
		fmt.Printf("URL exists, downloading %s\n", latestVersion)
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
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects
			return nil
		},
	}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// Accept both 200 OK and 302 Found (GitHub redirects)
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound
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
