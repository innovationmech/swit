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
	"strings"
	"syscall"
	"time"
)

// ErrorClassification provides sophisticated error categorization for retry logic,
// circuit breakers, and error handling strategies.
type ErrorClassification string

const (
	// Transient errors - temporary issues that may resolve with retries
	ClassificationTransient ErrorClassification = "TRANSIENT"

	// Permanent errors - issues that will not resolve with retries
	ClassificationPermanent ErrorClassification = "PERMANENT"

	// Resource errors - capacity or quota related issues
	ClassificationResource ErrorClassification = "RESOURCE"

	// Configuration errors - setup or configuration issues
	ClassificationConfig ErrorClassification = "CONFIG"

	// Client errors - issues caused by incorrect client usage
	ClassificationClient ErrorClassification = "CLIENT"

	// Infrastructure errors - underlying system or network issues
	ClassificationInfrastructure ErrorClassification = "INFRASTRUCTURE"
)

// RetryStrategy defines how errors should be handled in retry logic.
type RetryStrategy string

const (
	// Immediate retry with exponential backoff
	RetryStrategyExponential RetryStrategy = "EXPONENTIAL"

	// Fixed interval retries
	RetryStrategyFixed RetryStrategy = "FIXED"

	// Linear backoff retries
	RetryStrategyLinear RetryStrategy = "LINEAR"

	// No retries - immediate failure
	RetryStrategyNone RetryStrategy = "NONE"

	// Jittered exponential backoff
	RetryStrategyJittered RetryStrategy = "JITTERED"

	// Custom strategy (implementation-specific)
	RetryStrategyCustom RetryStrategy = "CUSTOM"
)

// ErrorClassificationRule defines how to classify specific error types.
type ErrorClassificationRule struct {
	// Pattern matching criteria
	ErrorType       MessagingErrorType `json:"error_type,omitempty"`
	ErrorCodes      []string           `json:"error_codes,omitempty"`
	MessagePatterns []string           `json:"message_patterns,omitempty"`
	CauseTypes      []string           `json:"cause_types,omitempty"`

	// Classification result
	Classification    ErrorClassification `json:"classification"`
	RetryStrategy     RetryStrategy       `json:"retry_strategy"`
	MaxRetries        int                 `json:"max_retries"`
	InitialDelay      time.Duration       `json:"initial_delay"`
	MaxDelay          time.Duration       `json:"max_delay"`
	BackoffMultiplier float64             `json:"backoff_multiplier"`

	// Circuit breaker settings
	CircuitBreakerEnabled   bool          `json:"circuit_breaker_enabled"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`

	// Priority for rule matching (higher priority rules are checked first)
	Priority int `json:"priority"`
}

// ErrorClassifier provides sophisticated error classification and retry strategy determination.
type ErrorClassifier struct {
	rules []ErrorClassificationRule
}

// NewErrorClassifier creates a new error classifier with default rules.
func NewErrorClassifier() *ErrorClassifier {
	classifier := &ErrorClassifier{}
	classifier.loadDefaultRules()
	return classifier
}

