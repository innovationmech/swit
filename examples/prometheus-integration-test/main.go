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

// Package main demonstrates Prometheus metrics integration with the framework
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// PrometheusTestService demonstrates Prometheus metrics integration
type PrometheusTestService struct {
	name string
}

// NewPrometheusTestService creates a new test service
func NewPrometheusTestService(name string) *PrometheusTestService {
	return &PrometheusTestService{
		name: name,
	}
}

// RegisterServices registers the HTTP handlers with metrics
func (s *PrometheusTestService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &PrometheusTestHTTPHandler{serviceName: s.name}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register health check
	healthCheck := &PrometheusTestHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// PrometheusTestHTTPHandler implements the HTTPHandler interface
type PrometheusTestHTTPHandler struct {
	serviceName string
}

// RegisterRoutes registers HTTP routes with the router
func (h *PrometheusTestHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Register API routes for testing metrics
	api := ginRouter.Group("/api/v1")
	{
		api.GET("/test", h.handleTest)
		api.POST("/slow", h.handleSlow)
		api.GET("/error", h.handleError)
		api.GET("/users/:id", h.handleUserByID)
	}

	return nil
}

// GetServiceName returns the service name
func (h *PrometheusTestHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// handleTest handles the test endpoint
func (h *PrometheusTestHTTPHandler) handleTest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "Test endpoint for Prometheus metrics",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	})
}

// handleSlow handles a slow endpoint to test duration metrics
func (h *PrometheusTestHTTPHandler) handleSlow(c *gin.Context) {
	// Simulate slow processing
	time.Sleep(500 * time.Millisecond)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Slow endpoint completed",
		"service":   h.serviceName,
		"duration":  "500ms",
		"timestamp": time.Now().UTC(),
	})
}

// handleError handles an endpoint that returns errors
func (h *PrometheusTestHTTPHandler) handleError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":     "Intentional error for testing",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	})
}

// handleUserByID handles parameterized path for testing path normalization
func (h *PrometheusTestHTTPHandler) handleUserByID(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"user_id":   userID,
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	})
}

// PrometheusTestHealthCheck implements the HealthCheck interface
type PrometheusTestHealthCheck struct {
	serviceName string
}

// Check performs a health check
func (h *PrometheusTestHealthCheck) Check(ctx context.Context) error {
	return nil
}

// GetServiceName returns the service name
func (h *PrometheusTestHealthCheck) GetServiceName() string {
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

// Close closes the dependency container
func (d *SimpleDependencyContainer) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	return nil
}

// AddService adds a service to the container
func (d *SimpleDependencyContainer) AddService(name string, service interface{}) {
	d.services[name] = service
}

// MetricsCollectorAdapter adapts server.PrometheusMetricsCollector to types.MetricsCollector
type MetricsCollectorAdapter struct {
	collector *server.PrometheusMetricsCollector
}

// NewMetricsCollectorAdapter creates a new adapter that bridges server.PrometheusMetricsCollector to types.MetricsCollector interface
func NewMetricsCollectorAdapter(collector *server.PrometheusMetricsCollector) *MetricsCollectorAdapter {
	return &MetricsCollectorAdapter{collector: collector}
}

// IncrementCounter increments a counter metric by 1
func (a *MetricsCollectorAdapter) IncrementCounter(name string, labels map[string]string) {
	a.collector.IncrementCounter(name, labels)
}

// AddToCounter adds a value to a counter metric
func (a *MetricsCollectorAdapter) AddToCounter(name string, value float64, labels map[string]string) {
	a.collector.AddToCounter(name, value, labels)
}

// SetGauge sets a gauge metric to a specific value
func (a *MetricsCollectorAdapter) SetGauge(name string, value float64, labels map[string]string) {
	a.collector.SetGauge(name, value, labels)
}

// IncrementGauge increments a gauge metric by 1
func (a *MetricsCollectorAdapter) IncrementGauge(name string, labels map[string]string) {
	a.collector.IncrementGauge(name, labels)
}

// DecrementGauge decrements a gauge metric by 1
func (a *MetricsCollectorAdapter) DecrementGauge(name string, labels map[string]string) {
	a.collector.DecrementGauge(name, labels)
}

// ObserveHistogram adds an observation to a histogram metric
func (a *MetricsCollectorAdapter) ObserveHistogram(name string, value float64, labels map[string]string) {
	a.collector.ObserveHistogram(name, value, labels)
}

