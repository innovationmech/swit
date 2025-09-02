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
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
	"go.uber.org/zap"
)

// BusinessServerImpl implements the BusinessServerCore interface providing common server functionality
type BusinessServerImpl struct {
	config               *ServerConfig
	transportManager     *transport.TransportCoordinator
	httpTransport        *transport.HTTPNetworkService
	grpcTransport        *transport.GRPCNetworkService
	discoveryManager     ServiceDiscoveryManager
	serviceRegistrations []*ServiceRegistration
	dependencies         BusinessDependencyContainer
	serviceRegistrar     BusinessServiceRegistrar

	// State management
	mu      sync.RWMutex
	started bool

	// Performance monitoring
	startTime time.Time
	metrics   *PerformanceMetrics
	monitor   *PerformanceMonitor

	// Observability and Prometheus metrics
	observabilityManager *ObservabilityManager

	// Error monitoring
	sentryManager *SentryManager
}

// NewBusinessServerCore creates a new base server instance with the provided configuration
func NewBusinessServerCore(config *ServerConfig, registrar BusinessServiceRegistrar, deps BusinessDependencyContainer) (*BusinessServerImpl, error) {
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

	// Initialize logger with configuration
	loggerConfig := &logger.LoggerConfig{
		Level:              config.Logging.Level,
		Development:        config.Logging.Development,
		Encoding:           config.Logging.Encoding,
		OutputPaths:        config.Logging.OutputPaths,
		ErrorOutputPaths:   config.Logging.ErrorOutputPaths,
		DisableCaller:      config.Logging.DisableCaller,
		DisableStacktrace:  config.Logging.DisableStacktrace,
		SamplingEnabled:    config.Logging.SamplingEnabled,
		SamplingInitial:    config.Logging.SamplingInitial,
		SamplingThereafter: config.Logging.SamplingThereafter,
	}
	logger.InitLoggerWithConfig(loggerConfig)

	// Log initialization with service info
	logger.Logger.Info("Initializing business server",
		zap.String("service", config.ServiceName),
		zap.String("log_level", config.Logging.Level),
		zap.String("log_encoding", config.Logging.Encoding))

	server := &BusinessServerImpl{
		config:               config,
		dependencies:         deps,
		serviceRegistrar:     registrar,
		transportManager:     transport.NewTransportCoordinator(),
		metrics:              NewPerformanceMetrics(),
		monitor:              NewPerformanceMonitor(),
		sentryManager:        NewSentryManager(&config.Sentry),
		observabilityManager: NewObservabilityManager(config.ServiceName, nil),
	}

	// Add default performance monitoring hooks
	server.monitor.AddHook(PerformanceLoggingHook)
	server.monitor.AddHook(PerformanceThresholdViolationHook)
	server.monitor.AddHook(PerformanceMetricsCollectionHook)

	// Initialize transports based on configuration
	if err := server.initializeTransports(); err != nil {
		return nil, fmt.Errorf("failed to initialize transports: %w", err)
	}

	// Initialize service discovery manager
	discoveryManager, err := NewDiscoveryManager(&config.Discovery)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service discovery: %w", err)
	}
	server.discoveryManager = discoveryManager

	// Create service registrations for discovery
	server.serviceRegistrations = CreateServiceRegistrations(config)

	// Register services with transport manager
	if err := server.registerServices(); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return server, nil
}