// loadDefaultRules loads the standard classification rules for common messaging errors.
func (ec *ErrorClassifier) loadDefaultRules() {
	ec.rules = []ErrorClassificationRule{
		// Connection errors - mostly transient
		{
			ErrorType:               ErrorTypeConnection,
			ErrorCodes:              []string{ErrCodeConnectionTimeout, ErrCodeConnectionLost},
			Classification:          ClassificationTransient,
			RetryStrategy:           RetryStrategyExponential,
			MaxRetries:              5,
			InitialDelay:            100 * time.Millisecond,
			MaxDelay:                30 * time.Second,
			BackoffMultiplier:       2.0,
			CircuitBreakerEnabled:   true,
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   60 * time.Second,
			Priority:                100,
		},
		{
			ErrorType:         ErrorTypeConnection,
			ErrorCodes:        []string{ErrCodeConnectionFailed, ErrCodeTLSHandshakeFailed},
			Classification:    ClassificationInfrastructure,
			RetryStrategy:     RetryStrategyExponential,
			MaxRetries:        3,
			InitialDelay:      500 * time.Millisecond,
			MaxDelay:          10 * time.Second,
			BackoffMultiplier: 2.0,
			Priority:          90,
		},

		// Authentication errors - mostly permanent
		{
			ErrorType:      ErrorTypeAuthentication,
			ErrorCodes:     []string{ErrCodeInvalidCredentials, ErrCodePermissionDenied},
			Classification: ClassificationPermanent,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       100,
		},
		{
			ErrorType:      ErrorTypeAuthentication,
			ErrorCodes:     []string{ErrCodeTokenExpired, ErrCodeAuthTimeout},
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     2,
			InitialDelay:   1 * time.Second,
			Priority:       90,
		},

		// Configuration errors - permanent
		{
			ErrorType:      ErrorTypeConfiguration,
			Classification: ClassificationConfig,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       100,
		},

		// Publishing errors - mixed strategies
		{
			ErrorType:         ErrorTypePublishing,
			ErrorCodes:        []string{ErrCodePublishTimeout, ErrCodeQuotaExceeded},
			Classification:    ClassificationTransient,
			RetryStrategy:     RetryStrategyJittered,
			MaxRetries:        3,
			InitialDelay:      200 * time.Millisecond,
			MaxDelay:          5 * time.Second,
			BackoffMultiplier: 1.5,
			Priority:          100,
		},
		{
			ErrorType:      ErrorTypePublishing,
			ErrorCodes:     []string{ErrCodeMessageTooLarge, ErrCodeTopicNotFound},
			Classification: ClassificationClient,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       90,
		},
		{
			ErrorType:         ErrorTypePublishing,
			ErrorCodes:        []string{ErrCodePublishFailed},
			Classification:    ClassificationTransient,
			RetryStrategy:     RetryStrategyExponential,
			MaxRetries:        4,
			InitialDelay:      100 * time.Millisecond,
			MaxDelay:          8 * time.Second,
			BackoffMultiplier: 2.0,
			Priority:          80,
		},

		// Serialization errors - permanent
		{
			ErrorType:      ErrorTypeSerialization,
			Classification: ClassificationPermanent,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       100,
		},

		// Subscription errors - mixed strategies
		{
			ErrorType:      ErrorTypeSubscription,
			ErrorCodes:     []string{ErrCodeRebalancing, ErrCodeOffsetCommitFailed},
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     3,
			InitialDelay:   500 * time.Millisecond,
			Priority:       100,
		},
		{
			ErrorType:      ErrorTypeSubscription,
			ErrorCodes:     []string{ErrCodeConsumerGroupError, ErrCodePartitionAssignment},
			Classification: ClassificationInfrastructure,
			RetryStrategy:  RetryStrategyLinear,
			MaxRetries:     2,
			InitialDelay:   1 * time.Second,
			Priority:       90,
		},

		// Processing errors - mostly permanent
		{
			ErrorType:      ErrorTypeProcessing,
			ErrorCodes:     []string{ErrCodeProcessingTimeout},
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     2,
			InitialDelay:   100 * time.Millisecond,
			Priority:       100,
		},
		{
			ErrorType:      ErrorTypeProcessing,
			ErrorCodes:     []string{ErrCodeInvalidMessage, ErrCodeMessageValidation},
			Classification: ClassificationPermanent,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       90,
		},
		{
			ErrorType:      ErrorTypeProcessing,
			ErrorCodes:     []string{ErrCodeHandlerPanic},
			Classification: ClassificationClient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     1,
			InitialDelay:   500 * time.Millisecond,
			Priority:       80,
		},

		// Resource errors - transient with backoff
		{
			ErrorType:               ErrorTypeResource,
			Classification:          ClassificationResource,
			RetryStrategy:           RetryStrategyJittered,
			MaxRetries:              5,
			InitialDelay:            1 * time.Second,
			MaxDelay:                30 * time.Second,
			BackoffMultiplier:       2.0,
			CircuitBreakerEnabled:   true,
			CircuitBreakerThreshold: 3,
			CircuitBreakerTimeout:   120 * time.Second,
			Priority:                100,
		},

		// Transaction errors - mixed strategies
		{
			ErrorType:         ErrorTypeTransaction,
			ErrorCodes:        []string{ErrCodeTransactionTimeout, ErrCodeTransactionConflict},
			Classification:    ClassificationTransient,
			RetryStrategy:     RetryStrategyExponential,
			MaxRetries:        3,
			InitialDelay:      200 * time.Millisecond,
			MaxDelay:          5 * time.Second,
			BackoffMultiplier: 1.8,
			Priority:          100,
		},
		{
			ErrorType:      ErrorTypeTransaction,
			ErrorCodes:     []string{ErrCodeTransactionNotSupported},
			Classification: ClassificationPermanent,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       90,
		},

		// Health check errors - transient
		{
			ErrorType:      ErrorTypeHealth,
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     3,
			InitialDelay:   2 * time.Second,
			Priority:       100,
		},

		// Internal errors - mixed strategies
		{
			ErrorType:      ErrorTypeInternal,
			ErrorCodes:     []string{ErrCodeNotImplemented},
			Classification: ClassificationPermanent,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
			Priority:       100,
		},
		{
			ErrorType:      ErrorTypeInternal,
			ErrorCodes:     []string{ErrCodeConcurrencyError, ErrCodeInvalidState},
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     2,
			InitialDelay:   100 * time.Millisecond,
			Priority:       90,
		},
	}
}

