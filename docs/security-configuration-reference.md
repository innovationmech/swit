# Swit 安全配置参考 / Security Configuration Reference

本文档提供 Swit 框架安全功能的完整配置参考，包括所有配置项的详细说明、默认值、推荐值和配置示例。

This document provides a complete configuration reference for Swit framework security features, including detailed descriptions of all configuration items, default values, recommended values, and configuration examples.

## 目录 / Table of Contents

- [概述 / Overview](#概述--overview)
- [主配置 / Main Configuration](#主配置--main-configuration)
- [OAuth2/OIDC 配置 / OAuth2/OIDC Configuration](#oauth2oidc-配置--oauth2oidc-configuration)
- [OPA 策略引擎配置 / OPA Policy Engine Configuration](#opa-策略引擎配置--opa-policy-engine-configuration)
- [安全扫描器配置 / Security Scanner Configuration](#安全扫描器配置--security-scanner-configuration)
- [密钥管理配置 / Secrets Management Configuration](#密钥管理配置--secrets-management-configuration)
- [审计日志配置 / Audit Logging Configuration](#审计日志配置--audit-logging-configuration)
- [环境变量映射 / Environment Variable Mapping](#环境变量映射--environment-variable-mapping)
- [完整配置示例 / Complete Configuration Examples](#完整配置示例--complete-configuration-examples)

---

## 概述 / Overview

Swit 框架提供了全面的安全功能,包括:

The Swit framework provides comprehensive security features, including:

- **OAuth2/OIDC 认证 / OAuth2/OIDC Authentication** - 支持多种身份提供商 / Support for multiple identity providers
- **OPA 策略引擎 / OPA Policy Engine** - 基于策略的授权控制 / Policy-based authorization control
- **安全扫描 / Security Scanning** - 自动化安全漏洞扫描 / Automated security vulnerability scanning
- **密钥管理 / Secrets Management** - 集中式密钥管理 / Centralized secrets management
- **审计日志 / Audit Logging** - 完整的安全审计日志记录 / Complete security audit logging

所有安全配置都在 `security` 配置块下统一管理。

All security configurations are managed under the unified `security` configuration block.

---

## 主配置 / Main Configuration

### `security.enabled`

**描述 / Description**: 是否启用安全功能 / Whether to enable security features

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: 
- 开发环境 / Development: `false`
- 生产环境 / Production: `true`

**说明 / Notes**: 
- 设置为 `true` 时,将启用配置的所有安全组件
- 设置为 `false` 时,所有安全功能将被禁用,适用于开发和测试环境

When set to `true`, all configured security components will be enabled.
When set to `false`, all security features will be disabled, suitable for development and testing environments.

**示例 / Example**:

```yaml
security:
  enabled: true
```

---

## OAuth2/OIDC 配置 / OAuth2/OIDC Configuration

OAuth2/OIDC 提供了标准的身份认证和单点登录功能。

OAuth2/OIDC provides standard authentication and single sign-on capabilities.

### 基本配置 / Basic Configuration

#### `security.oauth2.enabled`

**描述 / Description**: 是否启用 OAuth2 认证 / Whether to enable OAuth2 authentication

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: `true` (当需要认证时 / when authentication is required)

#### `security.oauth2.provider`

**描述 / Description**: OAuth2 提供商类型 / OAuth2 provider type

**类型 / Type**: `string`

**默认值 / Default**: `"custom"`

**可选值 / Available Values**:
- `keycloak` - Keycloak (推荐 / recommended for self-hosted)
- `auth0` - Auth0
- `google` - Google
- `microsoft` - Microsoft Azure AD
- `okta` - Okta
- `custom` - 自定义提供商 / Custom provider

**推荐值 / Recommended**: `keycloak` (自托管 / self-hosted) 或 / or `auth0` (SaaS)

#### `security.oauth2.client_id`

**描述 / Description**: OAuth2 客户端 ID / OAuth2 client ID

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当 OAuth2 启用时 / when OAuth2 is enabled)

**环境变量 / Environment Variable**: `OAUTH2_CLIENT_ID`

**示例 / Example**: `"my-service-client"`

#### `security.oauth2.client_secret`

**描述 / Description**: OAuth2 客户端密钥 / OAuth2 client secret

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当 OAuth2 启用时 / when OAuth2 is enabled)

**环境变量 / Environment Variable**: `OAUTH2_CLIENT_SECRET`

**安全建议 / Security Recommendation**: 
- **必须**使用环境变量或密钥管理系统
- **永远不要**在配置文件中硬编码

**Must** use environment variables or secrets management system.
**Never** hardcode in configuration files.

**示例 / Example**: `"${OAUTH2_CLIENT_SECRET}"`

#### `security.oauth2.redirect_url`

**描述 / Description**: OAuth2 回调 URL / OAuth2 callback URL

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (授权码流程 / for authorization code flow)

**格式 / Format**: `https://your-domain.com/callback` 或 / or `http://localhost:8080/callback` (开发环境 / development)

**示例 / Example**: `"http://localhost:8080/callback"`

**说明 / Notes**:
- 必须与 OAuth2 提供商中配置的回调 URL 完全匹配
- 生产环境必须使用 HTTPS

Must match exactly with the callback URL configured in the OAuth2 provider.
Production environments must use HTTPS.

### OIDC 发现配置 / OIDC Discovery Configuration

#### `security.oauth2.issuer_url`

**描述 / Description**: OIDC 颁发者 URL / OIDC issuer URL

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当使用 OIDC 发现时 / when using OIDC discovery)

**示例 / Examples**:
- Keycloak: `"http://localhost:8081/realms/swit"`
- Auth0: `"https://your-tenant.auth0.com/"`
- Google: `"https://accounts.google.com"`
- Microsoft: `"https://login.microsoftonline.com/{tenant-id}/v2.0"`

#### `security.oauth2.use_discovery`

**描述 / Description**: 是否使用 OIDC 自动发现 / Whether to use OIDC auto-discovery

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: `true` (当提供商支持时 / when provider supports it)

**说明 / Notes**:
- 设置为 `true` 时,将自动从 `issuer_url` 发现端点配置
- 设置为 `false` 时,需要手动配置 `auth_url`, `token_url`, `jwks_url` 等端点

When set to `true`, endpoint configuration will be auto-discovered from `issuer_url`.
When set to `false`, you need to manually configure `auth_url`, `token_url`, `jwks_url`, etc.

### 手动端点配置 / Manual Endpoint Configuration

#### `security.oauth2.auth_url`

**描述 / Description**: OAuth2 授权端点 URL / OAuth2 authorization endpoint URL

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当 `use_discovery` 为 `false` 时 / when `use_discovery` is `false`)

**示例 / Example**: `"https://auth.example.com/oauth2/authorize"`

#### `security.oauth2.token_url`

**描述 / Description**: OAuth2 令牌端点 URL / OAuth2 token endpoint URL

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当 `use_discovery` 为 `false` 时 / when `use_discovery` is `false`)

**示例 / Example**: `"https://auth.example.com/oauth2/token"`

#### `security.oauth2.user_info_url`

**描述 / Description**: OIDC 用户信息端点 URL / OIDC user info endpoint URL

**类型 / Type**: `string`

**必需 / Required**: 否 / No

**示例 / Example**: `"https://auth.example.com/oauth2/userinfo"`

#### `security.oauth2.jwks_url`

**描述 / Description**: JSON Web Key Set URL / JSON Web Key Set URL

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当 `use_discovery` 为 `false` 时 / when `use_discovery` is `false`)

**示例 / Example**: `"https://auth.example.com/oauth2/jwks"`

**说明 / Notes**: 用于 JWT 令牌签名验证 / Used for JWT token signature verification

### OAuth2 作用域 / OAuth2 Scopes

#### `security.oauth2.scopes`

**描述 / Description**: OAuth2 请求的作用域列表 / List of OAuth2 scopes to request

**类型 / Type**: `[]string`

**默认值 / Default**: `["openid", "profile", "email"]`

**推荐值 / Recommended**: 
- 基本认证 / Basic authentication: `["openid", "profile", "email"]`
- 包含角色 / With roles: `["openid", "profile", "email", "roles"]`
- 离线访问 / Offline access: `["openid", "profile", "email", "offline_access"]`

**示例 / Example**:

```yaml
security:
  oauth2:
    scopes:
      - openid
      - profile
      - email
      - offline_access
```

### HTTP 配置 / HTTP Configuration

#### `security.oauth2.http_timeout`

**描述 / Description**: OAuth2 HTTP 请求超时时间 / OAuth2 HTTP request timeout

**类型 / Type**: `duration`

**默认值 / Default**: `30s`

**推荐值 / Recommended**: `30s` - `60s`

**示例 / Example**: `"30s"`, `"1m"`

### JWT 令牌验证配置 / JWT Token Validation Configuration

#### `security.oauth2.jwt.signing_method`

**描述 / Description**: JWT 签名算法 / JWT signing algorithm

**类型 / Type**: `string`

**默认值 / Default**: `"RS256"`

**可选值 / Available Values**:
- **非对称算法 / Asymmetric (推荐 / recommended)**:
  - `RS256`, `RS384`, `RS512` - RSA with SHA-256/384/512
  - `ES256`, `ES384`, `ES512` - ECDSA with SHA-256/384/512
  - `PS256`, `PS384`, `PS512` - RSA-PSS with SHA-256/384/512
- **对称算法 / Symmetric** (不推荐用于分布式系统 / not recommended for distributed systems):
  - `HS256`, `HS384`, `HS512` - HMAC with SHA-256/384/512

**推荐值 / Recommended**: `"RS256"`

#### `security.oauth2.jwt.clock_skew`

**描述 / Description**: 时钟偏差容忍度 / Clock skew tolerance

**类型 / Type**: `duration`

**默认值 / Default**: `30s`

**推荐值 / Recommended**: `30s` - `60s`

**说明 / Notes**: 
- 用于容忍客户端和服务器之间的时钟差异
- 设置过大可能导致过期令牌仍被接受

Used to tolerate clock differences between client and server.
Setting too large may cause expired tokens to still be accepted.

#### `security.oauth2.jwt.audience`

**描述 / Description**: 期望的令牌受众 / Expected token audience

**类型 / Type**: `string`

**推荐值 / Recommended**: 设置为你的 `client_id` / Set to your `client_id`

**示例 / Example**: `"my-service-client"`

#### `security.oauth2.jwt.issuer`

**描述 / Description**: 期望的令牌颁发者 / Expected token issuer

**类型 / Type**: `string`

**推荐值 / Recommended**: 设置为 `issuer_url` 的值 / Set to the value of `issuer_url`

**示例 / Example**: `"http://localhost:8081/realms/swit"`

#### `security.oauth2.jwt.required_claims`

**描述 / Description**: 必需的 JWT 声明 / Required JWT claims

**类型 / Type**: `[]string`

**默认值 / Default**: `[]`

**推荐值 / Recommended**: `["sub", "email"]`

**示例 / Example**:

```yaml
security:
  oauth2:
    jwt:
      required_claims:
        - sub
        - email
        - exp
```

### 令牌缓存配置 / Token Cache Configuration

#### `security.oauth2.cache.enabled`

**描述 / Description**: 是否启用令牌缓存 / Whether to enable token caching

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: `true` (高并发场景 / for high-concurrency scenarios)

**性能影响 / Performance Impact**: 可提升 100 倍以上的验证速度 / Can improve validation speed by 100x or more

#### `security.oauth2.cache.type`

**描述 / Description**: 缓存类型 / Cache type

**类型 / Type**: `string`

**默认值 / Default**: `"memory"`

**可选值 / Available Values**:
- `memory` - 内存缓存 / In-memory cache (推荐单实例 / recommended for single instance)
- `redis` - Redis 缓存 / Redis cache (推荐多实例 / recommended for multiple instances)

#### `security.oauth2.cache.ttl`

**描述 / Description**: 缓存过期时间 / Cache expiration time

**类型 / Type**: `duration`

**默认值 / Default**: `5m`

**推荐值 / Recommended**: `5m` - `15m`

**说明 / Notes**: 不应超过令牌的有效期 / Should not exceed token validity period

#### `security.oauth2.cache.max_size`

**描述 / Description**: 最大缓存条目数 / Maximum cache entries

**类型 / Type**: `integer`

**默认值 / Default**: `1000`

**推荐值 / Recommended**: 
- 小规模 / Small scale: `1000`
- 中等规模 / Medium scale: `5000`
- 大规模 / Large scale: `10000+`

#### `security.oauth2.cache.cleanup_interval`

**描述 / Description**: 缓存清理间隔 / Cache cleanup interval

**类型 / Type**: `duration`

**默认值 / Default**: `5m`

**推荐值 / Recommended**: `5m` - `10m`

### TLS 配置 / TLS Configuration

#### `security.oauth2.tls.enabled`

**描述 / Description**: 是否启用 TLS / Whether to enable TLS

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: 
- 开发环境 / Development: `false`
- 生产环境 / Production: `true`

#### `security.oauth2.tls.insecure_skip_verify`

**描述 / Description**: 是否跳过证书验证 / Whether to skip certificate verification

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: 
- **开发环境 / Development**: `true` (仅用于测试 / for testing only)
- **生产环境 / Production**: `false` (必须 / must be)

**安全警告 / Security Warning**: 
⚠️ **永远不要**在生产环境中设置为 `true`
⚠️ **Never** set to `true` in production environments

#### `security.oauth2.tls.cert_file`

**描述 / Description**: 客户端证书文件路径 / Client certificate file path

**类型 / Type**: `string`

**必需 / Required**: 否 / No (mTLS 时需要 / required for mTLS)

**示例 / Example**: `"/etc/ssl/certs/client-cert.pem"`

#### `security.oauth2.tls.key_file`

**描述 / Description**: 客户端私钥文件路径 / Client private key file path

**类型 / Type**: `string`

**必需 / Required**: 否 / No (mTLS 时需要 / required for mTLS)

**示例 / Example**: `"/etc/ssl/private/client-key.pem"`

#### `security.oauth2.tls.ca_file`

**描述 / Description**: CA 证书文件路径 / CA certificate file path

**类型 / Type**: `string`

**必需 / Required**: 否 / No

**示例 / Example**: `"/etc/ssl/certs/ca-bundle.crt"`

#### `security.oauth2.tls.min_version`

**描述 / Description**: 最低 TLS 版本 / Minimum TLS version

**类型 / Type**: `string`

**默认值 / Default**: `"TLS1.2"`

**可选值 / Available Values**: `TLS1.0`, `TLS1.1`, `TLS1.2`, `TLS1.3`

**推荐值 / Recommended**: `"TLS1.2"` 或 / or `"TLS1.3"`

#### `security.oauth2.tls.max_version`

**描述 / Description**: 最高 TLS 版本 / Maximum TLS version

**类型 / Type**: `string`

**默认值 / Default**: (使用系统默认 / use system default)

**可选值 / Available Values**: `TLS1.0`, `TLS1.1`, `TLS1.2`, `TLS1.3`

**推荐值 / Recommended**: `"TLS1.3"`

### OAuth2 配置示例 / OAuth2 Configuration Example

```yaml
security:
  enabled: true
  
  oauth2:
    # 基本配置 / Basic configuration
    enabled: true
    provider: keycloak
    client_id: my-service
    client_secret: ${OAUTH2_CLIENT_SECRET}
    redirect_url: http://localhost:8080/callback
    
    # OIDC 发现 / OIDC discovery
    issuer_url: http://localhost:8081/realms/swit
    use_discovery: true
    
    # 作用域 / Scopes
    scopes:
      - openid
      - profile
      - email
      - offline_access
    
    # HTTP 配置 / HTTP configuration
    http_timeout: 30s
    
    # JWT 验证 / JWT validation
    jwt:
      signing_method: RS256
      clock_skew: 30s
      audience: my-service
      issuer: http://localhost:8081/realms/swit
      required_claims:
        - sub
        - email
    
    # 令牌缓存 / Token cache
    cache:
      enabled: true
      type: memory
      ttl: 10m
      max_size: 5000
      cleanup_interval: 5m
    
    # TLS 配置 / TLS configuration
    tls:
      enabled: true
      insecure_skip_verify: false
      min_version: TLS1.2
```

---

## OPA 策略引擎配置 / OPA Policy Engine Configuration

Open Policy Agent (OPA) 提供了灵活的策略驱动授权控制。

Open Policy Agent (OPA) provides flexible policy-driven authorization control.

### 基本配置 / Basic Configuration

#### `security.opa.mode`

**描述 / Description**: OPA 运行模式 / OPA running mode

**类型 / Type**: `string`

**默认值 / Default**: `"embedded"`

**可选值 / Available Values**:
- `embedded` - 嵌入式模式 (推荐 / recommended)
- `remote` - 远程 OPA 服务器模式 / Remote OPA server mode

**推荐值 / Recommended**: `"embedded"`

**说明 / Notes**:
- `embedded`: OPA 引擎嵌入到应用中,策略文件从本地加载
- `remote`: 连接到独立的 OPA 服务器,策略由服务器管理

`embedded`: OPA engine embedded in application, policies loaded from local files.
`remote`: Connect to standalone OPA server, policies managed by server.

#### `security.opa.policy_dir`

**描述 / Description**: 策略文件目录 / Policy files directory

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (嵌入式模式 / for embedded mode)

**默认值 / Default**: `"./policies"`

**示例 / Example**: `"./policies"`, `"/etc/opa/policies"`

**说明 / Notes**: 
- 目录下应包含 `.rego` 策略文件
- 支持子目录递归加载

Directory should contain `.rego` policy files.
Supports recursive loading from subdirectories.

#### `security.opa.server_url`

**描述 / Description**: 远程 OPA 服务器 URL / Remote OPA server URL

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (远程模式 / for remote mode)

**示例 / Example**: `"http://localhost:8181"`

#### `security.opa.decision_path`

**描述 / Description**: OPA 决策路径 / OPA decision path

**类型 / Type**: `string`

**默认值 / Default**: `"/v1/data/authz/allow"`

**示例 / Example**: `"/v1/data/authz/allow"`, `"/v1/data/myapp/policy/allow"`

**说明 / Notes**: 指定在 OPA 策略中要评估的决策路径 / Specifies the decision path to evaluate in OPA policy

### 策略配置 / Policy Configuration

#### `security.opa.watch_policy_changes`

**描述 / Description**: 是否监视策略文件变更 / Whether to watch policy file changes

**类型 / Type**: `boolean`

**默认值 / Default**: `true`

**推荐值 / Recommended**: 
- 开发环境 / Development: `true`
- 生产环境 / Production: `false` (使用版本控制和部署流程 / use version control and deployment process)

#### `security.opa.default_decision`

**描述 / Description**: 策略评估失败时的默认决策 / Default decision when policy evaluation fails

**类型 / Type**: `string`

**默认值 / Default**: `"deny"`

**可选值 / Available Values**:
- `deny` - 拒绝访问 (推荐 / recommended)
- `allow` - 允许访问 (不安全 / insecure)

**推荐值 / Recommended**: `"deny"`

**安全建议 / Security Recommendation**: 始终使用 `deny` 以确保安全失败 / Always use `deny` to ensure fail-secure

### OPA 配置示例 / OPA Configuration Example

```yaml
security:
  enabled: true
  
  opa:
    # 嵌入式模式 / Embedded mode
    mode: embedded
    policy_dir: ./policies
    decision_path: /v1/data/authz/allow
    watch_policy_changes: true
    default_decision: deny
```

```yaml
security:
  enabled: true
  
  opa:
    # 远程模式 / Remote mode
    mode: remote
    server_url: http://opa-server:8181
    decision_path: /v1/data/authz/allow
    default_decision: deny
```

---

## 安全扫描器配置 / Security Scanner Configuration

安全扫描器集成了多种安全扫描工具,用于自动化代码和依赖项安全扫描。

Security scanner integrates multiple security scanning tools for automated code and dependency security scanning.

### 基本配置 / Basic Configuration

#### `security.scanner.enabled`

**描述 / Description**: 是否启用安全扫描 / Whether to enable security scanning

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: 
- CI/CD 环境 / CI/CD: `true`
- 本地开发 / Local development: `false`

#### `security.scanner.tools`

**描述 / Description**: 启用的扫描工具列表 / List of enabled scanning tools

**类型 / Type**: `[]string`

**默认值 / Default**: `[]`

**可选值 / Available Values**:
- `gosec` - Go 安全扫描器 / Go security scanner
- `trivy` - 容器和依赖漏洞扫描 / Container and dependency vulnerability scanner
- `govulncheck` - Go 官方漏洞检查 / Official Go vulnerability check

**推荐值 / Recommended**: `["gosec", "trivy", "govulncheck"]`

**示例 / Example**:

```yaml
security:
  scanner:
    enabled: true
    tools:
      - gosec
      - trivy
      - govulncheck
```

### 扫描配置 / Scan Configuration

#### `security.scanner.scan_on_startup`

**描述 / Description**: 是否在启动时执行扫描 / Whether to run scan on startup

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: `false` (通过 CI/CD 运行 / run through CI/CD instead)

#### `security.scanner.fail_on_errors`

**描述 / Description**: 发现漏洞时是否失败 / Whether to fail when vulnerabilities are found

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: 
- CI/CD: `true`
- 开发环境 / Development: `false`

#### `security.scanner.severity_threshold`

**描述 / Description**: 漏洞严重性阈值 / Vulnerability severity threshold

**类型 / Type**: `string`

**默认值 / Default**: `"HIGH"`

**可选值 / Available Values**: `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`

**推荐值 / Recommended**: `"HIGH"` 或 / or `"CRITICAL"`

### 安全扫描器配置示例 / Security Scanner Configuration Example

```yaml
security:
  enabled: true
  
  scanner:
    enabled: true
    tools:
      - gosec
      - trivy
      - govulncheck
    scan_on_startup: false
    fail_on_errors: true
    severity_threshold: HIGH
```

---

## 密钥管理配置 / Secrets Management Configuration

密钥管理提供了统一的接口来访问和管理敏感配置信息。

Secrets management provides a unified interface to access and manage sensitive configuration information.

### 基本配置 / Basic Configuration

#### `security.secrets.provider`

**描述 / Description**: 密钥管理提供商 / Secrets management provider

**类型 / Type**: `string`

**默认值 / Default**: `"env"`

**可选值 / Available Values**:
- `env` - 环境变量 (开发环境 / development)
- `vault` - HashiCorp Vault (推荐生产环境 / recommended for production)
- `aws` - AWS Secrets Manager
- `gcp` - GCP Secret Manager
- `azure` - Azure Key Vault

**推荐值 / Recommended**: 
- 开发环境 / Development: `env`
- 生产环境 / Production: `vault` 或 / or cloud provider

### Vault 配置 / Vault Configuration

#### `security.secrets.vault.address`

**描述 / Description**: Vault 服务器地址 / Vault server address

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当使用 Vault 时 / when using Vault)

**示例 / Example**: `"http://localhost:8200"`, `"https://vault.example.com"`

#### `security.secrets.vault.token`

**描述 / Description**: Vault 访问令牌 / Vault access token

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (根认证方式 / for root authentication)

**环境变量 / Environment Variable**: `VAULT_TOKEN`

**安全建议 / Security Recommendation**: 使用环境变量 / Use environment variables

#### `security.secrets.vault.auth_method`

**描述 / Description**: Vault 认证方法 / Vault authentication method

**类型 / Type**: `string`

**默认值 / Default**: `"token"`

**可选值 / Available Values**:
- `token` - 令牌认证 / Token authentication
- `approle` - AppRole 认证 (推荐 / recommended)
- `kubernetes` - Kubernetes 认证 / Kubernetes authentication
- `aws` - AWS IAM 认证 / AWS IAM authentication

#### `security.secrets.vault.role_id`

**描述 / Description**: Vault AppRole Role ID

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当使用 AppRole 时 / when using AppRole)

**环境变量 / Environment Variable**: `VAULT_ROLE_ID`

#### `security.secrets.vault.secret_id`

**描述 / Description**: Vault AppRole Secret ID

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当使用 AppRole 时 / when using AppRole)

