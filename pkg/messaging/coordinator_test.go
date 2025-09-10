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

package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

// TestMessagingCoordinator tests the MessagingCoordinator interface and implementation.
func TestMessagingCoordinator(t *testing.T) {
	t.Run("NewMessagingCoordinator", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()

		assert.NotNil(t, coordinator)
		assert.False(t, coordinator.IsStarted())
		assert.Empty(t, coordinator.GetRegisteredBrokers())
		assert.Empty(t, coordinator.GetRegisteredHandlers())

		metrics := coordinator.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, 0, metrics.BrokerCount)
		assert.Equal(t, 0, metrics.HandlerCount)
		assert.Nil(t, metrics.StartedAt)
	})

	t.Run("RegisterBroker", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}

		// Setup mock for metrics (called during registration)
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})

		// Test successful registration
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		assert.NoError(t, err)

		// Verify broker is registered
		brokers := coordinator.GetRegisteredBrokers()
		assert.Contains(t, brokers, "test-broker")

		// Test retrieving broker
		broker, err := coordinator.GetBroker("test-broker")
		assert.NoError(t, err)
		assert.Equal(t, mockBroker, broker)

		// Test duplicate registration
		err = coordinator.RegisterBroker("test-broker", mockBroker)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("RegisterBroker_InvalidNames", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}

		// Setup mock for metrics (in case it's called)
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{}).Maybe()

		// Test empty name
		err := coordinator.RegisterBroker("", mockBroker)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")

		// Test nil broker
		err = coordinator.RegisterBroker("test", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")

		// Test invalid characters
		err = coordinator.RegisterBroker("test@broker", mockBroker)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid broker name")

		// Test name too long
		longName := strings.Repeat("a", 60)
		err = coordinator.RegisterBroker(longName, mockBroker)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "1-50 characters")
	})

	t.Run("GetBroker_NotFound", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()

		broker, err := coordinator.GetBroker("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, broker)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("RegisterEventHandler", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}
		mockHandler := &MockEventHandler{
			id:     "test-handler",
			topics: []string{"test-topic"},
			broker: "",
		}

		// Setup mock for metrics
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})

		// Register broker first
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)

		// Test successful registration
		err = coordinator.RegisterEventHandler(mockHandler)
		assert.NoError(t, err)

		// Verify handler is registered
		handlers := coordinator.GetRegisteredHandlers()
		assert.Contains(t, handlers, "test-handler")

		// Test duplicate registration
		err = coordinator.RegisterEventHandler(mockHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("RegisterEventHandler_InvalidHandler", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()

		// Test nil handler
		err := coordinator.RegisterEventHandler(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")

		// Test handler with empty ID
		mockHandler := &MockEventHandler{id: ""}
		err = coordinator.RegisterEventHandler(mockHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")

		// Test handler with no topics
		mockHandler = &MockEventHandler{
			id:     "test-handler",
			topics: []string{},
			broker: "",
		}
		err = coordinator.RegisterEventHandler(mockHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one topic")

		// Test handler requiring nonexistent broker
		mockHandler = &MockEventHandler{
			id:     "test-handler",
			topics: []string{"test-topic"},
			broker: "nonexistent-broker",
		}
		err = coordinator.RegisterEventHandler(mockHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("UnregisterEventHandler", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}
		mockHandler := &MockEventHandler{
			id:     "test-handler",
			topics: []string{"test-topic"},
			broker: "",
		}

		// Setup mock for metrics
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})

		// Register broker and handler
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)
		err = coordinator.RegisterEventHandler(mockHandler)
		require.NoError(t, err)

		// Test successful unregistration
		err = coordinator.UnregisterEventHandler("test-handler")
		assert.NoError(t, err)

		// Verify handler is unregistered
		handlers := coordinator.GetRegisteredHandlers()
		assert.NotContains(t, handlers, "test-handler")

		// Test unregistering nonexistent handler
		err = coordinator.UnregisterEventHandler("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("StartStop", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}
		mockSubscriber := &MockEventSubscriber{}
		mockHandler := &MockEventHandler{
			id:     "test-handler",
			topics: []string{"test-topic"},
			broker: "",
		}

		// Setup mocks
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})
		mockBroker.On("Connect", mock.Anything).Return(nil)
		mockBroker.On("CreateSubscriber", mock.Anything).Return(mockSubscriber, nil)
		mockBroker.On("Disconnect", mock.Anything).Return(nil)
		mockSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
		mockSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
		mockSubscriber.On("Close").Return(nil)
		mockHandler.On("Initialize", mock.Anything).Return(nil)
		mockHandler.On("Shutdown", mock.Anything).Return(nil)

		// Register broker and handler
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)
		err = coordinator.RegisterEventHandler(mockHandler)
		require.NoError(t, err)

		// Test start
		ctx := context.Background()
		err = coordinator.Start(ctx)
		assert.NoError(t, err)
		assert.True(t, coordinator.IsStarted())

		// Verify metrics after start
		metrics := coordinator.GetMetrics()
		assert.NotNil(t, metrics.StartedAt)
		assert.True(t, metrics.Uptime > 0)

		// Test double start
		err = coordinator.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		// Test stop
		err = coordinator.Stop(ctx)
		assert.NoError(t, err)
		assert.False(t, coordinator.IsStarted())

		// Test double stop (should be no-op)
		err = coordinator.Stop(ctx)
		assert.NoError(t, err)

		// Verify mock calls
		mockBroker.AssertExpectations(t)
		mockSubscriber.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
	})

	t.Run("Start_BrokerConnectionFailure", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}

		// Setup mocks
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})
		mockBroker.On("Connect", mock.Anything).Return(errors.New("connection failed"))

		// Register broker
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)

		// Test start failure
		ctx := context.Background()
		err = coordinator.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start broker")
		assert.False(t, coordinator.IsStarted())
	})

	t.Run("Start_HandlerInitializationFailure", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}
		mockHandler := &MockEventHandler{
			id:     "test-handler",
			topics: []string{"test-topic"},
			broker: "",
		}

		// Setup mocks
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})
		mockBroker.On("Connect", mock.Anything).Return(nil)
		mockBroker.On("Disconnect", mock.Anything).Return(nil)
		mockHandler.On("Initialize", mock.Anything).Return(errors.New("initialization failed"))

		// Register broker and handler
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)
		err = coordinator.RegisterEventHandler(mockHandler)
		require.NoError(t, err)

		// Test start failure
		ctx := context.Background()
		err = coordinator.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize handler")
		assert.False(t, coordinator.IsStarted())
	})

	t.Run("HealthCheck", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}

		// Setup mock for metrics
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})

		// Register broker
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)

		// Setup health check mock for the stopped coordinator test
		mockBroker.On("HealthCheck", mock.Anything).Return(&HealthStatus{Status: HealthStatusHealthy}, nil)

		// Test health check on stopped coordinator
		ctx := context.Background()
		health, err := coordinator.HealthCheck(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "stopped", health.Overall)
		assert.Contains(t, health.Details, "not started")

		// Setup mock for healthy broker
		healthyStatus := &HealthStatus{
			Status:  HealthStatusHealthy,
			Message: "All systems operational",
		}
		mockBroker.On("HealthCheck", mock.Anything).Return(healthyStatus, nil)
		mockBroker.On("Connect", mock.Anything).Return(nil)

		// Start coordinator
		err = coordinator.Start(ctx)
		require.NoError(t, err)

		// Test health check on running coordinator
		health, err = coordinator.HealthCheck(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health.Overall)
		assert.Equal(t, "healthy", health.BrokerHealth["test-broker"])
		assert.Contains(t, health.Details, "are healthy")
	})

	t.Run("HealthCheck_UnhealthyBroker", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}

		// Setup mock for metrics
		mockBroker.On("GetMetrics").Return(&BrokerMetrics{})

		// Register broker
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)

		// Setup mock for unhealthy broker
		unhealthyStatus := &HealthStatus{
			Status:  HealthStatusDegraded,
			Message: "Performance issues detected",
		}
		mockBroker.On("HealthCheck", mock.Anything).Return(unhealthyStatus, nil)
		mockBroker.On("Connect", mock.Anything).Return(nil)

		// Start coordinator
		ctx := context.Background()
		err = coordinator.Start(ctx)
		require.NoError(t, err)

		// Test health check
		health, err := coordinator.HealthCheck(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "degraded", health.Overall)
		assert.Equal(t, "degraded", health.BrokerHealth["test-broker"])
		assert.Contains(t, health.Details, "Performance issues detected")
	})

	t.Run("GetMetrics", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()
		mockBroker := &MockMessageBroker{}

		// Setup broker metrics
		brokerMetrics := &BrokerMetrics{
			MessagesPublished:  100,
			MessagesConsumed:   80,
			ConnectionFailures: 2,
			PublishErrors:      1,
			ConsumeErrors:      3,
		}
		mockBroker.On("GetMetrics").Return(brokerMetrics)

		// Register broker
		err := coordinator.RegisterBroker("test-broker", mockBroker)
		require.NoError(t, err)

		// Get metrics
		metrics := coordinator.GetMetrics()
		assert.Equal(t, 1, metrics.BrokerCount)
		assert.Equal(t, 0, metrics.HandlerCount)
		assert.Equal(t, int64(180), metrics.TotalMessagesProcessed) // 100 + 80
		assert.Equal(t, int64(6), metrics.TotalErrors)              // 2 + 1 + 3
		assert.Contains(t, metrics.BrokerMetrics, "test-broker")
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()

		// Test concurrent broker registrations
		brokerCount := 10
		done := make(chan bool, brokerCount)

		for i := 0; i < brokerCount; i++ {
			go func(index int) {
				mockBroker := &MockMessageBroker{}
				mockBroker.On("GetMetrics").Return(&BrokerMetrics{})
				err := coordinator.RegisterBroker(fmt.Sprintf("broker-%d", index), mockBroker)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all registrations
		for i := 0; i < brokerCount; i++ {
			<-done
		}

		// Verify all brokers registered
		brokers := coordinator.GetRegisteredBrokers()
		assert.Equal(t, brokerCount, len(brokers))

		// Test concurrent metrics access
		metricsCount := 50
		metricsDone := make(chan bool, metricsCount)

		for i := 0; i < metricsCount; i++ {
			go func() {
				metrics := coordinator.GetMetrics()
				assert.Equal(t, brokerCount, metrics.BrokerCount)
				metricsDone <- true
			}()
		}

		// Wait for all metrics calls
		for i := 0; i < metricsCount; i++ {
			<-metricsDone
		}
	})
}

