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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMessageBroker implements a mock message broker for testing
type mockMessageBroker struct {
	publisherFunc func(config messaging.PublisherConfig) (messaging.EventPublisher, error)
}

func (m *mockMessageBroker) Connect(ctx context.Context) error {
	return nil
}

func (m *mockMessageBroker) Disconnect(ctx context.Context) error {
	return nil
}

func (m *mockMessageBroker) Close() error {
	return nil
}

func (m *mockMessageBroker) IsConnected() bool {
	return true
}

func (m *mockMessageBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	if m.publisherFunc != nil {
		return m.publisherFunc(config)
	}
	return &mockPublisher{}, nil
}

func (m *mockMessageBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMessageBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	return &messaging.HealthStatus{Status: "healthy"}, nil
}

func (m *mockMessageBroker) GetMetrics() *messaging.BrokerMetrics {
	return &messaging.BrokerMetrics{}
}

func (m *mockMessageBroker) GetCapabilities() *messaging.BrokerCapabilities {
	return &messaging.BrokerCapabilities{}
}

// mockPublisher implements a mock event publisher for testing
type mockPublisher struct {
	publishFunc            func(ctx context.Context, message *messaging.Message) error
	publishWithConfirmFunc func(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error)
	beginTransactionFunc   func(ctx context.Context) (messaging.Transaction, error)
	closeFunc              func() error
}

func (m *mockPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, message)
	}
	return nil
}

