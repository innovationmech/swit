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
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/innovationmech/swit/internal/testing"
	"github.com/innovationmech/swit/pkg/messaging"
)

// MessagingFrameworkIntegrationTestSuite tests integration between messaging coordinator and server framework
type MessagingFrameworkIntegrationTestSuite struct {
	suite.Suite
	integrationSuite *testing.MessagingIntegrationTestSuite
}

// SetupSuite initializes the test suite
func (suite *MessagingFrameworkIntegrationTestSuite) SetupSuite() {
	suite.integrationSuite = testing.NewMessagingIntegrationTestSuite()
	suite.integrationSuite.SetupSuite()
}

// TearDownSuite cleans up after the test suite
func (suite *MessagingFrameworkIntegrationTestSuite) TearDownSuite() {
	suite.integrationSuite.TearDownSuite()
}

// SetupTest sets up each test case
func (suite *MessagingFrameworkIntegrationTestSuite) SetupTest() {
	suite.integrationSuite.SetupTest()
}

// TearDownTest cleans up after each test case
func (suite *MessagingFrameworkIntegrationTestSuite) TearDownTest() {
	suite.integrationSuite.TearDownTest()
}

// TestMessagingCoordinatorWithServerLifecycle tests messaging coordinator integration with server lifecycle
func (suite *MessagingFrameworkIntegrationTestSuite) TestMessagingCoordinatorWithServerLifecycle() {
	// Test: Messaging coordinator should be properly integrated with server startup/shutdown
	
	// Setup mock broker expectations
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	// Setup handler expectations
	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Start messaging coordinator
	err := suite.integrationSuite.StartMessagingCoordinator()
	require.NoError(suite.T(), err)
	assert.True(suite.T(), suite.integrationSuite.MessagingCoord.IsStarted())

	// Start server with messaging integration
	err = suite.integrationSuite.StartMessagingServer()
	require.NoError(suite.T(), err)
	assert.True(suite.T(), suite.integrationSuite.BaseServer.IsStarted())

	// Verify messaging coordinator is accessible through dependencies
	messagingCoord, err := suite.integrationSuite.TestDependencies.GetService("messaging-coordinator")
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), messagingCoord)
	assert.Equal(suite.T(), suite.integrationSuite.MessagingCoord, messagingCoord)

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop server first
	err = suite.integrationSuite.BaseServer.Stop(ctx)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), suite.integrationSuite.BaseServer.IsStarted())

	// Then stop messaging coordinator
	err = suite.integrationSuite.MessagingCoord.Stop(ctx)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), suite.integrationSuite.MessagingCoord.IsStarted())

	// Verify all mocks were called as expected
	suite.integrationSuite.TestBroker.AssertExpectations(suite.T())
	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.AssertExpectations(suite.T())
	}
}

// TestMessagingHealthCheckIntegration tests health check integration between messaging and server
func (suite *MessagingFrameworkIntegrationTestSuite) TestMessagingHealthCheckIntegration() {
	// Test: Server health checks should include messaging coordinator health

	// Setup mock broker with healthy status
	suite.integrationSuite.TestBroker.SetHealthStatus(messaging.HealthStatusHealthy, "All systems operational")
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})
	suite.integrationSuite.TestBroker.On("HealthCheck", mock.Anything).Return(
		&messaging.HealthStatus{Status: messaging.HealthStatusHealthy, Message: "All systems operational"}, nil)

	// Setup handler expectations
	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Start full integration
	err := suite.integrationSuite.StartFullIntegration()
	require.NoError(suite.T(), err)

	// Test server transport health includes messaging coordinator
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := suite.integrationSuite.BaseServer.GetTransportHealth(ctx)
	assert.NotEmpty(suite.T(), health)

	// Verify HTTP transport health is present
	httpHealth, exists := health["http"]
	assert.True(suite.T(), exists)
	assert.NotEmpty(suite.T(), httpHealth)

	// Verify gRPC transport health is present
	grpcHealth, exists := health["grpc"]
	assert.True(suite.T(), exists)
	assert.NotEmpty(suite.T(), grpcHealth)

	// Test messaging coordinator health check
	messagingHealth, err := suite.integrationSuite.MessagingCoord.HealthCheck(ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", messagingHealth.Overall)
	assert.Equal(suite.T(), "healthy", messagingHealth.BrokerHealth["test-broker"])

	// Shutdown gracefully
	err = suite.integrationSuite.BaseServer.Stop(ctx)
	require.NoError(suite.T(), err)
	err = suite.integrationSuite.MessagingCoord.Stop(ctx)
	require.NoError(suite.T(), err)
}

