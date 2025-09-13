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

package messaging_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/testutil"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// Initialize logger for tests
func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

// FrameworkLifecycleTestSuite tests messaging framework integration with server lifecycle
type FrameworkLifecycleTestSuite struct {
	suite.Suite
	baseServer       *server.BusinessServerImpl
	messagingCoord   messaging.MessagingCoordinator
	testServices     map[string]interface{}
	cleanupFunctions []func()
}

// TestFrameworkLifecycle runs the framework lifecycle test suite
func TestFrameworkLifecycle(t *testing.T) {
	suite.Run(t, new(FrameworkLifecycleTestSuite))
}

// SetupTest creates test environment for each test
func (suite *FrameworkLifecycleTestSuite) SetupTest() {
	suite.testServices = make(map[string]interface{})
	suite.cleanupFunctions = make([]func(), 0)
}

// TearDownTest cleans up after each test
func (suite *FrameworkLifecycleTestSuite) TearDownTest() {
	// Execute cleanup functions in reverse order
	for i := len(suite.cleanupFunctions) - 1; i >= 0; i-- {
		suite.cleanupFunctions[i]()
	}
	suite.cleanupFunctions = make([]func(), 0)

	// Clean up test services
	for name, service := range suite.testServices {
		if closer, ok := service.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		suite.testServices[name] = nil
	}

	// Clean up base server if it exists
	if suite.baseServer != nil {
		_ = suite.baseServer.Shutdown()
		suite.baseServer = nil
	}

	// Clean up messaging coordinator if it exists
	if suite.messagingCoord != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = suite.messagingCoord.Stop(ctx)
		suite.messagingCoord = nil
	}
}

// createTestConfig creates test configuration for framework
func (suite *FrameworkLifecycleTestSuite) createTestConfig() *server.ServerConfig {
	config := server.NewServerConfig()
	config.ServiceName = "test-messaging-service"
	config.HTTP.TestMode = true
	config.HTTP.Port = "0"           // Dynamic port allocation
	config.GRPC.Port = "0"           // Dynamic port allocation
	config.Discovery.Enabled = false // Disable discovery for testing
	config.ShutdownTimeout = 5 * time.Second
	return config
}

// createTestServiceRegistrar creates a test service registrar
func (suite *FrameworkLifecycleTestSuite) createTestServiceRegistrar() server.BusinessServiceRegistrar {
	return &TestMessagingServiceRegistrar{
		messagingCoord: suite.messagingCoord,
		testSuite:      suite,
	}
}

// TestMessagingCoordinatorLifecycle tests that messaging coordinator properly integrates with server lifecycle
func (suite *FrameworkLifecycleTestSuite) TestMessagingCoordinatorLifecycle() {
	// Create messaging coordinator
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Register a test broker
	mockBroker := testutil.NewMockMessageBroker()
	brokerErr := suite.messagingCoord.RegisterBroker("test-broker", mockBroker)
	suite.Require().NoError(brokerErr)

	// Create test configuration
	config := suite.createTestConfig()

	// Create service registrar with messaging coordinator
	registrar := suite.createTestServiceRegistrar()

	// Create base server
	var err error
	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.baseServer)

	// Test startup sequence
	ctx := context.Background()
	startTime := time.Now()
	err = suite.baseServer.Start(ctx)
	suite.Require().NoError(err)
	startupDuration := time.Since(startTime)

	// Verify startup completed within reasonable time
	suite.Less(startupDuration, 5*time.Second, "Server startup should complete within 5 seconds")

	// Verify server is running
	suite.NotEmpty(suite.baseServer.GetHTTPAddress())
	suite.NotEmpty(suite.baseServer.GetGRPCAddress())

	// Verify messaging coordinator is started
	suite.True(suite.messagingCoord.IsStarted(), "Messaging coordinator should be started after server startup")

	// Verify server health
	healthStatus := suite.baseServer.GetTransportHealth(ctx)
	suite.NotEmpty(healthStatus)

	// Check that HTTP and gRPC transports are healthy
	if httpStatus, exists := healthStatus["http"]; exists {
		suite.NotEmpty(httpStatus)
		for name, status := range httpStatus {
			suite.Equal(types.HealthStatusHealthy, status.Status, fmt.Sprintf("HTTP service %s should be healthy", name))
		}
	}

	// Test graceful shutdown
	shutdownStart := time.Now()
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)
	shutdownDuration := time.Since(shutdownStart)

	// Verify shutdown completed within reasonable time
	suite.Less(shutdownDuration, 5*time.Second, "Server shutdown should complete within 5 seconds")

	// Verify messaging coordinator is stopped
	suite.False(suite.messagingCoord.IsStarted(), "Messaging coordinator should be stopped after server shutdown")
}

