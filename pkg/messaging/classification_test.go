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
	"fmt"
	"syscall"
	"testing"
	"time"
)

func TestNewErrorClassifier(t *testing.T) {
	classifier := NewErrorClassifier()

	if classifier == nil {
		t.Fatal("Expected classifier to be non-nil")
	}

	if len(classifier.rules) == 0 {
		t.Error("Expected classifier to have default rules loaded")
	}
}

func TestErrorClassifier_ClassifyMessagingError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name             string
		error            *BaseMessagingError
		expectedClass    ErrorClassification
		expectedStrategy RetryStrategy
		expectedRetries  int
	}{
		{
			name: "connection timeout - transient",
			error: NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
				Message("connection timeout").
				Build(),
			expectedClass:    ClassificationTransient,
			expectedStrategy: RetryStrategyExponential,
			expectedRetries:  5,
		},
		{
			name: "authentication failed - permanent",
			error: NewError(ErrorTypeAuthentication, ErrCodeInvalidCredentials).
				Message("invalid credentials").
				Build(),
			expectedClass:    ClassificationPermanent,
			expectedStrategy: RetryStrategyNone,
			expectedRetries:  0,
		},
		{
			name: "configuration error - config",
			error: NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
				Message("invalid configuration").
				Build(),
			expectedClass:    ClassificationConfig,
			expectedStrategy: RetryStrategyNone,
			expectedRetries:  0,
		},
		{
			name: "publish timeout - transient",
			error: NewError(ErrorTypePublishing, ErrCodePublishTimeout).
				Message("publish timeout").
				Build(),
			expectedClass:    ClassificationTransient,
			expectedStrategy: RetryStrategyJittered,
			expectedRetries:  3,
		},
		{
			name: "message too large - client error",
			error: NewError(ErrorTypePublishing, ErrCodeMessageTooLarge).
				Message("message too large").
				Build(),
			expectedClass:    ClassificationClient,
			expectedStrategy: RetryStrategyNone,
			expectedRetries:  0,
		},
		{
			name: "resource exhausted - resource",
			error: NewError(ErrorTypeResource, ErrCodeResourceExhausted).
				Message("resource exhausted").
				Build(),
			expectedClass:    ClassificationResource,
			expectedStrategy: RetryStrategyJittered,
			expectedRetries:  5,
		},
		{
			name: "transaction timeout - transient",
			error: NewError(ErrorTypeTransaction, ErrCodeTransactionTimeout).
				Message("transaction timeout").
				Build(),
			expectedClass:    ClassificationTransient,
			expectedStrategy: RetryStrategyExponential,
			expectedRetries:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.error)

			if result.Classification != tt.expectedClass {
				t.Errorf("Expected classification %v, got %v", tt.expectedClass, result.Classification)
			}

			if result.RetryStrategy != tt.expectedStrategy {
				t.Errorf("Expected retry strategy %v, got %v", tt.expectedStrategy, result.RetryStrategy)
			}

			if result.MaxRetries != tt.expectedRetries {
				t.Errorf("Expected max retries %d, got %d", tt.expectedRetries, result.MaxRetries)
			}
		})
	}
}

func TestErrorClassifier_ClassifyGenericError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name             string
		error            error
		expectedClass    ErrorClassification
		expectedStrategy RetryStrategy
	}{
		{
			name:             "context canceled",
			error:            context.Canceled,
			expectedClass:    ClassificationPermanent,
			expectedStrategy: RetryStrategyNone,
		},
		{
			name:             "context deadline exceeded",
			error:            context.DeadlineExceeded,
			expectedClass:    ClassificationTransient,
			expectedStrategy: RetryStrategyFixed,
		},
		{
			name:             "network error",
			error:            syscall.ECONNREFUSED,
			expectedClass:    ClassificationTransient,
			expectedStrategy: RetryStrategyExponential,
		},
		{
			name:             "resource error",
			error:            fmt.Errorf("too many connections"),
			expectedClass:    ClassificationResource,
			expectedStrategy: RetryStrategyJittered,
		},
		{
			name:             "generic error",
			error:            fmt.Errorf("unknown error"),
			expectedClass:    ClassificationTransient,
			expectedStrategy: RetryStrategyFixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.error)

			if result.Classification != tt.expectedClass {
				t.Errorf("Expected classification %v, got %v", tt.expectedClass, result.Classification)
			}

			if result.RetryStrategy != tt.expectedStrategy {
				t.Errorf("Expected retry strategy %v, got %v", tt.expectedStrategy, result.RetryStrategy)
			}
		})
	}
}

