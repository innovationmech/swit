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
	"errors"
	"fmt"
	"maps"
	"time"
)

// BrokerType represents the type of message broker.
type BrokerType string

const (
	// BrokerTypeKafka represents Apache Kafka message broker
	BrokerTypeKafka BrokerType = "kafka"
	// BrokerTypeNATS represents NATS message broker
	BrokerTypeNATS BrokerType = "nats"
	// BrokerTypeRabbitMQ represents RabbitMQ message broker
	BrokerTypeRabbitMQ BrokerType = "rabbitmq"
	// BrokerTypeInMemory represents in-memory message broker for testing
	BrokerTypeInMemory BrokerType = "inmemory"
)

// String returns the string representation of BrokerType.
func (bt BrokerType) String() string {
	return string(bt)
}

// IsValid checks if the broker type is valid.
func (bt BrokerType) IsValid() bool {
	switch bt {
	case BrokerTypeKafka, BrokerTypeNATS, BrokerTypeRabbitMQ, BrokerTypeInMemory:
		return true
	default:
		return false
	}
}

// BrokerCapabilities describes what features a broker supports.
// This enables feature detection and allows applications to adapt
// their behavior based on broker capabilities.
type BrokerCapabilities struct {
	// SupportsTransactions indicates if the broker supports transactional publishing
	SupportsTransactions bool `json:"supports_transactions"`

	// SupportsOrdering indicates if the broker guarantees message ordering
	SupportsOrdering bool `json:"supports_ordering"`

	// SupportsPartitioning indicates if the broker supports topic partitioning
	SupportsPartitioning bool `json:"supports_partitioning"`

	// SupportsDeadLetter indicates if the broker supports dead letter queues
	SupportsDeadLetter bool `json:"supports_dead_letter"`

	// SupportsDelayedDelivery indicates if the broker supports delayed message delivery
	SupportsDelayedDelivery bool `json:"supports_delayed_delivery"`

	// SupportsPriority indicates if the broker supports message priority
	SupportsPriority bool `json:"supports_priority"`

	// SupportsStreaming indicates if the broker supports stream processing
	SupportsStreaming bool `json:"supports_streaming"`

	// SupportsSeek indicates if the broker supports seeking to specific positions
	SupportsSeek bool `json:"supports_seek"`

	// SupportsConsumerGroups indicates if the broker supports consumer groups
	SupportsConsumerGroups bool `json:"supports_consumer_groups"`

	// MaxMessageSize is the maximum message size in bytes (0 = unlimited)
	MaxMessageSize int64 `json:"max_message_size"`

	// MaxBatchSize is the maximum number of messages in a batch (0 = unlimited)
	MaxBatchSize int `json:"max_batch_size"`

	// MaxTopicNameLength is the maximum length of topic names (0 = unlimited)
	MaxTopicNameLength int `json:"max_topic_name_length"`

	// SupportedCompressionTypes lists supported compression algorithms
	SupportedCompressionTypes []CompressionType `json:"supported_compression_types"`

	// SupportedSerializationTypes lists supported serialization formats
	SupportedSerializationTypes []SerializationType `json:"supported_serialization_types"`

	// Additional broker-specific capabilities as key-value pairs
	Extended map[string]any `json:"extended,omitempty"`
}

// BrokerMetrics contains broker performance and operational metrics.
// These metrics help monitor broker health and performance characteristics.
type BrokerMetrics struct {
	// Connection metrics
	ConnectionStatus    string        `json:"connection_status"`
	ConnectionUptime    time.Duration `json:"connection_uptime"`
	ConnectionAttempts  int64         `json:"connection_attempts"`
	ConnectionFailures  int64         `json:"connection_failures"`
	LastConnectionError string        `json:"last_connection_error,omitempty"`
	LastConnectionTime  time.Time     `json:"last_connection_time"`

	// Publisher metrics
	PublishersCreated int64 `json:"publishers_created"`
	ActivePublishers  int64 `json:"active_publishers"`

	// Subscriber metrics
	SubscribersCreated int64 `json:"subscribers_created"`
	ActiveSubscribers  int64 `json:"active_subscribers"`

	// Message metrics
	MessagesPublished int64 `json:"messages_published"`
	MessagesConsumed  int64 `json:"messages_consumed"`
	PublishErrors     int64 `json:"publish_errors"`
	ConsumeErrors     int64 `json:"consume_errors"`

	// Performance metrics
	AvgPublishLatency   time.Duration `json:"avg_publish_latency"`
	AvgConsumeLatency   time.Duration `json:"avg_consume_latency"`
	ThroughputPerSecond float64       `json:"throughput_per_second"`

	// Resource metrics
	MemoryUsage       int64     `json:"memory_usage"`
	OpenConnections   int64     `json:"open_connections"`
	LastMetricsUpdate time.Time `json:"last_metrics_update"`

	// Broker-specific metrics
	Extended map[string]any `json:"extended,omitempty"`
}