// TestMessagingCoordinatorIntegration tests integration scenarios
func TestMessagingCoordinatorIntegration(t *testing.T) {
	t.Run("FullLifecycleWithMultipleBrokersAndHandlers", func(t *testing.T) {
		coordinator := NewMessagingCoordinator()

		// Create multiple brokers
		kafkaBroker := &MockMessageBroker{}
		natsBroker := &MockMessageBroker{}

		// Create subscribers
		kafkaSubscriber := &MockEventSubscriber{}
		natsSubscriber := &MockEventSubscriber{}

		// Create handlers
		userHandler := &MockEventHandler{
			id:     "user-handler",
			topics: []string{"user-events"},
			broker: "kafka",
		}
		orderHandler := &MockEventHandler{
			id:     "order-handler",
			topics: []string{"order-events"},
			broker: "nats",
		}
		notificationHandler := &MockEventHandler{
			id:     "notification-handler",
			topics: []string{"notification-events"},
			broker: "", // Can use any broker
		}

		// Setup mocks
		kafkaBroker.On("Connect", mock.Anything).Return(nil)
		kafkaBroker.On("CreateSubscriber", mock.MatchedBy(func(config SubscriberConfig) bool {
			return len(config.Topics) > 0 && config.ConsumerGroup != ""
		})).Return(kafkaSubscriber, nil)
		kafkaBroker.On("Disconnect", mock.Anything).Return(nil)
		kafkaBroker.On("GetMetrics").Return(&BrokerMetrics{MessagesPublished: 50})
		kafkaBroker.On("HealthCheck", mock.Anything).Return(&HealthStatus{Status: HealthStatusHealthy}, nil)

		natsBroker.On("Connect", mock.Anything).Return(nil)
		natsBroker.On("CreateSubscriber", mock.MatchedBy(func(config SubscriberConfig) bool {
			return len(config.Topics) > 0 && config.ConsumerGroup != ""
		})).Return(natsSubscriber, nil)
		natsBroker.On("Disconnect", mock.Anything).Return(nil)
		natsBroker.On("GetMetrics").Return(&BrokerMetrics{MessagesConsumed: 30})
		natsBroker.On("HealthCheck", mock.Anything).Return(&HealthStatus{Status: HealthStatusHealthy}, nil)

		kafkaSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
		kafkaSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
		kafkaSubscriber.On("Close").Return(nil)

		natsSubscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
		natsSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
		natsSubscriber.On("Close").Return(nil)

		userHandler.On("Initialize", mock.Anything).Return(nil)
		userHandler.On("Shutdown", mock.Anything).Return(nil)
		orderHandler.On("Initialize", mock.Anything).Return(nil)
		orderHandler.On("Shutdown", mock.Anything).Return(nil)
		notificationHandler.On("Initialize", mock.Anything).Return(nil)
		notificationHandler.On("Shutdown", mock.Anything).Return(nil)

		// Register brokers
		err := coordinator.RegisterBroker("kafka", kafkaBroker)
		require.NoError(t, err)
		err = coordinator.RegisterBroker("nats", natsBroker)
		require.NoError(t, err)

		// Register handlers
		err = coordinator.RegisterEventHandler(userHandler)
		require.NoError(t, err)
		err = coordinator.RegisterEventHandler(orderHandler)
		require.NoError(t, err)
		err = coordinator.RegisterEventHandler(notificationHandler)
		require.NoError(t, err)

		// Verify registration
		brokers := coordinator.GetRegisteredBrokers()
		assert.Len(t, brokers, 2)
		assert.Contains(t, brokers, "kafka")
		assert.Contains(t, brokers, "nats")

		handlers := coordinator.GetRegisteredHandlers()
		assert.Len(t, handlers, 3)
		assert.Contains(t, handlers, "user-handler")
		assert.Contains(t, handlers, "order-handler")
		assert.Contains(t, handlers, "notification-handler")

		// Start coordinator
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = coordinator.Start(ctx)
		require.NoError(t, err)
		assert.True(t, coordinator.IsStarted())

		// Check metrics
		metrics := coordinator.GetMetrics()
		assert.Equal(t, 2, metrics.BrokerCount)
		assert.Equal(t, 3, metrics.HandlerCount)
		assert.NotNil(t, metrics.StartedAt)
		assert.True(t, metrics.Uptime > 0)

		// Check health
		health, err := coordinator.HealthCheck(ctx)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health.Overall)
		assert.Equal(t, "healthy", health.BrokerHealth["kafka"])
		assert.Equal(t, "healthy", health.BrokerHealth["nats"])
		assert.Equal(t, "active", health.HandlerHealth["user-handler"])
		assert.Equal(t, "active", health.HandlerHealth["order-handler"])
		assert.Equal(t, "active", health.HandlerHealth["notification-handler"])

		// Stop coordinator
		err = coordinator.Stop(ctx)
		require.NoError(t, err)
		assert.False(t, coordinator.IsStarted())

		// Verify all mocks were called as expected
		kafkaBroker.AssertExpectations(t)
		natsBroker.AssertExpectations(t)
		kafkaSubscriber.AssertExpectations(t)
		natsSubscriber.AssertExpectations(t)
		userHandler.AssertExpectations(t)
		orderHandler.AssertExpectations(t)
		notificationHandler.AssertExpectations(t)
	})
}

