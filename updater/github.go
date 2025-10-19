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

	// Find the appropriate asset for current OS (always set URL)
	for _, asset := range release.Assets {
		// Specifically look for desktop-server.exe
		if asset.Name == "desktop-server.exe" {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	// Check if update is available
	if release.TagName != CurrentVersion && release.TagName != "" {
		info.Available = true
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
// Since the exe is running, we create a PowerShell script to do the update after exit
func ApplyUpdate(newExePath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create a PowerShell script to perform the update
	psScript := currentExe + "_update.ps1"

	// PowerShell script content:
	// 1. Wait for current process to exit
	// 2. Backup current exe
	// 3. Move new exe to current location
	// 4. Start new exe
	// 5. Delete script

	// Create log file path
	logPath := filepath.Join(filepath.Dir(currentExe), "logs", "update-script.log")

	scriptContent := fmt.Sprintf(`
# Log function
function Write-Log {
    param($Message)
    $timestamp = Get-Date -Format "yyyy/MM/dd HH:mm:ss"
    $logDir = Split-Path -Parent '%s'
    if (-not (Test-Path $logDir)) {
        New-Item -ItemType Directory -Force -Path $logDir | Out-Null
    }
    Add-Content -Path '%s' -Value "$timestamp $Message" -Encoding UTF8
}

Write-Log "UPDATE SCRIPT: Starting..."

# Wait for process to exit
Write-Log "UPDATE SCRIPT: Waiting 2 seconds for process to exit..."
Start-Sleep -Seconds 2

# Backup old exe
$oldExe = '%s'
$backupExe = '%s.bak'
$newExe = '%s'

Write-Log "UPDATE SCRIPT: Old exe: $oldExe"
Write-Log "UPDATE SCRIPT: New exe: $newExe"

try {
    if (Test-Path $backupExe) {
        Write-Log "UPDATE SCRIPT: Removing old backup..."
        Remove-Item $backupExe -Force
    }

    if (Test-Path $oldExe) {
        Write-Log "UPDATE SCRIPT: Moving old exe to backup..."
        Move-Item $oldExe $backupExe -Force
    }

    Write-Log "UPDATE SCRIPT: Moving new exe to location..."
    Move-Item $newExe $oldExe -Force

    Write-Log "UPDATE SCRIPT: Starting new exe..."
    Start-Process -FilePath $oldExe

    Write-Log "UPDATE SCRIPT: New exe started successfully"
} catch {
    Write-Log "UPDATE SCRIPT ERROR: $_"
}

# Wait a bit then delete this script
Start-Sleep -Seconds 1
Write-Log "UPDATE SCRIPT: Deleting script..."
Remove-Item $PSCommandPath -Force
`, logPath, logPath, currentExe, currentExe, newExePath)

	// Write with UTF-8 BOM so PowerShell can correctly read Japanese paths
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}
	scriptBytes := append(utf8BOM, []byte(scriptContent)...)
	if err := os.WriteFile(psScript, scriptBytes, 0755); err != nil {
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
// This executes the update PowerShell script and exits the current process
func RestartApplication() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// Execute the update PowerShell script
	psScript := executable + "_update.ps1"
	logPath := filepath.Join(filepath.Dir(executable), "logs", "update-script.log")

	// Create a wrapper script that redirects output to log file
	wrapperScript := executable + "_update_wrapper.ps1"
	wrapperContent := fmt.Sprintf(`
# Ensure logs directory exists
$logsDir = Split-Path -Parent '%s'
if (-not (Test-Path $logsDir)) {
    New-Item -ItemType Directory -Force -Path $logsDir | Out-Null
}

# Run the actual update script and capture all output
& '%s' *>&1 | Tee-Object -FilePath '%s'
`, logPath, psScript, logPath)

	// Write with UTF-8 BOM so PowerShell can correctly read Japanese paths
	utf8BOM := []byte{0xEF, 0xBB, 0xBF}
	wrapperBytes := append(utf8BOM, []byte(wrapperContent)...)
	if err := os.WriteFile(wrapperScript, wrapperBytes, 0755); err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}

	// Start wrapper PowerShell script in hidden window
	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-File", wrapperScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
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
