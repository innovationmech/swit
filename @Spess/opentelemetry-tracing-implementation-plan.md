# OpenTelemetry 分布式追踪实现详细方案

## 项目概述

为 Swit 微服务框架添加 OpenTelemetry 分布式追踪支持，实现跨服务的请求链路追踪，提升系统可观察性。这个方案将在现有的 switserve 和 switauth 服务中进行演示，为框架用户提供开箱即用的分布式追踪能力。

## 整体架构

```text
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │    │   gRPC Client   │    │  Database ORM   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ HTTP Middleware │    │ gRPC Middleware │    │   DB Hooks      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                                 ▼
                    ┌─────────────────────┐
                    │  Tracing Manager    │
                    │  - Context Prop     │
                    │  - Span Management  │
                    │  - Exporters        │
                    └─────────────────────┘
                                 │
                                 ▼
         ┌─────────────────────────────────────────────┐
         │              Exporters                      │
         ├─────────────────┬─────────────────┬─────────┤
         │     Jaeger      │      OTLP       │ Console │
         └─────────────────┴─────────────────┴─────────┘
```

## 实施计划

### 第一阶段：核心追踪包开发 (Phase 1)

**目标：** 创建独立的追踪管理包，提供 OpenTelemetry 的 Go 接口封装

#### 1.1 追踪管理器 (`pkg/tracing/tracer.go`)

**功能特性：**
- TracerProvider 的初始化和生命周期管理
- 全局 Tracer 实例管理
- Span 的创建、配置和结束
- 上下文传播工具函数
- 追踪属性和事件的标准化接口

**核心接口设计：**
```go
type TracingManager interface {
    // 初始化追踪器
    Initialize(ctx context.Context, config *TracingConfig) error
    
    // 创建追踪 span
    StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, Span)
    
    // 获取当前 span
    SpanFromContext(ctx context.Context) Span
    
    // 注入和提取追踪上下文
    InjectHTTPHeaders(ctx context.Context, headers http.Header)
    ExtractHTTPHeaders(headers http.Header) context.Context
    
    // 资源清理
    Shutdown(ctx context.Context) error
}

type Span interface {
    // 设置属性
    SetAttribute(key string, value interface{})
    SetAttributes(attrs ...attribute.KeyValue)
    
    // 记录事件
    AddEvent(name string, opts ...trace.EventOption)
    
    // 设置状态
    SetStatus(code codes.Code, description string)
    
    // 结束 span
    End(opts ...trace.SpanEndOption)
}
```

#### 1.2 追踪配置 (`pkg/tracing/config.go`)

**配置结构：**
```go
type TracingConfig struct {
    // 基础配置
    Enabled     bool   `yaml:"enabled" mapstructure:"enabled"`
    ServiceName string `yaml:"service_name" mapstructure:"service_name"`
    
    // 采样配置
    Sampling SamplingConfig `yaml:"sampling" mapstructure:"sampling"`
    
    // 导出器配置
    Exporter ExporterConfig `yaml:"exporter" mapstructure:"exporter"`
    
    // 资源属性
    ResourceAttributes map[string]string `yaml:"resource_attributes" mapstructure:"resource_attributes"`
    
    // 传播器配置
    Propagators []string `yaml:"propagators" mapstructure:"propagators"`
}

type SamplingConfig struct {
    Type string  `yaml:"type" mapstructure:"type"` // always_on, always_off, traceidratio
    Rate float64 `yaml:"rate" mapstructure:"rate"` // 0.0-1.0
}

type ExporterConfig struct {
    Type     string            `yaml:"type" mapstructure:"type"` // jaeger, otlp, console
    Endpoint string            `yaml:"endpoint" mapstructure:"endpoint"`
    Headers  map[string]string `yaml:"headers" mapstructure:"headers"`
    Timeout  string            `yaml:"timeout" mapstructure:"timeout"`
    
    // Jaeger 特定配置
    Jaeger JaegerConfig `yaml:"jaeger" mapstructure:"jaeger"`
    
    // OTLP 特定配置
    OTLP OTLPConfig `yaml:"otlp" mapstructure:"otlp"`
}
```

