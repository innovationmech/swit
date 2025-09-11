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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// healthTestMessagingCoordinator implements MessagingCoordinator for testing
type healthTestMessagingCoordinator struct {
	started          bool
	brokers          map[string]*healthTestMessageBroker
	handlers         map[string]*healthTestEventHandler
	healthStatus     *MessagingHealthStatus
	healthError      error
	shouldFailHealth bool
}

func newHealthTestMessagingCoordinator() *healthTestMessagingCoordinator {
	return &healthTestMessagingCoordinator{
		brokers:  make(map[string]*healthTestMessageBroker),
		handlers: make(map[string]*healthTestEventHandler),
		healthStatus: &MessagingHealthStatus{
			Overall:       "healthy",
			Details:       "All components are healthy",
			BrokerHealth:  make(map[string]string),
			HandlerHealth: make(map[string]string),
			Timestamp:     time.Now(),
		},
	}
}

func (m *healthTestMessagingCoordinator) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *healthTestMessagingCoordinator) Stop(ctx context.Context) error {
	m.started = false
	return nil
}

func (m *healthTestMessagingCoordinator) RegisterBroker(name string, broker MessageBroker) error {
	mockBroker, ok := broker.(*healthTestMessageBroker)
	if !ok {
		return errors.New("invalid broker type")
	}
	m.brokers[name] = mockBroker
	m.healthStatus.BrokerHealth[name] = "healthy"
	return nil
}

func (m *healthTestMessagingCoordinator) GetBroker(name string) (MessageBroker, error) {
	broker, exists := m.brokers[name]
	if !exists {
		return nil, errors.New("broker not found")
	}
	if broker == nil {
		return nil, errors.New("broker is nil")
	}
	return broker, nil
}

func (m *healthTestMessagingCoordinator) GetRegisteredBrokers() []string {
	names := make([]string, 0, len(m.brokers))
	for name := range m.brokers {
		names = append(names, name)
	}
	return names
}

func (m *healthTestMessagingCoordinator) RegisterEventHandler(handler EventHandler) error {
	mockHandler, ok := handler.(*healthTestEventHandler)
	if !ok {
		return errors.New("invalid handler type")
	}
	m.handlers[handler.GetHandlerID()] = mockHandler
	m.healthStatus.HandlerHealth[handler.GetHandlerID()] = "active"
	return nil
}

func (m *healthTestMessagingCoordinator) UnregisterEventHandler(handlerID string) error {
	delete(m.handlers, handlerID)
	delete(m.healthStatus.HandlerHealth, handlerID)
	return nil
}

func (m *healthTestMessagingCoordinator) GetRegisteredHandlers() []string {
	handlers := make([]string, 0, len(m.handlers))
	for id := range m.handlers {
		handlers = append(handlers, id)
	}
	return handlers
}

func (m *healthTestMessagingCoordinator) IsStarted() bool {
	return m.started
}

func (m *healthTestMessagingCoordinator) GetMetrics() *MessagingCoordinatorMetrics {
	metrics := &MessagingCoordinatorMetrics{
		BrokerCount:   len(m.brokers),
		HandlerCount:  len(m.handlers),
		BrokerMetrics: make(map[string]*BrokerMetrics),
	}
	if m.started {
		now := time.Now()
		metrics.StartedAt = &now
		metrics.Uptime = time.Since(now)
	}
	return metrics
}

func (m *healthTestMessagingCoordinator) HealthCheck(ctx context.Context) (*MessagingHealthStatus, error) {
	if m.shouldFailHealth {
		return nil, m.healthError
	}
	if m.healthError != nil {
		return m.healthStatus, m.healthError
	}
	return m.healthStatus, nil
}

func (m *healthTestMessagingCoordinator) GracefulShutdown(ctx context.Context, config *ShutdownConfig) (*GracefulShutdownManager, error) {
	manager := NewGracefulShutdownManager(m, config)
	return manager, nil
}

// healthTestMessageBroker implements MessageBroker for testing
type healthTestMessageBroker struct {
	connected    bool
	healthStatus *HealthStatus
	healthError  error
	metrics      *BrokerMetrics
}

func newHealthTestMessageBroker() *healthTestMessageBroker {
	return &healthTestMessageBroker{
		connected: true,
		healthStatus: &HealthStatus{
			Status:       HealthStatusHealthy,
			Message:      "Broker is healthy",
			LastChecked:  time.Now(),
			ResponseTime: 5 * time.Millisecond,
		},
		metrics: &BrokerMetrics{
			ConnectionStatus:   "connected",
			MessagesPublished:  100,
			MessagesConsumed:   95,
			ConnectionFailures: 0,
			PublishErrors:      0,
			ConsumeErrors:      0,
		},
	}
}

