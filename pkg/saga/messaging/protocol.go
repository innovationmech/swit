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

// Package messaging provides Saga messaging protocol definitions and utilities.
// This file defines the standardized message protocol for Saga communication across services.
package messaging

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// 协议版本常量
const (
	// ProtocolVersion1_0 is the initial protocol version
	ProtocolVersion1_0 = "1.0"

	// ProtocolVersion1_1 adds support for enhanced tracing
	ProtocolVersion1_1 = "1.1"

	// CurrentProtocolVersion is the current protocol version
	CurrentProtocolVersion = ProtocolVersion1_1

	// MinSupportedProtocolVersion is the minimum supported protocol version
	MinSupportedProtocolVersion = ProtocolVersion1_0
)

// 标准消息头常量
const (
	// HeaderSagaID identifies the Saga instance
	HeaderSagaID = "X-Saga-ID"

	// HeaderEventID identifies the specific event
	HeaderEventID = "X-Event-ID"

	// HeaderEventType specifies the type of the event
	HeaderEventType = "X-Event-Type"

	// HeaderTimestamp contains the event timestamp in RFC3339 format
	HeaderTimestamp = "X-Timestamp"

	// HeaderVersion specifies the protocol version
	HeaderVersion = "X-Protocol-Version"

	// HeaderCorrelation contains the correlation ID for request tracking
	HeaderCorrelation = "X-Correlation-ID"

	// HeaderSource identifies the source service
	HeaderSource = "X-Source-Service"

	// HeaderService identifies the target service
	HeaderService = "X-Service"

	// HeaderServiceVersion specifies the service version
	HeaderServiceVersion = "X-Service-Version"

	// HeaderStepID identifies the Saga step
	HeaderStepID = "X-Step-ID"

	// HeaderTraceID contains the distributed trace ID
	HeaderTraceID = "X-Trace-ID"

	// HeaderSpanID contains the span ID
	HeaderSpanID = "X-Span-ID"

	// HeaderParentSpanID contains the parent span ID
	HeaderParentSpanID = "X-Parent-Span-ID"

	// HeaderContentType specifies the payload content type
	HeaderContentType = "Content-Type"

	// HeaderEncoding specifies the payload encoding
	HeaderEncoding = "Content-Encoding"

	// HeaderAttempt indicates the retry attempt number
	HeaderAttempt = "X-Attempt"

	// HeaderMaxAttempts specifies the maximum number of retry attempts
	HeaderMaxAttempts = "X-Max-Attempts"

	// HeaderDuration contains the event duration in milliseconds
	HeaderDuration = "X-Duration-Ms"
)

// 消息类型常量
const (
	// MessageTypeSagaEvent indicates a Saga event message
	MessageTypeSagaEvent = "saga.event"

	// MessageTypeSagaCommand indicates a Saga command message
	MessageTypeSagaCommand = "saga.command"

	// MessageTypeSagaQuery indicates a Saga query message
	MessageTypeSagaQuery = "saga.query"

	// MessageTypeCompensation indicates a compensation event
	MessageTypeCompensation = "compensation.event"
)

// 主题命名常量
const (
	// TopicPrefixSagaEvents is the prefix for Saga event topics
	TopicPrefixSagaEvents = "saga.events"

	// TopicPrefixSagaCommands is the prefix for Saga command topics
	TopicPrefixSagaCommands = "saga.commands"

	// TopicPrefixCompensation is the prefix for compensation topics
	TopicPrefixCompensation = "saga.compensation"

	// TopicSeparator is the separator used in topic names
	TopicSeparator = "."
)

