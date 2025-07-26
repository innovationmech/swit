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

package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switserve/types"
)

// HandlerMetadata contains metadata information about a service
type HandlerMetadata struct {
	// Name is the service name
	Name string `json:"name"`
	// Version is the service version
	Version string `json:"version"`
	// Description is a brief description of the service
	Description string `json:"description"`
	// HealthEndpoint is the health check endpoint path
	HealthEndpoint string `json:"health_endpoint"`
	// Tags are optional tags for service categorization
	Tags []string `json:"tags,omitempty"`
	// Dependencies are the services this service depends on
	Dependencies []string `json:"dependencies,omitempty"`
}

// HandlerRegister defines the unified interface for service registration and management
// This interface extends the existing ServiceRegistrar with additional metadata and lifecycle methods
type HandlerRegister interface {
	// RegisterHTTP registers HTTP routes with the given router
	RegisterHTTP(router *gin.Engine) error

	// RegisterGRPC registers gRPC services with the given server
	RegisterGRPC(server *grpc.Server) error

	// GetMetadata returns service metadata information
	GetMetadata() *HandlerMetadata

	// GetHealthEndpoint returns the health check endpoint path
	GetHealthEndpoint() string

	// IsHealthy performs a health check and returns the current status
	IsHealthy(ctx context.Context) (*types.HealthStatus, error)

	// Initialize performs any necessary initialization before service registration
	Initialize(ctx context.Context) error

	// Shutdown performs graceful shutdown of the service
	Shutdown(ctx context.Context) error
}

// EnhancedHandlerRegistry manages service handlers with thread-safe operations
// This is an enhanced version of the existing ServiceRegistry that works with HandlerRegister interface
type EnhancedHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]HandlerRegister
	order    []string // Maintains registration order
}

// NewEnhancedServiceRegistry creates a new enhanced service registry
func NewEnhancedServiceRegistry() *EnhancedHandlerRegistry {
	return &EnhancedHandlerRegistry{
		handlers: make(map[string]HandlerRegister),
		order:    make([]string, 0),
	}
}

// Register adds a service handler to the registry
func (sr *EnhancedHandlerRegistry) Register(handler HandlerRegister) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	metadata := handler.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("service handler metadata cannot be nil")
	}

	if metadata.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check for duplicate service names
	if _, exists := sr.handlers[metadata.Name]; exists {
		return fmt.Errorf("service '%s' is already registered", metadata.Name)
	}

	sr.handlers[metadata.Name] = handler
	sr.order = append(sr.order, metadata.Name)

	return nil
}

// Unregister removes a service handler from the registry
func (sr *EnhancedHandlerRegistry) Unregister(serviceName string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.handlers[serviceName]; !exists {
		return fmt.Errorf("service '%s' is not registered", serviceName)
	}

	delete(sr.handlers, serviceName)

	// Remove from order slice
	for i, name := range sr.order {
		if name == serviceName {
			sr.order = append(sr.order[:i], sr.order[i+1:]...)
			break
		}
	}

	return nil
}

// GetHandler returns a specific service handler by name
func (sr *EnhancedHandlerRegistry) GetHandler(serviceName string) (HandlerRegister, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	handler, exists := sr.handlers[serviceName]
	if !exists {
		return nil, fmt.Errorf("service '%s' is not registered", serviceName)
	}

	return handler, nil
}

// GetAllHandlers returns all registered service handlers in registration order
func (sr *EnhancedHandlerRegistry) GetAllHandlers() []HandlerRegister {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	handlers := make([]HandlerRegister, 0, len(sr.order))
	for _, name := range sr.order {
		if handler, exists := sr.handlers[name]; exists {
			handlers = append(handlers, handler)
		}
	}

	return handlers
}

// GetServiceNames returns all registered service names in registration order
func (sr *EnhancedHandlerRegistry) GetServiceNames() []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	names := make([]string, len(sr.order))
	copy(names, sr.order)
	return names
}

// GetServiceMetadata returns metadata for all registered services
func (sr *EnhancedHandlerRegistry) GetServiceMetadata() []*HandlerMetadata {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	metadata := make([]*HandlerMetadata, 0, len(sr.order))
	for _, name := range sr.order {
		if handler, exists := sr.handlers[name]; exists {
			metadata = append(metadata, handler.GetMetadata())
		}
	}

	return metadata
}

// InitializeAll initializes all registered services in registration order
func (sr *EnhancedHandlerRegistry) InitializeAll(ctx context.Context) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	for _, name := range sr.order {
		if handler, exists := sr.handlers[name]; exists {
			if err := handler.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize service '%s': %w", name, err)
			}
		}
	}

	return nil
}

// RegisterAllHTTP registers HTTP routes for all services
func (sr *EnhancedHandlerRegistry) RegisterAllHTTP(router *gin.Engine) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	for _, name := range sr.order {
		if handler, exists := sr.handlers[name]; exists {
			if err := handler.RegisterHTTP(router); err != nil {
				return fmt.Errorf("failed to register HTTP routes for service '%s': %w", name, err)
			}
		}
	}

	return nil
}

// RegisterAllGRPC registers gRPC services for all handlers
func (sr *EnhancedHandlerRegistry) RegisterAllGRPC(server *grpc.Server) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	for _, name := range sr.order {
		if handler, exists := sr.handlers[name]; exists {
			if err := handler.RegisterGRPC(server); err != nil {
				return fmt.Errorf("failed to register gRPC services for service '%s': %w", name, err)
			}
		}
	}

	return nil
}

// CheckAllHealth performs health checks on all registered services
func (sr *EnhancedHandlerRegistry) CheckAllHealth(ctx context.Context) map[string]*types.HealthStatus {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	results := make(map[string]*types.HealthStatus)

	for _, name := range sr.order {
		if handler, exists := sr.handlers[name]; exists {
			// Create a timeout context for each health check
			healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			status, err := handler.IsHealthy(healthCtx)
			cancel()

			if err != nil {
				// Create an unhealthy status if health check failed
				status = &types.HealthStatus{
					Status:       "unhealthy",
					Timestamp:    time.Now(),
					Version:      "unknown",
					Uptime:       0,
					Dependencies: make(map[string]types.DependencyStatus),
				}
			}

			results[name] = status
		}
	}

	return results
}

// ShutdownAll gracefully shuts down all registered services in reverse order
func (sr *EnhancedHandlerRegistry) ShutdownAll(ctx context.Context) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	// Shutdown in reverse order
	for i := len(sr.order) - 1; i >= 0; i-- {
		name := sr.order[i]
		if handler, exists := sr.handlers[name]; exists {
			if err := handler.Shutdown(ctx); err != nil {
				return fmt.Errorf("failed to shutdown service '%s': %w", name, err)
			}
		}
	}

	return nil
}

// Count returns the number of registered services
func (sr *EnhancedHandlerRegistry) Count() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return len(sr.handlers)
}

// IsEmpty returns true if no services are registered
func (sr *EnhancedHandlerRegistry) IsEmpty() bool {
	return sr.Count() == 0
}
