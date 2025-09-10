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
	"fmt"
	"maps"
)

// CapabilityLevel defines the level of support for a capability.
type CapabilityLevel int

const (
	// CapabilityLevelNotSupported indicates the capability is not supported
	CapabilityLevelNotSupported CapabilityLevel = iota

	// CapabilityLevelBasic indicates basic support for the capability
	CapabilityLevelBasic

	// CapabilityLevelFull indicates full support for the capability
	CapabilityLevelFull

	// CapabilityLevelEnhanced indicates enhanced/optimized support
	CapabilityLevelEnhanced
)

// String returns the string representation of CapabilityLevel.
func (cl CapabilityLevel) String() string {
	switch cl {
	case CapabilityLevelNotSupported:
		return "not_supported"
	case CapabilityLevelBasic:
		return "basic"
	case CapabilityLevelFull:
		return "full"
	case CapabilityLevelEnhanced:
		return "enhanced"
	default:
		return "unknown"
	}
}

// IsSupported returns true if the capability is supported at any level.
func (cl CapabilityLevel) IsSupported() bool {
	return cl > CapabilityLevelNotSupported
}

// FeatureRequirement defines requirements for specific features.
type FeatureRequirement struct {
	// Name is the feature name
	Name string `json:"name"`

	// Level is the minimum required support level
	Level CapabilityLevel `json:"level"`

	// Optional indicates if this requirement is optional
	Optional bool `json:"optional"`

	// Fallback specifies alternative features if this one is not available
	Fallback []string `json:"fallback,omitempty"`

	// Description provides human-readable description of the requirement
	Description string `json:"description,omitempty"`
}

// CapabilityDetector defines the interface for capability detection.
// This allows brokers to implement dynamic capability detection.
type CapabilityDetector interface {
	// DetectCapabilities performs runtime capability detection.
	// This method can query the broker to determine actual capabilities.
	DetectCapabilities(ctx context.Context) (*BrokerCapabilities, error)

	// SupportsFeature checks if a specific feature is supported.
	SupportsFeature(ctx context.Context, feature string) (CapabilityLevel, error)

	// ValidateRequirements validates if the broker meets the specified requirements.
	ValidateRequirements(ctx context.Context, requirements []FeatureRequirement) (*ValidationResult, error)
}

// ValidationResult contains the result of capability validation.
type ValidationResult struct {
	// Valid indicates if all requirements are satisfied
	Valid bool `json:"valid"`

	// Satisfied contains requirements that were satisfied
	Satisfied []string `json:"satisfied"`

	// Missing contains requirements that were not satisfied
	Missing []string `json:"missing"`

	// Optional contains optional requirements that were not satisfied
	Optional []string `json:"optional"`

	// Fallbacks contains fallback features that could be used
	Fallbacks map[string][]string `json:"fallbacks,omitempty"`

	// Details provides additional validation details
	Details map[string]string `json:"details,omitempty"`
}

// CapabilityRegistry manages known broker capabilities and provides
// methods for querying and comparing capabilities.
type CapabilityRegistry interface {
	// RegisterBrokerCapabilities registers capabilities for a broker type.
	RegisterBrokerCapabilities(brokerType BrokerType, capabilities *BrokerCapabilities) error

	// GetBrokerCapabilities retrieves capabilities for a broker type.
	GetBrokerCapabilities(brokerType BrokerType) (*BrokerCapabilities, error)

	// FindCompatibleBrokers finds brokers that meet the specified requirements.
	FindCompatibleBrokers(requirements []FeatureRequirement) ([]BrokerType, error)

	// CompareCapabilities compares capabilities between different brokers.
	CompareCapabilities(broker1, broker2 BrokerType) (map[string]interface{}, error)

	// ListSupportedBrokers returns all registered broker types.
	ListSupportedBrokers() []BrokerType
}

// BrokerCapabilitiesBuilder provides a fluent interface for building
// BrokerCapabilities with validation and defaults.
type BrokerCapabilitiesBuilder struct {
	capabilities *BrokerCapabilities
	errors       []error
}

// NewBrokerCapabilitiesBuilder creates a new builder with default values.
func NewBrokerCapabilitiesBuilder() *BrokerCapabilitiesBuilder {
	return &BrokerCapabilitiesBuilder{
		capabilities: &BrokerCapabilities{
			Extended: make(map[string]any),
		},
	}
}