**默认配置：**
```go
func DefaultTracingConfig() *TracingConfig {
    return &TracingConfig{
        Enabled:     false,
        ServiceName: "swit-service",
        Sampling: SamplingConfig{
            Type: "traceidratio",
            Rate: 0.1,
        },
        Exporter: ExporterConfig{
            Type:     "console",
            Endpoint: "",
            Timeout:  "10s",
        },
        Propagators: []string{"tracecontext", "baggage"},
    }
}
```

#### 1.3 导出器实现 (`pkg/tracing/exporter.go`)

**支持的导出器：**

1. **Jaeger 导出器**
   - HTTP 和 gRPC 传输支持
   - 批量导出配置
   - 认证支持

2. **OTLP 导出器**
   - HTTP 和 gRPC 协议支持
   - 标准 OpenTelemetry 协议
   - 云原生兼容性

3. **Console 导出器**
   - 开发调试用
   - 标准输出格式化
   - 结构化日志集成

**导出器工厂：**
```go
type ExporterFactory interface {
    CreateSpanExporter(config ExporterConfig) (trace.SpanExporter, error)
}

func NewExporterFactory() ExporterFactory {
    return &exporterFactory{}
}
```

### 第二阶段：中间件集成 (Phase 2)

**目标：** 在现有中间件系统中集成追踪功能

#### 2.1 HTTP 中间件 (`pkg/middleware/tracing_http.go`)

**功能实现：**
- 自动创建根 span 或延续追踪链
- HTTP 请求/响应元数据记录
- 错误和异常状态追踪
- 自定义属性注入支持

**中间件实现：**
```go
func TracingMiddleware(tm tracing.TracingManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 HTTP headers 提取追踪上下文
        parentCtx := tm.ExtractHTTPHeaders(c.Request.Header)
        
        // 创建 span
        ctx, span := tm.StartSpan(parentCtx, 
            fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
            tracing.WithSpanKind(trace.SpanKindServer),
            tracing.WithAttributes(
                attribute.String("http.method", c.Request.Method),
                attribute.String("http.url", c.Request.URL.String()),
                attribute.String("http.route", c.FullPath()),
                attribute.String("user_agent", c.Request.UserAgent()),
            ),
        )
        defer span.End()
        
        // 设置上下文
        c.Request = c.Request.WithContext(ctx)
        
        // 继续请求处理
        c.Next()
        
        // 记录响应信息
        span.SetAttribute("http.status_code", c.Writer.Status())
        
        // 处理错误状态
        if c.Writer.Status() >= 400 {
            span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
        }
        
        // 记录响应大小
        span.SetAttribute("http.response_size", c.Writer.Size())
    }
}
```

#### 2.2 gRPC 中间件 (`pkg/middleware/tracing_grpc.go`)

**Unary 拦截器：**
```go
func UnaryServerInterceptor(tm tracing.TracingManager) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 从 gRPC metadata 提取追踪上下文
        md, ok := metadata.FromIncomingContext(ctx)
        if ok {
            ctx = tm.ExtractGRPCMetadata(ctx, md)
        }
        
        // 创建 span
        ctx, span := tm.StartSpan(ctx, info.FullMethod,
            tracing.WithSpanKind(trace.SpanKindServer),
            tracing.WithAttributes(
                attribute.String("rpc.system", "grpc"),
                attribute.String("rpc.method", info.FullMethod),
                attribute.String("rpc.service", extractServiceName(info.FullMethod)),
            ),
        )
        defer span.End()
        
        // 执行 RPC 方法
        resp, err := handler(ctx, req)
        
        // 记录结果
        if err != nil {
            span.SetStatus(codes.Error, err.Error())
            span.SetAttribute("rpc.grpc.status_code", status.Code(err).String())
        } else {
            span.SetStatus(codes.Ok, "")
        }
        
        return resp, err
    }
}
```

**Stream 拦截器：**
```go
func StreamServerInterceptor(tm tracing.TracingManager) grpc.StreamServerInterceptor {
    return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
        // 类似 Unary 的实现，但适配 Stream 接口
        // 需要包装 ServerStream 以传递追踪上下文
        wrappedStream := &tracingServerStream{
            ServerStream: ss,
            ctx:         ctx,
        }
        
        return handler(srv, wrappedStream)
    }
}
```

