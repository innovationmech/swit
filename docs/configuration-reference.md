# 分层配置设计与合并语义

本设计满足 Issue #359：提供分层配置结构及清晰的加载与合并规则，适用于 SWIT 框架与消息系统。

## 层级与优先级（低 → 高）

1. Defaults（代码内默认值，`SetDefault`）
2. Base 文件（`swit.yaml`）
3. Environment 文件（`swit.<env>.yaml`，如 `swit.dev.yaml`）
4. Override 文件（`swit.override.yaml`）
5. 环境变量（`SWIT_` 前缀，`.` → `_`）

更高优先级将覆盖更低优先级的相同键。嵌套 Map 采用浅覆盖：子键按键级别分别覆盖，未覆盖的键保持原值。

## 命名与环境变量映射

示例键：`messaging.publisher.batch_size` → 环境变量：`SWIT_MESSAGING_PUBLISHER_BATCH_SIZE`。

- **前缀**：默认 `SWIT`，可通过 `Options.EnvPrefix` 定制
- **键映射**：配置键中的 `.` 将映射为 `_`（通过 `viper.SetEnvKeyReplacer(".", "_")`）
- **优先级**：环境变量始终高于文件与默认值

## 文件与路径约定

- 基础名：`swit`（可配置）
- 类型：`yaml`（支持 `yml/json`）
- 工作目录：进程工作目录（可配置）

## 文件内变量插值（${VAR} 与 ${VAR:-default}）

- 启用方式：`Options.EnableInterpolation = true`（默认开启）
- 语法：
  - `${VAR}`：替换为环境变量 `VAR` 的值，未设置则替换为空字符串
  - `${VAR:-default}`：若 `VAR` 未设置或为空，则使用 `default`
  - `$VAR`：遵循系统标准展开（`os.ExpandEnv`）
- 示例：
```yaml
server:
  port: "${MY_PORT:-8080}"
```
在设置 `MY_PORT=9090` 时，最终 `server.port=9090`；未设置时，`server.port=8080`。

注意：插值发生在文件合并之后、环境变量覆盖之前，不改变 SWIT_* 覆盖优先级。

## 机密文件注入（*_FILE）

为便于从容器 Secret/挂载文件中注入机密，支持将 `*_FILE` 的内容注入为同名环境变量。

- 启用方式：默认开启；后缀可通过 `Options.SecretFileSuffix` 定制（默认 `_FILE`）
- 作用范围：仅处理前缀为 `EnvPrefix`（默认 `SWIT_`）的环境变量，例如 `SWIT_DB_PASSWORD_FILE`
- 冲突保护：若同时设置了 `SWIT_FOO` 与 `SWIT_FOO_FILE`，加载将报错以避免歧义
- 示例：
```bash
# 将文件内容注入为 SWIT_SERVER_PORT（会自动去掉结尾换行）
export SWIT_SERVER_PORT_FILE=/run/secrets/server_port
```
此时 `SWIT_SERVER_PORT` 会在加载时被设置为文件内容，随后按照常规 SWIT_* 覆盖规则生效。

## 示例配置

```yaml
# swit.yaml（Base）
server:
  port: "9000"
messaging:
  broker:
    type: rabbitmq
  publisher:
    batch_size: 10
    timeout: 5
```

```yaml
# swit.dev.yaml（Environment）
server:
  port: "9100"
messaging:
  broker:
    type: kafka
  publisher:
    timeout: 8
```

```yaml
# swit.override.yaml（Override）
server:
  port: "9200"
messaging:
  publisher:
    batch_size: 100
```

环境变量示例：

```bash
export SWIT_SERVER_PORT=9300
export SWIT_MESSAGING_PUBLISHER_BATCH_SIZE=200
```

最终生效：`server.port=9300`，`messaging.publisher.batch_size=200`，`messaging.publisher.timeout=8`，`messaging.broker.type=kafka`。

## 使用方法

