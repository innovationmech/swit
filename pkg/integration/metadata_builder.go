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
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/transport"
)

// ServiceMetadataBuilder provides a fluent interface for building ServiceMetadata
// with validation and sensible defaults.
//
// Example:
//
//	metadata := NewServiceMetadataBuilder("order-service", "1.0.0").
//	    WithDescription("Order processing service").
//	    WithMessagingCapabilities(
//	        NewMessagingCapabilitiesBuilder().
//	            WithBrokerType(messaging.BrokerTypeKafka).
//	            WithBatching(true).
//	            Build(),
//	    ).
//	    WithPublishedEvent(EventMetadata{
//	        EventType: "order.created",
//	        Version:   "1.0.0",
//	    }).
//	    Build()
type ServiceMetadataBuilder struct {
	metadata *ServiceMetadata
	errors   []error
}

// NewServiceMetadataBuilder creates a new service metadata builder.
//
// Parameters:
//   - name: Service name (required)
//   - version: Service version (required, must be valid semver)
//
// Returns:
//   - *ServiceMetadataBuilder: Builder instance
func NewServiceMetadataBuilder(name, version string) *ServiceMetadataBuilder {
	return &ServiceMetadataBuilder{
		metadata: &ServiceMetadata{
			Name:             name,
			Version:          version,
			PublishedEvents:  make([]EventMetadata, 0),
			SubscribedEvents: make([]EventMetadata, 0),
			Tags:             make([]string, 0),
			Dependencies:     make([]string, 0),
			Extended:         make(map[string]interface{}),
		},
		errors: make([]error, 0),
	}
}

// WithDescription sets the service description.
func (b *ServiceMetadataBuilder) WithDescription(description string) *ServiceMetadataBuilder {
	b.metadata.Description = description
	return b
}

// WithMessagingCapabilities sets the messaging capabilities.
func (b *ServiceMetadataBuilder) WithMessagingCapabilities(capabilities *MessagingCapabilities) *ServiceMetadataBuilder {
	b.metadata.MessagingCapabilities = capabilities
	return b
}

// WithPublishedEvent adds a published event to the metadata.
func (b *ServiceMetadataBuilder) WithPublishedEvent(event EventMetadata) *ServiceMetadataBuilder {
	b.metadata.PublishedEvents = append(b.metadata.PublishedEvents, event)
	return b
}

// WithSubscribedEvent adds a subscribed event to the metadata.
func (b *ServiceMetadataBuilder) WithSubscribedEvent(event EventMetadata) *ServiceMetadataBuilder {
	b.metadata.SubscribedEvents = append(b.metadata.SubscribedEvents, event)
	return b
}

// WithPatternSupport sets the pattern support configuration.
func (b *ServiceMetadataBuilder) WithPatternSupport(patterns PatternSupport) *ServiceMetadataBuilder {
	b.metadata.PatternSupport = patterns
	return b
}

// WithTag adds a tag to the metadata.
func (b *ServiceMetadataBuilder) WithTag(tag string) *ServiceMetadataBuilder {
	b.metadata.Tags = append(b.metadata.Tags, tag)
	return b
}

// WithDependency adds a dependency to the metadata.
func (b *ServiceMetadataBuilder) WithDependency(dependency string) *ServiceMetadataBuilder {
	b.metadata.Dependencies = append(b.metadata.Dependencies, dependency)
	return b
}

// WithExtended adds extended metadata fields.
func (b *ServiceMetadataBuilder) WithExtended(key string, value interface{}) *ServiceMetadataBuilder {
	b.metadata.Extended[key] = value
	return b
}

// Build creates the ServiceMetadata and validates it.
//
// Returns:
//   - *ServiceMetadata: Built metadata
//   - error: Validation error if metadata is invalid
func (b *ServiceMetadataBuilder) Build() (*ServiceMetadata, error) {
	if err := b.metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service metadata: %w", err)
	}

	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder has errors: %v", b.errors)
	}

	return b.metadata, nil
}

