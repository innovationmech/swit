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
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/innovationmech/swit/pkg/logger"
)

// IntegrationTestSuite provides integration tests for the base server framework
type IntegrationTestSuite struct {
	suite.Suite
	server      *BusinessServerImpl
	config      *ServerConfig
	httpPort    string
	grpcPort    string
	testService *TestServiceRegistrar
	testDeps    *TestDependencyContainer
	cancelFunc  context.CancelFunc
	serverCtx   context.Context
}

// TestServiceRegistrar implements ServiceRegistrar for testing
type TestServiceRegistrar struct {
	httpHandlers []BusinessHTTPHandler
	grpcServices []BusinessGRPCService
	healthChecks []BusinessHealthCheck
}

func NewTestServiceRegistrar() *TestServiceRegistrar {
	return &TestServiceRegistrar{
		httpHandlers: make([]BusinessHTTPHandler, 0),
		grpcServices: make([]BusinessGRPCService, 0),
		healthChecks: make([]BusinessHealthCheck, 0),
	}
}

func (t *TestServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	// Register HTTP handlers
	for _, handler := range t.httpHandlers {
		if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
			return fmt.Errorf("failed to register HTTP handler %s: %w", handler.GetServiceName(), err)
		}
	}

	// Register gRPC services
	for _, service := range t.grpcServices {
		if err := registry.RegisterBusinessGRPCService(service); err != nil {
			return fmt.Errorf("failed to register gRPC service %s: %w", service.GetServiceName(), err)
		}
	}

	// Register health checks
	for _, check := range t.healthChecks {
		if err := registry.RegisterBusinessHealthCheck(check); err != nil {
			return fmt.Errorf("failed to register health check %s: %w", check.GetServiceName(), err)
		}
	}

	return nil
}

func (t *TestServiceRegistrar) AddHTTPHandler(handler BusinessHTTPHandler) {
	t.httpHandlers = append(t.httpHandlers, handler)
}

func (t *TestServiceRegistrar) AddGRPCService(service BusinessGRPCService) {
	t.grpcServices = append(t.grpcServices, service)
}

func (t *TestServiceRegistrar) AddHealthCheck(check BusinessHealthCheck) {
	t.healthChecks = append(t.healthChecks, check)
}

// TestHTTPHandler implements HTTPHandler for testing
type TestHTTPHandler struct {
	serviceName string
	routes      map[string]gin.HandlerFunc
}

func NewTestHTTPHandler(serviceName string) *TestHTTPHandler {
	return &TestHTTPHandler{
		serviceName: serviceName,
		routes:      make(map[string]gin.HandlerFunc),
	}
}

func (h *TestHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	for path, handler := range h.routes {
		ginRouter.GET(path, handler)
	}

	// Add ready endpoint
	ginRouter.GET("/ready", h.readyHandler)

	return nil
}

func (h *TestHTTPHandler) GetServiceName() string {
	return h.serviceName
}

func (h *TestHTTPHandler) AddRoute(path string, handler gin.HandlerFunc) {
	h.routes[path] = handler
}

func (h *TestHTTPHandler) readyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// TestGRPCService implements GRPCService for testing
type TestGRPCService struct {
	serviceName string
	registered  bool
}

func NewTestGRPCService(serviceName string) *TestGRPCService {
	return &TestGRPCService{
		serviceName: serviceName,
		registered:  false,
	}
}

func (s *TestGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", server)
	}

	// Don't register health service here as it's already registered by the transport
	// Just mark as registered for testing purposes
	_ = grpcServer // Use the server parameter to avoid unused variable warning
	s.registered = true
	return nil
}

func (s *TestGRPCService) GetServiceName() string {
	return s.serviceName
}

func (s *TestGRPCService) IsRegistered() bool {
	return s.registered
}