// GetSnapshot returns a copy of the current metrics to prevent race conditions.
func (m *BrokerMetrics) GetSnapshot() *BrokerMetrics {
	if m == nil {
		return &BrokerMetrics{}
	}

	snapshot := *m

	// Deep copy extended map
	if m.Extended != nil {
		snapshot.Extended = make(map[string]any)
		maps.Copy(snapshot.Extended, m.Extended)
	}

	return &snapshot
}

// PublisherMetrics contains publisher-specific performance metrics.
type PublisherMetrics struct {
	// Message count metrics
	MessagesPublished int64 `json:"messages_published"`
	MessagesQueued    int64 `json:"messages_queued"`
	MessagesFailed    int64 `json:"messages_failed"`
	MessagesRetried   int64 `json:"messages_retried"`

	// Batch metrics
	BatchesPublished int64   `json:"batches_published"`
	AvgBatchSize     float64 `json:"avg_batch_size"`

	// Timing metrics
	AvgPublishLatency time.Duration `json:"avg_publish_latency"`
	MaxPublishLatency time.Duration `json:"max_publish_latency"`
	MinPublishLatency time.Duration `json:"min_publish_latency"`

	// Throughput metrics
	MessagesPerSecond float64 `json:"messages_per_second"`
	BytesPerSecond    float64 `json:"bytes_per_second"`

	// Error metrics
	LastError     string    `json:"last_error,omitempty"`
	LastErrorTime time.Time `json:"last_error_time"`
	ErrorRate     float64   `json:"error_rate"`

	// Queue metrics
	QueueDepth    int64 `json:"queue_depth"`
	QueueCapacity int64 `json:"queue_capacity"`

	// Connection metrics
	ConnectionStatus string    `json:"connection_status"`
	LastActivity     time.Time `json:"last_activity"`

	// Transaction metrics (if supported)
	TransactionsStarted   int64 `json:"transactions_started"`
	TransactionsCommitted int64 `json:"transactions_committed"`
	TransactionsAborted   int64 `json:"transactions_aborted"`

	// Publisher-specific metrics
	Extended map[string]any `json:"extended,omitempty"`
}

// SubscriberMetrics contains subscriber-specific performance metrics.
type SubscriberMetrics struct {
	// Message count metrics
	MessagesConsumed     int64 `json:"messages_consumed"`
	MessagesProcessed    int64 `json:"messages_processed"`
	MessagesFailed       int64 `json:"messages_failed"`
	MessagesRetried      int64 `json:"messages_retried"`
	MessagesDeadLettered int64 `json:"messages_dead_lettered"`

	// Processing metrics
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	MaxProcessingTime time.Duration `json:"max_processing_time"`
	MinProcessingTime time.Duration `json:"min_processing_time"`

	// Throughput metrics
	MessagesPerSecond float64 `json:"messages_per_second"`
	BytesPerSecond    float64 `json:"bytes_per_second"`

	// Error metrics
	LastError     string    `json:"last_error,omitempty"`
	LastErrorTime time.Time `json:"last_error_time"`
	ErrorRate     float64   `json:"error_rate"`

	// Lag metrics
	CurrentLag    int64     `json:"current_lag"`
	MaxLag        int64     `json:"max_lag"`
	LastLagUpdate time.Time `json:"last_lag_update"`

	// Consumer group metrics
	ConsumerGroup      string  `json:"consumer_group"`
	PartitionCount     int32   `json:"partition_count"`
	AssignedPartitions []int32 `json:"assigned_partitions"`

	// Connection metrics
	ConnectionStatus  string    `json:"connection_status"`
	LastActivity      time.Time `json:"last_activity"`
	Rebalances        int64     `json:"rebalances"`
	LastRebalanceTime time.Time `json:"last_rebalance_time"`

	// Subscriber-specific metrics
	Extended map[string]any `json:"extended,omitempty"`
}

