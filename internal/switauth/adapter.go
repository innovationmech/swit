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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	auth "github.com/innovationmech/swit/internal/switauth/handler/http/auth/v1"
	"github.com/innovationmech/swit/internal/switauth/handler/http/health"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements the server.BusinessServiceRegistrar interface for switauth
// It registers all switauth services with the base server framework
type ServiceRegistrar struct {
	deps *deps.Dependencies
}

// NewServiceRegistrar creates a new ServiceRegistrar for switauth
func NewServiceRegistrar(dependencies *deps.Dependencies) *ServiceRegistrar {
	return &ServiceRegistrar{
		deps: dependencies,
	}
}

// RegisterServices registers all switauth services with the provided registry
func (s *ServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	logger.Logger.Info("Registering switauth services with base server")

	// Register authentication HTTP handler
	authHandler := &AuthBusinessHTTPHandler{
		handler: auth.NewAuthHandler(s.deps.AuthSrv),
	}
	if err := registry.RegisterBusinessHTTPHandler(authHandler); err != nil {
		return fmt.Errorf("failed to register auth HTTP handler: %w", err)
	}

	// Register health HTTP handler
	healthHandler := &HealthBusinessHTTPHandler{
		handler: health.NewHandler(),
	}
	if err := registry.RegisterBusinessHTTPHandler(healthHandler); err != nil {
		return fmt.Errorf("failed to register health HTTP handler: %w", err)
	}

	// Register health checks
	authBusinessHealthCheck := &AuthBusinessHealthCheck{
		authService: s.deps.AuthSrv,
		startTime:   time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(authBusinessHealthCheck); err != nil {
		return fmt.Errorf("failed to register auth health check: %w", err)
	}

	healthCheck := &HealthServiceCheck{
		startTime: time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health service check: %w", err)
	}

	logger.Logger.Info("Successfully registered all switauth services")
	return nil
}

// RegisterEventHandlers implements server.MessagingServiceRegistrar to support event handler registration.
// switauth currently does not expose messaging handlers; this is a no-op implementation for future use.
func (s *ServiceRegistrar) RegisterEventHandlers(registry server.EventHandlerRegistry) error {
    if registry == nil {
        return fmt.Errorf("event handler registry cannot be nil")
    }
    return nil
}

// GetEventHandlerMetadata returns minimal metadata for messaging capabilities of switauth.
func (s *ServiceRegistrar) GetEventHandlerMetadata() *server.EventHandlerMetadata {
    return &server.EventHandlerMetadata{
        HandlerCount:       0,
        Topics:             nil,
        BrokerRequirements: nil,
        Description:        "switauth has no messaging handlers currently",
    }
}

// AuthBusinessHTTPHandler adapts the switauth auth handler to the base server BusinessHTTPHandler interface
type AuthBusinessHTTPHandler struct {
	handler *auth.AuthHandler
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *AuthBusinessHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.handler.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *AuthBusinessHTTPHandler) GetServiceName() string {
	return "auth-service"
}

// HealthBusinessHTTPHandler adapts the switauth health handler to the base server BusinessHTTPHandler interface
type HealthBusinessHTTPHandler struct {
	handler *health.Handler
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *HealthBusinessHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.handler.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *HealthBusinessHTTPHandler) GetServiceName() string {
	return "health-service"
}

// AuthBusinessHealthCheck implements the base server BusinessHealthCheck interface for auth service
type AuthBusinessHealthCheck struct {
	authService interface{}
	startTime   time.Time
}

// Check performs the health check and returns the status
func (h *AuthBusinessHealthCheck) Check(ctx context.Context) error {
	if h.authService == nil {
		return fmt.Errorf("auth service is not available")
	}
	return nil
}

// GetServiceName returns the service name for the health check
func (h *AuthBusinessHealthCheck) GetServiceName() string {
	return "auth-service"
}

// HealthServiceCheck implements the base server BusinessHealthCheck interface for health service
type HealthServiceCheck struct {
	startTime time.Time
}

// Check performs the health check and returns the status
func (h *HealthServiceCheck) Check(ctx context.Context) error {
	// Health service is always healthy if it can respond
	return nil
}

// GetServiceName returns the service name for the health check
func (h *HealthServiceCheck) GetServiceName() string {
	return "health-service"
}

// BusinessDependencyContainer adapts switauth dependencies to the base server BusinessDependencyContainer interface
type BusinessDependencyContainer struct {
	deps   *deps.Dependencies
	closed bool
}

// NewBusinessDependencyContainer creates a new BusinessDependencyContainer for switauth
func NewBusinessDependencyContainer(dependencies *deps.Dependencies) *BusinessDependencyContainer {
	return &BusinessDependencyContainer{
		deps:   dependencies,
		closed: false,
	}
}

// Close closes all managed dependencies and cleans up resources
func (d *BusinessDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	logger.Logger.Info("Closing switauth dependencies")

	// Close database connection if available
	if d.deps.DB != nil {
		if sqlDB, err := d.deps.DB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Logger.Error("Failed to close database connection", zap.Error(err))
			}
		}
	}

	d.closed = true
	logger.Logger.Info("Successfully closed switauth dependencies")
	return nil
}

