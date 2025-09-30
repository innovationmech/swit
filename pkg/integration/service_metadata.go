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

package integration

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// ServiceMetadata extends service metadata to include messaging capabilities,
// event schemas, and patterns that the service supports. This enables
// service discovery, capability matching, and runtime validation of
// messaging interactions.
//
// Features:
// - Messaging capability declarations
// - Event schema definitions for published/subscribed events
// - Pattern support indicators (saga, outbox, inbox, etc.)
// - Service-level messaging configuration metadata
type ServiceMetadata struct {
	// Name is the unique service identifier
	Name string `json:"name" yaml:"name" validate:"required"`

	// Version is the service version
	Version string `json:"version" yaml:"version" validate:"required,semver"`

	// Description provides human-readable service information
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// MessagingCapabilities describes the messaging features this service supports
	MessagingCapabilities *MessagingCapabilities `json:"messaging_capabilities,omitempty" yaml:"messaging_capabilities,omitempty"`

	// PublishedEvents lists all events this service publishes
	PublishedEvents []EventMetadata `json:"published_events,omitempty" yaml:"published_events,omitempty"`

	// SubscribedEvents lists all events this service subscribes to
	SubscribedEvents []EventMetadata `json:"subscribed_events,omitempty" yaml:"subscribed_events,omitempty"`

	// PatternSupport indicates which messaging patterns are implemented
	PatternSupport PatternSupport `json:"pattern_support,omitempty" yaml:"pattern_support,omitempty"`

	// Tags are optional categorization tags
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Dependencies lists services this service depends on
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`

	// Extended contains additional custom metadata
	Extended map[string]interface{} `json:"extended,omitempty" yaml:"extended,omitempty"`
}

// MessagingCapabilities describes the messaging features and configurations
// that a service supports. This enables capability-based service discovery
// and runtime validation.
type MessagingCapabilities struct {
	// BrokerTypes lists the message brokers this service can work with
	BrokerTypes []messaging.BrokerType `json:"broker_types" yaml:"broker_types" validate:"required,min=1"`

	// SupportsBatching indicates if the service supports batch message processing
	SupportsBatching bool `json:"supports_batching" yaml:"supports_batching"`

	// SupportsAsync indicates if the service supports async message publishing
	SupportsAsync bool `json:"supports_async" yaml:"supports_async"`

	// SupportsTransactions indicates if the service supports transactional messaging
	SupportsTransactions bool `json:"supports_transactions" yaml:"supports_transactions"`

	// SupportsOrdering indicates if the service requires ordered message delivery
	SupportsOrdering bool `json:"supports_ordering" yaml:"supports_ordering"`

	// CompressionFormats lists supported compression formats
	CompressionFormats []messaging.CompressionType `json:"compression_formats,omitempty" yaml:"compression_formats,omitempty"`

	// SerializationFormats lists supported serialization formats
	SerializationFormats []string `json:"serialization_formats,omitempty" yaml:"serialization_formats,omitempty"`

	// MaxConcurrency defines the maximum number of concurrent message handlers
	MaxConcurrency int `json:"max_concurrency,omitempty" yaml:"max_concurrency,omitempty" validate:"omitempty,min=1"`

	// MaxBatchSize defines the maximum batch size for batch operations
	MaxBatchSize int `json:"max_batch_size,omitempty" yaml:"max_batch_size,omitempty" validate:"omitempty,min=1"`

	// ProcessingTimeout is the maximum time allowed for message processing
	ProcessingTimeout time.Duration `json:"processing_timeout,omitempty" yaml:"processing_timeout,omitempty"`

	// DeadLetterSupport indicates if the service supports dead letter queues
	DeadLetterSupport bool `json:"dead_letter_support" yaml:"dead_letter_support"`

	// RetryPolicy describes the retry configuration
	RetryPolicy *RetryPolicyMetadata `json:"retry_policy,omitempty" yaml:"retry_policy,omitempty"`
}

// EventMetadata describes an event that a service publishes or subscribes to.
// It includes the event schema, routing information, and processing requirements.
type EventMetadata struct {
	// EventType is the unique identifier for this event type
	EventType string `json:"event_type" yaml:"event_type" validate:"required"`

	// Version is the schema version of this event
	Version string `json:"version" yaml:"version" validate:"required,semver"`

	// Description provides human-readable event information
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Schema defines the structure and validation rules for this event
	Schema *MessageSchema `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Topics lists the messaging topics/queues for this event
	Topics []string `json:"topics,omitempty" yaml:"topics,omitempty" validate:"omitempty,min=1"`

	// Required indicates if this event is required for service operation
	Required bool `json:"required" yaml:"required"`

	// Deprecated indicates if this event is deprecated
	Deprecated bool `json:"deprecated" yaml:"deprecated"`

	// DeprecationMessage provides information about deprecation
	DeprecationMessage string `json:"deprecation_message,omitempty" yaml:"deprecation_message,omitempty"`

	// Examples provides example payloads for documentation
	Examples []string `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// MessageSchema defines the structure and validation rules for a message.
// It supports both JSON Schema and custom schema definitions.
type MessageSchema struct {
	// Type is the schema type (e.g., "json-schema", "protobuf", "avro")
	Type SchemaType `json:"type" yaml:"type" validate:"required"`

	// Format specifies the format version (e.g., "draft-07" for JSON Schema)
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Definition is the actual schema definition
	// For JSON Schema, this would be a JSON Schema object
	// For Protobuf, this would be the .proto definition or reference
	Definition interface{} `json:"definition,omitempty" yaml:"definition,omitempty"`

	// ContentType is the MIME type of the message payload
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`

	// Encoding specifies the encoding format (e.g., "utf-8", "base64")
	Encoding string `json:"encoding,omitempty" yaml:"encoding,omitempty"`

	// RequiredFields lists fields that must be present
	RequiredFields []string `json:"required_fields,omitempty" yaml:"required_fields,omitempty"`

	// ValidationRules contains additional validation constraints
	ValidationRules map[string]interface{} `json:"validation_rules,omitempty" yaml:"validation_rules,omitempty"`
}

