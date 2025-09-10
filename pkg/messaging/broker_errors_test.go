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
	"strings"
	"testing"
)

func TestBrokerSpecificError_Error(t *testing.T) {
	baseErr := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("connection failed").
		Build()

	brokerErr := &BrokerSpecificError{
		BaseMessagingError: baseErr,
		BrokerType:         BrokerTypeKafka,
	}

	errorStr := brokerErr.Error()

	if !strings.Contains(errorStr, "kafka") {
		t.Errorf("Expected error string to contain 'kafka', got %q", errorStr)
	}

	if !strings.Contains(errorStr, "CONNECTION") {
		t.Errorf("Expected error string to contain 'CONNECTION', got %q", errorStr)
	}

	if !strings.Contains(errorStr, "connection failed") {
		t.Errorf("Expected error string to contain 'connection failed', got %q", errorStr)
	}
}

func TestBrokerSpecificError_GetBrokerDiagnostics(t *testing.T) {
	baseErr := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("connection failed").
		Severity(SeverityCritical).
		Build()

	clusterInfo := &BrokerClusterInfo{
		ClusterID:   "test-cluster",
		ClusterName: "test-cluster-name",
		NodeID:      "node-1",
		NodeAddress: "localhost:9092",
	}

	partitionInfo := &PartitionInfo{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    12345,
		Leader:    "node-1",
	}

	consumerGroupInfo := &ConsumerGroupInfo{
		GroupID:     "test-group",
		GroupState:  "Stable",
		Coordinator: "node-1",
		Generation:  5,
	}

	brokerErr := &BrokerSpecificError{
		BaseMessagingError: baseErr,
		BrokerType:         BrokerTypeKafka,
		BrokerVersion:      "2.8.0",
		BrokerErrorCode:    "NETWORK_EXCEPTION",
		BrokerErrorMessage: "Connection to node 1 terminated",
		ClusterInfo:        clusterInfo,
		PartitionInfo:      partitionInfo,
		ConsumerGroupInfo:  consumerGroupInfo,
	}

	diagnostics := brokerErr.GetBrokerDiagnostics()

	// Test base diagnostics
	if diagnostics["type"] != ErrorTypeConnection {
		t.Errorf("Expected type to be %v", ErrorTypeConnection)
	}

	if diagnostics["code"] != ErrCodeConnectionFailed {
		t.Errorf("Expected code to be %v", ErrCodeConnectionFailed)
	}

	if diagnostics["message"] != "connection failed" {
		t.Errorf("Expected message to be 'connection failed'")
	}

	if diagnostics["severity"] != SeverityCritical {
		t.Errorf("Expected severity to be %v", SeverityCritical)
	}

	// Test broker-specific diagnostics
	if diagnostics["broker_type"] != BrokerTypeKafka {
		t.Errorf("Expected broker_type to be %v", BrokerTypeKafka)
	}

	if diagnostics["broker_version"] != "2.8.0" {
		t.Errorf("Expected broker_version to be '2.8.0'")
	}

	if diagnostics["broker_error_code"] != "NETWORK_EXCEPTION" {
		t.Errorf("Expected broker_error_code to be 'NETWORK_EXCEPTION'")
	}

	if diagnostics["broker_error_message"] != "Connection to node 1 terminated" {
		t.Errorf("Expected broker_error_message to be 'Connection to node 1 terminated'")
	}

	// Test structured info
	if diagnostics["cluster_info"] != clusterInfo {
		t.Error("Expected cluster_info to be preserved")
	}

	if diagnostics["partition_info"] != partitionInfo {
		t.Error("Expected partition_info to be preserved")
	}

	if diagnostics["consumer_group_info"] != consumerGroupInfo {
		t.Error("Expected consumer_group_info to be preserved")
	}
}

func TestNewBrokerError(t *testing.T) {
	builder := NewBrokerError(BrokerTypeNATS, ErrorTypeConnection, ErrCodeConnectionFailed)

	if builder == nil {
		t.Fatal("Expected builder to be non-nil")
	}

	if builder.brokerError.BrokerType != BrokerTypeNATS {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeNATS, builder.brokerError.BrokerType)
	}

	if builder.brokerError.Type != ErrorTypeConnection {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeConnection, builder.brokerError.Type)
	}

	if builder.brokerError.Code != ErrCodeConnectionFailed {
		t.Errorf("Expected error code to be %v, got %v", ErrCodeConnectionFailed, builder.brokerError.Code)
	}
}