**环境变量 / Environment Variable**: `VAULT_SECRET_ID`

#### `security.secrets.vault.kv_version`

**描述 / Description**: Vault KV 引擎版本 / Vault KV engine version

**类型 / Type**: `integer`

**默认值 / Default**: `2`

**可选值 / Available Values**: `1`, `2`

**推荐值 / Recommended**: `2`

#### `security.secrets.vault.mount_path`

**描述 / Description**: Vault 挂载路径 / Vault mount path

**类型 / Type**: `string`

**默认值 / Default**: `"secret"`

**示例 / Example**: `"secret"`, `"kv"`

#### `security.secrets.vault.path`

**描述 / Description**: 密钥路径 / Secrets path

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes

**示例 / Example**: `"swit/production"`, `"myapp/config"`

### AWS Secrets Manager 配置 / AWS Secrets Manager Configuration

#### `security.secrets.aws.region`

**描述 / Description**: AWS 区域 / AWS region

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当使用 AWS 时 / when using AWS)

**示例 / Example**: `"us-west-2"`, `"eu-west-1"`

#### `security.secrets.aws.access_key_id`

**描述 / Description**: AWS 访问密钥 ID / AWS access key ID

**类型 / Type**: `string`

**环境变量 / Environment Variable**: `AWS_ACCESS_KEY_ID`

