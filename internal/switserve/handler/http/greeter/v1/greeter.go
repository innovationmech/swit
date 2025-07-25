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

package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/transport"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// GreeterHandler handles HTTP requests for greeter service and implements ServiceHandler interface
type GreeterHandler struct {
	service   interfaces.GreeterService
	startTime time.Time
}

// NewGreeterHandler creates a new greeter HTTP handler
func NewGreeterHandler(service interfaces.GreeterService) *GreeterHandler {
	return &GreeterHandler{
		service:   service,
		startTime: time.Now(),
	}
}

// SayHello handles HTTP version of SayHello
func (h *GreeterHandler) SayHello(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Language string `json:"language,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	greeting, err := h.service.GenerateGreeting(c.Request.Context(), req.Name, req.Language)
	if err != nil {
		logger.Logger.Error("Failed to generate greeting", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := gin.H{
		"message": greeting,
		"metadata": gin.H{
			"request_id": c.GetHeader("X-Request-ID"),
			"server_id":  "swit-serve-1",
		},
	}

	c.JSON(http.StatusOK, response)
}

// ServiceHandler interface implementation

// RegisterHTTP registers HTTP routes for the Greeter service
func (h *GreeterHandler) RegisterHTTP(router *gin.Engine) error {
	logger.Logger.Info("Registering greeter HTTP routes")

	// Greeter endpoints
	v1 := router.Group("/api/v1")
	v1.POST("/greet", h.SayHello)

	return nil
}

// RegisterGRPC registers gRPC services for the Greeter service
func (h *GreeterHandler) RegisterGRPC(server *grpc.Server) error {
	logger.Logger.Info("Greeter service gRPC registration - HTTP endpoints available")
	// Greeter service primarily uses HTTP endpoints
	// gRPC implementation can be added later if needed
	return nil
}

// GetMetadata returns service metadata information
func (h *GreeterHandler) GetMetadata() *transport.ServiceMetadata {
	return &transport.ServiceMetadata{
		Name:           "greeter",
		Version:        "v1",
		Description:    "Greeter service providing greeting functionality",
		HealthEndpoint: "/api/v1/health",
		Tags:           []string{"greeter", "messaging", "api"},
		Dependencies:   []string{},
	}
}

// GetHealthEndpoint returns the health check endpoint path
func (h *GreeterHandler) GetHealthEndpoint() string {
	return "/api/v1/health"
}

// IsHealthy performs a health check and returns the current status
func (h *GreeterHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Check if greeter service is healthy
	if h.service == nil {
		return types.NewHealthStatus(
			types.HealthStatusUnhealthy,
			"v1",
			time.Since(h.startTime),
		), nil
	}

	// Greeter service is healthy if it can process requests
	return types.NewHealthStatus(
		types.HealthStatusHealthy,
		"v1",
		time.Since(h.startTime),
	), nil
}

// Initialize performs any necessary initialization before service registration
func (h *GreeterHandler) Initialize(ctx context.Context) error {
	logger.Logger.Info("Initializing greeter service handler")
	// Greeter service initialization logic
	return nil
}

// Shutdown performs graceful shutdown of the service
func (h *GreeterHandler) Shutdown(ctx context.Context) error {
	logger.Logger.Info("Shutting down greeter service handler")
	// Greeter service cleanup logic
	return nil
}
