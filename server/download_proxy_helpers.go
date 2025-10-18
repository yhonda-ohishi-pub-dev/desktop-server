package server

import (
	"log"
	"strings"

	"desktop-server/systray"
)

// processAndSaveCSV processes a CSV file and saves records to database
func (p *DownloadServiceProxy) processAndSaveCSV(csvPath string, accounts []string) (saved int, errors int) {
	// Extract account ID from the first account (format: "accountid:password")
	accountID := ""
	if len(accounts) > 0 {
		parts := strings.Split(accounts[0], ":")
		if len(parts) > 0 {
			accountID = parts[0]
		}
	}

	log.Printf("Processing CSV file: %s for account: %s", csvPath, accountID)

	// Use systray's ProcessCSVFile function
	saved, errors, err := systray.ProcessCSVFile(csvPath, accountID)
	if err != nil {
		log.Printf("Failed to process CSV: %v", err)
		return 0, 1
	}

	log.Printf("CSV processing completed: %d saved, %d errors", saved, errors)
	return saved, errors
}
