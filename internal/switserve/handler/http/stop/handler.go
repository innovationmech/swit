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

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ShutdownStatus represents shutdown response
// This is defined here to avoid circular dependency between handler and service layers
type ShutdownStatus struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// StopService defines the interface for shutdown business logic
// This is defined here to avoid circular dependency between handler and service layers
// The service layer implements this interface via an adapter in the registrar
type StopService interface {
	InitiateShutdown(ctx context.Context) (*ShutdownStatus, error)
}

// Handler handles HTTP requests for stop service
type Handler struct {
	service StopService
}

// NewHandler creates a new stop HTTP handler
func NewHandler(service StopService) *Handler {
	return &Handler{
		service: service,
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
