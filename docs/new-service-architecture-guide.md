# SWIT 新子项目架构指导方案

## 概述

本文档基于最新的 ServiceHandler 架构模式，为新增子项目提供标准化的架构指导方案。新架构采用统一的 ServiceHandler 接口，提供完整的服务生命周期管理，包括初始化、健康检查、优雅关闭等功能。遵循本指南可以确保新项目与现有项目保持架构一致性，提高代码复用性和可维护性。

## 架构设计原则

### 1. 分层架构模式

新项目必须遵循以下分层架构：

```
┌─────────────────────────────────────────┐
│                Server                   │
├─────────────────────────────────────────┤
│           Transport Layer               │
│  ┌─────────────────┬─────────────────┐  │
│  │  HTTPTransport  │  GRPCTransport  │  │
│  └─────────────────┴─────────────────┘  │
├─────────────────────────────────────────┤
│            Service Layer                │
│  ┌─────────────────────────────────────┐ │
│  │        ServiceRegistry              │ │
│  │  ┌───────────┬───────────────────┐  │ │
│  │  │ Service A │ Service B │ ... │  │ │
│  │  └───────────┴───────────────────┘  │ │
│  └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│           Handler Layer                 │
│  ┌─────────────────┬─────────────────┐  │
│  │   HTTP Handler  │   gRPC Handler  │  │
│  └─────────────────┴─────────────────┘  │
└─────────────────────────────────────────┘
```

### 2. 核心设计原则

- **ServiceHandler 模式**: 所有服务必须实现统一的 ServiceHandler 接口
- **接口分离**: 业务接口定义在独立的 interfaces 包中，避免循环依赖
- **共享类型系统**: 使用 types 包定义共享的数据模型和错误类型
- **生命周期管理**: 支持服务初始化、健康检查和优雅关闭
- **领域驱动设计**: 按业务领域划分服务目录结构
- **版本化设计**: 需要版本化的服务接口采用版本化设计（如 v1、v2），支持向后兼容的 API 演进
- **统一错误处理**: 采用标准化的 ServiceError 类型和错误处理机制
- **依赖注入优化**: 使用 EnhancedServiceRegistry 进行服务注册和管理
- **单一职责**: 每个组件只负责一个明确的功能
- **配置外部化**: 所有配置通过配置文件管理
- **并发安全**: 所有共享资源必须线程安全

## 项目结构模板

### 目录结构

```
internal/new-service/
├── cmd/
│   └── root.go              # CLI 命令定义
├── config/
│   ├── config.go            # 配置结构定义
│   └── config_test.go       # 配置测试
├── db/
│   ├── db.go               # 数据库连接管理
│   └── migrations/         # 数据库迁移文件
├── interfaces/              # 业务接口定义（新增）
│   ├── user.go             # 用户服务接口
│   ├── auth.go             # 认证服务接口
│   └── health.go           # 健康检查接口
├── types/                   # 共享类型定义（新增）
│   ├── user.go             # 用户相关类型
│   ├── auth.go             # 认证相关类型
│   ├── common.go           # 通用类型
│   └── errors.go           # 错误类型定义
├── handler/
│   ├── http/
│   │   ├── user.go         # 用户HTTP处理器
│   │   ├── auth.go         # 认证HTTP处理器
│   │   └── health.go       # 健康检查HTTP处理器
│   ├── grpc/               # gRPC处理器（可选）
│   │   ├── user.go         # 用户gRPC处理器
│   │   └── auth.go         # 认证gRPC处理器
│   ├── user_service_handler.go      # 用户ServiceHandler实现
│   ├── auth_service_handler.go      # 认证ServiceHandler实现
│   ├── health_service_handler.go    # 健康检查ServiceHandler实现
│   └── middleware/         # 项目特定中间件（可选）
│       └── custom.go       # 自定义业务中间件
│
│   注意：通用中间件（如认证、CORS、日志等）统一放在 `/pkg/middleware/` 目录下，
│         供所有子项目复用。项目内的 middleware 目录仅用于项目特定的业务中间件。
├── repository/
│   ├── user.go             # 用户仓储实现
│   ├── auth.go             # 认证仓储实现
│   └── interfaces.go       # 仓储接口定义
├── service/                # 业务服务实现
│   ├── user/
│   │   ├── service.go      # 用户服务实现
│   │   ├── service_test.go # 用户服务测试
│   │   └── options.go      # 服务选项配置
│   ├── auth/
│   │   ├── service.go      # 认证服务实现
│   │   ├── service_test.go # 认证服务测试
│   │   └── options.go      # 服务选项配置
│   └── health/
│       ├── service.go      # 健康检查服务实现
│       └── service_test.go # 健康检查服务测试
├── transport/
│   ├── service_handler.go  # ServiceHandler接口定义
│   ├── registry.go         # EnhancedServiceRegistry实现
│   ├── http.go             # HTTP传输实现
│   ├── grpc.go             # gRPC传输实现
│   └── manager.go          # 传输管理器
├── server.go               # 服务器主文件
└── server_test.go          # 服务器测试
```

### 根目录文件

```
cmd/new-service/
├── new-service.go          # 主程序入口
└── new-service_test.go     # 主程序测试
```

## 核心组件实现指南

### 1. 中间件使用指南

#### 通用中间件复用

项目应优先使用 `pkg/middleware` 中的通用中间件：

```go
// internal/new-service/server.go
import (
    "github.com/innovationmech/swit/pkg/middleware"
)

// 在服务器配置中使用全局中间件注册器
func (s *Server) configureMiddleware() {
    // 使用全局中间件注册器
    globalMiddleware := middleware.NewGlobalMiddlewareRegistrar()
    globalMiddleware.RegisterMiddleware(s.httpTransport.GetRouter())
    
    // 如果需要认证中间件
    s.httpTransport.GetRouter().Use(middleware.AuthMiddleware())
    
    // 或使用自定义配置的认证中间件
    authConfig := &middleware.AuthConfig{
        WhiteList: []string{"/health", "/metrics", "/api/v1/public"},
        AuthServiceName: "switauth",
        AuthEndpoint: "/auth/validate",
    }
    s.httpTransport.GetRouter().Use(middleware.AuthMiddlewareWithConfig(authConfig))
}
```

#### 项目特定中间件

仅在需要项目特定业务逻辑时，才在项目内创建自定义中间件：

```go
// internal/new-service/middleware/custom.go
package middleware

import (
    "github.com/gin-gonic/gin"
)

// ProjectSpecificMiddleware 项目特定的业务中间件
func ProjectSpecificMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 项目特定的业务逻辑
        c.Next()
    }
}
```

### 2. ServiceHandler 接口实现

每个服务必须实现 `ServiceHandler` 接口（新架构模式）：

