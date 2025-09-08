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
	"context"
	"fmt"
	"strings"
	"time"
)

// MessagingErrorType represents the fundamental error categories in messaging systems.
// This classification helps with error handling strategies and retry logic.
type MessagingErrorType string

const (
	// Connection-related errors
	ErrorTypeConnection     MessagingErrorType = "CONNECTION"
	ErrorTypeAuthentication MessagingErrorType = "AUTHENTICATION"
	ErrorTypeAuthorization  MessagingErrorType = "AUTHORIZATION"

	// Configuration and validation errors
	ErrorTypeConfiguration MessagingErrorType = "CONFIGURATION"
	ErrorTypeValidation    MessagingErrorType = "VALIDATION"

	// Publishing errors
	ErrorTypePublishing    MessagingErrorType = "PUBLISHING"
	ErrorTypeSerialization MessagingErrorType = "SERIALIZATION"

	// Subscription and processing errors
	ErrorTypeSubscription MessagingErrorType = "SUBSCRIPTION"
	ErrorTypeProcessing   MessagingErrorType = "PROCESSING"

	// Transaction errors
	ErrorTypeTransaction MessagingErrorType = "TRANSACTION"

	// Resource and capacity errors
	ErrorTypeResource MessagingErrorType = "RESOURCE"
	ErrorTypeCapacity MessagingErrorType = "CAPACITY"

	// Health and monitoring errors
	ErrorTypeHealth     MessagingErrorType = "HEALTH"
	ErrorTypeMonitoring MessagingErrorType = "MONITORING"

	// Internal system errors
	ErrorTypeInternal     MessagingErrorType = "INTERNAL"
	ErrorTypeNotSupported MessagingErrorType = "NOT_SUPPORTED"
)

// ErrorSeverity indicates the impact level of an error.
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "CRITICAL" // System-wide impact, requires immediate attention
	SeverityHigh     ErrorSeverity = "HIGH"     // Service degradation, significant impact
	SeverityMedium   ErrorSeverity = "MEDIUM"   // Functional impact, manageable
	SeverityLow      ErrorSeverity = "LOW"      // Minor issues, low impact
	SeverityInfo     ErrorSeverity = "INFO"     // Informational, no functional impact
)

// ErrorContext provides additional contextual information for errors.
// This helps with debugging, monitoring, and creating actionable error messages.
type ErrorContext struct {
	// Component identifies which messaging component generated the error
	Component string `json:"component,omitempty"`

	// Operation specifies the operation that was being performed
	Operation string `json:"operation,omitempty"`

	// Topic/Queue identifies the messaging destination involved
	Topic string `json:"topic,omitempty"`
	Queue string `json:"queue,omitempty"`

	// Consumer group for subscription-related errors
	ConsumerGroup string `json:"consumer_group,omitempty"`

	// Message metadata for processing errors
	MessageID      string            `json:"message_id,omitempty"`
	MessageType    string            `json:"message_type,omitempty"`
	MessageHeaders map[string]string `json:"message_headers,omitempty"`

	// Connection details for connection-related errors
	BrokerEndpoints   []string `json:"broker_endpoints,omitempty"`
	ConnectedEndpoint string   `json:"connected_endpoint,omitempty"`

	// Timing information
	StartTime time.Time     `json:"start_time,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`

	// Additional custom context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BaseMessagingError is the fundamental error type that all messaging errors embed.
// It provides structured error information with context preservation and wrapping capabilities.
type BaseMessagingError struct {
	// Type categorizes the error for handling strategies
	Type MessagingErrorType `json:"type"`

	// Code provides specific error identification
	Code string `json:"code"`

	// Message is the human-readable error description
	Message string `json:"message"`

	// Severity indicates the impact level
	Severity ErrorSeverity `json:"severity"`

	// Retryable indicates if this error can be resolved by retrying
	Retryable bool `json:"retryable"`

	// Context provides additional error context
	Context ErrorContext `json:"context"`

	// Cause is the underlying error that caused this error
	Cause error `json:"-"`

	// Timestamp indicates when the error occurred
	Timestamp time.Time `json:"timestamp"`

	// Stack contains the call stack where the error occurred (optional)
	Stack string `json:"stack,omitempty"`

	// SuggestedAction provides actionable guidance for resolving the error
	SuggestedAction string `json:"suggested_action,omitempty"`

	// AdditionalDetails provides extra debugging information
	AdditionalDetails map[string]interface{} `json:"additional_details,omitempty"`
}

// Error implements the error interface.
func (e *BaseMessagingError) Error() string {
	var parts []string

	// Include type and code for identification
	parts = append(parts, fmt.Sprintf("[%s:%s]", e.Type, e.Code))

	// Add the main message
	parts = append(parts, e.Message)

	// Add context if available
	if e.Context.Component != "" {
		parts = append(parts, fmt.Sprintf("component=%s", e.Context.Component))
	}
	if e.Context.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Context.Operation))
	}
	if e.Context.Topic != "" {
		parts = append(parts, fmt.Sprintf("topic=%s", e.Context.Topic))
	}
	if e.Context.Queue != "" {
		parts = append(parts, fmt.Sprintf("queue=%s", e.Context.Queue))
	}

	errorStr := strings.Join(parts, " ")

	// Add cause if present
	if e.Cause != nil {
		errorStr += fmt.Sprintf(" (caused by: %v)", e.Cause)
	}

	return errorStr
}

