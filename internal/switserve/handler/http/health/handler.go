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

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Status represents health check response
// This is defined here to avoid circular dependency between handler and service layers
type Status struct {
	Status    string            `json:"status"`
	Timestamp int64             `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthService defines the interface for health check business logic
// This is defined here to avoid circular dependency between handler and service layers
// The service layer implements this interface via an adapter in the registrar
type HealthService interface {
	CheckHealth(ctx context.Context) (*Status, error)
}

// Handler handles HTTP requests for health service
type Handler struct {
	service HealthService
}

// NewHandler creates a new health HTTP handler
func NewHandler(service HealthService) *Handler {
	return &Handler{
		service: service,
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