func TestBrokerErrorBuilder_FluentInterface(t *testing.T) {
	clusterInfo := &BrokerClusterInfo{
		ClusterID: "test-cluster",
		NodeID:    "node-1",
	}

	partitionInfo := &PartitionInfo{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    12345,
	}

	consumerGroupInfo := &ConsumerGroupInfo{
		GroupID:    "test-group",
		GroupState: "Stable",
	}

	brokerErr := NewBrokerError(BrokerTypeKafka, ErrorTypePublishing, ErrCodePublishFailed).
		Message("Publish failed to partition").
		BrokerVersion("2.8.0").
		BrokerErrorCode("NETWORK_EXCEPTION").
		BrokerErrorMessage("Connection lost").
		ClusterInfo(clusterInfo).
		PartitionInfo(partitionInfo).
		ConsumerGroupInfo(consumerGroupInfo).
		Severity(SeverityHigh).
		Retryable(true).
		Context(ErrorContext{
			Component: "kafka-publisher",
			Operation: "publish",
		}).
		Cause(fmt.Errorf("network error")).
		SuggestedAction("Check network connectivity").
		Details(map[string]interface{}{
			"retry_count": 3,
		}).
		Component("updated-component").
		Operation("updated-operation").
		Topic("updated-topic").
		Queue("test-queue").
		Build()

	// Test base properties
	if brokerErr.Message != "Publish failed to partition" {
		t.Errorf("Expected message to be 'Publish failed to partition', got %q", brokerErr.Message)
	}

	if brokerErr.Severity != SeverityHigh {
		t.Errorf("Expected severity to be %v, got %v", SeverityHigh, brokerErr.Severity)
	}

	if !brokerErr.Retryable {
		t.Error("Expected error to be retryable")
	}

	// Test broker-specific properties
	if brokerErr.BrokerType != BrokerTypeKafka {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeKafka, brokerErr.BrokerType)
	}

	if brokerErr.BrokerVersion != "2.8.0" {
		t.Errorf("Expected broker version to be '2.8.0', got %q", brokerErr.BrokerVersion)
	}

	if brokerErr.BrokerErrorCode != "NETWORK_EXCEPTION" {
		t.Errorf("Expected broker error code to be 'NETWORK_EXCEPTION', got %q", brokerErr.BrokerErrorCode)
	}

	if brokerErr.BrokerErrorMessage != "Connection lost" {
		t.Errorf("Expected broker error message to be 'Connection lost', got %q", brokerErr.BrokerErrorMessage)
	}

	if brokerErr.ClusterInfo != clusterInfo {
		t.Error("Expected cluster info to be preserved")
	}

	if brokerErr.PartitionInfo != partitionInfo {
		t.Error("Expected partition info to be preserved")
	}

	if brokerErr.ConsumerGroupInfo != consumerGroupInfo {
		t.Error("Expected consumer group info to be preserved")
	}

	// Test context updates
	if brokerErr.Context.Component != "updated-component" {
		t.Errorf("Expected component to be 'updated-component', got %q", brokerErr.Context.Component)
	}

	if brokerErr.Context.Operation != "updated-operation" {
		t.Errorf("Expected operation to be 'updated-operation', got %q", brokerErr.Context.Operation)
	}

	if brokerErr.Context.Topic != "updated-topic" {
		t.Errorf("Expected topic to be 'updated-topic', got %q", brokerErr.Context.Topic)
	}

	if brokerErr.Context.Queue != "test-queue" {
		t.Errorf("Expected queue to be 'test-queue', got %q", brokerErr.Context.Queue)
	}

	// Test other properties
	if brokerErr.Cause == nil {
		t.Error("Expected cause to be set")
	}

	if brokerErr.SuggestedAction != "Check network connectivity" {
		t.Errorf("Expected suggested action to be 'Check network connectivity', got %q", brokerErr.SuggestedAction)
	}

	if brokerErr.AdditionalDetails["retry_count"] != 3 {
		t.Error("Expected retry_count detail to be 3")
	}
}

