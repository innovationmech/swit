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

	"github.com/innovationmech/swit/pkg/types"
)

// ServiceRegistryManager manages multiple service registries for different transports
type ServiceRegistryManager struct {
	mu         sync.RWMutex
	registries map[string]*EnhancedHandlerRegistry // Transport name -> registry
}

// validateTransportName validates transport name
func validateTransportName(name string) error {
	if name == "" {
		return fmt.Errorf("transport name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("transport name too long (max 50 characters)")
	}
	// Only allow alphanumeric characters and hyphens
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' {
			return fmt.Errorf("transport name can only contain letters, numbers, and hyphens")
		}
	}
	return nil
}

// NewServiceRegistryManager creates a new service registry manager
func NewServiceRegistryManager() *ServiceRegistryManager {
	return &ServiceRegistryManager{
		registries: make(map[string]*EnhancedHandlerRegistry),
	}
}

// GetOrCreateRegistry gets or creates a registry for the specified transport
func (srm *ServiceRegistryManager) GetOrCreateRegistry(transportName string) *EnhancedHandlerRegistry {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	if registry, exists := srm.registries[transportName]; exists {
		return registry
	}

	registry := NewEnhancedServiceRegistry()
	srm.registries[transportName] = registry
	return registry
}

// GetRegistry returns the registry for the specified transport
func (srm *ServiceRegistryManager) GetRegistry(transportName string) *EnhancedHandlerRegistry {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	return srm.registries[transportName]
}

// RegisterHandler registers a service handler with a specific transport
func (srm *ServiceRegistryManager) RegisterHandler(transportName string, handler HandlerRegister) error {
	if err := validateTransportName(transportName); err != nil {
		return fmt.Errorf("invalid transport name: %w", err)
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	registry := srm.GetOrCreateRegistry(transportName)
	return registry.Register(handler)
}

// RegisterHTTPHandler registers a service handler with the HTTP transport
func (srm *ServiceRegistryManager) RegisterHTTPHandler(handler HandlerRegister) error {
	return srm.RegisterHandler("http", handler)
}

// RegisterGRPCHandler registers a service handler with the gRPC transport
func (srm *ServiceRegistryManager) RegisterGRPCHandler(handler HandlerRegister) error {
	return srm.RegisterHandler("grpc", handler)
}

// RegisterAllHTTP registers HTTP routes for services in the HTTP transport registry
func (srm *ServiceRegistryManager) RegisterAllHTTP(router *gin.Engine) error {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	// Only register HTTP routes from the HTTP transport registry
	if httpRegistry, exists := srm.registries["http"]; exists {
		if err := httpRegistry.RegisterAllHTTP(router); err != nil {
			return fmt.Errorf("failed to register HTTP routes for transport 'http': %w", err)
		}
	}

	return nil
}

// RegisterAllGRPC registers gRPC services for services in the gRPC transport registry
func (srm *ServiceRegistryManager) RegisterAllGRPC(server *grpc.Server) error {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	// Only register gRPC services from the gRPC transport registry
	if grpcRegistry, exists := srm.registries["grpc"]; exists {
		if err := grpcRegistry.RegisterAllGRPC(server); err != nil {
			return fmt.Errorf("failed to register gRPC services for transport 'grpc': %w", err)
		}
	}

	return nil
}

// InitializeAll initializes all services across all transports
func (srm *ServiceRegistryManager) InitializeAll(ctx context.Context) error {
	srm.mu.RLock()
	registriesCopy := make(map[string]*EnhancedHandlerRegistry)
	for name, registry := range srm.registries {
		registriesCopy[name] = registry
	}
	srm.mu.RUnlock()

	// Use default timeout if no deadline is set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	for transportName, registry := range registriesCopy {
		if err := registry.InitializeAll(ctx); err != nil {
			return fmt.Errorf("failed to initialize services for transport '%s': %w", transportName, err)
		}
	}

	return nil
}

// CheckAllHealth performs health checks on all services across all transports
func (srm *ServiceRegistryManager) CheckAllHealth(ctx context.Context) map[string]map[string]*types.HealthStatus {
	srm.mu.RLock()
	registriesCopy := make(map[string]*EnhancedHandlerRegistry)
	for name, registry := range srm.registries {
		registriesCopy[name] = registry
	}
	srm.mu.RUnlock()

	// Use default timeout if no deadline is set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	results := make(map[string]map[string]*types.HealthStatus)

	for transportName, registry := range registriesCopy {
		transportResults := registry.CheckAllHealth(ctx)
		if len(transportResults) > 0 {
			results[transportName] = transportResults
		}
	}

	return results
}

// ShutdownAll gracefully shuts down all services across all transports
func (srm *ServiceRegistryManager) ShutdownAll(ctx context.Context) error {
	srm.mu.RLock()

	// Make a copy of registries to avoid race condition after unlock
	registriesCopy := make(map[string]*EnhancedHandlerRegistry)
	for name, registry := range srm.registries {
		registriesCopy[name] = registry
	}
	srm.mu.RUnlock()

	var shutdownErrors []error

	// Use default timeout if no deadline is set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Shutdown in reverse order for proper cleanup
	transportNames := make([]string, 0, len(registriesCopy))
	for name := range registriesCopy {
		transportNames = append(transportNames, name)
	}

	// Reverse the slice
	for i := len(transportNames) - 1; i >= 0; i-- {
		transportName := transportNames[i]
		registry := registriesCopy[transportName]
		if err := registry.ShutdownAll(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("failed to shutdown services for transport '%s': %w", transportName, err))
		}
	}

	if len(shutdownErrors) > 0 {
		// Return all errors aggregated
		var transportErrors []TransportError
		for _, err := range shutdownErrors {
			transportErrors = append(transportErrors, TransportError{
				TransportName: "registry",
				Err:           err,
			})
		}
		return &MultiError{Errors: transportErrors}
	}

	return nil
}

// GetAllServiceMetadata returns metadata for all services across all transports
func (srm *ServiceRegistryManager) GetAllServiceMetadata() map[string][]*HandlerMetadata {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	results := make(map[string][]*HandlerMetadata)

	for transportName, registry := range srm.registries {
		metadata := registry.GetServiceMetadata()
		if len(metadata) > 0 {
			results[transportName] = metadata
		}
	}

	return results
}

// GetTransportNames returns all registered transport names
func (srm *ServiceRegistryManager) GetTransportNames() []string {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	names := make([]string, 0, len(srm.registries))
	for name := range srm.registries {
		names = append(names, name)
	}

	return names
}

// GetTotalServiceCount returns the total number of services across all transports
func (srm *ServiceRegistryManager) GetTotalServiceCount() int {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	total := 0
	for _, registry := range srm.registries {
		total += registry.Count()
	}

	return total
}

// Clear removes all registries and services
func (srm *ServiceRegistryManager) Clear() {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	srm.registries = make(map[string]*EnhancedHandlerRegistry)
}

// IsEmpty returns true if no services are registered in any transport
func (srm *ServiceRegistryManager) IsEmpty() bool {
	return srm.GetTotalServiceCount() == 0
}
