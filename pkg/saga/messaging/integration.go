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

// Package messaging provides integration between Saga and pkg/messaging systems.
// It bridges the Saga event model with the generic messaging infrastructure,
// enabling Saga components to leverage various message brokers (NATS, RabbitMQ, Kafka).
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

// IntegrationConfig defines configuration for the MessagingIntegrationAdapter.
// It specifies how the adapter should connect to and interact with the message broker,
// including topic routing, consumer group settings, and publisher configuration.
type IntegrationConfig struct {
	// Topics is the list of message broker topics to subscribe to.
	// These topics will be monitored for Saga events.
	Topics []string `yaml:"topics" json:"topics"`

	// SubscriberGroup is the consumer group name for the subscriber.
	// This allows multiple instances to share message processing.
	SubscriberGroup string `yaml:"subscriber_group" json:"subscriber_group"`

	// Publisher contains the configuration for the SagaEventPublisher.
	Publisher *PublisherConfig `yaml:"publisher" json:"publisher"`

	// Handler contains the configuration for the SagaEventHandler.
	Handler *HandlerConfig `yaml:"handler" json:"handler"`

	// Concurrency specifies the maximum number of concurrent message processors.
	// Default is 1 for sequential processing.
	Concurrency int `yaml:"concurrency" json:"concurrency"`

	// MessageBufferSize is the size of the internal message buffer.
	// Default is 100.
	MessageBufferSize int `yaml:"message_buffer_size" json:"message_buffer_size"`

	// EnableMetrics enables metrics collection for the integration adapter.
	EnableMetrics bool `yaml:"enable_metrics" json:"enable_metrics"`

	// EnableTracing enables distributed tracing for message processing.
	EnableTracing bool `yaml:"enable_tracing" json:"enable_tracing"`

	// HealthCheckInterval specifies how often to perform health checks.
	// Default is 30 seconds.
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
}

// Validate validates the integration configuration.
func (c *IntegrationConfig) Validate() error {
	if len(c.Topics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}

	if c.SubscriberGroup == "" {
		return fmt.Errorf("subscriber_group is required")
	}

	if c.Publisher == nil {
		return fmt.Errorf("publisher config is required")
	}

	if err := c.Publisher.Validate(); err != nil {
		return fmt.Errorf("invalid publisher config: %w", err)
	}

	if c.Handler != nil {
		if err := c.Handler.Validate(); err != nil {
			return fmt.Errorf("invalid handler config: %w", err)
		}
	}

	return nil
}

// SetDefaults sets default values for unspecified configuration fields.
func (c *IntegrationConfig) SetDefaults() {
	if c.Concurrency <= 0 {
		c.Concurrency = 1
	}

	if c.MessageBufferSize <= 0 {
		c.MessageBufferSize = 100
	}

	if c.HealthCheckInterval <= 0 {
		c.HealthCheckInterval = 30 * time.Second
	}

	if c.Publisher != nil {
		c.Publisher.SetDefaults()
	}
}

// MessagingIntegrationAdapter bridges Saga components with pkg/messaging infrastructure.
// It provides bidirectional message transformation, subscription management, and
// lifecycle coordination between the Saga event model and the messaging layer.
//
// The adapter handles:
//   - Converting between saga.SagaEvent and messaging.Message formats
//   - Managing subscriptions to message broker topics
//   - Routing incoming messages to appropriate Saga event handlers
//   - Publishing Saga events through the messaging infrastructure
//   - Coordinating lifecycle (initialization, start, stop, shutdown)
//
// Thread Safety:
// The adapter is designed to be thread-safe. Multiple goroutines can safely
// call its methods concurrently.
type MessagingIntegrationAdapter struct {
	// broker is the underlying message broker connection
	broker messaging.MessageBroker

	// publisher is the Saga event publisher
	publisher *SagaEventPublisher

	// handler is the Saga event handler
	handler SagaEventHandler

	// subscriber is the messaging subscriber
	subscriber messaging.EventSubscriber

	// config is the integration configuration
	config *IntegrationConfig

	// logger is the logger instance
	logger *zap.Logger

	// isRunning indicates whether the adapter is running
	isRunning atomic.Bool

	// isInitialized indicates whether the adapter has been initialized
	isInitialized atomic.Bool

	// mu protects the adapter state
	mu sync.RWMutex

	// stopCh is used to signal shutdown
	stopCh chan struct{}

	// wg tracks active processing goroutines
	wg sync.WaitGroup

	// metrics tracks adapter metrics
	metrics *IntegrationMetrics

	// healthTicker for periodic health checks
	healthTicker *time.Ticker
}