func TestErrorClassifier_RuleMatching(t *testing.T) {
	classifier := NewErrorClassifier()

	// Test rule priority - more specific rules should match first
	connectionError := NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
		Message("connection timeout").
		Build()

	result := classifier.ClassifyError(connectionError)

	// Should match the timeout-specific rule, not the general connection rule
	if result.Classification != ClassificationTransient {
		t.Errorf("Expected classification %v, got %v", ClassificationTransient, result.Classification)
	}

	if result.RetryStrategy != RetryStrategyExponential {
		t.Errorf("Expected retry strategy %v, got %v", RetryStrategyExponential, result.RetryStrategy)
	}

	if result.MaxRetries != 5 {
		t.Errorf("Expected max retries %d, got %d", 5, result.MaxRetries)
	}
}

func TestErrorClassifier_AddRule(t *testing.T) {
	classifier := NewErrorClassifier()
	initialRuleCount := len(classifier.rules)

	// Add a custom rule
	customRule := ErrorClassificationRule{
		ErrorType:      ErrorTypePublishing,
		ErrorCodes:     []string{"CUSTOM_001"},
		Classification: ClassificationPermanent,
		RetryStrategy:  RetryStrategyNone,
		MaxRetries:     0,
		Priority:       1000, // High priority
	}

	classifier.AddRule(customRule)

	if len(classifier.rules) != initialRuleCount+1 {
		t.Errorf("Expected %d rules, got %d", initialRuleCount+1, len(classifier.rules))
	}

	// Test that the custom rule is applied
	customError := NewError(ErrorTypePublishing, "CUSTOM_001").
		Message("custom error").
		Build()

	result := classifier.ClassifyError(customError)

	if result.Classification != ClassificationPermanent {
		t.Errorf("Expected classification %v, got %v", ClassificationPermanent, result.Classification)
	}

	if result.RetryStrategy != RetryStrategyNone {
		t.Errorf("Expected retry strategy %v, got %v", RetryStrategyNone, result.RetryStrategy)
	}
}

func TestErrorClassifier_RemoveRule(t *testing.T) {
	classifier := NewErrorClassifier()

	// Count initial connection rules
	initialConnectionRules := 0
	for _, rule := range classifier.rules {
		if rule.ErrorType == ErrorTypeConnection {
			initialConnectionRules++
		}
	}

	// Remove all connection rules
	classifier.RemoveRule(ErrorTypeConnection, nil)

	// Count remaining connection rules
	remainingConnectionRules := 0
	for _, rule := range classifier.rules {
		if rule.ErrorType == ErrorTypeConnection {
			remainingConnectionRules++
		}
	}

	if remainingConnectionRules != 0 {
		t.Errorf("Expected 0 connection rules, got %d", remainingConnectionRules)
	}

	// Add a rule back and test specific removal
	testRule := ErrorClassificationRule{
		ErrorType:      ErrorTypeConnection,
		ErrorCodes:     []string{ErrCodeConnectionTimeout},
		Classification: ClassificationTransient,
		RetryStrategy:  RetryStrategyFixed,
		Priority:       100,
	}

	classifier.AddRule(testRule)
	classifier.RemoveRule(ErrorTypeConnection, []string{ErrCodeConnectionTimeout})

	// Should not match the removed rule
	timeoutError := NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
		Message("timeout").
		Build()

	result := classifier.ClassifyError(timeoutError)

	// Should fall back to default classification
	if result.Classification == ClassificationTransient && result.RetryStrategy == RetryStrategyFixed {
		t.Error("Rule was not properly removed")
	}
}

