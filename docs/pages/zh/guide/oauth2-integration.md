# OAuth2/OIDC 集成指南

本指南将帮助你在 Swit 框架中集成 OAuth2/OIDC 认证，支持多种主流提供商（Keycloak、Auth0、Google、Microsoft、Okta 等）。

## 概述

Swit 框架提供了完整的 OAuth2/OIDC 认证支持，主要特性包括：

### 核心特性

- ✅ **多提供商支持** - Keycloak、Auth0、Google、Microsoft、Okta 等
- ✅ **OIDC 自动发现** - 自动从 issuer URL 发现端点配置
- ✅ **多种授权流程** - 授权码、PKCE、客户端凭证、密码、刷新令牌
- ✅ **JWT 令牌验证** - 本地 JWT 签名验证和远程令牌校验
- ✅ **令牌缓存** - 内置令牌缓存机制提升性能
- ✅ **TLS/mTLS 支持** - 完整的传输层安全配置
- ✅ **Gin 中间件** - 开箱即用的路由保护中间件
- ✅ **角色权限控制** - 与 OPA 集成实现 RBAC/ABAC
- ✅ **健康检查** - 内置提供商连接健康检查

### 支持的提供商

| 提供商 | 类型标识 | OIDC 发现 | 默认作用域 |
|--------|----------|----------|-----------|
| Keycloak | `keycloak` | ✅ | openid, profile, email |
| Auth0 | `auth0` | ✅ | openid, profile, email |
| Google | `google` | ✅ | openid, profile, email |
| Microsoft | `microsoft` | ✅ | openid, profile, email |
| Okta | `okta` | ✅ | openid, profile, email |
| 自定义 | `custom` | 可选 | 自定义 |

---

## 快速开始

### 前置条件

- Go 1.23+
- 已安装并运行的 OAuth2 提供商（本示例使用 Keycloak）
- Docker 和 Docker Compose（可选，用于快速演示）

### 1. 使用 Docker Compose 快速体验

最快速的方式是使用我们提供的完整示例：

```bash
# 克隆仓库
git clone https://github.com/innovationmech/swit.git
cd swit/examples/oauth2-authentication

# 启动所有服务（Keycloak + PostgreSQL + 示例服务）
docker-compose up -d

# 等待服务就绪（约 60 秒）
docker-compose ps
```