#### 2.3 数据库追踪 (`pkg/tracing/db.go`)

**GORM 钩子集成：**
```go
type GormTracing struct {
    tm tracing.TracingManager
}

func (gt *GormTracing) Name() string {
    return "gorm:tracing"
}

func (gt *GormTracing) Initialize(db *gorm.DB) error {
    // 注册各类钩子
    db.Callback().Create().Before("gorm:create").Register("tracing:before_create", gt.beforeCreate)
    db.Callback().Create().After("gorm:create").Register("tracing:after_create", gt.afterCreate)
    
    db.Callback().Query().Before("gorm:query").Register("tracing:before_query", gt.beforeQuery)
    db.Callback().Query().After("gorm:query").Register("tracing:after_query", gt.afterQuery)
    
    // 类似地注册 Update, Delete 钩子
    return nil
}

func (gt *GormTracing) beforeCreate(db *gorm.DB) {
    ctx, span := gt.tm.StartSpan(db.Statement.Context, "gorm.create",
        tracing.WithSpanKind(trace.SpanKindClient),
        tracing.WithAttributes(
            attribute.String("db.system", "mysql"),
            attribute.String("db.operation", "create"),
            attribute.String("db.table", db.Statement.Table),
        ),
    )
    
    db.Statement.Context = ctx
    db.Set("tracing:span", span)
}

func (gt *GormTracing) afterCreate(db *gorm.DB) {
    span, exists := db.Get("tracing:span")
    if !exists {
        return
    }
    
    tracingSpan := span.(tracing.Span)
    defer tracingSpan.End()
    
    if db.Error != nil {
        tracingSpan.SetStatus(codes.Error, db.Error.Error())
    }
    
    tracingSpan.SetAttribute("db.rows_affected", db.RowsAffected)
}
```

### 第三阶段：框架集成 (Phase 3)

**目标：** 将追踪能力深度集成到 Swit 框架核心

#### 3.1 服务器配置扩展 (`pkg/server/config.go`)

**配置结构扩展：**
```go
type ServerConfig struct {
    // ... 现有配置字段
    
    // 新增追踪配置
    Tracing TracingConfig `yaml:"tracing" mapstructure:"tracing"`
}

// 配置验证扩展
func (c *ServerConfig) Validate() error {
    // ... 现有验证逻辑
    
    // 追踪配置验证
    if c.Tracing.Enabled {
        if err := c.Tracing.Validate(); err != nil {
            return fmt.Errorf("tracing configuration invalid: %w", err)
        }
    }
    
    return nil
}
```

#### 3.2 可观察性管理器扩展 (`pkg/server/observability.go`)

**ObservabilityManager 扩展：**
```go
type ObservabilityManager struct {
    // ... 现有字段
    
    // 新增追踪管理器
    tracingManager tracing.TracingManager
}

func NewObservabilityManager(serviceName string, prometheusConfig *PrometheusConfig, tracingConfig *tracing.TracingConfig, collector MetricsCollector) *ObservabilityManager {
    // ... 现有逻辑
    
    // 初始化追踪管理器
    var tracingManager tracing.TracingManager
    if tracingConfig != nil && tracingConfig.Enabled {
        tracingManager = tracing.NewTracingManager()
    }
    
    om := &ObservabilityManager{
        // ... 现有字段设置
        tracingManager: tracingManager,
    }
    
    return om
}

// 追踪生命周期管理
func (om *ObservabilityManager) InitializeTracing(ctx context.Context, config *tracing.TracingConfig) error {
    if om.tracingManager == nil {
        return nil
    }
    
    return om.tracingManager.Initialize(ctx, config)
}

func (om *ObservabilityManager) ShutdownTracing(ctx context.Context) error {
    if om.tracingManager == nil {
        return nil
    }
    
    return om.tracingManager.Shutdown(ctx)
}

func (om *ObservabilityManager) GetTracingManager() tracing.TracingManager {
    return om.tracingManager
}
```

