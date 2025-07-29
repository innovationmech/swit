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

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/transport"
	"google.golang.org/grpc"
)

// serviceRegistry implements ServiceRegistry interface
type serviceRegistry struct {
	transportManager *transport.Manager
	mu               sync.RWMutex
	httpHandlers     []HTTPHandler
	grpcServices     []GRPCService
	healthChecks     []HealthCheck
}

// newServiceRegistry creates a new service registry
func newServiceRegistry(transportManager *transport.Manager) *serviceRegistry {
	return &serviceRegistry{
		transportManager: transportManager,
		httpHandlers:     make([]HTTPHandler, 0),
		grpcServices:     make([]GRPCService, 0),
		healthChecks:     make([]HealthCheck, 0),
	}
}

// RegisterHTTPHandler registers an HTTP handler
func (r *serviceRegistry) RegisterHTTPHandler(handler HTTPHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if handler == nil {
		return fmt.Errorf("HTTP handler cannot be nil")
	}

	r.httpHandlers = append(r.httpHandlers, handler)
	return nil
}

// RegisterGRPCService registers a gRPC service
func (r *serviceRegistry) RegisterGRPCService(service GRPCService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if service == nil {
		return fmt.Errorf("gRPC service cannot be nil")
	}

	r.grpcServices = append(r.grpcServices, service)
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
}
