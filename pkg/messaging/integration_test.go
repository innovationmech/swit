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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

// MessagingIntegrationTestSuite provides comprehensive integration tests
// for the messaging framework with external dependencies
type MessagingIntegrationTestSuite struct {
	suite.Suite
	coordinator    messaging.MessagingCoordinator
	testBroker     *MockMessageBroker
	testHandlers   []*MockEventHandler
	testSubscriber *MockEventSubscriber
}

// SetupSuite initializes the test suite
func (suite *MessagingIntegrationTestSuite) SetupSuite() {
	suite.coordinator = messaging.NewMessagingCoordinator()
	suite.testBroker = &MockMessageBroker{}
	suite.testHandlers = suite.createTestHandlers()
	suite.testSubscriber = &MockEventSubscriber{}
}

// TearDownSuite cleans up after the test suite
func (suite *MessagingIntegrationTestSuite) TearDownSuite() {
	if suite.coordinator != nil && suite.coordinator.IsStarted() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = suite.coordinator.Stop(ctx)
	}
}

// SetupTest sets up each test case
func (suite *MessagingIntegrationTestSuite) SetupTest() {
	suite.coordinator = messaging.NewMessagingCoordinator()
	suite.testBroker = &MockMessageBroker{}
	suite.testHandlers = suite.createTestHandlers()
	suite.testSubscriber = &MockEventSubscriber{}
	
	// Reset all mocks
	suite.testBroker.Mock = mock.Mock{}
	suite.testSubscriber.Mock = mock.Mock{}
	for _, handler := range suite.testHandlers {
		handler.Mock = mock.Mock{}
	}
}

// TearDownTest cleans up after each test case
func (suite *MessagingIntegrationTestSuite) TearDownTest() {
	if suite.coordinator != nil && suite.coordinator.IsStarted() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = suite.coordinator.Stop(ctx)
	}
}

// TestMessagingCoordinatorFullLifecycle tests complete lifecycle with real components
func (suite *MessagingIntegrationTestSuite) TestMessagingCoordinatorFullLifecycle() {
	// Test: Full lifecycle from registration through graceful shutdown

	// Setup broker expectations
	suite.testBroker.On("Connect", mock.Anything).Return(nil)
	suite.testBroker.On("CreateSubscriber", mock.MatchedBy(func(config messaging.SubscriberConfig) bool {
		return len(config.Topics) > 0 && config.ConsumerGroup != ""
	})).Return(suite.testSubscriber, nil)
	suite.testBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.testBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{
		MessagesPublished: 100,
		MessagesConsumed:  80,
	})
	suite.testBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  messaging.HealthStatusHealthy,
		Message: "Broker is healthy",
	}, nil)

	// Setup subscriber expectations
	suite.testSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	suite.testSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	suite.testSubscriber.On("Close").Return(nil)

	// Setup handler expectations
	for _, handler := range suite.testHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Register broker
	err := suite.coordinator.RegisterBroker("integration-broker", suite.testBroker)
	require.NoError(suite.T(), err)

	// Register handlers
	for _, handler := range suite.testHandlers {
		err := suite.coordinator.RegisterEventHandler(handler)
		require.NoError(suite.T(), err)
	}

	// Verify registration
	brokers := suite.coordinator.GetRegisteredBrokers()
	assert.Len(suite.T(), brokers, 1)
	assert.Contains(suite.T(), brokers, "integration-broker")

	handlers := suite.coordinator.GetRegisteredHandlers()
	assert.Len(suite.T(), handlers, len(suite.testHandlers))

	// Start coordinator
	ctx := context.Background()
	err = suite.coordinator.Start(ctx)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), suite.coordinator.IsStarted())

	// Verify metrics after start
	metrics := suite.coordinator.GetMetrics()
	assert.Equal(suite.T(), 1, metrics.BrokerCount)
	assert.Equal(suite.T(), len(suite.testHandlers), metrics.HandlerCount)
	assert.NotNil(suite.T(), metrics.StartedAt)
	assert.True(suite.T(), metrics.Uptime > 0)

	// Verify health check
	health, err := suite.coordinator.HealthCheck(ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", health.Overall)
	assert.Equal(suite.T(), "healthy", health.BrokerHealth["integration-broker"])

	// Verify handlers are active
	for _, handlerID := range handlers {
		assert.Equal(suite.T(), "active", health.HandlerHealth[handlerID])
	}

	// Test graceful shutdown
	err = suite.coordinator.Stop(ctx)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), suite.coordinator.IsStarted())

	// Verify all mocks were called as expected
	suite.testBroker.AssertExpectations(suite.T())
	suite.testSubscriber.AssertExpectations(suite.T())
	for _, handler := range suite.testHandlers {
		handler.AssertExpectations(suite.T())
	}
}