#### 3.3 传输层集成 (`pkg/transport/`)

**HTTP 传输集成 (`pkg/transport/http.go`)：**
```go
type HTTPNetworkService struct {
    // ... 现有字段
    
    tracingManager tracing.TracingManager
}

func (h *HTTPNetworkService) setupMiddleware(router *gin.Engine) error {
    // ... 现有中间件设置
    
    // 添加追踪中间件
    if h.tracingManager != nil {
        router.Use(middleware.TracingMiddleware(h.tracingManager))
    }
    
    return nil
}
```

**gRPC 传输集成 (`pkg/transport/grpc.go`)：**
```go
type GRPCNetworkService struct {
    // ... 现有字段
    
    tracingManager tracing.TracingManager
}

func (g *GRPCNetworkService) setupInterceptors() []grpc.ServerOption {
    var opts []grpc.ServerOption
    
    // ... 现有拦截器设置
    
    // 添加追踪拦截器
    if g.tracingManager != nil {
        opts = append(opts,
            grpc.UnaryInterceptor(middleware.UnaryServerInterceptor(g.tracingManager)),
            grpc.StreamInterceptor(middleware.StreamServerInterceptor(g.tracingManager)),
        )
    }
    
    return opts
}
```

### 第四阶段：服务实现演示 (Phase 4)

**目标：** 在 switserve 和 switauth 服务中实际应用追踪功能

#### 4.1 switserve 服务集成

**配置文件 (`swit.yaml`)：**
```yaml
# ... 现有配置

tracing:
  enabled: true
  service_name: "swit-serve"
  sampling:
    type: "traceidratio"
    rate: 1.0  # 开发环境全量采样
  exporter:
    type: "jaeger"
    endpoint: "http://localhost:14268/api/traces"
  resource_attributes:
    environment: "development"
    version: "v1.0.0"
    service.instance.id: "${HOSTNAME:-localhost}"
```

**用户服务追踪 (`internal/switserve/service/user/v1/user.go`)：**
```go
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
    // 创建业务逻辑 span
    ctx, span := s.tracingManager.StartSpan(ctx, "UserService.CreateUser",
        tracing.WithAttributes(
            attribute.String("user.email", req.Email),
            attribute.String("user.name", req.Name),
        ),
    )
    defer span.End()
    
    // 验证输入 - 子 span
    ctx, validationSpan := s.tracingManager.StartSpan(ctx, "validate_user_input")
    if err := s.validateUserInput(req); err != nil {
        validationSpan.SetStatus(codes.Error, err.Error())
        validationSpan.End()
        span.SetStatus(codes.Error, "validation failed")
        return nil, err
    }
    validationSpan.End()
    
    // 数据库操作 - 将自动被 GORM 钩子追踪
    user := &model.User{
        Name:  req.Name,
        Email: req.Email,
    }
    
    if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
        span.SetStatus(codes.Error, err.Error())
        span.AddEvent("database_error", trace.WithAttributes(
            attribute.String("error", err.Error()),
        ))
        return nil, err
    }
    
    // 发送通知 - 跨服务调用追踪
    ctx, notificationSpan := s.tracingManager.StartSpan(ctx, "send_welcome_notification")
    if err := s.notificationService.SendWelcomeEmail(ctx, user.Email); err != nil {
        notificationSpan.SetStatus(codes.Error, err.Error())
        // 这里不中断主流程，只记录错误
        span.AddEvent("notification_failed", trace.WithAttributes(
            attribute.String("error", err.Error()),
        ))
    }
    notificationSpan.End()
    
    span.SetAttributes(
        attribute.Int64("user.id", int64(user.ID)),
        attribute.String("operation.status", "success"),
    )
    
    return &CreateUserResponse{User: user.ToProto()}, nil
}
```

#### 4.2 switauth 服务集成