func TestNewKafkaPartitionError(t *testing.T) {
	cause := fmt.Errorf("network timeout")
	brokerErr := NewKafkaPartitionError("test-topic", 2, 12345, cause)

	if brokerErr.BrokerType != BrokerTypeKafka {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeKafka, brokerErr.BrokerType)
	}

	if brokerErr.Type != ErrorTypePublishing {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypePublishing, brokerErr.Type)
	}

	if brokerErr.Code != ErrCodePublishFailed {
		t.Errorf("Expected error code to be %v, got %v", ErrCodePublishFailed, brokerErr.Code)
	}

	if !strings.Contains(brokerErr.Message, "test-topic") {
		t.Errorf("Expected message to contain 'test-topic', got %q", brokerErr.Message)
	}

	if !strings.Contains(brokerErr.Message, "partition 2") {
		t.Errorf("Expected message to contain 'partition 2', got %q", brokerErr.Message)
	}

	if !strings.Contains(brokerErr.Message, "offset 12345") {
		t.Errorf("Expected message to contain 'offset 12345', got %q", brokerErr.Message)
	}

	if brokerErr.Cause != cause {
		t.Error("Expected cause to be preserved")
	}

	if brokerErr.PartitionInfo == nil {
		t.Fatal("Expected partition info to be set")
	}

	if brokerErr.PartitionInfo.Topic != "test-topic" {
		t.Errorf("Expected partition topic to be 'test-topic', got %q", brokerErr.PartitionInfo.Topic)
	}

	if brokerErr.PartitionInfo.Partition != 2 {
		t.Errorf("Expected partition to be 2, got %d", brokerErr.PartitionInfo.Partition)
	}

	if brokerErr.PartitionInfo.Offset != 12345 {
		t.Errorf("Expected offset to be 12345, got %d", brokerErr.PartitionInfo.Offset)
	}

	if brokerErr.Severity != SeverityHigh {
		t.Errorf("Expected severity to be %v, got %v", SeverityHigh, brokerErr.Severity)
	}

	if !brokerErr.Retryable {
		t.Error("Expected error to be retryable")
	}
}

func TestNewKafkaConsumerGroupError(t *testing.T) {
	cause := fmt.Errorf("coordinator not available")
	brokerErr := NewKafkaConsumerGroupError("my-consumer-group", "Dead", cause)

	if brokerErr.BrokerType != BrokerTypeKafka {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeKafka, brokerErr.BrokerType)
	}

	if brokerErr.Type != ErrorTypeSubscription {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeSubscription, brokerErr.Type)
	}

	if brokerErr.Code != ErrCodeConsumerGroupError {
		t.Errorf("Expected error code to be %v, got %v", ErrCodeConsumerGroupError, brokerErr.Code)
	}

	if !strings.Contains(brokerErr.Message, "my-consumer-group") {
		t.Errorf("Expected message to contain 'my-consumer-group', got %q", brokerErr.Message)
	}

	if !strings.Contains(brokerErr.Message, "Dead") {
		t.Errorf("Expected message to contain 'Dead', got %q", brokerErr.Message)
	}

	if brokerErr.Cause != cause {
		t.Error("Expected cause to be preserved")
	}

	if brokerErr.ConsumerGroupInfo == nil {
		t.Fatal("Expected consumer group info to be set")
	}

	if brokerErr.ConsumerGroupInfo.GroupID != "my-consumer-group" {
		t.Errorf("Expected group ID to be 'my-consumer-group', got %q", brokerErr.ConsumerGroupInfo.GroupID)
	}

	if brokerErr.ConsumerGroupInfo.GroupState != "Dead" {
		t.Errorf("Expected group state to be 'Dead', got %q", brokerErr.ConsumerGroupInfo.GroupState)
	}
}

func TestNewNATSConnectionError(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	brokerErr := NewNATSConnectionError("nats://localhost:4222", cause)

	if brokerErr.BrokerType != BrokerTypeNATS {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeNATS, brokerErr.BrokerType)
	}

	if brokerErr.Type != ErrorTypeConnection {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeConnection, brokerErr.Type)
	}

	if brokerErr.Code != ErrCodeConnectionFailed {
		t.Errorf("Expected error code to be %v, got %v", ErrCodeConnectionFailed, brokerErr.Code)
	}

	if !strings.Contains(brokerErr.Message, "nats://localhost:4222") {
		t.Errorf("Expected message to contain server URL, got %q", brokerErr.Message)
	}

	if brokerErr.Cause != cause {
		t.Error("Expected cause to be preserved")
	}

	if brokerErr.Context.Component != "nats-client" {
		t.Errorf("Expected component to be 'nats-client', got %q", brokerErr.Context.Component)
	}

	if brokerErr.Context.Operation != "connect" {
		t.Errorf("Expected operation to be 'connect', got %q", brokerErr.Context.Operation)
	}

	if len(brokerErr.Context.BrokerEndpoints) != 1 || brokerErr.Context.BrokerEndpoints[0] != "nats://localhost:4222" {
		t.Errorf("Expected broker endpoints to contain server URL")
	}

	if brokerErr.Severity != SeverityCritical {
		t.Errorf("Expected severity to be %v, got %v", SeverityCritical, brokerErr.Severity)
	}

	if !brokerErr.Retryable {
		t.Error("Expected error to be retryable")
	}

	if brokerErr.SuggestedAction == "" {
		t.Error("Expected suggested action to be set")
	}
}

