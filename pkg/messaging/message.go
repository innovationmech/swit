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
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// StructMessageValidator provides comprehensive validation for Message structs.
// It uses struct tags for declarative validation rules and supports
// custom validation logic for complex business rules.
type StructMessageValidator struct {
	validator *validator.Validate
}

// NewStructMessageValidator creates a new message validator with standard rules.
// The validator includes built-in validations for message fields and
// can be extended with custom validation functions.
func NewStructMessageValidator() *StructMessageValidator {
	v := validator.New()

	// Register custom validation for topic names
	v.RegisterValidation("topic", validateTopicName)

	// Register custom validation for correlation IDs
	v.RegisterValidation("correlation_id", validateCorrelationID)

	return &StructMessageValidator{validator: v}
}

// validateTopicName validates that topic names follow naming conventions.
// Topic names must be alphanumeric with dots, hyphens, and underscores allowed.
// They cannot start or end with special characters.
func validateTopicName(fl validator.FieldLevel) bool {
	topic := fl.Field().String()
	if topic == "" {
		return false
	}

	// Topic name pattern: alphanumeric, dots, hyphens, underscores
	// Cannot start or end with special characters
	pattern := `^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$`
	if len(topic) == 1 {
		pattern = `^[a-zA-Z0-9]$`
	}

	matched, _ := regexp.MatchString(pattern, topic)
	return matched
}

// validateCorrelationID validates correlation ID format.
// Correlation IDs should be UUIDs or other structured identifiers.
func validateCorrelationID(fl validator.FieldLevel) bool {
	corrID := fl.Field().String()
	if corrID == "" {
		return true // Optional field
	}

	// UUID pattern or alphanumeric with hyphens
	pattern := `^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$`
	if len(corrID) == 1 {
		pattern = `^[a-zA-Z0-9]$`
	}

	matched, _ := regexp.MatchString(pattern, corrID)
	return matched
}

// ValidatedMessage extends the base Message struct with validation tags
// and additional metadata for enhanced message processing capabilities.
type ValidatedMessage struct {
	// ID is the unique message identifier (required)
	ID string `json:"id" validate:"required,uuid4"`

	// Topic is the destination topic/queue name (required)
	Topic string `json:"topic" validate:"required,topic"`

	// Key is an optional routing/partition key for message ordering
	Key []byte `json:"key,omitempty" validate:"omitempty,max=1024"`

	// Payload is the actual message content (required)
	Payload []byte `json:"payload" validate:"required,min=1,max=1048576"` // Max 1MB

	// Headers contains message metadata as key-value pairs
	Headers MessageHeaders `json:"headers,omitempty"`

	// Timestamp indicates when the message was created (required)
	Timestamp time.Time `json:"timestamp" validate:"required"`

	// CorrelationID is used for request-response patterns and distributed tracing
	CorrelationID string `json:"correlation_id,omitempty" validate:"omitempty,correlation_id"`

	// ReplyTo specifies the topic for responses in request-response patterns
	ReplyTo string `json:"reply_to,omitempty" validate:"omitempty,topic"`

	// Priority sets the message priority (0-9, where 9 is highest priority)
	Priority int `json:"priority,omitempty" validate:"min=0,max=9"`

	// TTL specifies the time-to-live in seconds
	TTL int `json:"ttl,omitempty" validate:"min=0,max=86400"` // Max 24 hours

	// DeliveryAttempt tracks how many times delivery has been attempted
	DeliveryAttempt int `json:"delivery_attempt,omitempty" validate:"min=0"`

	// TraceContext contains OpenTelemetry trace context for distributed tracing
	TraceContext TraceContext `json:"trace_context,omitempty"`

	// Routing contains message routing information
	Routing RoutingInfo `json:"routing,omitempty"`

	// BrokerMetadata contains broker-specific metadata (not serialized)
	BrokerMetadata interface{} `json:"-"`
}