// IntegrationMetrics contains metrics for the integration adapter.
type IntegrationMetrics struct {
	// TotalMessagesReceived is the total number of messages received
	TotalMessagesReceived atomic.Int64

	// TotalMessagesProcessed is the total number of messages successfully processed
	TotalMessagesProcessed atomic.Int64

	// TotalMessagesFailed is the total number of messages that failed processing
	TotalMessagesFailed atomic.Int64

	// TotalEventsPublished is the total number of Saga events published
	TotalEventsPublished atomic.Int64

	// TotalPublishFailed is the total number of publish failures
	TotalPublishFailed atomic.Int64

	// LastMessageReceivedAt is the timestamp of the last received message
	LastMessageReceivedAt atomic.Int64 // Unix timestamp in nanoseconds

	// LastHealthCheckAt is the timestamp of the last health check
	LastHealthCheckAt atomic.Int64 // Unix timestamp in nanoseconds

	// IsHealthy indicates whether the adapter is healthy
	IsHealthy atomic.Bool
}

// NewMessagingIntegrationAdapter creates a new integration adapter.
//
// Parameters:
//   - broker: The message broker instance to use
//   - config: Integration configuration
//
// Returns:
//   - *MessagingIntegrationAdapter: The configured adapter instance
//   - error: An error if creation fails
//
// Example usage:
//
//	broker, _ := messaging.NewNATSBroker(brokerConfig)
//	config := &IntegrationConfig{
//	    Topics:          []string{"saga.events.*"},
//	    SubscriberGroup: "saga-coordinator",
//	    Publisher:       publisherConfig,
//	}
//	adapter, err := NewMessagingIntegrationAdapter(broker, config)
//	if err != nil {
//	    return err
//	}
func NewMessagingIntegrationAdapter(
	broker messaging.MessageBroker,
	config *IntegrationConfig,
) (*MessagingIntegrationAdapter, error) {
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

	// Initialize logger
	log := logger.GetLogger().With(
		zap.String("component", "saga-messaging-integration"),
	)

	// Create metrics
	metrics := &IntegrationMetrics{}
	metrics.IsHealthy.Store(true)

	adapter := &MessagingIntegrationAdapter{
		broker:  broker,
		config:  config,
		logger:  log,
		stopCh:  make(chan struct{}),
		metrics: metrics,
	}

	log.Info("messaging integration adapter created",
		zap.Strings("topics", config.Topics),
		zap.String("subscriber_group", config.SubscriberGroup),
		zap.Int("concurrency", config.Concurrency),
	)

	return adapter, nil
}

// Initialize initializes the adapter and establishes connections.
// This method must be called before Start().
//
// The initialization process:
//  1. Creates the SagaEventPublisher using the broker
//  2. Creates the SagaEventHandler (if configured)
//  3. Creates the messaging subscriber
//  4. Verifies broker connectivity
//
// Parameters:
//   - ctx: Context for timeout control
//
// Returns:
//   - error: An error if initialization fails
func (a *MessagingIntegrationAdapter) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isInitialized.Load() {
		return fmt.Errorf("adapter already initialized")
	}

	a.logger.Info("initializing messaging integration adapter")

	// Check broker connectivity
	if !a.broker.IsConnected() {
		if err := a.broker.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to broker: %w", err)
		}
	}

	// Create SagaEventPublisher
	publisher, err := NewSagaEventPublisher(a.broker, a.config.Publisher)
	if err != nil {
		return fmt.Errorf("failed to create saga event publisher: %w", err)
	}
	a.publisher = publisher

	// Create SagaEventHandler if configured
	if a.config.Handler != nil {
		handler, err := NewSagaEventHandler(a.config.Handler)
		if err != nil {
			return fmt.Errorf("failed to create saga event handler: %w", err)
		}
		a.handler = handler
	}

	// Create subscriber configuration
	subscriberConfig := messaging.SubscriberConfig{
		Topics:        a.config.Topics,
		ConsumerGroup: a.config.SubscriberGroup,
		Concurrency:   a.config.Concurrency,
		PrefetchCount: a.config.MessageBufferSize,
	}

	// Create the messaging subscriber
	subscriber, err := a.broker.CreateSubscriber(subscriberConfig)
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}
	a.subscriber = subscriber

	a.isInitialized.Store(true)
	a.logger.Info("messaging integration adapter initialized successfully")

	return nil
}