```go
opts := config.DefaultOptions()
opts.WorkDir = "."
opts.EnvironmentName = "dev"
m := config.NewManager(opts)
m.SetDefault("server.port", "8080")
_ = m.Load()
var cfg MyConfig
_ = m.Unmarshal(&cfg)
```

## 与现有组件的关系

- `internal/switserve/config` 当前直接 `viper` 加载 `swit.yaml`。后续可切换为 `pkg/config.Manager`，以获得层级与覆盖能力。
- `internal/switctl/config` 已有分层实现，本包为框架通用实现，面向服务侧与库侧共用。

# Configuration Reference

This document provides a comprehensive reference for configuring the base server framework.

## ServerConfig

The main configuration structure for the base server.

```go
type ServerConfig struct {
    ServiceName     string            `yaml:"service_name"`
    HTTP            HTTPConfig        `yaml:"http"`
    GRPC            GRPCConfig        `yaml:"grpc"`
    ShutdownTimeout time.Duration     `yaml:"shutdown_timeout"`
    Discovery       DiscoveryConfig   `yaml:"discovery"`
    Middleware      MiddlewareConfig  `yaml:"middleware"`
    Messaging       MessagingConfig   `yaml:"messaging"`
    AccessControl   AccessControlConfig `yaml:"access_control"`
}
```

### ServiceName

- **Type**: `string`
- **Required**: Yes
- **Description**: The name of the service used for logging, metrics, and service discovery
- **Example**: `"my-service"`

### ShutdownTimeout

- **Type**: `time.Duration`
- **Required**: Yes
- **Description**: Maximum time to wait for graceful shutdown
- **Default**: None (must be specified)
- **Recommended**: `30 * time.Second`
- **Example**: `"30s"`

## HTTPConfig

Configuration for the HTTP transport layer.

```go
type HTTPConfig struct {
    Port         string            `yaml:"port"`
    Address      string            `yaml:"address"`
    EnableReady  bool              `yaml:"enable_ready"`
    Enabled      bool              `yaml:"enabled"`
    TestMode     bool              `yaml:"test_mode"`
    TestPort     string            `yaml:"test_port"`
    Middleware   HTTPMiddleware    `yaml:"middleware"`
    ReadTimeout  time.Duration     `yaml:"read_timeout"`
    WriteTimeout time.Duration     `yaml:"write_timeout"`
    IdleTimeout  time.Duration     `yaml:"idle_timeout"`
    Headers      map[string]string `yaml:"headers"`
}
```

### HTTP Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Port` | `string` | No | `"8080"` | HTTP port to listen on |
| `Address` | `string` | No | `":8080"` | HTTP address to bind to |
| `Enabled` | `bool` | No | `true` | Enable HTTP transport |
| `EnableReady` | `bool` | No | `true` | Enable ready channel for testing |
| `TestMode` | `bool` | No | `false` | Enable test mode features |
| `TestPort` | `string` | No | `""` | Test port override |
| `Middleware` | `HTTPMiddleware` | No | See HTTPMiddleware | HTTP middleware configuration |
| `ReadTimeout` | `time.Duration` | No | `30s` | HTTP read timeout |
| `WriteTimeout` | `time.Duration` | No | `30s` | HTTP write timeout |
| `IdleTimeout` | `time.Duration` | No | `120s` | HTTP idle timeout |
| `Headers` | `map[string]string` | No | `{}` | Default HTTP headers |

### HTTP Timeout Recommendations

```yaml
http:
  read_timeout: 30s    # Time to read request headers and body
  write_timeout: 30s   # Time to write response
  idle_timeout: 60s    # Time to keep connections alive
```

## GRPCConfig

Configuration for the gRPC transport layer.