// SchemaType represents the type of schema definition
type SchemaType string

const (
	// SchemaTypeJSONSchema indicates JSON Schema format
	SchemaTypeJSONSchema SchemaType = "json-schema"

	// SchemaTypeProtobuf indicates Protocol Buffers format
	SchemaTypeProtobuf SchemaType = "protobuf"

	// SchemaTypeAvro indicates Apache Avro format
	SchemaTypeAvro SchemaType = "avro"

	// SchemaTypeCustom indicates a custom schema format
	SchemaTypeCustom SchemaType = "custom"
)

// PatternSupport indicates which distributed messaging patterns
// the service implements or supports.
type PatternSupport struct {
	// Saga indicates saga pattern support for distributed transactions
	Saga bool `json:"saga" yaml:"saga"`

	// Outbox indicates outbox pattern support for reliable publishing
	Outbox bool `json:"outbox" yaml:"outbox"`

	// Inbox indicates inbox pattern support for idempotent processing
	Inbox bool `json:"inbox" yaml:"inbox"`

	// CQRS indicates command-query responsibility segregation support
	CQRS bool `json:"cqrs" yaml:"cqrs"`

	// EventSourcing indicates event sourcing pattern support
	EventSourcing bool `json:"event_sourcing" yaml:"event_sourcing"`

	// CircuitBreaker indicates circuit breaker pattern implementation
	CircuitBreaker bool `json:"circuit_breaker" yaml:"circuit_breaker"`

	// BulkOperations indicates support for bulk message operations
	BulkOperations bool `json:"bulk_operations" yaml:"bulk_operations"`
}

// RetryPolicyMetadata describes the retry configuration for message processing.
type RetryPolicyMetadata struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries" yaml:"max_retries" validate:"min=0"`

	// InitialInterval is the initial retry delay
	InitialInterval time.Duration `json:"initial_interval" yaml:"initial_interval"`

	// MaxInterval is the maximum retry delay
	MaxInterval time.Duration `json:"max_interval,omitempty" yaml:"max_interval,omitempty"`

	// Multiplier is the backoff multiplier
	Multiplier float64 `json:"multiplier,omitempty" yaml:"multiplier,omitempty" validate:"omitempty,min=1"`

	// BackoffType specifies the backoff strategy
	BackoffType BackoffType `json:"backoff_type" yaml:"backoff_type"`
}

// BackoffType represents the retry backoff strategy
type BackoffType string

const (
	// BackoffTypeConstant indicates constant delay between retries
	BackoffTypeConstant BackoffType = "constant"

	// BackoffTypeLinear indicates linear backoff
	BackoffTypeLinear BackoffType = "linear"

	// BackoffTypeExponential indicates exponential backoff
	BackoffTypeExponential BackoffType = "exponential"
)