// Start starts the adapter and begins processing messages.
// This method blocks until Stop is called or an error occurs.
//
// The start process:
//  1. Verifies the adapter is initialized
//  2. Starts the message handler (if configured)
//  3. Begins subscribing to configured topics
//  4. Starts health check monitoring
//
// Parameters:
//   - ctx: Context for cancellation control
//
// Returns:
//   - error: An error if startup fails
func (a *MessagingIntegrationAdapter) Start(ctx context.Context) error {
	if !a.isInitialized.Load() {
		return fmt.Errorf("adapter not initialized")
	}

	if a.isRunning.Load() {
		return fmt.Errorf("adapter already running")
	}

	a.logger.Info("starting messaging integration adapter")

	// Mark as running before starting subscription
	a.isRunning.Store(true)

	// Start health check monitoring
	a.startHealthCheckMonitoring()

	// Start subscription in a goroutine
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.subscribe(ctx); err != nil {
			a.logger.Error("subscription error", zap.Error(err))
			a.metrics.IsHealthy.Store(false)
		}
	}()

	a.logger.Info("messaging integration adapter started successfully")
	return nil
}

// Stop stops the adapter and waits for active processing to complete.
// This method blocks until all active message processing completes
// or the context times out.
//
// Parameters:
//   - ctx: Context for shutdown timeout control
//
// Returns:
//   - error: An error if shutdown fails
func (a *MessagingIntegrationAdapter) Stop(ctx context.Context) error {
	if !a.isRunning.Load() {
		return fmt.Errorf("adapter not running")
	}

	a.logger.Info("stopping messaging integration adapter")

	// Signal shutdown
	close(a.stopCh)
	a.isRunning.Store(false)

	// Stop health check monitoring
	a.stopHealthCheckMonitoring()

	// Unsubscribe from topics
	if a.subscriber != nil {
		if err := a.subscriber.Unsubscribe(ctx); err != nil {
			a.logger.Error("failed to unsubscribe", zap.Error(err))
		}
	}

	// Wait for active processing to complete with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("messaging integration adapter stopped successfully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
}

