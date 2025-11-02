package server

import (
	"fmt"
	"log"
	"net"

	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/etcscraper"
	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/reflector"

	"github.com/yhonda-ohishi/db_service/src/registry"
	dtakoeventsregistry "github.com/yhonda-ohishi/dtako_events/pkg/registry"
	dtakorowsregistry "github.com/yhonda-ohishi/dtako_rows/v3/pkg/registry"
	downloadpb "github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/pb"
	pb "github.com/yhonda-ohishi-pub-dev/desktop-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	grpcServer *grpc.Server
}

func NewGRPCServer(scraperManager *etcscraper.Manager, progressService *ProgressService) *GRPCServer {
	grpcSrv := grpc.NewServer()

	// Register db_service (excluding DTakoEventsService and DTakoRowsService)
	dbRegistry := registry.Register(grpcSrv, registry.WithExcludeServices("DTakoEventsService", "DTakoRowsService"))

	// Register dtako_rows services (integrated mode with db_service)
	if dbRegistry != nil && dbRegistry.DTakoRowsService != nil {
		if err := dtakorowsregistry.Register(grpcSrv, dbRegistry.DTakoRowsService); err != nil {
			log.Printf("Warning: Failed to register dtako_rows: %v", err)
		}
	}

	// Register dtako_events services (integrated mode with db_service)
	if dbRegistry != nil && dbRegistry.DTakoEventsService != nil {
		if err := dtakoeventsregistry.Register(grpcSrv, dbRegistry.DTakoEventsService); err != nil {
			log.Printf("Warning: Failed to register dtako_events: %v", err)
		}
	}

	// Register ProgressService for gRPC streaming
	pb.RegisterProgressServiceServer(grpcSrv, progressService)

	// Register DownloadService proxy
	downloadProxy := NewDownloadServiceProxy(scraperManager, progressService)
	downloadpb.RegisterDownloadServiceServer(grpcSrv, downloadProxy)

	// Register reflection service for grpcurl and other tools
	reflection.Register(grpcSrv)

	// Log registered services and methods
	if services, err := reflector.GetServices(grpcSrv); err == nil {
		log.Println("Registered gRPC services:")
		log.Print(reflector.FormatServices(services))
	} else {
		log.Printf("Warning: Failed to get service info: %v", err)
	}

	return &GRPCServer{grpcServer: grpcSrv}
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
