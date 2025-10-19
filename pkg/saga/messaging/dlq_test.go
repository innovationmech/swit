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

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEventPublisher is a mock implementation of saga.EventPublisher for testing
type MockEventPublisher struct {
	events    []*saga.SagaEvent
	publishFn func(ctx context.Context, event *saga.SagaEvent) error
}

func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		events: make([]*saga.SagaEvent, 0),
	}
}

func (m *MockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	m.events = append(m.events, event)
	if m.publishFn != nil {
		return m.publishFn(ctx, event)
	}
	return nil
}

func (m *MockEventPublisher) GetEvents() []*saga.SagaEvent {
	return m.events
}

func (m *MockEventPublisher) Clear() {
	m.events = make([]*saga.SagaEvent, 0)
}

func (m *MockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	// Mock implementation - not used in DLQ tests
	return nil, nil
}

func (m *MockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	// Mock implementation - not used in DLQ tests
	return nil
}

func (m *MockEventPublisher) Close() error {
	// Mock implementation - not used in DLQ tests
	return nil
}

func TestNewDeadLetterQueueHandler(t *testing.T) {
	tests := []struct {
		name        string
		config      *DeadLetterConfig
		publisher   saga.EventPublisher
		expectError bool
	}{
		{
			name: "valid config and publisher",
			config: &DeadLetterConfig{
				Enabled:                true,
				Topic:                  "dlq-topic",
				MaxRetentionDays:       7,
				IncludeOriginalMessage: true,
				IncludeErrorDetails:    true,
			},
			publisher:   NewMockEventPublisher(),
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			publisher:   NewMockEventPublisher(),
			expectError: true,
		},
		{
			name: "disabled DLQ",
			config: &DeadLetterConfig{
				Enabled: false,
			},
			publisher:   NewMockEventPublisher(),
			expectError: true,
		},
		{
			name: "nil publisher",
			config: &DeadLetterConfig{
				Enabled: true,
				Topic:   "dlq-topic",
			},
			publisher:   nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewDeadLetterQueueHandler(tt.config, tt.publisher)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestDLQMessageSerialization(t *testing.T) {
	serializer := NewDefaultDLQMessageSerializer()

	// Create a test DLQ message
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "test-saga-1",
		Type:      saga.EventSagaStepFailed,
		Timestamp: time.Now(),
	}

	handlerCtx := &EventHandlerContext{
		MessageID:  "test-message-1",
		Topic:      "test-topic",
		RetryCount: 2,
	}

	dlqMessage := &DLQMessage{
		ID:             "dlq-test-1",
		OriginalEvent:  event,
		HandlerContext: handlerCtx,
		Error:          "test error",
		ErrorType:      ErrorTypeRetryable,
		FailureReason:  "temporary failure",
		RetryCount:     2,
		MaxRetries:     3,
		FirstFailedAt:  time.Now(),
		LastFailedAt:   time.Now(),
		SagaID:         event.SagaID,
		EventType:      SagaEventType(event.Type),
		HandlerID:      "test-handler",
	}

	// Test serialization
	data, err := serializer.SerializeDLQMessage(dlqMessage)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Test deserialization
	deserializedMessage, err := serializer.DeserializeDLQMessage(data)
	require.NoError(t, err)
	require.NotNil(t, deserializedMessage)

	// Verify key fields
	assert.Equal(t, dlqMessage.ID, deserializedMessage.ID)
	assert.Equal(t, dlqMessage.Error, deserializedMessage.Error)
	assert.Equal(t, dlqMessage.ErrorType, deserializedMessage.ErrorType)
	assert.Equal(t, dlqMessage.SagaID, deserializedMessage.SagaID)
	assert.Equal(t, dlqMessage.EventType, deserializedMessage.EventType)
	assert.Equal(t, dlqMessage.RetryCount, deserializedMessage.RetryCount)
}

func TestDLQRetryPolicy(t *testing.T) {
	policy := NewDefaultDLQRetryPolicy()

	tests := []struct {
		name          string
		message       *DLQMessage
		expectedRetry bool
		expectedDelay time.Duration
		description   string
	}{
		{
			name: "retryable error with attempts remaining",
			message: &DLQMessage{
				ErrorType:        ErrorTypeRetryable,
				RecoveryAttempts: 1,
				RetryCount:       2,
				MaxRetries:       3,
			},
			expectedRetry: true,
			expectedDelay: 30 * time.Second, // Base delay
			description:   "Should retry when attempts remain and error is retryable",
		},
		{
			name: "permanent error",
			message: &DLQMessage{
				ErrorType:        ErrorTypePermanent,
				RecoveryAttempts: 0,
				RetryCount:       1,
				MaxRetries:       3,
			},
			expectedRetry: false,
			expectedDelay: 0,
			description:   "Should not retry permanent errors",
		},
		{
			name: "max recovery attempts exceeded",
			message: &DLQMessage{
				ErrorType:        ErrorTypeRetryable,
				RecoveryAttempts: 5,
				RetryCount:       2,
				MaxRetries:       3,
			},
			expectedRetry: false,
			expectedDelay: 0,
			description:   "Should not retry when max recovery attempts exceeded",
		},
		{
			name: "expired message",
			message: &DLQMessage{
				ErrorType:        ErrorTypeRetryable,
				RecoveryAttempts: 1,
				RetryCount:       2,
				MaxRetries:       3,
				ExpiresAt:        &[]time.Time{time.Now().Add(-time.Hour)}[0], // Expired 1 hour ago
			},
			expectedRetry: false,
			expectedDelay: 0,
			description:   "Should not retry expired messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRetry := policy.ShouldRetry(tt.message)
			assert.Equal(t, tt.expectedRetry, shouldRetry, tt.description)

			if shouldRetry {
				delay := policy.GetRetryDelay(tt.message)
				// Allow tolerance for delay calculation including jitter (up to 5% jitter + buffer)
				jitterTolerance := time.Duration(float64(tt.expectedDelay) * 0.1)
				tolerance := jitterTolerance + 2*time.Second
				assert.True(t, delay >= tt.expectedDelay-tolerance && delay <= tt.expectedDelay+tolerance,
					"Expected delay around %v, got %v", tt.expectedDelay, delay)
			}
		})
	}
}

func TestDLQErrorClassifier(t *testing.T) {
	classifier := NewDefaultDLQErrorClassifier()

	tests := []struct {
		name          string
		err           error
		expectedType  ErrorType
		expectedRetry bool
		description   string
	}{
		{
			name:          "timeout error",
			err:           errors.New("operation timed out"),
			expectedType:  ErrorTypeTimeout,
			expectedRetry: true,
			description:   "Should classify timeout errors correctly",
		},
		{
			name:          "validation error",
			err:           errors.New("invalid input format"),
			expectedType:  ErrorTypeValidation,
			expectedRetry: false,
			description:   "Should classify validation errors as non-retryable",
		},
		{
			name:          "network error",
			err:           errors.New("connection refused"),
			expectedType:  ErrorTypeRetryable,
			expectedRetry: true,
			description:   "Should classify network errors as retryable",
		},
		{
			name:          "rate limit error",
			err:           errors.New("rate limit exceeded"),
			expectedType:  ErrorTypeRateLimit,
			expectedRetry: true,
			description:   "Should classify rate limit errors correctly",
		},
		{
			name:          "unknown error",
			err:           errors.New("something unexpected happened"),
			expectedType:  ErrorTypeUnknown,
			expectedRetry: false,
			description:   "Should classify unknown errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := classifier.ClassifyError(tt.err)
			assert.Equal(t, tt.expectedType, errorType, tt.description)

			isRetryable := classifier.IsRetryableError(tt.err)
			assert.Equal(t, tt.expectedRetry, isRetryable, tt.description)

			failureReason := classifier.GetFailureReason(tt.err)
			assert.NotEmpty(t, failureReason)
		})
	}
}