// SagaMessageProtocol defines the standardized Saga message protocol.
// It provides a consistent message structure for cross-service Saga communication.
//
// The protocol supports:
//   - Multiple serialization formats (JSON, Protobuf)
//   - Protocol version negotiation and compatibility
//   - Distributed tracing integration
//   - Retry and compensation tracking
//   - Flexible metadata and headers
type SagaMessageProtocol struct {
	// Version specifies the protocol version (e.g., "1.0", "1.1")
	Version string `json:"version" yaml:"version"`

	// MessageType identifies the message category (event, command, query)
	MessageType string `json:"message_type" yaml:"message_type"`

	// SagaID uniquely identifies the Saga instance
	SagaID string `json:"saga_id" yaml:"saga_id"`

	// EventID uniquely identifies this specific event
	EventID string `json:"event_id" yaml:"event_id"`

	// EventType specifies the type of Saga event
	EventType string `json:"event_type" yaml:"event_type"`

	// Timestamp indicates when the event occurred
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// Headers contains message metadata as key-value pairs
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`

	// Payload contains the actual message data
	Payload interface{} `json:"payload,omitempty" yaml:"payload,omitempty"`

	// StepID identifies the Saga step (optional)
	StepID string `json:"step_id,omitempty" yaml:"step_id,omitempty"`

	// CorrelationID is used for request tracking and correlation
	CorrelationID string `json:"correlation_id,omitempty" yaml:"correlation_id,omitempty"`

	// Source identifies the originating service
	Source string `json:"source,omitempty" yaml:"source,omitempty"`

	// Service identifies the target service
	Service string `json:"service,omitempty" yaml:"service,omitempty"`

	// ServiceVersion specifies the service version
	ServiceVersion string `json:"service_version,omitempty" yaml:"service_version,omitempty"`

	// TraceID contains the distributed trace ID
	TraceID string `json:"trace_id,omitempty" yaml:"trace_id,omitempty"`

	// SpanID contains the span ID
	SpanID string `json:"span_id,omitempty" yaml:"span_id,omitempty"`

	// ParentSpanID contains the parent span ID
	ParentSpanID string `json:"parent_span_id,omitempty" yaml:"parent_span_id,omitempty"`

	// Attempt indicates the retry attempt number
	Attempt int `json:"attempt,omitempty" yaml:"attempt,omitempty"`

	// MaxAttempts specifies the maximum retry attempts
	MaxAttempts int `json:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`

	// Duration contains the event duration
	Duration time.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`

	// Metadata contains additional arbitrary metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ProtocolValidator validates message protocol compliance.
type ProtocolValidator interface {
	// Validate checks if a protocol message is valid
	Validate(msg *SagaMessageProtocol) error

	// ValidateVersion checks if the protocol version is supported
	ValidateVersion(version string) error

	// ValidateHeaders checks if required headers are present
	ValidateHeaders(headers map[string]string) error
}

// StandardProtocolValidator implements ProtocolValidator with standard rules.
type StandardProtocolValidator struct {
	// RequireVersion enforces protocol version validation
	RequireVersion bool

	// RequiredHeaders specifies which headers must be present
	RequiredHeaders []string

	// AllowedVersions specifies which protocol versions are allowed
	AllowedVersions []string

	// StrictMode enables strict validation rules
	StrictMode bool
}

// NewStandardProtocolValidator creates a new standard protocol validator.
func NewStandardProtocolValidator() *StandardProtocolValidator {
	return &StandardProtocolValidator{
		RequireVersion: true,
		RequiredHeaders: []string{
			HeaderSagaID,
			HeaderEventID,
			HeaderEventType,
			HeaderTimestamp,
		},
		AllowedVersions: []string{
			ProtocolVersion1_0,
			ProtocolVersion1_1,
		},
		StrictMode: false,
	}
}

// Validate implements ProtocolValidator.Validate.
func (v *StandardProtocolValidator) Validate(msg *SagaMessageProtocol) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Validate required fields
	if msg.SagaID == "" {
		return fmt.Errorf("saga_id is required")
	}

	if msg.EventID == "" {
		return fmt.Errorf("event_id is required")
	}

	if msg.EventType == "" {
		return fmt.Errorf("event_type is required")
	}

	if msg.MessageType == "" {
		return fmt.Errorf("message_type is required")
	}

	// Validate timestamp
	if msg.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate version if required
	if v.RequireVersion {
		if err := v.ValidateVersion(msg.Version); err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}
	}

	// Validate headers if strict mode
	if v.StrictMode && msg.Headers != nil {
		if err := v.ValidateHeaders(msg.Headers); err != nil {
			return fmt.Errorf("invalid headers: %w", err)
		}
	}

	return nil
}

// ValidateVersion implements ProtocolValidator.ValidateVersion.
func (v *StandardProtocolValidator) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Check if version is in allowed list
	for _, allowed := range v.AllowedVersions {
		if version == allowed {
			return nil
		}
	}

	return fmt.Errorf("unsupported protocol version: %s (supported: %v)", version, v.AllowedVersions)
}

