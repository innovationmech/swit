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
	"fmt"
	"sync"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"go.uber.org/zap"
)

// ServerFactory defines the interface for creating servers
type ServerFactory interface {
	// CreateServer creates a new server instance with the given configuration
	CreateServer(config *ServerConfig) (BaseServer, error)

	// GetSupportedTypes returns the list of server types this factory can create
	GetSupportedTypes() []string

	// SupportsType checks if the factory supports creating servers of the given type
	SupportsType(serverType string) bool
}

// ServerFactoryFunc is a function type that implements ServerFactory
type ServerFactoryFunc func(config *ServerConfig) (BaseServer, error)

// CreateServer implements ServerFactory
func (f ServerFactoryFunc) CreateServer(config *ServerConfig) (BaseServer, error) {
	return f(config)
}

// GetSupportedTypes implements ServerFactory (returns generic type)
func (f ServerFactoryFunc) GetSupportedTypes() []string {
	return []string{"generic"}
}

// SupportsType implements ServerFactory (supports all types)
func (f ServerFactoryFunc) SupportsType(serverType string) bool {
	return true
}

// DefaultServerFactory provides the default implementation for creating servers
type DefaultServerFactory struct {
	supportedTypes []string
}

// NewDefaultServerFactory creates a new default server factory
func NewDefaultServerFactory() *DefaultServerFactory {
	return &DefaultServerFactory{
		supportedTypes: []string{"http", "grpc", "mixed", "generic"},
	}
}

// CreateServer creates a new server instance
func (f *DefaultServerFactory) CreateServer(config *ServerConfig) (BaseServer, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	// Create dependency container
	dependencyContainer := NewSimpleDependencyContainer()

	// Create transport manager
	transportManager := transport.NewManager()

	// Create service registry
	serviceRegistry := newServiceRegistry(transportManager)

	// Create server instance
	server := &Server{
		config:           config,
		transportManager: transportManager,
		dependencies:     dependencyContainer,
		registry:         serviceRegistry,
		shutdownCh:       make(chan struct{}),
	}

	if logger.Logger != nil {
		logger.Logger.Info("Server created",
			zap.String("name", config.ServiceName),
			zap.String("version", config.ServiceVersion),
		)
	}

	return server, nil
}

// GetSupportedTypes returns the supported server types
func (f *DefaultServerFactory) GetSupportedTypes() []string {
	return f.supportedTypes
}

// SupportsType checks if the factory supports the given server type
func (f *DefaultServerFactory) SupportsType(serverType string) bool {
	for _, supportedType := range f.supportedTypes {
		if supportedType == serverType {
			return true
		}
	}
	return false
}

// FactoryRegistry manages multiple server factories
type FactoryRegistry struct {
	factories map[string]ServerFactory
	mu        sync.RWMutex
}

// NewFactoryRegistry creates a new factory registry
func NewFactoryRegistry() *FactoryRegistry {
	return &FactoryRegistry{
		factories: make(map[string]ServerFactory),
	}
}

// Register registers a factory for a specific server type
func (r *FactoryRegistry) Register(serverType string, factory ServerFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if serverType == "" {
		return fmt.Errorf("server type cannot be empty")
	}

	if factory == nil {
		return fmt.Errorf("factory cannot be nil")
	}

	if _, exists := r.factories[serverType]; exists {
		return fmt.Errorf("factory for server type '%s' is already registered", serverType)
	}

	r.factories[serverType] = factory

	if logger.Logger != nil {
		logger.Logger.Debug("Server factory registered",
			zap.String("type", serverType),
			zap.Strings("supported_types", factory.GetSupportedTypes()),
		)
	}

	return nil
}

// Unregister removes a factory for a specific server type
func (r *FactoryRegistry) Unregister(serverType string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.factories, serverType)

	if logger.Logger != nil {
		logger.Logger.Debug("Server factory unregistered", zap.String("type", serverType))
	}
}

// GetFactory returns the factory for a specific server type
func (r *FactoryRegistry) GetFactory(serverType string) (ServerFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[serverType]
	if !exists {
		return nil, fmt.Errorf("no factory registered for server type '%s'", serverType)
	}

	return factory, nil
}