#### `security.secrets.aws.secret_access_key`

**描述 / Description**: AWS 密钥访问密钥 / AWS secret access key

**类型 / Type**: `string`

**环境变量 / Environment Variable**: `AWS_SECRET_ACCESS_KEY`

### 密钥管理配置示例 / Secrets Management Configuration Example

```yaml
# 环境变量模式 / Environment variables mode
security:
  enabled: true
  
  secrets:
    provider: env
```

```yaml
# Vault 令牌认证 / Vault token authentication
security:
  enabled: true
  
  secrets:
    provider: vault
    vault:
      address: http://localhost:8200
      token: ${VAULT_TOKEN}
      kv_version: 2
      mount_path: secret
      path: swit/production
```

```yaml
# Vault AppRole 认证 / Vault AppRole authentication
security:
  enabled: true
  
  secrets:
    provider: vault
    vault:
      address: https://vault.example.com
      auth_method: approle
      role_id: ${VAULT_ROLE_ID}
      secret_id: ${VAULT_SECRET_ID}
      kv_version: 2
      mount_path: secret
      path: swit/production
```

```yaml
# AWS Secrets Manager
security:
  enabled: true
  
  secrets:
    provider: aws
    aws:
      region: us-west-2
      # 使用 IAM 角色或环境变量 / Use IAM role or environment variables
```