**认证服务追踪 (`internal/switauth/service/auth/v1/auth.go`)：**
```go
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    ctx, span := s.tracingManager.StartSpan(ctx, "AuthService.Login",
        tracing.WithAttributes(
            attribute.String("auth.username", req.Username),
            attribute.String("auth.method", "password"),
        ),
    )
    defer span.End()
    
    // 用户验证 - 调用 switserve
    ctx, userVerifySpan := s.tracingManager.StartSpan(ctx, "verify_user_credentials")
    user, err := s.userClient.VerifyCredentials(ctx, &VerifyCredentialsRequest{
        Username: req.Username,
        Password: req.Password,
    })
    if err != nil {
        userVerifySpan.SetStatus(codes.Error, err.Error())
        userVerifySpan.End()
        span.SetStatus(codes.Error, "credential verification failed")
        return nil, err
    }
    userVerifySpan.SetAttribute("user.id", user.Id)
    userVerifySpan.End()
    
    // Token 生成
    ctx, tokenSpan := s.tracingManager.StartSpan(ctx, "generate_access_token")
    token, err := s.tokenService.GenerateAccessToken(ctx, user.Id)
    if err != nil {
        tokenSpan.SetStatus(codes.Error, err.Error())
        tokenSpan.End()
        span.SetStatus(codes.Error, "token generation failed")
        return nil, err
    }
    tokenSpan.SetAttribute("token.expires_at", token.ExpiresAt.Unix())
    tokenSpan.End()
    
    // Token 存储 - 数据库操作
    if err := s.tokenRepo.Save(ctx, token); err != nil {
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetAttributes(
        attribute.String("user.id", user.Id),
        attribute.String("operation.status", "success"),
    )
    
    return &LoginResponse{
        AccessToken: token.Token,
        ExpiresAt:   token.ExpiresAt,
        User:        user,
    }, nil
}
```

#### 4.3 跨服务通信追踪

**HTTP 客户端追踪 (`pkg/client/http.go`)：**
```go
type HTTPClient struct {
    client         *http.Client
    tracingManager tracing.TracingManager
}

func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    if c.tracingManager == nil {
        return c.client.Do(req)
    }
    
    // 创建客户端 span
    ctx, span := c.tracingManager.StartSpan(ctx, 
        fmt.Sprintf("HTTP %s", req.Method),
        tracing.WithSpanKind(trace.SpanKindClient),
        tracing.WithAttributes(
            attribute.String("http.method", req.Method),
            attribute.String("http.url", req.URL.String()),
        ),
    )
    defer span.End()
    
    // 注入追踪上下文到 HTTP headers
    c.tracingManager.InjectHTTPHeaders(ctx, req.Header)
    
    // 执行请求
    req = req.WithContext(ctx)
    resp, err := c.client.Do(req)
    
    if err != nil {
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetAttribute("http.status_code", resp.StatusCode)
    if resp.StatusCode >= 400 {
        span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
    }
    
    return resp, nil
}
```

**gRPC 客户端追踪：**
```go
func UnaryClientInterceptor(tm tracing.TracingManager) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        ctx, span := tm.StartSpan(ctx, method,
            tracing.WithSpanKind(trace.SpanKindClient),
            tracing.WithAttributes(
                attribute.String("rpc.system", "grpc"),
                attribute.String("rpc.method", method),
            ),
        )
        defer span.End()
        
        // 注入追踪上下文到 gRPC metadata
        ctx = tm.InjectGRPCMetadata(ctx)
        
        err := invoker(ctx, method, req, reply, cc, opts...)
        
        if err != nil {
            span.SetStatus(codes.Error, err.Error())
            span.SetAttribute("rpc.grpc.status_code", status.Code(err).String())
        }
        
        return err
    }
}
```

### 第五阶段：示例和文档 (Phase 5)

**目标：** 提供完整的使用示例、部署配置和详细文档

#### 5.1 分布式追踪示例 (`examples/distributed-tracing/`)

**目录结构：**
```text
examples/distributed-tracing/
├── README.md
├── docker-compose.yml           # Jaeger + 示例服务
├── jaeger/
│   └── jaeger-config.yml
├── services/
│   ├── order-service/           # 订单服务示例
│   ├── payment-service/         # 支付服务示例
│   └── inventory-service/       # 库存服务示例
├── scripts/
│   ├── setup.sh                 # 环境搭建脚本
│   ├── load-test.sh             # 压力测试脚本
│   └── trace-analysis.sh        # 追踪分析脚本
└── docs/
    ├── setup-guide.md           # 搭建指南
    ├── analysis-guide.md        # 分析指南
    └── troubleshooting.md       # 故障排查
```

