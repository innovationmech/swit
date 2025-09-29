# 配置

本指南涵盖 Swit 框架中的综合配置选项，包括服务器设置、传输配置、中间件以及不同环境的最佳实践。

## 概述

Swit 使用分层配置系统，支持 YAML 文件、环境变量和程序化配置。`ServerConfig` 结构提供类型安全的配置，并具有验证和合理的默认值。

## 配置结构

### 主要配置

```yaml
service_name: "my-service"
shutdown_timeout: "30s"

http:
  # HTTP 传输配置
  
grpc:
  # gRPC 传输配置
  
discovery:
  # 服务发现配置
  
middleware:
  # 中间件配置
```

## HTTP 传输配置

### 基本 HTTP 设置

```yaml
http:
  enabled: true
  port: "8080"
  address: ":8080"
  enable_ready: true
  test_mode: false
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  headers:
    X-Service-Name: "my-service"
    X-Version: "1.0.0"
```

### HTTP 中间件配置

```yaml
http:
  middleware:
    enable_cors: true
    enable_auth: false
    enable_rate_limit: false
    enable_logging: true
    enable_timeout: true
    
    # CORS 配置
    cors:
      allow_origins:
        - "http://localhost:3000"
        - "https://myapp.com"
      allow_methods:
        - "GET"
        - "POST"
        - "PUT"
        - "DELETE"
        - "OPTIONS"
      allow_headers:
        - "Origin"
        - "Content-Type"
        - "Accept"
        - "Authorization"
      expose_headers:
        - "X-Total-Count"
      allow_credentials: true
      max_age: 86400
    
    # 速率限制配置
    rate_limit:
      requests_per_second: 100
      burst_size: 200
      window_size: "60s"
      key_func: "ip"  # "ip", "user", "custom"
    
    # 超时配置
    timeout:
      request_timeout: "30s"
      handler_timeout: "25s"
    
    # 自定义头部
    custom_headers:
      X-Request-ID: "auto"
      X-Service-Version: "1.0.0"
```

### CORS 安全考虑

**重要：** 永远不要在 `allow_credentials: true` 时使用通配符 (`*`) 来源。这违反了 CORS 规范并创建安全漏洞。

```yaml
# ✅ 安全的 CORS 配置
cors:
  allow_origins:
    - "https://app.example.com"
    - "https://admin.example.com"
  allow_credentials: true

# ❌ 不安全的配置 - 将导致验证错误
cors:
  allow_origins: ["*"]
  allow_credentials: true
```

## gRPC 传输配置

### 基本 gRPC 设置

```yaml
grpc:
  enabled: true
  port: "9080"
  address: ":9080"
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true
  test_mode: false
  max_recv_msg_size: 4194304  # 4MB
  max_send_msg_size: 4194304  # 4MB
```

### gRPC Keepalive 配置

```yaml
grpc:
  keepalive_params:
    max_connection_idle: "15s"
    max_connection_age: "30s"
    max_connection_age_grace: "5s"
    time: "5s"
    timeout: "1s"
  
  keepalive_policy:
    min_time: "5s"
    permit_without_stream: true
```

## 服务发现配置

### 基本发现设置

```yaml
discovery:
  enabled: true
  address: "127.0.0.1:8500"
  service_name: "my-service"
  tags:
    - "v1"
    - "production"
    - "api"
  health_check_required: false
  registration_timeout: "30s"
```

### 发现失败模式

```yaml
discovery:
  failure_mode: "graceful"  # "graceful", "fail_fast", "strict"
```

- **graceful**：即使发现注册失败，服务器也继续启动（默认）
- **fail_fast**：如果发现注册失败，服务器启动失败
- **strict**：需要发现健康检查，对任何发现问题都快速失败

## 监控配置（Sentry）

框架通过 Sentry 集成提供全面的错误监控和性能跟踪。

### 基本 Sentry 配置

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"          # 通过环境变量设置
  environment: "production"      # 部署环境
  release: "v1.2.3"             # 可选的发布版本
  sample_rate: 1.0              # 错误采样率 (0.0-1.0)
  traces_sample_rate: 0.1       # 性能采样率 (0.0-1.0)