---

## 审计日志配置 / Audit Logging Configuration

审计日志记录所有安全相关的事件,用于合规性和安全分析。

Audit logging records all security-related events for compliance and security analysis.

### 基本配置 / Basic Configuration

#### `security.audit.enabled`

**描述 / Description**: 是否启用审计日志 / Whether to enable audit logging

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: 
- 开发环境 / Development: `false`
- 生产环境 / Production: `true`

#### `security.audit.output_type`

**描述 / Description**: 审计日志输出类型 / Audit log output type

**类型 / Type**: `string`

**默认值 / Default**: `"stdout"`

**可选值 / Available Values**:
- `stdout` - 标准输出 / Standard output
- `file` - 文件输出 / File output
- `syslog` - Syslog 输出 / Syslog output

**推荐值 / Recommended**: 
- 开发环境 / Development: `stdout`
- 生产环境 / Production: `file` 或 / or `syslog`

### 文件输出配置 / File Output Configuration

#### `security.audit.file_path`

**描述 / Description**: 审计日志文件路径 / Audit log file path

**类型 / Type**: `string`

**必需 / Required**: 是 / Yes (当 `output_type` 为 `file` 时 / when `output_type` is `file`)

**示例 / Example**: `"/var/log/security-audit.log"`

#### `security.audit.format`

