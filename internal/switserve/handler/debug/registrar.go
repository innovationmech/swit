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

package debug

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/router"
)

// DebugController 调试控制器
type DebugController struct {
	registry router.RegistryProvider
	engine   *gin.Engine
}

// NewDebugController 创建调试控制器
func NewDebugController(registry router.RegistryProvider, engine *gin.Engine) *DebugController {
	return &DebugController{
		registry: registry,
		engine:   engine,
	}
}

// GetRoutes 获取已注册的路由信息
func (dc *DebugController) GetRoutes(c *gin.Context) {
	routes := dc.registry.GetRegisteredRoutes()
	c.JSON(http.StatusOK, gin.H{
		"routes": routes,
		"count":  len(routes),
	})
}

// GetMiddlewares 获取已注册的中间件信息
func (dc *DebugController) GetMiddlewares(c *gin.Context) {
	middlewares := dc.registry.GetRegisteredMiddlewares()
	c.JSON(http.StatusOK, gin.H{
		"middlewares": middlewares,
		"count":       len(middlewares),
	})
}

// GetGinRoutes 获取Gin路由信息
func (dc *DebugController) GetGinRoutes(c *gin.Context) {
	ginRoutes := dc.engine.Routes()
	c.JSON(http.StatusOK, gin.H{
		"gin_routes": ginRoutes,
		"count":      len(ginRoutes),
	})
}

// DebugRouteRegistrar 调试路由注册器
type DebugRouteRegistrar struct {
	controller *DebugController
}

// NewDebugRouteRegistrar 创建调试路由注册器
func NewDebugRouteRegistrar(registry router.RegistryProvider, engine *gin.Engine) *DebugRouteRegistrar {
	return &DebugRouteRegistrar{
		controller: NewDebugController(registry, engine),
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (drr *DebugRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	debugGroup := rg.Group("/debug")
	{
		debugGroup.GET("/routes", drr.controller.GetRoutes)
		debugGroup.GET("/middlewares", drr.controller.GetMiddlewares)
		debugGroup.GET("/gin-routes", drr.controller.GetGinRoutes)
	}

	return nil
}

// GetName 实现 RouteRegistrar 接口
func (drr *DebugRouteRegistrar) GetName() string {
	return "debug-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (drr *DebugRouteRegistrar) GetVersion() string {
	return "root" // 调试API不需要版本前缀
}

// GetPrefix 实现 RouteRegistrar 接口
func (drr *DebugRouteRegistrar) GetPrefix() string {
	return ""
}