// TestHealthServer implements grpc_health_v1.HealthServer for testing
type TestHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (s *TestHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// TestHealthCheck implements HealthCheck for testing
type TestHealthCheck struct {
	serviceName string
	healthy     bool
}

func NewTestHealthCheck(serviceName string, healthy bool) *TestHealthCheck {
	return &TestHealthCheck{
		serviceName: serviceName,
		healthy:     healthy,
	}
}

func (h *TestHealthCheck) GetServiceName() string {
	return h.serviceName
}

func (h *TestHealthCheck) Check(ctx context.Context) error {
	if !h.healthy {
		return fmt.Errorf("service is unhealthy")
	}
	return nil
}

func (h *TestHealthCheck) SetHealthy(healthy bool) {
	h.healthy = healthy
}

// TestDependencyContainer implements DependencyContainer for testing
type TestDependencyContainer struct {
	services    map[string]interface{}
	initialized bool
	closed      bool
}

func NewTestDependencyContainer() *TestDependencyContainer {
	return &TestDependencyContainer{
		services:    make(map[string]interface{}),
		initialized: false,
		closed:      false,
	}
}

func (d *TestDependencyContainer) GetService(name string) (interface{}, error) {
	if service, exists := d.services[name]; exists {
		return service, nil
	}
	return nil, fmt.Errorf("service %s not found", name)
}

func (d *TestDependencyContainer) Close() error {
	d.closed = true
	return nil
}

func (d *TestDependencyContainer) Initialize(ctx context.Context) error {
	d.initialized = true
	return nil
}

func (d *TestDependencyContainer) AddService(name string, service interface{}) {
	d.services[name] = service
}

func (d *TestDependencyContainer) IsInitialized() bool {
	return d.initialized
}

func (d *TestDependencyContainer) IsClosed() bool {
	return d.closed
}

// SetupSuite initializes the test suite
func (suite *IntegrationTestSuite) SetupSuite() {
	// Initialize logger for tests
	logger.InitLogger()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// SetupTest sets up each test case
func (suite *IntegrationTestSuite) SetupTest() {
	// Find available ports
	suite.httpPort = suite.findAvailablePort()
	suite.grpcPort = suite.findAvailablePort()

	// Create test configuration
	suite.config = &ServerConfig{
		ServiceName: "test-service",
		HTTP: HTTPConfig{
			Port:         suite.httpPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                suite.grpcPort,
			EnableKeepalive:     true,
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled:     false, // Disable discovery for integration tests
			ServiceName: "test-service",
		},
		Middleware: MiddlewareConfig{
			EnableCORS:      true,
			EnableAuth:      false,
			EnableRateLimit: false,
			EnableLogging:   true,
		},
		Sentry: SentryConfig{
			Enabled: false,
		},
		Logging: LoggingConfig{
			Level:       "info",
			Encoding:    "console",
			Development: true,
		},
	}

	// Create test dependencies
	suite.testDeps = NewTestDependencyContainer()
	suite.testDeps.AddService("test-db", "mock-database")

	// Create test service registrar
	suite.testService = NewTestServiceRegistrar()

	// Add test HTTP handler
	httpHandler := NewTestHTTPHandler("test-http")
	httpHandler.AddRoute("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test response"})
	})
	suite.testService.AddHTTPHandler(httpHandler)

	// Add test gRPC service
	grpcService := NewTestGRPCService("test-grpc")
	suite.testService.AddGRPCService(grpcService)

	// Add test health check
	healthCheck := NewTestHealthCheck("test-service", true)
	suite.testService.AddHealthCheck(healthCheck)

	// Create server context
	suite.serverCtx, suite.cancelFunc = context.WithCancel(context.Background())
}

// TearDownTest cleans up after each test case
func (suite *IntegrationTestSuite) TearDownTest() {
	if suite.server != nil {
		// Stop the server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := suite.server.Stop(ctx); err != nil {
			suite.T().Logf("Error stopping server: %v", err)
		}

		// Shutdown the server
		if err := suite.server.Shutdown(); err != nil {
			suite.T().Logf("Error shutting down server: %v", err)
		}
	}

	if suite.cancelFunc != nil {
		suite.cancelFunc()
	}
}

// findAvailablePort finds an available port for testing
func (suite *IntegrationTestSuite) findAvailablePort() string {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(suite.T(), err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}

// TestBaseServerCreation tests base server creation and configuration
func (suite *IntegrationTestSuite) TestBaseServerCreation() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), server)

	// Verify server configuration
	assert.Equal(suite.T(), "test-service", server.config.ServiceName)
	assert.Equal(suite.T(), suite.httpPort, server.config.HTTP.Port)
	assert.Equal(suite.T(), suite.grpcPort, server.config.GRPC.Port)
	assert.True(suite.T(), server.config.HTTP.EnableReady)
	assert.True(suite.T(), server.config.GRPC.EnableReflection)

	// Verify transports are initialized
	assert.NotNil(suite.T(), server.httpTransport)
	assert.NotNil(suite.T(), server.grpcTransport)
	assert.NotNil(suite.T(), server.transportManager)

	suite.server = server
}

// TestServerStartStop tests server startup and shutdown lifecycle
func (suite *IntegrationTestSuite) TestServerStartStop() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Test server start
	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Verify server is started
	assert.True(suite.T(), server.started)

	// Verify dependencies are initialized
	assert.True(suite.T(), suite.testDeps.IsInitialized())

	// Verify addresses are available
	httpAddr := server.GetHTTPAddress()
	grpcAddr := server.GetGRPCAddress()
	assert.NotEmpty(suite.T(), httpAddr)
	assert.NotEmpty(suite.T(), grpcAddr)
	assert.Contains(suite.T(), httpAddr, suite.httpPort)
	assert.Contains(suite.T(), grpcAddr, suite.grpcPort)

	// Test server stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	require.NoError(suite.T(), err)

	// Verify server is stopped
	assert.False(suite.T(), server.started)
}