func TestDLQHandlerSendToDeadLetterQueue(t *testing.T) {
	// Setup
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled:                true,
		Topic:                  "test-dlq",
		MaxRetentionDays:       7,
		IncludeOriginalMessage: true,
		IncludeErrorDetails:    true,
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	// Start the handler
	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Create test event and context
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "test-saga-1",
		Type:      saga.EventSagaStepFailed,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
	}

	handlerCtx := &EventHandlerContext{
		MessageID:           "test-message-1",
		Topic:               "test-topic",
		RetryCount:          2,
		OriginalMessageData: []byte(`{"test": "data"}`),
	}

	testError := errors.New("test error for DLQ")

	// Test sending to DLQ
	err = handler.SendToDeadLetterQueue(ctx, event, handlerCtx, testError)
	assert.NoError(t, err)

	// Verify event was published
	publishedEvents := mockPublisher.GetEvents()
	assert.Len(t, publishedEvents, 1)

	dlqEvent := publishedEvents[0]
	assert.Equal(t, saga.EventDeadLettered, dlqEvent.Type)
	assert.Equal(t, event.SagaID, dlqEvent.SagaID)
	assert.NotNil(t, dlqEvent.Data)

	// Check metrics
	metrics := handler.GetDLQMetrics()
	assert.Equal(t, int64(1), metrics.TotalMessagesReceived)
	assert.Equal(t, int64(1), metrics.CurrentQueueSize)
	assert.NotNil(t, metrics.LastMessageReceivedAt)
}

