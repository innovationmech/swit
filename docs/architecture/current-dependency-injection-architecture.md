# SWIT 项目依赖注入与服务注册架构文档

## 概述

本文档详细描述了 SWIT 项目当前的依赖注入和服务注册架构。该架构采用统一的 `ServiceHandler` 接口模式，通过 `EnhancedServiceRegistry` 实现服务的统一管理，提供完整的服务生命周期管理、健康检查和优雅关闭功能。

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                    服务器初始化流程                          │
├─────────────────────────────────────────────────────────────┤
│ 1. NewDependencies() 创建所有依赖                           │
│ 2. NewServer() 初始化服务器和传输层                         │
│ 3. registerServices() 注册所有服务处理器                    │
│ 4. Start() 启动所有传输层和服务                             │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                  依赖管理层 (Dependencies)                   │
├─────────────────────────────────────────────────────────────┤
│ • 基础设施 (数据库、配置、服务发现)                          │
│ • 仓储层 (Repository Layer)                                 │
│ • 服务层 (Service Layer)                                   │
│ • 资源生命周期管理                                          │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│               服务处理器层 (ServiceHandler)                  │
├─────────────────────────────────────────────────────────────┤
│ • HTTP/gRPC 路由注册                                        │
│ • 服务元数据管理                                            │
│ • 健康检查实现                                              │
│ • 生命周期管理 (初始化/关闭)                                 │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│            增强服务注册表 (EnhancedServiceRegistry)          │
├─────────────────────────────────────────────────────────────┤
│ • 线程安全的服务管理                                        │
│ • 统一的路由注册                                            │
│ • 批量健康检查                                              │
│ • 优雅关闭管理                                              │
└─────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. ServiceHandler 接口

`ServiceHandler` 是整个架构的核心接口，定义了服务的统一管理规范：

```go
// ServiceHandler 定义服务注册和管理的统一接口
type ServiceHandler interface {
    // RegisterHTTP 注册 HTTP 路由
    RegisterHTTP(router *gin.Engine) error
    
    // RegisterGRPC 注册 gRPC 服务
    RegisterGRPC(server *grpc.Server) error
    
    // GetMetadata 返回服务元数据
    GetMetadata() *ServiceMetadata
    
    // GetHealthEndpoint 返回健康检查端点
    GetHealthEndpoint() string
    
    // IsHealthy 执行健康检查
    IsHealthy(ctx context.Context) (*types.HealthStatus, error)
    
    // Initialize 初始化服务
    Initialize(ctx context.Context) error
    
    // Shutdown 优雅关闭服务
    Shutdown(ctx context.Context) error
}

// ServiceMetadata 服务元数据
type ServiceMetadata struct {
    Name           string            `json:"name"`
    Version        string            `json:"version"`
    Description    string            `json:"description"`
    HealthEndpoint string            `json:"health_endpoint"`
    Tags           []string          `json:"tags,omitempty"`
    Dependencies   []string          `json:"dependencies,omitempty"`
}
```

#### 接口特性

- **统一性**: 所有服务都实现相同的接口，确保一致的管理方式
- **生命周期管理**: 支持初始化、健康检查和优雅关闭
- **元数据驱动**: 通过元数据提供服务发现和监控信息
- **传输层无关**: 同时支持 HTTP 和 gRPC 协议

### 2. 依赖管理器 (Dependencies)

依赖管理器负责创建和管理所有服务依赖，采用分层初始化模式：

#### switserve 项目依赖结构