// initializeTransports creates and configures transport instances based on configuration
func (s *BusinessServerImpl) initializeTransports() error {
	// Pre-allocate slice for transports to reduce allocations
	var transports []transport.NetworkTransport
	if s.config.IsHTTPEnabled() && s.config.IsGRPCEnabled() {
		transports = make([]transport.NetworkTransport, 0, 2)
	} else {
		transports = make([]transport.NetworkTransport, 0, 1)
	}

	// Initialize HTTP transport if enabled
	if s.config.IsHTTPEnabled() {
		httpConfig := &transport.HTTPTransportConfig{
			Address:     s.config.GetHTTPAddress(),
			Port:        s.config.HTTP.Port,
			EnableReady: s.config.HTTP.EnableReady,
		}
		s.httpTransport = transport.NewHTTPNetworkServiceWithConfig(httpConfig)
		transports = append(transports, s.httpTransport)

		logger.Logger.Info("HTTP transport initialized",
			zap.String("address", httpConfig.Address),
			zap.String("port", httpConfig.Port))
	}

	// Initialize gRPC transport if enabled
	if s.config.IsGRPCEnabled() {
		grpcConfig := &transport.GRPCTransportConfig{
			Address:             s.config.GetGRPCAddress(),
			Port:                s.config.GRPC.Port,
			EnableKeepalive:     s.config.GRPC.EnableKeepalive,
			EnableReflection:    s.config.GRPC.EnableReflection,
			EnableHealthService: s.config.GRPC.EnableHealthService,
			MaxRecvMsgSize:      s.config.GRPC.MaxRecvMsgSize,
			MaxSendMsgSize:      s.config.GRPC.MaxSendMsgSize,
			KeepaliveParams:     s.config.toGRPCKeepaliveParams(),
			KeepalivePolicy:     s.config.toGRPCKeepalivePolicy(),
		}

		// Configure interceptors using middleware manager (reuse if possible)
		middlewareManager := NewMiddlewareManager(s.config)
		unaryInterceptors, streamInterceptors := middlewareManager.GetGRPCInterceptors()
		grpcConfig.UnaryInterceptors = unaryInterceptors
		grpcConfig.StreamInterceptors = streamInterceptors

		s.grpcTransport = transport.NewGRPCNetworkServiceWithConfig(grpcConfig)
		transports = append(transports, s.grpcTransport)

		logger.Logger.Info("gRPC transport initialized",
			zap.String("address", grpcConfig.Address),
			zap.Bool("keepalive", grpcConfig.EnableKeepalive),
			zap.Bool("reflection", grpcConfig.EnableReflection))
	}

	// Register all transports at once to reduce lock contention
	for _, t := range transports {
		s.transportManager.Register(t)
	}

	return nil
}

// registerServices registers all services with the transport manager using the service registrar
func (s *BusinessServerImpl) registerServices() error {
	registrationStart := time.Now()

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

	registrationDuration := time.Since(registrationStart)
	logger.Logger.Info("Services registered successfully",
		zap.Duration("registration_time", registrationDuration))
	return nil
}

