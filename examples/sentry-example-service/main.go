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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
)

// ExampleService demonstrates Sentry integration with the framework
type ExampleService struct {
	sentryManager *server.SentryManager
}

// NewExampleService creates a new example service
func NewExampleService() *ExampleService {
	return &ExampleService{}
}

// RegisterServices registers the service handlers
func (s *ExampleService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	handler := NewExampleHTTPHandler()
	if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	return nil
}

// SetSentryManager sets the Sentry manager for the service
func (s *ExampleService) SetSentryManager(manager *server.SentryManager) {
	s.sentryManager = manager
}

// ExampleHTTPHandler demonstrates HTTP endpoints with Sentry integration
type ExampleHTTPHandler struct{}

// NewExampleHTTPHandler creates a new example HTTP handler
func NewExampleHTTPHandler() *ExampleHTTPHandler {
	return &ExampleHTTPHandler{}
}

// RegisterRoutes registers HTTP routes
func (h *ExampleHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter := router.(*gin.Engine)

	// Health check endpoint (typically ignored by Sentry)
	ginRouter.GET("/health", h.handleHealth)

	// API endpoints that demonstrate Sentry integration
	v1 := ginRouter.Group("/api/v1")
	{
		v1.GET("/success", h.handleSuccess)
		v1.GET("/error", h.handleError)
		v1.GET("/panic", h.handlePanic)
		v1.GET("/custom-sentry", h.handleCustomSentry)
		v1.POST("/users", h.handleCreateUser)
	}

	return nil
}

// GetServiceName returns the service name
func (h *ExampleHTTPHandler) GetServiceName() string {
	return "sentry-example-service"
}

// handleHealth returns a health check response
func (h *ExampleHTTPHandler) handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

// handleSuccess demonstrates a successful response (not captured by Sentry)
func (h *ExampleHTTPHandler) handleSuccess(c *gin.Context) {
	// Add a breadcrumb for this successful operation
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Type:     "http",
		Category: "api",
		Message:  "Successful API call to /success",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"endpoint": "/api/v1/success",
			"method":   "GET",
		},
	})

	c.JSON(200, gin.H{
		"message": "This is a successful response",
		"data":    map[string]interface{}{"value": 42},
	})
}

// handleError demonstrates an error response (captured by Sentry)
func (h *ExampleHTTPHandler) handleError(c *gin.Context) {
	// Simulate an error condition
	err := fmt.Errorf("simulated error for Sentry demonstration")

	// Manually capture the error with additional context
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("error_type", "simulated")
		scope.SetContext("request_info", map[string]interface{}{
			"endpoint":   "/api/v1/error",
			"user_agent": c.GetHeader("User-Agent"),
			"ip":         c.ClientIP(),
		})
		sentry.CaptureException(err)
	})

	c.JSON(500, gin.H{
		"error":   "Internal server error",
		"message": "This error will be captured by Sentry",
	})
}

// handlePanic demonstrates panic recovery and Sentry capture
func (h *ExampleHTTPHandler) handlePanic(c *gin.Context) {
	// Add context before panic
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("panic_demo", "true")
		scope.SetLevel(sentry.LevelFatal)
	})

	// This will panic and be captured by Sentry middleware
	panic("This is a demonstration panic - it will be captured by Sentry")
}

// handleCustomSentry demonstrates custom Sentry usage
func (h *ExampleHTTPHandler) handleCustomSentry(c *gin.Context) {
	// Create a custom event
	event := &sentry.Event{
		Message: "Custom Sentry event from example service",
		Level:   sentry.LevelWarning,
		Tags: map[string]string{
			"custom_event": "true",
			"endpoint":     "/api/v1/custom-sentry",
		},
		Extra: map[string]interface{}{
			"request_id": c.GetString("request_id"),
			"timestamp":  time.Now().UTC(),
		},
	}

	// Add custom context
	event.Contexts = map[string]sentry.Context{
		"business_logic": {
			"operation": "custom_sentry_demo",
			"success":   true,
			"duration":  "50ms",
		},
	}

	// Capture the event
	eventID := sentry.CaptureEvent(event)

	c.JSON(200, gin.H{
		"message":  "Custom Sentry event sent",
		"event_id": eventID,
	})
}

