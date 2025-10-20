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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMessageBroker is a mock implementation of messaging.MessageBroker for testing.
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

func (m *MockMessageBroker) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMessageBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	args := m.Called(config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(messaging.EventPublisher), args.Error(1)
}

func (m *MockMessageBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	args := m.Called(config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(messaging.EventSubscriber), args.Error(1)
}

func (m *MockMessageBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messaging.HealthStatus), args.Error(1)
}

func (m *MockMessageBroker) GetMetrics() *messaging.BrokerMetrics {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*messaging.BrokerMetrics)
}

func (m *MockMessageBroker) GetCapabilities() *messaging.BrokerCapabilities {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*messaging.BrokerCapabilities)
}

func (m *MockMessageBroker) GetConfig() *messaging.BrokerConfig {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*messaging.BrokerConfig)
}

func (m *MockMessageBroker) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockMessagingEventPublisher is a mock implementation of messaging.EventPublisher for testing.
type MockMessagingEventPublisher struct {
	mock.Mock
	mu        sync.Mutex
	published []*messaging.Message
}

func (m *MockMessagingEventPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, message)
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessagingEventPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, messages...)
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *MockMessagingEventPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, message)
	args := m.Called(ctx, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messaging.PublishConfirmation), args.Error(1)
}

func (m *MockMessagingEventPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	args := m.Called(ctx, message, callback)
	return args.Error(0)
}

func (m *MockMessagingEventPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(messaging.Transaction), args.Error(1)
}

func (m *MockMessagingEventPublisher) Flush(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMessagingEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMessagingEventPublisher) GetMetrics() *messaging.PublisherMetrics {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*messaging.PublisherMetrics)
}

// MockEventSubscriber is a mock implementation of messaging.EventSubscriber for testing.
type MockEventSubscriber struct {
	mock.Mock
	handler messaging.MessageHandler
}

func (m *MockEventSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	m.handler = handler
	args := m.Called(ctx, handler)
	return args.Error(0)
}

func (m *MockEventSubscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	m.handler = handler
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

func (m *MockEventSubscriber) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEventSubscriber) GetMetrics() *messaging.SubscriberMetrics {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*messaging.SubscriberMetrics)
}

func (m *MockEventSubscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	args := m.Called(ctx, position)
	return args.Error(0)
}

