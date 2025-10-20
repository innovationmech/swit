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
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// PublisherConfig contains configuration for the SagaEventPublisher.
type PublisherConfig struct {
	// BrokerType specifies the message broker type (e.g., "nats", "rabbitmq", "kafka")
	BrokerType string `yaml:"broker_type" json:"broker_type"`

	// BrokerEndpoints contains the list of broker endpoints
	BrokerEndpoints []string `yaml:"broker_endpoints" json:"broker_endpoints"`

	// TopicPrefix is prepended to all topic names
	TopicPrefix string `yaml:"topic_prefix" json:"topic_prefix"`

	// SerializerType specifies the serialization format ("json" or "protobuf")
	SerializerType string `yaml:"serializer_type" json:"serializer_type"`

	// EnableConfirm enables publisher confirms for reliable delivery
	EnableConfirm bool `yaml:"enable_confirm" json:"enable_confirm"`

	// RetryAttempts specifies the maximum number of retry attempts
	RetryAttempts int `yaml:"retry_attempts" json:"retry_attempts"`

	// RetryInterval specifies the duration between retry attempts
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`

	// Timeout specifies the publish operation timeout
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// EnableMetrics enables metrics collection
	EnableMetrics bool `yaml:"enable_metrics" json:"enable_metrics"`

	// Reliability contains advanced reliability configuration
	Reliability *ReliabilityConfig `yaml:"reliability" json:"reliability"`
}

// Validate validates the publisher configuration.
func (c *PublisherConfig) Validate() error {
	if c.BrokerType == "" {
		return fmt.Errorf("broker_type is required")
	}

	if len(c.BrokerEndpoints) == 0 {
		return fmt.Errorf("broker_endpoints are required")
	}

	if c.SerializerType == "" {
		return fmt.Errorf("serializer_type is required")
	}

	if c.SerializerType != "json" && c.SerializerType != "protobuf" {
		return fmt.Errorf("serializer_type must be 'json' or 'protobuf'")
	}

	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts must be non-negative")
	}

	if c.RetryInterval < 0 {
		return fmt.Errorf("retry_interval must be non-negative")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// SetDefaults sets default values for unspecified configuration fields.
func (c *PublisherConfig) SetDefaults() {
	if c.TopicPrefix == "" {
		c.TopicPrefix = "saga"
	}

	if c.SerializerType == "" {
		c.SerializerType = "json"
	}

	if c.RetryAttempts == 0 {
		c.RetryAttempts = 3
	}

	if c.RetryInterval == 0 {
		c.RetryInterval = 1 * time.Second
	}

	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	if !c.EnableMetrics {
		c.EnableMetrics = true
	}

	// Initialize reliability config if not set
	if c.Reliability == nil {
		c.Reliability = &ReliabilityConfig{}
	}
	c.Reliability.SetDefaults()

	// Sync basic config with reliability config for backward compatibility
	if !c.Reliability.EnableConfirm && c.EnableConfirm {
		c.Reliability.EnableConfirm = c.EnableConfirm
	}
	if c.Reliability.MaxRetryAttempts == 0 && c.RetryAttempts > 0 {
		c.Reliability.MaxRetryAttempts = c.RetryAttempts
		c.Reliability.EnableRetry = true
	}
	if c.Reliability.PublishTimeout == 0 && c.Timeout > 0 {
		c.Reliability.PublishTimeout = c.Timeout
	}
}

// PublisherMetrics contains metrics for the saga event publisher.
type PublisherMetrics struct {
	// TotalPublished is the total number of events published
	TotalPublished atomic.Int64

	// TotalFailed is the total number of failed publish operations
	TotalFailed atomic.Int64

	// TotalRetries is the total number of retry attempts
	TotalRetries atomic.Int64

	// PublishDuration tracks the average publish duration
	PublishDuration atomic.Int64 // in nanoseconds

	// LastPublishedAt is the timestamp of the last published event
	LastPublishedAt atomic.Int64 // Unix timestamp in nanoseconds
}

// Reset resets all metrics to zero.
func (m *PublisherMetrics) Reset() {
	m.TotalPublished.Store(0)
	m.TotalFailed.Store(0)
	m.TotalRetries.Store(0)
	m.PublishDuration.Store(0)
	m.LastPublishedAt.Store(0)
}

// GetPublishRate returns the average publish rate (events per second).
func (m *PublisherMetrics) GetPublishRate() float64 {
	total := m.TotalPublished.Load()
	if total == 0 {
		return 0
	}

	lastPublishedAt := m.LastPublishedAt.Load()
	if lastPublishedAt == 0 {
		return 0
	}

	duration := time.Since(time.Unix(0, lastPublishedAt))
	if duration == 0 {
		return 0
	}

	return float64(total) / duration.Seconds()
}

// SagaEventPublisher publishes Saga events to a message broker.
type SagaEventPublisher struct {
	// publisher is the underlying messaging publisher
	publisher messaging.EventPublisher

	// serializer is the event serializer
	serializer SagaEventSerializer

	// config is the publisher configuration
	config *PublisherConfig

	// logger is the logger instance
	logger *zap.Logger

	// metrics tracks publisher metrics
	metrics *PublisherMetrics

	// reliabilityHandler handles reliability guarantees
	reliabilityHandler *ReliabilityHandler

	// mu protects the publisher state
	mu sync.RWMutex

	// closed indicates whether the publisher has been closed
	closed bool
}

// NewSagaEventPublisher creates a new saga event publisher.
//
// Parameters:
//   - broker: The message broker instance to use for publishing
//   - config: Publisher configuration
//
// Returns:
//   - *SagaEventPublisher: The configured publisher instance
//   - error: An error if creation fails
func NewSagaEventPublisher(
	broker messaging.MessageBroker,
	config *PublisherConfig,
) (*SagaEventPublisher, error) {
	if broker == nil {
		return nil, fmt.Errorf("broker cannot be nil")
	}

	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set defaults
	config.SetDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create serializer
	serializer, err := NewSagaEventSerializer(config.SerializerType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create serializer: %w", err)
	}

	// Create publisher config for the broker
	publisherConfig := messaging.PublisherConfig{
		Topic: fmt.Sprintf("%s.events", config.TopicPrefix),
		Confirmation: messaging.ConfirmationConfig{
			Required: config.EnableConfirm,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  config.RetryAttempts,
			InitialDelay: config.RetryInterval,
			MaxDelay:     config.RetryInterval * 10,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
		Timeout: messaging.TimeoutConfig{
			Publish: config.Timeout,
			Flush:   30 * time.Second,
			Close:   10 * time.Second,
		},
	}

	// Create the messaging publisher
	publisher, err := broker.CreatePublisher(publisherConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	// Initialize logger
	log := logger.GetLogger().With(
		zap.String("component", "saga-event-publisher"),
		zap.String("broker_type", config.BrokerType),
	)

	// Create metrics
	metrics := &PublisherMetrics{}

	// Create reliability handler if reliability config is provided
	var reliabilityHandler *ReliabilityHandler
	if config.Reliability != nil {
		reliabilityHandler, err = NewReliabilityHandler(config.Reliability)
		if err != nil {
			return nil, fmt.Errorf("failed to create reliability handler: %w", err)
		}
	}

	p := &SagaEventPublisher{
		publisher:          publisher,
		serializer:         serializer,
		config:             config,
		logger:             log,
		metrics:            metrics,
		reliabilityHandler: reliabilityHandler,
		closed:             false,
	}

	log.Info("saga event publisher created successfully",
		zap.String("serializer", config.SerializerType),
		zap.Bool("confirm_enabled", config.EnableConfirm),
		zap.Int("retry_attempts", config.RetryAttempts),
		zap.Bool("reliability_enabled", reliabilityHandler != nil),
	)

	return p, nil
}

// Close closes the publisher and releases all resources.
func (p *SagaEventPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.logger.Info("closing saga event publisher")

	// Close the underlying publisher
	if err := p.publisher.Close(); err != nil {
		p.logger.Error("failed to close underlying publisher", zap.Error(err))
		p.closed = true
		return fmt.Errorf("failed to close publisher: %w", err)
	}

	p.closed = true
	p.logger.Info("saga event publisher closed successfully")

	return nil
}

// IsClosed returns true if the publisher has been closed.
func (p *SagaEventPublisher) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// GetMetrics returns the current publisher metrics.
func (p *SagaEventPublisher) GetMetrics() *PublisherMetrics {
	return p.metrics
}

// GetConfig returns the publisher configuration.
func (p *SagaEventPublisher) GetConfig() *PublisherConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// GetReliabilityMetrics returns the reliability metrics if reliability handler is enabled.
func (p *SagaEventPublisher) GetReliabilityMetrics() *ReliabilityMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.reliabilityHandler != nil {
		return p.reliabilityHandler.GetMetrics()
	}
	return nil
}

// PublishSagaEvent publishes a single Saga event to the message broker.
// This is the public API for publishing Saga events.
//
// The method performs the following operations:
// 1. Validates the event structure
// 2. Serializes the event to the configured format (JSON or Protobuf)
// 3. Creates a message with appropriate headers and routing information
// 4. Publishes the message with retry logic and optional confirmation
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - event: The Saga event to publish
//
// Returns:
//   - error: An error if publishing fails after all retries, nil on success
//
// Example usage:
//
//	event := &saga.SagaEvent{
//	    ID:        "event-123",
//	    SagaID:    "saga-456",
//	    Type:      saga.EventSagaStarted,
//	    Timestamp: time.Now(),
//	    Version:   "1.0",
//	}
//	if err := publisher.PublishSagaEvent(ctx, event); err != nil {
//	    return fmt.Errorf("failed to publish saga event: %w", err)
//	}
func (p *SagaEventPublisher) PublishSagaEvent(
	ctx context.Context,
	event *saga.SagaEvent,
) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Validate event
	if err := p.validateEvent(event); err != nil {
		p.logger.Error("event validation failed",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.Error(err),
		)
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Serialize event
	data, err := p.serializer.Serialize(ctx, event)
	if err != nil {
		p.logger.Error("event serialization failed",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.Error(err),
		)
		return fmt.Errorf("event serialization failed: %w", err)
	}

	// Get topic for the event
	topic := p.getTopicForEvent(event)

	// Create message with headers
	msg := &messaging.Message{
		ID:            event.ID,
		Topic:         topic,
		Payload:       data,
		Timestamp:     event.Timestamp,
		CorrelationID: event.CorrelationID,
		Headers: map[string]string{
			"saga_id":      event.SagaID,
			"event_type":   string(event.Type),
			"timestamp":    event.Timestamp.Format(time.RFC3339),
			"version":      event.Version,
			"content_type": p.serializer.ContentType(),
		},
	}

	// Add optional headers
	if event.StepID != "" {
		msg.Headers["step_id"] = event.StepID
	}
	if event.TraceID != "" {
		msg.Headers["trace_id"] = event.TraceID
	}
	if event.SpanID != "" {
		msg.Headers["span_id"] = event.SpanID
	}
	if event.Source != "" {
		msg.Headers["source"] = event.Source
	}
	if event.Service != "" {
		msg.Headers["service"] = event.Service
	}

	// Track publish start time
	startTime := time.Now()

	// Use reliability handler if available, otherwise use legacy retry logic
	var publishErr error
	if p.reliabilityHandler != nil {
		publishErr = p.reliabilityHandler.PublishWithReliability(ctx, p.publisher, msg)
	} else {
		// Legacy retry logic for backward compatibility
		for attempt := 0; attempt <= p.config.RetryAttempts; attempt++ {
			if attempt > 0 {
				p.metrics.TotalRetries.Add(1)
				p.logger.Warn("retrying publish",
					zap.Int("attempt", attempt),
					zap.String("event_id", event.ID),
					zap.String("saga_id", event.SagaID),
				)

				// Wait before retry with exponential backoff
				backoffDuration := p.config.RetryInterval * time.Duration(attempt)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoffDuration):
				}
			}

			// Attempt to publish
			if p.config.EnableConfirm {
				_, publishErr = p.publisher.PublishWithConfirm(ctx, msg)
			} else {
				publishErr = p.publisher.Publish(ctx, msg)
			}

			if publishErr == nil {
				// Success
				break
			}

			p.logger.Warn("publish attempt failed",
				zap.Int("attempt", attempt),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.Error(publishErr),
			)
		}
	}

	// Update metrics
	duration := time.Since(startTime)
	p.metrics.PublishDuration.Store(duration.Nanoseconds())
	p.metrics.LastPublishedAt.Store(time.Now().UnixNano())

	if publishErr != nil {
		p.metrics.TotalFailed.Add(1)
		p.logger.Error("failed to publish event after retries",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.String("event_type", string(event.Type)),
			zap.Int("retry_attempts", p.config.RetryAttempts),
			zap.Error(publishErr),
		)
		return fmt.Errorf("failed to publish event after %d retries: %w", p.config.RetryAttempts, publishErr)
	}

	p.metrics.TotalPublished.Add(1)
	p.logger.Info("saga event published successfully",
		zap.String("event_id", event.ID),
		zap.String("saga_id", event.SagaID),
		zap.String("event_type", string(event.Type)),
		zap.String("topic", topic),
		zap.Duration("duration", duration),
	)

	return nil
}

// getTopicForEvent determines the routing topic for a Saga event based on its type.
// The topic follows the pattern: {prefix}.{event_type}
//
// For example, with prefix "saga" and event type "saga.started",
// the resulting topic would be "saga.saga.started"
//
// Parameters:
//   - event: The Saga event to route
//
// Returns:
//   - string: The topic name for routing the event
func (p *SagaEventPublisher) getTopicForEvent(event *saga.SagaEvent) string {
	return fmt.Sprintf("%s.%s", p.config.TopicPrefix, event.Type)
}

// validateEvent performs validation on a Saga event before publishing.
// It checks for required fields and ensures the event structure is valid.
//
// Parameters:
//   - event: The Saga event to validate
//
// Returns:
//   - error: An error if validation fails, nil if the event is valid
func (p *SagaEventPublisher) validateEvent(event *saga.SagaEvent) error {
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.SagaID == "" {
		return fmt.Errorf("saga ID is required")
	}
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if event.Version == "" {
		return fmt.Errorf("event version is required")
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}
	return nil
}

// PublishBatch publishes multiple Saga events in a single batch operation.
// This is more efficient than individual PublishSagaEvent calls for high-throughput scenarios.
//
// The method performs the following operations:
// 1. Validates all events in the batch
// 2. Serializes all events to the configured format (JSON or Protobuf)
// 3. Creates messages with appropriate headers and routing information
// 4. Publishes all messages in a single batch operation
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - events: Slice of Saga events to publish
//
// Returns:
//   - error: An error if any event validation or publishing fails, nil on success
//
// Example usage:
//
//	events := []*saga.SagaEvent{
//	    {ID: "event-1", SagaID: "saga-1", Type: saga.EventSagaStarted, Version: "1.0", Timestamp: time.Now()},
//	    {ID: "event-2", SagaID: "saga-1", Type: saga.EventSagaStepCompleted, Version: "1.0", Timestamp: time.Now()},
//	}
//	if err := publisher.PublishBatch(ctx, events); err != nil {
//	    return fmt.Errorf("failed to publish batch: %w", err)
//	}
func (p *SagaEventPublisher) PublishBatch(
	ctx context.Context,
	events []*saga.SagaEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Track batch start time
	startTime := time.Now()

	// 1. Pre-validate all events
	for i, event := range events {
		if event == nil {
			return fmt.Errorf("event %d is nil", i)
		}
		if err := p.validateEvent(event); err != nil {
			p.logger.Error("batch event validation failed",
				zap.Int("event_index", i),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.Error(err),
			)
			return fmt.Errorf("event %d validation failed: %w", i, err)
		}
	}

	// 2. Batch serialize all events
	messages := make([]*messaging.Message, 0, len(events))
	for i, event := range events {
		// Serialize event
		data, err := p.serializer.Serialize(ctx, event)
		if err != nil {
			p.logger.Error("batch event serialization failed",
				zap.Int("event_index", i),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.Error(err),
			)
			return fmt.Errorf("event %d serialization failed: %w", i, err)
		}

		// Get topic for the event
		topic := p.getTopicForEvent(event)

		// Create message with headers
		msg := &messaging.Message{
			ID:            event.ID,
			Topic:         topic,
			Payload:       data,
			Timestamp:     event.Timestamp,
			CorrelationID: event.CorrelationID,
			Headers: map[string]string{
				"saga_id":      event.SagaID,
				"event_type":   string(event.Type),
				"timestamp":    event.Timestamp.Format(time.RFC3339),
				"version":      event.Version,
				"content_type": p.serializer.ContentType(),
			},
		}

		// Add optional headers
		if event.StepID != "" {
			msg.Headers["step_id"] = event.StepID
		}
		if event.TraceID != "" {
			msg.Headers["trace_id"] = event.TraceID
		}
		if event.SpanID != "" {
			msg.Headers["span_id"] = event.SpanID
		}
		if event.Source != "" {
			msg.Headers["source"] = event.Source
		}
		if event.Service != "" {
			msg.Headers["service"] = event.Service
		}

		messages = append(messages, msg)
	}

	// 3. Publish batch with retry logic
	var publishErr error
	for attempt := 0; attempt <= p.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			p.metrics.TotalRetries.Add(int64(len(events)))
			p.logger.Warn("retrying batch publish",
				zap.Int("attempt", attempt),
				zap.Int("batch_size", len(events)),
			)

			// Wait before retry with exponential backoff
			backoffDuration := p.config.RetryInterval * time.Duration(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		// Attempt to publish batch
		publishErr = p.publisher.PublishBatch(ctx, messages)

		if publishErr == nil {
			// Success
			break
		}

		p.logger.Warn("batch publish attempt failed",
			zap.Int("attempt", attempt),
			zap.Int("batch_size", len(events)),
			zap.Error(publishErr),
		)
	}

	// Update metrics
	duration := time.Since(startTime)
	p.metrics.PublishDuration.Store(duration.Nanoseconds())
	p.metrics.LastPublishedAt.Store(time.Now().UnixNano())

	if publishErr != nil {
		p.metrics.TotalFailed.Add(int64(len(events)))
		p.logger.Error("failed to publish batch after retries",
			zap.Int("batch_size", len(events)),
			zap.Int("retry_attempts", p.config.RetryAttempts),
			zap.Error(publishErr),
		)
		return fmt.Errorf("failed to publish batch after %d retries: %w", p.config.RetryAttempts, publishErr)
	}

	p.metrics.TotalPublished.Add(int64(len(events)))
	p.logger.Info("saga event batch published successfully",
		zap.Int("batch_size", len(events)),
		zap.Duration("duration", duration),
	)

	return nil
}

// PublishBatchAsync publishes multiple Saga events asynchronously with callback notification.
// This method returns immediately after queuing the batch and calls the callback when the operation completes.
//
// Parameters:
//   - ctx: Context for cancellation control
//   - events: Slice of Saga events to publish
//   - callback: Function called when batch publishing completes (may be called on a different goroutine)
//
// Returns:
//   - error: An error if immediate validation fails or publisher is closed, nil if queued successfully
//
// Example usage:
//
//	events := []*saga.SagaEvent{...}
//	err := publisher.PublishBatchAsync(ctx, events, func(err error) {
//	    if err != nil {
//	        log.Printf("Batch publish failed: %v", err)
//	    } else {
//	        log.Printf("Batch published successfully")
//	    }
//	})
func (p *SagaEventPublisher) PublishBatchAsync(
	ctx context.Context,
	events []*saga.SagaEvent,
	callback func(error),
) error {
	if len(events) == 0 {
		if callback != nil {
			go callback(nil)
		}
		return nil
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Pre-validate all events synchronously
	for i, event := range events {
		if event == nil {
			return fmt.Errorf("event %d is nil", i)
		}
		if err := p.validateEvent(event); err != nil {
			return fmt.Errorf("event %d validation failed: %w", i, err)
		}
	}

	// Publish in background
	go func() {
		err := p.PublishBatch(ctx, events)
		if callback != nil {
			callback(err)
		}
	}()

	return nil
}

// PublishWithTransaction publishes multiple Saga events within a transaction.
// This ensures atomicity - either all events are published or none are.
//
// The method performs the following operations:
// 1. Checks if the broker supports transactions
// 2. Begins a new transaction
// 3. Serializes and publishes all events within the transaction
// 4. Commits the transaction on success or rolls back on failure
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - events: Slice of Saga events to publish transactionally
//
// Returns:
//   - error: An error if the broker doesn't support transactions, transaction fails, or any event processing fails
//
// Example usage:
//
//	events := []*saga.SagaEvent{
//	    {ID: "event-1", SagaID: "saga-1", Type: saga.EventSagaStarted, Version: "1.0", Timestamp: time.Now()},
//	    {ID: "event-2", SagaID: "saga-1", Type: saga.EventSagaStepCompleted, Version: "1.0", Timestamp: time.Now()},
//	}
//	if err := publisher.PublishWithTransaction(ctx, events); err != nil {
//	    return fmt.Errorf("failed to publish in transaction: %w", err)
//	}
func (p *SagaEventPublisher) PublishWithTransaction(
	ctx context.Context,
	events []*saga.SagaEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Check if broker supports transactions
	// Note: We access the broker through a method on the underlying publisher
	// Since EventPublisher interface doesn't expose GetBroker, we'll handle
	// the error from BeginTransaction directly
	// If transactions are not supported, BeginTransaction will return an error

	// Track transaction start time
	startTime := time.Now()

	// Pre-validate all events
	for i, event := range events {
		if event == nil {
			return fmt.Errorf("event %d is nil", i)
		}
		if err := p.validateEvent(event); err != nil {
			p.logger.Error("transaction event validation failed",
				zap.Int("event_index", i),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.Error(err),
			)
			return fmt.Errorf("event %d validation failed: %w", i, err)
		}
	}

	// Begin transaction
	tx, err := p.publisher.BeginTransaction(ctx)
	if err != nil {
		p.logger.Error("failed to begin transaction",
			zap.Int("event_count", len(events)),
			zap.Error(err),
		)
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Ensure rollback on failure
	defer func() {
		if tx != nil {
			// If we still have a transaction handle, something went wrong
			_ = tx.Rollback(ctx)
		}
	}()

	// Serialize and publish events in transaction
	for i, event := range events {
		// Serialize event
		data, err := p.serializer.Serialize(ctx, event)
		if err != nil {
			p.logger.Error("transaction event serialization failed",
				zap.Int("event_index", i),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.Error(err),
			)
			return fmt.Errorf("event %d serialization failed: %w", i, err)
		}

		// Get topic for the event
		topic := p.getTopicForEvent(event)

		// Create message with headers
		msg := &messaging.Message{
			ID:            event.ID,
			Topic:         topic,
			Payload:       data,
			Timestamp:     event.Timestamp,
			CorrelationID: event.CorrelationID,
			Headers: map[string]string{
				"saga_id":      event.SagaID,
				"event_type":   string(event.Type),
				"timestamp":    event.Timestamp.Format(time.RFC3339),
				"version":      event.Version,
				"content_type": p.serializer.ContentType(),
				"tx_id":        tx.GetID(),
			},
		}

		// Add optional headers
		if event.StepID != "" {
			msg.Headers["step_id"] = event.StepID
		}
		if event.TraceID != "" {
			msg.Headers["trace_id"] = event.TraceID
		}
		if event.SpanID != "" {
			msg.Headers["span_id"] = event.SpanID
		}
		if event.Source != "" {
			msg.Headers["source"] = event.Source
		}
		if event.Service != "" {
			msg.Headers["service"] = event.Service
		}

		// Publish in transaction
		if err := tx.Publish(ctx, msg); err != nil {
			p.logger.Error("failed to publish event in transaction",
				zap.Int("event_index", i),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.String("tx_id", tx.GetID()),
				zap.Error(err),
			)
			return fmt.Errorf("publish event %d in transaction: %w", i, err)
		}

		p.logger.Debug("event published in transaction",
			zap.Int("event_index", i),
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.String("tx_id", tx.GetID()),
		)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		p.logger.Error("failed to commit transaction",
			zap.Int("event_count", len(events)),
			zap.String("tx_id", tx.GetID()),
			zap.Error(err),
		)
		p.metrics.TotalFailed.Add(int64(len(events)))
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Clear transaction handle to prevent deferred rollback
	tx = nil

	// Update metrics
	duration := time.Since(startTime)
	p.metrics.PublishDuration.Store(duration.Nanoseconds())
	p.metrics.LastPublishedAt.Store(time.Now().UnixNano())
	p.metrics.TotalPublished.Add(int64(len(events)))

	p.logger.Info("saga events published in transaction successfully",
		zap.Int("event_count", len(events)),
		zap.Duration("duration", duration),
	)

	return nil
}

// WithTransaction executes a function within a transaction context.
// This provides a functional API for transactional operations with automatic
// commit/rollback handling.
//
// The method performs the following operations:
// 1. Checks if the broker supports transactions
// 2. Begins a new transaction
// 3. Executes the provided function with a transaction-aware publisher
// 4. Commits on success or rolls back on error
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fn: Function to execute within the transaction context
//
// Returns:
//   - error: An error if the broker doesn't support transactions, transaction fails, or the function returns an error
//
// Example usage:
//
//	err := publisher.WithTransaction(ctx, func(ctx context.Context, txPub *TransactionalEventPublisher) error {
//	    event1 := &saga.SagaEvent{...}
//	    if err := txPub.PublishEvent(ctx, event1); err != nil {
//	        return err
//	    }
//
//	    event2 := &saga.SagaEvent{...}
//	    if err := txPub.PublishEvent(ctx, event2); err != nil {
//	        return err
//	    }
//
//	    return nil
//	})
func (p *SagaEventPublisher) WithTransaction(
	ctx context.Context,
	fn func(ctx context.Context, txPub *TransactionalEventPublisher) error,
) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Track transaction start time
	startTime := time.Now()

	// Begin transaction
	tx, err := p.publisher.BeginTransaction(ctx)
	if err != nil {
		p.logger.Error("failed to begin transaction for WithTransaction",
			zap.Error(err),
		)
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Ensure rollback on failure
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Create transactional publisher wrapper
	txPub := &TransactionalEventPublisher{
		tx:         tx,
		serializer: p.serializer,
		config:     p.config,
		logger:     p.logger.With(zap.String("tx_id", tx.GetID())),
		metrics:    p.metrics,
	}

	// Execute the function
	if err := fn(ctx, txPub); err != nil {
		p.logger.Error("transaction function failed",
			zap.String("tx_id", tx.GetID()),
			zap.Error(err),
		)
		return fmt.Errorf("transaction function: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		p.logger.Error("failed to commit transaction in WithTransaction",
			zap.String("tx_id", tx.GetID()),
			zap.Error(err),
		)
		return fmt.Errorf("commit transaction: %w", err)
	}

	committed = true

	// Update metrics
	duration := time.Since(startTime)
	p.metrics.PublishDuration.Store(duration.Nanoseconds())
	p.metrics.LastPublishedAt.Store(time.Now().UnixNano())

	p.logger.Info("transaction completed successfully",
		zap.String("tx_id", tx.GetID()),
		zap.Duration("duration", duration),
	)

	return nil
}

// TransactionalEventPublisher provides a transaction-aware interface for publishing events.
// This wrapper ensures all operations are performed within the transaction context.
type TransactionalEventPublisher struct {
	tx         messaging.Transaction
	serializer SagaEventSerializer
	config     *PublisherConfig
	logger     *zap.Logger
	metrics    *PublisherMetrics
}

// PublishEvent publishes a single event within the transaction.
// The event will only be visible to consumers if the transaction is committed.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - event: The Saga event to publish
//
// Returns:
//   - error: An error if validation or publishing fails
func (t *TransactionalEventPublisher) PublishEvent(
	ctx context.Context,
	event *saga.SagaEvent,
) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Validate event
	if err := t.validateEvent(event); err != nil {
		t.logger.Error("transaction event validation failed",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.Error(err),
		)
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Serialize event
	data, err := t.serializer.Serialize(ctx, event)
	if err != nil {
		t.logger.Error("transaction event serialization failed",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.Error(err),
		)
		return fmt.Errorf("event serialization failed: %w", err)
	}

	// Get topic for the event
	topic := t.getTopicForEvent(event)

	// Create message with headers
	msg := &messaging.Message{
		ID:            event.ID,
		Topic:         topic,
		Payload:       data,
		Timestamp:     event.Timestamp,
		CorrelationID: event.CorrelationID,
		Headers: map[string]string{
			"saga_id":      event.SagaID,
			"event_type":   string(event.Type),
			"timestamp":    event.Timestamp.Format(time.RFC3339),
			"version":      event.Version,
			"content_type": t.serializer.ContentType(),
			"tx_id":        t.tx.GetID(),
		},
	}

	// Add optional headers
	if event.StepID != "" {
		msg.Headers["step_id"] = event.StepID
	}
	if event.TraceID != "" {
		msg.Headers["trace_id"] = event.TraceID
	}
	if event.SpanID != "" {
		msg.Headers["span_id"] = event.SpanID
	}
	if event.Source != "" {
		msg.Headers["source"] = event.Source
	}
	if event.Service != "" {
		msg.Headers["service"] = event.Service
	}

	// Publish in transaction
	if err := t.tx.Publish(ctx, msg); err != nil {
		t.logger.Error("failed to publish event in transaction",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.Error(err),
		)
		return fmt.Errorf("publish in transaction: %w", err)
	}

	t.metrics.TotalPublished.Add(1)
	t.logger.Debug("event published in transaction",
		zap.String("event_id", event.ID),
		zap.String("saga_id", event.SagaID),
	)

	return nil
}

// GetTransactionID returns the unique transaction identifier.
func (t *TransactionalEventPublisher) GetTransactionID() string {
	return t.tx.GetID()
}

// getTopicForEvent determines the routing topic for a Saga event.
func (t *TransactionalEventPublisher) getTopicForEvent(event *saga.SagaEvent) string {
	return fmt.Sprintf("%s.%s", t.config.TopicPrefix, event.Type)
}

// validateEvent validates a Saga event.
func (t *TransactionalEventPublisher) validateEvent(event *saga.SagaEvent) error {
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.SagaID == "" {
		return fmt.Errorf("saga ID is required")
	}
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if event.Version == "" {
		return fmt.Errorf("event version is required")
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}
	return nil
}

// publish is an internal helper that publishes an event with retry logic.
func (p *SagaEventPublisher) publish(ctx context.Context, event *saga.SagaEvent) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	// Validate event
	if err := p.serializer.Validate(ctx, event); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Serialize event
	data, err := p.serializer.Serialize(ctx, event)
	if err != nil {
		return fmt.Errorf("event serialization failed: %w", err)
	}

	// Create message
	msg := &messaging.Message{
		ID:            event.ID,
		Topic:         fmt.Sprintf("%s.events", p.config.TopicPrefix),
		Payload:       data,
		Timestamp:     event.Timestamp,
		CorrelationID: event.CorrelationID,
		Headers: map[string]string{
			"event-type":    string(event.Type),
			"saga-id":       event.SagaID,
			"content-type":  p.serializer.ContentType(),
			"serializer":    p.serializer.FormatName(),
			"event-version": event.Version,
		},
	}

	// Add trace information if available
	if event.TraceID != "" {
		msg.Headers["trace-id"] = event.TraceID
	}
	if event.SpanID != "" {
		msg.Headers["span-id"] = event.SpanID
	}

	// Track publish start time
	startTime := time.Now()

	// Publish with retry
	var publishErr error
	for attempt := 0; attempt <= p.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			p.metrics.TotalRetries.Add(1)
			p.logger.Warn("retrying publish",
				zap.Int("attempt", attempt),
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
			)

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.config.RetryInterval * time.Duration(attempt)):
			}
		}

		// Attempt to publish
		if p.config.EnableConfirm {
			_, publishErr = p.publisher.PublishWithConfirm(ctx, msg)
		} else {
			publishErr = p.publisher.Publish(ctx, msg)
		}

		if publishErr == nil {
			// Success
			break
		}

		p.logger.Warn("publish attempt failed",
			zap.Int("attempt", attempt),
			zap.String("event_id", event.ID),
			zap.Error(publishErr),
		)
	}

	// Update metrics
	duration := time.Since(startTime)
	p.metrics.PublishDuration.Store(duration.Nanoseconds())
	p.metrics.LastPublishedAt.Store(time.Now().UnixNano())

	if publishErr != nil {
		p.metrics.TotalFailed.Add(1)
		p.logger.Error("failed to publish event after retries",
			zap.String("event_id", event.ID),
			zap.String("saga_id", event.SagaID),
			zap.Error(publishErr),
		)
		return fmt.Errorf("failed to publish event: %w", publishErr)
	}

	p.metrics.TotalPublished.Add(1)
	p.logger.Debug("event published successfully",
		zap.String("event_id", event.ID),
		zap.String("saga_id", event.SagaID),
		zap.String("event_type", string(event.Type)),
		zap.Duration("duration", duration),
	)

	return nil
}