```go
// internal/switserve/deps/deps.go
type Dependencies struct {
    // 基础设施层
    DB *gorm.DB
    
    // 仓储层
    UserRepo repository.UserRepository
    
    // 服务层 - 使用接口类型实现更好的依赖注入
    UserSrv         interfaces.UserService
    GreeterSrv      interfaces.GreeterService
    NotificationSrv notificationv1.NotificationService
    HealthSrv       interfaces.HealthService
    StopSrv         interfaces.StopService
}

func NewDependencies(shutdownFunc func()) (*Dependencies, error) {
    // 1. 初始化基础设施
    database := db.GetDB()
    if database == nil {
        return nil, ErrDatabaseConnection
    }
    
    // 2. 初始化仓储层
    userRepo := repository.NewUserRepository(database)
    
    // 3. 初始化服务层
    userSrv, err := userv1.NewUserSrv(
        userv1.WithUserRepository(userRepo),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create user service: %w", err)
    }
    
    // 初始化其他服务
    greeterSrv := greeterv1.NewService()
    notificationSrv := notificationv1.NewService()
    healthSrv := health.NewHealthSrv()
    stopSrv := stop.NewService(shutdownFunc)
    
    return &Dependencies{
        DB:              database,
        UserRepo:        userRepo,
        UserSrv:         userSrv,
        GreeterSrv:      greeterSrv,
        NotificationSrv: notificationSrv,
        HealthSrv:       healthSrv,
        StopSrv:         stopSrv,
    }, nil
}
```

#### switauth 项目依赖结构

```go
// internal/switauth/deps/deps.go
type Dependencies struct {
    // 核心依赖
    DB     *gorm.DB
    Config *config.AuthConfig
    SD     *discovery.ServiceDiscovery
    
    // 仓储层
    TokenRepo repository.TokenRepository
    
    // 客户端层
    UserClient client.UserClient
    
    // 服务层
    AuthSrv   interfaces.AuthService
    HealthSrv interfaces.HealthService
}

func NewDependencies() (*Dependencies, error) {
    // 初始化配置
    cfg := config.GetConfig()
    
    // 初始化数据库
    database := db.GetDB()
    
    // 初始化服务发现
    sd, err := discovery.GetServiceDiscoveryByAddress(cfg.ServiceDiscovery.Address)
    if err != nil {
        return nil, err
    }
    
    // 初始化仓储层
    tokenRepo := repository.NewTokenRepository(database)
    
    // 初始化客户端层
    userClient := client.NewUserClient(sd)
    
    // 初始化服务层
    authSrv, err := authv1.NewAuthSrv(
        authv1.WithUserClient(userClient),
        authv1.WithTokenRepository(tokenRepo),
    )
    if err != nil {
        return nil, err
    }
    
    healthSrv := health.NewHealthService()
    
    return &Dependencies{
        DB:         database,
        Config:     cfg,
        SD:         sd,
        TokenRepo:  tokenRepo,
        UserClient: userClient,
        AuthSrv:    authSrv,
        HealthSrv:  healthSrv,
    }, nil
}
```

#### 依赖管理特性

- **分层初始化**: 基础设施 → 仓储层 → 服务层的有序初始化
- **错误处理**: 完整的错误处理和验证机制
- **资源管理**: 支持资源清理和优雅关闭
- **接口驱动**: 使用接口类型提高可测试性和可扩展性

### 3. 增强服务注册表 (EnhancedServiceRegistry)

`EnhancedServiceRegistry` 是服务管理的核心组件，提供线程安全的服务注册和管理功能：

```go
// EnhancedServiceRegistry 管理服务处理器的线程安全操作
type EnhancedServiceRegistry struct {
    mu       sync.RWMutex
    handlers map[string]ServiceHandler
    order    []string // 维护注册顺序
}

func NewEnhancedServiceRegistry() *EnhancedServiceRegistry {
    return &EnhancedServiceRegistry{
        handlers: make(map[string]ServiceHandler),
        order:    make([]string, 0),
    }
}
```

#### 核心功能

**1. 服务注册管理**

```go
// Register 添加服务处理器到注册表
func (sr *EnhancedServiceRegistry) Register(handler ServiceHandler) error {
    sr.mu.Lock()
    defer sr.mu.Unlock()
    
    metadata := handler.GetMetadata()
    if metadata == nil {
        return fmt.Errorf("service handler metadata cannot be nil")
    }
    
    if metadata.Name == "" {
        return fmt.Errorf("service name cannot be empty")
    }
    
    // 检查重复服务名
    if _, exists := sr.handlers[metadata.Name]; exists {
        return fmt.Errorf("service '%s' is already registered", metadata.Name)
    }
    
    sr.handlers[metadata.Name] = handler
    sr.order = append(sr.order, metadata.Name)
    
    return nil
}
```

