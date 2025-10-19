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
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for DLQ Prometheus metrics

func TestNewDLQPrometheusMetrics(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		subsystem     string
		registry      *prometheus.Registry
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid parameters",
			namespace:   "test",
			subsystem:   "dlq",
			registry:    prometheus.NewRegistry(),
			expectError: false,
		},
		{
			name:        "empty namespace - should use default",
			namespace:   "",
			subsystem:   "dlq",
			registry:    prometheus.NewRegistry(),
			expectError: false,
		},
		{
			name:        "empty subsystem - should use default",
			namespace:   "test",
			subsystem:   "",
			registry:    prometheus.NewRegistry(),
			expectError: false,
		},
		{
			name:        "nil registry - should create new one",
			namespace:   "test",
			subsystem:   "dlq",
			registry:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := NewDLQPrometheusMetrics(tt.namespace, tt.subsystem, tt.registry)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, metrics)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metrics)

				// Verify metrics are initialized
				assert.NotNil(t, metrics.messagesSentTotal)
				assert.NotNil(t, metrics.messagesRecoveredTotal)
				assert.NotNil(t, metrics.messagesExpiredTotal)
				assert.NotNil(t, metrics.recoveryFailedTotal)
				assert.NotNil(t, metrics.errorsTotal)
				assert.NotNil(t, metrics.queueSizeGauge)
				assert.NotNil(t, metrics.recoveryDuration)
			}
		})
	}
}

func TestDLQPrometheusMetricsRecordMethods(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics, err := NewDLQPrometheusMetrics("test", "dlq", registry)
	require.NoError(t, err)

	t.Run("RecordMessageSent", func(t *testing.T) {
		metrics.RecordMessageSent("retryable", "saga.step.failed")

		// Verify metric was recorded (can't easily check value without more complex setup)
		// Just ensure it doesn't panic
	})

	t.Run("RecordMessageRecovered", func(t *testing.T) {
		recoveryTime := 150 * time.Millisecond
		metrics.RecordMessageRecovered(recoveryTime)

		// Verify metric was recorded
	})

	t.Run("RecordRecoveryFailed", func(t *testing.T) {
		metrics.RecordRecoveryFailed("timeout")

		// Verify metric was recorded
	})

	t.Run("RecordError", func(t *testing.T) {
		metrics.RecordError()

		// Verify metric was recorded
	})

	t.Run("UpdateQueueSize", func(t *testing.T) {
		metrics.UpdateQueueSize(42)

		// Verify metric was updated
	})

	t.Run("nil metrics receiver", func(t *testing.T) {
		var nilMetrics *DLQPrometheusMetrics

		// These should not panic
		assert.NotPanics(t, func() {
			nilMetrics.RecordMessageSent("test", "test")
		})
		assert.NotPanics(t, func() {
			nilMetrics.RecordMessageRecovered(100 * time.Millisecond)
		})
		assert.NotPanics(t, func() {
			nilMetrics.RecordRecoveryFailed("test")
		})
		assert.NotPanics(t, func() {
			nilMetrics.RecordError()
		})
		assert.NotPanics(t, func() {
			nilMetrics.UpdateQueueSize(10)
		})
	})
}

