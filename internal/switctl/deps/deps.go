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

// Package deps provides dependency injection container implementation for switctl.
package deps

import (
	"fmt"
	"sync"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// Container represents the dependency injection container.
type Container struct {
	services   map[string]*serviceRegistration
	singletons map[string]interface{}
	mu         sync.RWMutex
	closed     bool
}

// serviceRegistration holds service factory and metadata.
type serviceRegistration struct {
	factory   interfaces.ServiceFactory
	singleton bool
	instance  interface{}
	created   bool
}

// NewContainer creates a new dependency injection container.
func NewContainer() *Container {
	return &Container{
		services:   make(map[string]*serviceRegistration),
		singletons: make(map[string]interface{}),
	}
}

// Register registers a service with the container using a factory function.
func (c *Container) Register(name string, factory interfaces.ServiceFactory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return interfaces.NewInternalError("container is closed", nil)
	}

	if _, exists := c.services[name]; exists {
		return interfaces.NewInternalError(fmt.Sprintf("service %s already registered", name), nil)
	}

	c.services[name] = &serviceRegistration{
		factory:   factory,
		singleton: false,
	}

	return nil
}

// RegisterSingleton registers a singleton service with the container.
func (c *Container) RegisterSingleton(name string, factory interfaces.ServiceFactory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return interfaces.NewInternalError("container is closed", nil)
	}

	if _, exists := c.services[name]; exists {
		return interfaces.NewInternalError(fmt.Sprintf("service %s already registered", name), nil)
	}

	c.services[name] = &serviceRegistration{
		factory:   factory,
		singleton: true,
	}

	return nil
}

// GetService retrieves a service instance by name.
func (c *Container) GetService(name string) (interface{}, error) {
	c.mu.RLock()

	if c.closed {
		c.mu.RUnlock()
		return nil, interfaces.NewInternalError("container is closed", nil)
	}

	registration, exists := c.services[name]
	if !exists {
		c.mu.RUnlock()
		return nil, interfaces.NewInternalError(fmt.Sprintf("service %s not found", name), nil)
	}

	// If it's not a singleton, create and return a new instance each time
	if !registration.singleton {
		factory := registration.factory
		c.mu.RUnlock()

		instance, err := factory()
		if err != nil {
			return nil, interfaces.NewInternalError(fmt.Sprintf("failed to create service %s", name), err)
		}
		return instance, nil
	}

	// For singletons, check if already created
	if registration.created {
		instance := registration.instance
		c.mu.RUnlock()
		return instance, nil
	}

	// Upgrade to write lock for singleton creation
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have created it)
	if registration.created {
		return registration.instance, nil
	}

	// Create singleton instance
	instance, err := registration.factory()
	if err != nil {
		return nil, interfaces.NewInternalError(fmt.Sprintf("failed to create service %s", name), err)
	}

	// Store the singleton instance
	registration.instance = instance
	registration.created = true

	return instance, nil
}

// HasService checks if a service is registered with the container.
func (c *Container) HasService(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.services[name]
	return exists
}

// ListServices returns a list of all registered service names.
func (c *Container) ListServices() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services := make([]string, 0, len(c.services))
	for name := range c.services {
		services = append(services, name)
	}
	return services
}

// Close closes the container and cleans up resources.
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	// Clean up singleton instances that implement cleanup
	for name, registration := range c.services {
		if registration.singleton && registration.created && registration.instance != nil {
			// Check if the instance implements a Close method
			if closer, ok := registration.instance.(interface{ Close() error }); ok {
				if err := closer.Close(); err != nil {
					// Log error but continue cleanup
					fmt.Printf("Error closing service %s: %v\n", name, err)
				}
			}
		}
	}

	c.services = nil
	c.singletons = nil
	c.closed = true

	return nil
}

// IsClosed returns true if the container has been closed.
func (c *Container) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// DefaultContainer is the global default container instance.
var (
	defaultContainer *Container
	defaultOnce      sync.Once
)

// GetDefaultContainer returns the default global container instance.
func GetDefaultContainer() *Container {
	defaultOnce.Do(func() {
		defaultContainer = NewContainer()
	})
	return defaultContainer
}

// RegisterDefault registers a service with the default container.
func RegisterDefault(name string, factory interfaces.ServiceFactory) error {
	return GetDefaultContainer().Register(name, factory)
}

// RegisterSingletonDefault registers a singleton service with the default container.
func RegisterSingletonDefault(name string, factory interfaces.ServiceFactory) error {
	return GetDefaultContainer().RegisterSingleton(name, factory)
}

// GetDefault retrieves a service from the default container.
func GetDefault(name string) (interface{}, error) {
	return GetDefaultContainer().GetService(name)
}

// HasDefault checks if a service exists in the default container.
func HasDefault(name string) bool {
	return GetDefaultContainer().HasService(name)
}

// Service name constants for commonly used services.
const (
	ServiceConfigManager     = "config_manager"
	ServiceLogger            = "logger"
	ServiceFileSystem        = "file_system"
	ServiceTemplateEngine    = "template_engine"
	ServiceInteractiveUI     = "interactive_ui"
	ServiceGenerator         = "generator"
	ServiceQualityChecker    = "quality_checker"
	ServicePluginManager     = "plugin_manager"
	ServiceProjectManager    = "project_manager"
	ServiceDependencyManager = "dependency_manager"
	ServiceVersionManager    = "version_manager"
)

