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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	interaction "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
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
	httpHandler := &FullFeaturedHTTPHandler{serviceName: s.name}
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
	}

	return nil
}

// GetServiceName returns the service name
func (h *FullFeaturedHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// handleGreet handles the HTTP greet endpoint
func (h *FullFeaturedHTTPHandler) handleGreet(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Hello, %s!", request.Name),
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"protocol":  "HTTP",
	})
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
