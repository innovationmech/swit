# OPA 环境变量配置示例

本文档展示如何使用环境变量配置 OPA 客户端。

## 方式 1: 使用 `LoadFromEnv()` 方法

OPA Config 提供了 `LoadFromEnv()` 方法，可以手动从环境变量加载配置：

```go
import "github.com/innovationmech/swit/pkg/security/opa"

// 创建配置
config := &opa.Config{}

// 从环境变量加载配置（会覆盖现有值）
config.LoadFromEnv()

// 设置默认值
config.SetDefaults()

// 验证配置
if err := config.Validate(); err != nil {
    log.Fatalf("Invalid OPA config: %v", err)
}

// 创建 OPA 客户端
client, err := opa.NewClient(ctx, config)
if err != nil {
    log.Fatalf("Failed to create OPA client: %v", err)
}
```

## 方式 2: 使用 Viper/Config Manager（推荐）

如果使用 `pkg/config.Manager`，环境变量会通过 Viper 自动处理：

```go
import (
    "github.com/innovationmech/swit/pkg/config"
    "github.com/innovationmech/swit/pkg/security/opa"
)

// 创建配置管理器
mgr := config.NewManager(config.Options{
    ConfigBaseName:      "swit",
    ConfigType:          "yaml",
    EnvPrefix:           "SWIT",
    EnableAutomaticEnv:  true,
})

// 加载配置文件和环境变量
if err := mgr.Load(); err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// 解析到结构体
type AppConfig struct {
    OPA *opa.Config `mapstructure:"opa"`
}

var appConfig AppConfig
if err := mgr.Unmarshal(&appConfig); err != nil {
    log.Fatalf("Failed to unmarshal config: %v", err)
}

// 使用 OPA 配置
client, err := opa.NewClient(ctx, appConfig.OPA)
```

## 环境变量命名规范

环境变量使用 `OPA_` 前缀，嵌套字段使用下划线分隔：

### 基本配置

```bash
# 模式：embedded, remote, sidecar
export OPA_MODE=remote

# 默认决策路径
export OPA_DEFAULT_DECISION_PATH=authz/allow
```

### 嵌入式模式配置

```bash
export OPA_MODE=embedded
export OPA_EMBEDDED_POLICY_DIR=/etc/opa/policies
export OPA_EMBEDDED_DATA_DIR=/var/opa/data
export OPA_EMBEDDED_ENABLE_LOGGING=true
export OPA_EMBEDDED_ENABLE_DECISION_LOGS=false
```

### Bundle 配置（嵌入式模式）

```bash
export OPA_BUNDLE_SERVICE_URL=http://bundle-server:8888
export OPA_BUNDLE_RESOURCE=/bundles/policies.tar.gz
export OPA_BUNDLE_POLLING_MIN_DELAY_SECONDS=60
export OPA_BUNDLE_POLLING_MAX_DELAY_SECONDS=120
export OPA_BUNDLE_ENABLE_SIGNING=true
export OPA_BUNDLE_SIGNING_KEY_ID=my-key-id
```

### 远程模式配置

```bash
export OPA_MODE=remote
export OPA_REMOTE_URL=http://opa-server:8181
export OPA_REMOTE_TIMEOUT=30s
export OPA_REMOTE_MAX_RETRIES=3
```

### Sidecar 模式配置

```bash
# sidecar 模式等价于 remote 模式，连接到本地 sidecar 容器
export OPA_MODE=sidecar
export OPA_REMOTE_URL=http://localhost:8181
export OPA_REMOTE_TIMEOUT=10s
```

### TLS 配置

```bash
export OPA_REMOTE_TLS_ENABLED=true
export OPA_REMOTE_TLS_CERT_FILE=/etc/opa/certs/client.crt
export OPA_REMOTE_TLS_KEY_FILE=/etc/opa/certs/client.key
export OPA_REMOTE_TLS_CA_FILE=/etc/opa/certs/ca.crt
export OPA_REMOTE_TLS_INSECURE_SKIP_VERIFY=false
```

### 认证配置

#### Bearer Token 认证

```bash
export OPA_REMOTE_AUTH_TYPE=bearer
export OPA_REMOTE_AUTH_TOKEN=your-token-here
```

#### 基本认证

```bash
export OPA_REMOTE_AUTH_TYPE=basic
export OPA_REMOTE_AUTH_USERNAME=admin
export OPA_REMOTE_AUTH_PASSWORD=secret
```

#### API Key 认证

```bash
export OPA_REMOTE_AUTH_TYPE=api_key
export OPA_REMOTE_AUTH_API_KEY=your-api-key
export OPA_REMOTE_AUTH_API_KEY_HEADER=X-API-Key
```

### 缓存配置

```bash
export OPA_CACHE_ENABLED=true
export OPA_CACHE_MAX_SIZE=10000
export OPA_CACHE_TTL=5m
export OPA_CACHE_ENABLE_METRICS=true
```

## 完整示例

### 示例 1: 开发环境 - 嵌入式模式

```bash
#!/bin/bash
# dev-opa.sh

export OPA_MODE=embedded
export OPA_EMBEDDED_POLICY_DIR=./pkg/security/opa/policies
export OPA_EMBEDDED_ENABLE_LOGGING=true
export OPA_DEFAULT_DECISION_PATH=rbac/allow

# 启用缓存
export OPA_CACHE_ENABLED=true
export OPA_CACHE_MAX_SIZE=5000
export OPA_CACHE_TTL=3m

./your-app
```

