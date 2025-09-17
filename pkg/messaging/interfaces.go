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
)

// MessageBroker defines the core interface for message broker operations.
// It provides connection management, factory methods for publishers and subscribers,
// and health monitoring capabilities. All methods support context-based cancellation
// and timeout handling.
//
// This interface follows the SWIT framework patterns for lifecycle management
// and extensibility, enabling different broker implementations (Kafka, NATS, RabbitMQ)
// while maintaining a consistent API.
//
// Context Handling:
// All methods that accept a context.Context parameter expect a valid, non-nil context.
// Passing a nil context will result in undefined behavior or panic. Use context.Background()
// or context.TODO() if you need a default context. Methods will respect context cancellation
// and timeouts, returning appropriate errors when the context is cancelled or times out.
//
// Example usage:
//
//	config := &BrokerConfig{Type: BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
//	broker, err := NewMessageBroker(config)
//	if err != nil {
//		return err
//	}
//	defer broker.Close()
//
//	ctx := context.Background()
//	if err := broker.Connect(ctx); err != nil {
//		return err
//	}
//
//	publisher, err := broker.CreatePublisher(PublisherConfig{Topic: "events"})
//	if err != nil {
//		return err
//	}
type MessageBroker interface {
	// Connect establishes connection to the message broker.
	// This method should be idempotent - calling it multiple times should not cause issues.
	// It blocks until the connection is established or the context is cancelled.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//
	// Returns:
	//   - error: ConnectionError if connection fails, nil on success
	Connect(ctx context.Context) error

	// Disconnect gracefully closes the broker connection.
	// This method ensures all pending operations are completed before closing.
	// It should be called before application shutdown to ensure clean resource cleanup.
	//
	// Parameters:
	//   - ctx: Context for timeout control during graceful disconnect
	//
	// Returns:
	//   - error: DisconnectionError if disconnect fails, nil on success
	Disconnect(ctx context.Context) error

	// Close immediately closes the broker connection and releases all resources.
	// Unlike Disconnect, this method does not wait for pending operations to complete.
	// This method implements io.Closer interface for consistency with Go patterns.
	//
	// Returns:
	//   - error: nil if successful, error if cleanup fails
	Close() error

	// IsConnected returns the current connection status.
	// This method is thread-safe and can be called concurrently.
	//
	// Returns:
	//   - bool: true if connected, false otherwise
	IsConnected() bool

	// CreatePublisher creates a new publisher instance with the specified configuration.
	// Each publisher maintains its own connection resources and can be configured
	// independently for different topics or publishing patterns.
	//
	// Parameters:
	//   - config: Publisher configuration including topic, batching, and retry settings
	//
	// Returns:
	//   - EventPublisher: Publisher instance for sending messages
	//   - error: ConfigurationError if config is invalid, BrokerError if creation fails
	CreatePublisher(config PublisherConfig) (EventPublisher, error)

	// CreateSubscriber creates a new subscriber instance with the specified configuration.
	// Each subscriber manages its own consumer group and processing pipeline.
	// Supports concurrent processing and middleware integration.
	//
	// Parameters:
	//   - config: Subscriber configuration including topics, consumer group, and processing settings
	//
	// Returns:
	//   - EventSubscriber: Subscriber instance for receiving messages
	//   - error: ConfigurationError if config is invalid, BrokerError if creation fails
	CreateSubscriber(config SubscriberConfig) (EventSubscriber, error)

	// HealthCheck performs a comprehensive health check on the broker connection.
	// This includes connectivity check, broker-specific health indicators,
	// and resource availability status.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - *HealthStatus: Detailed health information including status and metrics
	//   - error: HealthCheckError if health check fails
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// GetMetrics returns current broker metrics and performance indicators.
	// Metrics include connection stats, message throughput, error rates,
	// and broker-specific operational metrics.
	//
	// Returns:
	//   - *BrokerMetrics: Current metrics snapshot, never nil
	GetMetrics() *BrokerMetrics

	// GetCapabilities returns broker-specific capabilities and feature support.
	// This allows applications to adapt behavior based on broker capabilities
	// and enables feature detection for cross-broker compatibility.
	//
	// Returns:
	//   - *BrokerCapabilities: Capability information, never nil
	GetCapabilities() *BrokerCapabilities
}