func TestErrorClassificationResult_ShouldRetry(t *testing.T) {
	tests := []struct {
		name         string
		result       ErrorClassificationResult
		attemptCount int
		expected     bool
	}{
		{
			name: "should retry - attempt within limit",
			result: ErrorClassificationResult{
				RetryStrategy: RetryStrategyExponential,
				MaxRetries:    3,
			},
			attemptCount: 2,
			expected:     true,
		},
		{
			name: "should not retry - attempt exceeds limit",
			result: ErrorClassificationResult{
				RetryStrategy: RetryStrategyExponential,
				MaxRetries:    3,
			},
			attemptCount: 3,
			expected:     false,
		},
		{
			name: "should not retry - no retry strategy",
			result: ErrorClassificationResult{
				RetryStrategy: RetryStrategyNone,
				MaxRetries:    3,
			},
			attemptCount: 1,
			expected:     false,
		},
		{
			name: "should not retry - zero max retries",
			result: ErrorClassificationResult{
				RetryStrategy: RetryStrategyFixed,
				MaxRetries:    0,
			},
			attemptCount: 0,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.result.ShouldRetry(tt.attemptCount)
			if actual != tt.expected {
				t.Errorf("ShouldRetry(%d) = %v, expected %v", tt.attemptCount, actual, tt.expected)
			}
		})
	}
}

func TestErrorClassificationResult_CalculateDelay(t *testing.T) {
	tests := []struct {
		name         string
		result       ErrorClassificationResult
		attemptCount int
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{
			name: "fixed delay",
			result: ErrorClassificationResult{
				RetryStrategy: RetryStrategyFixed,
				InitialDelay:  100 * time.Millisecond,
			},
			attemptCount: 3,
			expectedMin:  100 * time.Millisecond,
			expectedMax:  100 * time.Millisecond,
		},
		{
			name: "linear delay",
			result: ErrorClassificationResult{
				RetryStrategy: RetryStrategyLinear,
				InitialDelay:  100 * time.Millisecond,
			},
			attemptCount: 3,
			expectedMin:  300 * time.Millisecond,
			expectedMax:  300 * time.Millisecond,
		},
		{
			name: "exponential delay",
			result: ErrorClassificationResult{
				RetryStrategy:     RetryStrategyExponential,
				InitialDelay:      100 * time.Millisecond,
				BackoffMultiplier: 2.0,
			},
			attemptCount: 3,
			expectedMin:  800 * time.Millisecond, // 100 * 2^3
			expectedMax:  800 * time.Millisecond,
		},
		{
			name: "exponential delay with max",
			result: ErrorClassificationResult{
				RetryStrategy:     RetryStrategyExponential,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          500 * time.Millisecond,
				BackoffMultiplier: 2.0,
			},
			attemptCount: 5,
			expectedMin:  500 * time.Millisecond, // Capped at MaxDelay
			expectedMax:  500 * time.Millisecond,
		},
		{
			name: "jittered delay",
			result: ErrorClassificationResult{
				RetryStrategy:     RetryStrategyJittered,
				InitialDelay:      100 * time.Millisecond,
				BackoffMultiplier: 2.0,
			},
			attemptCount: 2,
			expectedMin:  300 * time.Millisecond, // Base 400ms minus 25% jitter
			expectedMax:  500 * time.Millisecond, // Base 400ms plus 25% jitter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := tt.result.CalculateDelay(tt.attemptCount)

			if tt.result.RetryStrategy == RetryStrategyJittered {
				// For jittered strategy, check range
				if delay < tt.expectedMin || delay > tt.expectedMax {
					t.Errorf("CalculateDelay(%d) = %v, expected between %v and %v",
						tt.attemptCount, delay, tt.expectedMin, tt.expectedMax)
				}
			} else {
				// For other strategies, check exact value
				if delay != tt.expectedMin {
					t.Errorf("CalculateDelay(%d) = %v, expected %v",
						tt.attemptCount, delay, tt.expectedMin)
				}
			}
		})
	}
}

