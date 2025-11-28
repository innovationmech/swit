# Swit 安全功能迁移指南

本文档提供从旧版本 Swit 升级到 v0.9.0 并启用安全功能的完整迁移指南。

## 目录

- [概述](#概述)
- [前置条件](#前置条件)
- [升级步骤](#升级步骤)
- [OAuth2/OIDC 集成](#oauth2oidc-集成)
- [OPA 策略引擎集成](#opa-策略引擎集成)
- [密钥管理集成](#密钥管理集成)
- [TLS/mTLS 配置](#tlsmtls-配置)
- [安全扫描集成](#安全扫描集成)
- [审计日志配置](#审计日志配置)
- [安全指标配置](#安全指标配置)
- [已知问题](#已知问题)
- [故障排查](#故障排查)

---

## 概述

Swit v0.9.0 引入了完整的企业级安全功能，包括：

- **OAuth2/OIDC 认证** - 支持 Keycloak、Auth0 等主流身份提供商
- **OPA 策略引擎** - 基于策略的细粒度访问控制（RBAC/ABAC）
- **密钥管理** - 支持环境变量、文件、HashiCorp Vault
- **TLS/mTLS** - 传输层安全和双向认证
- **安全扫描** - gosec、Trivy、govulncheck 集成
- **审计日志** - 完整的安全事件追踪
- **安全指标** - Prometheus 安全指标收集

### 兼容性说明

- ✅ **完全向后兼容** - 现有代码无需修改即可升级
- ✅ **可选功能** - 所有安全功能都是可选的，按需启用
- ✅ **渐进式采用** - 可以逐步启用各个安全组件

---

## 前置条件

### Go 版本

- **最低版本**: Go 1.23.12+
- **推荐版本**: Go 1.23.12 或更高

### 依赖更新

升级前请确保更新依赖：

```bash
go get -u github.com/innovationmech/swit@v0.9.0
go mod tidy
```

### 新增依赖

v0.9.0 引入以下新依赖（自动安装）：

| 依赖包 | 版本 | 用途 |
|--------|------|------|
| `github.com/open-policy-agent/opa` | v1.4.2+ | OPA 策略引擎 |
| `github.com/coreos/go-oidc/v3` | v3.11.0+ | OIDC 支持 |
| `golang.org/x/oauth2` | v0.26.0+ | OAuth2 客户端 |
| `github.com/hashicorp/vault/api` | v1.16.0+ | Vault 集成 |
| `github.com/golang-jwt/jwt/v5` | v5.2.1+ | JWT 处理 |

---

## 升级步骤

### 步骤 1: 更新依赖

```bash
# 更新到 v0.9.0
go get -u github.com/innovationmech/swit@v0.9.0

# 清理依赖
go mod tidy

# 验证依赖
go mod verify
```

### 步骤 2: 运行安全扫描（可选但推荐）

```bash
# 运行完整安全扫描
make security

# 或单独运行各扫描工具
govulncheck ./...
```

### 步骤 3: 选择并配置安全功能

根据您的需求选择启用以下安全功能：

- [OAuth2/OIDC 认证](#oauth2oidc-集成)
- [OPA 策略引擎](#opa-策略引擎集成)
- [密钥管理](#密钥管理集成)
- [TLS/mTLS](#tlsmtls-配置)

### 步骤 4: 测试和验证

```bash
# 运行测试
make test

# 运行集成测试
make test-advanced TYPE=integration
```

---

## OAuth2/OIDC 集成

### 基本配置

在 `swit.yaml` 中添加 OAuth2 配置：

```yaml
oauth2:
  enabled: true
  provider: keycloak  # 或 auth0, custom
  client_id: my-service
  client_secret: ${OAUTH2_CLIENT_SECRET}  # 从环境变量读取
  issuer_url: https://auth.example.com/realms/production
  use_discovery: true
  
  scopes:
    - openid
    - profile
    - email
  
  jwt:
    signing_method: RS256
    clock_skew: 5m
    required_claims:
      - sub
      - email
  
  cache:
    enabled: true
    max_size: 5000
    ttl: 15m
  
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-bundle.crt
```

### 代码集成

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/gin-gonic/gin"
)

func main() {
    // 加载配置
    config := &oauth2.Config{
        Enabled:      true,
        Provider:     "keycloak",
        ClientID:     "my-service",
        ClientSecret: os.Getenv("OAUTH2_CLIENT_SECRET"),
        IssuerURL:    "https://auth.example.com/realms/production",
        UseDiscovery: true,
        Scopes:       []string{"openid", "profile", "email"},
    }
    config.SetDefaults()
    
    // 创建 OAuth2 客户端
    client, err := oauth2.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 创建 Gin 路由
    router := gin.Default()
    
    // 应用认证中间件
    router.Use(oauth2AuthMiddleware(client))
    
    // 受保护的路由
    router.GET("/api/v1/protected", protectedHandler)
    
    router.Run(":8080")
}

func oauth2AuthMiddleware(client *oauth2.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 Authorization 头获取令牌
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing authorization header"})
            return
        }
        
        // 验证令牌
        claims, err := client.ValidateToken(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        
        // 将用户信息存入上下文
        c.Set("user_id", claims.Subject)
        c.Set("email", claims.Email)
        c.Next()
    }
}
```

### 环境变量覆盖

OAuth2 配置支持环境变量覆盖：

| 环境变量 | 配置项 |
|----------|--------|
| `OAUTH2_ENABLED` | `oauth2.enabled` |
| `OAUTH2_PROVIDER` | `oauth2.provider` |
| `OAUTH2_CLIENT_ID` | `oauth2.client_id` |
| `OAUTH2_CLIENT_SECRET` | `oauth2.client_secret` |
| `OAUTH2_ISSUER_URL` | `oauth2.issuer_url` |
| `OAUTH2_USE_DISCOVERY` | `oauth2.use_discovery` |
| `OAUTH2_JWT_SIGNING_METHOD` | `oauth2.jwt.signing_method` |

---

## OPA 策略引擎集成

### 嵌入式模式配置

```yaml
opa:
  mode: embedded
  embedded:
    policy_dir: ./policies
    enable_logging: true
    enable_decision_logs: true
  default_decision_path: authz/allow
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
```

### 远程模式配置

```yaml
opa:
  mode: remote
  remote:
    url: http://opa-server:8181
    timeout: 30s
    max_retries: 3
    tls:
      enabled: true
      ca_file: /etc/ssl/certs/ca.pem
  default_decision_path: authz/allow
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
```

### 创建 RBAC 策略

创建 `policies/rbac.rego`：

```rego
package authz

import rego.v1

# 默认拒绝
default allow := false

# 超级管理员拥有所有权限
allow if {
    "admin" in input.subject.roles
}

# 角色权限映射
role_permissions := {
    "editor": ["read", "write"],
    "viewer": ["read"],
    "manager": ["read", "write", "delete", "approve"],
}

# 检查角色权限
allow if {
    some role in input.subject.roles
    permissions := role_permissions[role]
    input.action in permissions
}

# 资源所有者可以访问自己的资源
allow if {
    input.resource.owner == input.subject.user
}
```

### 代码集成

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    
    // 创建 OPA 客户端
    config := &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./policies",
        },
        DefaultDecisionPath: "authz/allow",
        CacheConfig: &opa.CacheConfig{
            Enabled: true,
            MaxSize: 10000,
            TTL:     5 * time.Minute,
        },
    }
    
    client, err := opa.NewClient(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close(ctx)
    
    // 创建 Gin 路由
    router := gin.Default()
    
    // 应用授权中间件
    router.Use(opaAuthzMiddleware(client))
    
    router.Run(":8080")
}

func opaAuthzMiddleware(client opa.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 构建 OPA 输入
        input := map[string]interface{}{
            "subject": map[string]interface{}{
                "user":  c.GetString("user_id"),
                "roles": c.GetStringSlice("roles"),
            },
            "action":   c.Request.Method,
            "resource": c.Request.URL.Path,
        }
        
        // 评估策略
        result, err := client.Evaluate(c.Request.Context(), input)
        if err != nil {
            c.AbortWithStatusJSON(500, gin.H{"error": "policy evaluation failed"})
            return
        }
        
        if !result.Allow {
            c.AbortWithStatusJSON(403, gin.H{"error": "access denied"})
            return
        }
        
        c.Next()
    }
}
```

### 环境变量覆盖

| 环境变量 | 配置项 |
|----------|--------|
| `OPA_MODE` | `opa.mode` |
| `OPA_DEFAULT_DECISION_PATH` | `opa.default_decision_path` |
| `OPA_EMBEDDED_POLICY_DIR` | `opa.embedded.policy_dir` |
| `OPA_REMOTE_URL` | `opa.remote.url` |
| `OPA_CACHE_ENABLED` | `opa.cache.enabled` |
| `OPA_CACHE_MAX_SIZE` | `opa.cache.max_size` |
| `OPA_CACHE_TTL` | `opa.cache.ttl` |

---

## 密钥管理集成

### 多 Provider 配置

```yaml
secrets:
  providers:
    # 环境变量 Provider（优先级最高）
    - type: env
      enabled: true
      env:
        prefix: APP_SECRET_
    
    # 文件 Provider
    - type: file
      enabled: true
      file:
        path: /etc/secrets
        format: yaml
    
    # HashiCorp Vault Provider
    - type: vault
      enabled: true
      vault:
        address: https://vault.example.com:8200
        auth_method: approle
        role_id: ${VAULT_ROLE_ID}
        secret_id: ${VAULT_SECRET_ID}
        mount_path: secret
        path: swit/production
  
  cache:
    enabled: true
    ttl: 5m
    max_size: 1000
  
  refresh:
    enabled: true
    interval: 10m
    keys:
      - database_password
      - api_key
```

### 代码集成

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/security/secrets"
)

func main() {
    // 创建密钥管理器
    config := &secrets.ManagerConfig{
        Providers: []*secrets.ProviderConfig{
            {
                Type:    secrets.ProviderTypeEnv,
                Enabled: true,
                Env: &secrets.EnvProviderConfig{
                    Prefix: "APP_SECRET_",
                },
            },
            {
                Type:    secrets.ProviderTypeVault,
                Enabled: true,
                Vault: &secrets.VaultProviderConfig{
                    Address:    "https://vault.example.com:8200",
                    AuthMethod: "approle",
                    RoleID:     os.Getenv("VAULT_ROLE_ID"),
                    SecretID:   os.Getenv("VAULT_SECRET_ID"),
                },
            },
        },
        Cache: &secrets.CacheConfig{
            Enabled: true,
            TTL:     5 * time.Minute,
        },
    }
    
    manager, err := secrets.NewManager(config)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()
    
    // 获取密钥
    ctx := context.Background()
    dbPassword, err := manager.GetSecretValue(ctx, "database_password")
    if err != nil {
        log.Fatal(err)
    }
    
    // 使用密钥
    connectDatabase(dbPassword)
}
```

---

## TLS/mTLS 配置

### 服务端 TLS 配置

```yaml
server:
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "TLS1.2"
      max_version: "TLS1.3"
  
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "TLS1.2"
```

### mTLS 配置（双向认证）

```yaml
server:
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      # 要求并验证客户端证书
      client_auth: require_and_verify
      client_ca_file: /etc/ssl/certs/client-ca.pem
      min_version: "TLS1.2"
```

### 证书生成

```bash
# 生成 CA 证书
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
  -subj "/CN=Swit CA"

# 生成服务器证书
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=myservice.example.com"
openssl x509 -req -days 365 -in server.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -set_serial 01 \
  -out server-cert.pem

# 生成客户端证书（用于 mTLS）
openssl genrsa -out client-key.pem 4096
openssl req -new -key client-key.pem -out client.csr \
  -subj "/CN=client"
openssl x509 -req -days 365 -in client.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -set_serial 02 \
  -out client-cert.pem
```

---

## 安全扫描集成

### Makefile 集成

项目已集成安全扫描命令：

```bash
# 运行完整安全扫描
make security

# 单独运行 gosec
make security-gosec

# 单独运行 Trivy
make security-trivy

# 单独运行 govulncheck
make security-govulncheck
```

### CI/CD 集成

在 GitHub Actions 中添加安全扫描：

```yaml
# .github/workflows/security.yml
name: Security Scan

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: ./...
      
      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
      
      - name: Run Trivy
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
```

---

## 审计日志配置

### 配置示例

```yaml
security:
  audit:
    enabled: true
    log_level: info
    
    events:
      - authentication_success
      - authentication_failure
      - authorization_denied
      - token_issued
      - token_revoked
      - sensitive_data_accessed
    
    outputs:
      - type: file
        path: /var/log/swit/audit.log
        format: json
        max_size: 100  # MB
        max_backups: 10
        max_age: 30    # 天
    
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
    
    redact_policy: mask
```

### 代码集成

```go
import "github.com/innovationmech/swit/pkg/security/audit"

// 创建审计日志记录器
auditor := audit.NewLogger(auditConfig)

// 记录认证事件
auditor.Log(audit.Event{
    Type:      "authentication_success",
    Actor:     userID,
    Action:    "login",
    Resource:  "system",
    Result:    "success",
    Timestamp: time.Now(),
    Metadata: map[string]interface{}{
        "ip_address": clientIP,
        "user_agent": userAgent,
    },
})
```

---

## 安全指标配置

### 配置示例

```yaml
security:
  metrics:
    enabled: true
    namespace: swit
    subsystem: security
    
    collect:
      - authentication_total
      - authentication_errors
      - authorization_total
      - authorization_denied
      - token_validation_duration
      - policy_evaluation_duration
```

### Prometheus 查询示例

```promql
# 认证失败率
rate(swit_security_authentication_errors_total[5m])

# 授权拒绝率
rate(swit_security_authorization_denied_total[5m])

# 令牌验证 P95 延迟
histogram_quantile(0.95, 
  rate(swit_security_token_validation_duration_bucket[5m]))

# 策略评估 P99 延迟
histogram_quantile(0.99, 
  rate(swit_security_policy_evaluation_duration_bucket[5m]))
```

### 告警规则示例

```yaml
groups:
  - name: security
    rules:
      - alert: HighAuthenticationFailureRate
        expr: rate(swit_security_authentication_errors_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate"
      
      - alert: FrequentAuthorizationDenials
        expr: rate(swit_security_authorization_denied_total[5m]) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Frequent authorization denials"
```

---

## 已知问题

### 1. JWKS 缓存刷新

**问题**: 在高并发场景下，JWKS 缓存刷新可能导致短暂的令牌验证延迟。

**解决方案**: 配置合适的缓存 TTL 和预刷新时间：

```yaml
oauth2:
  cache:
    enabled: true
    ttl: 1h
    refresh_before: 10m  # 提前 10 分钟刷新
```

### 2. OPA 策略热更新

**问题**: 嵌入式模式下策略文件更新不会自动生效。

**解决方案**: 使用 Bundle 配置或重启服务：

```yaml
opa:
  embedded:
    bundle:
      service_url: http://bundle-server:8080
      resource: policies/bundle.tar.gz
      polling_min_delay_seconds: 60
```

### 3. Vault 连接超时

**问题**: 网络不稳定时 Vault 连接可能超时。

**解决方案**: 配置重试和超时：

```yaml
secrets:
  providers:
    - type: vault
      vault:
        timeout: 30s
        max_retries: 3
        retry_wait: 1s
```

---

## 故障排查

### 认证失败

1. **检查令牌格式**
   ```bash
   # 解码 JWT 令牌
   echo $TOKEN | cut -d'.' -f2 | base64 -d | jq
   ```

2. **验证 OIDC 发现端点**
   ```bash
   curl https://auth.example.com/.well-known/openid-configuration
   ```

3. **检查时钟同步**
   ```bash
   # 确保服务器时钟与认证服务器同步
   ntpdate -q pool.ntp.org
   ```

### 授权失败

1. **启用 OPA 决策日志**
   ```yaml
   opa:
     embedded:
       enable_decision_logs: true
   ```

2. **测试策略**
   ```bash
   # 使用 OPA CLI 测试策略
   opa eval -i input.json -d policies/ "data.authz.allow"
   ```

### 密钥获取失败

1. **检查 Provider 状态**
   ```bash
   # 检查 Vault 状态
   vault status
   
   # 检查环境变量
   env | grep APP_SECRET_
   ```

2. **验证 Vault 权限**
   ```bash
   vault token lookup
   vault kv get secret/swit/production
   ```

### TLS 握手失败

1. **验证证书链**
   ```bash
   openssl verify -CAfile ca-cert.pem server-cert.pem
   ```

2. **检查证书有效期**
   ```bash
   openssl x509 -in server-cert.pem -noout -dates
   ```

3. **测试 TLS 连接**
   ```bash
   openssl s_client -connect localhost:8080 -CAfile ca-cert.pem
   ```

---

## 相关文档

- [安全最佳实践](./security-best-practices.md)
- [安全检查清单](./security-checklist.md)
- [配置参考](./configuration-reference.md)
- [OPA RBAC 指南](./opa-rbac-guide.md)
- [OPA ABAC 指南](./opa-abac-guide.md)
- [OAuth2 集成指南](./oauth2-integration-guide.md)

---

## 许可证

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.


