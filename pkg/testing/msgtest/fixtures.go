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

package msgtest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// TestMessagePayload represents the JSON payload structure produced by
// NewTestMessage. It can be used to unmarshal message payloads in assertions.
type TestMessagePayload struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// NewTestMessage creates a test message with the specified identifier, type
// and payload. A nil payload is replaced with an empty map.
func NewTestMessage(id, msgType string, payload map[string]interface{}) *messaging.Message {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	testMsg := TestMessagePayload{
		ID:        id,
		Type:      msgType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	payloadBytes, _ := json.Marshal(testMsg)

	return &messaging.Message{
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

// NewTestMessages creates a slice of test messages for batch testing.
func NewTestMessages(count int, msgType string) []*messaging.Message {
	messages := make([]*messaging.Message, count)
	for i := 0; i < count; i++ {
		payload := map[string]interface{}{
			"index": i,
			"data":  fmt.Sprintf("test-data-%d", i),
		}
		messages[i] = NewTestMessage(fmt.Sprintf("msg-%d", i), msgType, payload)
	}
	return messages
}

// NewTestBrokerConfig creates a broker configuration suitable for unit tests
// (in-memory broker type with short timeouts).
func NewTestBrokerConfig() *messaging.BrokerConfig {
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

// NewTestPublisherConfig creates a publisher configuration suitable for unit tests.
func NewTestPublisherConfig() *messaging.PublisherConfig {
	return &messaging.PublisherConfig{
		Topic: "test-topic",
		Routing: messaging.RoutingConfig{
			Strategy: "round_robin",
		},
		Batching: messaging.BatchingConfig{
			Enabled:       true,
			MaxMessages:   10,
			FlushInterval: 1 * time.Second,
			MaxBytes:      1024 * 1024,
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

// NewTestSubscriberConfig creates a subscriber configuration suitable for unit tests.
func NewTestSubscriberConfig() *messaging.SubscriberConfig {
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

// NewHealthyStatus creates a healthy broker status fixture.
func NewHealthyStatus() *messaging.HealthStatus {
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

// NewUnhealthyStatus creates an unhealthy broker status fixture.
func NewUnhealthyStatus() *messaging.HealthStatus {
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