// MessageBrokerFactory defines the interface for creating message broker instances.
// This factory pattern enables dependency injection and testability while providing
// a consistent creation interface across different broker implementations.
//
// The factory should validate configurations and return appropriate errors
// for invalid settings or missing dependencies.
type MessageBrokerFactory interface {
	// CreateBroker creates a new MessageBroker instance with the provided configuration.
	// The factory validates the configuration and initializes the broker but does not
	// establish the connection - that should be done via MessageBroker.Connect().
	//
	// Parameters:
	//   - config: Broker configuration including type, endpoints, auth, and settings
	//
	// Returns:
	//   - MessageBroker: Configured broker instance ready for connection
	//   - error: ConfigurationError for invalid config, BrokerError for initialization failures
	CreateBroker(config *BrokerConfig) (MessageBroker, error)

	// GetSupportedBrokerTypes returns a list of broker types supported by this factory.
	// This enables runtime discovery of available broker implementations.
	//
	// Returns:
	//   - []BrokerType: List of supported broker types
	GetSupportedBrokerTypes() []BrokerType

	// ValidateConfig validates a broker configuration without creating a broker instance.
	// This is useful for configuration validation in CLI tools or admin interfaces.
	//
	// Parameters:
	//   - config: Broker configuration to validate
	//
	// Returns:
	//   - error: ConfigurationError with details if invalid, nil if valid
	ValidateConfig(config *BrokerConfig) error
}

// EventPublisher defines the interface for publishing messages to the broker.
// It supports various publishing patterns including synchronous, asynchronous,
// batch, and transactional publishing. All methods are thread-safe unless
// otherwise specified.
//
// Context Handling:
// All methods expect a valid, non-nil context. Methods will respect context
// cancellation and timeouts, returning appropriate errors when cancelled.
type EventPublisher interface {
	// Publish sends a single message synchronously.
	// This method blocks until the message is accepted by the broker
	// or the context is cancelled.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - message: Message to publish
	//
	// Returns:
	//   - error: PublishError if publishing fails, nil on success
	Publish(ctx context.Context, message *Message) error

	// PublishBatch sends multiple messages in a single batch operation.
	// This is more efficient than individual Publish calls for high-throughput scenarios.
	// The batch operation is atomic where supported by the broker.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - messages: Slice of messages to publish
	//
	// Returns:
	//   - error: BatchPublishError if any message fails, nil if all succeed
	PublishBatch(ctx context.Context, messages []*Message) error

	// PublishWithConfirm sends a message and waits for broker confirmation.
	// This provides stronger delivery guarantees at the cost of increased latency.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - message: Message to publish
	//
	// Returns:
	//   - *PublishConfirmation: Confirmation details including message ID and metadata
	//   - error: PublishError if publishing fails
	PublishWithConfirm(ctx context.Context, message *Message) (*PublishConfirmation, error)

	// PublishAsync sends a message asynchronously with callback notification.
	// This method returns immediately and calls the callback when the operation completes.
	// The callback may be called on a different goroutine.
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - message: Message to publish
	//   - callback: Function called when publishing completes
	//
	// Returns:
	//   - error: PublishError if immediate validation fails, nil if queued successfully
	PublishAsync(ctx context.Context, message *Message, callback PublishCallback) error

	// BeginTransaction starts a new transaction for transactional publishing.
	// Only available if the broker supports transactions (check BrokerCapabilities).
	// The transaction must be committed or rolled back to release resources.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - Transaction: Transaction instance for publishing messages
	//   - error: TransactionError if transactions not supported or start fails
	BeginTransaction(ctx context.Context) (Transaction, error)

	// Flush ensures all pending asynchronous messages are sent.
	// This method blocks until all queued messages are processed
	// or the context is cancelled.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: FlushError if flush fails, nil on success
	Flush(ctx context.Context) error

	// Close closes the publisher and releases all associated resources.
	// Any pending messages will be lost unless Flush is called first.
	// This method implements io.Closer interface.
	//
	// Returns:
	//   - error: nil if successful, error if cleanup fails
	Close() error

	// GetMetrics returns current publisher metrics.
	// Includes message counts, error rates, latency statistics, and queue status.
	//
	// Returns:
	//   - *PublisherMetrics: Current metrics snapshot, never nil
	GetMetrics() *PublisherMetrics
}

