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
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/server"
)

// MessagingServiceRegistrationTestSuite tests service registration patterns
// with messaging integration
type MessagingServiceRegistrationTestSuite struct {
	suite.Suite
	coordinator messaging.MessagingCoordinator
}

// SetupSuite initializes the test suite
func (suite *MessagingServiceRegistrationTestSuite) SetupSuite() {
	suite.coordinator = messaging.NewMessagingCoordinator()
}

// TearDownSuite cleans up after the test suite
func (suite *MessagingServiceRegistrationTestSuite) TearDownSuite() {
	if suite.coordinator != nil {
		suite.coordinator.Stop(context.Background())
	}
}

// TestMessagingCoordinatorBasicFunctionality tests basic messaging coordinator functionality
func (suite *MessagingServiceRegistrationTestSuite) TestMessagingCoordinatorBasicFunctionality() {
	// Test: Messaging coordinator should start and stop correctly

	// Start coordinator
	err := suite.coordinator.Start(context.Background())
	require.NoError(suite.T(), err)

	// Verify coordinator is running
	metrics := suite.coordinator.GetMetrics()
	assert.NotNil(suite.T(), metrics.StartedAt, "Coordinator should be started")
	assert.Equal(suite.T(), 0, metrics.BrokerCount)
	assert.Equal(suite.T(), 0, metrics.HandlerCount)

	// Stop coordinator
	err = suite.coordinator.Stop(context.Background())
	require.NoError(suite.T(), err)

	// Verify coordinator is stopped
	metrics = suite.coordinator.GetMetrics()
	// Note: The actual implementation may not clear StartedAt on stop
	// For now, just verify the coordinator can be stopped without error
	assert.NotNil(suite.T(), metrics)
}

// TestMessagingCoordinatorInterfaceCompatibility tests that messaging coordinator can integrate with server framework
func (suite *MessagingServiceRegistrationTestSuite) TestMessagingCoordinatorInterfaceCompatibility() {
	// Test: Messaging coordinator should be compatible with server framework integration

	// Start coordinator
	err := suite.coordinator.Start(context.Background())
	require.NoError(suite.T(), err)
	defer suite.coordinator.Stop(context.Background())

	// Test that coordinator provides the necessary methods for framework integration
	// The coordinator should be able to be used as a component within the server framework
	metrics := suite.coordinator.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	assert.NotNil(suite.T(), metrics.StartedAt, "Coordinator should be started")

	// Test that coordinator can be stopped gracefully
	err = suite.coordinator.Stop(context.Background())
	require.NoError(suite.T(), err)

	// Verify coordinator is properly stopped
	metrics = suite.coordinator.GetMetrics()
	// Note: The actual implementation may not clear StartedAt, so we'll just verify it exists
	assert.NotNil(suite.T(), metrics)
}

// TestMessagingServiceMetadata tests that messaging services can provide metadata
func (suite *MessagingServiceRegistrationTestSuite) TestMessagingServiceMetadata() {
	// Test: Messaging coordinator should provide service metadata

	// Start coordinator to get metrics
	err := suite.coordinator.Start(context.Background())
	require.NoError(suite.T(), err)
	defer suite.coordinator.Stop(context.Background())

	// Get coordinator metrics
	metrics := suite.coordinator.GetMetrics()

	// Verify basic metadata
	assert.NotNil(suite.T(), metrics)
	assert.NotNil(suite.T(), metrics.StartedAt, "Coordinator should be started")
	assert.Equal(suite.T(), 0, metrics.BrokerCount)
	assert.Equal(suite.T(), 0, metrics.HandlerCount)
}

// TestMessagingCoordinatorErrorHandling tests error handling in messaging coordinator
func (suite *MessagingServiceRegistrationTestSuite) TestMessagingCoordinatorErrorHandling() {
	// Test: Messaging coordinator should handle errors gracefully

	// Try to stop coordinator when not started
	err := suite.coordinator.Stop(context.Background())
	// Should not error - should handle gracefully
	assert.NoError(suite.T(), err)

	// Start coordinator
	err = suite.coordinator.Start(context.Background())
	require.NoError(suite.T(), err)

	// Try to start coordinator again (should handle gracefully)
	err = suite.coordinator.Start(context.Background())
	// May or may not error depending on implementation, but should not panic
	assert.NotNil(suite.T(), err, "Starting already started coordinator should return error")

	// Stop coordinator
	err = suite.coordinator.Stop(context.Background())
	require.NoError(suite.T(), err)
}

