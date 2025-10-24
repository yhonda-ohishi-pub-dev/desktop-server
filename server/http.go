package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/yhonda-ohishi-pub-dev/desktop-server/frontend"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
)

type HTTPServer struct {
	grpcServer      *GRPCServer
	httpServer      *http.Server
	progressService *ProgressService
}

func NewHTTPServer(grpcServer *GRPCServer, progressService *ProgressService) *HTTPServer {
	return &HTTPServer{
		grpcServer:      grpcServer,
		progressService: progressService,
	}
}

func (s *HTTPServer) Start(addr string) error {
	// Create gRPC-Web wrapper
	wrappedGrpc := grpcweb.WrapServer(s.grpcServer.grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true // Allow all origins for local development
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}),
	)

	mux := http.NewServeMux()

	// gRPC-Web endpoint (includes ProgressService streaming)
	mux.Handle("/api/", http.StripPrefix("/api", wrappedGrpc))

	// Serve embedded frontend files
	distFS, err := frontend.GetDistFS()
	if err != nil {
		fmt.Printf("Warning: Failed to load frontend files: %v\n", err)
		// Fallback to placeholder
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Desktop Server</title>
</head>
<body>
    <h1>Desktop Server is Running</h1>
    <p>Frontend files not available. Please run with -update flag to download.</p>
    <p>gRPC-Web API is available at <a href="/api">/api</a></p>
</body>
</html>
`)
		})
	} else {
		// Serve static files from embedded filesystem
		fileServer := http.FileServer(http.FS(distFS))
		mux.Handle("/", fileServer)
	}

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
		// ReadTimeout: keep default (no timeout) for streaming
		// WriteTimeout: 0 means no timeout - required for gRPC streaming
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("HTTP server listening on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

func (s *HTTPServer) Stop() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}
