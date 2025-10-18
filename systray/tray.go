package systray

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"desktop-server/updater"

	"github.com/getlantern/systray"
)

func Run(ctx context.Context, onExit func()) {
	systray.Run(onReady(ctx, onExit), onExitFunc(onExit))
}

func onReady(ctx context.Context, onExit func()) func() {
	return func() {
		// Set title and tooltip (icon is optional)
		systray.SetTitle("DS")
		systray.SetTooltip("Desktop Server - Running on localhost:8080")

		// Add menu items
		mOpen := systray.AddMenuItem("Open App", "Open in browser")
		systray.AddSeparator()
		mCheckUpdate := systray.AddMenuItem("Check for Updates", "Check for new version")
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

func showAbout() {
	message := fmt.Sprintf("Desktop Server %s\n\nLocal database management tool with gRPC-Web API\n\nRunning on: localhost:8080",
		updater.CurrentVersion)
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

// getIcon returns a simple icon for the system tray
// This is a simple 16x16 ICO format icon (database symbol)
func getIcon() []byte {
	// Simple ICO file format: 16x16 monochrome icon
	return []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x00, 0x00, 0x01, 0x00,
		0x20, 0x00, 0x68, 0x04, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00,
		// Pixel data (16x16 RGBA)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		// Blue database icon pattern
		0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF,
		0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF,
		0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF,
		0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF, 0xFF, 0x80, 0x00, 0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
}