**2. 批量路由注册**

```go
// RegisterAllHTTP 为所有服务注册 HTTP 路由
func (sr *EnhancedServiceRegistry) RegisterAllHTTP(router *gin.Engine) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    for _, name := range sr.order {
        if handler, exists := sr.handlers[name]; exists {
            if err := handler.RegisterHTTP(router); err != nil {
                return fmt.Errorf("failed to register HTTP routes for service '%s': %w", name, err)
            }
        }
    }
    
    return nil
}

// RegisterAllGRPC 为所有处理器注册 gRPC 服务
func (sr *EnhancedServiceRegistry) RegisterAllGRPC(server *grpc.Server) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    for _, name := range sr.order {
        if handler, exists := sr.handlers[name]; exists {
            if err := handler.RegisterGRPC(server); err != nil {
                return fmt.Errorf("failed to register gRPC services for service '%s': %w", name, err)
            }
        }
    }
    
    return nil
}
```

**3. 健康检查管理**

```go
// CheckAllHealth 对所有注册服务执行健康检查
func (sr *EnhancedServiceRegistry) CheckAllHealth(ctx context.Context) map[string]*types.HealthStatus {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    results := make(map[string]*types.HealthStatus)
    
    for _, name := range sr.order {
        if handler, exists := sr.handlers[name]; exists {
            // 为每个健康检查创建超时上下文
            healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
            status, err := handler.IsHealthy(healthCtx)
            cancel()
            
            if err != nil {
                // 如果健康检查失败，创建不健康状态
                status = &types.HealthStatus{
                    Status:       "unhealthy",
                    Timestamp:    time.Now(),
                    Version:      "unknown",
                    Uptime:       0,
                    Dependencies: make(map[string]types.DependencyStatus),
                }
            }
            
            results[name] = status
        }
    }
    
    return results
}
```

**4. 优雅关闭管理**

```go
// ShutdownAll 按逆序优雅关闭所有注册服务
func (sr *EnhancedServiceRegistry) ShutdownAll(ctx context.Context) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    // 按逆序关闭
    for i := len(sr.order) - 1; i >= 0; i-- {
        name := sr.order[i]
        if handler, exists := sr.handlers[name]; exists {
            if err := handler.Shutdown(ctx); err != nil {
                return fmt.Errorf("failed to shutdown service '%s': %w", name, err)
            }
        }
    }
    
    return nil
}
```

### 4. 传输层集成

传输层通过 `HTTPTransport` 和 `GRPCTransport` 与服务注册表集成：

```go
// HTTPTransport 实现 HTTP 传输层
type HTTPTransport struct {
    server          *http.Server
    router          *gin.Engine
    address         string
    ready           chan struct{}
    readyOnce       sync.Once
    mu              sync.RWMutex
    serviceRegistry *EnhancedServiceRegistry
}

// RegisterService 注册服务处理器到传输层
func (h *HTTPTransport) RegisterService(handler ServiceHandler) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    // 将服务注册到注册表
    if err := h.serviceRegistry.Register(handler); err != nil {
        return fmt.Errorf("failed to register service: %w", err)
    }
    
    return nil
}

// RegisterAllRoutes 为所有注册服务注册 HTTP 路由
func (h *HTTPTransport) RegisterAllRoutes() error {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    return h.serviceRegistry.RegisterAllHTTP(h.router)
}
```

## 服务实现示例

### 用户服务处理器实现