// GetMetrics returns all collected metrics, converting from server.Metric to types.Metric format
func (a *MetricsCollectorAdapter) GetMetrics() []types.Metric {
	serverMetrics := a.collector.GetMetrics()
	typesMetrics := make([]types.Metric, len(serverMetrics))

	for i, sm := range serverMetrics {
		// Convert value to float64
		var value float64
		switch v := sm.Value.(type) {
		case int64:
			value = float64(v)
		case float64:
			value = v
		case map[string]interface{}:
			// For histogram/summary, use the sum if available
			if sum, ok := v["sum"].(float64); ok {
				value = sum
			}
		default:
			value = 0
		}

		typesMetrics[i] = types.Metric{
			Name:        sm.Name,
			Type:        types.MetricType(sm.Type),
			Value:       value,
			Labels:      sm.Labels,
			Timestamp:   sm.Timestamp,
			Description: sm.Description,
		}
	}

	return typesMetrics
}

// GetMetric returns a specific metric by name, converting from server.Metric to types.Metric format
func (a *MetricsCollectorAdapter) GetMetric(name string) (*types.Metric, bool) {
	serverMetric, found := a.collector.GetMetric(name)
	if !found {
		return nil, false
	}

	// Convert value to float64
	var value float64
	switch v := serverMetric.Value.(type) {
	case int64:
		value = float64(v)
	case float64:
		value = v
	case map[string]interface{}:
		// For histogram/summary, use the sum if available
		if sum, ok := v["sum"].(float64); ok {
			value = sum
		}
	default:
		value = 0
	}

	typesMetric := &types.Metric{
		Name:        serverMetric.Name,
		Type:        types.MetricType(serverMetric.Type),
		Value:       value,
		Labels:      serverMetric.Labels,
		Timestamp:   serverMetric.Timestamp,
		Description: serverMetric.Description,
	}

	return typesMetric, true
}

// Reset clears all metrics from the underlying collector
func (a *MetricsCollectorAdapter) Reset() {
	a.collector.Reset()
}

func main() {
	// Initialize logger
	logger.InitLogger()

	// Create a Prometheus metrics collector for demonstration
	prometheusConfig := server.DefaultPrometheusConfig()
	prometheusConfig.Namespace = "swit_test"
	prometheusConfig.Subsystem = "prometheus_integration"

	prometheusCollector := server.NewPrometheusMetricsCollector(prometheusConfig)

	// Create an adapter to make it compatible with types.MetricsCollector
	metricsAdapter := NewMetricsCollectorAdapter(prometheusCollector)

	// Create HTTP middleware for Prometheus
	httpMiddlewareConfig := middleware.DefaultPrometheusHTTPConfig()
	httpMiddlewareConfig.ServiceName = "prometheus-test-service"
	httpMiddleware := middleware.NewPrometheusHTTPMiddleware(metricsAdapter, httpMiddlewareConfig)

	// Create server configuration
	config := &server.ServerConfig{
		ServiceName: "prometheus-test-service",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Enabled: false,
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery: server.DiscoveryConfig{
			Enabled: false,
		},
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.GetLogger().Fatal("Invalid configuration",
			zap.Error(err))
	}

	// Create service and dependencies
	service := NewPrometheusTestService("prometheus-test-service")
	deps := NewSimpleDependencyContainer()

	// Add dependencies
	deps.AddService("config", config)
	deps.AddService("version", "1.0.0-prometheus-test")

	// Create base server
	baseServer, err := server.NewBusinessServerCore(config, service, deps)
	if err != nil {
		logger.GetLogger().Fatal("Failed to create server",
			zap.Error(err))
	}

	// Note: In a production setup, you would integrate the middleware through
	// the server's middleware manager. For this demo, we'll show how the
	// middleware works conceptually.

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(ctx); err != nil {
		logger.GetLogger().Fatal("Failed to start server",
			zap.Error(err))
	}

	httpAddr := baseServer.GetHTTPAddress()
	logger.GetLogger().Info("Prometheus test service started successfully",
		zap.String("http_address", httpAddr),
		zap.String("service_name", "prometheus-test-service"))

	// Log example endpoints
	logger.GetLogger().Info("Test endpoints available:",
		zap.String("test", fmt.Sprintf("http://%s/api/v1/test", httpAddr)),
		zap.String("slow", fmt.Sprintf("http://%s/api/v1/slow", httpAddr)),
		zap.String("error", fmt.Sprintf("http://%s/api/v1/error", httpAddr)),
		zap.String("user", fmt.Sprintf("http://%s/api/v1/users/123", httpAddr)),
		zap.String("metrics", fmt.Sprintf("http://%s/debug/metrics", httpAddr)),
		zap.String("health", fmt.Sprintf("http://%s/health", httpAddr)))

	logger.GetLogger().Info("Prometheus middleware created successfully",
		zap.String("middleware_config", fmt.Sprintf("%+v", httpMiddlewareConfig)),
		zap.Int("path_cardinality", httpMiddleware.GetPathCardinality()))

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
