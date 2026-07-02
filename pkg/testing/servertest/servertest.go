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

// Package servertest provides framework-level test utilities for the SWIT
// base server framework (pkg/server). It offers ready-to-use service
// registrar, HTTP/gRPC handler, health check and dependency container
// implementations, plus a harness for starting a fully wired test server
// with dynamic port allocation.
//
// Typical usage:
//
//	registrar := servertest.NewServiceRegistrar()
//	handler := servertest.NewHTTPHandler("my-service")
//	handler.AddRoute(http.MethodGet, "/ping", func(c *gin.Context) {
//		c.JSON(http.StatusOK, gin.H{"status": "ok"})
//	})
//	registrar.AddHTTPHandler(handler)
//
//	srv := servertest.StartServer(t, servertest.NewServerConfig("my-service"), registrar, nil)
//	resp, err := http.Get(servertest.HTTPBaseURL(srv) + "/ping")
package servertest

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
)

// Compile-time interface compliance checks.
var (
	_ server.BusinessServiceRegistrar    = (*ServiceRegistrar)(nil)
	_ server.BusinessHTTPHandler         = (*HTTPHandler)(nil)
	_ server.BusinessGRPCService         = (*GRPCService)(nil)
	_ server.BusinessHealthCheck         = (*HealthCheck)(nil)
	_ server.BusinessDependencyContainer = (*DependencyContainer)(nil)
)

// NewServerConfig returns a server configuration suitable for tests:
// dynamic port allocation, HTTP test mode enabled and service discovery
// disabled.
func NewServerConfig(serviceName string) *server.ServerConfig {
	config := &server.ServerConfig{ServiceName: serviceName}
	config.SetDefaults()

	config.HTTP.Enabled = true
	config.HTTP.Port = "0"
	config.HTTP.Address = ":0"
	config.HTTP.TestMode = true
	config.GRPC.Enabled = true
	config.GRPC.Port = "0"
	config.GRPC.Address = ":0"
	config.Discovery.Enabled = false
	config.ShutdownTimeout = 10 * time.Second

	return config
}

// ServiceRegistrar is a composite server.BusinessServiceRegistrar that
// registers all added HTTP handlers, gRPC services and health checks.
type ServiceRegistrar struct {
	httpHandlers []server.BusinessHTTPHandler
	grpcServices []server.BusinessGRPCService
	healthChecks []server.BusinessHealthCheck
}

// NewServiceRegistrar creates an empty composite service registrar.
func NewServiceRegistrar() *ServiceRegistrar {
	return &ServiceRegistrar{}
}

// AddHTTPHandler adds an HTTP handler to be registered.
func (r *ServiceRegistrar) AddHTTPHandler(handler server.BusinessHTTPHandler) {
	r.httpHandlers = append(r.httpHandlers, handler)
}

// AddGRPCService adds a gRPC service to be registered.
func (r *ServiceRegistrar) AddGRPCService(service server.BusinessGRPCService) {
	r.grpcServices = append(r.grpcServices, service)
}

// AddHealthCheck adds a health check to be registered.
func (r *ServiceRegistrar) AddHealthCheck(check server.BusinessHealthCheck) {
	r.healthChecks = append(r.healthChecks, check)
}

// RegisterServices registers all added components with the framework registry.
func (r *ServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	for _, handler := range r.httpHandlers {
		if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
			return fmt.Errorf("failed to register HTTP handler %s: %w", handler.GetServiceName(), err)
		}
	}
	for _, service := range r.grpcServices {
		if err := registry.RegisterBusinessGRPCService(service); err != nil {
			return fmt.Errorf("failed to register gRPC service %s: %w", service.GetServiceName(), err)
		}
	}
	for _, check := range r.healthChecks {
		if err := registry.RegisterBusinessHealthCheck(check); err != nil {
			return fmt.Errorf("failed to register health check %s: %w", check.GetServiceName(), err)
		}
	}
	return nil
}

type httpRoute struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

// HTTPHandler is a configurable server.BusinessHTTPHandler backed by Gin
// routes.
type HTTPHandler struct {
	serviceName string
	routes      []httpRoute
}

// NewHTTPHandler creates a new test HTTP handler with the given service name.
func NewHTTPHandler(serviceName string) *HTTPHandler {
	return &HTTPHandler{serviceName: serviceName}
}

// AddRoute registers a route with the given HTTP method and path.
func (h *HTTPHandler) AddRoute(method, path string, handler gin.HandlerFunc) {
	h.routes = append(h.routes, httpRoute{method: method, path: path, handler: handler})
}

// RegisterRoutes implements server.BusinessHTTPHandler.
func (h *HTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}
	for _, route := range h.routes {
		ginRouter.Handle(route.method, route.path, route.handler)
	}
	return nil
}