```go
type ServiceHandler interface {
    // RegisterHTTP registers HTTP routes with the given router
    RegisterHTTP(router *gin.Engine) error

    // RegisterGRPC registers gRPC services with the given server
    RegisterGRPC(server *grpc.Server) error

    // GetMetadata returns service metadata information
    GetMetadata() *ServiceMetadata

    // GetHealthEndpoint returns the health check endpoint path
    GetHealthEndpoint() string

    // IsHealthy performs a health check and returns the current status
    IsHealthy(ctx context.Context) (*types.HealthStatus, error)

    // Initialize performs any necessary initialization before service registration
    Initialize(ctx context.Context) error

    // Shutdown performs graceful shutdown of the service
    Shutdown(ctx context.Context) error
}

type ServiceMetadata struct {
    // Name is the service name
    Name string `json:"name"`
    // Version is the service version
    Version string `json:"version"`
    // Description is a brief description of the service
    Description string `json:"description"`
    // HealthEndpoint is the health check endpoint path
    HealthEndpoint string `json:"health_endpoint"`
    // Tags are optional tags for service categorization
    Tags []string `json:"tags,omitempty"`
    // Dependencies are the services this service depends on
    Dependencies []string `json:"dependencies,omitempty"`
}
```

**实现模板：**

```go
// internal/new-service/handler/{domain-name}_service_handler.go
package handler

import (
    "context"
    "fmt"
    "log/slog"
    "time"
    
    "google.golang.org/grpc"
    "github.com/gin-gonic/gin"
    
    "your-project/internal/new-service/interfaces"
    "your-project/internal/new-service/transport"
    "your-project/internal/new-service/types"
    httpv1 "your-project/internal/new-service/handler/http/v1"
)

// DomainServiceHandler 实现 ServiceHandler 接口
type DomainServiceHandler struct {
    service   interfaces.DomainService
    logger    *slog.Logger
    healthy   bool
    startTime time.Time
}

// NewDomainServiceHandler 创建新的领域服务处理器
func NewDomainServiceHandler(service interfaces.DomainService, logger *slog.Logger) *DomainServiceHandler {
    return &DomainServiceHandler{
        service:   service,
        logger:    logger,
        healthy:   false,
        startTime: time.Now(),
    }
}

// GetMetadata 返回服务元数据
func (h *DomainServiceHandler) GetMetadata() *transport.ServiceMetadata {
    return &transport.ServiceMetadata{
        Name:           "domain-service",
        Version:        "v1.0.0",
        Description:    "Domain service for managing entities",
        HealthEndpoint: "/api/v1/domain/health",
        Tags:           []string{"api", "domain"},
        Dependencies:   []string{"database", "cache"},
    }
}

// GetHealthEndpoint 返回健康检查端点
func (h *DomainServiceHandler) GetHealthEndpoint() string {
    return "/api/v1/domain/health"
}

// Initialize 初始化服务
func (h *DomainServiceHandler) Initialize(ctx context.Context) error {
    h.logger.Info("Initializing domain service handler")
    
    // 执行初始化逻辑
    if err := h.service.Initialize(ctx); err != nil {
        h.logger.Error("Failed to initialize domain service", "error", err)
        return fmt.Errorf("failed to initialize domain service: %w", err)
    }
    
    h.healthy = true
    h.logger.Info("Domain service handler initialized successfully")
    return nil
}

// RegisterHTTP 注册HTTP路由
func (h *DomainServiceHandler) RegisterHTTP(router *gin.Engine) error {
    h.logger.Info("Registering HTTP routes for domain service")
    
    // 创建HTTP处理器
    httpHandler := httpv1.NewDomainHandler(h.service, h.logger)
    
    // 注册API路由
    v1Group := router.Group("/api/v1/domain")
    {
        v1Group.GET("/health", httpHandler.HealthCheck)
        v1Group.GET("/entities", httpHandler.ListEntities)
        v1Group.GET("/entities/:id", httpHandler.GetEntity)
        v1Group.POST("/entities", httpHandler.CreateEntity)
        v1Group.PUT("/entities/:id", httpHandler.UpdateEntity)
        v1Group.DELETE("/entities/:id", httpHandler.DeleteEntity)
    }
    
    h.logger.Info("HTTP routes registered successfully")
    return nil
}

// RegisterGRPC 注册gRPC服务
func (h *DomainServiceHandler) RegisterGRPC(server *grpc.Server) error {
    h.logger.Info("Registering gRPC service for domain service")
    
    // 如果需要gRPC支持，在这里注册
    // grpcHandler := grpcv1.NewDomainGRPCHandler(h.service, h.logger)
    // pb.RegisterDomainServiceServer(server, grpcHandler)
    
    h.logger.Info("gRPC service registered successfully")
    return nil
}

// IsHealthy 执行健康检查并返回健康状态
func (h *DomainServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    if !h.healthy {
        return &types.HealthStatus{
            Status:    "unhealthy",
            Timestamp: time.Now(),
            Version:   "v1.0.0",
            Message:   "service not initialized",
        }, fmt.Errorf("service not initialized")
    }
    
    // 检查依赖服务的健康状态
    if err := h.service.HealthCheck(ctx); err != nil {
        h.logger.Error("Domain service health check failed", "error", err)
        return &types.HealthStatus{
            Status:    "unhealthy",
            Timestamp: time.Now(),
            Version:   "v1.0.0",
            Message:   fmt.Sprintf("health check failed: %v", err),
        }, fmt.Errorf("domain service health check failed: %w", err)
    }
    
    return &types.HealthStatus{
        Status:       "healthy",
        Timestamp:    time.Now(),
        Version:      "v1.0.0",
        Uptime:       time.Since(h.startTime),
        Dependencies: make(map[string]types.DependencyStatus),
    }, nil
}

// Shutdown 优雅关闭服务
func (h *DomainServiceHandler) Shutdown(ctx context.Context) error {
    h.logger.Info("Shutting down domain service handler")
    
    h.healthy = false
    
    // 关闭服务资源
    if err := h.service.Shutdown(ctx); err != nil {
        h.logger.Error("Failed to shutdown domain service", "error", err)
        return fmt.Errorf("failed to shutdown domain service: %w", err)
    }
    
    h.logger.Info("Domain service handler shutdown completed")
    return nil
}
```

### 3. 依赖注入和服务注册

#### 3.1 依赖管理器

创建统一的依赖管理器来管理所有服务依赖：