// TestBrokerIntegrationWithHandlers tests broker integration with multiple handlers
func (suite *MessagingIntegrationTestSuite) TestBrokerIntegrationWithHandlers() {
	// Test: Broker should properly integrate with multiple handlers on different topics

	// Setup broker with multiple topics support
	suite.testBroker.On("Connect", mock.Anything).Return(nil)
	suite.testBroker.On("CreateSubscriber", mock.MatchedBy(func(config messaging.SubscriberConfig) bool {
		return len(config.Topics) > 0 && config.ConsumerGroup != ""
	})).Return(suite.testSubscriber, nil).Times(len(suite.testHandlers)) // Expect one call per handler
	suite.testBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.testBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{
		MessagesPublished: 200,
		MessagesConsumed: 150,
	})
	suite.testBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  messaging.HealthStatusHealthy,
		Message: "Multi-topic broker is healthy",
	}, nil)

	// Setup subscriber for multiple topics
	suite.testSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Times(len(suite.testHandlers))
	suite.testSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	suite.testSubscriber.On("Close").Return(nil)

	// Setup handlers with different topics
	for _, handler := range suite.testHandlers {
		handler.On("Initialize", mock.Anything).Return(nil)
		handler.On("Shutdown", mock.Anything).Return(nil)
	}

	// Register broker
	err := suite.coordinator.RegisterBroker("multi-topic-broker", suite.testBroker)
	require.NoError(suite.T(), err)

	// Register handlers
	for _, handler := range suite.testHandlers {
		err := suite.coordinator.RegisterEventHandler(handler)
		require.NoError(suite.T(), err)
	}

	// Start coordinator
	ctx := context.Background()
	err = suite.coordinator.Start(ctx)
	require.NoError(suite.T(), err)

	// Verify all handlers are registered and topics are properly assigned
	metrics := suite.coordinator.GetMetrics()
	assert.Equal(suite.T(), len(suite.testHandlers), metrics.HandlerCount)

	// Test that each handler gets its own subscriber
	health, err := suite.coordinator.HealthCheck(ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", health.Overall)

	// Verify all handlers are active
	for _, handler := range suite.testHandlers {
		assert.Equal(suite.T(), "active", health.HandlerHealth[handler.GetHandlerID()])
	}

	// Stop coordinator
	err = suite.coordinator.Stop(ctx)
	require.NoError(suite.T(), err)

	// Verify cleanup
	suite.testBroker.AssertExpectations(suite.T())
	suite.testSubscriber.AssertExpectations(suite.T())
	for _, handler := range suite.testHandlers {
		handler.AssertExpectations(suite.T())
	}
}

