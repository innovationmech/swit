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

package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	"github.com/innovationmech/swit/pkg/server"
)

// SwitauthServer wraps the base server with switauth-specific functionality
type SwitauthServer struct {
	baseServer server.BaseServer
	deps       *deps.Dependencies
	config     *config.AuthConfig
}

// NewSwitauthServer creates a new switauth server using the base server framework
func NewSwitauthServer(cfg *config.AuthConfig, dependencies *deps.Dependencies) (*SwitauthServer, error) {
	// Convert switauth config to base server config
	serverConfig := &server.ServerConfig{
		ServiceName:     "switauth",
		HTTPPort:        cfg.Server.Port,
		GRPCPort:        cfg.Server.GRPCPort,
		EnableHTTP:      true,
		EnableGRPC:      true,
		ShutdownTimeout: 30 * time.Second,
		StartupTimeout:  30 * time.Second,
		Discovery: server.DiscoveryConfig{
			Enabled:             true, // Enable discovery by default
			ServiceName:         "switauth",
			HealthCheckPath:     "/health",
			HealthCheckInterval: 10 * time.Second,
			RetryAttempts:       3,
			RetryInterval:       5 * time.Second,
			Consul: server.ConsulConfig{
				Address: cfg.ServiceDiscovery.Address,
			},
		},
	}

	// Create service registrar
	registrar := NewSwitauthServiceRegistrar(dependencies)

	// Create base server
	baseServer, err := server.NewServer(serverConfig, registrar, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	// Create switauth server wrapper
	switauthServer := &SwitauthServer{
		baseServer: baseServer,
		deps:       dependencies,
		config:     cfg,
	}

	// Register switauth services
	if err := switauthServer.registerServices(); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return switauthServer, nil
}

// registerServices is no longer needed as services are registered during server creation
// This method is kept for backward compatibility
func (s *SwitauthServer) registerServices() error {
	// Services are already registered during server creation
	return nil
}

// Start starts the switauth server
func (s *SwitauthServer) Start(ctx context.Context) error {
	return s.baseServer.Start(ctx)
}

// Stop stops the switauth server
func (s *SwitauthServer) Stop(ctx context.Context) error {
	return s.baseServer.Stop(ctx)
}

// GetHTTPAddress returns the HTTP server address
func (s *SwitauthServer) GetHTTPAddress() string {
	if s.baseServer == nil {
		return ""
	}
	return s.baseServer.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *SwitauthServer) GetGRPCAddress() string {
	if s.baseServer == nil {
		return ""
	}
	return s.baseServer.GetGRPCAddress()
}

// GetRegisteredServices returns information about registered services
// Note: This functionality is not available in the BaseServer interface
func (s *SwitauthServer) GetRegisteredServices() map[string]interface{} {
	if s.baseServer == nil {
		return make(map[string]interface{})
	}

	// Create a map with service information
	services := make(map[string]interface{})
	services["switauth-service"] = map[string]string{
		"http_address": s.GetHTTPAddress(),
		"grpc_address": s.GetGRPCAddress(),
	}
	return services
}

// IsHealthy checks if the server is healthy
// Note: This functionality is not available in the BaseServer interface
func (s *SwitauthServer) IsHealthy(ctx context.Context) bool {
	// For now, we assume the server is healthy if it's running
	// In the future, this could be enhanced with actual health checks
	return true
}
