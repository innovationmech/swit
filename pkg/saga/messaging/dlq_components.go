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
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Default DLQ component implementations

// defaultDLQMessageSerializer is the default implementation of DLQMessageSerializer.
type defaultDLQMessageSerializer struct{}

// NewDefaultDLQMessageSerializer creates a new default DLQ message serializer.
func NewDefaultDLQMessageSerializer() DLQMessageSerializer {
	return &defaultDLQMessageSerializer{}
}

// SerializeDLQMessage implements DLQMessageSerializer interface.
func (s *defaultDLQMessageSerializer) SerializeDLQMessage(msg *DLQMessage) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("DLQ message cannot be nil")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	return data, nil
}

// DeserializeDLQMessage implements DLQMessageSerializer interface.
func (s *defaultDLQMessageSerializer) DeserializeDLQMessage(data []byte) (*DLQMessage, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	var msg DLQMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ message: %w", err)
	}

	return &msg, nil
}

// defaultDLQRetryPolicy is the default implementation of DLQRetryPolicy.
type defaultDLQRetryPolicy struct {
	// maxRecoveryAttempts is the maximum number of recovery attempts.
	maxRecoveryAttempts int

	// baseRetryDelay is the base delay between retry attempts.
	baseRetryDelay time.Duration

	// maxRetryDelay is the maximum delay between retry attempts.
	maxRetryDelay time.Duration

	// backoffMultiplier is the multiplier for exponential backoff.
	backoffMultiplier float64
}

// NewDefaultDLQRetryPolicy creates a new default DLQ retry policy with sensible defaults.
func NewDefaultDLQRetryPolicy() DLQRetryPolicy {
	return &defaultDLQRetryPolicy{
		maxRecoveryAttempts: 5,
		baseRetryDelay:      30 * time.Second,
		maxRetryDelay:       30 * time.Minute,
		backoffMultiplier:   2.0,
	}
}

// NewDLQRetryPolicy creates a DLQ retry policy with custom parameters.
func NewDLQRetryPolicy(maxRecoveryAttempts int, baseDelay, maxDelay time.Duration, multiplier float64) DLQRetryPolicy {
	if maxRecoveryAttempts <= 0 {
		maxRecoveryAttempts = 5
	}
	if baseDelay <= 0 {
		baseDelay = 30 * time.Second
	}
	if maxDelay <= 0 {
		maxDelay = 30 * time.Minute
	}
	if multiplier <= 1.0 {
		multiplier = 2.0
	}

	return &defaultDLQRetryPolicy{
		maxRecoveryAttempts: maxRecoveryAttempts,
		baseRetryDelay:      baseDelay,
		maxRetryDelay:       maxDelay,
		backoffMultiplier:   multiplier,
	}
}

// ShouldRetry implements DLQRetryPolicy interface.
func (p *defaultDLQRetryPolicy) ShouldRetry(msg *DLQMessage) bool {
	if msg == nil {
		return false
	}

	// Check if expired first
	if p.IsExpired(msg) {
		return false
	}

	// Don't retry permanent errors
	if msg.ErrorType == ErrorTypePermanent || msg.ErrorType == ErrorTypeValidation {
		return false
	}

	// Check if we've exceeded recovery attempts
	if msg.RecoveryAttempts >= p.maxRecoveryAttempts {
		return false
	}

	// Check if it's time to retry
	if msg.NextRetryAt != nil && time.Now().Before(*msg.NextRetryAt) {
		return false
	}

	// Check if original retry count was exceeded
	if msg.RetryCount >= msg.MaxRetries {
		return false
	}

	return true
}

// GetRetryDelay implements DLQRetryPolicy interface.
func (p *defaultDLQRetryPolicy) GetRetryDelay(msg *DLQMessage) time.Duration {
	if msg == nil {
		return p.baseRetryDelay
	}

	// Use exponential backoff based on recovery attempts
	exponentialFactor := 1.0
	if msg.RecoveryAttempts > 1 {
		// Simpler exponential: 2^(attempts-1) but capped
		power := msg.RecoveryAttempts - 1
		if power > 5 { // Cap at 2^5 = 32 to avoid huge delays
			power = 5
		}
		exponentialFactor = float64(int(1) << power) // 2^power
	}
	delay := time.Duration(float64(p.baseRetryDelay) * exponentialFactor)

	// Cap at maximum delay
	if delay > p.maxRetryDelay {
		delay = p.maxRetryDelay
	}

	// Add jitter to prevent thundering herd
	jitter := time.Duration(float64(delay) * 0.1 * (0.5 - float64(time.Now().UnixNano()%1000)/1000.0))
	delay += jitter

	if delay < 0 {
		delay = p.baseRetryDelay
	}

	return delay
}

