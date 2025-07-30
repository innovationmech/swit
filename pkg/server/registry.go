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
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
	"google.golang.org/grpc"
)

// healthAggregator manages health check aggregation
type healthAggregator struct {
	mu           sync.RWMutex
	lastCheck    time.Time
	cachedStatus *types.HealthStatus
	cacheTTL     time.Duration
}

// newHealthAggregator creates a new health aggregator
func newHealthAggregator() *healthAggregator {
	return &healthAggregator{
		cacheTTL: 30 * time.Second, // Cache health status for 30 seconds
	}
}

// validateServiceName validates service name format and content
func (r *serviceRegistry) validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("service name cannot contain whitespace characters")
	}

	// Check length
	if len(name) > 64 {
		return fmt.Errorf("service name cannot exceed 64 characters")
	}

	return nil
}

// serviceRegistry implements ServiceRegistry interface
type serviceRegistry struct {
	transportManager *transport.Manager
	mu               sync.RWMutex
	httpHandlers     []HTTPHandler
	grpcServices     []GRPCService
	healthChecks     []HealthCheck
	// Service name tracking for duplicate prevention by type
	registeredNames map[string]map[string]bool
	// Health check aggregation
	healthAggregator *healthAggregator
}

// newServiceRegistry creates a new service registry
func newServiceRegistry(transportManager *transport.Manager) *serviceRegistry {
	return &serviceRegistry{
		transportManager: transportManager,
		httpHandlers:     make([]HTTPHandler, 0),
		grpcServices:     make([]GRPCService, 0),
		healthChecks:     make([]HealthCheck, 0),
		registeredNames:  make(map[string]map[string]bool),
		healthAggregator: newHealthAggregator(),
	}
}

// RegisterHTTPHandler registers an HTTP handler with validation
func (r *serviceRegistry) RegisterHTTPHandler(handler HTTPHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if handler == nil {
		return fmt.Errorf("HTTP handler cannot be nil")
	}

	// Validate service name
	serviceName := handler.GetServiceName()
	if err := r.validateServiceName(serviceName); err != nil {
		return fmt.Errorf("invalid service name: %w", err)
	}

	// Check for duplicate registration of HTTP handler
	if r.registeredNames["http"] == nil {
		r.registeredNames["http"] = make(map[string]bool)
	}
	if r.registeredNames["http"][serviceName] {
		return fmt.Errorf("HTTP service '%s' is already registered", serviceName)
	}

	r.httpHandlers = append(r.httpHandlers, handler)
	r.registeredNames["http"][serviceName] = true
	return nil
}

// RegisterGRPCService registers a gRPC service with validation
func (r *serviceRegistry) RegisterGRPCService(service GRPCService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if service == nil {
		return fmt.Errorf("gRPC service cannot be nil")
	}

	// Validate service name
	serviceName := service.GetServiceName()
	if err := r.validateServiceName(serviceName); err != nil {
		return fmt.Errorf("invalid service name: %w", err)
	}

	// Check for duplicate registration of gRPC service
	if r.registeredNames["grpc"] == nil {
		r.registeredNames["grpc"] = make(map[string]bool)
	}
	if r.registeredNames["grpc"][serviceName] {
		return fmt.Errorf("gRPC service '%s' is already registered", serviceName)
	}

	r.grpcServices = append(r.grpcServices, service)
	r.registeredNames["grpc"][serviceName] = true
	return nil
}

// RegisterHealthCheck registers a health check
func (r *serviceRegistry) RegisterHealthCheck(check HealthCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if check == nil {
		return fmt.Errorf("health check cannot be nil")
	}

	r.healthChecks = append(r.healthChecks, check)
	return nil
}

// GetHTTPHandlers returns all registered HTTP handlers
func (r *serviceRegistry) GetHTTPHandlers() []HTTPHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	handlers := make([]HTTPHandler, len(r.httpHandlers))
	copy(handlers, r.httpHandlers)
	return handlers
}

// GetGRPCServices returns all registered gRPC services
func (r *serviceRegistry) GetGRPCServices() []GRPCService {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	services := make([]GRPCService, len(r.grpcServices))
	copy(services, r.grpcServices)
	return services
}

// GetHealthChecks returns all registered health checks
func (r *serviceRegistry) GetHealthChecks() []HealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	checks := make([]HealthCheck, len(r.healthChecks))
	copy(checks, r.healthChecks)
	return checks
}

// RegisterAllHTTP registers all HTTP handlers with the given router
func (r *serviceRegistry) RegisterAllHTTP(router *gin.Engine) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, handler := range r.httpHandlers {
		if err := handler.RegisterRoutes(router); err != nil {
			return fmt.Errorf("failed to register HTTP routes for handler: %w", err)
		}
	}

	return nil
}