func (m *MockEventSubscriber) GetLag(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockSagaCoordinator is a mock implementation of saga.SagaCoordinator for testing.
type MockSagaCoordinator struct {
	mock.Mock
}

func (m *MockSagaCoordinator) StartSaga(ctx context.Context, def saga.SagaDefinition, input interface{}) (saga.SagaInstance, error) {
	args := m.Called(ctx, def, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *MockSagaCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	args := m.Called(sagaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *MockSagaCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	args := m.Called(ctx, sagaID, reason)
	return args.Error(0)
}

func (m *MockSagaCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]saga.SagaInstance), args.Error(1)
}

func (m *MockSagaCoordinator) RecoverSagas(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSagaCoordinator) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestIntegrationConfigValidation tests IntegrationConfig validation.
func TestIntegrationConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *IntegrationConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &IntegrationConfig{
				Topics:          []string{"saga.events"},
				SubscriberGroup: "test-group",
				Publisher: &PublisherConfig{
					BrokerType:      "nats",
					BrokerEndpoints: []string{"nats://localhost:4222"},
					TopicPrefix:     "saga",
					SerializerType:  "json",
					Timeout:         30 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "missing topics",
			config: &IntegrationConfig{
				SubscriberGroup: "test-group",
				Publisher: &PublisherConfig{
					BrokerType:      "nats",
					BrokerEndpoints: []string{"nats://localhost:4222"},
					TopicPrefix:     "saga",
					SerializerType:  "json",
					Timeout:         30 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "missing subscriber group",
			config: &IntegrationConfig{
				Topics: []string{"saga.events"},
				Publisher: &PublisherConfig{
					BrokerType:      "nats",
					BrokerEndpoints: []string{"nats://localhost:4222"},
					TopicPrefix:     "saga",
					SerializerType:  "json",
					Timeout:         30 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "missing publisher config",
			config: &IntegrationConfig{
				Topics:          []string{"saga.events"},
				SubscriberGroup: "test-group",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIntegrationConfigSetDefaults tests IntegrationConfig default value setting.
func TestIntegrationConfigSetDefaults(t *testing.T) {
	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			SerializerType:  "json",
		},
	}

	config.SetDefaults()

	assert.Equal(t, 1, config.Concurrency)
	assert.Equal(t, 100, config.MessageBufferSize)
	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
	assert.NotNil(t, config.Publisher)
}

// TestNewMessagingIntegrationAdapter tests creation of MessagingIntegrationAdapter.
func TestNewMessagingIntegrationAdapter(t *testing.T) {
	tests := []struct {
		name    string
		broker  messaging.MessageBroker
		config  *IntegrationConfig
		wantErr bool
	}{
		{
			name:   "valid adapter",
			broker: &MockMessageBroker{},
			config: &IntegrationConfig{
				Topics:          []string{"saga.events"},
				SubscriberGroup: "test-group",
				Publisher: &PublisherConfig{
					BrokerType:      "nats",
					BrokerEndpoints: []string{"nats://localhost:4222"},
					TopicPrefix:     "saga",
					SerializerType:  "json",
					Timeout:         30 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil broker",
			broker:  nil,
			config:  &IntegrationConfig{},
			wantErr: true,
		},
		{
			name:    "nil config",
			broker:  &MockMessageBroker{},
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewMessagingIntegrationAdapter(tt.broker, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, adapter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, adapter)
				assert.False(t, adapter.IsInitialized())
				assert.False(t, adapter.IsRunning())
			}
		})
	}
}

// TestIntegrationAdapterInitialize tests adapter initialization.
func TestIntegrationAdapterInitialize(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	// Set up mock expectations
	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	assert.True(t, adapter.IsInitialized())
	assert.NotNil(t, adapter.GetPublisher())

	mockBroker.AssertExpectations(t)
}

// TestIntegrationAdapterDoubleInitialize tests that double initialization is prevented.
func TestIntegrationAdapterDoubleInitialize(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Second initialization should fail
	err = adapter.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")
}

// TestIntegrationAdapterHealthCheck tests adapter health check.
func TestIntegrationAdapterHealthCheck(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)
	mockBroker.On("HealthCheck", mock.Anything).Return(&messaging.HealthStatus{
		Status:  "healthy",
		Message: "All systems operational",
	}, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Health check before initialization should fail
	err = adapter.HealthCheck(ctx)
	assert.Error(t, err)

	// Initialize adapter
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Health check after initialization should succeed
	err = adapter.HealthCheck(ctx)
	assert.NoError(t, err)

	mockBroker.AssertExpectations(t)
}

// TestMessageToSagaEvent tests message to Saga event conversion.
func TestMessageToSagaEvent(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Create a test Saga event
	testEvent := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Version:   "1.0",
		Data: map[string]interface{}{
			"test": "data",
		},
	}

	// Serialize to JSON using the messaging framework's serializer
	jsonSerializer := messaging.NewJSONSerializer(nil)
	payload, err := jsonSerializer.Serialize(ctx, testEvent)
	require.NoError(t, err)

	// Create message
	msg := &messaging.Message{
		ID:        "msg-123",
		Topic:     "saga.events",
		Payload:   payload,
		Timestamp: time.Now(),
		Headers: map[string]string{
			"saga_id":      testEvent.SagaID,
			"event_type":   string(testEvent.Type),
			"content_type": "application/json",
		},
	}

	// Convert message to Saga event
	convertedEvent, err := adapter.messageToSagaEvent(msg)
	require.NoError(t, err)
	assert.NotNil(t, convertedEvent)
	assert.Equal(t, testEvent.SagaID, convertedEvent.SagaID)
	assert.Equal(t, testEvent.Type, convertedEvent.Type)
}

// TestSagaEventToMessage tests Saga event to message conversion.
func TestSagaEventToMessage(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Create a test Saga event
	testEvent := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Version:   "1.0",
		StepID:    "step-1",
		TraceID:   "trace-123",
		SpanID:    "span-456",
		Data: map[string]interface{}{
			"test": "data",
		},
	}

	// Convert Saga event to message
	msg, err := adapter.sagaEventToMessage(testEvent)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, testEvent.ID, msg.ID)
	assert.Equal(t, "saga.saga.started", msg.Topic)
	assert.Equal(t, testEvent.SagaID, msg.Headers["saga_id"])
	assert.Equal(t, string(testEvent.Type), msg.Headers["event_type"])
	assert.Equal(t, testEvent.StepID, msg.Headers["step_id"])
	assert.Equal(t, testEvent.TraceID, msg.Headers["trace_id"])
	assert.Equal(t, testEvent.SpanID, msg.Headers["span_id"])
}

// TestIntegrationAdapterMetrics tests adapter metrics tracking.
func TestIntegrationAdapterMetrics(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	// Get metrics before initialization
	metrics := adapter.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalMessagesReceived.Load())
	assert.Equal(t, int64(0), metrics.TotalMessagesProcessed.Load())
	assert.Equal(t, int64(0), metrics.TotalMessagesFailed.Load())
	assert.True(t, metrics.IsHealthy.Load())
}

// TestIntegrationAdapterClose tests adapter close functionality.
func TestIntegrationAdapterClose(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)
	mockPublisher.On("Close").Return(nil)
	mockSubscriber.On("Close").Return(nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Close adapter
	err = adapter.Close()
	assert.NoError(t, err)
	assert.False(t, adapter.IsInitialized())

	mockPublisher.AssertExpectations(t)
	mockSubscriber.AssertExpectations(t)
}

// TestIntegrationAdapterCloseWhileRunning tests that close fails when adapter is still running.
func TestIntegrationAdapterCloseWhileRunning(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Manually mark as running
	adapter.isRunning.Store(true)

	// Try to close while running - should fail
	err = adapter.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "still running")
}

// TestIntegrationAdapterStartSubscriptionFailure tests that Start method properly handles subscription failures.
func TestIntegrationAdapterStartSubscriptionFailure(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	// Set up Subscribe to fail with an error (simulating broker outage, bad credentials, etc.)
	mockSubscriber.On("Subscribe", mock.Anything, mock.AnythingOfType("*messaging.messageHandlerAdapter")).Return(fmt.Errorf("broker connection failed"))

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Start should fail due to subscription error
	err = adapter.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start messaging integration adapter")
	assert.Contains(t, err.Error(), "subscription failed")
	assert.Contains(t, err.Error(), "broker connection failed")

	// Adapter should not be marked as running after subscription failure
	assert.False(t, adapter.IsRunning())

	// Health should be false due to subscription failure
	metrics := adapter.GetMetrics()
	assert.False(t, metrics.IsHealthy.Load())

	mockBroker.AssertExpectations(t)
	mockSubscriber.AssertExpectations(t)
}

// TestIntegrationAdapterStartContextCancellation tests that Start method respects context cancellation.
func TestIntegrationAdapterStartContextCancellation(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	// Set up Subscribe to hang (simulate slow broker response)
	subscribeCalled := make(chan struct{})
	mockSubscriber.On("Subscribe", mock.Anything, mock.AnythingOfType("*messaging.messageHandlerAdapter")).Run(func(args mock.Arguments) {
		close(subscribeCalled)
		// Block until context is cancelled
		ctx := args.Get(0).(context.Context)
		<-ctx.Done()
	}).Return(fmt.Errorf("context cancelled"))

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Create a cancellable context with short timeout
	cancelCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should fail due to context cancellation
	err = adapter.Start(cancelCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "startup cancelled")
	assert.Contains(t, err.Error(), "context deadline exceeded")

	// Adapter should not be marked as running after context cancellation
	assert.False(t, adapter.IsRunning())

	// Wait for Subscribe to be called to ensure proper test behavior
	select {
	case <-subscribeCalled:
		// Subscribe was called as expected
	case <-time.After(time.Second):
		t.Fatal("Subscribe was not called within expected time")
	}

	mockBroker.AssertExpectations(t)
	mockSubscriber.AssertExpectations(t)
}

// TestIntegrationAdapterStartSuccess tests that Start method succeeds when subscription works.
func TestIntegrationAdapterStartSuccess(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	// Set up Subscribe to succeed
	mockSubscriber.On("Subscribe", mock.Anything, mock.AnythingOfType("*messaging.messageHandlerAdapter")).Return(nil)
	mockSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	mockSubscriber.On("Close").Return(nil)
	mockPublisher.On("Close").Return(nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Start should succeed
	err = adapter.Start(ctx)
	assert.NoError(t, err)

	// Adapter should be marked as running after successful subscription
	assert.True(t, adapter.IsRunning())

	// Health should be true after successful subscription
	metrics := adapter.GetMetrics()
	assert.True(t, metrics.IsHealthy.Load())

	// Clean up - stop the adapter
	err = adapter.Stop(ctx)
	assert.NoError(t, err)
	err = adapter.Close()
	assert.NoError(t, err)

	mockBroker.AssertExpectations(t)
	mockSubscriber.AssertExpectations(t)
}

// TestIntegrationAdapterConcurrentStop tests that multiple concurrent Stop calls are handled safely.
func TestIntegrationAdapterConcurrentStop(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	// Set up Subscribe to succeed
	mockSubscriber.On("Subscribe", mock.Anything, mock.AnythingOfType("*messaging.messageHandlerAdapter")).Return(nil)
	mockSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	mockSubscriber.On("Close").Return(nil)
	mockPublisher.On("Close").Return(nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Start the adapter
	err = adapter.Start(ctx)
	require.NoError(t, err)
	assert.True(t, adapter.IsRunning())

	// Test concurrent Stop calls
	numGoroutines := 10
	errCh := make(chan error, numGoroutines)

	// Launch multiple goroutines calling Stop concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := adapter.Stop(ctx)
			errCh <- err
		}()
	}

	// Collect results
	var successCount int
	var errorCount int
	for i := 0; i < numGoroutines; i++ {
		err := <-errCh
		if err == nil {
			successCount++
		} else {
			errorCount++
			assert.Contains(t, err.Error(), "adapter not running")
		}
	}

	// Only one call should succeed, the rest should fail with "adapter not running"
	assert.Equal(t, 1, successCount, "Only one Stop call should succeed")
	assert.Equal(t, numGoroutines-1, errorCount, "Other Stop calls should fail with 'adapter not running'")

	// Adapter should not be running
	assert.False(t, adapter.IsRunning())

	// Clean up
	err = adapter.Close()
	assert.NoError(t, err)

	mockBroker.AssertExpectations(t)
	mockSubscriber.AssertExpectations(t)
}

// TestIntegrationAdapterStopWhenNotRunning tests that Stop returns appropriate error when adapter is not running.
func TestIntegrationAdapterStopWhenNotRunning(t *testing.T) {
	mockBroker := new(MockMessageBroker)
	mockPublisher := new(MockMessagingEventPublisher)
	mockSubscriber := new(MockEventSubscriber)

	config := &IntegrationConfig{
		Topics:          []string{"saga.events"},
		SubscriberGroup: "test-group",
		Publisher: &PublisherConfig{
			BrokerType:      "nats",
			BrokerEndpoints: []string{"nats://localhost:4222"},
			TopicPrefix:     "saga",
			SerializerType:  "json",
			Timeout:         30 * time.Second,
		},
	}

	mockBroker.On("IsConnected").Return(true)
	mockBroker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(mockPublisher, nil)
	mockBroker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(mockSubscriber, nil)

	adapter, err := NewMessagingIntegrationAdapter(mockBroker, config)
	require.NoError(t, err)

	ctx := context.Background()
	err = adapter.Initialize(ctx)
	require.NoError(t, err)

	// Stop when not running should fail
	err = adapter.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adapter not running")

	// Start and then stop the adapter to test subsequent calls
	mockSubscriber.On("Subscribe", mock.Anything, mock.AnythingOfType("*messaging.messageHandlerAdapter")).Return(nil)
	mockSubscriber.On("Unsubscribe", mock.Anything).Return(nil)
	mockSubscriber.On("Close").Return(nil)
	mockPublisher.On("Close").Return(nil)

	err = adapter.Start(ctx)
	require.NoError(t, err)

	err = adapter.Stop(ctx)
	assert.NoError(t, err)

	// Second call to Stop should also fail
	err = adapter.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adapter not running")

	// Clean up
	err = adapter.Close()
	assert.NoError(t, err)

	mockBroker.AssertExpectations(t)
	mockSubscriber.AssertExpectations(t)
}
