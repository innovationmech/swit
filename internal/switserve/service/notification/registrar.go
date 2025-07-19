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

package notification

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	notificationv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/communication/v1"
	v1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements unified service registration
type ServiceRegistrar struct {
	service     v1.NotificationService
	grpcHandler *v1.GRPCHandler
}

// NewServiceRegistrar creates a new service registrar with dependency injection
func NewServiceRegistrar(service v1.NotificationService) *ServiceRegistrar {
	grpcHandler := v1.NewGRPCHandler(service)

	return &ServiceRegistrar{
		service:     service,
		grpcHandler: grpcHandler,
	}
}

// NewServiceRegistrarLegacy creates a new service registrar using the old pattern.
// Deprecated: Use NewServiceRegistrar with dependency injection instead.
func NewServiceRegistrarLegacy() *ServiceRegistrar {
	service := v1.NewService()
	grpcHandler := v1.NewGRPCHandler(service)

	return &ServiceRegistrar{
		service:     service,
		grpcHandler: grpcHandler,
	}
}

// RegisterGRPC implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	notificationv1.RegisterNotificationServiceServer(server, sr.grpcHandler)
	logger.Logger.Info("Registered Notification gRPC service")
	return nil
}

// RegisterHTTP implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Create HTTP endpoints that mirror gRPC functionality
	v1 := router.Group("/api/v1")
	{
		notifications := v1.Group("/notifications")
		{
			notifications.POST("", sr.createNotificationHTTP)
			notifications.GET("", sr.getNotificationsHTTP)
			notifications.PATCH("/:id/read", sr.markAsReadHTTP)
			notifications.DELETE("/:id", sr.deleteNotificationHTTP)
			// TODO: Add streaming endpoint for Server-Sent Events
		}
	}

	logger.Logger.Info("Registered Notification HTTP routes")
	return nil
}

// GetName implements ServiceRegistrar interface
func (sr *ServiceRegistrar) GetName() string {
	return "notification"
}

// createNotificationHTTP handles HTTP version of CreateNotification
func (sr *ServiceRegistrar) createNotificationHTTP(c *gin.Context) {
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

	notification, err := sr.service.CreateNotification(c.Request.Context(), req.UserID, req.Title, req.Content)
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

// getNotificationsHTTP handles HTTP version of GetNotifications
func (sr *ServiceRegistrar) getNotificationsHTTP(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := sr.service.GetNotifications(c.Request.Context(), userID, limit, offset)
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

// markAsReadHTTP handles HTTP version of MarkAsRead
func (sr *ServiceRegistrar) markAsReadHTTP(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification_id is required"})
		return
	}

	err := sr.service.MarkAsRead(c.Request.Context(), notificationID)
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

// deleteNotificationHTTP handles HTTP version of DeleteNotification
func (sr *ServiceRegistrar) deleteNotificationHTTP(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification_id is required"})
		return
	}

	err := sr.service.DeleteNotification(c.Request.Context(), notificationID)
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