func TestErrorClassificationResult_CalculateDelay_ZeroAttempt(t *testing.T) {
	result := ErrorClassificationResult{
		RetryStrategy: RetryStrategyExponential,
		InitialDelay:  200 * time.Millisecond,
	}

	delay := result.CalculateDelay(0)
	if delay != 200*time.Millisecond {
		t.Errorf("CalculateDelay(0) = %v, expected %v", delay, 200*time.Millisecond)
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected bool
	}{
		{
			name:     "connection refused",
			error:    syscall.ECONNREFUSED,
			expected: true,
		},
		{
			name:     "connection reset",
			error:    syscall.ECONNRESET,
			expected: true,
		},
		{
			name:     "timeout",
			error:    syscall.ETIMEDOUT,
			expected: true,
		},
		{
			name:     "host unreachable",
			error:    syscall.EHOSTUNREACH,
			expected: true,
		},
		{
			name:     "network unreachable",
			error:    syscall.ENETUNREACH,
			expected: true,
		},
		{
			name:     "connection refused in message",
			error:    fmt.Errorf("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "network timeout in message",
			error:    fmt.Errorf("network timeout occurred"),
			expected: true,
		},
		{
			name:     "tcp error in message",
			error:    fmt.Errorf("tcp connection failed"),
			expected: true,
		},
		{
			name:     "broken pipe in message",
			error:    fmt.Errorf("broken pipe"),
			expected: true,
		},
		{
			name:     "non-network error",
			error:    fmt.Errorf("file not found"),
			expected: false,
		},
		{
			name:     "generic error",
			error:    fmt.Errorf("something went wrong"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isNetworkError(tt.error)
			if actual != tt.expected {
				t.Errorf("isNetworkError(%v) = %v, expected %v", tt.error, actual, tt.expected)
			}
		})
	}
}

func TestIsResourceError(t *testing.T) {
	tests := []struct {
		name     string
		errorStr string
		expected bool
	}{
		{
			name:     "too many connections",
			errorStr: "too many connections",
			expected: true,
		},
		{
			name:     "limit exceeded",
			errorStr: "rate limit exceeded",
			expected: true,
		},
		{
			name:     "quota exceeded",
			errorStr: "quota exceeded",
			expected: true,
		},
		{
			name:     "out of memory",
			errorStr: "out of memory",
			expected: true,
		},
		{
			name:     "disk full",
			errorStr: "disk full",
			expected: true,
		},
		{
			name:     "no space",
			errorStr: "no space left on device",
			expected: true,
		},
		{
			name:     "resource temporarily unavailable",
			errorStr: "resource temporarily unavailable",
			expected: true,
		},
		{
			name:     "throttled",
			errorStr: "request throttled",
			expected: true,
		},
		{
			name:     "backpressure",
			errorStr: "backpressure detected",
			expected: true,
		},
		{
			name:     "non-resource error",
			errorStr: "file not found",
			expected: false,
		},
		{
			name:     "generic error",
			errorStr: "unknown error",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isResourceError(tt.errorStr)
			if actual != tt.expected {
				t.Errorf("isResourceError(%q) = %v, expected %v", tt.errorStr, actual, tt.expected)
			}
		})
	}
}

func TestGetCauseErrorType(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected string
	}{
		{
			name:     "nil error",
			error:    nil,
			expected: "",
		},
		{
			name:     "connection error",
			error:    fmt.Errorf("connection failed"),
			expected: "connection",
		},
		{
			name:     "timeout error",
			error:    fmt.Errorf("operation timeout"),
			expected: "timeout",
		},
		{
			name:     "context error",
			error:    context.Canceled,
			expected: "context",
		},
		{
			name:     "generic error",
			error:    fmt.Errorf("something went wrong"),
			expected: "generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getCauseErrorType(tt.error)
			if actual != tt.expected {
				t.Errorf("getCauseErrorType(%v) = %q, expected %q", tt.error, actual, tt.expected)
			}
		})
	}
}

