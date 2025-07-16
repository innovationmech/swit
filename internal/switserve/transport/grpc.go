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

package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// GRPCTransport implements Transport interface for gRPC
type GRPCTransport struct {
	server   *grpc.Server
	listener net.Listener
	address  string
	testPort string // Override port for testing (empty means use config)
	mu       sync.RWMutex
}

// NewGRPCTransport creates a new gRPC transport
func NewGRPCTransport() *GRPCTransport {
	return &GRPCTransport{}
}

// Start implements Transport interface
func (g *GRPCTransport) Start(ctx context.Context) error {
	// Get gRPC port from configuration
	grpcPort := g.getGRPCPort()
	g.address = fmt.Sprintf(":%s", grpcPort)

	lis, err := net.Listen("tcp", g.address)
	if err != nil {
		logger.Logger.Error("Failed to listen on gRPC port",
			zap.String("address", g.address),
			zap.Error(err))
		return err
	}

	g.mu.Lock()
	g.listener = lis
	g.server = g.createConfiguredGRPCServer()
	server := g.server // Capture server reference inside the lock
	g.mu.Unlock()

	logger.Logger.Info("Starting gRPC server", zap.String("address", g.address))

	// Start serving in a goroutine
	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Logger.Error("gRPC server failed to serve", zap.Error(err))
		}
	}()

	return nil
}

// Stop implements Transport interface
func (g *GRPCTransport) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.server == nil {
		logger.Logger.Debug("gRPC server not initialized, skipping shutdown")
		return nil
	}

	// Channel to signal when graceful shutdown is complete
	done := make(chan struct{})

	go func() {
		g.server.GracefulStop()
		close(done)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		logger.Logger.Info("gRPC server gracefully stopped")
	case <-ctx.Done():
		logger.Logger.Warn("gRPC server shutdown timeout, forcing stop")
		g.server.Stop()
	}

	return nil
}

// Name implements Transport interface
func (g *GRPCTransport) Name() string {
	return "grpc"
}

// Address implements Transport interface
func (g *GRPCTransport) Address() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.address
}

// GetServer returns the gRPC server instance for service registration
func (g *GRPCTransport) GetServer() *grpc.Server {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.server
}

// SetTestPort sets a custom port for testing (use "0" for dynamic port allocation)
func (g *GRPCTransport) SetTestPort(port string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.testPort = port
}

// getGRPCPort returns the gRPC port from configuration or default
func (g *GRPCTransport) getGRPCPort() string {
	g.mu.RLock()
	if g.testPort != "" {
		g.mu.RUnlock()
		return g.testPort
	}
	g.mu.RUnlock()

	cfg := config.GetConfig()

	// First try to get dedicated gRPC port from config
	if cfg.Server.GRPCPort != "" {
		if isValidPort(parsePort(cfg.Server.GRPCPort)) {
			return cfg.Server.GRPCPort
		}
		logger.Logger.Warn("Invalid gRPC port in config, using fallback",
			zap.String("configured_port", cfg.Server.GRPCPort))
	}

	// Fallback: use HTTP port + 1000 for gRPC (e.g., 8080 -> 9080)
	if cfg.Server.Port != "" {
		httpPort := parsePort(cfg.Server.Port)
		grpcPort := httpPort + 1000

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
func (g *GRPCTransport) createConfiguredGRPCServer() *grpc.Server {
	// Apply keepalive parameters for better connection management
	keepAliveParams := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}

	keepAlivePolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	// Create new server with optimized options and middleware
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepAliveParams),
		grpc.KeepaliveEnforcementPolicy(keepAlivePolicy),
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB max receive message size
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB max send message size
		// Add interceptors for middleware
		grpc.ChainUnaryInterceptor(
			middleware.GRPCRecoveryInterceptor(),
			middleware.GRPCLoggingInterceptor(),
			middleware.GRPCValidationInterceptor(),
		),
	}

	return grpc.NewServer(opts...)
}

// Helper functions
func parsePort(portStr string) int {
	// Implementation from original server code
	if portStr == "" {
		return 8080
	}

	port := 8080
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		logger.Logger.Error("Failed to parse port string",
			zap.String("port", portStr),
			zap.Error(err))
		return 8080
	}

	return port
}

func isValidPort(port int) bool {
	return port > 0 && port <= 65535
}
