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

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
)

// BaseServer defines the core interface for all server implementations
type BaseServer interface {
	// Start initializes and starts the server with the given context
	Start(ctx context.Context) error

	// Stop gracefully stops the server with the given context
	Stop(ctx context.Context) error

	// Shutdown performs immediate shutdown of the server
	Shutdown() error

	// GetHTTPAddress returns the HTTP server address
	GetHTTPAddress() string

	// GetGRPCAddress returns the gRPC server address
	GetGRPCAddress() string

	// GetTransports returns all registered transports
	GetTransports() []transport.Transport

	// GetServiceName returns the service name
	GetServiceName() string
}

// ServiceRegistrar defines the interface for registering services with the server
type ServiceRegistrar interface {
	// RegisterServices registers all service handlers with the provided registry
	RegisterServices(registry ServiceRegistry) error
}

// ServiceRegistry defines the interface for service registration
type ServiceRegistry interface {
	// RegisterHTTPHandler registers an HTTP handler
	RegisterHTTPHandler(handler HTTPHandler) error

	// RegisterGRPCService registers a gRPC service
	RegisterGRPCService(service GRPCService) error

	// RegisterHealthCheck registers a health check
	RegisterHealthCheck(check HealthCheck) error
}

// HTTPHandler defines the interface for HTTP service handlers
type HTTPHandler interface {
	// RegisterRoutes registers HTTP routes with the given router
	RegisterRoutes(router gin.IRouter) error

	// GetServiceName returns the service name
	GetServiceName() string

	// GetVersion returns the service version
	GetVersion() string
}

// GRPCService defines the interface for gRPC service handlers
type GRPCService interface {
	// RegisterService registers the gRPC service with the given server
	RegisterService(server *grpc.Server) error

	// GetServiceName returns the service name
	GetServiceName() string

	// GetServiceDesc returns the service descriptor
	GetServiceDesc() *grpc.ServiceDesc
}

// HealthCheck defines the interface for health check implementations
type HealthCheck interface {
	// Check performs the health check and returns the status
	Check(ctx context.Context) *types.HealthStatus

	// GetName returns the health check name
	GetName() string
}

// DependencyContainer defines the interface for dependency injection
type DependencyContainer interface {
	// Initialize initializes the container
	Initialize(ctx context.Context) error

	// Cleanup cleans up the container
	Cleanup(ctx context.Context) error

	// Get retrieves a dependency by name
	Get(name string) (interface{}, error)

	// Register registers a dependency
	Register(name string, dependency interface{}) error
}

// LifecycleHook defines the interface for server lifecycle hooks
type LifecycleHook interface {
	// OnStarting is called before server starts
	OnStarting(ctx context.Context) error

	// OnStarted is called after server starts successfully
	OnStarted(ctx context.Context) error

	// OnStopping is called before server stops
	OnStopping(ctx context.Context) error

	// OnStopped is called after server stops
	OnStopped(ctx context.Context) error
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates the configuration and returns any errors
	Validate() error

	// SetDefaults sets default values for unspecified configuration
	SetDefaults()
}
