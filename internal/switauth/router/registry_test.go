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
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	// 在测试中初始化logger为测试模式
	logger.Logger = zap.NewNop()
}

// MockRouteRegistrar 路由注册器的mock实现
type MockRouteRegistrar struct {
	name            string
	version         string
	prefix          string
	errorOnRegister bool
}

func (m *MockRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	if m.errorOnRegister {
		return errors.New("mock register error")
	}
	return nil
}

func (m *MockRouteRegistrar) GetName() string {
	return m.name
}

func (m *MockRouteRegistrar) GetVersion() string {
	return m.version
}

func (m *MockRouteRegistrar) GetPrefix() string {
	return m.prefix
}

// MockMiddlewareRegistrar 中间件注册器的mock实现
type MockMiddlewareRegistrar struct {
	name            string
	priority        int
	errorOnRegister bool
}

func (m *MockMiddlewareRegistrar) RegisterMiddleware(router *gin.Engine) error {
	if m.errorOnRegister {
		return errors.New("mock middleware error")
	}
	return nil
}

func (m *MockMiddlewareRegistrar) GetName() string {
	return m.name
}

func (m *MockMiddlewareRegistrar) GetPriority() int {
	return m.priority
}

func TestNew(t *testing.T) {
	registry := New()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.routeRegistrars)
	assert.NotNil(t, registry.middlewareRegistrars)
	assert.Equal(t, 0, len(registry.routeRegistrars))
	assert.Equal(t, 0, len(registry.middlewareRegistrars))
}

func TestRegistry_RegisterRoute(t *testing.T) {
	tests := []struct {
		name      string
		registrar RouteRegistrar
	}{
		{
			name: "成功注册路由注册器",
			registrar: &MockRouteRegistrar{
				name:    "test-route",
				version: "v1",
				prefix:  "test",
			},
		},
		{
			name: "注册空前缀的路由注册器",
			registrar: &MockRouteRegistrar{
				name:    "test-route-2",
				version: "v2",
				prefix:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := New()
			registry.RegisterRoute(tt.registrar)

			assert.Equal(t, 1, len(registry.routeRegistrars))
			assert.Equal(t, tt.registrar, registry.routeRegistrars[0])
		})
	}
}

func TestRegistry_RegisterMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		registrar MiddlewareRegistrar
	}{
		{
			name: "成功注册中间件注册器",
			registrar: &MockMiddlewareRegistrar{
				name:     "test-middleware",
				priority: 1,
			},
		},
		{
			name: "注册高优先级中间件注册器",
			registrar: &MockMiddlewareRegistrar{
				name:     "high-priority-middleware",
				priority: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := New()
			registry.RegisterMiddleware(tt.registrar)

			assert.Equal(t, 1, len(registry.middlewareRegistrars))
			assert.Equal(t, tt.registrar, registry.middlewareRegistrars[0])
		})
	}
}

func TestRegistry_Setup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                 string
		routeRegistrars      []RouteRegistrar
		middlewareRegistrars []MiddlewareRegistrar
		expectError          bool
		errorMessage         string
	}{
		{
			name: "成功设置路由和中间件",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{name: "route1", version: "v1", prefix: "test"},
			},
			middlewareRegistrars: []MiddlewareRegistrar{
				&MockMiddlewareRegistrar{name: "middleware1", priority: 1},
			},
			expectError: false,
		},
		{
			name:                 "设置空的注册器",
			routeRegistrars:      []RouteRegistrar{},
			middlewareRegistrars: []MiddlewareRegistrar{},
			expectError:          false,
		},
		{
			name: "中间件注册失败",
			middlewareRegistrars: []MiddlewareRegistrar{
				&MockMiddlewareRegistrar{
					name:            "failing-middleware",
					priority:        1,
					errorOnRegister: true,
				},
			},
			expectError:  true,
			errorMessage: "mock middleware error",
		},
		{
			name: "路由注册失败",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{
					name:            "failing-route",
					version:         "v1",
					prefix:          "test",
					errorOnRegister: true,
				},
			},
			expectError:  true,
			errorMessage: "mock register error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := New()

			// 注册路由注册器
			for _, rr := range tt.routeRegistrars {
				registry.RegisterRoute(rr)
			}

			// 注册中间件注册器
			for _, mr := range tt.middlewareRegistrars {
				registry.RegisterMiddleware(mr)
			}

			router := gin.New()
			err := registry.Setup(router)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistry_setupMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                 string
		middlewareRegistrars []MiddlewareRegistrar
		expectError          bool
	}{
		{
			name: "按优先级排序中间件",
			middlewareRegistrars: []MiddlewareRegistrar{
				&MockMiddlewareRegistrar{name: "middleware3", priority: 3},
				&MockMiddlewareRegistrar{name: "middleware1", priority: 1},
				&MockMiddlewareRegistrar{name: "middleware2", priority: 2},
			},
			expectError: false,
		},
		{
			name: "单个中间件",
			middlewareRegistrars: []MiddlewareRegistrar{
				&MockMiddlewareRegistrar{name: "middleware1", priority: 1},
			},
			expectError: false,
		},
		{
			name:                 "无中间件",
			middlewareRegistrars: []MiddlewareRegistrar{},
			expectError:          false,
		},
		{
			name: "中间件注册失败",
			middlewareRegistrars: []MiddlewareRegistrar{
				&MockMiddlewareRegistrar{
					name:            "failing-middleware",
					priority:        1,
					errorOnRegister: true,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := New()
			registry.middlewareRegistrars = tt.middlewareRegistrars

			router := gin.New()
			err := registry.setupMiddlewares(router)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// 验证中间件是否按优先级排序
				if len(tt.middlewareRegistrars) > 1 {
					for i := 1; i < len(registry.middlewareRegistrars); i++ {
						assert.LessOrEqual(t,
							registry.middlewareRegistrars[i-1].GetPriority(),
							registry.middlewareRegistrars[i].GetPriority(),
							"中间件应该按优先级升序排列")
					}
				}
			}
		})
	}
}