这将启动：
- **Keycloak** (http://localhost:8081) - OIDC 提供商
- **PostgreSQL** (localhost:5432) - Keycloak 数据库
- **OAuth2 示例服务** (http://localhost:8080) - 演示应用

Keycloak 已预配置：
- Realm: `swit`
- Client: `swit-example` (密钥: `swit-example-secret`)
- 用户: `testuser`/`password`, `admin`/`admin`

### 2. 测试认证流程

#### 步骤 1: 获取服务信息

```bash
curl http://localhost:8080/api/v1/public/info
```

#### 步骤 2: 发起登录

```bash
curl http://localhost:8080/api/v1/public/login
```

复制返回的 `authorization_url`，在浏览器中打开。

#### 步骤 3: 使用 Keycloak 登录

1. 输入凭证：`testuser` / `password`
2. 授权后重定向到回调 URL
3. 复制返回的 `access_token`

#### 步骤 4: 访问受保护端点

```bash
export ACCESS_TOKEN="your-access-token-here"

curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/protected/profile
```

### 3. 基础代码示例

以下是一个最小化的集成示例：

```go
package main

import (
    "context"
    "log"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/middleware"
    "github.com/innovationmech/swit/pkg/security/jwt"
    "github.com/innovationmech/swit/pkg/security/oauth2"
)

func main() {
    // 1. 配置 OAuth2 客户端
    oauth2Config := &oauth2.Config{
        Enabled:      true,
        Provider:     "keycloak",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        IssuerURL:    "http://localhost:8081/realms/swit",
        RedirectURL:  "http://localhost:8080/callback",
        UseDiscovery: true,
        Scopes:       []string{"openid", "profile", "email"},
    }
    
    // 2. 创建 OAuth2 客户端
    ctx := context.Background()
    oauth2Client, err := oauth2.NewClient(ctx, oauth2Config)
    if err != nil {
        log.Fatalf("Failed to create OAuth2 client: %v", err)
    }
    defer oauth2Client.Close()
    
    // 3. 创建 JWT 验证器（用于本地令牌验证）
    jwtConfig := &jwt.Config{
        SigningMethod: "RS256",
        JWKSURL:       oauth2Client.GetJWKSURL(),
        Issuer:        oauth2Config.IssuerURL,
        Audience:      oauth2Config.ClientID,
    }
    jwtValidator, err := jwt.NewValidator(jwtConfig)
    if err != nil {
        log.Fatalf("Failed to create JWT validator: %v", err)
    }
    
    // 4. 创建 Gin 路由并应用中间件
    router := gin.Default()
    
    // 公开端点
    router.GET("/login", func(c *gin.Context) {
        authURL := oauth2Client.AuthCodeURL("random-state")
        c.JSON(200, gin.H{"authorization_url": authURL})
    })
    
    // 受保护端点
    protected := router.Group("/protected")
    protected.Use(middleware.OAuth2Middleware(oauth2Client, jwtValidator))
    {
        protected.GET("/profile", func(c *gin.Context) {
            userInfo, _ := middleware.GetUserInfo(c)
            c.JSON(200, userInfo)
        })
    }
    
    // 启动服务
    router.Run(":8080")
}
```

---

## 配置指南

### 完整配置示例

创建 `swit.yaml` 配置文件：

```yaml
service_name: "my-oauth2-service"

# OAuth2/OIDC 配置
oauth2:
  # 基本配置
  enabled: true
  provider: keycloak  # keycloak, auth0, google, microsoft, okta, custom
  client_id: my-app
  client_secret: ${OAUTH2_CLIENT_SECRET}  # 建议使用环境变量
  redirect_url: http://localhost:8080/callback
  
  # OIDC 发现配置
  issuer_url: http://localhost:8081/realms/myrealm
  use_discovery: true  # 启用自动端点发现
  
  # OAuth2 作用域
  scopes:
    - openid
    - profile
    - email
    - offline_access  # 用于刷新令牌
  
  # HTTP 超时
  http_timeout: 30s
  
  # JWT 令牌验证配置
  jwt:
    signing_method: RS256  # RS256, RS384, RS512, HS256, ES256
    clock_skew: 30s        # 时钟偏差容忍度
    audience: my-app       # 期望的 audience 声明
    issuer: http://localhost:8081/realms/myrealm
    required_claims:       # 必需的声明
      - sub
      - email
  
  # 令牌缓存配置
  cache:
    enabled: true
    type: memory
    ttl: 10m              # 缓存过期时间
    max_size: 1000        # 最大缓存条目数
    cleanup_interval: 5m  # 清理间隔
  
  # TLS 配置（生产环境必需）
  tls:
    enabled: true
    insecure_skip_verify: false  # 生产环境必须为 false
    min_version: TLS1.2
```

### 环境变量覆盖

所有配置都可以通过环境变量覆盖：

```bash
# OAuth2 基本配置
export OAUTH2_ENABLED=true
export OAUTH2_PROVIDER=keycloak
export OAUTH2_CLIENT_ID=my-app
export OAUTH2_CLIENT_SECRET=my-secret
export OAUTH2_ISSUER_URL=http://localhost:8081/realms/swit
export OAUTH2_REDIRECT_URL=http://localhost:8080/callback

# JWT 配置
export OAUTH2_JWT_SIGNING_METHOD=RS256
export OAUTH2_JWT_AUDIENCE=my-app

# 缓存配置
export OAUTH2_CACHE_ENABLED=true
export OAUTH2_CACHE_TTL=10m
```

---

## 多提供商集成

### Keycloak 集成

Keycloak 是最推荐的开源 OAuth2/OIDC 提供商。

#### 1. Keycloak 服务器配置

使用 Docker 快速启动：

```bash
docker run -d \
  --name keycloak \
  -p 8081:8080 \
  -e KEYCLOAK_ADMIN=admin \
  -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:latest \
  start-dev
```

#### 2. 创建 Realm 和 Client

访问 http://localhost:8081/admin，使用 `admin`/`admin` 登录：

1. **创建 Realm**
   - 点击左上角下拉菜单，选择 "Create Realm"
   - Name: `swit`
   - 点击 "Create"

2. **创建 Client**
   - 进入 Clients 页面，点击 "Create client"
   - Client ID: `swit-app`
   - Client Protocol: `openid-connect`
   - 点击 "Next"

3. **配置 Client 设置**
   - Client authentication: `On` (机密客户端)
   - Authorization: `Off`
   - Authentication flow: 勾选 `Standard flow` 和 `Direct access grants`
   - 点击 "Next"

4. **配置 Redirect URIs**
   - Valid redirect URIs: `http://localhost:8080/*`
   - Web origins: `http://localhost:8080`
   - 点击 "Save"

5. **获取 Client Secret**
   - 进入 "Credentials" 标签
   - 复制 "Client secret" 值

#### 3. 配置 Swit 应用

```yaml
oauth2:
  enabled: true
  provider: keycloak
  client_id: swit-app
  client_secret: <your-client-secret>
  issuer_url: http://localhost:8081/realms/swit
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

### Auth0 集成

#### 1. 创建 Auth0 应用

1. 登录 [Auth0 Dashboard](https://manage.auth0.com/)
2. Applications → Create Application
3. 选择 "Regular Web Applications"
4. 记录 Domain、Client ID、Client Secret

#### 2. 配置应用设置

- **Allowed Callback URLs**: `http://localhost:8080/callback`
- **Allowed Logout URLs**: `http://localhost:8080`
- **Allowed Web Origins**: `http://localhost:8080`

#### 3. 配置 Swit 应用

```yaml
oauth2:
  enabled: true
  provider: auth0
  client_id: <your-client-id>
  client_secret: <your-client-secret>
  issuer_url: https://<your-tenant>.auth0.com/
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

### Google 集成

#### 1. 创建 Google OAuth2 凭证

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. APIs & Services → Credentials
3. Create Credentials → OAuth client ID
4. Application type: Web application
5. Authorized redirect URIs: `http://localhost:8080/callback`

#### 2. 配置 Swit 应用

```yaml
oauth2:
  enabled: true
  provider: google
  client_id: <your-client-id>.apps.googleusercontent.com
  client_secret: <your-client-secret>
  issuer_url: https://accounts.google.com
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

### Microsoft (Azure AD) 集成

#### 1. 注册 Azure AD 应用

1. 访问 [Azure Portal](https://portal.azure.com/)
2. Azure Active Directory → App registrations → New registration
3. Redirect URI: `http://localhost:8080/callback`
4. Certificates & secrets → New client secret

#### 2. 配置 Swit 应用

```yaml
oauth2:
  enabled: true
  provider: microsoft
  client_id: <your-application-id>
  client_secret: <your-client-secret>
  issuer_url: https://login.microsoftonline.com/<tenant-id>/v2.0
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

### Okta 集成

#### 1. 创建 Okta 应用

1. 登录 [Okta Admin Console](https://your-domain.okta.com/admin)
2. Applications → Create App Integration
3. Sign-in method: OIDC
4. Application type: Web Application
5. Sign-in redirect URIs: `http://localhost:8080/callback`

#### 2. 配置 Swit 应用

```yaml
oauth2:
  enabled: true
  provider: okta
  client_id: <your-client-id>
  client_secret: <your-client-secret>
  issuer_url: https://<your-domain>.okta.com/oauth2/default
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

---

## 授权流程

Swit 框架支持多种 OAuth2 授权流程。

### 授权码流程（Authorization Code Flow）

最安全且最常用的流程，适用于 Web 应用。

```go
package main

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/security/oauth2"
)

func handleLogin(oauth2Client *oauth2.Client, flowManager *oauth2.FlowManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 生成随机 state 防止 CSRF
        state := generateRandomString(32)
        
        // 保存 state（生产环境应使用 session 或 Redis）
        flowManager.SaveState(state, "user-session-id")
        
        // 生成授权 URL
        authURL := oauth2Client.AuthCodeURL(state)
        
        c.JSON(200, gin.H{
            "authorization_url": authURL,
            "state": state,
        })
    }
}