// HealthStatus represents the health status of a broker component.
type HealthStatus struct {
	// Status indicates the overall health status
	Status HealthStatusType `json:"status"`

	// Message provides human-readable status description
	Message string `json:"message"`

	// LastChecked is the timestamp of the last health check
	LastChecked time.Time `json:"last_checked"`

	// ResponseTime is how long the health check took
	ResponseTime time.Duration `json:"response_time"`

	// Details contains component-specific health information
	Details map[string]any `json:"details,omitempty"`

	// Checks contains sub-component health check results
	Checks map[string]*HealthStatus `json:"checks,omitempty"`
}

// HealthStatusType represents the possible health status values.
type HealthStatusType string

const (
	// HealthStatusHealthy indicates the component is operating normally
	HealthStatusHealthy HealthStatusType = "healthy"

	// HealthStatusDegraded indicates the component is working but with reduced performance
	HealthStatusDegraded HealthStatusType = "degraded"

	// HealthStatusUnhealthy indicates the component is not working properly
	HealthStatusUnhealthy HealthStatusType = "unhealthy"

	// HealthStatusUnknown indicates the health status could not be determined
	HealthStatusUnknown HealthStatusType = "unknown"
)

// IsHealthy returns true if the status indicates a healthy state.
func (s HealthStatusType) IsHealthy() bool {
	return s == HealthStatusHealthy
}

// String returns the string representation of HealthStatusType.
func (s HealthStatusType) String() string {
	return string(s)
}

// CompressionType represents supported compression algorithms.
type CompressionType string

const (
	// CompressionNone indicates no compression
	CompressionNone CompressionType = "none"

	// CompressionGZIP indicates GZIP compression
	CompressionGZIP CompressionType = "gzip"

	// CompressionSnappy indicates Snappy compression
	CompressionSnappy CompressionType = "snappy"

	// CompressionLZ4 indicates LZ4 compression
	CompressionLZ4 CompressionType = "lz4"

	// CompressionZSTD indicates ZSTD compression
	CompressionZSTD CompressionType = "zstd"
)

// SerializationType represents supported serialization formats.
type SerializationType string

const (
	// SerializationJSON indicates JSON serialization
	SerializationJSON SerializationType = "json"

	// SerializationProtobuf indicates Protocol Buffers serialization
	SerializationProtobuf SerializationType = "protobuf"

	// SerializationAvro indicates Apache Avro serialization
	SerializationAvro SerializationType = "avro"

	// SerializationMsgPack indicates MessagePack serialization
	SerializationMsgPack SerializationType = "msgpack"
)

// ErrorAction defines what to do when message processing fails.
type ErrorAction int

const (
	// ErrorActionRetry indicates the message should be retried
	ErrorActionRetry ErrorAction = iota

	// ErrorActionDeadLetter indicates the message should be sent to dead letter queue
	ErrorActionDeadLetter

	// ErrorActionDiscard indicates the message should be discarded
	ErrorActionDiscard

	// ErrorActionPause indicates processing should be paused
	ErrorActionPause
)

// String returns the string representation of ErrorAction.
func (ea ErrorAction) String() string {
	switch ea {
	case ErrorActionRetry:
		return "retry"
	case ErrorActionDeadLetter:
		return "dead_letter"
	case ErrorActionDiscard:
		return "discard"
	case ErrorActionPause:
		return "pause"
	default:
		return "unknown"
	}
}

