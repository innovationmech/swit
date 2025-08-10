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

package server

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// DependencyFactory defines a function type for creating dependencies
type DependencyFactory func(container BusinessDependencyContainer) (interface{}, error)

// DependencyLifecycle defines the lifecycle management interface for dependencies
type DependencyLifecycle interface {
	// Initialize initializes the dependency with the provided context
	Initialize(ctx context.Context) error
	// Shutdown gracefully shuts down the dependency
	Shutdown(ctx context.Context) error
}

// DependencyMetadata holds metadata about a registered dependency
type DependencyMetadata struct {
	Name        string
	Type        reflect.Type
	Singleton   bool
	Factory     DependencyFactory
	Instance    interface{}
	Initialized bool
}

// SimpleBusinessDependencyContainer provides a basic implementation of BusinessDependencyContainer
// It supports singleton and transient dependencies with lifecycle management
type SimpleBusinessDependencyContainer struct {
	mu           sync.RWMutex
	dependencies map[string]*DependencyMetadata
	initialized  bool
	closed       bool
}

// NewSimpleBusinessDependencyContainer creates a new SimpleBusinessDependencyContainer
func NewSimpleBusinessDependencyContainer() *SimpleBusinessDependencyContainer {
	return &SimpleBusinessDependencyContainer{
		dependencies: make(map[string]*DependencyMetadata),
		initialized:  false,
		closed:       false,
	}
}

// RegisterSingleton registers a singleton dependency with the container
// Singleton dependencies are created once and reused for all requests
func (c *SimpleBusinessDependencyContainer) RegisterSingleton(name string, factory DependencyFactory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("cannot register dependency '%s': container is closed", name)
	}

	if _, exists := c.dependencies[name]; exists {
		return fmt.Errorf("dependency '%s' is already registered", name)
	}

	c.dependencies[name] = &DependencyMetadata{
		Name:        name,
		Singleton:   true,
		Factory:     factory,
		Instance:    nil,
		Initialized: false,
	}

	logger.Logger.Debug("Registered singleton dependency", zap.String("name", name))
	return nil
}

// RegisterTransient registers a transient dependency with the container
// Transient dependencies are created fresh for each request
func (c *SimpleBusinessDependencyContainer) RegisterTransient(name string, factory DependencyFactory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("cannot register dependency '%s': container is closed", name)
	}

	if _, exists := c.dependencies[name]; exists {
		return fmt.Errorf("dependency '%s' is already registered", name)
	}

	c.dependencies[name] = &DependencyMetadata{
		Name:        name,
		Singleton:   false,
		Factory:     factory,
		Instance:    nil,
		Initialized: false,
	}

	logger.Logger.Debug("Registered transient dependency", zap.String("name", name))
	return nil
}

// RegisterInstance registers a pre-created instance as a singleton dependency
func (c *SimpleBusinessDependencyContainer) RegisterInstance(name string, instance interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("cannot register dependency '%s': container is closed", name)
	}

	if _, exists := c.dependencies[name]; exists {
		return fmt.Errorf("dependency '%s' is already registered", name)
	}

	c.dependencies[name] = &DependencyMetadata{
		Name:        name,
		Type:        reflect.TypeOf(instance),
		Singleton:   true,
		Factory:     nil,
		Instance:    instance,
		Initialized: true,
	}

	logger.Logger.Debug("Registered instance dependency", zap.String("name", name))
	return nil
}

// GetService retrieves a service by name from the container
func (c *SimpleBusinessDependencyContainer) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	metadata, exists := c.dependencies[name]
	if !exists {
		c.mu.RUnlock()
		return nil, fmt.Errorf("dependency '%s' not found", name)
	}

	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("cannot get dependency '%s': container is closed", name)
	}

	// For singleton dependencies, return existing instance if available
	if metadata.Singleton && metadata.Instance != nil {
		instance := metadata.Instance
		c.mu.RUnlock()
		return instance, nil
	}

	// Create new instance using factory
	if metadata.Factory == nil {
		c.mu.RUnlock()
		return nil, fmt.Errorf("no factory available for dependency '%s'", name)
	}

	factory := metadata.Factory
	isSingleton := metadata.Singleton
	c.mu.RUnlock()

	instance, err := factory(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create dependency '%s': %w", name, err)
	}

	// Store singleton instances
	if isSingleton {
		c.mu.Lock()
		// Double-check pattern: another goroutine might have created the instance
		if metadata.Instance == nil {
			metadata.Instance = instance
			metadata.Type = reflect.TypeOf(instance)
		}
		c.mu.Unlock()
	}

	return instance, nil
}

// Initialize initializes all singleton dependencies
func (c *SimpleBusinessDependencyContainer) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	if c.closed {
		return fmt.Errorf("cannot initialize: container is closed")
	}

	logger.Logger.Info("Initializing dependency container")

	// Initialize all singleton dependencies
	for name, metadata := range c.dependencies {
		if metadata.Singleton && !metadata.Initialized {
			if err := c.initializeDependency(ctx, name, metadata); err != nil {
				return fmt.Errorf("failed to initialize dependency '%s': %w", name, err)
			}
		}
	}

	c.initialized = true
	logger.Logger.Info("Dependency container initialized successfully")
	return nil
}

