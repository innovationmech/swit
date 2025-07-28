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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
	"go.uber.org/zap"
)

// BaseServerImpl implements the BaseServer interface providing common server functionality
type BaseServerImpl struct {
	config           *ServerConfig
	transportManager *transport.Manager
	httpTransport    *transport.HTTPTransport
	grpcTransport    *transport.GRPCTransport
	serviceDiscovery *discovery.ServiceDiscovery
	dependencies     DependencyContainer
	serviceRegistrar ServiceRegistrar

	// State management
	mu      sync.RWMutex
	started bool
}

// NewBaseServer creates a new base server instance with the provided configuration
func NewBaseServer(config *ServerConfig, registrar ServiceRegistrar, deps DependencyContainer) (*BaseServerImpl, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	if registrar == nil {
		return nil, fmt.Errorf("service registrar cannot be nil")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	server := &BaseServerImpl{
		config:           config,
		dependencies:     deps,
		serviceRegistrar: registrar,
		transportManager: transport.NewManager(),
	}

	// Initialize transports based on configuration
	if err := server.initializeTransports(); err != nil {
		return nil, fmt.Errorf("failed to initialize transports: %w", err)
	}

	// Initialize service discovery if enabled
	if config.IsDiscoveryEnabled() {
		if err := server.initializeServiceDiscovery(); err != nil {
			return nil, fmt.Errorf("failed to initialize service discovery: %w", err)
		}
	}

	// Register services with transport manager
	if err := server.registerServices(); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return server, nil
}

// initializeTransports creates and configures transport instances based on configuration
func (s *BaseServerImpl) initializeTransports() error {
	// Initialize HTTP transport if enabled
	if s.config.IsHTTPEnabled() {
		httpConfig := &transport.HTTPTransportConfig{
			Address:     s.config.GetHTTPAddress(),
			Port:        s.config.HTTP.Port,
			EnableReady: s.config.HTTP.EnableReady,
		}
		s.httpTransport = transport.NewHTTPTransportWithConfig(httpConfig)
		s.transportManager.Register(s.httpTransport)

		logger.Logger.Info("HTTP transport initialized",
			zap.String("address", httpConfig.Address),
			zap.String("port", httpConfig.Port))
	}

	// Initialize gRPC transport if enabled
	if s.config.IsGRPCEnabled() {
		grpcConfig := transport.DefaultGRPCConfig()
		grpcConfig.Address = s.config.GetGRPCAddress()
		grpcConfig.EnableKeepalive = s.config.GRPC.EnableKeepalive
		grpcConfig.EnableReflection = s.config.GRPC.EnableReflection
		grpcConfig.EnableHealthService = s.config.GRPC.EnableHealthService

		// Add default middleware
		grpcConfig.UnaryInterceptors = []grpc.UnaryServerInterceptor{
			middleware.GRPCRecoveryInterceptor(),
			middleware.GRPCLoggingInterceptor(),
			middleware.GRPCValidationInterceptor(),
		}

		s.grpcTransport = transport.NewGRPCTransportWithConfig(grpcConfig)
		s.transportManager.Register(s.grpcTransport)

		logger.Logger.Info("gRPC transport initialized",
			zap.String("address", grpcConfig.Address),
			zap.Bool("keepalive", grpcConfig.EnableKeepalive),
			zap.Bool("reflection", grpcConfig.EnableReflection))
	}

	return nil
}

// initializeServiceDiscovery creates and configures the service discovery client
func (s *BaseServerImpl) initializeServiceDiscovery() error {
	sd, err := discovery.GetServiceDiscoveryByAddress(s.config.Discovery.Address)
	if err != nil {
		return fmt.Errorf("failed to create service discovery client: %w", err)
	}

	s.serviceDiscovery = sd
	logger.Logger.Info("Service discovery initialized",
		zap.String("address", s.config.Discovery.Address))

	return nil
}

// registerServices registers all services with the transport manager using the service registrar
func (s *BaseServerImpl) registerServices() error {
	// Create a service registry adapter that bridges our interface to the transport layer
	registry := &serviceRegistryAdapter{
		transportManager: s.transportManager,
		httpTransport:    s.httpTransport,
		grpcTransport:    s.grpcTransport,
	}

	// Use the service registrar to register services
	if err := s.serviceRegistrar.RegisterServices(registry); err != nil {
		return fmt.Errorf("service registration failed: %w", err)
	}

	logger.Logger.Info("Services registered successfully")
	return nil
}

// Start starts the server with all registered services
func (s *BaseServerImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server is already started")
	}

	logger.Logger.Info("Starting base server", zap.String("service", s.config.ServiceName))

	// Configure middleware for HTTP transport
	if s.httpTransport != nil {
		s.configureHTTPMiddleware()
	}

	// Initialize all services
	if err := s.transportManager.InitializeAllServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Register HTTP routes
	if s.httpTransport != nil {
		if err := s.transportManager.RegisterAllHTTPRoutes(s.httpTransport.GetRouter()); err != nil {
			return fmt.Errorf("failed to register HTTP routes: %w", err)
		}
	}

	// Register gRPC services
	if s.grpcTransport != nil {
		if err := s.transportManager.RegisterAllGRPCServices(s.grpcTransport.GetServer()); err != nil {
			return fmt.Errorf("failed to register gRPC services: %w", err)
		}
	}

	// Start all transports
	if err := s.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transports: %w", err)
	}

	// Register with service discovery
	if s.config.IsDiscoveryEnabled() {
		if err := s.registerWithDiscovery(); err != nil {
			// Log warning but don't fail startup
			logger.Logger.Warn("Failed to register with service discovery", zap.Error(err))
		}
	}

	s.started = true

	logger.Logger.Info("Base server started successfully",
		zap.String("service", s.config.ServiceName),
		zap.String("http_address", s.GetHTTPAddress()),
		zap.String("grpc_address", s.GetGRPCAddress()))

	return nil
}

