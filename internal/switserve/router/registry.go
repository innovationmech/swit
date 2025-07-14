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

package router

import (
	"sort"
	"sync"

	"github.com/innovationmech/swit/pkg/logger"

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

// RegistryProvider 路由注册表提供者接口（用于调试和监控）
type RegistryProvider interface {
	GetRegisteredRoutes() []map[string]interface{}
	GetRegisteredMiddlewares() []map[string]interface{}
}

// Registry 路由注册表管理器
type Registry struct {
	mu                   sync.RWMutex
	routeRegistrars      []RouteRegistrar
	middlewareRegistrars []MiddlewareRegistrar
}

// New 创建新的路由注册表
func New() *Registry {
	return &Registry{
		routeRegistrars:      make([]RouteRegistrar, 0),
		middlewareRegistrars: make([]MiddlewareRegistrar, 0),
	}
}

// RegisterRoute 注册路由注册器
func (r *Registry) RegisterRoute(registrar RouteRegistrar) {
	logger.Logger.Info("Registering route",
		zap.String("name", registrar.GetName()),
		zap.String("version", registrar.GetVersion()),
		zap.String("prefix", registrar.GetPrefix()))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routeRegistrars = append(r.routeRegistrars, registrar)
}

// RegisterMiddleware 注册中间件注册器
func (r *Registry) RegisterMiddleware(registrar MiddlewareRegistrar) {
	logger.Logger.Info("Registering middleware",
		zap.String("name", registrar.GetName()),
		zap.Int("priority", registrar.GetPriority()))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewareRegistrars = append(r.middlewareRegistrars, registrar)
}

// Setup 设置所有路由和中间件
func (r *Registry) Setup(router *gin.Engine) error {
	// 按优先级排序并注册中间件
	if err := r.setupMiddlewares(router); err != nil {
		return err
	}

	// 注册路由
	if err := r.setupRoutes(router); err != nil {
		return err
	}

	r.mu.RLock()
	routeCount := len(r.routeRegistrars)
	middlewareCount := len(r.middlewareRegistrars)
	r.mu.RUnlock()

	logger.Logger.Info("All routes and middlewares registered successfully",
		zap.Int("route_count", routeCount),
		zap.Int("middleware_count", middlewareCount))

	return nil
}

// setupMiddlewares 设置中间件
func (r *Registry) setupMiddlewares(router *gin.Engine) error {
	r.mu.Lock()
	// 按优先级排序
	sort.Slice(r.middlewareRegistrars, func(i, j int) bool {
		return r.middlewareRegistrars[i].GetPriority() < r.middlewareRegistrars[j].GetPriority()
	})
	// 创建副本以避免在解锁后访问
	middlewares := make([]MiddlewareRegistrar, len(r.middlewareRegistrars))
	copy(middlewares, r.middlewareRegistrars)
	r.mu.Unlock()

	// 注册中间件
	for _, registrar := range middlewares {
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
func (r *Registry) setupRoutes(router *gin.Engine) error {
	r.mu.RLock()
	// 创建副本以避免在解锁后访问
	routes := make([]RouteRegistrar, len(r.routeRegistrars))
	copy(routes, r.routeRegistrars)
	r.mu.RUnlock()

	// 按版本分组路由
	versionGroups := make(map[string]*gin.RouterGroup)

	for _, registrar := range routes {
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
func (r *Registry) GetRegisteredRoutes() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]map[string]interface{}, 0)

	for _, registrar := range r.routeRegistrars {
		routes = append(routes, map[string]interface{}{
			"name":    registrar.GetName(),
			"version": registrar.GetVersion(),
			"prefix":  registrar.GetPrefix(),
		})
	}

	return routes
}

// GetRegisteredMiddlewares 获取已注册的中间件信息（用于调试和监控）
func (r *Registry) GetRegisteredMiddlewares() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	middlewares := make([]map[string]interface{}, 0)

	for _, registrar := range r.middlewareRegistrars {
		middlewares = append(middlewares, map[string]interface{}{
			"name":     registrar.GetName(),
			"priority": registrar.GetPriority(),
		})
	}

	return middlewares
}
