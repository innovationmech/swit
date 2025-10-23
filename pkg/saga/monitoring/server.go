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

// Package monitoring provides web monitoring dashboard capabilities for Saga execution.
// It includes a web server framework with routing, middleware support, and health check endpoints
// for monitoring and managing Saga instances in real-time.
package monitoring

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ServerConfig contains configuration for the monitoring web server.
type ServerConfig struct {
	// Address is the server listening address (e.g., ":8080", "0.0.0.0:9090").
	Address string

	// Port is the server listening port. If both Address and Port are specified,
	// Port takes precedence.
	Port string

	// EnableTLS enables HTTPS support.
	EnableTLS bool

	// TLSConfig contains TLS configuration when EnableTLS is true.
	TLSConfig *TLSConfig

	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the next request.
	IdleTimeout time.Duration

	// MaxHeaderBytes controls the maximum number of bytes the server will read parsing the request header.
	MaxHeaderBytes int

	// GinMode sets the Gin framework mode ("debug", "release", "test").
	// Default is "release".
	GinMode string

	// HealthCheckPath is the URL path for the health check endpoint.
	// Default is "/api/health".
	HealthCheckPath string

	// HealthManager is the optional health manager for advanced health checks.
	// If not provided, a basic health check will be used.
	HealthManager *SagaHealthManager

	// EnableGracefulShutdown enables graceful shutdown with configurable timeout.
	EnableGracefulShutdown bool

	// GracefulShutdownTimeout is the maximum time to wait for graceful shutdown.
	// Default is 30 seconds.
	GracefulShutdownTimeout time.Duration
}

// TLSConfig contains TLS configuration for HTTPS support.
type TLSConfig struct {
	// CertFile is the path to the TLS certificate file.
	CertFile string

	// KeyFile is the path to the TLS private key file.
	KeyFile string

	// MinVersion is the minimum TLS version to accept (e.g., tls.VersionTLS12).
	MinVersion uint16

	// MaxVersion is the maximum TLS version to accept.
	MaxVersion uint16

	// ClientAuth determines the server's policy for TLS client authentication.
	ClientAuth tls.ClientAuthType

	// ClientCAs is the set of root certificate authorities that servers use
	// to verify client certificates.
	ClientCAs []string
}

// DefaultServerConfig returns a default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:                 ":8090",
		Port:                    "8090",
		EnableTLS:               false,
		ReadTimeout:             30 * time.Second,
		WriteTimeout:            30 * time.Second,
		IdleTimeout:             60 * time.Second,
		MaxHeaderBytes:          1 << 20, // 1 MB
		GinMode:                 gin.ReleaseMode,
		HealthCheckPath:         "/api/health",
		EnableGracefulShutdown:  true,
		GracefulShutdownTimeout: 30 * time.Second,
	}
}

// Validate validates the server configuration.
func (c *ServerConfig) Validate() error {
	if c.Address == "" && c.Port == "" {
		return fmt.Errorf("either Address or Port must be specified")
	}

	if c.EnableTLS {
		if c.TLSConfig == nil {
			return fmt.Errorf("TLS is enabled but TLSConfig is nil")
		}
		if c.TLSConfig.CertFile == "" || c.TLSConfig.KeyFile == "" {
			return fmt.Errorf("TLS certificate and key files must be specified")
		}
	}

	if c.ReadTimeout < 0 || c.WriteTimeout < 0 || c.IdleTimeout < 0 {
		return fmt.Errorf("timeouts cannot be negative")
	}

	if c.GracefulShutdownTimeout < 0 {
		return fmt.Errorf("graceful shutdown timeout cannot be negative")
	}

	return nil
}

// GetAddress returns the server address to bind to.
// If Port is specified, it takes precedence over Address.
func (c *ServerConfig) GetAddress() string {
	if c.Port != "" {
		return ":" + c.Port
	}
	return c.Address
}

// Server is the web server for Saga monitoring dashboard.
type Server struct {
	config       *ServerConfig
	router       *gin.Engine
	server       *http.Server
	middleware   *Middleware
	routeManager *RouteManager

	mu            sync.RWMutex
	running       atomic.Bool
	actualAddress string // Actual listening address after server start
}