// WithTransactions sets transaction support.
func (b *BrokerCapabilitiesBuilder) WithTransactions(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsTransactions = supported
	return b
}

// WithOrdering sets message ordering support.
func (b *BrokerCapabilitiesBuilder) WithOrdering(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsOrdering = supported
	return b
}

// WithPartitioning sets partitioning support.
func (b *BrokerCapabilitiesBuilder) WithPartitioning(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsPartitioning = supported
	return b
}

// WithDeadLetter sets dead letter queue support.
func (b *BrokerCapabilitiesBuilder) WithDeadLetter(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsDeadLetter = supported
	return b
}

// WithDelayedDelivery sets delayed delivery support.
func (b *BrokerCapabilitiesBuilder) WithDelayedDelivery(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsDelayedDelivery = supported
	return b
}

// WithPriority sets message priority support.
func (b *BrokerCapabilitiesBuilder) WithPriority(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsPriority = supported
	return b
}

// WithStreaming sets streaming support.
func (b *BrokerCapabilitiesBuilder) WithStreaming(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsStreaming = supported
	return b
}

// WithSeek sets seek operation support.
func (b *BrokerCapabilitiesBuilder) WithSeek(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsSeek = supported
	return b
}

// WithConsumerGroups sets consumer group support.
func (b *BrokerCapabilitiesBuilder) WithConsumerGroups(supported bool) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportsConsumerGroups = supported
	return b
}

// WithMaxMessageSize sets maximum message size.
func (b *BrokerCapabilitiesBuilder) WithMaxMessageSize(size int64) *BrokerCapabilitiesBuilder {
	if size < 0 {
		b.errors = append(b.errors, errors.New("max message size cannot be negative"))
		return b
	}
	b.capabilities.MaxMessageSize = size
	return b
}

// WithMaxBatchSize sets maximum batch size.
func (b *BrokerCapabilitiesBuilder) WithMaxBatchSize(size int) *BrokerCapabilitiesBuilder {
	if size < 0 {
		b.errors = append(b.errors, errors.New("max batch size cannot be negative"))
		return b
	}
	b.capabilities.MaxBatchSize = size
	return b
}

// WithMaxTopicNameLength sets maximum topic name length.
func (b *BrokerCapabilitiesBuilder) WithMaxTopicNameLength(length int) *BrokerCapabilitiesBuilder {
	if length < 0 {
		b.errors = append(b.errors, errors.New("max topic name length cannot be negative"))
		return b
	}
	b.capabilities.MaxTopicNameLength = length
	return b
}

// WithCompressionTypes sets supported compression types.
func (b *BrokerCapabilitiesBuilder) WithCompressionTypes(types ...CompressionType) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportedCompressionTypes = make([]CompressionType, len(types))
	copy(b.capabilities.SupportedCompressionTypes, types)
	return b
}

// WithSerializationTypes sets supported serialization types.
func (b *BrokerCapabilitiesBuilder) WithSerializationTypes(types ...SerializationType) *BrokerCapabilitiesBuilder {
	b.capabilities.SupportedSerializationTypes = make([]SerializationType, len(types))
	copy(b.capabilities.SupportedSerializationTypes, types)
	return b
}

// WithExtendedCapability adds an extended capability.
func (b *BrokerCapabilitiesBuilder) WithExtendedCapability(key string, value any) *BrokerCapabilitiesBuilder {
	if b.capabilities.Extended == nil {
		b.capabilities.Extended = make(map[string]any)
	}
	b.capabilities.Extended[key] = value
	return b
}

// Build creates the final BrokerCapabilities instance.
func (b *BrokerCapabilitiesBuilder) Build() (*BrokerCapabilities, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("capability builder errors: %v", b.errors)
	}

	// Create a deep copy to prevent modifications to the builder
	result := *b.capabilities
	if b.capabilities.Extended != nil {
		result.Extended = make(map[string]any)
		maps.Copy(result.Extended, b.capabilities.Extended)
	}

	// Set default compression types if none specified
	if len(result.SupportedCompressionTypes) == 0 {
		result.SupportedCompressionTypes = []CompressionType{CompressionNone}
	}

	// Set default serialization types if none specified
	if len(result.SupportedSerializationTypes) == 0 {
		result.SupportedSerializationTypes = []SerializationType{SerializationJSON}
	}

	return &result, nil
}

