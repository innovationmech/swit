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

	"github.com/gin-gonic/gin"
	httphandler "github.com/innovationmech/swit/internal/switserve/handler/http/stop"
	"github.com/innovationmech/swit/pkg/logger"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements unified service registration for server shutdown
type ServiceRegistrar struct {
	httpHandler *httphandler.Handler
}

// stopServiceAdapter adapts service layer StopService to handler layer StopService
// This is necessary to avoid circular dependency between service and handler packages
// The adapter converts between the two different ShutdownStatus types
type stopServiceAdapter struct {
	service StopService
}

func (a *stopServiceAdapter) InitiateShutdown(ctx context.Context) (*httphandler.ShutdownStatus, error) {
	status, err := a.service.InitiateShutdown(ctx)
	if err != nil {
		return nil, err
	}
	// Convert service.ShutdownStatus to handler.ShutdownStatus
	return &httphandler.ShutdownStatus{
		Status:    status.Status,
		Message:   status.Message,
		Timestamp: status.Timestamp,
	}, nil
}

// NewServiceRegistrar creates a new stop service registrar with dependency injection
func NewServiceRegistrar(service StopService) *ServiceRegistrar {
	// Use adapter to bridge service and handler interfaces
	adapter := &stopServiceAdapter{service: service}
	httpHandler := httphandler.NewHandler(adapter)
	return &ServiceRegistrar{
		httpHandler: httpHandler,
	}
}

// RegisterGRPC implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	// gRPC shutdown can be implemented as a service method if needed
	// For now, we'll just log that shutdown is available via HTTP only
	logger.Logger.Info("Stop service available via HTTP endpoint only")
	return nil
}

// RegisterHTTP implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Register stop endpoint at root level
	router.POST("/stop", sr.httpHandler.StopServer)

	logger.Logger.Info("Registered Stop HTTP routes")
	return nil
}

// GetName implements ServiceRegistrar interface
func (sr *ServiceRegistrar) GetName() string {
	return "stop"
}
