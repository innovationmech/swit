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

	"go.uber.org/zap"
)

// AuditEventType defines types of audit events for messaging operations
type AuditEventType string

const (
	// AuditEventMessageCreated represents message creation events
	AuditEventMessageCreated AuditEventType = "message_created"
	// AuditEventMessagePublished represents message publish events
	AuditEventMessagePublished AuditEventType = "message_published"
	// AuditEventMessageReceived represents message receive events
	AuditEventMessageReceived AuditEventType = "message_received"
	// AuditEventMessageProcessed represents message processing events
	AuditEventMessageProcessed AuditEventType = "message_processed"
	// AuditEventMessageFailed represents message processing failure events
	AuditEventMessageFailed AuditEventType = "message_failed"
	// AuditEventBrokerConnected represents broker connection events
	AuditEventBrokerConnected AuditEventType = "broker_connected"
	// AuditEventBrokerDisconnected represents broker disconnection events
	AuditEventBrokerDisconnected AuditEventType = "broker_disconnected"
	// AuditEventPublisherCreated represents publisher creation events
	AuditEventPublisherCreated AuditEventType = "publisher_created"
	// AuditEventSubscriberCreated represents subscriber creation events
	AuditEventSubscriberCreated AuditEventType = "subscriber_created"
	// AuditEventSubscriptionStarted represents subscription start events
	AuditEventSubscriptionStarted AuditEventType = "subscription_started"
	// AuditEventSubscriptionStopped represents subscription stop events
	AuditEventSubscriptionStopped AuditEventType = "subscription_stopped"
	// AuditEventAuthSuccess represents authentication success events
	AuditEventAuthSuccess AuditEventType = "auth_success"
	// AuditEventAuthFailure represents authentication failure events
	AuditEventAuthFailure AuditEventType = "auth_failure"
	// AuditEventPolicyViolation represents security policy violation events
	AuditEventPolicyViolation AuditEventType = "policy_violation"
)

// AuditLevel defines the level of audit logging
type AuditLevel string

const (
	// AuditLevelMinimal logs only critical events
	AuditLevelMinimal AuditLevel = "minimal"
	// AuditLevelStandard logs most operational events
	AuditLevelStandard AuditLevel = "standard"
	// AuditLevelComprehensive logs all events including debug information
	AuditLevelComprehensive AuditLevel = "comprehensive"
)

// AuditConfig defines audit logging configuration for messaging operations
type MessagingAuditConfig struct {
	// Enabled indicates if audit logging is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// LogLevel defines the audit logging level
	LogLevel AuditLevel `json:"log_level" yaml:"log_level" default:"standard"`

	// Events to audit (empty means all events)
	Events []AuditEventType `json:"events" yaml:"events"`

	// IncludePayload indicates if message payload should be included in audit logs
	IncludePayload bool `json:"include_payload" yaml:"include_payload" default:"false"`

	// IncludeHeaders indicates if message headers should be included in audit logs
	IncludeHeaders bool `json:"include_headers" yaml:"include_headers" default:"true"`

	// IncludeAuthContext indicates if authentication context should be included
	IncludeAuthContext bool `json:"include_auth_context" yaml:"include_auth_context" default:"true"`

	// RetentionPeriod defines how long to keep audit logs
	RetentionPeriod time.Duration `json:"retention_period" yaml:"retention_period" default:"168h"`

	// SamplingRate for high-volume events (0.0-1.0)
	SamplingRate float64 `json:"sampling_rate" yaml:"sampling_rate" default:"1.0"`

	// BufferSize for async audit logging
	BufferSize int `json:"buffer_size" yaml:"buffer_size" default:"1000"`

	// FlushInterval for async audit log flushing
	FlushInterval time.Duration `json:"flush_interval" yaml:"flush_interval" default:"5s"`
}

// Validate validates the audit configuration
func (c *MessagingAuditConfig) Validate() error {
	if c.SamplingRate < 0.0 || c.SamplingRate > 1.0 {
		return fmt.Errorf("sampling_rate must be between 0.0 and 1.0")
	}

	if c.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be positive")
	}

	if c.FlushInterval <= 0 {
		return fmt.Errorf("flush_interval must be positive")
	}

	// Validate log level
	switch c.LogLevel {
	case AuditLevelMinimal, AuditLevelStandard, AuditLevelComprehensive:
		// Valid levels
	default:
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	return nil
}