// ClassifyError analyzes an error and returns its classification and retry strategy.
func (ec *ErrorClassifier) ClassifyError(err error) ErrorClassificationResult {
	if err == nil {
		return ErrorClassificationResult{
			Classification: ClassificationTransient, // Default for nil errors
			RetryStrategy:  RetryStrategyNone,
		}
	}

	// Try to extract messaging error information
	var msgErr *BaseMessagingError
	if errors.As(err, &msgErr) {
		return ec.classifyMessagingError(msgErr)
	}

	// Fallback to generic error classification
	return ec.classifyGenericError(err)
}

// classifyMessagingError classifies structured messaging errors.
func (ec *ErrorClassifier) classifyMessagingError(err *BaseMessagingError) ErrorClassificationResult {
	// Find the best matching rule
	var bestRule *ErrorClassificationRule
	bestPriority := -1

	for _, rule := range ec.rules {
		if ec.ruleMatches(rule, err) && rule.Priority > bestPriority {
			bestRule = &rule
			bestPriority = rule.Priority
		}
	}

	if bestRule == nil {
		// No specific rule found, use defaults based on error type
		return ec.getDefaultClassification(err.Type, err.Retryable)
	}

	return ErrorClassificationResult{
		Classification:          bestRule.Classification,
		RetryStrategy:           bestRule.RetryStrategy,
		MaxRetries:              bestRule.MaxRetries,
		InitialDelay:            bestRule.InitialDelay,
		MaxDelay:                bestRule.MaxDelay,
		BackoffMultiplier:       bestRule.BackoffMultiplier,
		CircuitBreakerEnabled:   bestRule.CircuitBreakerEnabled,
		CircuitBreakerThreshold: bestRule.CircuitBreakerThreshold,
		CircuitBreakerTimeout:   bestRule.CircuitBreakerTimeout,
		MatchedRule:             bestRule,
	}
}

// ruleMatches checks if a rule matches the given error.
func (ec *ErrorClassifier) ruleMatches(rule ErrorClassificationRule, err *BaseMessagingError) bool {
	// Check error type
	if rule.ErrorType != "" && rule.ErrorType != err.Type {
		return false
	}

	// Check error codes
	if len(rule.ErrorCodes) > 0 {
		codeMatch := false
		for _, code := range rule.ErrorCodes {
			if code == err.Code {
				codeMatch = true
				break
			}
		}
		if !codeMatch {
			return false
		}
	}

	// Check message patterns
	if len(rule.MessagePatterns) > 0 {
		patternMatch := false
		for _, pattern := range rule.MessagePatterns {
			if strings.Contains(strings.ToLower(err.Message), strings.ToLower(pattern)) {
				patternMatch = true
				break
			}
		}
		if !patternMatch {
			return false
		}
	}

	// Check cause types
	if len(rule.CauseTypes) > 0 && err.Cause != nil {
		causeMatch := false
		causeType := getCauseErrorType(err.Cause)
		for _, expectedType := range rule.CauseTypes {
			if strings.Contains(causeType, expectedType) {
				causeMatch = true
				break
			}
		}
		if !causeMatch {
			return false
		}
	}

	return true
}

