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
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/types"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"google.golang.org/grpc"
)

// Handler handles health check operations and implements HandlerRegister interface
type Handler struct {
	startTime time.Time
}

// NewHandler creates a new health Handler instance
func NewHandler() *Handler {
	return &Handler{
		startTime: time.Now(),
	}
}

// HealthCheck checks if the authentication service is healthy
//
//	@Summary		Health check
//	@Description	Check if the authentication service is healthy
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.HealthResponse	"Service is healthy"
//	@Router			/health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Message: "pong",
	})
}

// ReadinessCheck checks if the service is ready to serve requests
func (h *Handler) ReadinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"uptime": time.Since(h.startTime).String(),
	})
}

// LivenessCheck checks if the service is alive
func (h *Handler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().Unix(),
	})
}

// HandlerRegister interface implementation

// RegisterHTTP registers HTTP routes for health checks
func (h *Handler) RegisterHTTP(router *gin.Engine) error {
	logger.Logger.Info("Registering health HTTP routes")

	// Register health check routes
	router.GET("/health", h.HealthCheck)
	router.GET("/health/ready", h.ReadinessCheck)
	router.GET("/health/live", h.LivenessCheck)

	return nil
}

// RegisterGRPC registers gRPC services for health checks
func (h *Handler) RegisterGRPC(server *grpc.Server) error {
	logger.Logger.Info("Registering health gRPC services")
	// TODO: Implement gRPC health service registration when needed
	return nil
}

// GetMetadata returns service metadata
func (h *Handler) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:           "health-service",
		Version:        "v1.0.0",
		Description:    "Health check service for authentication service",
		HealthEndpoint: "/health",
		Tags:           []string{"health", "monitoring"},
		Dependencies:   []string{},
	}
}

// GetHealthEndpoint returns the health check endpoint
func (h *Handler) GetHealthEndpoint() string {
	return "/health"
}

// IsHealthy performs a health check
func (h *Handler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	return types.NewHealthStatus(
		types.HealthStatusHealthy,
		"v1",
		time.Since(h.startTime),
	), nil
}

// Initialize initializes the service
func (h *Handler) Initialize(ctx context.Context) error {
	logger.Logger.Info("Initializing health service handler")
	// Perform any initialization logic here
	return nil
}

// Shutdown gracefully shuts down the service
func (h *Handler) Shutdown(ctx context.Context) error {
	logger.Logger.Info("Shutting down health service handler")
	// Perform any cleanup logic here
	return nil
}