```go
type GRPCConfig struct {
    Port                string                `yaml:"port"`
    Address             string                `yaml:"address"`
    EnableKeepalive     bool                  `yaml:"enable_keepalive"`
    EnableReflection    bool                  `yaml:"enable_reflection"`
    EnableHealthService bool                  `yaml:"enable_health_service"`
    Enabled             bool                  `yaml:"enabled"`
    TestMode            bool                  `yaml:"test_mode"`
    TestPort            string                `yaml:"test_port"`
    MaxRecvMsgSize      int                   `yaml:"max_recv_msg_size"`
    MaxSendMsgSize      int                   `yaml:"max_send_msg_size"`
    KeepaliveParams     GRPCKeepaliveParams   `yaml:"keepalive_params"`
    KeepalivePolicy     GRPCKeepalivePolicy   `yaml:"keepalive_policy"`
    Interceptors        GRPCInterceptorConfig `yaml:"interceptors"`
    TLS                 GRPCTLSConfig         `yaml:"tls"`
}
```

### gRPC Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Port` | `string` | No | `"9080"` | gRPC port to listen on (HTTP port + 1000) |
| `Address` | `string` | No | `":9080"` | gRPC address to bind to |
| `Enabled` | `bool` | No | `true` | Enable gRPC transport |
| `EnableKeepalive` | `bool` | No | `true` | Enable keepalive parameters |
| `EnableReflection` | `bool` | No | `true` | Enable gRPC reflection |
| `EnableHealthService` | `bool` | No | `true` | Enable gRPC health service |
| `TestMode` | `bool` | No | `false` | Enable test mode features |
| `TestPort` | `string` | No | `""` | Test port override |
| `MaxRecvMsgSize` | `int` | No | `4MB` | Maximum receive message size |
| `MaxSendMsgSize` | `int` | No | `4MB` | Maximum send message size |
| `KeepaliveParams` | `GRPCKeepaliveParams` | No | See below | Keepalive parameters |
| `KeepalivePolicy` | `GRPCKeepalivePolicy` | No | See below | Keepalive enforcement policy |
| `Interceptors` | `GRPCInterceptorConfig` | No | See below | Interceptor configuration |
| `TLS` | `GRPCTLSConfig` | No | See below | TLS configuration |

### GRPCKeepaliveParams

Server-side keepalive parameters.

