# Switserve 路由注册系统使用指南

## 概述

Switserve 项目已升级为更加企业级的路由注册系统，提供了更好的扩展性、可维护性和统一性。

## 核心概念

### 1. RouteRegistrar 接口

所有路由模块都需要实现 `RouteRegistrar` 接口：

```go
type RouteRegistrar interface {
    RegisterRoutes(rg *gin.RouterGroup) error
    GetName() string
    GetVersion() string
    GetPrefix() string
}
```

### 2. MiddlewareRegistrar 接口

中间件注册需要实现 `MiddlewareRegistrar` 接口：

```go
type MiddlewareRegistrar interface {
    RegisterMiddleware(router *gin.Engine) error
    GetName() string
    GetPriority() int
}
```

### 3. RouteRegistry 管理器

`RouteRegistry` 是核心管理器，负责：
- 收集所有路由和中间件注册器
- 按优先级排序中间件
- 按版本分组路由
- 统一设置所有路由和中间件

## 主要优势

### 1. 版本管理
- 自动按 API 版本分组路由（如 `/v1/users`, `/v2/users`）
- 支持无版本路由（`version: "root"`）

### 2. 命名统一
- 统一的接口命名规范
- 一致的注册方法签名

### 3. 易于扩展
- 添加新模块只需要实现接口
- 无需修改核心路由文件

### 4. 中间件优先级
- 按数字优先级自动排序中间件
- 确保中间件执行顺序的正确性

### 5. 调试支持
- 内置调试端点查看已注册的路由和中间件
- 便于开发和维护

## 如何添加新模块

### 1. 创建控制器

```go
// internal/switserve/handler/v1/article/handler.go
package article

type ArticleController struct {
    // 你的依赖
}

func NewArticleController() *ArticleController {
    return &ArticleController{}
}

func (ac *ArticleController) GetArticles(c *gin.Context) {
    // 实现逻辑
}
```

### 2. 实现路由注册器

```go
// internal/switserve/handler/v1/article/registrar.go
package article

import "github.com/gin-gonic/gin"

type ArticleRouteRegistrar struct {
    controller *ArticleController
}

func NewArticleRouteRegistrar() *ArticleRouteRegistrar {
    return &ArticleRouteRegistrar{
        controller: NewArticleController(),
    }
}

func (arr *ArticleRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
    articles := rg.Group("/articles")
    {
        articles.GET("", arr.controller.GetArticles)
        articles.POST("", arr.controller.CreateArticle)
    }
    return nil
}

func (arr *ArticleRouteRegistrar) GetName() string {
    return "article-api"
}

func (arr *ArticleRouteRegistrar) GetVersion() string {
    return "v1"
}

func (arr *ArticleRouteRegistrar) GetPrefix() string {
    return ""
}
```

### 3. 注册到路由表

在 `internal/switserve/server/router.go` 中添加：

```go
import "github.com/innovationmech/swit/internal/switserve/handler/v1/article"

func (s *Server) SetupRoutes() {
    registry := NewRouteRegistry()
    
    // ... 其他注册 ...
    
    // 注册新的文章路由
    registry.RegisterRoute(article.NewArticleRouteRegistrar())
    
    // ... 设置路由 ...
}
```

## 最佳实践

### 1. 模块组织
```
internal/switserve/controller/
├── v1/
│   ├── user/
│   │   ├── controller.go
│   │   └── registrar.go
│   └── article/
│       ├── controller.go
│       └── registrar.go
├── health/
│   ├── health.go
│   └── registrar.go
└── debug/
    └── registrar.go
```

### 2. 版本管理策略
- 使用 `v1`, `v2` 等表示API版本
- 使用 `root` 表示无版本前缀
- 基础功能（health, debug）使用 `root`

### 3. 中间件优先级
- 全局中间件: 1-10
- 认证中间件: 11-20
- 业务中间件: 21-30
- 数字越小优先级越高

### 4. 路由前缀
- 通常为空，让版本和控制器名称组成路径
- 特殊情况下可以使用前缀（如 `/admin`）

## 调试功能

系统提供了调试端点来查看注册信息：

- `GET /debug/routes` - 查看已注册的路由
- `GET /debug/middlewares` - 查看已注册的中间件
- `GET /debug/gin-routes` - 查看Gin原生路由信息

## 迁移现有代码

从旧的注册方式迁移到新系统：

### 旧方式：
```go
func (s *Server) SetupRoutes() {
    user.RegisterMiddleware(s.router)
    health.RegisterRoutes(s.router)
}
```

### 新方式：
```go
func (s *Server) SetupRoutes() {
    registry := NewRouteRegistry()
    registry.RegisterRoute(user.NewUserRouteRegistrar())
    registry.RegisterRoute(health.NewHealthRouteRegistrar())
    registry.Setup(s.router)
}
```

## 配置示例

生成的路由结构：
```
/health                    (健康检查)
/stop                      (停止服务)
/debug/routes              (调试-路由信息)
/debug/middlewares         (调试-中间件信息)
/internal/validate-user    (内部API)
/v1/users/create           (用户API v1)
/v1/users/username/:name   (用户API v1)
/v1/articles               (文章API v1)
/v2/articles               (文章API v2，如果存在)
```

## 总结

新的路由注册系统提供了：
- 更好的代码组织和可维护性
- 统一的接口和命名规范
- 灵活的版本管理
- 强大的调试和监控功能
- 易于扩展的架构

这种设计模式符合企业级开发的最佳实践，使代码更加模块化、可测试和易于维护。