```go
// internal/new-service/deps/deps.go
package deps

import (
    "fmt"
    "log/slog"
    
    "gorm.io/gorm"
    
    "your-project/internal/new-service/interfaces"
    "your-project/internal/new-service/repository"
    "your-project/internal/new-service/service/domain/v1"
    "your-project/internal/new-service/db"
)

// Dependencies 管理所有服务依赖
type Dependencies struct {
    // 基础设施
    DB     *gorm.DB
    Logger *slog.Logger
    
    // 仓储层
    DomainRepo repository.DomainRepository
    
    // 服务层
    DomainSrv interfaces.DomainService
}

// NewDependencies 创建新的依赖管理器
func NewDependencies(logger *slog.Logger) (*Dependencies, error) {
    // 1. 初始化基础设施
    database := db.GetDB()
    if database == nil {
        return nil, fmt.Errorf("failed to initialize database")
    }
    
    // 2. 初始化仓储层
    domainRepo := repository.NewDomainRepository(database, logger)
    
    // 3. 初始化服务层
    domainSrv, err := v1.NewDomainService(
        v1.WithRepository(domainRepo),
        v1.WithLogger(logger),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create domain service: %w", err)
    }
    
    return &Dependencies{
        DB:        database,
        Logger:    logger,
        DomainRepo: domainRepo,
        DomainSrv: domainSrv,
    }, nil
}

// Close 关闭所有资源
func (d *Dependencies) Close() error {
    if d.DB != nil {
        sqlDB, err := d.DB.DB()
        if err == nil {
            sqlDB.Close()
        }
    }
    return nil
}
```

#### 3.2 服务注册和启动

在服务器初始化中使用EnhancedServiceRegistry：

```go
// internal/new-service/server.go
package newservice

import (
    "context"
    "fmt"
    "log/slog"
    
    "your-project/internal/new-service/deps"
    "your-project/internal/new-service/handler"
    "your-project/internal/new-service/transport"
)

type Server struct {
    transportManager *transport.Manager
    serviceRegistry  *transport.EnhancedServiceRegistry
    deps             *deps.Dependencies
    logger           *slog.Logger
}

// NewServer 创建新的服务器实例
func NewServer(logger *slog.Logger) (*Server, error) {
    // 1. 初始化依赖
    dependencies, err := deps.NewDependencies(logger)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
    }
    
    server := &Server{
        transportManager: transport.NewManager(),
        serviceRegistry:  transport.NewEnhancedServiceRegistry(logger),
        deps:             dependencies,
        logger:           logger,
    }
    
    // 2. 注册服务
    if err := server.registerServices(); err != nil {
        return nil, fmt.Errorf("failed to register services: %w", err)
    }
    
    return server, nil
}

// registerServices 注册所有服务
func (s *Server) registerServices() error {
    // 注册领域服务
    domainHandler := handler.NewDomainServiceHandler(s.deps.DomainSrv, s.logger)
    if err := s.serviceRegistry.Register(domainHandler); err != nil {
        return fmt.Errorf("failed to register domain service: %w", err)
    }
    
    // 注册健康检查服务
    healthHandler := handler.NewHealthServiceHandler(s.logger)
    if err := s.serviceRegistry.Register(healthHandler); err != nil {
        return fmt.Errorf("failed to register health service: %w", err)
    }
    
    return nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
    // 初始化所有注册的服务
    if err := s.serviceRegistry.InitializeAll(ctx); err != nil {
        return fmt.Errorf("failed to initialize services: %w", err)
    }
    
    // 启动传输层
    httpTransport := transport.NewHTTPTransport(":8080", s.logger)
    s.transportManager.AddTransport(httpTransport)
    
    // 注册HTTP路由
    if err := s.serviceRegistry.RegisterAllHTTP(httpTransport.GetRouter()); err != nil {
        return fmt.Errorf("failed to register HTTP routes: %w", err)
    }
    
    // 启动所有传输层
    return s.transportManager.StartAll()
}

// Stop 停止服务器
func (s *Server) Stop(ctx context.Context) error {
    // 优雅关闭所有服务
    if err := s.serviceRegistry.ShutdownAll(ctx); err != nil {
        s.logger.Error("Failed to shutdown services", "error", err)
    }
    
    // 停止传输层
    if err := s.transportManager.StopAll(ctx); err != nil {
        s.logger.Error("Failed to stop transports", "error", err)
    }
    
    // 关闭依赖
    if err := s.deps.Close(); err != nil {
        s.logger.Error("Failed to close dependencies", "error", err)
    }
    
    return nil
}
```

### 4. 统一响应格式定义

**基于 `switauth` v1.0 的标准响应格式：**

```go
// internal/new-service/model/response.go
package model

import (
    "net/http"
    "time"
)

// StandardResponse 统一响应格式
type StandardResponse struct {
    Success   bool        `json:"success"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    Error     *ErrorInfo  `json:"error,omitempty"`
    Timestamp time.Time   `json:"timestamp"`
}

// ErrorInfo 错误信息结构
type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}, message string) *StandardResponse {
    return &StandardResponse{
        Success:   true,
        Message:   message,
        Data:      data,
        Timestamp: time.Now(),
    }
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code, message, details string) *StandardResponse {
    return &StandardResponse{
        Success: false,
        Message: message,
        Error: &ErrorInfo{
            Code:    code,
            Message: message,
            Details: details,
        },
        Timestamp: time.Now(),
    }
}
```

### 2. 版本化服务接口定义

基于 `switauth` v1.0 的服务接口模式：

```go
// internal/new-service/service/{domain-name}/v1/service.go
package v1

import (
    "context"
    "log/slog"
    
    "your-project/internal/new-service/model"
    "your-project/internal/new-service/repository"
)

// EntityResponse 领域实体响应
type EntityResponse struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DomainSrv v1 领域服务接口
type DomainSrv interface {
    GetEntity(ctx context.Context, id string) (*EntityResponse, error)
    ListEntities(ctx context.Context, limit, offset int) ([]*EntityResponse, error)
    CreateEntity(ctx context.Context, req *CreateEntityRequest) (*EntityResponse, error)
    UpdateEntity(ctx context.Context, id string, req *UpdateEntityRequest) (*EntityResponse, error)
    DeleteEntity(ctx context.Context, id string) error
}

