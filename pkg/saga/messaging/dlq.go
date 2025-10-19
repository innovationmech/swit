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
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// DeadLetterQueueHandler defines the interface for handling dead-letter queue operations.
// It provides functionality to send failed events to DLQ, recover messages from DLQ,
// and manage DLQ-related operations like monitoring and cleanup.
type DeadLetterQueueHandler interface {
	// Start initializes and starts the DLQ handler.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//
	// Returns:
	//   - error: nil if start succeeds, or an error describing the failure
	Start(ctx context.Context) error

	// Stop gracefully shuts down the DLQ handler.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//
	// Returns:
	//   - error: nil if shutdown succeeds, or an error
	Stop(ctx context.Context) error

	// SendToDeadLetterQueue sends a failed event to the dead-letter queue.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - event: The failed saga event to send to DLQ
	//   - handlerCtx: Additional context information about the failure
	//   - err: The error that caused the event to fail
	//
	// Returns:
	//   - error: nil if successfully sent to DLQ, or an error describing the failure
	SendToDeadLetterQueue(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext, err error) error

	// RecoverFromDeadLetterQueue attempts to recover and reprocess a message from DLQ.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - dlqMessage: The DLQ message to recover
	//
	// Returns:
	//   - error: nil if recovery was successful, or an error describing the failure
	RecoverFromDeadLetterQueue(ctx context.Context, dlqMessage *DLQMessage) error

	// GetDLQMetrics returns metrics about DLQ operations.
	//
	// Returns:
	//   - *DLQMetrics: Current DLQ metrics snapshot
	GetDLQMetrics() *DLQMetrics

	// GetDLQConfiguration returns the current DLQ configuration.
	//
	// Returns:
	//   - *DeadLetterConfig: Current DLQ configuration
	GetDLQConfiguration() *DeadLetterConfig

	// UpdateDLQConfiguration updates the DLQ configuration at runtime.
	//
	// Parameters:
	//   - config: New DLQ configuration
	//
	// Returns:
	//   - error: nil if update succeeds, or an error
	UpdateDLQConfiguration(config *DeadLetterConfig) error

	// StartDLQRecovery starts the background DLQ recovery process.
	//
	// Parameters:
	//   - ctx: Context for the recovery process
	//   - recoveryInterval: How often to attempt recovery
	//
	// Returns:
	//   - error: nil if recovery started successfully, or an error
	StartDLQRecovery(ctx context.Context, recoveryInterval time.Duration) error

	// StopDLQRecovery stops the background DLQ recovery process.
	StopDLQRecovery()

	// CleanupExpiredMessages removes expired messages from the DLQ.
	//
	// Parameters:
	//   - ctx: Context for the cleanup operation
	//
	// Returns:
	//   - int: Number of messages cleaned up
	//   - error: nil if cleanup succeeds, or an error
	CleanupExpiredMessages(ctx context.Context) (int, error)

	// HealthCheck verifies the DLQ handler is functioning correctly.
	//
	// Parameters:
	//   - ctx: Context for the health check
	//
	// Returns:
	//   - error: nil if healthy, or an error describing the issue
	HealthCheck(ctx context.Context) error
}

