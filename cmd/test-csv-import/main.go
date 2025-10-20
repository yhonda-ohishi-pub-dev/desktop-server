package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/etcdb"

	"github.com/joho/godotenv"
	"github.com/yhonda-ohishi/etc_data_processor/src/pkg/parser"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Get CSV file path from command line
	if len(os.Args) < 2 {
		log.Fatal("Usage: test-csv-import <csv_file_path>")
	}

	csvFilePath := os.Args[1]
	accountID := "testaccount"

	if len(os.Args) >= 3 {
		accountID = os.Args[2]
	}

	// Check if file exists
	if _, err := os.Stat(csvFilePath); os.IsNotExist(err) {
		log.Fatalf("CSV file not found: %s", csvFilePath)
	}

	log.Printf("Processing CSV file: %s", csvFilePath)
	log.Printf("Account ID: %s", accountID)

	// Parse CSV file
	log.Println("Parsing CSV file...")
	csvParser := parser.NewETCCSVParser()
	records, err := csvParser.ParseFile(csvFilePath)
	if err != nil {
		log.Fatalf("Failed to parse CSV: %v", err)
	}

	log.Printf("Parsed %d records from CSV", len(records))

	// Create DB client (connect to desktop-server's gRPC, not MySQL)
	dbAddress := "localhost:50051"
	log.Printf("Connecting to db_service at: %s", dbAddress)

	dbClient, err := etcdb.NewETCDBClient(dbAddress)
	if err != nil {
		log.Fatalf("Failed to create DB client: %v", err)
	}
	defer dbClient.Close()

	// Save records to database
	ctx := context.Background()
	saved := 0
	errors := 0

	log.Println("Saving records to database...")
	for i, record := range records {
		if err := dbClient.SaveETCRecord(ctx, record); err != nil {
			log.Printf("ERROR Record %d: %v", i+1, err)
			errors++
		} else {
			saved++
		}

		// Progress update every 100 records
		if (i+1)%100 == 0 {
			log.Printf("Progress: %d/%d records processed", i+1, len(records))
		}
	}

	// Print results
	fmt.Println("\n=== Processing Results ===")
	fmt.Printf("Total Records:   %d\n", len(records))
	fmt.Printf("Saved Records:   %d\n", saved)
	fmt.Printf("Error Records:   %d\n", errors)

	if saved > 0 {
		fmt.Println("\n✅ CSV processing completed successfully!")
	} else {
		fmt.Println("\n❌ CSV processing failed")
		os.Exit(1)
	}
}
