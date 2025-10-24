package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/yhonda-ohishi-pub-dev/desktop-server/frontend"
	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/etcscraper"
	"github.com/yhonda-ohishi-pub-dev/desktop-server/internal/process"
	"github.com/yhonda-ohishi-pub-dev/desktop-server/server"
	"github.com/yhonda-ohishi-pub-dev/desktop-server/systray"

	"github.com/joho/godotenv"
)

func setupLogging() (*os.File, error) {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(exeDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	logFileName := fmt.Sprintf("desktop-server_%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(logsDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Set log output to file only (os.Stdout doesn't work with -H windowsgui)
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return logFile, nil
}

func main() {
	// Setup file logging
	logFile, err := setupLogging()
	if err != nil {
		log.Printf("Warning: Failed to setup file logging: %v", err)
		// Continue without file logging
	} else {
		defer logFile.Close()
		log.Println("Logging initialized successfully")
	}

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

	// Initialize etc_meisai_scraper manager (auto-start enabled)
	scraperManager := etcscraper.NewManager("localhost:50052", "", true)
	defer scraperManager.Stop()
	systray.SetScraperManager(scraperManager)

	// Start etc_meisai_scraper immediately on startup
	if err := scraperManager.Start(); err != nil {
		log.Printf("Warning: Failed to start etc_meisai_scraper: %v", err)
		log.Println("ETC download functionality will not be available until etc_meisai_scraper.exe is available")
	}

	// Initialize progress service for gRPC streaming
	progressService := server.NewProgressService()

	// Start gRPC server with ProgressService and DownloadService proxy
	grpcServer := server.NewGRPCServer(scraperManager, progressService)
	go func() {
		if err := grpcServer.Start(":50051"); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Start HTTP + gRPC-Web proxy server
	httpServer := server.NewHTTPServer(grpcServer, progressService)
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
