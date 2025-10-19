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
	"strings"
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

// Mock DLQ components for testing

type MockDLQMessageSerializer struct {
	serializeFunc   func(msg *DLQMessage) ([]byte, error)
	deserializeFunc func(data []byte) (*DLQMessage, error)
}

func (m *MockDLQMessageSerializer) SerializeDLQMessage(msg *DLQMessage) ([]byte, error) {
	if m.serializeFunc != nil {
		return m.serializeFunc(msg)
	}
	return []byte("mock-serialized"), nil
}

func (m *MockDLQMessageSerializer) DeserializeDLQMessage(data []byte) (*DLQMessage, error) {
	if m.deserializeFunc != nil {
		return m.deserializeFunc(data)
	}
	return &DLQMessage{ID: "mock-deserialized"}, nil
}

type MockDLQRetryPolicy struct {
	shouldRetryFunc     func(msg *DLQMessage) bool
	getRetryDelayFunc   func(msg *DLQMessage) time.Duration
	isExpiredFunc       func(msg *DLQMessage) bool
	maxRecoveryAttempts int
}

func (m *MockDLQRetryPolicy) ShouldRetry(msg *DLQMessage) bool {
	if m.shouldRetryFunc != nil {
		return m.shouldRetryFunc(msg)
	}
	return true
}

func (m *MockDLQRetryPolicy) GetRetryDelay(msg *DLQMessage) time.Duration {
	if m.getRetryDelayFunc != nil {
		return m.getRetryDelayFunc(msg)
	}
	return 5 * time.Second
}

func (m *MockDLQRetryPolicy) IsExpired(msg *DLQMessage) bool {
	if m.isExpiredFunc != nil {
		return m.isExpiredFunc(msg)
	}
	return false
}

func (m *MockDLQRetryPolicy) GetMaxRecoveryAttempts() int {
	return m.maxRecoveryAttempts
}

type MockDLQErrorClassifier struct {
	classifyErrorFunc    func(err error) ErrorType
	isRetryableErrorFunc func(err error) bool
	getFailureReasonFunc func(err error) string
}

func (m *MockDLQErrorClassifier) ClassifyError(err error) ErrorType {
	if m.classifyErrorFunc != nil {
		return m.classifyErrorFunc(err)
	}
	return ErrorTypeUnknown
}

func (m *MockDLQErrorClassifier) IsRetryableError(err error) bool {
	if m.isRetryableErrorFunc != nil {
		return m.isRetryableErrorFunc(err)
	}
	return false
}

func (m *MockDLQErrorClassifier) GetFailureReason(err error) string {
	if m.getFailureReasonFunc != nil {
		return m.getFailureReasonFunc(err)
	}
	return "mock failure reason"
}

// Tests for DLQ functional options