func handleCallback(oauth2Client *oauth2.Client, flowManager *oauth2.FlowManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取授权码和 state
        code := c.Query("code")
        state := c.Query("state")
        
        // 验证 state
        if !flowManager.ValidateState(state) {
            c.JSON(400, gin.H{"error": "invalid_state"})
            return
        }
        
        // 交换授权码获取令牌
        ctx := context.Background()
        token, err := oauth2Client.Exchange(ctx, code)
        if err != nil {
            c.JSON(500, gin.H{"error": "token_exchange_failed"})
            return
        }
        
        // 返回令牌（生产环境应保存到 session）
        c.JSON(200, gin.H{
            "access_token": token.AccessToken,
            "token_type": token.TokenType,
            "expires_in": token.Expiry.Sub(time.Now()).Seconds(),
            "refresh_token": token.RefreshToken,
        })
    }
}

func generateRandomString(length int) string {
    b := make([]byte, length)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)[:length]
}
```

### 授权码流程 + PKCE

PKCE（Proof Key for Code Exchange）为授权码流程增加了额外的安全层，推荐用于公共客户端（如 SPA、移动应用）。

```go
import (
    "crypto/sha256"
    "golang.org/x/oauth2"
)

func handleLoginWithPKCE(oauth2Client *oauth2.Client, flowManager *oauth2.FlowManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 生成 PKCE 参数
        verifier := generateRandomString(64)
        challenge := generateCodeChallenge(verifier)
        
        state := generateRandomString(32)
        
        // 保存 state 和 verifier
        flowManager.SavePKCESession(state, verifier)
        
        // 生成带 PKCE 的授权 URL
        authURL := oauth2Client.AuthCodeURL(
            state,
            oauth2.SetAuthURLParam("code_challenge", challenge),
            oauth2.SetAuthURLParam("code_challenge_method", "S256"),
        )
        
        c.JSON(200, gin.H{
            "authorization_url": authURL,
            "state": state,
        })
    }
}