// CreateEntityRequest 创建实体请求
type CreateEntityRequest struct {
    Name        string                 `json:"name" binding:"required"`
    Description string                 `json:"description"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateEntityRequest 更新实体请求
type UpdateEntityRequest struct {
    Name        *string                `json:"name,omitempty"`
    Description *string                `json:"description,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DomainServiceConfig 服务配置
type DomainServiceConfig struct {
    Repository repository.DomainRepository
    Logger     *slog.Logger
    Cache      CacheService
}

// DomainServiceOption 服务选项函数
type DomainServiceOption func(*domainService)

// WithRepository 设置仓储
func WithRepository(repo repository.DomainRepository) DomainServiceOption {
    return func(s *domainService) {
        s.repository = repo
    }
}

// WithServiceLogger 设置日志器
func WithServiceLogger(logger *slog.Logger) DomainServiceOption {
    return func(s *domainService) {
        s.logger = logger
    }
}

// WithCache 设置缓存服务
func WithCache(cache CacheService) DomainServiceOption {
    return func(s *domainService) {
        s.cache = cache
    }
}

// domainService 服务实现
type domainService struct {
    repository repository.DomainRepository
    logger     *slog.Logger
    cache      CacheService
}

// NewDomainSrv 创建领域服务实例
func NewDomainSrv(opts ...DomainServiceOption) DomainSrv {
    s := &domainService{
        logger: slog.Default(),
    }
    
    for _, opt := range opts {
        opt(s)
    }
    
    return s
}

// NewDomainSrvWithConfig 使用配置创建服务实例
func NewDomainSrvWithConfig(config *DomainServiceConfig) DomainSrv {
    return NewDomainSrv(
        WithRepository(config.Repository),
        WithServiceLogger(config.Logger),
        WithCache(config.Cache),
    )
}

// GetEntity 获取实体
func (s *domainService) GetEntity(ctx context.Context, id string) (*EntityResponse, error) {
    s.logger.InfoContext(ctx, "Getting entity", "id", id)
    
    // 尝试从缓存获取
    if s.cache != nil {
        if cached, err := s.cache.Get(ctx, "entity:"+id); err == nil {
            return cached.(*EntityResponse), nil
        }
    }
    
    // 从仓储获取
    entity, err := s.repository.GetByID(ctx, id)
    if err != nil {
        s.logger.ErrorContext(ctx, "Failed to get entity", "id", id, "error", err)
        return nil, fmt.Errorf("failed to get entity: %w", err)
    }
    
    response := &EntityResponse{
        ID:          entity.ID,
        Name:        entity.Name,
        Description: entity.Description,
        CreatedAt:   entity.CreatedAt,
        UpdatedAt:   entity.UpdatedAt,
        Metadata:    entity.Metadata,
    }
    
    // 缓存结果
    if s.cache != nil {
        s.cache.Set(ctx, "entity:"+id, response, 5*time.Minute)
    }
    
    return response, nil
}

// CreateEntity 创建实体
func (s *domainService) CreateEntity(ctx context.Context, req *CreateEntityRequest) (*EntityResponse, error) {
    s.logger.InfoContext(ctx, "Creating entity", "name", req.Name)
    
    entity := &model.Entity{
        ID:          generateID(),
        Name:        req.Name,
        Description: req.Description,
        Metadata:    req.Metadata,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    if err := s.repository.Create(ctx, entity); err != nil {
        s.logger.ErrorContext(ctx, "Failed to create entity", "error", err)
        return nil, fmt.Errorf("failed to create entity: %w", err)
    }
    
    return &EntityResponse{
        ID:          entity.ID,
        Name:        entity.Name,
        Description: entity.Description,
        CreatedAt:   entity.CreatedAt,
        UpdatedAt:   entity.UpdatedAt,
        Metadata:    entity.Metadata,
    }, nil
}
```

### 4. Transport Layer 实现

**HTTP Transport 模板：**

```go
// internal/new-service/transport/http.go
package transport

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/gin-gonic/gin"
)

type HTTPTransport struct {
    server    *http.Server
    router    *gin.Engine
    address   string
    testPort  string // 测试端口覆盖
    ready     chan struct{}
    readyOnce sync.Once
    mu        sync.RWMutex
}

func NewHTTPTransport() *HTTPTransport {
    router := gin.New()
    router.Use(gin.Recovery())
    
    return &HTTPTransport{
        router: router,
        ready:  make(chan struct{}),
    }
}

func (ht *HTTPTransport) Start(ctx context.Context) error {
    ht.mu.Lock()
    defer ht.mu.Unlock()
    
    address := ht.address
    if ht.testPort != "" {
        address = ":" + ht.testPort
    }
    
    ht.server = &http.Server{
        Addr:    address,
        Handler: ht.router,
    }
    
    go func() {
        ht.readyOnce.Do(func() {
            close(ht.ready)
        })
        
        if err := ht.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            // 记录错误日志
        }
    }()
    
    return nil
}

func (ht *HTTPTransport) Stop(ctx context.Context) error {
    ht.mu.Lock()
    defer ht.mu.Unlock()
    
    if ht.server != nil {
        return ht.server.Shutdown(ctx)
    }
    return nil
}

func (ht *HTTPTransport) Name() string {
    return "http"
}

func (ht *HTTPTransport) Address() string {
    ht.mu.RLock()
    defer ht.mu.RUnlock()
    return ht.address
}

func (ht *HTTPTransport) SetAddress(addr string) {
    ht.mu.Lock()
    defer ht.mu.Unlock()
    ht.address = addr
}

func (ht *HTTPTransport) GetRouter() *gin.Engine {
    return ht.router
}

func (ht *HTTPTransport) WaitReady(timeout time.Duration) error {
    select {
    case <-ht.ready:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("http transport not ready within %v", timeout)
    }
}
```

**gRPC Transport 模板：**

```go
// internal/new-service/transport/grpc.go
package transport

import (
    "context"
    "net"
    "sync"
    
    "google.golang.org/grpc"
)

type GRPCTransport struct {
    server   *grpc.Server
    listener net.Listener
    address  string
    testPort string
    mu       sync.RWMutex
}

func NewGRPCTransport() *GRPCTransport {
    return &GRPCTransport{
        server: grpc.NewServer(),
    }
}

func (gt *GRPCTransport) Start(ctx context.Context) error {
    gt.mu.Lock()
    defer gt.mu.Unlock()
    
    address := gt.address
    if gt.testPort != "" {
        address = ":" + gt.testPort
    }
    
    listener, err := net.Listen("tcp", address)
    if err != nil {
        return err
    }
    
    gt.listener = listener
    
    go func() {
        if err := gt.server.Serve(listener); err != nil {
            // 记录错误日志
        }
    }()
    
    return nil
}

func (gt *GRPCTransport) Stop(ctx context.Context) error {
    gt.mu.Lock()
    defer gt.mu.Unlock()
    
    if gt.server != nil {
        gt.server.GracefulStop()
    }
    return nil
}

func (gt *GRPCTransport) Name() string {
    return "grpc"
}

func (gt *GRPCTransport) Address() string {
    gt.mu.RLock()
    defer gt.mu.RUnlock()
    return gt.address
}

func (gt *GRPCTransport) SetAddress(addr string) {
    gt.mu.Lock()
    defer gt.mu.Unlock()
    gt.address = addr
}

func (gt *GRPCTransport) GetServer() *grpc.Server {
    return gt.server
}
```

### 5. HTTP 处理器实现

**基于 `switauth` v1.0 的处理器模式：**

```go
// internal/new-service/handler/http/v1/{domain-name}.go
package v1

import (
    "log/slog"
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    
    "your-project/internal/new-service/model"
    servicev1 "your-project/internal/new-service/service/{domain-name}/v1"
)

// DomainHandler v1 领域处理器
type DomainHandler struct {
    service servicev1.DomainSrv
    logger  *slog.Logger
}

// DomainHandlerOption 处理器选项函数
type DomainHandlerOption func(*DomainHandler)

