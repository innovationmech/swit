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
	"fmt"
	"sort"
	"strings"
	"sync"
)

// messageBrokerFactoryImpl implements the MessageBrokerFactory interface.
// It provides a registry for different broker implementations and handles
// their creation based on configuration. This implementation now supports
// both legacy factory functions and the new adapter pattern.
type messageBrokerFactoryImpl struct {
	// factories maps broker types to their factory functions (legacy)
	factories map[BrokerType]func(*BrokerConfig) (MessageBroker, error)

	// adapters provides access to the adapter registry for new implementations
	adapters AdapterRegistryProvider

	// mu protects concurrent access to the factories map
	mu sync.RWMutex
}

// AdapterRegistryProvider provides access to broker adapter registry.
// This interface allows the factory to work with different registry implementations.
type AdapterRegistryProvider interface {
	GetAdapterForBrokerType(brokerType BrokerType) (MessageBrokerAdapter, error)
	GetSupportedBrokerTypes() []BrokerType
	ValidateConfiguration(config *BrokerConfig) (*AdapterValidationResult, error)
}

// defaultFactory is the default factory instance.
var defaultFactory = &messageBrokerFactoryImpl{
	factories: make(map[BrokerType]func(*BrokerConfig) (MessageBroker, error)),
}

// NewMessageBrokerFactory creates a new MessageBrokerFactory instance.
// This allows creating custom factories for testing or specialized use cases.
func NewMessageBrokerFactory() MessageBrokerFactory {
	return &messageBrokerFactoryImpl{
		factories: make(map[BrokerType]func(*BrokerConfig) (MessageBroker, error)),
	}
}

// GetDefaultFactory returns the default factory instance.
// This factory includes all registered broker implementations.
func GetDefaultFactory() MessageBrokerFactory {
	return defaultFactory
}

// RegisterBrokerFactory registers a factory function for a specific broker type.
// This allows pluggable broker implementations to be added at runtime.
//
// Parameters:
//   - brokerType: The broker type to register
//   - factory: Factory function that creates broker instances
//
// Example:
//
//	RegisterBrokerFactory(BrokerTypeKafka, func(config *BrokerConfig) (MessageBroker, error) {
//	    return kafka.NewKafkaBroker(config)
//	})
func RegisterBrokerFactory(brokerType BrokerType, factory func(*BrokerConfig) (MessageBroker, error)) {
	defaultFactory.RegisterBrokerFactory(brokerType, factory)
}

// RegisterBrokerFactory registers a factory function for a specific broker type.
func (f *messageBrokerFactoryImpl) RegisterBrokerFactory(brokerType BrokerType, factory func(*BrokerConfig) (MessageBroker, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.factories[brokerType] = factory
}

// CreateBroker creates a new MessageBroker instance with the provided configuration.
func (f *messageBrokerFactoryImpl) CreateBroker(config *BrokerConfig) (MessageBroker, error) {
	if config == nil {
		return nil, NewConfigError("broker config cannot be nil", nil)
	}

	// Validate configuration
	if err := f.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Try adapter registry first (new pattern)
	if f.adapters != nil {
		if adapter, err := f.adapters.GetAdapterForBrokerType(config.Type); err == nil {
			broker, err := adapter.CreateBroker(config)
			if err != nil {
				return nil, fmt.Errorf("adapter failed to create broker: %w", err)
			}
			return broker, nil
		}
	}

	// Fall back to legacy factory functions
	f.mu.RLock()
	factory, exists := f.factories[config.Type]
	f.mu.RUnlock()

	if !exists {
		// Enrich error with supported types for better guidance
		supported := f.GetSupportedBrokerTypes()
		supportedStrs := make([]string, 0, len(supported))
		for _, t := range supported {
			supportedStrs = append(supportedStrs, t.String())
		}
		sort.Strings(supportedStrs)

		return nil, NewConfigError(
			fmt.Sprintf("unsupported broker type: %s (supported: %s)", config.Type, strings.Join(supportedStrs, ", ")),
			nil,
		)
	}

	broker, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create broker: %w", err)
	}

	return broker, nil
}