// Unwrap implements the unwrapping interface for Go 1.13+ error handling.
func (e *BaseMessagingError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for Go 1.13+ error handling.
func (e *BaseMessagingError) Is(target error) bool {
	if t, ok := target.(*BaseMessagingError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// WithContext adds or updates error context and returns a new error.
func (e *BaseMessagingError) WithContext(ctx ErrorContext) *BaseMessagingError {
	newErr := *e

	// Merge contexts, with new context taking precedence
	if newErr.Context.Component == "" {
		newErr.Context.Component = ctx.Component
	}
	if newErr.Context.Operation == "" {
		newErr.Context.Operation = ctx.Operation
	}
	if newErr.Context.Topic == "" {
		newErr.Context.Topic = ctx.Topic
	}
	if newErr.Context.Queue == "" {
		newErr.Context.Queue = ctx.Queue
	}
	if newErr.Context.ConsumerGroup == "" {
		newErr.Context.ConsumerGroup = ctx.ConsumerGroup
	}
	if newErr.Context.MessageID == "" {
		newErr.Context.MessageID = ctx.MessageID
	}
	if newErr.Context.MessageType == "" {
		newErr.Context.MessageType = ctx.MessageType
	}
	if newErr.Context.ConnectedEndpoint == "" {
		newErr.Context.ConnectedEndpoint = ctx.ConnectedEndpoint
	}

	// Merge slices and maps
	if ctx.BrokerEndpoints != nil {
		newErr.Context.BrokerEndpoints = append(newErr.Context.BrokerEndpoints, ctx.BrokerEndpoints...)
	}
	if ctx.MessageHeaders != nil {
		if newErr.Context.MessageHeaders == nil {
			newErr.Context.MessageHeaders = make(map[string]string)
		}
		for k, v := range ctx.MessageHeaders {
			newErr.Context.MessageHeaders[k] = v
		}
	}
	if ctx.Metadata != nil {
		if newErr.Context.Metadata == nil {
			newErr.Context.Metadata = make(map[string]interface{})
		}
		for k, v := range ctx.Metadata {
			newErr.Context.Metadata[k] = v
		}
	}

	// Update timing if provided
	if !ctx.StartTime.IsZero() {
		newErr.Context.StartTime = ctx.StartTime
	}
	if ctx.Duration > 0 {
		newErr.Context.Duration = ctx.Duration
	}

	return &newErr
}

// WithCause wraps another error as the cause.
func (e *BaseMessagingError) WithCause(cause error) *BaseMessagingError {
	newErr := *e
	newErr.Cause = cause
	return &newErr
}

// WithSuggestedAction adds an actionable suggestion for resolving the error.
func (e *BaseMessagingError) WithSuggestedAction(action string) *BaseMessagingError {
	newErr := *e
	newErr.SuggestedAction = action
	return &newErr
}

// WithDetails adds additional debugging details.
func (e *BaseMessagingError) WithDetails(details map[string]interface{}) *BaseMessagingError {
	newErr := *e
	if newErr.AdditionalDetails == nil {
		newErr.AdditionalDetails = make(map[string]interface{})
	}
	for k, v := range details {
		newErr.AdditionalDetails[k] = v
	}
	return &newErr
}

// GetUserFacingMessage returns a simplified, user-friendly error message.
func (e *BaseMessagingError) GetUserFacingMessage() string {
	msg := e.Message
	if e.SuggestedAction != "" {
		msg += ". " + e.SuggestedAction
	}
	return msg
}

// GetDiagnosticInfo returns detailed diagnostic information for debugging.
func (e *BaseMessagingError) GetDiagnosticInfo() map[string]interface{} {
	info := map[string]interface{}{
		"type":      e.Type,
		"code":      e.Code,
		"message":   e.Message,
		"severity":  e.Severity,
		"retryable": e.Retryable,
		"timestamp": e.Timestamp,
		"context":   e.Context,
	}

	if e.Cause != nil {
		info["cause"] = e.Cause.Error()
	}

	if e.Stack != "" {
		info["stack"] = e.Stack
	}

	if e.SuggestedAction != "" {
		info["suggested_action"] = e.SuggestedAction
	}

	if e.AdditionalDetails != nil {
		info["additional_details"] = e.AdditionalDetails
	}

	return info
}

// ErrorBuilder provides a fluent interface for constructing messaging errors.
type ErrorBuilder struct {
	err *BaseMessagingError
}

// NewError creates a new error builder with the specified type and code.
func NewError(errType MessagingErrorType, code string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &BaseMessagingError{
			Type:      errType,
			Code:      code,
			Severity:  SeverityMedium, // Default severity
			Retryable: false,          // Default to non-retryable
			Timestamp: time.Now(),
		},
	}
}

// Message sets the error message.
func (b *ErrorBuilder) Message(msg string) *ErrorBuilder {
	b.err.Message = msg
	return b
}

// Messagef sets the error message using format string.
func (b *ErrorBuilder) Messagef(format string, args ...interface{}) *ErrorBuilder {
	b.err.Message = fmt.Sprintf(format, args...)
	return b
}

// Severity sets the error severity.
func (b *ErrorBuilder) Severity(severity ErrorSeverity) *ErrorBuilder {
	b.err.Severity = severity
	return b
}

// Retryable marks the error as retryable or not.
func (b *ErrorBuilder) Retryable(retryable bool) *ErrorBuilder {
	b.err.Retryable = retryable
	return b
}

// Context sets the error context.
func (b *ErrorBuilder) Context(ctx ErrorContext) *ErrorBuilder {
	b.err.Context = ctx
	return b
}

// Cause sets the underlying cause error.
func (b *ErrorBuilder) Cause(cause error) *ErrorBuilder {
	b.err.Cause = cause
	return b
}

// SuggestedAction sets the suggested action for resolving the error.
func (b *ErrorBuilder) SuggestedAction(action string) *ErrorBuilder {
	b.err.SuggestedAction = action
	return b
}

// Details adds additional debugging details.
func (b *ErrorBuilder) Details(details map[string]interface{}) *ErrorBuilder {
	if b.err.AdditionalDetails == nil {
		b.err.AdditionalDetails = make(map[string]interface{})
	}
	for k, v := range details {
		b.err.AdditionalDetails[k] = v
	}
	return b
}

// Component sets the component context.
func (b *ErrorBuilder) Component(component string) *ErrorBuilder {
	b.err.Context.Component = component
	return b
}

// Operation sets the operation context.
func (b *ErrorBuilder) Operation(operation string) *ErrorBuilder {
	b.err.Context.Operation = operation
	return b
}

// Topic sets the topic context.
func (b *ErrorBuilder) Topic(topic string) *ErrorBuilder {
	b.err.Context.Topic = topic
	return b
}

// Queue sets the queue context.
func (b *ErrorBuilder) Queue(queue string) *ErrorBuilder {
	b.err.Context.Queue = queue
	return b
}

// Build creates the final error.
func (b *ErrorBuilder) Build() *BaseMessagingError {
	return b.err
}

// Error creates the final error and returns it as an error interface.
func (b *ErrorBuilder) Error() error {
	return b.err
}

// Common error codes for consistent error identification
const (
	// Connection error codes
	ErrCodeConnectionFailed    = "CONN_001"
	ErrCodeConnectionTimeout   = "CONN_002"
	ErrCodeConnectionLost      = "CONN_003"
	ErrCodeDisconnectionFailed = "CONN_004"
	ErrCodeConnectionPoolFull  = "CONN_005"
	ErrCodeTLSHandshakeFailed  = "CONN_006"

	// Authentication error codes
	ErrCodeAuthFailed         = "AUTH_001"
	ErrCodeInvalidCredentials = "AUTH_002"
	ErrCodeTokenExpired       = "AUTH_003"
	ErrCodeAuthTimeout        = "AUTH_004"
	ErrCodePermissionDenied   = "AUTH_005"

	// Configuration error codes
	ErrCodeInvalidConfig       = "CONF_001"
	ErrCodeMissingConfig       = "CONF_002"
	ErrCodeConfigValidation    = "CONF_003"
	ErrCodeUnsupportedBroker   = "CONF_004"
	ErrCodeIncompatibleVersion = "CONF_005"

	// Publishing error codes
	ErrCodePublishFailed      = "PUB_001"
	ErrCodePublishTimeout     = "PUB_002"
	ErrCodeMessageTooLarge    = "PUB_003"
	ErrCodeTopicNotFound      = "PUB_004"
	ErrCodeBatchPublishFailed = "PUB_005"
	ErrCodePublisherClosed    = "PUB_006"
	ErrCodeQuotaExceeded      = "PUB_007"

	// Serialization error codes
	ErrCodeSerializationFailed   = "SER_001"
	ErrCodeDeserializationFailed = "SER_002"
	ErrCodeUnsupportedFormat     = "SER_003"
	ErrCodeSchemaValidation      = "SER_004"
	ErrCodeEncodingError         = "SER_005"

	// Subscription error codes
	ErrCodeSubscriptionFailed  = "SUB_001"
	ErrCodeConsumerGroupError  = "SUB_002"
	ErrCodeRebalancing         = "SUB_003"
	ErrCodeSubscriberClosed    = "SUB_004"
	ErrCodeUnsubscribeFailed   = "SUB_005"
	ErrCodeOffsetCommitFailed  = "SUB_006"
	ErrCodePartitionAssignment = "SUB_007"

	// Processing error codes
	ErrCodeProcessingFailed  = "PROC_001"
	ErrCodeProcessingTimeout = "PROC_002"
	ErrCodeInvalidMessage    = "PROC_003"
	ErrCodeMessageValidation = "PROC_004"
	ErrCodeMiddlewareFailure = "PROC_005"
	ErrCodeHandlerPanic      = "PROC_006"
	ErrCodeDeadLetter        = "PROC_007"

	// Transaction error codes
	ErrCodeTransactionFailed       = "TXN_001"
	ErrCodeTransactionAborted      = "TXN_002"
	ErrCodeTransactionTimeout      = "TXN_003"
	ErrCodeTransactionNotSupported = "TXN_004"
	ErrCodeTransactionConflict     = "TXN_005"

	// Resource error codes
	ErrCodeResourceExhausted   = "RES_001"
	ErrCodeRateLimitExceeded   = "RES_002"
	ErrCodeQueueFull           = "RES_003"
	ErrCodeMemoryLimitExceeded = "RES_004"
	ErrCodeDiskSpaceFull       = "RES_005"

	// Health check error codes
	ErrCodeHealthCheckFailed  = "HEALTH_001"
	ErrCodeHealthCheckTimeout = "HEALTH_002"
	ErrCodeServiceUnavailable = "HEALTH_003"

	// Seeking and positioning error codes
	ErrCodeSeekFailed       = "SEEK_001"
	ErrCodeSeekNotSupported = "SEEK_002"
	ErrCodeInvalidPosition  = "SEEK_003"

	// Internal error codes
	ErrCodeInternal         = "INT_001"
	ErrCodeNotImplemented   = "INT_002"
	ErrCodeOperationAborted = "INT_003"
	ErrCodeInvalidState     = "INT_004"
	ErrCodeConcurrencyError = "INT_005"
)

// Error aggregation utilities for handling multiple errors

// ErrorGroup collects multiple errors and provides aggregated error information.
type ErrorGroup struct {
	errors []error
	ctx    context.Context
}

// NewErrorGroup creates a new error group.
func NewErrorGroup(ctx context.Context) *ErrorGroup {
	return &ErrorGroup{
		ctx: ctx,
	}
}

// Add adds an error to the group.
func (eg *ErrorGroup) Add(err error) {
	if err != nil {
		eg.errors = append(eg.errors, err)
	}
}

// HasErrors returns true if the group contains any errors.
func (eg *ErrorGroup) HasErrors() bool {
	return len(eg.errors) > 0
}

// Errors returns all errors in the group.
func (eg *ErrorGroup) Errors() []error {
	return eg.errors
}

// FirstError returns the first error in the group, or nil if no errors.
func (eg *ErrorGroup) FirstError() error {
	if len(eg.errors) == 0 {
		return nil
	}
	return eg.errors[0]
}

// Error implements the error interface by combining all error messages.
func (eg *ErrorGroup) Error() string {
	if len(eg.errors) == 0 {
		return ""
	}

	if len(eg.errors) == 1 {
		return eg.errors[0].Error()
	}

	var messages []string
	for i, err := range eg.errors {
		messages = append(messages, fmt.Sprintf("[%d] %s", i+1, err.Error()))
	}

	return fmt.Sprintf("multiple errors occurred: %s", strings.Join(messages, "; "))
}

// ToMessagingError converts the error group into a structured messaging error.
func (eg *ErrorGroup) ToMessagingError() *BaseMessagingError {
	if len(eg.errors) == 0 {
		return nil
	}

	if len(eg.errors) == 1 {
		if msgErr, ok := eg.errors[0].(*BaseMessagingError); ok {
			return msgErr
		}
	}

	return NewError(ErrorTypeInternal, ErrCodeInternal).
		Message(fmt.Sprintf("multiple errors occurred (%d errors)", len(eg.errors))).
		Severity(SeverityHigh).
		Details(map[string]interface{}{
			"error_count": len(eg.errors),
			"errors":      eg.errors,
		}).
		Build()
}