// Validate performs validation on ServiceMetadata to ensure all required
// fields are present and valid.
//
// Returns:
//   - error: Validation error if metadata is invalid
func (sm *ServiceMetadata) Validate() error {
	if sm == nil {
		return fmt.Errorf("service metadata cannot be nil")
	}

	if strings.TrimSpace(sm.Name) == "" {
		return fmt.Errorf("service name is required")
	}

	if strings.TrimSpace(sm.Version) == "" {
		return fmt.Errorf("service version is required")
	}

	// Validate semantic versioning format
	if err := validateSemver(sm.Version); err != nil {
		return fmt.Errorf("invalid service version: %w", err)
	}

	// Validate messaging capabilities if present
	if sm.MessagingCapabilities != nil {
		if err := sm.MessagingCapabilities.Validate(); err != nil {
			return fmt.Errorf("invalid messaging capabilities: %w", err)
		}
	}

	// Validate published events
	for i, event := range sm.PublishedEvents {
		if err := event.Validate(); err != nil {
			return fmt.Errorf("invalid published event at index %d: %w", i, err)
		}
	}

	// Validate subscribed events
	for i, event := range sm.SubscribedEvents {
		if err := event.Validate(); err != nil {
			return fmt.Errorf("invalid subscribed event at index %d: %w", i, err)
		}
	}

	return nil
}

// Validate performs validation on MessagingCapabilities.
//
// Returns:
//   - error: Validation error if capabilities are invalid
func (mc *MessagingCapabilities) Validate() error {
	if mc == nil {
		return fmt.Errorf("messaging capabilities cannot be nil")
	}

	if len(mc.BrokerTypes) == 0 {
		return fmt.Errorf("at least one broker type is required")
	}

	// Validate broker types
	for i, brokerType := range mc.BrokerTypes {
		if brokerType == "" {
			return fmt.Errorf("broker type at index %d cannot be empty", i)
		}
	}

	// Validate concurrency settings
	if mc.MaxConcurrency < 0 {
		return fmt.Errorf("max concurrency cannot be negative")
	}

	if mc.MaxBatchSize < 0 {
		return fmt.Errorf("max batch size cannot be negative")
	}

	// Validate retry policy if present
	if mc.RetryPolicy != nil {
		if err := mc.RetryPolicy.Validate(); err != nil {
			return fmt.Errorf("invalid retry policy: %w", err)
		}
	}

	return nil
}

// Validate performs validation on EventMetadata.
//
// Returns:
//   - error: Validation error if event metadata is invalid
func (em *EventMetadata) Validate() error {
	if em == nil {
		return fmt.Errorf("event metadata cannot be nil")
	}

	if strings.TrimSpace(em.EventType) == "" {
		return fmt.Errorf("event type is required")
	}

	if strings.TrimSpace(em.Version) == "" {
		return fmt.Errorf("event version is required")
	}

	// Validate semantic versioning format
	if err := validateSemver(em.Version); err != nil {
		return fmt.Errorf("invalid event version: %w", err)
	}

	// Validate schema if present
	if em.Schema != nil {
		if err := em.Schema.Validate(); err != nil {
			return fmt.Errorf("invalid event schema: %w", err)
		}
	}

	// Validate deprecation message if deprecated
	if em.Deprecated && strings.TrimSpace(em.DeprecationMessage) == "" {
		return fmt.Errorf("deprecation message is required when event is deprecated")
	}

	return nil
}