// Stop gracefully stops the server
func (s *BaseServerImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil // Already stopped
	}

	logger.Logger.Info("Stopping base server", zap.String("service", s.config.ServiceName))

	// Deregister from service discovery
	if s.config.IsDiscoveryEnabled() && s.serviceDiscovery != nil {
		if err := s.deregisterFromDiscovery(); err != nil {
			logger.Logger.Warn("Failed to deregister from service discovery", zap.Error(err))
		}
	}

	// Stop all transports
	if err := s.transportManager.Stop(s.config.ShutdownTimeout); err != nil {
		return fmt.Errorf("failed to stop transports: %w", err)
	}

	s.started = false

	logger.Logger.Info("Base server stopped successfully", zap.String("service", s.config.ServiceName))
	return nil
}

// Shutdown performs complete server shutdown with resource cleanup
func (s *BaseServerImpl) Shutdown() error {
	ctx := context.Background()

	// Stop the server
	if err := s.Stop(ctx); err != nil {
		logger.Logger.Error("Error during server stop", zap.Error(err))
	}

	// Close dependencies if available
	if s.dependencies != nil {
		if err := s.dependencies.Close(); err != nil {
			logger.Logger.Error("Failed to close dependencies", zap.Error(err))
			return fmt.Errorf("failed to close dependencies: %w", err)
		}
	}

	logger.Logger.Info("Base server shutdown completed", zap.String("service", s.config.ServiceName))
	return nil
}

// GetHTTPAddress returns the HTTP server listening address
func (s *BaseServerImpl) GetHTTPAddress() string {
	if s.httpTransport != nil {
		return s.httpTransport.GetAddress()
	}
	return ""
}

// GetGRPCAddress returns the gRPC server listening address
func (s *BaseServerImpl) GetGRPCAddress() string {
	if s.grpcTransport != nil {
		return s.grpcTransport.GetAddress()
	}
	return ""
}

