package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"desktop-server/internal/etcscraper"
	"desktop-server/systray"

	downloadpb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
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

// getAccountsFromEnv retrieves account information from ETC_CORP_ACCOUNTS environment variable
// Expected format: ETC_CORP_ACCOUNTS=["userid1:password1","userid2:password2"]
func getAccountsFromEnv() []string {
	accountsEnv := os.Getenv("ETC_CORP_ACCOUNTS")
	if accountsEnv == "" {
		return []string{}
	}

	// Parse JSON array format
	accountsEnv = strings.TrimSpace(accountsEnv)
	accountsEnv = strings.Trim(accountsEnv, "[]")
	accountsEnv = strings.ReplaceAll(accountsEnv, "\"", "")

	if accountsEnv == "" {
		return []string{}
	}

	accounts := strings.Split(accountsEnv, ",")

	// Trim whitespace from each account
	for i := range accounts {
		accounts[i] = strings.TrimSpace(accounts[i])
	}

	log.Printf("Loaded %d account(s) from ETC_CORP_ACCOUNTS", len(accounts))
	return accounts
}

// waitForJobCompletion polls job status until completion
func (p *DownloadServiceProxy) waitForJobCompletion(ctx context.Context, client *etcscraper.Client, jobID string) (int32, error) {
	log.Printf("Waiting for job %s to complete...", jobID)

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			status, err := client.GetDownloadService().GetJobStatus(ctx, &downloadpb.GetJobStatusRequest{
				JobId: jobID,
			})
			if err != nil {
				return 0, fmt.Errorf("failed to get job status: %w", err)
			}

			log.Printf("Job %s status: %s, progress: %d/%d", jobID, status.Status, status.Progress, status.TotalRecords)

			switch status.Status {
			case "completed", "success":
				return status.TotalRecords, nil
			case "failed", "error":
				return 0, fmt.Errorf("job failed: %s", status.ErrorMessage)
			}

			// Wait 2 seconds before polling again
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(2 * time.Second):
				// Continue polling
			}
		}
	}
}

// processDownloadedCSVFiles processes all CSV files in the downloads folder
func (p *DownloadServiceProxy) processDownloadedCSVFiles(accounts []string) (saved int, errors int) {
	log.Printf("Processing downloaded CSV files in downloads folder")

	// Use systray's processDownloadedCSVFiles logic
	// Find the most recent download folder and process CSV files
	downloadDir := "downloads"
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		log.Printf("Failed to read downloads directory: %v", err)
		return 0, 1
	}

	// Find the most recent folder (folders are named with timestamp)
	var latestFolder string
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.IsDir() {
			latestFolder = filepath.Join(downloadDir, entry.Name())
			break
		}
	}

	if latestFolder == "" {
		log.Printf("No download folders found")
		return 0, 1
	}

	log.Printf("Processing CSV files in folder: %s", latestFolder)

	// Find CSV files in the folder
	csvFiles, err := filepath.Glob(filepath.Join(latestFolder, "*.csv"))
	if err != nil {
		log.Printf("Failed to find CSV files: %v", err)
		return 0, 1
	}

	log.Printf("Found %d CSV files", len(csvFiles))

	totalSaved := 0
	totalErrors := 0

	for _, csvFile := range csvFiles {
		// Extract account ID from filename (format: accountname_timestamp.csv)
		filename := filepath.Base(csvFile)
		accountID := strings.Split(filename, "_")[0]

		log.Printf("Processing CSV file: %s for account: %s", csvFile, accountID)

		saved, errors, err := systray.ProcessCSVFile(csvFile, accountID)
		if err != nil {
			log.Printf("Failed to process CSV file %s: %v", csvFile, err)
			totalErrors++
			continue
		}

		totalSaved += saved
		totalErrors += errors
	}

	return totalSaved, totalErrors
}