// TestServiceRegistrationWithMessagingHandlers tests service registration with messaging handlers
func (suite *MessagingFrameworkIntegrationTestSuite) TestServiceRegistrationWithMessagingHandlers() {
	// Test: Service registrar should properly register both HTTP/gRPC services and messaging handlers

	// Add HTTP handler to the service registrar
	httpHandler := &TestHTTPHandler{
		serviceName: "integration-http-service",
		routes:      make(map[string]gin.HandlerFunc),
	}
	httpHandler.AddRoute("/integration-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "integration test successful",
			"service": "messaging-integration",
		})
	})
	suite.integrationSuite.ServiceRegistrar.AddHTTPHandler(httpHandler)

	// Add gRPC service
	grpcService := &TestGRPCService{
		serviceName: "integration-grpc-service",
	}
	suite.integrationSuite.ServiceRegistrar.AddGRPCService(grpcService)

	// Add health check
	healthCheck := &TestHealthCheck{
		serviceName: "integration-service",
		healthy:     true,
	}
	suite.integrationSuite.ServiceRegistrar.AddHealthCheck(healthCheck)

	// Setup messaging expectations
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Start full integration
	err := suite.integrationSuite.StartFullIntegration()
	require.NoError(suite.T(), err)

	// Verify all services are registered and accessible
	httpAddr := suite.integrationSuite.BaseServer.GetHTTPAddress()
	client := suite.integrationSuite.GetHTTPClient()

	// Test HTTP endpoint
	resp, err := client.Get(fmt.Sprintf("http://%s/integration-test", httpAddr))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Test gRPC connection
	grpcConn, err := suite.integrationSuite.GetGRPCConnection()
	require.NoError(suite.T(), err)
	defer grpcConn.Close()

	healthClient := grpc_health_v1.NewHealthClient(grpcConn)
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()

	healthResp, err := healthClient.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), grpc_health_v1.HealthCheckResponse_SERVING, healthResp.Status)

	// Verify messaging handlers are registered
	messagingHandlers := suite.integrationSuite.ServiceRegistrar.GetMessagingHandlers()
	assert.Len(suite.T(), messagingHandlers, 3) // user, order, notification handlers

	for _, handler := range messagingHandlers {
		mockHandler, ok := handler.(*testing.MockEventHandler)
		require.True(suite.T(), ok, "Handler should be MockEventHandler")
		assert.True(suite.T(), mockHandler.IsInitialized())
	}

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = suite.integrationSuite.BaseServer.Stop(ctx)
	require.NoError(suite.T(), err)
	err = suite.integrationSuite.MessagingCoord.Stop(ctx)
	require.NoError(suite.T(), err)
}

// TestDependencyInjectionIntegration tests dependency injection for messaging components
func (suite *MessagingFrameworkIntegrationTestSuite) TestDependencyInjectionIntegration() {
	// Test: Messaging coordinator should be properly injectable through dependency container

	// Setup messaging expectations
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Start messaging coordinator
	err := suite.integrationSuite.StartMessagingCoordinator()
	require.NoError(suite.T(), err)

	// Inject messaging coordinator into dependencies
	suite.integrationSuite.TestDependencies.AddService("messaging-coordinator", suite.integrationSuite.MessagingCoord)

	// Create service that depends on messaging coordinator
	messagingDependentService := &MessagingDependentService{
		dependencyContainer: suite.integrationSuite.TestDependencies,
	}
	suite.integrationSuite.ServiceRegistrar.AddHTTPHandler(messagingDependentService)

	// Start server
	err = suite.integrationSuite.StartMessagingServer()
	require.NoError(suite.T(), err)

	// Test that the service can access messaging coordinator through dependency injection
	httpAddr := suite.integrationSuite.BaseServer.GetHTTPAddress()
	client := suite.integrationSuite.GetHTTPClient()

	resp, err := client.Get(fmt.Sprintf("http://%s/messaging-status", httpAddr))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = suite.integrationSuite.BaseServer.Stop(ctx)
	require.NoError(suite.T(), err)
	err = suite.integrationSuite.MessagingCoord.Stop(ctx)
	require.NoError(suite.T(), err)
}

