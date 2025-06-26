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
	"github.com/innovationmech/swit/internal/pkg/logger"
	"github.com/innovationmech/swit/internal/switserve/controller/debug"
	"github.com/innovationmech/swit/internal/switserve/controller/health"
	"github.com/innovationmech/swit/internal/switserve/controller/stop"
	"github.com/innovationmech/swit/internal/switserve/controller/v1/user"
	"github.com/innovationmech/swit/internal/switserve/middleware"
	"go.uber.org/zap"
)

// SetupRoutes sets up the routes for the application using the new route registry system.
func (s *Server) SetupRoutes() {
	// 创建路由注册表
	registry := NewRouteRegistry()

	// 注册全局中间件
	registry.RegisterMiddleware(middleware.NewGlobalMiddlewareRegistrar())

	// 注册路由
	registry.RegisterRoute(health.NewHealthRouteRegistrar())
	registry.RegisterRoute(stop.NewStopRouteRegistrar(s.Shutdown))
	registry.RegisterRoute(user.NewUserRouteRegistrar())
	registry.RegisterRoute(user.NewUserInternalRouteRegistrar())
	// 注册调试路由（仅在开发环境）
	if gin.Mode() == gin.DebugMode {
		registry.RegisterRoute(debug.NewDebugRouteRegistrar(registry, s.router))
	}

	// 设置所有路由和中间件
	if err := registry.Setup(s.router); err != nil {
		logger.Logger.Fatal("Failed to setup routes", zap.Error(err))
	}

	logger.Logger.Info("Route registry setup completed")
}