func generateCodeChallenge(verifier string) string {
    h := sha256.New()
    h.Write([]byte(verifier))
    return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
}
```

---

## JWT 令牌验证

Swit 支持两种令牌验证方式：本地 JWT 验证和远程令牌校验。

### 本地 JWT 验证（推荐）

本地验证速度快，适合高并发场景。

```go
import (
    "context"
    "github.com/innovationmech/swit/pkg/security/jwt"
)

func setupJWTValidator(oauth2Client *oauth2.Client) (*jwt.Validator, error) {
    // 配置 JWT 验证器
    jwtConfig := &jwt.Config{
        // 签名方法
        SigningMethod: "RS256",
        
        // JWKS URL（从 OAuth2 客户端获取）
        JWKSURL: oauth2Client.GetJWKSURL(),
        
        // 颁发者
        Issuer: oauth2Client.GetIssuerURL(),
        
        // 预期受众
        Audience: "your-client-id",
        
        // 时钟偏差容忍度
        ClockSkew: 30 * time.Second,
        
        // 必需声明
        RequiredClaims: []string{"sub", "email"},
    }
    
    // 创建验证器
    validator, err := jwt.NewValidator(jwtConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create JWT validator: %w", err)
    }
    
    return validator, nil
}
```

---

## 中间件保护

Swit 提供了开箱即用的 Gin 中间件，用于保护 HTTP 端点。

### 基本用法

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/middleware"
)

func setupRouter(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) *gin.Engine {
    router := gin.Default()
    
    // 公开端点（无需认证）
    router.GET("/public/info", handlePublicInfo)
    
    // 受保护端点（需要认证）
    protected := router.Group("/protected")
    protected.Use(middleware.OAuth2Middleware(oauth2Client, jwtValidator))
    {
        protected.GET("/profile", handleProfile)
        protected.GET("/data", handleData)
    }
    
    return router
}

func handleProfile(c *gin.Context) {
    // 从上下文获取用户信息
    userInfo, exists := middleware.GetUserInfo(c)
    if !exists {
        c.JSON(500, gin.H{"error": "user_info_not_found"})
        return
    }
    
    c.JSON(200, gin.H{
        "subject": userInfo.Subject,
        "email": userInfo.Email,
        "username": userInfo.Username,
        "roles": userInfo.Roles,
    })
}
```