// ValidateHeaders implements ProtocolValidator.ValidateHeaders.
func (v *StandardProtocolValidator) ValidateHeaders(headers map[string]string) error {
	if headers == nil {
		return fmt.Errorf("headers cannot be nil")
	}

	// Check required headers
	for _, required := range v.RequiredHeaders {
		if _, exists := headers[required]; !exists {
			return fmt.Errorf("required header missing: %s", required)
		}
	}

	return nil
}

// TopicRouter handles topic naming and routing for Saga messages.
type TopicRouter interface {
	// GetTopicForEvent returns the topic name for a Saga event
	GetTopicForEvent(eventType saga.SagaEventType) string

	// GetTopicForCommand returns the topic name for a Saga command
	GetTopicForCommand(commandType string) string

	// GetTopicForCompensation returns the topic name for compensation
	GetTopicForCompensation(eventType saga.SagaEventType) string

	// ParseTopic extracts information from a topic name
	ParseTopic(topic string) (*TopicInfo, error)

	// ValidateTopic checks if a topic name is valid
	ValidateTopic(topic string) error
}

// TopicInfo contains parsed topic information.
type TopicInfo struct {
	// Prefix is the topic prefix (e.g., "saga.events")
	Prefix string

	// Category is the topic category (e.g., "started", "completed")
	Category string

	// FullTopic is the complete topic name
	FullTopic string

	// IsWildcard indicates if this is a wildcard topic
	IsWildcard bool
}

// StandardTopicRouter implements TopicRouter with standard naming conventions.
type StandardTopicRouter struct {
	// EventTopicPrefix is the prefix for event topics
	EventTopicPrefix string

	// CommandTopicPrefix is the prefix for command topics
	CommandTopicPrefix string

	// CompensationTopicPrefix is the prefix for compensation topics
	CompensationTopicPrefix string

	// Separator is the topic separator character
	Separator string
}

// NewStandardTopicRouter creates a new standard topic router.
func NewStandardTopicRouter() *StandardTopicRouter {
	return &StandardTopicRouter{
		EventTopicPrefix:        TopicPrefixSagaEvents,
		CommandTopicPrefix:      TopicPrefixSagaCommands,
		CompensationTopicPrefix: TopicPrefixCompensation,
		Separator:               TopicSeparator,
	}
}

// GetTopicForEvent implements TopicRouter.GetTopicForEvent.
func (r *StandardTopicRouter) GetTopicForEvent(eventType saga.SagaEventType) string {
	// Extract the last part of the event type for the topic name
	// e.g., "saga.started" -> "saga.events.started"
	parts := strings.Split(string(eventType), ".")
	category := strings.Join(parts, r.Separator)

	return fmt.Sprintf("%s%s%s", r.EventTopicPrefix, r.Separator, category)
}

// GetTopicForCommand implements TopicRouter.GetTopicForCommand.
func (r *StandardTopicRouter) GetTopicForCommand(commandType string) string {
	return fmt.Sprintf("%s%s%s", r.CommandTopicPrefix, r.Separator, commandType)
}

// GetTopicForCompensation implements TopicRouter.GetTopicForCompensation.
func (r *StandardTopicRouter) GetTopicForCompensation(eventType saga.SagaEventType) string {
	parts := strings.Split(string(eventType), ".")
	category := strings.Join(parts, r.Separator)

	return fmt.Sprintf("%s%s%s", r.CompensationTopicPrefix, r.Separator, category)
}

// ParseTopic implements TopicRouter.ParseTopic.
func (r *StandardTopicRouter) ParseTopic(topic string) (*TopicInfo, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	info := &TopicInfo{
		FullTopic:  topic,
		IsWildcard: strings.Contains(topic, "*"),
	}

	// Split the topic by separator
	parts := strings.Split(topic, r.Separator)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid topic format: %s", topic)
	}

	// Extract prefix and category
	// For "saga.events.started", prefix is "saga.events", category is "started"
	if len(parts) >= 3 {
		info.Prefix = strings.Join(parts[:2], r.Separator)
		info.Category = strings.Join(parts[2:], r.Separator)
	} else {
		info.Prefix = parts[0]
		info.Category = parts[1]
	}

	return info, nil
}