// TestConfigurationIntegration tests configuration integration for messaging components
func (suite *MessagingFrameworkIntegrationTestSuite) TestConfigurationIntegration() {
	// Test: Server configuration should properly integrate with messaging coordinator

	// Create custom configuration with messaging settings
	customConfig := &ServerConfig{
		ServiceName: "messaging-config-test",
		HTTP: HTTPConfig{
			Port:         suite.integrationSuite.HTTPPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                suite.integrationSuite.GRPCPort,
			EnableKeepalive:     true,
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024,
			MaxSendMsgSize:      4 * 1024 * 1024,
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled: false,
		},
	}

	suite.integrationSuite.ServerConfig = customConfig

	// Setup messaging expectations
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Start full integration with custom configuration
	err := suite.integrationSuite.StartFullIntegration()
	require.NoError(suite.T(), err)

	// Verify server configuration is applied
	assert.Equal(suite.T(), "messaging-config-test", suite.integrationSuite.BaseServer.GetConfig().ServiceName)
	assert.Equal(suite.T(), suite.integrationSuite.HTTPPort, suite.integrationSuite.BaseServer.GetConfig().HTTP.Port)
	assert.Equal(suite.T(), suite.integrationSuite.GRPCPort, suite.integrationSuite.BaseServer.GetConfig().GRPC.Port)

	// Verify messaging coordinator is running
	assert.True(suite.T(), suite.integrationSuite.MessagingCoord.IsStarted())

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = suite.integrationSuite.BaseServer.Stop(ctx)
	require.NoError(suite.T(), err)
	err = suite.integrationSuite.MessagingCoord.Stop(ctx)
	require.NoError(suite.T(), err)
}

// TestConcurrentOperations tests concurrent operations across server and messaging components
func (suite *MessagingFrameworkIntegrationTestSuite) TestConcurrentOperations() {
	// Test: Server and messaging coordinator should handle concurrent operations safely

	// Setup messaging expectations
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	for _, handler := range suite.integrationSuite.TestHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Start full integration
	err := suite.integrationSuite.StartFullIntegration()
	require.NoError(suite.T(), err)

	// Test concurrent HTTP requests
	const numRequests = 20
	results := make(chan error, numRequests)
	httpAddr := suite.integrationSuite.BaseServer.GetHTTPAddress()
	client := suite.integrationSuite.GetHTTPClient()

	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			resp, err := client.Get(fmt.Sprintf("http://%s/ready", httpAddr))
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
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(suite.T(), err)
	}

	// Test concurrent messaging coordinator operations
	const numOperations = 10
	operationResults := make(chan error, numOperations)

	for i := 0; i < numOperations; i++ {
		go func(opID int) {
			// Test concurrent metrics access
			metrics := suite.integrationSuite.MessagingCoord.GetMetrics()
			if metrics == nil {
				operationResults <- fmt.Errorf("metrics should not be nil")
				return
			}

			// Test concurrent health checks
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			health, err := suite.integrationSuite.MessagingCoord.HealthCheck(ctx)
			if err != nil {
				operationResults <- fmt.Errorf("health check failed: %w", err)
				return
			}

			if health.Overall != "healthy" {
				operationResults <- fmt.Errorf("expected healthy status, got: %s", health.Overall)
				return
			}

			operationResults <- nil
		}(i)
	}

	// Collect operation results
	for i := 0; i < numOperations; i++ {
		err := <-operationResults
		assert.NoError(suite.T(), err)
	}

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = suite.integrationSuite.BaseServer.Stop(ctx)
	require.NoError(suite.T(), err)
	err = suite.integrationSuite.MessagingCoord.Stop(ctx)
	require.NoError(suite.T(), err)
}

