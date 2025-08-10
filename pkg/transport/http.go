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
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// HTTPTransportConfig contains configuration for HTTP transport
type HTTPTransportConfig struct {
	Address     string
	Port        string
	TestMode    bool   // Enables test-specific features
	TestPort    string // Override port for testing
	EnableReady bool   // Enables ready channel for testing
}

// HTTPNetworkService implements NetworkTransport interface for HTTP
type HTTPNetworkService struct {
	server          *http.Server
	router          *gin.Engine
	address         string
	config          *HTTPTransportConfig
	ready           chan struct{}
	readyOnce       sync.Once
	mu              sync.RWMutex
	serviceRegistry *TransportServiceRegistry
}

// NewHTTPNetworkService creates a new HTTP network service with default configuration
func NewHTTPNetworkService() *HTTPNetworkService {
	return NewHTTPNetworkServiceWithConfig(&HTTPTransportConfig{
		Address:     ":8080",
		EnableReady: true,
	})
}

// NewHTTPNetworkServiceWithConfig creates a new HTTP network service with custom configuration
func NewHTTPNetworkServiceWithConfig(config *HTTPTransportConfig) *HTTPNetworkService {
	if config == nil {
		config = &HTTPTransportConfig{
			Address:     ":8080",
			EnableReady: true,
		}
	}

	transport := &HTTPNetworkService{
		router:          gin.New(), // Use gin.New() instead of gin.Default() for more control
		config:          config,
		serviceRegistry: NewTransportServiceRegistry(),
	}

	if config.EnableReady {
		transport.ready = make(chan struct{})
	}

	return transport
}

// Start implements NetworkTransport interface
func (h *HTTPNetworkService) Start(ctx context.Context) error {
	// Determine address to use
	address := h.determineAddress()
	h.address = address

	h.mu.Lock()
	// Check if server is already running
	if h.server != nil {
		h.mu.Unlock()
		return nil // Already started
	}

	// Create listener
	ln, err := net.Listen("tcp", address)
	if err != nil {
		h.mu.Unlock()
		return fmt.Errorf("failed to create HTTP listener: %w", err)
	}

	// Update actual address in case of :0 port
	h.address = ln.Addr().String()

	h.server = &http.Server{
		Addr:    h.address,
		Handler: h.router,
	}

	// Reset ready channel for each start if enabled
	if h.config.EnableReady {
		h.ready = make(chan struct{})
		h.readyOnce = sync.Once{}
	}

	// Capture references inside the lock to avoid race conditions
	ready := h.ready
	readyOnce := &h.readyOnce
	server := h.server
	h.mu.Unlock()

	logger.Logger.Info("Starting HTTP transport", zap.String("address", h.address))

	// Start serving in a goroutine
	go func() {
		// Signal that server is ready (only once) if enabled
		if h.config.EnableReady && ready != nil {
			readyOnce.Do(func() {
				close(ready)
			})
		}

		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Logger.Error("HTTP server failed to serve", zap.Error(err))
		}
	}()

	return nil
}

// Stop implements Transport interface
func (h *HTTPNetworkService) Stop(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.server == nil {
		logger.Logger.Debug("HTTP server not initialized, skipping shutdown")
		return nil
	}

	// First shutdown all services
	if err := h.serviceRegistry.ShutdownAll(ctx); err != nil {
		logger.Logger.Error("Failed to shutdown services", zap.Error(err))
		// Continue with server shutdown even if service shutdown fails
	}

	if err := h.server.Shutdown(ctx); err != nil {
		logger.Logger.Error("HTTP server shutdown error", zap.Error(err))
		return err
	}

	// Reset server to nil so it can be started again
	h.server = nil

	logger.Logger.Info("HTTP transport stopped successfully")
	return nil
}

// GetName implements Transport interface
func (h *HTTPNetworkService) GetName() string {
	return "http"
}

// GetAddress implements Transport interface
func (h *HTTPNetworkService) GetAddress() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Try actual address first (set when server is running)
	address := h.address
	// If not running, try configured address
	if address == "" && h.config != nil && h.config.Address != "" {
		address = h.config.Address
	}

	return address
}

// GetRouter returns the Gin router for route registration
func (h *HTTPNetworkService) GetRouter() *gin.Engine {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.router
}

// GetPort returns the actual port the server is listening on
func (h *HTTPNetworkService) GetPort() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Try actual address first (set when server is running)
	address := h.address
	// If not running, try configured address
	if address == "" && h.config != nil && h.config.Address != "" {
		address = h.config.Address
	}

	if address == "" {
		return 0
	}

	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return 0
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}

	return port
}

// WaitReady waits for the HTTP server to be ready (for testing)
func (h *HTTPNetworkService) WaitReady() <-chan struct{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.ready != nil {
		return h.ready
	}
	// Return a closed channel if ready is disabled
	closed := make(chan struct{})
	close(closed)
	return closed
}

// SetTestPort sets a custom port for testing (use "0" for dynamic port allocation)
func (h *HTTPNetworkService) SetTestPort(port string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.config == nil {
		h.config = &HTTPTransportConfig{}
	}
	h.config.TestPort = port
	h.config.TestMode = true
}

// SetAddress sets the HTTP server address
func (h *HTTPNetworkService) SetAddress(addr string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.config == nil {
		h.config = &HTTPTransportConfig{}
	}
	h.config.Address = addr
}

// RegisterHandler registers a service handler with the transport
func (h *HTTPNetworkService) RegisterHandler(handler TransportServiceHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Register the service with the registry
	if err := h.serviceRegistry.Register(handler); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	return nil
}

// RegisterService is an alias for RegisterHandler for backward compatibility
func (h *HTTPNetworkService) RegisterService(handler TransportServiceHandler) error {
	return h.RegisterHandler(handler)
}

// InitializeServices initializes all registered services
func (h *HTTPNetworkService) InitializeServices(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.serviceRegistry.InitializeTransportServices(ctx)
}

// RegisterAllRoutes registers HTTP routes for all registered services
func (h *HTTPNetworkService) RegisterAllRoutes() error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.serviceRegistry.BindAllHTTPEndpoints(h.router)
}

// GetServiceRegistry returns the service registry
func (h *HTTPNetworkService) GetServiceRegistry() *TransportServiceRegistry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.serviceRegistry
}

// ShutdownServices gracefully shuts down all registered services
func (h *HTTPNetworkService) ShutdownServices(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.serviceRegistry.ShutdownAll(ctx)
}

// determineAddress determines the address to use for the HTTP server
func (h *HTTPNetworkService) determineAddress() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Test port override takes highest priority
	if h.config != nil && h.config.TestMode && h.config.TestPort != "" {
		return ":" + h.config.TestPort
	}

	// Use configured address
	if h.config != nil && h.config.Address != "" {
		return h.config.Address
	}

	// Default fallback
	return ":8080"
}
