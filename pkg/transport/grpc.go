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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// GRPCTransportConfig contains configuration for gRPC transport
type GRPCTransportConfig struct {
	Address             string
	Port                string
	TestMode            bool   // Enables test-specific features
	TestPort            string // Override port for testing
	EnableKeepalive     bool   // Enable keepalive settings
	EnableReflection    bool   // Enable reflection service
	EnableHealthService bool   // Enable health check service
	MaxRecvMsgSize      int    // Maximum receive message size
	MaxSendMsgSize      int    // Maximum send message size
	KeepaliveParams     *keepalive.ServerParameters
	KeepalivePolicy     *keepalive.EnforcementPolicy
	UnaryInterceptors   []grpc.UnaryServerInterceptor
	StreamInterceptors  []grpc.StreamServerInterceptor
}

// DefaultGRPCConfig returns a default gRPC configuration
func DefaultGRPCConfig() *GRPCTransportConfig {
	return &GRPCTransportConfig{
		Address:             ":50051",
		EnableKeepalive:     true,
		EnableReflection:    true,
		EnableHealthService: true,
		MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
		KeepaliveParams: &keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		},
		KeepalivePolicy: &keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		},
	}
}

// GRPCNetworkService implements NetworkTransport interface for gRPC
type GRPCNetworkService struct {
	server          *grpc.Server
	listener        net.Listener
	address         string
	config          *GRPCTransportConfig
	serviceRegistry *TransportServiceRegistry
	mu              sync.RWMutex
}

// NewGRPCNetworkService creates a new gRPC network service with default configuration
func NewGRPCNetworkService() *GRPCNetworkService {
	return NewGRPCNetworkServiceWithConfig(DefaultGRPCConfig())
}

// NewGRPCNetworkServiceWithConfig creates a new gRPC network service with custom configuration
func NewGRPCNetworkServiceWithConfig(config *GRPCTransportConfig) *GRPCNetworkService {
	if config == nil {
		config = DefaultGRPCConfig()
	}

	transport := &GRPCNetworkService{
		config:          config,
		serviceRegistry: NewTransportServiceRegistry(),
	}

	// Create the gRPC server immediately so it's available for service registration
	transport.server = transport.createConfiguredGRPCServer()

	return transport
}

// Start implements Transport interface
func (g *GRPCNetworkService) Start(ctx context.Context) error {
	// Determine address to use
	address := g.determineAddress()

	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Logger.Error("Failed to listen on gRPC port",
			zap.String("address", address),
			zap.Error(err))
		return fmt.Errorf("failed to create gRPC listener: %w", err)
	}

	g.mu.Lock()
	g.listener = lis
	g.address = lis.Addr().String() // Update actual address in case of :0 port
	server := g.server              // Capture server reference inside the lock
	g.mu.Unlock()

	logger.Logger.Info("Starting gRPC transport", zap.String("address", g.address))

	// Start serving in a goroutine
	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Logger.Error("gRPC server failed to serve", zap.Error(err))
		}
	}()

	return nil
}

// Stop implements Transport interface
func (g *GRPCNetworkService) Stop(ctx context.Context) error {
	g.mu.Lock()
	server := g.server // Capture server reference before unlocking
	listener := g.listener
	g.mu.Unlock()

	if server == nil {
		logger.Logger.Debug("gRPC server not initialized, skipping shutdown")
		return nil
	}

	logger.Logger.Info("Stopping gRPC transport")

	// First shutdown all services
	if err := g.serviceRegistry.ShutdownAll(ctx); err != nil {
		logger.Logger.Error("Failed to shutdown gRPC services", zap.Error(err))
		// Continue with server shutdown even if service shutdown fails
	}

	// Channel to signal when graceful shutdown is complete
	done := make(chan struct{})

	go func() {
		server.GracefulStop()
		close(done)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		logger.Logger.Info("gRPC server gracefully stopped")
	case <-ctx.Done():
		logger.Logger.Warn("gRPC server shutdown timeout, forcing stop")
		server.Stop()
	}

	// Reset server and listener after shutdown
	g.mu.Lock()
	g.server = g.createConfiguredGRPCServer() // Create new server for potential restart
	if listener != nil {
		listener.Close()
		g.listener = nil
	}
	g.address = "" // Reset address
	g.mu.Unlock()

	return nil
}

// GetName implements Transport interface
func (g *GRPCNetworkService) GetName() string {
	return "grpc"
}

// GetAddress implements Transport interface
func (g *GRPCNetworkService) GetAddress() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Try actual address first (set when server is running)
	address := g.address
	// If not running, try configured address
	if address == "" && g.config != nil && g.config.Address != "" {
		address = g.config.Address
	}

	return address
}

// GetServer returns the gRPC server instance for service registration
func (g *GRPCNetworkService) GetServer() *grpc.Server {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.server
}