// GetSupportedBrokerTypes returns a list of broker types supported by this factory.
func (f *messageBrokerFactoryImpl) GetSupportedBrokerTypes() []BrokerType {
	typeSet := make(map[BrokerType]bool)

	// Add types from adapter registry
	if f.adapters != nil {
		adapterTypes := f.adapters.GetSupportedBrokerTypes()
		for _, brokerType := range adapterTypes {
			typeSet[brokerType] = true
		}
	}

	// Add types from legacy factories
	f.mu.RLock()
	factoryKeys := make([]BrokerType, 0, len(f.factories))
	for brokerType := range f.factories {
		factoryKeys = append(factoryKeys, brokerType)
	}
	f.mu.RUnlock()
	for _, brokerType := range factoryKeys {
		typeSet[brokerType] = true
	}

	// Convert to slice
	types := make([]BrokerType, 0, len(typeSet))
	for brokerType := range typeSet {
		types = append(types, brokerType)
	}

	return types
}

// ValidateConfig validates a broker configuration without creating a broker instance.
func (f *messageBrokerFactoryImpl) ValidateConfig(config *BrokerConfig) error {
	if config == nil {
		return NewConfigError("config cannot be nil", nil)
	}

	// Try adapter registry first for more detailed validation
	if f.adapters != nil {
		if result, err := f.adapters.ValidateConfiguration(config); err == nil {
			if !result.Valid {
				// Compose detailed validation error message with field-level details
				var parts []string
				for _, e := range result.Errors {
					if e.Field != "" {
						parts = append(parts, fmt.Sprintf("%s: %s", e.Field, e.Message))
					} else {
						parts = append(parts, e.Message)
					}
				}
				msg := "adapter configuration validation failed"
				if len(parts) > 0 {
					msg = fmt.Sprintf("%s: %s", msg, strings.Join(parts, "; "))
				}
				return NewConfigError(msg, nil)
			}
			return nil // Valid according to adapter
		}
	}

	// Fall back to legacy validation
	f.mu.RLock()
	_, supported := f.factories[config.Type]
	f.mu.RUnlock()

	if !supported {
		// Enrich error with supported types for better guidance
		supportedTypes := f.GetSupportedBrokerTypes()
		names := make([]string, 0, len(supportedTypes))
		for _, t := range supportedTypes {
			names = append(names, t.String())
		}
		sort.Strings(names)
		return NewConfigError(
			fmt.Sprintf("unsupported broker type: %s (supported: %s)", config.Type, strings.Join(names, ", ")),
			nil,
		)
	}

	// Delegate to the configuration's validation method
	return config.Validate()
}

// NewMessageBroker creates a new MessageBroker using the default factory.
// This is a convenience function for the most common use case.
//
// Parameters:
//   - config: Broker configuration
//
// Returns:
//   - MessageBroker: Configured broker instance
//   - error: Configuration or creation error
//
// Example:
//
//	config := &BrokerConfig{
//	    Type: BrokerTypeKafka,
//	    Endpoints: []string{"localhost:9092"},
//	}
//	broker, err := NewMessageBroker(config)
//	if err != nil {
//	    return err
//	}
func NewMessageBroker(config *BrokerConfig) (MessageBroker, error) {
	return defaultFactory.CreateBroker(config)
}

// GetSupportedBrokerTypes returns the list of supported broker types from the default factory.
func GetSupportedBrokerTypes() []BrokerType {
	return defaultFactory.GetSupportedBrokerTypes()
}

// ValidateBrokerConfig validates a broker configuration using the default factory.
func ValidateBrokerConfig(config *BrokerConfig) error {
	return defaultFactory.ValidateConfig(config)
}

// SetDefaultAdapterRegistry configures the default factory to use an adapter registry.
// This enables the factory to use the adapter pattern for broker creation.
func SetDefaultAdapterRegistry(registry AdapterRegistryProvider) {
	defaultFactory.mu.Lock()
	defer defaultFactory.mu.Unlock()
	defaultFactory.adapters = registry
}

// GetDefaultAdapterRegistry returns the adapter registry used by the default factory.
func GetDefaultAdapterRegistry() AdapterRegistryProvider {
	defaultFactory.mu.RLock()
	defer defaultFactory.mu.RUnlock()
	return defaultFactory.adapters
}

// NewMessageBrokerFactoryWithAdapters creates a factory with adapter registry support.
func NewMessageBrokerFactoryWithAdapters(registry AdapterRegistryProvider) MessageBrokerFactory {
	return &messageBrokerFactoryImpl{
		factories: make(map[BrokerType]func(*BrokerConfig) (MessageBroker, error)),
		adapters:  registry,
	}
}
