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
	"sync"
	"time"
)

// Note: PublisherMetrics is defined in broker.go to maintain consistency across the messaging package.

// StreamingPublisher extends EventPublisher with high-throughput streaming capabilities.
// This interface is designed for scenarios requiring maximum throughput and minimal latency
// for continuous message streams.
//
// Streaming publishers maintain internal buffers and use advanced batching strategies
// to optimize network utilization and broker performance. They provide flow control
// mechanisms to prevent memory overflow and handle back-pressure gracefully.
//
// Example usage for high-throughput scenarios:
//
//	// Create streaming publisher with optimized config
//	config := PublisherConfig{
//		Topic: "high-volume-events",
//		Batching: BatchingConfig{
//			Enabled:       true,
//			MaxMessages:   1000,
//			MaxBytes:      10 * 1024 * 1024, // 10MB
//			FlushInterval: 50 * time.Millisecond,
//		},
//		Async: true,
//	}
//
//	publisher, err := broker.CreatePublisher(config)
//	streamPublisher := publisher.(StreamingPublisher) // Type assertion if supported
//
//	// Start streaming with flow control
//	stream, err := streamPublisher.StartStream(ctx, StreamConfig{
//		BufferSize:    10000,
//		FlowControl:   true,
//		MaxBatchSize:  500,
//	})
//
//	// Publish continuously with back-pressure handling
//	for event := range eventChannel {
//		if err := stream.Publish(ctx, event); err != nil {
//			// Handle back-pressure or errors
//			break
//		}
//	}
//
//	stream.Close()
type StreamingPublisher interface {
	EventPublisher

	// StartStream initiates a high-throughput streaming session.
	// Returns a stream handle for continuous publishing with flow control.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - config: Stream-specific configuration including buffer sizes and flow control
	//
	// Returns:
	//   - PublishStream: Stream handle for continuous publishing
	//   - error: StreamingError if stream creation fails
	StartStream(ctx context.Context, config StreamConfig) (PublishStream, error)

	// GetStreamMetrics returns streaming-specific metrics.
	// Includes buffer utilization, throughput rates, and back-pressure statistics.
	//
	// Returns:
	//   - *StreamingMetrics: Current streaming metrics snapshot, never nil
	GetStreamMetrics() *StreamingMetrics
}

// PublishStream represents an active streaming session for high-throughput publishing.
// Streams maintain internal buffers and provide flow control to handle back-pressure.
type PublishStream interface {
	// Publish adds a message to the stream buffer.
	// This method is non-blocking and returns immediately if buffer has space.
	// If the buffer is full, behavior depends on flow control settings.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//   - message: Message to publish
	//
	// Returns:
	//   - error: BufferFullError if buffer is full and flow control is disabled
	Publish(ctx context.Context, message *Message) error

	// PublishWithBackpressure publishes a message with explicit back-pressure handling.
	// Blocks if buffer is full until space becomes available or context times out.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - message: Message to publish
	//
	// Returns:
	//   - error: PublishError if publishing fails, context error if cancelled/timed out
	PublishWithBackpressure(ctx context.Context, message *Message) error

	// Flush forces immediate dispatch of all buffered messages.
	// Blocks until all messages are sent or context is cancelled.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: FlushError if flush fails
	Flush(ctx context.Context) error

	// Close closes the stream and releases all resources.
	// Any buffered messages are flushed before closing.
	//
	// Returns:
	//   - error: nil if successful, error if cleanup fails
	Close() error

	// GetBufferStatus returns current buffer utilization statistics.
	//
	// Returns:
	//   - *BufferStatus: Buffer status including size, capacity, and fill rate
	GetBufferStatus() *BufferStatus

	// SetFlowControl enables or disables flow control for back-pressure handling.
	//
	// Parameters:
	//   - enabled: Whether to enable flow control
	SetFlowControl(enabled bool)
}

// StreamConfig defines configuration for streaming publishers.
type StreamConfig struct {
	// BufferSize is the internal buffer capacity for messages
	BufferSize int `json:"buffer_size" default:"10000"`

	// FlowControl enables back-pressure handling when buffer is full
	FlowControl bool `json:"flow_control" default:"true"`

	// MaxBatchSize is the maximum number of messages per batch
	MaxBatchSize int `json:"max_batch_size" default:"1000"`

	// FlushInterval is the maximum time to wait before flushing partial batches
	FlushInterval time.Duration `json:"flush_interval" default:"10ms"`

	// CompressionEnabled enables message compression for better throughput
	CompressionEnabled bool `json:"compression_enabled" default:"true"`

	// CompressionLevel sets the compression level (1-9, higher = better compression)
	CompressionLevel int `json:"compression_level" default:"6"`

	// MaxMemoryUsage sets the maximum memory usage for buffering (in bytes)
	MaxMemoryUsage int64 `json:"max_memory_usage" default:"104857600"` // 100MB
}