```go
// internal/switserve/handler/http/user/v1/user.go
type Handler struct {
    userSrv   interfaces.UserService
    startTime time.Time
}

func NewUserController(userSrv interfaces.UserService) *Handler {
    return &Handler{
        userSrv:   userSrv,
        startTime: time.Now(),
    }
}

// GetMetadata 返回服务元数据
func (h *Handler) GetMetadata() *transport.ServiceMetadata {
    return &transport.ServiceMetadata{
        Name:           "user-service",
        Version:        "v1.0.0",
        Description:    "User management service",
        HealthEndpoint: "/api/v1/users/health",
        Tags:           []string{"user", "management"},
        Dependencies:   []string{"database"},
    }
}

// RegisterHTTP 注册 HTTP 路由
func (h *Handler) RegisterHTTP(router *gin.Engine) error {
    v1 := router.Group("/api/v1")
    
    users := v1.Group("/users")
    {
        users.GET("", h.GetUsers)
        users.POST("", h.CreateUser)
        users.GET("/:username", h.GetUserByUsername)
        users.GET("/email/:email", h.GetUserByEmail)
        users.DELETE("/:username", h.DeleteUser)
        users.POST("/validate", h.ValidateUserCredentials)
    }
    
    // 健康检查端点
    v1.GET("/users/health", h.HealthCheck)
    
    return nil
}

// RegisterGRPC 注册 gRPC 服务
func (h *Handler) RegisterGRPC(server *grpc.Server) error {
    // 如果需要 gRPC 支持，在这里注册
    return nil
}

// IsHealthy 执行健康检查
func (h *Handler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    uptime := time.Since(h.startTime)
    
    status := &types.HealthStatus{
        Status:       "healthy",
        Timestamp:    time.Now(),
        Version:      "v1.0.0",
        Uptime:       uptime,
        Dependencies: make(map[string]types.DependencyStatus),
    }
    
    // 检查数据库连接
    if h.userSrv != nil {
        // 可以添加具体的健康检查逻辑
        status.AddDependency("database", types.NewDependencyStatus("healthy", 0))
    }
    
    return status, nil
}

// Initialize 初始化服务
func (h *Handler) Initialize(ctx context.Context) error {
    // 执行初始化逻辑
    return nil
}

// Shutdown 优雅关闭服务
func (h *Handler) Shutdown(ctx context.Context) error {
    // 执行清理逻辑
    return nil
}

// GetHealthEndpoint 返回健康检查端点
func (h *Handler) GetHealthEndpoint() string {
    return "/api/v1/users/health"
}
```

### 认证服务处理器实现

```go
// internal/switauth/handler/http/auth/v1/auth.go
type Controller struct {
    authService interfaces.AuthService
    startTime   time.Time
}

func NewAuthController(authService interfaces.AuthService) *Controller {
    return &Controller{
        authService: authService,
        startTime:   time.Now(),
    }
}

// GetMetadata 返回服务元数据
func (c *Controller) GetMetadata() *transport.ServiceMetadata {
    return &transport.ServiceMetadata{
        Name:           "auth-service",
        Version:        "v1.0.0",
        Description:    "Authentication and authorization service",
        HealthEndpoint: "/api/v1/auth/health",
        Tags:           []string{"auth", "security"},
        Dependencies:   []string{"database", "user-service"},
    }
}

// RegisterHTTP 注册 HTTP 路由
func (c *Controller) RegisterHTTP(router *gin.Engine) error {
    v1 := router.Group("/api/v1")
    
    auth := v1.Group("/auth")
    {
        auth.POST("/login", c.Login)
        auth.POST("/logout", c.Logout)
        auth.GET("/verify", c.Verify)
        auth.POST("/refresh", c.RefreshToken)
    }
    
    // 健康检查端点
    v1.GET("/auth/health", c.HealthCheck)
    
    return nil
}

// IsHealthy 执行健康检查
func (c *Controller) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    uptime := time.Since(c.startTime)
    
    status := &types.HealthStatus{
        Status:       "healthy",
        Timestamp:    time.Now(),
        Version:      "v1.0.0",
        Uptime:       uptime,
        Dependencies: make(map[string]types.DependencyStatus),
    }
    
    // 检查认证服务依赖
    if c.authService != nil {
        status.AddDependency("auth-service", types.NewDependencyStatus("healthy", 0))
        status.AddDependency("database", types.NewDependencyStatus("healthy", 0))
        status.AddDependency("user-service", types.NewDependencyStatus("healthy", 0))
    }
    
    return status, nil
}
```

