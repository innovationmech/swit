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
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

// HTTPHandlerRegistrar defines the interface for HTTP route registration
type HTTPHandlerRegistrar interface {
	RegisterHTTP(router *gin.Engine) error
}

// GRPCServiceRegistrar defines the interface for gRPC service registration
type GRPCServiceRegistrar interface {
	RegisterGRPC(server *grpc.Server) error
}

// GenericHTTPHandler provides a generic adapter for HTTP handlers
type GenericHTTPHandler struct {
	serviceName string
	registrar   HTTPHandlerRegistrar
}

// NewGenericHTTPHandler creates a new generic HTTP handler adapter
func NewGenericHTTPHandler(serviceName string, registrar HTTPHandlerRegistrar) *GenericHTTPHandler {
	return &GenericHTTPHandler{
		serviceName: serviceName,
		registrar:   registrar,
	}
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *GenericHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.registrar.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *GenericHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// GenericGRPCService provides a generic adapter for gRPC services
type GenericGRPCService struct {
	serviceName string
	registrar   GRPCServiceRegistrar
}

// NewGenericGRPCService creates a new generic gRPC service adapter
func NewGenericGRPCService(serviceName string, registrar GRPCServiceRegistrar) *GenericGRPCService {
	return &GenericGRPCService{
		serviceName: serviceName,
		registrar:   registrar,
	}
}

// RegisterGRPC registers the gRPC service with the provided server
func (s *GenericGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", server)
	}

	return s.registrar.RegisterGRPC(grpcServer)
}

// GetServiceName returns the service name for identification
func (s *GenericGRPCService) GetServiceName() string {
	return s.serviceName
}

// HealthCheckFunc defines a function type for health checks
type HealthCheckFunc func(ctx context.Context) error

// GenericHealthCheck provides a generic health check implementation
type GenericHealthCheck struct {
	serviceName string
	checkFunc   HealthCheckFunc
	startTime   time.Time
}

// NewGenericHealthCheck creates a new generic health check
func NewGenericHealthCheck(serviceName string, checkFunc HealthCheckFunc) *GenericHealthCheck {
	return &GenericHealthCheck{
		serviceName: serviceName,
		checkFunc:   checkFunc,
		startTime:   time.Now(),
	}
}

// Check performs the health check and returns the status
func (h *GenericHealthCheck) Check(ctx context.Context) error {
	if h.checkFunc == nil {
		// Default to healthy if no check function is provided
		return nil
	}
	return h.checkFunc(ctx)
}

// GetServiceName returns the service name for the health check
func (h *GenericHealthCheck) GetServiceName() string {
	return h.serviceName
}

// GetUptime returns the uptime since the health check was created
func (h *GenericHealthCheck) GetUptime() time.Duration {
	return time.Since(h.startTime)
}

// ServiceAvailabilityChecker creates a health check function that verifies service availability
func ServiceAvailabilityChecker(service interface{}) HealthCheckFunc {
	return func(ctx context.Context) error {
		if service == nil {
			return fmt.Errorf("service is not available")
		}
		return nil
	}
}

// ConfigMapperBase provides common configuration mapping utilities
type ConfigMapperBase struct{}

// NewConfigMapperBase creates a new base configuration mapper
func NewConfigMapperBase() *ConfigMapperBase {
	return &ConfigMapperBase{}
}

// SetDefaultHTTPConfig sets default HTTP configuration values
func (m *ConfigMapperBase) SetDefaultHTTPConfig(config *ServerConfig, port, defaultPort string, enableReady bool) {
	if port != "" {
		config.HTTP.Port = port
		config.HTTP.Address = ":" + port
	} else {
		config.HTTP.Port = defaultPort
		config.HTTP.Address = ":" + defaultPort
	}
	config.HTTP.Enabled = true
	config.HTTP.EnableReady = enableReady
}

// SetDefaultGRPCConfig sets default gRPC configuration values
func (m *ConfigMapperBase) SetDefaultGRPCConfig(config *ServerConfig, port, defaultPort string, enableKeepalive, enableReflection, enableHealthService bool) {
	if port != "" {
		config.GRPC.Port = port
		config.GRPC.Address = ":" + port
	} else {
		config.GRPC.Port = defaultPort
		config.GRPC.Address = ":" + defaultPort
	}
	config.GRPC.Enabled = true
	config.GRPC.EnableKeepalive = enableKeepalive
	config.GRPC.EnableReflection = enableReflection
	config.GRPC.EnableHealthService = enableHealthService
}

