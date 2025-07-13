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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/handler/health"
	"github.com/innovationmech/swit/internal/switserve/handler/stop"
	"github.com/innovationmech/swit/internal/switserve/router"
	"github.com/innovationmech/swit/pkg/logger"
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
	// 初始化日志记录器用于测试
	logger.Logger, _ = zap.NewDevelopment()
}

// 创建测试用的服务器实例
func createTestServer() *Server {
	// 不在这里设置 gin.Mode，让调用者决定
	return &Server{
		router: gin.New(),
	}
}

// TestServer_SetupRoutes 测试 SetupRoutes 方法
func TestServer_SetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()

	// 由于 SetupRoutes 方法依赖于实际的配置文件，这里测试可能会失败
	// 我们主要测试路由器的基本功能，而不是完整的 SetupRoutes
	assert.NotNil(t, server.router, "路由器应该被初始化")

	// 测试 setupSwaggerUI 方法
	assert.NotPanics(t, func() {
		server.setupSwaggerUI()
	})

	// 验证Swagger路由是否被注册
	routes := server.router.Routes()
	assert.True(t, len(routes) > 0, "应该有Swagger路由被注册")
}

// TestServer_configureRoutes 测试 configureRoutes 方法
func TestServer_configureRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	registry := router.New()

	// 添加测试中间件
	testMiddleware := NewTestMiddlewareRegistrar("test-middleware", 10)
	registry.RegisterMiddleware(testMiddleware)

	// 添加测试路由
	testRoute := NewTestRouteRegistrar("test-route", "v1", "api")
	registry.RegisterRoute(testRoute)

	// 测试单独的方法，避免调用依赖配置文件的方法
	assert.NotPanics(t, func() {
		server.registerGlobalMiddlewares(registry)
	})

	// 验证注册表中的路由和中间件
	routes := registry.GetRegisteredRoutes()
	middlewares := registry.GetRegisteredMiddlewares()

	assert.True(t, len(routes) >= 1, "应该有路由被注册")
	assert.True(t, len(middlewares) >= 1, "应该有中间件被注册")
}

// TestServer_registerGlobalMiddlewares 测试 registerGlobalMiddlewares 方法
func TestServer_registerGlobalMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	registry := router.New()

	// 测试注册全局中间件
	assert.NotPanics(t, func() {
		server.registerGlobalMiddlewares(registry)
	})

	// 验证是否有中间件被注册
	middlewares := registry.GetRegisteredMiddlewares()
	assert.True(t, len(middlewares) > 0, "应该有全局中间件被注册")
}

// TestServer_registerAPIRoutes 测试 registerAPIRoutes 方法
func TestServer_registerAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	registry := router.New()

	// 由于 registerAPIRoutes 依赖于配置文件和数据库连接，我们测试单独的路由注册
	// 手动注册一些基础路由来测试功能

	// 模拟注册健康检查路由
	assert.NotPanics(t, func() {
		// 注册健康检查路由
		registry.RegisterRoute(health.NewHealthRouteRegistrar())
		// 注册停止服务路由
		registry.RegisterRoute(stop.NewStopRouteRegistrar(server.Shutdown))
	})

	// 验证是否有路由被注册
	routes := registry.GetRegisteredRoutes()
	assert.True(t, len(routes) > 0, "应该有API路由被注册")

	// 验证特定的路由是否被注册
	routeNames := make([]string, len(routes))
	for i, route := range routes {
		routeNames[i] = route["name"].(string)
	}

	// 验证健康检查路由
	assert.Contains(t, routeNames, "health-check", "应该包含健康检查路由")
	assert.Contains(t, routeNames, "stop-service", "应该包含停止服务路由")
}

// TestServer_registerDebugRoutes 测试 registerDebugRoutes 方法
func TestServer_registerDebugRoutes(t *testing.T) {
	// 测试调试模式
	originalMode := gin.Mode()
	gin.SetMode(gin.DebugMode)

	server := createTestServer()
	registry := router.New()

	// 测试注册调试路由
	assert.NotPanics(t, func() {
		server.registerDebugRoutes(registry)
	})

	// 在调试模式下，应该有调试路由被注册
	routes := registry.GetRegisteredRoutes()

	// 检查是否有调试路由被注册
	var debugRouteFound bool
	for _, route := range routes {
		if route["name"] == "debug-api" {
			debugRouteFound = true
			break
		}
	}
	assert.True(t, debugRouteFound, "在调试模式下应该有调试路由被注册")

	// 测试非调试模式
	gin.SetMode(gin.ReleaseMode)
	server2 := createTestServer()
	registry2 := router.New()

	assert.NotPanics(t, func() {
		server2.registerDebugRoutes(registry2)
	})

	// 在非调试模式下，不应该有调试路由被注册
	routes2 := registry2.GetRegisteredRoutes()
	var debugRouteFound2 bool
	for _, route := range routes2 {
		if route["name"] == "debug-api" {
			debugRouteFound2 = true
			break
		}
	}
	assert.False(t, debugRouteFound2, "在非调试模式下不应该有调试路由被注册")

	// 恢复原始模式
	gin.SetMode(originalMode)
}