func TestWithDLQMessageSerializer(t *testing.T) {
	tests := []struct {
		name          string
		serializer    DLQMessageSerializer
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid custom serializer",
			serializer:  &MockDLQMessageSerializer{},
			expectError: false,
		},
		{
			name:          "nil serializer",
			serializer:    nil,
			expectError:   true,
			errorContains: "DLQ message serializer cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DeadLetterConfig{
				Enabled: true,
				Topic:   "test-dlq",
			}
			publisher := NewMockEventPublisher()

			handler, err := NewDeadLetterQueueHandler(
				config,
				publisher,
				WithDLQMessageSerializer(tt.serializer),
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestWithDLQRetryPolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        DLQRetryPolicy
		expectError   bool
		errorContains string
	}{
		{
			name: "valid custom retry policy",
			policy: &MockDLQRetryPolicy{
				maxRecoveryAttempts: 10,
			},
			expectError: false,
		},
		{
			name:          "nil retry policy",
			policy:        nil,
			expectError:   true,
			errorContains: "DLQ retry policy cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DeadLetterConfig{
				Enabled: true,
				Topic:   "test-dlq",
			}
			publisher := NewMockEventPublisher()

			handler, err := NewDeadLetterQueueHandler(
				config,
				publisher,
				WithDLQRetryPolicy(tt.policy),
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestWithDLQErrorClassifier(t *testing.T) {
	tests := []struct {
		name          string
		classifier    DLQErrorClassifier
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid custom error classifier",
			classifier:  &MockDLQErrorClassifier{},
			expectError: false,
		},
		{
			name:          "nil error classifier",
			classifier:    nil,
			expectError:   true,
			errorContains: "DLQ error classifier cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DeadLetterConfig{
				Enabled: true,
				Topic:   "test-dlq",
			}
			publisher := NewMockEventPublisher()

			handler, err := NewDeadLetterQueueHandler(
				config,
				publisher,
				WithDLQErrorClassifier(tt.classifier),
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestDLQFunctionalOptionsIntegration(t *testing.T) {
	// Test that all functional options work together
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	mockSerializer := &MockDLQMessageSerializer{}
	mockPolicy := &MockDLQRetryPolicy{maxRecoveryAttempts: 7}
	mockClassifier := &MockDLQErrorClassifier{}

	handler, err := NewDeadLetterQueueHandler(
		config,
		publisher,
		WithDLQMessageSerializer(mockSerializer),
		WithDLQRetryPolicy(mockPolicy),
		WithDLQErrorClassifier(mockClassifier),
	)

	assert.NoError(t, err)
	assert.NotNil(t, handler)

	// Verify the handler was created successfully
	dlqConfig := handler.GetDLQConfiguration()
	assert.Equal(t, config.Topic, dlqConfig.Topic)
	assert.True(t, config.Enabled)
}

// Tests for DLQ component constructors

func TestNewDLQMessageSerializer(t *testing.T) {
	serializer := NewDefaultDLQMessageSerializer()
	assert.NotNil(t, serializer)

	// Test that it implements the interface
	var _ DLQMessageSerializer = serializer
}

func TestNewDLQRetryPolicy(t *testing.T) {
	tests := []struct {
		name                 string
		maxRecoveryAttempts  int
		baseDelay            time.Duration
		maxDelay             time.Duration
		multiplier           float64
		expectedMaxAttempts  int
		expectedBaseDelay    time.Duration
		expectedMaxDelay     time.Duration
		expectedMultiplier   float64
		testRecoveryAttempts int
	}{
		{
			name:                 "valid parameters",
			maxRecoveryAttempts:  10,
			baseDelay:            60 * time.Second,
			maxDelay:             60 * time.Minute,
			multiplier:           3.0,
			expectedMaxAttempts:  10,
			expectedBaseDelay:    60 * time.Second,
			expectedMaxDelay:     60 * time.Minute,
			expectedMultiplier:   3.0,
			testRecoveryAttempts: 1,
		},
		{
			name:                 "zero recovery attempts - should use default",
			maxRecoveryAttempts:  0,
			baseDelay:            60 * time.Second,
			maxDelay:             60 * time.Minute,
			multiplier:           3.0,
			expectedMaxAttempts:  5, // default
			expectedBaseDelay:    60 * time.Second,
			expectedMaxDelay:     60 * time.Minute,
			expectedMultiplier:   3.0,
			testRecoveryAttempts: 1,
		},
		{
			name:                 "negative recovery attempts - should use default",
			maxRecoveryAttempts:  -5,
			baseDelay:            60 * time.Second,
			maxDelay:             60 * time.Minute,
			multiplier:           3.0,
			expectedMaxAttempts:  5, // default
			expectedBaseDelay:    60 * time.Second,
			expectedMaxDelay:     60 * time.Minute,
			expectedMultiplier:   3.0,
			testRecoveryAttempts: 1,
		},
		{
			name:                 "zero base delay - should use default",
			maxRecoveryAttempts:  10,
			baseDelay:            0,
			maxDelay:             60 * time.Minute,
			multiplier:           3.0,
			expectedMaxAttempts:  10,
			expectedBaseDelay:    30 * time.Second, // default
			expectedMaxDelay:     60 * time.Minute,
			expectedMultiplier:   3.0,
			testRecoveryAttempts: 1,
		},
		{
			name:                 "zero max delay - should use default",
			maxRecoveryAttempts:  10,
			baseDelay:            60 * time.Second,
			maxDelay:             0,
			multiplier:           3.0,
			expectedMaxAttempts:  10,
			expectedBaseDelay:    60 * time.Second,
			expectedMaxDelay:     30 * time.Minute, // default
			expectedMultiplier:   3.0,
			testRecoveryAttempts: 1,
		},
		{
			name:                 "multiplier less than 1.0 - should use default",
			maxRecoveryAttempts:  10,
			baseDelay:            60 * time.Second,
			maxDelay:             60 * time.Minute,
			multiplier:           0.5,
			expectedMaxAttempts:  10,
			expectedBaseDelay:    60 * time.Second,
			expectedMaxDelay:     60 * time.Minute,
			expectedMultiplier:   2.0, // default
			testRecoveryAttempts: 1,
		},
		{
			name:                 "all zero values - should use all defaults",
			maxRecoveryAttempts:  0,
			baseDelay:            0,
			maxDelay:             0,
			multiplier:           0,
			expectedMaxAttempts:  5,                // default
			expectedBaseDelay:    30 * time.Second, // default
			expectedMaxDelay:     30 * time.Minute, // default
			expectedMultiplier:   2.0,              // default
			testRecoveryAttempts: 0,                // ensure it's retryable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := NewDLQRetryPolicy(tt.maxRecoveryAttempts, tt.baseDelay, tt.maxDelay, tt.multiplier)
			assert.NotNil(t, policy)

			// Test that it implements the interface
			var _ DLQRetryPolicy = policy

			// Test the methods work
			assert.Equal(t, tt.expectedMaxAttempts, policy.GetMaxRecoveryAttempts())

			// Test with a sample message
			testMsg := &DLQMessage{
				ID:               "test",
				RecoveryAttempts: tt.testRecoveryAttempts,
				ErrorType:        ErrorTypeRetryable,
				RetryCount:       0, // Set to 0 to ensure it's retryable
				MaxRetries:       3, // Set to a value > RetryCount to ensure it's retryable
			}

			delay := policy.GetRetryDelay(testMsg)
			assert.True(t, delay > 0)

			shouldRetry := policy.ShouldRetry(testMsg)
			assert.True(t, shouldRetry)
		})
	}
}

func TestNewDLQErrorClassifier(t *testing.T) {
	tests := []struct {
		name                string
		retryableErrors     []string
		permanentErrors     []string
		expectRetryable     bool
		expectPermanent     bool
		testError           error
		expectedErrorType   ErrorType
		expectedIsRetryable bool
	}{
		{
			name:                "custom retryable and permanent errors",
			retryableErrors:     []string{"custom timeout", "custom network"},
			permanentErrors:     []string{"custom invalid", "custom forbidden"},
			expectRetryable:     true,
			expectPermanent:     true,
			testError:           errors.New("custom timeout error"),
			expectedErrorType:   ErrorTypeTimeout,
			expectedIsRetryable: true,
		},
		{
			name:                "empty retryable errors - should use defaults",
			retryableErrors:     []string{},
			permanentErrors:     []string{"custom invalid"},
			expectRetryable:     false, // should use defaults
			expectPermanent:     true,
			testError:           errors.New("connection refused"),
			expectedErrorType:   ErrorTypeRetryable,
			expectedIsRetryable: true,
		},
		{
			name:                "empty permanent errors - should use defaults",
			retryableErrors:     []string{"custom timeout"},
			permanentErrors:     []string{},
			expectRetryable:     true,
			expectPermanent:     false, // should use defaults
			testError:           errors.New("invalid input format"),
			expectedErrorType:   ErrorTypeValidation,
			expectedIsRetryable: false,
		},
		{
			name:                "both empty - should use all defaults",
			retryableErrors:     []string{},
			permanentErrors:     []string{},
			expectRetryable:     false,
			expectPermanent:     false,
			testError:           errors.New("connection refused"),
			expectedErrorType:   ErrorTypeRetryable,
			expectedIsRetryable: true,
		},
		{
			name:                "custom permanent error matching",
			retryableErrors:     []string{"custom timeout"},
			permanentErrors:     []string{"invalid critical"},
			expectRetryable:     true,
			expectPermanent:     true,
			testError:           errors.New("invalid critical failure occurred"),
			expectedErrorType:   ErrorTypeValidation, // "invalid" matches permanent pattern in classifyByPattern
			expectedIsRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewDLQErrorClassifier(tt.retryableErrors, tt.permanentErrors)
			assert.NotNil(t, classifier)

			// Test that it implements the interface
			var _ DLQErrorClassifier = classifier

			// Test classification
			errorType := classifier.ClassifyError(tt.testError)
			assert.Equal(t, tt.expectedErrorType, errorType)

			// Test retryable check
			isRetryable := classifier.IsRetryableError(tt.testError)
			assert.Equal(t, tt.expectedIsRetryable, isRetryable)

			// Test failure reason
			failureReason := classifier.GetFailureReason(tt.testError)
			assert.NotEmpty(t, failureReason)
		})
	}
}

func TestDLQMessageSerializerEdgeCases(t *testing.T) {
	serializer := NewDefaultDLQMessageSerializer()

	t.Run("serialize nil message", func(t *testing.T) {
		data, err := serializer.SerializeDLQMessage(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DLQ message cannot be nil")
		assert.Nil(t, data)
	})

	t.Run("deserialize empty data", func(t *testing.T) {
		msg, err := serializer.DeserializeDLQMessage([]byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data cannot be empty")
		assert.Nil(t, msg)
	})

	t.Run("deserialize nil data", func(t *testing.T) {
		msg, err := serializer.DeserializeDLQMessage(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data cannot be empty")
		assert.Nil(t, msg)
	})

	t.Run("deserialize invalid JSON", func(t *testing.T) {
		invalidJSON := []byte("{ invalid json }")
		msg, err := serializer.DeserializeDLQMessage(invalidJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal DLQ message")
		assert.Nil(t, msg)
	})

	t.Run("round trip with complex message", func(t *testing.T) {
		now := time.Now()
		originalMsg := &DLQMessage{
			ID:               "complex-test-1",
			Error:            "complex error",
			ErrorType:        ErrorTypeRetryable,
			FailureReason:    "complex failure",
			RetryCount:       5,
			MaxRetries:       10,
			RecoveryAttempts: 2,
			SagaID:           "complex-saga-1",
			EventType:        EventTypeSagaStepFailed,
			HandlerID:        "complex-handler-1",
			FirstFailedAt:    now,
			LastFailedAt:     now.Add(time.Hour),
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
		}

		// Serialize
		data, err := serializer.SerializeDLQMessage(originalMsg)
		require.NoError(t, err)

		// Deserialize
		deserializedMsg, err := serializer.DeserializeDLQMessage(data)
		require.NoError(t, err)

		// Verify key fields
		assert.Equal(t, originalMsg.ID, deserializedMsg.ID)
		assert.Equal(t, originalMsg.Error, deserializedMsg.Error)
		assert.Equal(t, originalMsg.ErrorType, deserializedMsg.ErrorType)
		assert.Equal(t, originalMsg.FailureReason, deserializedMsg.FailureReason)
		assert.Equal(t, originalMsg.RetryCount, deserializedMsg.RetryCount)
		assert.Equal(t, originalMsg.MaxRetries, deserializedMsg.MaxRetries)
		assert.Equal(t, originalMsg.RecoveryAttempts, deserializedMsg.RecoveryAttempts)
		assert.Equal(t, originalMsg.SagaID, deserializedMsg.SagaID)
		assert.Equal(t, originalMsg.EventType, deserializedMsg.EventType)
		assert.Equal(t, originalMsg.HandlerID, deserializedMsg.HandlerID)
	})
}

// Tests for DLQ metrics update functions and internal methods

func TestDLQHandlerInternalMethods(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	handler, err := NewDeadLetterQueueHandler(config, publisher)
	require.NoError(t, err)

	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Cast to access internal methods
	defaultHandler := handler.(*defaultDeadLetterQueueHandler)

	t.Run("generateDLQMessageID", func(t *testing.T) {
		event := &saga.SagaEvent{
			ID:     "test-event-1",
			SagaID: "test-saga-1",
		}
		handlerCtx := &EventHandlerContext{
			MessageID: "test-message-1",
		}

		messageID := defaultHandler.generateDLQMessageID(event, handlerCtx)
		assert.NotEmpty(t, messageID)
		assert.Contains(t, messageID, "dlq_")
		assert.Contains(t, messageID, event.SagaID)
		assert.Contains(t, messageID, event.ID)
	})

	t.Run("getMaxRetries", func(t *testing.T) {
		handlerCtx := &EventHandlerContext{
			RetryCount: 2,
		}

		maxRetries := defaultHandler.getMaxRetries(handlerCtx)
		assert.Equal(t, 3, maxRetries) // default value
	})

	t.Run("getHandlerID with metadata", func(t *testing.T) {
		handlerCtx := &EventHandlerContext{
			Metadata: map[string]interface{}{
				"handler_id": "test-handler-123",
			},
		}

		handlerID := defaultHandler.getHandlerID(handlerCtx)
		assert.Equal(t, "test-handler-123", handlerID)
	})

	t.Run("getHandlerID without metadata", func(t *testing.T) {
		handlerCtx := &EventHandlerContext{}

		handlerID := defaultHandler.getHandlerID(handlerCtx)
		assert.Equal(t, "unknown", handlerID)
	})

	t.Run("getHandlerID with nil metadata", func(t *testing.T) {
		handlerCtx := &EventHandlerContext{
			Metadata: nil,
		}

		handlerID := defaultHandler.getHandlerID(handlerCtx)
		assert.Equal(t, "unknown", handlerID)
	})
}

func TestDLQMetricsUpdateFunctions(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	handler, err := NewDeadLetterQueueHandler(config, publisher)
	require.NoError(t, err)

	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Cast to access internal methods
	defaultHandler := handler.(*defaultDeadLetterQueueHandler)

	t.Run("updateMetricsExpired", func(t *testing.T) {
		// Set up initial state
		defaultHandler.metrics.TotalMessagesExpired = 10
		defaultHandler.metrics.CurrentQueueSize = 5

		// Call updateMetricsExpired
		defaultHandler.updateMetricsExpired()

		// Verify metrics were updated
		updatedMetrics := handler.GetDLQMetrics()
		assert.Equal(t, int64(11), updatedMetrics.TotalMessagesExpired)
		assert.Equal(t, int64(4), updatedMetrics.CurrentQueueSize)
	})

	t.Run("updateMetricsRecoveryFailed", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:        "test-recovery-failed",
			ErrorType: ErrorTypeRetryable,
		}
		recoveryTime := 250 * time.Millisecond

		// This function mainly updates Prometheus metrics, so we just test it doesn't panic
		defaultHandler.updateMetricsRecoveryFailed(testMessage, recoveryTime)

		// Verify handler is still healthy
		metrics := handler.GetDLQMetrics()
		assert.True(t, metrics.IsHealthy)
	})

	t.Run("updateMetricsError", func(t *testing.T) {
		testError := errors.New("test metrics error")

		// Get initial metrics
		initialMetrics := handler.GetDLQMetrics()
		assert.True(t, initialMetrics.IsHealthy)
		assert.Empty(t, initialMetrics.LastError)

		// Call updateMetricsError
		defaultHandler.updateMetricsError(testError)

		// Verify metrics were updated
		updatedMetrics := handler.GetDLQMetrics()
		assert.False(t, updatedMetrics.IsHealthy)
		assert.Equal(t, testError.Error(), updatedMetrics.LastError)
		assert.NotNil(t, updatedMetrics.LastErrorAt)
	})

	t.Run("updateMetricsSent", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:        "test-metrics-sent",
			ErrorType: ErrorTypeRetryable,
			EventType: EventTypeSagaStepFailed,
		}

		// Get initial metrics
		initialMetrics := handler.GetDLQMetrics()
		initialReceived := initialMetrics.TotalMessagesReceived
		initialQueueSize := initialMetrics.CurrentQueueSize

		// Call updateMetricsSent
		defaultHandler.updateMetricsSent(testMessage)

		// Verify metrics were updated
		updatedMetrics := handler.GetDLQMetrics()
		assert.Equal(t, initialReceived+1, updatedMetrics.TotalMessagesReceived)
		assert.Equal(t, initialQueueSize+1, updatedMetrics.CurrentQueueSize)
		assert.Equal(t, int64(1), updatedMetrics.MessagesByErrorType[ErrorTypeRetryable])
		assert.Equal(t, int64(1), updatedMetrics.MessagesByEventType[EventTypeSagaStepFailed])
		assert.NotNil(t, updatedMetrics.LastMessageReceivedAt)
	})

	t.Run("updateMetricsRecovered", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:        "test-metrics-recovered",
			ErrorType: ErrorTypeRetryable,
		}
		recoveryTime := 150 * time.Millisecond

		// Set up initial state
		defaultHandler.metrics.TotalMessagesReceived = 10
		defaultHandler.metrics.CurrentQueueSize = 5
		defaultHandler.metrics.TotalMessagesRecovered = 2
		defaultHandler.metrics.AverageRecoveryTime = 200 * time.Millisecond

		// Call updateMetricsRecovered
		defaultHandler.updateMetricsRecovered(testMessage, recoveryTime)

		// Verify metrics were updated
		updatedMetrics := handler.GetDLQMetrics()
		assert.Equal(t, int64(3), updatedMetrics.TotalMessagesRecovered)
		assert.Equal(t, int64(4), updatedMetrics.CurrentQueueSize)
		assert.True(t, updatedMetrics.AverageRecoveryTime > 0)
		assert.True(t, updatedMetrics.RecoverySuccessRate > 0)
		assert.NotNil(t, updatedMetrics.LastRecoveryAt)
	})
}

func TestDLQHandlerEdgeCases(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	handler, err := NewDeadLetterQueueHandler(config, publisher)
	require.NoError(t, err)

	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	// Cast to access internal methods
	defaultHandler := handler.(*defaultDeadLetterQueueHandler)

	t.Run("reprocessEvent simulated success", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:        "test-reprocess",
			ErrorType: ErrorTypeRetryable,
		}

		start := time.Now()
		err := defaultHandler.reprocessEvent(ctx, testMessage)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, duration >= 100*time.Millisecond) // Should simulate processing time
		assert.True(t, duration < 200*time.Millisecond)  // But not too long
	})

	t.Run("sendToDeadLetterQueue with nil error", func(t *testing.T) {
		event := &saga.SagaEvent{
			ID:     "test-event",
			SagaID: "test-saga",
		}
		handlerCtx := &EventHandlerContext{
			MessageID: "test-message",
		}

		err := handler.SendToDeadLetterQueue(ctx, event, handlerCtx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error cannot be nil")
	})

	t.Run("sendToDeadLetterQueue with nil event", func(t *testing.T) {
		handlerCtx := &EventHandlerContext{
			MessageID: "test-message",
		}
		testError := errors.New("test error")

		err := handler.SendToDeadLetterQueue(ctx, nil, handlerCtx, testError)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidEvent))
	})

	t.Run("sendToDeadLetterQueue with nil context", func(t *testing.T) {
		event := &saga.SagaEvent{
			ID:     "test-event",
			SagaID: "test-saga",
		}
		testError := errors.New("test error")

		err := handler.SendToDeadLetterQueue(ctx, event, nil, testError)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidContext))
	})

	t.Run("recoverFromDeadLetterQueue with nil message", func(t *testing.T) {
		err := handler.RecoverFromDeadLetterQueue(ctx, nil)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidEvent))
	})

	t.Run("recoverFromDeadLetterQueue with non-retryable message", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:        "test-non-retryable",
			ErrorType: ErrorTypePermanent,
		}

		err := handler.RecoverFromDeadLetterQueue(ctx, testMessage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message should not be retried")
	})

	t.Run("recoverFromDeadLetterQueue with expired message", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		testMessage := &DLQMessage{
			ID:               "test-expired",
			ErrorType:        ErrorTypeRetryable,
			ExpiresAt:        &pastTime,
			RecoveryAttempts: 0, // Should be retryable based on attempts
			RetryCount:       1,
			MaxRetries:       3,
		}

		err := handler.RecoverFromDeadLetterQueue(ctx, testMessage)
		assert.Error(t, err)
		// The error might be either expired or retry policy based on implementation order
		assert.True(t,
			strings.Contains(err.Error(), "message has expired") ||
				strings.Contains(err.Error(), "message should not be retried"),
			"Error should mention expiration or retry policy, got: %s", err.Error())
	})
}