### 示例 2: 生产环境 - 远程模式 + TLS + Bearer Token

```bash
#!/bin/bash
# prod-opa.sh

export OPA_MODE=remote
export OPA_REMOTE_URL=https://opa.prod.example.com:8181
export OPA_REMOTE_TIMEOUT=30s
export OPA_REMOTE_MAX_RETRIES=3

# TLS 配置
export OPA_REMOTE_TLS_ENABLED=true
export OPA_REMOTE_TLS_CA_FILE=/etc/opa/certs/ca.crt
export OPA_REMOTE_TLS_CERT_FILE=/etc/opa/certs/client.crt
export OPA_REMOTE_TLS_KEY_FILE=/etc/opa/certs/client.key

# Bearer Token 认证
export OPA_REMOTE_AUTH_TYPE=bearer
export OPA_REMOTE_AUTH_TOKEN=${OPA_TOKEN}  # 从密钥管理系统获取

# 缓存配置
export OPA_CACHE_ENABLED=true
export OPA_CACHE_MAX_SIZE=10000
export OPA_CACHE_TTL=5m
export OPA_CACHE_ENABLE_METRICS=true

./your-app
```

### 示例 3: Kubernetes Sidecar 模式

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      containers:
      # 应用容器
      - name: app
        image: myapp:latest
        env:
        - name: OPA_MODE
          value: "sidecar"
        - name: OPA_REMOTE_URL
          value: "http://localhost:8181"
        - name: OPA_REMOTE_TIMEOUT
          value: "10s"
        - name: OPA_CACHE_ENABLED
          value: "true"
        - name: OPA_CACHE_MAX_SIZE
          value: "1000"
        - name: OPA_DEFAULT_DECISION_PATH
          value: "authz/allow"
      
      # OPA Sidecar 容器
      - name: opa
        image: openpolicyagent/opa:latest
        args:
        - "run"
        - "--server"
        - "--addr=0.0.0.0:8181"
        - "/policies"
        volumeMounts:
        - name: policies
          mountPath: /policies
      
      volumes:
      - name: policies
        configMap:
          name: opa-policies
```

### 示例 4: Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    image: myapp:latest
    environment:
      - OPA_MODE=remote
      - OPA_REMOTE_URL=http://opa:8181
      - OPA_REMOTE_TIMEOUT=30s
      - OPA_CACHE_ENABLED=true
      - OPA_CACHE_MAX_SIZE=5000
      - OPA_DEFAULT_DECISION_PATH=rbac/allow
    depends_on:
      - opa
  
  opa:
    image: openpolicyagent/opa:latest
    command:
      - "run"
      - "--server"
      - "--addr=0.0.0.0:8181"
      - "/policies"
    volumes:
      - ./policies:/policies
    ports:
      - "8181:8181"
```

## 使用 Viper 的环境变量映射

当使用 `pkg/config.Manager` 时，环境变量自动映射到配置结构体：

```yaml
# swit.yaml
opa:
  mode: embedded
  embedded:
    policy_dir: ./policies
  cache:
    enabled: true
    max_size: 10000
```

环境变量覆盖（使用 `SWIT_` 前缀）：

```bash
# 覆盖 mode
export SWIT_OPA_MODE=remote

# 覆盖 remote URL
export SWIT_OPA_REMOTE_URL=http://opa:8181

# 覆盖缓存配置
export SWIT_OPA_CACHE_MAX_SIZE=5000
```

Viper 会自动将 `SWIT_OPA_CACHE_MAX_SIZE` 映射到 `opa.cache.max_size`。

## 配置优先级

配置的优先级从高到低：

1. **环境变量** - 最高优先级
2. **覆盖文件** (`swit.override.yaml`)
3. **环境特定文件** (`swit.prod.yaml`)
4. **基础配置文件** (`swit.yaml`)
5. **默认值** (`SetDefaults()`)

## 注意事项

1. **布尔值**：环境变量中的布尔值可以使用 `true/false`, `1/0`, `t/f`, `yes/no`
2. **时间段**：使用 Go duration 格式，如 `30s`, `5m`, `1h`
3. **整数**：直接使用数字字符串，如 `10000`
4. **密钥安全**：
   - 生产环境中应使用密钥管理系统（如 Vault、AWS Secrets Manager）
   - 避免在代码或配置文件中硬编码密钥
   - 使用 `_FILE` 后缀从文件读取密钥（如果 config.Manager 支持）

## 验证配置

完整的配置加载和验证流程：

```go
// 加载配置
config := &opa.Config{}
config.LoadFromEnv()
config.SetDefaults()

// 验证配置
if err := config.Validate(); err != nil {
    log.Fatalf("Invalid OPA configuration: %v", err)
}

// 打印配置（用于调试，注意不要打印敏感信息）
log.Printf("OPA Mode: %s", config.Mode)
if config.RemoteConfig != nil {
    log.Printf("OPA Remote URL: %s", config.RemoteConfig.URL)
}
if config.CacheConfig != nil && config.CacheConfig.Enabled {
    log.Printf("OPA Cache: enabled (max_size=%d, ttl=%s)", 
        config.CacheConfig.MaxSize, config.CacheConfig.TTL)
}
```

## 相关文档

- [OPA 配置参考](../../pkg/security/opa/config.go)
- [OPA 客户端使用指南](../../pkg/security/opa/README.md)
- [配置管理器文档](../../pkg/config/manager.go)