func TestRegistry_setupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		routeRegistrars []RouteRegistrar
		expectError     bool
	}{
		{
			name: "不同版本的路由",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{name: "route1", version: "v1", prefix: "test"},
				&MockRouteRegistrar{name: "route2", version: "v2", prefix: "api"},
			},
			expectError: false,
		},
		{
			name: "相同版本不同前缀的路由",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{name: "route1", version: "v1", prefix: "test"},
				&MockRouteRegistrar{name: "route2", version: "v1", prefix: "api"},
			},
			expectError: false,
		},
		{
			name: "空版本的路由（默认为v1）",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{name: "route1", version: "", prefix: "test"},
			},
			expectError: false,
		},
		{
			name: "root版本的路由",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{name: "route1", version: "root", prefix: ""},
			},
			expectError: false,
		},
		{
			name: "空前缀的路由",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{name: "route1", version: "v1", prefix: ""},
			},
			expectError: false,
		},
		{
			name: "路由注册失败",
			routeRegistrars: []RouteRegistrar{
				&MockRouteRegistrar{
					name:            "failing-route",
					version:         "v1",
					prefix:          "test",
					errorOnRegister: true,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := New()
			registry.routeRegistrars = tt.routeRegistrars

			router := gin.New()
			err := registry.setupRoutes(router)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistry_GetRegisteredRoutes(t *testing.T) {
	registry := New()

	// 测试空注册表
	routes := registry.GetRegisteredRoutes()
	assert.NotNil(t, routes)
	assert.Equal(t, 0, len(routes))

	// 添加路由注册器
	registrar1 := &MockRouteRegistrar{name: "route1", version: "v1", prefix: "test"}
	registrar2 := &MockRouteRegistrar{name: "route2", version: "v2", prefix: "api"}

	registry.RegisterRoute(registrar1)
	registry.RegisterRoute(registrar2)

	routes = registry.GetRegisteredRoutes()
	require.Equal(t, 2, len(routes))

	// 验证返回的路由信息
	assert.Equal(t, "route1", routes[0]["name"])
	assert.Equal(t, "v1", routes[0]["version"])
	assert.Equal(t, "test", routes[0]["prefix"])

	assert.Equal(t, "route2", routes[1]["name"])
	assert.Equal(t, "v2", routes[1]["version"])
	assert.Equal(t, "api", routes[1]["prefix"])
}

func TestRegistry_GetRegisteredMiddlewares(t *testing.T) {
	registry := New()

	// 测试空注册表
	middlewares := registry.GetRegisteredMiddlewares()
	assert.NotNil(t, middlewares)
	assert.Equal(t, 0, len(middlewares))

	// 添加中间件注册器
	registrar1 := &MockMiddlewareRegistrar{name: "middleware1", priority: 1}
	registrar2 := &MockMiddlewareRegistrar{name: "middleware2", priority: 2}

	registry.RegisterMiddleware(registrar1)
	registry.RegisterMiddleware(registrar2)

	middlewares = registry.GetRegisteredMiddlewares()
	require.Equal(t, 2, len(middlewares))

	// 验证返回的中间件信息
	assert.Equal(t, "middleware1", middlewares[0]["name"])
	assert.Equal(t, 1, middlewares[0]["priority"])

	assert.Equal(t, "middleware2", middlewares[1]["name"])
	assert.Equal(t, 2, middlewares[1]["priority"])
}

func TestRegistry_IntegrationTest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := New()

	// 注册多个中间件（不同优先级）
	registry.RegisterMiddleware(&MockMiddlewareRegistrar{name: "cors", priority: 1})
	registry.RegisterMiddleware(&MockMiddlewareRegistrar{name: "auth", priority: 2})
	registry.RegisterMiddleware(&MockMiddlewareRegistrar{name: "logger", priority: 0})

	// 注册多个路由（不同版本和前缀）
	registry.RegisterRoute(&MockRouteRegistrar{name: "user", version: "v1", prefix: "api"})
	registry.RegisterRoute(&MockRouteRegistrar{name: "auth", version: "v1", prefix: "auth"})
	registry.RegisterRoute(&MockRouteRegistrar{name: "health", version: "root", prefix: ""})
	registry.RegisterRoute(&MockRouteRegistrar{name: "admin", version: "v2", prefix: "admin"})

	// 设置路由器
	router := gin.New()
	err := registry.Setup(router)
	assert.NoError(t, err)

	// 验证注册信息
	routes := registry.GetRegisteredRoutes()
	assert.Equal(t, 4, len(routes))

	middlewares := registry.GetRegisteredMiddlewares()
	assert.Equal(t, 3, len(middlewares))
}

// BenchmarkRegistry_Setup 性能测试
func BenchmarkRegistry_Setup(b *testing.B) {
	gin.SetMode(gin.TestMode)

	registry := New()

	// 添加大量注册器
	for i := 0; i < 100; i++ {
		registry.RegisterRoute(&MockRouteRegistrar{
			name:    "route",
			version: "v1",
			prefix:  "test",
		})
		registry.RegisterMiddleware(&MockMiddlewareRegistrar{
			name:     "middleware",
			priority: i,
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router := gin.New()
		_ = registry.Setup(router)
	}
}