// EventSubscriber defines the interface for consuming messages from the broker.
// It supports various consumption patterns including push/pull models,
// middleware integration, and consumer group management.
//
// Context Handling:
// All methods expect a valid, non-nil context. Long-running operations like
// Subscribe will monitor context cancellation to enable graceful shutdown.
type EventSubscriber interface {
	// Subscribe starts consuming messages with the given handler.
	// This method blocks and processes messages until the context is cancelled
	// or an unrecoverable error occurs.
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - handler: Handler for processing messages
	//
	// Returns:
	//   - error: SubscriptionError if subscription fails
	Subscribe(ctx context.Context, handler MessageHandler) error

	// SubscribeWithMiddleware starts consuming with a middleware chain.
	// Middleware is applied in the order provided, creating a processing pipeline
	// for cross-cutting concerns like logging, metrics, and error handling.
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - handler: Base handler for processing messages
	//   - middleware: Middleware chain for message processing
	//
	// Returns:
	//   - error: SubscriptionError if subscription fails
	SubscribeWithMiddleware(ctx context.Context, handler MessageHandler, middleware ...Middleware) error

	// Unsubscribe stops consuming messages and releases subscription resources.
	// This method ensures graceful shutdown of message processing.
	//
	// Parameters:
	//   - ctx: Context for timeout control during unsubscribe
	//
	// Returns:
	//   - error: UnsubscribeError if unsubscribe fails, nil on success
	Unsubscribe(ctx context.Context) error

	// Pause temporarily stops message consumption without releasing resources.
	// Messages may continue to be received by the broker but won't be processed.
	// Call Resume to restart processing.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: PauseError if pause fails, nil on success
	Pause(ctx context.Context) error

	// Resume resumes message consumption after a Pause operation.
	// Processing will continue where it left off based on offset management settings.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: ResumeError if resume fails, nil on success
	Resume(ctx context.Context) error

	// Seek seeks to a specific position in the message stream.
	// Only available if supported by the broker (check BrokerCapabilities).
	// This allows replaying messages from a specific point.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//   - position: Target position in the stream
	//
	// Returns:
	//   - error: SeekError if seek fails or not supported, nil on success
	Seek(ctx context.Context, position SeekPosition) error

	// GetLag returns the current consumer lag (messages behind the latest).
	// This metric helps monitor processing performance and detect issues.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - int64: Number of messages behind, -1 if not available
	//   - error: LagError if lag calculation fails
	GetLag(ctx context.Context) (int64, error)

	// Close closes the subscriber and releases all associated resources.
	// This will stop message processing and clean up connections.
	// This method implements io.Closer interface.
	//
	// Returns:
	//   - error: nil if successful, error if cleanup fails
	Close() error

	// GetMetrics returns current subscriber metrics.
	// Includes message counts, processing times, error rates, and lag information.
	//
	// Returns:
	//   - *SubscriberMetrics: Current metrics snapshot, never nil
	GetMetrics() *SubscriberMetrics
}

// MessageHandler defines the interface for handling messages.
// Implementations should be thread-safe if used with concurrent subscribers.
type MessageHandler interface {
	// Handle processes a single message.
	// This method should be idempotent where possible to handle retries gracefully.
	// Processing time should be monitored to avoid consumer group rebalancing.
	//
	// Parameters:
	//   - ctx: Context with processing deadline and cancellation
	//   - message: Message to process
	//
	// Returns:
	//   - error: ProcessingError if processing fails, nil on success
	Handle(ctx context.Context, message *Message) error

	// OnError handles processing errors and determines the recovery action.
	// This method is called when Handle returns an error and allows
	// the handler to specify how the error should be handled.
	//
	// Parameters:
	//   - ctx: Context for the failed processing attempt
	//   - message: Message that failed to process
	//   - err: Error returned by Handle method
	//
	// Returns:
	//   - ErrorAction: Action to take (retry, dead letter, discard, pause)
	OnError(ctx context.Context, message *Message, err error) ErrorAction
}

// MessageHandlerFunc is a function adapter that implements MessageHandler.
// It allows using functions as message handlers for simple use cases.
type MessageHandlerFunc func(ctx context.Context, message *Message) error