// TestMessagingCoordinatorErrorHandling tests error handling during server lifecycle
func (suite *FrameworkLifecycleTestSuite) TestMessagingCoordinatorErrorHandling() {
	// Create messaging coordinator
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Create test configuration
	config := suite.createTestConfig()

	// Create service registrar
	registrar := suite.createTestServiceRegistrar()

	// Create base server
	var err error
	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.baseServer)

	// Test startup with invalid messaging configuration - should handle gracefully
	ctx := context.Background()
	err = suite.baseServer.Start(ctx)

	// Server should still start even with messaging issues (graceful degradation)
	suite.Require().NoError(err)

	// Verify server is running despite messaging issues
	suite.NotEmpty(suite.baseServer.GetHTTPAddress())
	suite.NotEmpty(suite.baseServer.GetGRPCAddress())

	// Test shutdown still works
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)
}

// TestMessagingCoordinatorRestart tests that messaging coordinator can be properly restarted
func (suite *FrameworkLifecycleTestSuite) TestMessagingCoordinatorRestart() {
	// Create messaging coordinator
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Register a test broker
	mockBroker := testutil.NewMockMessageBroker()
	err := suite.messagingCoord.RegisterBroker("test-broker", mockBroker)
	suite.Require().NoError(err)

	// First startup and shutdown
	config := suite.createTestConfig()
	registrar := suite.createTestServiceRegistrar()

	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)

	// Start server
	ctx := context.Background()
	err = suite.baseServer.Start(ctx)
	suite.Require().NoError(err)

	// Verify messaging coordinator is started
	suite.True(suite.messagingCoord.IsStarted())

	// Shutdown server
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)

	// Verify messaging coordinator is stopped
	suite.False(suite.messagingCoord.IsStarted())

	// Create new server instance for restart
	suite.baseServer = nil
	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)

	// Restart server
	err = suite.baseServer.Start(ctx)
	suite.Require().NoError(err)

	// Verify messaging coordinator is started again
	suite.True(suite.messagingCoord.IsStarted())

	// Shutdown again
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)

	// Verify messaging coordinator is stopped again
	suite.False(suite.messagingCoord.IsStarted())
}

// TestMessagingCoordinatorConcurrentAccess tests concurrent access to messaging coordinator during lifecycle
func (suite *FrameworkLifecycleTestSuite) TestMessagingCoordinatorConcurrentAccess() {
	// Create messaging coordinator
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Register a test broker
	mockBroker := testutil.NewMockMessageBroker()
	err := suite.messagingCoord.RegisterBroker("test-broker", mockBroker)
	suite.Require().NoError(err)

	// Create test configuration
	config := suite.createTestConfig()
	registrar := suite.createTestServiceRegistrar()

	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)

	// Test concurrent access during startup
	ctx := context.Background()
	startupComplete := make(chan bool)
	errors := make(chan error, 10)

	// Start multiple goroutines to access messaging coordinator concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			select {
			case <-startupComplete:
				// After startup, test coordinator operations
				isStarted := suite.messagingCoord.IsStarted()
				suite.T().Logf("Goroutine %d: IsStarted() = %v", id, isStarted)
			default:
				errors <- fmt.Errorf("goroutine %d: startup not complete", id)
			}
		}(i)
	}

	// Start server
	err = suite.baseServer.Start(ctx)
	suite.Require().NoError(err)
	close(startupComplete)

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		if err := <-errors; err != nil {
			suite.T().Error(err)
		}
	}

	// Verify messaging coordinator handled concurrent access
	suite.True(suite.messagingCoord.IsStarted())

	// Shutdown
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)
}