// classifyGenericError classifies non-messaging errors.
func (ec *ErrorClassifier) classifyGenericError(err error) ErrorClassificationResult {
	// Check for common Go error types
	errorStr := err.Error()

	// Network and connection errors
	if isNetworkError(err) {
		return ErrorClassificationResult{
			Classification:    ClassificationTransient,
			RetryStrategy:     RetryStrategyExponential,
			MaxRetries:        3,
			InitialDelay:      200 * time.Millisecond,
			MaxDelay:          10 * time.Second,
			BackoffMultiplier: 2.0,
		}
	}

	// Context errors
	if errors.Is(err, context.Canceled) {
		return ErrorClassificationResult{
			Classification: ClassificationPermanent,
			RetryStrategy:  RetryStrategyNone,
			MaxRetries:     0,
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorClassificationResult{
			Classification: ClassificationTransient,
			RetryStrategy:  RetryStrategyFixed,
			MaxRetries:     2,
			InitialDelay:   500 * time.Millisecond,
		}
	}

	// Resource errors based on common patterns
	if isResourceError(errorStr) {
		return ErrorClassificationResult{
			Classification:    ClassificationResource,
			RetryStrategy:     RetryStrategyJittered,
			MaxRetries:        5,
			InitialDelay:      1 * time.Second,
			MaxDelay:          30 * time.Second,
			BackoffMultiplier: 1.5,
		}
	}

	// Default classification for unknown errors
	return ErrorClassificationResult{
		Classification: ClassificationTransient,
		RetryStrategy:  RetryStrategyFixed,
		MaxRetries:     1,
		InitialDelay:   1 * time.Second,
	}
}

// getDefaultClassification provides fallback classification based on error type.
func (ec *ErrorClassifier) getDefaultClassification(errType MessagingErrorType, retryable bool) ErrorClassificationResult {
	var classification ErrorClassification
	var strategy RetryStrategy
	maxRetries := 0
	initialDelay := time.Second

	if retryable {
		classification = ClassificationTransient
		strategy = RetryStrategyFixed
		maxRetries = 1
	} else {
		classification = ClassificationPermanent
		strategy = RetryStrategyNone
	}

	// Adjust based on error type
	switch errType {
	case ErrorTypeConnection, ErrorTypeSubscription:
		if retryable {
			strategy = RetryStrategyExponential
			maxRetries = 3
			initialDelay = 200 * time.Millisecond
		}
	case ErrorTypeResource:
		classification = ClassificationResource
		if retryable {
			strategy = RetryStrategyJittered
			maxRetries = 5
			initialDelay = 500 * time.Millisecond
		}
	case ErrorTypeConfiguration:
		classification = ClassificationConfig
		strategy = RetryStrategyNone
		maxRetries = 0
	}

	return ErrorClassificationResult{
		Classification:    classification,
		RetryStrategy:     strategy,
		MaxRetries:        maxRetries,
		InitialDelay:      initialDelay,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// ErrorClassificationResult contains the result of error classification.
type ErrorClassificationResult struct {
	Classification          ErrorClassification      `json:"classification"`
	RetryStrategy           RetryStrategy            `json:"retry_strategy"`
	MaxRetries              int                      `json:"max_retries"`
	InitialDelay            time.Duration            `json:"initial_delay"`
	MaxDelay                time.Duration            `json:"max_delay"`
	BackoffMultiplier       float64                  `json:"backoff_multiplier"`
	CircuitBreakerEnabled   bool                     `json:"circuit_breaker_enabled"`
	CircuitBreakerThreshold int                      `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration            `json:"circuit_breaker_timeout"`
	MatchedRule             *ErrorClassificationRule `json:"matched_rule,omitempty"`
}

// ShouldRetry returns true if the error should be retried based on the classification.
func (r ErrorClassificationResult) ShouldRetry(attemptCount int) bool {
	return r.RetryStrategy != RetryStrategyNone && attemptCount < r.MaxRetries
}

// CalculateDelay calculates the delay for the next retry based on the strategy.
func (r ErrorClassificationResult) CalculateDelay(attemptCount int) time.Duration {
	if attemptCount <= 0 {
		return r.InitialDelay
	}

	var delay time.Duration

	switch r.RetryStrategy {
	case RetryStrategyFixed:
		delay = r.InitialDelay

	case RetryStrategyLinear:
		delay = time.Duration(attemptCount) * r.InitialDelay

	case RetryStrategyExponential:
		multiplier := 1.0
		for i := 0; i < attemptCount; i++ {
			multiplier *= r.BackoffMultiplier
		}
		delay = time.Duration(float64(r.InitialDelay) * multiplier)

	case RetryStrategyJittered:
		// Exponential backoff with jitter
		multiplier := 1.0
		for i := 0; i < attemptCount; i++ {
			multiplier *= r.BackoffMultiplier
		}
		baseDelay := time.Duration(float64(r.InitialDelay) * multiplier)
		// Add up to 25% jitter
		jitter := time.Duration(float64(baseDelay) * 0.25 * (0.5 - float64(time.Now().UnixNano()%1000)/1000.0))
		delay = baseDelay + jitter

	default:
		delay = r.InitialDelay
	}

	// Apply maximum delay limit
	if r.MaxDelay > 0 && delay > r.MaxDelay {
		delay = r.MaxDelay
	}

	return delay
}

// AddRule adds a custom classification rule.
func (ec *ErrorClassifier) AddRule(rule ErrorClassificationRule) {
	ec.rules = append(ec.rules, rule)
	// Keep rules sorted by priority (highest first)
	for i := len(ec.rules) - 1; i > 0; i-- {
		if ec.rules[i].Priority > ec.rules[i-1].Priority {
			ec.rules[i], ec.rules[i-1] = ec.rules[i-1], ec.rules[i]
		} else {
			break
		}
	}
}

// RemoveRule removes rules matching the given criteria.
func (ec *ErrorClassifier) RemoveRule(errorType MessagingErrorType, errorCodes []string) {
	newRules := make([]ErrorClassificationRule, 0, len(ec.rules))

	for _, rule := range ec.rules {
		shouldRemove := false

		if rule.ErrorType == errorType {
			if len(errorCodes) == 0 {
				shouldRemove = true
			} else {
				for _, code := range errorCodes {
					for _, ruleCode := range rule.ErrorCodes {
						if code == ruleCode {
							shouldRemove = true
							break
						}
					}
					if shouldRemove {
						break
					}
				}
			}
		}

		if !shouldRemove {
			newRules = append(newRules, rule)
		}
	}

	ec.rules = newRules
}

// Helper functions for error type detection

// isNetworkError checks if an error is network-related.
func isNetworkError(err error) bool {
	// Check for syscall errors
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		switch sysErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED,
			syscall.ETIMEDOUT, syscall.EHOSTUNREACH, syscall.ENETUNREACH:
			return true
		}
	}

	// Check error message patterns
	errorStr := strings.ToLower(err.Error())
	networkPatterns := []string{
		"connection refused", "connection reset", "connection timeout",
		"network unreachable", "host unreachable", "no route to host",
		"broken pipe", "connection aborted", "tcp", "udp", "dial",
		"network timeout", "timeout",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isResourceError checks if an error is resource-related.
func isResourceError(errorStr string) bool {
	errorStr = strings.ToLower(errorStr)
	resourcePatterns := []string{
		"too many", "limit exceeded", "quota exceeded", "out of memory",
		"disk full", "no space", "resource temporarily unavailable",
		"rate limit", "throttled", "backpressure",
	}

	for _, pattern := range resourcePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// getCauseErrorType extracts the error type string from an error for cause analysis.
func getCauseErrorType(err error) string {
	if err == nil {
		return ""
	}

	// Get the type name using reflection-like approach
	errorStr := err.Error()

	// For wrapped errors, try to get the underlying type
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return getCauseErrorType(unwrapped)
	}

	// Extract type information from error string patterns
	if strings.Contains(errorStr, "connection") {
		return "connection"
	}
	if strings.Contains(errorStr, "timeout") {
		return "timeout"
	}
	if strings.Contains(errorStr, "context") {
		return "context"
	}

	return "generic"
}
