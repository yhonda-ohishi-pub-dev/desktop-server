package etcdb

import (
	"context"
	"fmt"
	"time"

	pb "github.com/yhonda-ohishi/db_service/src/proto"
	"github.com/yhonda-ohishi/etc_data_processor/src/pkg/parser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ETCDBClient wraps db_service ETCMeisai operations
type ETCDBClient struct {
	conn    *grpc.ClientConn
	client  pb.Db_ETCMeisaiServiceClient
}

// NewETCDBClient creates a new ETC database client
func NewETCDBClient(address string) (*ETCDBClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db_service: %w", err)
	}

	return &ETCDBClient{
		conn:   conn,
		client: pb.NewDb_ETCMeisaiServiceClient(conn),
	}, nil
}

// SaveETCRecord saves a single ETC record to database
func (c *ETCDBClient) SaveETCRecord(ctx context.Context, record parser.ActualETCRecord) error {
	// Parse exit date - fallback to entry date if exit date is empty
	var exitDate time.Time
	var err error

	if record.ExitDate != "" {
		exitDate, err = parseDate(record.ExitDate)
		if err != nil {
			return fmt.Errorf("invalid exit date: %w", err)
		}
	} else if record.EntryDate != "" {
		// Fallback: use entry date if exit date is missing
		exitDate, err = parseDate(record.EntryDate)
		if err != nil {
			return fmt.Errorf("invalid entry date (used as fallback): %w", err)
		}
	} else {
		return fmt.Errorf("both exit date and entry date are empty")
	}

	// Format dates for db_service
	// date_to: RFC3339形式推奨 (例: 2025-10-18T15:30:00Z)
	exitDateTime := fmt.Sprintf("%sT%s:00Z", exitDate.Format("2006-01-02"), record.ExitTime)
	// date_to_date: YYYY-MM-DD形式 (例: 2025-10-18)
	exitDateOnly := exitDate.Format("2006-01-02")

	// date_fr (入口日時) は optional - *string型
	var dateFr *string
	if record.EntryDate != "" && record.EntryTime != "" {
		entryDate, err := parseDate(record.EntryDate)
		if err != nil {
			return fmt.Errorf("invalid entry date: %w", err)
		}
		entryDateTime := fmt.Sprintf("%sT%s:00Z", entryDate.Format("2006-01-02"), record.EntryTime)
		dateFr = &entryDateTime
	}

	// ic_fr (入口IC) は optional - *string型
	var icFr *string
	if record.EntryIC != "" {
		icFr = &record.EntryIC
	}

	req := &pb.Db_CreateETCMeisaiRequest{
		EtcMeisai: &pb.Db_ETCMeisai{
			DateFr:     dateFr,         // optional: 入口日時 (*string型)
			DateTo:     exitDateTime,   // 必須: 出口日時
			DateToDate: exitDateOnly,   // 必須: 出口日付のみ
			IcFr:       icFr,           // optional: 入口IC (*string型、実データの22.3%が空)
			IcTo:       record.ExitIC,  // 必須: 出口IC
			Price:      int32(record.ETCAmount), // 必須: ETC料金
			Shashu:     int32(record.VehicleClass), // 必須: 車種
			EtcNum:     record.CardNumber, // 必須: ETCカード番号
			Hash:       "", // 空文字列 (db_service側で自動生成)
		},
	}

	_, err = c.client.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to save ETC record: %w", err)
	}

	return nil
}

// parseDate parses various date formats
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"06/01/02",
		"2006/01/02",
		"2006-01-02",
		"06-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// Close closes the connection
func (c *ETCDBClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