// DLQMessage represents a message in the dead-letter queue.
// It contains the original event, error information, and metadata for recovery attempts.
type DLQMessage struct {
	// ID is the unique identifier for this DLQ message.
	ID string `json:"id"`

	// OriginalEvent is the saga event that failed to process.
	OriginalEvent *saga.SagaEvent `json:"original_event"`

	// HandlerContext contains the original handler context information.
	HandlerContext *EventHandlerContext `json:"handler_context"`

	// Error is the error that caused the event to fail.
	Error string `json:"error"`

	// ErrorType classifies the error type (retryable, permanent, etc.).
	ErrorType ErrorType `json:"error_type"`

	// FailureReason provides a human-readable reason for the failure.
	FailureReason string `json:"failure_reason"`

	// RetryCount indicates how many times this event has been retried.
	RetryCount int `json:"retry_count"`

	// MaxRetries indicates the maximum allowed retries for this event.
	MaxRetries int `json:"max_retries"`

	// FirstFailedAt is when the event first failed.
	FirstFailedAt time.Time `json:"first_failed_at"`

	// LastFailedAt is when the event most recently failed.
	LastFailedAt time.Time `json:"last_failed_at"`

	// NextRetryAt is when the next retry should be attempted.
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`

	// RecoveryAttempts is how many recovery attempts have been made.
	RecoveryAttempts int `json:"recovery_attempts"`

	// ExpiresAt is when this DLQ message expires and should be cleaned up.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Metadata contains additional information about the failure.
	Metadata map[string]interface{} `json:"metadata"`

	// SagaID is the ID of the saga this event belongs to.
	SagaID string `json:"saga_id"`

	// EventType is the type of the original event.
	EventType SagaEventType `json:"event_type"`

	// HandlerID is the ID of the handler that failed to process this event.
	HandlerID string `json:"handler_id"`
}

// ErrorType classifies errors for DLQ processing.
type ErrorType string

const (
	// ErrorTypeRetryable indicates the error is temporary and the event should be retried.
	ErrorTypeRetryable ErrorType = "retryable"

	// ErrorTypePermanent indicates the error is permanent and the event should not be retried.
	ErrorTypePermanent ErrorType = "permanent"

	// ErrorTypeTimeout indicates the error was caused by a timeout.
	ErrorTypeTimeout ErrorType = "timeout"

	// ErrorTypeRateLimit indicates the error was caused by rate limiting.
	ErrorTypeRateLimit ErrorType = "rate_limit"

	// ErrorTypeResourceLimit indicates the error was caused by resource constraints.
	ErrorTypeResourceLimit ErrorType = "resource_limit"

	// ErrorTypeValidation indicates the event failed validation.
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypeDeserialization indicates the event could not be deserialized.
	ErrorTypeDeserialization ErrorType = "deserialization"

	// ErrorTypeUnknown indicates an unknown error type.
	ErrorTypeUnknown ErrorType = "unknown"
)

// DLQMetrics contains metrics about DLQ operations.
type DLQMetrics struct {
	// TotalMessagesReceived is the total number of messages sent to DLQ.
	TotalMessagesReceived int64 `json:"total_messages_received"`

	// TotalMessagesRecovered is the total number of messages successfully recovered from DLQ.
	TotalMessagesRecovered int64 `json:"total_messages_recovered"`

	// TotalMessagesExpired is the total number of messages that expired in DLQ.
	TotalMessagesExpired int64 `json:"total_messages_expired"`

	// TotalMessagesCleanedUp is the total number of messages cleaned up from DLQ.
	TotalMessagesCleanedUp int64 `json:"total_messages_cleaned_up"`

	// CurrentQueueSize is the current number of messages in DLQ.
	CurrentQueueSize int64 `json:"current_queue_size"`

	// AverageRecoveryTime is the average time to recover a message from DLQ.
	AverageRecoveryTime time.Duration `json:"average_recovery_time"`

	// MessagesByErrorType tracks the number of messages by error type.
	MessagesByErrorType map[ErrorType]int64 `json:"messages_by_error_type"`

	// MessagesByEventType tracks the number of messages by event type.
	MessagesByEventType map[SagaEventType]int64 `json:"messages_by_event_type"`

	// RecoverySuccessRate is the percentage of messages successfully recovered.
	RecoverySuccessRate float64 `json:"recovery_success_rate"`

	// LastMessageReceivedAt is the timestamp when the last message was received.
	LastMessageReceivedAt *time.Time `json:"last_message_received_at"`

	// LastRecoveryAt is the timestamp when the last recovery was attempted.
	LastRecoveryAt *time.Time `json:"last_recovery_at"`

	// IsHealthy indicates if the DLQ handler is healthy.
	IsHealthy bool `json:"is_healthy"`

	// LastError is the last error encountered by the DLQ handler.
	LastError string `json:"last_error,omitempty"`

	// LastErrorAt is the timestamp of the last error.
	LastErrorAt *time.Time `json:"last_error_at,omitempty"`
}

// defaultDeadLetterQueueHandler is the default implementation of DeadLetterQueueHandler.
type defaultDeadLetterQueueHandler struct {
	// config holds the DLQ configuration.
	config *DeadLetterConfig

	// eventPublisher publishes events to the DLQ topic.
	eventPublisher saga.EventPublisher

	// messageSerializer serializes DLQ messages.
	messageSerializer DLQMessageSerializer

	// retryPolicy determines retry behavior for DLQ messages.
	retryPolicy DLQRetryPolicy

	// errorClassifier classifies errors for DLQ processing.
	errorClassifier DLQErrorClassifier

	// metrics tracks DLQ performance metrics.
	metrics *DLQMetrics

	// logger is the logger for DLQ operations.
	logger *zap.Logger

	// prometheusMetrics tracks Prometheus metrics.
	prometheusMetrics *DLQPrometheusMetrics

	// mu protects concurrent access to the handler.
	mu sync.RWMutex

	// recoveryCtx is the context for the recovery process.
	recoveryCtx context.Context

	// recoveryCancel cancels the recovery process.
	recoveryCancel context.CancelFunc

	// recoveryWG waits for the recovery process to finish.
	recoveryWG sync.WaitGroup

	// isRecoveryRunning indicates if recovery is currently running.
	isRecoveryRunning bool

	// started indicates if the handler has been started.
	started bool

	// shutdown indicates if the handler has been shut down.
	shutdown bool
}

// DLQMessageSerializer defines the interface for serializing/deserializing DLQ messages.
type DLQMessageSerializer interface {
	// SerializeDLQMessage serializes a DLQ message to bytes.
	SerializeDLQMessage(msg *DLQMessage) ([]byte, error)

	// DeserializeDLQMessage deserializes bytes to a DLQ message.
	DeserializeDLQMessage(data []byte) (*DLQMessage, error)
}

// DLQRetryPolicy defines the interface for DLQ retry policies.
type DLQRetryPolicy interface {
	// ShouldRetry determines if a DLQ message should be retried.
	ShouldRetry(msg *DLQMessage) bool

	// GetRetryDelay returns the delay before the next retry attempt.
	GetRetryDelay(msg *DLQMessage) time.Duration

	// IsExpired determines if a DLQ message has expired.
	IsExpired(msg *DLQMessage) bool

	// GetMaxRecoveryAttempts returns the maximum number of recovery attempts.
	GetMaxRecoveryAttempts() int
}

// DLQErrorClassifier classifies errors for DLQ processing.
type DLQErrorClassifier interface {
	// ClassifyError classifies an error into an ErrorType.
	ClassifyError(err error) ErrorType

	// IsRetryableError determines if an error is retryable.
	IsRetryableError(err error) bool

	// GetFailureReason returns a human-readable failure reason for an error.
	GetFailureReason(err error) string
}

// DLQOption is a functional option for configuring the DLQ handler.
type DLQOption func(*defaultDeadLetterQueueHandler) error

// NewDeadLetterQueueHandler creates a new dead-letter queue handler with the given configuration.
//
// Parameters:
//   - config: DLQ configuration
//   - eventPublisher: Event publisher for sending messages to DLQ
//   - opts: Variadic list of functional options
//
// Returns:
//   - DeadLetterQueueHandler: Configured DLQ handler instance
//   - error: Configuration or validation error
func NewDeadLetterQueueHandler(config *DeadLetterConfig, eventPublisher saga.EventPublisher, opts ...DLQOption) (DeadLetterQueueHandler, error) {
	if config == nil {
		return nil, ErrDeadLetterQueueDisabled
	}
	if !config.Enabled {
		return nil, ErrDeadLetterQueueDisabled
	}
	if eventPublisher == nil {
		return nil, ErrDeadLetterPublishFailed
	}

	handler := &defaultDeadLetterQueueHandler{
		config:            config,
		eventPublisher:    eventPublisher,
		messageSerializer: NewDefaultDLQMessageSerializer(),
		retryPolicy:       NewDefaultDLQRetryPolicy(),
		errorClassifier:   NewDefaultDLQErrorClassifier(),
		metrics: &DLQMetrics{
			MessagesByErrorType: make(map[ErrorType]int64),
			MessagesByEventType: make(map[SagaEventType]int64),
			IsHealthy:           true,
		},
		logger: logger.GetLogger().Named("dlq"),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(handler); err != nil {
			return nil, err
		}
	}

	// Initialize Prometheus metrics
	registry := prometheus.NewRegistry()
	prometheusMetrics, err := NewDLQPrometheusMetrics("saga", "dlq", registry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Prometheus metrics: %w", err)
	}
	handler.prometheusMetrics = prometheusMetrics

	return handler, nil
}

// WithDLQMessageSerializer sets a custom DLQ message serializer.
func WithDLQMessageSerializer(serializer DLQMessageSerializer) DLQOption {
	return func(h *defaultDeadLetterQueueHandler) error {
		if serializer == nil {
			return fmt.Errorf("DLQ message serializer cannot be nil")
		}
		h.messageSerializer = serializer
		return nil
	}
}

// WithDLQRetryPolicy sets a custom DLQ retry policy.
func WithDLQRetryPolicy(policy DLQRetryPolicy) DLQOption {
	return func(h *defaultDeadLetterQueueHandler) error {
		if policy == nil {
			return fmt.Errorf("DLQ retry policy cannot be nil")
		}
		h.retryPolicy = policy
		return nil
	}
}

// WithDLQErrorClassifier sets a custom DLQ error classifier.
func WithDLQErrorClassifier(classifier DLQErrorClassifier) DLQOption {
	return func(h *defaultDeadLetterQueueHandler) error {
		if classifier == nil {
			return fmt.Errorf("DLQ error classifier cannot be nil")
		}
		h.errorClassifier = classifier
		return nil
	}
}

// SendToDeadLetterQueue implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) SendToDeadLetterQueue(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext, err error) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if !h.started {
		return ErrHandlerNotInitialized
	}

	// Validate inputs
	if event == nil {
		return ErrInvalidEvent
	}
	if handlerCtx == nil {
		return ErrInvalidContext
	}
	if err == nil {
		return fmt.Errorf("error cannot be nil for DLQ")
	}

	// Classify the error
	errorType := h.errorClassifier.ClassifyError(err)
	failureReason := h.errorClassifier.GetFailureReason(err)

	// Create DLQ message
	dlqMessage := &DLQMessage{
		ID:             h.generateDLQMessageID(event, handlerCtx),
		OriginalEvent:  event,
		HandlerContext: handlerCtx,
		Error:          err.Error(),
		ErrorType:      errorType,
		FailureReason:  failureReason,
		RetryCount:     handlerCtx.RetryCount,
		MaxRetries:     h.getMaxRetries(handlerCtx),
		FirstFailedAt:  time.Now(),
		LastFailedAt:   time.Now(),
		Metadata:       make(map[string]interface{}),
		SagaID:         event.SagaID,
		EventType:      SagaEventType(event.Type),
		HandlerID:      h.getHandlerID(handlerCtx),
	}

	// Set expiration time
	if h.config.MaxRetentionDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, h.config.MaxRetentionDays)
		dlqMessage.ExpiresAt = &expiresAt
	}

	// Add error details to metadata if configured
	if h.config.IncludeErrorDetails {
		dlqMessage.Metadata["error_stack"] = getStackTrace(err)
		dlqMessage.Metadata["error_type_detailed"] = fmt.Sprintf("%T", err)
	}

	// Add original message data if configured
	if h.config.IncludeOriginalMessage && len(handlerCtx.OriginalMessageData) > 0 {
		dlqMessage.Metadata["original_message_data"] = handlerCtx.OriginalMessageData
	}

	// Serialize the DLQ message
	messageData, err := h.messageSerializer.SerializeDLQMessage(dlqMessage)
	if err != nil {
		h.updateMetricsError(fmt.Errorf("failed to serialize DLQ message: %w", err))
		return fmt.Errorf("failed to serialize DLQ message: %w", err)
	}

	// Create DLQ event
	dlqEvent := &saga.SagaEvent{
		ID:        dlqMessage.ID,
		SagaID:    event.SagaID,
		Type:      saga.EventDeadLettered,
		Timestamp: time.Now(),
		Data:      messageData,
		Metadata: map[string]interface{}{
			"original_event_id":   event.ID,
			"original_event_type": event.Type,
			"dlq_message_id":      dlqMessage.ID,
			"error_type":          string(errorType),
			"failure_reason":      failureReason,
			"handler_id":          dlqMessage.HandlerID,
		},
	}

	// Publish to DLQ topic
	if err := h.eventPublisher.PublishEvent(ctx, dlqEvent); err != nil {
		h.updateMetricsError(fmt.Errorf("failed to publish to DLQ: %w", err))
		return fmt.Errorf("failed to publish to DLQ topic %s: %w", h.config.Topic, err)
	}

	// Update metrics
	h.updateMetricsSent(dlqMessage)

	h.logger.Info("message sent to dead-letter queue",
		zap.String("dlq_message_id", dlqMessage.ID),
		zap.String("original_event_id", event.ID),
		zap.String("saga_id", event.SagaID),
		zap.String("error_type", string(errorType)),
		zap.String("failure_reason", failureReason),
		zap.Int("retry_count", handlerCtx.RetryCount),
	)

	return nil
}

// RecoverFromDeadLetterQueue implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) RecoverFromDeadLetterQueue(ctx context.Context, dlqMessage *DLQMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if !h.started {
		return ErrHandlerNotInitialized
	}

	if dlqMessage == nil {
		return ErrInvalidEvent
	}

	// Check if message should be retried
	if !h.retryPolicy.ShouldRetry(dlqMessage) {
		h.logger.Info("DLQ message should not be retried",
			zap.String("dlq_message_id", dlqMessage.ID),
			zap.String("reason", "retry_policy"),
		)
		return fmt.Errorf("message should not be retried: %s", dlqMessage.FailureReason)
	}

	// Check if message has expired
	if h.retryPolicy.IsExpired(dlqMessage) {
		h.logger.Info("DLQ message has expired",
			zap.String("dlq_message_id", dlqMessage.ID),
			zap.Time("expires_at", *dlqMessage.ExpiresAt),
		)
		h.updateMetricsExpired()
		return fmt.Errorf("message has expired: %s", dlqMessage.FailureReason)
	}

	// Update recovery attempt count
	dlqMessage.RecoveryAttempts++
	dlqMessage.LastFailedAt = time.Now()

	// Calculate next retry time
	delay := h.retryPolicy.GetRetryDelay(dlqMessage)
	nextRetryAt := time.Now().Add(delay)
	dlqMessage.NextRetryAt = &nextRetryAt

	startTime := time.Now()

	// Attempt to reprocess the original event
	// In a real implementation, this would involve sending the event back to the original handler
	// For now, we'll simulate the recovery process
	err := h.reprocessEvent(ctx, dlqMessage)
	recoveryTime := time.Since(startTime)

	if err != nil {
		h.logger.Warn("DLQ message recovery failed",
			zap.String("dlq_message_id", dlqMessage.ID),
			zap.Int("recovery_attempt", dlqMessage.RecoveryAttempts),
			zap.Duration("recovery_time", recoveryTime),
			zap.Error(err),
		)

		h.updateMetricsRecoveryFailed(dlqMessage, recoveryTime)

		// Check if we should try again
		if dlqMessage.RecoveryAttempts < h.retryPolicy.GetMaxRecoveryAttempts() {
			return fmt.Errorf("recovery failed, will retry: %w", err)
		}

		return fmt.Errorf("recovery failed, max attempts exceeded: %w", err)
	}

	// Recovery successful
	h.logger.Info("DLQ message recovered successfully",
		zap.String("dlq_message_id", dlqMessage.ID),
		zap.Int("recovery_attempt", dlqMessage.RecoveryAttempts),
		zap.Duration("recovery_time", recoveryTime),
	)

	h.updateMetricsRecovered(dlqMessage, recoveryTime)

	return nil
}

// GetDLQMetrics implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) GetDLQMetrics() *DLQMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := *h.metrics

	// Deep copy maps
	metricsCopy.MessagesByErrorType = make(map[ErrorType]int64)
	for k, v := range h.metrics.MessagesByErrorType {
		metricsCopy.MessagesByErrorType[k] = v
	}

	metricsCopy.MessagesByEventType = make(map[SagaEventType]int64)
	for k, v := range h.metrics.MessagesByEventType {
		metricsCopy.MessagesByEventType[k] = v
	}

	return &metricsCopy
}

// GetDLQConfiguration implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) GetDLQConfiguration() *DeadLetterConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config
}

// UpdateDLQConfiguration implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) UpdateDLQConfiguration(config *DeadLetterConfig) error {
	if config == nil {
		return fmt.Errorf("DLQ configuration cannot be nil")
	}

	if !config.Enabled {
		return ErrDeadLetterQueueDisabled
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	h.config = config

	h.logger.Info("DLQ configuration updated")
	return nil
}

// StartDLQRecovery implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) StartDLQRecovery(ctx context.Context, recoveryInterval time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if h.isRecoveryRunning {
		return fmt.Errorf("recovery is already running")
	}

	// Validate recovery interval to prevent panic in time.NewTicker
	if recoveryInterval <= 0 {
		return fmt.Errorf("recovery interval must be positive, got %v", recoveryInterval)
	}

	h.recoveryCtx, h.recoveryCancel = context.WithCancel(ctx)
	h.isRecoveryRunning = true

	h.recoveryWG.Add(1)
	go h.recoveryWorker(h.recoveryCtx, recoveryInterval)

	h.logger.Info("DLQ recovery process started",
		zap.Duration("recovery_interval", recoveryInterval),
	)

	return nil
}

// StopDLQRecovery implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) StopDLQRecovery() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isRecoveryRunning {
		return
	}

	if h.recoveryCancel != nil {
		h.recoveryCancel()
	}

	h.recoveryWG.Wait()
	h.isRecoveryRunning = false

	h.logger.Info("DLQ recovery process stopped")
}

// CleanupExpiredMessages implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) CleanupExpiredMessages(ctx context.Context) (int, error) {
	// In a real implementation, this would query the DLQ storage and remove expired messages
	// For now, we'll simulate the cleanup process
	h.logger.Info("cleaning up expired DLQ messages")

	// This would involve:
	// 1. Querying DLQ for expired messages
	// 2. Removing them from storage
	// 3. Updating metrics

	cleanedUp := 0 // Placeholder

	h.logger.Info("expired DLQ messages cleaned up",
		zap.Int("cleaned_up", cleanedUp),
	)

	return cleanedUp, nil
}

// HealthCheck implements DeadLetterQueueHandler interface.
func (h *defaultDeadLetterQueueHandler) HealthCheck(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if !h.started {
		return ErrHandlerNotInitialized
	}

	if !h.metrics.IsHealthy {
		return fmt.Errorf("DLQ handler is unhealthy: %s", h.metrics.LastError)
	}

	return nil
}

// Start starts the DLQ handler.
func (h *defaultDeadLetterQueueHandler) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if h.started {
		return ErrHandlerAlreadyInitialized
	}

	h.started = true
	h.metrics.IsHealthy = true

	h.logger.Info("DLQ handler started")
	return nil
}

// Stop stops the DLQ handler.
func (h *defaultDeadLetterQueueHandler) Stop(ctx context.Context) error {
	h.mu.Lock()
	if h.shutdown {
		h.mu.Unlock()
		return ErrHandlerShutdown
	}
	if !h.started {
		h.mu.Unlock()
		return ErrHandlerNotInitialized
	}

	h.shutdown = true
	h.mu.Unlock()

	// Stop recovery process
	h.StopDLQRecovery()

	h.logger.Info("DLQ handler stopped")
	return nil
}

// Helper methods

func (h *defaultDeadLetterQueueHandler) generateDLQMessageID(event *saga.SagaEvent, handlerCtx *EventHandlerContext) string {
	return fmt.Sprintf("dlq_%s_%s_%d", event.SagaID, event.ID, time.Now().UnixNano())
}

func (h *defaultDeadLetterQueueHandler) getMaxRetries(handlerCtx *EventHandlerContext) int {
	// Extract max retries from handler context or use default
	// In a real implementation, this might come from configuration
	return 3
}

func (h *defaultDeadLetterQueueHandler) getHandlerID(handlerCtx *EventHandlerContext) string {
	// Extract handler ID from context or return default
	if handlerCtx.Metadata != nil {
		if handlerID, ok := handlerCtx.Metadata["handler_id"].(string); ok {
			return handlerID
		}
	}
	return "unknown"
}

func (h *defaultDeadLetterQueueHandler) reprocessEvent(ctx context.Context, dlqMessage *DLQMessage) error {
	// In a real implementation, this would:
	// 1. Deserialize the original event
	// 2. Send it back to the appropriate handler
	// 3. Wait for processing to complete
	// 4. Return the result

	// For now, we'll simulate successful reprocessing
	time.Sleep(100 * time.Millisecond) // Simulate processing time
	return nil
}

func (h *defaultDeadLetterQueueHandler) updateMetricsSent(dlqMessage *DLQMessage) {
	h.metrics.TotalMessagesReceived++
	h.metrics.CurrentQueueSize++
	h.metrics.MessagesByErrorType[dlqMessage.ErrorType]++
	h.metrics.MessagesByEventType[dlqMessage.EventType]++

	now := time.Now()
	h.metrics.LastMessageReceivedAt = &now

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordMessageSent(string(dlqMessage.ErrorType), string(dlqMessage.EventType))
	}
}

func (h *defaultDeadLetterQueueHandler) updateMetricsRecovered(dlqMessage *DLQMessage, recoveryTime time.Duration) {
	h.metrics.TotalMessagesRecovered++
	h.metrics.CurrentQueueSize--

	// Update average recovery time
	totalRecovered := h.metrics.TotalMessagesRecovered
	currentAvg := h.metrics.AverageRecoveryTime
	h.metrics.AverageRecoveryTime = (currentAvg*time.Duration(totalRecovered-1) + recoveryTime) / time.Duration(totalRecovered)

	// Update success rate
	if h.metrics.TotalMessagesReceived > 0 {
		h.metrics.RecoverySuccessRate = float64(h.metrics.TotalMessagesRecovered) / float64(h.metrics.TotalMessagesReceived)
	}

	now := time.Now()
	h.metrics.LastRecoveryAt = &now

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordMessageRecovered(recoveryTime)
	}
}

func (h *defaultDeadLetterQueueHandler) updateMetricsExpired() {
	h.metrics.TotalMessagesExpired++
	h.metrics.CurrentQueueSize--
}

func (h *defaultDeadLetterQueueHandler) updateMetricsRecoveryFailed(dlqMessage *DLQMessage, recoveryTime time.Duration) {
	// Update recovery failure metrics
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordRecoveryFailed(string(dlqMessage.ErrorType))
	}
}

func (h *defaultDeadLetterQueueHandler) updateMetricsError(err error) {
	h.metrics.IsHealthy = false
	h.metrics.LastError = err.Error()
	now := time.Now()
	h.metrics.LastErrorAt = &now

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordError()
	}
}

func (h *defaultDeadLetterQueueHandler) recoveryWorker(ctx context.Context, recoveryInterval time.Duration) {
	defer h.recoveryWG.Done()

	ticker := time.NewTicker(recoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("DLQ recovery worker stopping due to context cancellation")
			return
		case <-ticker.C:
			h.performRecoveryCycle(ctx)
		}
	}
}

func (h *defaultDeadLetterQueueHandler) performRecoveryCycle(ctx context.Context) {
	// In a real implementation, this would:
	// 1. Query for messages ready for recovery
	// 2. Attempt to recover them
	// 3. Update metrics
	// 4. Handle failures appropriately

	h.logger.Debug("performing DLQ recovery cycle")

	// This is a placeholder for the actual recovery logic
	// In a real implementation, you would query the DLQ storage and process messages
}

// Helper functions

func getStackTrace(err error) string {
	// In a real implementation, this would capture the actual stack trace
	// For now, return a placeholder
	return fmt.Sprintf("Stack trace for error: %v", err)
}
