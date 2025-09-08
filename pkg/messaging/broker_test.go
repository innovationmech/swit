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

package messaging

import (
	"errors"
	"testing"
	"time"
)

// TestBrokerType tests broker type methods.
func TestBrokerType(t *testing.T) {
	tests := []struct {
		name     string
		bt       BrokerType
		valid    bool
		expected string
	}{
		{"Kafka", BrokerTypeKafka, true, "kafka"},
		{"NATS", BrokerTypeNATS, true, "nats"},
		{"RabbitMQ", BrokerTypeRabbitMQ, true, "rabbitmq"},
		{"InMemory", BrokerTypeInMemory, true, "inmemory"},
		{"Invalid", BrokerType("invalid"), false, "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String method
			if got := tt.bt.String(); got != tt.expected {
				t.Errorf("BrokerType.String() = %v, want %v", got, tt.expected)
			}

			// Test IsValid method
			if got := tt.bt.IsValid(); got != tt.valid {
				t.Errorf("BrokerType.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// TestHealthStatusType tests health status type methods.
func TestHealthStatusType(t *testing.T) {
	tests := []struct {
		name      string
		status    HealthStatusType
		isHealthy bool
		expected  string
	}{
		{"Healthy", HealthStatusHealthy, true, "healthy"},
		{"Degraded", HealthStatusDegraded, false, "degraded"},
		{"Unhealthy", HealthStatusUnhealthy, false, "unhealthy"},
		{"Unknown", HealthStatusUnknown, false, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String method
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("HealthStatusType.String() = %v, want %v", got, tt.expected)
			}

			// Test IsHealthy method
			if got := tt.status.IsHealthy(); got != tt.isHealthy {
				t.Errorf("HealthStatusType.IsHealthy() = %v, want %v", got, tt.isHealthy)
			}
		})
	}
}

// TestErrorAction tests error action string representation.
func TestErrorAction(t *testing.T) {
	tests := []struct {
		name     string
		action   ErrorAction
		expected string
	}{
		{"Retry", ErrorActionRetry, "retry"},
		{"DeadLetter", ErrorActionDeadLetter, "dead_letter"},
		{"Discard", ErrorActionDiscard, "discard"},
		{"Pause", ErrorActionPause, "pause"},
		{"Unknown", ErrorAction(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("ErrorAction.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestMessagingError tests the MessagingError type.
func TestMessagingError(t *testing.T) {
	cause := errors.New("underlying error")

	// Test error without cause
	err1 := &MessagingError{
		Code:      ErrConnectionFailed,
		Message:   "connection failed",
		Retryable: true,
		Timestamp: time.Now(),
	}

	expected1 := "CONNECTION_FAILED: connection failed"
	if got := err1.Error(); got != expected1 {
		t.Errorf("MessagingError.Error() = %v, want %v", got, expected1)
	}

	// Test error with cause
	err2 := &MessagingError{
		Code:      ErrPublishFailed,
		Message:   "publish failed",
		Cause:     cause,
		Retryable: true,
		Timestamp: time.Now(),
	}

	expected2 := "PUBLISH_FAILED: publish failed (cause: underlying error)"
	if got := err2.Error(); got != expected2 {
		t.Errorf("MessagingError.Error() = %v, want %v", got, expected2)
	}

	// Test Unwrap
	if got := err2.Unwrap(); got != cause {
		t.Errorf("MessagingError.Unwrap() = %v, want %v", got, cause)
	}

	// Test Is method
	err3 := &MessagingError{Code: ErrConnectionFailed}
	if !err1.Is(err3) {
		t.Error("Expected err1.Is(err3) to be true for same error codes")
	}

	err4 := &MessagingError{Code: ErrPublishFailed}
	if err1.Is(err4) {
		t.Error("Expected err1.Is(err4) to be false for different error codes")
	}

	// Test Is with non-MessagingError
	if err1.Is(cause) {
		t.Error("Expected err1.Is(cause) to be false for non-MessagingError")
	}
}

// TestErrorConstructors tests the error constructor functions.
func TestErrorConstructors(t *testing.T) {
	cause := errors.New("underlying error")
	message := "test message"

	// Test NewConnectionError
	connErr := NewConnectionError(message, cause)
	if connErr.Code != ErrConnectionFailed {
		t.Errorf("Expected connection error code, got: %v", connErr.Code)
	}
	if connErr.Message != message {
		t.Errorf("Expected message %v, got: %v", message, connErr.Message)
	}
	if connErr.Cause != cause {
		t.Errorf("Expected cause %v, got: %v", cause, connErr.Cause)
	}
	if !connErr.Retryable {
		t.Error("Expected connection error to be retryable")
	}

	// Test NewPublishError
	pubErr := NewPublishError(message, cause)
	if pubErr.Code != ErrPublishFailed {
		t.Errorf("Expected publish error code, got: %v", pubErr.Code)
	}
	if !pubErr.Retryable {
		t.Error("Expected publish error to be retryable")
	}

	// Test NewProcessingError
	procErr := NewProcessingError(message, cause)
	if procErr.Code != ErrProcessingFailed {
		t.Errorf("Expected processing error code, got: %v", procErr.Code)
	}
	if procErr.Retryable {
		t.Error("Expected processing error to not be retryable")
	}

	// Test NewConfigError
	confErr := NewConfigError(message, cause)
	if confErr.Code != ErrInvalidConfig {
		t.Errorf("Expected config error code, got: %v", confErr.Code)
	}
	if confErr.Retryable {
		t.Error("Expected config error to not be retryable")
	}

	// Test NewTransactionError
	txErr := NewTransactionError(message, cause)
	if txErr.Code != ErrTransactionFailed {
		t.Errorf("Expected transaction error code, got: %v", txErr.Code)
	}
	if !txErr.Retryable {
		t.Error("Expected transaction error to be retryable")
	}
}

// TestErrorHelpers tests the error helper functions.
func TestErrorHelpers(t *testing.T) {
	// Test with MessagingError
	retryableErr := &MessagingError{
		Code:      ErrConnectionFailed,
		Retryable: true,
	}

	nonRetryableErr := &MessagingError{
		Code:      ErrProcessingFailed,
		Retryable: false,
	}

	// Test IsRetryableError
	if !IsRetryableError(retryableErr) {
		t.Error("Expected retryable error to be identified as retryable")
	}

	if IsRetryableError(nonRetryableErr) {
		t.Error("Expected non-retryable error to be identified as non-retryable")
	}

	genericErr := errors.New("generic error")
	if IsRetryableError(genericErr) {
		t.Error("Expected generic error to be non-retryable")
	}

	// Test IsConnectionError
	connectionErr := &MessagingError{Code: ErrConnectionFailed}
	if !IsConnectionError(connectionErr) {
		t.Error("Expected connection error to be identified as connection error")
	}

	publishErr := &MessagingError{Code: ErrPublishFailed}
	if IsConnectionError(publishErr) {
		t.Error("Expected publish error to not be identified as connection error")
	}

	// Test IsConfigurationError
	configErr := &MessagingError{Code: ErrInvalidConfig}
	if !IsConfigurationError(configErr) {
		t.Error("Expected config error to be identified as configuration error")
	}

	if IsConfigurationError(connectionErr) {
		t.Error("Expected connection error to not be identified as configuration error")
	}

	// Test IsPublishError
	if !IsPublishError(publishErr) {
		t.Error("Expected publish error to be identified as publish error")
	}

	if IsPublishError(configErr) {
		t.Error("Expected config error to not be identified as publish error")
	}
}

// TestBrokerMetrics tests the BrokerMetrics GetSnapshot method.
func TestBrokerMetrics(t *testing.T) {
	// Test with nil metrics
	var nilMetrics *BrokerMetrics
	snapshot := nilMetrics.GetSnapshot()
	if snapshot == nil {
		t.Error("Expected non-nil snapshot for nil metrics")
	}

	// Test with metrics
	original := &BrokerMetrics{
		ConnectionStatus:  "connected",
		ConnectionUptime:  time.Hour,
		PublishersCreated: 5,
		ActivePublishers:  3,
		MessagesPublished: 1000,
		Extended: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	snapshot = original.GetSnapshot()
	if snapshot == original {
		t.Error("Expected snapshot to be a different instance")
	}

	// Verify fields are copied correctly
	if snapshot.ConnectionStatus != original.ConnectionStatus {
		t.Errorf("Expected connection status %v, got %v", original.ConnectionStatus, snapshot.ConnectionStatus)
	}

	if snapshot.MessagesPublished != original.MessagesPublished {
		t.Errorf("Expected messages published %v, got %v", original.MessagesPublished, snapshot.MessagesPublished)
	}

	// Verify Extended map is deep copied
	if original.Extended != nil && snapshot.Extended == nil {
		t.Error("Expected Extended map to be copied")
	}

	if len(snapshot.Extended) != len(original.Extended) {
		t.Errorf("Expected Extended map length %v, got %v", len(original.Extended), len(snapshot.Extended))
	}

	for k, v := range original.Extended {
		if snapshot.Extended[k] != v {
			t.Errorf("Expected Extended[%s] = %v, got %v", k, v, snapshot.Extended[k])
		}
	}

	// Test with nil Extended map
	original.Extended = nil
	snapshot = original.GetSnapshot()
	if snapshot.Extended != nil {
		t.Error("Expected nil Extended map in snapshot")
	}
}

// TestPredefinedErrors tests the predefined error variables.
func TestPredefinedErrors(t *testing.T) {
	// Test ErrBrokerNotConnected
	if ErrBrokerNotConnected.Code != ErrConnectionFailed {
		t.Errorf("Expected ErrBrokerNotConnected to have code %v, got %v", ErrConnectionFailed, ErrBrokerNotConnected.Code)
	}

	if !ErrBrokerNotConnected.Retryable {
		t.Error("Expected ErrBrokerNotConnected to be retryable")
	}

	// Test ErrPublisherAlreadyClosed
	if ErrPublisherAlreadyClosed.Code != ErrPublisherClosed {
		t.Errorf("Expected ErrPublisherAlreadyClosed to have code %v, got %v", ErrPublisherClosed, ErrPublisherAlreadyClosed.Code)
	}

	if ErrPublisherAlreadyClosed.Retryable {
		t.Error("Expected ErrPublisherAlreadyClosed to not be retryable")
	}

	// Test ErrSubscriberAlreadyClosed
	if ErrSubscriberAlreadyClosed.Code != ErrSubscriberClosed {
		t.Errorf("Expected ErrSubscriberAlreadyClosed to have code %v, got %v", ErrSubscriberClosed, ErrSubscriberAlreadyClosed.Code)
	}

	if ErrSubscriberAlreadyClosed.Retryable {
		t.Error("Expected ErrSubscriberAlreadyClosed to not be retryable")
	}

	// Test ErrTransactionsNotSupported
	if ErrTransactionsNotSupported.Code != ErrTransactionNotSupported {
		t.Errorf("Expected ErrTransactionsNotSupported to have code %v, got %v", ErrTransactionNotSupported, ErrTransactionsNotSupported.Code)
	}

	if ErrTransactionsNotSupported.Retryable {
		t.Error("Expected ErrTransactionsNotSupported to not be retryable")
	}
}