// Handle implements MessageHandler interface for MessageHandlerFunc.
func (f MessageHandlerFunc) Handle(ctx context.Context, message *Message) error {
	return f(ctx, message)
}

// OnError provides default error handling for MessageHandlerFunc.
// It returns ErrorActionRetry for retryable errors and ErrorActionDeadLetter for others.
func (f MessageHandlerFunc) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	// Prefer new unified error type when available
	if be := ToBaseMessagingError(err); be != nil {
		if !be.Retryable {
			return ErrorActionDeadLetter
		}
		return ErrorActionRetry
	}

	// Fallback to legacy behavior
	var msgErr *MessagingError
	if errors.As(err, &msgErr) && !msgErr.Retryable {
		return ErrorActionDeadLetter
	}
	return ErrorActionRetry
}

// Middleware defines the interface for message processing middleware.
// Middleware enables cross-cutting concerns like logging, metrics, tracing,
// validation, and error handling in a composable way.
type Middleware interface {
	// Name returns the middleware name for identification and debugging.
	//
	// Returns:
	//   - string: Unique name for this middleware
	Name() string

	// Wrap wraps a handler with middleware logic.
	// The middleware should call next.Handle() to continue the chain
	// and can perform pre/post processing around the call.
	//
	// Parameters:
	//   - next: Next handler in the middleware chain
	//
	// Returns:
	//   - MessageHandler: Wrapped handler with middleware logic
	Wrap(next MessageHandler) MessageHandler
}

// Transaction defines the interface for transactional message publishing.
// Transactions provide atomicity guarantees for message publishing operations.
// A transaction must be committed or rolled back to release resources.
type Transaction interface {
	// Publish publishes a message within the transaction.
	// Messages are not visible to consumers until the transaction is committed.
	// All messages in a transaction either succeed or fail together.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//   - message: Message to publish in the transaction
	//
	// Returns:
	//   - error: TransactionError if publish fails, nil if queued successfully
	Publish(ctx context.Context, message *Message) error

	// Commit commits the transaction, making all published messages visible.
	// This operation is atomic - either all messages are committed or none are.
	// After commit, the transaction cannot be used for further operations.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: TransactionError if commit fails, nil on success
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction, discarding all published messages.
	// This operation cleans up transaction resources and makes messages invisible.
	// After rollback, the transaction cannot be used for further operations.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: TransactionError if rollback fails, nil on success
	Rollback(ctx context.Context) error

	// GetID returns the unique transaction identifier.
	// This ID can be used for logging and debugging purposes.
	//
	// Returns:
	//   - string: Unique transaction identifier
	GetID() string
}

// MessageBrokerAdapter defines the adapter interface for pluggable broker implementations.
// This adapter pattern enables different broker implementations while maintaining
// consistent framework integration and providing a unified abstraction layer.
//
// The adapter pattern separates broker-specific implementation details from the
// core messaging framework, allowing for:
// - Hot-swappable broker implementations
// - Consistent configuration patterns
// - Unified error handling and monitoring
// - Framework-wide compatibility guarantees
//
// Adapters are responsible for:
// - Translating framework requests to broker-specific operations
// - Normalizing broker responses to framework expectations
// - Managing broker-specific connection lifecycles
// - Providing capability reporting for feature detection
type MessageBrokerAdapter interface {
	// GetAdapterInfo returns metadata about this adapter implementation.
	// This information is used for registration, monitoring, and debugging.
	//
	// Returns:
	//   - *BrokerAdapterInfo: Adapter metadata including name, version, and supported broker types
	GetAdapterInfo() *BrokerAdapterInfo

	// CreateBroker creates a broker instance from the adapter's broker type.
	// The adapter validates the configuration and creates an appropriately configured
	// broker instance. This method should not establish connections - that's handled
	// by the returned MessageBroker's Connect() method.
	//
	// Parameters:
	//   - config: Broker configuration with adapter-specific settings
	//
	// Returns:
	//   - MessageBroker: Configured broker instance ready for connection
	//   - error: ConfigurationError if config is invalid, AdapterError for creation failures
	CreateBroker(config *BrokerConfig) (MessageBroker, error)

	// ValidateConfiguration validates a broker configuration for this adapter.
	// This allows early validation without creating broker instances and provides
	// detailed error messages for configuration issues.
	//
	// Parameters:
	//   - config: Configuration to validate
	//
	// Returns:
	//   - *AdapterValidationResult: Validation result with errors, warnings, and suggestions
	ValidateConfiguration(config *BrokerConfig) *AdapterValidationResult

	// GetCapabilities returns the capabilities supported by this adapter.
	// Capabilities describe what features are available and any adapter-specific
	// limitations or extensions.
	//
	// Returns:
	//   - *BrokerCapabilities: Capability information for this adapter
	GetCapabilities() *BrokerCapabilities

	// GetDefaultConfiguration returns a default configuration template for this adapter.
	// This helps with configuration generation and provides sensible defaults
	// for new broker setups.
	//
	// Returns:
	//   - *BrokerConfig: Default configuration template with reasonable defaults
	GetDefaultConfiguration() *BrokerConfig

	// HealthCheck performs an adapter-level health check.
	// This validates that the adapter is functioning correctly and can create
	// broker instances. It does not check actual broker connectivity.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - *HealthStatus: Health status of the adapter itself
	//   - error: AdapterError if health check fails
	HealthCheck(ctx context.Context) (*HealthStatus, error)
}

