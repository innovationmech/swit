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

package user

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/db"
	v2 "github.com/innovationmech/swit/internal/switserve/handler/http/user/v1"
	"github.com/innovationmech/swit/internal/switserve/repository"
	v1 "github.com/innovationmech/swit/internal/switserve/service/user/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceRegistrar implements unified service registration for user management
type ServiceRegistrar struct {
	controller *v2.Controller
	userSrv    v1.UserSrv
}

// NewServiceRegistrar creates a new user service registrar
func NewServiceRegistrar() *ServiceRegistrar {
	// Create user service with repository
	userSrv, err := v1.NewUserSrv(
		v1.WithUserRepository(repository.NewUserRepository(db.GetDB())),
	)
	if err != nil {
		logger.Logger.Error("failed to create user service", zap.Error(err))
		return nil
	}

	// Create controller
	controller := v2.NewUserController()

	return &ServiceRegistrar{
		controller: controller,
		userSrv:    userSrv,
	}
}

// RegisterGRPC implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	// gRPC user service can be implemented when proto definitions are available
	// For now, we'll just log that user service is available via HTTP
	logger.Logger.Info("User service available via HTTP endpoints")
	return nil
}

// RegisterHTTP implements ServiceRegistrar interface
func (sr *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	// Create API v1 group
	v1Group := router.Group("/api/v1")
	{
		// Public user endpoints (with auth middleware and whitelist)
		userGroup := v1Group.Group("/users")
		authWhiteList := []string{"/users/create", "/health", "/stop"}
		userGroup.Use(middleware.AuthMiddlewareWithWhiteList(authWhiteList))
		{
			userGroup.POST("/create", sr.controller.CreateUser)
			userGroup.GET("/username/:username", sr.controller.GetUserByUsername)
			userGroup.GET("/email/:email", sr.controller.GetUserByEmail)
			userGroup.DELETE("/:id", sr.controller.DeleteUser)
		}

		// Internal user endpoints (no auth required)
		internalGroup := v1Group.Group("/internal")
		{
			internalGroup.POST("/validate-user", sr.controller.ValidateUserCredentials)
		}
	}

	logger.Logger.Info("Registered User HTTP routes")
	return nil
}

// GetName implements ServiceRegistrar interface
func (sr *ServiceRegistrar) GetName() string {
	return "user"
}