// IsExpired implements DLQRetryPolicy interface.
func (p *defaultDLQRetryPolicy) IsExpired(msg *DLQMessage) bool {
	if msg == nil || msg.ExpiresAt == nil {
		return false
	}

	return time.Now().After(*msg.ExpiresAt)
}

// GetMaxRecoveryAttempts implements DLQRetryPolicy interface.
func (p *defaultDLQRetryPolicy) GetMaxRecoveryAttempts() int {
	return p.maxRecoveryAttempts
}

// defaultDLQErrorClassifier is the default implementation of DLQErrorClassifier.
type defaultDLQErrorClassifier struct {
	// retryableErrors is a map of retryable error patterns.
	retryableErrors map[string]bool

	// permanentErrors is a map of permanent error patterns.
	permanentErrors map[string]bool
}

// NewDefaultDLQErrorClassifier creates a new default DLQ error classifier.
func NewDefaultDLQErrorClassifier() DLQErrorClassifier {
	classifier := &defaultDLQErrorClassifier{
		retryableErrors: make(map[string]bool),
		permanentErrors: make(map[string]bool),
	}

	// Initialize common error patterns
	classifier.initializeErrorPatterns()

	return classifier
}

// NewDLQErrorClassifier creates a DLQ error classifier with custom error patterns.
func NewDLQErrorClassifier(retryableErrors, permanentErrors []string) DLQErrorClassifier {
	classifier := &defaultDLQErrorClassifier{
		retryableErrors: make(map[string]bool),
		permanentErrors: make(map[string]bool),
	}

	// Add custom retryable errors
	for _, pattern := range retryableErrors {
		classifier.retryableErrors[pattern] = true
	}

	// Add custom permanent errors
	for _, pattern := range permanentErrors {
		classifier.permanentErrors[pattern] = true
	}

	// Initialize default patterns if not provided
	if len(retryableErrors) == 0 {
		classifier.initializeRetryablePatterns()
	}
	if len(permanentErrors) == 0 {
		classifier.initializePermanentPatterns()
	}

	return classifier
}

// initializeErrorPatterns initializes the default error patterns.
func (c *defaultDLQErrorClassifier) initializeErrorPatterns() {
	c.initializeRetryablePatterns()
	c.initializePermanentPatterns()
}

// initializeRetryablePatterns initializes common retryable error patterns.
func (c *defaultDLQErrorClassifier) initializeRetryablePatterns() {
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"network",
		"temporary",
		"rate limit",
		"service unavailable",
		"deadline exceeded",
		"resource temporarily unavailable",
		"too many connections",
		"connection reset",
		"connection timed out",
		"read timed out",
		"write timed out",
		"temporary failure",
		"transient",
		"throttled",
		"overloaded",
		"busy",
		"unavailable",
	}

	for _, pattern := range retryablePatterns {
		c.retryableErrors[pattern] = true
	}
}

// initializePermanentPatterns initializes common permanent error patterns.
func (c *defaultDLQErrorClassifier) initializePermanentPatterns() {
	permanentPatterns := []string{
		"invalid",
		"not found",
		"unauthorized",
		"forbidden",
		"authentication",
		"permission",
		"validation",
		"malformed",
		"syntax",
		"unsupported",
		"conflict",
		"duplicate",
		"constraint",
		"foreign key",
		"unique constraint",
		"null value",
		"invalid argument",
		"bad request",
		"not implemented",
		"method not allowed",
		"not acceptable",
		"gone",
		"length required",
		"precondition failed",
		"request entity too large",
		"uri too long",
		"unsupported media type",
		"requested range not satisfiable",
		"expectation failed",
		"upgrade required",
		"precondition required",
		"too many requests",
		"request header fields too large",
	}

	for _, pattern := range permanentPatterns {
		c.permanentErrors[pattern] = true
	}
}