// TestHTTPTransportIntegration tests HTTP transport with real requests
func (suite *IntegrationTestSuite) TestHTTPTransportIntegration() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test HTTP endpoint
	httpAddr := server.GetHTTPAddress()
	client := &http.Client{Timeout: 5 * time.Second}

	// Convert IPv6 address to localhost for HTTP client
	testAddr := httpAddr
	if strings.HasPrefix(httpAddr, "[::]:") {
		port := strings.TrimPrefix(httpAddr, "[::]:")
		testAddr = "localhost:" + port
	}

	// Test custom route
	testURL := fmt.Sprintf("http://%s/test", testAddr)
	resp, err := client.Get(testURL)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Test health endpoint
	healthURL := fmt.Sprintf("http://%s/health", testAddr)
	resp, err = client.Get(healthURL)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Test ready endpoint (if enabled)
	if suite.config.HTTP.EnableReady {
		readyURL := fmt.Sprintf("http://%s/ready", testAddr)
		resp, err = client.Get(readyURL)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	}
}

// TestGRPCTransportIntegration tests gRPC transport with real connections
func (suite *IntegrationTestSuite) TestGRPCTransportIntegration() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test gRPC connection
	grpcAddr := server.GetGRPCAddress()
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(suite.T(), err)
	defer conn.Close()

	// Test health check service
	healthClient := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthResp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), grpc_health_v1.HealthCheckResponse_SERVING, healthResp.Status)
}

// TestServerLifecycleWithErrors tests server behavior with various error conditions
func (suite *IntegrationTestSuite) TestServerLifecycleWithErrors() {
	// Test with invalid configuration
	invalidConfig := &ServerConfig{
		ServiceName: "", // Invalid empty service name
		HTTP: HTTPConfig{
			Port: "invalid-port",
		},
	}

	_, err := NewBusinessServerCore(invalidConfig, suite.testService, suite.testDeps)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid server configuration")

	// Test with nil registrar
	_, err = NewBusinessServerCore(suite.config, nil, suite.testDeps)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "service registrar cannot be nil")

	// Test double start
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Try to start again
	err = server.Start(suite.serverCtx)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "server is already started")
}

// TestTransportStatus tests transport status reporting
func (suite *IntegrationTestSuite) TestTransportStatus() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Check status before start
	status := server.GetTransportStatus()
	assert.Len(suite.T(), status, 2) // HTTP and gRPC transports

	httpStatus, exists := status["http"]
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), "http", httpStatus.Name)
	assert.False(suite.T(), httpStatus.Running) // Not started yet

	grpcStatus, exists := status["grpc"]
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), "grpc", grpcStatus.Name)
	assert.False(suite.T(), grpcStatus.Running) // Not started yet

	// Start server and check status again
	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	status = server.GetTransportStatus()
	httpStatus = status["http"]
	grpcStatus = status["grpc"]

	assert.True(suite.T(), httpStatus.Running)
	assert.True(suite.T(), grpcStatus.Running)
	assert.NotEmpty(suite.T(), httpStatus.Address)
	assert.NotEmpty(suite.T(), grpcStatus.Address)
}

// TestTransportHealth tests transport health checking
func (suite *IntegrationTestSuite) TestTransportHealth() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Check transport health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := server.GetTransportHealth(ctx)
	assert.NotEmpty(suite.T(), health)

	// Should have health information for registered services
	for transportName, services := range health {
		assert.NotEmpty(suite.T(), transportName)
		for serviceName, healthStatus := range services {
			assert.NotEmpty(suite.T(), serviceName)
			assert.NotNil(suite.T(), healthStatus)
			assert.NotZero(suite.T(), healthStatus.Timestamp)
		}
	}
}

// TestGracefulShutdown tests graceful server shutdown
func (suite *IntegrationTestSuite) TestGracefulShutdown() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Verify server is running
	assert.True(suite.T(), server.started)
	assert.True(suite.T(), suite.testDeps.IsInitialized())
	assert.False(suite.T(), suite.testDeps.IsClosed())

	// Test graceful shutdown
	err = server.Shutdown()
	require.NoError(suite.T(), err)

	// Verify server is stopped and dependencies are closed
	assert.False(suite.T(), server.started)
	assert.True(suite.T(), suite.testDeps.IsClosed())
}

