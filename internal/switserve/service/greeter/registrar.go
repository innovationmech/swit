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

package greeter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	greeterv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	v1 "github.com/innovationmech/swit/internal/switserve/service/greeter/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements unified service registration
type ServiceRegistrar struct {
	service     v1.GreeterService
	grpcHandler *v1.GRPCHandler
}

// NewServiceRegistrar creates a new service registrar with dependency injection
func NewServiceRegistrar(service v1.GreeterService) *ServiceRegistrar {
	grpcHandler := v1.NewGRPCHandler(service)

	return &ServiceRegistrar{
		service:     service,
		grpcHandler: grpcHandler,
	}
}

// RegisterGRPC implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	greeterv1.RegisterGreeterServiceServer(server, sr.grpcHandler)
	logger.Logger.Info("Registered Greeter gRPC service")
	return nil
}

// RegisterHTTP implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Create HTTP endpoints that mirror gRPC functionality
	v1 := router.Group("/api/v1")
	{
		greeter := v1.Group("/greeter")
		{
			greeter.POST("/hello", sr.sayHelloHTTP)
			// Add more HTTP endpoints as needed
		}
	}

	logger.Logger.Info("Registered Greeter HTTP routes")
	return nil
}

// GetName implements ServiceRegistrar interface
func (sr *ServiceRegistrar) GetName() string {
	return "greeter"
}

// sayHelloHTTP handles HTTP version of SayHello
func (sr *ServiceRegistrar) sayHelloHTTP(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Language string `json:"language,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	greeting, err := sr.service.GenerateGreeting(c.Request.Context(), req.Name, req.Language)
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
