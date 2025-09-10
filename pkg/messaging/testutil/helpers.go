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

package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestMessage represents a simple test message structure for testing purposes.
type TestMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// CreateTestMessage creates a test message with the specified parameters.
func CreateTestMessage(id, msgType string, payload map[string]interface{}) messaging.Message {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	testMsg := TestMessage{
		ID:        id,
		Type:      msgType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	payloadBytes, _ := json.Marshal(testMsg)

	return messaging.Message{
		ID: id,
		Headers: map[string]string{
			"type":      msgType,
			"timestamp": testMsg.Timestamp.Format(time.RFC3339),
		},
		Payload:   payloadBytes,
		Topic:     "test-topic",
		Key:       []byte(id),
		Timestamp: testMsg.Timestamp,
	}
}

// CreateTestMessages creates a slice of test messages for batch testing.
func CreateTestMessages(count int, msgType string) []messaging.Message {
	messages := make([]messaging.Message, count)
	for i := 0; i < count; i++ {
		payload := map[string]interface{}{
			"index": i,
			"data":  fmt.Sprintf("test-data-%d", i),
		}
		messages[i] = CreateTestMessage(fmt.Sprintf("msg-%d", i), msgType, payload)
	}
	return messages
}

// CreateTestBrokerConfig creates a test broker configuration.
func CreateTestBrokerConfig() *messaging.BrokerConfig {
	return &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
		Connection: messaging.ConnectionConfig{
			Timeout:     30 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 300 * time.Second,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
		Monitoring: messaging.MonitoringConfig{
			Enabled:             true,
			MetricsInterval:     30 * time.Second,
			HealthCheckInterval: 60 * time.Second,
		},
	}
}

// CreateTestPublisherConfig creates a test publisher configuration.
func CreateTestPublisherConfig() *messaging.PublisherConfig {
	return &messaging.PublisherConfig{
		Topic: "test-topic",
		Routing: messaging.RoutingConfig{
			Strategy: "round_robin",
		},
		Batching: messaging.BatchingConfig{
			Enabled:       true,
			MaxMessages:   10,
			FlushInterval: 1 * time.Second,
			MaxBytes:      1024 * 1024, // 1MB
		},
		Confirmation: messaging.ConfirmationConfig{
			Required: false,
			Timeout:  5 * time.Second,
		},
		Timeout: messaging.TimeoutConfig{
			Publish: 30 * time.Second,
			Flush:   5 * time.Second,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}
}

// CreateTestSubscriberConfig creates a test subscriber configuration.
func CreateTestSubscriberConfig() *messaging.SubscriberConfig {
	return &messaging.SubscriberConfig{
		Topics:        []string{"test-topic"},
		ConsumerGroup: "test-group",
		Processing: messaging.ProcessingConfig{
			MaxProcessingTime: 30 * time.Second,
			AckMode:           messaging.AckModeAuto,
			MaxInFlight:       100,
			Ordered:           false,
		},
		DeadLetter: messaging.DeadLetterConfig{
			Enabled:    true,
			Topic:      "test-topic-dlq",
			MaxRetries: 3,
			TTL:        168 * time.Hour,
		},
		Offset: messaging.OffsetConfig{
			Initial:    messaging.OffsetLatest,
			Interval:   1 * time.Second,
			AutoCommit: true,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}
}

// AssertMessagesEqual asserts that two messages are equal in content.
func AssertMessagesEqual(t *testing.T, expected, actual messaging.Message) {
	assert.Equal(t, expected.ID, actual.ID, "Message ID should match")
	assert.Equal(t, expected.Topic, actual.Topic, "Message topic should match")
	assert.Equal(t, expected.Headers, actual.Headers, "Message headers should match")
	assert.Equal(t, expected.Payload, actual.Payload, "Message payload should match")
	assert.Equal(t, expected.Key, actual.Key, "Message key should match")
}

// AssertMessageSlicesEqual asserts that two message slices are equal.
func AssertMessageSlicesEqual(t *testing.T, expected, actual []messaging.Message) {
	require.Equal(t, len(expected), len(actual), "Message slice lengths should match")

	for i, expectedMsg := range expected {
		AssertMessagesEqual(t, expectedMsg, actual[i])
	}
}

// WaitForCondition waits for a condition to become true within a timeout period.
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for condition: %s", message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// WaitForMessages waits for a specific number of messages to be received.
func WaitForMessages(t *testing.T, getCount func() int, expectedCount int, timeout time.Duration) {
	WaitForCondition(t, func() bool {
		return getCount() >= expectedCount
	}, timeout, fmt.Sprintf("expected %d messages", expectedCount))
}

// TestBrokerSetup provides a complete test setup for message broker testing.
type TestBrokerSetup struct {
	Broker           *MockMessageBroker
	Publisher        *MockEventPublisher
	Subscriber       *MockEventSubscriber
	Handler          *MockMessageHandler
	Factory          *MockBrokerFactory
	BrokerConfig     *messaging.BrokerConfig
	PublisherConfig  *messaging.PublisherConfig
	SubscriberConfig *messaging.SubscriberConfig
}

// NewTestBrokerSetup creates a complete test setup with all necessary mocks.
func NewTestBrokerSetup() *TestBrokerSetup {
	setup := &TestBrokerSetup{
		Broker:           NewMockMessageBroker(),
		Publisher:        NewMockEventPublisher(),
		Subscriber:       NewMockEventSubscriber(),
		Handler:          NewMockMessageHandler(),
		Factory:          NewMockBrokerFactory(),
		BrokerConfig:     CreateTestBrokerConfig(),
		PublisherConfig:  CreateTestPublisherConfig(),
		SubscriberConfig: CreateTestSubscriberConfig(),
	}

	// Set up default mock expectations
	setup.SetupDefaultExpectations()

	return setup
}

// SetupDefaultExpectations sets up common mock expectations.
func (s *TestBrokerSetup) SetupDefaultExpectations() {
	// Default broker expectations
	s.Broker.On("Connect", mock.Anything).Return(nil).Maybe()
	s.Broker.On("Disconnect", mock.Anything).Return(nil).Maybe()
	s.Broker.On("Close").Return(nil).Maybe()
	s.Broker.On("IsConnected").Return(true).Maybe()

	// Default publisher expectations
	s.Broker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(s.Publisher, nil).Maybe()
	s.Publisher.On("Publish", mock.Anything, mock.AnythingOfType("messaging.Message")).Return(nil).Maybe()
	s.Publisher.On("PublishBatch", mock.Anything, mock.AnythingOfType("[]messaging.Message")).Return(nil).Maybe()
	s.Publisher.On("Close").Return(nil).Maybe()

	// Default subscriber expectations
	s.Broker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(s.Subscriber, nil).Maybe()
	s.Subscriber.On("Subscribe", mock.Anything, mock.AnythingOfType("*testutil.MockMessageHandler")).Return(nil).Maybe()
	s.Subscriber.On("Unsubscribe", mock.Anything).Return(nil).Maybe()
	s.Subscriber.On("Close").Return(nil).Maybe()

	// Default handler expectations
	s.Handler.On("HandleMessage", mock.Anything, mock.AnythingOfType("messaging.Message")).Return(nil).Maybe()

	// Default factory expectations
	s.Factory.On("CreateBroker", mock.AnythingOfType("*messaging.BrokerConfig")).Return(s.Broker, nil).Maybe()
	s.Factory.On("GetSupportedBrokerTypes").Return([]messaging.BrokerType{messaging.BrokerTypeInMemory}).Maybe()
	s.Factory.On("ValidateConfig", mock.AnythingOfType("*messaging.BrokerConfig")).Return(nil).Maybe()
}

// AssertAllExpectations asserts that all mock expectations were met.
func (s *TestBrokerSetup) AssertAllExpectations(t *testing.T) {
	s.Broker.AssertExpectations(t)
	s.Publisher.AssertExpectations(t)
	s.Subscriber.AssertExpectations(t)
	s.Handler.AssertExpectations(t)
	s.Factory.AssertExpectations(t)
}

// CreateHealthyHealthStatus creates a healthy status for testing.
func CreateHealthyHealthStatus() *messaging.HealthStatus {
	return &messaging.HealthStatus{
		Status:       messaging.HealthStatusHealthy,
		Message:      "All systems operational",
		LastChecked:  time.Now(),
		ResponseTime: 10 * time.Millisecond,
		Details: map[string]any{
			"connections": "active",
			"queues":      "available",
		},
	}
}

// CreateUnhealthyHealthStatus creates an unhealthy status for testing.
func CreateUnhealthyHealthStatus() *messaging.HealthStatus {
	return &messaging.HealthStatus{
		Status:       messaging.HealthStatusUnhealthy,
		Message:      "Connection failed",
		LastChecked:  time.Now(),
		ResponseTime: 5 * time.Second,
		Details: map[string]any{
			"error": "connection timeout",
		},
	}
}

// SimulateAsyncError creates a channel that sends an error after a delay for testing async operations.
func SimulateAsyncError(err error, delay time.Duration) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		errChan <- err
		close(errChan)
	}()
	return errChan
}

// SimulateAsyncSuccess creates a channel that sends nil after a delay for testing async operations.
func SimulateAsyncSuccess(delay time.Duration) <-chan error {
	return SimulateAsyncError(nil, delay)
}

// BenchmarkHelper provides utilities for performance benchmarking.
type BenchmarkHelper struct {
	MessageCount int
	MessageSize  int
	BatchSize    int
}

// NewBenchmarkHelper creates a new benchmark helper with default values.
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{
		MessageCount: 1000,
		MessageSize:  1024,
		BatchSize:    10,
	}
}

// CreateBenchmarkMessages creates messages suitable for benchmarking.
func (b *BenchmarkHelper) CreateBenchmarkMessages() []messaging.Message {
	messages := make([]messaging.Message, b.MessageCount)

	// Create payload of specified size
	payload := make([]byte, b.MessageSize)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	for i := 0; i < b.MessageCount; i++ {
		messages[i] = messaging.Message{
			ID:        fmt.Sprintf("bench-msg-%d", i),
			Headers:   map[string]string{"benchmark": "true"},
			Payload:   payload,
			Topic:     "benchmark-topic",
			Key:       []byte(fmt.Sprintf("key-%d", i%10)), // 10 different keys for distribution
			Timestamp: time.Now(),
		}
	}

	return messages
}

// CreateBenchmarkBatches creates message batches for batch benchmarking.
func (b *BenchmarkHelper) CreateBenchmarkBatches() [][]messaging.Message {
	messages := b.CreateBenchmarkMessages()
	batches := make([][]messaging.Message, 0, (len(messages)+b.BatchSize-1)/b.BatchSize)

	for i := 0; i < len(messages); i += b.BatchSize {
		end := i + b.BatchSize
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}

	return batches
}
