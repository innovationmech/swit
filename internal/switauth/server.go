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

	"github.com/innovationmech/swit/internal/base/server"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	"github.com/innovationmech/swit/pkg/logger"
	pkgserver "github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/transport"
	"go.uber.org/zap"
)

// Server represents the switauth server using the base server framework
// It wraps the base server and provides backward compatibility
type Server struct {
	baseServer pkgserver.BusinessServerCore
	config     *config.AuthConfig
	deps       *deps.Dependencies
}

// NewServer creates a new server instance using the base server framework
func NewServer() (*Server, error) {
	// Initialize dependencies
	dependencies, err := deps.NewDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	// Create configuration mapper and convert to base server config
	configMapper := NewConfigMapper(dependencies.Config)
	serverConfig := configMapper.ToServerConfig()

	// Create service registrar
	serviceRegistrar := NewServiceRegistrar(dependencies)

	// Create dependency container
	depContainer := NewBusinessDependencyContainer(dependencies)

	// Create base server using factory
	factory, err := server.NewServerFactory(serverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create server factory: %w", err)
	}

	baseServer, err := factory.CreateServer(serviceRegistrar, depContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	// Create switauth server wrapper
	switauthServer := &Server{
		baseServer: baseServer,
		config:     dependencies.Config,
		deps:       dependencies,
	}

	logger.Logger.Info("Created switauth server using base server framework")
	return switauthServer, nil
}

// No longer needed - service registration is handled by the ServiceRegistrar
// and middleware configuration is handled by the base server

// Start starts the server using the base server framework
func (s *Server) Start(ctx context.Context) error {
	logger.Logger.Info("Starting switauth server using base server framework")

	if err := s.baseServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start base server: %w", err)
	}

	logger.Logger.Info("Switauth server started successfully",
		zap.String("http_addr", s.baseServer.GetHTTPAddress()),
		zap.String("grpc_addr", s.baseServer.GetGRPCAddress()))

	return nil
}

// Stop gracefully stops the server using the base server framework
func (s *Server) Stop(ctx context.Context) error {
	logger.Logger.Info("Stopping switauth server using base server framework")

	if err := s.baseServer.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop base server: %w", err)
	}

	logger.Logger.Info("Switauth server stopped successfully")
	return nil
}

// Service discovery registration/deregistration is now handled by the base server

// GetHTTPAddress returns the HTTP server address
func (s *Server) GetHTTPAddress() string {
	return s.baseServer.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *Server) GetGRPCAddress() string {
	return s.baseServer.GetGRPCAddress()
}

// GetTransports returns all registered transports
func (s *Server) GetTransports() []transport.NetworkTransport {
	return s.baseServer.GetTransports()
}

// Shutdown performs complete server shutdown with resource cleanup
func (s *Server) Shutdown() error {
	logger.Logger.Info("Shutting down switauth server")

	if err := s.baseServer.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown base server: %w", err)
	}

	logger.Logger.Info("Switauth server shutdown completed")
	return nil
}
