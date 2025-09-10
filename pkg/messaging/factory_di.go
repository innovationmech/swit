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
	"fmt"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MessagingConfig represents the configuration for messaging components
type MessagingConfig struct {
	Enabled bool                     `json:"enabled"`
	Brokers map[string]*BrokerConfig `json:"brokers"`
}

// DependencyContainerProvider defines the interface for accessing dependency containers
// This interface allows messaging components to access the server's dependency injection system
type DependencyContainerProvider interface {
	// GetService retrieves a service by name from the container
	GetService(name string) (interface{}, error)
	// Close closes the container and releases resources
	Close() error
}

// MessagingCoordinatorFactory creates a messaging coordinator with dependency injection support
// This factory integrates with the server framework's dependency container
func CreateMessagingCoordinatorFactory() func(container DependencyContainerProvider) (interface{}, error) {
	return func(container DependencyContainerProvider) (interface{}, error) {
		// Create a new messaging coordinator instance
		coordinator := NewMessagingCoordinator()

		// Create a DI-aware wrapper that can inject dependencies
		wrapper := &dependencyInjectedCoordinator{
			MessagingCoordinator: coordinator,
			container:            container,
		}

		logger.Logger.Info("Created messaging coordinator with dependency injection support")
		return wrapper, nil
	}
}

// BrokerFactoryWithDI creates a broker factory that supports dependency injection
func CreateBrokerFactory(brokerType string) func(container DependencyContainerProvider, config interface{}) (interface{}, error) {
	return func(container DependencyContainerProvider, config interface{}) (interface{}, error) {
		brokerConfig, ok := config.(*BrokerConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for broker factory: expected *BrokerConfig, got %T", config)
		}

		// Validate broker type matches
		if brokerConfig.Type.String() != brokerType {
			return nil, fmt.Errorf("broker type mismatch: expected %s, got %s", brokerType, brokerConfig.Type)
		}

		// Create broker using the default factory
		broker, err := NewMessageBroker(brokerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create broker: %w", err)
		}

		// Create a DI-aware wrapper if the broker needs dependencies
		wrapper := &dependencyInjectedBroker{
			MessageBroker: broker,
			container:     container,
		}

		logger.Logger.Info("Created broker with dependency injection support",
			zap.String("broker_type", brokerType))
		return wrapper, nil
	}
}

// EventHandlerFactoryWithDI creates an event handler factory that supports dependency injection
func CreateEventHandlerFactory(handlerType string, handlerConstructor func(container DependencyContainerProvider) (EventHandler, error)) func(container DependencyContainerProvider) (interface{}, error) {
	return func(container DependencyContainerProvider) (interface{}, error) {
		// Use the provided constructor to create the handler with DI
		handler, err := handlerConstructor(container)
		if err != nil {
			return nil, fmt.Errorf("failed to create event handler: %w", err)
		}

		// Create a DI-aware wrapper
		wrapper := &dependencyInjectedEventHandler{
			EventHandler: handler,
			container:    container,
		}

		logger.Logger.Info("Created event handler with dependency injection support",
			zap.String("handler_type", handlerType))
		return wrapper, nil
	}
}

// dependencyInjectedCoordinator wraps MessagingCoordinator with dependency injection capabilities
type dependencyInjectedCoordinator struct {
	MessagingCoordinator
	container DependencyContainerProvider
}

// Start overrides the base Start method to inject dependencies into registered handlers
func (d *dependencyInjectedCoordinator) Start(ctx context.Context) error {
	logger.Logger.Debug("Starting messaging coordinator with dependency injection")

	// Start the underlying coordinator
	return d.MessagingCoordinator.Start(ctx)
}

// RegisterEventHandler overrides to support dependency injection for event handlers
func (d *dependencyInjectedCoordinator) RegisterEventHandler(handler EventHandler) error {
	// Check if handler needs dependency injection
	if diHandler, ok := handler.(DependencyAware); ok {
		if err := diHandler.InjectDependencies(d.container); err != nil {
			return fmt.Errorf("failed to inject dependencies into handler: %w", err)
		}
	}

	return d.MessagingCoordinator.RegisterEventHandler(handler)
}

// dependencyInjectedBroker wraps MessageBroker with dependency injection capabilities
type dependencyInjectedBroker struct {
	MessageBroker
	container DependencyContainerProvider
}

// Connect overrides to inject dependencies before connecting
func (d *dependencyInjectedBroker) Connect(ctx context.Context) error {
	logger.Logger.Debug("Connecting broker with dependency injection")

	// Check if broker needs dependency injection
	if diBroker, ok := d.MessageBroker.(DependencyAware); ok {
		if err := diBroker.InjectDependencies(d.container); err != nil {
			return fmt.Errorf("failed to inject dependencies into broker: %w", err)
		}
	}

	return d.MessageBroker.Connect(ctx)
}

// dependencyInjectedEventHandler wraps EventHandler with dependency injection capabilities
type dependencyInjectedEventHandler struct {
	EventHandler
	container DependencyContainerProvider
}

// Initialize overrides to inject dependencies before initializing
func (d *dependencyInjectedEventHandler) Initialize(ctx context.Context) error {
	logger.Logger.Debug("Initializing event handler with dependency injection",
		zap.String("handler_id", d.EventHandler.GetHandlerID()))

	// Check if handler needs dependency injection
	if diHandler, ok := d.EventHandler.(DependencyAware); ok {
		if err := diHandler.InjectDependencies(d.container); err != nil {
			return fmt.Errorf("failed to inject dependencies into event handler: %w", err)
		}
	}

	return d.EventHandler.Initialize(ctx)
}

// DependencyAware interface for components that need dependency injection
type DependencyAware interface {
	// InjectDependencies allows components to receive dependencies from the container
	InjectDependencies(container DependencyContainerProvider) error
}

// MessagingConfigurationProvider creates a factory for injecting messaging configuration
func CreateMessagingConfigurationProvider(config *MessagingConfig) func(container DependencyContainerProvider) (interface{}, error) {
	return func(container DependencyContainerProvider) (interface{}, error) {
		if config == nil {
			return nil, fmt.Errorf("messaging configuration cannot be nil")
		}

		// Return a copy to prevent external modification
		configCopy := *config

		logger.Logger.Debug("Providing messaging configuration for dependency injection")
		return &configCopy, nil
	}
}

// BrokerConfigurationProvider creates a factory for injecting broker configurations
func CreateBrokerConfigurationProvider(configs map[string]*BrokerConfig) func(container DependencyContainerProvider) (interface{}, error) {
	return func(container DependencyContainerProvider) (interface{}, error) {
		if configs == nil {
			return nil, fmt.Errorf("broker configurations cannot be nil")
		}

		// Return a copy to prevent external modification
		configsCopy := make(map[string]*BrokerConfig)
		for name, config := range configs {
			if config != nil {
				configCopy := *config
				configsCopy[name] = &configCopy
			}
		}

		logger.Logger.Debug("Providing broker configurations for dependency injection",
			zap.Int("config_count", len(configsCopy)))
		return configsCopy, nil
	}
}
