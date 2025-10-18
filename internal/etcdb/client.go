package etcdb

import (
	"context"
	"fmt"

	pb "github.com/yhonda-ohishi/db_service/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ETCDBClient implements DBClient interface for etc_data_processor
type ETCDBClient struct {
	conn    *grpc.ClientConn
	client  pb.ETCMeisaiServiceClient
}

// NewETCDBClient creates a new ETC database client
func NewETCDBClient(address string) (*ETCDBClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db_service: %w", err)
	}

	return &ETCDBClient{
		conn:   conn,
		client: pb.NewETCMeisaiServiceClient(conn),
	}, nil
}

// SaveETCData saves ETC data to database via db_service
func (c *ETCDBClient) SaveETCData(data interface{}) error {
	// Convert data to map format (from etc_data_processor)
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data type, expected map[string]interface{}")
	}

	ctx := context.Background()

	// Extract fields from map
	date, _ := dataMap["date"].(string)
	entryIC, _ := dataMap["entry_ic"].(string)
	exitIC, _ := dataMap["exit_ic"].(string)
	route, _ := dataMap["route"].(string)
	vehicleType, _ := dataMap["vehicle_type"].(string)
	amount, _ := dataMap["amount"].(int)
	cardNumber, _ := dataMap["card_number"].(string)

	// Map to ETCMeisai fields
	// date_to_date: 利用日
	// ic_fr: 入口IC
	// ic_to: 出口IC
	// detail: ルート情報
	// shashu: 車種
	// price: 料金
	// etc_num: カード番号

	req := &pb.CreateETCMeisaiRequest{
		EtcMeisai: &pb.ETCMeisai{
			DateToDate: date,
			IcFr:       entryIC,
			IcTo:       exitIC,
			Price:      int32(amount),
			Shashu:     parseVehicleType(vehicleType),
			EtcNum:     cardNumber,
			Detail:     &route,
		},
	}

	_, err := c.client.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to save ETC record: %w", err)
	}

	return nil
}

// parseVehicleType converts vehicle type string to int32
func parseVehicleType(vtype string) int32 {
	// Simple mapping - adjust based on actual vehicle types
	switch vtype {
	case "普通車":
		return 1
	case "大型車":
		return 2
	case "中型車":
		return 3
	default:
		return 1 // Default to 普通車
	}
}

// Close closes the connection
func (c *ETCDBClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