// GetCapabilityProfile returns a standardized capability profile for common broker types.
func GetCapabilityProfile(brokerType BrokerType) (*BrokerCapabilities, error) {
	switch brokerType {
	case BrokerTypeKafka:
		return NewBrokerCapabilitiesBuilder().
			WithTransactions(true).
			WithOrdering(true).
			WithPartitioning(true).
			WithDeadLetter(false). // Kafka doesn't have native DLQ, requires custom implementation
			WithDelayedDelivery(false).
			WithPriority(false).
			WithStreaming(true).
			WithSeek(true).
			WithConsumerGroups(true).
			WithMaxMessageSize(1024*1024). // 1MB default
			WithMaxBatchSize(1000).        // 1000 messages
			WithMaxTopicNameLength(255).   // Kafka limit
			WithCompressionTypes(CompressionNone, CompressionGZIP, CompressionSnappy, CompressionLZ4, CompressionZSTD).
			WithSerializationTypes(SerializationJSON, SerializationProtobuf, SerializationAvro).
			WithExtendedCapability("kafka.exactly_once_semantics", true).
			WithExtendedCapability("kafka.compaction", true).
			WithExtendedCapability("kafka.replication", true).
			Build()

	case BrokerTypeNATS:
		return NewBrokerCapabilitiesBuilder().
			WithTransactions(false). // NATS Streaming has limited transaction support
			WithOrdering(true).      // Within subject ordering
			WithPartitioning(false). // No native partitioning
			WithDeadLetter(false).   // No native DLQ
			WithDelayedDelivery(false).
			WithPriority(false).
			WithStreaming(true). // NATS JetStream
			WithSeek(true).      // JetStream supports replay
			WithConsumerGroups(true).
			WithMaxMessageSize(64*1024*1024). // 64MB default
			WithMaxBatchSize(0).              // No hard limit
			WithMaxTopicNameLength(0).        // No hard limit
			WithCompressionTypes(CompressionNone).
			WithSerializationTypes(SerializationJSON, SerializationProtobuf, SerializationMsgPack).
			WithExtendedCapability("nats.jetstream", true).
			WithExtendedCapability("nats.key_value_store", true).
			WithExtendedCapability("nats.object_store", true).
			Build()

	case BrokerTypeRabbitMQ:
		return NewBrokerCapabilitiesBuilder().
			WithTransactions(true). // RabbitMQ publisher confirms
			WithOrdering(false).    // No guaranteed ordering across queues
			WithPartitioning(false).
			WithDeadLetter(true). // Native DLX support
			WithDelayedDelivery(true).
			WithPriority(true). // Message priority support
			WithStreaming(false).
			WithSeek(false). // No seek capability
			WithConsumerGroups(true).
			WithMaxMessageSize(128*1024*1024). // 128MB default
			WithMaxBatchSize(0).               // No hard limit
			WithMaxTopicNameLength(255).       // Queue/exchange name limit
			WithCompressionTypes(CompressionNone, CompressionGZIP).
			WithSerializationTypes(SerializationJSON, SerializationProtobuf, SerializationMsgPack).
			WithExtendedCapability("rabbitmq.exchanges", true).
			WithExtendedCapability("rabbitmq.routing_keys", true).
			WithExtendedCapability("rabbitmq.ttl", true).
			Build()

	case BrokerTypeInMemory:
		return NewBrokerCapabilitiesBuilder().
			WithTransactions(true). // Mock implementation can support all features
			WithOrdering(true).
			WithPartitioning(true).
			WithDeadLetter(true).
			WithDelayedDelivery(true).
			WithPriority(true).
			WithStreaming(true).
			WithSeek(true).
			WithConsumerGroups(true).
			WithMaxMessageSize(0). // No limits for testing
			WithMaxBatchSize(0).
			WithMaxTopicNameLength(0).
			WithCompressionTypes(CompressionNone, CompressionGZIP).
			WithSerializationTypes(SerializationJSON, SerializationProtobuf, SerializationAvro, SerializationMsgPack).
			WithExtendedCapability("inmemory.persistence", false).
			WithExtendedCapability("inmemory.testing_mode", true).
			Build()

	default:
		return nil, fmt.Errorf("unknown broker type: %s", brokerType)
	}
}

