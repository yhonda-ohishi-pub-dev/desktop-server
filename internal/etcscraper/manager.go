package etcscraper

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager manages etc_meisai_scraper process lifecycle
type Manager struct {
	address    string
	binaryPath string
	process    *exec.Cmd
	client     *Client
	autoStart  bool
}

// NewManager creates a new etc_meisai_scraper manager
func NewManager(address, binaryPath string, autoStart bool) *Manager {
	return &Manager{
		address:    address,
		binaryPath: binaryPath,
		autoStart:  autoStart,
	}
}

// Start starts etc_meisai_scraper process if not running
func (m *Manager) Start() error {
	// Check if process is already running
	if m.process != nil && m.process.ProcessState == nil {
		log.Println("etc_meisai_scraper is already running")
		return nil
	}

	// Find binary path
	if m.binaryPath == "" {
		// Look for binary in same directory as desktop-server
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		dir := filepath.Dir(exePath)
		m.binaryPath = filepath.Join(dir, "etc_meisai_scraper.exe")
	}

	// Check if binary exists
	if _, err := os.Stat(m.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("etc_meisai_scraper.exe not found at %s", m.binaryPath)
	}

	// Start process
	log.Printf("Starting etc_meisai_scraper at %s", m.binaryPath)
	m.process = exec.Command(m.binaryPath, "--grpc-port", "50052")
	m.process.Stdout = os.Stdout
	m.process.Stderr = os.Stderr

	if err := m.process.Start(); err != nil {
		return fmt.Errorf("failed to start etc_meisai_scraper: %w", err)
	}

	// Wait for service to be ready
	if err := m.waitForReady(10 * time.Second); err != nil {
		m.Stop()
		return fmt.Errorf("etc_meisai_scraper failed to start: %w", err)
	}

	log.Printf("etc_meisai_scraper started successfully (PID: %d)", m.process.Process.Pid)
	return nil
}

// Stop stops etc_meisai_scraper process
func (m *Manager) Stop() error {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	if m.process != nil && m.process.ProcessState == nil {
		log.Printf("Stopping etc_meisai_scraper (PID: %d)", m.process.Process.Pid)
		if err := m.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop etc_meisai_scraper: %w", err)
		}
		m.process.Wait()
		m.process = nil
	}

	return nil
}

// GetClient returns a gRPC client, starting the process if needed
func (m *Manager) GetClient() (*Client, error) {
	// If client exists and connection is alive, return it
	if m.client != nil {
		return m.client, nil
	}

	// Auto-start if enabled
	if m.autoStart {
		if err := m.Start(); err != nil {
			return nil, fmt.Errorf("failed to auto-start etc_meisai_scraper: %w", err)
		}
	}

	// Create client
	client, err := NewClient(m.address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etc_meisai_scraper: %w", err)
	}

	m.client = client
	return client, nil
}

// waitForReady waits for etc_meisai_scraper to be ready
func (m *Manager) waitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for etc_meisai_scraper to be ready")
		case <-ticker.C:
			conn, err := grpc.DialContext(ctx, m.address,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

// IsRunning checks if etc_meisai_scraper is running
func (m *Manager) IsRunning() bool {
	return m.process != nil && m.process.ProcessState == nil
}
