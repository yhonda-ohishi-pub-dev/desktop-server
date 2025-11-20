package server

import (
	"fmt"
	"log"
	"net"

	"github.com/yhonda-ohishi/db_service/src/registry"
	dtakoeventsregistry "github.com/yhonda-ohishi/dtako_events/pkg/registry"
	dtakorowsregistry "github.com/yhonda-ohishi/dtako_rows/v3/pkg/registry"
	pb "github.com/yhonda-ohishi-pub-dev/desktop-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	grpcServer *grpc.Server
}

func NewGRPCServer(progressService *ProgressService) *GRPCServer {
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

	// Register reflection service for grpcurl and other tools
	reflection.Register(grpcSrv)

	// Log registered services (using simple ServiceInfo from gRPC server)
	serviceInfo := grpcSrv.GetServiceInfo()
	log.Println("Registered gRPC services:")
	for serviceName := range serviceInfo {
		log.Printf("  - %s", serviceName)
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