// StreamingMetrics provides metrics specific to streaming publishers.
type StreamingMetrics struct {
	// ActiveStreams is the number of currently active streams
	ActiveStreams int64 `json:"active_streams"`

	// TotalStreamsCreated is the cumulative number of streams created
	TotalStreamsCreated int64 `json:"total_streams_created"`

	// BufferUtilization is the average buffer utilization across all streams
	BufferUtilization float64 `json:"buffer_utilization"`

	// ThroughputMBps is the current throughput in megabytes per second
	ThroughputMBps float64 `json:"throughput_mbps"`

	// BackpressureEvents is the count of back-pressure events encountered
	BackpressureEvents int64 `json:"backpressure_events"`

	// CompressionRatio is the average compression ratio achieved
	CompressionRatio float64 `json:"compression_ratio"`

	// MemoryUsage is the current memory usage for all stream buffers
	MemoryUsage int64 `json:"memory_usage"`

	// FlushOperations is the total number of flush operations performed
	FlushOperations int64 `json:"flush_operations"`
}

// BufferStatus provides real-time buffer utilization information.
type BufferStatus struct {
	// Size is the current number of messages in the buffer
	Size int64 `json:"size"`

	// Capacity is the maximum buffer capacity
	Capacity int64 `json:"capacity"`

	// Utilization is the buffer utilization percentage (0-100)
	Utilization float64 `json:"utilization"`

	// BytesBuffered is the total size of buffered messages in bytes
	BytesBuffered int64 `json:"bytes_buffered"`

	// AverageMessageSize is the average size of messages in the buffer
	AverageMessageSize int64 `json:"average_message_size"`

	// HighWaterMark is the maximum buffer size reached since last reset
	HighWaterMark int64 `json:"high_water_mark"`
}

// BatchPublisher provides optimized batch publishing capabilities with advanced batching strategies.
// This interface extends EventPublisher with sophisticated batching logic for high-throughput scenarios.
type BatchPublisher interface {
	EventPublisher

	// PublishBatchAsync publishes a batch of messages asynchronously.
	// Each message in the batch can have its own callback for individual confirmation.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - messages: Batch of messages to publish
	//   - callbacks: Optional callbacks for each message (can be nil for no callbacks)
	//
	// Returns:
	//   - error: BatchPublishError if batch queuing fails
	PublishBatchAsync(ctx context.Context, messages []*Message, callbacks []PublishCallback) error

	// PublishBatchWithPartitioning publishes messages with automatic partitioning.
	// Messages are automatically distributed across partitions based on routing strategy.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - messages: Messages to publish with automatic partitioning
	//   - strategy: Partitioning strategy (hash, round-robin, random)
	//
	// Returns:
	//   - []*PublishConfirmation: Confirmations for each message
	//   - error: BatchPublishError if publishing fails
	PublishBatchWithPartitioning(ctx context.Context, messages []*Message, strategy PartitioningStrategy) ([]*PublishConfirmation, error)

	// SetBatchConfig updates the batching configuration dynamically.
	// Changes take effect for subsequent batch operations.
	//
	// Parameters:
	//   - config: New batching configuration
	//
	// Returns:
	//   - error: ConfigurationError if config is invalid
	SetBatchConfig(config BatchingConfig) error

	// GetBatchMetrics returns batch-specific performance metrics.
	//
	// Returns:
	//   - *BatchMetrics: Current batch metrics snapshot, never nil
	GetBatchMetrics() *BatchMetrics
}

// PartitioningStrategy defines message partitioning strategies for batch publishing.
type PartitioningStrategy string

const (
	// PartitioningHash uses message key hash for consistent partitioning
	PartitioningHash PartitioningStrategy = "hash"

	// PartitioningRoundRobin distributes messages evenly across partitions
	PartitioningRoundRobin PartitioningStrategy = "round_robin"

	// PartitioningRandom assigns partitions randomly
	PartitioningRandom PartitioningStrategy = "random"

	// PartitioningStickyRandom uses sticky partitioning with random assignment
	PartitioningStickyRandom PartitioningStrategy = "sticky_random"
)

// BatchMetrics provides detailed metrics for batch publishing operations.
type BatchMetrics struct {
	// BatchesCreated is the total number of batches created
	BatchesCreated int64 `json:"batches_created"`

	// BatchesPublished is the total number of successfully published batches
	BatchesPublished int64 `json:"batches_published"`

	// BatchesFailed is the total number of failed batch operations
	BatchesFailed int64 `json:"batches_failed"`

	// AverageBatchSize is the average number of messages per batch
	AverageBatchSize float64 `json:"average_batch_size"`

	// MaxBatchSize is the largest batch size processed
	MaxBatchSize int64 `json:"max_batch_size"`

	// AverageBatchLatency is the average time for batch processing
	AverageBatchLatency time.Duration `json:"average_batch_latency"`

	// BatchCompressionRatio is the average compression ratio for batches
	BatchCompressionRatio float64 `json:"batch_compression_ratio"`

	// BatchPartitionDistribution shows message distribution across partitions
	BatchPartitionDistribution map[string]int64 `json:"batch_partition_distribution"`
}