func TestDLQHandlerRecovery(t *testing.T) {
	// Setup
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled:          true,
		Topic:            "test-dlq",
		MaxRetentionDays: 7,
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	// Start the handler
	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Create a test DLQ message
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "test-saga-1",
		Type:      saga.EventSagaStepFailed,
		Timestamp: time.Now(),
	}

	handlerCtx := &EventHandlerContext{
		MessageID:  "test-message-1",
		Topic:      "test-topic",
		RetryCount: 1,
	}

	dlqMessage := &DLQMessage{
		ID:             "dlq-test-1",
		OriginalEvent:  event,
		HandlerContext: handlerCtx,
		Error:          "temporary network error",
		ErrorType:      ErrorTypeRetryable,
		FailureReason:  "temporary failure",
		RetryCount:     1,
		MaxRetries:     3,
		FirstFailedAt:  time.Now().Add(-5 * time.Minute),
		LastFailedAt:   time.Now(),
		SagaID:         event.SagaID,
		EventType:      SagaEventType(event.Type),
		HandlerID:      "test-handler",
	}

	// First send the message to DLQ to increment TotalMessagesReceived
	err = handler.SendToDeadLetterQueue(ctx, event, handlerCtx, errors.New("temporary network error"))
	assert.NoError(t, err)

	// Check that message was received
	initialMetrics := handler.GetDLQMetrics()
	assert.Equal(t, int64(1), initialMetrics.TotalMessagesReceived)

	// Test recovery
	err = handler.RecoverFromDeadLetterQueue(ctx, dlqMessage)
	assert.NoError(t, err)

	// Check metrics
	metrics := handler.GetDLQMetrics()
	assert.Equal(t, int64(1), metrics.TotalMessagesRecovered)
	assert.True(t, metrics.RecoverySuccessRate > 0)
	assert.NotNil(t, metrics.LastRecoveryAt)
}

func TestDLQMetrics(t *testing.T) {
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled:          true,
		Topic:            "test-dlq",
		MaxRetentionDays: 7,
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	// Start the handler
	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Get initial metrics
	metrics := handler.GetDLQMetrics()
	assert.Equal(t, int64(0), metrics.TotalMessagesReceived)
	assert.Equal(t, int64(0), metrics.TotalMessagesRecovered)
	assert.True(t, metrics.IsHealthy)

	// Send a message to DLQ
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "test-saga-1",
		Type:      saga.EventSagaStepFailed,
		Timestamp: time.Now(),
	}

	handlerCtx := &EventHandlerContext{
		MessageID:  "test-message-1",
		Topic:      "test-topic",
		RetryCount: 1,
	}

	err = handler.SendToDeadLetterQueue(ctx, event, handlerCtx, errors.New("test error"))
	require.NoError(t, err)

	// Check updated metrics
	metrics = handler.GetDLQMetrics()
	assert.Equal(t, int64(1), metrics.TotalMessagesReceived)
	assert.Equal(t, int64(1), metrics.CurrentQueueSize)
	assert.NotNil(t, metrics.LastMessageReceivedAt)
}

func TestDLQHandlerConfiguration(t *testing.T) {
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled:          true,
		Topic:            "test-dlq",
		MaxRetentionDays: 7,
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	// Test getting configuration
	retrievedConfig := handler.GetDLQConfiguration()
	assert.Equal(t, config.Enabled, retrievedConfig.Enabled)
	assert.Equal(t, config.Topic, retrievedConfig.Topic)
	assert.Equal(t, config.MaxRetentionDays, retrievedConfig.MaxRetentionDays)

	// Test updating configuration
	newConfig := &DeadLetterConfig{
		Enabled:          true,
		Topic:            "updated-dlq",
		MaxRetentionDays: 14,
	}

	err = handler.UpdateDLQConfiguration(newConfig)
	assert.NoError(t, err)

	// Verify configuration was updated
	updatedConfig := handler.GetDLQConfiguration()
	assert.Equal(t, newConfig.Topic, updatedConfig.Topic)
	assert.Equal(t, newConfig.MaxRetentionDays, updatedConfig.MaxRetentionDays)
}