// handleCreateUser demonstrates user context and transaction tracking
func (h *ExampleHTTPHandler) handleCreateUser(c *gin.Context) {
	// Start a transaction for this operation
	transaction := sentry.StartTransaction(c.Request.Context(), "create_user")
	defer transaction.Finish()

	// Set user context
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:        "demo-user-123",
			Username:  "demo_user",
			Email:     "demo@example.com",
			IPAddress: c.ClientIP(),
		})
	})

	// Simulate some business logic with spans
	span := transaction.StartChild("validate_input")
	time.Sleep(10 * time.Millisecond) // Simulate validation
	span.Finish()

	span = transaction.StartChild("save_to_database")
	time.Sleep(20 * time.Millisecond) // Simulate database save
	span.Finish()

	// Add breadcrumb for successful user creation
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Type:     "user",
		Category: "auth",
		Message:  "User created successfully",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"user_id": "demo-user-123",
			"method":  "POST",
		},
	})

	transaction.SetTag("operation", "user_creation")
	transaction.SetTag("success", "true")

	c.JSON(201, gin.H{
		"message": "User created successfully",
		"user_id": "demo-user-123",
	})
}

func main() {
	// Initialize logger first
	logger.InitLogger()

	// Create server configuration
	config := server.NewServerConfig()
	config.ServiceName = "sentry-example-service"
	config.HTTP.Port = "8090"
	config.GRPC.Port = "9090"

	// Configure Sentry - you'll need to set SENTRY_DSN environment variable
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN == "" {
		log.Println("Warning: SENTRY_DSN environment variable not set. Sentry will be disabled.")
		log.Println("To enable Sentry, set SENTRY_DSN=your_sentry_dsn_here")
		config.Sentry.Enabled = false
	} else {
		config.Sentry.Enabled = true
		config.Sentry.DSN = sentryDSN
	}

	config.Sentry.Environment = "development"
	config.Sentry.SampleRate = 1.0        // Capture 100% of errors in development
	config.Sentry.TracesSampleRate = 0.1  // Capture 10% of performance data
	config.Sentry.Debug = true            // Enable debug mode for development
	config.Sentry.AttachStacktrace = true // Include stack traces
	config.Sentry.EnableTracing = true    // Enable performance monitoring

	// Configure Sentry tags
	config.Sentry.Tags = map[string]string{
		"service":     "sentry-example",
		"version":     "1.0.0",
		"environment": "development",
	}

	// Configure HTTP paths to ignore (health checks, metrics)
	config.Sentry.HTTPIgnorePaths = []string{"/health", "/metrics"}

	// Configure HTTP status codes to ignore (don't capture client errors)
	config.Sentry.HTTPIgnoreStatusCode = []int{400, 401, 403, 404}

	// Disable service discovery for this example
	config.Discovery.Enabled = false

	// Create service
	service := NewExampleService()

	// Create server
	srv, err := server.NewBusinessServerCore(config, service, nil)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Set the Sentry manager on the service (for demonstration)
	service.SetSentryManager(srv.GetSentryManager())

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	log.Printf("Starting Sentry example service on HTTP :%s and gRPC :%s",
		config.HTTP.Port, config.GRPC.Port)

	if config.Sentry.Enabled {
		log.Printf("Sentry error monitoring is ENABLED")
	} else {
		log.Printf("Sentry error monitoring is DISABLED")
	}

	go func() {
		if err := srv.Start(ctx); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Print example endpoints
	log.Printf("\nExample endpoints:")
	log.Printf("  Health check:    curl http://localhost:%s/health", config.HTTP.Port)
	log.Printf("  Success:         curl http://localhost:%s/api/v1/success", config.HTTP.Port)
	log.Printf("  Error (5xx):     curl http://localhost:%s/api/v1/error", config.HTTP.Port)
	log.Printf("  Custom Sentry:   curl http://localhost:%s/api/v1/custom-sentry", config.HTTP.Port)
	log.Printf("  Create User:     curl -X POST http://localhost:%s/api/v1/users", config.HTTP.Port)
	log.Printf("  Panic Demo:      curl http://localhost:%s/api/v1/panic", config.HTTP.Port)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	if err := srv.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server shut down successfully")
}
