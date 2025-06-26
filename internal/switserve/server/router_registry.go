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

package server

import (
	"github.com/innovationmech/swit/pkg/logger"
	"sort"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RouteRegistrar 定义路由注册器接口
type RouteRegistrar interface {
	// RegisterRoutes 注册路由到指定的路由组
	RegisterRoutes(rg *gin.RouterGroup) error
	// GetName 获取注册器名称，用于日志和调试
	GetName() string
	// GetVersion 获取API版本
	GetVersion() string
	// GetPrefix 获取路由前缀
	GetPrefix() string
}

// MiddlewareRegistrar 定义中间件注册器接口
type MiddlewareRegistrar interface {
	// RegisterMiddleware 注册中间件到路由器
	RegisterMiddleware(router *gin.Engine) error
	// GetName 获取中间件名称
	GetName() string
	// GetPriority 获取中间件优先级（数字越小优先级越高）
	GetPriority() int
}

// RouteRegistry 路由注册表管理器
type RouteRegistry struct {
	routeRegistrars      []RouteRegistrar
	middlewareRegistrars []MiddlewareRegistrar
}

// NewRouteRegistry 创建新的路由注册表
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		routeRegistrars:      make([]RouteRegistrar, 0),
		middlewareRegistrars: make([]MiddlewareRegistrar, 0),
	}
}

// RegisterRoute 注册路由注册器
func (rr *RouteRegistry) RegisterRoute(registrar RouteRegistrar) {
	logger.Logger.Info("Registering route",
		zap.String("name", registrar.GetName()),
		zap.String("version", registrar.GetVersion()),
		zap.String("prefix", registrar.GetPrefix()))
	rr.routeRegistrars = append(rr.routeRegistrars, registrar)
}

// RegisterMiddleware 注册中间件注册器
func (rr *RouteRegistry) RegisterMiddleware(registrar MiddlewareRegistrar) {
	logger.Logger.Info("Registering middleware",
		zap.String("name", registrar.GetName()),
		zap.Int("priority", registrar.GetPriority()))
	rr.middlewareRegistrars = append(rr.middlewareRegistrars, registrar)
}

// Setup 设置所有路由和中间件
func (rr *RouteRegistry) Setup(router *gin.Engine) error {
	// 按优先级排序并注册中间件
	if err := rr.setupMiddlewares(router); err != nil {
		return err
	}

	// 注册路由
	if err := rr.setupRoutes(router); err != nil {
		return err
	}

	logger.Logger.Info("All routes and middlewares registered successfully",
		zap.Int("route_count", len(rr.routeRegistrars)),
		zap.Int("middleware_count", len(rr.middlewareRegistrars)))

	return nil
}

// setupMiddlewares 设置中间件
func (rr *RouteRegistry) setupMiddlewares(router *gin.Engine) error {
	// 按优先级排序
	sort.Slice(rr.middlewareRegistrars, func(i, j int) bool {
		return rr.middlewareRegistrars[i].GetPriority() < rr.middlewareRegistrars[j].GetPriority()
	})

	// 注册中间件
	for _, registrar := range rr.middlewareRegistrars {
		if err := registrar.RegisterMiddleware(router); err != nil {
			logger.Logger.Error("Failed to register middleware",
				zap.String("name", registrar.GetName()),
				zap.Error(err))
			return err
		}
	}
	return nil
}

// setupRoutes 设置路由
func (rr *RouteRegistry) setupRoutes(router *gin.Engine) error {
	// 按版本分组路由
	versionGroups := make(map[string]*gin.RouterGroup)

	for _, registrar := range rr.routeRegistrars {
		version := registrar.GetVersion()
		if version == "" {
			logger.Logger.Warn("Version is missing, defaulting to 'v1'",
				zap.String("name", registrar.GetName()))
			version = "v1" // 默认版本
		}

		// 创建版本路由组
		if _, exists := versionGroups[version]; !exists {
			if version == "root" {
				// 对于root版本，直接使用router的Group方法创建一个根级路由组
				versionGroups[version] = router.Group("")
			} else {
				versionGroups[version] = router.Group("/" + version)
			}
		}

		// 创建带前缀的路由组
		var routeGroup *gin.RouterGroup
		prefix := registrar.GetPrefix()
		if prefix != "" {
			routeGroup = versionGroups[version].Group("/" + prefix)
		} else {
			routeGroup = versionGroups[version]
		}

		// 注册路由
		if err := registrar.RegisterRoutes(routeGroup); err != nil {
			logger.Logger.Error("Failed to register routes",
				zap.String("name", registrar.GetName()),
				zap.String("version", version),
				zap.String("prefix", prefix),
				zap.Error(err))
			return err
		}

		logger.Logger.Info("Routes registered successfully",
			zap.String("name", registrar.GetName()),
			zap.String("version", version),
			zap.String("prefix", prefix))
	}

	return nil
}

// GetRegisteredRoutes 获取已注册的路由信息（用于调试和监控）
func (rr *RouteRegistry) GetRegisteredRoutes() []map[string]interface{} {
	routes := make([]map[string]interface{}, 0)

	for _, registrar := range rr.routeRegistrars {
		routes = append(routes, map[string]interface{}{
			"name":    registrar.GetName(),
			"version": registrar.GetVersion(),
			"prefix":  registrar.GetPrefix(),
		})
	}

	return routes
}

func (rr *RouteRegistry) GetRegisteredMiddlewares() []map[string]interface{} {
	middlewares := make([]map[string]interface{}, 0)

	for _, registrar := range rr.middlewareRegistrars {
		middlewares = append(middlewares, map[string]interface{}{
			"name":     registrar.GetName(),
			"priority": registrar.GetPriority(),
		})
	}

	return middlewares
}