// TransactionalPublisher extends EventPublisher with enhanced transactional capabilities.
// Provides advanced transaction management with savepoints, nested transactions, and rollback strategies.
type TransactionalPublisher interface {
	EventPublisher

	// BeginNestedTransaction starts a nested transaction within the current transaction.
	// Nested transactions provide finer-grained control over commit/rollback operations.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//   - parent: Parent transaction (nil for top-level transaction)
	//
	// Returns:
	//   - Transaction: Nested transaction instance
	//   - error: TransactionError if nested transactions not supported
	BeginNestedTransaction(ctx context.Context, parent Transaction) (Transaction, error)

	// CreateSavepoint creates a savepoint within the current transaction.
	// Savepoints allow partial rollback to specific points in the transaction.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//   - tx: Active transaction
	//   - name: Savepoint name for identification
	//
	// Returns:
	//   - Savepoint: Savepoint handle for rollback operations
	//   - error: TransactionError if savepoints not supported
	CreateSavepoint(ctx context.Context, tx Transaction, name string) (Savepoint, error)

	// GetTransactionMetrics returns transaction-specific metrics.
	//
	// Returns:
	//   - *TransactionMetrics: Current transaction metrics, never nil
	GetTransactionMetrics() *TransactionMetrics
}

// Savepoint represents a point within a transaction that can be rolled back to.
type Savepoint interface {
	// GetName returns the savepoint name.
	GetName() string

	// RollbackTo rolls back the transaction to this savepoint.
	// All changes after the savepoint are discarded.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: TransactionError if rollback fails
	RollbackTo(ctx context.Context) error

	// Release releases the savepoint, making it unavailable for rollback.
	// This is an optimization to free resources when the savepoint is no longer needed.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: TransactionError if release fails
	Release(ctx context.Context) error
}

// TransactionMetrics provides detailed metrics for transactional operations.
type TransactionMetrics struct {
	// ActiveTransactions is the number of currently active transactions
	ActiveTransactions int64 `json:"active_transactions"`

	// TransactionsCommitted is the total number of successfully committed transactions
	TransactionsCommitted int64 `json:"transactions_committed"`

	// TransactionsRolledBack is the total number of rolled back transactions
	TransactionsRolledBack int64 `json:"transactions_rolled_back"`

	// AverageTransactionDuration is the average time for transaction completion
	AverageTransactionDuration time.Duration `json:"average_transaction_duration"`

	// SavepointsCreated is the total number of savepoints created
	SavepointsCreated int64 `json:"savepoints_created"`

	// SavepointRollbacks is the total number of savepoint rollbacks
	SavepointRollbacks int64 `json:"savepoint_rollbacks"`

	// NestedTransactions is the count of nested transactions
	NestedTransactions int64 `json:"nested_transactions"`

	// MaxTransactionDepth is the maximum nesting depth reached
	MaxTransactionDepth int64 `json:"max_transaction_depth"`
}

// PublisherBuilder provides a fluent interface for constructing publishers with complex configurations.
// This builder pattern simplifies the creation of publishers with multiple optional features.
//
// Example usage:
//
//	publisher, err := NewPublisherBuilder(broker).
//		WithTopic("events").
//		WithBatching(100, 5*time.Second).
//		WithCompression(CompressionGZIP).
//		WithRetries(3, time.Second).
//		WithStreaming(true).
//		WithTransactions(true).
//		Build(ctx)
type PublisherBuilder interface {
	// WithTopic sets the target topic for publishing.
	WithTopic(topic string) PublisherBuilder

	// WithBatching enables batching with specified parameters.
	WithBatching(maxMessages int, flushInterval time.Duration) PublisherBuilder

	// WithCompression enables compression with the specified type.
	WithCompression(compressionType CompressionType) PublisherBuilder

	// WithRetries configures retry behavior for failed publishes.
	WithRetries(maxAttempts int, initialDelay time.Duration) PublisherBuilder

	// WithStreaming enables high-throughput streaming capabilities.
	WithStreaming(enabled bool) PublisherBuilder

	// WithTransactions enables transactional publishing support.
	WithTransactions(enabled bool) PublisherBuilder

	// WithAsync enables asynchronous publishing by default.
	WithAsync(enabled bool) PublisherBuilder

	// WithTimeout sets timeout values for various operations.
	WithTimeout(publish, flush, close time.Duration) PublisherBuilder

	// WithMiddleware adds middleware to the publisher pipeline.
	WithMiddleware(middleware ...PublisherMiddleware) PublisherBuilder

	// Build creates the configured publisher instance.
	Build(ctx context.Context) (EventPublisher, error)
}

