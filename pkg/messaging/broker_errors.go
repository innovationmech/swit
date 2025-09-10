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
	"fmt"
	"time"
)

// Additional broker types not defined in broker.go
const (
	BrokerTypeRedis   BrokerType = "redis"
	BrokerTypePulsar  BrokerType = "pulsar"
	BrokerTypeKinesis BrokerType = "kinesis"
	BrokerTypeSQS     BrokerType = "sqs"
)

// BrokerSpecificError extends BaseMessagingError with broker-specific information.
type BrokerSpecificError struct {
	*BaseMessagingError

	// BrokerType identifies which broker generated this error
	BrokerType BrokerType `json:"broker_type"`

	// BrokerVersion contains the broker version information
	BrokerVersion string `json:"broker_version,omitempty"`

	// BrokerErrorCode is the native error code from the broker
	BrokerErrorCode string `json:"broker_error_code,omitempty"`

	// BrokerErrorMessage is the original error message from the broker
	BrokerErrorMessage string `json:"broker_error_message,omitempty"`

	// ClusterInfo provides information about the broker cluster
	ClusterInfo *BrokerClusterInfo `json:"cluster_info,omitempty"`

	// PartitionInfo provides partition-specific information for partition-based brokers
	PartitionInfo *PartitionInfo `json:"partition_info,omitempty"`

	// ConsumerGroupInfo provides consumer group information for subscription errors
	ConsumerGroupInfo *ConsumerGroupInfo `json:"consumer_group_info,omitempty"`
}

// BrokerClusterInfo contains cluster-related information.
type BrokerClusterInfo struct {
	ClusterID   string   `json:"cluster_id,omitempty"`
	ClusterName string   `json:"cluster_name,omitempty"`
	NodeID      string   `json:"node_id,omitempty"`
	NodeAddress string   `json:"node_address,omitempty"`
	Leaders     []string `json:"leaders,omitempty"`
	Replicas    []string `json:"replicas,omitempty"`
}

// PartitionInfo contains partition-specific information.
type PartitionInfo struct {
	Topic         string `json:"topic"`
	Partition     int32  `json:"partition"`
	Offset        int64  `json:"offset,omitempty"`
	HighWatermark int64  `json:"high_watermark,omitempty"`
	Leader        string `json:"leader,omitempty"`
}

