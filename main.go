package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"desktop-server/frontend"
	"desktop-server/internal/etcscraper"
	"desktop-server/internal/process"
	"desktop-server/server"
	"desktop-server/systray"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Debug: Log ETC environment variables
	log.Printf("ETC_HEADLESS=%s", os.Getenv("ETC_HEADLESS"))
	log.Printf("ETC_CORP_ACCOUNTS=%s", os.Getenv("ETC_CORP_ACCOUNTS"))

	// Parse command line flags
	updateFrontend := flag.Bool("update", false, "Force download latest frontend")
	flag.Parse()

	// Kill any existing instances of this application
	if err := process.KillExistingProcesses(); err != nil {
		log.Printf("Warning: Failed to kill existing processes: %v", err)
	}

	// Download frontend if missing or update requested
	if err := frontend.DownloadLatestRelease(*updateFrontend); err != nil {
		log.Printf("Warning: Failed to download frontend: %v", err)
		if *updateFrontend {
			log.Fatal("Update requested but failed")
		}
	}

	// If only updating, exit after download
	if *updateFrontend {
		fmt.Println("Frontend updated successfully")
		return
	}

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

	// Initialize etc_meisai_scraper manager (auto-start enabled)
	scraperManager := etcscraper.NewManager("localhost:50052", "", true)
	defer scraperManager.Stop()
	systray.SetScraperManager(scraperManager)

	// Start gRPC server with DownloadService proxy
	grpcServer := server.NewGRPCServer(db, scraperManager)
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