// BrokerAdapterInfo contains metadata about a broker adapter implementation.
type BrokerAdapterInfo struct {
	// Name is the unique identifier for this adapter
	Name string `json:"name"`

	// Version is the adapter version for compatibility tracking
	Version string `json:"version"`

	// Description provides human-readable information about the adapter
	Description string `json:"description"`

	// SupportedBrokerTypes lists the broker types this adapter can create
	SupportedBrokerTypes []BrokerType `json:"supported_broker_types"`

	// Author identifies the adapter author/maintainer
	Author string `json:"author,omitempty"`

	// License specifies the adapter's license
	License string `json:"license,omitempty"`

	// Documentation provides links to adapter documentation
	Documentation string `json:"documentation,omitempty"`

	// Extended contains adapter-specific metadata
	Extended map[string]any `json:"extended,omitempty"`
}

// AdapterValidationResult represents the result of adapter configuration validation.
type AdapterValidationResult struct {
	// Valid indicates if the configuration is valid
	Valid bool `json:"valid"`

	// Errors contains validation errors that must be fixed
	Errors []AdapterValidationError `json:"errors,omitempty"`

	// Warnings contains validation warnings that should be addressed
	Warnings []AdapterValidationWarning `json:"warnings,omitempty"`

	// Suggestions contains configuration improvement suggestions
	Suggestions []AdapterValidationSuggestion `json:"suggestions,omitempty"`
}

// AdapterValidationError represents an adapter configuration validation error.
type AdapterValidationError struct {
	// Field is the configuration field that has an error
	Field string `json:"field"`

	// Message describes the validation error
	Message string `json:"message"`

	// Code identifies the specific error type
	Code string `json:"code"`

	// Severity indicates the error severity level
	Severity AdapterValidationSeverity `json:"severity"`
}

// AdapterValidationWarning represents an adapter configuration validation warning.
type AdapterValidationWarning struct {
	// Field is the configuration field that has a warning
	Field string `json:"field"`

	// Message describes the validation warning
	Message string `json:"message"`

	// Code identifies the specific warning type
	Code string `json:"code"`
}

// AdapterValidationSuggestion represents an adapter configuration improvement suggestion.
type AdapterValidationSuggestion struct {
	// Field is the configuration field for the suggestion
	Field string `json:"field"`

	// Message describes the suggestion
	Message string `json:"message"`

	// SuggestedValue provides a recommended value if applicable
	SuggestedValue any `json:"suggested_value,omitempty"`
}

// AdapterValidationSeverity represents the severity level of adapter validation issues.
type AdapterValidationSeverity string

const (
	// AdapterValidationSeverityError indicates a critical error that prevents operation
	AdapterValidationSeverityError AdapterValidationSeverity = "error"

	// AdapterValidationSeverityWarning indicates a non-critical issue that should be addressed
	AdapterValidationSeverityWarning AdapterValidationSeverity = "warning"

	// AdapterValidationSeverityInfo indicates informational feedback
	AdapterValidationSeverityInfo AdapterValidationSeverity = "info"
)
