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

package server

import (
	"fmt"

	"github.com/innovationmech/swit/pkg/server"
)

// ServerFactory provides factory methods for creating configured server instances
// It encapsulates server creation logic and provides validation for factory parameters
type ServerFactory struct {
	config *server.ServerConfig
}

// NewServerFactory creates a new ServerFactory with the provided configuration
// The configuration is validated during factory creation to fail fast on invalid configs
func NewServerFactory(config *server.ServerConfig) (*ServerFactory, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	// Validate configuration during factory creation
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	return &ServerFactory{
		config: config,
	}, nil
}

// CreateServer creates a new base server instance with the provided service registrar and dependencies
// This is the primary factory method for creating configured server instances
func (f *ServerFactory) CreateServer(registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) (server.BusinessServerCore, error) {
	if registrar == nil {
		return nil, fmt.Errorf("service registrar cannot be nil")
	}

	// Create the base server with validated configuration
	baseServer, err := server.NewBusinessServerCore(f.config, registrar, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}

	return baseServer, nil
}

// CreateHTTPOnlyServer creates a server instance with only HTTP transport enabled
// This is a convenience method for services that only need HTTP functionality
func (f *ServerFactory) CreateHTTPOnlyServer(registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) (server.BusinessServerCore, error) {
	if registrar == nil {
		return nil, fmt.Errorf("service registrar cannot be nil")
	}

	// Create a deep copy of the config with only HTTP enabled
	config := f.config.DeepCopy()
	config.HTTP.Enabled = true
	config.GRPC.Enabled = false

	// Validate the modified configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid HTTP-only configuration: %w", err)
	}

	// Create the base server with HTTP-only configuration
	baseServer, err := server.NewBusinessServerCore(config, registrar, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP-only server: %w", err)
	}

	return baseServer, nil
}

// CreateGRPCOnlyServer creates a server instance with only gRPC transport enabled
// This is a convenience method for services that only need gRPC functionality
func (f *ServerFactory) CreateGRPCOnlyServer(registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) (server.BusinessServerCore, error) {
	if registrar == nil {
		return nil, fmt.Errorf("service registrar cannot be nil")
	}

	// Create a deep copy of the config with only gRPC enabled
	config := f.config.DeepCopy()
	config.HTTP.Enabled = false
	config.GRPC.Enabled = true

	// Validate the modified configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid gRPC-only configuration: %w", err)
	}

	// Create the base server with gRPC-only configuration
	baseServer, err := server.NewBusinessServerCore(config, registrar, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC-only server: %w", err)
	}

	return baseServer, nil
}

// CreateTestServer creates a server instance configured for testing
// This method enables test mode and uses test ports to avoid conflicts
func (f *ServerFactory) CreateTestServer(registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) (server.BusinessServerCore, error) {
	if registrar == nil {
		return nil, fmt.Errorf("service registrar cannot be nil")
	}

	// Create a deep copy of the config with test mode enabled
	config := f.config.DeepCopy()
	config.HTTP.TestMode = true
	config.GRPC.TestMode = true
	config.Discovery.Enabled = false // Disable discovery in test mode

	// Use test ports if configured
	if config.HTTP.TestPort != "" {
		config.HTTP.Port = config.HTTP.TestPort
		config.HTTP.Address = ":" + config.HTTP.TestPort
	}
	if config.GRPC.TestPort != "" {
		config.GRPC.Port = config.GRPC.TestPort
		config.GRPC.Address = ":" + config.GRPC.TestPort
	}

	// Validate the test configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid test configuration: %w", err)
	}

	// Create the base server with test configuration
	baseServer, err := server.NewBusinessServerCore(config, registrar, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create test server: %w", err)
	}

	return baseServer, nil
}

// CreateServerWithCustomConfig creates a server instance with a custom configuration
// This method allows for complete configuration customization while maintaining validation
func (f *ServerFactory) CreateServerWithCustomConfig(config *server.ServerConfig, registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) (server.BusinessServerCore, error) {
	if config == nil {
		return nil, fmt.Errorf("custom config cannot be nil")
	}

	if registrar == nil {
		return nil, fmt.Errorf("service registrar cannot be nil")
	}

	// Validate the custom configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid custom configuration: %w", err)
	}

	// Create the base server with custom configuration
	baseServer, err := server.NewBusinessServerCore(config, registrar, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create server with custom config: %w", err)
	}

	return baseServer, nil
}

// GetConfig returns a copy of the factory's configuration
// This allows inspection of the factory configuration without modification
func (f *ServerFactory) GetConfig() *server.ServerConfig {
	// Return a copy to prevent external modification
	config := *f.config
	return &config
}

// ValidateFactoryParameters validates the parameters commonly used in factory methods
// This is a utility method for consistent parameter validation across factory methods
func (f *ServerFactory) ValidateFactoryParameters(registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) error {
	if registrar == nil {
		return fmt.Errorf("service registrar cannot be nil")
	}

	// Dependencies are optional, so we don't validate them as required
	// The base server will handle nil dependencies appropriately

	return nil
}