```go
type GRPCKeepaliveParams struct {
    MaxConnectionIdle     time.Duration `yaml:"max_connection_idle"`
    MaxConnectionAge      time.Duration `yaml:"max_connection_age"`
    MaxConnectionAgeGrace time.Duration `yaml:"max_connection_age_grace"`
    Time                  time.Duration `yaml:"time"`
    Timeout               time.Duration `yaml:"timeout"`
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `MaxConnectionIdle` | Close idle connections after this duration | `15s` |
| `MaxConnectionAge` | Close connections after this duration | `30s` |
| `MaxConnectionAgeGrace` | Grace period for closing connections | `5s` |
| `Time` | Ping interval | `5s` |
| `Timeout` | Ping timeout | `1s` |

### GRPCKeepalivePolicy

Client keepalive enforcement policy.

```go
type GRPCKeepalivePolicy struct {
    MinTime             time.Duration `yaml:"min_time"`
    PermitWithoutStream bool          `yaml:"permit_without_stream"`
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `MinTime` | Minimum time between client pings | `5s` |
| `PermitWithoutStream` | Allow pings without active streams | `true` |

### gRPC Configuration Example

```yaml
grpc:
  port: "9090"
  enabled: true
  enable_reflection: true
  enable_health_service: true
  max_recv_msg_size: 4194304  # 4MB
  max_send_msg_size: 4194304  # 4MB
  keepalive_params:
    max_connection_idle: 15m
    max_connection_age: 30m
    max_connection_age_grace: 5m
    time: 5m
    timeout: 1m
  keepalive_policy:
    min_time: 5m
    permit_without_stream: false
```

## DiscoveryConfig

Configuration for service discovery integration.

```go
type DiscoveryConfig struct {
    Enabled     bool     `yaml:"enabled"`
    Address     string   `yaml:"address"`
    ServiceName string   `yaml:"service_name"`
    Tags        []string `yaml:"tags"`
}
```

### Discovery Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Enabled` | `bool` | No | `true` | Enable service discovery |
| `Address` | `string` | No | `"127.0.0.1:8500"` | Consul address |
| `ServiceName` | `string` | No | Same as `ServiceName` | Service name in discovery |
| `Tags` | `[]string` | No | `["v1"]` | Service tags |

### Discovery Configuration Example

```yaml
discovery:
  enabled: true
  address: "consul.example.com:8500"
  service_name: "my-service"
  tags:
    - "api"
    - "v1"
    - "production"
```

## MiddlewareConfig

Configuration for middleware components.

```go
type MiddlewareConfig struct {
    EnableCORS      bool `yaml:"enable_cors"`
    EnableAuth      bool `yaml:"enable_auth"`
    EnableRateLimit bool `yaml:"enable_rate_limit"`
    EnableLogging   bool `yaml:"enable_logging"`
}
```

### Middleware Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `EnableCORS` | `bool` | `true` | Enable CORS middleware |
| `EnableAuth` | `bool` | `false` | Enable authentication middleware |
| `EnableRateLimit` | `bool` | `false` | Enable rate limiting middleware |
| `EnableLogging` | `bool` | `true` | Enable request logging middleware |

## Additional Configuration Structures

### HTTPMiddleware

Detailed HTTP middleware configuration.

```go
type HTTPMiddleware struct {
    EnableCORS      bool              `yaml:"enable_cors"`
    EnableAuth      bool              `yaml:"enable_auth"`
    EnableRateLimit bool              `yaml:"enable_rate_limit"`
    EnableLogging   bool              `yaml:"enable_logging"`
    EnableTimeout   bool              `yaml:"enable_timeout"`
    CORSConfig      CORSConfig        `yaml:"cors"`
    RateLimitConfig RateLimitConfig   `yaml:"rate_limit"`
    TimeoutConfig   TimeoutConfig     `yaml:"timeout"`
    CustomHeaders   map[string]string `yaml:"custom_headers"`
}
```

#### HTTPMiddleware Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `EnableCORS` | `bool` | `true` | Enable CORS middleware |
| `EnableAuth` | `bool` | `false` | Enable authentication middleware |
| `EnableRateLimit` | `bool` | `false` | Enable rate limiting |
| `EnableLogging` | `bool` | `true` | Enable request logging |
| `EnableTimeout` | `bool` | `true` | Enable request timeout |
| `CORSConfig` | `CORSConfig` | See CORSConfig | CORS configuration |
| `RateLimitConfig` | `RateLimitConfig` | See RateLimitConfig | Rate limiting configuration |
| `TimeoutConfig` | `TimeoutConfig` | See TimeoutConfig | Timeout configuration |
| `CustomHeaders` | `map[string]string` | `{}` | Custom headers to add |

### CORSConfig

CORS middleware configuration.

```go
type CORSConfig struct {
    AllowOrigins     []string `yaml:"allow_origins"`
    AllowMethods     []string `yaml:"allow_methods"`
    AllowHeaders     []string `yaml:"allow_headers"`
    ExposeHeaders    []string `yaml:"expose_headers"`
    AllowCredentials bool     `yaml:"allow_credentials"`
    MaxAge           int      `yaml:"max_age"`
}
```

#### CORSConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `AllowOrigins` | `[]string` | `["*"]` | Allowed origins |
| `AllowMethods` | `[]string` | `["GET", "POST", "PUT", "DELETE", "OPTIONS"]` | Allowed HTTP methods |
| `AllowHeaders` | `[]string` | `["Origin", "Content-Type", "Accept", "Authorization"]` | Allowed headers |
| `ExposeHeaders` | `[]string` | `[]` | Headers to expose |
| `AllowCredentials` | `bool` | `true` | Allow credentials |
| `MaxAge` | `int` | `86400` | Preflight cache duration (seconds) |

### RateLimitConfig

Rate limiting configuration.

```go
type RateLimitConfig struct {
    RequestsPerSecond int           `yaml:"requests_per_second"`
    BurstSize         int           `yaml:"burst_size"`
    WindowSize        time.Duration `yaml:"window_size"`
    KeyFunc           string        `yaml:"key_func"`
}
```

#### RateLimitConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `RequestsPerSecond` | `int` | `100` | Maximum requests per second |
| `BurstSize` | `int` | `200` | Burst size for rate limiting |
| `WindowSize` | `time.Duration` | `1m` | Rate limiting window size |
| `KeyFunc` | `string` | `"ip"` | Key function for rate limiting ("ip", "user", "custom") |

### TimeoutConfig

Request timeout configuration.

```go
type TimeoutConfig struct {
    RequestTimeout time.Duration `yaml:"request_timeout"`
    HandlerTimeout time.Duration `yaml:"handler_timeout"`
}
```

#### TimeoutConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `RequestTimeout` | `time.Duration` | `30s` | Total request timeout |
| `HandlerTimeout` | `time.Duration` | `25s` | Handler execution timeout |

### GRPCInterceptorConfig

gRPC interceptor configuration.

```go
type GRPCInterceptorConfig struct {
    EnableAuth      bool `yaml:"enable_auth"`
    EnableLogging   bool `yaml:"enable_logging"`
    EnableMetrics   bool `yaml:"enable_metrics"`
    EnableRecovery  bool `yaml:"enable_recovery"`
    EnableRateLimit bool `yaml:"enable_rate_limit"`
}
```

#### GRPCInterceptorConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `EnableAuth` | `bool` | `false` | Enable authentication interceptor |
| `EnableLogging` | `bool` | `true` | Enable logging interceptor |
| `EnableMetrics` | `bool` | `false` | Enable metrics interceptor |
| `EnableRecovery` | `bool` | `true` | Enable panic recovery interceptor |
| `EnableRateLimit` | `bool` | `false` | Enable rate limiting interceptor |

### GRPCTLSConfig

gRPC TLS configuration.

```go
type GRPCTLSConfig struct {
    Enabled    bool   `yaml:"enabled"`
    CertFile   string `yaml:"cert_file"`
    KeyFile    string `yaml:"key_file"`
    CAFile     string `yaml:"ca_file"`
    ServerName string `yaml:"server_name"`
}
```

## AccessControlConfig

Role-Based Access Control (RBAC) configuration for services.

```go
type AccessControlConfig struct {
    Enabled           bool                          `yaml:"enabled"`
    StrictMode        bool                          `yaml:"strict_mode"`
    SuperAdminRole    string                        `yaml:"super_admin_role"`
    DefaultRoles      []string                      `yaml:"default_roles"`
    ResourceSeparator string                        `yaml:"resource_separator"`
    Permissions       map[string]PermissionDefinition `yaml:"permissions"`
    Roles             map[string]RoleDefinition       `yaml:"roles"`
}

type PermissionDefinition struct {
    Description string `yaml:"description"`
    Resource    string `yaml:"resource"`
    Action      string `yaml:"action"`
    IsSystem    bool   `yaml:"is_system"`
}

type RoleDefinition struct {
    Description string   `yaml:"description"`
    Permissions []string `yaml:"permissions"`
    Inherits    []string `yaml:"inherits"`
    IsSystem    bool     `yaml:"is_system"`
}
```

### AccessControl Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Enabled` | `bool` | `false` | Enable RBAC configuration validation |
| `StrictMode` | `bool` | `false` | Enforce strict validation at startup |
| `SuperAdminRole` | `string` | `""` | Role that has all permissions |
| `DefaultRoles` | `[]string` | `[]` | Roles assigned to new identities |
| `ResourceSeparator` | `string` | `":"` | Separator for inline permissions (e.g., `orders:read`) |
| `Permissions` | `map[string]PermissionDefinition` | `{}` | Registry of named permissions |
| `Roles` | `map[string]RoleDefinition` | `{}` | Registry of roles with permissions and inheritance |

### AccessControl Example

```yaml
access_control:
  enabled: true
  strict_mode: true
  super_admin_role: "admin"
  default_roles: ["reader"]
  resource_separator: ":"
  permissions:
    orders_read:
      description: "Read orders"
      resource: "orders"
      action: "read"
  roles:
    reader:
      description: "Read-only"
      permissions: ["orders_read", "inventory:read"]
      inherits: []
    admin:
      description: "Administrator"
      permissions: ["*"]
      inherits: ["reader"]
      is_system: true
```

#### GRPCTLSConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Enabled` | `bool` | `false` | Enable TLS |
| `CertFile` | `string` | `""` | Certificate file path |
| `KeyFile` | `string` | `""` | Private key file path |
| `CAFile` | `string` | `""` | CA certificate file path |
| `ServerName` | `string` | `""` | Server name for TLS |

## Complete Configuration Example

```yaml
service_name: "my-service"

http:
  port: "8080"
  address: ":8080"
  enabled: true
  enable_ready: true
  test_mode: false
  test_port: ""
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  headers:
    X-Service-Version: "v1.0.0"
    X-Request-ID: "auto-generated"
  middleware:
    enable_cors: true
    enable_auth: false
    enable_rate_limit: true
    enable_logging: true
    enable_timeout: true
    cors:
      allow_origins: ["*"]
      allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
      allow_headers: ["Origin", "Content-Type", "Accept", "Authorization"]
      expose_headers: []
      allow_credentials: true
      max_age: 86400
    rate_limit:
      requests_per_second: 100
      burst_size: 200
      window_size: 1m
      key_func: "ip"
    timeout:
      request_timeout: 30s
      handler_timeout: 25s
    custom_headers:
      X-API-Version: "v1"
      X-Service-Name: "my-service"

grpc:
  port: "9080"
  address: ":9080"
  enabled: true
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true
  test_mode: false
  test_port: ""
  max_recv_msg_size: 4194304
  max_send_msg_size: 4194304
  keepalive_params:
    max_connection_idle: 15s
    max_connection_age: 30s
    max_connection_age_grace: 5s
    time: 5s
    timeout: 1s
  keepalive_policy:
    min_time: 5s
    permit_without_stream: true
  interceptors:
    enable_auth: false
    enable_logging: true
    enable_metrics: false
    enable_recovery: true
    enable_rate_limit: false
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""
    server_name: ""

shutdown_timeout: 30s

discovery:
  enabled: true
  address: "127.0.0.1:8500"
  service_name: "my-service"
  tags:
    - "api"
    - "v1"

middleware:
  enable_cors: true
  enable_auth: false
  enable_rate_limit: false
  enable_logging: true
```

## Environment Variable Configuration

You can use environment variables to override configuration values:

```go
func loadConfig() *server.ServerConfig {
    return &server.ServerConfig{
        ServiceName: getEnv("SERVICE_NAME", "my-service"),
        HTTP: server.HTTPConfig{
            Port:         getEnv("HTTP_PORT", "8080"),
            Address:      getEnv("HTTP_ADDRESS", ":8080"),
            Enabled:      getBoolEnv("HTTP_ENABLED", true),
            EnableReady:  getBoolEnv("HTTP_ENABLE_READY", true),
            TestMode:     getBoolEnv("HTTP_TEST_MODE", false),
            TestPort:     getEnv("HTTP_TEST_PORT", ""),
            ReadTimeout:  getDurationEnv("HTTP_READ_TIMEOUT", 30*time.Second),
            WriteTimeout: getDurationEnv("HTTP_WRITE_TIMEOUT", 30*time.Second),
            IdleTimeout:  getDurationEnv("HTTP_IDLE_TIMEOUT", 120*time.Second),
            Headers:      getMapEnv("HTTP_HEADERS", map[string]string{}),
            Middleware: server.HTTPMiddleware{
                EnableCORS:      getBoolEnv("HTTP_MIDDLEWARE_CORS", true),
                EnableAuth:      getBoolEnv("HTTP_MIDDLEWARE_AUTH", false),
                EnableRateLimit: getBoolEnv("HTTP_MIDDLEWARE_RATE_LIMIT", false),
                EnableLogging:   getBoolEnv("HTTP_MIDDLEWARE_LOGGING", true),
                EnableTimeout:   getBoolEnv("HTTP_MIDDLEWARE_TIMEOUT", true),
                CORSConfig: server.CORSConfig{
                    AllowOrigins:     getStringSliceEnv("CORS_ALLOW_ORIGINS", []string{"*"}),
                    AllowMethods:     getStringSliceEnv("CORS_ALLOW_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
                    AllowHeaders:     getStringSliceEnv("CORS_ALLOW_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"}),
                    AllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", true),
                    MaxAge:           getIntEnv("CORS_MAX_AGE", 86400),
                },
                RateLimitConfig: server.RateLimitConfig{
                    RequestsPerSecond: getIntEnv("RATE_LIMIT_RPS", 100),
                    BurstSize:         getIntEnv("RATE_LIMIT_BURST", 200),
                    WindowSize:        getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
                    KeyFunc:           getEnv("RATE_LIMIT_KEY_FUNC", "ip"),
                },
                TimeoutConfig: server.TimeoutConfig{
                    RequestTimeout: getDurationEnv("TIMEOUT_REQUEST", 30*time.Second),
                    HandlerTimeout: getDurationEnv("TIMEOUT_HANDLER", 25*time.Second),
                },
                CustomHeaders: getMapEnv("HTTP_CUSTOM_HEADERS", map[string]string{}),
            },
        },
        GRPC: server.GRPCConfig{
            Port:                getEnv("GRPC_PORT", "9080"),
            Address:             getEnv("GRPC_ADDRESS", ":9080"),
            Enabled:             getBoolEnv("GRPC_ENABLED", true),
            EnableKeepalive:     getBoolEnv("GRPC_ENABLE_KEEPALIVE", true),
            EnableReflection:    getBoolEnv("GRPC_ENABLE_REFLECTION", true),
            EnableHealthService: getBoolEnv("GRPC_ENABLE_HEALTH", true),
            TestMode:            getBoolEnv("GRPC_TEST_MODE", false),
            TestPort:            getEnv("GRPC_TEST_PORT", ""),
            MaxRecvMsgSize:      getIntEnv("GRPC_MAX_RECV_MSG_SIZE", 4*1024*1024),
            MaxSendMsgSize:      getIntEnv("GRPC_MAX_SEND_MSG_SIZE", 4*1024*1024),
            KeepaliveParams: server.GRPCKeepaliveParams{
                MaxConnectionIdle:     getDurationEnv("GRPC_KEEPALIVE_MAX_IDLE", 15*time.Second),
                MaxConnectionAge:      getDurationEnv("GRPC_KEEPALIVE_MAX_AGE", 30*time.Second),
                MaxConnectionAgeGrace: getDurationEnv("GRPC_KEEPALIVE_MAX_AGE_GRACE", 5*time.Second),
                Time:                  getDurationEnv("GRPC_KEEPALIVE_TIME", 5*time.Second),
                Timeout:               getDurationEnv("GRPC_KEEPALIVE_TIMEOUT", 1*time.Second),
            },
            KeepalivePolicy: server.GRPCKeepalivePolicy{
                MinTime:             getDurationEnv("GRPC_KEEPALIVE_MIN_TIME", 5*time.Second),
                PermitWithoutStream: getBoolEnv("GRPC_KEEPALIVE_PERMIT_WITHOUT_STREAM", true),
            },
            Interceptors: server.GRPCInterceptorConfig{
                EnableAuth:      getBoolEnv("GRPC_INTERCEPTOR_AUTH", false),
                EnableLogging:   getBoolEnv("GRPC_INTERCEPTOR_LOGGING", true),
                EnableMetrics:   getBoolEnv("GRPC_INTERCEPTOR_METRICS", false),
                EnableRecovery:  getBoolEnv("GRPC_INTERCEPTOR_RECOVERY", true),
                EnableRateLimit: getBoolEnv("GRPC_INTERCEPTOR_RATE_LIMIT", false),
            },
            TLS: server.GRPCTLSConfig{
                Enabled:    getBoolEnv("GRPC_TLS_ENABLED", false),
                CertFile:   getEnv("GRPC_TLS_CERT_FILE", ""),
                KeyFile:    getEnv("GRPC_TLS_KEY_FILE", ""),
                CAFile:     getEnv("GRPC_TLS_CA_FILE", ""),
                ServerName: getEnv("GRPC_TLS_SERVER_NAME", ""),
            },
        },
        ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
        Discovery: server.DiscoveryConfig{
            Enabled:     getBoolEnv("DISCOVERY_ENABLED", true),
            Address:     getEnv("CONSUL_ADDRESS", "127.0.0.1:8500"),
            ServiceName: getEnv("DISCOVERY_SERVICE_NAME", "my-service"),
            Tags:        getStringSliceEnv("DISCOVERY_TAGS", []string{"v1"}),
        },
        Middleware: server.MiddlewareConfig{
            EnableCORS:      getBoolEnv("MIDDLEWARE_CORS", true),
            EnableAuth:      getBoolEnv("MIDDLEWARE_AUTH", false),
            EnableRateLimit: getBoolEnv("MIDDLEWARE_RATE_LIMIT", false),
            EnableLogging:   getBoolEnv("MIDDLEWARE_LOGGING", true),
        },
    }
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if parsed, err := strconv.ParseBool(value); err == nil {
            return parsed
        }
    }
    return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if parsed, err := strconv.Atoi(value); err == nil {
            return parsed
        }
    }
    return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if parsed, err := time.ParseDuration(value); err == nil {
            return parsed
        }
    }
    return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
    if value := os.Getenv(key); value != "" {
        return strings.Split(value, ",")
    }
    return defaultValue
}

func getMapEnv(key string, defaultValue map[string]string) map[string]string {
    if value := os.Getenv(key); value != "" {
        result := make(map[string]string)
        pairs := strings.Split(value, ",")
        for _, pair := range pairs {
            if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
                result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
            }
        }
        return result
    }
    return defaultValue
}
```

## Configuration Validation

### Compliance Configuration

```yaml
messaging:
  brokers:
    kafka:
      type: kafka
      endpoints: ["localhost:9092"]
      compliance:
        enabled: true
        retention:
          enabled: true
          default_period: "30d"
          topic_overrides:
            audit.logs: "90d"
        redaction:
          enabled: true
          fields: ["authorization", "x-api-key"]
          payload_enabled: false
        residency:
          enforce: true
          allowed_regions: ["eu-west-1", "eu-central-1"]
        reporting:
          enabled: true
          metrics_enabled: true
          log_enabled: true
          interval: "1m"
```

Compliance 验证规则：
- enabled=false 时跳过校验
- retention.enabled=true 时 `default_period` 必须为正数，`topic_overrides` 中每个时长必须为正数
- residency.enforce=true 时 `allowed_regions` 不能为空
- reporting.enabled=true 时 `interval` 必须为正数

The framework provides built-in configuration validation:

```go
config := loadConfig()
if err := config.Validate(); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

### Validation Rules

1. **ServiceName**: Must not be empty
2. **HTTP**: If enabled, timeouts must be positive
3. **gRPC**: If enabled, message sizes must be positive
4. **ShutdownTimeout**: Must be positive
5. **Ports**: Must be valid port numbers or "0" for auto-assignment

## Best Practices

1. **Use Environment Variables**: For deployment-specific values
2. **Set Reasonable Timeouts**: Based on your service requirements
3. **Enable Health Checks**: For monitoring and load balancing
4. **Use Service Discovery**: For production deployments
5. **Configure Keepalive**: For long-lived gRPC connections
6. **Validate Early**: Check configuration at startup
7. **Document Defaults**: Make default values clear to operators