// TestHealthCheckIntegration tests health check integration across components
func (suite *MessagingIntegrationTestSuite) TestHealthCheckIntegration() {
	// Test: Health checks should aggregate status across all components

	// Create multiple brokers with different health statuses
	healthyBroker := &MockMessageBroker{}
	healthySubscriber := &MockEventSubscriber{}
	healthyBroker.On("Connect", mock.Anything).Return(nil)
	healthyBroker.On("CreateSubscriber", mock.Anything).Return(healthySubscriber, nil).Maybe()
	healthyBroker.On("Disconnect", mock.Anything).Return(nil)
	healthyBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})
	healthyBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  messaging.HealthStatusHealthy,
		Message: "Healthy broker",
	}, nil)
	healthySubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	healthySubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	healthySubscriber.On("Close").Return(nil)

	degradedBroker := &MockMessageBroker{}
	degradedSubscriber := &MockEventSubscriber{}
	degradedBroker.On("Connect", mock.Anything).Return(nil)
	degradedBroker.On("CreateSubscriber", mock.Anything).Return(degradedSubscriber, nil).Maybe()
	degradedBroker.On("Disconnect", mock.Anything).Return(nil)
	degradedBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})
	degradedBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  messaging.HealthStatusDegraded,
		Message: "Performance issues",
	}, nil)
	degradedSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	degradedSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	degradedSubscriber.On("Close").Return(nil)

	// Register both brokers
	err := suite.coordinator.RegisterBroker("healthy-broker", healthyBroker)
	require.NoError(suite.T(), err)

	err = suite.coordinator.RegisterBroker("degraded-broker", degradedBroker)
	require.NoError(suite.T(), err)

	// Setup handler expectations
	handler := suite.testHandlers[0]
	handler.On("Initialize", mock.Anything).Return(nil)
	handler.On("Shutdown", mock.Anything).Return(nil)

	// Register handler
	err = suite.coordinator.RegisterEventHandler(handler)
	require.NoError(suite.T(), err)

	// Start coordinator
	ctx := context.Background()
	err = suite.coordinator.Start(ctx)
	require.NoError(suite.T(), err)

	// Test health check - should report degraded overall status due to one unhealthy broker
	health, err := suite.coordinator.HealthCheck(ctx)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "degraded", health.Overall)
	assert.Equal(suite.T(), "healthy", health.BrokerHealth["healthy-broker"])
	assert.Equal(suite.T(), "degraded", health.BrokerHealth["degraded-broker"])
	assert.Contains(suite.T(), health.Details, "Performance issues")

	// Stop coordinator
	err = suite.coordinator.Stop(ctx)
	require.NoError(suite.T(), err)

	// Verify all expectations were met
	healthyBroker.AssertExpectations(suite.T())
	degradedBroker.AssertExpectations(suite.T())
	handler.AssertExpectations(suite.T())
}

// TestMetricsIntegration tests metrics collection across all components
func (suite *MessagingIntegrationTestSuite) TestMetricsIntegration() {
	// Test: Metrics should be properly aggregated across all brokers and handlers

	// Setup broker with specific metrics
	suite.testBroker.On("Connect", mock.Anything).Return(nil)
	suite.testBroker.On("CreateSubscriber", mock.Anything).Return(suite.testSubscriber, nil)
	suite.testBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.testBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{
		MessagesPublished:  1000,
		MessagesConsumed:   800,
		ConnectionFailures: 2,
		PublishErrors:      5,
		ConsumeErrors:      3,
	})
	suite.testBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  messaging.HealthStatusHealthy,
		Message: "Metrics test broker",
	}, nil).Maybe()

	// Setup subscriber
	suite.testSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	suite.testSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	suite.testSubscriber.On("Close").Return(nil)

	// Setup handler
	handler := suite.testHandlers[0]
	handler.On("Initialize", mock.Anything).Return(nil)
	handler.On("Shutdown", mock.Anything).Return(nil)

	// Register broker and handler
	err := suite.coordinator.RegisterBroker("metrics-broker", suite.testBroker)
	require.NoError(suite.T(), err)

	err = suite.coordinator.RegisterEventHandler(handler)
	require.NoError(suite.T(), err)

	// Start coordinator
	ctx := context.Background()
	err = suite.coordinator.Start(ctx)
	require.NoError(suite.T(), err)

	// Test metrics aggregation
	metrics := suite.coordinator.GetMetrics()
	assert.Equal(suite.T(), 1, metrics.BrokerCount)
	assert.Equal(suite.T(), 1, metrics.HandlerCount)
	assert.Equal(suite.T(), int64(1800), metrics.TotalMessagesProcessed) // 1000 + 800
	assert.Equal(suite.T(), int64(10), metrics.TotalErrors)                // 2 + 5 + 3

	// Verify broker-specific metrics are included
	brokerMetrics, exists := metrics.BrokerMetrics["metrics-broker"]
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), int64(1000), brokerMetrics.MessagesPublished)
	assert.Equal(suite.T(), int64(800), brokerMetrics.MessagesConsumed)
	assert.Equal(suite.T(), int64(2), brokerMetrics.ConnectionFailures)

	// Stop coordinator
	err = suite.coordinator.Stop(ctx)
	require.NoError(suite.T(), err)

	// Verify expectations
	suite.testBroker.AssertExpectations(suite.T())
	suite.testSubscriber.AssertExpectations(suite.T())
	handler.AssertExpectations(suite.T())
}