**Docker Compose 配置：**
```yaml
version: '3.8'

services:
  jaeger:
    image: jaegertracing/all-in-one:1.42
    ports:
      - "5775:5775/udp"
      - "6831:6831/udp"
      - "6832:6832/udp"
      - "5778:5778"
      - "16686:16686"
      - "14268:14268"
      - "14250:14250"
      - "9411:9411"
    environment:
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
    networks:
      - tracing-network

  order-service:
    build: ./services/order-service
    ports:
      - "8081:8080"
      - "9081:9080"
    environment:
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
      - PAYMENT_SERVICE_URL=http://payment-service:8080
      - INVENTORY_SERVICE_URL=http://inventory-service:8080
    depends_on:
      - jaeger
      - payment-service
      - inventory-service
    networks:
      - tracing-network

  payment-service:
    build: ./services/payment-service
    ports:
      - "8082:8080"
      - "9082:9080"
    environment:
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    depends_on:
      - jaeger
    networks:
      - tracing-network

  inventory-service:
    build: ./services/inventory-service
    ports:
      - "8083:8080"
      - "9083:9080"
    environment:
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    depends_on:
      - jaeger
    networks:
      - tracing-network

networks:
  tracing-network:
    driver: bridge
```

#### 5.2 订单服务示例 (`examples/distributed-tracing/services/order-service/`)

**完整的端到端追踪演示：**
```go
type OrderService struct {
    paymentClient   PaymentServiceClient
    inventoryClient InventoryServiceClient
    tracingManager  tracing.TracingManager
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
    ctx, span := s.tracingManager.StartSpan(ctx, "OrderService.CreateOrder",
        tracing.WithAttributes(
            attribute.String("order.customer_id", req.CustomerId),
            attribute.String("order.product_id", req.ProductId),
            attribute.Int("order.quantity", int(req.Quantity)),
        ),
    )
    defer span.End()
    
    // 1. 检查库存
    ctx, inventorySpan := s.tracingManager.StartSpan(ctx, "check_inventory")
    inventoryResp, err := s.inventoryClient.CheckInventory(ctx, &CheckInventoryRequest{
        ProductId: req.ProductId,
        Quantity:  req.Quantity,
    })
    if err != nil {
        inventorySpan.SetStatus(codes.Error, err.Error())
        inventorySpan.End()
        span.SetStatus(codes.Error, "inventory check failed")
        return nil, err
    }
    inventorySpan.SetAttribute("inventory.available", inventoryResp.Available)
    inventorySpan.End()
    
    if !inventoryResp.Available {
        span.SetStatus(codes.Error, "insufficient inventory")
        return nil, errors.New("insufficient inventory")
    }
    
    // 2. 处理支付
    ctx, paymentSpan := s.tracingManager.StartSpan(ctx, "process_payment")
    paymentResp, err := s.paymentClient.ProcessPayment(ctx, &ProcessPaymentRequest{
        CustomerId: req.CustomerId,
        Amount:     req.Amount,
        Currency:   "USD",
    })
    if err != nil {
        paymentSpan.SetStatus(codes.Error, err.Error())
        paymentSpan.End()
        span.SetStatus(codes.Error, "payment processing failed")
        return nil, err
    }
    paymentSpan.SetAttributes(
        attribute.String("payment.transaction_id", paymentResp.TransactionId),
        attribute.String("payment.status", paymentResp.Status),
    )
    paymentSpan.End()
    
    // 3. 创建订单记录
    ctx, orderSpan := s.tracingManager.StartSpan(ctx, "create_order_record")
    order := &Order{
        ID:            generateOrderID(),
        CustomerID:    req.CustomerId,
        ProductID:     req.ProductId,
        Quantity:      req.Quantity,
        Amount:        req.Amount,
        TransactionID: paymentResp.TransactionId,
        Status:        "confirmed",
        CreatedAt:     time.Now(),
    }
    
    if err := s.db.WithContext(ctx).Create(order).Error; err != nil {
        orderSpan.SetStatus(codes.Error, err.Error())
        orderSpan.End()
        span.SetStatus(codes.Error, "order creation failed")
        return nil, err
    }
    orderSpan.SetAttribute("order.id", order.ID)
    orderSpan.End()
    
    // 4. 更新库存
    ctx, updateInventorySpan := s.tracingManager.StartSpan(ctx, "update_inventory")
    _, err = s.inventoryClient.UpdateInventory(ctx, &UpdateInventoryRequest{
        ProductId: req.ProductId,
        Quantity:  -req.Quantity, // 减少库存
    })
    if err != nil {
        updateInventorySpan.SetStatus(codes.Error, err.Error())
        updateInventorySpan.End()
        // 这里应该触发补偿事务
        span.AddEvent("inventory_update_failed", trace.WithAttributes(
            attribute.String("error", err.Error()),
        ))
    } else {
        updateInventorySpan.End()
    }
    
    span.SetAttributes(
        attribute.String("order.id", order.ID),
        attribute.String("operation.status", "success"),
    )
    
    return &CreateOrderResponse{
        OrderId:       order.ID,
        Status:        order.Status,
        TransactionId: order.TransactionID,
    }, nil
}
```

