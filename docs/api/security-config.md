# 安全配置参考

本文档提供 Swit 框架安全功能的完整配置参考，包括 OAuth2/OIDC、OPA、TLS、密钥管理和审计等。

## 目录

- [概述](#概述)
- [OAuth2/OIDC 配置](#oauth2oidc-配置)
- [OPA 策略配置](#opa-策略配置)
- [TLS/mTLS 配置](#tlsmtls-配置)
- [JWT 配置](#jwt-配置)
- [密钥管理配置](#密钥管理配置)
- [审计日志配置](#审计日志配置)
- [安全指标配置](#安全指标配置)
- [完整配置示例](#完整配置示例)
- [环境变量](#环境变量)
- [最佳实践](#最佳实践)

## 概述

Swit 框架提供了多层次的安全配置选项，支持：

- **身份认证** - OAuth2/OIDC、JWT、API 密钥
- **授权** - OPA 策略引擎（RBAC/ABAC）
- **传输安全** - TLS/mTLS 加密
- **密钥管理** - 环境变量、文件、Vault 集成
- **审计** - 安全事件日志和追踪

### 配置方式

1. **YAML 配置文件** - 推荐用于生产环境
2. **环境变量** - 用于容器化部署和敏感信息
3. **代码配置** - 用于动态配置和测试

---

## OAuth2/OIDC 配置

### 基本配置

```yaml
# swit.yaml 或 switauth.yaml
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-app
  client_secret: ${OAUTH2_CLIENT_SECRET}  # 从环境变量读取
  issuer_url: https://auth.example.com/realms/myrealm
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
  http_timeout: 30s
```

### 完整配置选项

```yaml
oauth2:
  # 基本配置
  enabled: true                    # 启用 OAuth2 认证
  provider: keycloak               # 提供商: keycloak, auth0, google, microsoft, okta, custom
  client_id: my-app                # 客户端 ID
  client_secret: secret            # 客户端密钥（建议使用环境变量）
  redirect_url: http://localhost:8080/callback  # 重定向 URL
  
  # OIDC 发现
  issuer_url: https://auth.example.com/realms/myrealm  # 颁发者 URL
  use_discovery: true              # 启用 OIDC 自动发现
  
  # 手动配置端点（use_discovery=false 时）
  auth_url: https://auth.example.com/oauth2/auth       # 授权端点
  token_url: https://auth.example.com/oauth2/token     # 令牌端点
  user_info_url: https://auth.example.com/oauth2/userinfo  # 用户信息端点
  jwks_url: https://auth.example.com/oauth2/jwks       # JWKS 端点
  
  # 作用域
  scopes:
    - openid
    - profile
    - email
    - offline_access           # 用于刷新令牌
  
  # 超时和重试
  http_timeout: 30s              # HTTP 请求超时
  
  # 测试选项（生产环境中不要使用）
  skip_issuer_verification: false  # 跳过颁发者验证
  skip_expiry_check: false         # 跳过过期检查
  
  # JWT 配置
  jwt:
    signing_method: RS256          # 签名方法: RS256, RS384, RS512, HS256, ES256
    clock_skew: 5m                 # 时钟偏差容忍度
    skip_claims_validation: false   # 跳过声明验证（测试用）
    audience: my-api               # 期望的 audience 声明
    required_claims:               # 必需的声明
      - sub
      - email
  
  # 令牌缓存
  cache:
    enabled: true                  # 启用令牌缓存
    max_size: 1000                 # 最大缓存条目数
    ttl: 10m                       # 缓存过期时间
    cleanup_interval: 5m           # 清理间隔
  
  # TLS 配置
  tls:
    enabled: true                  # 启用 TLS
    cert_file: /path/to/client-cert.pem  # 客户端证书
    key_file: /path/to/client-key.pem    # 客户端私钥
    ca_file: /path/to/ca-cert.pem        # CA 证书
    insecure_skip_verify: false    # 跳过证书验证（不推荐）
```

### 提供商特定配置

#### Keycloak

```yaml
oauth2:
  provider: keycloak
  issuer_url: https://auth.example.com/realms/myrealm
  use_discovery: true
  # Keycloak 会自动设置 openid 作用域
```

#### Auth0

```yaml
oauth2:
  provider: auth0
  issuer_url: https://your-domain.auth0.com/
  use_discovery: true
  # Auth0 需要指定 audience
  jwt:
    audience: https://your-api.example.com
```

#### Google

```yaml
oauth2:
  provider: google
  issuer_url: https://accounts.google.com
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
    - https://www.googleapis.com/auth/userinfo.profile
```

#### Microsoft/Azure AD

```yaml
oauth2:
  provider: microsoft
  issuer_url: https://login.microsoftonline.com/{tenant}/v2.0
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
    - offline_access
```

---

## OPA 策略配置

### 嵌入式模式

```yaml
opa:
  mode: embedded                   # 运行模式: embedded, remote, sidecar
  default_decision_path: authz/allow  # 默认决策路径
  log_level: info                  # 日志级别: debug, info, warn, error
  
  # 嵌入式模式配置
  embedded:
    policy_dir: ./pkg/security/opa/policies  # 策略目录
    data_dir: ./data                         # 数据目录（可选）
    enable_logging: true                     # 启用 OPA 内部日志
    enable_decision_logs: false              # 启用决策日志
  
  # 决策缓存
  cache:
    enabled: true                  # 启用缓存
    max_size: 10000                # 最大缓存条目数
    ttl: 5m                        # 缓存过期时间
    enable_metrics: true           # 启用缓存指标
```

### 远程模式

```yaml
opa:
  mode: remote                     # 远程模式
  default_decision_path: authz/allow
  log_level: info
  
  # 远程 OPA 服务器配置
  remote:
    url: http://opa-server:8181    # OPA 服务器 URL
    timeout: 30s                   # 请求超时
    max_retries: 3                 # 最大重试次数
    auth_token: ${OPA_AUTH_TOKEN}  # 认证令牌（可选）
    
    # TLS 配置（可选）
    tls:
      enabled: true
      cert_file: /path/to/client-cert.pem
      key_file: /path/to/client-key.pem
      ca_file: /path/to/ca-cert.pem
      insecure_skip_verify: false
  
  # 决策缓存
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
    enable_metrics: true
```

### Sidecar 模式（Kubernetes）

```yaml
opa:
  mode: sidecar                    # Sidecar 模式
  default_decision_path: authz/allow
  
  remote:
    url: http://localhost:8181     # 本地 sidecar
    timeout: 10s                   # 更短的超时
    max_retries: 2
  
  cache:
    enabled: true
    max_size: 5000
    ttl: 3m
```

---

## TLS/mTLS 配置

### 服务器 TLS 配置

```yaml
server:
  name: my-service
  
  # HTTP TLS 配置
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /path/to/server-cert.pem
      key_file: /path/to/server-key.pem
      # 客户端证书验证（mTLS）
      client_auth: require_and_verify
      client_ca_file: /path/to/client-ca.pem
      # TLS 版本
      min_version: "1.2"           # 1.0, 1.1, 1.2, 1.3
      max_version: "1.3"
      # 密码套件（可选）
      cipher_suites:
        - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
        - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  
  # gRPC TLS 配置
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /path/to/server-cert.pem
      key_file: /path/to/server-key.pem
      client_auth: require_and_verify
      client_ca_file: /path/to/client-ca.pem
```

### 客户端 TLS 配置

```yaml
# OAuth2 客户端 TLS
oauth2:
  tls:
    enabled: true
    cert_file: /path/to/client-cert.pem
    key_file: /path/to/client-key.pem
    ca_file: /path/to/ca-cert.pem

# OPA 客户端 TLS
opa:
  remote:
    tls:
      enabled: true
      cert_file: /path/to/client-cert.pem
      key_file: /path/to/client-key.pem
      ca_file: /path/to/ca-cert.pem
```

### TLS 选项说明

**client_auth 选项：**
- `no_client_cert` - 不要求客户端证书
- `request_client_cert` - 请求但不验证客户端证书
- `require_any_client_cert` - 要求客户端证书但不验证
- `verify_client_cert_if_given` - 如果提供则验证客户端证书
- `require_and_verify` - 要求并验证客户端证书（推荐用于 mTLS）

**TLS 版本：**
- `1.0` - TLS 1.0（不推荐）
- `1.1` - TLS 1.1（不推荐）
- `1.2` - TLS 1.2（推荐最小版本）
- `1.3` - TLS 1.3（推荐）

---

## JWT 配置

### JWT 验证器配置

```yaml
jwt:
  # 签名验证
  signing_method: RS256            # RS256, RS384, RS512, HS256, ES256
  
  # JWKS 配置（用于 RS256 等公钥算法）
  jwks:
    url: https://auth.example.com/.well-known/jwks.json
    cache_enabled: true
    cache_ttl: 1h
    refresh_interval: 15m
  
  # HMAC 密钥（用于 HS256 等对称算法）
  hmac_secret: ${JWT_HMAC_SECRET}  # 从环境变量读取
  
  # 声明验证
  issuer: https://auth.example.com    # 期望的 issuer
  audience: my-api                    # 期望的 audience
  clock_skew: 5m                      # 时钟偏差容忍度
  
  # 必需的声明
  required_claims:
    - sub
    - exp
    - iat
  
  # 黑名单配置
  blacklist:
    enabled: true
    redis:
      addr: localhost:6379
      password: ${REDIS_PASSWORD}
      db: 0
```

### JWT 提取器配置

```yaml
jwt:
  # 令牌提取位置
  extract_from:
    - header: Authorization          # Authorization 头
      prefix: "Bearer "              # 前缀
    - query: access_token            # 查询参数
    - cookie: access_token           # Cookie
  
  # 跳过路径（不需要 JWT 验证）
  skip_paths:
    - /api/v1/public/*
    - /health
    - /metrics
```

---

## 密钥管理配置

### 环境变量提供者

```yaml
secrets:
  provider: env                      # 环境变量提供者
  env:
    prefix: SWIT_SECRET_             # 密钥前缀
```

使用：
```bash
export SWIT_SECRET_DATABASE_PASSWORD=mypassword
export SWIT_SECRET_API_KEY=myapikey
```

### 文件提供者

```yaml
secrets:
  provider: file                     # 文件提供者
  file:
    path: /etc/secrets               # 密钥目录
    # 密钥文件: /etc/secrets/database_password, /etc/secrets/api_key
```

### Vault 提供者

```yaml
secrets:
  provider: vault                    # HashiCorp Vault 提供者
  vault:
    addr: https://vault.example.com:8200
    token: ${VAULT_TOKEN}            # Vault 令牌
    # 或使用 AppRole 认证
    auth_method: approle
    role_id: ${VAULT_ROLE_ID}
    secret_id: ${VAULT_SECRET_ID}
    
    # KV 存储配置
    kv_version: 2                    # KV v1 或 v2
    mount_path: secret               # 挂载路径
    path: swit/production            # 密钥路径
    
    # TLS 配置
    tls:
      enabled: true
      ca_cert: /path/to/vault-ca.pem
      client_cert: /path/to/client-cert.pem
      client_key: /path/to/client-key.pem
```

---

## 审计日志配置

### 审计事件配置

```yaml
security:
  audit:
    enabled: true                    # 启用审计
    log_level: info                  # 日志级别
    
    # 审计事件类型
    events:
      - authentication_success
      - authentication_failure
      - authorization_denied
      - token_issued
      - token_revoked
      - policy_evaluated
      - sensitive_data_accessed
    
    # 输出配置
    outputs:
      # 文件输出
      - type: file
        path: /var/log/swit/audit.log
        format: json                 # json 或 text
        max_size: 100                # MB
        max_backups: 10
        max_age: 30                  # 天
        
      # 系统日志输出
      - type: syslog
        network: udp
        addr: localhost:514
        tag: swit-audit
        
      # 外部服务
      - type: webhook
        url: https://audit-collector.example.com/events
        headers:
          Authorization: Bearer ${AUDIT_WEBHOOK_TOKEN}
        batch_size: 100
        flush_interval: 10s
```

### 敏感数据过滤

```yaml
security:
  audit:
    # 敏感字段过滤
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
      - refresh_token
      - access_token
    
    # 过滤策略
    redact_policy: mask              # mask, remove, hash
    mask_char: "*"
    mask_length: 8                   # 显示前8个字符
```

---

## 安全指标配置

### Prometheus 指标

```yaml
security:
  metrics:
    enabled: true                    # 启用安全指标
    namespace: swit                  # 指标命名空间
    subsystem: security              # 指标子系统
    
    # 指标类型
    collect:
      - authentication_total         # 认证总数
      - authentication_errors        # 认证错误数
      - authorization_total          # 授权总数
      - authorization_denied         # 授权拒绝数
      - token_validation_duration    # 令牌验证时长
      - policy_evaluation_duration   # 策略评估时长
      - jwt_cache_hits               # JWT 缓存命中数
      - jwt_cache_misses             # JWT 缓存未命中数
    
    # 标签
    labels:
      service: my-service
      environment: production
      version: v1.0.0
```

---

## 完整配置示例

### 生产环境配置

```yaml
# swit.yaml - 生产环境完整配置
server:
  name: my-service
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/swit/certs/server-cert.pem
      key_file: /etc/swit/certs/server-key.pem
      client_auth: require_and_verify
      client_ca_file: /etc/swit/certs/client-ca.pem
      min_version: "1.2"
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /etc/swit/certs/server-cert.pem
      key_file: /etc/swit/certs/server-key.pem

# OAuth2/OIDC 配置
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-service
  client_secret: ${OAUTH2_CLIENT_SECRET}
  issuer_url: https://auth.example.com/realms/production
  redirect_url: https://my-service.example.com/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
  http_timeout: 30s
  jwt:
    signing_method: RS256
    clock_skew: 5m
    audience: my-service-api
    required_claims:
      - sub
      - email
  cache:
    enabled: true
    max_size: 5000
    ttl: 15m
  tls:
    enabled: true
    ca_file: /etc/swit/certs/ca-cert.pem

# OPA 策略配置
opa:
  mode: remote
  default_decision_path: authz/allow
  log_level: info
  remote:
    url: https://opa.example.com
    timeout: 30s
    max_retries: 3
    auth_token: ${OPA_AUTH_TOKEN}
    tls:
      enabled: true
      ca_file: /etc/swit/certs/ca-cert.pem
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
    enable_metrics: true

# JWT 配置
jwt:
  signing_method: RS256
  jwks:
    url: https://auth.example.com/.well-known/jwks.json
    cache_enabled: true
    cache_ttl: 1h
    refresh_interval: 15m
  issuer: https://auth.example.com
  audience: my-service-api
  clock_skew: 5m
  required_claims:
    - sub
    - exp
  blacklist:
    enabled: true
    redis:
      addr: redis:6379
      password: ${REDIS_PASSWORD}
      db: 0

# 密钥管理
secrets:
  provider: vault
  vault:
    addr: https://vault.example.com:8200
    auth_method: approle
    role_id: ${VAULT_ROLE_ID}
    secret_id: ${VAULT_SECRET_ID}
    kv_version: 2
    mount_path: secret
    path: swit/production
    tls:
      enabled: true
      ca_cert: /etc/swit/certs/vault-ca.pem

# 审计日志
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
      - policy_evaluated
    outputs:
      - type: file
        path: /var/log/swit/audit.log
        format: json
        max_size: 100
        max_backups: 10
        max_age: 30
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
    redact_policy: mask
  
  # 安全指标
  metrics:
    enabled: true
    namespace: swit
    subsystem: security
    collect:
      - authentication_total
      - authorization_total
      - token_validation_duration
      - policy_evaluation_duration
    labels:
      service: my-service
      environment: production
```

### 开发环境配置

```yaml
# swit-dev.yaml - 开发环境配置
server:
  name: my-service-dev
  http:
    addr: :8080
    tls:
      enabled: false               # 开发环境禁用 TLS
  grpc:
    addr: :9090
    tls:
      enabled: false

# OAuth2/OIDC 配置
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-service-dev
  client_secret: dev-secret        # 开发环境可以明文
  issuer_url: http://localhost:8180/realms/dev
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  http_timeout: 30s
  skip_issuer_verification: true   # 开发环境可以跳过
  cache:
    enabled: true
    max_size: 100

# OPA 策略配置
opa:
  mode: embedded                   # 开发环境使用嵌入式
  default_decision_path: authz/allow
  log_level: debug
  embedded:
    policy_dir: ./pkg/security/opa/policies
    enable_logging: true
  cache:
    enabled: false                 # 开发环境禁用缓存便于调试

# 审计日志
security:
  audit:
    enabled: true
    log_level: debug
    events:
      - authentication_success
      - authentication_failure
      - authorization_denied
    outputs:
      - type: file
        path: ./logs/audit.log
        format: json
    redact_policy: mask
```

---

## 环境变量

### OAuth2/OIDC

```bash
# 基本配置
OAUTH2_ENABLED=true
OAUTH2_PROVIDER=keycloak
OAUTH2_CLIENT_ID=my-app
OAUTH2_CLIENT_SECRET=secret
OAUTH2_ISSUER_URL=https://auth.example.com/realms/myrealm
OAUTH2_REDIRECT_URL=http://localhost:8080/callback
OAUTH2_SCOPES=openid,profile,email
OAUTH2_USE_DISCOVERY=true

# TLS 配置
OAUTH2_TLS_ENABLED=true
OAUTH2_TLS_CA_FILE=/path/to/ca.pem
```

### OPA

```bash
# 基本配置
OPA_MODE=remote
OPA_REMOTE_URL=http://opa-server:8181
OPA_DEFAULT_DECISION_PATH=authz/allow

# 缓存配置
OPA_CACHE_ENABLED=true
OPA_CACHE_MAX_SIZE=10000
OPA_CACHE_TTL=5m

# 认证
OPA_AUTH_TOKEN=secret-token
```

### JWT

```bash
# JWKS 配置
JWT_JWKS_URL=https://auth.example.com/.well-known/jwks.json
JWT_ISSUER=https://auth.example.com
JWT_AUDIENCE=my-api

# HMAC 密钥
JWT_HMAC_SECRET=your-secret-key

# 黑名单 Redis
JWT_BLACKLIST_REDIS_ADDR=localhost:6379
JWT_BLACKLIST_REDIS_PASSWORD=redis-password
```

### 密钥管理

```bash
# Vault 配置
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=vault-token
VAULT_ROLE_ID=role-id
VAULT_SECRET_ID=secret-id

# 密钥前缀（环境变量提供者）
SWIT_SECRET_DATABASE_PASSWORD=dbpass
SWIT_SECRET_API_KEY=apikey
```

---

## 最佳实践

### 1. 使用环境变量存储敏感信息

❌ **避免：**
```yaml
oauth2:
  client_secret: hardcoded-secret    # 不要硬编码
```

✅ **推荐：**
```yaml
oauth2:
  client_secret: ${OAUTH2_CLIENT_SECRET}
```

### 2. 启用 TLS

生产环境始终启用 TLS：

```yaml
server:
  http:
    tls:
      enabled: true
      min_version: "1.2"              # 最低 TLS 1.2
```

### 3. 使用 mTLS 进行服务间通信

```yaml
server:
  grpc:
    tls:
      enabled: true
      client_auth: require_and_verify  # 要求客户端证书
```

### 4. 配置审计日志

记录所有安全相关事件：

```yaml
security:
  audit:
    enabled: true
    events:
      - authentication_failure
      - authorization_denied
```

### 5. 启用决策缓存

提升性能：

```yaml
opa:
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
```

### 6. 定期轮换密钥

- 定期轮换 OAuth2 客户端密钥
- 定期更新 TLS 证书
- 使用自动化工具管理密钥生命周期

### 7. 监控安全指标

```yaml
security:
  metrics:
    enabled: true
    collect:
      - authentication_errors
      - authorization_denied
```

### 8. 使用最小权限原则

OAuth2 只请求必需的作用域：

```yaml
oauth2:
  scopes:
    - openid
    - profile    # 只请求需要的
```

### 9. 分离开发和生产配置

使用不同的配置文件：
- `swit.yaml` - 生产
- `swit-dev.yaml` - 开发
- `swit-test.yaml` - 测试

### 10. 验证配置

启动时验证配置：

```bash
# 使用 --validate-config 标志
./my-service --validate-config --config swit.yaml
```

---

## 相关文档

- [OAuth2/OIDC API 参考](./security-oauth2.md)
- [OPA 策略 API 参考](./security-opa.md)
- [服务器配置参考](../configuration-reference.md)
- [安全最佳实践指南](../../SECURITY.md)

## 许可证

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.







