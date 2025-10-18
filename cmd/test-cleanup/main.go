package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Get current directory (should be desktop-server root)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	downloadsDir := filepath.Join(dir, "downloads")

	// Count folders before
	entries, _ := os.ReadDir(downloadsDir)
	fmt.Printf("Folders in %s:\n", downloadsDir)

	var dirs []struct {
		name    string
		modTime time.Time
	}

	for _, entry := range entries {
		if entry.IsDir() {
			info, _ := entry.Info()
			dirs = append(dirs, struct {
				name    string
				modTime time.Time
			}{entry.Name(), info.ModTime()})
		}
	}

	fmt.Printf("\nTotal: %d folders\n", len(dirs))

	// Sort by time
	for i := 0; i < len(dirs); i++ {
		for j := i + 1; j < len(dirs); j++ {
			if dirs[i].modTime.After(dirs[j].modTime) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}

	fmt.Println("\nFolders sorted by modification time (oldest first):")
	for i, d := range dirs {
		fmt.Printf("%2d. %s (modified: %s)\n", i+1, d.name, d.modTime.Format("2006-01-02 15:04:05"))
	}

	// Simulate cleanup
	keepCount := 10
	if len(dirs) > keepCount {
		deleteCount := len(dirs) - keepCount
		fmt.Printf("\nWould delete %d oldest folders:\n", deleteCount)
		for i := 0; i < deleteCount; i++ {
			fmt.Printf("  - %s\n", dirs[i].name)
		}

		// Actually delete them
		fmt.Print("\nDelete them? (y/n): ")
		var response string
		fmt.Scanln(&response)

		if response == "y" || response == "Y" {
			for i := 0; i < deleteCount; i++ {
				dirPath := filepath.Join(downloadsDir, dirs[i].name)
				fmt.Printf("Deleting: %s\n", dirs[i].name)
				if err := os.RemoveAll(dirPath); err != nil {
					fmt.Printf("Error deleting %s: %v\n", dirs[i].name, err)
				}
			}
			fmt.Println("Cleanup complete!")
		}
	} else {
		fmt.Printf("\nNo cleanup needed (have %d, limit is %d)\n", len(dirs), keepCount)
	}
}
