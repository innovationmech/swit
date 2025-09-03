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

// Package main demonstrates a full-featured service with both HTTP and gRPC using the base server framework
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"

	interaction "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// FullFeaturedService implements the ServiceRegistrar interface
type FullFeaturedService struct {
	name string
}

// NewFullFeaturedService creates a new full-featured service
func NewFullFeaturedService(name string) *FullFeaturedService {
	return &FullFeaturedService{
		name: name,
	}
}

// RegisterServices registers both HTTP and gRPC services with the server
func (s *FullFeaturedService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &FullFeaturedHTTPHandler{
		serviceName: s.name,
		userCount: 1000, // Simulate starting user count
		processingQueueSize: 0,
		cacheHits: 0,
		cacheMisses: 0,
	}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register gRPC service
	grpcService := &FullFeaturedGRPCService{serviceName: s.name}
	if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	// Register health check
	healthCheck := &FullFeaturedHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// FullFeaturedHTTPHandler implements the HTTPHandler interface
type FullFeaturedHTTPHandler struct {
	serviceName string
	metricsCollector types.MetricsCollector
	// Business data structures
	userCount int64
	processingQueueSize int64
	cacheHits, cacheMisses int64
	mu sync.RWMutex
}

// RegisterRoutes registers HTTP routes with the router
func (h *FullFeaturedHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Register API routes
	api := ginRouter.Group("/api/v1")
	{
		// Greeter endpoints (HTTP versions of gRPC methods)
		api.POST("/greet", h.handleGreet)

		// Additional HTTP-only endpoints
		api.GET("/status", h.handleStatus)
		api.GET("/metrics", h.handleMetrics)
		api.POST("/echo", h.handleEcho)
		
		// Business endpoints demonstrating various metrics
		api.POST("/orders", h.handleCreateOrder)
		api.GET("/analytics", h.handleAnalytics)
		api.POST("/process", h.handleProcessJob)
	}

	return nil
}

// GetServiceName returns the service name
func (h *FullFeaturedHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// handleGreet handles the HTTP greet endpoint
func (h *FullFeaturedHTTPHandler) handleGreet(c *gin.Context) {
	start := time.Now()
	
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Track validation errors
		if h.metricsCollector != nil {
			h.metricsCollector.IncrementCounter("http_greet_errors_total", map[string]string{
				"error_type": "validation_error",
				"endpoint": "/api/v1/greet",
				"protocol": "http",
			})
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}
	
	// Simulate user registration (business logic)
	h.mu.Lock()
	h.userCount++
	currentUsers := h.userCount
	h.mu.Unlock()
	
	// Business metrics - track greetings and user count
	name := request.Name
	if h.metricsCollector != nil {
		h.metricsCollector.IncrementCounter("http_greet_requests_total", map[string]string{
			"name": name,
			"endpoint": "/api/v1/greet",
			"protocol": "http",
		})
		
		// Track current user count as gauge
		h.metricsCollector.SetGauge("business_users_total", float64(currentUsers), map[string]string{
			"service": h.serviceName,
		})
		
		// Simulate cache lookup with hit/miss ratio
		cacheHit := rand.Intn(100) < 80 // 80% cache hit rate
		if cacheHit {
			h.mu.Lock()
			h.cacheHits++
			h.mu.Unlock()
			h.metricsCollector.IncrementCounter("cache_operations_total", map[string]string{
				"result": "hit",
				"operation": "user_lookup",
			})
		} else {
			h.mu.Lock()
			h.cacheMisses++
			h.mu.Unlock()
			h.metricsCollector.IncrementCounter("cache_operations_total", map[string]string{
				"result": "miss",
				"operation": "user_lookup",
			})
		}
		
		// Cache hit rate gauge
		h.mu.RLock()
		totalOps := h.cacheHits + h.cacheMisses
		hitRate := float64(h.cacheHits) / float64(totalOps)
		h.mu.RUnlock()
		
		if totalOps > 0 {
			h.metricsCollector.SetGauge("cache_hit_rate", hitRate, map[string]string{
				"service": h.serviceName,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Hello, %s! You are user #%d", request.Name, currentUsers),
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"protocol":  "HTTP",
		"user_id":   currentUsers,
	})
	
	// Track response time
	if h.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		h.metricsCollector.ObserveHistogram("http_request_duration_seconds", duration, map[string]string{
			"endpoint": "/api/v1/greet",
			"protocol": "http",
			"name": name,
		})
	}
}

// Note: handleFarewell removed as SayGoodbye method is not available in current protobuf definition

// handleStatus handles the status endpoint
func (h *FullFeaturedHTTPHandler) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"uptime":    "running",
		"protocols": []string{"HTTP", "gRPC"},
	})
}