## 服务器启动流程

### switserve 服务器启动

```go
// internal/switserve/server.go
func NewServer() (*Server, error) {
    server := &Server{
        transportManager: transport.NewManager(),
    }
    
    // 初始化依赖，传入关闭回调
    dependencies, err := deps.NewDependencies(func() {
        if err := server.Shutdown(); err != nil {
            logger.Logger.Error("Failed to shutdown server during dependency cleanup", zap.Error(err))
        }
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize dependencies: %v", err)
    }
    
    server.deps = dependencies
    
    // 初始化传输层
    server.httpTransport = transport.NewHTTPTransport()
    server.grpcTransport = transport.NewGRPCTransport()
    
    // 注册传输层
    server.transportManager.Register(server.httpTransport)
    server.transportManager.Register(server.grpcTransport)
    
    // 注册服务
    server.registerServices()
    
    return server, nil
}

// registerServices 注册所有服务到服务注册表
func (s *Server) registerServices() {
    // 注册 Greeter 服务
    greeterHandler := greeterv1.NewHandler(s.deps.GreeterSrv)
    s.httpTransport.RegisterService(greeterHandler)
    
    // 注册 Notification 服务
    notificationHandler := notificationv1.NewHandler(s.deps.NotificationSrv)
    s.httpTransport.RegisterService(notificationHandler)
    
    // 注册 Health 服务
    healthHandler := health.NewHandler(s.deps.HealthSrv)
    s.httpTransport.RegisterService(healthHandler)
    
    // 注册 Stop 服务
    stopHandler := stop.NewHandler(s.deps.StopSrv)
    s.httpTransport.RegisterService(stopHandler)
    
    // 注册 User 服务
    userHandler := userv1.NewUserController(s.deps.UserSrv)
    s.httpTransport.RegisterService(userHandler)
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
    // 获取 gRPC 服务器
    grpcServer := s.grpcTransport.GetServer()
    
    // 获取 HTTP 路由器
    httpRouter := s.httpTransport.GetRouter()
    
    // 通过 HTTP 传输层注册所有 HTTP 路由
    if err := s.httpTransport.RegisterAllRoutes(); err != nil {
        return fmt.Errorf("failed to register HTTP routes: %v", err)
    }
    
    // 通过 HTTP 传输层的服务注册表注册所有 gRPC 服务
    serviceRegistry := s.httpTransport.GetServiceRegistry()
    if err := serviceRegistry.RegisterAllGRPC(grpcServer); err != nil {
        return fmt.Errorf("failed to register gRPC services: %v", err)
    }
    
    // 启动所有传输层
    if err := s.transportManager.Start(ctx); err != nil {
        return fmt.Errorf("failed to start transports: %v", err)
    }
    
    // 等待 HTTP 传输层就绪
    <-s.httpTransport.WaitReady()
    
    // 在服务发现中注册服务
    cfg := config.GetConfig()
    port, _ := strconv.Atoi(cfg.Server.Port)
    if err := s.sd.RegisterService("swit-serve", "localhost", port); err != nil {
        logger.Logger.Error("failed to register swit-serve service", zap.Error(err))
        return err
    }
    
    logger.Logger.Info("Server started successfully",
        zap.String("http_address", s.httpTransport.Address()),
        zap.String("grpc_address", s.grpcTransport.Address()),
    )
    
    return nil
}
```

### switauth 服务器启动

