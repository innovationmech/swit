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

	"github.com/innovationmech/swit/internal/switserve/adapter"
	"github.com/innovationmech/swit/internal/switserve/deps"
	"github.com/innovationmech/swit/pkg/server"
)

// SwitserveServer represents the switserve service server using base server framework
type SwitserveServer struct {
	baseServer server.BaseServer
	adapter    *adapter.SwitserveAdapter
	deps       *deps.Dependencies
}

// NewSwitserveServer creates a new switserve server instance
func NewSwitserveServer() (*SwitserveServer, error) {
	// Create base server configuration
	config, err := adapter.CreateBaseServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create server config: %w", err)
	}

	// Create dependency container
	depContainer, err := adapter.CreateDependencyContainer(func() {
		// Shutdown callback - will be called when server shuts down
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dependency container: %w", err)
	}

	// Get dependencies from container
	dependencies, err := depContainer.Get("dependencies")
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	deps, ok := dependencies.(*deps.Dependencies)
	if !ok {
		return nil, fmt.Errorf("invalid dependencies type")
	}

	// Create service adapter
	adapter := adapter.NewSwitserveAdapter(deps)

	// Create base server with config, registrar and dependencies
	baseServer, err := server.NewServer(config, adapter, depContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	return &SwitserveServer{
		baseServer: baseServer,
		adapter:    adapter,
		deps:       deps,
	}, nil
}

// Start starts the switserve server
func (s *SwitserveServer) Start(ctx context.Context) error {
	if s.baseServer == nil {
		return fmt.Errorf("base server not initialized")
	}

	return s.baseServer.Start(ctx)
}

// Stop stops the switserve server
func (s *SwitserveServer) Stop(ctx context.Context) error {
	if s.baseServer == nil {
		return nil
	}

	return s.baseServer.Stop(ctx)
}

// Shutdown performs graceful shutdown of the switserve server
func (s *SwitserveServer) Shutdown() error {
	if s.baseServer == nil {
		return nil
	}

	return s.baseServer.Shutdown()
}

// GetHTTPAddress returns the HTTP server address
func (s *SwitserveServer) GetHTTPAddress() string {
	if s.baseServer == nil {
		return ""
	}

	return s.baseServer.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *SwitserveServer) GetGRPCAddress() string {
	if s.baseServer == nil {
		return ""
	}

	return s.baseServer.GetGRPCAddress()
}

// GetServiceName returns the service name
func (s *SwitserveServer) GetServiceName() string {
	return "swit-serve"
}

// IsHealthy checks if the server is healthy
func (s *SwitserveServer) IsHealthy(ctx context.Context) bool {
	if s.baseServer == nil || s.deps == nil {
		return false
	}

	// Check if dependencies are healthy
	if s.deps.HealthSrv != nil {
		_, err := s.deps.HealthSrv.CheckHealth(ctx)
		return err == nil
	}

	return true
}