// MessageHeaders provides a structured way to handle message headers
// with support for tracing, correlation, and custom metadata.
type MessageHeaders struct {
	// Standard headers for distributed tracing and correlation
	TraceID      string            `json:"trace_id,omitempty"`
	SpanID       string            `json:"span_id,omitempty"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	TraceFlags   string            `json:"trace_flags,omitempty"`
	Baggage      map[string]string `json:"baggage,omitempty"`

	// Message correlation and routing headers
	CorrelationID   string `json:"correlation_id,omitempty"`
	CausationID     string `json:"causation_id,omitempty"`
	MessageType     string `json:"message_type,omitempty"`
	ContentType     string `json:"content_type,omitempty"`
	ContentEncoding string `json:"content_encoding,omitempty"`

	// Timing and retry headers
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RetryCount int        `json:"retry_count,omitempty"`
	MaxRetries int        `json:"max_retries,omitempty"`

	// Custom headers for application-specific metadata
	Custom map[string]string `json:"custom,omitempty"`
}

// TraceContext encapsulates OpenTelemetry trace context information
// for distributed tracing support across message boundaries.
type TraceContext struct {
	// TraceID uniquely identifies the trace
	TraceID trace.TraceID `json:"trace_id"`

	// SpanID uniquely identifies the span within the trace
	SpanID trace.SpanID `json:"span_id"`

	// TraceFlags contains trace sampling and other flags
	TraceFlags trace.TraceFlags `json:"trace_flags"`

	// TraceState holds vendor-specific trace state
	TraceState trace.TraceState `json:"trace_state,omitempty"`

	// Baggage contains cross-cutting concerns data
	Baggage map[string]string `json:"baggage,omitempty"`
}

// Validate performs comprehensive validation on the ValidatedMessage.
// It checks both struct-level validation rules and business logic constraints.
func (v *StructMessageValidator) Validate(msg *ValidatedMessage) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Perform struct validation using tags
	if err := v.validator.Struct(msg); err != nil {
		return &MessageValidationError{
			Field:   getFieldName(err),
			Message: formatValidationError(err),
			Cause:   err,
		}
	}

	// Additional business logic validation
	if err := v.validateBusinessRules(msg); err != nil {
		return err
	}

	return nil
}

// validateBusinessRules performs complex validation that cannot be expressed
// through struct tags alone.
func (v *StructMessageValidator) validateBusinessRules(msg *ValidatedMessage) error {
	// Validate timestamp is not in the future
	if msg.Timestamp.After(time.Now().Add(5 * time.Minute)) {
		return &MessageValidationError{
			Field:   "timestamp",
			Message: "message timestamp cannot be more than 5 minutes in the future",
		}
	}

	// Validate TTL expiry time
	if msg.TTL > 0 {
		expiryTime := msg.Timestamp.Add(time.Duration(msg.TTL) * time.Second)
		if expiryTime.Before(time.Now()) {
			return &MessageValidationError{
				Field:   "ttl",
				Message: "message has already expired based on TTL",
			}
		}
	}

	// Validate correlation ID consistency with headers
	if msg.CorrelationID != "" && msg.Headers.CorrelationID != "" {
		if msg.CorrelationID != msg.Headers.CorrelationID {
			return &MessageValidationError{
				Field:   "correlation_id",
				Message: "correlation ID in message and headers must match",
			}
		}
	}

	return nil
}

// MessageValidationError represents a message validation failure with detailed information.
type MessageValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// Error implements the error interface for MessageValidationError.
func (e *MessageValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

// Unwrap returns the underlying cause of the validation error.
func (e *MessageValidationError) Unwrap() error {
	return e.Cause
}

// getFieldName extracts the field name from a validation error.
func getFieldName(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			return fieldError.Field()
		}
	}
	return ""
}

// formatValidationError creates a human-readable validation error message.
func formatValidationError(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			switch fieldError.Tag() {
			case "required":
				return "field is required"
			case "uuid4":
				return "field must be a valid UUID v4"
			case "topic":
				return "field must be a valid topic name"
			case "correlation_id":
				return "field must be a valid correlation ID"
			case "min":
				return fmt.Sprintf("field must be at least %s", fieldError.Param())
			case "max":
				return fmt.Sprintf("field must be at most %s", fieldError.Param())
			default:
				return fmt.Sprintf("field validation failed: %s", fieldError.Tag())
			}
		}
	}
	return err.Error()
}

// MessageBuilder provides a fluent interface for constructing validated messages
// with proper defaults and validation.
type MessageBuilder struct {
	message *ValidatedMessage
}

// NewMessageBuilder creates a new message builder with sensible defaults.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		message: &ValidatedMessage{
			Headers:   MessageHeaders{Custom: make(map[string]string)},
			Timestamp: time.Now(),
			Priority:  5, // Default priority
		},
	}
}

// WithID sets the message ID.
func (b *MessageBuilder) WithID(id string) *MessageBuilder {
	b.message.ID = id
	return b
}

// WithTopic sets the destination topic.
func (b *MessageBuilder) WithTopic(topic string) *MessageBuilder {
	b.message.Topic = topic
	return b
}

// WithPayload sets the message payload.
func (b *MessageBuilder) WithPayload(payload []byte) *MessageBuilder {
	b.message.Payload = payload
	return b
}

// WithCorrelationID sets the correlation ID for request-response patterns.
func (b *MessageBuilder) WithCorrelationID(correlationID string) *MessageBuilder {
	b.message.CorrelationID = correlationID
	b.message.Headers.CorrelationID = correlationID
	return b
}

// WithHeader adds a custom header to the message.
func (b *MessageBuilder) WithHeader(key, value string) *MessageBuilder {
	if b.message.Headers.Custom == nil {
		b.message.Headers.Custom = make(map[string]string)
	}
	b.message.Headers.Custom[key] = value
	return b
}

// WithTraceContext sets the distributed tracing context.
func (b *MessageBuilder) WithTraceContext(ctx context.Context) *MessageBuilder {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		b.message.TraceContext = TraceContext{
			TraceID:    span.SpanContext().TraceID(),
			SpanID:     span.SpanContext().SpanID(),
			TraceFlags: span.SpanContext().TraceFlags(),
			TraceState: span.SpanContext().TraceState(),
		}

		// Set trace headers
		b.message.Headers.TraceID = span.SpanContext().TraceID().String()
		b.message.Headers.SpanID = span.SpanContext().SpanID().String()
		b.message.Headers.TraceFlags = span.SpanContext().TraceFlags().String()
	}
	return b
}

// WithPriority sets the message priority (0-9, where 9 is highest).
func (b *MessageBuilder) WithPriority(priority int) *MessageBuilder {
	b.message.Priority = priority
	return b
}

// WithTTL sets the time-to-live in seconds.
func (b *MessageBuilder) WithTTL(ttlSeconds int) *MessageBuilder {
	b.message.TTL = ttlSeconds
	if ttlSeconds > 0 {
		expiryTime := b.message.Timestamp.Add(time.Duration(ttlSeconds) * time.Second)
		b.message.Headers.ExpiresAt = &expiryTime
	}
	return b
}

// WithRouting sets the routing information.
func (b *MessageBuilder) WithRouting(routing RoutingInfo) *MessageBuilder {
	b.message.Routing = routing
	return b
}

// Build creates and validates the final message.
func (b *MessageBuilder) Build() (*ValidatedMessage, error) {
	validator := NewStructMessageValidator()
	if err := validator.Validate(b.message); err != nil {
		return nil, err
	}

	// Create a copy to prevent modification after build
	result := *b.message
	return &result, nil
}

// InjectTraceContext injects trace context into message headers using OpenTelemetry propagation.
func (msg *ValidatedMessage) InjectTraceContext(ctx context.Context, propagator propagation.TextMapPropagator) {
	carrier := &HeaderCarrier{headers: msg.Headers.Custom}
	propagator.Inject(ctx, carrier)
}

// ExtractTraceContext extracts trace context from message headers using OpenTelemetry propagation.
func (msg *ValidatedMessage) ExtractTraceContext(ctx context.Context, propagator propagation.TextMapPropagator) context.Context {
	carrier := &HeaderCarrier{headers: msg.Headers.Custom}
	return propagator.Extract(ctx, carrier)
}

// HeaderCarrier implements propagation.TextMapCarrier for message headers.
type HeaderCarrier struct {
	headers map[string]string
}

// Get returns the value associated with the key.
func (hc *HeaderCarrier) Get(key string) string {
	return hc.headers[key]
}

// Set sets the key-value pair.
func (hc *HeaderCarrier) Set(key, value string) {
	if hc.headers == nil {
		hc.headers = make(map[string]string)
	}
	hc.headers[key] = value
}

// Keys lists the keys stored in the carrier.
func (hc *HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(hc.headers))
	for k := range hc.headers {
		keys = append(keys, k)
	}
	return keys
}