// initializeDependency initializes a single dependency
func (c *SimpleBusinessDependencyContainer) initializeDependency(ctx context.Context, name string, metadata *DependencyMetadata) error {
	// Create instance if not already created
	if metadata.Instance == nil && metadata.Factory != nil {
		instance, err := metadata.Factory(c)
		if err != nil {
			return fmt.Errorf("factory failed for dependency '%s': %w", name, err)
		}
		metadata.Instance = instance
		metadata.Type = reflect.TypeOf(instance)
	}

	// Initialize if it implements DependencyLifecycle
	if lifecycle, ok := metadata.Instance.(DependencyLifecycle); ok {
		if err := lifecycle.Initialize(ctx); err != nil {
			return fmt.Errorf("lifecycle initialization failed for dependency '%s': %w", name, err)
		}
	}

	metadata.Initialized = true
	logger.Logger.Debug("Dependency initialized", zap.String("name", name))
	return nil
}

// Close closes all managed dependencies and cleans up resources
func (c *SimpleBusinessDependencyContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	logger.Logger.Info("Closing dependency container")

	ctx := context.Background()
	var errors []error

	// Shutdown all dependencies in reverse order of initialization
	for name, metadata := range c.dependencies {
		if metadata.Instance != nil {
			if err := c.shutdownDependency(ctx, name, metadata); err != nil {
				errors = append(errors, err)
			}
		}
	}

	c.closed = true
	c.initialized = false

	if len(errors) > 0 {
		return fmt.Errorf("errors during container shutdown: %v", errors)
	}

	logger.Logger.Info("Dependency container closed successfully")
	return nil
}

// shutdownDependency shuts down a single dependency
func (c *SimpleBusinessDependencyContainer) shutdownDependency(ctx context.Context, name string, metadata *DependencyMetadata) error {
	// Shutdown if it implements DependencyLifecycle
	if lifecycle, ok := metadata.Instance.(DependencyLifecycle); ok {
		if err := lifecycle.Shutdown(ctx); err != nil {
			logger.Logger.Error("Failed to shutdown dependency",
				zap.String("name", name),
				zap.Error(err))
			return fmt.Errorf("lifecycle shutdown failed for dependency '%s': %w", name, err)
		}
	}

	logger.Logger.Debug("Dependency shutdown", zap.String("name", name))
	return nil
}

// GetDependencyNames returns the names of all registered dependencies
func (c *SimpleBusinessDependencyContainer) GetDependencyNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.dependencies))
	for name := range c.dependencies {
		names = append(names, name)
	}
	return names
}

// GetDependencyMetadata returns metadata for a specific dependency
func (c *SimpleBusinessDependencyContainer) GetDependencyMetadata(name string) (*DependencyMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metadata, exists := c.dependencies[name]
	if !exists {
		return nil, fmt.Errorf("dependency '%s' not found", name)
	}

	// Return a copy to prevent external modification
	return &DependencyMetadata{
		Name:        metadata.Name,
		Type:        metadata.Type,
		Singleton:   metadata.Singleton,
		Factory:     metadata.Factory,
		Instance:    metadata.Instance,
		Initialized: metadata.Initialized,
	}, nil
}

// IsInitialized returns true if the container has been initialized
func (c *SimpleBusinessDependencyContainer) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// IsClosed returns true if the container has been closed
func (c *SimpleBusinessDependencyContainer) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// BusinessDependencyContainerBuilder provides a fluent interface for building dependency containers
type BusinessDependencyContainerBuilder struct {
	container *SimpleBusinessDependencyContainer
}

// NewBusinessDependencyContainerBuilder creates a new BusinessDependencyContainerBuilder
func NewBusinessDependencyContainerBuilder() *BusinessDependencyContainerBuilder {
	return &BusinessDependencyContainerBuilder{
		container: NewSimpleBusinessDependencyContainer(),
	}
}

// AddSingleton adds a singleton dependency to the container being built
func (b *BusinessDependencyContainerBuilder) AddSingleton(name string, factory DependencyFactory) *BusinessDependencyContainerBuilder {
	if err := b.container.RegisterSingleton(name, factory); err != nil {
		logger.Logger.Error("Failed to add singleton dependency",
			zap.String("name", name),
			zap.Error(err))
	}
	return b
}

// AddTransient adds a transient dependency to the container being built
func (b *BusinessDependencyContainerBuilder) AddTransient(name string, factory DependencyFactory) *BusinessDependencyContainerBuilder {
	if err := b.container.RegisterTransient(name, factory); err != nil {
		logger.Logger.Error("Failed to add transient dependency",
			zap.String("name", name),
			zap.Error(err))
	}
	return b
}

// AddInstance adds a pre-created instance to the container being built
func (b *BusinessDependencyContainerBuilder) AddInstance(name string, instance interface{}) *BusinessDependencyContainerBuilder {
	if err := b.container.RegisterInstance(name, instance); err != nil {
		logger.Logger.Error("Failed to add instance dependency",
			zap.String("name", name),
			zap.Error(err))
	}
	return b
}

// Build returns the built dependency container
func (b *BusinessDependencyContainerBuilder) Build() BusinessDependencyContainer {
	return b.container
}
