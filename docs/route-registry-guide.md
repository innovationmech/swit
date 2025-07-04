# SWIT 路由注册系统使用指南

## 概述

SWIT 项目采用了企业级的路由注册系统，提供了更好的扩展性、可维护性和统一性。该系统支持版本管理、中间件优先级排序、调试监控等高级功能。

## 🏗 架构概览

```
Registry (注册表管理器)
├── RouteRegistrar[] (路由注册器数组)
│   ├── 版本分组 (v1, v2, root)
│   ├── 前缀管理 (prefix)
│   └── 路由注册 (RegisterRoutes)
└── MiddlewareRegistrar[] (中间件注册器数组)
    ├── 优先级排序 (priority)
    └── 中间件注册 (RegisterMiddleware)
```

## 🔧 核心接口

### 1. RouteRegistrar 接口

所有路由模块都需要实现 `RouteRegistrar` 接口：

```go
// 位置：internal/switserve/router/registry.go
type RouteRegistrar interface {
    // RegisterRoutes 注册路由到指定的路由组
    RegisterRoutes(rg *gin.RouterGroup) error
    // GetName 获取注册器名称，用于日志和调试
    GetName() string
    // GetVersion 获取API版本 (如: "v1", "v2", "root")
    GetVersion() string
    // GetPrefix 获取路由前缀 (可选)
    GetPrefix() string
}
```

### 2. MiddlewareRegistrar 接口

中间件注册需要实现 `MiddlewareRegistrar` 接口：

```go
type MiddlewareRegistrar interface {
    // RegisterMiddleware 注册中间件到路由器
    RegisterMiddleware(router *gin.Engine) error
    // GetName 获取中间件名称
    GetName() string
    // GetPriority 获取中间件优先级（数字越小优先级越高）
    GetPriority() int
}
```

### 3. Registry 管理器

`Registry` 是核心管理器，提供以下功能：

```go
type Registry struct {
    routeRegistrars      []RouteRegistrar
    middlewareRegistrars []MiddlewareRegistrar
}

// 主要方法
func New() *Registry                                    // 创建注册表
func (r *Registry) RegisterRoute(RouteRegistrar)        // 注册路由
func (r *Registry) RegisterMiddleware(MiddlewareRegistrar) // 注册中间件
func (r *Registry) Setup(*gin.Engine) error            // 设置所有路由和中间件
```

## 🚀 系统特性

### 1. 智能版本管理
- **版本前缀**：自动为不同版本创建路由组
  - `v1` → `/v1/...`
  - `v2` → `/v2/...`
  - `root` → `/...` (无版本前缀)
- **向后兼容**：支持多版本API并存
- **默认处理**：空版本自动设为 `v1`

### 2. 中间件优先级排序
```go
// 自动按优先级排序
sort.Slice(middlewares, func(i, j int) bool {
    return middlewares[i].GetPriority() < middlewares[j].GetPriority()
})
```

### 3. 调试和监控
- **注册信息查询**：`GetRegisteredRoutes()`, `GetRegisteredMiddlewares()`
- **实时日志**：详细的注册过程日志
- **错误处理**：完整的错误信息和回滚机制

## 📝 实际使用示例

### 1. 创建用户路由注册器

```go
// internal/switserve/handler/v1/user/registrar.go
package user

import (
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/internal/switserve/middleware"
)

type UserRouteRegistrar struct {
    controller *UserController
}

func NewUserRouteRegistrar() *UserRouteRegistrar {
    return &UserRouteRegistrar{
        controller: NewUserController(),
    }
}

func (urr *UserRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
    // 需要认证的用户API
    userGroup := rg.Group("/users")
    userGroup.Use(middleware.AuthMiddleware()) // 使用认证中间件
    {
        userGroup.GET("/username/:username", urr.controller.GetUserByUsername)
        userGroup.GET("/email/:email", urr.controller.GetUserByEmail)
        userGroup.DELETE("/:id", urr.controller.DeleteUser)
    }
    
    // 公开的用户API
    publicGroup := rg.Group("/users")
    {
        publicGroup.POST("/create", urr.controller.CreateUser)
    }
    
    return nil
}

func (urr *UserRouteRegistrar) GetName() string {
    return "user-api"
}

func (urr *UserRouteRegistrar) GetVersion() string {
    return "v1"
}

func (urr *UserRouteRegistrar) GetPrefix() string {
    return "" // 无额外前缀
}
```

### 2. 创建中间件注册器

