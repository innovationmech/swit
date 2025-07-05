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

package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/innovationmech/swit/api/pb"
	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/service"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func (s *Server) runGRPCServer(wg *sync.WaitGroup) {
	defer wg.Done()

	// Get gRPC port from configuration or use default
	grpcPort := s.getGRPCPort()
	grpcAddr := fmt.Sprintf(":%s", grpcPort)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Logger.Fatal("Failed to listen on gRPC port",
			zap.String("address", grpcAddr),
			zap.Error(err))
		return
	}

	// Create gRPC server with optimized configuration
	grpcServer := s.createConfiguredGRPCServer()

	// Safely set the gRPC server with proper synchronization
	s.setGRPCServer(grpcServer)

	// Register services
	pb.RegisterGreeterServer(grpcServer, &service.GreeterService{})

	logger.Logger.Info("Starting gRPC server", zap.String("address", grpcAddr))

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		logger.Logger.Error("gRPC server failed to serve", zap.Error(err))
	}
}

// getGRPCPort returns the gRPC port from configuration or default
func (s *Server) getGRPCPort() string {
	cfg := config.GetConfig()

	// First try to get dedicated gRPC port from config
	if cfg.Server.GRPCPort != "" {
		// Validate the gRPC port
		port := parsePort(cfg.Server.GRPCPort)
		if isValidPort(port) {
			return fmt.Sprintf("%d", port)
		}
		logger.Logger.Warn("Invalid gRPC port in config, using fallback",
			zap.String("configured_port", cfg.Server.GRPCPort),
			zap.Int("parsed_port", port))
	}

	// Fallback: use HTTP port + 1000 for gRPC (e.g., 8080 -> 9080)
	if cfg.Server.Port != "" {
		httpPort := parsePort(cfg.Server.Port)
		grpcPort := httpPort + 1000

		// Validate that the calculated gRPC port is within valid range
		if isValidPort(grpcPort) {
			return fmt.Sprintf("%d", grpcPort)
		}

		logger.Logger.Warn("Calculated gRPC port exceeds valid range, using default",
			zap.Int("http_port", httpPort),
			zap.Int("calculated_grpc_port", grpcPort))
	}

	// Final fallback: use default gRPC port
	return "50051"
}

// createConfiguredGRPCServer creates a new gRPC server with optimized configuration
func (s *Server) createConfiguredGRPCServer() *grpc.Server {
	// Apply keepalive parameters for better connection management
	keepAliveParams := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // Close idle connections after 15s
		MaxConnectionAge:      30 * time.Second, // Close connections after 30s
		MaxConnectionAgeGrace: 5 * time.Second,  // Grace period for closing connections
		Time:                  5 * time.Second,  // Send keepalive pings every 5s
		Timeout:               1 * time.Second,  // Wait 1s for ping response
	}

	keepAlivePolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // Minimum time between pings
		PermitWithoutStream: true,            // Allow pings without active streams
	}

	// Create new server with optimized options
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepAliveParams),
		grpc.KeepaliveEnforcementPolicy(keepAlivePolicy),
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB max receive message size
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB max send message size
	}

	// Return a new configured gRPC server
	return grpc.NewServer(opts...)
}

// GracefulShutdown gracefully shuts down the gRPC server
func (s *Server) GracefulShutdownGRPC(timeout time.Duration) {
	// Safely get the gRPC server with proper synchronization
	grpcServer := s.getGRPCServer()
	if grpcServer == nil {
		logger.Logger.Debug("gRPC server not initialized, skipping shutdown")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to signal when graceful shutdown is complete
	done := make(chan struct{})

	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		logger.Logger.Info("gRPC server gracefully stopped")
	case <-ctx.Done():
		logger.Logger.Warn("gRPC server shutdown timeout, forcing stop")
		grpcServer.Stop()
	}
}

// parsePort safely parses port string to int
func parsePort(portStr string) int {
	if portStr == "" {
		return 8080 // default HTTP port
	}

	// Use strconv.Atoi for robust parsing
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Logger.Error("Failed to parse port string",
			zap.String("port", portStr),
			zap.Error(err))
		return 8080
	}

	// Return parsed port (validation will be done by isValidPort)
	return port
}

// isValidPort validates if a port number is within the valid range
func isValidPort(port int) bool {
	return port > 0 && port <= 65535
}
