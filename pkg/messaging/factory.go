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
	"fmt"
	"sync"
)

// messageBrokerFactoryImpl implements the MessageBrokerFactory interface.
// It provides a registry for different broker implementations and handles
// their creation based on configuration.
type messageBrokerFactoryImpl struct {
	// factories maps broker types to their factory functions
	factories map[BrokerType]func(*BrokerConfig) (MessageBroker, error)

	// mu protects concurrent access to the factories map
	mu sync.RWMutex
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

	f.mu.RLock()
	factory, exists := f.factories[config.Type]
	f.mu.RUnlock()

	if !exists {
		return nil, NewConfigError(
			fmt.Sprintf("unsupported broker type: %s", config.Type),
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
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]BrokerType, 0, len(f.factories))
	for brokerType := range f.factories {
		types = append(types, brokerType)
	}

	return types
}

// ValidateConfig validates a broker configuration without creating a broker instance.
func (f *messageBrokerFactoryImpl) ValidateConfig(config *BrokerConfig) error {
	if config == nil {
		return NewConfigError("config cannot be nil", nil)
	}

	// Validate that the broker type is supported
	f.mu.RLock()
	_, supported := f.factories[config.Type]
	f.mu.RUnlock()

	if !supported {
		return NewConfigError(
			fmt.Sprintf("unsupported broker type: %s", config.Type),
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