// Close closes the adapter and releases all resources.
// This method should be called after Stop().
//
// Returns:
//   - error: An error if cleanup fails
func (a *MessagingIntegrationAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isRunning.Load() {
		return fmt.Errorf("adapter still running, call Stop first")
	}

	a.logger.Info("closing messaging integration adapter")

	var errs []error

	// Close publisher
	if a.publisher != nil {
		if err := a.publisher.Close(); err != nil {
			a.logger.Error("failed to close publisher", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// Close subscriber
	if a.subscriber != nil {
		if err := a.subscriber.Close(); err != nil {
			a.logger.Error("failed to close subscriber", zap.Error(err))
			errs = append(errs, err)
		}
	}

	a.isInitialized.Store(false)

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	a.logger.Info("messaging integration adapter closed successfully")
	return nil
}

// GetPublisher returns the Saga event publisher.
//
// Returns:
//   - *SagaEventPublisher: The publisher instance
func (a *MessagingIntegrationAdapter) GetPublisher() *SagaEventPublisher {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.publisher
}

// GetHandler returns the Saga event handler.
//
// Returns:
//   - SagaEventHandler: The handler instance, may be nil if not configured
func (a *MessagingIntegrationAdapter) GetHandler() SagaEventHandler {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.handler
}

// GetMetrics returns the current adapter metrics.
//
// Returns:
//   - *IntegrationMetrics: Current metrics snapshot
func (a *MessagingIntegrationAdapter) GetMetrics() *IntegrationMetrics {
	return a.metrics
}

// IsRunning returns true if the adapter is currently running.
//
// Returns:
//   - bool: true if running, false otherwise
func (a *MessagingIntegrationAdapter) IsRunning() bool {
	return a.isRunning.Load()
}

// IsInitialized returns true if the adapter has been initialized.
//
// Returns:
//   - bool: true if initialized, false otherwise
func (a *MessagingIntegrationAdapter) IsInitialized() bool {
	return a.isInitialized.Load()
}

// HealthCheck performs a health check on the adapter and its dependencies.
//
// Parameters:
//   - ctx: Context for timeout control
//
// Returns:
//   - error: An error if health check fails, nil if healthy
func (a *MessagingIntegrationAdapter) HealthCheck(ctx context.Context) error {
	a.metrics.LastHealthCheckAt.Store(time.Now().UnixNano())

	// Check if adapter is initialized
	if !a.isInitialized.Load() {
		a.metrics.IsHealthy.Store(false)
		return fmt.Errorf("adapter not initialized")
	}

	// Check broker health
	if _, err := a.broker.HealthCheck(ctx); err != nil {
		a.metrics.IsHealthy.Store(false)
		return fmt.Errorf("broker health check failed: %w", err)
	}

	// Check handler health if configured
	if a.handler != nil {
		if err := a.handler.HealthCheck(ctx); err != nil {
			a.metrics.IsHealthy.Store(false)
			return fmt.Errorf("handler health check failed: %w", err)
		}
	}

	a.metrics.IsHealthy.Store(true)
	return nil
}

// subscribe starts consuming messages from the configured topics.
// This method runs in a goroutine and processes messages until stopped.
func (a *MessagingIntegrationAdapter) subscribe(ctx context.Context) error {
	// Create message handler adapter
	messageHandler := &messageHandlerAdapter{
		adapter: a,
	}

	// Start subscription
	if err := a.subscriber.Subscribe(ctx, messageHandler); err != nil {
		return fmt.Errorf("subscription failed: %w", err)
	}

	return nil
}

// messageHandlerAdapter adapts the integration adapter to the messaging.MessageHandler interface.
type messageHandlerAdapter struct {
	adapter *MessagingIntegrationAdapter
}

// Handle implements messaging.MessageHandler.
func (m *messageHandlerAdapter) Handle(ctx context.Context, msg *messaging.Message) error {
	return m.adapter.handleMessage(ctx, msg)
}

// OnError implements messaging.MessageHandler.
func (m *messageHandlerAdapter) OnError(ctx context.Context, msg *messaging.Message, err error) messaging.ErrorAction {
	// Log the error
	m.adapter.logger.Error("message handling error",
		zap.String("message_id", msg.ID),
		zap.Error(err),
	)

	// Default to retry for transient errors
	return messaging.ErrorActionRetry
}

// handleMessage processes an incoming message from the broker.
// It converts the messaging.Message to a saga.SagaEvent and routes it
// to the appropriate handler.
func (a *MessagingIntegrationAdapter) handleMessage(ctx context.Context, msg *messaging.Message) error {
	a.metrics.TotalMessagesReceived.Add(1)
	a.metrics.LastMessageReceivedAt.Store(time.Now().UnixNano())

	// Log message receipt
	a.logger.Debug("received message",
		zap.String("message_id", msg.ID),
		zap.String("topic", msg.Topic),
		zap.Time("timestamp", msg.Timestamp),
	)

	// Convert message to Saga event
	event, err := a.messageToSagaEvent(msg)
	if err != nil {
		a.metrics.TotalMessagesFailed.Add(1)
		a.logger.Error("failed to convert message to saga event",
			zap.String("message_id", msg.ID),
			zap.Error(err),
		)
		return fmt.Errorf("message conversion failed: %w", err)
	}

	// Create handler context
	handlerCtx := &EventHandlerContext{
		MessageID:           msg.ID,
		Topic:               msg.Topic,
		Timestamp:           msg.Timestamp,
		Headers:             msg.Headers,
		OriginalMessageData: msg.Payload,
		ConsumerGroup:       a.config.SubscriberGroup,
		Metadata:            make(map[string]interface{}),
	}

	// Handle the event
	if a.handler != nil {
		if err := a.handler.HandleSagaEvent(ctx, event, handlerCtx); err != nil {
			a.metrics.TotalMessagesFailed.Add(1)
			a.logger.Error("failed to handle saga event",
				zap.String("event_id", event.ID),
				zap.String("saga_id", event.SagaID),
				zap.Error(err),
			)
			return fmt.Errorf("event handling failed: %w", err)
		}
	}

	a.metrics.TotalMessagesProcessed.Add(1)
	a.logger.Debug("message processed successfully",
		zap.String("message_id", msg.ID),
		zap.String("event_id", event.ID),
		zap.String("saga_id", event.SagaID),
	)

	return nil
}

// messageToSagaEvent converts a messaging.Message to a saga.SagaEvent.
// It deserializes the message payload and reconstructs the Saga event
// with metadata from message headers.
func (a *MessagingIntegrationAdapter) messageToSagaEvent(msg *messaging.Message) (*saga.SagaEvent, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Create deserializer
	deserializer := NewDefaultMessageDeserializer()

	// Deserialize the event using headers
	event, err := deserializer.Deserialize(context.Background(), msg.Payload, msg.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize event: %w", err)
	}

	// Enrich event with metadata from headers
	if sagaID := msg.Headers["saga_id"]; sagaID != "" {
		event.SagaID = sagaID
	}
	if eventType := msg.Headers["event_type"]; eventType != "" {
		event.Type = saga.SagaEventType(eventType)
	}
	if stepID := msg.Headers["step_id"]; stepID != "" {
		event.StepID = stepID
	}
	if traceID := msg.Headers["trace_id"]; traceID != "" {
		event.TraceID = traceID
	}
	if spanID := msg.Headers["span_id"]; spanID != "" {
		event.SpanID = spanID
	}
	if source := msg.Headers["source"]; source != "" {
		event.Source = source
	}
	if service := msg.Headers["service"]; service != "" {
		event.Service = service
	}

	// Use message ID if event ID is not set
	if event.ID == "" {
		event.ID = msg.ID
	}

	// Use message timestamp if event timestamp is zero
	if event.Timestamp.IsZero() {
		event.Timestamp = msg.Timestamp
	}

	// Set correlation ID from message
	if event.CorrelationID == "" {
		event.CorrelationID = msg.CorrelationID
	}

	return event, nil
}

// sagaEventToMessage converts a saga.SagaEvent to a messaging.Message.
// This is used when publishing Saga events through the messaging infrastructure.
func (a *MessagingIntegrationAdapter) sagaEventToMessage(event *saga.SagaEvent) (*messaging.Message, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	// Use the publisher's serializer
	serializer := a.publisher.serializer
	data, err := serializer.Serialize(context.Background(), event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine topic from event type
	topic := fmt.Sprintf("%s.%s", a.config.Publisher.TopicPrefix, event.Type)

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
			"content_type": serializer.ContentType(),
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

	return msg, nil
}

// startHealthCheckMonitoring starts periodic health check monitoring.
func (a *MessagingIntegrationAdapter) startHealthCheckMonitoring() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.healthTicker != nil {
		return // Already started
	}

	a.healthTicker = time.NewTicker(a.config.HealthCheckInterval)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case <-a.healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := a.HealthCheck(ctx); err != nil {
					a.logger.Warn("health check failed", zap.Error(err))
				}
				cancel()
			case <-a.stopCh:
				return
			}
		}
	}()
}

// stopHealthCheckMonitoring stops periodic health check monitoring.
func (a *MessagingIntegrationAdapter) stopHealthCheckMonitoring() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.healthTicker != nil {
		a.healthTicker.Stop()
		a.healthTicker = nil
	}
}
