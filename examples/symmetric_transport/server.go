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

// Package main demonstrates symmetric transport architecture where services
// can be registered independently with HTTP and gRPC transports, eliminating
// the coupling issue where gRPC services had to register through HTTP transport's registry.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
)

// SymmetricServer demonstrates the new symmetric transport architecture
type SymmetricServer struct {
	transportManager *transport.Manager
	httpTransport    *transport.HTTPTransport
	grpcTransport    *transport.GRPCTransport
}

// NewSymmetricServer creates a server with symmetric transport architecture
func NewSymmetricServer() *SymmetricServer {
	// Create transport manager with integrated service registry management
	manager := transport.NewManager()

	// Create HTTP transport
	httpConfig := &transport.HTTPTransportConfig{
		Address:     ":8080",
		EnableReady: true,
	}
	httpTransport := transport.NewHTTPTransportWithConfig(httpConfig)

	// Create gRPC transport
	grpcConfig := transport.DefaultGRPCConfig()
	grpcConfig.Address = ":9090"
	grpcTransport := transport.NewGRPCTransportWithConfig(grpcConfig)

	// Register transports with manager
	manager.Register(httpTransport)
	manager.Register(grpcTransport)

	return &SymmetricServer{
		transportManager: manager,
		httpTransport:    httpTransport,
		grpcTransport:    grpcTransport,
	}
}

// RegisterHTTPOnlyService registers a service that only supports HTTP
func (s *SymmetricServer) RegisterHTTPOnlyService(handler transport.HandlerRegister) error {
	return s.transportManager.RegisterHTTPHandler(handler)
}

// RegisterGRPCOnlyService registers a service that only supports gRPC
func (s *SymmetricServer) RegisterGRPCOnlyService(handler transport.HandlerRegister) error {
	return s.transportManager.RegisterGRPCHandler(handler)
}

// RegisterDualProtocolService registers a service that supports both HTTP and gRPC
func (s *SymmetricServer) RegisterDualProtocolService(handler transport.HandlerRegister) error {
	// Register with both transports
	if err := s.transportManager.RegisterHTTPHandler(handler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}
	if err := s.transportManager.RegisterGRPCHandler(handler); err != nil {
		return fmt.Errorf("failed to register gRPC handler: %w", err)
	}
	return nil
}

// Start starts the server with symmetric transport initialization
func (s *SymmetricServer) Start(ctx context.Context) error {
	// Initialize all services
	if err := s.transportManager.InitializeAllServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Start all transports
	if err := s.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transports: %w", err)
	}

	// Register HTTP routes with HTTP transport
	httpRouter := s.httpTransport.GetRouter()
	if err := s.transportManager.RegisterAllHTTPRoutes(httpRouter); err != nil {
		return fmt.Errorf("failed to register HTTP routes: %w", err)
	}

	// Register gRPC services with gRPC transport
	grpcServer := s.grpcTransport.GetServer()
	if grpcServer != nil {
		if err := s.transportManager.RegisterAllGRPCServices(grpcServer); err != nil {
			return fmt.Errorf("failed to register gRPC services: %w", err)
		}
	}

	return nil
}

// Stop gracefully stops the server
func (s *SymmetricServer) Stop(ctx context.Context) error {
	// Shutdown all services first
	if err := s.transportManager.ShutdownAllServices(ctx); err != nil {
		log.Printf("Error shutting down services: %v", err)
	}

	// Stop all transports
	if err := s.transportManager.Stop(5 * time.Second); err != nil {
		return fmt.Errorf("failed to stop transports: %w", err)
	}

	return nil
}

// ExampleService demonstrates a service that supports both protocols
type ExampleService struct {
	name    string
	version string
}

// NewExampleService creates a new example service
func NewExampleService(name, version string) *ExampleService {
	return &ExampleService{
		name:    name,
		version: version,
	}
}

// RegisterHTTP registers HTTP routes with the provided router
func (e *ExampleService) RegisterHTTP(router *gin.Engine) error {
	// Use service-specific paths to avoid conflicts
	servicePath := "/" + e.name
	healthPath := "/" + e.name + "/health"

	router.GET(servicePath, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello from HTTP",
			"service": e.name,
			"version": e.version,
		})
	})
	router.GET(healthPath, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": e.name,
		})
	})
	return nil
}

// RegisterGRPC registers gRPC services with the provided server
func (e *ExampleService) RegisterGRPC(server *grpc.Server) error {
	// In a real implementation, you would register actual gRPC services here
	// For this example, we'll just return nil to indicate successful registration
	return nil
}

// GetMetadata returns metadata about the service
func (e *ExampleService) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:           e.name,
		Version:        e.version,
		Description:    "Example service supporting both HTTP and gRPC",
		HealthEndpoint: "/" + e.name + "/health",
		Tags:           []string{"example", "symmetric"},
	}
}

// GetHealthEndpoint returns the health check endpoint for the service
func (e *ExampleService) GetHealthEndpoint() string {
	return "/" + e.name + "/health"
}

// IsHealthy checks if the service is healthy
func (e *ExampleService) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	return &types.HealthStatus{
		Status:    types.HealthStatusHealthy,
		Timestamp: time.Now(),
		Version:   e.version,
		Uptime:    60 * time.Second,
	}, nil
}

// Initialize initializes the service
func (e *ExampleService) Initialize(ctx context.Context) error {
	log.Printf("Initializing %s service v%s", e.name, e.version)
	return nil
}

// Shutdown gracefully shuts down the service
func (e *ExampleService) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down %s service v%s", e.name, e.version)
	return nil
}

func main() {
	// Create symmetric server
	server := NewSymmetricServer()

	// Register services with different transport strategies
	httpOnlyService := NewExampleService("http-api", "1.0.0")
	grpcOnlyService := NewExampleService("grpc-api", "1.0.0")
	dualService := NewExampleService("dual-api", "1.0.0")

	// Register services according to their capabilities
	if err := server.RegisterHTTPOnlyService(httpOnlyService); err != nil {
		log.Fatalf("Failed to register HTTP service: %v", err)
	}

	if err := server.RegisterGRPCOnlyService(grpcOnlyService); err != nil {
		log.Fatalf("Failed to register gRPC service: %v", err)
	}

	if err := server.RegisterDualProtocolService(dualService); err != nil {
		log.Fatalf("Failed to register dual protocol service: %v", err)
	}

	// Start the server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Symmetric server started successfully")
	log.Printf("HTTP server: http://localhost:8080")
	log.Printf("gRPC server: localhost:9090")

	// In a real application, you would set up signal handling here
	// For this example, we'll just wait a bit and then shutdown
	time.Sleep(30 * time.Second)

	log.Println("Shutting down server...")
	if err := server.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("Server shutdown complete")
}
