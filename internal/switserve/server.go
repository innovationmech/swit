// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package switserve

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Server represents the main server instance using base server framework
type Server struct {
	switserveServer *SwitserveServer
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	// Create switserve server using base server framework
	switserveServer, err := NewSwitserveServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create switserve server: %w", err)
	}

	return &Server{
		switserveServer: switserveServer,
	}, nil
}

// Start starts the server
func (s *Server) Start() error {
	if s.switserveServer == nil {
		return fmt.Errorf("switserve server not initialized")
	}

	logger.Logger.Info("Starting switserve server")

	// Start the server in a goroutine
	ctx := context.Background()
	go func() {
		if err := s.switserveServer.Start(ctx); err != nil {
			logger.Logger.Fatal("Failed to start switserve server", zap.Error(err))
		}
	}()

	logger.Logger.Info("Switserve server started successfully",
		zap.String("http_address", s.switserveServer.GetHTTPAddress()),
		zap.String("grpc_address", s.switserveServer.GetGRPCAddress()),
	)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("Shutting down switserve server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop the switserve server
	if err := s.switserveServer.Stop(ctx); err != nil {
		logger.Logger.Error("Switserve server forced to shutdown", zap.Error(err))
		return err
	}

	// Perform final shutdown
	if err := s.switserveServer.Shutdown(); err != nil {
		logger.Logger.Error("Failed to shutdown switserve server", zap.Error(err))
		return err
	}

	logger.Logger.Info("Switserve server exited")
	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	if s.switserveServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.switserveServer.Stop(ctx)
}

// GetAddress returns the server HTTP address
func (s *Server) GetAddress() string {
	if s.switserveServer == nil {
		return ""
	}
	return s.switserveServer.GetHTTPAddress()
}

// GetHTTPAddress returns the HTTP server address
func (s *Server) GetHTTPAddress() string {
	if s.switserveServer == nil {
		return ""
	}
	return s.switserveServer.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *Server) GetGRPCAddress() string {
	if s.switserveServer == nil {
		return ""
	}
	return s.switserveServer.GetGRPCAddress()
}

// GetServiceName returns the service name
func (s *Server) GetServiceName() string {
	if s.switserveServer == nil {
		return "swit-serve"
	}
	return s.switserveServer.GetServiceName()
}

// IsHealthy checks if the server is healthy
func (s *Server) IsHealthy(ctx context.Context) bool {
	if s.switserveServer == nil {
		return false
	}
	return s.switserveServer.IsHealthy(ctx)
}
