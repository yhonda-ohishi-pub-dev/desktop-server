package systray

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"desktop-server/frontend"
	"desktop-server/internal/etcscraper"
	"desktop-server/updater"

	"github.com/getlantern/systray"
)

var (
	scraperManager *etcscraper.Manager
	currentJobID   string
	jobStatusMenu  *systray.MenuItem
)

func Run(ctx context.Context, onExit func()) {
	systray.Run(onReady(ctx, onExit), onExitFunc(onExit))
}

// SetScraperManager sets the etc_meisai_scraper manager
func SetScraperManager(manager *etcscraper.Manager) {
	scraperManager = manager
}

func onReady(ctx context.Context, onExit func()) func() {
	return func() {
		// Set icon, title and tooltip
		systray.SetIcon(getIcon())
		systray.SetTitle("DS")
		systray.SetTooltip(fmt.Sprintf("Desktop Server %s - Running on localhost:8080", updater.CurrentVersion))

		// Add menu items
		mOpen := systray.AddMenuItem("Open App", "Open in browser")
		systray.AddSeparator()

		// Version info (disabled menu items)
		mBackendVersion := systray.AddMenuItem(fmt.Sprintf("Backend: %s", updater.CurrentVersion), "Current backend version")
		mBackendVersion.Disable()
		mFrontendVersion := systray.AddMenuItem(fmt.Sprintf("Frontend: %s", frontend.FrontendVersion), "Current frontend version")
		mFrontendVersion.Disable()
		systray.AddSeparator()

		mCheckUpdate := systray.AddMenuItem("Update Backend", "Check for new backend version")
		mUpdateFrontend := systray.AddMenuItem("Update Frontend", "Download latest frontend")
		systray.AddSeparator()
		mETCDownload := systray.AddMenuItem("Download ETC Data", "Download ETC meisai data")

		// Check if etc_meisai_scraper.exe exists
		if !isETCScraperAvailable() {
			mETCDownload.SetTitle("Download ETC Data (Not Available)")
			mETCDownload.Disable()
		}

		jobStatusMenu = systray.AddMenuItem("ETC Status: Idle", "ETC download job status")
		jobStatusMenu.Disable()
		jobStatusMenu.Hide()
		mAbout := systray.AddMenuItem("About", "About Desktop Server")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")

		// Handle menu clicks
		go func() {
			for {
				select {
				case <-ctx.Done():
					systray.Quit()
					return
				case <-mOpen.ClickedCh:
					openBrowser("http://localhost:8080")
				case <-mCheckUpdate.ClickedCh:
					go checkForUpdates()
				case <-mUpdateFrontend.ClickedCh:
					go updateFrontend()
				case <-mETCDownload.ClickedCh:
					go downloadETCData()
				case <-mAbout.ClickedCh:
					showAbout()
				case <-mQuit.ClickedCh:
					systray.Quit()
					if onExit != nil {
						onExit()
					}
					return
				}
			}
		}()
	}
}

func onExitFunc(onExit func()) func() {
	return func() {
		if onExit != nil {
			onExit()
		}
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}
}

func checkForUpdates() {
	fmt.Println("Checking for updates...")

	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		showMessage("Update Check Failed", fmt.Sprintf("Failed to check for updates: %v", err))
		return
	}

	if !updateInfo.Available {
		showMessage("No Updates Available", fmt.Sprintf("You are running the latest version (%s)", updateInfo.CurrentVersion))
		return
	}

	// New version available
	message := fmt.Sprintf("A new version is available!\n\nCurrent: %s\nLatest: %s\n\nWould you like to download it?",
		updateInfo.CurrentVersion, updateInfo.LatestVersion)

	if confirmUpdate(message) {
		performUpdate(updateInfo.DownloadURL)
	}
}

func performUpdate(downloadURL string) {
	fmt.Println("Downloading update...")

	tmpFile, err := updater.DownloadUpdate(downloadURL)
	if err != nil {
		showMessage("Update Failed", fmt.Sprintf("Failed to download update: %v", err))
		return
	}

	if confirmUpdate("Update downloaded. Apply update and restart?") {
		if err := updater.ApplyUpdate(tmpFile); err != nil {
			showMessage("Update Failed", fmt.Sprintf("Failed to apply update: %v", err))
			return
		}

		// Restart application
		updater.RestartApplication()
		os.Exit(0)
	}
}