// MessagingError is the base error type for messaging operations.
// It provides structured error information with categorization and context.
type MessagingError struct {
	// Code identifies the specific error type
	Code ErrorCode `json:"code"`

	// Message provides a human-readable error description
	Message string `json:"message"`

	// Cause is the underlying error that caused this error
	Cause error `json:"-"`

	// Retryable indicates if this error can be resolved by retrying
	Retryable bool `json:"retryable"`

	// Details provides additional context-specific information
	Details map[string]any `json:"details,omitempty"`

	// Timestamp indicates when the error occurred
	Timestamp time.Time `json:"timestamp"`
}

// Error implements the error interface.
func (e *MessagingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap implements the unwrapping interface for Go 1.13+ error handling.
func (e *MessagingError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for Go 1.13+ error handling.
func (e *MessagingError) Is(target error) bool {
	if t, ok := target.(*MessagingError); ok {
		return e.Code == t.Code
	}
	return false
}

// ErrorCode defines specific error codes for different failure scenarios.
type ErrorCode string

const (
	// Connection errors
	ErrConnectionFailed    ErrorCode = "CONNECTION_FAILED"
	ErrConnectionTimeout   ErrorCode = "CONNECTION_TIMEOUT"
	ErrConnectionLost      ErrorCode = "CONNECTION_LOST"
	ErrDisconnectionFailed ErrorCode = "DISCONNECTION_FAILED"

	// Authentication and authorization errors
	ErrAuthenticationFailed ErrorCode = "AUTH_FAILED"
	ErrAuthorizationDenied  ErrorCode = "AUTH_DENIED"
	ErrInvalidCredentials   ErrorCode = "INVALID_CREDENTIALS"

	// Configuration errors
	ErrInvalidConfig     ErrorCode = "INVALID_CONFIG"
	ErrUnsupportedBroker ErrorCode = "UNSUPPORTED_BROKER"
	ErrConfigValidation  ErrorCode = "CONFIG_VALIDATION"

	// Publishing errors
	ErrPublishFailed      ErrorCode = "PUBLISH_FAILED"
	ErrPublishTimeout     ErrorCode = "PUBLISH_TIMEOUT"
	ErrMessageTooLarge    ErrorCode = "MESSAGE_TOO_LARGE"
	ErrTopicNotFound      ErrorCode = "TOPIC_NOT_FOUND"
	ErrBatchPublishFailed ErrorCode = "BATCH_PUBLISH_FAILED"
	ErrPublisherClosed    ErrorCode = "PUBLISHER_CLOSED"

	// Subscription errors
	ErrSubscriptionFailed ErrorCode = "SUBSCRIPTION_FAILED"
	ErrConsumerGroupError ErrorCode = "CONSUMER_GROUP_ERROR"
	ErrRebalancing        ErrorCode = "REBALANCING"
	ErrSubscriberClosed   ErrorCode = "SUBSCRIBER_CLOSED"
	ErrUnsubscribeFailed  ErrorCode = "UNSUBSCRIBE_FAILED"

	// Processing errors
	ErrProcessingFailed       ErrorCode = "PROCESSING_FAILED"
	ErrProcessingTimeout      ErrorCode = "PROCESSING_TIMEOUT"
	ErrInvalidMessage         ErrorCode = "INVALID_MESSAGE"
	ErrMessageDeserialization ErrorCode = "MESSAGE_DESERIALIZATION"
	ErrMiddlewareFailure      ErrorCode = "MIDDLEWARE_FAILURE"

	// Transaction errors
	ErrTransactionFailed       ErrorCode = "TRANSACTION_FAILED"
	ErrTransactionAborted      ErrorCode = "TRANSACTION_ABORTED"
	ErrTransactionTimeout      ErrorCode = "TRANSACTION_TIMEOUT"
	ErrTransactionNotSupported ErrorCode = "TRANSACTION_NOT_SUPPORTED"

	// Health check errors
	ErrHealthCheckFailed  ErrorCode = "HEALTH_CHECK_FAILED"
	ErrHealthCheckTimeout ErrorCode = "HEALTH_CHECK_TIMEOUT"

	// Seeking and positioning errors
	ErrSeekFailed       ErrorCode = "SEEK_FAILED"
	ErrSeekNotSupported ErrorCode = "SEEK_NOT_SUPPORTED"
	ErrInvalidPosition  ErrorCode = "INVALID_POSITION"

	// Resource errors
	ErrResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrQueueFull         ErrorCode = "QUEUE_FULL"

	// Generic errors
	ErrInternal         ErrorCode = "INTERNAL_ERROR"
	ErrNotImplemented   ErrorCode = "NOT_IMPLEMENTED"
	ErrOperationAborted ErrorCode = "OPERATION_ABORTED"
)

// Common error constructors for convenience and consistency.

// NewConnectionError creates a new connection-related error.
func NewConnectionError(message string, cause error) *MessagingError {
	return &MessagingError{
		Code:      ErrConnectionFailed,
		Message:   message,
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// NewPublishError creates a new publishing-related error.
func NewPublishError(message string, cause error) *MessagingError {
	return &MessagingError{
		Code:      ErrPublishFailed,
		Message:   message,
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// NewProcessingError creates a new processing-related error.
func NewProcessingError(message string, cause error) *MessagingError {
	return &MessagingError{
		Code:      ErrProcessingFailed,
		Message:   message,
		Cause:     cause,
		Retryable: false,
		Timestamp: time.Now(),
	}
}

// NewConfigError creates a new configuration-related error.
func NewConfigError(message string, cause error) *MessagingError {
	return &MessagingError{
		Code:      ErrInvalidConfig,
		Message:   message,
		Cause:     cause,
		Retryable: false,
		Timestamp: time.Now(),
	}
}

// NewTransactionError creates a new transaction-related error.
func NewTransactionError(message string, cause error) *MessagingError {
	return &MessagingError{
		Code:      ErrTransactionFailed,
		Message:   message,
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// Common error variables for frequently used errors.
var (
	// ErrBrokerNotConnected indicates the broker is not connected
	ErrBrokerNotConnected = &MessagingError{
		Code:      ErrConnectionFailed,
		Message:   "broker is not connected",
		Retryable: true,
		Timestamp: time.Now(),
	}

	// ErrPublisherAlreadyClosed indicates the publisher has been closed
	ErrPublisherAlreadyClosed = &MessagingError{
		Code:      ErrPublisherClosed,
		Message:   "publisher has been closed",
		Retryable: false,
		Timestamp: time.Now(),
	}

	// ErrSubscriberAlreadyClosed indicates the subscriber has been closed
	ErrSubscriberAlreadyClosed = &MessagingError{
		Code:      ErrSubscriberClosed,
		Message:   "subscriber has been closed",
		Retryable: false,
		Timestamp: time.Now(),
	}

	// ErrTransactionsNotSupported indicates the broker doesn't support transactions
	ErrTransactionsNotSupported = &MessagingError{
		Code:      ErrTransactionNotSupported,
		Message:   "broker does not support transactions",
		Retryable: false,
		Timestamp: time.Now(),
	}
)

// Helper functions for error type checking.

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	var msgErr *MessagingError
	if errors.As(err, &msgErr) {
		return msgErr.Retryable
	}
	return false
}

// IsConnectionError checks if an error is connection-related.
func IsConnectionError(err error) bool {
	var msgErr *MessagingError
	if errors.As(err, &msgErr) {
		switch msgErr.Code {
		case ErrConnectionFailed, ErrConnectionTimeout, ErrConnectionLost, ErrDisconnectionFailed:
			return true
		}
	}
	return false
}

// IsConfigurationError checks if an error is configuration-related.
func IsConfigurationError(err error) bool {
	var msgErr *MessagingError
	if errors.As(err, &msgErr) {
		switch msgErr.Code {
		case ErrInvalidConfig, ErrUnsupportedBroker, ErrConfigValidation:
			return true
		}
	}
	return false
}

// IsPublishError checks if an error is publishing-related.
func IsPublishError(err error) bool {
	var msgErr *MessagingError
	if errors.As(err, &msgErr) {
		switch msgErr.Code {
		case ErrPublishFailed, ErrPublishTimeout, ErrMessageTooLarge,
			ErrTopicNotFound, ErrBatchPublishFailed, ErrPublisherClosed:
			return true
		}
	}
	return false
}