// WithDomainService 设置领域服务
func WithDomainService(service servicev1.DomainSrv) DomainHandlerOption {
    return func(h *DomainHandler) {
        h.service = service
    }
}

// WithHandlerLogger 设置日志器
func WithHandlerLogger(logger *slog.Logger) DomainHandlerOption {
    return func(h *DomainHandler) {
        h.logger = logger
    }
}

// NewDomainHandler 创建领域处理器
func NewDomainHandler(opts ...DomainHandlerOption) *DomainHandler {
    h := &DomainHandler{
        logger: slog.Default(),
    }
    
    for _, opt := range opts {
        opt(h)
    }
    
    return h
}

// GetEntity 获取实体
func (h *DomainHandler) GetEntity(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        response := model.NewErrorResponse("INVALID_PARAMETER", "ID parameter is required", "")
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    entity, err := h.service.GetEntity(c.Request.Context(), id)
    if err != nil {
        h.logger.ErrorContext(c.Request.Context(), "Failed to get entity", "id", id, "error", err)
        response := model.NewErrorResponse("INTERNAL_ERROR", "Failed to get entity", err.Error())
        c.JSON(http.StatusInternalServerError, response)
        return
    }
    
    response := model.NewSuccessResponse(entity, "Entity retrieved successfully")
    c.JSON(http.StatusOK, response)
}

// ListEntities 列出实体
func (h *DomainHandler) ListEntities(c *gin.Context) {
    limitStr := c.DefaultQuery("limit", "10")
    offsetStr := c.DefaultQuery("offset", "0")
    
    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 {
        response := model.NewErrorResponse("INVALID_PARAMETER", "Invalid limit parameter", "")
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    offset, err := strconv.Atoi(offsetStr)
    if err != nil || offset < 0 {
        response := model.NewErrorResponse("INVALID_PARAMETER", "Invalid offset parameter", "")
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    entities, err := h.service.ListEntities(c.Request.Context(), limit, offset)
    if err != nil {
        h.logger.ErrorContext(c.Request.Context(), "Failed to list entities", "error", err)
        response := model.NewErrorResponse("INTERNAL_ERROR", "Failed to list entities", err.Error())
        c.JSON(http.StatusInternalServerError, response)
        return
    }
    
    response := model.NewSuccessResponse(entities, "Entities retrieved successfully")
    c.JSON(http.StatusOK, response)
}

// CreateEntity 创建实体
func (h *DomainHandler) CreateEntity(c *gin.Context) {
    var req servicev1.CreateEntityRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response := model.NewErrorResponse("INVALID_REQUEST", "Invalid request body", err.Error())
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    entity, err := h.service.CreateEntity(c.Request.Context(), &req)
    if err != nil {
        h.logger.ErrorContext(c.Request.Context(), "Failed to create entity", "error", err)
        response := model.NewErrorResponse("INTERNAL_ERROR", "Failed to create entity", err.Error())
        c.JSON(http.StatusInternalServerError, response)
        return
    }
    
    response := model.NewSuccessResponse(entity, "Entity created successfully")
    c.JSON(http.StatusCreated, response)
}

// UpdateEntity 更新实体
func (h *DomainHandler) UpdateEntity(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        response := model.NewErrorResponse("INVALID_PARAMETER", "ID parameter is required", "")
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    var req servicev1.UpdateEntityRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response := model.NewErrorResponse("INVALID_REQUEST", "Invalid request body", err.Error())
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    entity, err := h.service.UpdateEntity(c.Request.Context(), id, &req)
    if err != nil {
        h.logger.ErrorContext(c.Request.Context(), "Failed to update entity", "id", id, "error", err)
        response := model.NewErrorResponse("INTERNAL_ERROR", "Failed to update entity", err.Error())
        c.JSON(http.StatusInternalServerError, response)
        return
    }
    
    response := model.NewSuccessResponse(entity, "Entity updated successfully")
    c.JSON(http.StatusOK, response)
}

// DeleteEntity 删除实体
func (h *DomainHandler) DeleteEntity(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        response := model.NewErrorResponse("INVALID_PARAMETER", "ID parameter is required", "")
        c.JSON(http.StatusBadRequest, response)
        return
    }
    
    if err := h.service.DeleteEntity(c.Request.Context(), id); err != nil {
        h.logger.ErrorContext(c.Request.Context(), "Failed to delete entity", "id", id, "error", err)
        response := model.NewErrorResponse("INTERNAL_ERROR", "Failed to delete entity", err.Error())
        c.JSON(http.StatusInternalServerError, response)
        return
    }
    
    response := model.NewSuccessResponse(nil, "Entity deleted successfully")
    c.JSON(http.StatusOK, response)
}
```

### 6. ServiceHandler接口实现

每个服务都需要实现`ServiceHandler`接口来支持统一的服务管理：

```go
// internal/new-service/handler/domain_handler.go
package handler

import (
    "context"
    "log/slog"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    
    "your-project/internal/new-service/interfaces"
    "your-project/internal/switserve/transport"
)

// DomainServiceHandler 实现ServiceHandler接口
type DomainServiceHandler struct {
    service interfaces.DomainService
    logger  *slog.Logger
}

// NewDomainServiceHandler 创建新的领域服务处理器
func NewDomainServiceHandler(service interfaces.DomainService, logger *slog.Logger) *DomainServiceHandler {
    return &DomainServiceHandler{
        service: service,
        logger:  logger,
    }
}

// GetMetadata 返回服务元数据
func (h *DomainServiceHandler) GetMetadata() *transport.ServiceMetadata {
    return &transport.ServiceMetadata{
        Name:        "domain-service",
        Version:     "v1.0.0",
        Description: "Domain business logic service",
        Tags:        []string{"domain", "business"},
    }
}

// Initialize 初始化服务
func (h *DomainServiceHandler) Initialize() error {
    h.logger.Info("Initializing domain service handler")
    // 执行服务初始化逻辑
    return nil
}

// RegisterHTTP 注册HTTP路由
func (h *DomainServiceHandler) RegisterHTTP(router *gin.Engine) error {
    v1 := router.Group("/api/v1/domain")
    {
        v1.GET("/entities", h.listEntities)
        v1.POST("/entities", h.createEntity)
        v1.GET("/entities/:id", h.getEntity)
        v1.PUT("/entities/:id", h.updateEntity)
        v1.DELETE("/entities/:id", h.deleteEntity)
    }
    
    h.logger.Info("Domain service HTTP routes registered")
    return nil
}

// RegisterGRPC 注册gRPC服务
func (h *DomainServiceHandler) RegisterGRPC(server *grpc.Server) error {
    // 注册gRPC服务实现
    // pb.RegisterDomainServiceServer(server, h)
    
    h.logger.Info("Domain service gRPC handlers registered")
    return nil
}

// HealthCheck 执行健康检查
func (h *DomainServiceHandler) HealthCheck(ctx context.Context) error {
    // 检查服务依赖的健康状态
    return h.service.HealthCheck(ctx)
}

// IsHealthy 返回服务健康状态
func (h *DomainServiceHandler) IsHealthy() bool {
    return h.service.IsHealthy()
}

// Shutdown 优雅关闭服务
func (h *DomainServiceHandler) Shutdown() error {
    h.logger.Info("Shutting down domain service handler")
    return h.service.Shutdown()
}

// HTTP处理器方法
func (h *DomainServiceHandler) listEntities(c *gin.Context) {
    // 实现列表查询逻辑
}

func (h *DomainServiceHandler) createEntity(c *gin.Context) {
    // 实现创建逻辑
}

func (h *DomainServiceHandler) getEntity(c *gin.Context) {
    // 实现单个查询逻辑
}

func (h *DomainServiceHandler) updateEntity(c *gin.Context) {
    // 实现更新逻辑
}

func (h *DomainServiceHandler) deleteEntity(c *gin.Context) {
    // 实现删除逻辑
}
```

#### 健康检查服务Handler示例

```go
// internal/new-service/handler/health_handler.go
package handler

import (
    "context"
    "log/slog"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    
    "your-project/internal/switserve/transport"
)

// HealthServiceHandler 健康检查服务处理器
type HealthServiceHandler struct {
    logger *slog.Logger
}

// NewHealthServiceHandler 创建健康检查服务处理器
func NewHealthServiceHandler(logger *slog.Logger) *HealthServiceHandler {
    return &HealthServiceHandler{
        logger: logger,
    }
}

// GetMetadata 返回服务元数据
func (h *HealthServiceHandler) GetMetadata() *transport.ServiceMetadata {
    return &transport.ServiceMetadata{
        Name:        "health-service",
        Version:     "v1.0.0",
        Description: "Health check service",
        Tags:        []string{"health", "monitoring"},
    }
}

// Initialize 初始化服务
func (h *HealthServiceHandler) Initialize() error {
    h.logger.Info("Initializing health service handler")
    return nil
}

// RegisterHTTP 注册HTTP路由
func (h *HealthServiceHandler) RegisterHTTP(router *gin.Engine) error {
    router.GET("/health", h.healthCheck)
    router.GET("/ready", h.readinessCheck)
    
    h.logger.Info("Health service HTTP routes registered")
    return nil
}

// RegisterGRPC 注册gRPC服务
func (h *HealthServiceHandler) RegisterGRPC(server *grpc.Server) error {
    // 健康检查服务通常不需要gRPC接口
    return nil
}

// HealthCheck 执行健康检查
func (h *HealthServiceHandler) HealthCheck(ctx context.Context) error {
    // 检查系统健康状态
    return nil
}

// IsHealthy 返回服务健康状态
func (h *HealthServiceHandler) IsHealthy() bool {
    return true
}

// Shutdown 优雅关闭服务
func (h *HealthServiceHandler) Shutdown() error {
    h.logger.Info("Shutting down health service handler")
    return nil
}

// HTTP处理器方法
func (h *HealthServiceHandler) healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "timestamp": time.Now(),
    })
}