// MockEventHandler is a mock implementation of EventHandler for testing
type MockEventHandler struct {
	mock.Mock
	id     string
	topics []string
	broker string
}

func (m *MockEventHandler) Handle(ctx context.Context, message *Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockEventHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	args := m.Called(ctx, message, err)
	return args.Get(0).(ErrorAction)
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

func (m *MockMessageBroker) CreatePublisher(config PublisherConfig) (EventPublisher, error) {
	args := m.Called(config)
	return args.Get(0).(EventPublisher), args.Error(1)
}

func (m *MockMessageBroker) CreateSubscriber(config SubscriberConfig) (EventSubscriber, error) {
	args := m.Called(config)
	return args.Get(0).(EventSubscriber), args.Error(1)
}

func (m *MockMessageBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*HealthStatus), args.Error(1)
}

func (m *MockMessageBroker) GetMetrics() *BrokerMetrics {
	args := m.Called()
	return args.Get(0).(*BrokerMetrics)
}

func (m *MockMessageBroker) GetCapabilities() *BrokerCapabilities {
	args := m.Called()
	return args.Get(0).(*BrokerCapabilities)
}

// MockEventSubscriber is a mock implementation of EventSubscriber for testing
type MockEventSubscriber struct {
	mock.Mock
}

func (m *MockEventSubscriber) Subscribe(ctx context.Context, handler MessageHandler) error {
	args := m.Called(ctx, handler)
	return args.Error(0)
}

func (m *MockEventSubscriber) SubscribeWithMiddleware(ctx context.Context, handler MessageHandler, middleware ...Middleware) error {
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

func (m *MockEventSubscriber) Seek(ctx context.Context, position SeekPosition) error {
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

func (m *MockEventSubscriber) GetMetrics() *SubscriberMetrics {
	args := m.Called()
	return args.Get(0).(*SubscriberMetrics)
}