// SetDefaults sets default values for the audit configuration
func (c *MessagingAuditConfig) SetDefaults() {
	c.Enabled = true
	c.LogLevel = AuditLevelStandard
	c.Events = []AuditEventType{} // Empty means all events
	c.IncludePayload = false
	c.IncludeHeaders = true
	c.IncludeAuthContext = true
	c.RetentionPeriod = 168 * time.Hour
	c.SamplingRate = 1.0
	c.BufferSize = 1000
	c.FlushInterval = 5 * time.Second
}

// AuditEvent represents a messaging audit event
type AuditEvent struct {
	// Type of audit event
	Type AuditEventType `json:"type"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Message ID associated with the event
	MessageID string `json:"message_id,omitempty"`

	// Topic associated with the event
	Topic string `json:"topic,omitempty"`

	// Broker type associated with the event
	BrokerType BrokerType `json:"broker_type,omitempty"`

	// Operation that triggered the event
	Operation string `json:"operation"`

	// Success indicates if the operation was successful
	Success bool `json:"success"`

	// Error if the operation failed
	Error error `json:"error,omitempty"`

	// Duration of the operation in milliseconds
	Duration float64 `json:"duration_ms,omitempty"`

	// Message size in bytes
	MessageSize int64 `json:"message_size,omitempty"`

	// Authentication context
	AuthContext *AuthContext `json:"auth_context,omitempty"`

	// Message headers (if included)
	Headers map[string]string `json:"headers,omitempty"`

	// Metadata about the event
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Correlation ID for tracing
	CorrelationID string `json:"correlation_id,omitempty"`

	// User ID from context
	UserID string `json:"user_id,omitempty"`
}

// AuditMetrics represents audit logging metrics
type AuditMetrics struct {
	// Total number of audit events logged
	TotalEvents int64 `json:"total_events"`

	// Number of events by type
	EventsByType map[AuditEventType]int64 `json:"events_by_type"`

	// Number of successful events
	SuccessfulEvents int64 `json:"successful_events"`

	// Number of failed events
	FailedEvents int64 `json:"failed_events"`

	// Average processing time in milliseconds
	AverageProcessingTime float64 `json:"average_processing_time_ms"`

	// Buffer utilization percentage
	BufferUtilization float64 `json:"buffer_utilization"`

	// Sampling rate applied
	AppliedSamplingRate float64 `json:"applied_sampling_rate"`

	// Timestamp when metrics were collected
	CollectedAt time.Time `json:"collected_at"`
}

// MessagingAuditor handles audit logging for messaging operations
type MessagingAuditor struct {
	config *MessagingAuditConfig
	logger *zap.Logger
	mu     sync.RWMutex

	// Metrics tracking
	metrics *AuditMetrics

	// Event channel for async processing
	eventChan chan *AuditEvent

	// Shutdown channel
	shutdownChan chan struct{}

	// Wait group for graceful shutdown
	wg sync.WaitGroup
}

// NewMessagingAuditor creates a new messaging auditor
func NewMessagingAuditor(config *MessagingAuditConfig) *MessagingAuditor {
	if config == nil {
		config = &MessagingAuditConfig{Enabled: false}
	}

	auditor := &MessagingAuditor{
		config:       config,
		logger:       zap.NewNop(),
		metrics:      &AuditMetrics{CollectedAt: time.Now(), EventsByType: make(map[AuditEventType]int64)},
		eventChan:    make(chan *AuditEvent, config.BufferSize),
		shutdownChan: make(chan struct{}),
	}

	if config.Enabled {
		auditor.startAsyncProcessor()
	}

	return auditor
}

// SetLogger sets the logger for the auditor
func (a *MessagingAuditor) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

// LogAuditEvent logs an audit event
func (a *MessagingAuditor) LogAuditEvent(ctx context.Context, event *AuditEvent) {
	if !a.config.Enabled {
		return
	}

	// Check if this event type should be logged based on configuration
	if !a.shouldLogEvent(event.Type) {
		return
	}

	// Apply sampling for high-volume events
	if !a.shouldSampleEvent(event.Type) {
		return
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Extract context information
	a.extractContextInfo(ctx, event)

	// Send to async processor
	select {
	case a.eventChan <- event:
		// Event queued successfully
	default:
		// Buffer full, log synchronously as fallback
		a.mu.Lock()
		a.logEventSynchronously(event)
		a.mu.Unlock()
	}
}

// shouldLogEvent determines if an event type should be logged based on configuration
func (a *MessagingAuditor) shouldLogEvent(eventType AuditEventType) bool {
	// If no specific events are configured, log all events
	if len(a.config.Events) == 0 {
		return true
	}

	// Check if this event type is in the configured list
	for _, configuredType := range a.config.Events {
		if configuredType == eventType {
			return true
		}
	}

	return false
}

// shouldSampleEvent determines if an event should be sampled
func (a *MessagingAuditor) shouldSampleEvent(eventType AuditEventType) bool {
	// For comprehensive logging, don't sample
	if a.config.LogLevel == AuditLevelComprehensive {
		return true
	}

	// For minimal logging, only sample critical events
	if a.config.LogLevel == AuditLevelMinimal {
		criticalEvents := []AuditEventType{
			AuditEventMessageFailed,
			AuditEventAuthFailure,
			AuditEventPolicyViolation,
		}

		for _, criticalEvent := range criticalEvents {
			if eventType == criticalEvent {
				return true
			}
		}
		return false
	}

	// Standard logging with sampling
	return a.config.SamplingRate >= 1.0 ||
		(a.config.SamplingRate > 0 && time.Now().UnixNano()%1000 < int64(a.config.SamplingRate*1000))
}

// extractContextInfo extracts correlation ID and user ID from context
func (a *MessagingAuditor) extractContextInfo(ctx context.Context, event *AuditEvent) {
	if ctx == nil {
		return
	}

	// Extract correlation ID
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			event.CorrelationID = id
		}
	}

	// Extract user ID
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			event.UserID = id
		}
	}

	// Extract auth context if enabled
	if a.config.IncludeAuthContext {
		if authCtx := ctx.Value("auth_context"); authCtx != nil {
			if ctx, ok := authCtx.(*AuthContext); ok {
				event.AuthContext = ctx
			}
		}
	}
}

// startAsyncProcessor starts the async event processor
func (a *MessagingAuditor) StartAsyncProcessor() {
	a.wg.Add(1)
	go a.processEvents()
}

func (a *MessagingAuditor) startAsyncProcessor() {
	a.StartAsyncProcessor()
}

// processEvents processes audit events asynchronously
func (a *MessagingAuditor) processEvents() {
	defer a.wg.Done()

	flushTicker := time.NewTicker(a.config.FlushInterval)
	defer flushTicker.Stop()

	buffer := make([]*AuditEvent, 0, a.config.BufferSize)

	for {
		select {
		case event := <-a.eventChan:
			buffer = append(buffer, event)

			// Flush if buffer is full
			if len(buffer) >= a.config.BufferSize {
				a.flushEvents(buffer)
				buffer = buffer[:0]
			}

		case <-flushTicker.C:
			// Periodic flush
			if len(buffer) > 0 {
				a.flushEvents(buffer)
				buffer = buffer[:0]
			}

		case <-a.shutdownChan:
			// Shutdown requested, flush remaining events
			if len(buffer) > 0 {
				a.flushEvents(buffer)
			}
			return
		}
	}
}

// flushEvents flushes a batch of events to the log
func (a *MessagingAuditor) flushEvents(events []*AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, event := range events {
		a.logEventSynchronously(event)
	}
}

// logEventSynchronously logs an event synchronously
func (a *MessagingAuditor) logEventSynchronously(event *AuditEvent) {
	// Update metrics
	a.updateMetrics(event)

	// Determine log level based on event type and success
	logFunc := a.logger.Info
	switch event.Type {
	case AuditEventMessageFailed, AuditEventAuthFailure, AuditEventPolicyViolation:
		logFunc = a.logger.Error
	case AuditEventBrokerDisconnected:
		if !event.Success {
			logFunc = a.logger.Warn
		}
	}

	// Create log fields
	fields := []zap.Field{
		zap.String("audit_event_type", string(event.Type)),
		zap.Time("timestamp", event.Timestamp),
		zap.String("operation", event.Operation),
		zap.Bool("success", event.Success),
	}

	if event.MessageID != "" {
		fields = append(fields, zap.String("message_id", event.MessageID))
	}

	if event.Topic != "" {
		fields = append(fields, zap.String("topic", event.Topic))
	}

	if event.BrokerType != "" {
		fields = append(fields, zap.String("broker_type", string(event.BrokerType)))
	}

	if event.Duration > 0 {
		fields = append(fields, zap.Float64("duration_ms", event.Duration))
	}

	if event.MessageSize > 0 {
		fields = append(fields, zap.Int64("message_size", event.MessageSize))
	}

	if event.CorrelationID != "" {
		fields = append(fields, zap.String("correlation_id", event.CorrelationID))
	}

	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}

	if event.Error != nil {
		fields = append(fields, zap.Error(event.Error))
	}

	if event.AuthContext != nil && a.config.IncludeAuthContext && event.AuthContext.Credentials != nil {
		fields = append(fields,
			zap.String("auth_user_id", event.AuthContext.Credentials.UserID),
			zap.Strings("auth_scopes", event.AuthContext.Credentials.Scopes),
		)
	}

	if event.Headers != nil && a.config.IncludeHeaders {
		// Add headers as individual fields
		for key, value := range event.Headers {
			fields = append(fields, zap.String("header_"+key, value))
		}
	}

	if event.Metadata != nil {
		// Add metadata as individual fields
		for key, value := range event.Metadata {
			switch v := value.(type) {
			case string:
				fields = append(fields, zap.String("meta_"+key, v))
			case int:
				fields = append(fields, zap.Int("meta_"+key, v))
			case int64:
				fields = append(fields, zap.Int64("meta_"+key, v))
			case float64:
				fields = append(fields, zap.Float64("meta_"+key, v))
			case bool:
				fields = append(fields, zap.Bool("meta_"+key, v))
			default:
				fields = append(fields, zap.String("meta_"+key, fmt.Sprintf("%v", v)))
			}
		}
	}

	logFunc("Messaging audit event", fields...)
}

// updateMetrics updates the audit metrics
func (a *MessagingAuditor) updateMetrics(event *AuditEvent) {
	a.metrics.TotalEvents++
	a.metrics.EventsByType[event.Type]++

	if event.Success {
		a.metrics.SuccessfulEvents++
	} else {
		a.metrics.FailedEvents++
	}

	if event.Duration > 0 {
		// Simple moving average for processing time
		if a.metrics.AverageProcessingTime == 0 {
			a.metrics.AverageProcessingTime = event.Duration
		} else {
			a.metrics.AverageProcessingTime = (a.metrics.AverageProcessingTime*0.9 + event.Duration*0.1)
		}
	}

	// Update buffer utilization
	a.metrics.BufferUtilization = float64(len(a.eventChan)) / float64(cap(a.eventChan)) * 100.0
	a.metrics.AppliedSamplingRate = a.config.SamplingRate
	a.metrics.CollectedAt = time.Now()
}

// GetMetrics returns current audit metrics
func (a *MessagingAuditor) GetMetrics() *AuditMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to prevent external modification
	metrics := *a.metrics
	metrics.CollectedAt = time.Now()
	return &metrics
}

// Close gracefully shuts down the auditor
func (a *MessagingAuditor) Close() error {
	if !a.config.Enabled {
		return nil
	}

	// Signal shutdown
	close(a.shutdownChan)

	// Wait for processor to finish
	a.wg.Wait()

	return nil
}

// Convenience methods for common audit events

// LogMessageCreated logs a message creation event
func (a *MessagingAuditor) LogMessageCreated(ctx context.Context, messageID, topic string, size int64) {
	event := &AuditEvent{
		Type:        AuditEventMessageCreated,
		MessageID:   messageID,
		Topic:       topic,
		Operation:   "create_message",
		Success:     true,
		MessageSize: size,
	}
	a.LogAuditEvent(ctx, event)
}

// LogMessagePublished logs a message publish event
func (a *MessagingAuditor) LogMessagePublished(ctx context.Context, messageID, topic string, brokerType BrokerType, duration float64, success bool, err error) {
	event := &AuditEvent{
		Type:       AuditEventMessagePublished,
		MessageID:  messageID,
		Topic:      topic,
		BrokerType: brokerType,
		Operation:  "publish_message",
		Success:    success,
		Duration:   duration,
		Error:      err,
	}
	a.LogAuditEvent(ctx, event)
}

// LogMessageReceived logs a message receive event
func (a *MessagingAuditor) LogMessageReceived(ctx context.Context, messageID, topic string, brokerType BrokerType, size int64) {
	event := &AuditEvent{
		Type:        AuditEventMessageReceived,
		MessageID:   messageID,
		Topic:       topic,
		BrokerType:  brokerType,
		Operation:   "receive_message",
		Success:     true,
		MessageSize: size,
	}
	a.LogAuditEvent(ctx, event)
}

// LogMessageProcessed logs a message processing event
func (a *MessagingAuditor) LogMessageProcessed(ctx context.Context, messageID, topic string, duration float64, success bool, err error) {
	event := &AuditEvent{
		Type:      AuditEventMessageProcessed,
		MessageID: messageID,
		Topic:     topic,
		Operation: "process_message",
		Success:   success,
		Duration:  duration,
		Error:     err,
	}
	a.LogAuditEvent(ctx, event)
}

// LogMessageFailed logs a message processing failure event
func (a *MessagingAuditor) LogMessageFailed(ctx context.Context, messageID, topic string, duration float64, err error) {
	event := &AuditEvent{
		Type:      AuditEventMessageFailed,
		MessageID: messageID,
		Topic:     topic,
		Operation: "process_message",
		Success:   false,
		Duration:  duration,
		Error:     err,
	}
	a.LogAuditEvent(ctx, event)
}

// LogBrokerConnected logs a broker connection event
func (a *MessagingAuditor) LogBrokerConnected(ctx context.Context, brokerType BrokerType, success bool, err error) {
	event := &AuditEvent{
		Type:       AuditEventBrokerConnected,
		BrokerType: brokerType,
		Operation:  "connect_broker",
		Success:    success,
		Error:      err,
	}
	a.LogAuditEvent(ctx, event)
}

// LogBrokerDisconnected logs a broker disconnection event
func (a *MessagingAuditor) LogBrokerDisconnected(ctx context.Context, brokerType BrokerType) {
	event := &AuditEvent{
		Type:       AuditEventBrokerDisconnected,
		BrokerType: brokerType,
		Operation:  "disconnect_broker",
		Success:    true,
	}
	a.LogAuditEvent(ctx, event)
}

// LogSubscriptionStarted logs a subscription start event
func (a *MessagingAuditor) LogSubscriptionStarted(ctx context.Context, topic, consumerGroup string) {
	event := &AuditEvent{
		Type:      AuditEventSubscriptionStarted,
		Topic:     topic,
		Operation: "start_subscription",
		Success:   true,
		Metadata: map[string]interface{}{
			"consumer_group": consumerGroup,
		},
	}
	a.LogAuditEvent(ctx, event)
}

// LogSubscriptionStopped logs a subscription stop event
func (a *MessagingAuditor) LogSubscriptionStopped(ctx context.Context, topic, consumerGroup string) {
	event := &AuditEvent{
		Type:      AuditEventSubscriptionStopped,
		Topic:     topic,
		Operation: "stop_subscription",
		Success:   true,
		Metadata: map[string]interface{}{
			"consumer_group": consumerGroup,
		},
	}
	a.LogAuditEvent(ctx, event)
}

// LogAuthSuccess logs an authentication success event
func (a *MessagingAuditor) LogAuthSuccess(ctx context.Context, userID, method string) {
	event := &AuditEvent{
		Type:      AuditEventAuthSuccess,
		Operation: "authenticate",
		Success:   true,
		UserID:    userID,
		Metadata: map[string]interface{}{
			"auth_method": method,
		},
	}
	a.LogAuditEvent(ctx, event)
}

// LogAuthFailure logs an authentication failure event
func (a *MessagingAuditor) LogAuthFailure(ctx context.Context, userID, method string, err error) {
	event := &AuditEvent{
		Type:      AuditEventAuthFailure,
		Operation: "authenticate",
		Success:   false,
		UserID:    userID,
		Error:     err,
		Metadata: map[string]interface{}{
			"auth_method": method,
		},
	}
	a.LogAuditEvent(ctx, event)
}

// LogPolicyViolation logs a security policy violation event
func (a *MessagingAuditor) LogPolicyViolation(ctx context.Context, policyName, messageID string, violation string) {
	event := &AuditEvent{
		Type:      AuditEventPolicyViolation,
		MessageID: messageID,
		Operation: "policy_check",
		Success:   false,
		Metadata: map[string]interface{}{
			"policy_name": policyName,
			"violation":   violation,
		},
	}
	a.LogAuditEvent(ctx, event)
}
