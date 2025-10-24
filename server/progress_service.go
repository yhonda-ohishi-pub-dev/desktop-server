package server

import (
	"log"
	"sync"
	"time"

	pb "github.com/yhonda-ohishi-pub-dev/desktop-server/proto"
)

// streamInfo holds stream and its associated job ID
type streamInfo struct {
	stream pb.ProgressService_StreamDownloadProgressServer
	jobId  string
	done   chan struct{}
}

// ProgressService implements the ProgressService gRPC service
type ProgressService struct {
	pb.UnimplementedProgressServiceServer
	mu      sync.RWMutex
	streams []*streamInfo
}

// NewProgressService creates a new ProgressService
func NewProgressService() *ProgressService {
	return &ProgressService{
		streams: make([]*streamInfo, 0),
	}
}

// StreamDownloadProgress streams download progress updates to clients
func (s *ProgressService) StreamDownloadProgress(req *pb.StreamProgressRequest, stream pb.ProgressService_StreamDownloadProgressServer) error {
	log.Printf("New progress stream client connected (job_id filter: %s)", req.JobId)

	info := &streamInfo{
		stream: stream,
		jobId:  req.JobId,
		done:   make(chan struct{}),
	}

	// Add stream to list
	s.mu.Lock()
	s.streams = append(s.streams, info)
	s.mu.Unlock()

	// Remove stream when done
	defer func() {
		s.mu.Lock()
		for i, si := range s.streams {
			if si == info {
				s.streams = append(s.streams[:i], s.streams[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		log.Println("Progress stream client disconnected")
	}()

	// Wait for completion or context cancellation
	select {
	case <-info.done:
		return nil
	case <-stream.Context().Done():
		return stream.Context().Err()
	}
}

// BroadcastProgress sends a progress update to all connected clients
func (s *ProgressService) BroadcastProgress(update *pb.ProgressUpdate) {
	// Set timestamp if not already set
	if update.Timestamp == 0 {
		update.Timestamp = time.Now().Unix()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.streams) == 0 {
		return
	}

	log.Printf("Broadcasting progress to %d clients: %s (type: %v, jobId: %s)", len(s.streams), update.Message, update.Type, update.JobId)

	// Send to all connected streams (or matching job ID)
	for i, info := range s.streams {
		// Filter by job ID if specified
		if update.JobId != "" && info.jobId != "" && info.jobId != update.JobId {
			continue
		}

		if err := info.stream.Send(update); err != nil {
			log.Printf("Failed to send progress to client %d: %v", i, err)
			continue
		}

		// Close stream if job is complete or error
		if update.Type == pb.ProgressType_PROGRESS_TYPE_COMPLETE || update.Type == pb.ProgressType_PROGRESS_TYPE_ERROR {
			log.Printf("Closing stream for job %s (type: %v)", update.JobId, update.Type)
			close(info.done)
		}
	}
}
