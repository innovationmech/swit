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
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/transport"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// NotificationHandler handles HTTP requests for notification service and implements HandlerRegister interface
type NotificationHandler struct {
	service   v1.NotificationService
	startTime time.Time
}

// NewNotificationHandler creates a new notification HTTP handler
func NewNotificationHandler(service v1.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service:   service,
		startTime: time.Now(),
	}
}

// CreateNotification handles HTTP version of CreateNotification
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	notification, err := h.service.CreateNotification(c.Request.Context(), req.UserID, req.Title, req.Content)
	if err != nil {
		logger.Logger.Error("Failed to create notification", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := gin.H{
		"notification": gin.H{
			"id":         notification.ID,
			"user_id":    notification.UserID,
			"title":      notification.Title,
			"content":    notification.Content,
			"is_read":    notification.IsRead,
			"created_at": notification.CreatedAt,
			"updated_at": notification.UpdatedAt,
		},
		"metadata": gin.H{
			"request_id": c.GetHeader("X-Request-ID"),
			"server_id":  "swit-serve-1",
		},
	}

	c.JSON(http.StatusCreated, response)
}

// GetNotifications handles HTTP version of GetNotifications
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := h.service.GetNotifications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logger.Logger.Error("Failed to get notifications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Convert to response format
	notificationList := make([]gin.H, len(notifications))
	for i, notification := range notifications {
		notificationList[i] = gin.H{
			"id":         notification.ID,
			"user_id":    notification.UserID,
			"title":      notification.Title,
			"content":    notification.Content,
			"is_read":    notification.IsRead,
			"created_at": notification.CreatedAt,
			"updated_at": notification.UpdatedAt,
		}
	}

	response := gin.H{
		"notifications": notificationList,
		"total_count":   len(notifications),
		"metadata": gin.H{
			"request_id": c.GetHeader("X-Request-ID"),
			"server_id":  "swit-serve-1",
		},
	}

	c.JSON(http.StatusOK, response)
}

// HandlerRegister interface implementation

// RegisterHTTP registers HTTP routes for the Notification service
func (h *NotificationHandler) RegisterHTTP(router *gin.Engine) error {
	logger.Logger.Info("Registering notification HTTP routes")

	// Notification endpoints
	v1 := router.Group("/api/v1")
	v1.POST("/notifications", h.CreateNotification)
	v1.GET("/notifications", h.GetNotifications)
	v1.PUT("/notifications/:id/read", h.MarkAsRead)
	v1.DELETE("/notifications/:id", h.DeleteNotification)

	return nil
}

// RegisterGRPC registers gRPC services for the Notification service
func (h *NotificationHandler) RegisterGRPC(server *grpc.Server) error {
	logger.Logger.Info("Notification service gRPC registration - HTTP endpoints available")
	// Notification service primarily uses HTTP endpoints
	// gRPC implementation can be added later if needed
	return nil
}

// GetMetadata returns service metadata information
func (h *NotificationHandler) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:           "notification",
		Version:        "v1",
		Description:    "Notification service providing notification management functionality",
		HealthEndpoint: "/api/v1/health",
		Tags:           []string{"notification", "messaging", "api"},
		Dependencies:   []string{},
	}
}

// GetHealthEndpoint returns the health check endpoint path
func (h *NotificationHandler) GetHealthEndpoint() string {
	return "/api/v1/health"
}

// IsHealthy performs a health check and returns the current status
func (h *NotificationHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Check if notification service is healthy
	if h.service == nil {
		return types.NewHealthStatus(
			types.HealthStatusUnhealthy,
			"v1",
			time.Since(h.startTime),
		), nil
	}

	// Notification service is healthy if it can process requests
	return types.NewHealthStatus(
		types.HealthStatusHealthy,
		"v1",
		time.Since(h.startTime),
	), nil
}

// Initialize performs any necessary initialization before service registration
func (h *NotificationHandler) Initialize(ctx context.Context) error {
	logger.Logger.Info("Initializing notification service handler")
	// Notification service initialization logic
	return nil
}

// Shutdown performs graceful shutdown of the service
func (h *NotificationHandler) Shutdown(ctx context.Context) error {
	logger.Logger.Info("Shutting down notification service handler")
	// Notification service cleanup logic
	return nil
}

// MarkAsRead handles HTTP version of MarkAsRead
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification_id is required"})
		return
	}

	err := h.service.MarkAsRead(c.Request.Context(), notificationID)
	if err != nil {
		logger.Logger.Error("Failed to mark notification as read", zap.Error(err))
		if errors.Is(err, v1.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := gin.H{
		"success": true,
		"metadata": gin.H{
			"request_id": c.GetHeader("X-Request-ID"),
			"server_id":  "swit-serve-1",
		},
	}

	c.JSON(http.StatusOK, response)
}

// DeleteNotification handles HTTP version of DeleteNotification
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification_id is required"})
		return
	}

	err := h.service.DeleteNotification(c.Request.Context(), notificationID)
	if err != nil {
		logger.Logger.Error("Failed to delete notification", zap.Error(err))
		if errors.Is(err, v1.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := gin.H{
		"success": true,
		"metadata": gin.H{
			"request_id": c.GetHeader("X-Request-ID"),
			"server_id":  "swit-serve-1",
		},
	}

	c.JSON(http.StatusOK, response)
}