**描述 / Description**: 日志格式 / Log format

**类型 / Type**: `string`

**默认值 / Default**: `"json"`

**可选值 / Available Values**:
- `json` - JSON 格式 (推荐 / recommended)
- `text` - 文本格式 / Text format

**推荐值 / Recommended**: `"json"`

#### `security.audit.max_size_mb`

**描述 / Description**: 单个日志文件最大大小 (MB) / Maximum size of single log file (MB)

**类型 / Type**: `integer`

**默认值 / Default**: `100`

**推荐值 / Recommended**: `100` - `500`

#### `security.audit.max_backups`

**描述 / Description**: 保留的日志文件数量 / Number of log files to retain

**类型 / Type**: `integer`

**默认值 / Default**: `3`

**推荐值 / Recommended**: `7` - `30`

#### `security.audit.max_age_days`

**描述 / Description**: 日志文件保留天数 / Log file retention days

**类型 / Type**: `integer`

**默认值 / Default**: `30`

**推荐值 / Recommended**: 
- 一般应用 / General applications: `30` - `90`
- 合规要求 / Compliance requirements: `365` - `2555` (7 years)

#### `security.audit.compress`

**描述 / Description**: 是否压缩旧日志文件 / Whether to compress old log files

**类型 / Type**: `boolean`

