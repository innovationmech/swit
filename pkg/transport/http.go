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
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	tlsconfig "github.com/innovationmech/swit/pkg/security/tls"
	"github.com/innovationmech/swit/pkg/tracing"
	"go.uber.org/zap"
)

// HTTPTransportConfig contains configuration for HTTP transport
type HTTPTransportConfig struct {
	Address        string
	Port           string
	TestMode       bool                   // Enables test-specific features
	TestPort       string                 // Override port for testing
	EnableReady    bool                   // Enables ready channel for testing
	TracingManager tracing.TracingManager // Tracing manager for automatic middleware setup
	TLS            *tlsconfig.TLSConfig   // TLS/mTLS configuration
}

// HTTPNetworkService implements NetworkTransport interface for HTTP
type HTTPNetworkService struct {
	server          *http.Server
	router          *gin.Engine
	address         string
	config          *HTTPTransportConfig
	ready           chan struct{}
	readySignaled   int32 // atomic boolean to track if ready was signaled
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

	// Setup tracing middleware if tracing manager is provided
	if config.TracingManager != nil {
		tracingMiddleware := middleware.TracingMiddleware(config.TracingManager)
		transport.router.Use(tracingMiddleware)
	}

	// Setup client certificate middleware if mTLS is enabled
	if config.TLS != nil && config.TLS.Enabled && config.TLS.ClientAuth != "" && config.TLS.ClientAuth != "none" {
		transport.router.Use(ClientCertificateMiddleware())
		logger.Logger.Info("Client certificate middleware enabled",
			zap.String("client_auth", config.TLS.ClientAuth))
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
	defer h.mu.Unlock()

	// Check if server is already running
	if h.server != nil {
		return nil // Already started
	}

	// Create listener
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to create HTTP listener: %w", err)
	}

	// Update actual address in case of :0 port
	h.address = ln.Addr().String()

	h.server = &http.Server{
		Addr:    h.address,
		Handler: h.router,
	}

	// Configure TLS if enabled
	if h.config.TLS != nil && h.config.TLS.Enabled {
		tlsConfig, err := tlsconfig.NewTLSConfig(h.config.TLS)
		if err != nil {
			return fmt.Errorf("failed to create TLS config: %w", err)
		}
		h.server.TLSConfig = tlsConfig
		logger.Logger.Info("HTTP transport TLS enabled",
			zap.String("address", h.address),
			zap.String("client_auth", h.config.TLS.ClientAuth))
	}

	// Reset ready channel for each start if enabled
	if h.config.EnableReady {
		h.ready = make(chan struct{})
		// Reset ready signaled flag
		atomic.StoreInt32(&h.readySignaled, 0)
	}

	// Capture server reference to avoid race conditions
	server := h.server

	logger.Logger.Info("Starting HTTP transport", zap.String("address", h.address))

	// Start serving in a goroutine
	go func() {
		// Signal that server is ready (only once) if enabled
		if h.config.EnableReady {
			h.mu.RLock()
			ready := h.ready
			h.mu.RUnlock()

			if ready != nil {
				// Use atomic compare-and-swap to ensure only one signal
				if atomic.CompareAndSwapInt32(&h.readySignaled, 0, 1) {
					close(ready)
				}
			}
		}

		var err error
		if h.config.TLS != nil && h.config.TLS.Enabled {
			// Use ServeTLS for TLS-enabled servers
			// The TLS config is already set on the server
			err = server.ServeTLS(ln, "", "")
		} else {
			// Use regular Serve for non-TLS servers
			err = server.Serve(ln)
		}
		if err != nil && err != http.ErrServerClosed {
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

// WaitReady returns a channel that will be closed when the service is ready
func (h *HTTPNetworkService) WaitReady() <-chan struct{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// If ready channel exists and server is running, return it
	if h.ready != nil && h.server != nil {
		return h.ready
	}

	// Return a closed channel if service is not running or ready channel is nil
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

// ClientCertificateMiddleware is a Gin middleware that extracts client certificate
// information from mTLS connections and makes it available in the request context.
// This middleware should be registered when mTLS is enabled.
func ClientCertificateMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if TLS is enabled and client certificates are provided
		if c.Request.TLS != nil && len(c.Request.TLS.PeerCertificates) > 0 {
			// Extract the first client certificate (leaf certificate)
			clientCert := c.Request.TLS.PeerCertificates[0]

			// Extract certificate information
			certInfo := tlsconfig.ExtractCertificateInfo(clientCert)

			// Store certificate information in context for use by handlers
			c.Set("client_cert", clientCert)
			c.Set("client_cert_info", certInfo)
			c.Set("client_cn", certInfo.CommonName)

			// Log client certificate information
			logger.Logger.Debug("Client certificate received",
				zap.String("cn", certInfo.CommonName),
				zap.Strings("organization", certInfo.Organization),
				zap.String("serial", certInfo.SerialNumber))
		}

		c.Next()
	}
}

// GetClientCertificate retrieves the client certificate from the Gin context.
// Returns nil if no client certificate is present.
func GetClientCertificate(c *gin.Context) *x509.Certificate {
	if cert, exists := c.Get("client_cert"); exists {
		if clientCert, ok := cert.(*x509.Certificate); ok {
			return clientCert
		}
	}
	return nil
}

// GetClientCertificateInfo retrieves extracted client certificate information from the Gin context.
// Returns nil if no client certificate information is present.
func GetClientCertificateInfo(c *gin.Context) *tlsconfig.CertificateInfo {
	if info, exists := c.Get("client_cert_info"); exists {
		if certInfo, ok := info.(*tlsconfig.CertificateInfo); ok {
			return certInfo
		}
	}
	return nil
}

// GetClientCommonName retrieves the client certificate's Common Name from the Gin context.
// Returns empty string if no client certificate is present.
func GetClientCommonName(c *gin.Context) string {
	if cn, exists := c.Get("client_cn"); exists {
		if commonName, ok := cn.(string); ok {
			return commonName
		}
	}
	return ""
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