// TestConcurrentRequests tests server behavior under concurrent load
func (suite *IntegrationTestSuite) TestConcurrentRequests() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	err = server.Start(suite.serverCtx)
	require.NoError(suite.T(), err)

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	httpAddr := server.GetHTTPAddress()
	client := &http.Client{Timeout: 5 * time.Second}

	// Make concurrent HTTP requests
	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := client.Get(fmt.Sprintf("http://%s/test", httpAddr))
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				return
			}

			results <- nil
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(suite.T(), err)
	}
}

// TestConfigurationValidation tests various configuration scenarios
func (suite *IntegrationTestSuite) TestConfigurationValidation() {
	// Test HTTP-only configuration
	httpOnlyConfig := &ServerConfig{
		ServiceName: "http-only-service",
		HTTP: HTTPConfig{
			Port:         suite.findAvailablePort(),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:    "", // Disabled
			Enabled: false,
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled: false,
		},
	}

	// Create HTTP-only service registrar
	httpOnlyService := NewTestServiceRegistrar()
	httpHandler := NewTestHTTPHandler("http-only-test")
	httpHandler.AddRoute("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "http only test"})
	})
	httpOnlyService.AddHTTPHandler(httpHandler)
	httpOnlyService.AddHealthCheck(NewTestHealthCheck("http-only-service", true))

	server, err := NewBusinessServerCore(httpOnlyConfig, httpOnlyService, suite.testDeps)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), server.config.IsHTTPEnabled())
	assert.False(suite.T(), server.config.IsGRPCEnabled())
	assert.NotNil(suite.T(), server.httpTransport)
	assert.Nil(suite.T(), server.grpcTransport)

	// Test gRPC-only configuration
	grpcOnlyConfig := &ServerConfig{
		ServiceName: "grpc-only-service",
		HTTP: HTTPConfig{
			Port:    "", // Disabled
			Enabled: false,
		},
		GRPC: GRPCConfig{
			Port:                suite.findAvailablePort(),
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled: false,
		},
	}

	// Create gRPC-only service registrar
	grpcOnlyService := NewTestServiceRegistrar()
	grpcService := NewTestGRPCService("grpc-only-test")
	grpcOnlyService.AddGRPCService(grpcService)
	grpcOnlyService.AddHealthCheck(NewTestHealthCheck("grpc-only-service", true))

	server, err = NewBusinessServerCore(grpcOnlyConfig, grpcOnlyService, suite.testDeps)
	require.NoError(suite.T(), err)

	assert.False(suite.T(), server.config.IsHTTPEnabled())
	assert.True(suite.T(), server.config.IsGRPCEnabled())
	assert.Nil(suite.T(), server.httpTransport)
	assert.NotNil(suite.T(), server.grpcTransport)
}

// Run the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// TestBaseServerIntegrationWithRealTransports tests the base server with real transport implementations
func TestBaseServerIntegrationWithRealTransports(t *testing.T) {
	// Initialize logger for test
	logger.InitLogger()
	gin.SetMode(gin.TestMode)

	// Find available ports
	httpPort := findAvailablePort(t)
	grpcPort := findAvailablePort(t)

	// Create configuration
	config := &ServerConfig{
		ServiceName: "integration-test-service",
		HTTP: HTTPConfig{
			Port:         httpPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                grpcPort,
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled: false, // Disable for integration test
		},
		Middleware: MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Create test service registrar
	registrar := NewTestServiceRegistrar()

	// Add HTTP handler
	httpHandler := NewTestHTTPHandler("integration-test")
	httpHandler.AddRoute("/integration", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "integration test successful",
			"service": "base-server-framework",
		})
	})
	registrar.AddHTTPHandler(httpHandler)

	// Add gRPC service
	grpcService := NewTestGRPCService("integration-grpc")
	registrar.AddGRPCService(grpcService)

	// Create dependencies
	deps := NewTestDependencyContainer()
	deps.AddService("integration-db", "integration-database")

	// Create and start server
	server, err := NewBusinessServerCore(config, registrar, deps)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)

	// Verify server is running
	assert.True(t, server.started)
	assert.True(t, deps.IsInitialized())

	// Test HTTP transport
	httpAddr := server.GetHTTPAddress()
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("http://%s/integration", httpAddr))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test gRPC transport
	grpcAddr := server.GetGRPCAddress()
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()

	healthResp, err := healthClient.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, healthResp.Status)

	// Test graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	err = server.Stop(shutdownCtx)
	require.NoError(t, err)

	err = server.Shutdown()
	require.NoError(t, err)

	// Verify cleanup
	assert.False(t, server.started)
	assert.True(t, deps.IsClosed())
}

// Helper function to find available port
func findAvailablePort(t *testing.T) string {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}
