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

package stop

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

// Handler handles HTTP requests for stop service and implements ServiceHandler interface
type Handler struct {
	service   interfaces.StopService
	startTime time.Time
}

// NewHandler creates a new stop HTTP handler
func NewHandler(service interfaces.StopService) *Handler {
	return &Handler{
		service:   service,
		startTime: time.Now(),
	}
}

// StopServer handles HTTP server shutdown requests
func (h *Handler) StopServer(c *gin.Context) {
	logger.Logger.Info("Shutdown request received via HTTP")

	status, err := h.service.InitiateShutdown(c.Request.Context())
	if err != nil {
		logger.Logger.Error("Failed to initiate shutdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "failed to initiate shutdown",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ServiceHandler interface implementation

// RegisterHTTP registers HTTP routes for the Stop service
func (h *Handler) RegisterHTTP(router *gin.Engine) error {
	// Create API v1 group
	v1Group := router.Group("/api/v1")

	// Stop endpoints
	stopGroup := v1Group.Group("/stop")
	{
		stopGroup.POST("/shutdown", h.StopServer)
	}

	if logger.Logger != nil {
		logger.Logger.Info("Registered Stop HTTP routes")
	}
	return nil
}

// RegisterGRPC registers gRPC services for the Stop service
func (h *Handler) RegisterGRPC(server *grpc.Server) error {
	// TODO: Implement gRPC registration when stop gRPC service is available
	if logger.Logger != nil {
		logger.Logger.Info("Registered Stop gRPC services")
	}
	return nil
}

// GetMetadata returns service metadata information
func (h *Handler) GetMetadata() *transport.ServiceMetadata {
	return &transport.ServiceMetadata{
		Name:           "stop-service",
		Version:        "v1.0.0",
		Description:    "Stop service for handling server shutdown requests",
		HealthEndpoint: "/api/v1/stop/health",
		Tags:           []string{"stop", "shutdown"},
		Dependencies:   []string{},
	}
}

// GetHealthEndpoint returns the health check endpoint path
func (h *Handler) GetHealthEndpoint() string {
	return "/api/v1/stop/health"
}

// IsHealthy performs a health check and returns the current status
func (h *Handler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Check if stop service is healthy
	if h.service == nil {
		return types.NewHealthStatus(
			types.HealthStatusUnhealthy,
			"v1",
			time.Since(h.startTime),
		), nil
	}

	return types.NewHealthStatus(
		types.HealthStatusHealthy,
		"v1",
		time.Since(h.startTime),
	), nil
}

// Initialize performs any necessary initialization before service registration
func (h *Handler) Initialize(ctx context.Context) error {
	// Perform any initialization logic for the stop service
	if logger.Logger != nil {
		logger.Logger.Info("Initializing Stop service handler")
	}
	return nil
}

// Shutdown performs graceful shutdown of the service
func (h *Handler) Shutdown(ctx context.Context) error {
	// Perform any cleanup logic for the stop service
	if logger.Logger != nil {
		logger.Logger.Info("Shutting down Stop service handler")
	}
	return nil
}