// Start starts the server with all registered services
func (s *BusinessServerImpl) Start(ctx context.Context) error {
	startTime := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server is already started")
	}

	s.startTime = startTime
	logger.Logger.Info("Starting base server",
		zap.String("service", s.config.ServiceName),
		zap.Bool("discovery_enabled", s.config.IsDiscoveryEnabled()),
		zap.Bool("sentry_enabled", s.config.Sentry.Enabled),
		zap.String("discovery_failure_mode", string(s.config.Discovery.FailureMode)),
		zap.String("discovery_description", s.config.GetDiscoveryFailureModeDescription()))

	// Initialize Sentry if enabled
	if s.config.Sentry.Enabled {
		if err := s.sentryManager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize Sentry: %w", err)
		}

		// Add context tags for the service
		s.sentryManager.AddContextTags(s.config.ServiceName, "", s.config.Sentry.Environment)
		logger.Logger.Info("Sentry error monitoring initialized",
			zap.String("environment", s.config.Sentry.Environment))
	}

	// Initialize dependencies if available
	if s.dependencies != nil {
		if initializer, ok := s.dependencies.(interface{ Initialize(context.Context) error }); ok {
			if err := initializer.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}
			logger.Logger.Info("Dependencies initialized successfully")
		}
	}

	// Configure middleware for HTTP transport
	if s.httpTransport != nil {
		if err := s.configureHTTPMiddleware(); err != nil {
			return fmt.Errorf("failed to configure HTTP middleware: %w", err)
		}
	}

	// Initialize all services
	if err := s.transportManager.InitializeTransportServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Register HTTP routes
	if s.httpTransport != nil {
		if err := s.transportManager.BindAllHTTPEndpoints(s.httpTransport.GetRouter()); err != nil {
			return fmt.Errorf("failed to register HTTP routes: %w", err)
		}
	}

	// Register gRPC services
	if s.grpcTransport != nil {
		if err := s.transportManager.BindAllGRPCServices(s.grpcTransport.GetServer()); err != nil {
			return fmt.Errorf("failed to register gRPC services: %w", err)
		}
	}

	// Start all transports
	if err := s.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transports: %w", err)
	}

	// Register with service discovery
	if s.config.IsDiscoveryEnabled() {
		if err := s.registerWithDiscovery(ctx); err != nil {
			// Handle discovery failure based on configured failure mode
			switch s.config.Discovery.FailureMode {
			case DiscoveryFailureModeFailFast, DiscoveryFailureModeStrict:
				logger.Logger.Error("Service discovery registration failed with fail-fast mode enabled",
					zap.Error(err),
					zap.String("failure_mode", string(s.config.Discovery.FailureMode)),
					zap.String("action", "failing server startup"))
				return fmt.Errorf("failed to register with service discovery (failure_mode=%s): %w",
					s.config.Discovery.FailureMode, err)
			case DiscoveryFailureModeGraceful:
				fallthrough
			default:
				// Log warning but continue startup - graceful handling of discovery failures
				logger.Logger.Warn("Failed to register with service discovery in graceful mode",
					zap.Error(err),
					zap.String("failure_mode", string(s.config.Discovery.FailureMode)),
					zap.String("action", "continuing without discovery registration"))
			}
		} else {
			logger.Logger.Info("Successfully registered with service discovery",
				zap.String("failure_mode", string(s.config.Discovery.FailureMode)),
				zap.String("service", s.config.Discovery.ServiceName))
		}
	}

	s.started = true

	// Record performance metrics
	startupDuration := time.Since(startTime)
	s.metrics.RecordStartupTime(startupDuration)
	s.metrics.RecordMemoryUsage()

	transports := s.transportManager.GetTransports()
	s.metrics.RecordServiceMetrics(0, len(transports)) // Service count would need registry integration

	// Record Prometheus metrics via observability manager
	s.observabilityManager.RecordServerStartup(startupDuration)
	s.observabilityManager.UpdateSystemMetrics()

	// Record transport startup metrics
	for _, transport := range transports {
		transportName := "unknown"
		if s.httpTransport != nil && transport == s.httpTransport {
			transportName = "http"
		} else if s.grpcTransport != nil && transport == s.grpcTransport {
			transportName = "grpc"
		}
		s.observabilityManager.RecordTransportStart(transportName)
	}

	// Trigger performance monitoring hooks
	s.monitor.RecordEvent("server_startup_success")

	// Start periodic metrics collection
	go s.monitor.StartPeriodicCollection(context.Background(), 30*time.Second)

	// Start Prometheus system metrics collection
	go s.observabilityManager.StartSystemMetricsCollection(context.Background(), 30*time.Second)

	logger.Logger.Info("Base server started successfully",
		zap.String("service", s.config.ServiceName),
		zap.String("http_address", s.GetHTTPAddress()),
		zap.String("grpc_address", s.GetGRPCAddress()),
		zap.Duration("startup_time", startupDuration),
		zap.Int("goroutines", runtime.NumGoroutine()))

	return nil
}