func (m *healthTestMessageBroker) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *healthTestMessageBroker) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *healthTestMessageBroker) Close() error {
	m.connected = false
	return nil
}

func (m *healthTestMessageBroker) IsConnected() bool {
	return m.connected
}

func (m *healthTestMessageBroker) CreatePublisher(config PublisherConfig) (EventPublisher, error) {
	return nil, errors.New("not implemented")
}

func (m *healthTestMessageBroker) CreateSubscriber(config SubscriberConfig) (EventSubscriber, error) {
	return nil, errors.New("not implemented")
}

func (m *healthTestMessageBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.healthError != nil {
		return nil, m.healthError
	}
	return m.healthStatus, nil
}

func (m *healthTestMessageBroker) GetMetrics() *BrokerMetrics {
	return m.metrics
}

func (m *healthTestMessageBroker) GetCapabilities() *BrokerCapabilities {
	return &BrokerCapabilities{
		SupportsTransactions:    false,
		SupportsOrdering:        true,
		SupportsPartitioning:    true,
		SupportsDeadLetter:      true,
		SupportsDelayedDelivery: false,
		SupportsPriority:        false,
		SupportsStreaming:       true,
		SupportsSeek:            true,
		SupportsConsumerGroups:  true,
		MaxMessageSize:          1024 * 1024, // 1MB
		MaxBatchSize:            100,
		MaxTopicNameLength:      255,
	}
}

// healthTestEventHandler implements EventHandler for testing
type healthTestEventHandler struct {
	id                string
	topics            []string
	brokerRequirement string
	initialized       bool
}

func newHealthTestEventHandler(id string, topics []string) *healthTestEventHandler {
	return &healthTestEventHandler{
		id:     id,
		topics: topics,
	}
}

func (m *healthTestEventHandler) GetHandlerID() string {
	return m.id
}

func (m *healthTestEventHandler) GetTopics() []string {
	return m.topics
}

func (m *healthTestEventHandler) GetBrokerRequirement() string {
	return m.brokerRequirement
}

func (m *healthTestEventHandler) Initialize(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *healthTestEventHandler) Shutdown(ctx context.Context) error {
	m.initialized = false
	return nil
}

func (m *healthTestEventHandler) Handle(ctx context.Context, message *Message) error {
	return nil
}

func (m *healthTestEventHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	return ErrorActionRetry
}

func TestHealthCheckConfig(t *testing.T) {
	t.Run("DefaultHealthCheckConfig", func(t *testing.T) {
		config := DefaultHealthCheckConfig()

		assert.True(t, config.Enabled)
		assert.Equal(t, 30*time.Second, config.Interval)
		assert.Equal(t, 10*time.Second, config.Timeout)
		assert.True(t, config.BrokerChecksEnabled)
		assert.True(t, config.HandlerChecksEnabled)
		assert.Equal(t, 3, config.FailureThreshold)
	})
}

func TestMessagingCoordinatorHealthChecker(t *testing.T) {
	coordinator := newHealthTestMessagingCoordinator()
	config := DefaultHealthCheckConfig()
	checker := NewMessagingCoordinatorHealthChecker(coordinator, "test-coordinator", config)

	t.Run("SuccessfulHealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := checker.Check(ctx)

		assert.NoError(t, err)
		assert.Equal(t, "test-coordinator", checker.GetServiceName())
	})

	t.Run("HealthCheckDisabled", func(t *testing.T) {
		disabledConfig := DefaultHealthCheckConfig()
		disabledConfig.Enabled = false
		disabledChecker := NewMessagingCoordinatorHealthChecker(coordinator, "test-coordinator", disabledConfig)

		ctx := context.Background()
		err := disabledChecker.Check(ctx)

		assert.NoError(t, err) // Should pass when disabled
	})

	t.Run("UnhealthyCoordinator", func(t *testing.T) {
		coordinator.healthStatus.Overall = "degraded"
		coordinator.healthStatus.Details = "Some brokers are unhealthy"

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "degraded")
	})

	t.Run("HealthCheckError", func(t *testing.T) {
		coordinator.healthError = errors.New("health check failed")

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "messaging coordinator health check failed")

		// Reset for other tests
		coordinator.healthError = nil
		coordinator.healthStatus.Overall = "healthy"
	})

	t.Run("HealthCheckTimeout", func(t *testing.T) {
		quickConfig := DefaultHealthCheckConfig()
		quickConfig.Timeout = 1 * time.Millisecond
		quickChecker := NewMessagingCoordinatorHealthChecker(coordinator, "test-coordinator", quickConfig)

		// This should still pass since our mock is fast
		ctx := context.Background()
		err := quickChecker.Check(ctx)
		assert.NoError(t, err)
	})

	t.Run("GetHealthStatus", func(t *testing.T) {
		ctx := context.Background()
		status, err := checker.GetHealthStatus(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, "healthy", status.Overall)
	})

	t.Run("GetHealthCheckConfig", func(t *testing.T) {
		returnedConfig := checker.GetHealthCheckConfig()
		assert.Equal(t, config, returnedConfig)
	})
}