// MustBuild creates the ServiceMetadata and panics if validation fails.
// Use this only when you are certain the metadata is valid.
//
// Returns:
//   - *ServiceMetadata: Built metadata
func (b *ServiceMetadataBuilder) MustBuild() *ServiceMetadata {
	metadata, err := b.Build()
	if err != nil {
		panic(err)
	}
	return metadata
}

// MessagingCapabilitiesBuilder provides a fluent interface for building
// MessagingCapabilities with validation and defaults.
type MessagingCapabilitiesBuilder struct {
	capabilities *MessagingCapabilities
}

// NewMessagingCapabilitiesBuilder creates a new messaging capabilities builder.
func NewMessagingCapabilitiesBuilder() *MessagingCapabilitiesBuilder {
	return &MessagingCapabilitiesBuilder{
		capabilities: &MessagingCapabilities{
			BrokerTypes:          make([]messaging.BrokerType, 0),
			CompressionFormats:   make([]messaging.CompressionType, 0),
			SerializationFormats: make([]string, 0),
		},
	}
}

// WithBrokerType adds a broker type to the capabilities.
func (b *MessagingCapabilitiesBuilder) WithBrokerType(brokerType messaging.BrokerType) *MessagingCapabilitiesBuilder {
	b.capabilities.BrokerTypes = append(b.capabilities.BrokerTypes, brokerType)
	return b
}

// WithBatching sets batching support.
func (b *MessagingCapabilitiesBuilder) WithBatching(supported bool) *MessagingCapabilitiesBuilder {
	b.capabilities.SupportsBatching = supported
	return b
}

// WithAsync sets async support.
func (b *MessagingCapabilitiesBuilder) WithAsync(supported bool) *MessagingCapabilitiesBuilder {
	b.capabilities.SupportsAsync = supported
	return b
}

// WithTransactions sets transaction support.
func (b *MessagingCapabilitiesBuilder) WithTransactions(supported bool) *MessagingCapabilitiesBuilder {
	b.capabilities.SupportsTransactions = supported
	return b
}

// WithOrdering sets ordering support.
func (b *MessagingCapabilitiesBuilder) WithOrdering(supported bool) *MessagingCapabilitiesBuilder {
	b.capabilities.SupportsOrdering = supported
	return b
}

// WithCompression adds a compression format.
func (b *MessagingCapabilitiesBuilder) WithCompression(format messaging.CompressionType) *MessagingCapabilitiesBuilder {
	b.capabilities.CompressionFormats = append(b.capabilities.CompressionFormats, format)
	return b
}

// WithSerialization adds a serialization format.
func (b *MessagingCapabilitiesBuilder) WithSerialization(format string) *MessagingCapabilitiesBuilder {
	b.capabilities.SerializationFormats = append(b.capabilities.SerializationFormats, format)
	return b
}

// WithConcurrency sets the maximum concurrency.
func (b *MessagingCapabilitiesBuilder) WithConcurrency(maxConcurrency int) *MessagingCapabilitiesBuilder {
	b.capabilities.MaxConcurrency = maxConcurrency
	return b
}

// WithBatchSize sets the maximum batch size.
func (b *MessagingCapabilitiesBuilder) WithBatchSize(maxBatchSize int) *MessagingCapabilitiesBuilder {
	b.capabilities.MaxBatchSize = maxBatchSize
	return b
}

// WithProcessingTimeout sets the processing timeout.
func (b *MessagingCapabilitiesBuilder) WithProcessingTimeout(timeout time.Duration) *MessagingCapabilitiesBuilder {
	b.capabilities.ProcessingTimeout = timeout
	return b
}

// WithDeadLetterSupport sets dead letter queue support.
func (b *MessagingCapabilitiesBuilder) WithDeadLetterSupport(supported bool) *MessagingCapabilitiesBuilder {
	b.capabilities.DeadLetterSupport = supported
	return b
}

// WithRetryPolicy sets the retry policy.
func (b *MessagingCapabilitiesBuilder) WithRetryPolicy(policy *RetryPolicyMetadata) *MessagingCapabilitiesBuilder {
	b.capabilities.RetryPolicy = policy
	return b
}