// SupportsFeature checks if a specific feature is supported by the capabilities.
func (bc *BrokerCapabilities) SupportsFeature(feature string) CapabilityLevel {
	switch feature {
	case "transactions":
		if bc.SupportsTransactions {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "ordering":
		if bc.SupportsOrdering {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "partitioning":
		if bc.SupportsPartitioning {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "dead_letter":
		if bc.SupportsDeadLetter {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "delayed_delivery":
		if bc.SupportsDelayedDelivery {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "priority":
		if bc.SupportsPriority {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "streaming":
		if bc.SupportsStreaming {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "seek":
		if bc.SupportsSeek {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	case "consumer_groups":
		if bc.SupportsConsumerGroups {
			return CapabilityLevelFull
		}
		return CapabilityLevelNotSupported

	default:
		// Check extended capabilities
		if bc.Extended != nil {
			if val, exists := bc.Extended[feature]; exists {
				if supported, ok := val.(bool); ok && supported {
					return CapabilityLevelFull
				}
			}
		}
		return CapabilityLevelNotSupported
	}
}

// ValidateRequirements validates if the capabilities meet the specified requirements.
func (bc *BrokerCapabilities) ValidateRequirements(requirements []FeatureRequirement) *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Details:   make(map[string]string),
		Fallbacks: make(map[string][]string),
	}

	for _, req := range requirements {
		level := bc.SupportsFeature(req.Name)

		if level >= req.Level {
			result.Satisfied = append(result.Satisfied, req.Name)
			result.Details[req.Name] = fmt.Sprintf("satisfied with %s support", level.String())
		} else if req.Optional {
			result.Optional = append(result.Optional, req.Name)
			result.Details[req.Name] = fmt.Sprintf("optional feature not supported (required: %s, available: %s)", req.Level.String(), level.String())

			// Check fallbacks
			if len(req.Fallback) > 0 {
				availableFallbacks := make([]string, 0, len(req.Fallback))
				for _, fallback := range req.Fallback {
					if bc.SupportsFeature(fallback).IsSupported() {
						availableFallbacks = append(availableFallbacks, fallback)
					}
				}
				if len(availableFallbacks) > 0 {
					result.Fallbacks[req.Name] = availableFallbacks
				}
			}
		} else {
			result.Valid = false
			result.Missing = append(result.Missing, req.Name)
			result.Details[req.Name] = fmt.Sprintf("required feature not supported (required: %s, available: %s)", req.Level.String(), level.String())

			// Check fallbacks even for required features
			if len(req.Fallback) > 0 {
				availableFallbacks := make([]string, 0, len(req.Fallback))
				for _, fallback := range req.Fallback {
					if bc.SupportsFeature(fallback).IsSupported() {
						availableFallbacks = append(availableFallbacks, fallback)
					}
				}
				if len(availableFallbacks) > 0 {
					result.Fallbacks[req.Name] = availableFallbacks
				}
			}
		}
	}

	return result
}

// GetExtendedCapability returns an extended capability value.
func (bc *BrokerCapabilities) GetExtendedCapability(key string) (any, bool) {
	if bc.Extended == nil {
		return nil, false
	}
	val, exists := bc.Extended[key]
	return val, exists
}

// Clone creates a deep copy of the BrokerCapabilities.
func (bc *BrokerCapabilities) Clone() *BrokerCapabilities {
	if bc == nil {
		return nil
	}

	clone := *bc

	// Deep copy slices
	if bc.SupportedCompressionTypes != nil {
		clone.SupportedCompressionTypes = make([]CompressionType, len(bc.SupportedCompressionTypes))
		copy(clone.SupportedCompressionTypes, bc.SupportedCompressionTypes)
	}

	if bc.SupportedSerializationTypes != nil {
		clone.SupportedSerializationTypes = make([]SerializationType, len(bc.SupportedSerializationTypes))
		copy(clone.SupportedSerializationTypes, bc.SupportedSerializationTypes)
	}

	// Deep copy extended map
	if bc.Extended != nil {
		clone.Extended = make(map[string]any)
		maps.Copy(clone.Extended, bc.Extended)
	}

	return &clone
}