// RegisterAllGRPC registers all gRPC services with the given server
func (r *serviceRegistry) RegisterAllGRPC(server *grpc.Server) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, service := range r.grpcServices {
		if err := service.RegisterService(server); err != nil {
			return fmt.Errorf("failed to register gRPC service: %w", err)
		}
	}

	return nil
}

// Count returns the total number of registered services
func (r *serviceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.httpHandlers) + len(r.grpcServices)
}

// Clear removes all registered services
func (r *serviceRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.httpHandlers = r.httpHandlers[:0]
	r.grpcServices = r.grpcServices[:0]
	r.healthChecks = r.healthChecks[:0]
	r.registeredNames = make(map[string]map[string]bool)
}

// CheckAggregatedHealth performs aggregated health check of all registered health checks
func (r *serviceRegistry) CheckAggregatedHealth(ctx context.Context) *types.HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if we have cached result
	if cachedStatus := r.healthAggregator.getCachedStatus(); cachedStatus != nil {
		return cachedStatus
	}

	// Perform health checks
	status := &types.HealthStatus{
		Status:       types.HealthStatusHealthy,
		Timestamp:    time.Now(),
		Version:      "1.0.0",
		Dependencies: make(map[string]types.DependencyStatus),
	}

	// Check all registered health checks
	for _, check := range r.healthChecks {
		checkResult := check.Check(ctx)
		if checkResult != nil {
			depStatus := types.DependencyStatus{
				Status:    checkResult.Status,
				Timestamp: checkResult.Timestamp,
			}
			status.Dependencies[check.GetName()] = depStatus

			// If any check is unhealthy, mark overall status as unhealthy
			if !checkResult.IsHealthy() {
				status.Status = types.HealthStatusUnhealthy
			}
		}
	}

	// Cache the result
	r.healthAggregator.cacheStatus(status)
	return status
}

// getCachedStatus returns cached health status if still valid
func (h *healthAggregator) getCachedStatus() *types.HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.cachedStatus != nil && time.Since(h.lastCheck) < h.cacheTTL {
		return h.cachedStatus
	}
	return nil
}

// cacheStatus caches the health status
func (h *healthAggregator) cacheStatus(status *types.HealthStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cachedStatus = status
	h.lastCheck = time.Now()
}

// AutoRegisterService automatically registers a service that implements multiple interfaces
func (r *serviceRegistry) AutoRegisterService(service interface{}) error {
	var errors []string

	// Try to register as HTTP handler
	if httpHandler, ok := service.(HTTPHandler); ok {
		if err := r.RegisterHTTPHandler(httpHandler); err != nil {
			errors = append(errors, fmt.Sprintf("HTTP registration failed: %v", err))
		}
	}

	// Try to register as gRPC service
	if grpcService, ok := service.(GRPCService); ok {
		if err := r.RegisterGRPCService(grpcService); err != nil {
			errors = append(errors, fmt.Sprintf("gRPC registration failed: %v", err))
		}
	}

	// Try to register as health check
	if healthCheck, ok := service.(HealthCheck); ok {
		if err := r.RegisterHealthCheck(healthCheck); err != nil {
			errors = append(errors, fmt.Sprintf("health check registration failed: %v", err))
		}
	}

	// Return error if any registration failed
	if len(errors) > 0 {
		return fmt.Errorf("auto registration failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// AutoRegisterServices automatically registers multiple services
func (r *serviceRegistry) AutoRegisterServices(services ...interface{}) error {
	var errors []string

	for i, service := range services {
		if err := r.AutoRegisterService(service); err != nil {
			errors = append(errors, fmt.Sprintf("service %d: %v", i, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch auto registration failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// IsServiceRegistered checks if a service with the given name is already registered for a specific type
func (r *serviceRegistry) IsServiceRegistered(serviceType, serviceName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.registeredNames[serviceType] == nil {
		return false
	}
	return r.registeredNames[serviceType][serviceName]
}

// GetRegisteredServiceNames returns a list of all registered service names by type
func (r *serviceRegistry) GetRegisteredServiceNames() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)
	for serviceType, services := range r.registeredNames {
		names := make([]string, 0, len(services))
		for name := range services {
			names = append(names, name)
		}
		result[serviceType] = names
	}
	return result
}

// GetServiceCounts returns counts of registered services by type
func (r *serviceRegistry) GetServiceCounts() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalServices := 0
	for _, services := range r.registeredNames {
		totalServices += len(services)
	}

	return map[string]int{
		"http":        len(r.httpHandlers),
		"grpc":        len(r.grpcServices),
		"healthCheck": len(r.healthChecks),
		"total":       totalServices,
	}
}