---

## 常见问题解答

### Q1: 如何选择合适的 OAuth2 流程？

**A:** 根据应用类型选择：

- **Web 应用**：授权码流程（Authorization Code Flow）
- **单页应用（SPA）**：授权码流程 + PKCE
- **移动应用**：授权码流程 + PKCE
- **服务间通信**：客户端凭证流程（Client Credentials Flow）
- **受信任的应用**：密码流程（不推荐，仅在必要时使用）

### Q2: 本地 JWT 验证 vs 远程令牌校验，如何选择？

**A:** 权衡因素：

- **性能要求高**：本地 JWT 验证
- **需要即时撤销**：远程令牌校验
- **短令牌有效期（< 15分钟）**：本地 JWT 验证
- **敏感操作**：远程令牌校验

推荐：大部分请求使用本地验证，敏感操作使用远程校验。

### Q3: 如何在生产环境安全地存储 Client Secret？

**A:** 最佳实践：

1. **环境变量**:
```bash
export OAUTH2_CLIENT_SECRET=$(cat /run/secrets/oauth2_secret)
```

2. **Kubernetes Secrets**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oauth2-secret
type: Opaque
data:
  client_secret: <base64-encoded-secret>
```

3. **HashiCorp Vault**:
```go
secret, err := vaultClient.Logical().Read("secret/data/oauth2")
config.ClientSecret = secret.Data["client_secret"].(string)
```

---

## 故障排查

### 问题 1: OIDC 发现失败

**症状**：
```
oauth2: discovery failed: failed to create OIDC provider: connection refused
```

**解决方案**：

1. 确认提供商正在运行：
```bash
curl http://localhost:8081/realms/swit/.well-known/openid-configuration
```

2. 检查 issuer_url 配置是否正确
3. 检查网络连接和防火墙设置

### 问题 2: 令牌验证失败

**症状**：
```
oauth2: failed to verify ID token: unable to verify JWT signature
```

**解决方案**：

1. 检查 JWKS URL 是否可访问：
```bash
curl http://localhost:8081/realms/swit/protocol/openid-connect/certs
```

2. 验证 JWT 签名方法配置
3. 检查系统时钟同步
4. 增加时钟偏差容忍度：
```yaml
oauth2:
  jwt:
    clock_skew: 60s  # 增加到 60 秒
```

### 问题 3: Invalid Redirect URI

**症状**：
```
error: invalid_request
error_description: Invalid redirect_uri
```

**解决方案**：

1. 确保回调 URL 完全匹配（包括协议、端口、路径）
2. 在提供商管理控制台中添加重定向 URI
3. 使用通配符（仅开发环境）：
```
http://localhost:8080/*
```

---

## 相关文档

- [OAuth2/OIDC API 参考](/api/security-oauth2)
- [安全配置参考](/api/security-config)
- [安全最佳实践](/guide/security-best-practices)

## 示例代码

- [OAuth2 认证示例](https://github.com/innovationmech/swit/tree/master/examples/oauth2-authentication)
- [多提供商集成示例](https://github.com/innovationmech/swit/tree/master/examples/multi-provider-oauth2)
- [RBAC 权限控制示例](https://github.com/innovationmech/swit/tree/master/examples/oauth2-rbac)

## 外部资源

- [OAuth 2.0 规范 (RFC 6749)](https://datatracker.ietf.org/doc/html/rfc6749)
- [OIDC Core 规范](https://openid.net/specs/openid-connect-core-1_0.html)
- [PKCE 规范 (RFC 7636)](https://datatracker.ietf.org/doc/html/rfc7636)
- [JWT 规范 (RFC 7519)](https://datatracker.ietf.org/doc/html/rfc7519)
- [Keycloak 文档](https://www.keycloak.org/documentation)
- [Auth0 文档](https://auth0.com/docs)

---

**更新日期**: 2025-11-24  
**版本**: 1.0.0  
**维护者**: Swit Framework Team