func TestNewRabbitMQChannelError(t *testing.T) {
	cause := fmt.Errorf("channel exception")
	brokerErr := NewRabbitMQChannelError(5, "ACCESS_REFUSED", cause)

	if brokerErr.BrokerType != BrokerTypeRabbitMQ {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeRabbitMQ, brokerErr.BrokerType)
	}

	if brokerErr.Type != ErrorTypeConnection {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeConnection, brokerErr.Type)
	}

	if brokerErr.Code != ErrCodeConnectionLost {
		t.Errorf("Expected error code to be %v, got %v", ErrCodeConnectionLost, brokerErr.Code)
	}

	if !strings.Contains(brokerErr.Message, "channel 5") {
		t.Errorf("Expected message to contain 'channel 5', got %q", brokerErr.Message)
	}

	if !strings.Contains(brokerErr.Message, "ACCESS_REFUSED") {
		t.Errorf("Expected message to contain 'ACCESS_REFUSED', got %q", brokerErr.Message)
	}

	if brokerErr.Cause != cause {
		t.Error("Expected cause to be preserved")
	}

	if brokerErr.Context.Component != "rabbitmq-channel" {
		t.Errorf("Expected component to be 'rabbitmq-channel', got %q", brokerErr.Context.Component)
	}

	if brokerErr.Context.Metadata["channel_id"] != 5 {
		t.Error("Expected channel_id in metadata")
	}

	if brokerErr.Context.Metadata["close_reason"] != "ACCESS_REFUSED" {
		t.Error("Expected close_reason in metadata")
	}
}

func TestNewRedisStreamError(t *testing.T) {
	cause := fmt.Errorf("stream not found")
	brokerErr := NewRedisStreamError("events", "processor-group", "consumer-1", cause)

	if brokerErr.BrokerType != BrokerTypeRedis {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeRedis, brokerErr.BrokerType)
	}

	if brokerErr.Type != ErrorTypeSubscription {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeSubscription, brokerErr.Type)
	}

	if brokerErr.Code != ErrCodeSubscriptionFailed {
		t.Errorf("Expected error code to be %v, got %v", ErrCodeSubscriptionFailed, brokerErr.Code)
	}

	if !strings.Contains(brokerErr.Message, "events") {
		t.Errorf("Expected message to contain stream name, got %q", brokerErr.Message)
	}

	if !strings.Contains(brokerErr.Message, "processor-group") {
		t.Errorf("Expected message to contain consumer group, got %q", brokerErr.Message)
	}

	if brokerErr.Context.Topic != "events" {
		t.Errorf("Expected topic to be 'events', got %q", brokerErr.Context.Topic)
	}

	if brokerErr.Context.ConsumerGroup != "processor-group" {
		t.Errorf("Expected consumer group to be 'processor-group', got %q", brokerErr.Context.ConsumerGroup)
	}

	if brokerErr.ConsumerGroupInfo.GroupID != "processor-group" {
		t.Errorf("Expected group ID to be 'processor-group', got %q", brokerErr.ConsumerGroupInfo.GroupID)
	}
}

func TestNewSQSThrottleError(t *testing.T) {
	cause := fmt.Errorf("throttling exception")
	brokerErr := NewSQSThrottleError("https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue", cause)

	if brokerErr.BrokerType != BrokerTypeSQS {
		t.Errorf("Expected broker type to be %v, got %v", BrokerTypeSQS, brokerErr.BrokerType)
	}

	if brokerErr.Type != ErrorTypeResource {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeResource, brokerErr.Type)
	}

	if brokerErr.Code != ErrCodeRateLimitExceeded {
		t.Errorf("Expected error code to be %v, got %v", ErrCodeRateLimitExceeded, brokerErr.Code)
	}

	if !strings.Contains(brokerErr.Message, "MyQueue") {
		t.Errorf("Expected message to contain queue URL, got %q", brokerErr.Message)
	}

	if brokerErr.Context.Component != "sqs-client" {
		t.Errorf("Expected component to be 'sqs-client', got %q", brokerErr.Context.Component)
	}

	if brokerErr.Severity != SeverityMedium {
		t.Errorf("Expected severity to be %v, got %v", SeverityMedium, brokerErr.Severity)
	}

	if !strings.Contains(brokerErr.SuggestedAction, "exponential backoff") {
		t.Errorf("Expected suggested action to mention exponential backoff, got %q", brokerErr.SuggestedAction)
	}
}

