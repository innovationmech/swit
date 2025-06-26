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
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestRouteRegistrar 测试路由注册器
type TestRouteRegistrar struct {
	name    string
	version string
	prefix  string
}

func NewTestRouteRegistrar(name, version, prefix string) *TestRouteRegistrar {
	return &TestRouteRegistrar{
		name:    name,
		version: version,
		prefix:  prefix,
	}
}

func (trr *TestRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	rg.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
	return nil
}

func (trr *TestRouteRegistrar) GetName() string {
	return trr.name
}

func (trr *TestRouteRegistrar) GetVersion() string {
	return trr.version
}

func (trr *TestRouteRegistrar) GetPrefix() string {
	return trr.prefix
}

// TestMiddlewareRegistrar 测试中间件注册器
type TestMiddlewareRegistrar struct {
	name     string
	priority int
}

func NewTestMiddlewareRegistrar(name string, priority int) *TestMiddlewareRegistrar {
	return &TestMiddlewareRegistrar{
		name:     name,
		priority: priority,
	}
}

func (tmr *TestMiddlewareRegistrar) RegisterMiddleware(router *gin.Engine) error {
	router.Use(func(c *gin.Context) {
		c.Header("X-Test-Middleware", tmr.name)
		c.Next()
	})
	return nil
}

func (tmr *TestMiddlewareRegistrar) GetName() string {
	return tmr.name
}

func (tmr *TestMiddlewareRegistrar) GetPriority() int {
	return tmr.priority
}

func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

func TestRouteRegistry_RegisterRoute(t *testing.T) {
	registry := NewRouteRegistry()
	
	registrar := NewTestRouteRegistrar("test-api", "v1", "")
	registry.RegisterRoute(registrar)
	
	assert.Len(t, registry.routeRegistrars, 1)
	assert.Equal(t, "test-api", registry.routeRegistrars[0].GetName())
}

func TestRouteRegistry_RegisterMiddleware(t *testing.T) {
	registry := NewRouteRegistry()
	
	registrar := NewTestMiddlewareRegistrar("test-middleware", 10)
	registry.RegisterMiddleware(registrar)
	
	assert.Len(t, registry.middlewareRegistrars, 1)
	assert.Equal(t, "test-middleware", registry.middlewareRegistrars[0].GetName())
}

func TestRouteRegistry_Setup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registry := NewRouteRegistry()
	
	// 注册中间件
	middleware1 := NewTestMiddlewareRegistrar("middleware-1", 20)
	middleware2 := NewTestMiddlewareRegistrar("middleware-2", 10)
	registry.RegisterMiddleware(middleware1)
	registry.RegisterMiddleware(middleware2)
	
	// 注册路由
	route1 := NewTestRouteRegistrar("api-v1", "v1", "")
	route2 := NewTestRouteRegistrar("api-v2", "v2", "")
	route3 := NewTestRouteRegistrar("root-api", "root", "")
	registry.RegisterRoute(route1)
	registry.RegisterRoute(route2)
	registry.RegisterRoute(route3)
	
	err := registry.Setup(router)
	assert.NoError(t, err)
	
	// 验证路由是否正确注册
	routes := router.Routes()
	assert.True(t, len(routes) > 0)
	
	// 验证路由路径
	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.Path
	}
	
	assert.Contains(t, routePaths, "/v1/test")  // v1版本路由
	assert.Contains(t, routePaths, "/v2/test")  // v2版本路由
	assert.Contains(t, routePaths, "/test")     // root版本路由
}

func TestRouteRegistry_GetRegisteredRoutes(t *testing.T) {
	registry := NewRouteRegistry()
	
	registrar1 := NewTestRouteRegistrar("api-1", "v1", "prefix1")
	registrar2 := NewTestRouteRegistrar("api-2", "v2", "prefix2")
	registry.RegisterRoute(registrar1)
	registry.RegisterRoute(registrar2)
	
	routes := registry.GetRegisteredRoutes()
	assert.Len(t, routes, 2)
	
	assert.Equal(t, "api-1", routes[0]["name"])
	assert.Equal(t, "v1", routes[0]["version"])
	assert.Equal(t, "prefix1", routes[0]["prefix"])
	
	assert.Equal(t, "api-2", routes[1]["name"])
	assert.Equal(t, "v2", routes[1]["version"])
	assert.Equal(t, "prefix2", routes[1]["prefix"])
}

func TestRouteRegistry_GetRegisteredMiddlewares(t *testing.T) {
	registry := NewRouteRegistry()
	
	middleware1 := NewTestMiddlewareRegistrar("middleware-1", 10)
	middleware2 := NewTestMiddlewareRegistrar("middleware-2", 20)
	registry.RegisterMiddleware(middleware1)
	registry.RegisterMiddleware(middleware2)
	
	middlewares := registry.GetRegisteredMiddlewares()
	assert.Len(t, middlewares, 2)
	
	assert.Equal(t, "middleware-1", middlewares[0]["name"])
	assert.Equal(t, 10, middlewares[0]["priority"])
	
	assert.Equal(t, "middleware-2", middlewares[1]["name"])
	assert.Equal(t, 20, middlewares[1]["priority"])
}

func TestMiddlewarePrioritySort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registry := NewRouteRegistry()
	
	// 以错误的顺序注册中间件
	middleware3 := NewTestMiddlewareRegistrar("middleware-3", 30)
	middleware1 := NewTestMiddlewareRegistrar("middleware-1", 10)
	middleware2 := NewTestMiddlewareRegistrar("middleware-2", 20)
	
	registry.RegisterMiddleware(middleware3)
	registry.RegisterMiddleware(middleware1)
	registry.RegisterMiddleware(middleware2)
	
	err := registry.Setup(router)
	assert.NoError(t, err)
	
	// 验证中间件是否按优先级正确排序
	middlewares := registry.GetRegisteredMiddlewares()
	assert.Equal(t, "middleware-1", middlewares[0]["name"]) // 优先级10，应该第一
	assert.Equal(t, "middleware-2", middlewares[1]["name"]) // 优先级20，应该第二
	assert.Equal(t, "middleware-3", middlewares[2]["name"]) // 优先级30，应该第三
}