// GetTransports returns all registered transports
func (s *BaseServerImpl) GetTransports() []transport.Transport {
	return s.transportManager.GetTransports()
}

// configureHTTPMiddleware configures global middleware for HTTP transport
func (s *BaseServerImpl) configureHTTPMiddleware() {
	if s.httpTransport == nil {
		return
	}

	router := s.httpTransport.GetRouter()
	if router == nil {
		return
	}

	// Apply global middleware based on configuration
	registrar := middleware.NewGlobalMiddlewareRegistrar()
	registrar.RegisterMiddleware(router)

	logger.Logger.Info("HTTP middleware configured",
		zap.Bool("cors", s.config.Middleware.EnableCORS),
		zap.Bool("auth", s.config.Middleware.EnableAuth),
		zap.Bool("rate_limit", s.config.Middleware.EnableRateLimit),
		zap.Bool("logging", s.config.Middleware.EnableLogging))
}

// registerWithDiscovery registers the service with service discovery
func (s *BaseServerImpl) registerWithDiscovery() error {
	if s.serviceDiscovery == nil {
		return fmt.Errorf("service discovery not initialized")
	}

	// Register HTTP service if enabled
	if s.config.IsHTTPEnabled() && s.httpTransport != nil {
		port := s.httpTransport.GetPort()
		serviceName := s.config.Discovery.ServiceName
		if err := s.serviceDiscovery.RegisterService(serviceName, "localhost", port); err != nil {
			return fmt.Errorf("failed to register HTTP service '%s' on port %d: %w", serviceName, port, err)
		}
		logger.Logger.Info("HTTP service registered with discovery",
			zap.String("service", serviceName),
			zap.Int("port", port))
	}

	// Register gRPC service if enabled and on different port
	if s.config.IsGRPCEnabled() && s.grpcTransport != nil {
		grpcPort := s.grpcTransport.GetPort()
		httpPort := 0
		if s.httpTransport != nil {
			httpPort = s.httpTransport.GetPort()
		}

		if grpcPort != httpPort {
			grpcServiceName := s.config.Discovery.ServiceName + "-grpc"
			if err := s.serviceDiscovery.RegisterService(grpcServiceName, "localhost", grpcPort); err != nil {
				logger.Logger.Warn("Failed to register gRPC service with discovery", zap.Error(err))
			} else {
				logger.Logger.Info("gRPC service registered with discovery",
					zap.String("service", grpcServiceName),
					zap.Int("port", grpcPort))
			}
		}
	}

	return nil
}

// deregisterFromDiscovery deregisters the service from service discovery
func (s *BaseServerImpl) deregisterFromDiscovery() error {
	if s.serviceDiscovery == nil {
		return nil
	}

	// Deregister HTTP service
	if s.config.IsHTTPEnabled() && s.httpTransport != nil {
		port := s.httpTransport.GetPort()
		serviceName := s.config.Discovery.ServiceName
		if err := s.serviceDiscovery.DeregisterService(serviceName, "localhost", port); err != nil {
			return fmt.Errorf("failed to deregister HTTP service: %w", err)
		}
	}

	// Deregister gRPC service
	if s.config.IsGRPCEnabled() && s.grpcTransport != nil {
		grpcPort := s.grpcTransport.GetPort()
		httpPort := 0
		if s.httpTransport != nil {
			httpPort = s.httpTransport.GetPort()
		}

		if grpcPort != httpPort {
			grpcServiceName := s.config.Discovery.ServiceName + "-grpc"
			if err := s.serviceDiscovery.DeregisterService(grpcServiceName, "localhost", grpcPort); err != nil {
				logger.Logger.Warn("Failed to deregister gRPC service", zap.Error(err))
			}
		}
	}

	return nil
}

// serviceRegistryAdapter adapts our ServiceRegistry interface to the transport layer
type serviceRegistryAdapter struct {
	transportManager *transport.Manager
	httpTransport    *transport.HTTPTransport
	grpcTransport    *transport.GRPCTransport
}