// ValidateTopic implements TopicRouter.ValidateTopic.
func (r *StandardTopicRouter) ValidateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	// Check for valid characters (alphanumeric, dots, hyphens, underscores, asterisks)
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9._*-]+$`)
	if !validPattern.MatchString(topic) {
		return fmt.Errorf("topic contains invalid characters: %s", topic)
	}

	// Check minimum length
	if len(topic) < 3 {
		return fmt.Errorf("topic is too short: %s", topic)
	}

	// Check for double separators
	if strings.Contains(topic, r.Separator+r.Separator) {
		return fmt.Errorf("topic contains consecutive separators: %s", topic)
	}

	return nil
}

// ProtocolVersionComparer compares protocol versions for compatibility.
type ProtocolVersionComparer interface {
	// IsCompatible checks if two versions are compatible
	IsCompatible(version1, version2 string) bool

	// Compare compares two versions (-1: v1 < v2, 0: v1 == v2, 1: v1 > v2)
	Compare(version1, version2 string) int

	// IsSupported checks if a version is supported
	IsSupported(version string) bool
}

// StandardVersionComparer implements ProtocolVersionComparer.
type StandardVersionComparer struct {
	// MinVersion is the minimum supported version
	MinVersion string

	// MaxVersion is the maximum supported version
	MaxVersion string

	// CurrentVersion is the current protocol version
	CurrentVersion string
}

// NewStandardVersionComparer creates a new standard version comparer.
func NewStandardVersionComparer() *StandardVersionComparer {
	return &StandardVersionComparer{
		MinVersion:     MinSupportedProtocolVersion,
		MaxVersion:     CurrentProtocolVersion,
		CurrentVersion: CurrentProtocolVersion,
	}
}

// IsCompatible implements ProtocolVersionComparer.IsCompatible.
func (c *StandardVersionComparer) IsCompatible(version1, version2 string) bool {
	// For now, same major version is compatible
	v1Major := strings.Split(version1, ".")[0]
	v2Major := strings.Split(version2, ".")[0]

	return v1Major == v2Major
}

// Compare implements ProtocolVersionComparer.Compare.
func (c *StandardVersionComparer) Compare(version1, version2 string) int {
	if version1 == version2 {
		return 0
	}

	v1Parts := strings.Split(version1, ".")
	v2Parts := strings.Split(version2, ".")

	// Compare major version
	if len(v1Parts) > 0 && len(v2Parts) > 0 {
		if v1Parts[0] < v2Parts[0] {
			return -1
		}
		if v1Parts[0] > v2Parts[0] {
			return 1
		}
	}

	// Compare minor version
	if len(v1Parts) > 1 && len(v2Parts) > 1 {
		if v1Parts[1] < v2Parts[1] {
			return -1
		}
		if v1Parts[1] > v2Parts[1] {
			return 1
		}
	}

	return 0
}

// IsSupported implements ProtocolVersionComparer.IsSupported.
func (c *StandardVersionComparer) IsSupported(version string) bool {
	return c.Compare(version, c.MinVersion) >= 0 && c.Compare(version, c.MaxVersion) <= 0
}

// HeaderBuilder helps construct standardized message headers.
type HeaderBuilder struct {
	headers map[string]string
}

// NewHeaderBuilder creates a new header builder.
func NewHeaderBuilder() *HeaderBuilder {
	return &HeaderBuilder{
		headers: make(map[string]string),
	}
}

// WithSagaID sets the Saga ID header.
func (b *HeaderBuilder) WithSagaID(sagaID string) *HeaderBuilder {
	b.headers[HeaderSagaID] = sagaID
	return b
}

// WithEventID sets the event ID header.
func (b *HeaderBuilder) WithEventID(eventID string) *HeaderBuilder {
	b.headers[HeaderEventID] = eventID
	return b
}

// WithEventType sets the event type header.
func (b *HeaderBuilder) WithEventType(eventType string) *HeaderBuilder {
	b.headers[HeaderEventType] = eventType
	return b
}

// WithTimestamp sets the timestamp header.
func (b *HeaderBuilder) WithTimestamp(timestamp time.Time) *HeaderBuilder {
	b.headers[HeaderTimestamp] = timestamp.Format(time.RFC3339Nano)
	return b
}

// WithVersion sets the protocol version header.
func (b *HeaderBuilder) WithVersion(version string) *HeaderBuilder {
	b.headers[HeaderVersion] = version
	return b
}

// WithCorrelation sets the correlation ID header.
func (b *HeaderBuilder) WithCorrelation(correlationID string) *HeaderBuilder {
	b.headers[HeaderCorrelation] = correlationID
	return b
}

// WithSource sets the source service header.
func (b *HeaderBuilder) WithSource(source string) *HeaderBuilder {
	b.headers[HeaderSource] = source
	return b
}

// WithService sets the service header.
func (b *HeaderBuilder) WithService(service string) *HeaderBuilder {
	b.headers[HeaderService] = service
	return b
}

// WithServiceVersion sets the service version header.
func (b *HeaderBuilder) WithServiceVersion(version string) *HeaderBuilder {
	b.headers[HeaderServiceVersion] = version
	return b
}

// WithStepID sets the step ID header.
func (b *HeaderBuilder) WithStepID(stepID string) *HeaderBuilder {
	b.headers[HeaderStepID] = stepID
	return b
}

// WithTraceID sets the trace ID header.
func (b *HeaderBuilder) WithTraceID(traceID string) *HeaderBuilder {
	b.headers[HeaderTraceID] = traceID
	return b
}

// WithSpanID sets the span ID header.
func (b *HeaderBuilder) WithSpanID(spanID string) *HeaderBuilder {
	b.headers[HeaderSpanID] = spanID
	return b
}

// WithParentSpanID sets the parent span ID header.
func (b *HeaderBuilder) WithParentSpanID(parentSpanID string) *HeaderBuilder {
	b.headers[HeaderParentSpanID] = parentSpanID
	return b
}

// WithContentType sets the content type header.
func (b *HeaderBuilder) WithContentType(contentType string) *HeaderBuilder {
	b.headers[HeaderContentType] = contentType
	return b
}

// WithAttempt sets the attempt header.
func (b *HeaderBuilder) WithAttempt(attempt int) *HeaderBuilder {
	b.headers[HeaderAttempt] = fmt.Sprintf("%d", attempt)
	return b
}

// WithMaxAttempts sets the max attempts header.
func (b *HeaderBuilder) WithMaxAttempts(maxAttempts int) *HeaderBuilder {
	b.headers[HeaderMaxAttempts] = fmt.Sprintf("%d", maxAttempts)
	return b
}

// WithDuration sets the duration header.
func (b *HeaderBuilder) WithDuration(duration time.Duration) *HeaderBuilder {
	b.headers[HeaderDuration] = fmt.Sprintf("%d", duration.Milliseconds())
	return b
}

// WithCustomHeader sets a custom header.
func (b *HeaderBuilder) WithCustomHeader(key, value string) *HeaderBuilder {
	b.headers[key] = value
	return b
}

// Build returns the constructed headers map.
func (b *HeaderBuilder) Build() map[string]string {
	result := make(map[string]string, len(b.headers))
	for k, v := range b.headers {
		result[k] = v
	}
	return result
}

// BuildFromSagaEvent creates headers from a SagaEvent.
func BuildFromSagaEvent(event *saga.SagaEvent) map[string]string {
	builder := NewHeaderBuilder()

	builder.WithSagaID(event.SagaID).
		WithEventID(event.ID).
		WithEventType(string(event.Type)).
		WithTimestamp(event.Timestamp).
		WithVersion(event.Version)

	if event.CorrelationID != "" {
		builder.WithCorrelation(event.CorrelationID)
	}

	if event.StepID != "" {
		builder.WithStepID(event.StepID)
	}

	if event.Source != "" {
		builder.WithSource(event.Source)
	}

	if event.Service != "" {
		builder.WithService(event.Service)
	}

	if event.ServiceVersion != "" {
		builder.WithServiceVersion(event.ServiceVersion)
	}

	if event.TraceID != "" {
		builder.WithTraceID(event.TraceID)
	}

	if event.SpanID != "" {
		builder.WithSpanID(event.SpanID)
	}

	if event.ParentSpanID != "" {
		builder.WithParentSpanID(event.ParentSpanID)
	}

	if event.Attempt > 0 {
		builder.WithAttempt(event.Attempt)
	}

	if event.MaxAttempts > 0 {
		builder.WithMaxAttempts(event.MaxAttempts)
	}

	if event.Duration > 0 {
		builder.WithDuration(event.Duration)
	}

	return builder.Build()
}