func updateFrontend() {
	fmt.Println("Updating frontend...")

	err := frontend.DownloadLatestRelease(true)
	if err != nil {
		showMessage("Frontend Update Failed", fmt.Sprintf("Failed to update frontend: %v", err))
		return
	}

	showMessage("Frontend Updated", "Frontend updated successfully! Please restart the application to apply changes.")
}

func downloadETCData() {
	if scraperManager == nil {
		log.Println("ERROR: scraperManager is nil")
		showMessage("ETC Download Failed", "ETC scraper is not configured")
		return
	}

	log.Println("Starting ETC data download...")

	// Get client (auto-starts etc_meisai_scraper.exe if needed)
	log.Println("Getting ETC scraper client...")
	client, err := scraperManager.GetClient()
	if err != nil {
		log.Printf("ERROR: Failed to get client: %v", err)
		showMessage("ETC Download Failed", fmt.Sprintf("Failed to start ETC scraper: %v", err))
		return
	}
	log.Println("Client obtained successfully")

	// Start async download
	log.Println("Starting async download...")
	ctx := context.Background()

	// Get account info from environment variable
	// Expected format: ETC_CORP_ACCOUNTS=["userid1:password1","userid2:password2"]
	accountsEnv := os.Getenv("ETC_CORP_ACCOUNTS")
	if accountsEnv == "" {
		log.Println("ERROR: ETC_CORP_ACCOUNTS environment variable not set")
		showMessage("ETC Download Failed", "ETC_CORP_ACCOUNTS environment variable not set.\n\nPlease set it with format: [\"userid:password\"]")
		return
	}

	// Parse JSON array format
	accountsEnv = strings.TrimSpace(accountsEnv)
	accountsEnv = strings.Trim(accountsEnv, "[]")
	accountsEnv = strings.ReplaceAll(accountsEnv, "\"", "")

	accounts := strings.Split(accountsEnv, ",")
	log.Printf("Found %d account(s) from ETC_CORP_ACCOUNTS", len(accounts))

	jobID, err := client.DownloadAsync(ctx, accounts, "", "")
	if err != nil {
		log.Printf("ERROR: Failed to start download: %v", err)
		showMessage("ETC Download Failed", fmt.Sprintf("Failed to start download: %v", err))
		return
	}

	log.Printf("ETC download started, job ID: %s", jobID)
	currentJobID = jobID

	// Show status menu and start polling
	log.Println("Showing status menu and starting polling")
	jobStatusMenu.SetTitle("ETC Status: Starting...")
	jobStatusMenu.Show()

	// Start polling job status in background
	go pollJobStatus(client, jobID)
}

func pollJobStatus(client *etcscraper.Client, jobID string) {
	ctx := context.Background()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("Starting job status polling for job ID: %s", jobID)

	for {
		select {
		case <-ticker.C:
			log.Printf("Polling job status for: %s", jobID)
			status, err := client.GetJobStatus(ctx, jobID)
			if err != nil {
				log.Printf("Failed to get job status: %v", err)
				jobStatusMenu.SetTitle("ETC Status: Error")
				time.Sleep(5 * time.Second)
				jobStatusMenu.Hide()
				return
			}

			if status == nil {
				log.Printf("Job status is nil for job ID: %s", jobID)
				jobStatusMenu.SetTitle("ETC Status: Job not found")
				time.Sleep(5 * time.Second)
				jobStatusMenu.Hide()
				return
			}

			log.Printf("Job status: %s, Progress: %d/%d", status.Status, status.Progress, status.TotalRecords)

			// Update menu based on status
			switch status.Status {
			case "pending":
				jobStatusMenu.SetTitle("ETC Status: Pending...")
			case "running", "processing":
				if status.TotalRecords > 0 {
					progress := (status.Progress * 100) / status.TotalRecords
					jobStatusMenu.SetTitle(fmt.Sprintf("ETC Status: %d%% (%d/%d)", progress, status.Progress, status.TotalRecords))
				} else {
					jobStatusMenu.SetTitle("ETC Status: Processing...")
				}
			case "completed", "success":
				jobStatusMenu.SetTitle(fmt.Sprintf("ETC Status: Completed (%d records)", status.TotalRecords))

				// Clean up old download folders (keep only 10 most recent)
				if err := CleanupDownloadFolders(10); err != nil {
					log.Printf("Warning: Failed to cleanup old downloads: %v", err)
				}

				showMessage("ETC Download Completed", fmt.Sprintf("Successfully downloaded %d records", status.TotalRecords))
				time.Sleep(10 * time.Second)
				jobStatusMenu.Hide()
				return
			case "failed", "error":
				errorMsg := status.ErrorMessage
				if errorMsg == "" {
					errorMsg = "Unknown error"
				}
				jobStatusMenu.SetTitle("ETC Status: Failed")
				showMessage("ETC Download Failed", fmt.Sprintf("Download failed: %s", errorMsg))
				time.Sleep(10 * time.Second)
				jobStatusMenu.Hide()
				return
			default:
				jobStatusMenu.SetTitle(fmt.Sprintf("ETC Status: %s", status.Status))
			}
		}
	}
}