func TestBrokerHealthChecker(t *testing.T) {
	broker := newHealthTestMessageBroker()
	config := DefaultHealthCheckConfig()
	checker := NewBrokerHealthChecker(broker, "test-broker", config)

	t.Run("SuccessfulHealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := checker.Check(ctx)

		assert.NoError(t, err)
		assert.Equal(t, "messaging-broker-test-broker", checker.GetServiceName())
	})

	t.Run("BrokerHealthChecksDisabled", func(t *testing.T) {
		disabledConfig := DefaultHealthCheckConfig()
		disabledConfig.BrokerChecksEnabled = false
		disabledChecker := NewBrokerHealthChecker(broker, "test-broker", disabledConfig)

		ctx := context.Background()
		err := disabledChecker.Check(ctx)

		assert.NoError(t, err) // Should pass when disabled
	})

	t.Run("BrokerDisconnected", func(t *testing.T) {
		broker.connected = false

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")

		// Reset for other tests
		broker.connected = true
	})

	t.Run("UnhealthyBroker", func(t *testing.T) {
		broker.healthStatus.Status = HealthStatusDegraded
		broker.healthStatus.Message = "Broker performance degraded"

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "degraded")

		// Reset for other tests
		broker.healthStatus.Status = HealthStatusHealthy
	})

	t.Run("BrokerHealthCheckError", func(t *testing.T) {
		broker.healthError = errors.New("broker health check failed")

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "health check failed")

		// Reset for other tests
		broker.healthError = nil
	})

	t.Run("GetHealthStatus", func(t *testing.T) {
		ctx := context.Background()
		status, err := checker.GetHealthStatus(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, HealthStatusHealthy, status.Status)
	})
}

func TestMessagingSubsystemHealthChecker(t *testing.T) {
	coordinator := newHealthTestMessagingCoordinator()
	config := DefaultHealthCheckConfig()
	checker := NewMessagingSubsystemHealthChecker(coordinator, config)

	// Add some test brokers
	broker1 := newHealthTestMessageBroker()
	broker2 := newHealthTestMessageBroker()
	coordinator.RegisterBroker("broker1", broker1)
	coordinator.RegisterBroker("broker2", broker2)

	brokerChecker1 := NewBrokerHealthChecker(broker1, "broker1", config)
	brokerChecker2 := NewBrokerHealthChecker(broker2, "broker2", config)
	checker.RegisterBrokerChecker("broker1", brokerChecker1)
	checker.RegisterBrokerChecker("broker2", brokerChecker2)

	t.Run("SuccessfulHealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := checker.Check(ctx)

		assert.NoError(t, err)
		assert.Equal(t, "messaging-subsystem", checker.GetServiceName())
	})

	t.Run("CoordinatorUnhealthy", func(t *testing.T) {
		coordinator.healthStatus.Overall = "unhealthy"

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unhealthy")

		// Reset
		coordinator.healthStatus.Overall = "healthy"
	})

	t.Run("BrokerUnhealthy", func(t *testing.T) {
		broker1.healthStatus.Status = HealthStatusUnhealthy

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "broker1")

		// Reset
		broker1.healthStatus.Status = HealthStatusHealthy
	})

	t.Run("FailureThreshold", func(t *testing.T) {
		// Trigger failures to exceed threshold
		coordinator.healthError = errors.New("coordinator error")

		for i := 0; i < config.FailureThreshold; i++ {
			ctx := context.Background()
			_ = checker.Check(ctx)
		}

		ctx := context.Background()
		err := checker.Check(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeded failure threshold")

		// Reset coordinator error and failure counters
		coordinator.healthError = nil
		checker.ResetFailureCounters()
	})

	t.Run("GetHealthStatus", func(t *testing.T) {
		ctx := context.Background()
		status, err := checker.GetHealthStatus(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, "healthy", status.Overall)
		assert.NotEmpty(t, status.BrokerHealth)
		assert.Contains(t, status.BrokerHealth, "broker1")
		assert.Contains(t, status.BrokerHealth, "broker2")
	})
}

