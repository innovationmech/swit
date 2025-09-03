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

// Package main demonstrates a simple HTTP service using the base server framework
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// SimpleHTTPService implements the ServiceRegistrar interface
type SimpleHTTPService struct {
	name string
}

// NewSimpleHTTPService creates a new simple HTTP service
func NewSimpleHTTPService(name string) *SimpleHTTPService {
	return &SimpleHTTPService{
		name: name,
	}
}

// RegisterServices registers the HTTP handlers with the server
func (s *SimpleHTTPService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &SimpleHTTPHandler{
		serviceName: s.name,
		// Note: metrics collector would be injected in a real application
	}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register health check
	healthCheck := &SimpleHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// SimpleHTTPHandler implements the HTTPHandler interface
type SimpleHTTPHandler struct {
	serviceName string
	metricsCollector types.MetricsCollector
}

// RegisterRoutes registers HTTP routes with the router
func (h *SimpleHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Register API routes
	api := ginRouter.Group("/api/v1")
	{
		api.GET("/hello", h.handleHello)
		api.GET("/status", h.handleStatus)
		api.POST("/echo", h.handleEcho)
	}

	return nil
}

// GetServiceName returns the service name
func (h *SimpleHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// handleHello handles the hello endpoint
func (h *SimpleHTTPHandler) handleHello(c *gin.Context) {
	start := time.Now()
	
	// Business metrics - track hello requests by name
	name := c.Query("name")
	if name == "" {
		name = "World"
	}
	
	// Track request count and duration
	if h.metricsCollector != nil {
		h.metricsCollector.IncrementCounter("hello_requests_total", map[string]string{
			"name": name,
			"endpoint": "/api/v1/hello",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Hello, %s!", name),
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	})
	
	// Track response time
	if h.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		h.metricsCollector.ObserveHistogram("hello_request_duration_seconds", duration, map[string]string{
			"name": name,
			"endpoint": "/api/v1/hello",
		})
	}
}

// handleStatus handles the status endpoint
func (h *SimpleHTTPHandler) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"uptime":    "running", // In a real service, calculate actual uptime
	})
}

// handleEcho handles the echo endpoint
func (h *SimpleHTTPHandler) handleEcho(c *gin.Context) {
	start := time.Now()
	
	var request struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Track errors
		if h.metricsCollector != nil {
			h.metricsCollector.IncrementCounter("echo_errors_total", map[string]string{
				"error_type": "validation_error",
				"endpoint": "/api/v1/echo",
			})
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}
	
	// Track message length for business metrics
	messageLength := float64(len(request.Message))
	if h.metricsCollector != nil {
		h.metricsCollector.IncrementCounter("echo_requests_total", map[string]string{
			"endpoint": "/api/v1/echo",
		})
		h.metricsCollector.ObserveHistogram("echo_message_length_bytes", messageLength, map[string]string{
			"endpoint": "/api/v1/echo",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"echo":      request.Message,
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	})
	
	// Track response time
	if h.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		h.metricsCollector.ObserveHistogram("echo_request_duration_seconds", duration, map[string]string{
			"endpoint": "/api/v1/echo",
		})
	}
}

// SimpleHealthCheck implements the HealthCheck interface
type SimpleHealthCheck struct {
	serviceName string
}

// Check performs a health check
func (h *SimpleHealthCheck) Check(ctx context.Context) error {
	// In a real service, check database connections, external services, etc.
	// For this example, we'll always return healthy
	return nil
}

// GetServiceName returns the service name
func (h *SimpleHealthCheck) GetServiceName() string {
	return h.serviceName
}