// GetServiceName implements server.BusinessHTTPHandler.
func (h *HTTPHandler) GetServiceName() string {
	return h.serviceName
}

// GRPCService is a configurable server.BusinessGRPCService. The optional
// register function is invoked with the underlying *grpc.Server, allowing
// tests to register real gRPC service implementations.
type GRPCService struct {
	serviceName string
	registerFn  func(*grpc.Server) error
	registered  bool
}

// NewGRPCService creates a new test gRPC service. registerFn may be nil when
// only registration bookkeeping is needed.
func NewGRPCService(serviceName string, registerFn func(*grpc.Server) error) *GRPCService {
	return &GRPCService{serviceName: serviceName, registerFn: registerFn}
}

// RegisterGRPC implements server.BusinessGRPCService.
func (s *GRPCService) RegisterGRPC(srv interface{}) error {
	grpcServer, ok := srv.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", srv)
	}
	if s.registerFn != nil {
		if err := s.registerFn(grpcServer); err != nil {
			return err
		}
	}
	s.registered = true
	return nil
}

// GetServiceName implements server.BusinessGRPCService.
func (s *GRPCService) GetServiceName() string {
	return s.serviceName
}

// IsRegistered reports whether RegisterGRPC has been invoked.
func (s *GRPCService) IsRegistered() bool {
	return s.registered
}

// HealthCheck is a togglable server.BusinessHealthCheck implementation.
type HealthCheck struct {
	serviceName string
	mu          sync.RWMutex
	healthy     bool
}

// NewHealthCheck creates a new test health check with the given initial state.
func NewHealthCheck(serviceName string, healthy bool) *HealthCheck {
	return &HealthCheck{serviceName: serviceName, healthy: healthy}
}

// Check implements server.BusinessHealthCheck.
func (h *HealthCheck) Check(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.healthy {
		return fmt.Errorf("service %s is unhealthy", h.serviceName)
	}
	return nil
}

// GetServiceName implements server.BusinessHealthCheck.
func (h *HealthCheck) GetServiceName() string {
	return h.serviceName
}

// SetHealthy toggles the health state.
func (h *HealthCheck) SetHealthy(healthy bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.healthy = healthy
}

// DependencyContainer is a map-backed server.BusinessDependencyContainer
// with initialization and close tracking.
type DependencyContainer struct {
	mu          sync.RWMutex
	services    map[string]interface{}
	initialized bool
	closed      bool
}

// NewDependencyContainer creates an empty test dependency container.
func NewDependencyContainer() *DependencyContainer {
	return &DependencyContainer{services: make(map[string]interface{})}
}

// AddService registers a service instance under the given name.
func (d *DependencyContainer) AddService(name string, service interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.services[name] = service
}

// GetService implements server.BusinessDependencyContainer.
func (d *DependencyContainer) GetService(name string) (interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if service, exists := d.services[name]; exists {
		return service, nil
	}
	return nil, fmt.Errorf("service %s not found", name)
}

// Initialize marks the container as initialized.
func (d *DependencyContainer) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.initialized = true
	return nil
}

// Close implements server.BusinessDependencyContainer.
func (d *DependencyContainer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

// IsInitialized reports whether Initialize has been called.
func (d *DependencyContainer) IsInitialized() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.initialized
}

// IsClosed reports whether Close has been called.
func (d *DependencyContainer) IsClosed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.closed
}

// StartServer creates and starts a framework server with the given
// configuration, registrar and optional dependency container. The server is
// automatically shut down via t.Cleanup when the test finishes. It fails the
// test if creation or startup fails.
func StartServer(t testing.TB, config *server.ServerConfig, registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) *server.BusinessServerImpl {
	t.Helper()

	logger.InitLogger()
	gin.SetMode(gin.TestMode)

	srv, err := server.NewBusinessServerCore(config, registrar, deps)
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}

	t.Cleanup(func() {
		if err := srv.Shutdown(); err != nil {
			t.Logf("test server shutdown error: %v", err)
		}
	})

	return srv
}

// HTTPBaseURL returns a base URL (http://host:port) usable by HTTP clients
// to reach the test server, normalizing wildcard listen addresses such as
// "[::]:8080" or ":8080" to localhost.
func HTTPBaseURL(srv *server.BusinessServerImpl) string {
	return "http://" + NormalizeAddr(srv.GetHTTPAddress())
}

// NormalizeAddr rewrites wildcard listen addresses to a dialable
// localhost-based address.
func NormalizeAddr(addr string) string {
	if strings.HasPrefix(addr, "[::]:") {
		return "localhost:" + strings.TrimPrefix(addr, "[::]:")
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "localhost:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

// WaitForHTTPReady polls the given URL until it returns any HTTP response or
// the timeout elapses. It fails the test on timeout.
func WaitForHTTPReady(t testing.TB, url string, timeout time.Duration) {
	t.Helper()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for HTTP server at %s", url)
}