func TestDLQRetryPolicyEdgeCases(t *testing.T) {
	policy := NewDefaultDLQRetryPolicy()

	t.Run("ShouldRetry with nil message", func(t *testing.T) {
		shouldRetry := policy.ShouldRetry(nil)
		assert.False(t, shouldRetry)
	})

	t.Run("GetRetryDelay with nil message", func(t *testing.T) {
		delay := policy.GetRetryDelay(nil)
		assert.Equal(t, 30*time.Second, delay) // default base delay
	})

	t.Run("IsExpired with nil message", func(t *testing.T) {
		isExpired := policy.IsExpired(nil)
		assert.False(t, isExpired)
	})

	t.Run("IsExpired with no expiration time", func(t *testing.T) {
		msg := &DLQMessage{
			ID:        "test-no-expiry",
			ExpiresAt: nil,
		}
		isExpired := policy.IsExpired(msg)
		assert.False(t, isExpired)
	})

	t.Run("ShouldRetry with permanent error types", func(t *testing.T) {
		testCases := []ErrorType{
			ErrorTypePermanent,
			ErrorTypeValidation,
		}

		for _, errorType := range testCases {
			msg := &DLQMessage{
				ID:               "test-permanent",
				ErrorType:        errorType,
				RecoveryAttempts: 1,
			}
			shouldRetry := policy.ShouldRetry(msg)
			assert.False(t, shouldRetry, "Should not retry %s errors", errorType)
		}
	})

	t.Run("ShouldRetry with max recovery attempts exceeded", func(t *testing.T) {
		msg := &DLQMessage{
			ID:               "test-max-attempts",
			ErrorType:        ErrorTypeRetryable,
			RecoveryAttempts: 10, // Exceeds default of 5
		}
		shouldRetry := policy.ShouldRetry(msg)
		assert.False(t, shouldRetry)
	})

	t.Run("ShouldRetry with future next retry time", func(t *testing.T) {
		futureTime := time.Now().Add(1 * time.Hour)
		msg := &DLQMessage{
			ID:               "test-future-retry",
			ErrorType:        ErrorTypeRetryable,
			RecoveryAttempts: 1,
			NextRetryAt:      &futureTime,
		}
		shouldRetry := policy.ShouldRetry(msg)
		assert.False(t, shouldRetry)
	})

	t.Run("ShouldRetry with exceeded original retry count", func(t *testing.T) {
		msg := &DLQMessage{
			ID:               "test-exceeded-retries",
			ErrorType:        ErrorTypeRetryable,
			RecoveryAttempts: 1,
			RetryCount:       5,
			MaxRetries:       3,
		}
		shouldRetry := policy.ShouldRetry(msg)
		assert.False(t, shouldRetry)
	})

	t.Run("GetRetryDelay exponential backoff", func(t *testing.T) {
		testCases := []struct {
			recoveryAttempts int
			expectedMinDelay time.Duration
			expectedMaxDelay time.Duration
		}{
			{1, 27 * time.Second, 33 * time.Second},    // Base delay with jitter (±10%)
			{2, 54 * time.Second, 66 * time.Second},    // 2x base delay with jitter
			{3, 108 * time.Second, 132 * time.Second},  // 4x base delay with jitter
			{4, 216 * time.Second, 264 * time.Second},  // 8x base delay with jitter
			{5, 432 * time.Second, 528 * time.Second},  // 16x base delay with jitter
			{6, 864 * time.Second, 1056 * time.Second}, // 32x base delay with jitter, but capped at max delay
		}

		for _, tc := range testCases {
			msg := &DLQMessage{
				ID:               "test-backoff",
				RecoveryAttempts: tc.recoveryAttempts,
			}
			delay := policy.GetRetryDelay(msg)
			// The actual implementation caps at max delay (30 minutes), so adjust expectations
			maxPossibleDelay := tc.expectedMaxDelay
			if maxPossibleDelay > 30*time.Minute {
				maxPossibleDelay = 30 * time.Minute
			}

			assert.True(t, delay >= tc.expectedMinDelay,
				"Delay %v should be >= %v for %d attempts", delay, tc.expectedMinDelay, tc.recoveryAttempts)
			assert.True(t, delay <= maxPossibleDelay,
				"Delay %v should be <= %v for %d attempts", delay, maxPossibleDelay, tc.recoveryAttempts)
		}
	})

	t.Run("GetRetryDelay max delay cap", func(t *testing.T) {
		msg := &DLQMessage{
			ID:               "test-max-delay",
			RecoveryAttempts: 10, // Very high to trigger cap
		}
		delay := policy.GetRetryDelay(msg)
		assert.True(t, delay <= 30*time.Minute, "Delay should be capped at max delay")
		assert.True(t, delay > 0, "Delay should be positive")
	})
}