```go
// internal/switauth/server.go
func NewServer() (*Server, error) {
    // 初始化依赖
    dependencies, err := deps.NewDependencies()
    if err != nil {
        return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
    }
    
    server := &Server{
        transportManager: transport.NewManager(),
        deps:             dependencies,
    }
    
    // 初始化传输层
    server.httpTransport = transport.NewHTTPTransport()
    server.grpcTransport = transport.NewGRPCTransport()
    
    // 注册传输层
    server.transportManager.Register(server.httpTransport)
    server.transportManager.Register(server.grpcTransport)
    
    // 注册服务
    server.registerServices()
    
    return server, nil
}

// registerServices 注册所有服务
func (s *Server) registerServices() {
    // 注册认证服务
    authHandler := auth.NewAuthController(s.deps.AuthSrv)
    s.httpTransport.RegisterService(authHandler)
    
    // 注册健康检查服务
    healthHandler := health.NewHandler(s.deps.HealthSrv)
    s.httpTransport.RegisterService(healthHandler)
}
```

## 架构优势

### 1. 统一性

- **接口标准化**: 所有服务实现相同的 `ServiceHandler` 接口
- **生命周期管理**: 统一的初始化、健康检查和关闭流程
- **元数据驱动**: 通过元数据实现服务发现和监控

### 2. 可扩展性

- **插件化架构**: 新服务只需实现 `ServiceHandler` 接口即可集成
- **传输层无关**: 同时支持 HTTP 和 gRPC 协议
- **依赖注入**: 通过接口实现松耦合设计

### 3. 可维护性

- **分层架构**: 清晰的依赖管理和服务分层
- **错误处理**: 完整的错误处理和验证机制
- **资源管理**: 自动化的资源清理和优雅关闭

### 4. 可测试性

- **接口驱动**: 所有依赖都通过接口注入，便于模拟测试
- **依赖隔离**: 每个服务的依赖都是独立的，便于单元测试
- **生命周期控制**: 可以独立测试服务的各个生命周期阶段

### 5. 可观测性

- **健康检查**: 内置的健康检查机制
- **服务元数据**: 丰富的服务信息用于监控和调试
- **依赖追踪**: 清晰的服务依赖关系

## 最佳实践

### 1. 服务实现模式

```go
// 推荐的服务处理器实现模式
type ServiceHandler struct {
    service     interfaces.Service  // 业务服务接口
    httpHandler *http.Handler      // HTTP 处理器
    grpcHandler *grpc.Handler      // gRPC 处理器
    startTime   time.Time          // 服务启动时间
    logger      *slog.Logger       // 日志记录器
}

func NewServiceHandler(service interfaces.Service, logger *slog.Logger) *ServiceHandler {
    return &ServiceHandler{
        service:   service,
        startTime: time.Now(),
        logger:    logger,
    }
}
```

### 2. 依赖注入模式

```go
// 使用选项模式进行依赖注入
type ServiceOptions struct {
    Repository repository.Repository
    Client     client.Client
    Logger     *slog.Logger
}

type ServiceOption func(*ServiceOptions)

func WithRepository(repo repository.Repository) ServiceOption {
    return func(opts *ServiceOptions) {
        opts.Repository = repo
    }
}

func NewService(opts ...ServiceOption) (interfaces.Service, error) {
    options := &ServiceOptions{}
    for _, opt := range opts {
        opt(options)
    }
    
    // 验证必需的依赖
    if options.Repository == nil {
        return nil, fmt.Errorf("repository is required")
    }
    
    return &service{
        repo:   options.Repository,
        client: options.Client,
        logger: options.Logger,
    }, nil
}
```

### 3. 健康检查实现