// CreateServer creates a server using the appropriate factory
func (r *FactoryRegistry) CreateServer(config *ServerConfig) (BaseServer, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	// For now, use a default server type since ServerConfig doesn't have Type field
	serverType := "generic"
	factory, err := r.GetFactory(serverType)
	if err != nil {
		// Try to find a factory that supports this type
		factory = r.findSupportingFactory(serverType)
		if factory == nil {
			return nil, fmt.Errorf("no factory found for server type '%s'", serverType)
		}
	}

	return factory.CreateServer(config)
}

// findSupportingFactory finds a factory that supports the given server type
func (r *FactoryRegistry) findSupportingFactory(serverType string) ServerFactory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, factory := range r.factories {
		if factory.SupportsType(serverType) {
			return factory
		}
	}

	return nil
}

// ListFactories returns a list of all registered factory types
func (r *FactoryRegistry) ListFactories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for serverType := range r.factories {
		types = append(types, serverType)
	}

	return types
}

// HasFactory checks if a factory is registered for the given server type
func (r *FactoryRegistry) HasFactory(serverType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[serverType]
	return exists
}

// Clear removes all registered factories
func (r *FactoryRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories = make(map[string]ServerFactory)

	if logger.Logger != nil {
		logger.Logger.Debug("All server factories cleared")
	}
}

// Count returns the number of registered factories
func (r *FactoryRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.factories)
}

// Global factory registry instance
var (
	globalFactoryRegistry *FactoryRegistry
	globalFactoryOnce     sync.Once
)

// GetGlobalFactoryRegistry returns the global factory registry instance
func GetGlobalFactoryRegistry() *FactoryRegistry {
	globalFactoryOnce.Do(func() {
		globalFactoryRegistry = NewFactoryRegistry()

		// Register default factory
		defaultFactory := NewDefaultServerFactory()
		for _, serverType := range defaultFactory.GetSupportedTypes() {
			if err := globalFactoryRegistry.Register(serverType, defaultFactory); err != nil {
				if logger.Logger != nil {
					logger.Logger.Warn("Failed to register default factory",
						zap.String("type", serverType),
						zap.Error(err),
					)
				}
			}
		}
	})

	return globalFactoryRegistry
}

// RegisterServerFactory registers a factory in the global registry
func RegisterServerFactory(serverType string, factory ServerFactory) error {
	return GetGlobalFactoryRegistry().Register(serverType, factory)
}

// CreateServerWithFactory creates a server using the global factory registry
func CreateServerWithFactory(config *ServerConfig) (BaseServer, error) {
	return GetGlobalFactoryRegistry().CreateServer(config)
}

// ServerBuilder provides a fluent interface for building servers
type ServerBuilder struct {
	config  *ServerConfig
	factory ServerFactory
}

// NewServerBuilder creates a new server builder
func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{
		config: &ServerConfig{},
	}
}

// WithConfig sets the server configuration
func (b *ServerBuilder) WithConfig(config *ServerConfig) *ServerBuilder {
	b.config = config
	return b
}

// WithName sets the server name
func (b *ServerBuilder) WithName(name string) *ServerBuilder {
	b.config.ServiceName = name
	return b
}

// WithVersion sets the server version
func (b *ServerBuilder) WithVersion(version string) *ServerBuilder {
	b.config.ServiceVersion = version
	return b
}

// WithHTTPPort sets the HTTP port
func (b *ServerBuilder) WithHTTPPort(port string) *ServerBuilder {
	b.config.HTTPPort = port
	return b
}

// WithGRPCPort sets the gRPC port
func (b *ServerBuilder) WithGRPCPort(port string) *ServerBuilder {
	b.config.GRPCPort = port
	return b
}

// WithFactory sets a custom factory
func (b *ServerBuilder) WithFactory(factory ServerFactory) *ServerBuilder {
	b.factory = factory
	return b
}

// Build creates the server instance
func (b *ServerBuilder) Build() (BaseServer, error) {
	if b.config == nil {
		return nil, fmt.Errorf("server config is required")
	}

	// Set defaults if not already set
	b.config.SetDefaults()

	// Use custom factory if provided, otherwise use global registry
	if b.factory != nil {
		return b.factory.CreateServer(b.config)
	}

	return CreateServerWithFactory(b.config)
}
