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

package switauth

import (
	"context"
	"fmt"

	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	auth "github.com/innovationmech/swit/internal/switauth/handler/http/auth/v1"
	health "github.com/innovationmech/swit/internal/switauth/handler/http/health"
	"github.com/innovationmech/swit/internal/switauth/transport"
	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"go.uber.org/zap"
)

// Server represents the server structure with transport manager
// Uses the service-centric architecture with ServiceHandler pattern
type Server struct {
	transportManager *transport.Manager
	httpTransport    *transport.HTTPTransport
	grpcTransport    *transport.GRPCTransport
	sd               *discovery.ServiceDiscovery
	config           *config.AuthConfig
	deps             *deps.Dependencies
}

// NewServer creates a new server instance with transport manager
func NewServer() (*Server, error) {
	// Initialize dependencies
	dependencies, err := deps.NewDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	// Create HTTP transport
	httpTransport := transport.NewHTTPTransport()
	httpTransport.SetAddress(":" + dependencies.Config.Server.Port)

	// Create gRPC transport
	grpcTransport := transport.NewGRPCTransport()
	grpcPort := dependencies.Config.Server.GRPCPort
	if grpcPort == "" {
		grpcPort = "50051" // Default fallback
	}
	grpcTransport.SetAddress(":" + grpcPort)

	// Create transport manager
	transportManager := transport.NewManager()
	transportManager.Register(httpTransport)
	transportManager.Register(grpcTransport)

	// Create server
	server := &Server{
		transportManager: transportManager,
		httpTransport:    httpTransport,
		grpcTransport:    grpcTransport,
		sd:               dependencies.SD,
		config:           dependencies.Config,
		deps:             dependencies,
	}

	// Register services
	server.registerServices()

	return server, nil
}

// registerServices registers all services with the HTTP transport's enhanced service registry
func (s *Server) registerServices() {
	// Register services with the HTTP transport's enhanced service registry
	// This uses the new ServiceHandler interface for unified service management

	// Register authentication service with dependency injection
	authHandler := auth.NewAuthController(s.deps.AuthSrv)
	if err := s.httpTransport.RegisterService(authHandler); err != nil {
		logger.Logger.Error("Failed to register auth service", zap.Error(err))
	}

	// Register health service with dependency injection
	healthHandler := health.NewHandler()
	if err := s.httpTransport.RegisterService(healthHandler); err != nil {
		logger.Logger.Error("Failed to register health service", zap.Error(err))
	}
}

// configureMiddleware configures global middleware for HTTP transport
func (s *Server) configureMiddleware() {
	router := s.httpTransport.GetRouter()

	// Apply global middleware via registrar
	registrar := middleware.NewGlobalMiddlewareRegistrar()
	registrar.RegisterMiddleware(router)
}

// Start starts the server with all registered services
func (s *Server) Start(ctx context.Context) error {
	// Configure middleware
	s.configureMiddleware()

	// Initialize all services
	if err := s.httpTransport.InitializeServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Register all HTTP routes through HTTP transport
	if err := s.httpTransport.RegisterAllRoutes(); err != nil {
		return fmt.Errorf("failed to register HTTP routes: %w", err)
	}

	// Register all gRPC services through HTTP transport's service registry
	serviceRegistry := s.httpTransport.GetServiceRegistry()
	if err := serviceRegistry.RegisterAllGRPC(s.grpcTransport.GetServer()); err != nil {
		return fmt.Errorf("failed to register gRPC services: %w", err)
	}

	// Start all transports
	if err := s.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transports: %w", err)
	}

	// Register with service discovery
	if err := s.registerWithDiscovery(); err != nil {
		return fmt.Errorf("failed to register with discovery: %w", err)
	}

	logger.Logger.Info("Server started successfully",
		zap.String("http_addr", s.httpTransport.GetAddress()),
		zap.String("grpc_addr", s.grpcTransport.GetAddress()))

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	// Deregister from service discovery
	if err := s.deregisterFromDiscovery(); err != nil {
		logger.Logger.Error("Failed to deregister from discovery", zap.Error(err))
	}

	// Stop all transports (this will shutdown services through HTTP transport)
	if err := s.transportManager.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop transports: %w", err)
	}

	logger.Logger.Info("Server stopped successfully")
	return nil
}

// registerWithDiscovery registers the service with service discovery
func (s *Server) registerWithDiscovery() error {
	httpPort := s.httpTransport.GetPort()
	grpcPort := s.grpcTransport.GetPort()

	// Register HTTP service
	if err := s.sd.RegisterService("swit-auth", "localhost", httpPort); err != nil {
		return fmt.Errorf("failed to register HTTP service with port %d: %w", httpPort, err)
	}

	// Register gRPC service (if different port)
	if grpcPort != httpPort {
		if err := s.sd.RegisterService("swit-auth-grpc", "localhost", grpcPort); err != nil {
			logger.Logger.Warn("Failed to register gRPC service", zap.Error(err))
		}
	}

	return nil
}

// deregisterFromDiscovery deregisters the service from service discovery
func (s *Server) deregisterFromDiscovery() error {
	httpPort := s.httpTransport.GetPort()
	grpcPort := s.grpcTransport.GetPort()

	// Deregister HTTP service
	if err := s.sd.DeregisterService("swit-auth", "localhost", httpPort); err != nil {
		return fmt.Errorf("failed to deregister HTTP service: %w", err)
	}

	// Deregister gRPC service
	if grpcPort != httpPort {
		if err := s.sd.DeregisterService("swit-auth-grpc", "localhost", grpcPort); err != nil {
			logger.Logger.Warn("Failed to deregister gRPC service", zap.Error(err))
		}
	}

	return nil
}

// GetHTTPAddress returns the HTTP server address
func (s *Server) GetHTTPAddress() string {
	return s.httpTransport.GetAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *Server) GetGRPCAddress() string {
	return s.grpcTransport.GetAddress()
}

// GetServices returns the list of registered services
func (s *Server) GetServices() []string {
	return s.httpTransport.GetServiceRegistry().GetServiceNames()
}
