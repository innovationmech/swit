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

	"github.com/innovationmech/swit/internal/switauth/adapter"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Server represents the server structure using the new base server framework
// This is a wrapper around the new SwitauthServer for backward compatibility
type Server struct {
	switauthServer *adapter.SwitauthServer
	config         *config.AuthConfig
	deps           *deps.Dependencies
}

// NewServer creates a new server instance using the base server framework
func NewServer() (*Server, error) {
	// Initialize dependencies
	dependencies, err := deps.NewDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	// Create switauth server using the new adapter
	switauthServer, err := adapter.NewSwitauthServer(dependencies.Config, dependencies)
	if err != nil {
		return nil, fmt.Errorf("failed to create switauth server: %w", err)
	}

	// Create wrapper server for backward compatibility
	server := &Server{
		switauthServer: switauthServer,
		config:         dependencies.Config,
		deps:           dependencies,
	}

	return server, nil
}

// registerServices and configureMiddleware are no longer needed
// as they are handled by the base server framework

// Start starts the server using the new base server framework
func (s *Server) Start(ctx context.Context) error {
	if s.switauthServer == nil {
		return fmt.Errorf("switauth server not initialized")
	}

	// Start the switauth server
	if err := s.switauthServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start switauth server: %w", err)
	}

	logger.Logger.Info("Server started successfully",
		zap.String("http_addr", s.switauthServer.GetHTTPAddress()),
		zap.String("grpc_addr", s.switauthServer.GetGRPCAddress()))

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.switauthServer == nil {
		return nil // Nothing to stop
	}

	// Stop the switauth server
	if err := s.switauthServer.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop switauth server: %w", err)
	}

	logger.Logger.Info("Server stopped successfully")
	return nil
}

// Discovery registration/deregistration is handled by base server

// GetHTTPAddress returns the HTTP server address
func (s *Server) GetHTTPAddress() string {
	if s.switauthServer == nil {
		return ""
	}
	return s.switauthServer.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *Server) GetGRPCAddress() string {
	if s.switauthServer == nil {
		return ""
	}
	return s.switauthServer.GetGRPCAddress()
}

// GetServices returns the list of registered services
func (s *Server) GetServices() []string {
	if s.switauthServer == nil {
		return []string{}
	}
	services := s.switauthServer.GetRegisteredServices()
	var serviceNames []string
	for name := range services {
		serviceNames = append(serviceNames, name)
	}
	return serviceNames
}