**默认值 / Default**: `true`

**推荐值 / Recommended**: `true`

### 事件过滤配置 / Event Filtering Configuration

#### `security.audit.event_types`

**描述 / Description**: 要记录的事件类型 / Event types to log

**类型 / Type**: `[]string`

**默认值 / Default**: `["all"]`

**可选值 / Available Values**:
- `all` - 所有事件 / All events
- `authentication` - 认证事件 / Authentication events
- `authorization` - 授权事件 / Authorization events
- `token_issued` - 令牌颁发 / Token issued
- `token_revoked` - 令牌撤销 / Token revoked
- `policy_evaluated` - 策略评估 / Policy evaluated
- `access_denied` - 访问拒绝 / Access denied
- `config_changed` - 配置变更 / Configuration changed

**示例 / Example**:

```yaml
security:
  audit:
    event_types:
      - authentication
      - authorization
      - access_denied
```

#### `security.audit.include_request_body`

**描述 / Description**: 是否包含请求体 / Whether to include request body

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: `false` (避免记录敏感数据 / avoid logging sensitive data)

#### `security.audit.include_response_body`

**描述 / Description**: 是否包含响应体 / Whether to include response body

**类型 / Type**: `boolean`

**默认值 / Default**: `false`

**推荐值 / Recommended**: `false`

#### `security.audit.sensitive_fields`

**描述 / Description**: 敏感字段列表 (将被脱敏) / Sensitive fields list (will be redacted)

**类型 / Type**: `[]string`

**默认值 / Default**: `["password", "token", "secret", "api_key"]`

**推荐值 / Recommended**: 根据应用需求配置 / Configure based on application needs

**示例 / Example**:

```yaml
security:
  audit:
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
      - ssn
      - credit_card
```

### 审计日志配置示例 / Audit Logging Configuration Example

```yaml
# 标准输出 / Standard output
security:
  enabled: true
  
  audit:
    enabled: true
    output_type: stdout
    format: json
    event_types:
      - authentication
      - authorization
      - access_denied
```

```yaml
# 文件输出 / File output
security:
  enabled: true
  
  audit:
    enabled: true
    output_type: file
    file_path: /var/log/swit/security-audit.log
    format: json
    max_size_mb: 100
    max_backups: 30
    max_age_days: 90
    compress: true
    
    # 事件过滤 / Event filtering
    event_types:
      - all
    
    # 数据保护 / Data protection
    include_request_body: false
    include_response_body: false
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
```

---

## 环境变量映射 / Environment Variable Mapping

所有安全配置都可以通过环境变量覆盖。环境变量名称遵循以下规则:

All security configurations can be overridden using environment variables. Environment variable names follow these rules:

1. 前缀为 `SWIT_SECURITY_` / Prefix with `SWIT_SECURITY_`
2. 使用大写字母 / Use uppercase letters
3. 使用下划线分隔层级 / Use underscores to separate levels

### OAuth2 环境变量 / OAuth2 Environment Variables

| 配置项 / Config | 环境变量 / Environment Variable | 示例 / Example |
|----------------|--------------------------------|----------------|
| `security.oauth2.enabled` | `SWIT_SECURITY_OAUTH2_ENABLED` | `true` |
| `security.oauth2.provider` | `SWIT_SECURITY_OAUTH2_PROVIDER` | `keycloak` |
| `security.oauth2.client_id` | `SWIT_SECURITY_OAUTH2_CLIENT_ID` 或 / or `OAUTH2_CLIENT_ID` | `my-client` |
| `security.oauth2.client_secret` | `SWIT_SECURITY_OAUTH2_CLIENT_SECRET` 或 / or `OAUTH2_CLIENT_SECRET` | `my-secret` |
| `security.oauth2.redirect_url` | `SWIT_SECURITY_OAUTH2_REDIRECT_URL` 或 / or `OAUTH2_REDIRECT_URL` | `http://localhost:8080/callback` |
| `security.oauth2.issuer_url` | `SWIT_SECURITY_OAUTH2_ISSUER_URL` 或 / or `OAUTH2_ISSUER_URL` | `http://localhost:8081/realms/swit` |
| `security.oauth2.use_discovery` | `SWIT_SECURITY_OAUTH2_USE_DISCOVERY` 或 / or `OAUTH2_USE_DISCOVERY` | `true` |
| `security.oauth2.jwt.signing_method` | `SWIT_SECURITY_OAUTH2_JWT_SIGNING_METHOD` 或 / or `OAUTH2_JWT_SIGNING_METHOD` | `RS256` |
| `security.oauth2.jwt.audience` | `SWIT_SECURITY_OAUTH2_JWT_AUDIENCE` 或 / or `OAUTH2_JWT_AUDIENCE` | `my-client` |
| `security.oauth2.cache.enabled` | `SWIT_SECURITY_OAUTH2_CACHE_ENABLED` 或 / or `OAUTH2_CACHE_ENABLED` | `true` |
| `security.oauth2.cache.ttl` | `SWIT_SECURITY_OAUTH2_CACHE_TTL` 或 / or `OAUTH2_CACHE_TTL` | `10m` |
| `security.oauth2.tls.enabled` | `SWIT_SECURITY_OAUTH2_TLS_ENABLED` 或 / or `OAUTH2_TLS_ENABLED` | `true` |

### OPA 环境变量 / OPA Environment Variables

| 配置项 / Config | 环境变量 / Environment Variable | 示例 / Example |
|----------------|--------------------------------|----------------|
| `security.opa.mode` | `SWIT_SECURITY_OPA_MODE` | `embedded` |
| `security.opa.policy_dir` | `SWIT_SECURITY_OPA_POLICY_DIR` | `./policies` |
| `security.opa.server_url` | `SWIT_SECURITY_OPA_SERVER_URL` | `http://localhost:8181` |