// TestConcurrentOperations tests concurrent access to coordinator
func (suite *MessagingIntegrationTestSuite) TestConcurrentOperations() {
	// Test: Coordinator should handle concurrent operations safely

	// Setup minimal broker and handler for concurrency testing
	suite.testBroker.On("Connect", mock.Anything).Return(nil)
	suite.testBroker.On("CreateSubscriber", mock.Anything).Return(suite.testSubscriber, nil)
	suite.testBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.testBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})
	suite.testBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  messaging.HealthStatusHealthy,
		Message: "Concurrent test broker",
	}, nil)

	suite.testSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	suite.testSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	suite.testSubscriber.On("Close").Return(nil)

	handler := suite.testHandlers[0]
	handler.On("Initialize", mock.Anything).Return(nil)
	handler.On("Shutdown", mock.Anything).Return(nil)

	// Register and start coordinator
	err := suite.coordinator.RegisterBroker("concurrent-broker", suite.testBroker)
	require.NoError(suite.T(), err)

	err = suite.coordinator.RegisterEventHandler(handler)
	require.NoError(suite.T(), err)

	ctx := context.Background()
	err = suite.coordinator.Start(ctx)
	require.NoError(suite.T(), err)

	// Test concurrent metrics access
	const numGoroutines = 50
	metricsChan := make(chan *messaging.MessagingCoordinatorMetrics, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			metrics := suite.coordinator.GetMetrics()
			metricsChan <- metrics
		}()
	}

	// Collect all metrics
	for i := 0; i < numGoroutines; i++ {
		metrics := <-metricsChan
		assert.NotNil(suite.T(), metrics)
		assert.Equal(suite.T(), 1, metrics.BrokerCount)
		assert.Equal(suite.T(), 1, metrics.HandlerCount)
	}

	// Test concurrent health checks
	healthChan := make(chan *messaging.MessagingHealthStatus, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			health, err := suite.coordinator.HealthCheck(ctx)
			assert.NoError(suite.T(), err)
			healthChan <- health
		}()
	}

	// Collect all health statuses
	for i := 0; i < numGoroutines; i++ {
		health := <-healthChan
		assert.NotNil(suite.T(), health)
		assert.Equal(suite.T(), "healthy", health.Overall)
	}

	// Stop coordinator
	err = suite.coordinator.Stop(ctx)
	require.NoError(suite.T(), err)

	// Verify expectations
	suite.testBroker.AssertExpectations(suite.T())
	suite.testSubscriber.AssertExpectations(suite.T())
	handler.AssertExpectations(suite.T())
}

// TestErrorHandlingIntegration tests error handling across components
func (suite *MessagingIntegrationTestSuite) TestErrorHandlingIntegration() {
	// Test: Error handling should be properly integrated across all components

	// Test broker connection failure
	suite.testBroker.On("Connect", mock.Anything).Return(assert.AnError)
	suite.testBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	err := suite.coordinator.RegisterBroker("failing-broker", suite.testBroker)
	require.NoError(suite.T(), err)

	// Start should fail due to broker connection failure
	ctx := context.Background()
	err = suite.coordinator.Start(ctx)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to start broker")
	assert.False(suite.T(), suite.coordinator.IsStarted())

	// Reset coordinator and test handler initialization failure
	suite.coordinator = messaging.NewMessagingCoordinator()
	suite.testBroker = &MockMessageBroker{}

	// Setup broker to succeed but handler to fail
	suite.testBroker.On("Connect", mock.Anything).Return(nil)
	suite.testBroker.On("CreateSubscriber", mock.Anything).Return(suite.testSubscriber, nil)
	suite.testBroker.On("Disconnect", mock.Anything).Return(nil)
	suite.testBroker.On("GetMetrics").Return(&messaging.BrokerMetrics{})

	suite.testSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	suite.testSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	suite.testSubscriber.On("Close").Return(nil)

	handler := suite.testHandlers[0]
	handler.On("Initialize", mock.Anything).Return(assert.AnError)

	// Register components
	err = suite.coordinator.RegisterBroker("partial-broker", suite.testBroker)
	require.NoError(suite.T(), err)

	err = suite.coordinator.RegisterEventHandler(handler)
	require.NoError(suite.T(), err)

	// Start should fail due to handler initialization failure
	err = suite.coordinator.Start(ctx)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to initialize handler")
	assert.False(suite.T(), suite.coordinator.IsStarted())
}