func TestCreateMessagingHealthCheckers(t *testing.T) {
	coordinator := newHealthTestMessagingCoordinator()
	config := DefaultHealthCheckConfig()

	// Add some test brokers
	broker1 := newHealthTestMessageBroker()
	broker2 := newHealthTestMessageBroker()
	coordinator.RegisterBroker("broker1", broker1)
	coordinator.RegisterBroker("broker2", broker2)

	t.Run("SuccessfulCreation", func(t *testing.T) {
		coordinatorChecker, brokerCheckers, subsystemChecker, err := CreateMessagingHealthCheckers(coordinator, config)

		assert.NoError(t, err)
		assert.NotNil(t, coordinatorChecker)
		assert.Len(t, brokerCheckers, 2)
		assert.NotNil(t, subsystemChecker)

		assert.Equal(t, "messaging-coordinator", coordinatorChecker.GetServiceName())
		assert.Equal(t, "messaging-subsystem", subsystemChecker.GetServiceName())
	})

	t.Run("WithNilConfig", func(t *testing.T) {
		coordinatorChecker, brokerCheckers, subsystemChecker, err := CreateMessagingHealthCheckers(coordinator, nil)

		assert.NoError(t, err)
		assert.NotNil(t, coordinatorChecker)
		assert.Len(t, brokerCheckers, 2)
		assert.NotNil(t, subsystemChecker)

		// Should use default config
		assert.True(t, coordinatorChecker.GetHealthCheckConfig().Enabled)
	})

	t.Run("WithBrokerError", func(t *testing.T) {
		// Add a broker that will cause an error when retrieved
		coordinator.brokers["error-broker"] = nil

		coordinatorChecker, brokerCheckers, subsystemChecker, err := CreateMessagingHealthCheckers(coordinator, config)

		assert.NoError(t, err) // Should not fail, just skip problematic brokers
		assert.NotNil(t, coordinatorChecker)
		assert.Len(t, brokerCheckers, 2) // Should still have the 2 good brokers
		assert.NotNil(t, subsystemChecker)

		// Clean up
		delete(coordinator.brokers, "error-broker")
	})
}

func TestHealthCheckIntegration(t *testing.T) {
	coordinator := newHealthTestMessagingCoordinator()
	broker := newHealthTestMessageBroker()
	coordinator.RegisterBroker("test-broker", broker)

	handler := newHealthTestEventHandler("test-handler", []string{"test-topic"})
	coordinator.RegisterEventHandler(handler)

	config := DefaultHealthCheckConfig()

	t.Run("EndToEndHealthCheck", func(t *testing.T) {
		// Create all health checkers
		coordinatorChecker, brokerCheckers, subsystemChecker, err := CreateMessagingHealthCheckers(coordinator, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Test coordinator health check
		err = coordinatorChecker.Check(ctx)
		assert.NoError(t, err)

		// Test broker health checks
		for _, brokerChecker := range brokerCheckers {
			err = brokerChecker.Check(ctx)
			assert.NoError(t, err)
		}

		// Test subsystem health check
		err = subsystemChecker.Check(ctx)
		assert.NoError(t, err)
	})

	t.Run("CascadingFailures", func(t *testing.T) {
		// Simulate broker failure
		broker.healthStatus.Status = HealthStatusUnhealthy
		broker.healthStatus.Message = "Broker connection lost"

		coordinatorChecker, brokerCheckers, subsystemChecker, err := CreateMessagingHealthCheckers(coordinator, config)
		require.NoError(t, err)

		ctx := context.Background()

		// Coordinator should still be healthy
		err = coordinatorChecker.Check(ctx)
		assert.NoError(t, err)

		// Broker health check should fail
		for _, brokerChecker := range brokerCheckers {
			err = brokerChecker.Check(ctx)
			assert.Error(t, err)
		}

		// Subsystem health check should detect the broker issue
		err = subsystemChecker.Check(ctx)
		assert.Error(t, err)

		// Reset
		broker.healthStatus.Status = HealthStatusHealthy
	})
}
