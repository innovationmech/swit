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

	"github.com/innovationmech/swit/internal/base/server"
	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/deps"
	"github.com/innovationmech/swit/pkg/logger"
	baseserver "github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/transport"
	"go.uber.org/zap"
)

// Server represents the SWIT server implementation with base server framework
type Server struct {
	baseServer *baseserver.BusinessServerImpl
}

// NewServer creates a new server instance using the base server framework
func NewServer() (*Server, error) {
	logger.Logger.Info("Creating switserve server with base framework")

	// Initialize dependencies with shutdown callback
	var baseServer *baseserver.BusinessServerImpl
	dependencies, err := deps.NewDependencies(func() {
		if baseServer != nil {
			if err := baseServer.Shutdown(); err != nil {
				logger.Logger.Error("Failed to shutdown server during dependency cleanup", zap.Error(err))
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %v", err)
	}

	// Create service registrar
	serviceRegistrar := NewServiceRegistrar(dependencies)

	// Create dependency container
	dependencyContainer := NewBusinessDependencyContainer(dependencies)

	// Map configuration
	cfg := config.GetConfig()
	configMapper := NewConfigMapper(cfg)
	serverConfig := configMapper.ToServerConfig()

	// Create server factory
	factory, err := server.NewServerFactory(serverConfig)
	if err != nil {
		// Clean up dependencies if factory creation fails
		if closeErr := dependencyContainer.Close(); closeErr != nil {
			logger.Logger.Error("Failed to close dependencies after factory creation failure", zap.Error(closeErr))
		}
		return nil, fmt.Errorf("failed to create server factory: %v", err)
	}

	// Create base server
	baseServerInterface, err := factory.CreateServer(serviceRegistrar, dependencyContainer)
	if err != nil {
		// Clean up dependencies if server creation fails
		if closeErr := dependencyContainer.Close(); closeErr != nil {
			logger.Logger.Error("Failed to close dependencies after server creation failure", zap.Error(closeErr))
		}
		return nil, fmt.Errorf("failed to create base server: %v", err)
	}

	// Type assert to BusinessServerImpl
	var ok bool
	baseServer, ok = baseServerInterface.(*baseserver.BusinessServerImpl)
	if !ok {
		// Clean up dependencies if type assertion fails
		if closeErr := dependencyContainer.Close(); closeErr != nil {
			logger.Logger.Error("Failed to close dependencies after type assertion failure", zap.Error(closeErr))
		}
		return nil, fmt.Errorf("failed to cast base server to BusinessServerImpl")
	}

	// Create wrapper server
	wrapper := &Server{
		baseServer: baseServer,
	}

	logger.Logger.Info("Successfully created switserve server with base framework")
	return wrapper, nil
}

// Start starts the server with all transports
func (s *Server) Start(ctx context.Context) error {
	if s.baseServer == nil {
		return fmt.Errorf("base server is not initialized")
	}
	return s.baseServer.Start(ctx)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.baseServer == nil {
		return nil
	}
	return s.baseServer.Stop(context.Background())
}

// Shutdown provides a graceful shutdown method for the stop service
func (s *Server) Shutdown() error {
	if s.baseServer == nil {
		return nil
	}
	return s.baseServer.Shutdown()
}

// GetTransports returns all registered transports for inspection
func (s *Server) GetTransports() []transport.NetworkTransport {
	if s.baseServer == nil {
		return []transport.NetworkTransport{}
	}
	return s.baseServer.GetTransports()
}

// GetHTTPAddress returns the HTTP server listening address
func (s *Server) GetHTTPAddress() string {
	if s.baseServer == nil {
		return ""
	}
	return s.baseServer.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server listening address
func (s *Server) GetGRPCAddress() string {
	if s.baseServer == nil {
		return ""
	}
	return s.baseServer.GetGRPCAddress()
}
