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

// Package main demonstrates an order service using the swit framework with OpenTelemetry tracing
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/config"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/handler"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/repository"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/service"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/tracing"
)

// OrderService implements the ServiceRegistrar interface
type OrderService struct {
	name string
}

// NewOrderService creates a new order service
func NewOrderService(name string) *OrderService {
	return &OrderService{
		name: name,
	}
}

// RegisterServices registers HTTP and gRPC services with the server
func (s *OrderService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Get dependencies from container
	deps, err := registry.GetDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to get dependency container: %w", err)
	}

	// Get repository
	repoService, err := deps.GetService("repository")
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}
	repo := repoService.(*repository.OrderRepository)

	// Get business service
	businessService, err := deps.GetService("business_service")
	if err != nil {
		return fmt.Errorf("failed to get business service: %w", err)
	}
	orderService := businessService.(*service.OrderService)

	// Register HTTP handler
	httpHandler := handler.NewOrderHTTPHandler(orderService)
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register gRPC service
	grpcService := handler.NewOrderGRPCService(orderService)
	if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	// Register health check
	healthCheck := &OrderHealthCheck{serviceName: s.name, repository: repo}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// OrderHealthCheck implements the HealthCheck interface
type OrderHealthCheck struct {
	serviceName string
	repository  *repository.OrderRepository
}

// Check performs a health check
func (h *OrderHealthCheck) Check(ctx context.Context) error {
	// Check database connectivity
	if err := h.repository.HealthCheck(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// GetServiceName returns the service name
func (h *OrderHealthCheck) GetServiceName() string {
	return h.serviceName
}

// OrderDependencyContainer implements the DependencyContainer interface
type OrderDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewOrderDependencyContainer creates a new dependency container
func NewOrderDependencyContainer() *OrderDependencyContainer {
	return &OrderDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

// GetService retrieves a service by name
func (d *OrderDependencyContainer) GetService(name string) (interface{}, error) {
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
func (d *OrderDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	// Close repository
	if repo, exists := d.services["repository"]; exists {
		if orderRepo, ok := repo.(*repository.OrderRepository); ok {
			orderRepo.Close()
		}
	}

	// Close tracing
	if tracer, exists := d.services["tracer"]; exists {
		if tracingShutdown, ok := tracer.(func(context.Context) error); ok {
			tracingShutdown(context.Background())
		}
	}

	d.closed = true
	return nil
}

// AddService adds a service to the container
func (d *OrderDependencyContainer) AddService(name string, service interface{}) {
	d.services[name] = service
}

// loadConfigFromFile loads configuration from YAML file
func loadConfigFromFile(configPath string) (*config.Config, error) {
	cfg := config.DefaultConfig()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.GetLogger().Warn("Config file not found, using defaults", zap.String("path", configPath))
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	cfg.ApplyEnvironmentOverrides()

	return cfg, nil
}

// createServerConfig converts app config to server config
func createServerConfig(cfg *config.Config) *server.ServerConfig {
	return &server.ServerConfig{
		ServiceName: cfg.Server.Name,
		HTTP: server.HTTPConfig{
			Port:         cfg.Server.HTTPPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Port:                cfg.Server.GRPCPort,
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
			Enabled: false, // Disable for this example
		},
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
		Prometheus: *types.DefaultPrometheusConfig(),
	}
}

func main() {
	// Initialize logger
	logger.InitLogger()
	log := logger.GetLogger()

	// Load configuration
	configPath := getEnv("CONFIG_PATH", "config/config.yaml")
	if !filepath.IsAbs(configPath) {
		if execDir, err := os.Executable(); err == nil {
			configPath = filepath.Join(filepath.Dir(execDir), configPath)
		}
	}

	cfg, err := loadConfigFromFile(configPath)
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize tracing
	tracingShutdown, err := tracing.InitTracing(cfg.Tracing)
	if err != nil {
		log.Fatal("Failed to initialize tracing", zap.Error(err))
	}

	// Create dependencies
	deps := NewOrderDependencyContainer()
	deps.AddService("config", cfg)
	deps.AddService("tracer", tracingShutdown)

	// Initialize repository
	repo, err := repository.NewOrderRepository(cfg.Database)
	if err != nil {
		log.Fatal("Failed to create repository", zap.Error(err))
	}
	deps.AddService("repository", repo)

	// Initialize business service with external service clients
	businessService, err := service.NewOrderService(repo, cfg.ExternalServices)
	if err != nil {
		log.Fatal("Failed to create business service", zap.Error(err))
	}
	deps.AddService("business_service", businessService)

	// Create server configuration
	serverConfig := createServerConfig(cfg)
	if err := serverConfig.Validate(); err != nil {
		log.Fatal("Invalid server configuration", zap.Error(err))
	}

	// Create service
	orderServiceInstance := NewOrderService("order-service")

	// Create base server
	baseServer, err := server.NewBusinessServerCore(serverConfig, orderServiceInstance, deps)
	if err != nil {
		log.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(ctx); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}

	log.Info("Order service started successfully",
		zap.String("http_address", baseServer.GetHTTPAddress()),
		zap.String("grpc_address", baseServer.GetGRPCAddress()),
		zap.String("service_name", "order-service"))

	log.Info("Example endpoints",
		zap.String("create_order", fmt.Sprintf("POST http://%s/api/v1/orders", baseServer.GetHTTPAddress())),
		zap.String("get_order", fmt.Sprintf("GET http://%s/api/v1/orders/{id}", baseServer.GetHTTPAddress())),
		zap.String("grpc_endpoint", fmt.Sprintf("grpc://%s", baseServer.GetGRPCAddress())))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutdown signal received, stopping server")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		log.Error("Error during shutdown", zap.Error(err))
	} else {
		log.Info("Server stopped gracefully")
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
