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

package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Handler handles HTTP requests for health service and implements HandlerRegister interface
type Handler struct {
	service   interfaces.HealthService
	startTime time.Time
}

// NewHandler creates a new health HTTP handler
func NewHandler(service interfaces.HealthService) *Handler {
	return &Handler{
		service:   service,
		startTime: time.Now(),
	}
}

// HealthCheck handles HTTP health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	status, err := h.service.CheckHealth(c.Request.Context())
	if err != nil {
		logger.Logger.Error("Health check failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "unhealthy",
			"error":  "health check failed",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ReadinessCheck handles GET /api/v1/health/ready
func (h *Handler) ReadinessCheck(c *gin.Context) {
	// Readiness check - service is ready to accept traffic
	healthStatus, err := h.service.CheckHealth(c.Request.Context())
	if err != nil {
		logger.Logger.Error("Readiness check failed", zap.Error(err))
		c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
		return
	}

	if healthStatus.Status == "healthy" {
		c.JSON(200, gin.H{"status": "ready"})
	} else {
		c.JSON(503, gin.H{"status": "not ready"})
	}
}

// LivenessCheck handles GET /api/v1/health/live
func (h *Handler) LivenessCheck(c *gin.Context) {
	// Liveness check - service is alive (basic check)
	if h.service != nil {
		c.JSON(200, gin.H{"status": "alive"})
	} else {
		c.JSON(503, gin.H{"status": "not alive"})
	}
}

// HandlerRegister interface implementation

// RegisterHTTP registers HTTP routes for the Health service
func (h *Handler) RegisterHTTP(router *gin.Engine) error {
	logger.Logger.Info("Registering health HTTP routes")

	// Health check endpoint
	v1 := router.Group("/api/v1")
	v1.GET("/health", h.HealthCheck)
	v1.GET("/health/ready", h.ReadinessCheck)
	v1.GET("/health/live", h.LivenessCheck)

	return nil
}

// RegisterGRPC registers gRPC services for the Health service
func (h *Handler) RegisterGRPC(server *grpc.Server) error {
	logger.Logger.Info("Health service gRPC registration - HTTP endpoints available")
	// Health service primarily uses HTTP endpoints
	// gRPC health checking can be implemented later if needed
	return nil
}

// GetMetadata returns service metadata information
func (h *Handler) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:           "health",
		Version:        "v1",
		Description:    "Health check service providing system health status",
		HealthEndpoint: "/api/v1/health",
		Tags:           []string{"health", "monitoring", "system"},
		Dependencies:   []string{},
	}
}

// GetHealthEndpoint returns the health check endpoint path
func (h *Handler) GetHealthEndpoint() string {
	return "/api/v1/health"
}

// IsHealthy performs a health check and returns the current status
func (h *Handler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Check if health service is healthy
	if h.service == nil {
		return types.NewHealthStatus(
			types.HealthStatusUnhealthy,
			"v1",
			time.Since(h.startTime),
		), nil
	}

	// Use the health service to check overall system health
	healthStatus, err := h.service.CheckHealth(ctx)
	if err != nil {
		return types.NewHealthStatus(
			types.HealthStatusUnhealthy,
			"v1",
			time.Since(h.startTime),
		), err
	}

	// Convert interfaces.HealthStatus to types.HealthStatus
	typesHealthStatus := types.NewHealthStatus(
		healthStatus.Status,
		"v1",
		time.Since(h.startTime),
	)

	return typesHealthStatus, nil
}

// Initialize performs any necessary initialization before service registration
func (h *Handler) Initialize(ctx context.Context) error {
	logger.Logger.Info("Initializing health service handler")
	// Health service initialization logic
	return nil
}

// Shutdown performs graceful shutdown of the service
func (h *Handler) Shutdown(ctx context.Context) error {
	logger.Logger.Info("Shutting down health service handler")
	// Health service cleanup logic
	return nil
}