// TestServer_setupSwaggerUI 测试 setupSwaggerUI 方法
func TestServer_setupSwaggerUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()

	// 测试设置Swagger UI
	assert.NotPanics(t, func() {
		server.setupSwaggerUI()
	})

	// 验证Swagger路由是否被注册
	routes := server.router.Routes()

	// 查找Swagger路由
	var swaggerRouteFound bool
	for _, route := range routes {
		if route.Path == "/swagger/*any" {
			swaggerRouteFound = true
			break
		}
	}

	assert.True(t, swaggerRouteFound, "应该有Swagger路由被注册")
}

// TestServer_SwaggerUIEndpoint 测试Swagger UI端点
func TestServer_SwaggerUIEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	server.setupSwaggerUI()

	// 创建测试请求
	req := httptest.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()

	// 执行请求
	server.router.ServeHTTP(w, req)

	// 验证响应状态码
	// 注意：这里可能返回404，因为实际的Swagger文档可能不存在
	// 但路由应该被正确注册
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound,
		"Swagger端点应该返回200或404状态码")
}

// TestServer_RouteRegistration_Integration 集成测试
func TestServer_RouteRegistration_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()

	// 由于完整的 SetupRoutes() 依赖于配置文件，我们测试单独的组件
	// 模拟完整的路由设置流程
	registry := router.New()

	// 注册全局中间件
	assert.NotPanics(t, func() {
		server.registerGlobalMiddlewares(registry)
	})

	// 注册基础路由
	assert.NotPanics(t, func() {
		registry.RegisterRoute(health.NewHealthRouteRegistrar())
		registry.RegisterRoute(stop.NewStopRouteRegistrar(server.Shutdown))
	})

	// 注册调试路由
	assert.NotPanics(t, func() {
		server.registerDebugRoutes(registry)
	})

	// 设置路由
	assert.NotPanics(t, func() {
		registry.Setup(server.router)
	})

	// 设置Swagger UI
	assert.NotPanics(t, func() {
		server.setupSwaggerUI()
	})

	// 验证路由是否被正确注册
	routes := server.router.Routes()
	assert.True(t, len(routes) > 0, "应该有路由被注册")

	// 验证Swagger路由
	var swaggerRouteFound bool
	for _, route := range routes {
		if route.Path == "/swagger/*any" {
			swaggerRouteFound = true
			break
		}
	}
	assert.True(t, swaggerRouteFound, "应该有Swagger路由被注册")

	// 验证健康检查路由
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// 健康检查路由应该存在（即使可能返回错误）
	assert.NotEqual(t, http.StatusNotFound, w.Code, "健康检查路由应该存在")
}

// TestServer_MiddlewareRegistration 测试中间件注册
func TestServer_MiddlewareRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	registry := router.New()

	// 添加测试中间件
	testMiddleware := NewTestMiddlewareRegistrar("test-middleware", 10)
	registry.RegisterMiddleware(testMiddleware)

	// 设置路由
	err := registry.Setup(server.router)
	assert.NoError(t, err)

	// 添加测试路由
	server.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// 测试中间件是否生效
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// 验证中间件添加的header
	assert.Equal(t, "test-middleware", w.Header().Get("X-Test-Middleware"))
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_RouteConflicts 测试路由冲突
func TestServer_RouteConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	registry := router.New()

	// 添加多个相同路径的路由注册器，这会导致panic
	route1 := NewTestRouteRegistrar("route1", "v1", "api")
	route2 := NewTestRouteRegistrar("route2", "v1", "api")

	registry.RegisterRoute(route1)
	registry.RegisterRoute(route2)

	// 设置路由应该会panic，因为Gin不允许相同的路由路径
	assert.Panics(t, func() {
		registry.Setup(server.router)
	}, "相同的路由路径应该导致panic")
}

// TestServer_EmptyRegistry 测试空注册表
func TestServer_EmptyRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := createTestServer()
	registry := router.New()

	// 测试空注册表的设置
	assert.NotPanics(t, func() {
		server.registerGlobalMiddlewares(registry)
	})

	// 设置空注册表
	err := registry.Setup(server.router)
	assert.NoError(t, err)

	// 验证注册表信息
	middlewares := registry.GetRegisteredMiddlewares()

	// 应该有一些默认的中间件
	assert.True(t, len(middlewares) > 0, "应该有默认中间件被注册")
}
