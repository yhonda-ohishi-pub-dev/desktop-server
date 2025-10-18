package server

import (
	"log"
	"os"
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
