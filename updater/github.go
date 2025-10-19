package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const (
	// Set your GitHub repository here
	GitHubOwner = "yhonda-ohishi-pub-dev"
	GitHubRepo  = "desktop-server"
)

var (
	// Version is set via ldflags during build
	Version        = "dev"
	CurrentVersion = Version // Use injected version
)

type GitHubRelease struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type UpdateInfo struct {
	Available      bool
	LatestVersion  string
	CurrentVersion string
	DownloadURL    string
	ReleaseNotes   string
}

// CheckForUpdates checks if a new version is available on GitHub
func CheckForUpdates() (*UpdateInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	info := &UpdateInfo{
		LatestVersion:  release.TagName,
		CurrentVersion: CurrentVersion,
		ReleaseNotes:   release.Body,
	}

	// Check if update is available
	if release.TagName != CurrentVersion && release.TagName != "" {
		info.Available = true

		// Find the appropriate asset for current OS
		for _, asset := range release.Assets {
			// Specifically look for desktop-server.exe
			if asset.Name == "desktop-server.exe" {
				info.DownloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	return info, nil
}

// DownloadUpdate downloads the new version
func DownloadUpdate(downloadURL string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "desktop-server-update.exe")

	out, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write update: %w", err)
	}

	return tmpFile, nil
}

// ApplyUpdate replaces the current executable with the new one
// Since the exe is running, we create a batch script to do the update after exit
func ApplyUpdate(newExePath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create a batch script to perform the update
	batchScript := currentExe + "_update.bat"

	// Batch script content:
	// 1. Wait for current process to exit
	// 2. Backup current exe
	// 3. Copy new exe to current location
	// 4. Start new exe
	// 5. Delete batch script
	scriptContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak > nul
if exist "%s.bak" del "%s.bak"
move "%s" "%s.bak"
move "%s" "%s"
start "" "%s"
del "%%~f0"
`, currentExe, currentExe, currentExe, currentExe, newExePath, currentExe, currentExe)

	if err := os.WriteFile(batchScript, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create update script: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// RestartApplication restarts the application after update
// This executes the update batch script and exits the current process
func RestartApplication() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// Execute the update batch script
	batchScript := executable + "_update.bat"

	// Start the batch script in a detached process
	cmd := exec.Command("cmd", "/C", batchScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000, // CREATE_NO_WINDOW
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update script: %w", err)
	}

	return nil
}

// UpdateETCScraper downloads the latest etc_meisai_scraper.exe from GitHub releases
func UpdateETCScraper() error {
	// Get latest release info
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Find etc_meisai_scraper.exe asset
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == "etc_meisai_scraper.exe" {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("etc_meisai_scraper.exe not found in release assets")
	}

	// Download the file
	client = &http.Client{Timeout: 5 * time.Minute}
	resp, err = client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Get current executable directory
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	targetPath := filepath.Join(filepath.Dir(currentExe), "etc_meisai_scraper.exe")

	// Create temp file first
	tmpFile := targetPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Backup existing file if it exists
	if _, err := os.Stat(targetPath); err == nil {
		backupPath := targetPath + ".bak"
		os.Remove(backupPath) // Remove old backup if exists
		if err := os.Rename(targetPath, backupPath); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to backup existing file: %w", err)
		}
	}

	// Move temp file to final location
	if err := os.Rename(tmpFile, targetPath); err != nil {
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	return nil
}