func TestDLQRecoveryScenarios(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	handler, err := NewDeadLetterQueueHandler(config, publisher)
	require.NoError(t, err)

	ctx := context.Background()
	err = handler.Start(ctx)
	require.NoError(t, err)
	defer handler.Stop(ctx)

	t.Run("recovery with max attempts exceeded", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:               "test-max-attempts-exceeded",
			ErrorType:        ErrorTypeRetryable,
			RecoveryAttempts: 5, // Max attempts for default policy
			RetryCount:       2,
			MaxRetries:       3,
		}

		err := handler.RecoverFromDeadLetterQueue(ctx, testMessage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message should not be retried")
	})

	t.Run("recovery with nil expiration time", func(t *testing.T) {
		testMessage := &DLQMessage{
			ID:               "test-no-expiration",
			ErrorType:        ErrorTypeRetryable,
			ExpiresAt:        nil,
			RecoveryAttempts: 1,
			RetryCount:       1,
			MaxRetries:       3,
		}

		// Should not panic and should attempt recovery
		err := handler.RecoverFromDeadLetterQueue(ctx, testMessage)
		// May succeed or fail based on retry policy, but shouldn't panic
		if err != nil {
			assert.NotEmpty(t, err.Error())
		}
	})

	t.Run("recovery with custom retry policy", func(t *testing.T) {
		// Create a new handler with custom retry policy
		customPolicy := &MockDLQRetryPolicy{
			maxRecoveryAttempts: 1,
			shouldRetryFunc: func(msg *DLQMessage) bool {
				return msg.RecoveryAttempts < 1
			},
			getRetryDelayFunc: func(msg *DLQMessage) time.Duration {
				return 50 * time.Millisecond
			},
		}

		customHandler, err := NewDeadLetterQueueHandler(
			config,
			publisher,
			WithDLQRetryPolicy(customPolicy),
		)
		require.NoError(t, err)

		err = customHandler.Start(ctx)
		require.NoError(t, err)
		defer customHandler.Stop(ctx)

		testMessage := &DLQMessage{
			ID:               "test-custom-policy",
			ErrorType:        ErrorTypeRetryable,
			RecoveryAttempts: 0,
			RetryCount:       1,
			MaxRetries:       3,
		}

		// First recovery should succeed
		err = customHandler.RecoverFromDeadLetterQueue(ctx, testMessage)
		assert.NoError(t, err)

		// Second recovery should fail due to custom policy
		testMessage.RecoveryAttempts = 1
		err = customHandler.RecoverFromDeadLetterQueue(ctx, testMessage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message should not be retried")
	})
}

