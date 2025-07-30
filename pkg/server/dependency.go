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

// Dependency represents a registered dependency
type Dependency struct {
	Name        string
	Type        reflect.Type
	Value       interface{}
	Factory     func() (interface{}, error)
	Singleton   bool
	Initialized bool
}

// SimpleDependencyContainer provides a basic implementation of DependencyContainer
type SimpleDependencyContainer struct {
	dependencies map[string]*Dependency
	mu           sync.RWMutex
	initialized  bool
}

// NewSimpleDependencyContainer creates a new simple dependency container
func NewSimpleDependencyContainer() *SimpleDependencyContainer {
	return &SimpleDependencyContainer{
		dependencies: make(map[string]*Dependency),
	}
}

// Register registers a dependency with the container
func (c *SimpleDependencyContainer) Register(name string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if name == "" {
		return fmt.Errorf("dependency name cannot be empty")
	}

	if value == nil {
		return fmt.Errorf("dependency value cannot be nil")
	}

	if _, exists := c.dependencies[name]; exists {
		return fmt.Errorf("dependency '%s' is already registered", name)
	}

	dep := &Dependency{
		Name:        name,
		Type:        reflect.TypeOf(value),
		Value:       value,
		Singleton:   true,
		Initialized: true,
	}

	c.dependencies[name] = dep

	if logger.Logger != nil {
		logger.Logger.Debug("Dependency registered",
			zap.String("name", name),
			zap.String("type", dep.Type.String()),
		)
	}

	return nil
}

// RegisterFactory registers a factory function for creating dependencies
func (c *SimpleDependencyContainer) RegisterFactory(name string, factory func() (interface{}, error), singleton bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if name == "" {
		return fmt.Errorf("dependency name cannot be empty")
	}

	if factory == nil {
		return fmt.Errorf("factory function cannot be nil")
	}

	if _, exists := c.dependencies[name]; exists {
		return fmt.Errorf("dependency '%s' is already registered", name)
	}

	dep := &Dependency{
		Name:      name,
		Factory:   factory,
		Singleton: singleton,
	}

	c.dependencies[name] = dep

	if logger.Logger != nil {
		logger.Logger.Debug("Dependency factory registered",
			zap.String("name", name),
			zap.Bool("singleton", singleton),
		)
	}

	return nil
}

// Get retrieves a dependency by name
func (c *SimpleDependencyContainer) Get(name string) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	dep, exists := c.dependencies[name]
	if !exists {
		return nil, fmt.Errorf("dependency '%s' not found", name)
	}

	// If it's a factory-based dependency
	if dep.Factory != nil {
		// For singletons, create once and cache
		if dep.Singleton {
			if dep.Value == nil {
				value, err := dep.Factory()
				if err != nil {
					return nil, fmt.Errorf("failed to create dependency '%s': %w", name, err)
				}
				dep.Value = value
				dep.Type = reflect.TypeOf(value)
				dep.Initialized = true
			}
			return dep.Value, nil
		}

		// For non-singletons, create new instance each time
		value, err := dep.Factory()
		if err != nil {
			return nil, fmt.Errorf("failed to create dependency '%s': %w", name, err)
		}
		return value, nil
	}

	// If it's a direct value, return it
	if dep.Value != nil {
		return dep.Value, nil
	}

	return nil, fmt.Errorf("dependency '%s' has no value or factory", name)
}

// GetByType retrieves a dependency by type
func (c *SimpleDependencyContainer) GetByType(t reflect.Type) (interface{}, error) {
	c.mu.RLock()
	var depName string
	var found bool

	for _, dep := range c.dependencies {
		if dep.Type != nil && dep.Type == t {
			depName = dep.Name
			found = true
			break
		}
	}
	c.mu.RUnlock()

	if found {
		return c.Get(depName)
	}

	return nil, fmt.Errorf("no dependency found for type %s", t.String())
}

// Has checks if a dependency exists
func (c *SimpleDependencyContainer) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.dependencies[name]
	return exists
}

// Initialize initializes all dependencies
func (c *SimpleDependencyContainer) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	if logger.Logger != nil {
		logger.Logger.Info("Initializing dependencies", zap.Int("count", len(c.dependencies)))
	}

	// Initialize factory-based singletons
	for name, dep := range c.dependencies {
		if dep.Factory != nil && dep.Singleton && !dep.Initialized {
			if logger.Logger != nil {
				logger.Logger.Debug("Initializing dependency", zap.String("name", name))
			}

			value, err := dep.Factory()
			if err != nil {
				return fmt.Errorf("failed to initialize dependency '%s': %w", name, err)
			}

			dep.Value = value
			dep.Type = reflect.TypeOf(value)
			dep.Initialized = true
		}
	}

	c.initialized = true
	if logger.Logger != nil {
		logger.Logger.Info("Dependencies initialized successfully")
	}

	return nil
}

// Cleanup cleans up all dependencies
func (c *SimpleDependencyContainer) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if logger.Logger != nil {
		logger.Logger.Info("Cleaning up dependencies", zap.Int("count", len(c.dependencies)))
	}

	var errors []error

	// Call cleanup on dependencies that implement cleanup interface
	for name, dep := range c.dependencies {
		if dep.Value != nil {
			if cleaner, ok := dep.Value.(interface{ Cleanup(context.Context) error }); ok {
				if logger.Logger != nil {
					logger.Logger.Debug("Cleaning up dependency", zap.String("name", name))
				}
				if err := cleaner.Cleanup(ctx); err != nil {
					errors = append(errors, fmt.Errorf("failed to cleanup dependency '%s': %w", name, err))
				}
			}
		}
	}

	c.initialized = false

	if len(errors) > 0 {
		return fmt.Errorf("dependency cleanup completed with %d errors: %v", len(errors), errors)
	}

	if logger.Logger != nil {
		logger.Logger.Info("Dependencies cleaned up successfully")
	}
	return nil
}

// List returns a list of all registered dependency names
func (c *SimpleDependencyContainer) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.dependencies))
	for name := range c.dependencies {
		names = append(names, name)
	}

	return names
}

// Count returns the number of registered dependencies
func (c *SimpleDependencyContainer) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.dependencies)
}

// Clear removes all dependencies
func (c *SimpleDependencyContainer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dependencies = make(map[string]*Dependency)
	c.initialized = false
}

// GetDependencyInfo returns information about a dependency
func (c *SimpleDependencyContainer) GetDependencyInfo(name string) (*Dependency, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dep, exists := c.dependencies[name]
	if !exists {
		return nil, fmt.Errorf("dependency '%s' not found", name)
	}

	// Return a copy to prevent external modification
	return &Dependency{
		Name:        dep.Name,
		Type:        dep.Type,
		Singleton:   dep.Singleton,
		Initialized: dep.Initialized,
		// Don't expose Value and Factory for security
	}, nil
}

// IsInitialized returns whether the container has been initialized
func (c *SimpleDependencyContainer) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.initialized
}
