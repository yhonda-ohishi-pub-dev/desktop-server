package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"desktop-server/server"
	"desktop-server/systray"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize database connection (optional)
	db, err := server.NewDatabaseConnection()
	if err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Starting without database connection. Configure DB settings to use database features.")
		db = nil
	} else {
		log.Println("Database connected successfully")
	}
	if db != nil {
		defer db.Close()
	}

	// Start gRPC server
	grpcServer := server.NewGRPCServer(db)
	go func() {
		if err := grpcServer.Start(":50051"); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Start HTTP + gRPC-Web proxy server
	httpServer := server.NewHTTPServer(grpcServer)
	go func() {
		if err := httpServer.Start(":8080"); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	fmt.Println("Server started on:")
	fmt.Println("  - gRPC: localhost:50051")
	fmt.Println("  - HTTP: http://localhost:8080")

	// Start systray
	go systray.Run(ctx, func() {
		cancel()
	})

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down...")

	grpcServer.Stop()
	httpServer.Stop()
}