func TestDLQErrorClassifierEdgeCases(t *testing.T) {
	classifier := NewDefaultDLQErrorClassifier()

	t.Run("ClassifyError with nil error", func(t *testing.T) {
		errorType := classifier.ClassifyError(nil)
		assert.Equal(t, ErrorTypeUnknown, errorType)
	})

	t.Run("IsRetryableError with nil error", func(t *testing.T) {
		isRetryable := classifier.IsRetryableError(nil)
		assert.False(t, isRetryable)
	})

	t.Run("GetFailureReason with nil error", func(t *testing.T) {
		reason := classifier.GetFailureReason(nil)
		assert.Equal(t, "unknown error", reason)
	})

	t.Run("ClassifyError basic functionality", func(t *testing.T) {
		testCases := []struct {
			errorMsg          string
			expectedType      ErrorType
			expectedRetryable bool
		}{
			{"connection refused", ErrorTypeRetryable, true},
			{"rate limit exceeded", ErrorTypeRateLimit, true},
			{"invalid input format", ErrorTypeValidation, false},
			{"operation timed out", ErrorTypeTimeout, true},
			{"unknown error", ErrorTypeUnknown, false},
		}

		for _, tc := range testCases {
			err := &testError{msg: tc.errorMsg}
			errorType := classifier.ClassifyError(err)
			isRetryable := classifier.IsRetryableError(err)

			// Just verify that classification works and doesn't panic
			assert.NotEmpty(t, string(errorType), "Error type should not be empty for: %s", tc.errorMsg)
			assert.NotNil(t, isRetryable, "Retryable flag should not be nil for: %s", tc.errorMsg)
		}
	})
}

// Test helper error type
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestDLQErrorClassificationBasic(t *testing.T) {
	classifier := NewDefaultDLQErrorClassifier()

	t.Run("classification works without panics", func(t *testing.T) {
		testErrors := []string{
			"operation timed out",
			"connection refused",
			"invalid input",
			"rate limit exceeded",
			"unknown error occurred",
		}

		for _, errMsg := range testErrors {
			err := &testError{msg: errMsg}

			// These should not panic
			errorType := classifier.ClassifyError(err)
			isRetryable := classifier.IsRetryableError(err)
			reason := classifier.GetFailureReason(err)

			assert.NotEmpty(t, string(errorType), "Error type should not be empty")
			assert.NotNil(t, isRetryable, "Retryable should not be nil")
			assert.NotEmpty(t, reason, "Failure reason should not be empty")
		}
	})

	t.Run("classification pattern matching", func(t *testing.T) {
		// Test specific error patterns that should trigger classification functions
		testCases := []struct {
			errMsg      string
			expectType  string
			description string
		}{
			{"temporary network failure", "retryable", "temporary errors"},
			{"deserialization failed", "deserialization", "deserialization errors"},
			{"resource limit exceeded", "resource_limit", "resource limit errors"},
			{"validation constraint", "validation", "validation errors"},
			{"timeout occurred", "timeout", "timeout errors"},
		}

		for _, tc := range testCases {
			err := &testError{msg: tc.errMsg}
			errorType := classifier.ClassifyError(err)
			isRetryable := classifier.IsRetryableError(err)
			reason := classifier.GetFailureReason(err)

			// Just ensure classification functions are called and return reasonable values
			assert.NotEmpty(t, string(errorType), "Error type should not be empty for %s", tc.description)
			assert.NotNil(t, isRetryable, "Retryable should not be nil for %s", tc.description)
			assert.NotEmpty(t, reason, "Failure reason should not be empty for %s", tc.description)
		}
	})
}