// GetService retrieves a service by name from the container
func (d *BusinessDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	switch name {
	case "auth-service":
		return d.deps.AuthSrv, nil
	case "health-service":
		return d.deps.HealthSrv, nil
	case "database":
		return d.deps.DB, nil
	case "config":
		return d.deps.Config, nil
	case "service-discovery":
		return d.deps.SD, nil
	case "token-repository":
		return d.deps.TokenRepo, nil
	case "user-client":
		return d.deps.UserClient, nil
	default:
		return nil, fmt.Errorf("service %s not found", name)
	}
}

// ConfigMapper maps switauth configuration to base server configuration
type ConfigMapper struct {
	authConfig *config.AuthConfig
}

// NewConfigMapper creates a new ConfigMapper for switauth
func NewConfigMapper(authConfig *config.AuthConfig) *ConfigMapper {
	return &ConfigMapper{
		authConfig: authConfig,
	}
}

// ToServerConfig converts AuthConfig to ServerConfig
func (m *ConfigMapper) ToServerConfig() *server.ServerConfig {
	serverConfig := server.NewServerConfig()

	// Set service name
	serverConfig.ServiceName = "swit-auth"

	// Map HTTP configuration
	if m.authConfig.Server.Port != "" {
		serverConfig.HTTP.Port = m.authConfig.Server.Port
		serverConfig.HTTP.Address = ":" + m.authConfig.Server.Port
	} else {
		serverConfig.HTTP.Port = "9001"
		serverConfig.HTTP.Address = ":9001"
	}
	serverConfig.HTTP.Enabled = true
	serverConfig.HTTP.EnableReady = false // Keep existing behavior

	// Map gRPC configuration
	if m.authConfig.Server.GRPCPort != "" {
		serverConfig.GRPC.Port = m.authConfig.Server.GRPCPort
		serverConfig.GRPC.Address = ":" + m.authConfig.Server.GRPCPort
	} else {
		serverConfig.GRPC.Port = "50051"
		serverConfig.GRPC.Address = ":50051"
	}
	serverConfig.GRPC.Enabled = true
	serverConfig.GRPC.EnableKeepalive = false    // Keep existing behavior
	serverConfig.GRPC.EnableReflection = true    // Keep existing behavior
	serverConfig.GRPC.EnableHealthService = true // Keep existing behavior

	// Map service discovery configuration
	if m.authConfig.ServiceDiscovery.Address != "" {
		serverConfig.Discovery.Address = m.authConfig.ServiceDiscovery.Address
	}
	serverConfig.Discovery.ServiceName = "swit-auth"
	serverConfig.Discovery.Tags = []string{"auth", "v1"}
	serverConfig.Discovery.Enabled = true

	// Set middleware configuration to match existing behavior
	serverConfig.Middleware.EnableCORS = true
	serverConfig.Middleware.EnableAuth = false
	serverConfig.Middleware.EnableRateLimit = true
	serverConfig.Middleware.EnableLogging = true

	// Set HTTP middleware configuration
	serverConfig.HTTP.Middleware.EnableCORS = true
	serverConfig.HTTP.Middleware.EnableAuth = false
	serverConfig.HTTP.Middleware.EnableRateLimit = true
	serverConfig.HTTP.Middleware.EnableLogging = true

	// Set shutdown timeout
	serverConfig.ShutdownTimeout = 5 * time.Second

	return serverConfig
}

// AuthBusinessGRPCService adapts switauth gRPC services to the base server BusinessGRPCService interface
// Currently placeholder as switauth doesn't have gRPC services implemented
type AuthBusinessGRPCService struct {
	serviceName string
}

// NewAuthBusinessGRPCService creates a new AuthBusinessGRPCService
func NewAuthBusinessGRPCService() *AuthBusinessGRPCService {
	return &AuthBusinessGRPCService{
		serviceName: "auth-grpc-service",
	}
}

// RegisterGRPC registers the gRPC service with the provided server
func (s *AuthBusinessGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", server)
	}

	logger.Logger.Info("Registering auth gRPC service", zap.String("service", s.serviceName))

	// TODO: Register actual gRPC services when implemented
	// For now, just log that gRPC registration was called
	_ = grpcServer // Use the server to avoid unused variable warning

	return nil
}

// GetServiceName returns the service name for identification
func (s *AuthBusinessGRPCService) GetServiceName() string {
	return s.serviceName
}