func TestDLQHandlerLifecycleEdgeCases(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	handler, err := NewDeadLetterQueueHandler(config, publisher)
	require.NoError(t, err)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:     "test-event",
		SagaID: "test-saga",
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "test-message",
	}
	testError := errors.New("test error")

	t.Run("operations before start", func(t *testing.T) {
		// Should fail operations before start
		err := handler.SendToDeadLetterQueue(ctx, event, handlerCtx, testError)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrHandlerNotInitialized))

		err = handler.HealthCheck(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrHandlerNotInitialized))
	})

	t.Run("start handler", func(t *testing.T) {
		err = handler.Start(ctx)
		assert.NoError(t, err)
	})

	t.Run("operations after stop", func(t *testing.T) {
		// Stop the handler
		err = handler.Stop(ctx)
		assert.NoError(t, err)

		// Should fail operations after stop
		err = handler.SendToDeadLetterQueue(ctx, event, handlerCtx, testError)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrHandlerShutdown))

		err = handler.HealthCheck(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrHandlerShutdown))
	})

	t.Run("double start", func(t *testing.T) {
		// Create new handler for double start test
		newHandler, err := NewDeadLetterQueueHandler(config, publisher)
		require.NoError(t, err)

		err = newHandler.Start(ctx)
		assert.NoError(t, err)

		// Second start should fail
		err = newHandler.Start(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrHandlerAlreadyInitialized))

		newHandler.Stop(ctx)
	})

	t.Run("double stop", func(t *testing.T) {
		// Create new handler for double stop test
		newHandler, err := NewDeadLetterQueueHandler(config, publisher)
		require.NoError(t, err)

		err = newHandler.Start(ctx)
		assert.NoError(t, err)

		err = newHandler.Stop(ctx)
		assert.NoError(t, err)

		// Second stop should fail
		err = newHandler.Stop(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrHandlerShutdown))
	})
}