// createTestHandlers creates test handlers for the suite
func (suite *MessagingIntegrationTestSuite) createTestHandlers() []*MockEventHandler {
	return []*MockEventHandler{
		NewMockEventHandler("user-events", []string{"user.created", "user.updated"}, ""),
		NewMockEventHandler("order-events", []string{"order.created", "order.completed"}, ""),
		NewMockEventHandler("notification-events", []string{"notification.sent"}, ""),
	}
}

// Mock implementations - these are duplicates of the ones in coordinator_test.go
// but we need them here since they're not exported

// MockEventHandler is a mock implementation of EventHandler for testing
type MockEventHandler struct {
	mock.Mock
	id     string
	topics []string
	broker string
}

func (m *MockEventHandler) Handle(ctx context.Context, message *messaging.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockEventHandler) OnError(ctx context.Context, message *messaging.Message, err error) messaging.ErrorAction {
	args := m.Called(ctx, message, err)
	return args.Get(0).(messaging.ErrorAction)
}

func (m *MockEventHandler) GetHandlerID() string {
	return m.id
}

func (m *MockEventHandler) GetTopics() []string {
	return m.topics
}

func (m *MockEventHandler) GetBrokerRequirement() string {
	return m.broker
}

func (m *MockEventHandler) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventHandler) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockMessageBroker is a mock implementation of MessageBroker for testing
type MockMessageBroker struct {
	mock.Mock
}

func (m *MockMessageBroker) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMessageBroker) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMessageBroker) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMessageBroker) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMessageBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	args := m.Called(config)
	return args.Get(0).(messaging.EventPublisher), args.Error(1)
}

func (m *MockMessageBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	args := m.Called(config)
	return args.Get(0).(messaging.EventSubscriber), args.Error(1)
}

func (m *MockMessageBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*messaging.HealthStatus), args.Error(1)
}

func (m *MockMessageBroker) GetMetrics() *messaging.BrokerMetrics {
	args := m.Called()
	return args.Get(0).(*messaging.BrokerMetrics)
}

func (m *MockMessageBroker) GetCapabilities() *messaging.BrokerCapabilities {
	args := m.Called()
	return args.Get(0).(*messaging.BrokerCapabilities)
}

// MockEventSubscriber is a mock implementation of EventSubscriber for testing
type MockEventSubscriber struct {
	mock.Mock
}

func (m *MockEventSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	args := m.Called(ctx, handler)
	return args.Error(0)
}

func (m *MockEventSubscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	args := m.Called(ctx, handler, middleware)
	return args.Error(0)
}

func (m *MockEventSubscriber) Unsubscribe(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventSubscriber) Pause(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventSubscriber) Resume(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventSubscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	args := m.Called(ctx, position)
	return args.Error(0)
}

func (m *MockEventSubscriber) GetLag(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEventSubscriber) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEventSubscriber) GetMetrics() *messaging.SubscriberMetrics {
	args := m.Called()
	return args.Get(0).(*messaging.SubscriberMetrics)
}

// NewMockEventHandler creates a new MockEventHandler with the specified parameters
func NewMockEventHandler(id string, topics []string, broker string) *MockEventHandler {
	return &MockEventHandler{
		id:     id,
		topics: topics,
		broker: broker,
	}
}

// Run the messaging integration test suite
func TestMessagingIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MessagingIntegrationTestSuite))
}