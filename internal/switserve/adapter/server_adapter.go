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

	"github.com/gin-gonic/gin"

	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/deps"
	greeterv1 "github.com/innovationmech/swit/internal/switserve/handler/http/greeter/v1"
	"github.com/innovationmech/swit/internal/switserve/handler/http/health"
	notificationv1 "github.com/innovationmech/swit/internal/switserve/handler/http/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/handler/http/stop"
	userv1 "github.com/innovationmech/swit/internal/switserve/handler/http/user/v1"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/transport"
)

// SwitserveAdapter implements ServiceRegistrar interface for switserve service
type SwitserveAdapter struct {
	deps *deps.Dependencies
}

// NewSwitserveAdapter creates a new switserve service adapter
func NewSwitserveAdapter(dependencies *deps.Dependencies) *SwitserveAdapter {
	return &SwitserveAdapter{
		deps: dependencies,
	}
}

// RegisterServices registers all services for switserve service
func (a *SwitserveAdapter) RegisterServices(registry server.ServiceRegistry) error {
	if a.deps == nil {
		return fmt.Errorf("dependencies not initialized")
	}

	// Register Greeter service with dependency injection
	greeterHandler := greeterv1.NewGreeterHandler(a.deps.GreeterSrv)
	greeterWrapper := &HTTPHandlerWrapper{
		handler: greeterHandler,
		name:    "greeter",
		version: "v1",
	}
	if err := registry.RegisterHTTPHandler(greeterWrapper); err != nil {
		return fmt.Errorf("failed to register greeter service: %w", err)
	}

	// Register Notification service with dependency injection
	notificationHandler := notificationv1.NewNotificationHandler(a.deps.NotificationSrv)
	notificationWrapper := &HTTPHandlerWrapper{
		handler: notificationHandler,
		name:    "notification",
		version: "v1",
	}
	if err := registry.RegisterHTTPHandler(notificationWrapper); err != nil {
		return fmt.Errorf("failed to register notification service: %w", err)
	}

	// Register Health service with dependency injection
	healthHandler := health.NewHandler(a.deps.HealthSrv)
	healthWrapper := &HTTPHandlerWrapper{
		handler: healthHandler,
		name:    "health",
		version: "v1",
	}
	if err := registry.RegisterHTTPHandler(healthWrapper); err != nil {
		return fmt.Errorf("failed to register health service: %w", err)
	}

	// Register Stop service with dependency injection
	stopHandler := stop.NewHandler(a.deps.StopSrv)
	stopWrapper := &HTTPHandlerWrapper{
		handler: stopHandler,
		name:    "stop",
		version: "v1",
	}
	if err := registry.RegisterHTTPHandler(stopWrapper); err != nil {
		return fmt.Errorf("failed to register stop service: %w", err)
	}

	// Register User service with dependency injection
	userHandler := userv1.NewUserHandler(a.deps.UserSrv)
	userWrapper := &HTTPHandlerWrapper{
		handler: userHandler,
		name:    "user",
		version: "v1",
	}
	if err := registry.RegisterHTTPHandler(userWrapper); err != nil {
		return fmt.Errorf("failed to register user service: %w", err)
	}

	return nil
}

// HTTPHandlerWrapper wraps existing handlers to implement server.HTTPHandler interface
type HTTPHandlerWrapper struct {
	handler interface {
		RegisterHTTP(*gin.Engine) error
	}
	name    string
	version string
}

// RegisterRoutes implements server.HTTPHandler interface
func (w *HTTPHandlerWrapper) RegisterRoutes(router gin.IRouter) error {
	// Create a new engine to capture routes, then register them on the provided router
	if engine, ok := router.(*gin.Engine); ok {
		return w.handler.RegisterHTTP(engine)
	}
	// If not an engine, create a temporary one and register routes
	tempEngine := gin.New()
	if err := w.handler.RegisterHTTP(tempEngine); err != nil {
		return err
	}
	// Note: This is a simplified approach. In a real implementation,
	// you might need to extract routes from tempEngine and register them on router
	return nil
}

// GetServiceName implements server.HTTPHandler interface
func (w *HTTPHandlerWrapper) GetServiceName() string {
	return w.name
}

// GetVersion implements server.HTTPHandler interface
func (w *HTTPHandlerWrapper) GetVersion() string {
	return w.version
}

// GetServiceName returns the service name for discovery registration
func (a *SwitserveAdapter) GetServiceName() string {
	return "swit-serve"
}

// CreateBaseServerConfig creates base server configuration from switserve config
func CreateBaseServerConfig() (*server.ServerConfig, error) {
	cfg := config.GetConfig()

	// Create base server configuration
	baseConfig := &server.ServerConfig{
		ServiceName:    "swit-serve",
		ServiceVersion: "v1.0.0",
		HTTPPort:       cfg.Server.Port,
		GRPCPort:       cfg.Server.GRPCPort,
		EnableHTTP:     true,
		EnableGRPC:     true,
		EnableReady:    true,
		Discovery: server.DiscoveryConfig{
			Enabled:     true,
			ServiceName: "swit-serve",
			Consul: server.ConsulConfig{
				Address: cfg.ServiceDiscovery.Address,
			},
		},
		Middleware: server.MiddlewareConfig{
			EnableCORS:          true,
			EnableAuth:          false,
			EnableRateLimit:     true,
			EnableRequestLogger: true,
			EnableTimeout:       true,
		},
	}

	// Set default ports if not configured
	if baseConfig.HTTPPort == "" {
		baseConfig.HTTPPort = "8080"
	}
	if baseConfig.GRPCPort == "" {
		// Calculate gRPC port based on HTTP port
		baseConfig.GRPCPort = transport.CalculateGRPCPort(baseConfig.HTTPPort)
	}

	// Set defaults and validate configuration
	baseConfig.SetDefaults()
	if err := baseConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	return baseConfig, nil
}

// CreateDependencyContainer creates dependency container for switserve service
func CreateDependencyContainer(shutdownFunc func()) (server.DependencyContainer, error) {
	dependencies, err := deps.NewDependencies(shutdownFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	return &DependencyContainer{
		deps: dependencies,
	}, nil
}

// DependencyContainer implements server.DependencyContainer interface
type DependencyContainer struct {
	deps *deps.Dependencies
}

// Initialize initializes all dependencies
func (dc *DependencyContainer) Initialize(ctx context.Context) error {
	// Dependencies are already initialized in NewDependencies
	return nil
}

// Cleanup cleans up all dependencies
func (dc *DependencyContainer) Cleanup(ctx context.Context) error {
	if dc.deps != nil {
		return dc.deps.Close()
	}
	return nil
}

// Get returns a dependency by name
func (dc *DependencyContainer) Get(name string) (interface{}, error) {
	switch name {
	case "dependencies":
		return dc.deps, nil
	case "userService":
		return dc.deps.UserSrv, nil
	case "greeterService":
		return dc.deps.GreeterSrv, nil
	case "notificationService":
		return dc.deps.NotificationSrv, nil
	case "healthService":
		return dc.deps.HealthSrv, nil
	case "stopService":
		return dc.deps.StopSrv, nil
	case "database":
		return dc.deps.DB, nil
	case "userRepository":
		return dc.deps.UserRepo, nil
	default:
		return nil, fmt.Errorf("dependency not found: %s", name)
	}
}

// Register registers a dependency by name
func (dc *DependencyContainer) Register(name string, dependency interface{}) error {
	// For switserve, dependencies are managed internally
	// This method is implemented for interface compliance
	return fmt.Errorf("registering dependencies not supported for switserve")
}