```go
// internal/switserve/middleware/registrar.go
package middleware

import "github.com/gin-gonic/gin"

type GlobalMiddlewareRegistrar struct{}

func NewGlobalMiddlewareRegistrar() *GlobalMiddlewareRegistrar {
    return &GlobalMiddlewareRegistrar{}
}

func (gmr *GlobalMiddlewareRegistrar) RegisterMiddleware(router *gin.Engine) error {
    // 注册全局中间件
    router.Use(CORSMiddleware())
    router.Use(LoggerMiddleware())
    router.Use(TimeoutMiddleware())
    return nil
}

func (gmr *GlobalMiddlewareRegistrar) GetName() string {
    return "global-middleware"
}

func (gmr *GlobalMiddlewareRegistrar) GetPriority() int {
    return 1 // 最高优先级
}
```

### 3. 在服务器中注册

```go
// internal/switserve/server/router.go
func (s *Server) SetupRoutes() {
    // 创建路由注册表
    registry := router.New()

    // 配置应用路由
    s.configureRoutes(registry)

    // 设置所有路由和中间件
    if err := registry.Setup(s.router); err != nil {
        logger.Logger.Fatal("Failed to setup routes", zap.Error(err))
    }

    logger.Logger.Info("Route registry setup completed")
}

func (s *Server) configureRoutes(registry *router.Registry) {
    // 注册全局中间件
    registry.RegisterMiddleware(middleware.NewGlobalMiddlewareRegistrar())

    // 注册API路由
    registry.RegisterRoute(health.NewHealthRouteRegistrar())
    registry.RegisterRoute(stop.NewStopRouteRegistrar(s.Shutdown))
    registry.RegisterRoute(user.NewUserRouteRegistrar())
    registry.RegisterRoute(user.NewUserInternalRouteRegistrar())

    // 注册调试路由（仅在开发环境）
    if gin.Mode() == gin.DebugMode {
        registry.RegisterRoute(debug.NewDebugRouteRegistrar(registry, s.router))
    }
}
```

## 📁 目录结构最佳实践

```
internal/switserve/
├── handler/
│   ├── v1/                    # v1版本API
│   │   ├── user/
│   │   │   ├── user.go        # 用户业务逻辑
│   │   │   ├── create.go      # 创建用户
│   │   │   ├── get.go         # 获取用户
│   │   │   ├── delete.go      # 删除用户
│   │   │   ├── internal.go    # 内部API
│   │   │   └── registrar.go   # 路由注册器
│   │   └── article/           # 文章模块
│   │       ├── article.go
│   │       └── registrar.go
│   ├── v2/                    # v2版本API (未来)
│   ├── health/                # 健康检查 (root版本)
│   │   ├── health.go
│   │   └── registrar.go
│   ├── stop/                  # 停止服务 (root版本)
│   │   ├── stop.go
│   │   └── registrar.go
│   └── debug/                 # 调试端点 (root版本)
│       └── registrar.go
├── middleware/                # 中间件
│   ├── auth.go
│   ├── cors.go
│   ├── timeout.go
│   └── registrar.go           # 中间件注册器
├── router/                    # 路由注册系统
│   ├── registry.go            # 核心注册表
│   └── registry_test.go       # 测试
└── server/
    └── router.go              # 服务器路由配置
```

## ⚙️ 版本管理策略

### 版本类型
- **`v1`, `v2`, `v3`**：版本化API，生成 `/v1/`, `/v2/` 前缀
- **`root`**：根级API，无版本前缀，用于基础服务

### 示例路由结构
```
/                              # root版本
├── health                     # 健康检查
├── stop                       # 停止服务  
└── debug/                     # 调试端点
    ├── routes
    └── middlewares

/v1/                           # v1版本API
├── users/
│   ├── create
│   ├── username/:username
│   └── email/:email
└── internal/
    └── validate-user

/v2/                           # v2版本API (将来)
└── users/
    └── ...
```

## 🔧 中间件优先级指南

```go
// 推荐的优先级分配
const (
    PriorityGlobal     = 1-10    // 全局中间件 (CORS, Logger)
    PriorityAuth       = 11-20   // 认证中间件  
    PrioritySecurity   = 21-30   // 安全中间件 (Rate Limit, Security Headers)
    PriorityBusiness   = 31-40   // 业务中间件
    PriorityCustom     = 41-50   // 自定义中间件
)
```

### 示例
```go
func (gmr *GlobalMiddlewareRegistrar) GetPriority() int {
    return 1 // 最高优先级，最先执行
}

func (amr *AuthMiddlewareRegistrar) GetPriority() int {
    return 15 // 在全局中间件之后，业务中间件之前
}
```

## 🐛 调试功能

### 内置调试端点
```bash
# 查看已注册的路由信息
GET /debug/routes

# 查看已注册的中间件信息  
GET /debug/middlewares

# 查看Gin原生路由信息
GET /debug/gin-routes
```