func showAbout() {
	message := fmt.Sprintf("Desktop Server\n\nBackend: %s\nFrontend: %s\n\nLocal database management tool with gRPC-Web API\n\nRunning on: localhost:8080",
		updater.CurrentVersion,
		frontend.FrontendVersion)
	showMessage("About Desktop Server", message)
}

func showMessage(title, message string) {
	// Windows message box
	if runtime.GOOS == "windows" {
		cmd := exec.Command("mshta", fmt.Sprintf("javascript:alert('%s');close();", message))
		cmd.Run()
	} else {
		fmt.Printf("%s: %s\n", title, message)
	}
}

func confirmUpdate(message string) bool {
	// For simplicity, auto-confirm on non-Windows
	if runtime.GOOS != "windows" {
		return true
	}

	// Windows: use simple confirmation (you can improve this with proper dialog)
	// For now, just return true (auto-update)
	fmt.Println(message)
	return true
}

func isETCScraperAvailable() bool {
	// Check if etc_meisai_scraper.exe exists in the same directory
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	dir := filepath.Dir(exePath)
	scraperPath := filepath.Join(dir, "etc_meisai_scraper.exe")

	_, err = os.Stat(scraperPath)
	return err == nil
}

// CleanupDownloadFolders keeps only the N most recent download folders
func CleanupDownloadFolders(keepCount int) error {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	downloadsDir := filepath.Join(filepath.Dir(exePath), "downloads")

	// Check if downloads directory exists
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	// Read all entries in downloads directory
	entries, err := os.ReadDir(downloadsDir)
	if err != nil {
		return fmt.Errorf("failed to read downloads directory: %w", err)
	}

	// Filter only directories and get their info
	type dirInfo struct {
		name    string
		modTime time.Time
	}

	var dirs []dirInfo
	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				log.Printf("Warning: Failed to get info for %s: %v", entry.Name(), err)
				continue
			}
			dirs = append(dirs, dirInfo{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	// If we have fewer than or equal to keepCount, nothing to delete
	if len(dirs) <= keepCount {
		log.Printf("Download folders: %d (within limit of %d)", len(dirs), keepCount)
		return nil
	}

	// Sort by modification time (oldest first)
	for i := 0; i < len(dirs); i++ {
		for j := i + 1; j < len(dirs); j++ {
			if dirs[i].modTime.After(dirs[j].modTime) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}

	// Delete oldest folders
	deleteCount := len(dirs) - keepCount
	log.Printf("Cleaning up %d old download folders (keeping %d most recent)", deleteCount, keepCount)

	for i := 0; i < deleteCount; i++ {
		dirPath := filepath.Join(downloadsDir, dirs[i].name)
		log.Printf("Deleting old download folder: %s", dirs[i].name)
		if err := os.RemoveAll(dirPath); err != nil {
			log.Printf("Warning: Failed to delete %s: %v", dirs[i].name, err)
		}
	}

	return nil
}

