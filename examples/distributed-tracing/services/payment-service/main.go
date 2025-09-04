// Package main demonstrates a payment service using the swit framework with OpenTelemetry tracing
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

	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/config"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/handler"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/service"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/tracing"
)

// PaymentService implements the ServiceRegistrar interface
type PaymentService struct {
	name string
}

// NewPaymentService creates a new payment service
func NewPaymentService(name string) *PaymentService {
	return &PaymentService{
		name: name,
	}
}

// RegisterServices registers gRPC services with the server
func (s *PaymentService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Get dependencies from container
	deps, err := registry.GetDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to get dependency container: %w", err)
	}

	// Get business service
	businessService, err := deps.GetService("business_service")
	if err != nil {
		return fmt.Errorf("failed to get business service: %w", err)
	}
	paymentService := businessService.(*service.PaymentService)

	// Register gRPC service (payment service is gRPC-only)
	grpcService := handler.NewPaymentGRPCService(paymentService)
	if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	// Register health check
	healthCheck := &PaymentHealthCheck{serviceName: s.name, service: paymentService}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// PaymentHealthCheck implements the HealthCheck interface
type PaymentHealthCheck struct {
	serviceName string
	service     *service.PaymentService
}

// Check performs a health check
func (h *PaymentHealthCheck) Check(ctx context.Context) error {
	// Check service health
	if err := h.service.HealthCheck(ctx); err != nil {
		return fmt.Errorf("payment service health check failed: %w", err)
	}
	return nil
}

// GetServiceName returns the service name
func (h *PaymentHealthCheck) GetServiceName() string {
	return h.serviceName
}

// PaymentDependencyContainer implements the DependencyContainer interface
type PaymentDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewPaymentDependencyContainer creates a new dependency container
func NewPaymentDependencyContainer() *PaymentDependencyContainer {
	return &PaymentDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

// GetService retrieves a service by name
func (d *PaymentDependencyContainer) GetService(name string) (interface{}, error) {
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
func (d *PaymentDependencyContainer) Close() error {
	if d.closed {
		return nil
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
func (d *PaymentDependencyContainer) AddService(name string, service interface{}) {
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
			Enabled: false, // Payment service is gRPC-only
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
	deps := NewPaymentDependencyContainer()
	deps.AddService("config", cfg)
	deps.AddService("tracer", tracingShutdown)

	// Initialize business service
	businessService := service.NewPaymentService(cfg.Payment)
	deps.AddService("business_service", businessService)

	// Create server configuration
	serverConfig := createServerConfig(cfg)
	if err := serverConfig.Validate(); err != nil {
		log.Fatal("Invalid server configuration", zap.Error(err))
	}

	// Create service
	paymentServiceInstance := NewPaymentService("payment-service")

	// Create base server
	baseServer, err := server.NewBusinessServerCore(serverConfig, paymentServiceInstance, deps)
	if err != nil {
		log.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(ctx); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}

	log.Info("Payment service started successfully",
		zap.String("grpc_address", baseServer.GetGRPCAddress()),
		zap.String("service_name", "payment-service"))

	log.Info("gRPC endpoints available",
		zap.String("process_payment", fmt.Sprintf("grpc://%s/swit.payment.v1.PaymentService/ProcessPayment", baseServer.GetGRPCAddress())),
		zap.String("validate_payment", fmt.Sprintf("grpc://%s/swit.payment.v1.PaymentService/ValidatePayment", baseServer.GetGRPCAddress())),
		zap.String("get_payment_status", fmt.Sprintf("grpc://%s/swit.payment.v1.PaymentService/GetPaymentStatus", baseServer.GetGRPCAddress())),
		zap.String("refund_payment", fmt.Sprintf("grpc://%s/swit.payment.v1.PaymentService/RefundPayment", baseServer.GetGRPCAddress())))

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