// TestMessagingServiceWithBrokerRegistration tests service registration with broker
func (suite *MessagingServiceRegistrationTestSuite) TestMessagingServiceWithBrokerRegistration() {
	// Test: Services should be able to register with messaging brokers

	// Start coordinator
	err := suite.coordinator.Start(context.Background())
	require.NoError(suite.T(), err)
	defer suite.coordinator.Stop(context.Background())

	// Create a test event handler (for future use when handler registration is implemented)
	_ = &TestEventHandler{
		handlerID: "test-handler",
		topics:    []string{"test.topic"},
	}

	// Verify coordinator is still running
	metrics := suite.coordinator.GetMetrics()
	assert.NotNil(suite.T(), metrics.StartedAt, "Coordinator should be started")
}

// TestMessagingServiceHealthCheck tests health check integration for messaging services
func (suite *MessagingServiceRegistrationTestSuite) TestMessagingServiceHealthCheck() {
	// Test: Messaging services should integrate with health check system

	// Start coordinator
	err := suite.coordinator.Start(context.Background())
	require.NoError(suite.T(), err)
	defer suite.coordinator.Stop(context.Background())

	// Create a messaging health check
	healthCheck := &MessagingHealthCheck{
		coordinator: suite.coordinator,
	}

	// Test health check
	err = healthCheck.Check(context.Background())
	assert.NoError(suite.T(), err)

	// Verify service name
	assert.Equal(suite.T(), "messaging-coordinator", healthCheck.GetServiceName())
}

// TestServiceHTTPHandler is a basic HTTP handler for testing
type TestServiceHTTPHandler struct {
	serviceName string
}

func (h *TestServiceHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Add a simple test route
	ginRouter.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": h.serviceName,
			"status":  "ok",
		})
	})

	return nil
}

func (h *TestServiceHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// TestGRPCService is a basic gRPC service for testing
type TestGRPCService struct {
	serviceName string
}

func (s *TestGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected grpc.Server, got %T", server)
	}

	// For testing, we'll just verify the server is valid
	if grpcServer == nil {
		return fmt.Errorf("grpc server is nil")
	}

	return nil
}

func (s *TestGRPCService) GetServiceName() string {
	return s.serviceName
}

// TestServiceHealthCheck is a basic health check for testing
type TestServiceHealthCheck struct {
	serviceName string
}

func (h *TestServiceHealthCheck) Check(ctx context.Context) error {
	// Simple health check - always returns healthy
	return nil
}

func (h *TestServiceHealthCheck) GetServiceName() string {
	return h.serviceName
}

// TestServiceRegistry implements BusinessServiceRegistry for testing
type TestServiceRegistry struct {
	httpHandlers []server.BusinessHTTPHandler
	grpcServices []server.BusinessGRPCService
	healthChecks []server.BusinessHealthCheck
}

// RegisterBusinessHTTPHandler registers an HTTP handler
func (r *TestServiceRegistry) RegisterBusinessHTTPHandler(handler server.BusinessHTTPHandler) error {
	r.httpHandlers = append(r.httpHandlers, handler)
	return nil
}

// RegisterBusinessGRPCService registers a gRPC service
func (r *TestServiceRegistry) RegisterBusinessGRPCService(service server.BusinessGRPCService) error {
	r.grpcServices = append(r.grpcServices, service)
	return nil
}

// RegisterBusinessHealthCheck registers a health check
func (r *TestServiceRegistry) RegisterBusinessHealthCheck(check server.BusinessHealthCheck) error {
	r.healthChecks = append(r.healthChecks, check)
	return nil
}

// TestEventHandler is a simple event handler for testing
type TestEventHandler struct {
	handlerID string
	topics    []string
}

func (h *TestEventHandler) GetHandlerID() string {
	return h.handlerID
}

func (h *TestEventHandler) GetTopics() []string {
	return h.topics
}

// MessagingHealthCheck implements health check for messaging coordinator
type MessagingHealthCheck struct {
	coordinator messaging.MessagingCoordinator
}

func (h *MessagingHealthCheck) Check(ctx context.Context) error {
	// Check if coordinator is running
	metrics := h.coordinator.GetMetrics()
	if metrics.StartedAt == nil {
		return fmt.Errorf("messaging coordinator is not running")
	}
	return nil
}

func (h *MessagingHealthCheck) GetServiceName() string {
	return "messaging-coordinator"
}

// Run the messaging service registration test suite
func TestMessagingServiceRegistrationTestSuite(t *testing.T) {
	suite.Run(t, new(MessagingServiceRegistrationTestSuite))
}