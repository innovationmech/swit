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

package server

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/handler"
	"github.com/innovationmech/swit/internal/switauth/router"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	// Import swagger docs
	_ "github.com/innovationmech/swit/internal/switauth/docs"
)

// SetupRoutes 使用新的路由注册系统设置路由
func (s *Server) SetupRoutes(authController *handler.AuthController) {
	// 创建路由注册表
	registry := router.New()

	// 配置应用路由
	s.configureRoutes(registry, authController)

	// 设置所有路由和中间件
	if err := registry.Setup(s.router); err != nil {
		logger.Logger.Fatal("Failed to setup routes", zap.Error(err))
	}

	// 设置 Swagger UI
	s.setupSwaggerUI()

	logger.Logger.Info("Route registry setup completed")
}

// configureRoutes 配置应用的所有路由和中间件
func (s *Server) configureRoutes(registry *router.Registry, authController *handler.AuthController) {
	// 注册全局中间件
	s.registerGlobalMiddlewares(registry)

	// 注册API路由
	s.registerAPIRoutes(registry, authController)
}

// registerGlobalMiddlewares 注册全局中间件
func (s *Server) registerGlobalMiddlewares(registry *router.Registry) {
	registry.RegisterMiddleware(middleware.NewGlobalMiddlewareRegistrar())
}

// registerAPIRoutes 注册API路由
func (s *Server) registerAPIRoutes(registry *router.Registry, authController *handler.AuthController) {
	// 健康检查路由
	registry.RegisterRoute(handler.NewHealthRouteRegistrar())

	// 认证相关路由
	registry.RegisterRoute(handler.NewAuthRouteRegistrar(authController))
}

// setupSwaggerUI 设置Swagger UI
func (s *Server) setupSwaggerUI() {
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// RegisterRoutes 保持向后兼容性的旧方法
// Deprecated: 使用 SetupRoutes 方法代替
func RegisterRoutes(authController *handler.AuthController) *gin.Engine {
	r := gin.Default()

	server := &Server{
		router: r,
	}

	server.SetupRoutes(authController)

	return r
}
