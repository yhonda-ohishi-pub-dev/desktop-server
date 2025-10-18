package etcscraper

import (
	"context"
	"fmt"
	"time"

	pb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps gRPC client for etc_meisai_scraper
type Client struct {
	conn            *grpc.ClientConn
	downloadService pb.DownloadServiceClient
}

// NewClient creates a new client for etc_meisai_scraper
func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to etc_meisai_scraper gRPC server
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etc_meisai_scraper: %w", err)
	}

	return &Client{
		conn:            conn,
		downloadService: pb.NewDownloadServiceClient(conn),
	}, nil
}

// GetDownloadService returns the Download service client
func (c *Client) GetDownloadService() pb.DownloadServiceClient {
	return c.downloadService
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// DownloadSync starts a sync download
func (c *Client) DownloadSync(ctx context.Context, accounts []string, fromDate, toDate string) (*pb.DownloadResponse, error) {
	req := &pb.DownloadRequest{
		Accounts: accounts,
		FromDate: fromDate,
		ToDate:   toDate,
	}

	return c.downloadService.DownloadSync(ctx, req)
}

// DownloadAsync starts an async download job
func (c *Client) DownloadAsync(ctx context.Context, accounts []string, fromDate, toDate string) (string, error) {
	req := &pb.DownloadRequest{
		Accounts: accounts,
		FromDate: fromDate,
		ToDate:   toDate,
	}

	resp, err := c.downloadService.DownloadAsync(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.JobId, nil
}

// GetJobStatus retrieves job status
func (c *Client) GetJobStatus(ctx context.Context, jobID string) (*pb.JobStatus, error) {
	req := &pb.GetJobStatusRequest{
		JobId: jobID,
	}

	return c.downloadService.GetJobStatus(ctx, req)
}

// GetAllAccountIDs retrieves all configured account IDs
func (c *Client) GetAllAccountIDs(ctx context.Context) ([]string, error) {
	req := &pb.GetAllAccountIDsRequest{}

	resp, err := c.downloadService.GetAllAccountIDs(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.AccountIds, nil
}
