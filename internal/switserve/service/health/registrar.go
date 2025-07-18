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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements unified service registration for health checks
type ServiceRegistrar struct {
	service HealthService
}

// NewServiceRegistrar creates a new health service registrar with dependency injection
func NewServiceRegistrar(service HealthService) *ServiceRegistrar {
	return &ServiceRegistrar{
		service: service,
	}
}

// NewServiceRegistrarLegacy creates a new health service registrar using the old pattern.
// Deprecated: Use NewServiceRegistrar with dependency injection instead.
func NewServiceRegistrarLegacy() *ServiceRegistrar {
	service := NewService()

	return &ServiceRegistrar{
		service: service,
	}
}

// RegisterGRPC implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	// gRPC health check can be implemented using grpc_health_v1 if needed
	// For now, we'll just log that gRPC health is available via reflection
	logger.Logger.Info("Health service available via gRPC reflection")
	return nil
}

// RegisterHTTP implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Register health check endpoint at root level
	router.GET("/health", sr.healthCheckHTTP)

	logger.Logger.Info("Registered Health HTTP routes")
	return nil
}

// GetName implements ServiceRegistrar interface
func (sr *ServiceRegistrar) GetName() string {
	return "health"
}

// healthCheckHTTP handles HTTP health check requests
func (sr *ServiceRegistrar) healthCheckHTTP(c *gin.Context) {
	status, err := sr.service.CheckHealth(c.Request.Context())
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
