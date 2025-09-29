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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/deps"
	greeterv1 "github.com/innovationmech/swit/internal/switserve/handler/http/greeter/v1"
	"github.com/innovationmech/swit/internal/switserve/handler/http/health"
	notificationv1 "github.com/innovationmech/swit/internal/switserve/handler/http/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/handler/http/stop"
	userv1 "github.com/innovationmech/swit/internal/switserve/handler/http/user/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements the server.BusinessServiceRegistrar interface for switserve
// It registers all switserve services with the base server framework
type ServiceRegistrar struct {
	deps *deps.Dependencies
}

// NewServiceRegistrar creates a new ServiceRegistrar for switserve
func NewServiceRegistrar(dependencies *deps.Dependencies) *ServiceRegistrar {
	return &ServiceRegistrar{
		deps: dependencies,
	}
}

// RegisterServices registers all switserve services with the provided registry
func (s *ServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	logger.Logger.Info("Registering switserve services with base server")

	// Register greeter HTTP handler
	greeterHandler := &GreeterBusinessHTTPHandler{
		handler: greeterv1.NewGreeterHandler(s.deps.GreeterSrv),
	}
	if err := registry.RegisterBusinessHTTPHandler(greeterHandler); err != nil {
		return fmt.Errorf("failed to register greeter HTTP handler: %w", err)
	}

	// Register notification HTTP handler
	notificationHandler := &NotificationBusinessHTTPHandler{
		handler: notificationv1.NewNotificationHandler(s.deps.NotificationSrv),
	}
	if err := registry.RegisterBusinessHTTPHandler(notificationHandler); err != nil {
		return fmt.Errorf("failed to register notification HTTP handler: %w", err)
	}

	// Register health HTTP handler
	healthHandler := &HealthBusinessHTTPHandler{
		handler: health.NewHandler(s.deps.HealthSrv),
	}
	if err := registry.RegisterBusinessHTTPHandler(healthHandler); err != nil {
		return fmt.Errorf("failed to register health HTTP handler: %w", err)
	}

	// Register stop HTTP handler
	stopHandler := &StopBusinessHTTPHandler{
		handler: stop.NewHandler(s.deps.StopSrv),
	}
	if err := registry.RegisterBusinessHTTPHandler(stopHandler); err != nil {
		return fmt.Errorf("failed to register stop HTTP handler: %w", err)
	}

	// Register user HTTP handler with tracing
	userHandler := &UserBusinessHTTPHandler{
		handler: userv1.NewUserHandlerWithTracing(s.deps.UserSrv, s.deps.TracingManager),
	}
	if err := registry.RegisterBusinessHTTPHandler(userHandler); err != nil {
		return fmt.Errorf("failed to register user HTTP handler: %w", err)
	}

	// Register health checks
	greeterBusinessHealthCheck := &GreeterBusinessHealthCheck{
		greeterService: s.deps.GreeterSrv,
		startTime:      time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(greeterBusinessHealthCheck); err != nil {
		return fmt.Errorf("failed to register greeter health check: %w", err)
	}

	userBusinessHealthCheck := &UserBusinessHealthCheck{
		userService: s.deps.UserSrv,
		startTime:   time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(userBusinessHealthCheck); err != nil {
		return fmt.Errorf("failed to register user health check: %w", err)
	}

	notificationBusinessHealthCheck := &NotificationBusinessHealthCheck{
		notificationService: s.deps.NotificationSrv,
		startTime:           time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(notificationBusinessHealthCheck); err != nil {
		return fmt.Errorf("failed to register notification health check: %w", err)
	}

	healthServiceCheck := &HealthServiceCheck{
		healthService: s.deps.HealthSrv,
		startTime:     time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(healthServiceCheck); err != nil {
		return fmt.Errorf("failed to register health service check: %w", err)
	}

	stopServiceCheck := &StopServiceCheck{
		stopService: s.deps.StopSrv,
		startTime:   time.Now(),
	}
	if err := registry.RegisterBusinessHealthCheck(stopServiceCheck); err != nil {
		return fmt.Errorf("failed to register stop service check: %w", err)
	}

	logger.Logger.Info("Successfully registered all switserve services")
	return nil
}

// RegisterEventHandlers implements server.MessagingServiceRegistrar to support event handler registration.
// Current service does not expose messaging handlers; provide minimal no-op to satisfy interface and enable future extension.
func (s *ServiceRegistrar) RegisterEventHandlers(registry server.EventHandlerRegistry) error {
	if registry == nil {
		return fmt.Errorf("event handler registry cannot be nil")
	}
	// No messaging handlers yet for switserve.
	return nil
}

// GetEventHandlerMetadata returns minimal metadata for messaging capabilities.
func (s *ServiceRegistrar) GetEventHandlerMetadata() *server.EventHandlerMetadata {
	return &server.EventHandlerMetadata{
		HandlerCount:       0,
		Topics:             nil,
		BrokerRequirements: nil,
		Description:        "switserve has no messaging handlers currently",
	}
}

// GreeterBusinessHTTPHandler adapts the switserve greeter handler to the base server BusinessHTTPHandler interface
type GreeterBusinessHTTPHandler struct {
	handler *greeterv1.GreeterHandler
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *GreeterBusinessHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.handler.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *GreeterBusinessHTTPHandler) GetServiceName() string {
	return "greeter-service"
}

// NotificationBusinessHTTPHandler adapts the switserve notification handler to the base server BusinessHTTPHandler interface
type NotificationBusinessHTTPHandler struct {
	handler *notificationv1.NotificationHandler
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *NotificationBusinessHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.handler.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *NotificationBusinessHTTPHandler) GetServiceName() string {
	return "notification-service"
}

// HealthBusinessHTTPHandler adapts the switserve health handler to the base server BusinessHTTPHandler interface
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

// StopBusinessHTTPHandler adapts the switserve stop handler to the base server BusinessHTTPHandler interface
type StopBusinessHTTPHandler struct {
	handler *stop.Handler
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *StopBusinessHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.handler.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *StopBusinessHTTPHandler) GetServiceName() string {
	return "stop-service"
}

// UserBusinessHTTPHandler adapts the switserve user handler to the base server BusinessHTTPHandler interface
type UserBusinessHTTPHandler struct {
	handler *userv1.UserHandler
}

// RegisterRoutes registers HTTP routes with the provided router
func (h *UserBusinessHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	return h.handler.RegisterHTTP(ginRouter)
}

// GetServiceName returns the service name for identification
func (h *UserBusinessHTTPHandler) GetServiceName() string {
	return "user-service"
}

// GreeterBusinessHealthCheck implements the base server BusinessHealthCheck interface for greeter service
type GreeterBusinessHealthCheck struct {
	greeterService interface{}
	startTime      time.Time
}

// Check performs the health check and returns the status
func (h *GreeterBusinessHealthCheck) Check(ctx context.Context) error {
	if h.greeterService == nil {
		return fmt.Errorf("greeter service is not available")
	}
	return nil
}

// GetServiceName returns the service name for the health check
func (h *GreeterBusinessHealthCheck) GetServiceName() string {
	return "greeter-service"
}

// UserBusinessHealthCheck implements the base server BusinessHealthCheck interface for user service
type UserBusinessHealthCheck struct {
	userService interface{}
	startTime   time.Time
}

// Check performs the health check and returns the status
func (h *UserBusinessHealthCheck) Check(ctx context.Context) error {
	if h.userService == nil {
		return fmt.Errorf("user service is not available")
	}
	return nil
}

// GetServiceName returns the service name for the health check
func (h *UserBusinessHealthCheck) GetServiceName() string {
	return "user-service"
}

// NotificationBusinessHealthCheck implements the base server BusinessHealthCheck interface for notification service
type NotificationBusinessHealthCheck struct {
	notificationService interface{}
	startTime           time.Time
}

// Check performs the health check and returns the status
func (h *NotificationBusinessHealthCheck) Check(ctx context.Context) error {
	if h.notificationService == nil {
		return fmt.Errorf("notification service is not available")
	}
	return nil
}

// GetServiceName returns the service name for the health check
func (h *NotificationBusinessHealthCheck) GetServiceName() string {
	return "notification-service"
}

// HealthServiceCheck implements the base server BusinessHealthCheck interface for health service
type HealthServiceCheck struct {
	healthService interface{}
	startTime     time.Time
}

// Check performs the health check and returns the status
func (h *HealthServiceCheck) Check(ctx context.Context) error {
	if h.healthService == nil {
		return fmt.Errorf("health service is not available")
	}
	return nil
}

// GetServiceName returns the service name for the health check
func (h *HealthServiceCheck) GetServiceName() string {
	return "health-service"
}

// StopServiceCheck implements the base server BusinessHealthCheck interface for stop service
type StopServiceCheck struct {
	stopService interface{}
	startTime   time.Time
}

// Check performs the health check and returns the status
func (h *StopServiceCheck) Check(ctx context.Context) error {
	if h.stopService == nil {
		return fmt.Errorf("stop service is not available")
	}
	return nil
}

// GetServiceName returns the service name for the health check
func (h *StopServiceCheck) GetServiceName() string {
	return "stop-service"
}

// BusinessDependencyContainer adapts switserve dependencies to the base server BusinessDependencyContainer interface
type BusinessDependencyContainer struct {
	deps   *deps.Dependencies
	closed bool
}

// NewBusinessDependencyContainer creates a new BusinessDependencyContainer for switserve
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

	logger.Logger.Info("Closing switserve dependencies")

	// Use the existing Close method from deps.Dependencies
	if err := d.deps.Close(); err != nil {
		logger.Logger.Error("Failed to close switserve dependencies", zap.Error(err))
		return err
	}

	d.closed = true
	logger.Logger.Info("Successfully closed switserve dependencies")
	return nil
}

// GetService retrieves a service by name from the container
func (d *BusinessDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	switch name {
	case "greeter-service":
		return d.deps.GreeterSrv, nil
	case "notification-service":
		return d.deps.NotificationSrv, nil
	case "health-service":
		return d.deps.HealthSrv, nil
	case "stop-service":
		return d.deps.StopSrv, nil
	case "user-service":
		return d.deps.UserSrv, nil
	case "database":
		return d.deps.DB, nil
	case "user-repository":
		return d.deps.UserRepo, nil
	default:
		return nil, fmt.Errorf("service %s not found", name)
	}
}

// ConfigMapper maps switserve configuration to base server configuration
type ConfigMapper struct {
	serveConfig *config.ServeConfig
}

// NewConfigMapper creates a new ConfigMapper for switserve
func NewConfigMapper(serveConfig *config.ServeConfig) *ConfigMapper {
	return &ConfigMapper{
		serveConfig: serveConfig,
	}
}

// ToServerConfig converts ServeConfig to ServerConfig
func (m *ConfigMapper) ToServerConfig() *server.ServerConfig {
	serverConfig := server.NewServerConfig()

	// Set service name
	serverConfig.ServiceName = "swit-serve"

	// Map HTTP configuration
	if m.serveConfig.Server.Port != "" {
		serverConfig.HTTP.Port = m.serveConfig.Server.Port
		serverConfig.HTTP.Address = ":" + m.serveConfig.Server.Port
	} else {
		serverConfig.HTTP.Port = "9000"
		serverConfig.HTTP.Address = ":9000"
	}
	serverConfig.HTTP.Enabled = true
	serverConfig.HTTP.EnableReady = true // Keep existing behavior

	// Map gRPC configuration
	if m.serveConfig.Server.GRPCPort != "" {
		serverConfig.GRPC.Port = m.serveConfig.Server.GRPCPort
		serverConfig.GRPC.Address = ":" + m.serveConfig.Server.GRPCPort
	} else {
		serverConfig.GRPC.Port = "10000"
		serverConfig.GRPC.Address = ":10000"
	}
	serverConfig.GRPC.Enabled = true
	serverConfig.GRPC.EnableKeepalive = true     // Keep existing behavior
	serverConfig.GRPC.EnableReflection = true    // Keep existing behavior
	serverConfig.GRPC.EnableHealthService = true // Keep existing behavior

	// Map service discovery configuration
	if m.serveConfig.ServiceDiscovery.Address != "" {
		serverConfig.Discovery.Address = m.serveConfig.ServiceDiscovery.Address
	}
	serverConfig.Discovery.ServiceName = "swit-serve"
	serverConfig.Discovery.Tags = []string{"api", "v1"}
	serverConfig.Discovery.Enabled = true

	// Set middleware configuration to match existing behavior
	serverConfig.Middleware.EnableCORS = true
	serverConfig.Middleware.EnableAuth = true
	serverConfig.Middleware.EnableRateLimit = true
	serverConfig.Middleware.EnableLogging = true

	// Set HTTP middleware configuration
	serverConfig.HTTP.Middleware.EnableCORS = true
	serverConfig.HTTP.Middleware.EnableAuth = true
	serverConfig.HTTP.Middleware.EnableRateLimit = true
	serverConfig.HTTP.Middleware.EnableLogging = true

	// Set shutdown timeout
	serverConfig.ShutdownTimeout = 5 * time.Second

	return serverConfig
}

// ServeBusinessGRPCService adapts switserve gRPC services to the base server BusinessGRPCService interface
// Currently placeholder as switserve has limited gRPC services implemented
type ServeBusinessGRPCService struct {
	serviceName string
}

// NewServeBusinessGRPCService creates a new ServeBusinessGRPCService
func NewServeBusinessGRPCService() *ServeBusinessGRPCService {
	return &ServeBusinessGRPCService{
		serviceName: "serve-grpc-service",
	}
}

// RegisterGRPC registers the gRPC service with the provided server
func (s *ServeBusinessGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", server)
	}

	logger.Logger.Info("Registering serve gRPC service", zap.String("service", s.serviceName))

	// TODO: Register actual gRPC services when implemented
	// For now, just log that gRPC registration was called
	_ = grpcServer // Use the server to avoid unused variable warning

	return nil
}

// GetServiceName returns the service name for identification
func (s *ServeBusinessGRPCService) GetServiceName() string {
	return s.serviceName
}