// RegisterHTTPHandler registers an HTTP service handler
func (a *serviceRegistryAdapter) RegisterHTTPHandler(handler HTTPHandler) error {
	if a.httpTransport == nil {
		return fmt.Errorf("HTTP transport not available")
	}

	// Create an adapter that implements transport.HandlerRegister
	adapter := &httpHandlerAdapter{handler: handler}
	return a.httpTransport.RegisterService(adapter)
}

// RegisterGRPCService registers a gRPC service
func (a *serviceRegistryAdapter) RegisterGRPCService(service GRPCService) error {
	if a.grpcTransport == nil {
		return fmt.Errorf("gRPC transport not available")
	}

	// Create an adapter that implements transport.HandlerRegister
	adapter := &grpcServiceAdapter{service: service}
	return a.transportManager.RegisterGRPCHandler(adapter)
}

// RegisterHealthCheck registers a health check for a service
func (a *serviceRegistryAdapter) RegisterHealthCheck(check HealthCheck) error {
	// Health checks are typically handled through the service handlers themselves
	// This is a placeholder for future health check registration logic
	logger.Logger.Info("Health check registered", zap.String("service", check.GetServiceName()))
	return nil
}

// httpHandlerAdapter adapts HTTPHandler to transport.HandlerRegister
type httpHandlerAdapter struct {
	handler HTTPHandler
}

func (a *httpHandlerAdapter) RegisterHTTP(router *gin.Engine) error {
	return a.handler.RegisterRoutes(router)
}

func (a *httpHandlerAdapter) RegisterGRPC(server *grpc.Server) error {
	// HTTP handlers don't register gRPC services
	return nil
}

func (a *httpHandlerAdapter) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:        a.handler.GetServiceName(),
		Version:     "v1",
		Description: fmt.Sprintf("HTTP service: %s", a.handler.GetServiceName()),
	}
}

func (a *httpHandlerAdapter) GetHealthEndpoint() string {
	return fmt.Sprintf("/health/%s", a.handler.GetServiceName())
}

func (a *httpHandlerAdapter) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Default healthy status for HTTP handlers
	return &types.HealthStatus{
		Status:    types.HealthStatusHealthy,
		Timestamp: time.Now(),
		Version:   "v1",
	}, nil
}

func (a *httpHandlerAdapter) Initialize(ctx context.Context) error {
	// HTTP handlers typically don't need initialization
	return nil
}

func (a *httpHandlerAdapter) Shutdown(ctx context.Context) error {
	// HTTP handlers typically don't need shutdown logic
	return nil
}

// grpcServiceAdapter adapts GRPCService to transport.HandlerRegister
type grpcServiceAdapter struct {
	service GRPCService
}

func (a *grpcServiceAdapter) RegisterHTTP(router *gin.Engine) error {
	// gRPC services don't register HTTP routes
	return nil
}

func (a *grpcServiceAdapter) RegisterGRPC(server *grpc.Server) error {
	return a.service.RegisterGRPC(server)
}

func (a *grpcServiceAdapter) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:        a.service.GetServiceName(),
		Version:     "v1",
		Description: fmt.Sprintf("gRPC service: %s", a.service.GetServiceName()),
	}
}

func (a *grpcServiceAdapter) GetHealthEndpoint() string {
	return fmt.Sprintf("/health/%s", a.service.GetServiceName())
}

func (a *grpcServiceAdapter) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Default healthy status for gRPC services
	return &types.HealthStatus{
		Status:    types.HealthStatusHealthy,
		Timestamp: time.Now(),
		Version:   "v1",
	}, nil
}

func (a *grpcServiceAdapter) Initialize(ctx context.Context) error {
	// gRPC services typically don't need initialization
	return nil
}

func (a *grpcServiceAdapter) Shutdown(ctx context.Context) error {
	// gRPC services typically don't need shutdown logic
	return nil
}
