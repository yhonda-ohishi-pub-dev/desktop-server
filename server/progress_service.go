package server

import (
	"log"
	"sync"
	"time"

	pb "github.com/yhonda-ohishi-pub-dev/desktop-server/proto"
)

// ProgressService implements the ProgressService gRPC service
type ProgressService struct {
	pb.UnimplementedProgressServiceServer
	mu      sync.RWMutex
	streams []pb.ProgressService_StreamDownloadProgressServer
}

// NewProgressService creates a new ProgressService
func NewProgressService() *ProgressService {
	return &ProgressService{
		streams: make([]pb.ProgressService_StreamDownloadProgressServer, 0),
	}
}

// StreamDownloadProgress streams download progress updates to clients
func (s *ProgressService) StreamDownloadProgress(req *pb.StreamProgressRequest, stream pb.ProgressService_StreamDownloadProgressServer) error {
	log.Printf("New progress stream client connected (job_id filter: %s)", req.JobId)

	// Add stream to list
	s.mu.Lock()
	s.streams = append(s.streams, stream)
	streamIndex := len(s.streams) - 1
	s.mu.Unlock()

	// Remove stream when done
	defer func() {
		s.mu.Lock()
		// Remove by replacing with last element and shrinking slice
		if streamIndex < len(s.streams) {
			s.streams[streamIndex] = s.streams[len(s.streams)-1]
			s.streams = s.streams[:len(s.streams)-1]
		}
		s.mu.Unlock()
		log.Println("Progress stream client disconnected")
	}()

	// Keep connection alive
	<-stream.Context().Done()
	return stream.Context().Err()
}

// BroadcastProgress sends a progress update to all connected clients
func (s *ProgressService) BroadcastProgress(update *pb.ProgressUpdate) {
	// Set timestamp if not already set
	if update.Timestamp == 0 {
		update.Timestamp = time.Now().Unix()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.streams) == 0 {
		return
	}

	log.Printf("Broadcasting progress to %d clients: %s", len(s.streams), update.Message)

	// Send to all connected streams
	for i, stream := range s.streams {
		if err := stream.Send(update); err != nil {
			log.Printf("Failed to send progress to client %d: %v", i, err)
		}
	}
}