func (h *HealthServiceHandler) readinessCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "ready",
        "timestamp": time.Now(),
    })
}
```

### 7. ServiceRegistry 实现

**注意：必须包含并发安全保护**

```go
// internal/new-service/transport/registry.go
package transport

import (
    "fmt"
    "sync"
    
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
)

type ServiceRegistry struct {
    mu         sync.RWMutex // 必须包含锁保护
    registrars []ServiceRegistrar
}

func NewServiceRegistry() *ServiceRegistry {
    return &ServiceRegistry{
        registrars: make([]ServiceRegistrar, 0),
    }
}

func (sr *ServiceRegistry) Register(registrar ServiceRegistrar) {
    sr.mu.Lock()
    defer sr.mu.Unlock()
    sr.registrars = append(sr.registrars, registrar)
}

func (sr *ServiceRegistry) RegisterAllHTTP(router *gin.Engine) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    for _, registrar := range sr.registrars {
        if err := registrar.RegisterHTTP(router); err != nil {
            return fmt.Errorf("failed to register HTTP for %s: %w", registrar.GetName(), err)
        }
    }
    return nil
}

func (sr *ServiceRegistry) RegisterAllGRPC(server *grpc.Server) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    for _, registrar := range sr.registrars {
        if err := registrar.RegisterGRPC(server); err != nil {
            return fmt.Errorf("failed to register gRPC for %s: %w", registrar.GetName(), err)
        }
    }
    return nil
}

func (sr *ServiceRegistry) GetRegistrars() []ServiceRegistrar {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    result := make([]ServiceRegistrar, len(sr.registrars))
    copy(result, sr.registrars)
    return result
}
```

### 8. Server 主文件实现

```go
// internal/new-service/server.go
package newservice

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "your-project/internal/new-service/config"
    "your-project/internal/new-service/service/example"
    "your-project/internal/new-service/service/health"
    "your-project/internal/new-service/transport"
    "your-project/pkg/discovery"
    "your-project/pkg/middleware"
)

type Server struct {
    transportManager *transport.Manager
    serviceRegistry  *transport.ServiceRegistry
    httpTransport    *transport.HTTPTransport
    grpcTransport    *transport.GRPCTransport
    sd               *discovery.ServiceDiscovery
    config           *config.Config
}

func NewServer() (*Server, error) {
    cfg := config.GetConfig()
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    // 创建传输管理器和服务注册表
    transportManager := transport.NewManager()
    serviceRegistry := transport.NewServiceRegistry()
    
    // 创建传输层
    httpTransport := transport.NewHTTPTransport()
    httpTransport.SetAddress(":" + cfg.Server.Port)
    
    grpcTransport := transport.NewGRPCTransport()
    grpcTransport.SetAddress(":" + cfg.Server.GRPCPort)
    
    // 注册传输层
    transportManager.Register(httpTransport)
    transportManager.Register(grpcTransport)
    
    // 设置服务发现
    sd, err := discovery.GetServiceDiscoveryByAddress(cfg.ServiceDiscovery.Address)
    if err != nil {
        return nil, fmt.Errorf("failed to setup service discovery: %w", err)
    }
    
    server := &Server{
        transportManager: transportManager,
        serviceRegistry:  serviceRegistry,
        httpTransport:    httpTransport,
        grpcTransport:    grpcTransport,
        sd:               sd,
        config:           cfg,
    }
    
    // 注册服务
    if err := server.registerServices(); err != nil {
        return nil, fmt.Errorf("failed to register services: %w", err)
    }
    
    return server, nil
}

func (s *Server) registerServices() error {
    // 注册健康检查服务
    healthService := health.NewServiceRegistrar()
    s.serviceRegistry.Register(healthService)
    
    // 注册业务服务
    exampleService := example.NewServiceRegistrar(/* 依赖注入 */)
    s.serviceRegistry.Register(exampleService)
    
    return nil
}

