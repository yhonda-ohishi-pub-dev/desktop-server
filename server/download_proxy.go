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
	log.Printf("DownloadSync called with accounts: %v, from: %s, to: %s", req.Accounts, req.FromDate, req.ToDate)

	client, err := p.scraperManager.GetClient()
	if err != nil {
		return &downloadpb.DownloadResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get scraper client: %v", err),
		}, nil
	}

	// Use the gRPC client directly
	return client.GetDownloadService().DownloadSync(ctx, req)
}

func (p *DownloadServiceProxy) DownloadAsync(ctx context.Context, req *downloadpb.DownloadRequest) (*downloadpb.DownloadJobResponse, error) {
	log.Printf("DownloadAsync called with accounts: %v, from: %s, to: %s", req.Accounts, req.FromDate, req.ToDate)

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