func TestWrapWithBrokerContext(t *testing.T) {
	tests := []struct {
		name         string
		error        error
		expectedType string
	}{
		{
			name:         "broker-specific error",
			error:        NewKafkaError(ErrorTypeConnection, ErrCodeConnectionFailed).Build(),
			expectedType: "*messaging.BrokerSpecificError",
		},
		{
			name: "messaging error",
			error: NewError(ErrorTypePublishing, ErrCodePublishFailed).
				Message("publish failed").
				Build(),
			expectedType: "*messaging.BrokerSpecificError",
		},
		{
			name:         "generic error",
			error:        fmt.Errorf("generic error"),
			expectedType: "*messaging.BrokerSpecificError",
		},
		{
			name:         "nil error",
			error:        nil,
			expectedType: "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithBrokerContext(BrokerTypeKafka, tt.error, "publish")

			if tt.error == nil && result != nil {
				t.Error("Expected nil result for nil error")
				return
			}

			if tt.error != nil && result == nil {
				t.Error("Expected non-nil result for non-nil error")
				return
			}

			if result != nil {
				actualType := fmt.Sprintf("%T", result)
				if actualType != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, actualType)
				}

				// Test that broker context is properly set
				if brokerErr, ok := result.(*BrokerSpecificError); ok {
					if brokerErr.BrokerType != BrokerTypeKafka {
						t.Errorf("Expected broker type to be %v, got %v", BrokerTypeKafka, brokerErr.BrokerType)
					}
				}
			}
		})
	}
}

func TestExtractBrokerInfo(t *testing.T) {
	clusterInfo := &BrokerClusterInfo{ClusterID: "test-cluster"}
	partitionInfo := &PartitionInfo{Topic: "test-topic", Partition: 0}
	consumerGroupInfo := &ConsumerGroupInfo{GroupID: "test-group"}

	brokerErr := &BrokerSpecificError{
		BaseMessagingError: NewError(ErrorTypeConnection, ErrCodeConnectionFailed).Build(),
		BrokerType:         BrokerTypeKafka,
		ClusterInfo:        clusterInfo,
		PartitionInfo:      partitionInfo,
		ConsumerGroupInfo:  consumerGroupInfo,
	}

	brokerType, cluster, partition, consumerGroup := ExtractBrokerInfo(brokerErr)

	if brokerType != BrokerTypeKafka {
		t.Errorf("Expected broker type %v, got %v", BrokerTypeKafka, brokerType)
	}

	if cluster != clusterInfo {
		t.Error("Expected cluster info to be extracted")
	}

	if partition != partitionInfo {
		t.Error("Expected partition info to be extracted")
	}

	if consumerGroup != consumerGroupInfo {
		t.Error("Expected consumer group info to be extracted")
	}

	// Test with non-broker error
	genericErr := fmt.Errorf("generic error")
	brokerType, cluster, partition, consumerGroup = ExtractBrokerInfo(genericErr)

	if brokerType != "" {
		t.Errorf("Expected empty broker type, got %v", brokerType)
	}

	if cluster != nil {
		t.Error("Expected nil cluster info")
	}

	if partition != nil {
		t.Error("Expected nil partition info")
	}

	if consumerGroup != nil {
		t.Error("Expected nil consumer group info")
	}
}

func TestIsBrokerError(t *testing.T) {
	kafkaErr := NewKafkaError(ErrorTypeConnection, ErrCodeConnectionFailed).Build()
	natsErr := NewNATSError(ErrorTypeConnection, ErrCodeConnectionFailed).Build()
	genericErr := fmt.Errorf("generic error")

	if !IsBrokerError(kafkaErr, BrokerTypeKafka) {
		t.Error("Expected IsBrokerError to return true for Kafka error")
	}

	if IsBrokerError(kafkaErr, BrokerTypeNATS) {
		t.Error("Expected IsBrokerError to return false for wrong broker type")
	}

	if !IsBrokerError(natsErr, BrokerTypeNATS) {
		t.Error("Expected IsBrokerError to return true for NATS error")
	}

	if IsBrokerError(genericErr, BrokerTypeKafka) {
		t.Error("Expected IsBrokerError to return false for generic error")
	}
}

