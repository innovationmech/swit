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
	"context"

	"github.com/innovationmech/swit/pkg/transport"
)

// BaseServer defines the core interface for all server implementations
// It provides consistent lifecycle management and transport access
type BaseServer interface {
	// Start starts the server with all registered services
	Start(ctx context.Context) error
	// Stop gracefully stops the server
	Stop(ctx context.Context) error
	// Shutdown performs complete server shutdown with resource cleanup
	Shutdown() error
	// GetHTTPAddress returns the HTTP server listening address
	GetHTTPAddress() string
	// GetGRPCAddress returns the gRPC server listening address
	GetGRPCAddress() string
	// GetTransports returns all registered transports
	GetTransports() []transport.Transport
}

// ServiceRegistrar defines the interface for services to register themselves
// with the server's transport layer
type ServiceRegistrar interface {
	// RegisterServices registers all service handlers with the provided registry
	RegisterServices(registry ServiceRegistry) error
}

// ServiceRegistry defines the interface for registering different types of services
// It abstracts the underlying transport registration mechanisms
type ServiceRegistry interface {
	// RegisterHTTPHandler registers an HTTP service handler
	RegisterHTTPHandler(handler HTTPHandler) error
	// RegisterGRPCService registers a gRPC service
	RegisterGRPCService(service GRPCService) error
	// RegisterHealthCheck registers a health check for a service
	RegisterHealthCheck(check HealthCheck) error
}

// HTTPHandler defines the interface for HTTP service handlers
type HTTPHandler interface {
	// RegisterRoutes registers HTTP routes with the provided router
	RegisterRoutes(router interface{}) error
	// GetServiceName returns the service name for identification
	GetServiceName() string
}

// GRPCService defines the interface for gRPC service implementations
type GRPCService interface {
	// RegisterGRPC registers the gRPC service with the provided server
	RegisterGRPC(server interface{}) error
	// GetServiceName returns the service name for identification
	GetServiceName() string
}

// HealthCheck defines the interface for service health checks
type HealthCheck interface {
	// Check performs the health check and returns the status
	Check(ctx context.Context) error
	// GetServiceName returns the service name for the health check
	GetServiceName() string
}

// DependencyContainer defines the interface for dependency injection containers
// It provides access to service dependencies and manages their lifecycle
type DependencyContainer interface {
	// Close closes all managed dependencies and cleans up resources
	Close() error
	// GetService retrieves a service by name from the container
	GetService(name string) (interface{}, error)
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates the configuration and returns any errors
	Validate() error
	// SetDefaults sets default values for unspecified configuration options
	SetDefaults()
}