// handleMetrics handles the metrics endpoint
func (h *FullFeaturedHTTPHandler) handleMetrics(c *gin.Context) {
	// In a real service, this would return actual metrics
	c.JSON(http.StatusOK, gin.H{
		"service":        h.serviceName,
		"requests_total": 100,
		"errors_total":   5,
		"uptime_seconds": 3600,
		"timestamp":      time.Now().UTC(),
	})
}

// handleEcho handles the echo endpoint
func (h *FullFeaturedHTTPHandler) handleEcho(c *gin.Context) {
	var request struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"echo":      request.Message,
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"protocol":  "HTTP",
	})
}

// handleCreateOrder handles the order creation endpoint (demonstrates business KPI metrics)
func (h *FullFeaturedHTTPHandler) handleCreateOrder(c *gin.Context) {
	start := time.Now()
	
	var request struct {
		CustomerID string  `json:"customer_id" binding:"required"`
		Items      []struct {
			Product  string  `json:"product" binding:"required"`
			Quantity int     `json:"quantity" binding:"required"`
			Price    float64 `json:"price" binding:"required"`
		} `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		if h.metricsCollector != nil {
			h.metricsCollector.IncrementCounter("order_errors_total", map[string]string{
				"error_type": "validation_error",
				"endpoint": "/api/v1/orders",
			})
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}
	
	// Calculate order total
	var orderTotal float64
	itemCount := 0
	for _, item := range request.Items {
		orderTotal += item.Price * float64(item.Quantity)
		itemCount += item.Quantity
	}
	
	// Business metrics
	if h.metricsCollector != nil {
		// Order metrics
		h.metricsCollector.IncrementCounter("orders_created_total", map[string]string{
			"customer_type": "regular", // Could be derived from customer_id
		})
		
		// Revenue metrics (in cents to avoid float issues)
		h.metricsCollector.ObserveHistogram("order_value_dollars", orderTotal, map[string]string{
			"order_type": "online",
		})
		
		// Item metrics
		h.metricsCollector.ObserveHistogram("order_item_count", float64(itemCount), map[string]string{
			"order_type": "online",
		})
		
		// Processing queue simulation
		h.mu.Lock()
		h.processingQueueSize++
		currentQueueSize := h.processingQueueSize
		h.mu.Unlock()
		
		h.metricsCollector.SetGauge("processing_queue_size", float64(currentQueueSize), map[string]string{
			"queue_type": "orders",
		})
	}
	
	// Simulate processing time based on order complexity
	processingTime := time.Duration(50+len(request.Items)*10) * time.Millisecond
	time.Sleep(processingTime)
	
	orderID := fmt.Sprintf("order_%d", time.Now().Unix())
	
	c.JSON(http.StatusCreated, gin.H{
		"order_id":    orderID,
		"customer_id": request.CustomerID,
		"total":       orderTotal,
		"items":       len(request.Items),
		"status":      "created",
		"timestamp":   time.Now().UTC(),
	})
	
	// Track response time
	if h.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		h.metricsCollector.ObserveHistogram("http_request_duration_seconds", duration, map[string]string{
			"endpoint": "/api/v1/orders",
			"protocol": "http",
		})
		
		// Simulate order processing completion
		go func() {
			time.Sleep(2 * time.Second) // Simulate processing
			h.mu.Lock()
			h.processingQueueSize--
			h.mu.Unlock()
			
			if h.metricsCollector != nil {
				h.metricsCollector.SetGauge("processing_queue_size", float64(h.processingQueueSize), map[string]string{
					"queue_type": "orders",
				})
				h.metricsCollector.IncrementCounter("orders_processed_total", map[string]string{
					"status": "completed",
				})
			}
		}()
	}
}

// handleAnalytics handles the analytics endpoint (demonstrates gauge metrics)
func (h *FullFeaturedHTTPHandler) handleAnalytics(c *gin.Context) {
	start := time.Now()
	
	h.mu.RLock()
	currentUsers := h.userCount
	queueSize := h.processingQueueSize
	cacheHits := h.cacheHits
	cacheMisses := h.cacheMisses
	h.mu.RUnlock()
	
	// Update analytics gauges
	if h.metricsCollector != nil {
		h.metricsCollector.SetGauge("active_users_gauge", float64(currentUsers), map[string]string{
			"service": h.serviceName,
		})
		
		h.metricsCollector.SetGauge("system_load_percent", float64(rand.Intn(100)), map[string]string{
			"component": "cpu",
		})
		
		h.metricsCollector.SetGauge("system_load_percent", float64(rand.Intn(80)), map[string]string{
			"component": "memory",
		})
		
		// Multi-dimensional business metrics
		regions := []string{"us-east", "us-west", "eu-west", "asia"}
		for _, region := range regions {
			h.metricsCollector.SetGauge("regional_active_sessions", float64(rand.Intn(500)+100), map[string]string{
				"region": region,
				"service": h.serviceName,
			})
		}
	}
	
	totalCacheOps := cacheHits + cacheMisses
	hitRate := 0.0
	if totalCacheOps > 0 {
		hitRate = float64(cacheHits) / float64(totalCacheOps)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"analytics": map[string]interface{}{
			"total_users":           currentUsers,
			"processing_queue_size": queueSize,
			"cache_stats": map[string]interface{}{
				"hits":     cacheHits,
				"misses":   cacheMisses,
				"hit_rate": hitRate,
			},
			"system_health": map[string]interface{}{
				"cpu_usage":    rand.Intn(100),
				"memory_usage": rand.Intn(80),
				"disk_usage":   rand.Intn(60),
			},
		},
		"timestamp": time.Now().UTC(),
	})
	
	if h.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		h.metricsCollector.ObserveHistogram("http_request_duration_seconds", duration, map[string]string{
			"endpoint": "/api/v1/analytics",
			"protocol": "http",
		})
	}
}

// handleProcessJob handles the job processing endpoint (demonstrates summary metrics)
func (h *FullFeaturedHTTPHandler) handleProcessJob(c *gin.Context) {
	start := time.Now()
	
	var request struct {
		JobType   string                 `json:"job_type" binding:"required"`
		Priority  string                 `json:"priority"`
		Data      map[string]interface{} `json:"data"`
		BatchSize int                    `json:"batch_size"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		if h.metricsCollector != nil {
			h.metricsCollector.IncrementCounter("job_errors_total", map[string]string{
				"error_type": "validation_error",
				"endpoint": "/api/v1/process",
			})
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}
	
	// Default values
	if request.Priority == "" {
		request.Priority = "normal"
	}
	if request.BatchSize == 0 {
		request.BatchSize = 1
	}
	
	// Simulate processing time based on job type and batch size
	baseTime := 100 * time.Millisecond
	switch request.JobType {
	case "image_processing":
		baseTime = 500 * time.Millisecond
	case "data_analysis":
		baseTime = 1000 * time.Millisecond
	case "email_sending":
		baseTime = 200 * time.Millisecond
	}
	
	processingTime := time.Duration(int64(baseTime) * int64(request.BatchSize))
	time.Sleep(processingTime)
	
	// Track job metrics
	if h.metricsCollector != nil {
		h.metricsCollector.IncrementCounter("jobs_processed_total", map[string]string{
			"job_type": request.JobType,
			"priority": request.Priority,
			"status":   "completed",
		})
		
		h.metricsCollector.ObserveHistogram("job_processing_duration_seconds", processingTime.Seconds(), map[string]string{
			"job_type": request.JobType,
			"priority": request.Priority,
		})
		
		h.metricsCollector.ObserveHistogram("job_batch_size", float64(request.BatchSize), map[string]string{
			"job_type": request.JobType,
		})
	}
	
	jobID := fmt.Sprintf("job_%s_%d", request.JobType, time.Now().Unix())
	
	c.JSON(http.StatusOK, gin.H{
		"job_id":       jobID,
		"job_type":     request.JobType,
		"priority":     request.Priority,
		"batch_size":   request.BatchSize,
		"processing_time_ms": processingTime.Milliseconds(),
		"status":       "completed",
		"timestamp":    time.Now().UTC(),
	})
	
	if h.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		h.metricsCollector.ObserveHistogram("http_request_duration_seconds", duration, map[string]string{
			"endpoint": "/api/v1/process",
			"protocol": "http",
		})
	}
}