// SetDefaultDiscoveryConfig sets default service discovery configuration
func (m *ConfigMapperBase) SetDefaultDiscoveryConfig(config *ServerConfig, address, serviceName string, tags []string) {
	if address != "" {
		config.Discovery.Address = address
	}
	config.Discovery.ServiceName = serviceName
	config.Discovery.Tags = tags
	config.Discovery.Enabled = true
}

// SetDefaultMiddlewareConfig sets default middleware configuration
func (m *ConfigMapperBase) SetDefaultMiddlewareConfig(config *ServerConfig, enableCORS, enableAuth, enableRateLimit, enableLogging bool) {
	// Global middleware configuration
	config.Middleware.EnableCORS = enableCORS
	config.Middleware.EnableAuth = enableAuth
	config.Middleware.EnableRateLimit = enableRateLimit
	config.Middleware.EnableLogging = enableLogging

	// HTTP-specific middleware configuration
	config.HTTP.Middleware.EnableCORS = enableCORS
	config.HTTP.Middleware.EnableAuth = enableAuth
	config.HTTP.Middleware.EnableRateLimit = enableRateLimit
	config.HTTP.Middleware.EnableLogging = enableLogging
}

// SetDefaultServerConfig sets common server configuration defaults
func (m *ConfigMapperBase) SetDefaultServerConfig(config *ServerConfig, serviceName string) {
	config.ServiceName = serviceName
	config.ShutdownTimeout = 5 * time.Second
}

// GenericDependencyContainer provides a generic dependency container implementation
type GenericDependencyContainer struct {
	services  map[string]interface{}
	closeFunc func() error
	closed    bool
}

// NewGenericDependencyContainer creates a new generic dependency container
func NewGenericDependencyContainer(closeFunc func() error) *GenericDependencyContainer {
	return &GenericDependencyContainer{
		services:  make(map[string]interface{}),
		closeFunc: closeFunc,
		closed:    false,
	}
}

// RegisterService registers a service with the container
func (d *GenericDependencyContainer) RegisterService(name string, service interface{}) {
	d.services[name] = service
}

// GetService retrieves a service by name from the container
func (d *GenericDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// Close closes the dependency container and cleans up resources
func (d *GenericDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	var err error
	if d.closeFunc != nil {
		err = d.closeFunc()
	}

	d.closed = true
	return err
}

// ServiceRegistrationHelper provides utilities for service registration
type ServiceRegistrationHelper struct {
	registry BusinessServiceRegistry
}

// NewServiceRegistrationHelper creates a new service registration helper
func NewServiceRegistrationHelper(registry BusinessServiceRegistry) *ServiceRegistrationHelper {
	return &ServiceRegistrationHelper{
		registry: registry,
	}
}

// RegisterHTTPHandlerWithHealthCheck registers an HTTP handler along with its health check
func (h *ServiceRegistrationHelper) RegisterHTTPHandlerWithHealthCheck(
	serviceName string,
	registrar HTTPHandlerRegistrar,
	healthCheckFunc HealthCheckFunc,
) error {
	// Register HTTP handler
	httpHandler := NewGenericHTTPHandler(serviceName, registrar)
	if err := h.registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler for %s: %w", serviceName, err)
	}

	// Register health check
	healthCheck := NewGenericHealthCheck(serviceName, healthCheckFunc)
	if err := h.registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check for %s: %w", serviceName, err)
	}

	return nil
}

// RegisterGRPCServiceWithHealthCheck registers a gRPC service along with its health check
func (h *ServiceRegistrationHelper) RegisterGRPCServiceWithHealthCheck(
	serviceName string,
	registrar GRPCServiceRegistrar,
	healthCheckFunc HealthCheckFunc,
) error {
	// Register gRPC service
	grpcService := NewGenericGRPCService(serviceName, registrar)
	if err := h.registry.RegisterBusinessGRPCService(grpcService); err != nil {
		return fmt.Errorf("failed to register gRPC service for %s: %w", serviceName, err)
	}

	// Register health check
	healthCheck := NewGenericHealthCheck(serviceName, healthCheckFunc)
	if err := h.registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check for %s: %w", serviceName, err)
	}

	return nil
}
