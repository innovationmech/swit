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
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// GRPCTransport handles gRPC transport
// Implements the Transport interface for gRPC protocol
type GRPCTransport struct {
	server *grpc.Server
	addr   string
	mu     sync.RWMutex
}

// NewGRPCTransport creates a new gRPC transport instance
func NewGRPCTransport() *GRPCTransport {
	return &GRPCTransport{}
}

// Start starts the gRPC server on the specified address
func (g *GRPCTransport) Start(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.addr == "" {
		g.addr = ":50051" // Default gRPC port
	}

	// Create gRPC server with default options
	g.server = grpc.NewServer()

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(g.server, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection service for debugging
	reflection.Register(g.server)

	ln, err := net.Listen("tcp", g.addr)
	if err != nil {
		return fmt.Errorf("failed to create gRPC listener: %w", err)
	}

	// Update actual address in case of :0 port
	g.addr = ln.Addr().String()

	logger.Logger.Info("Starting gRPC transport",
		zap.String("address", g.addr))

	go func() {
		if err := g.server.Serve(ln); err != nil {
			logger.Logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (g *GRPCTransport) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.server == nil {
		return nil
	}

	logger.Logger.Info("Stopping gRPC transport")
	g.server.GracefulStop()
	return nil
}

// GetName returns the transport name
func (g *GRPCTransport) GetName() string {
	return "grpc"
}

// SetAddress sets the gRPC server address
func (g *GRPCTransport) SetAddress(addr string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.addr = addr
}

// GetAddress returns the current gRPC server address
func (g *GRPCTransport) GetAddress() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.addr
}

// GetServer returns the gRPC server instance for service registration
func (g *GRPCTransport) GetServer() *grpc.Server {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.server
}

// GetPort returns the actual port the server is listening on
func (g *GRPCTransport) GetPort() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.addr == "" {
		return 0
	}

	_, portStr, err := net.SplitHostPort(g.addr)
	if err != nil {
		return 0
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}

	return port
}
