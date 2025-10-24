package server

import (
	"fmt"
	"log"
	"net"

	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/etcscraper"

	"github.com/yhonda-ohishi/db_service/src/registry"
	dtakoeventsregistry "github.com/yhonda-ohishi/dtako_events/pkg/registry"
	dtakorowsregistry "github.com/yhonda-ohishi/dtako_rows/v3/pkg/registry"
	downloadpb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	pb "github.com/yhonda-ohishi-pub-dev/desktop-server/proto"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	grpcServer *grpc.Server
}

func NewGRPCServer(scraperManager *etcscraper.Manager, progressService *ProgressService) *GRPCServer {
	// Initialize gRPC server immediately
	grpcSrv := grpc.NewServer()

	srv := &GRPCServer{
		grpcServer: grpcSrv,
	}

	// Register all db_service services automatically (excluding DTakoRowsService and DTakoEventsService)
	registry.Register(grpcSrv, registry.WithExcludeServices("DTakoRowsService", "DTakoEventsService"))

	// Register dtako_rows services automatically
	if err := dtakorowsregistry.Register(grpcSrv); err != nil {
		log.Printf("Warning: Failed to register dtako_rows services: %v", err)
	}

	// Register dtako_events services automatically
	if err := dtakoeventsregistry.Register(grpcSrv); err != nil {
		log.Printf("Warning: Failed to register dtako_events services: %v", err)
	}

	// Register ProgressService for gRPC streaming
	pb.RegisterProgressServiceServer(grpcSrv, progressService)

	// Register DownloadService proxy
	downloadProxy := NewDownloadServiceProxy(scraperManager, progressService)
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