### 响应示例
```json
// GET /debug/routes
{
  "routes": [
    {
      "name": "user-api",
      "version": "v1", 
      "prefix": ""
    },
    {
      "name": "health-api",
      "version": "root",
      "prefix": ""
    }
  ]
}

// GET /debug/middlewares  
{
  "middlewares": [
    {
      "name": "global-middleware",
      "priority": 1
    },
    {
      "name": "auth-middleware", 
      "priority": 15
    }
  ]
}
```

## 🧪 单元测试

系统提供了完整的测试支持：

```go
// registry_test.go 示例
func TestRegistry_Setup(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    registry := New()

    // 测试中间件优先级排序
    middleware1 := NewTestMiddlewareRegistrar("middleware-1", 20)
    middleware2 := NewTestMiddlewareRegistrar("middleware-2", 10)
    registry.RegisterMiddleware(middleware1)
    registry.RegisterMiddleware(middleware2)

    // 测试版本路由
    route1 := NewTestRouteRegistrar("api-v1", "v1", "")
    route2 := NewTestRouteRegistrar("root-api", "root", "")
    registry.RegisterRoute(route1)
    registry.RegisterRoute(route2)

    err := registry.Setup(router)
    assert.NoError(t, err)

    // 验证路由是否正确注册
    routes := router.Routes()
    routePaths := []string{}
    for _, route := range routes {
        routePaths = append(routePaths, route.Path)
    }

    assert.Contains(t, routePaths, "/v1/test") // v1版本路由
    assert.Contains(t, routePaths, "/test")    // root版本路由
}
```

## 🚀 迁移指南

### 从旧系统迁移

**旧方式**：
```go
func (s *Server) SetupRoutes() {
    s.router.Use(middleware.CORS())
    s.router.Use(middleware.Logger())
    
    v1 := s.router.Group("/v1")
    user.RegisterUserRoutes(v1)
    article.RegisterArticleRoutes(v1)
}
```

**新方式**：
```go
func (s *Server) SetupRoutes() {
    registry := router.New()
    
    // 注册中间件
    registry.RegisterMiddleware(middleware.NewGlobalMiddlewareRegistrar())
    
    // 注册路由
    registry.RegisterRoute(user.NewUserRouteRegistrar())
    registry.RegisterRoute(article.NewArticleRouteRegistrar())
    
    // 统一设置
    registry.Setup(s.router)
}
```

## 📊 性能优化

### 1. 路由组合
- 相同版本的路由自动组合到同一个路由组
- 减少中间件重复执行

### 2. 中间件排序
- 一次性排序，避免运行时排序开销
- 按优先级预先排列执行顺序

### 3. 延迟初始化
- 路由注册器支持延迟初始化
- 减少启动时间和内存占用

## 🔒 安全考虑

### 1. 中间件安全
```go
// 推荐的安全中间件顺序
registry.RegisterMiddleware(NewSecurityHeadersMiddleware())  // priority: 5
registry.RegisterMiddleware(NewRateLimitMiddleware())        // priority: 10  
registry.RegisterMiddleware(NewAuthMiddleware())             // priority: 15
```

### 2. 调试端点安全
```go
// 仅在开发环境启用调试端点
if gin.Mode() == gin.DebugMode {
    registry.RegisterRoute(debug.NewDebugRouteRegistrar(registry, s.router))
}
```

## 💡 最佳实践总结

1. **接口实现**：严格实现 `RouteRegistrar` 和 `MiddlewareRegistrar` 接口
2. **版本管理**：使用语义化版本号，保持向后兼容
3. **中间件优先级**：遵循推荐的优先级分配原则
4. **错误处理**：在 `RegisterRoutes` 方法中妥善处理错误
5. **日志记录**：利用内置的日志功能进行调试
6. **测试覆盖**：为每个路由注册器编写单元测试
7. **文档维护**：及时更新API文档和使用说明

## 🎯 高级功能

### 1. 条件注册
```go
// 根据环境条件注册不同的路由
if config.IsProduction() {
    registry.RegisterRoute(NewProductionRouteRegistrar())
} else {
    registry.RegisterRoute(NewDevelopmentRouteRegistrar())
}
```

### 2. 插件式架构
```go
// 支持插件式的路由扩展
for _, plugin := range plugins {
    if registrar, ok := plugin.(RouteRegistrar); ok {
        registry.RegisterRoute(registrar)
    }
}
```

### 3. 动态路由
```go
// 支持运行时动态添加路由（谨慎使用）
func (r *Registry) AddRoute(registrar RouteRegistrar) error {
    // 动态添加逻辑
}
```

这个路由注册系统为 SWIT 项目提供了企业级的路由管理能力，支持复杂的多版本API架构，同时保持了代码的简洁性和可维护性。