// SimpleDependencyContainer implements the DependencyContainer interface
type SimpleDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewSimpleDependencyContainer creates a new dependency container
func NewSimpleDependencyContainer() *SimpleDependencyContainer {
	return &SimpleDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

// GetService retrieves a service by name
func (d *SimpleDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// Close closes the dependency container and cleans up resources
func (d *SimpleDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	// In a real service, close database connections, etc.
	d.closed = true
	return nil
}

// AddService adds a service to the container
func (d *SimpleDependencyContainer) AddService(name string, service interface{}) {
	d.services[name] = service
}
// loadConfigFromFile loads configuration from YAML file
func loadConfigFromFile(configPath string) (*server.ServerConfig, error) {
	config := &server.ServerConfig{}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, return default config
		return createDefaultConfig(), nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Apply environment variable overrides
	applyEnvironmentOverrides(config)
	
	return config, nil
}

// createDefaultConfig creates default configuration
func createDefaultConfig() *server.ServerConfig {
	return &server.ServerConfig{
		ServiceName: "simple-http-service",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Enabled: false, // Disable gRPC for this simple HTTP service
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery: server.DiscoveryConfig{
			Enabled:     getBoolEnv("DISCOVERY_ENABLED", false),
			Address:     getEnv("CONSUL_ADDRESS", "localhost:8500"),
			ServiceName: "simple-http-service",
			Tags:        []string{"http", "api", "v1"},
		},
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
		Prometheus: *types.DefaultPrometheusConfig(),
	}
}

// applyEnvironmentOverrides applies environment variable overrides
func applyEnvironmentOverrides(config *server.ServerConfig) {
	if httpPort := os.Getenv("HTTP_PORT"); httpPort != "" {
		config.HTTP.Port = httpPort
	}
	if grpcPort := os.Getenv("GRPC_PORT"); grpcPort != "" {
		config.GRPC.Port = grpcPort
	}
	if discoveryEnabled := os.Getenv("DISCOVERY_ENABLED"); discoveryEnabled != "" {
		config.Discovery.Enabled = discoveryEnabled == "true" || discoveryEnabled == "1"
	}
	if consulAddr := os.Getenv("CONSUL_ADDRESS"); consulAddr != "" {
		config.Discovery.Address = consulAddr
	}
	if prometheusEnabled := os.Getenv("PROMETHEUS_ENABLED"); prometheusEnabled != "" {
		config.Prometheus.Enabled = prometheusEnabled == "true" || prometheusEnabled == "1"
	}
}

func main() {
	// Initialize logger
	logger.InitLogger()

	// Load configuration from file or use defaults
	configPath := getEnv("CONFIG_PATH", "swit.yaml")
	if !filepath.IsAbs(configPath) {
		// Make path relative to executable directory
		if execDir, err := os.Executable(); err == nil {
			configPath = filepath.Join(filepath.Dir(execDir), configPath)
		}
	}
	
	config, err := loadConfigFromFile(configPath)
	if err != nil {
		logger.GetLogger().Fatal("Failed to load configuration", zap.Error(err))
	}
	
	// Fallback to hardcoded config if file loading fails
	if config == nil {
		logger.GetLogger().Warn("Using default configuration")
		config = createDefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.GetLogger().Fatal("Invalid configuration",
			zap.Error(err))
	}

	// Create service and dependencies
	service := NewSimpleHTTPService("simple-http-service")
	deps := NewSimpleDependencyContainer()

	// Add some example dependencies
	deps.AddService("config", config)
	deps.AddService("version", "1.0.0")

	// Create base server
	baseServer, err := server.NewBusinessServerCore(config, service, deps)
	if err != nil {
		logger.GetLogger().Fatal("Failed to create server",
			zap.Error(err))
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(ctx); err != nil {
		logger.GetLogger().Fatal("Failed to start server",
			zap.Error(err))
	}

	logger.GetLogger().Info("Simple HTTP service started successfully",
		zap.String("http_address", baseServer.GetHTTPAddress()),
		zap.String("service_name", "simple-http-service"))
	logger.GetLogger().Info("Example endpoint",
		zap.String("url", fmt.Sprintf("http://%s/api/v1/hello?name=YourName", baseServer.GetHTTPAddress())))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.GetLogger().Info("Shutdown signal received, stopping server")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		logger.GetLogger().Error("Error during shutdown",
			zap.Error(err))
	} else {
		logger.GetLogger().Info("Server stopped gracefully")
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable with a default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
