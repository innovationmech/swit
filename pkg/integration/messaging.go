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
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/server"
)

// MessagingServiceRegistrar integrates the messaging system with the SWIT framework.
// It implements the BusinessServiceRegistrar interface to enable messaging services
// to be registered with the server's transport layer and dependency injection system.
type MessagingServiceRegistrar struct {
	broker    messaging.MessageBroker
	publisher messaging.EventPublisher
	config    *messaging.BrokerConfig
}

// NewMessagingServiceRegistrar creates a new messaging service registrar.
// This registrar integrates messaging components with the SWIT framework patterns.
//
// Parameters:
//   - config: Broker configuration for the messaging system
//
// Returns:
//   - *MessagingServiceRegistrar: Configured messaging service registrar
//   - error: Configuration or initialization error
func NewMessagingServiceRegistrar(config *messaging.BrokerConfig) (*MessagingServiceRegistrar, error) {
	if config == nil {
		return nil, messaging.NewConfigError("broker config cannot be nil", nil)
	}

	// Validate configuration
	if err := messaging.ValidateBrokerConfig(config); err != nil {
		return nil, fmt.Errorf("invalid broker configuration: %w", err)
	}

	return &MessagingServiceRegistrar{
		config: config,
	}, nil
}

// RegisterServices registers messaging-related services with the SWIT framework.
// This method implements the BusinessServiceRegistrar interface and sets up
// messaging system integration with the server's dependency injection container
// and health check system.
func (m *MessagingServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register messaging health check
	healthCheck := &MessagingHealthCheck{
		serviceName: "messaging-system",
		config:      m.config,
	}

	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register messaging health check: %w", err)
	}

	// Note: HTTP and gRPC services for messaging would be registered here
	// if the messaging system needs to expose HTTP/gRPC endpoints
	// For now, the messaging system primarily provides programmatic interfaces

	return nil
}

// MessagingHealthCheck implements the BusinessHealthCheck interface
// to provide health checking for the messaging system within the SWIT framework.
type MessagingHealthCheck struct {
	serviceName string
	config      *BrokerConfig
	broker      MessageBroker
}

// Check performs a health check on the messaging system.
// It verifies that the broker connection is healthy and operational.
func (h *MessagingHealthCheck) Check(ctx context.Context) error {
	// If broker is not initialized yet, try to create one for health check
	if h.broker == nil {
		broker, err := NewMessageBroker(h.config)
		if err != nil {
			return fmt.Errorf("messaging system unavailable: failed to create broker: %w", err)
		}
		h.broker = broker
	}

	// Create a context with timeout for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify broker connection health
	if err := h.broker.Connect(checkCtx); err != nil {
		return fmt.Errorf("messaging broker connection failed: %w", err)
	}

	return nil
}

// GetServiceName returns the service name for the messaging health check.
func (h *MessagingHealthCheck) GetServiceName() string {
	return h.serviceName
}

// MessagingDependencyFactory provides factory functions for integrating
// messaging components with the SWIT dependency injection system.
type MessagingDependencyFactory struct{}

// CreateBrokerFactory returns a dependency factory function for MessageBroker instances.
// This factory integrates with the SWIT dependency injection container.
//
// Parameters:
//   - config: Broker configuration
//
// Returns:
//   - server.DependencyFactory: Factory function for dependency injection
func (f *MessagingDependencyFactory) CreateBrokerFactory(config *BrokerConfig) server.DependencyFactory {
	return func(container server.BusinessDependencyContainer) (interface{}, error) {
		if config == nil {
			return nil, NewConfigError("broker config cannot be nil for dependency factory", nil)
		}

		broker, err := NewMessageBroker(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create message broker dependency: %w", err)
		}

		return broker, nil
	}
}

