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

// Package main demonstrates a gRPC service using the base server framework
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"

	interaction "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// GreeterService implements the ServiceRegistrar interface
type GreeterService struct {
	name string
}

// NewGreeterService creates a new greeter service
func NewGreeterService(name string) *GreeterService {
	return &GreeterService{
		name: name,
	}
}

// RegisterServices registers the gRPC services with the server
func (s *GreeterService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register gRPC service
	grpcService := &GreeterGRPCService{serviceName: s.name}
	if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	// Register health check
	healthCheck := &GreeterHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// GreeterGRPCService implements the GRPCService interface
type GreeterGRPCService struct {
	serviceName string
	metricsCollector types.MetricsCollector
	interaction.UnimplementedGreeterServiceServer
}

// RegisterGRPC registers the gRPC service with the server
func (s *GreeterGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return fmt.Errorf("expected *grpc.Server, got %T", server)
	}

	// Register the greeter service
	interaction.RegisterGreeterServiceServer(grpcServer, s)
	return nil
}

// GetServiceName returns the service name
func (s *GreeterGRPCService) GetServiceName() string {
	return s.serviceName
}

// SayHello implements the SayHello RPC method
func (s *GreeterGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
	start := time.Now()
	
	// Validate request
	if req.GetName() == "" {
		// Track validation errors
		if s.metricsCollector != nil {
			s.metricsCollector.IncrementCounter("grpc_errors_total", map[string]string{
				"method": "SayHello",
				"error_type": "validation_error",
				"error_code": "invalid_argument",
			})
		}
		return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
	}
	
	// Track request by name for business metrics
	name := req.GetName()
	if s.metricsCollector != nil {
		s.metricsCollector.IncrementCounter("grpc_hello_requests_total", map[string]string{
			"method": "SayHello",
			"name": name,
		})
	}

	// Create response
	response := &interaction.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}
	
	// Track response time
	if s.metricsCollector != nil {
		duration := time.Since(start).Seconds()
		s.metricsCollector.ObserveHistogram("grpc_request_duration_seconds", duration, map[string]string{
			"method": "SayHello",
			"name": name,
		})
	}

	return response, nil
}

// Note: Only SayHello method is available in the current protobuf definition
// SayGoodbye method would require adding it to the .proto file first

// GreeterHealthCheck implements the HealthCheck interface
type GreeterHealthCheck struct {
	serviceName string
}

// Check performs a health check
func (h *GreeterHealthCheck) Check(ctx context.Context) error {
	// In a real service, check database connections, external services, etc.
	// For this example, we'll always return healthy
	return nil
}

// GetServiceName returns the service name
func (h *GreeterHealthCheck) GetServiceName() string {
	return h.serviceName
}

// GreeterDependencyContainer implements the DependencyContainer interface
type GreeterDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewGreeterDependencyContainer creates a new dependency container
func NewGreeterDependencyContainer() *GreeterDependencyContainer {
	return &GreeterDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

// GetService retrieves a service by name
func (d *GreeterDependencyContainer) GetService(name string) (interface{}, error) {
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
func (d *GreeterDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	// In a real service, close database connections, etc.
	d.closed = true
	return nil
}

// AddService adds a service to the container
func (d *GreeterDependencyContainer) AddService(name string, service interface{}) {
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
		ServiceName: "grpc-greeter-service",
		HTTP: server.HTTPConfig{
			Enabled: false, // Disable HTTP for this gRPC-only service
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
			ServiceName: "grpc-greeter-service",
			Tags:        []string{"grpc", "greeter", "v1"},
		},
		Middleware: server.MiddlewareConfig{
			EnableLogging: true,
		},
		Prometheus: *types.DefaultPrometheusConfig(),
	}
}

// applyEnvironmentOverrides applies environment variable overrides
func applyEnvironmentOverrides(config *server.ServerConfig) {
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
		log.Fatal("Failed to load configuration:", err)
	}
	
	// Fallback to hardcoded config if file loading fails
	if config == nil {
		log.Println("Using default configuration")
		config = createDefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatal("Invalid configuration:", err)
	}

	// Create service and dependencies
	service := NewGreeterService("grpc-greeter-service")
	deps := NewGreeterDependencyContainer()

	// Add some example dependencies
	deps.AddService("config", config)
	deps.AddService("version", "1.0.0")

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

	log.Printf("gRPC Greeter service started successfully")
	log.Printf("gRPC server listening on: %s", baseServer.GetGRPCAddress())
	log.Printf("Try: grpcurl -plaintext %s swit.interaction.v1.GreeterService/SayHello", baseServer.GetGRPCAddress())

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