// GetPort returns the actual port the server is listening on
func (g *GRPCNetworkService) GetPort() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Try actual address first (set when server is running)
	address := g.address
	// If not running, try configured address
	if address == "" && g.config != nil && g.config.Address != "" {
		address = g.config.Address
	}

	if address == "" {
		return 0
	}

	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return 0
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}

	return port
}

// SetTestPort sets a custom port for testing (use "0" for dynamic port allocation)
func (g *GRPCNetworkService) SetTestPort(port string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.config == nil {
		g.config = DefaultGRPCConfig()
	}
	g.config.TestPort = port
	g.config.TestMode = true
}

// SetAddress sets the gRPC server address
func (g *GRPCNetworkService) SetAddress(addr string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.config == nil {
		g.config = DefaultGRPCConfig()
	}
	g.config.Address = addr
}

// AddUnaryInterceptor adds a unary interceptor to the configuration
func (g *GRPCNetworkService) AddUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.config == nil {
		g.config = DefaultGRPCConfig()
	}
	g.config.UnaryInterceptors = append(g.config.UnaryInterceptors, interceptor)
}

// AddStreamInterceptor adds a stream interceptor to the configuration
func (g *GRPCNetworkService) AddStreamInterceptor(interceptor grpc.StreamServerInterceptor) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.config == nil {
		g.config = DefaultGRPCConfig()
	}
	g.config.StreamInterceptors = append(g.config.StreamInterceptors, interceptor)
}

// createConfiguredGRPCServer creates a new gRPC server with configuration
func (g *GRPCNetworkService) createConfiguredGRPCServer() *grpc.Server {
	opts := []grpc.ServerOption{}

	// Add keepalive settings if enabled
	if g.config.EnableKeepalive {
		if g.config.KeepaliveParams != nil {
			opts = append(opts, grpc.KeepaliveParams(*g.config.KeepaliveParams))
		}
		if g.config.KeepalivePolicy != nil {
			opts = append(opts, grpc.KeepaliveEnforcementPolicy(*g.config.KeepalivePolicy))
		}
	}

	// Add message size limits
	if g.config.MaxRecvMsgSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(g.config.MaxRecvMsgSize))
	}
	if g.config.MaxSendMsgSize > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(g.config.MaxSendMsgSize))
	}

	// Add interceptors
	if len(g.config.UnaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(g.config.UnaryInterceptors...))
	}
	if len(g.config.StreamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(g.config.StreamInterceptors...))
	}

	// Create server
	server := grpc.NewServer(opts...)

	// Register health check service if enabled
	if g.config.EnableHealthService {
		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(server, healthServer)
		healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	}

	// Register reflection service if enabled
	if g.config.EnableReflection {
		reflection.Register(server)
	}

	return server
}

// determineAddress determines the address to use for the gRPC server
func (g *GRPCNetworkService) determineAddress() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Test port override takes highest priority
	if g.config != nil && g.config.TestMode && g.config.TestPort != "" {
		return ":" + g.config.TestPort
	}

	// Use configured address
	if g.config != nil && g.config.Address != "" {
		return g.config.Address
	}

	// Default fallback
	return ":50051"
}

// Helper function to calculate gRPC port from HTTP port
func CalculateGRPCPort(httpPort string) string {
	if httpPort == "" {
		return "50051"
	}

	port := parsePort(httpPort)
	grpcPort := port + 1000

	if isValidPort(grpcPort) {
		return fmt.Sprintf("%d", grpcPort)
	}

	return "50051"
}

// Helper functions
func parsePort(portStr string) int {
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

// RegisterHandler registers a service handler with the gRPC transport
func (g *GRPCNetworkService) RegisterHandler(handler TransportServiceHandler) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Register the service with the registry
	if err := g.serviceRegistry.Register(handler); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	return nil
}

// RegisterService is an alias for RegisterHandler for backward compatibility
func (g *GRPCNetworkService) RegisterService(handler TransportServiceHandler) error {
	return g.RegisterHandler(handler)
}

// InitializeServices initializes all registered services
func (g *GRPCNetworkService) InitializeServices(ctx context.Context) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.serviceRegistry.InitializeTransportServices(ctx)
}

// RegisterAllServices registers gRPC services for all registered services
func (g *GRPCNetworkService) RegisterAllServices() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.server == nil {
		return fmt.Errorf("gRPC server not initialized")
	}

	return g.serviceRegistry.BindAllGRPCServices(g.server)
}

// GetServiceRegistry returns the service registry
func (g *GRPCNetworkService) GetServiceRegistry() *TransportServiceRegistry {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.serviceRegistry
}

// ShutdownServices gracefully shuts down all registered services
func (g *GRPCNetworkService) ShutdownServices(ctx context.Context) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.serviceRegistry.ShutdownAll(ctx)
}