func TestGetBrokerType(t *testing.T) {
	kafkaErr := NewKafkaError(ErrorTypeConnection, ErrCodeConnectionFailed).Build()
	genericErr := fmt.Errorf("generic error")

	if GetBrokerType(kafkaErr) != BrokerTypeKafka {
		t.Errorf("Expected broker type %v, got %v", BrokerTypeKafka, GetBrokerType(kafkaErr))
	}

	if GetBrokerType(genericErr) != "" {
		t.Errorf("Expected empty broker type for generic error, got %v", GetBrokerType(genericErr))
	}
}

func TestPropagateError(t *testing.T) {
	tests := []struct {
		name         string
		error        error
		middleware   string
		expectedType string
	}{
		{
			name:         "nil error",
			error:        nil,
			middleware:   "test-middleware",
			expectedType: "<nil>",
		},
		{
			name: "messaging error",
			error: NewError(ErrorTypeProcessing, ErrCodeProcessingFailed).
				Message("processing failed").
				Build(),
			middleware:   "validation-middleware",
			expectedType: "*messaging.BaseMessagingError",
		},
		{
			name:         "broker error",
			error:        NewKafkaError(ErrorTypePublishing, ErrCodePublishFailed).Build(),
			middleware:   "retry-middleware",
			expectedType: "*messaging.BrokerSpecificError",
		},
		{
			name:         "generic error",
			error:        fmt.Errorf("generic error"),
			middleware:   "auth-middleware",
			expectedType: "*messaging.BaseMessagingError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PropagateError(tt.error, tt.middleware)

			actualType := fmt.Sprintf("%T", result)
			if actualType != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, actualType)
			}

			if result != nil {
				// Test that middleware context is added
				var contextComponent string
				var contextOperation string

				if msgErr, ok := result.(*BaseMessagingError); ok {
					contextComponent = msgErr.Context.Component
					contextOperation = msgErr.Context.Operation
				} else if brokerErr, ok := result.(*BrokerSpecificError); ok {
					contextComponent = brokerErr.Context.Component
					contextOperation = brokerErr.Context.Operation
				}

				if contextComponent != "middleware" {
					t.Errorf("Expected component to be 'middleware', got %q", contextComponent)
				}

				if contextOperation != tt.middleware {
					t.Errorf("Expected operation to be %q, got %q", tt.middleware, contextOperation)
				}
			}
		})
	}
}

func TestCommonBrokerErrors(t *testing.T) {
	// Test that common error instances are properly configured
	tests := []struct {
		name         string
		error        *BrokerSpecificError
		expectedType MessagingErrorType
		expectedCode string
	}{
		{
			name:         "Kafka broker not available",
			error:        ErrKafkaBrokerNotAvailable,
			expectedType: ErrorTypeConnection,
			expectedCode: ErrCodeConnectionFailed,
		},
		{
			name:         "NATS connection lost",
			error:        ErrNATSConnectionLost,
			expectedType: ErrorTypeConnection,
			expectedCode: ErrCodeConnectionLost,
		},
		{
			name:         "RabbitMQ connection blocked",
			error:        ErrRabbitMQConnectionBlocked,
			expectedType: ErrorTypeResource,
			expectedCode: ErrCodeResourceExhausted,
		},
		{
			name:         "Kafka message too large",
			error:        ErrKafkaMessageTooLarge,
			expectedType: ErrorTypePublishing,
			expectedCode: ErrCodeMessageTooLarge,
		},
		{
			name:         "SQS message attribute limit",
			error:        ErrSQSMessageAttributeLimit,
			expectedType: ErrorTypeValidation,
			expectedCode: ErrCodeInvalidMessage,
		},
		{
			name:         "Kafka offset out of range",
			error:        ErrKafkaOffsetOutOfRange,
			expectedType: ErrorTypeSubscription,
			expectedCode: ErrCodeSeekFailed,
		},
		{
			name:         "NATS consumer not found",
			error:        ErrNATSConsumerNotFound,
			expectedType: ErrorTypeSubscription,
			expectedCode: ErrCodeSubscriptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, tt.error.Type)
			}

			if tt.error.Code != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, tt.error.Code)
			}

			if tt.error.Message == "" {
				t.Error("Expected non-empty message")
			}

			if tt.error.SuggestedAction == "" {
				t.Error("Expected non-empty suggested action")
			}
		})
	}
}