// ClassifyError implements DLQErrorClassifier interface.
func (c *defaultDLQErrorClassifier) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	errMsg := err.Error()

	// Check for specific error patterns
	for pattern := range c.permanentErrors {
		if contains(errMsg, pattern) {
			return c.classifyByPattern(pattern)
		}
	}

	for pattern := range c.retryableErrors {
		if contains(errMsg, pattern) {
			return c.classifyByPattern(pattern)
		}
	}

	// Check error type
	switch {
	case isTimeoutError(err):
		return ErrorTypeTimeout
	case isRateLimitError(err):
		return ErrorTypeRateLimit
	case isResourceLimitError(err):
		return ErrorTypeResourceLimit
	case isValidationError(err):
		return ErrorTypeValidation
	case isDeserializationError(err):
		return ErrorTypeDeserialization
	default:
		return ErrorTypeUnknown
	}
}

// IsRetryableError implements DLQErrorClassifier interface.
func (c *defaultDLQErrorClassifier) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorType := c.ClassifyError(err)
	return errorType == ErrorTypeRetryable ||
		errorType == ErrorTypeTimeout ||
		errorType == ErrorTypeRateLimit ||
		errorType == ErrorTypeResourceLimit
}

// GetFailureReason implements DLQErrorClassifier interface.
func (c *defaultDLQErrorClassifier) GetFailureReason(err error) string {
	if err == nil {
		return "unknown error"
	}

	errMsg := err.Error()

	// Check for specific patterns to provide detailed reasons
	for pattern := range c.permanentErrors {
		if contains(errMsg, pattern) {
			return fmt.Sprintf("permanent error: %s", pattern)
		}
	}

	for pattern := range c.retryableErrors {
		if contains(errMsg, pattern) {
			return fmt.Sprintf("temporary error: %s", pattern)
		}
	}

	// Provide generic reasons based on error type
	switch {
	case isTimeoutError(err):
		return "operation timed out"
	case isRateLimitError(err):
		return "rate limit exceeded"
	case isResourceLimitError(err):
		return "resource limit exceeded"
	case isValidationError(err):
		return "validation failed"
	case isDeserializationError(err):
		return "message deserialization failed"
	default:
		return fmt.Sprintf("unknown error: %s", errMsg)
	}
}

// classifyByPattern classifies an error based on a known pattern.
func (c *defaultDLQErrorClassifier) classifyByPattern(pattern string) ErrorType {
	switch {
	case contains(pattern, "timeout"):
		return ErrorTypeTimeout
	case contains(pattern, "rate limit"), contains(pattern, "throttled"):
		return ErrorTypeRateLimit
	case contains(pattern, "resource"), contains(pattern, "overloaded"), contains(pattern, "busy"):
		return ErrorTypeResourceLimit
	case contains(pattern, "validation"), contains(pattern, "invalid"), contains(pattern, "malformed"):
		return ErrorTypeValidation
	case contains(pattern, "deserialization"), contains(pattern, "unmarshal"), contains(pattern, "decode"):
		return ErrorTypeDeserialization
	case contains(pattern, "temporary"), contains(pattern, "transient"), contains(pattern, "network"),
		contains(pattern, "connection"):
		return ErrorTypeRetryable
	default:
		return ErrorTypeUnknown
	}
}

// Helper functions for error classification

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	timeoutPatterns := []string{
		"timeout", "timed out", "deadline exceeded", "context deadline exceeded",
		"operation timed out", "connection timed out", "read timed out", "write timed out",
	}

	for _, pattern := range timeoutPatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isRateLimitError checks if an error is a rate limit error.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	rateLimitPatterns := []string{
		"rate limit", "rate-limit", "rate limited", "too many requests",
		"quota exceeded", "throttled", "throttling", "request limit",
	}

	for _, pattern := range rateLimitPatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isResourceLimitError checks if an error is a resource limit error.
func isResourceLimitError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	resourcePatterns := []string{
		"resource", "memory", "disk", "connection", "overloaded", "busy",
		"too many connections", "connection pool", "out of memory",
	}

	for _, pattern := range resourcePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isValidationError checks if an error is a validation error.
func isValidationError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	validationPatterns := []string{
		"validation", "invalid", "malformed", "syntax", "format",
		"required", "missing", "constraint", "violat",
	}

	for _, pattern := range validationPatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isDeserializationError checks if an error is a deserialization error.
func isDeserializationError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	deserializationPatterns := []string{
		"deserialize", "unmarshal", "marshal", "decode", "encode",
		"json", "protobuf", "parse", "syntax error", "invalid format",
	}

	for _, pattern := range deserializationPatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	// Simple case-insensitive contains check
	sLower := toLower(s)
	substrLower := toLower(substr)

	return findSubstring(sLower, substrLower)
}