### Scanner 环境变量 / Scanner Environment Variables

| 配置项 / Config | 环境变量 / Environment Variable | 示例 / Example |
|----------------|--------------------------------|----------------|
| `security.scanner.enabled` | `SWIT_SECURITY_SCANNER_ENABLED` | `true` |
| `security.scanner.fail_on_errors` | `SWIT_SECURITY_SCANNER_FAIL_ON_ERRORS` | `true` |
| `security.scanner.severity_threshold` | `SWIT_SECURITY_SCANNER_SEVERITY_THRESHOLD` | `HIGH` |

### Secrets 环境变量 / Secrets Environment Variables

| 配置项 / Config | 环境变量 / Environment Variable | 示例 / Example |
|----------------|--------------------------------|----------------|
| `security.secrets.provider` | `SWIT_SECURITY_SECRETS_PROVIDER` | `vault` |
| `security.secrets.vault.address` | `SWIT_SECURITY_SECRETS_VAULT_ADDRESS` 或 / or `VAULT_ADDR` | `http://localhost:8200` |
| `security.secrets.vault.token` | `SWIT_SECURITY_SECRETS_VAULT_TOKEN` 或 / or `VAULT_TOKEN` | `my-token` |
| `security.secrets.vault.role_id` | `SWIT_SECURITY_SECRETS_VAULT_ROLE_ID` 或 / or `VAULT_ROLE_ID` | `my-role-id` |
| `security.secrets.vault.secret_id` | `SWIT_SECURITY_SECRETS_VAULT_SECRET_ID` 或 / or `VAULT_SECRET_ID` | `my-secret-id` |

### Audit 环境变量 / Audit Environment Variables

| 配置项 / Config | 环境变量 / Environment Variable | 示例 / Example |
|----------------|--------------------------------|----------------|
| `security.audit.enabled` | `SWIT_SECURITY_AUDIT_ENABLED` | `true` |
| `security.audit.output_type` | `SWIT_SECURITY_AUDIT_OUTPUT_TYPE` | `file` |
| `security.audit.file_path` | `SWIT_SECURITY_AUDIT_FILE_PATH` | `/var/log/audit.log` |

### 环境变量使用示例 / Environment Variable Usage Example

```bash
# OAuth2 配置 / OAuth2 configuration
export OAUTH2_CLIENT_ID="my-service"
export OAUTH2_CLIENT_SECRET="super-secret-value"
export OAUTH2_ISSUER_URL="http://localhost:8081/realms/swit"
export OAUTH2_USE_DISCOVERY="true"

# JWT 配置 / JWT configuration
export OAUTH2_JWT_SIGNING_METHOD="RS256"
export OAUTH2_JWT_AUDIENCE="my-service"

# 缓存配置 / Cache configuration
export OAUTH2_CACHE_ENABLED="true"
export OAUTH2_CACHE_TTL="10m"

# Vault 配置 / Vault configuration
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="my-vault-token"

# 审计配置 / Audit configuration
export SWIT_SECURITY_AUDIT_ENABLED="true"
export SWIT_SECURITY_AUDIT_OUTPUT_TYPE="file"
export SWIT_SECURITY_AUDIT_FILE_PATH="/var/log/swit-audit.log"

# 启动应用 / Start application
./swit-serve
```

---

## 完整配置示例 / Complete Configuration Examples

### 开发环境配置 / Development Environment Configuration

```yaml
# swit-dev.yaml
service_name: my-service-dev

security:
  # 开发环境启用基本安全功能 / Enable basic security features for development
  enabled: true
  
  # OAuth2 配置 / OAuth2 configuration
  oauth2:
    enabled: true
    provider: keycloak
    client_id: my-service-dev
    client_secret: ${OAUTH2_CLIENT_SECRET}
    redirect_url: http://localhost:8080/callback
    
    # 使用本地 Keycloak / Use local Keycloak
    issuer_url: http://localhost:8081/realms/swit
    use_discovery: true
    
    scopes:
      - openid
      - profile
      - email
    
    # 开发环境配置 / Development settings
    http_timeout: 30s
    skip_issuer_verification: false
    skip_expiry_check: false
    
    jwt:
      signing_method: RS256
      clock_skew: 60s
      required_claims:
        - sub
        - email
    
    cache:
      enabled: true
      type: memory
      ttl: 5m
      max_size: 1000
    
    # 开发环境不强制 TLS / No TLS enforcement for development
    tls:
      enabled: false
  
  # OPA 配置 / OPA configuration
  opa:
    mode: embedded
    policy_dir: ./policies
    decision_path: /v1/data/authz/allow
    watch_policy_changes: true
    default_decision: deny
  
  # 扫描器在开发环境禁用 / Scanner disabled for development
  scanner:
    enabled: false
  
  # 使用环境变量作为密钥源 / Use environment variables for secrets
  secrets:
    provider: env
  
  # 审计日志输出到标准输出 / Audit logs to stdout
  audit:
    enabled: true
    output_type: stdout
    format: json
    event_types:
      - authentication
      - authorization
      - access_denied
```

### 生产环境配置 / Production Environment Configuration

