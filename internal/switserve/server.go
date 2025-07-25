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
	"strconv"
	"time"

	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/deps"
	greeterv1 "github.com/innovationmech/swit/internal/switserve/handler/http/greeter/v1"
	"github.com/innovationmech/swit/internal/switserve/handler/http/health"
	notificationv1 "github.com/innovationmech/swit/internal/switserve/handler/http/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/handler/http/stop"
	userv1 "github.com/innovationmech/swit/internal/switserve/handler/http/user/v1"
	"github.com/innovationmech/swit/internal/switserve/transport"
	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Server represents the SWIT server implementation
type Server struct {
	transportManager *transport.Manager
	sd               *discovery.ServiceDiscovery
	httpTransport    *transport.HTTPTransport
	grpcTransport    *transport.GRPCTransport
	deps             *deps.Dependencies
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	server := &Server{
		transportManager: transport.NewManager(),
	}

	// Initialize dependencies with shutdown callback
	dependencies, err := deps.NewDependencies(func() {
		if err := server.Shutdown(); err != nil {
			logger.Logger.Error("Failed to shutdown server during dependency cleanup", zap.Error(err))
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %v", err)
	}

	server.deps = dependencies

	// Setup service discovery
	cfg := config.GetConfig()
	sd, err := discovery.GetServiceDiscoveryByAddress(cfg.ServiceDiscovery.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery client: %v", err)
	}
	server.sd = sd

	// Initialize transports
	server.httpTransport = transport.NewHTTPTransport()
	server.grpcTransport = transport.NewGRPCTransport()

	// Register transports
	server.transportManager.Register(server.httpTransport)
	server.transportManager.Register(server.grpcTransport)

	// Register services
	if err := server.registerServices(); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return server, nil
}

// registerServices registers all services with the service registry
func (s *Server) registerServices() error {
	// Register services with the HTTP transport's service registry
	// This uses the new ServiceHandler interface for unified service management

	// Register Greeter service with dependency injection
	greeterHandler := greeterv1.NewGreeterHandler(s.deps.GreeterSrv)
	if err := s.httpTransport.RegisterService(greeterHandler); err != nil {
		return fmt.Errorf("failed to register greeter service: %w", err)
	}

	// Register Notification service with dependency injection
	notificationHandler := notificationv1.NewHandler(s.deps.NotificationSrv)
	if err := s.httpTransport.RegisterService(notificationHandler); err != nil {
		return fmt.Errorf("failed to register notification service: %w", err)
	}

	// Register Health service with dependency injection
	healthHandler := health.NewHandler(s.deps.HealthSrv)
	if err := s.httpTransport.RegisterService(healthHandler); err != nil {
		return fmt.Errorf("failed to register health service: %w", err)
	}

	// Register Stop service with dependency injection
	stopHandler := stop.NewHandler(s.deps.StopSrv)
	if err := s.httpTransport.RegisterService(stopHandler); err != nil {
		return fmt.Errorf("failed to register stop service: %w", err)
	}

	// Register User service with dependency injection
	userHandler := userv1.NewUserController(s.deps.UserSrv)
	if err := s.httpTransport.RegisterService(userHandler); err != nil {
		return fmt.Errorf("failed to register user service: %w", err)
	}

	return nil
}

// Start starts the server with all transports
func (s *Server) Start(ctx context.Context) error {
	// Get gRPC server (middleware is already configured in transport layer)
	grpcServer := s.grpcTransport.GetServer()

	// Get HTTP router
	httpRouter := s.httpTransport.GetRouter()
	if httpRouter != nil {
		// Add HTTP middleware here if needed
	}

	// Register all HTTP routes through HTTP transport
	if err := s.httpTransport.RegisterAllRoutes(); err != nil {
		return fmt.Errorf("failed to register HTTP routes: %v", err)
	}

	// Register all gRPC services through HTTP transport's service registry
	serviceRegistry := s.httpTransport.GetServiceRegistry()
	if err := serviceRegistry.RegisterAllGRPC(grpcServer); err != nil {
		return fmt.Errorf("failed to register gRPC services: %v", err)
	}

	// Start all transports
	if err := s.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transports: %v", err)
	}

	// Wait for HTTP transport to be ready
	<-s.httpTransport.WaitReady()

	// Register service in service discovery
	cfg := config.GetConfig()
	port, _ := strconv.Atoi(cfg.Server.Port)
	if err := s.sd.RegisterService("swit-serve", "localhost", port); err != nil {
		logger.Logger.Error("failed to register swit-serve service", zap.Error(err))
		return err
	}

	logger.Logger.Info("Server started successfully",
		zap.String("http_address", s.httpTransport.Address()),
		zap.String("grpc_address", s.grpcTransport.Address()),
	)

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	shutdownTimeout := 5 * time.Second

	// Deregister from service discovery if available
	if s.sd != nil {
		cfg := config.GetConfig()
		port, _ := strconv.Atoi(cfg.Server.Port)
		if err := s.sd.DeregisterService("swit-serve", "localhost", port); err != nil {
			logger.Logger.Error("Deregister service error", zap.Error(err))
		} else {
			logger.Logger.Info("Service deregistered successfully", zap.String("service", "swit-serve"))
		}
	}

	// Stop all transports if available
	if s.transportManager != nil {
		if err := s.transportManager.Stop(shutdownTimeout); err != nil {
			logger.Logger.Error("Failed to stop transports", zap.Error(err))
			return err
		}
	}

	logger.Logger.Info("Server stopped successfully")
	return nil
}

// Shutdown provides a graceful shutdown method for the stop service
func (s *Server) Shutdown() error {
	// Close dependencies before stopping the server
	if s.deps != nil {
		if err := s.deps.Close(); err != nil {
			logger.Logger.Error("Failed to close dependencies", zap.Error(err))
		}
	}

	return s.Stop()
}

// GetTransports returns all registered transports for inspection
func (s *Server) GetTransports() []transport.Transport {
	return s.transportManager.GetTransports()
}
