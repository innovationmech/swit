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
	"github.com/gin-gonic/gin"
	notificationv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/communication/v1"
	httpv1 "github.com/innovationmech/swit/internal/switserve/handler/http/notification/v1"
	v1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements unified service registration
type ServiceRegistrar struct {
	httpHandler *httpv1.Handler
	grpcHandler *v1.GRPCHandler
}

// NewServiceRegistrar creates a new service registrar with dependency injection
func NewServiceRegistrar(service v1.NotificationService) *ServiceRegistrar {
	httpHandler := httpv1.NewHandler(service)
	grpcHandler := v1.NewGRPCHandler(service)

	return &ServiceRegistrar{
		httpHandler: httpHandler,
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
			notifications.POST("", sr.httpHandler.CreateNotification)
			notifications.GET("", sr.httpHandler.GetNotifications)
			notifications.PATCH("/:id/read", sr.httpHandler.MarkAsRead)
			notifications.DELETE("/:id", sr.httpHandler.DeleteNotification)
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
