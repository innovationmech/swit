// Example service demonstrating Sentry integration with the swit framework
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
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ExampleService demonstrates how to use Sentry with the swit framework
type ExampleService struct{}

// RegisterServices implements BusinessServiceRegistrar interface
func (s *ExampleService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handlers
	httpHandler := &ExampleHTTPHandler{}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	return nil
}

// ExampleHTTPHandler demonstrates HTTP endpoints that can generate errors for Sentry
type ExampleHTTPHandler struct{}

// RegisterRoutes registers HTTP routes that demonstrate Sentry integration
func (h *ExampleHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected *gin.Engine, got %T", router)
	}

	// Add routes that demonstrate different error scenarios
	ginRouter.GET("/health", h.healthCheck)
	ginRouter.GET("/success", h.successResponse)
	ginRouter.GET("/error", h.errorResponse)
	ginRouter.GET("/panic", h.panicResponse)
	ginRouter.GET("/slow", h.slowResponse)

	return nil
}

// GetServiceName returns the service name
func (h *ExampleHTTPHandler) GetServiceName() string {
	return "example-http-service"
}

// healthCheck endpoint for health checks
func (h *ExampleHTTPHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "sentry-example-service",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// successResponse demonstrates a successful request (no Sentry event)
func (h *ExampleHTTPHandler) successResponse(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "This request will not generate a Sentry event",
		"status":  "success",
	})
}

// errorResponse demonstrates a 400-level error (Sentry warning)
func (h *ExampleHTTPHandler) errorResponse(c *gin.Context) {
	// Add error to gin context - this will be captured by Sentry middleware
	err := fmt.Errorf("example validation error: missing required parameter")
	c.Error(err)
	
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "Bad Request",
		"message": "This will generate a Sentry warning event",
		"code":    "VALIDATION_ERROR",
	})
}

// panicResponse demonstrates a panic (Sentry fatal error)
func (h *ExampleHTTPHandler) panicResponse(c *gin.Context) {
	// This panic will be caught by Sentry middleware and reported as a fatal error
	panic("Example panic for Sentry demonstration - this is intentional!")
}

// slowResponse demonstrates a slow request (for performance monitoring)
func (h *ExampleHTTPHandler) slowResponse(c *gin.Context) {
	// Simulate slow processing
	time.Sleep(2 * time.Second)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "This slow response will be tracked by Sentry performance monitoring",
		"duration": "2 seconds",
	})
}

func main() {
	// Initialize logger
	logger.InitLogger()
	defer logger.Logger.Sync()

	// Load configuration
	config := loadConfiguration()
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create service
	service := &ExampleService{}

	// Create server
	baseServer, err := server.NewBusinessServerCore(config, service, nil)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := baseServer.Start(ctx); err != nil {
			logger.Logger.Error("Server start failed", zap.Error(err))
			cancel()
		}
	}()

	logger.Logger.Info("Sentry example service started",
		zap.String("http_address", baseServer.GetHTTPAddress()),
		zap.String("grpc_address", baseServer.GetGRPCAddress()))

	logger.Logger.Info("Try these endpoints to test Sentry integration:",
		zap.String("health", "GET http://localhost:8080/health"),
		zap.String("success", "GET http://localhost:8080/success (no Sentry event)"),
		zap.String("error", "GET http://localhost:8080/error (Sentry warning)"),
		zap.String("panic", "GET http://localhost:8080/panic (Sentry fatal - will recover)"),
		zap.String("slow", "GET http://localhost:8080/slow (performance tracking)"))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Logger.Info("Shutting down server...")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		logger.Logger.Error("Server shutdown failed", zap.Error(err))
	} else {
		logger.Logger.Info("Server shutdown completed")
	}
}

// loadConfiguration loads server configuration with Sentry settings
func loadConfiguration() *server.ServerConfig {
	config := server.NewServerConfig()
	
	// Configure Sentry from environment variables
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		config.Sentry.Enabled = true
		config.Sentry.DSN = dsn
		config.Sentry.Environment = getEnvOrDefault("SENTRY_ENVIRONMENT", "development")
		config.Sentry.Release = getEnvOrDefault("SENTRY_RELEASE", "v1.0.0")
		config.Sentry.Debug = getEnvOrDefault("SENTRY_DEBUG", "false") == "true"
		
		// Performance monitoring settings
		config.Sentry.EnableTracing = true
		config.Sentry.TracesSampleRate = 0.1
		config.Sentry.EnableProfiling = false
		
		// Add custom tags
		config.Sentry.Tags = map[string]string{
			"service":   "sentry-example",
			"component": "swit-framework",
			"version":   "v1.0.0",
		}
		
		logger.Logger.Info("Sentry integration enabled",
			zap.String("environment", config.Sentry.Environment),
			zap.String("release", config.Sentry.Release),
			zap.Bool("debug", config.Sentry.Debug))
	} else {
		logger.Logger.Warn("SENTRY_DSN not set, Sentry integration disabled")
		logger.Logger.Info("Set SENTRY_DSN environment variable to enable Sentry error monitoring")
	}
	
	return config
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}