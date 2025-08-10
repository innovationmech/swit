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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
)

// DefaultServiceRegistry implements the ServiceRegistry interface
// It bridges the server framework with the transport layer
type DefaultServiceRegistry struct {
	transportManager *transport.TransportCoordinator
	httpHandlers     []BusinessHTTPHandler
	grpcServices     []BusinessGRPCService
	healthChecks     []BusinessHealthCheck
	mu               sync.RWMutex
}

// NewDefaultServiceRegistry creates a new service registry with the given transport manager
func NewDefaultServiceRegistry(transportManager *transport.TransportCoordinator) *DefaultServiceRegistry {
	return &DefaultServiceRegistry{
		transportManager: transportManager,
		httpHandlers:     make([]BusinessHTTPHandler, 0),
		grpcServices:     make([]BusinessGRPCService, 0),
		healthChecks:     make([]BusinessHealthCheck, 0),
	}
}

// RegisterHTTPHandler registers an HTTP service handler with automatic route registration
func (r *DefaultServiceRegistry) RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error {
	if handler == nil {
		return fmt.Errorf("HTTP handler cannot be nil")
	}

	serviceName := handler.GetServiceName()
	if err := r.validateServiceName(serviceName); err != nil {
		return fmt.Errorf("invalid HTTP handler service name: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate service names across all service types
	if r.isServiceNameRegistered(serviceName) {
		return fmt.Errorf("service with name '%s' is already registered", serviceName)
	}

	// Create adapter to bridge server framework handler with transport layer
	adapter := &httpHandlerAdapter{
		handler: handler,
	}

	// Automatically register with transport manager
	if err := r.transportManager.RegisterHTTPService(adapter); err != nil {
		return fmt.Errorf("failed to automatically register HTTP handler '%s' with transport manager: %w", serviceName, err)
	}

	r.httpHandlers = append(r.httpHandlers, handler)
	return nil
}

// RegisterGRPCService registers a gRPC service with automatic server binding
func (r *DefaultServiceRegistry) RegisterBusinessGRPCService(service BusinessGRPCService) error {
	if service == nil {
		return fmt.Errorf("gRPC service cannot be nil")
	}

	serviceName := service.GetServiceName()
	if err := r.validateServiceName(serviceName); err != nil {
		return fmt.Errorf("invalid gRPC service name: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate service names across all service types
	if r.isServiceNameRegistered(serviceName) {
		return fmt.Errorf("service with name '%s' is already registered", serviceName)
	}

	// Create adapter to bridge server framework service with transport layer
	adapter := &grpcServiceAdapter{
		service: service,
	}

	// Automatically register with transport manager
	if err := r.transportManager.RegisterGRPCService(adapter); err != nil {
		return fmt.Errorf("failed to automatically register gRPC service '%s' with transport manager: %w", serviceName, err)
	}

	r.grpcServices = append(r.grpcServices, service)
	return nil
}

// RegisterHealthCheck registers a health check for a service with automatic endpoint creation
func (r *DefaultServiceRegistry) RegisterBusinessHealthCheck(check BusinessHealthCheck) error {
	if check == nil {
		return fmt.Errorf("health check cannot be nil")
	}

	serviceName := check.GetServiceName()
	if err := r.validateServiceName(serviceName); err != nil {
		return fmt.Errorf("invalid health check service name: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate health check service names
	for _, existing := range r.healthChecks {
		if existing.GetServiceName() == serviceName {
			return fmt.Errorf("health check with service name '%s' is already registered", serviceName)
		}
	}

	r.healthChecks = append(r.healthChecks, check)
	return nil
}

// GetHTTPHandlers returns all registered HTTP handlers
func (r *DefaultServiceRegistry) GetBusinessHTTPHandlers() []BusinessHTTPHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]BusinessHTTPHandler, len(r.httpHandlers))
	copy(handlers, r.httpHandlers)
	return handlers
}

// GetGRPCServices returns all registered gRPC services
func (r *DefaultServiceRegistry) GetBusinessGRPCServices() []BusinessGRPCService {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]BusinessGRPCService, len(r.grpcServices))
	copy(services, r.grpcServices)
	return services
}

// GetHealthChecks returns all registered health checks
func (r *DefaultServiceRegistry) GetBusinessHealthChecks() []BusinessHealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checks := make([]BusinessHealthCheck, len(r.healthChecks))
	copy(checks, r.healthChecks)
	return checks
}

// CheckAllHealth performs health checks on all registered services
func (r *DefaultServiceRegistry) CheckAllHealth(ctx context.Context) map[string]*types.HealthStatus {
	r.mu.RLock()
	checks := make([]BusinessHealthCheck, len(r.healthChecks))
	copy(checks, r.healthChecks)
	r.mu.RUnlock()

	results := make(map[string]*types.HealthStatus)

	for _, check := range checks {
		serviceName := check.GetServiceName()

		// Create a timeout context for each health check
		healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		if err := check.Check(healthCtx); err != nil {
			// Create an unhealthy status if health check failed
			results[serviceName] = &types.HealthStatus{
				Status:       types.HealthStatusUnhealthy,
				Timestamp:    time.Now(),
				Version:      "unknown",
				Uptime:       0,
				Dependencies: make(map[string]types.DependencyStatus),
			}
		} else {
			// Create a healthy status
			results[serviceName] = &types.HealthStatus{
				Status:       types.HealthStatusHealthy,
				Timestamp:    time.Now(),
				Version:      "1.0.0", // Default version
				Uptime:       0,       // Would need to be tracked separately
				Dependencies: make(map[string]types.DependencyStatus),
			}
		}

		cancel()
	}

	return results
}

