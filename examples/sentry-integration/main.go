// Example demonstrating Sentry integration with the Swit framework
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/server"
)

// ExampleService demonstrates a service with Sentry integration
type ExampleService struct{}

func (s *ExampleService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &ExampleHTTPHandler{}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	return nil
}

// ExampleHTTPHandler provides HTTP endpoints
type ExampleHTTPHandler struct{}

func (h *ExampleHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(*gin.Engine)
	if !ok {
		return fmt.Errorf("expected gin.Engine, got %T", router)
	}

	// Register routes that will demonstrate Sentry error capture
	ginRouter.GET("/success", h.handleSuccess)
	ginRouter.GET("/error", h.handleError)
	ginRouter.GET("/panic", h.handlePanic)

	return nil
}

func (h *ExampleHTTPHandler) GetServiceName() string {
	return "example-service"
}

func (h *ExampleHTTPHandler) handleSuccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Success! This request will not be sent to Sentry.",
		"time":    time.Now(),
	})
}

func (h *ExampleHTTPHandler) handleError(c *gin.Context) {
	// This will be captured by Sentry as an HTTP 500 error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "This error will be captured by Sentry",
		"time":  time.Now(),
	})
}

func (h *ExampleHTTPHandler) handlePanic(c *gin.Context) {
	// This panic will be captured by Sentry
	panic("This panic will be captured by Sentry!")
}

func main() {
	// Create server configuration with Sentry enabled
	config := server.NewServerConfig()
	config.ServiceName = "sentry-example-service"
	config.HTTP.Port = "8080"
	config.GRPC.Enabled = false // Disable gRPC for this example

	// Configure Sentry - Replace with your actual DSN
	config.Sentry.Enabled = true
	// Note: Replace this with your actual Sentry DSN
	// config.Sentry.DSN = "https://your-dsn@your-org.sentry.io/your-project"
	config.Sentry.Environment = "development"
	config.Sentry.Release = "1.0.0"
	config.Sentry.Debug = true // Enable debug for testing

	// For this example, we'll keep Sentry disabled since we don't have a real DSN
	config.Sentry.Enabled = false

	fmt.Println("=== Sentry Integration Example ===")
	fmt.Printf("Service: %s\n", config.ServiceName)
	fmt.Printf("Sentry Enabled: %t\n", config.Sentry.Enabled)
	fmt.Printf("Sentry Environment: %s\n", config.Sentry.Environment)
	fmt.Printf("Sentry Sample Rate: %.1f\n", config.Sentry.SampleRate)
	fmt.Printf("Sentry Traces Sample Rate: %.1f\n", config.Sentry.TracesSampleRate)
	fmt.Println()

	// Create service instance
	service := &ExampleService{}

	// Create and start server
	srv, err := server.NewBusinessServerCore(config, service, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create server: %v", err))
	}

	fmt.Println("Starting server...")
	fmt.Printf("Visit http://localhost:%s/success for a successful request\n", config.HTTP.Port)
	fmt.Printf("Visit http://localhost:%s/error for an error that would be sent to Sentry\n", config.HTTP.Port)
	fmt.Printf("Visit http://localhost:%s/panic for a panic that would be sent to Sentry\n", config.HTTP.Port)
	fmt.Println()

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}

	// Keep the server running for a short time for demonstration
	fmt.Println("Server is running... (will stop automatically in 30 seconds)")
	time.Sleep(30 * time.Second)

	fmt.Println("Shutting down server...")
	if err := srv.Shutdown(); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
	}

	fmt.Println("Server shutdown complete.")
	fmt.Println()
	fmt.Println("To enable Sentry in production:")
	fmt.Println("1. Get your DSN from https://sentry.io")
	fmt.Println("2. Set config.Sentry.DSN to your actual DSN")
	fmt.Println("3. Set config.Sentry.Enabled = true")
	fmt.Println("4. Configure appropriate environment and release values")
}