// FullFeaturedGRPCService implements the GRPCService interface
type FullFeaturedGRPCService struct {
	serviceName string
	interaction.UnimplementedGreeterServiceServer
}

// RegisterGRPC registers the gRPC service with the server
func (s *FullFeaturedGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", server)
	}

	// Register the greeter service
	interaction.RegisterGreeterServiceServer(grpcServer, s)
	return nil
}

// GetServiceName returns the service name
func (s *FullFeaturedGRPCService) GetServiceName() string {
	return s.serviceName
}

// SayHello implements the SayHello RPC method
func (s *FullFeaturedGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
	// Validate request
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
	}

	// Create response
	response := &interaction.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}

	return response, nil
}

// Note: Only SayHello method is available in the current protobuf definition
// SayGoodbye method would require adding it to the .proto file first

// FullFeaturedHealthCheck implements the HealthCheck interface
type FullFeaturedHealthCheck struct {
	serviceName string
}

// Check performs a health check
func (h *FullFeaturedHealthCheck) Check(ctx context.Context) error {
	// In a real service, check database connections, external services, etc.
	// For this example, we'll always return healthy
	return nil
}

// GetServiceName returns the service name
func (h *FullFeaturedHealthCheck) GetServiceName() string {
	return h.serviceName
}

// FullFeaturedDependencyContainer implements the DependencyContainer interface
type FullFeaturedDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewFullFeaturedDependencyContainer creates a new dependency container
func NewFullFeaturedDependencyContainer() *FullFeaturedDependencyContainer {
	return &FullFeaturedDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

// GetService retrieves a service by name
func (d *FullFeaturedDependencyContainer) GetService(name string) (interface{}, error) {
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
func (d *FullFeaturedDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	// In a real service, close database connections, etc.
	d.closed = true
	return nil
}

// AddService adds a service to the container
func (d *FullFeaturedDependencyContainer) AddService(name string, service interface{}) {
	d.services[name] = service
}
func main() {
	// Initialize logger
	logger.InitLogger()

	// Create configuration
	config := &server.ServerConfig{
		ServiceName: "full-featured-service",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Port:                getEnv("GRPC_PORT", "9090"),
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: server.GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: server.GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery: server.DiscoveryConfig{
			Enabled:     getBoolEnv("DISCOVERY_ENABLED", false),
			Address:     getEnv("CONSUL_ADDRESS", "localhost:8500"),
			ServiceName: "full-featured-service",
			Tags:        []string{"http", "grpc", "api", "v1"},
		},
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatal("Invalid configuration:", err)
	}

	// Create service and dependencies
	service := NewFullFeaturedService("full-featured-service")
	deps := NewFullFeaturedDependencyContainer()

	// Add some example dependencies
	deps.AddService("config", config)
	deps.AddService("version", "1.0.0")
	deps.AddService("environment", getEnv("ENVIRONMENT", "development"))

	// Create base server
	baseServer, err := server.NewBusinessServerCore(config, service, deps)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(ctx); err != nil {
		log.Fatal("Failed to start server:", err)
	}

	log.Printf("Full-featured service started successfully")
	log.Printf("HTTP server listening on: %s", baseServer.GetHTTPAddress())
	log.Printf("gRPC server listening on: %s", baseServer.GetGRPCAddress())
	log.Printf("Try HTTP: curl -X POST http://%s/api/v1/greet -H 'Content-Type: application/json' -d '{\"name\":\"Alice\"}'", baseServer.GetHTTPAddress())
	log.Printf("Try gRPC: grpcurl -plaintext -d '{\"name\":\"Bob\"}' %s swit.interaction.v1.GreeterService/SayHello", baseServer.GetGRPCAddress())

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received, stopping server...")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	} else {
		log.Println("Server stopped gracefully")
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