// Stop gracefully stops the server
func (s *BusinessServerImpl) Stop(ctx context.Context) error {
	stopTime := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil // Already stopped
	}

	logger.Logger.Info("Stopping base server", zap.String("service", s.config.ServiceName))

	// Deregister from service discovery
	if s.config.IsDiscoveryEnabled() && s.discoveryManager != nil {
		if err := s.deregisterFromDiscovery(ctx); err != nil {
			logger.Logger.Warn("Failed to deregister from service discovery", zap.Error(err))
		}
	}

	// Stop all transports
	if err := s.transportManager.Stop(s.config.ShutdownTimeout); err != nil {
		return fmt.Errorf("failed to stop transports: %w", err)
	}

	s.started = false

	// Record shutdown performance metrics
	shutdownDuration := time.Since(stopTime)
	s.metrics.RecordShutdownTime(shutdownDuration)
	s.metrics.RecordMemoryUsage()

	// Record Prometheus shutdown metrics via observability manager
	s.observabilityManager.RecordServerShutdown(shutdownDuration)
	s.observabilityManager.UpdateSystemMetrics()

	// Record transport shutdown metrics
	transports := s.transportManager.GetTransports()
	for _, transport := range transports {
		transportName := "unknown"
		if s.httpTransport != nil && transport == s.httpTransport {
			transportName = "http"
		} else if s.grpcTransport != nil && transport == s.grpcTransport {
			transportName = "grpc"
		}
		s.observabilityManager.RecordTransportStop(transportName)
	}

	// Trigger performance monitoring hooks
	s.monitor.RecordEvent("server_shutdown_success")

	logger.Logger.Info("Base server stopped successfully",
		zap.String("service", s.config.ServiceName),
		zap.Duration("shutdown_time", shutdownDuration),
		zap.Int("goroutines", runtime.NumGoroutine()))
	return nil
}

// Shutdown performs complete server shutdown with resource cleanup
func (s *BusinessServerImpl) Shutdown() error {
	// Use a timeout context for the entire shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop the server
	if err := s.Stop(ctx); err != nil {
		logger.Logger.Error("Error during server stop", zap.Error(err))
	}

	// Close Sentry manager
	if s.sentryManager != nil {
		if err := s.sentryManager.Close(); err != nil {
			logger.Logger.Error("Failed to close Sentry manager", zap.Error(err))
		} else if s.sentryManager.IsEnabled() {
			logger.Logger.Info("Sentry manager closed successfully")
		}
	}

	// Close dependencies if available
	if s.dependencies != nil {
		// Create a separate timeout context for dependency cleanup
		depCtx, depCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer depCancel()

		// Use a goroutine with timeout to prevent hanging
		done := make(chan error, 1)
		go func() {
			done <- s.dependencies.Close()
		}()

		select {
		case err := <-done:
			if err != nil {
				logger.Logger.Error("Failed to close dependencies", zap.Error(err))
				return fmt.Errorf("failed to close dependencies: %w", err)
			}
		case <-depCtx.Done():
			logger.Logger.Warn("Dependency cleanup timed out", zap.Duration("timeout", 10*time.Second))
			return fmt.Errorf("dependency cleanup timed out")
		}
	}

	logger.Logger.Info("Base server shutdown completed", zap.String("service", s.config.ServiceName))
	return nil
}

// GetHTTPAddress returns the HTTP server listening address
func (s *BusinessServerImpl) GetHTTPAddress() string {
	if s.httpTransport != nil {
		return s.httpTransport.GetAddress()
	}
	return ""
}

// GetGRPCAddress returns the gRPC server listening address
func (s *BusinessServerImpl) GetGRPCAddress() string {
	if s.grpcTransport != nil {
		return s.grpcTransport.GetAddress()
	}
	return ""
}

// GetTransports returns all registered transports
func (s *BusinessServerImpl) GetTransports() []transport.NetworkTransport {
	return s.transportManager.GetTransports()
}

// GetTransportStatus returns the status of all transports
func (s *BusinessServerImpl) GetTransportStatus() map[string]TransportStatus {
	transports := s.transportManager.GetTransports()
	status := make(map[string]TransportStatus)

	for _, t := range transports {
		transportStatus := TransportStatus{
			Name:    t.GetName(),
			Address: t.GetAddress(),
			Running: s.isTransportRunning(t),
		}
		status[t.GetName()] = transportStatus
	}

	return status
}

