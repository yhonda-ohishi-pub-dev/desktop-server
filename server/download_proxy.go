package server

import (
	"context"
	"fmt"
	"log"

	"desktop-server/internal/etcscraper"

	downloadpb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
)

// DownloadServiceProxy proxies requests to etc_meisai_scraper's DownloadService
type DownloadServiceProxy struct {
	downloadpb.UnimplementedDownloadServiceServer
	scraperManager *etcscraper.Manager
}

func NewDownloadServiceProxy(scraperManager *etcscraper.Manager) *DownloadServiceProxy {
	return &DownloadServiceProxy{
		scraperManager: scraperManager,
	}
}

func (p *DownloadServiceProxy) DownloadSync(ctx context.Context, req *downloadpb.DownloadRequest) (*downloadpb.DownloadResponse, error) {
	log.Printf("DownloadSync called with accounts: %v, from: %s, to: %s, mode: %s", req.Accounts, req.FromDate, req.ToDate, req.Mode)

	// If accounts is empty, get from environment variable
	if len(req.Accounts) == 0 {
		accounts := getAccountsFromEnv()
		if len(accounts) == 0 {
			return &downloadpb.DownloadResponse{
				Success: false,
				Error:   "no accounts specified and ETC_CORP_ACCOUNTS environment variable not set",
			}, nil
		}
		req.Accounts = accounts
		log.Printf("Using accounts from environment variable: %v", accounts)
	}

	client, err := p.scraperManager.GetClient()
	if err != nil {
		return &downloadpb.DownloadResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get scraper client: %v", err),
		}, nil
	}

	// Use DownloadAsync internally (DownloadSync is not implemented in etc_meisai_scraper)
	log.Printf("Starting async download job...")
	jobResp, err := client.GetDownloadService().DownloadAsync(ctx, req)
	if err != nil {
		return &downloadpb.DownloadResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to start download: %v", err),
		}, err
	}

	if jobResp.Status == "error" || jobResp.Status == "failed" {
		return &downloadpb.DownloadResponse{
			Success: false,
			Error:   jobResp.Message,
		}, nil
	}

	log.Printf("Download job started with ID: %s", jobResp.JobId)

	// Run download and CSV processing in background
	go func() {
		// Poll for job completion
		totalRecords, err := p.waitForJobCompletion(context.Background(), client, jobResp.JobId)
		if err != nil {
			log.Printf("Download job failed: %v", err)
			return
		}

		log.Printf("Download completed successfully, total records: %d", totalRecords)

		// If mode is "db", process CSV and save to database
		if req.Mode == "db" {
			log.Printf("Processing downloaded CSV files and saving to database...")
			saved, errors := p.processDownloadedCSVFiles(req.Accounts)
			log.Printf("Database save completed: %d saved, %d errors", saved, errors)
		}
	}()

	// Return immediately with job ID
	return &downloadpb.DownloadResponse{
		Success: true,
		// Job will continue in background
	}, nil
}

func (p *DownloadServiceProxy) DownloadAsync(ctx context.Context, req *downloadpb.DownloadRequest) (*downloadpb.DownloadJobResponse, error) {
	log.Printf("DownloadAsync called with accounts: %v, from: %s, to: %s", req.Accounts, req.FromDate, req.ToDate)

	// If accounts is empty, get from environment variable
	if len(req.Accounts) == 0 {
		accounts := getAccountsFromEnv()
		if len(accounts) == 0 {
			return &downloadpb.DownloadJobResponse{
				Status:  "error",
				Message: "no accounts specified and ETC_CORP_ACCOUNTS environment variable not set",
			}, nil
		}
		req.Accounts = accounts
		log.Printf("Using accounts from environment variable: %v", accounts)
	}

	client, err := p.scraperManager.GetClient()
	if err != nil {
		return &downloadpb.DownloadJobResponse{
			Status:  "error",
			Message: fmt.Sprintf("failed to get scraper client: %v", err),
		}, nil
	}

	// Use the gRPC client directly
	return client.GetDownloadService().DownloadAsync(ctx, req)
}

func (p *DownloadServiceProxy) GetJobStatus(ctx context.Context, req *downloadpb.GetJobStatusRequest) (*downloadpb.JobStatus, error) {
	client, err := p.scraperManager.GetClient()
	if err != nil {
		return &downloadpb.JobStatus{
			JobId:        req.JobId,
			Status:       "error",
			ErrorMessage: fmt.Sprintf("failed to get scraper client: %v", err),
		}, nil
	}

	// Use the gRPC client directly
	return client.GetDownloadService().GetJobStatus(ctx, req)
}

func (p *DownloadServiceProxy) GetAllAccountIDs(ctx context.Context, req *downloadpb.GetAllAccountIDsRequest) (*downloadpb.GetAllAccountIDsResponse, error) {
	client, err := p.scraperManager.GetClient()
	if err != nil {
		return &downloadpb.GetAllAccountIDsResponse{
			AccountIds: []string{},
		}, fmt.Errorf("failed to get scraper client: %v", err)
	}

	// Use the gRPC client directly
	return client.GetDownloadService().GetAllAccountIDs(ctx, req)
}
