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
	closeFunc              func() error
}

func (m *mockPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, message)
	}
	return nil
}

func (m *mockPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	return errors.New("not implemented")
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
	return nil, errors.New("not implemented")
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