// GetTransportHealth returns health status of all services across all transports
func (s *BusinessServerImpl) GetTransportHealth(ctx context.Context) map[string]map[string]*types.HealthStatus {
	return s.transportManager.CheckAllServicesHealth(ctx)
}

// GetPerformanceMetrics returns current performance metrics
func (s *BusinessServerImpl) GetPerformanceMetrics() *PerformanceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Update current metrics before returning
	s.metrics.RecordMemoryUsage()
	s.metrics.RecordUptime(s.GetUptime())
	return s.metrics
}

// GetPerformanceMonitor returns the performance monitor instance
func (s *BusinessServerImpl) GetPerformanceMonitor() *PerformanceMonitor {
	return s.monitor
}

// GetUptime returns the server uptime
func (s *BusinessServerImpl) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return 0
	}
	return time.Since(s.startTime)
}

// GetSentryManager returns the Sentry manager instance
func (s *BusinessServerImpl) GetSentryManager() *SentryManager {
	return s.sentryManager
}

// GetObservabilityManager returns the observability manager instance
func (s *BusinessServerImpl) GetObservabilityManager() *ObservabilityManager {
	return s.observabilityManager
}

// GetPrometheusCollector returns the Prometheus metrics collector
func (s *BusinessServerImpl) GetPrometheusCollector() *PrometheusMetricsCollector {
	if s.observabilityManager != nil {
		return s.observabilityManager.GetPrometheusCollector()
	}
	return nil
}

// GetBusinessMetricsManager returns the business metrics manager
func (s *BusinessServerImpl) GetBusinessMetricsManager() *BusinessMetricsManager {
	if s.observabilityManager != nil {
		return s.observabilityManager.GetBusinessMetricsManager()
	}
	return nil
}

// isTransportRunning checks if a transport is currently running
func (s *BusinessServerImpl) isTransportRunning(t transport.NetworkTransport) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// A transport is considered running if the server is started and the transport has an address
	return s.started && t.GetAddress() != ""
}

// SetDiscoveryManager sets the discovery manager (for testing purposes)
func (s *BusinessServerImpl) SetDiscoveryManager(manager ServiceDiscoveryManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discoveryManager = manager
}

// configureHTTPMiddleware configures global middleware for HTTP transport
func (s *BusinessServerImpl) configureHTTPMiddleware() error {
	if s.httpTransport == nil {
		return nil
	}

	router := s.httpTransport.GetRouter()
	if router == nil {
		return fmt.Errorf("HTTP router not available")
	}

	// Create middleware manager and configure middleware
	middlewareManager := NewMiddlewareManager(s.config)
	if err := middlewareManager.ConfigureHTTPMiddleware(router); err != nil {
		return fmt.Errorf("failed to configure HTTP middleware: %w", err)
	}

	// Register observability endpoints including Prometheus metrics
	if s.observabilityManager != nil {
		s.observabilityManager.RegisterObservabilityEndpoints(router, s)
	}

	return nil
}