// TestErrorHandlingIntegration tests error handling integration between server and messaging
func (suite *MessagingFrameworkIntegrationTestSuite) TestErrorHandlingIntegration() {
	// Test: Error handling should be properly integrated between server and messaging components

	// Setup broker to fail on connection
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(fmt.Errorf("connection failed"))
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	// Test that messaging coordinator startup fails
	err := suite.integrationSuite.StartMessagingCoordinator()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to start broker")
	assert.False(suite.T(), suite.integrationSuite.MessagingCoord.IsStarted())

	// Reset broker for successful scenario
	suite.integrationSuite.TestBroker = testing.NewMockMessageBroker()
	suite.integrationSuite.TestBroker.On("Connect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("CreateSubscriber", mock.Anything).Return(&testing.MockEventSubscriber{}, nil)
	suite.integrationSuite.TestBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.integrationSuite.TestBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	// Setup handler to fail on initialization
	suite.integrationSuite.TestHandlers[0].On("Initialize", mock.Anything).Return(fmt.Errorf("initialization failed"))
	for i := 1; i < len(suite.integrationSuite.TestHandlers); i++ {
		suite.integrationSuite.TestHandlers[i].On("Initialize", mock.Anything).Return(nil)
		suite.integrationSuite.TestHandlers[i].On("Shutdown", mock.Anything).Return(nil)
	}

	// Test that messaging coordinator startup fails due to handler initialization
	err = suite.integrationSuite.StartMessagingCoordinator()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to initialize handler")
	assert.False(suite.T(), suite.integrationSuite.MessagingCoord.IsStarted())
}

// MessagingDependentService is a service that depends on messaging coordinator
type MessagingDependentService struct {
	dependencyContainer *testing.TestDependencyContainer
	serviceName         string
	routes              map[string]gin.HandlerFunc
}

// NewMessagingDependentService creates a new messaging dependent service
func NewMessagingDependentService(deps *testing.TestDependencyContainer) *MessagingDependentService {
	return &MessagingDependentService{
		dependencyContainer: deps,
		serviceName:         "messaging-dependent-service",
		routes:              make(map[string]gin.HandlerFunc),
	}
}

// RegisterRoutes implements BusinessHTTPHandler
func (s *MessagingDependentService) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Add messaging status endpoint
	ginRouter.GET("/messaging-status", s.getMessagingStatus)

	// Add other routes
	for path, handler := range s.routes {
		ginRouter.GET(path, handler)
	}

	return nil
}

// GetServiceName implements BusinessHTTPHandler
func (s *MessagingDependentService) GetServiceName() string {
	return s.serviceName
}

// AddRoute adds a route to the service
func (s *MessagingDependentService) AddRoute(path string, handler gin.HandlerFunc) {
	s.routes[path] = handler
}

// getMessagingStatus handles requests to check messaging status
func (s *MessagingDependentService) getMessagingStatus(c *gin.Context) {
	// Get messaging coordinator from dependencies
	messagingCoord, err := s.dependencyContainer.GetService("messaging-coordinator")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "messaging coordinator not available",
		})
		return
	}

	coordinator, ok := messagingCoord.(messaging.MessagingCoordinator)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid messaging coordinator type",
		})
		return
	}

	// Get messaging status
	metrics := coordinator.GetMetrics()
	health, err := coordinator.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "messaging health check failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messaging_status": "healthy",
		"brokers":         metrics.BrokerCount,
		"handlers":        metrics.HandlerCount,
		"overall_health":  health.Overall,
		"uptime":          metrics.Uptime.String(),
	})
}

// Run the messaging framework integration test suite
func TestMessagingFrameworkIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MessagingFrameworkIntegrationTestSuite))
}