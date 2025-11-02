package csvprocessor

import (
	"context"
	"fmt"
	"log"

	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/etcdb"
	"github.com/yhonda-ohishi/etc_data_processor/src/pkg/parser"
)

// ProcessCSVFile parses and saves ETC CSV records to database
func ProcessCSVFile(filePath, accountID string) (int, int, error) {
	// Create parser
	csvParser := parser.NewETCCSVParser()

	// Parse CSV file
	records, err := csvParser.ParseFile(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Create DB client
	dbClient, err := etcdb.NewETCDBClient("localhost:50051")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create DB client: %w", err)
	}
	defer dbClient.Close()

	// Save each record to database
	ctx := context.Background()
	saved := 0
	errors := 0

	for _, record := range records {
		if err := dbClient.SaveETCRecord(ctx, record); err != nil {
			log.Printf("Failed to save record: %v", err)
			errors++
		} else {
			saved++
		}
	}

	return saved, errors, nil
}