// registerWithDiscovery registers the service with service discovery using the new abstraction
func (s *BusinessServerImpl) registerWithDiscovery(ctx context.Context) error {
	if s.discoveryManager == nil {
		return fmt.Errorf("discovery manager not initialized")
	}

	if len(s.serviceRegistrations) == 0 {
		logger.Logger.Debug("No service registrations configured")
		return nil
	}

	// Create a timeout context for the registration operation
	regCtx, cancel := context.WithTimeout(ctx, s.config.Discovery.RegistrationTimeout)
	defer cancel()

	// For strict mode, check discovery health before attempting registration
	if s.config.Discovery.FailureMode == DiscoveryFailureModeStrict {
		if s.config.Discovery.HealthCheckRequired {
			logger.Logger.Debug("Checking discovery service health before registration (strict mode)")

			healthCtx, healthCancel := context.WithTimeout(regCtx, 10*time.Second)
			isHealthy := s.discoveryManager.IsHealthy(healthCtx)
			healthCancel()

			if !isHealthy {
				return fmt.Errorf("discovery service is not healthy and strict mode is enabled")
			}

			logger.Logger.Debug("Discovery service health check passed in strict mode")
		}
	}

	// Log the registration attempt with failure mode context
	logger.Logger.Info("Attempting service discovery registration",
		zap.String("failure_mode", string(s.config.Discovery.FailureMode)),
		zap.String("service", s.config.Discovery.ServiceName),
		zap.Int("endpoints", len(s.serviceRegistrations)),
		zap.Duration("timeout", s.config.Discovery.RegistrationTimeout))

	// Register multiple endpoints with configured timeout
	if err := s.discoveryManager.RegisterMultipleEndpoints(regCtx, s.serviceRegistrations); err != nil {
		// Check if the error is due to timeout
		if regCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("service discovery registration timed out after %v: %w",
				s.config.Discovery.RegistrationTimeout, err)
		}
		return fmt.Errorf("failed to register service endpoints: %w", err)
	}

	logger.Logger.Info("Service endpoints registered with discovery",
		zap.String("service", s.config.Discovery.ServiceName),
		zap.String("failure_mode", string(s.config.Discovery.FailureMode)),
		zap.Int("endpoints", len(s.serviceRegistrations)))

	return nil
}

// deregisterFromDiscovery deregisters the service from service discovery using the new abstraction
func (s *BusinessServerImpl) deregisterFromDiscovery(ctx context.Context) error {
	if s.discoveryManager == nil {
		return nil
	}

	if len(s.serviceRegistrations) == 0 {
		logger.Logger.Debug("No service registrations to deregister")
		return nil
	}

	// Deregister multiple endpoints with graceful failure handling
	if err := s.discoveryManager.DeregisterMultipleEndpoints(ctx, s.serviceRegistrations); err != nil {
		return fmt.Errorf("failed to deregister service endpoints: %w", err)
	}

	logger.Logger.Info("Service endpoints deregistered from discovery",
		zap.String("service", s.config.Discovery.ServiceName),
		zap.Int("endpoints", len(s.serviceRegistrations)))

	return nil
}

// serviceRegistryAdapter adapts our BusinessServiceRegistry interface to the transport layer
type serviceRegistryAdapter struct {
	transportManager *transport.TransportCoordinator
	httpTransport    *transport.HTTPNetworkService
	grpcTransport    *transport.GRPCNetworkService
}

// RegisterBusinessHTTPHandler registers an HTTP service handler
func (a *serviceRegistryAdapter) RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error {
	if a.httpTransport == nil {
		return fmt.Errorf("HTTP transport not available")
	}

	// Create an adapter that implements transport.TransportServiceHandler
	adapter := &httpHandlerAdapter{handler: handler}
	return a.transportManager.RegisterHTTPService(adapter)
}

// RegisterBusinessGRPCService registers a gRPC service
func (a *serviceRegistryAdapter) RegisterBusinessGRPCService(service BusinessGRPCService) error {
	if a.grpcTransport == nil {
		return fmt.Errorf("gRPC transport not available")
	}

	// Create an adapter that implements transport.TransportServiceHandler
	adapter := &grpcServiceAdapter{service: service}
	return a.transportManager.RegisterGRPCService(adapter)
}

// RegisterBusinessHealthCheck registers a health check for a service
func (a *serviceRegistryAdapter) RegisterBusinessHealthCheck(check BusinessHealthCheck) error {
	// Health checks are typically handled through the service handlers themselves
	// This is a placeholder for future health check registration logic
	logger.Logger.Info("Health check registered", zap.String("service", check.GetServiceName()))
	return nil
}

// httpHandlerAdapter adapts BusinessHTTPHandler to transport.NetworkTransportServiceHandler
type httpHandlerAdapter struct {
	handler BusinessHTTPHandler
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

// grpcServiceAdapter adapts BusinessGRPCService to transport.NetworkTransportServiceHandler
type grpcServiceAdapter struct {
	service BusinessGRPCService
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
