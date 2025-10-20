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

	p := &SagaEventPublisher{
		publisher:  publisher,
		serializer: serializer,
		config:     config,
		logger:     log,
		metrics:    metrics,
		closed:     false,
	}

	log.Info("saga event publisher created successfully",
		zap.String("serializer", config.SerializerType),
		zap.Bool("confirm_enabled", config.EnableConfirm),
		zap.Int("retry_attempts", config.RetryAttempts),
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

	// Publish with retry logic
	var publishErr error
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