```

### 完整 Sentry 配置

```yaml
sentry:
  # 基本设置
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  release: "v1.2.3"
  server_name: "my-server-01"
  
  # 采样配置
  sample_rate: 1.0              # 捕获所有错误
  traces_sample_rate: 0.1       # 捕获 10% 的性能跟踪
  profiles_sample_rate: 0.1     # 捕获 10% 的性能分析数据
  
  # 性能和跟踪
  attach_stacktrace: true       # 在错误中包含堆栈跟踪
  enable_tracing: true          # 启用性能监控
  enable_profiling: true        # 启用性能分析（需要跟踪）
  
  # 调试和开发
  debug: false                  # 启用调试日志
  
  # 框架集成
  integrate_http: true          # 启用 HTTP 中间件
  integrate_grpc: true          # 启用 gRPC 中间件
  capture_panics: true          # 捕获和恢复恐慌
  max_breadcrumbs: 30          # 最大面包屑轨迹长度
  
  # 上下文和数据
  max_request_body_size: 1024   # 捕获的最大请求体（字节）
  send_default_pii: false       # 不发送个人识别信息
  
  # 自定义标签（添加到所有事件）
  tags:
    service: "user-management"
    version: "1.2.3"
    datacenter: "us-west"
    team: "platform"
  
  # 错误过滤
  ignore_errors:
    - "connection timeout"
    - "user not found"
    - "context deadline exceeded"
  
  # HTTP 特定过滤
  http_ignore_paths:
    - "/health"
    - "/metrics"
    - "/favicon.ico"
    - "/robots.txt"
  
  # HTTP 状态码过滤
  http_ignore_status_codes:
    - 404    # 未找到错误
    - 400    # 错误请求
    - 401    # 未授权（预期的）
```

### 环境特定 Sentry 配置

#### 开发环境
```yaml
sentry:
  enabled: true
  debug: true                   # 用于故障排除的详细日志
  sample_rate: 1.0             # 捕获所有错误以进行测试
  traces_sample_rate: 1.0      # 捕获所有跟踪以进行开发
  environment: "development"
  integrate_http: true
  integrate_grpc: true
```

#### 预发环境
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN_STAGING}"
  environment: "staging"
  sample_rate: 1.0             # 在预发环境中捕获所有错误
  traces_sample_rate: 0.5      # 50% 性能采样
  enable_profiling: true       # 在预发环境中测试性能分析
  ignore_errors:
    - "test error"             # 过滤特定于测试的错误
```

#### 生产环境
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN_PRODUCTION}"
  environment: "production"
  release: "${APP_VERSION}"    # 从 CI/CD 流水线设置
  sample_rate: 1.0             # 捕获所有错误
  traces_sample_rate: 0.1      # 10% 性能采样
  profiles_sample_rate: 0.1    # 10% 性能分析
  enable_profiling: true
  
  # 生产特定过滤
  http_ignore_status_codes:
    - 404
    - 400
    - 401
    - 403
  
  ignore_errors:
    - "connection refused"
    - "timeout"
    - "rate limit exceeded"
```

### Sentry 配置最佳实践

#### 采样策略
- **开发环境**：高采样率 (1.0) 用于全面调试
- **预发环境**：中等采样率 (0.5) 用于现实测试
- **生产环境**：低采样率 (0.1) 用于管理数量和成本

#### 错误过滤
- 过滤预期错误（4xx HTTP 状态码）
- 忽略健康检查和指标端点
- 过滤噪音连接错误
- 使用特定错误模式而不是广泛过滤

#### 性能监控
- 在所有环境中启用跟踪
- 在生产环境中使用保守采样
- 监控性能分析开销
- 设置适当的请求体大小限制

#### 安全考虑
- 永远不要记录敏感数据（PII、凭据）
- 对 DSN 配置使用环境变量
- 过滤包含秘密的请求体和标头
- 在生产环境中谨慎使用调试模式

### Sentry 环境变量

```bash
# 必需
export SENTRY_DSN="https://public@sentry.io/project-id"

# 可选覆盖
export SENTRY_ENVIRONMENT="production"
export SENTRY_RELEASE="v1.2.3"
export SENTRY_SAMPLE_RATE="1.0"
export SENTRY_TRACES_SAMPLE_RATE="0.1"
export SENTRY_DEBUG="false"
export SENTRY_SERVER_NAME="my-server-01"
```

### Sentry 验证

框架在启动时验证 Sentry 配置：

- **DSN 格式**：验证 DSN URL 格式
- **采样率**：确保值在 0.0 和 1.0 之间
- **环境**：验证环境字符串格式
- **依赖项**：检查所需的 Sentry SDK 版本

```go
// 配置验证示例
config := &ServerConfig{
    Sentry: SentryConfig{
        Enabled:           true,
        DSN:              "invalid-dsn",  // 这将导致验证错误
        SampleRate:       2.0,            // 这将导致验证错误
        TracesSampleRate: -0.1,           // 这将导致验证错误
    },
}

// 验证将失败并显示详细错误消息
if err := config.Validate(); err != nil {
    log.Fatal("配置验证失败:", err)
}
```

## 环境特定配置

### 开发配置

```yaml
# config/development.yaml
service_name: "my-service-dev"
http:
  port: "8080"
  middleware:
    enable_cors: true
    cors:
      allow_origins:
        - "http://localhost:3000"
        - "http://localhost:8080"
      allow_credentials: true