// Build creates the MessagingCapabilities and validates it.
//
// Returns:
//   - *MessagingCapabilities: Built capabilities
//   - error: Validation error if capabilities are invalid
func (b *MessagingCapabilitiesBuilder) Build() (*MessagingCapabilities, error) {
	if err := b.capabilities.Validate(); err != nil {
		return nil, fmt.Errorf("invalid messaging capabilities: %w", err)
	}
	return b.capabilities, nil
}

// MustBuild creates the MessagingCapabilities and panics if validation fails.
func (b *MessagingCapabilitiesBuilder) MustBuild() *MessagingCapabilities {
	capabilities, err := b.Build()
	if err != nil {
		panic(err)
	}
	return capabilities
}

// ConvertToHandlerMetadata converts ServiceMetadata to transport.HandlerMetadata.
// This enables integration with the existing transport layer.
//
// Parameters:
//   - serviceMetadata: Service metadata to convert
//   - healthEndpoint: Health check endpoint path
//
// Returns:
//   - *transport.HandlerMetadata: Converted handler metadata
func ConvertToHandlerMetadata(serviceMetadata *ServiceMetadata, healthEndpoint string) *transport.HandlerMetadata {
	if serviceMetadata == nil {
		return nil
	}

	return &transport.HandlerMetadata{
		Name:              serviceMetadata.Name,
		Version:           serviceMetadata.Version,
		Description:       serviceMetadata.Description,
		HealthEndpoint:    healthEndpoint,
		Tags:              serviceMetadata.Tags,
		Dependencies:      serviceMetadata.Dependencies,
		MessagingMetadata: serviceMetadata,
	}
}

// ExtractServiceMetadata extracts ServiceMetadata from transport.HandlerMetadata.
// Returns nil if no messaging metadata is present.
//
// Parameters:
//   - handlerMetadata: Handler metadata to extract from
//
// Returns:
//   - *ServiceMetadata: Extracted service metadata, or nil if not present
func ExtractServiceMetadata(handlerMetadata *transport.HandlerMetadata) *ServiceMetadata {
	if handlerMetadata == nil || handlerMetadata.MessagingMetadata == nil {
		return nil
	}

	serviceMetadata, ok := handlerMetadata.MessagingMetadata.(*ServiceMetadata)
	if !ok {
		return nil
	}

	return serviceMetadata
}

// NewRetryPolicy creates a default retry policy with sensible defaults.
//
// Parameters:
//   - maxRetries: Maximum number of retries
//   - initialInterval: Initial retry interval
//   - backoffType: Backoff strategy type
//
// Returns:
//   - *RetryPolicyMetadata: Retry policy configuration
func NewRetryPolicy(maxRetries int, initialInterval time.Duration, backoffType BackoffType) *RetryPolicyMetadata {
	policy := &RetryPolicyMetadata{
		MaxRetries:      maxRetries,
		InitialInterval: initialInterval,
		BackoffType:     backoffType,
	}

	// Set sensible defaults based on backoff type
	switch backoffType {
	case BackoffTypeExponential:
		policy.Multiplier = 2.0
		policy.MaxInterval = initialInterval * 60 // 60x initial
	case BackoffTypeLinear:
		policy.Multiplier = 1.0
		policy.MaxInterval = initialInterval * 10 // 10x initial
	case BackoffTypeConstant:
		policy.Multiplier = 1.0
		policy.MaxInterval = initialInterval
	}

	return policy
}

// NewEventMetadata creates a new event metadata with minimal required fields.
//
// Parameters:
//   - eventType: Event type identifier
//   - version: Event version
//
// Returns:
//   - EventMetadata: Event metadata configuration
func NewEventMetadata(eventType, version string) EventMetadata {
	return EventMetadata{
		EventType: eventType,
		Version:   version,
		Topics:    make([]string, 0),
		Examples:  make([]string, 0),
	}
}

// NewMessageSchema creates a new message schema with the specified type.
//
// Parameters:
//   - schemaType: Schema type (json-schema, protobuf, avro, custom)
//
// Returns:
//   - *MessageSchema: Message schema configuration
func NewMessageSchema(schemaType SchemaType) *MessageSchema {
	return &MessageSchema{
		Type:            schemaType,
		RequiredFields:  make([]string, 0),
		ValidationRules: make(map[string]interface{}),
	}
}
