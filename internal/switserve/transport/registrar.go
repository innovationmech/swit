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
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

// ServiceRegistrar defines the interface for service registration
type ServiceRegistrar interface {
	// RegisterGRPC registers gRPC services
	RegisterGRPC(server *grpc.Server) error
	// RegisterHTTP registers HTTP routes
	RegisterHTTP(router *gin.Engine) error
	// GetName returns the service name
	GetName() string
}

// ServiceRegistry manages service registrations
type ServiceRegistry struct {
	mu         sync.RWMutex
	registrars []ServiceRegistrar
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		registrars: make([]ServiceRegistrar, 0),
	}
}

// Register adds a service registrar
func (sr *ServiceRegistry) Register(registrar ServiceRegistrar) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.registrars = append(sr.registrars, registrar)
}

// RegisterAllGRPC registers all gRPC services
func (sr *ServiceRegistry) RegisterAllGRPC(server *grpc.Server) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	for _, registrar := range sr.registrars {
		if err := registrar.RegisterGRPC(server); err != nil {
			return err
		}
	}
	return nil
}

// RegisterAllHTTP registers all HTTP routes
func (sr *ServiceRegistry) RegisterAllHTTP(router *gin.Engine) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	for _, registrar := range sr.registrars {
		if err := registrar.RegisterHTTP(router); err != nil {
			return err
		}
	}
	return nil
}

// GetRegistrars returns all registered service registrars for debugging purposes
func (sr *ServiceRegistry) GetRegistrars() []ServiceRegistrar {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	// Return a copy to avoid race conditions
	result := make([]ServiceRegistrar, len(sr.registrars))
	copy(result, sr.registrars)
	return result
}