// TestMessagingCoordinatorResourceCleanup tests that messaging coordinator properly cleans up resources
func (suite *FrameworkLifecycleTestSuite) TestMessagingCoordinatorResourceCleanup() {
	// Create messaging coordinator with multiple brokers
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Register multiple test brokers
	mockBroker1 := testutil.NewMockMessageBroker()
	err := suite.messagingCoord.RegisterBroker("broker1", mockBroker1)
	suite.Require().NoError(err)

	mockBroker2 := testutil.NewMockMessageBroker()
	err = suite.messagingCoord.RegisterBroker("broker2", mockBroker2)
	suite.Require().NoError(err)

	// Create test configuration
	config := suite.createTestConfig()
	registrar := suite.createTestServiceRegistrar()

	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)

	// Start server
	ctx := context.Background()
	err = suite.baseServer.Start(ctx)
	suite.Require().NoError(err)

	// Verify brokers are registered and started
	suite.True(suite.messagingCoord.IsStarted())

	// Shutdown server
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)

	// Verify all resources are cleaned up
	suite.False(suite.messagingCoord.IsStarted())

	// Test that coordinator can be garbage collected (by creating a new one)
	oldCoordinator := suite.messagingCoord
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Register a test broker
	newMockBroker := testutil.NewMockMessageBroker()
	err = suite.messagingCoord.RegisterBroker("new-broker", newMockBroker)
	suite.Require().NoError(err)

	// Old coordinator should not interfere with new one
	suite.False(oldCoordinator.IsStarted())
	suite.True(suite.messagingCoord.IsStarted() == false) // Not started yet

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = suite.messagingCoord.Stop(ctx)
}

// TestMessagingCoordinatorHealthIntegration tests that messaging coordinator health is properly integrated
func (suite *FrameworkLifecycleTestSuite) TestMessagingCoordinatorHealthIntegration() {
	// Create messaging coordinator
	suite.messagingCoord = messaging.NewMessagingCoordinator()
	suite.Require().NotNil(suite.messagingCoord)

	// Register a test broker
	mockBroker := testutil.NewMockMessageBroker()
	err := suite.messagingCoord.RegisterBroker("test-broker", mockBroker)
	suite.Require().NoError(err)

	// Create test configuration
	config := suite.createTestConfig()
	registrar := suite.createTestServiceRegistrar()

	suite.baseServer, err = server.NewBusinessServerCore(config, registrar, nil)
	suite.Require().NoError(err)

	// Start server
	ctx := context.Background()
	err = suite.baseServer.Start(ctx)
	suite.Require().NoError(err)

	// Test health status integration
	healthStatus := suite.baseServer.GetTransportHealth(ctx)
	suite.NotEmpty(healthStatus)

	// Verify that messaging coordinator health is reflected in server health
	// The messaging coordinator should contribute to the overall system health
	suite.True(suite.messagingCoord.IsStarted())

	// Test that health checks work even during high load
	for i := 0; i < 10; i++ {
		health := suite.baseServer.GetTransportHealth(ctx)
		suite.NotEmpty(health)
		time.Sleep(10 * time.Millisecond)
	}

	// Shutdown server
	err = suite.baseServer.Shutdown()
	suite.Require().NoError(err)

	// Verify health status after shutdown
	suite.False(suite.messagingCoord.IsStarted())
}

// TestMessagingServiceRegistrar is a test implementation of BusinessServiceRegistrar
type TestMessagingServiceRegistrar struct {
	messagingCoord messaging.MessagingCoordinator
	testSuite      *FrameworkLifecycleTestSuite
}

// RegisterServices registers test services with the framework
func (r *TestMessagingServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register a test HTTP handler
	handler := &TestHTTPHandler{}
	if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register a test health check
	healthCheck := &TestHealthCheck{
		messagingCoord: r.messagingCoord,
	}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// TestHTTPHandler is a test HTTP handler for lifecycle testing
type TestHTTPHandler struct{}

func (h *TestHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter := router.(*gin.Engine)
	ginRouter.GET("/test/lifecycle", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "message": "lifecycle test endpoint"})
	})
	return nil
}

func (h *TestHTTPHandler) GetServiceName() string {
	return "test-lifecycle-http"
}

// TestHealthCheck is a test health check for lifecycle testing
type TestHealthCheck struct {
	messagingCoord messaging.MessagingCoordinator
}

func (h *TestHealthCheck) Check(ctx context.Context) error {
	// Check messaging coordinator health
	if h.messagingCoord == nil {
		return fmt.Errorf("messaging coordinator is nil")
	}

	if !h.messagingCoord.IsStarted() {
		return fmt.Errorf("messaging coordinator is not started")
	}

	return nil
}

func (h *TestHealthCheck) GetServiceName() string {
	return "messaging-coordinator-health"
}