// NewMonitoringServer creates a new monitoring server with the given configuration.
//
// Parameters:
//   - config: Server configuration. If nil, default configuration is used.
//
// Returns:
//   - A configured Server ready to start.
//   - An error if the configuration is invalid.
//
// Example:
//
//	config := monitoring.DefaultServerConfig()
//	config.Port = "8090"
//	server, err := monitoring.NewMonitoringServer(config)
//	if err != nil {
//	    return err
//	}
//	if err := server.Start(context.Background()); err != nil {
//	    return err
//	}
func NewMonitoringServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	// Set Gin mode (must be done before gin.New())
	if config.GinMode != "" {
		gin.SetMode(config.GinMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router - each call to gin.New() creates an independent router instance
	router := gin.New()

	// Create monitoring middleware
	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableLogging:     true,
		EnableCORS:        true,
		EnableErrorHandle: true,
		EnableRecovery:    true,
	})

	// Create route manager
	routeManager := NewRouteManager(router, config)

	server := &Server{
		config:       config,
		router:       router,
		middleware:   middleware,
		routeManager: routeManager,
	}

	// Apply middleware
	if err := server.applyMiddleware(); err != nil {
		return nil, fmt.Errorf("failed to apply middleware: %w", err)
	}

	// Setup routes
	if err := server.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return server, nil
}

// applyMiddleware applies all configured middleware to the router.
func (s *Server) applyMiddleware() error {
	// Apply middleware in the correct order
	s.middleware.ApplyToRouter(s.router)
	return nil
}

// setupRoutes initializes all routes for the monitoring server.
func (s *Server) setupRoutes() error {
	return s.routeManager.SetupRoutes()
}

// Start starts the monitoring server.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//
// Returns:
//   - An error if the server fails to start.
//
// Example:
//
//	if err := server.Start(context.Background()); err != nil {
//	    log.Fatalf("Failed to start server: %v", err)
//	}
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if server is already running
	if s.running.Load() {
		return fmt.Errorf("server is already running")
	}

	// Create listener
	address := s.config.GetAddress()
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to create listener on %s: %w", address, err)
	}

	// Store actual listening address
	s.actualAddress = ln.Addr().String()

	// Create HTTP server
	s.server = &http.Server{
		Addr:           s.actualAddress,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	// Configure TLS if enabled
	if s.config.EnableTLS && s.config.TLSConfig != nil {
		tlsConfig := &tls.Config{
			MinVersion: s.config.TLSConfig.MinVersion,
			MaxVersion: s.config.TLSConfig.MaxVersion,
			ClientAuth: s.config.TLSConfig.ClientAuth,
		}
		s.server.TLSConfig = tlsConfig
	}

	if logger.Logger != nil {
		logger.Logger.Info("Starting Saga monitoring server",
			zap.String("address", s.actualAddress),
			zap.Bool("tls", s.config.EnableTLS),
			zap.String("health_check_path", s.config.HealthCheckPath))
	}

	// Channel to communicate startup result
	startupResult := make(chan error, 1)

	// Start server in a goroutine with better startup error handling
	go func() {
		defer func() {
			// If this goroutine exits due to an error or panic, ensure the running flag is cleared
			if r := recover(); r != nil {
				if logger.Logger != nil {
					logger.Logger.Error("Monitoring server panic during startup", zap.Any("panic", r))
				}
				// Clear running flag and send panic error
				s.running.Store(false)
				select {
				case startupResult <- fmt.Errorf("server panic: %v", r):
				default:
				}
			}
		}()

		var err error
		if s.config.EnableTLS {
			err = s.server.ServeTLS(ln, s.config.TLSConfig.CertFile, s.config.TLSConfig.KeyFile)
		} else {
			err = s.server.Serve(ln)
		}

		if err != nil && err != http.ErrServerClosed {
			if logger.Logger != nil {
				logger.Logger.Error("Monitoring server failed to start", zap.Error(err))
			}
			// Clear running flag and send startup error
			s.running.Store(false)
			select {
			case startupResult <- err:
			default:
				// Channel is full or closed, ignore
			}
		}
	}()

	// Wait for startup errors (like TLS certificate issues, port binding problems)
	// These typically happen immediately when Serve() is called, but use a longer timeout
	// for CI environments and slower systems
	select {
	case err := <-startupResult:
		// Immediate startup failure occurred
		return fmt.Errorf("server startup failed: %w", err)
	case <-time.After(1 * time.Second):
		// No error received within timeout, assume server started successfully
		// Mark server as running only after we're reasonably confident it started
		s.running.Store(true)
		if logger.Logger != nil {
			logger.Logger.Info("Saga monitoring server started successfully",
				zap.String("address", s.actualAddress))
		}
		return nil
	}
}