```yaml
# swit-production.yaml
service_name: my-service-prod

security:
  # 生产环境必须启用 / Must enable for production
  enabled: true
  
  # OAuth2 配置 / OAuth2 configuration
  oauth2:
    enabled: true
    provider: keycloak
    client_id: my-service-prod
    client_secret: ${OAUTH2_CLIENT_SECRET}  # 从 Vault 或环境变量读取 / Read from Vault or environment
    redirect_url: https://api.example.com/callback
    
    # 生产环境 Keycloak / Production Keycloak
    issuer_url: https://auth.example.com/realms/production
    use_discovery: true
    
    scopes:
      - openid
      - profile
      - email
      - offline_access
    
    # 生产环境超时配置 / Production timeout settings
    http_timeout: 30s
    skip_issuer_verification: false  # 必须为 false / Must be false
    skip_expiry_check: false         # 必须为 false / Must be false
    
    jwt:
      signing_method: RS256
      clock_skew: 30s
      audience: my-service-prod
      issuer: https://auth.example.com/realms/production
      required_claims:
        - sub
        - email
        - exp
    
    # 生产环境使用大缓存 / Use large cache for production
    cache:
      enabled: true
      type: memory  # 或 redis 用于多实例 / or redis for multiple instances
      ttl: 10m
      max_size: 10000
      cleanup_interval: 5m
    
    # 生产环境必须启用 TLS / Must enable TLS for production
    tls:
      enabled: true
      insecure_skip_verify: false  # 永远不要在生产环境设置为 true / Never set to true in production
      ca_file: /etc/ssl/certs/ca-bundle.crt
      min_version: TLS1.2
      max_version: TLS1.3
  
  # OPA 配置 / OPA configuration
  opa:
    mode: embedded
    policy_dir: /etc/opa/policies
    decision_path: /v1/data/authz/allow
    watch_policy_changes: false  # 生产环境使用版本控制 / Use version control in production
    default_decision: deny       # 必须为 deny / Must be deny
  
  # 扫描器通过 CI/CD 运行 / Scanner runs through CI/CD
  scanner:
    enabled: false  # 不在运行时扫描 / Don't scan at runtime
  
  # 生产环境使用 Vault / Use Vault for production
  secrets:
    provider: vault
    vault:
      address: https://vault.example.com
      auth_method: approle
      role_id: ${VAULT_ROLE_ID}
      secret_id: ${VAULT_SECRET_ID}
      kv_version: 2
      mount_path: secret
      path: swit/production
      tls:
        enabled: true
        ca_file: /etc/ssl/certs/vault-ca.crt
  
  # 生产环境完整审计 / Complete audit for production
  audit:
    enabled: true
    output_type: file
    file_path: /var/log/swit/security-audit.log
    format: json
    max_size_mb: 500
    max_backups: 30
    max_age_days: 365  # 根据合规要求调整 / Adjust based on compliance requirements
    compress: true
    
    # 记录所有安全事件 / Log all security events
    event_types:
      - all
    
    # 数据保护 / Data protection
    include_request_body: false
    include_response_body: false
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
      - access_token
      - refresh_token
      - authorization
      - ssn
      - credit_card
```

### Kubernetes 环境配置 / Kubernetes Environment Configuration

```yaml
# ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: swit-security-config
  namespace: production
data:
  swit.yaml: |
    service_name: my-service
    
    security:
      enabled: true
      
      oauth2:
        enabled: true
        provider: keycloak
        client_id: my-service
        # client_secret 从 Secret 读取 / client_secret read from Secret
        redirect_url: https://api.example.com/callback
        issuer_url: https://auth.example.com/realms/production
        use_discovery: true
        scopes: [openid, profile, email]
        http_timeout: 30s
        
        jwt:
          signing_method: RS256
          clock_skew: 30s
        
        cache:
          enabled: true
          type: memory
          ttl: 10m
          max_size: 10000
        
        tls:
          enabled: true
          insecure_skip_verify: false
          ca_file: /etc/ssl/certs/ca-bundle.crt
          min_version: TLS1.2
      
      opa:
        mode: embedded
        policy_dir: /etc/opa/policies
        default_decision: deny
      
      secrets:
        provider: vault
        vault:
          address: http://vault:8200
          auth_method: kubernetes
          role: my-service
          mount_path: secret
          path: swit/production
      
      audit:
        enabled: true
        output_type: file
        file_path: /var/log/swit/audit.log
        format: json
        max_size_mb: 500
        max_backups: 30
        max_age_days: 90
        compress: true

---
# Secret
apiVersion: v1
kind: Secret
metadata:
  name: swit-security-secrets
  namespace: production
type: Opaque
stringData:
  OAUTH2_CLIENT_SECRET: "your-client-secret-here"
  VAULT_ROLE_ID: "your-role-id-here"
  VAULT_SECRET_ID: "your-secret-id-here"

---
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: swit-service
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: swit-service
  template:
    metadata:
      labels:
        app: swit-service
    spec:
      containers:
      - name: swit-service
        image: swit-service:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9080
          name: grpc
        
        # 挂载配置 / Mount configuration
        volumeMounts:
        - name: config
          mountPath: /etc/swit
          readOnly: true
        - name: policies
          mountPath: /etc/opa/policies
          readOnly: true
        - name: logs
          mountPath: /var/log/swit
        
        # 环境变量 / Environment variables
        env:
        - name: OAUTH2_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: swit-security-secrets
              key: OAUTH2_CLIENT_SECRET
        - name: VAULT_ROLE_ID
          valueFrom:
            secretKeyRef:
              name: swit-security-secrets
              key: VAULT_ROLE_ID
        - name: VAULT_SECRET_ID
          valueFrom:
            secretKeyRef:
              name: swit-security-secrets
              key: VAULT_SECRET_ID
        
        # 资源限制 / Resource limits
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        
        # 健康检查 / Health checks
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      
      volumes:
      - name: config
        configMap:
          name: swit-security-config
      - name: policies
        configMap:
          name: opa-policies
      - name: logs
        emptyDir: {}
```

---

## 相关文档 / Related Documentation

- [OAuth2/OIDC 集成指南](./oauth2-integration-guide.md)
- [OPA RBAC 指南](./opa-rbac-guide.md)
- [OPA ABAC 指南](./opa-abac-guide.md)
- [安全最佳实践](./security-best-practices.md)
- [安全检查清单](./security-checklist.md)

---

**版本 / Version**: 1.0.0  
**最后更新 / Last Updated**: 2025-01-24  
**维护者 / Maintainer**: Swit Framework Team

**许可证 / License**: Apache License 2.0  
Copyright © 2024-2025 Six-Thirty Labs, Inc.