func TestDLQHandlerHealthCheck(t *testing.T) {
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	ctx := context.Background()

	// Health check should fail before starting
	err = handler.HealthCheck(ctx)
	assert.Error(t, err)

	// Start the handler
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Health check should pass after starting
	err = handler.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestDLQRecoveryProcessInvalidInterval(t *testing.T) {
	// Setup
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	ctx := context.Background()

	// Start the handler
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Test with zero interval
	err = handler.StartDLQRecovery(ctx, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recovery interval must be positive")

	// Test with negative interval
	err = handler.StartDLQRecovery(ctx, -1*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recovery interval must be positive")

	// Test with positive interval (should work)
	err = handler.StartDLQRecovery(ctx, 100*time.Millisecond)
	assert.NoError(t, err)
}

func TestDLQRecoveryProcess(t *testing.T) {
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	ctx := context.Background()

	// Start the handler
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Start recovery process
	err = handler.StartDLQRecovery(ctx, 100*time.Millisecond)
	assert.NoError(t, err)

	// Wait a bit for recovery to run
	time.Sleep(200 * time.Millisecond)

	// Stop recovery process
	handler.StopDLQRecovery()

	// Verify recovery process stopped by trying to start it again
	err = handler.StartDLQRecovery(ctx, 100*time.Millisecond)
	assert.NoError(t, err)
	handler.StopDLQRecovery()
}

func TestDLQCleanupExpiredMessages(t *testing.T) {
	mockPublisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}

	handler, err := NewDeadLetterQueueHandler(config, mockPublisher)
	require.NoError(t, err)

	ctx := context.Background()

	// Start the handler
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Test cleanup (this is a placeholder test)
	cleanedUp, err := handler.CleanupExpiredMessages(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, cleanedUp, 0)
}

// Integration test for DLQ handler with saga event handler
func TestDLQIntegrationWithSagaEventHandler(t *testing.T) {
	// Setup
	mockPublisher := NewMockEventPublisher()
	dlqConfig := &DeadLetterConfig{
		Enabled:          true,
		Topic:            "integration-dlq",
		MaxRetentionDays: 7,
	}

	dlqHandler, err := NewDeadLetterQueueHandler(dlqConfig, mockPublisher)
	require.NoError(t, err)

	// Create saga event handler with DLQ support
	handlerConfig := &HandlerConfig{
		HandlerID:         "integration-test-handler",
		HandlerName:       "Integration Test Handler",
		Topics:            []string{"test-topic"},
		Concurrency:       1,
		ProcessingTimeout: 30 * time.Second,
		DeadLetterConfig:  dlqConfig,
	}

	sagaHandler, err := NewSagaEventHandler(
		handlerConfig,
		WithCoordinator(&mockCoordinator{}),
		WithEventPublisher(mockPublisher),
		WithDLQHandler(dlqHandler),
	)
	require.NoError(t, err)

	// Start handlers
	ctx := context.Background()
	err = dlqHandler.Start(ctx)
	require.NoError(t, err)
	defer dlqHandler.Stop(ctx)

	// Note: SagaEventHandler doesn't have Start/Stop methods in this interface
	// The handler lifecycle is managed externally

	// Create a test event that will fail
	event := &saga.SagaEvent{
		ID:        "integration-test-event",
		SagaID:    "integration-test-saga",
		Type:      saga.EventSagaStepFailed,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	handlerCtx := &EventHandlerContext{
		MessageID:  "integration-test-message",
		Topic:      "test-topic",
		RetryCount: 3, // Max retries reached
	}

	testError := errors.New("integration test error - should go to DLQ")

	// Send to DLQ using the saga handler's DLQ mechanism
	err = sagaHandler.(*defaultSagaEventHandler).sendToDeadLetterQueue(ctx, event, handlerCtx, testError)
	assert.NoError(t, err)

	// Verify DLQ handler metrics
	dlqMetrics := dlqHandler.GetDLQMetrics()
	assert.Equal(t, int64(1), dlqMetrics.TotalMessagesReceived)

	// Verify event was published
	publishedEvents := mockPublisher.GetEvents()
	assert.Len(t, publishedEvents, 1)
	assert.Equal(t, saga.EventDeadLettered, publishedEvents[0].Type)
}