grpc:
  enable_reflection: true
  enable_health_service: true
discovery:
  enabled: false  # 本地开发时禁用
```

### 生产配置

```yaml
# config/production.yaml
service_name: "my-service"
shutdown_timeout: "60s"
http:
  port: "8080"
  read_timeout: "30s"
  write_timeout: "30s"
  middleware:
    enable_cors: true
    enable_rate_limit: true
    enable_auth: true
    cors:
      allow_origins:
        - "https://app.example.com"
        - "https://admin.example.com"
      allow_credentials: true
    rate_limit:
      requests_per_second: 1000
      burst_size: 2000
grpc:
  enable_reflection: false  # 生产环境中禁用
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/server.crt"
    key_file: "/etc/ssl/private/server.key"
discovery:
  enabled: true
  failure_mode: "fail_fast"
  health_check_required: true
```

## 环境变量

### 使用环境变量覆盖配置

环境变量遵循模式：`SWIT_<SECTION>_<FIELD>`

```bash
# 覆盖服务名称
export SWIT_SERVICE_NAME="my-service-prod"

# 覆盖 HTTP 端口
export SWIT_HTTP_PORT="9000"

# 覆盖 gRPC 设置
export SWIT_GRPC_ENABLED="true"
export SWIT_GRPC_PORT="9001"

# 覆盖发现设置
export SWIT_DISCOVERY_ENABLED="true"
export SWIT_DISCOVERY_ADDRESS="consul.example.com:8500"

# 覆盖嵌套配置
export SWIT_HTTP_MIDDLEWARE_ENABLE_CORS="false"
export SWIT_GRPC_TLS_ENABLED="true"
```

## 程序化配置

### 基本程序化设置

```go
config := server.NewServerConfig()
config.ServiceName = "my-service"
config.ShutdownTimeout = 30 * time.Second

// 配置 HTTP
config.HTTP.Enabled = true
config.HTTP.Port = "8080"
config.HTTP.Middleware.EnableCORS = true
config.HTTP.Middleware.CORSConfig.AllowOrigins = []string{
    "https://app.example.com",
}

// 配置 gRPC
config.GRPC.Enabled = true
config.GRPC.Port = "9080"
config.GRPC.EnableReflection = true

// 配置发现
config.Discovery.Enabled = true
config.Discovery.Address = "127.0.0.1:8500"
config.Discovery.ServiceName = "my-service"

// 验证配置
if err := config.Validate(); err != nil {
    log.Fatalf("配置无效: %v", err)
}
```

## 配置验证

### 内置验证

框架提供全面的验证：

```go
// 验证配置
if err := config.Validate(); err != nil {
    // 处理验证错误
    fmt.Printf("配置验证失败: %v\n", err)
}
```

### 常见验证错误

```go
// 端口冲突
"http.port and grpc.port must be different"

// 缺少必需字段
"service_name is required"
"http.port is required when HTTP is enabled"

// 无效值
"http.read_timeout must be positive"
"discovery.failure_mode must be one of: graceful, fail_fast, strict"

// CORS 安全违规
"CORS security violation: cannot use wildcard origin '*' with AllowCredentials=true"
```

## 配置加载

### 从文件加载

```go
// 从 YAML 文件加载
config, err := server.LoadConfigFromFile("config/production.yaml")
if err != nil {
    log.Fatalf("加载配置失败: %v", err)
}

// 使用环境覆盖加载
config, err := server.LoadConfigWithEnv("config/production.yaml")
if err != nil {
    log.Fatalf("加载配置失败: %v", err)
}
```

### 配置优先级

1. **程序化配置**（最高优先级）
2. **环境变量**
3. **配置文件**
4. **默认值**（最低优先级）

## 最佳实践

### 配置管理

1. **使用环境特定文件**：为开发、测试、生产分离配置
2. **早期验证**：在应用启动期间始终验证配置
3. **记录默认值**：清楚记录所有默认值及其含义
4. **保护机密**：永远不要在配置文件中存储机密；使用环境变量或机密管理
5. **版本化配置**：将配置文件保存在版本控制中

- 另见： [配置最佳实践](/zh/guide/configuration-best-practices)

### 安全最佳实践

1. **CORS 安全**：永远不要在带有凭据时使用通配符来源
2. **TLS 配置**：在生产环境中始终启用 TLS
3. **速率限制**：为您的用例配置适当的速率限制
4. **超时**：设置合理的超时以防止资源耗尽
5. **反射**：在生产中禁用 gRPC 反射

这个配置指南提供了 Swit 框架所有配置选项、最佳实践和故障排除指导的全面覆盖。


### 完整配置参考

- 查看自动生成参考：[/zh/guide/configuration-reference](/zh/guide/configuration-reference)