```go
func (h *ServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    uptime := time.Since(h.startTime)
    
    status := &types.HealthStatus{
        Status:       "healthy",
        Timestamp:    time.Now(),
        Version:      "v1.0.0",
        Uptime:       uptime,
        Dependencies: make(map[string]types.DependencyStatus),
    }
    
    // 检查服务依赖
    if h.service != nil {
        // 检查数据库连接
        if dbStatus := h.checkDatabase(ctx); dbStatus != nil {
            status.AddDependency("database", *dbStatus)
        }
        
        // 检查外部服务
        if extStatus := h.checkExternalService(ctx); extStatus != nil {
            status.AddDependency("external-service", *extStatus)
        }
    }
    
    return status, nil
}

func (h *ServiceHandler) checkDatabase(ctx context.Context) *types.DependencyStatus {
    start := time.Now()
    
    // 执行数据库健康检查
    if err := h.service.CheckDatabaseHealth(ctx); err != nil {
        return &types.DependencyStatus{
            Status:    "unhealthy",
            Error:     err.Error(),
            Timestamp: time.Now(),
        }
    }
    
    latency := time.Since(start)
    return &types.DependencyStatus{
        Status:    "healthy",
        Latency:   latency,
        Timestamp: time.Now(),
    }
}
```

### 4. 错误处理模式

```go
// 定义服务特定的错误类型
var (
    ErrDatabaseConnection    = errors.New("database connection failed")
    ErrServiceInitialization = errors.New("service initialization failed")
    ErrDependencyValidation  = errors.New("dependency validation failed")
)

// 在依赖初始化中使用包装错误
func NewDependencies() (*Dependencies, error) {
    database := db.GetDB()
    if database == nil {
        return nil, fmt.Errorf("%w: unable to connect to database", ErrDatabaseConnection)
    }
    
    userSrv, err := userv1.NewUserSrv(userv1.WithUserRepository(userRepo))
    if err != nil {
        return nil, fmt.Errorf("%w: user service - %v", ErrServiceInitialization, err)
    }
    
    return deps, nil
}
```

## 迁移指南

### 为新服务添加 ServiceHandler 支持

1. **实现 ServiceHandler 接口**:
   ```go
   func NewServiceHandler(service interfaces.Service) *ServiceHandler {
       return &ServiceHandler{
           service:   service,
           startTime: time.Now(),
       }
   }
   ```

2. **更新依赖结构**:
   ```go
   type Dependencies struct {
       // ... 现有字段
       NewSrv interfaces.NewService
   }
   ```

3. **更新服务注册**:
   ```go
   func (s *Server) registerServices() {
       // ... 现有注册
       newHandler := newservice.NewServiceHandler(s.deps.NewSrv)
       s.httpTransport.RegisterService(newHandler)
   }
   ```

### 从旧架构迁移

1. **替换 ServiceRegistrar**: 将现有的 `ServiceRegistrar` 接口实现迁移到 `ServiceHandler`
2. **添加生命周期方法**: 实现 `Initialize`、`IsHealthy` 和 `Shutdown` 方法
3. **更新元数据**: 添加 `GetMetadata` 和 `GetHealthEndpoint` 方法
4. **测试验证**: 确保所有功能正常工作

## 未来考虑

### 1. 扩展到复杂依赖

对于具有复杂依赖图的项目，考虑：

- **Google Wire**: 编译时依赖注入，适用于大型项目
- **工厂模式**: 按领域分组相关依赖
- **构建器模式**: 复杂对象构造和验证
- **插件架构**: 动态服务加载和注册

### 2. 性能优化

- **延迟初始化**: 按需初始化服务
- **连接池**: 优化数据库和外部服务连接
- **缓存策略**: 实现智能缓存机制
- **监控集成**: 添加 Prometheus 指标和分布式追踪

### 3. 云原生支持

- **Kubernetes 集成**: 支持 Kubernetes 健康检查和生命周期
- **服务网格**: 与 Istio/Linkerd 集成
- **配置管理**: 支持动态配置更新
- **可观测性**: 集成 OpenTelemetry 和日志聚合

## 总结

当前的依赖注入和服务注册架构通过统一的 `ServiceHandler` 接口和 `EnhancedServiceRegistry` 实现了：

- **统一的服务管理**: 所有服务都遵循相同的生命周期和管理模式
- **强大的扩展性**: 新服务可以轻松集成到现有架构中
- **完整的可观测性**: 内置健康检查、监控和日志记录
- **优雅的资源管理**: 自动化的初始化和清理流程

这个架构为 SWIT 项目提供了坚实的基础，支持未来的扩展和演进需求。