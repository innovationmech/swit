// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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
	"github.com/innovationmech/swit/internal/switserve/handler/debug"
	"github.com/innovationmech/swit/internal/switserve/handler/health"
	"github.com/innovationmech/swit/internal/switserve/handler/stop"
	"github.com/innovationmech/swit/internal/switserve/handler/v1/user"
	"github.com/innovationmech/swit/internal/switserve/middleware"
	"github.com/innovationmech/swit/internal/switserve/router"
	"github.com/innovationmech/swit/pkg/logger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	// Import swagger docs
	_ "github.com/innovationmech/swit/internal/switserve/docs"
)

// SetupRoutes sets up the routes for the application using the new route registry system.
func (s *Server) SetupRoutes() {
	// 创建路由注册表
	registry := router.New()

	// 配置应用路由
	s.configureRoutes(registry)

	// 设置所有路由和中间件
	if err := registry.Setup(s.router); err != nil {
		logger.Logger.Fatal("Failed to setup routes", zap.Error(err))
	}

	// 设置 Swagger UI
	s.setupSwaggerUI()

	logger.Logger.Info("Route registry setup completed")
}

// configureRoutes 配置应用的所有路由和中间件
func (s *Server) configureRoutes(registry *router.Registry) {
	// 注册全局中间件
	s.registerGlobalMiddlewares(registry)

	// 注册API路由
	s.registerAPIRoutes(registry)

	// 注册调试路由（仅在开发环境）
	s.registerDebugRoutes(registry)
}

// registerGlobalMiddlewares 注册全局中间件
func (s *Server) registerGlobalMiddlewares(registry *router.Registry) {
	registry.RegisterMiddleware(middleware.NewGlobalMiddlewareRegistrar())
}

// registerAPIRoutes 注册API路由
func (s *Server) registerAPIRoutes(registry *router.Registry) {
	// 基础服务路由
	registry.RegisterRoute(health.NewHealthRouteRegistrar())
	registry.RegisterRoute(stop.NewStopRouteRegistrar(s.Shutdown))

	// 用户相关路由
	registry.RegisterRoute(user.NewUserRouteRegistrar())
	registry.RegisterRoute(user.NewUserInternalRouteRegistrar())
}

// registerDebugRoutes 注册调试路由（仅在开发环境）
func (s *Server) registerDebugRoutes(registry *router.Registry) {
	if gin.Mode() == gin.DebugMode {
		registry.RegisterRoute(debug.NewDebugRouteRegistrar(registry, s.router))
	}
}

// setupSwaggerUI 设置Swagger UI
func (s *Server) setupSwaggerUI() {
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
