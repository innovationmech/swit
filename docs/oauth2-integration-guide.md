# OAuth2/OIDC 集成指南

本指南将帮助你在 Swit 框架中集成 OAuth2/OIDC 认证，支持多种主流提供商（Keycloak、Auth0、Google、Microsoft、Okta 等）。

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [配置指南](#配置指南)
- [多提供商集成](#多提供商集成)
- [授权流程](#授权流程)
- [JWT 令牌验证](#jwt-令牌验证)
- [中间件保护](#中间件保护)
- [高级用法](#高级用法)
- [常见问题解答](#常见问题解答)
- [故障排查](#故障排查)

---

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

### 手动端点配置（不使用 OIDC 发现）

如果提供商不支持 OIDC 发现，可以手动配置端点：

```yaml
oauth2:
  enabled: true
  provider: custom
  client_id: my-app
  client_secret: ${OAUTH2_CLIENT_SECRET}
  redirect_url: http://localhost:8080/callback
  
  # 手动端点配置
  use_discovery: false
  auth_url: https://auth.example.com/oauth2/authorize
  token_url: https://auth.example.com/oauth2/token
  user_info_url: https://auth.example.com/oauth2/userinfo
  jwks_url: https://auth.example.com/oauth2/jwks
  
  scopes:
    - openid
    - profile
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

### 自定义提供商集成

对于不在支持列表中的提供商：

```yaml
oauth2:
  enabled: true
  provider: custom
  client_id: <your-client-id>
  client_secret: <your-client-secret>
  
  # 使用 OIDC 发现（如果支持）
  issuer_url: https://custom-auth.example.com
  use_discovery: true
  
  # 或手动配置端点
  # use_discovery: false
  # auth_url: https://custom-auth.example.com/oauth2/authorize
  # token_url: https://custom-auth.example.com/oauth2/token
  # jwks_url: https://custom-auth.example.com/oauth2/jwks
  
  redirect_url: http://localhost:8080/callback
  scopes:
    - openid
    - profile
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
import "golang.org/x/oauth2"

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

func handleCallbackWithPKCE(oauth2Client *oauth2.Client, flowManager *oauth2.FlowManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        code := c.Query("code")
        state := c.Query("state")
        
        // 获取保存的 verifier
        verifier, ok := flowManager.GetPKCEVerifier(state)
        if !ok {
            c.JSON(400, gin.H{"error": "invalid_session"})
            return
        }
        
        // 交换授权码（包含 code_verifier）
        ctx := context.Background()
        token, err := oauth2Client.GetConfig().Exchange(
            ctx,
            code,
            oauth2.SetAuthURLParam("code_verifier", verifier),
        )
        if err != nil {
            c.JSON(500, gin.H{"error": "token_exchange_failed"})
            return
        }
        
        c.JSON(200, gin.H{"access_token": token.AccessToken})
    }
}

func generateCodeChallenge(verifier string) string {
    h := sha256.New()
    h.Write([]byte(verifier))
    return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
}
```

### 客户端凭证流程（Client Credentials Flow）

适用于服务间通信（Machine-to-Machine）。

```go
import (
    "context"
    "golang.org/x/oauth2/clientcredentials"
)

func getClientCredentialsToken(oauth2Client *oauth2.Client) (*oauth2.Token, error) {
    ctx := context.Background()
    
    // 配置客户端凭证流程
    config := &clientcredentials.Config{
        ClientID:     oauth2Client.GetConfig().ClientID,
        ClientSecret: oauth2Client.GetConfig().ClientSecret,
        TokenURL:     oauth2Client.GetConfig().Endpoint.TokenURL,
        Scopes:       []string{"api.read", "api.write"},
    }
    
    // 获取令牌
    token, err := config.Token(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get client credentials token: %w", err)
    }
    
    return token, nil
}
```

### 刷新令牌流程（Refresh Token Flow）

自动刷新过期的访问令牌。

```go
func handleRefreshToken(oauth2Client *oauth2.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req struct {
            RefreshToken string `json:"refresh_token" binding:"required"`
        }
        
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": "invalid_request"})
            return
        }
        
        // 创建旧令牌对象
        oldToken := &oauth2.Token{
            RefreshToken: req.RefreshToken,
        }
        
        // 刷新令牌
        ctx := context.Background()
        newToken, err := oauth2Client.RefreshToken(ctx, oldToken)
        if err != nil {
            c.JSON(401, gin.H{"error": "refresh_failed"})
            return
        }
        
        c.JSON(200, gin.H{
            "access_token": newToken.AccessToken,
            "token_type": newToken.TokenType,
            "expires_in": newToken.Expiry.Sub(time.Now()).Seconds(),
            "refresh_token": newToken.RefreshToken,
        })
    }
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

func validateTokenLocally(validator *jwt.Validator, tokenString string) error {
    ctx := context.Background()
    
    // 验证令牌
    token, err := validator.ValidateToken(ctx, tokenString)
    if err != nil {
        return fmt.Errorf("token validation failed: %w", err)
    }
    
    // 提取声明
    claims, err := validator.ExtractClaims(token)
    if err != nil {
        return fmt.Errorf("failed to extract claims: %w", err)
    }
    
    // 访问声明
    subject := claims["sub"].(string)
    email := claims["email"].(string)
    
    fmt.Printf("User: %s (%s)\n", subject, email)
    
    return nil
}
```

### 远程令牌校验（Token Introspection）

通过提供商的 introspection 端点验证令牌。

```go
func validateTokenRemotely(oauth2Client *oauth2.Client, tokenString string) error {
    ctx := context.Background()
    
    // 调用令牌校验端点
    introspection, err := oauth2Client.IntrospectToken(ctx, tokenString)
    if err != nil {
        return fmt.Errorf("token introspection failed: %w", err)
    }
    
    // 检查令牌是否有效
    if !introspection.Active {
        return fmt.Errorf("token is not active")
    }
    
    // 访问令牌元数据
    fmt.Printf("Token is valid. Subject: %s\n", introspection.Subject)
    fmt.Printf("Scopes: %v\n", introspection.Scope)
    fmt.Printf("Expires at: %v\n", introspection.ExpiresAt)
    
    return nil
}
```

### 验证策略选择

| 验证方式 | 优点 | 缺点 | 适用场景 |
|---------|------|------|---------|
| 本地 JWT 验证 | 快速、不依赖网络 | 无法即时撤销 | 高并发、短令牌有效期 |
| 远程令牌校验 | 即时撤销、获取最新状态 | 网络延迟、提供商负载 | 需要即时撤销、敏感操作 |

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

### 自定义中间件配置

```go
func setupAdvancedMiddleware(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) gin.HandlerFunc {
    config := &middleware.OAuth2MiddlewareConfig{
        OAuth2Client: oauth2Client,
        JWTValidator: jwtValidator,
        
        // 使用远程令牌校验而非本地验证
        UseIntrospection: true,
        
        // 跳过认证的路径
        SkipPaths: []string{
            "/health",
            "/metrics",
        },
        
        // 从 Cookie 读取令牌
        CookieName: "access_token",
        
        // 自定义错误处理
        ErrorHandler: func(c *gin.Context, err error) {
            c.JSON(401, gin.H{
                "error": "unauthorized",
                "message": err.Error(),
            })
            c.Abort()
        },
        
        // 自定义令牌提取
        TokenExtractor: func(c *gin.Context) (string, error) {
            // 优先从 Header 提取
            if token := c.GetHeader("Authorization"); token != "" {
                return strings.TrimPrefix(token, "Bearer "), nil
            }
            // 其次从 Cookie 提取
            if token, err := c.Cookie("access_token"); err == nil {
                return token, nil
            }
            return "", fmt.Errorf("no token found")
        },
    }
    
    return middleware.OAuth2MiddlewareWithConfig(config)
}
```

### 可选认证

允许匿名和认证用户同时访问：

```go
func setupOptionalAuth(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) *gin.Engine {
    router := gin.Default()
    
    // 可选认证端点
    optional := router.Group("/optional")
    optional.Use(middleware.OptionalAuth(oauth2Client, jwtValidator))
    {
        optional.GET("/content", handleOptionalContent)
    }
    
    return router
}

func handleOptionalContent(c *gin.Context) {
    userInfo, authenticated := middleware.GetUserInfo(c)
    
    if authenticated {
        // 为认证用户提供个性化内容
        c.JSON(200, gin.H{
            "message": fmt.Sprintf("Welcome back, %s!", userInfo.Username),
            "premium": true,
        })
    } else {
        // 为匿名用户提供基础内容
        c.JSON(200, gin.H{
            "message": "Welcome! Please login for more features.",
            "premium": false,
        })
    }
}
```

### 角色权限控制

结合角色信息进行访问控制：

```go
// 角色检查中间件
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userInfo, exists := middleware.GetUserInfo(c)
        if !exists {
            c.JSON(403, gin.H{"error": "forbidden"})
            c.Abort()
            return
        }
        
        // 检查用户是否拥有所需角色
        hasRole := false
        for _, requiredRole := range roles {
            for _, userRole := range userInfo.Roles {
                if userRole == requiredRole {
                    hasRole = true
                    break
                }
            }
        }
        
        if !hasRole {
            c.JSON(403, gin.H{
                "error": "insufficient_permissions",
                "required_roles": roles,
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// 使用示例
func setupRoleBasedAccess(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) *gin.Engine {
    router := gin.Default()
    
    // 管理员端点
    admin := router.Group("/admin")
    admin.Use(middleware.OAuth2Middleware(oauth2Client, jwtValidator))
    admin.Use(RequireRole("admin"))
    {
        admin.GET("/dashboard", handleAdminDashboard)
        admin.POST("/users", handleCreateUser)
    }
    
    return router
}
```

---

## 高级用法

### 令牌缓存

启用令牌缓存以提升性能：

```yaml
oauth2:
  cache:
    enabled: true
    type: memory  # 或 redis（需要额外配置）
    ttl: 10m
    max_size: 1000
    cleanup_interval: 5m
```

代码中使用缓存：

```go
import "github.com/innovationmech/swit/pkg/security/oauth2"

func setupCachedValidation() {
    // 令牌缓存会自动生效
    // 验证过的令牌会被缓存，避免重复验证
    
    // 缓存命中时验证速度可提升 100 倍以上
}
```

### 令牌撤销

实现登出功能：

```go
func handleLogout(oauth2Client *oauth2.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取令牌
        tokenString := c.GetHeader("Authorization")
        tokenString = strings.TrimPrefix(tokenString, "Bearer ")
        
        // 撤销访问令牌
        ctx := context.Background()
        if err := oauth2Client.RevokeToken(ctx, tokenString); err != nil {
            c.JSON(500, gin.H{"error": "revocation_failed"})
            return
        }
        
        // 清除客户端 Cookie
        c.SetCookie("access_token", "", -1, "/", "", false, true)
        
        c.JSON(200, gin.H{"message": "logged_out"})
    }
}
```

### 健康检查

集成 OAuth2 提供商健康检查：

```go
import "github.com/innovationmech/swit/pkg/server"

type OAuth2HealthCheck struct {
    oauth2Client *oauth2.Client
}

func (h *OAuth2HealthCheck) GetName() string {
    return "oauth2-provider"
}

func (h *OAuth2HealthCheck) CheckHealth(ctx context.Context) server.HealthCheckResult {
    // 测试与提供商的连接
    _, err := h.oauth2Client.GetMetadata()
    
    if err != nil {
        return server.HealthCheckResult{
            Status:  "unhealthy",
            Message: fmt.Sprintf("OAuth2 provider unreachable: %v", err),
        }
    }
    
    return server.HealthCheckResult{
        Status:  "healthy",
        Message: "OAuth2 provider is accessible",
    }
}
```

### TLS/mTLS 配置

生产环境建议启用 TLS：

```yaml
oauth2:
  tls:
    enabled: true
    insecure_skip_verify: false  # 生产环境必须 false
    
    # 客户端证书（mTLS）
    cert_file: /path/to/client-cert.pem
    key_file: /path/to/client-key.pem
    
    # CA 证书
    ca_file: /path/to/ca-cert.pem
    
    # TLS 版本
    min_version: TLS1.2
    max_version: TLS1.3
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

1. **环境变量**：
```bash
export OAUTH2_CLIENT_SECRET=$(cat /run/secrets/oauth2_secret)
```

2. **Kubernetes Secrets**：
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oauth2-secret
type: Opaque
data:
  client_secret: <base64-encoded-secret>
```

3. **HashiCorp Vault**：
```go
secret, err := vaultClient.Logical().Read("secret/data/oauth2")
config.ClientSecret = secret.Data["client_secret"].(string)
```

### Q4: 访问令牌应该存储在哪里？

**A:** 安全存储建议：

- **Web 应用（服务端渲染）**：HTTP-only Cookie
- **SPA**：内存中（不要存储在 localStorage）
- **移动应用**：系统安全存储（Keychain/KeyStore）
- **服务端**：内存或 Redis（加密存储）

### Q5: 如何处理令牌过期？

**A:** 自动刷新示例：

```go
func autoRefreshToken(oauth2Client *oauth2.Client, token *oauth2.Token) *oauth2.Token {
    // 检查令牌是否即将过期（5分钟内）
    if oauth2.IsTokenExpiring(token, 5*time.Minute) {
        ctx := context.Background()
        newToken, err := oauth2Client.RefreshToken(ctx, token)
        if err != nil {
            log.Printf("Failed to refresh token: %v", err)
            return token
        }
        return newToken
    }
    return token
}
```

### Q6: 如何获取用户的角色和权限？

**A:** 从 JWT 声明中提取：

```go
func getUserRoles(c *gin.Context, oauth2Config *oauth2.Config) []string {
    claims, _ := middleware.GetClaims(c)
    
    // 不同提供商的角色声明路径不同
    rolesClaimName := oauth2Config.GetRolesClaimName()
    
    // Keycloak: realm_access.roles
    if rolesClaimName == "realm_access.roles" {
        if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
            if roles, ok := realmAccess["roles"].([]interface{}); ok {
                result := make([]string, len(roles))
                for i, role := range roles {
                    result[i] = role.(string)
                }
                return result
            }
        }
    }
    
    // Auth0/Okta: roles
    if rolesInterface, ok := claims["roles"].([]interface{}); ok {
        result := make([]string, len(rolesInterface))
        for i, role := range rolesInterface {
            result[i] = role.(string)
        }
        return result
    }
    
    return []string{}
}
```

### Q7: 如何测试 OAuth2 集成？

**A:** 使用模拟提供商：

```go
import "github.com/innovationmech/swit/pkg/security/oauth2"

func TestOAuth2Integration(t *testing.T) {
    // 使用测试配置
    config := &oauth2.Config{
        Enabled: true,
        Provider: "custom",
        ClientID: "test-client",
        ClientSecret: "test-secret",
        UseDiscovery: false,
        SkipIssuerVerification: true,
        SkipExpiryCheck: true,
        // 手动配置测试端点
        AuthURL: "http://localhost:9999/auth",
        TokenURL: "http://localhost:9999/token",
        JWKSURL: "http://localhost:9999/jwks",
    }
    
    // 创建客户端
    ctx := context.Background()
    client, err := oauth2.NewClient(ctx, config)
    assert.NoError(t, err)
    
    // 测试授权 URL 生成
    authURL := client.AuthCodeURL("test-state")
    assert.Contains(t, authURL, "test-client")
}
```

### Q8: 如何实现单点登出（Single Logout）？

**A:** 实现示例：

```go
func handleSingleLogout(oauth2Client *oauth2.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 撤销访问令牌
        accessToken := c.GetHeader("Authorization")
        accessToken = strings.TrimPrefix(accessToken, "Bearer ")
        oauth2Client.RevokeToken(c.Request.Context(), accessToken)
        
        // 2. 撤销刷新令牌
        refreshToken := c.PostForm("refresh_token")
        if refreshToken != "" {
            oauth2Client.RevokeToken(c.Request.Context(), refreshToken)
        }
        
        // 3. 重定向到提供商的登出端点
        logoutURL := fmt.Sprintf(
            "%s/protocol/openid-connect/logout?redirect_uri=%s",
            oauth2Client.GetIssuerURL(),
            url.QueryEscape("http://localhost:8080"),
        )
        
        c.Redirect(302, logoutURL)
    }
}
```

---

## 故障排查

### 问题 1: OIDC 发现失败

**症状**：
```
oauth2: discovery failed: failed to create OIDC provider: Get "http://localhost:8081/realms/swit/.well-known/openid-configuration": dial tcp 127.0.0.1:8081: connect: connection refused
```

**解决方案**：

1. 确认提供商正在运行：
```bash
curl http://localhost:8081/realms/swit/.well-known/openid-configuration
```

2. 检查 issuer_url 配置是否正确

3. 检查网络连接和防火墙设置

4. 如果使用 Docker，确保网络配置正确：
```yaml
# docker-compose.yml
networks:
  default:
    name: oauth2-network
```

### 问题 2: 令牌验证失败

**症状**：
```
oauth2: failed to verify ID token: oidc: unable to verify JWT signature: failed to verify signature: failed to get key
```

**解决方案**：

1. 检查 JWKS URL 是否可访问：
```bash
curl http://localhost:8081/realms/swit/protocol/openid-connect/certs
```

2. 验证 JWT 签名方法配置：
```yaml
oauth2:
  jwt:
    signing_method: RS256  # 确保与提供商一致
```

3. 检查系统时钟同步：
```bash
date
# 如果时间不准确，同步时钟
sudo ntpdate time.apple.com
```

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

1. 确保回调 URL 完全匹配（包括协议、端口、路径）：
```yaml
oauth2:
  redirect_url: http://localhost:8080/callback  # 必须与提供商配置完全一致
```

2. 在提供商管理控制台中添加重定向 URI：

**Keycloak**:
- Clients → Your Client → Valid Redirect URIs
- 添加: `http://localhost:8080/*`

**Auth0**:
- Applications → Your App → Allowed Callback URLs
- 添加: `http://localhost:8080/callback`

3. 使用通配符（仅开发环境）：
```
http://localhost:8080/*
http://localhost:*/callback
```

### 问题 4: Token Exchange 失败

**症状**：
```
oauth2: token_exchange_failed: oauth2: cannot fetch token: 400 Bad Request
```

**解决方案**：

1. 验证 client_id 和 client_secret：
```bash
# 测试客户端凭证
curl -X POST http://localhost:8081/realms/swit/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=your-client-id" \
  -d "client_secret=your-client-secret"
```

2. 检查授权码是否过期（授权码通常只有 60 秒有效期）

3. 确保授权码未被重复使用（授权码只能使用一次）

4. 查看详细错误信息：
```go
token, err := oauth2Client.Exchange(ctx, code)
if err != nil {
    log.Printf("Exchange failed: %+v", err)
}
```

### 问题 5: 角色/权限未包含在令牌中

**症状**：
```
userInfo.Roles is empty
```

**解决方案**：

1. **Keycloak** - 配置 Client Scopes：
   - Clients → Your Client → Client Scopes
   - 添加 "roles" scope
   - Client Scopes → roles → Mappers
   - 确保 "realm roles" mapper 已启用

2. **Auth0** - 配置 Actions：
```javascript
exports.onExecutePostLogin = async (event, api) => {
  const namespace = 'https://your-app.com';
  if (event.authorization) {
    api.idToken.setCustomClaim(`${namespace}/roles`, event.authorization.roles);
  }
};
```

3. 请求正确的作用域：
```yaml
oauth2:
  scopes:
    - openid
    - profile
    - email
    - roles  # 确保包含 roles
```

### 问题 6: 生产环境 TLS 错误

**症状**：
```
x509: certificate signed by unknown authority
```

**解决方案**：

1. 安装 CA 证书：
```yaml
oauth2:
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-certificates.crt
```

2. 更新系统 CA 证书：
```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install ca-certificates
sudo update-ca-certificates

# CentOS/RHEL
sudo yum install ca-certificates
sudo update-ca-trust
```

3. 仅在开发环境跳过验证（不要在生产环境使用）：
```yaml
oauth2:
  tls:
    insecure_skip_verify: true  # 仅开发环境！
```

### 问题 7: 高并发下性能问题

**症状**：
- 响应时间慢
- OAuth2 提供商负载高

**解决方案**：

1. 启用令牌缓存：
```yaml
oauth2:
  cache:
    enabled: true
    ttl: 10m
    max_size: 10000
```

2. 使用本地 JWT 验证而非远程校验：
```go
config := &middleware.OAuth2MiddlewareConfig{
    OAuth2Client: oauth2Client,
    JWTValidator: jwtValidator,
    UseIntrospection: false,  // 使用本地验证
}
```

3. 启用连接池优化：
```yaml
oauth2:
  http_timeout: 30s
  # 框架已内置连接池优化
```

4. 使用短期令牌和自动刷新：
```yaml
oauth2:
  jwt:
    # 令牌有效期设置为 5-15 分钟
    # 配置在提供商端
```

### 问题 8: Docker 容器间网络问题

**症状**：
```
connect: connection refused
```

**解决方案**：

1. 使用 Docker 服务名而非 localhost：
```yaml
# 服务配置（容器内部）
oauth2:
  issuer_url: http://keycloak:8080/realms/swit  # 使用服务名
  
# 浏览器访问（容器外部）
# authorization_url 应该使用: http://localhost:8081/realms/swit
```

2. 配置 Docker Compose 网络：
```yaml
version: '3.8'
services:
  keycloak:
    image: quay.io/keycloak/keycloak
    networks:
      - oauth2-net
    ports:
      - "8081:8080"
  
  app:
    build: .
    networks:
      - oauth2-net
    environment:
      OAUTH2_ISSUER_URL: http://keycloak:8080/realms/swit

networks:
  oauth2-net:
    driver: bridge
```

### 调试技巧

#### 启用详细日志

```go
import "go.uber.org/zap"

// 配置 zap logger 为 Debug 级别
logger, _ := zap.NewDevelopment()
defer logger.Sync()

// 在 OAuth2 客户端创建前设置
oauth2.SetLogger(logger)
```

#### 使用 HTTP 代理抓包

```bash
# 使用 mitmproxy 或 Charles Proxy
export HTTP_PROXY=http://localhost:8888
export HTTPS_PROXY=http://localhost:8888

# 运行应用
go run main.go
```

#### 测试 OIDC 端点

```bash
# 获取 OIDC 配置
curl http://localhost:8081/realms/swit/.well-known/openid-configuration | jq

# 测试授权端点
curl "http://localhost:8081/realms/swit/protocol/openid-connect/auth?client_id=test&redirect_uri=http://localhost:8080/callback&response_type=code&scope=openid"

# 测试 JWKS 端点
curl http://localhost:8081/realms/swit/protocol/openid-connect/certs | jq
```

---

## 相关文档

- [OAuth2/OIDC API 参考](./api/security-oauth2.md)
- [安全配置参考](./api/security-config.md)
- [安全最佳实践](./security-best-practices.md)
- [JWT 令牌管理指南](./pages/zh/guide/jwt-management.md)

## 示例代码

- [OAuth2 认证示例](../examples/oauth2-authentication/)
- [多提供商集成示例](../examples/multi-provider-oauth2/)
- [RBAC 权限控制示例](../examples/oauth2-rbac/)

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