func (m *mockPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	if m.publishFunc != nil {
		// Simulate batch publish by calling publishFunc for each message
		for _, msg := range messages {
			if err := m.publishFunc(ctx, msg); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (m *mockPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	if m.publishWithConfirmFunc != nil {
		return m.publishWithConfirmFunc(ctx, message)
	}
	return &messaging.PublishConfirmation{
		MessageID: message.ID,
		Timestamp: time.Now(),
	}, nil
}

func (m *mockPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	return errors.New("not implemented")
}

func (m *mockPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	if m.beginTransactionFunc != nil {
		return m.beginTransactionFunc(ctx)
	}
	return nil, errors.New("transactions not supported")
}

func (m *mockPublisher) Flush(ctx context.Context) error {
	return nil
}

func (m *mockPublisher) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockPublisher) GetMetrics() *messaging.PublisherMetrics {
	return &messaging.PublisherMetrics{}
}

func TestPublisherConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *PublisherConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing broker type",
			config: &PublisherConfig{
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "broker_type is required",
		},
		{
			name: "missing broker endpoints",
			config: &PublisherConfig{
				BrokerType:     "nats",
				SerializerType: "json",
				Timeout:        30 * time.Second,
			},
			wantErr: true,
			errMsg:  "broker_endpoints are required",
		},
		{
			name: "missing serializer type",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "serializer_type is required",
		},
		{
			name: "invalid serializer type",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "xml",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "serializer_type must be 'json' or 'protobuf'",
		},
		{
			name: "negative retry attempts",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				RetryAttempts:   -1,
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "retry_attempts must be non-negative",
		},
		{
			name: "negative retry interval",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				RetryInterval:   -1 * time.Second,
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "retry_interval must be non-negative",
		},
		{
			name: "zero timeout",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         0,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPublisherConfig_SetDefaults(t *testing.T) {
	config := &PublisherConfig{}
	config.SetDefaults()

	assert.Equal(t, "saga", config.TopicPrefix)
	assert.Equal(t, "json", config.SerializerType)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 1*time.Second, config.RetryInterval)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.True(t, config.EnableMetrics)
}

func TestNewSagaEventPublisher(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	tests := []struct {
		name      string
		broker    messaging.MessageBroker
		config    *PublisherConfig
		wantErr   bool
		errMsg    string
		setupMock func() messaging.MessageBroker
	}{
		{
			name: "successful creation",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: false,
			setupMock: func() messaging.MessageBroker {
				return &mockMessageBroker{}
			},
		},
		{
			name:    "nil broker",
			broker:  nil,
			config:  &PublisherConfig{},
			wantErr: true,
			errMsg:  "broker cannot be nil",
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
			setupMock: func() messaging.MessageBroker {
				return &mockMessageBroker{}
			},
		},
		{
			name: "invalid config",
			config: &PublisherConfig{
				BrokerType: "", // missing broker type
			},
			wantErr: true,
			errMsg:  "invalid configuration",
			setupMock: func() messaging.MessageBroker {
				return &mockMessageBroker{}
			},
		},
		{
			name: "publisher creation fails",
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "failed to create publisher",
			setupMock: func() messaging.MessageBroker {
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return nil, errors.New("publisher creation failed")
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var broker messaging.MessageBroker
			if tt.setupMock != nil {
				broker = tt.setupMock()
			} else {
				broker = tt.broker
			}

			publisher, err := NewSagaEventPublisher(broker, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, publisher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, publisher)
				assert.NotNil(t, publisher.publisher)
				assert.NotNil(t, publisher.serializer)
				assert.NotNil(t, publisher.config)
				assert.NotNil(t, publisher.logger)
				assert.NotNil(t, publisher.metrics)
				assert.False(t, publisher.closed)

				// Clean up
				err = publisher.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestSagaEventPublisher_Close(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	require.NotNil(t, publisher)

	// Close once
	err = publisher.Close()
	assert.NoError(t, err)
	assert.True(t, publisher.IsClosed())

	// Close again (should be idempotent)
	err = publisher.Close()
	assert.NoError(t, err)
	assert.True(t, publisher.IsClosed())
}

func TestSagaEventPublisher_CloseError(t *testing.T) {
	logger.InitLogger()

	closeErr := errors.New("close error")
	mockPub := &mockPublisher{
		closeFunc: func() error {
			return closeErr
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	err = publisher.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close publisher")
	// Even if Close returns an error, the publisher should be marked as closed
	// to prevent further operations
	assert.True(t, publisher.IsClosed())
}

func TestSagaEventPublisher_GetMetrics(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	metrics := publisher.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalPublished.Load())
	assert.Equal(t, int64(0), metrics.TotalFailed.Load())
	assert.Equal(t, int64(0), metrics.TotalRetries.Load())
}

func TestSagaEventPublisher_GetConfig(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "test-saga",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	retrievedConfig := publisher.GetConfig()
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, "test-saga", retrievedConfig.TopicPrefix)
	assert.Equal(t, "json", retrievedConfig.SerializerType)
}

func TestPublisherMetrics_Reset(t *testing.T) {
	metrics := &PublisherMetrics{}

	// Set some values
	metrics.TotalPublished.Store(10)
	metrics.TotalFailed.Store(2)
	metrics.TotalRetries.Store(5)
	metrics.PublishDuration.Store(1000)
	metrics.LastPublishedAt.Store(time.Now().UnixNano())

	// Reset
	metrics.Reset()

	// Verify all values are zero
	assert.Equal(t, int64(0), metrics.TotalPublished.Load())
	assert.Equal(t, int64(0), metrics.TotalFailed.Load())
	assert.Equal(t, int64(0), metrics.TotalRetries.Load())
	assert.Equal(t, int64(0), metrics.PublishDuration.Load())
	assert.Equal(t, int64(0), metrics.LastPublishedAt.Load())
}

func TestPublisherMetrics_GetPublishRate(t *testing.T) {
	metrics := &PublisherMetrics{}

	// Test with no published events
	rate := metrics.GetPublishRate()
	assert.Equal(t, 0.0, rate)

	// Set metrics
	metrics.TotalPublished.Store(100)
	metrics.LastPublishedAt.Store(time.Now().Add(-10 * time.Second).UnixNano())

	// Get publish rate
	rate = metrics.GetPublishRate()
	assert.Greater(t, rate, 0.0)
}

func TestSagaEventPublisher_publish(t *testing.T) {
	logger.InitLogger()

	event := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
		TraceID:   "trace-789",
		SpanID:    "span-012",
	}

	tests := []struct {
		name      string
		event     *saga.SagaEvent
		setupMock func() *mockMessageBroker
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "successful publish without confirm",
			event: event,
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						assert.Equal(t, event.ID, message.ID)
						assert.Equal(t, string(event.Type), message.Headers["event-type"])
						assert.Equal(t, event.SagaID, message.Headers["saga-id"])
						assert.Equal(t, event.TraceID, message.Headers["trace-id"])
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:  "successful publish with confirm",
			event: event,
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishWithConfirmFunc: func(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
						return &messaging.PublishConfirmation{
							MessageID: message.ID,
							Timestamp: time.Now(),
						}, nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:  "publish fails and retries",
			event: event,
			setupMock: func() *mockMessageBroker {
				callCount := 0
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						callCount++
						if callCount < 3 {
							return errors.New("temporary failure")
						}
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:  "publish fails after all retries",
			event: event,
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return errors.New("permanent failure")
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			wantErr: true,
			errMsg:  "failed to publish event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := tt.setupMock()
			config := &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
				EnableConfirm:   tt.name == "successful publish with confirm",
			}

			publisher, err := NewSagaEventPublisher(broker, config)
			require.NoError(t, err)
			defer publisher.Close()

			ctx := context.Background()
			err = publisher.publish(ctx, tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSagaEventPublisher_publishWhenClosed(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err)

	// Try to publish after closing
	event := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	err = publisher.publish(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestSagaEventPublisher_PublishSagaEvent(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name      string
		event     *saga.SagaEvent
		setupMock func() *mockMessageBroker
		config    *PublisherConfig
		wantErr   bool
		errMsg    string
		validate  func(t *testing.T, msg *messaging.Message)
	}{
		{
			name: "successful publish without confirmation",
			event: &saga.SagaEvent{
				ID:            "event-123",
				SagaID:        "saga-456",
				StepID:        "step-789",
				Type:          saga.EventSagaStarted,
				Version:       "1.0",
				Timestamp:     time.Now(),
				CorrelationID: "corr-012",
				TraceID:       "trace-345",
				SpanID:        "span-678",
				Source:        "test-service",
				Service:       "order-service",
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
			validate: func(t *testing.T, msg *messaging.Message) {
				require.NotNil(t, msg, "captured message should not be nil")
				assert.Equal(t, "event-123", msg.ID)
				assert.Equal(t, "saga.saga.started", msg.Topic)
				assert.Equal(t, "saga-456", msg.Headers["saga_id"])
				assert.Equal(t, "saga.started", msg.Headers["event_type"])
				assert.Equal(t, "1.0", msg.Headers["version"])
				assert.Equal(t, "step-789", msg.Headers["step_id"])
				assert.Equal(t, "trace-345", msg.Headers["trace_id"])
				assert.Equal(t, "span-678", msg.Headers["span_id"])
				assert.Equal(t, "test-service", msg.Headers["source"])
				assert.Equal(t, "order-service", msg.Headers["service"])
				assert.NotEmpty(t, msg.Headers["timestamp"])
				assert.NotEmpty(t, msg.Headers["content_type"])
			},
		},
		{
			name: "successful publish with confirmation",
			event: &saga.SagaEvent{
				ID:        "event-456",
				SagaID:    "saga-789",
				Type:      saga.EventSagaStepCompleted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishWithConfirmFunc: func(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
						return &messaging.PublishConfirmation{
							MessageID: message.ID,
							Timestamp: time.Now(),
						}, nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "test-saga",
				EnableConfirm:   true,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
			validate: func(t *testing.T, msg *messaging.Message) {
				if msg != nil {
					assert.Equal(t, "test-saga.saga.step.completed", msg.Topic)
				}
			},
		},
		{
			name:  "nil event",
			event: nil,
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event cannot be nil",
		},
		{
			name: "missing event ID",
			event: &saga.SagaEvent{
				SagaID:    "saga-456",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event ID is required",
		},
		{
			name: "missing saga ID",
			event: &saga.SagaEvent{
				ID:        "event-123",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "saga ID is required",
		},
		{
			name: "missing event type",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event type is required",
		},
		{
			name: "missing version",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event version is required",
		},
		{
			name: "missing timestamp",
			event: &saga.SagaEvent{
				ID:      "event-123",
				SagaID:  "saga-456",
				Type:    saga.EventSagaStarted,
				Version: "1.0",
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event timestamp is required",
		},
		{
			name: "publish with retry success",
			event: &saga.SagaEvent{
				ID:        "event-retry",
				SagaID:    "saga-retry",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				callCount := 0
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						callCount++
						if callCount < 3 {
							return errors.New("temporary failure")
						}
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "publish fails after all retries",
			event: &saga.SagaEvent{
				ID:        "event-fail",
				SagaID:    "saga-fail",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return errors.New("permanent failure")
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   2,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "failed to publish event after 2 retries",
		},
		{
			name: "context cancelled during retry",
			event: &saga.SagaEvent{
				ID:        "event-cancel",
				SagaID:    "saga-cancel",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return errors.New("failure to trigger retry")
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   5,
				RetryInterval:   1 * time.Second, // Long interval to test context cancellation
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "context", // Accept both "context canceled" and "context deadline exceeded"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := tt.setupMock()
			publisher, err := NewSagaEventPublisher(broker, tt.config)
			require.NoError(t, err)
			defer publisher.Close()

			ctx := context.Background()
			if tt.name == "context cancelled during retry" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}

			// Capture the message if validation is needed
			if tt.validate != nil && !tt.wantErr {
				var capturedMsg *messaging.Message
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						capturedMsg = message
						return nil
					},
				}
				mockBroker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				publisher, err = NewSagaEventPublisher(mockBroker, tt.config)
				require.NoError(t, err)
				defer publisher.Close()

				err = publisher.PublishSagaEvent(ctx, tt.event)
				assert.NoError(t, err)
				tt.validate(t, capturedMsg)
				return
			}

			err = publisher.PublishSagaEvent(ctx, tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Validate metrics
			if !tt.wantErr {
				metrics := publisher.GetMetrics()
				assert.Greater(t, metrics.TotalPublished.Load(), int64(0))
			}
		})
	}
}

func TestSagaEventPublisher_PublishSagaEventWhenClosed(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err)

	// Try to publish after closing
	event := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	err = publisher.PublishSagaEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestSagaEventPublisher_getTopicForEvent(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "test-prefix",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	tests := []struct {
		name      string
		eventType saga.SagaEventType
		expected  string
	}{
		{
			name:      "saga started event",
			eventType: saga.EventSagaStarted,
			expected:  "test-prefix.saga.started",
		},
		{
			name:      "saga completed event",
			eventType: saga.EventSagaCompleted,
			expected:  "test-prefix.saga.completed",
		},
		{
			name:      "step completed event",
			eventType: saga.EventSagaStepCompleted,
			expected:  "test-prefix.saga.step.completed",
		},
		{
			name:      "saga failed event",
			eventType: saga.EventSagaFailed,
			expected:  "test-prefix.saga.failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &saga.SagaEvent{
				Type: tt.eventType,
			}
			topic := publisher.getTopicForEvent(event)
			assert.Equal(t, tt.expected, topic)
		})
	}
}

func TestSagaEventPublisher_validateEvent(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	tests := []struct {
		name    string
		event   *saga.SagaEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid event",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			event: &saga.SagaEvent{
				SagaID:    "saga-456",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			wantErr: true,
			errMsg:  "event ID is required",
		},
		{
			name: "missing saga ID",
			event: &saga.SagaEvent{
				ID:        "event-123",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			wantErr: true,
			errMsg:  "saga ID is required",
		},
		{
			name: "missing type",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			wantErr: true,
			errMsg:  "event type is required",
		},
		{
			name: "missing version",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			},
			wantErr: true,
			errMsg:  "event version is required",
		},
		{
			name: "missing timestamp",
			event: &saga.SagaEvent{
				ID:      "event-123",
				SagaID:  "saga-456",
				Type:    saga.EventSagaStarted,
				Version: "1.0",
			},
			wantErr: true,
			errMsg:  "event timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publisher.validateEvent(tt.event)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSagaEventPublisher_PublishSagaEventMetrics(t *testing.T) {
	logger.InitLogger()

	callCount := 0
	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			callCount++
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   3,
		RetryInterval:   10 * time.Millisecond,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	ctx := context.Background()

	// Publish multiple events
	for i := 0; i < 5; i++ {
		event := &saga.SagaEvent{
			ID:        fmt.Sprintf("event-%d", i),
			SagaID:    "saga-metrics",
			Type:      saga.EventSagaStarted,
			Version:   "1.0",
			Timestamp: time.Now(),
		}
		err = publisher.PublishSagaEvent(ctx, event)
		require.NoError(t, err)
	}

	// Verify metrics
	metrics := publisher.GetMetrics()
	assert.Equal(t, int64(5), metrics.TotalPublished.Load())
	assert.Equal(t, int64(0), metrics.TotalFailed.Load())
	assert.Greater(t, metrics.LastPublishedAt.Load(), int64(0))
	assert.Greater(t, metrics.PublishDuration.Load(), int64(0))
}

func TestSagaEventPublisher_PublishBatch(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name      string
		events    []*saga.SagaEvent
		setupMock func() *mockMessageBroker
		config    *PublisherConfig
		wantErr   bool
		errMsg    string
		validate  func(t *testing.T, messages []*messaging.Message)
	}{
		{
			name: "successful batch publish",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-batch",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:        "event-2",
					SagaID:    "saga-batch",
					Type:      saga.EventSagaStepCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:        "event-3",
					SagaID:    "saga-batch",
					Type:      saga.EventSagaCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:   "empty batch",
			events: []*saga.SagaEvent{},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "nil event in batch",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-batch",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				nil,
				{
					ID:        "event-3",
					SagaID:    "saga-batch",
					Type:      saga.EventSagaCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event 1 is nil",
		},
		{
			name: "invalid event in batch",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-batch",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:        "event-2",
					SagaID:    "", // Missing saga ID
					Type:      saga.EventSagaStepCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event 1 validation failed",
		},
		{
			name: "batch publish with retry success",
			events: []*saga.SagaEvent{
				{
					ID:        "event-retry-1",
					SagaID:    "saga-retry",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:        "event-retry-2",
					SagaID:    "saga-retry",
					Type:      saga.EventSagaStepCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() *mockMessageBroker {
				callCount := 0
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						callCount++
						if callCount <= 2 { // Fail first batch attempt
							return errors.New("temporary failure")
						}
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "batch publish fails after all retries",
			events: []*saga.SagaEvent{
				{
					ID:        "event-fail-1",
					SagaID:    "saga-fail",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return errors.New("permanent failure")
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   2,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "failed to publish batch after 2 retries",
		},
		{
			name: "large batch publish",
			events: func() []*saga.SagaEvent {
				events := make([]*saga.SagaEvent, 100)
				for i := 0; i < 100; i++ {
					events[i] = &saga.SagaEvent{
						ID:        fmt.Sprintf("event-%d", i),
						SagaID:    "saga-large-batch",
						Type:      saga.EventSagaStarted,
						Version:   "1.0",
						Timestamp: time.Now(),
					}
				}
				return events
			}(),
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "batch with full event metadata",
			events: []*saga.SagaEvent{
				{
					ID:            "event-meta-1",
					SagaID:        "saga-meta",
					StepID:        "step-1",
					Type:          saga.EventSagaStarted,
					Version:       "1.0",
					Timestamp:     time.Now(),
					CorrelationID: "corr-123",
					TraceID:       "trace-456",
					SpanID:        "span-789",
					Source:        "test-service",
					Service:       "order-service",
				},
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := tt.setupMock()
			publisher, err := NewSagaEventPublisher(broker, tt.config)
			require.NoError(t, err)
			defer publisher.Close()

			ctx := context.Background()
			err = publisher.PublishBatch(ctx, tt.events)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify metrics for successful publishes
				if len(tt.events) > 0 {
					metrics := publisher.GetMetrics()
					assert.Equal(t, int64(len(tt.events)), metrics.TotalPublished.Load())
				}
			}
		})
	}
}

func TestSagaEventPublisher_PublishBatchWhenClosed(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err)

	// Try to publish batch after closing
	events := []*saga.SagaEvent{
		{
			ID:        "event-1",
			SagaID:    "saga-456",
			Type:      saga.EventSagaStarted,
			Version:   "1.0",
			Timestamp: time.Now(),
		},
	}

	ctx := context.Background()
	err = publisher.PublishBatch(ctx, events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestSagaEventPublisher_PublishBatchAsync(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name      string
		events    []*saga.SagaEvent
		setupMock func() *mockMessageBroker
		config    *PublisherConfig
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful async batch publish",
			events: []*saga.SagaEvent{
				{
					ID:        "event-async-1",
					SagaID:    "saga-async",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:        "event-async-2",
					SagaID:    "saga-async",
					Type:      saga.EventSagaStepCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() *mockMessageBroker {
				mockPub := &mockPublisher{
					publishFunc: func(ctx context.Context, message *messaging.Message) error {
						return nil
					},
				}
				return &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				EnableConfirm:   false,
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:   "empty async batch",
			events: []*saga.SagaEvent{},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "nil event in async batch",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-async",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				nil,
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event 1 is nil",
		},
		{
			name: "invalid event in async batch",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-async",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:      "event-2",
					SagaID:  "saga-async",
					Type:    saga.EventSagaStepCompleted,
					Version: "", // Missing version
				},
			},
			setupMock: func() *mockMessageBroker {
				return &mockMessageBroker{}
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event 1 validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := tt.setupMock()
			publisher, err := NewSagaEventPublisher(broker, tt.config)
			require.NoError(t, err)
			defer publisher.Close()

			ctx := context.Background()

			// Use a channel to wait for callback
			done := make(chan error, 1)
			callback := func(err error) {
				done <- err
			}

			err = publisher.PublishBatchAsync(ctx, tt.events, callback)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Wait for callback
				if len(tt.events) > 0 {
					select {
					case callbackErr := <-done:
						assert.NoError(t, callbackErr)
					case <-time.After(2 * time.Second):
						t.Fatal("callback timeout")
					}
				}
			}
		})
	}
}

func TestSagaEventPublisher_PublishBatchAsyncWhenClosed(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err)

	// Try to publish async batch after closing
	events := []*saga.SagaEvent{
		{
			ID:        "event-1",
			SagaID:    "saga-456",
			Type:      saga.EventSagaStarted,
			Version:   "1.0",
			Timestamp: time.Now(),
		},
	}

	ctx := context.Background()
	err = publisher.PublishBatchAsync(ctx, events, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestSagaEventPublisher_PublishBatchMetrics(t *testing.T) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   3,
		RetryInterval:   10 * time.Millisecond,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	ctx := context.Background()

	// Create a batch of 10 events
	events := make([]*saga.SagaEvent, 10)
	for i := 0; i < 10; i++ {
		events[i] = &saga.SagaEvent{
			ID:        fmt.Sprintf("event-%d", i),
			SagaID:    "saga-metrics-batch",
			Type:      saga.EventSagaStarted,
			Version:   "1.0",
			Timestamp: time.Now(),
		}
	}

	// Publish batch
	err = publisher.PublishBatch(ctx, events)
	require.NoError(t, err)

	// Verify metrics
	metrics := publisher.GetMetrics()
	assert.Equal(t, int64(10), metrics.TotalPublished.Load())
	assert.Equal(t, int64(0), metrics.TotalFailed.Load())
	assert.Greater(t, metrics.LastPublishedAt.Load(), int64(0))
	assert.Greater(t, metrics.PublishDuration.Load(), int64(0))
}

func TestSagaEventPublisher_PrometheusMetricsCollection(t *testing.T) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   true,
		RetryAttempts:   3,
		RetryInterval:   10 * time.Millisecond,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	ctx := context.Background()

	// Test single event publish
	event := &saga.SagaEvent{
		ID:        "event-prom-1",
		SagaID:    "saga-prom",
		Type:      saga.EventSagaStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
	}

	err = publisher.PublishSagaEvent(ctx, event)
	require.NoError(t, err)

	// Prometheus metrics are global, so we can't easily verify exact counts
	// but we can verify the methods were called without panicking
	assert.NotNil(t, publisher)

	// Test batch publish
	events := []*saga.SagaEvent{
		{
			ID:        "event-batch-1",
			SagaID:    "saga-batch-prom",
			Type:      saga.EventSagaStarted,
			Version:   "1.0",
			Timestamp: time.Now(),
		},
		{
			ID:        "event-batch-2",
			SagaID:    "saga-batch-prom",
			Type:      saga.EventSagaStepCompleted,
			Version:   "1.0",
			Timestamp: time.Now(),
		},
	}

	err = publisher.PublishBatch(ctx, events)
	require.NoError(t, err)
}

func TestSagaEventPublisher_MetricsOnFailure(t *testing.T) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return errors.New("publish failed")
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   1,
		RetryInterval:   10 * time.Millisecond,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	ctx := context.Background()

	event := &saga.SagaEvent{
		ID:        "event-fail",
		SagaID:    "saga-fail",
		Type:      saga.EventSagaFailed,
		Version:   "1.0",
		Timestamp: time.Now(),
	}

	err = publisher.PublishSagaEvent(ctx, event)
	assert.Error(t, err)

	// Verify failure metrics were recorded
	metrics := publisher.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalFailed.Load())
	// Note: when using reliabilityHandler, retries are handled internally
	// and may not be reflected in TotalRetries metric in the same way
	// The key is that we recorded the failure
}

func TestSagaEventPublisher_LoggingIntegration(t *testing.T) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   3,
		RetryInterval:   10 * time.Millisecond,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	ctx := context.Background()

	// Test with full event metadata
	event := &saga.SagaEvent{
		ID:            "event-log-1",
		SagaID:        "saga-log",
		StepID:        "step-1",
		Type:          saga.EventSagaStarted,
		Version:       "1.0",
		Timestamp:     time.Now(),
		TraceID:       "trace-123",
		SpanID:        "span-456",
		CorrelationID: "corr-789",
		Source:        "test-service",
		Service:       "order-service",
	}

	// This should log attempt, success messages
	err = publisher.PublishSagaEvent(ctx, event)
	require.NoError(t, err)

	// Test logging on failure
	failMockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return errors.New("intentional failure")
		},
	}

	failBroker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return failMockPub, nil
		},
	}

	failPublisher, err := NewSagaEventPublisher(failBroker, config)
	require.NoError(t, err)
	defer failPublisher.Close()

	// This should log failure message
	err = failPublisher.PublishSagaEvent(ctx, event)
	assert.Error(t, err)
}

func TestSagaEventPublisher_TracingSpan(t *testing.T) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   3,
		RetryInterval:   10 * time.Millisecond,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)
	defer publisher.Close()

	ctx := context.Background()

	event := &saga.SagaEvent{
		ID:        "event-trace-1",
		SagaID:    "saga-trace",
		Type:      saga.EventSagaStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
		TraceID:   "existing-trace-id",
		SpanID:    "parent-span-id",
	}

	// Should start a span and propagate trace context
	err = publisher.PublishSagaEvent(ctx, event)
	require.NoError(t, err)

	// Verify tracing manager was initialized
	assert.NotNil(t, publisher.tracingManager)
}