func (s *Server) configureMiddleware() {
    router := s.httpTransport.GetRouter()
    
    // 注册全局中间件
    middlewareRegistrar := middleware.NewGlobalMiddlewareRegistrar()
    if err := middlewareRegistrar.RegisterMiddleware(router); err != nil {
        // 记录错误日志
    }
}

func (s *Server) Start(ctx context.Context) error {
    // 注册服务到传输层
    if err := s.serviceRegistry.RegisterAllHTTP(s.httpTransport.GetRouter()); err != nil {
        return fmt.Errorf("failed to register HTTP services: %w", err)
    }
    
    if err := s.serviceRegistry.RegisterAllGRPC(s.grpcTransport.GetServer()); err != nil {
        return fmt.Errorf("failed to register gRPC services: %w", err)
    }
    
    // 配置中间件
    s.configureMiddleware()
    
    // 启动传输层
    if err := s.transportManager.Start(ctx); err != nil {
        return fmt.Errorf("failed to start transports: %w", err)
    }
    
    // 注册到服务发现
    if err := s.registerWithDiscovery(); err != nil {
        return fmt.Errorf("failed to register with discovery: %w", err)
    }
    
    return nil
}

func (s *Server) registerWithDiscovery() error {
    // HTTP 服务注册
    httpServiceID := fmt.Sprintf("%s-http-%s", s.config.ServiceName, s.config.Server.Port)
    if err := s.sd.RegisterService(httpServiceID, s.config.ServiceName, s.config.Server.Port, "http"); err != nil {
        return fmt.Errorf("failed to register HTTP service: %w", err)
    }
    
    // gRPC 服务注册
    grpcServiceID := fmt.Sprintf("%s-grpc-%s", s.config.ServiceName, s.config.Server.GRPCPort)
    if err := s.sd.RegisterService(grpcServiceID, s.config.ServiceName, s.config.Server.GRPCPort, "grpc"); err != nil {
        return fmt.Errorf("failed to register gRPC service: %w", err)
    }
    
    return nil
}

func (s *Server) Shutdown() error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // 从服务发现注销
    if err := s.deregisterFromDiscovery(); err != nil {
        // 记录错误日志，但不阻止关闭流程
    }
    
    // 停止传输层
    return s.transportManager.Stop(ctx)
}

func (s *Server) deregisterFromDiscovery() error {
    httpServiceID := fmt.Sprintf("%s-http-%s", s.config.ServiceName, s.config.Server.Port)
    grpcServiceID := fmt.Sprintf("%s-grpc-%s", s.config.ServiceName, s.config.Server.GRPCPort)
    
    if err := s.sd.DeregisterService(httpServiceID); err != nil {
        return fmt.Errorf("failed to deregister HTTP service: %w", err)
    }
    
    if err := s.sd.DeregisterService(grpcServiceID); err != nil {
        return fmt.Errorf("failed to deregister gRPC service: %w", err)
    }
    
    return nil
}

func (s *Server) Run() error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // 启动服务器
    if err := s.Start(ctx); err != nil {
        return err
    }
    
    // 等待信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigChan
    
    // 优雅关闭
    return s.Shutdown()
}
```

### 5. 配置管理

```go
// internal/new-service/config/config.go
package config

import (
    "errors"
    "sync"
    
    "github.com/spf13/viper"
)

type Config struct {
    ServiceName string `json:"serviceName" yaml:"serviceName"`
    Database    struct {
        Host     string `json:"host" yaml:"host"`
        Port     string `json:"port" yaml:"port"`
        Username string `json:"username" yaml:"username"`
        Password string `json:"password" yaml:"password"`
        DBName   string `json:"dbname" yaml:"dbname"`
    } `json:"database" yaml:"database"`
    Server struct {
        Port     string `json:"port" yaml:"port"`
        GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
    } `json:"server" yaml:"server"`
    ServiceDiscovery struct {
        Address string `json:"address" yaml:"address"`
    } `json:"serviceDiscovery" yaml:"serviceDiscovery"`
}

var (
    config *Config
    once   sync.Once
)

func GetConfig() *Config {
    once.Do(func() {
        config = loadConfig()
    })
    return config
}

func loadConfig() *Config {
    viper.SetConfigName("new-service") // 配置文件名
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./configs")
    viper.AddConfigPath("/etc/new-service")
    
    // 设置默认值
    viper.SetDefault("serviceName", "new-service")
    viper.SetDefault("server.port", "8080")
    viper.SetDefault("server.grpcPort", "9090")
    viper.SetDefault("serviceDiscovery.address", "localhost:8500")
    
    if err := viper.ReadInConfig(); err != nil {
        // 记录警告日志，使用默认配置
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        panic(err)
    }
    
    return &cfg
}