// ConsumerGroupInfo contains consumer group information.
type ConsumerGroupInfo struct {
	GroupID     string   `json:"group_id"`
	GroupState  string   `json:"group_state,omitempty"`
	Coordinator string   `json:"coordinator,omitempty"`
	Members     []string `json:"members,omitempty"`
	Generation  int32    `json:"generation,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
}

// Error implements the error interface for BrokerSpecificError.
func (e *BrokerSpecificError) Error() string {
	baseError := e.BaseMessagingError.Error()

	if e.BrokerType != "" {
		return fmt.Sprintf("[%s] %s", e.BrokerType, baseError)
	}

	return baseError
}

// GetBrokerDiagnostics returns broker-specific diagnostic information.
func (e *BrokerSpecificError) GetBrokerDiagnostics() map[string]interface{} {
	diagnostics := e.BaseMessagingError.GetDiagnosticInfo()

	if e.BrokerType != "" {
		diagnostics["broker_type"] = e.BrokerType
	}
	if e.BrokerVersion != "" {
		diagnostics["broker_version"] = e.BrokerVersion
	}
	if e.BrokerErrorCode != "" {
		diagnostics["broker_error_code"] = e.BrokerErrorCode
	}
	if e.BrokerErrorMessage != "" {
		diagnostics["broker_error_message"] = e.BrokerErrorMessage
	}
	if e.ClusterInfo != nil {
		diagnostics["cluster_info"] = e.ClusterInfo
	}
	if e.PartitionInfo != nil {
		diagnostics["partition_info"] = e.PartitionInfo
	}
	if e.ConsumerGroupInfo != nil {
		diagnostics["consumer_group_info"] = e.ConsumerGroupInfo
	}

	return diagnostics
}

// BrokerErrorBuilder provides a fluent interface for building broker-specific errors.
type BrokerErrorBuilder struct {
	*ErrorBuilder
	brokerError *BrokerSpecificError
}

// NewBrokerError creates a new broker-specific error builder.
func NewBrokerError(brokerType BrokerType, errType MessagingErrorType, code string) *BrokerErrorBuilder {
	baseBuilder := NewError(errType, code)
	brokerError := &BrokerSpecificError{
		BaseMessagingError: baseBuilder.err,
		BrokerType:         brokerType,
	}

	return &BrokerErrorBuilder{
		ErrorBuilder: baseBuilder,
		brokerError:  brokerError,
	}
}

// BrokerVersion sets the broker version.
func (b *BrokerErrorBuilder) BrokerVersion(version string) *BrokerErrorBuilder {
	b.brokerError.BrokerVersion = version
	return b
}

// BrokerErrorCode sets the native broker error code.
func (b *BrokerErrorBuilder) BrokerErrorCode(code string) *BrokerErrorBuilder {
	b.brokerError.BrokerErrorCode = code
	return b
}

// BrokerErrorMessage sets the original broker error message.
func (b *BrokerErrorBuilder) BrokerErrorMessage(message string) *BrokerErrorBuilder {
	b.brokerError.BrokerErrorMessage = message
	return b
}

// ClusterInfo sets the cluster information.
func (b *BrokerErrorBuilder) ClusterInfo(info *BrokerClusterInfo) *BrokerErrorBuilder {
	b.brokerError.ClusterInfo = info
	return b
}

// PartitionInfo sets the partition information.
func (b *BrokerErrorBuilder) PartitionInfo(info *PartitionInfo) *BrokerErrorBuilder {
	b.brokerError.PartitionInfo = info
	return b
}

// ConsumerGroupInfo sets the consumer group information.
func (b *BrokerErrorBuilder) ConsumerGroupInfo(info *ConsumerGroupInfo) *BrokerErrorBuilder {
	b.brokerError.ConsumerGroupInfo = info
	return b
}

// Override base methods to return BrokerErrorBuilder for proper chaining

// Message sets the error message.
func (b *BrokerErrorBuilder) Message(msg string) *BrokerErrorBuilder {
	b.ErrorBuilder.Message(msg)
	return b
}

// Messagef sets the error message using format string.
func (b *BrokerErrorBuilder) Messagef(format string, args ...interface{}) *BrokerErrorBuilder {
	b.ErrorBuilder.Messagef(format, args...)
	return b
}

// Severity sets the error severity.
func (b *BrokerErrorBuilder) Severity(severity ErrorSeverity) *BrokerErrorBuilder {
	b.ErrorBuilder.Severity(severity)
	return b
}

// Retryable marks the error as retryable or not.
func (b *BrokerErrorBuilder) Retryable(retryable bool) *BrokerErrorBuilder {
	b.ErrorBuilder.Retryable(retryable)
	return b
}

// Context sets the error context.
func (b *BrokerErrorBuilder) Context(ctx ErrorContext) *BrokerErrorBuilder {
	b.ErrorBuilder.Context(ctx)
	return b
}

// Cause sets the underlying cause error.
func (b *BrokerErrorBuilder) Cause(cause error) *BrokerErrorBuilder {
	b.ErrorBuilder.Cause(cause)
	return b
}

// SuggestedAction sets the suggested action for resolving the error.
func (b *BrokerErrorBuilder) SuggestedAction(action string) *BrokerErrorBuilder {
	b.ErrorBuilder.SuggestedAction(action)
	return b
}

// Details adds additional debugging details.
func (b *BrokerErrorBuilder) Details(details map[string]interface{}) *BrokerErrorBuilder {
	b.ErrorBuilder.Details(details)
	return b
}

// Component sets the component context.
func (b *BrokerErrorBuilder) Component(component string) *BrokerErrorBuilder {
	b.ErrorBuilder.Component(component)
	return b
}

// Operation sets the operation context.
func (b *BrokerErrorBuilder) Operation(operation string) *BrokerErrorBuilder {
	b.ErrorBuilder.Operation(operation)
	return b
}

// Topic sets the topic context.
func (b *BrokerErrorBuilder) Topic(topic string) *BrokerErrorBuilder {
	b.ErrorBuilder.Topic(topic)
	return b
}

// Queue sets the queue context.
func (b *BrokerErrorBuilder) Queue(queue string) *BrokerErrorBuilder {
	b.ErrorBuilder.Queue(queue)
	return b
}

// Build creates the final broker-specific error.
func (b *BrokerErrorBuilder) Build() *BrokerSpecificError {
	return b.brokerError
}

// Error creates the final broker-specific error and returns it as an error interface.
func (b *BrokerErrorBuilder) Error() error {
	return b.brokerError
}

// Common broker-specific error constructors

// Kafka Error Constructors

// NewKafkaError creates a new Kafka-specific error.
func NewKafkaError(errType MessagingErrorType, code string) *BrokerErrorBuilder {
	return NewBrokerError(BrokerTypeKafka, errType, code)
}

// NewKafkaPartitionError creates a Kafka partition-related error.
func NewKafkaPartitionError(topic string, partition int32, offset int64, cause error) *BrokerSpecificError {
	return NewKafkaError(ErrorTypePublishing, ErrCodePublishFailed).
		Message(fmt.Sprintf("partition error for topic %s, partition %d at offset %d", topic, partition, offset)).
		Cause(cause).
		PartitionInfo(&PartitionInfo{
			Topic:     topic,
			Partition: partition,
			Offset:    offset,
		}).
		Severity(SeverityHigh).
		Retryable(true).
		Build()
}

// NewKafkaConsumerGroupError creates a Kafka consumer group error.
func NewKafkaConsumerGroupError(groupID, state string, cause error) *BrokerSpecificError {
	return NewKafkaError(ErrorTypeSubscription, ErrCodeConsumerGroupError).
		Message(fmt.Sprintf("consumer group %s error (state: %s)", groupID, state)).
		Cause(cause).
		ConsumerGroupInfo(&ConsumerGroupInfo{
			GroupID:    groupID,
			GroupState: state,
		}).
		Severity(SeverityMedium).
		Retryable(true).
		Build()
}

// NATS Error Constructors

// NewNATSError creates a new NATS-specific error.
func NewNATSError(errType MessagingErrorType, code string) *BrokerErrorBuilder {
	return NewBrokerError(BrokerTypeNATS, errType, code)
}

// NewNATSConnectionError creates a NATS connection error.
func NewNATSConnectionError(serverURL string, cause error) *BrokerSpecificError {
	return NewNATSError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message(fmt.Sprintf("failed to connect to NATS server at %s", serverURL)).
		Cause(cause).
		Context(ErrorContext{
			Component:       "nats-client",
			Operation:       "connect",
			BrokerEndpoints: []string{serverURL},
		}).
		Severity(SeverityCritical).
		Retryable(true).
		SuggestedAction("Check NATS server availability and network connectivity").
		Build()
}

// RabbitMQ Error Constructors

// NewRabbitMQError creates a new RabbitMQ-specific error.
func NewRabbitMQError(errType MessagingErrorType, code string) *BrokerErrorBuilder {
	return NewBrokerError(BrokerTypeRabbitMQ, errType, code)
}

// NewRabbitMQChannelError creates a RabbitMQ channel error.
func NewRabbitMQChannelError(channelID int, reason string, cause error) *BrokerSpecificError {
	return NewRabbitMQError(ErrorTypeConnection, ErrCodeConnectionLost).
		Message(fmt.Sprintf("RabbitMQ channel %d closed: %s", channelID, reason)).
		Cause(cause).
		Context(ErrorContext{
			Component: "rabbitmq-channel",
			Operation: "channel-operation",
			Metadata: map[string]interface{}{
				"channel_id":   channelID,
				"close_reason": reason,
			},
		}).
		Severity(SeverityHigh).
		Retryable(true).
		SuggestedAction("Reconnect and recreate the channel").
		Build()
}

// Redis Error Constructors

// NewRedisError creates a new Redis-specific error.
func NewRedisError(errType MessagingErrorType, code string) *BrokerErrorBuilder {
	return NewBrokerError(BrokerTypeRedis, errType, code)
}

// NewRedisStreamError creates a Redis streams error.
func NewRedisStreamError(stream, consumerGroup, consumer string, cause error) *BrokerSpecificError {
	return NewRedisError(ErrorTypeSubscription, ErrCodeSubscriptionFailed).
		Message(fmt.Sprintf("Redis stream error for stream %s, group %s, consumer %s", stream, consumerGroup, consumer)).
		Cause(cause).
		Context(ErrorContext{
			Topic:         stream,
			ConsumerGroup: consumerGroup,
			Component:     "redis-streams",
			Operation:     "consume",
		}).
		ConsumerGroupInfo(&ConsumerGroupInfo{
			GroupID: consumerGroup,
		}).
		Severity(SeverityMedium).
		Retryable(true).
		Build()
}

// AWS SQS Error Constructors

// NewSQSError creates a new SQS-specific error.
func NewSQSError(errType MessagingErrorType, code string) *BrokerErrorBuilder {
	return NewBrokerError(BrokerTypeSQS, errType, code)
}

// NewSQSThrottleError creates an SQS throttling error.
func NewSQSThrottleError(queueURL string, cause error) *BrokerSpecificError {
	return NewSQSError(ErrorTypeResource, ErrCodeRateLimitExceeded).
		Message(fmt.Sprintf("SQS throttling error for queue %s", queueURL)).
		Cause(cause).
		Context(ErrorContext{
			Queue:     queueURL,
			Component: "sqs-client",
			Operation: "send-message",
		}).
		Severity(SeverityMedium).
		Retryable(true).
		SuggestedAction("Implement exponential backoff and reduce request rate").
		Build()
}

// Generic Error Propagation Utilities

// WrapWithBrokerContext wraps a generic error with broker-specific context.
func WrapWithBrokerContext(brokerType BrokerType, err error, operation string) error {
	if err == nil {
		return nil
	}

	// If it's already a broker-specific error, return as-is
	if brokerErr, ok := err.(*BrokerSpecificError); ok {
		return brokerErr
	}

	// If it's a messaging error, wrap it
	if msgErr, ok := err.(*BaseMessagingError); ok {
		return &BrokerSpecificError{
			BaseMessagingError: msgErr.WithContext(ErrorContext{
				Component: fmt.Sprintf("%s-client", brokerType),
				Operation: operation,
			}),
			BrokerType: brokerType,
		}
	}

	// For generic errors, create a new broker error
	return NewBrokerError(brokerType, ErrorTypeInternal, ErrCodeInternal).
		Message(fmt.Sprintf("%s operation failed", operation)).
		Cause(err).
		Context(ErrorContext{
			Component: fmt.Sprintf("%s-client", brokerType),
			Operation: operation,
		}).
		Severity(SeverityMedium).
		Retryable(true).
		Build()
}

// ExtractBrokerInfo extracts broker-specific information from an error.
func ExtractBrokerInfo(err error) (BrokerType, *BrokerClusterInfo, *PartitionInfo, *ConsumerGroupInfo) {
	if brokerErr, ok := err.(*BrokerSpecificError); ok {
		return brokerErr.BrokerType, brokerErr.ClusterInfo, brokerErr.PartitionInfo, brokerErr.ConsumerGroupInfo
	}
	return "", nil, nil, nil
}

// IsBrokerError checks if an error is a broker-specific error.
func IsBrokerError(err error, brokerType BrokerType) bool {
	if brokerErr, ok := err.(*BrokerSpecificError); ok {
		return brokerErr.BrokerType == brokerType
	}
	return false
}

// GetBrokerType extracts the broker type from an error.
func GetBrokerType(err error) BrokerType {
	if brokerErr, ok := err.(*BrokerSpecificError); ok {
		return brokerErr.BrokerType
	}
	return ""
}

// Common broker error instances for frequently used errors

var (
	// Connection errors
	ErrKafkaBrokerNotAvailable = NewKafkaError(ErrorTypeConnection, ErrCodeConnectionFailed).
					Message("Kafka broker not available").
					Severity(SeverityCritical).
					Retryable(true).
					SuggestedAction("Check Kafka cluster health and network connectivity").
					Build()

	ErrNATSConnectionLost = NewNATSError(ErrorTypeConnection, ErrCodeConnectionLost).
				Message("NATS connection lost").
				Severity(SeverityHigh).
				Retryable(true).
				SuggestedAction("Check NATS server status and reconnect").
				Build()

	ErrRabbitMQConnectionBlocked = NewRabbitMQError(ErrorTypeResource, ErrCodeResourceExhausted).
					Message("RabbitMQ connection blocked due to memory/disk alarm").
					Severity(SeverityHigh).
					Retryable(true).
					SuggestedAction("Check RabbitMQ server resources and clear alarms").
					Build()

	// Publishing errors
	ErrKafkaMessageTooLarge = NewKafkaError(ErrorTypePublishing, ErrCodeMessageTooLarge).
				Message("message size exceeds Kafka broker limits").
				Severity(SeverityMedium).
				Retryable(false).
				SuggestedAction("Reduce message size or increase broker message.max.bytes configuration").
				Build()

	ErrSQSMessageAttributeLimit = NewSQSError(ErrorTypeValidation, ErrCodeInvalidMessage).
					Message("SQS message attributes limit exceeded").
					Severity(SeverityLow).
					Retryable(false).
					SuggestedAction("Reduce number of message attributes or use message body for additional data").
					Build()

	// Subscription errors
	ErrKafkaOffsetOutOfRange = NewKafkaError(ErrorTypeSubscription, ErrCodeSeekFailed).
					Message("Kafka offset out of range").
					Severity(SeverityMedium).
					Retryable(true).
					SuggestedAction("Reset consumer offset to latest or earliest position").
					Build()

	ErrNATSConsumerNotFound = NewNATSError(ErrorTypeSubscription, ErrCodeSubscriptionFailed).
				Message("NATS JetStream consumer not found").
				Severity(SeverityMedium).
				Retryable(true).
				SuggestedAction("Recreate the consumer or check consumer configuration").
				Build()
)

// Error propagation through middleware chain
func PropagateError(err error, middlewareName string) error {
	if err == nil {
		return nil
	}

	// If it's already a messaging error, add middleware context
	if msgErr, ok := err.(*BaseMessagingError); ok {
		return msgErr.WithContext(ErrorContext{
			Component: "middleware",
			Operation: middlewareName,
			StartTime: time.Now(),
		})
	}

	// If it's a broker error, add middleware context
	if brokerErr, ok := err.(*BrokerSpecificError); ok {
		brokerErr.BaseMessagingError = brokerErr.BaseMessagingError.WithContext(ErrorContext{
			Component: "middleware",
			Operation: middlewareName,
			StartTime: time.Now(),
		})
		return brokerErr
	}

	// Wrap generic error
	return NewError(ErrorTypeProcessing, ErrCodeMiddlewareFailure).
		Message(fmt.Sprintf("middleware '%s' failed", middlewareName)).
		Cause(err).
		Context(ErrorContext{
			Component: "middleware",
			Operation: middlewareName,
			StartTime: time.Now(),
		}).
		Severity(SeverityMedium).
		Retryable(false).
		Build()
}