// PublisherMiddleware defines middleware for publisher operations.
// Middleware can add cross-cutting concerns like metrics, logging, and validation.
type PublisherMiddleware interface {
	// Name returns the middleware name for identification.
	Name() string

	// WrapPublish wraps the publish operation with middleware logic.
	WrapPublish(next PublishFunc) PublishFunc

	// WrapPublishBatch wraps the batch publish operation with middleware logic.
	WrapPublishBatch(next PublishBatchFunc) PublishBatchFunc
}

// PublishFunc is the function signature for publish operations.
type PublishFunc func(ctx context.Context, message *Message) error

// PublishBatchFunc is the function signature for batch publish operations.
type PublishBatchFunc func(ctx context.Context, messages []*Message) error

// publisherBuilderImpl provides a concrete implementation of PublisherBuilder.
type publisherBuilderImpl struct {
	broker MessageBroker
	config PublisherConfig

	streaming     bool
	transactional bool
	middleware    []PublisherMiddleware

	mu sync.RWMutex
}

// NewPublisherBuilder creates a new publisher builder with the specified broker.
func NewPublisherBuilder(broker MessageBroker) PublisherBuilder {
	return &publisherBuilderImpl{
		broker: broker,
		config: PublisherConfig{
			Batching: BatchingConfig{
				Enabled:       true,
				MaxMessages:   100,
				MaxBytes:      1024 * 1024, // 1MB
				FlushInterval: 100 * time.Millisecond,
			},
			Compression: CompressionNone,
			Retry: RetryConfig{
				MaxAttempts:  3,
				InitialDelay: time.Second,
				MaxDelay:     30 * time.Second,
				Multiplier:   2.0,
				Jitter:       0.1,
			},
			Timeout: TimeoutConfig{
				Publish: 30 * time.Second,
				Flush:   30 * time.Second,
				Close:   30 * time.Second,
			},
		},
	}
}

// WithTopic sets the target topic for publishing.
func (b *publisherBuilderImpl) WithTopic(topic string) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Topic = topic
	return b
}

// WithBatching enables batching with specified parameters.
func (b *publisherBuilderImpl) WithBatching(maxMessages int, flushInterval time.Duration) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Batching.Enabled = true
	b.config.Batching.MaxMessages = maxMessages
	b.config.Batching.FlushInterval = flushInterval
	return b
}

// WithCompression enables compression with the specified type.
func (b *publisherBuilderImpl) WithCompression(compressionType CompressionType) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Compression = compressionType
	return b
}

// WithRetries configures retry behavior for failed publishes.
func (b *publisherBuilderImpl) WithRetries(maxAttempts int, initialDelay time.Duration) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Retry.MaxAttempts = maxAttempts
	b.config.Retry.InitialDelay = initialDelay
	return b
}

// WithStreaming enables high-throughput streaming capabilities.
func (b *publisherBuilderImpl) WithStreaming(enabled bool) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.streaming = enabled
	return b
}

// WithTransactions enables transactional publishing support.
func (b *publisherBuilderImpl) WithTransactions(enabled bool) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Transactional = enabled
	b.transactional = enabled
	return b
}

// WithAsync enables asynchronous publishing by default.
func (b *publisherBuilderImpl) WithAsync(enabled bool) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Async = enabled
	return b
}

// WithTimeout sets timeout values for various operations.
func (b *publisherBuilderImpl) WithTimeout(publish, flush, close time.Duration) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config.Timeout.Publish = publish
	b.config.Timeout.Flush = flush
	b.config.Timeout.Close = close
	return b
}

// WithMiddleware adds middleware to the publisher pipeline.
func (b *publisherBuilderImpl) WithMiddleware(middleware ...PublisherMiddleware) PublisherBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middleware = append(b.middleware, middleware...)
	return b
}

// Build creates the configured publisher instance.
func (b *publisherBuilderImpl) Build(ctx context.Context) (EventPublisher, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Validate configuration
	if b.config.Topic == "" {
		return nil, NewConfigError("topic is required", nil)
	}

	// Create base publisher
	publisher, err := b.broker.CreatePublisher(b.config)
	if err != nil {
		return nil, err
	}

	// Wrap with middleware if any
	if len(b.middleware) > 0 {
		publisher = b.wrapWithMiddleware(publisher)
	}

	return publisher, nil
}

// wrapWithMiddleware wraps the publisher with middleware chain.
func (b *publisherBuilderImpl) wrapWithMiddleware(publisher EventPublisher) EventPublisher {
	// Implementation would wrap the publisher with middleware
	// This is a simplified placeholder - actual implementation would
	// create a middleware chain wrapper
	return publisher
}