func (c *Config) Validate() error {
    if c.ServiceName == "" {
        return errors.New("service name is required")
    }
    
    if c.Server.Port == "" {
        return errors.New("server port is required")
    }
    
    if c.Server.GRPCPort == "" {
        return errors.New("gRPC port is required")
    }
    
    if c.ServiceDiscovery.Address == "" {
        return errors.New("service discovery address is required")
    }
    
    return nil
}
```

## 实施步骤

### 第一阶段：项目初始化

1. **创建项目目录结构**
   ```bash
   mkdir -p internal/new-service/{cmd,config,db,handler/http/v1,model,repository,service/v1,transport}
   mkdir -p cmd/new-service
   ```

2. **创建配置文件**
   ```yaml
   # new-service.yaml
   serviceName: "new-service"
   database:
     host: "localhost"
     port: "3306"
     username: "root"
     password: "password"
     dbname: "new_service_db"
   server:
     port: "8080"
     grpcPort: "9090"
   serviceDiscovery:
     address: "localhost:8500"
   ```

3. **实现基础传输层**
   - 复制并修改 `transport` 包
   - 确保包含线程安全保护
   - 添加适当的错误处理

### 第二阶段：服务层开发

1. **按领域划分服务结构**
    - 在 `service/` 目录下按业务领域创建子目录（如 `{domain-a}/`、`{domain-b}/` 等）
    - 对于需要版本化的业务服务，在领域目录下创建版本子目录（如 `v1/`）
    - 对于通用服务（如 `health/`、监控、停止等），可直接在领域目录下实现，无需版本化
    - 在每个领域目录下实现 `registrar.go` 负责服务注册

2. **定义版本化业务接口**
   ```go
   // internal/new-service/service/{domain-name}/v1/service.go
   type DomainSrv interface {
       GetEntity(ctx context.Context, id string) (*EntityResponse, error)
       ListEntities(ctx context.Context, limit, offset int) ([]*EntityResponse, error)
       CreateEntity(ctx context.Context, req *CreateEntityRequest) (*EntityResponse, error)
       UpdateEntity(ctx context.Context, id string, req *UpdateEntityRequest) (*EntityResponse, error)
       DeleteEntity(ctx context.Context, id string) error
   }
   ```

3. **实现统一响应格式**
   - 基于 `switauth` v1.0 的 `StandardResponse` 格式
   - 统一的错误码和消息结构
   - 包含时间戳和成功标识

4. **实现选项模式依赖注入**
   - 使用函数选项模式进行依赖注入
   - 支持灵活的配置和扩展
   - 便于单元测试和模块替换

5. **配置中间件使用**
   - 优先使用 `pkg/middleware` 中的通用中间件
   - 使用 `GlobalMiddlewareRegistrar` 注册全局中间件
   - 根据需要配置认证中间件的白名单和服务发现
   - 仅在必要时创建项目特定中间件

6. **实现 ServiceHandler 接口**
   - 每个服务实现 ServiceHandler 接口
   - 支持 HTTP 和 gRPC 双协议注册
   - 包含服务元数据、初始化、健康检查和优雅关闭
   - 使用 EnhancedServiceRegistry 进行统一管理

7. **实现处理器层**
   - HTTP 处理器使用 Gin 和版本化路由
   - gRPC 处理器实现 protobuf 接口
   - 统一错误处理和响应格式
   - 集成到 ServiceHandler 接口中

### 第三阶段：集成和测试

1. **集成服务发现**
   - 使用现有的 `pkg/discovery` 包
   - 实现服务注册和注销
   - 添加健康检查端点

2. **添加中间件**
   - 使用 `pkg/middleware` 包
   - 配置认证、日志、CORS 等中间件
   - 确保中间件顺序正确

3. **编写测试**
   - 单元测试覆盖所有组件
   - 集成测试验证端到端流程
   - 性能测试确保满足要求

### 第四阶段：部署准备

1. **Docker 化**
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o new-service cmd/new-service/new-service.go
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   WORKDIR /root/
   COPY --from=builder /app/new-service .
   COPY --from=builder /app/new-service.yaml .
   CMD ["./new-service"]
   ```

2. **Kubernetes 配置**
   - 创建 Deployment、Service、ConfigMap
   - 配置健康检查和资源限制
   - 设置环境变量和密钥管理

3. **监控和日志**
   - 集成 Prometheus 指标
   - 配置结构化日志
   - 添加链路追踪

## 最佳实践清单

### ✅ 必须遵循的规范

- [ ] 使用版本化设计模式（v1、v2 等）
- [ ] 实现统一的 `StandardResponse` 响应格式
- [ ] 使用选项模式进行依赖注入
- [ ] 优先使用 `pkg/middleware` 中的通用中间件
- [ ] 使用统一的 `ServiceRegistrar` 接口
- [ ] 实现线程安全的 `ServiceRegistry`
- [ ] 支持 HTTP 和 gRPC 双协议
- [ ] 集成服务发现机制
- [ ] 实现优雅关闭
- [ ] 添加配置验证
- [ ] 包含健康检查端点
- [ ] 实现统一错误处理和错误码
- [ ] 添加结构化日志（使用 `slog`）
- [ ] 包含完整的单元测试覆盖

### ✅ 推荐的增强功能

- [ ] 添加 Prometheus 指标监控
- [ ] 实现分布式链路追踪
- [ ] 配置熔断器和限流
- [ ] 添加多级缓存层
- [ ] 实现数据库迁移和版本管理
- [ ] 配置 Swagger/OpenAPI 文档生成
- [ ] 添加性能测试和压力测试
- [ ] 实现配置热重载
- [ ] 添加请求验证中间件
- [ ] 实现 API 版本兼容性检查

### ⚠️ 常见陷阱避免

- **版本兼容性**: 确保 API 版本变更时保持向后兼容
- **响应格式一致性**: 严格遵循 `StandardResponse` 格式，避免不一致的响应结构
- **选项模式误用**: 正确使用函数选项模式，避免过度复杂化
- **并发安全**: 确保所有共享资源都有适当的锁保护
- **资源泄漏**: 正确关闭数据库连接、HTTP 客户端等资源
- **错误处理**: 使用统一的错误码和消息格式，提供有意义的错误上下文
- **配置管理**: 不要硬编码配置，使用环境变量或配置文件
- **测试覆盖**: 确保关键路径都有测试覆盖，特别是错误处理路径
- **依赖循环**: 避免包之间的循环依赖
- **接口设计**: 保持接口简单和专注，遵循单一职责原则
- **日志记录**: 使用结构化日志，避免敏感信息泄露

## 示例项目模板

为了快速开始，可以参考以下命令创建项目模板：

```bash
# 1. 创建项目结构
mkdir -p internal/new-service/{cmd,config,db,handler/http,model,repository,service,transport}
mkdir -p cmd/new-service

# 2. 复制基础文件
cp internal/switserve/transport/* internal/new-service/transport/
cp internal/switserve/config/* internal/new-service/config/

# 3. 修改包名和配置
# 使用 sed 或手动修改文件中的包名引用

# 4. 创建配置文件
cp switserve.yaml new-service.yaml
# 修改配置文件中的服务名称和端口

# 5. 创建主程序
cp cmd/swit-serve/swit-serve.go cmd/new-service/new-service.go
# 修改导入路径和服务名称
```

## 总结

遵循本指导方案可以确保新项目：

1. **架构一致性**: 与现有项目保持相同的设计模式，特别是基于 `switauth` v1.0 重构后的成熟架构
2. **版本化设计**: 采用统一的版本化接口设计，支持 API 演进和向后兼容
3. **标准化响应**: 使用统一的 `StandardResponse` 格式，确保 API 响应的一致性
4. **现代化依赖注入**: 采用选项模式进行依赖注入，提高代码的灵活性和可测试性
5. **代码复用**: 最大化利用现有的基础设施代码和最佳实践
6. **可维护性**: 清晰的分层和职责分离，遵循 SOLID 原则
7. **可扩展性**: 易于添加新功能和服务，支持水平扩展
8. **生产就绪**: 包含监控、日志、健康检查、错误处理等生产环境必需功能
9. **测试友好**: 完整的测试覆盖和易于测试的架构设计

### 关键优势

- **基于成熟实践**: 基于 `switauth` v1.0 重构后的成熟架构模式
- **统一标准**: 统一的接口设计、响应格式和错误处理机制
- **现代化模式**: 采用最新的 Go 语言最佳实践和设计模式
- **完整覆盖**: 从项目初始化到生产部署的完整指导

建议在开始新项目开发前，先深入研究 `switauth` v1.0 重构后的代码，理解其版本化设计思路和选项模式实现，然后按照本指南逐步实施。如有疑问，可以参考 `switauth` 和 `switserve` 的具体实现，特别关注 `switauth` 的最新架构模式。