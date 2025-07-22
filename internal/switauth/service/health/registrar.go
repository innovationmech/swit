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

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/innovationmech/swit/internal/switauth/transport"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ServiceRegistrar implements the ServiceRegistrar interface for health services
type ServiceRegistrar struct {
	service Service
}

// NewServiceRegistrar creates a new health service registrar with dependency injection
func NewServiceRegistrar(healthSrv Service) *ServiceRegistrar {
	return &ServiceRegistrar{
		service: healthSrv,
	}
}

// RegisterGRPC implements the ServiceRegistrar interface for gRPC
// Registers the gRPC health check service
func (h *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	// Register the gRPC health service
	healthImpl := &healthGRPCService{service: h.service}
	grpc_health_v1.RegisterHealthServer(server, healthImpl)

	logger.Logger.Info("Registered Health gRPC service")
	return nil
}

// RegisterHTTP implements the ServiceRegistrar interface for HTTP
// Registers the HTTP health check endpoints
func (h *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Register health check endpoints
	router.GET("/health", h.healthCheck)
	router.GET("/health/detailed", h.healthCheckDetailed)

	logger.Logger.Info("Registered Health HTTP routes",
		zap.String("paths", "/health, /health/detailed"))
	return nil
}

// GetName returns the service name
func (h *ServiceRegistrar) GetName() string {
	return "health"
}

// Ensure ServiceRegistrar implements transport.ServiceRegistrar
var _ transport.ServiceRegistrar = (*ServiceRegistrar)(nil)

// HTTP handlers
func (h *ServiceRegistrar) healthCheck(c *gin.Context) {
	status := h.service.CheckHealth(c.Request.Context())
	c.JSON(200, status)
}

func (h *ServiceRegistrar) healthCheckDetailed(c *gin.Context) {
	details := h.service.GetHealthDetails(c.Request.Context())
	c.JSON(200, details)
}

// gRPC service implementation

// healthGRPCService implements the gRPC health service
type healthGRPCService struct {
	service Service
}

// Check implements the gRPC health check
func (h *healthGRPCService) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	status := h.service.CheckHealth(ctx)

	var servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus
	if status.Status == "healthy" {
		servingStatus = grpc_health_v1.HealthCheckResponse_SERVING
	} else {
		servingStatus = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: servingStatus,
	}, nil
}

// Watch implements the gRPC health watch (streaming health checks)
func (h *healthGRPCService) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// Simple implementation - always return SERVING
	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}

// List implements the gRPC health list service method
func (h *healthGRPCService) List(ctx context.Context, req *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	// Return the list of services and their health status
	return &grpc_health_v1.HealthListResponse{
		Statuses: map[string]*grpc_health_v1.HealthCheckResponse{
			"": {
				Status: grpc_health_v1.HealthCheckResponse_SERVING,
			},
		},
	}, nil
}