// Validate performs validation on MessageSchema.
//
// Returns:
//   - error: Validation error if schema is invalid
func (ms *MessageSchema) Validate() error {
	if ms == nil {
		return fmt.Errorf("message schema cannot be nil")
	}

	if ms.Type == "" {
		return fmt.Errorf("schema type is required")
	}

	// Validate schema type
	validTypes := []SchemaType{
		SchemaTypeJSONSchema,
		SchemaTypeProtobuf,
		SchemaTypeAvro,
		SchemaTypeCustom,
	}

	valid := false
	for _, validType := range validTypes {
		if ms.Type == validType {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid schema type: %s", ms.Type)
	}

	return nil
}

// Validate performs validation on RetryPolicyMetadata.
//
// Returns:
//   - error: Validation error if retry policy is invalid
func (rpm *RetryPolicyMetadata) Validate() error {
	if rpm == nil {
		return fmt.Errorf("retry policy metadata cannot be nil")
	}

	if rpm.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if rpm.InitialInterval < 0 {
		return fmt.Errorf("initial interval cannot be negative")
	}

	if rpm.MaxInterval > 0 && rpm.MaxInterval < rpm.InitialInterval {
		return fmt.Errorf("max interval cannot be less than initial interval")
	}

	if rpm.Multiplier > 0 && rpm.Multiplier < 1 {
		return fmt.Errorf("multiplier must be >= 1.0")
	}

	// Validate backoff type
	validBackoffs := []BackoffType{
		BackoffTypeConstant,
		BackoffTypeLinear,
		BackoffTypeExponential,
	}

	valid := false
	for _, validBackoff := range validBackoffs {
		if rpm.BackoffType == validBackoff {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid backoff type: %s", rpm.BackoffType)
	}

	return nil
}

// validateSemver validates semantic versioning format.
// Supports formats like: 1.0.0, 1.0.0-alpha, 1.0.0-beta.1, 1.0.0+20130313144700
func validateSemver(version string) error {
	// Semantic versioning regex pattern
	semverPattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
		`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

	matched, err := regexp.MatchString(semverPattern, version)
	if err != nil {
		return fmt.Errorf("failed to validate version: %w", err)
	}

	if !matched {
		return fmt.Errorf("version %s does not match semantic versioning format (e.g., 1.0.0)", version)
	}

	return nil
}

// HasCapability checks if the service supports a specific messaging capability.
//
// Parameters:
//   - capability: The capability name to check
//
// Returns:
//   - bool: True if the capability is supported
func (sm *ServiceMetadata) HasCapability(capability string) bool {
	if sm.MessagingCapabilities == nil {
		return false
	}

	switch capability {
	case "batching":
		return sm.MessagingCapabilities.SupportsBatching
	case "async":
		return sm.MessagingCapabilities.SupportsAsync
	case "transactions":
		return sm.MessagingCapabilities.SupportsTransactions
	case "ordering":
		return sm.MessagingCapabilities.SupportsOrdering
	case "dead_letter":
		return sm.MessagingCapabilities.DeadLetterSupport
	default:
		return false
	}
}

// HasPattern checks if the service implements a specific messaging pattern.
//
// Parameters:
//   - pattern: The pattern name to check
//
// Returns:
//   - bool: True if the pattern is implemented
func (sm *ServiceMetadata) HasPattern(pattern string) bool {
	switch pattern {
	case "saga":
		return sm.PatternSupport.Saga
	case "outbox":
		return sm.PatternSupport.Outbox
	case "inbox":
		return sm.PatternSupport.Inbox
	case "cqrs":
		return sm.PatternSupport.CQRS
	case "event_sourcing":
		return sm.PatternSupport.EventSourcing
	case "circuit_breaker":
		return sm.PatternSupport.CircuitBreaker
	case "bulk_operations":
		return sm.PatternSupport.BulkOperations
	default:
		return false
	}
}

// GetPublishedEvent retrieves metadata for a published event by type.
//
// Parameters:
//   - eventType: The event type to find
//
// Returns:
//   - *EventMetadata: Event metadata if found, nil otherwise
func (sm *ServiceMetadata) GetPublishedEvent(eventType string) *EventMetadata {
	for i := range sm.PublishedEvents {
		if sm.PublishedEvents[i].EventType == eventType {
			return &sm.PublishedEvents[i]
		}
	}
	return nil
}

// GetSubscribedEvent retrieves metadata for a subscribed event by type.
//
// Parameters:
//   - eventType: The event type to find
//
// Returns:
//   - *EventMetadata: Event metadata if found, nil otherwise
func (sm *ServiceMetadata) GetSubscribedEvent(eventType string) *EventMetadata {
	for i := range sm.SubscribedEvents {
		if sm.SubscribedEvents[i].EventType == eventType {
			return &sm.SubscribedEvents[i]
		}
	}
	return nil
}

// SupportsBrokerType checks if the service supports a specific broker type.
//
// Parameters:
//   - brokerType: The broker type to check
//
// Returns:
//   - bool: True if the broker type is supported
func (sm *ServiceMetadata) SupportsBrokerType(brokerType messaging.BrokerType) bool {
	if sm.MessagingCapabilities == nil {
		return false
	}

	for _, bt := range sm.MessagingCapabilities.BrokerTypes {
		if bt == brokerType {
			return true
		}
	}

	return false
}
