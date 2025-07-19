// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
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

package auth

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switauth/client"
	v1handler "github.com/innovationmech/swit/internal/switauth/handler/http/auth/v1"
	"github.com/innovationmech/swit/internal/switauth/repository"
	"github.com/innovationmech/swit/internal/switauth/service/auth/v1"
	"github.com/innovationmech/swit/internal/switauth/transport"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"go.uber.org/zap"
)

// ServiceRegistrar implements the ServiceRegistrar interface for authentication services
type ServiceRegistrar struct {
	service    v1.AuthSrv
	controller *v1handler.Controller
}

// NewServiceRegistrar creates a new authentication service registrar with dependency injection
func NewServiceRegistrar(authSrv v1.AuthSrv) *ServiceRegistrar {
	controller := v1handler.NewAuthController(authSrv)

	return &ServiceRegistrar{
		service:    authSrv,
		controller: controller,
	}
}

// NewServiceRegistrarWithDeps creates a new authentication service registrar with dependencies (legacy)
// Deprecated: Use NewServiceRegistrar with pre-constructed AuthSrv instead
func NewServiceRegistrarWithDeps(userClient client.UserClient, tokenRepo repository.TokenRepository) (*ServiceRegistrar, error) {
	service, err := v1.NewAuthSrv(
		v1.WithUserClient(userClient),
		v1.WithTokenRepository(tokenRepo),
	)
	if err != nil {
		return nil, err
	}

	controller := v1handler.NewAuthController(service)

	return &ServiceRegistrar{
		service:    service,
		controller: controller,
	}, nil
}

// RegisterGRPC implements the ServiceRegistrar interface for gRPC
// Currently no gRPC services are implemented for authentication
// This is a placeholder for future gRPC authentication service
func (a *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	// TODO: Implement gRPC authentication service when protobuf definitions are available
	// For now, this is a placeholder
	logger.Logger.Info("Authentication gRPC service registration skipped (not implemented)")
	return nil
}

// RegisterHTTP implements the ServiceRegistrar interface for HTTP
// Registers all authentication-related HTTP routes
func (a *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Create rate limiter for authentication endpoints
	authRateLimiter := middleware.NewAuthRateLimiter()

	// Create API v1 group
	v1Group := router.Group("/api/v1")

	// Register authentication routes
	authGroup := v1Group.Group("/auth")
	{
		// Apply rate limiting to sensitive endpoints
		authGroup.POST("/login", authRateLimiter.Middleware(), a.controller.Login)
		authGroup.POST("/logout", a.controller.Logout)
		authGroup.POST("/refresh", authRateLimiter.Middleware(), a.controller.RefreshToken)
		authGroup.GET("/validate", a.controller.ValidateToken)
	}

	logger.Logger.Info("Registered Authentication HTTP routes with rate limiting",
		zap.String("base_path", "/api/v1/auth"))
	return nil
}

// GetName returns the service name
func (a *ServiceRegistrar) GetName() string {
	return "auth"
}

// GetService returns the underlying authentication service
func (a *ServiceRegistrar) GetService() v1.AuthSrv {
	return a.service
}

// Ensure ServiceRegistrar implements transport.ServiceRegistrar
var _ transport.ServiceRegistrar = (*ServiceRegistrar)(nil)