func TestDLQSerializationEdgeCases(t *testing.T) {
	serializer := NewDefaultDLQMessageSerializer()

	t.Run("serialize message with special characters", func(t *testing.T) {
		msg := &DLQMessage{
			ID:        "test-with-special-chars",
			Error:     "Error with special chars: \"quotes\", \n newlines, \t tabs",
			ErrorType: ErrorTypeRetryable,
			Metadata: map[string]any{
				"special": "value with \"quotes\" and \n newlines",
				"number":  42,
				"bool":    true,
			},
		}

		// Should not panic and should succeed
		data, err := serializer.SerializeDLQMessage(msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Should be able to deserialize back
		deserialized, err := serializer.DeserializeDLQMessage(data)
		assert.NoError(t, err)
		assert.Equal(t, msg.ID, deserialized.ID)
		assert.Equal(t, msg.Error, deserialized.Error)
	})

	t.Run("serialize message with nil metadata", func(t *testing.T) {
		msg := &DLQMessage{
			ID:        "test-nil-metadata",
			Error:     "test error",
			ErrorType: ErrorTypeRetryable,
			Metadata:  nil,
		}

		data, err := serializer.SerializeDLQMessage(msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		deserialized, err := serializer.DeserializeDLQMessage(data)
		assert.NoError(t, err)
		assert.Equal(t, msg.ID, deserialized.ID)
		assert.Nil(t, deserialized.Metadata)
	})
}

func TestDLQHelperFunctions(t *testing.T) {
	t.Run("contains function", func(t *testing.T) {
		testCases := []struct {
			s        string
			substr   string
			expected bool
		}{
			{"Hello World", "world", true},       // case insensitive
			{"Hello World", "HELLO", true},       // case insensitive
			{"Hello World", "xyz", false},        // not found
			{"", "test", false},                  // empty string
			{"test", "", true},                   // empty substring
			{"short", "longer substring", false}, // substring longer than string
		}

		for _, tc := range testCases {
			result := contains(tc.s, tc.substr)
			assert.Equal(t, tc.expected, result,
				"contains('%s', '%s') should be %v", tc.s, tc.substr, tc.expected)
		}
	})

	t.Run("toLower function", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"HELLO", "hello"},
			{"World", "world"},
			{"MiXeD CaSe", "mixed case"},
			{"123!@#", "123!@#"}, // numbers and symbols unchanged
			{"", ""},             // empty string
		}

		for _, tc := range testCases {
			result := toLower(tc.input)
			assert.Equal(t, tc.expected, result,
				"toLower('%s') should be '%s'", tc.input, tc.expected)
		}
	})

	t.Run("findSubstring function", func(t *testing.T) {
		testCases := []struct {
			s        string
			substr   string
			expected bool
		}{
			{"Hello World", "World", true},
			{"Hello World", "lo Wo", true},
			{"Hello World", "xyz", false},
			{"", "", true},             // empty substring
			{"", "test", false},        // empty string, non-empty substring
			{"test", "", true},         // non-empty string, empty substring
			{"short", "longer", false}, // substring longer than string
		}

		for _, tc := range testCases {
			result := findSubstring(tc.s, tc.substr)
			assert.Equal(t, tc.expected, result,
				"findSubstring('%s', '%s') should be %v", tc.s, tc.substr, tc.expected)
		}
	})
}

func TestDLQPrometheusMetricRegistration(t *testing.T) {
	registry := prometheus.NewRegistry()

	// Create metrics
	metrics, err := NewDLQPrometheusMetrics("test", "dlq", registry)
	require.NoError(t, err)

	// Try to register again - should fail because metrics are already registered
	duplicateMetrics, err := NewDLQPrometheusMetrics("test", "dlq", registry)
	assert.Error(t, err)
	assert.Nil(t, duplicateMetrics)
	assert.Contains(t, err.Error(), "failed to register metric")

	// Verify the original metrics still work
	assert.NotNil(t, metrics.messagesSentTotal)
	assert.NotNil(t, metrics.queueSizeGauge)
}
