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
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// HTTPTransport implements Transport interface for HTTP
type HTTPTransport struct {
	server          *http.Server
	router          *gin.Engine
	address         string
	testPort        string // Override port for testing (empty means use config)
	ready           chan struct{}
	readyOnce       sync.Once
	mu              sync.RWMutex
	serviceRegistry *EnhancedHandlerRegistry
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport() *HTTPTransport {
	return &HTTPTransport{
		router:          gin.Default(),
		ready:           make(chan struct{}),
		serviceRegistry: NewEnhancedServiceRegistry(),
	}
}

// Start implements Transport interface
func (h *HTTPTransport) Start(ctx context.Context) error {
	// Check for test port override first
	h.mu.RLock()
	testPort := h.testPort
	h.mu.RUnlock()

	var port string
	if testPort != "" {
		port = testPort
	} else {
		cfg := config.GetConfig()
		port = cfg.Server.Port
		if port == "" {
			port = "8080"
		}
	}

	h.address = fmt.Sprintf(":%s", port)

	h.mu.Lock()
	// Check if server is already running
	if h.server != nil {
		h.mu.Unlock()
		return nil // Already started
	}

	h.server = &http.Server{
		Addr:    h.address,
		Handler: h.router,
	}
	// Reset ready channel for each start
	h.ready = make(chan struct{})
	h.readyOnce = sync.Once{}
	// Capture references inside the lock to avoid race conditions
	ready := h.ready
	readyOnce := &h.readyOnce
	server := h.server
	h.mu.Unlock()

	logger.Logger.Info("Starting HTTP server", zap.String("address", h.address))

	// Start serving in a goroutine
	go func() {
		// Signal that server is ready (only once)
		readyOnce.Do(func() {
			close(ready)
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Error("HTTP server failed to serve", zap.Error(err))
		}
	}()

	return nil
}

// Stop implements Transport interface
func (h *HTTPTransport) Stop(ctx context.Context) error {
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

	logger.Logger.Info("HTTP server shut down successfully")
	return nil
}

// Name implements Transport interface
func (h *HTTPTransport) Name() string {
	return "http"
}

// Address implements Transport interface
func (h *HTTPTransport) Address() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.address
}

// GetRouter returns the Gin router for route registration
func (h *HTTPTransport) GetRouter() *gin.Engine {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.router
}

// WaitReady waits for the HTTP server to be ready
func (h *HTTPTransport) WaitReady() <-chan struct{} {
	return h.ready
}

// SetTestPort sets a custom port for testing (use "0" for dynamic port allocation)
func (h *HTTPTransport) SetTestPort(port string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.testPort = port
}

// RegisterHandler registers a service handler with the transport
func (h *HTTPTransport) RegisterHandler(handler HandlerRegister) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// HandlerRegister the service with the registry
	if err := h.serviceRegistry.Register(handler); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	return nil
}

// InitializeServices initializes all registered services
func (h *HTTPTransport) InitializeServices(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.serviceRegistry.InitializeAll(ctx)
}

// RegisterAllRoutes registers HTTP routes for all registered services
func (h *HTTPTransport) RegisterAllRoutes() error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.serviceRegistry.RegisterAllHTTP(h.router)
}

// GetServiceRegistry returns the service registry
func (h *HTTPTransport) GetServiceRegistry() *EnhancedHandlerRegistry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.serviceRegistry
}

// ShutdownServices gracefully shuts down all registered services
func (h *HTTPTransport) ShutdownServices(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.serviceRegistry.ShutdownAll(ctx)
}