// CreatePublisherFactory returns a dependency factory function for EventPublisher instances.
// This factory integrates with the SWIT dependency injection container.
//
// Parameters:
//   - config: Publisher configuration
//   - brokerServiceName: Name of the broker service in the dependency container
//
// Returns:
//   - server.DependencyFactory: Factory function for dependency injection
func (f *MessagingDependencyFactory) CreatePublisherFactory(config *PublisherConfig, brokerServiceName string) server.DependencyFactory {
	return func(container server.BusinessDependencyContainer) (interface{}, error) {
		if config == nil {
			return nil, NewConfigError("publisher config cannot be nil for dependency factory", nil)
		}

		// Retrieve broker from dependency container
		brokerInterface, err := container.GetService(brokerServiceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get broker dependency '%s': %w", brokerServiceName, err)
		}

		broker, ok := brokerInterface.(MessageBroker)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' is not a MessageBroker", brokerServiceName)
		}

		publisher, err := broker.CreatePublisher(*config)
		if err != nil {
			return nil, fmt.Errorf("failed to create event publisher dependency: %w", err)
		}

		return publisher, nil
	}
}

// CreateSubscriberFactory returns a dependency factory function for EventSubscriber instances.
// This factory integrates with the SWIT dependency injection container.
//
// Parameters:
//   - config: Subscriber configuration
//   - brokerServiceName: Name of the broker service in the dependency container
//
// Returns:
//   - server.DependencyFactory: Factory function for dependency injection
func (f *MessagingDependencyFactory) CreateSubscriberFactory(config *SubscriberConfig, brokerServiceName string) server.DependencyFactory {
	return func(container server.BusinessDependencyContainer) (interface{}, error) {
		if config == nil {
			return nil, NewConfigError("subscriber config cannot be nil for dependency factory", nil)
		}

		// Retrieve broker from dependency container
		brokerInterface, err := container.GetService(brokerServiceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get broker dependency '%s': %w", brokerServiceName, err)
		}

		broker, ok := brokerInterface.(MessageBroker)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' is not a MessageBroker", brokerServiceName)
		}

		subscriber, err := broker.CreateSubscriber(*config)
		if err != nil {
			return nil, fmt.Errorf("failed to create event subscriber dependency: %w", err)
		}

		return subscriber, nil
	}
}

// MessagingServiceConfigurer provides convenience methods for configuring
// messaging services within the SWIT framework dependency injection system.
type MessagingServiceConfigurer struct {
	factory *MessagingDependencyFactory
}

// NewMessagingServiceConfigurer creates a new messaging service configurer.
func NewMessagingServiceConfigurer() *MessagingServiceConfigurer {
	return &MessagingServiceConfigurer{
		factory: &MessagingDependencyFactory{},
	}
}

// RegisterMessagingDependencies registers all messaging-related dependencies
// with the provided dependency registry. This method follows SWIT framework
// patterns for dependency registration and lifecycle management.
//
// Parameters:
//   - registry: SWIT dependency registry
//   - brokerConfig: Configuration for the message broker
//   - publisherConfig: Configuration for the event publisher (optional)
//   - subscriberConfig: Configuration for the event subscriber (optional)
//
// Returns:
//   - error: Registration error if any dependency fails to register
//
// Example usage:
//
//	configurer := NewMessagingServiceConfigurer()
//	err := configurer.RegisterMessagingDependencies(
//	    dependencyRegistry,
//	    &BrokerConfig{Type: BrokerTypeInMemory},
//	    &PublisherConfig{BatchSize: 100},
//	    &SubscriberConfig{Topic: "events"},
//	)
func (c *MessagingServiceConfigurer) RegisterMessagingDependencies(
	registry server.BusinessDependencyRegistry,
	brokerConfig *BrokerConfig,
	publisherConfig *PublisherConfig,
	subscriberConfig *SubscriberConfig,
) error {
	// Register message broker as singleton
	brokerFactory := c.factory.CreateBrokerFactory(brokerConfig)
	if err := registry.RegisterSingleton("message-broker", brokerFactory); err != nil {
		return fmt.Errorf("failed to register message broker dependency: %w", err)
	}

	// Register event publisher if configuration provided
	if publisherConfig != nil {
		publisherFactory := c.factory.CreatePublisherFactory(publisherConfig, "message-broker")
		if err := registry.RegisterSingleton("event-publisher", publisherFactory); err != nil {
			return fmt.Errorf("failed to register event publisher dependency: %w", err)
		}
	}

	// Register event subscriber if configuration provided
	if subscriberConfig != nil {
		subscriberFactory := c.factory.CreateSubscriberFactory(subscriberConfig, "message-broker")
		if err := registry.RegisterSingleton("event-subscriber", subscriberFactory); err != nil {
			return fmt.Errorf("failed to register event subscriber dependency: %w", err)
		}
	}

	return nil
}