// toLower converts a string to lowercase (simple implementation).
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// findSubstring finds a substring within a string.
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// DLQPrometheusMetrics handles Prometheus metrics for DLQ operations.
type DLQPrometheusMetrics struct {
	// Prometheus metrics collectors
	messagesSentTotal      *prometheus.CounterVec
	messagesRecoveredTotal *prometheus.CounterVec
	messagesExpiredTotal   *prometheus.CounterVec
	recoveryFailedTotal    *prometheus.CounterVec
	errorsTotal            *prometheus.CounterVec
	queueSizeGauge         prometheus.Gauge
	recoveryDuration       *prometheus.HistogramVec
}

// NewDLQPrometheusMetrics creates a new DLQ Prometheus metrics collector.
func NewDLQPrometheusMetrics(namespace, subsystem string, registry *prometheus.Registry) (*DLQPrometheusMetrics, error) {
	if namespace == "" {
		namespace = "saga"
	}
	if subsystem == "" {
		subsystem = "dlq"
	}
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	metrics := &DLQPrometheusMetrics{}

	// Initialize metrics
	metrics.messagesSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_sent_total",
			Help:      "Total number of messages sent to DLQ",
		},
		[]string{"error_type", "event_type"},
	)

	metrics.messagesRecoveredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_recovered_total",
			Help:      "Total number of messages recovered from DLQ",
		},
		[]string{"error_type"},
	)

	metrics.messagesExpiredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_expired_total",
			Help:      "Total number of messages expired in DLQ",
		},
		[]string{"error_type"},
	)

	metrics.recoveryFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "recovery_failed_total",
			Help:      "Total number of failed recovery attempts",
		},
		[]string{"error_type"},
	)

	metrics.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "errors_total",
			Help:      "Total number of DLQ handler errors",
		},
		[]string{"error_type"},
	)

	metrics.queueSizeGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_size",
			Help:      "Current size of the DLQ",
		},
	)

	metrics.recoveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "recovery_duration_seconds",
			Help:      "Duration of message recovery operations in seconds",
			Buckets:   []float64{0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0, 300.0},
		},
		[]string{"status"},
	)

	// Register all metrics
	metricsList := []prometheus.Collector{
		metrics.messagesSentTotal,
		metrics.messagesRecoveredTotal,
		metrics.messagesExpiredTotal,
		metrics.recoveryFailedTotal,
		metrics.errorsTotal,
		metrics.queueSizeGauge,
		metrics.recoveryDuration,
	}

	for _, metric := range metricsList {
		if err := registry.Register(metric); err != nil {
			return nil, fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return metrics, nil
}

// RecordMessageSent records a message being sent to DLQ.
func (m *DLQPrometheusMetrics) RecordMessageSent(errorType, eventType string) {
	if m != nil && m.messagesSentTotal != nil {
		m.messagesSentTotal.WithLabelValues(errorType, eventType).Inc()
	}
}

// RecordMessageRecovered records a message being recovered from DLQ.
func (m *DLQPrometheusMetrics) RecordMessageRecovered(recoveryTime time.Duration) {
	if m != nil {
		if m.messagesRecoveredTotal != nil {
			m.messagesRecoveredTotal.WithLabelValues("success").Inc()
		}
		if m.recoveryDuration != nil {
			m.recoveryDuration.WithLabelValues("success").Observe(recoveryTime.Seconds())
		}
	}
}

// RecordRecoveryFailed records a failed recovery attempt.
func (m *DLQPrometheusMetrics) RecordRecoveryFailed(errorType string) {
	if m != nil {
		if m.recoveryFailedTotal != nil {
			m.recoveryFailedTotal.WithLabelValues(errorType).Inc()
		}
		if m.recoveryDuration != nil {
			m.recoveryDuration.WithLabelValues("failed").Observe(0)
		}
	}
}

// RecordError records a DLQ handler error.
func (m *DLQPrometheusMetrics) RecordError() {
	if m != nil && m.errorsTotal != nil {
		m.errorsTotal.WithLabelValues("handler_error").Inc()
	}
}

// UpdateQueueSize updates the current queue size.
func (m *DLQPrometheusMetrics) UpdateQueueSize(size int64) {
	if m != nil && m.queueSizeGauge != nil {
		m.queueSizeGauge.Set(float64(size))
	}
}
