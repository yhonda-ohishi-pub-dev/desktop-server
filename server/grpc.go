package server

import (
	"context"
	"fmt"
	"net"

	pb "github.com/yhonda-ohishi-pub-dev/desktop-server/proto"
	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/etcscraper"

	"github.com/yhonda-ohishi/db_service/src/registry"
	downloadpb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	pb.UnimplementedDatabaseServiceServer
	db         *DatabaseConnection
	grpcServer *grpc.Server
}

func NewGRPCServer(db *DatabaseConnection, scraperManager *etcscraper.Manager) *GRPCServer {
	// Initialize gRPC server immediately
	grpcSrv := grpc.NewServer()

	srv := &GRPCServer{
		db:         db,
		grpcServer: grpcSrv,
	}

	// Register desktop-server's own database service
	pb.RegisterDatabaseServiceServer(grpcSrv, srv)

	// Register all db_service services automatically
	registry.Register(grpcSrv)

	// Register DownloadService proxy
	downloadProxy := NewDownloadServiceProxy(scraperManager)
	downloadpb.RegisterDownloadServiceServer(grpcSrv, downloadProxy)

	return srv
}

func (s *GRPCServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	fmt.Printf("gRPC server listening on %s\n", addr)
	return s.grpcServer.Serve(lis)
}

func (s *GRPCServer) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

func (s *GRPCServer) QueryDatabase(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Convert params to interface slice
	args := make([]interface{}, len(req.Params))
	for i, p := range req.Params {
		args[i] = p
	}

	rows, err := s.db.Query(req.Sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var result []*pb.Row
	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		rowData := make(map[string]string)
		for i, col := range columns {
			val := values[i]
			if val != nil {
				rowData[col] = fmt.Sprintf("%v", val)
			} else {
				rowData[col] = ""
			}
		}

		result = append(result, &pb.Row{Columns: rowData})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &pb.QueryResponse{
		Rows:  result,
		Count: int32(len(result)),
	}, nil
}

func (s *GRPCServer) StreamQuery(req *pb.StreamQueryRequest, stream pb.DatabaseService_StreamQueryServer) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	rows, err := s.db.Query(req.Sql)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		rowData := make(map[string]string)
		for i, col := range columns {
			val := values[i]
			if val != nil {
				rowData[col] = fmt.Sprintf("%v", val)
			} else {
				rowData[col] = ""
			}
		}

		if err := stream.Send(&pb.QueryRow{Columns: rowData}); err != nil {
			return fmt.Errorf("failed to send row: %w", err)
		}
	}

	return rows.Err()
}

func (s *GRPCServer) GetTables(ctx context.Context, req *pb.GetTablesRequest) (*pb.GetTablesResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tables, err := s.db.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	return &pb.GetTablesResponse{Tables: tables}, nil
}

func (s *GRPCServer) ExecuteSQL(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	args := make([]interface{}, len(req.Params))
	for i, p := range req.Params {
		args[i] = p
	}

	result, err := s.db.Exec(req.Sql, args...)
	if err != nil {
		return nil, fmt.Errorf("execute failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return &pb.ExecuteResponse{AffectedRows: int32(rowsAffected)}, nil
}