// GetOverallHealthStatus aggregates all service health checks into an overall status
func (r *DefaultServiceRegistry) GetOverallHealthStatus(ctx context.Context) *types.HealthStatus {
	serviceHealthResults := r.CheckAllHealth(ctx)

	overallStatus := &types.HealthStatus{
		Status:       types.HealthStatusHealthy,
		Timestamp:    time.Now(),
		Version:      "1.0.0",
		Uptime:       0,
		Dependencies: make(map[string]types.DependencyStatus),
	}

	// If no health checks are registered, consider the system healthy
	if len(serviceHealthResults) == 0 {
		return overallStatus
	}

	// Check if any service is unhealthy
	hasUnhealthyService := false
	for serviceName, status := range serviceHealthResults {
		if status.Status == types.HealthStatusUnhealthy {
			hasUnhealthyService = true
		}

		// Add service status as a dependency
		dependencyStatus := types.NewDependencyStatus(types.DependencyStatusUp, 0)
		if status.Status == types.HealthStatusUnhealthy {
			dependencyStatus = types.NewDependencyStatus(types.DependencyStatusDown, 0)
		}

		overallStatus.Dependencies[serviceName] = dependencyStatus
	}

	// Set overall status based on individual service health
	if hasUnhealthyService {
		overallStatus.Status = types.HealthStatusUnhealthy
	}

	return overallStatus
}

// CreateHealthCheckEndpoint creates a default health check implementation for a service
func (r *DefaultServiceRegistry) CreateBusinessHealthCheckEndpoint(serviceName string, checkFunc func(ctx context.Context) error) BusinessHealthCheck {
	return &defaultHealthCheck{
		serviceName: serviceName,
		checkFunc:   checkFunc,
	}
}

// RegisterServiceWithHealthCheck registers a service along with its health check
func (r *DefaultServiceRegistry) RegisterBusinessServiceWithHealthCheck(handler BusinessHTTPHandler, healthCheckFunc func(ctx context.Context) error) error {
	// Register the HTTP handler
	if err := r.RegisterBusinessHTTPHandler(handler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Create and register health check
	healthCheck := r.CreateBusinessHealthCheckEndpoint(handler.GetServiceName(), healthCheckFunc)
	if err := r.RegisterBusinessHealthCheck(healthCheck); err != nil {
		// If health check registration fails, we should clean up the HTTP handler
		// For now, we'll just return the error
		return fmt.Errorf("failed to register health check for service '%s': %w", handler.GetServiceName(), err)
	}

	return nil
}

// RegisterGRPCServiceWithHealthCheck registers a gRPC service along with its health check
func (r *DefaultServiceRegistry) RegisterBusinessGRPCServiceWithHealthCheck(service BusinessGRPCService, healthCheckFunc func(ctx context.Context) error) error {
	// Register the gRPC service
	if err := r.RegisterBusinessGRPCService(service); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	// Create and register health check
	healthCheck := r.CreateBusinessHealthCheckEndpoint(service.GetServiceName(), healthCheckFunc)
	if err := r.RegisterBusinessHealthCheck(healthCheck); err != nil {
		// If health check registration fails, we should clean up the gRPC service
		// For now, we'll just return the error
		return fmt.Errorf("failed to register health check for service '%s': %w", service.GetServiceName(), err)
	}

	return nil
}

// validateServiceName validates that a service name meets the requirements
func (r *DefaultServiceRegistry) validateServiceName(serviceName string) error {
	if serviceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if len(serviceName) > 50 {
		return fmt.Errorf("service name too long (max 50 characters)")
	}

	// Only allow alphanumeric characters, hyphens, and underscores
	for _, r := range serviceName {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return fmt.Errorf("service name can only contain letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

// isServiceNameRegistered checks if a service name is already registered across all service types
// This method assumes the caller holds the mutex lock
func (r *DefaultServiceRegistry) isServiceNameRegistered(serviceName string) bool {
	// Check HTTP handlers
	for _, handler := range r.httpHandlers {
		if handler.GetServiceName() == serviceName {
			return true
		}
	}

	// Check gRPC services
	for _, service := range r.grpcServices {
		if service.GetServiceName() == serviceName {
			return true
		}
	}

	// Note: Health checks are allowed to have the same name as services
	// since they are typically associated with a specific service

	return false
}

// GetRegisteredServiceNames returns all registered service names
func (r *DefaultServiceRegistry) GetRegisteredServiceNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string

	// Add HTTP handler names
	for _, handler := range r.httpHandlers {
		names = append(names, handler.GetServiceName())
	}

	// Add gRPC service names
	for _, service := range r.grpcServices {
		names = append(names, service.GetServiceName())
	}

	return names
}

// GetServiceCount returns the total number of registered services
func (r *DefaultServiceRegistry) GetServiceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.httpHandlers) + len(r.grpcServices)
}

// defaultHealthCheck is a default implementation of the HealthCheck interface
type defaultHealthCheck struct {
	serviceName string
	checkFunc   func(ctx context.Context) error
}

// Check performs the health check using the provided function
func (h *defaultHealthCheck) Check(ctx context.Context) error {
	if h.checkFunc == nil {
		// If no check function is provided, assume the service is healthy
		return nil
	}
	return h.checkFunc(ctx)
}

// GetServiceName returns the service name for this health check
func (h *defaultHealthCheck) GetServiceName() string {
	return h.serviceName
}