// Stop stops the monitoring server gracefully.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//
// Returns:
//   - An error if the server fails to stop gracefully.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := server.Stop(ctx); err != nil {
//	    log.Printf("Error during shutdown: %v", err)
//	}
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if server is running
	if !s.running.Load() {
		return nil // Already stopped
	}

	if logger.Logger != nil {
		logger.Logger.Info("Stopping Saga monitoring server",
			zap.String("address", s.actualAddress))
	}

	if s.server == nil {
		s.running.Store(false)
		return nil
	}

	var err error
	if s.config.EnableGracefulShutdown {
		// Create context with timeout for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, s.config.GracefulShutdownTimeout)
		defer cancel()

		err = s.server.Shutdown(shutdownCtx)
	} else {
		// Force close immediately
		err = s.server.Close()
	}

	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Error stopping monitoring server", zap.Error(err))
		}
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Mark server as stopped
	s.running.Store(false)

	if logger.Logger != nil {
		logger.Logger.Info("Saga monitoring server stopped successfully")
	}

	return nil
}

// IsRunning returns true if the server is currently running.
func (s *Server) IsRunning() bool {
	return s.running.Load()
}

// GetAddress returns the actual listening address after the server has started.
// Returns empty string if the server hasn't started yet.
func (s *Server) GetAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.actualAddress
}

// GetRouter returns the Gin router for advanced configuration.
// This allows users to register custom routes and middleware.
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// GetConfig returns the server configuration.
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// RegisterCustomRoute allows registering custom routes on the server.
// This should be called before Start().
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, DELETE, etc.)
//   - path: Route path (e.g., "/api/custom")
//   - handlers: Gin handler functions
//
// Example:
//
//	server.RegisterCustomRoute("GET", "/api/custom", func(c *gin.Context) {
//	    c.JSON(200, gin.H{"message": "Custom endpoint"})
//	})
func (s *Server) RegisterCustomRoute(method, path string, handlers ...gin.HandlerFunc) {
	s.router.Handle(method, path, handlers...)
}

// SetQueryAPI sets the Saga query API handler and registers its routes.
// This can be called before or after server creation, but must be called before Start()
// to ensure the routes are available.
func (s *Server) SetQueryAPI(queryAPI *SagaQueryAPI) {
	s.routeManager.SetQueryAPI(queryAPI)
}

// SetControlAPI sets the Saga control API handler and registers its routes.
// This can be called before or after server creation, but must be called before Start()
// to ensure the routes are available.
func (s *Server) SetControlAPI(controlAPI *SagaControlAPI) {
	s.routeManager.SetControlAPI(controlAPI)
}

// SetMetricsAPI sets the metrics API handler and registers its routes.
// This can be called before or after server creation, but must be called before Start()
// to ensure the routes are available.
func (s *Server) SetMetricsAPI(metricsAPI *MetricsAPI) {
	s.routeManager.SetMetricsAPI(metricsAPI)
}

// SetVisualizationAPI sets the Saga visualization API handler and registers its routes.
// This can be called before or after server creation, but must be called before Start()
// to ensure the routes are available.
func (s *Server) SetVisualizationAPI(visualizationAPI *SagaVisualizationAPI) {
	s.routeManager.SetVisualizationAPI(visualizationAPI)
}

// SetAlertsAPI sets the alerts API handler and registers its routes.
// This can be called before or after server creation, but must be called before Start()
// to ensure the routes are available.
func (s *Server) SetAlertsAPI(alertsAPI *AlertsAPI) {
	s.routeManager.SetAlertsAPI(alertsAPI)
}

// SetRealtimePusher sets the realtime pusher for SSE streaming.
// This can be called before or after server creation, but must be called before Start()
// to ensure the SSE endpoint is available.
func (s *Server) SetRealtimePusher(pusher *RealtimePusher) {
	s.routeManager.SetRealtimePusher(pusher)
}