// ServiceBuilder provides a fluent interface for service registration.
type ServiceBuilder struct {
	container *Container
	name      string
	factory   interfaces.ServiceFactory
	singleton bool
	err       error
}

// NewServiceBuilder creates a new service builder for the given container.
func NewServiceBuilder(container *Container, name string) *ServiceBuilder {
	return &ServiceBuilder{
		container: container,
		name:      name,
	}
}

// WithFactory sets the service factory function.
func (sb *ServiceBuilder) WithFactory(factory interfaces.ServiceFactory) *ServiceBuilder {
	if sb.err != nil {
		return sb
	}
	sb.factory = factory
	return sb
}

// AsSingleton marks the service as a singleton.
func (sb *ServiceBuilder) AsSingleton() *ServiceBuilder {
	if sb.err != nil {
		return sb
	}
	sb.singleton = true
	return sb
}

// Build registers the service with the container.
func (sb *ServiceBuilder) Build() error {
	if sb.err != nil {
		return sb.err
	}

	if sb.factory == nil {
		return interfaces.NewInternalError("service factory is required", nil)
	}

	if sb.singleton {
		return sb.container.RegisterSingleton(sb.name, sb.factory)
	}
	return sb.container.Register(sb.name, sb.factory)
}

// ContainerConfig represents configuration for the dependency container.
type ContainerConfig struct {
	Services map[string]ServiceConfig `yaml:"services" json:"services"`
}

// ServiceConfig represents configuration for a single service.
type ServiceConfig struct {
	Type      string                 `yaml:"type" json:"type"`
	Singleton bool                   `yaml:"singleton" json:"singleton"`
	Config    map[string]interface{} `yaml:"config" json:"config"`
	Enabled   bool                   `yaml:"enabled" json:"enabled"`
}

// InitializeServices initializes services based on configuration.
func (c *Container) InitializeServices(config ContainerConfig) error {
	for name, serviceConfig := range config.Services {
		if !serviceConfig.Enabled {
			continue
		}

		factory, err := c.createServiceFactory(serviceConfig)
		if err != nil {
			return interfaces.NewInternalError(fmt.Sprintf("failed to create factory for service %s", name), err)
		}

		if serviceConfig.Singleton {
			if err := c.RegisterSingleton(name, factory); err != nil {
				return err
			}
		} else {
			if err := c.Register(name, factory); err != nil {
				return err
			}
		}
	}
	return nil
}

// createServiceFactory creates a service factory based on configuration.
func (c *Container) createServiceFactory(config ServiceConfig) (interfaces.ServiceFactory, error) {
	switch config.Type {
	case "config_manager":
		return func() (interface{}, error) {
			return NewConfigManager(config.Config)
		}, nil
	case "logger":
		return func() (interface{}, error) {
			return NewLogger(config.Config)
		}, nil
	// Add more service factories as needed
	default:
		return nil, interfaces.NewInternalError(fmt.Sprintf("unknown service type: %s", config.Type), nil)
	}
}

// Placeholder factory functions - these will be implemented in separate files
func NewConfigManager(config map[string]interface{}) (interfaces.ConfigManager, error) {
	// TODO: Implement config manager
	return nil, interfaces.NewNotImplementedError("config_manager")
}

func NewLogger(config map[string]interface{}) (interfaces.Logger, error) {
	// TODO: Implement logger
	return nil, interfaces.NewNotImplementedError("logger")
}

// ServiceRegistry provides a centralized way to register commonly used services.
type ServiceRegistry struct {
	container *Container
}

// NewServiceRegistry creates a new service registry.
func NewServiceRegistry(container *Container) *ServiceRegistry {
	return &ServiceRegistry{container: container}
}

// RegisterCoreServices registers all core services required by switctl.
func (sr *ServiceRegistry) RegisterCoreServices() error {
	// Register config manager as singleton
	if err := sr.container.RegisterSingleton(ServiceConfigManager, func() (interface{}, error) {
		return NewConfigManager(nil)
	}); err != nil {
		return err
	}

	// Register logger as singleton
	if err := sr.container.RegisterSingleton(ServiceLogger, func() (interface{}, error) {
		return NewLogger(nil)
	}); err != nil {
		return err
	}

	// More services will be registered as they are implemented
	// TODO: Register other core services

	return nil
}

// GetConfigManager retrieves the config manager from the container.
func (sr *ServiceRegistry) GetConfigManager() (interfaces.ConfigManager, error) {
	service, err := sr.container.GetService(ServiceConfigManager)
	if err != nil {
		return nil, err
	}
	return service.(interfaces.ConfigManager), nil
}

// GetLogger retrieves the logger from the container.
func (sr *ServiceRegistry) GetLogger() (interfaces.Logger, error) {
	service, err := sr.container.GetService(ServiceLogger)
	if err != nil {
		return nil, err
	}
	return service.(interfaces.Logger), nil
}
