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

// HTTPTransport handles HTTP transport using Gin
// Implements the Transport interface for HTTP protocol
type HTTPTransport struct {
	server          *http.Server
	router          *gin.Engine
	addr            string
	serviceRegistry *EnhancedHandlerRegistry
	mu              sync.RWMutex
}

// NewHTTPTransport creates a new HTTP transport instance
func NewHTTPTransport() *HTTPTransport {
	return &HTTPTransport{
		router:          gin.New(),
		serviceRegistry: NewEnhancedServiceRegistry(),
	}
}

// Start starts the HTTP server on the specified address
func (h *HTTPTransport) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.addr == "" {
		h.addr = ":8080" // Default port
	}

	h.server = &http.Server{
		Addr:    h.addr,
		Handler: h.router,
	}

	ln, err := net.Listen("tcp", h.addr)
	if err != nil {
		return fmt.Errorf("failed to create HTTP listener: %w", err)
	}

	// Update actual address in case of :0 port
	h.addr = ln.Addr().String()

	logger.Logger.Info("Starting HTTP transport",
		zap.String("address", h.addr))

	go func() {
		if err := h.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (h *HTTPTransport) Stop(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.server == nil {
		return nil
	}

	logger.Logger.Info("Stopping HTTP transport")

	// Shutdown all services first
	if err := h.serviceRegistry.ShutdownAll(ctx); err != nil {
		logger.Logger.Error("Failed to shutdown services", zap.Error(err))
	}

	return h.server.Shutdown(ctx)
}

// GetName returns the transport name
func (h *HTTPTransport) GetName() string {
	return "http"
}

// SetAddress sets the HTTP server address
func (h *HTTPTransport) SetAddress(addr string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.addr = addr
}

// GetAddress returns the current HTTP server address
func (h *HTTPTransport) GetAddress() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.addr
}

// GetRouter returns the Gin router for route registration
func (h *HTTPTransport) GetRouter() *gin.Engine {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.router
}

// GetPort returns the actual port the server is listening on
func (h *HTTPTransport) GetPort() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.addr == "" {
		return 0
	}

	_, portStr, err := net.SplitHostPort(h.addr)
	if err != nil {
		return 0
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}

	return port
}

// GetServiceRegistry returns the enhanced service registry
func (h *HTTPTransport) GetServiceRegistry() *EnhancedHandlerRegistry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.serviceRegistry
}

// RegisterService registers a service handler with the transport
func (h *HTTPTransport) RegisterService(handler HandlerRegister) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.serviceRegistry.Register(handler)
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