func TestErrorClassifier_MessagePatternMatching(t *testing.T) {
	classifier := NewErrorClassifier()

	// Add a rule that matches message patterns
	customRule := ErrorClassificationRule{
		ErrorType:       ErrorTypeConnection,
		MessagePatterns: []string{"network", "connection refused"},
		Classification:  ClassificationInfrastructure,
		RetryStrategy:   RetryStrategyExponential,
		MaxRetries:      5,
		Priority:        200,
	}

	classifier.AddRule(customRule)

	// Test error that should match the pattern
	patternError := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("Network connection refused by server").
		Build()

	result := classifier.ClassifyError(patternError)

	if result.Classification != ClassificationInfrastructure {
		t.Errorf("Expected classification %v, got %v", ClassificationInfrastructure, result.Classification)
	}

	if result.RetryStrategy != RetryStrategyExponential {
		t.Errorf("Expected retry strategy %v, got %v", RetryStrategyExponential, result.RetryStrategy)
	}

	if result.MaxRetries != 5 {
		t.Errorf("Expected max retries %d, got %d", 5, result.MaxRetries)
	}
}

func TestErrorClassifier_CauseTypeMatching(t *testing.T) {
	classifier := NewErrorClassifier()

	// Add a rule that matches cause types
	customRule := ErrorClassificationRule{
		ErrorType:      ErrorTypePublishing,
		CauseTypes:     []string{"timeout", "connection"},
		Classification: ClassificationTransient,
		RetryStrategy:  RetryStrategyFixed,
		MaxRetries:     2,
		Priority:       150,
	}

	classifier.AddRule(customRule)

	// Test error with matching cause type
	causeError := fmt.Errorf("connection timeout occurred")
	publishError := NewError(ErrorTypePublishing, ErrCodePublishFailed).
		Message("Publish failed").
		Cause(causeError).
		Build()

	result := classifier.ClassifyError(publishError)

	if result.Classification != ClassificationTransient {
		t.Errorf("Expected classification %v, got %v", ClassificationTransient, result.Classification)
	}

	if result.RetryStrategy != RetryStrategyFixed {
		t.Errorf("Expected retry strategy %v, got %v", RetryStrategyFixed, result.RetryStrategy)
	}

	if result.MaxRetries != 2 {
		t.Errorf("Expected max retries %d, got %d", 2, result.MaxRetries)
	}
}

func TestErrorClassifier_CircuitBreakerSettings(t *testing.T) {
	classifier := NewErrorClassifier()

	// Test connection timeout which should have circuit breaker enabled
	timeoutError := NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
		Message("connection timeout").
		Build()

	result := classifier.ClassifyError(timeoutError)

	if !result.CircuitBreakerEnabled {
		t.Error("Expected circuit breaker to be enabled for connection timeouts")
	}

	if result.CircuitBreakerThreshold != 5 {
		t.Errorf("Expected circuit breaker threshold %d, got %d", 5, result.CircuitBreakerThreshold)
	}

	if result.CircuitBreakerTimeout != 60*time.Second {
		t.Errorf("Expected circuit breaker timeout %v, got %v", 60*time.Second, result.CircuitBreakerTimeout)
	}
}

// Benchmark tests
func BenchmarkErrorClassification(b *testing.B) {
	classifier := NewErrorClassifier()

	err := NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
		Message("connection timeout").
		Context(ErrorContext{
			Component: "test-component",
			Operation: "connect",
		}).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.ClassifyError(err)
	}
}

func BenchmarkCalculateDelay(b *testing.B) {
	result := ErrorClassificationResult{
		RetryStrategy:     RetryStrategyExponential,
		InitialDelay:      100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxDelay:          30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.CalculateDelay(i % 10)
	}
}

func BenchmarkRuleMatching(b *testing.B) {
	classifier := NewErrorClassifier()

	// Add many rules to test performance
	for i := 0; i < 100; i++ {
		classifier.AddRule(ErrorClassificationRule{
			ErrorType:      ErrorTypeProcessing,
			ErrorCodes:     []string{fmt.Sprintf("TEST_%03d", i)},
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			Priority:       i,
		})
	}

	err := NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
		Message("timeout").
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.ClassifyError(err)
	}
}
