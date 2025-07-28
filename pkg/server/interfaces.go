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
	"time"

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
)

// TransportStatus represents the status of a transport
type TransportStatus struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Running bool   `json:"running"`
}

// BusinessServerCore defines the core interface for all server implementations
// It provides consistent lifecycle management and transport access
type BusinessServerCore interface {
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
	GetTransports() []transport.NetworkTransport
	// GetTransportStatus returns the status of all transports
	GetTransportStatus() map[string]TransportStatus
	// GetTransportHealth returns health status of all services across all transports
	GetTransportHealth(ctx context.Context) map[string]map[string]*types.HealthStatus
}

// BusinessServerWithPerformance extends BusinessServerCore with performance monitoring capabilities
type BusinessServerWithPerformance interface {
	BusinessServerCore
	// GetPerformanceMetrics returns current performance metrics
	GetPerformanceMetrics() *PerformanceMetrics
	// GetUptime returns the server uptime
	GetUptime() time.Duration
	// GetPerformanceMonitor returns the performance monitor instance
	GetPerformanceMonitor() *PerformanceMonitor
}

// BusinessServiceRegistrar defines the interface for services to register themselves
// with the server's transport layer
type BusinessServiceRegistrar interface {
	// RegisterServices registers all service handlers with the provided registry
	RegisterServices(registry BusinessServiceRegistry) error
}

// BusinessServiceRegistry defines the interface for registering different types of services
// It abstracts the underlying transport registration mechanisms
type BusinessServiceRegistry interface {
	// RegisterBusinessHTTPHandler registers an HTTP service handler
	RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error
	// RegisterBusinessGRPCService registers a gRPC service
	RegisterBusinessGRPCService(service BusinessGRPCService) error
	// RegisterBusinessHealthCheck registers a health check for a service
	RegisterBusinessHealthCheck(check BusinessHealthCheck) error
}

// BusinessHTTPHandler defines the interface for HTTP service handlers
type BusinessHTTPHandler interface {
	// RegisterRoutes registers HTTP routes with the provided router
	RegisterRoutes(router interface{}) error
	// GetServiceName returns the service name for identification
	GetServiceName() string
}

// BusinessGRPCService defines the interface for gRPC service implementations
type BusinessGRPCService interface {
	// RegisterGRPC registers the gRPC service with the provided server
	RegisterGRPC(server interface{}) error
	// GetServiceName returns the service name for identification
	GetServiceName() string
}

// BusinessHealthCheck defines the interface for service health checks
type BusinessHealthCheck interface {
	// Check performs the health check and returns the status
	Check(ctx context.Context) error
	// GetServiceName returns the service name for the health check
	GetServiceName() string
}

// BusinessDependencyContainer defines the interface for dependency injection containers
// It provides access to service dependencies and manages their lifecycle
type BusinessDependencyContainer interface {
	// Close closes all managed dependencies and cleans up resources
	Close() error
	// GetService retrieves a service by name from the container
	GetService(name string) (interface{}, error)
}

// BusinessDependencyRegistry extends BusinessDependencyContainer with additional functionality
// for registration and lifecycle management
type BusinessDependencyRegistry interface {
	BusinessDependencyContainer
	// Initialize initializes all registered dependencies
	Initialize(ctx context.Context) error
	// RegisterSingleton registers a singleton dependency
	RegisterSingleton(name string, factory DependencyFactory) error
	// RegisterTransient registers a transient dependency
	RegisterTransient(name string, factory DependencyFactory) error
	// RegisterInstance registers a pre-created instance
	RegisterInstance(name string, instance interface{}) error
	// GetDependencyNames returns all registered dependency names
	GetDependencyNames() []string
	// IsInitialized returns true if the container has been initialized
	IsInitialized() bool
	// IsClosed returns true if the container has been closed
	IsClosed() bool
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates the configuration and returns any errors
	Validate() error
	// SetDefaults sets default values for unspecified configuration options
	SetDefaults()
}
