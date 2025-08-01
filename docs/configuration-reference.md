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
    Port         string        `yaml:"port"`
    Enabled      bool          `yaml:"enabled"`
    EnableReady  bool          `yaml:"enable_ready"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout"`
}
```

### HTTP Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Port` | `string` | No | `"8080"` | HTTP port to listen on |
| `Enabled` | `bool` | No | `false` | Enable HTTP transport |
| `EnableReady` | `bool` | No | `true` | Enable `/ready` endpoint |
| `ReadTimeout` | `time.Duration` | Yes* | None | HTTP read timeout |
| `WriteTimeout` | `time.Duration` | Yes* | None | HTTP write timeout |
| `IdleTimeout` | `time.Duration` | Yes* | None | HTTP idle timeout |

*Required when HTTP is enabled

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
    Port                string               `yaml:"port"`
    Enabled             bool                 `yaml:"enabled"`
    EnableReflection    bool                 `yaml:"enable_reflection"`
    EnableHealthService bool                 `yaml:"enable_health_service"`
    MaxRecvMsgSize      int                  `yaml:"max_recv_msg_size"`
    MaxSendMsgSize      int                  `yaml:"max_send_msg_size"`
    KeepaliveParams     GRPCKeepaliveParams  `yaml:"keepalive_params"`
    KeepalivePolicy     GRPCKeepalivePolicy  `yaml:"keepalive_policy"`
}
```

### gRPC Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Port` | `string` | No | `"9090"` | gRPC port to listen on |
| `Enabled` | `bool` | No | `false` | Enable gRPC transport |
| `EnableReflection` | `bool` | No | `true` | Enable gRPC reflection |
| `EnableHealthService` | `bool` | No | `true` | Enable gRPC health service |
| `MaxRecvMsgSize` | `int` | Yes* | None | Maximum receive message size |
| `MaxSendMsgSize` | `int` | Yes* | None | Maximum send message size |

*Required when gRPC is enabled

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

| Field | Description | Recommended |
|-------|-------------|-------------|
| `MaxConnectionIdle` | Close idle connections after this duration | `15m` |
| `MaxConnectionAge` | Close connections after this duration | `30m` |
| `MaxConnectionAgeGrace` | Grace period for closing connections | `5m` |
| `Time` | Ping interval | `5m` |
| `Timeout` | Ping timeout | `1m` |

### GRPCKeepalivePolicy

Client keepalive enforcement policy.

```go
type GRPCKeepalivePolicy struct {
    MinTime             time.Duration `yaml:"min_time"`
    PermitWithoutStream bool          `yaml:"permit_without_stream"`
}
```

| Field | Description | Recommended |
|-------|-------------|-------------|
| `MinTime` | Minimum time between client pings | `5m` |
| `PermitWithoutStream` | Allow pings without active streams | `false` |

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
| `Enabled` | `bool` | No | `false` | Enable service discovery |
| `Address` | `string` | No | `"localhost:8500"` | Consul address |
| `ServiceName` | `string` | No | Same as `ServiceName` | Service name in discovery |
| `Tags` | `[]string` | No | `[]` | Service tags |

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
| `EnableCORS` | `bool` | `false` | Enable CORS middleware |
| `EnableAuth` | `bool` | `false` | Enable authentication middleware |
| `EnableRateLimit` | `bool` | `false` | Enable rate limiting middleware |
| `EnableLogging` | `bool` | `false` | Enable request logging middleware |

## Complete Configuration Example

```yaml
service_name: "my-service"

http:
  port: "8080"
  enabled: true
  enable_ready: true
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

grpc:
  port: "9090"
  enabled: true
  enable_reflection: true
  enable_health_service: true
  max_recv_msg_size: 4194304
  max_send_msg_size: 4194304
  keepalive_params:
    max_connection_idle: 15m
    max_connection_age: 30m
    max_connection_age_grace: 5m
    time: 5m
    timeout: 1m
  keepalive_policy:
    min_time: 5m
    permit_without_stream: false

shutdown_timeout: 30s

discovery:
  enabled: true
  address: "localhost:8500"
  service_name: "my-service"
  tags:
    - "api"
    - "v1"

middleware:
  enable_cors: true
  enable_auth: false
  enable_rate_limit: true
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
            Enabled:      getBoolEnv("HTTP_ENABLED", true),
            EnableReady:  getBoolEnv("HTTP_ENABLE_READY", true),
            ReadTimeout:  getDurationEnv("HTTP_READ_TIMEOUT", 30*time.Second),
            WriteTimeout: getDurationEnv("HTTP_WRITE_TIMEOUT", 30*time.Second),
            IdleTimeout:  getDurationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
        },
        GRPC: server.GRPCConfig{
            Port:                getEnv("GRPC_PORT", "9090"),
            Enabled:             getBoolEnv("GRPC_ENABLED", true),
            EnableReflection:    getBoolEnv("GRPC_ENABLE_REFLECTION", true),
            EnableHealthService: getBoolEnv("GRPC_ENABLE_HEALTH", true),
            MaxRecvMsgSize:      getIntEnv("GRPC_MAX_RECV_MSG_SIZE", 4*1024*1024),
            MaxSendMsgSize:      getIntEnv("GRPC_MAX_SEND_MSG_SIZE", 4*1024*1024),
        },
        ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
        Discovery: server.DiscoveryConfig{
            Enabled:     getBoolEnv("DISCOVERY_ENABLED", false),
            Address:     getEnv("CONSUL_ADDRESS", "localhost:8500"),
            ServiceName: getEnv("DISCOVERY_SERVICE_NAME", "my-service"),
            Tags:        getStringSliceEnv("DISCOVERY_TAGS", []string{}),
        },
        Middleware: server.MiddlewareConfig{
            EnableCORS:      getBoolEnv("MIDDLEWARE_CORS", true),
            EnableAuth:      getBoolEnv("MIDDLEWARE_AUTH", false),
            EnableRateLimit: getBoolEnv("MIDDLEWARE_RATE_LIMIT", false),
            EnableLogging:   getBoolEnv("MIDDLEWARE_LOGGING", true),
        },
    }
}
```

## Configuration Validation

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