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
	"github.com/innovationmech/swit/internal/switserve/service/greeter"
	"github.com/innovationmech/swit/internal/switserve/service/health"
	"github.com/innovationmech/swit/internal/switserve/service/notification"
	"github.com/innovationmech/swit/internal/switserve/service/stop"
	"github.com/innovationmech/swit/internal/switserve/service/user"
	"github.com/innovationmech/swit/internal/switserve/transport"
	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Server represents the SWIT server implementation
type Server struct {
	transportManager *transport.Manager
	serviceRegistry  *transport.ServiceRegistry
	sd               *discovery.ServiceDiscovery
	httpTransport    *transport.HTTPTransport
	grpcTransport    *transport.GRPCTransport
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	server := &Server{
		transportManager: transport.NewManager(),
		serviceRegistry:  transport.NewServiceRegistry(),
	}

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
	server.registerServices()

	return server, nil
}

// registerServices registers all services with both transports
func (s *Server) registerServices() {
	// Register Greeter service
	greeterRegistrar := greeter.NewServiceRegistrar()
	s.serviceRegistry.Register(greeterRegistrar)

	// Register Notification service
	notificationRegistrar := notification.NewServiceRegistrar()
	s.serviceRegistry.Register(notificationRegistrar)

	// Register Health service
	healthRegistrar := health.NewServiceRegistrar()
	s.serviceRegistry.Register(healthRegistrar)

	// Register Stop service
	stopRegistrar := stop.NewServiceRegistrar(func() {
		if err := s.Shutdown(); err != nil {
			logger.Logger.Error("Failed to shutdown server", zap.Error(err))
		}
	})
	s.serviceRegistry.Register(stopRegistrar)

	// Register User service
	userRegistrar := user.NewServiceRegistrar()
	if userRegistrar != nil {
		s.serviceRegistry.Register(userRegistrar)
	}

	// Register Debug service (needs access to service registry and gin engine)
	// We'll register this after HTTP transport is created
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

	// Register all gRPC services
	if err := s.serviceRegistry.RegisterAllGRPC(grpcServer); err != nil {
		return fmt.Errorf("failed to register gRPC services: %v", err)
	}

	// Register all HTTP routes
	if err := s.serviceRegistry.RegisterAllHTTP(httpRouter); err != nil {
		return fmt.Errorf("failed to register HTTP routes: %v", err)
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

	// Deregister from service discovery
	cfg := config.GetConfig()
	port, _ := strconv.Atoi(cfg.Server.Port)
	if err := s.sd.DeregisterService("swit-serve", "localhost", port); err != nil {
		logger.Logger.Error("Deregister service error", zap.Error(err))
	} else {
		logger.Logger.Info("Service deregistered successfully", zap.String("service", "swit-serve"))
	}

	// Stop all transports
	if err := s.transportManager.Stop(shutdownTimeout); err != nil {
		logger.Logger.Error("Failed to stop transports", zap.Error(err))
		return err
	}

	logger.Logger.Info("Server stopped successfully")
	return nil
}

// Shutdown provides a graceful shutdown method for the stop service
func (s *Server) Shutdown() error {
	return s.Stop()
}

// GetTransports returns all registered transports for inspection
func (s *Server) GetTransports() []transport.Transport {
	return s.transportManager.GetTransports()
}