#### 5.3 文档和指南

**追踪使用指南 (`docs/tracing-guide.md`)：**

1. **快速开始**
   - 配置说明
   - 基础使用示例
   - 常见问题解答

2. **高级配置**
   - 采样策略配置
   - 多种导出器配置
   - 性能优化建议

3. **最佳实践**
   - Span 命名规范
   - 属性设置指南
   - 错误处理模式

4. **故障排查**
   - 常见配置错误
   - 性能问题诊断
   - 追踪数据缺失排查

**Jaeger UI 使用指南：**
- 追踪查询和过滤
- 性能瓶颈识别
- 错误分析方法
- 服务依赖关系分析

## 技术规格

### 依赖管理

**新增 Go 模块依赖：**
```go
require (
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/trace v1.24.0
    go.opentelemetry.io/otel/sdk v1.24.0
    go.opentelemetry.io/otel/exporters/jaeger v1.17.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
    go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.24.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
    go.opentelemetry.io/contrib/instrumentation/gorm.io/gorm/otelgorm v0.49.0
)
```

### 性能考虑

1. **采样策略**
   - 生产环境建议采样率 0.1-0.01
   - 开发环境可使用 1.0 全量采样
   - 支持自适应采样

2. **内存使用**
   - 批量导出缓解内存压力
   - 配置合理的缓冲区大小
   - 定期监控内存使用情况

3. **网络开销**
   - 异步导出减少请求延迟
   - 压缩导出数据
   - 合理配置导出间隔

### 安全注意事项

1. **敏感数据保护**
   - 不在追踪中记录密码、token 等敏感信息
   - 提供数据脱敏接口
   - 支持属性白名单配置

2. **网络安全**
   - 支持 TLS 加密传输
   - 认证机制集成
   - 访问权限控制

## 测试策略

### 单元测试

1. **追踪管理器测试**
   - 初始化和配置测试
   - Span 创建和管理测试
   - 上下文传播测试

2. **中间件测试**
   - HTTP 中间件功能测试
   - gRPC 拦截器测试
   - 数据库钩子测试

3. **导出器测试**
   - 各种导出器功能测试
   - 配置解析测试
   - 错误处理测试

### 集成测试

1. **端到端追踪测试**
   - 跨服务追踪验证
   - 追踪上下文传播验证
   - 导出数据完整性验证

2. **性能测试**
   - 追踪开销测量
   - 内存使用测试
   - 并发性能测试

### 测试环境

**测试用 Docker Compose：**
```yaml
version: '3.8'
services:
  jaeger-test:
    image: jaegertracing/all-in-one:1.42
    ports:
      - "14268:14268"
      - "16686:16686"
    environment:
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
      - SPAN_STORAGE_TYPE=memory
    networks:
      - test-network

  test-service:
    build: 
      context: .
      dockerfile: test/Dockerfile
    environment:
      - JAEGER_ENDPOINT=http://jaeger-test:14268/api/traces
      - GO_ENV=test
    depends_on:
      - jaeger-test
    networks:
      - test-network

networks:
  test-network:
    driver: bridge
```