// GetBroker retrieves the message broker from the dependency container.
// This is a convenience method that follows SWIT framework patterns.
func (c *MessagingServiceConfigurer) GetBroker(container server.BusinessDependencyContainer) (MessageBroker, error) {
	brokerInterface, err := container.GetService("message-broker")
	if err != nil {
		return nil, fmt.Errorf("failed to get message broker: %w", err)
	}

	broker, ok := brokerInterface.(MessageBroker)
	if !ok {
		return nil, fmt.Errorf("dependency 'message-broker' is not a MessageBroker")
	}

	return broker, nil
}

// GetPublisher retrieves the event publisher from the dependency container.
// This is a convenience method that follows SWIT framework patterns.
func (c *MessagingServiceConfigurer) GetPublisher(container server.BusinessDependencyContainer) (EventPublisher, error) {
	publisherInterface, err := container.GetService("event-publisher")
	if err != nil {
		return nil, fmt.Errorf("failed to get event publisher: %w", err)
	}

	publisher, ok := publisherInterface.(EventPublisher)
	if !ok {
		return nil, fmt.Errorf("dependency 'event-publisher' is not an EventPublisher")
	}

	return publisher, nil
}

// GetSubscriber retrieves the event subscriber from the dependency container.
// This is a convenience method that follows SWIT framework patterns.
func (c *MessagingServiceConfigurer) GetSubscriber(container server.BusinessDependencyContainer) (EventSubscriber, error) {
	subscriberInterface, err := container.GetService("event-subscriber")
	if err != nil {
		return nil, fmt.Errorf("failed to get event subscriber: %w", err)
	}

	subscriber, ok := subscriberInterface.(EventSubscriber)
	if !ok {
		return nil, fmt.Errorf("dependency 'event-subscriber' is not an EventSubscriber")
	}

	return subscriber, nil
}

// MessagingServiceLifecycleManager provides lifecycle management for messaging
// services within the SWIT framework, ensuring proper startup and shutdown
// coordination with the server lifecycle.
type MessagingServiceLifecycleManager struct {
	container  server.BusinessDependencyContainer
	configurer *MessagingServiceConfigurer
}

// NewMessagingServiceLifecycleManager creates a new lifecycle manager for messaging services.
func NewMessagingServiceLifecycleManager(container server.BusinessDependencyContainer) *MessagingServiceLifecycleManager {
	return &MessagingServiceLifecycleManager{
		container:  container,
		configurer: NewMessagingServiceConfigurer(),
	}
}

// Initialize initializes all messaging services and ensures proper startup order.
// This method should be called during server startup after dependency initialization.
func (m *MessagingServiceLifecycleManager) Initialize(ctx context.Context) error {
	// Get broker and ensure it's connected
	broker, err := m.configurer.GetBroker(m.container)
	if err != nil {
		return fmt.Errorf("failed to get message broker for initialization: %w", err)
	}

	// Connect to the broker
	if err := broker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect message broker: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down all messaging services.
// This method should be called during server shutdown to ensure proper cleanup.
func (m *MessagingServiceLifecycleManager) Shutdown(ctx context.Context) error {
	// Get broker and disconnect
	broker, err := m.configurer.GetBroker(m.container)
	if err != nil {
		// If we can't get the broker, it might already be cleaned up
		return nil
	}

	// Disconnect from the broker
	if err := broker.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect message broker: %w", err)
	}

	return nil
}