## 实施时间表

| 阶段 | 持续时间 | 主要输出 | 验收标准 |
|------|----------|----------|----------|
| Phase 1: 核心包开发 | 2 周 | 追踪管理器、配置、导出器 | 单元测试覆盖率 >90% |
| Phase 2: 中间件集成 | 1.5 周 | HTTP/gRPC 中间件、DB 钩子 | 中间件功能完整验证 |
| Phase 3: 框架集成 | 1 周 | 框架配置、生命周期管理 | 框架集成测试通过 |
| Phase 4: 服务演示 | 1.5 周 | switserve/switauth 集成 | 跨服务追踪演示 |
| Phase 5: 示例文档 | 1 周 | 示例项目、使用文档 | 完整文档和示例 |

**总计：7 周**

## 风险评估和缓解策略

### 主要风险

1. **性能影响风险**
   - **风险描述：** 追踪功能可能影响服务性能
   - **缓解策略：** 
     - 实施性能基准测试
     - 提供可配置的采样率
     - 异步导出机制
     - 性能监控和告警

2. **兼容性风险**
   - **风险描述：** 与现有中间件和组件不兼容
   - **缓解策略：**
     - 渐进式集成策略
     - 向后兼容性保证
     - 全面的集成测试
     - 可选启用机制

3. **复杂性风险**
   - **风险描述：** 增加系统复杂度，影响维护性
   - **缓解策略：**
     - 清晰的接口设计
     - 完善的文档和示例
     - 自动化测试覆盖
     - 逐步演进策略

### 回滚策略

1. **功能开关**
   - 配置级别的开关控制
   - 运行时动态启用/禁用
   - 默认禁用状态

2. **向前兼容**
   - 保持现有 API 不变
   - 增量添加新功能
   - 版本管理策略

## 成功指标

### 功能指标

1. **追踪覆盖率**
   - HTTP 请求追踪覆盖率 100%
   - gRPC 调用追踪覆盖率 100%
   - 数据库操作追踪覆盖率 >95%
   - 跨服务调用追踪覆盖率 100%

2. **功能完整性**
   - 支持主流导出器（Jaeger、OTLP、Console）
   - 支持标准传播协议（W3C Trace Context）
   - 支持自定义采样策略
   - 支持分布式追踪关联

### 性能指标

1. **性能开销**
   - 追踪启用后性能下降 <5%
   - 内存使用增长 <10%
   - 延迟增加 <1ms（p95）

2. **可靠性指标**
   - 追踪数据丢失率 <0.1%
   - 服务稳定性不受影响
   - 导出器故障自动恢复

### 用户体验指标

1. **易用性**
   - 零配置快速启用
   - 5分钟内完成基础配置
   - 直观的追踪数据展示

2. **文档质量**
   - 完整的 API 文档
   - 端到端使用示例
   - 故障排查指南

## 后续规划

### 短期增强（3个月内）

1. **高级追踪特性**
   - Baggage 传播支持
   - 自定义传播器
   - 追踪采样决策算法

2. **集成增强**
   - Redis 操作追踪
   - 消息队列追踪
   - 外部 API 调用追踪

### 中期规划（6个月内）

1. **可视化增强**
   - Grafana 面板集成
   - 自定义追踪分析仪表板
   - 告警规则模板

2. **性能优化**
   - 零拷贝导出器
   - 内存池优化
   - 批量处理优化

### 长期愿景（1年内）

1. **智能追踪**
   - AI 驱动的异常检测
   - 自动性能瓶颈识别
   - 智能采样策略

2. **生态系统集成**
   - Kubernetes Operator 支持
   - 服务网格集成
   - 云原生可观察性平台集成

---

*本实现方案为 Swit 框架添加 OpenTelemetry 分布式追踪支持提供了详细的技术路线图和实施指南。通过分阶段的开发和集成，将为框架用户提供企业级的分布式追踪能力。*
