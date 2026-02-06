# OAuth2/OIDC API 参考

本文档提供 Swit 框架中 OAuth2/OIDC 安全功能的完整 API 参考，包括配置、客户端、流程和最佳实践。

## 目录

- [概述](#概述)
- [配置 API](#配置-api)
- [客户端 API](#客户端-api)
- [OAuth2 流程](#oauth2-流程)
- [令牌管理](#令牌管理)
- [提供商集成](#提供商集成)
- [缓存管理](#缓存管理)
- [错误处理](#错误处理)
- [代码示例](#代码示例)
- [最佳实践](#最佳实践)

## 概述

OAuth2/OIDC 包 (`pkg/security/oauth2`) 为 Swit 框架提供了企业级的身份认证和授权功能。主要特性包括：

- ✅ **多提供商支持** - Keycloak、Auth0、Google、Microsoft、Okta 等
- ✅ **OIDC 自动发现** - 自动从 issuer URL 发现端点
- ✅ **多种授权流程** - 授权码、客户端凭证、密码、刷新令牌
- ✅ **JWT 令牌验证** - 完整的 JWT 签名和声明验证
- ✅ **令牌缓存** - 内置令牌缓存机制提升性能
- ✅ **TLS 支持** - 完整的 TLS/mTLS 配置
- ✅ **健康检查** - 内置提供商连接健康检查

### 支持的提供商

| 提供商 | 类型常量 | OIDC 发现 | 默认作用域 |
|--------|----------|----------|-----------|
| Keycloak | `ProviderKeycloak` | ✅ | openid, profile, email |
| Auth0 | `ProviderAuth0` | ✅ | openid, profile, email |
| Google | `ProviderGoogle` | ✅ | openid, profile, email |
| Microsoft | `ProviderMicrosoft` | ✅ | openid, profile, email |
| Okta | `ProviderOkta` | ✅ | openid, profile, email |
| 自定义 | `ProviderCustom` | 可选 | 自定义 |

---

## 配置 API

### Config 结构体

`Config` 是 OAuth2/OIDC 客户端的主要配置结构。

```go
type Config struct {
    // Enabled 表示是否启用 OAuth2 认证
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // Provider 是 OAuth2 提供商名称
    // 支持: "keycloak", "auth0", "google", "microsoft", "okta", "custom"
    Provider string `json:"provider" yaml:"provider" mapstructure:"provider"`
    
    // ClientID 是 OAuth2 客户端标识符
    ClientID string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
    
    // ClientSecret 是 OAuth2 客户端密钥
    ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
    
    // RedirectURL 是授权码流程的重定向 URL
    RedirectURL string `json:"redirect_url" yaml:"redirect_url" mapstructure:"redirect_url"`
    
    // Scopes 是请求的 OAuth2 作用域列表
    Scopes []string `json:"scopes" yaml:"scopes" mapstructure:"scopes"`
    
    // IssuerURL 是 OIDC 颁发者 URL（用于发现）
    IssuerURL string `json:"issuer_url" yaml:"issuer_url" mapstructure:"issuer_url"`
    
    // AuthURL 是 OAuth2 授权端点 URL（可选，通过发现获取）
    AuthURL string `json:"auth_url,omitempty" yaml:"auth_url,omitempty" mapstructure:"auth_url"`
    
    // TokenURL 是 OAuth2 令牌端点 URL（可选，通过发现获取）
    TokenURL string `json:"token_url,omitempty" yaml:"token_url,omitempty" mapstructure:"token_url"`
    
    // UserInfoURL 是 OIDC 用户信息端点 URL（可选）
    UserInfoURL string `json:"user_info_url,omitempty" yaml:"user_info_url,omitempty" mapstructure:"user_info_url"`
    
    // JWKSURL 是 JSON Web Key Set URL（用于令牌验证）
    JWKSURL string `json:"jwks_url,omitempty" yaml:"jwks_url,omitempty" mapstructure:"jwks_url"`
    
    // UseDiscovery 表示是否使用 OIDC 自动发现
    UseDiscovery bool `json:"use_discovery" yaml:"use_discovery" mapstructure:"use_discovery"`
    
    // HTTPTimeout 是 OAuth2 请求的 HTTP 客户端超时
    HTTPTimeout time.Duration `json:"http_timeout" yaml:"http_timeout" mapstructure:"http_timeout"`
    
    // SkipIssuerVerification 跳过颁发者验证（仅用于测试）
    SkipIssuerVerification bool `json:"skip_issuer_verification,omitempty" yaml:"skip_issuer_verification,omitempty" mapstructure:"skip_issuer_verification"`
    
    // SkipExpiryCheck 跳过令牌过期检查（仅用于测试）
    SkipExpiryCheck bool `json:"skip_expiry_check,omitempty" yaml:"skip_expiry_check,omitempty" mapstructure:"skip_expiry_check"`
    
    // JWTConfig 是 JWT 令牌验证配置
    JWTConfig JWTConfig `json:"jwt" yaml:"jwt" mapstructure:"jwt"`
    
    // CacheConfig 是令牌缓存配置
    CacheConfig TokenCacheConfig `json:"cache" yaml:"cache" mapstructure:"cache"`
    
    // TLSConfig 是 OAuth2 连接的 TLS 配置
    TLSConfig TLSConfig `json:"tls" yaml:"tls" mapstructure:"tls"`
}
```

#### 配置方法

##### SetDefaults()

设置配置的默认值。

```go
func (c *Config) SetDefaults()
```

**示例：**

```go
config := &oauth2.Config{
    Provider:  "keycloak",
    ClientID:  "my-client",
    IssuerURL: "https://auth.example.com/realms/myrealm",
}
config.SetDefaults()
// 现在 HTTPTimeout = 30s, UseDiscovery = true (对于 Keycloak)
```

##### Validate()

验证配置的有效性。

```go
func (c *Config) Validate() error
```

**返回值：**
- `error` - 如果配置无效，返回描述性错误

**验证规则：**
- `ClientID` 不能为空
- `IssuerURL` 不能为空（如果 `UseDiscovery` 为 true）
- 如果不使用发现，必须设置 `AuthURL` 和 `TokenURL`

**示例：**

```go
config := &oauth2.Config{
    Provider:  "keycloak",
    ClientID:  "my-client",
    IssuerURL: "https://auth.example.com/realms/myrealm",
}
if err := config.Validate(); err != nil {
    log.Fatalf("无效的配置: %v", err)
}
```

##### ApplyProviderDefaults()

根据提供商类型应用特定的默认值。

```go
func (c *Config) ApplyProviderDefaults() error
```

**示例：**

```go
config := &oauth2.Config{
    Provider: "keycloak",
    ClientID: "my-client",
}
config.ApplyProviderDefaults()
// 自动设置: UseDiscovery=true, Scopes=["openid","profile","email"]
```

##### LoadFromEnv()

从环境变量加载配置。

```go
func (c *Config) LoadFromEnv()
```

**支持的环境变量：**
- `OAUTH2_ENABLED` - 启用 OAuth2
- `OAUTH2_PROVIDER` - 提供商类型
- `OAUTH2_CLIENT_ID` - 客户端 ID
- `OAUTH2_CLIENT_SECRET` - 客户端密钥
- `OAUTH2_ISSUER_URL` - 颁发者 URL
- `OAUTH2_REDIRECT_URL` - 重定向 URL
- `OAUTH2_SCOPES` - 作用域（逗号分隔）
- `OAUTH2_USE_DISCOVERY` - 启用 OIDC 发现

**示例：**

```bash
export OAUTH2_ENABLED=true
export OAUTH2_PROVIDER=keycloak
export OAUTH2_CLIENT_ID=my-client
export OAUTH2_CLIENT_SECRET=secret
export OAUTH2_ISSUER_URL=https://auth.example.com/realms/myrealm
```

```go
config := &oauth2.Config{}
config.LoadFromEnv()
```

---

### JWTConfig 结构体

JWT 令牌验证配置。

```go
type JWTConfig struct {
    // SigningMethod 是预期的 JWT 签名方法
    // 支持: "RS256", "RS384", "RS512", "HS256", "HS384", "HS512", "ES256", "ES384", "ES512"
    SigningMethod string `json:"signing_method" yaml:"signing_method" mapstructure:"signing_method"`
    
    // ClockSkew 是验证时间相关声明时容忍的时间偏差
    ClockSkew time.Duration `json:"clock_skew" yaml:"clock_skew" mapstructure:"clock_skew"`
    
    // SkipClaimsValidation 跳过标准声明验证（iss, aud, exp, nbf, iat）
    // 仅用于测试目的
    SkipClaimsValidation bool `json:"skip_claims_validation,omitempty" yaml:"skip_claims_validation,omitempty" mapstructure:"skip_claims_validation"`
    
    // RequiredClaims 是令牌中必须存在的声明列表
    RequiredClaims []string `json:"required_claims,omitempty" yaml:"required_claims,omitempty" mapstructure:"required_claims"`
    
    // Audience 是预期的 audience 声明值
    Audience string `json:"audience,omitempty" yaml:"audience,omitempty" mapstructure:"audience"`
}
```

**默认值：**
- `SigningMethod`: "RS256"
- `ClockSkew`: 5 分钟

**示例：**

```yaml
jwt:
  signing_method: RS256
  clock_skew: 5m
  audience: my-api
  required_claims:
    - sub
    - email
```

---

### TokenCacheConfig 结构体

令牌缓存配置。

```go
type TokenCacheConfig struct {
    // Enabled 表示是否启用令牌缓存
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // MaxSize 是缓存中的最大令牌数量
    MaxSize int `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
    
    // TTL 是缓存条目的生存时间
    TTL time.Duration `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
    
    // CleanupInterval 是缓存清理间隔
    CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" mapstructure:"cleanup_interval"`
}
```

**默认值：**
- `Enabled`: true
- `MaxSize`: 1000
- `TTL`: 10 分钟
- `CleanupInterval`: 5 分钟

**示例：**

```yaml
cache:
  enabled: true
  max_size: 5000
  ttl: 15m
  cleanup_interval: 10m
```

---

### TLSConfig 结构体

OAuth2 连接的 TLS 配置。

```go
type TLSConfig struct {
    // Enabled 表示是否启用 TLS
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // CertFile 是客户端证书文件路径
    CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty" mapstructure:"cert_file"`
    
    // KeyFile 是客户端私钥文件路径
    KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty" mapstructure:"key_file"`
    
    // CAFile 是 CA 证书文件路径
    CAFile string `json:"ca_file,omitempty" yaml:"ca_file,omitempty" mapstructure:"ca_file"`
    
    // InsecureSkipVerify 跳过服务器证书验证（不推荐用于生产）
    InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`
}
```

**示例：**

```yaml
tls:
  enabled: true
  cert_file: /path/to/client-cert.pem
  key_file: /path/to/client-key.pem
  ca_file: /path/to/ca-cert.pem
```

---

## 客户端 API

### Client 结构体

OAuth2/OIDC 客户端。

```go
type Client struct {
    // 内部字段（不导出）
}
```

### 创建客户端

#### NewClient()

创建新的 OAuth2/OIDC 客户端。

```go
func NewClient(ctx context.Context, cfg *Config) (*Client, error)
```

**参数：**
- `ctx` - 上下文
- `cfg` - OAuth2 配置

**返回值：**
- `*Client` - OAuth2 客户端实例
- `error` - 如果初始化失败

**示例：**

```go
ctx := context.Background()
config := &oauth2.Config{
    Provider:     "keycloak",
    ClientID:     "my-client",
    ClientSecret: "my-secret",
    IssuerURL:    "https://auth.example.com/realms/myrealm",
    RedirectURL:  "http://localhost:8080/callback",
    UseDiscovery: true,
}

client, err := oauth2.NewClient(ctx, config)
if err != nil {
    log.Fatalf("创建客户端失败: %v", err)
}
defer client.Close(ctx)
```

### 客户端方法

#### GetAuthorizationURL()

获取授权 URL（用于授权码流程）。

```go
func (c *Client) GetAuthorizationURL(state string, opts ...oauth2.AuthCodeOption) string
```

**参数：**
- `state` - 状态参数（用于 CSRF 保护）
- `opts` - 可选的授权选项

**返回值：**
- `string` - 授权 URL

**示例：**

```go
state := "random-state-value"
authURL := client.GetAuthorizationURL(state)
// 重定向用户到 authURL
http.Redirect(w, r, authURL, http.StatusFound)
```

#### ExchangeAuthorizationCode()

交换授权码以获取访问令牌。

```go
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*TokenResponse, error)
```

**参数：**
- `ctx` - 上下文
- `code` - 授权码
- `opts` - 可选的交换选项

**返回值：**
- `*TokenResponse` - 令牌响应
- `error` - 如果交换失败

**示例：**

```go
code := r.URL.Query().Get("code")
tokenResp, err := client.ExchangeAuthorizationCode(ctx, code)
if err != nil {
    log.Printf("交换授权码失败: %v", err)
    return
}

log.Printf("访问令牌: %s", tokenResp.AccessToken)
log.Printf("刷新令牌: %s", tokenResp.RefreshToken)
log.Printf("ID 令牌: %s", tokenResp.IDToken)
```

#### ClientCredentials()

使用客户端凭证流程获取访问令牌。

```go
func (c *Client) ClientCredentials(ctx context.Context, scopes ...string) (*TokenResponse, error)
```

**参数：**
- `ctx` - 上下文
- `scopes` - 请求的作用域（可选）

**返回值：**
- `*TokenResponse` - 令牌响应
- `error` - 如果请求失败

**示例：**

```go
tokenResp, err := client.ClientCredentials(ctx, "read:data", "write:data")
if err != nil {
    log.Fatalf("获取客户端凭证失败: %v", err)
}

log.Printf("访问令牌: %s", tokenResp.AccessToken)
```

#### PasswordCredentials()

使用密码凭证流程获取访问令牌。

```go
func (c *Client) PasswordCredentials(ctx context.Context, username, password string, scopes ...string) (*TokenResponse, error)
```

**参数：**
- `ctx` - 上下文
- `username` - 用户名
- `password` - 密码
- `scopes` - 请求的作用域（可选）

**返回值：**
- `*TokenResponse` - 令牌响应
- `error` - 如果认证失败

**⚠️ 警告：** 密码流程不安全，仅在可信的第一方应用中使用。

**示例：**

```go
tokenResp, err := client.PasswordCredentials(ctx, "alice", "password123")
if err != nil {
    log.Printf("密码认证失败: %v", err)
    return
}

log.Printf("访问令牌: %s", tokenResp.AccessToken)
```

#### RefreshToken()

使用刷新令牌获取新的访问令牌。

```go
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
```

**参数：**
- `ctx` - 上下文
- `refreshToken` - 刷新令牌

**返回值：**
- `*TokenResponse` - 新的令牌响应
- `error` - 如果刷新失败

**示例：**

```go
newTokenResp, err := client.RefreshToken(ctx, oldTokenResp.RefreshToken)
if err != nil {
    log.Printf("刷新令牌失败: %v", err)
    return
}

log.Printf("新访问令牌: %s", newTokenResp.AccessToken)
```

#### ValidateToken()

验证访问令牌的有效性。

```go
func (c *Client) ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
```

**参数：**
- `ctx` - 上下文
- `token` - 访问令牌

**返回值：**
- `*TokenClaims` - 令牌声明
- `error` - 如果验证失败

**示例：**

```go
claims, err := client.ValidateToken(ctx, accessToken)
if err != nil {
    log.Printf("令牌验证失败: %v", err)
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}

log.Printf("用户 ID: %s", claims.Subject)
log.Printf("电子邮件: %s", claims.Email)
```

#### VerifyIDToken()

验证 OIDC ID 令牌。

```go
func (c *Client) VerifyIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error)
```

**参数：**
- `ctx` - 上下文
- `idToken` - ID 令牌

**返回值：**
- `*IDTokenClaims` - ID 令牌声明
- `error` - 如果验证失败

**示例：**

```go
idClaims, err := client.VerifyIDToken(ctx, tokenResp.IDToken)
if err != nil {
    log.Printf("ID 令牌验证失败: %v", err)
    return
}

log.Printf("用户 ID: %s", idClaims.Subject)
log.Printf("姓名: %s", idClaims.Name)
log.Printf("电子邮件: %s", idClaims.Email)
```

#### GetUserInfo()

获取用户信息（通过 OIDC UserInfo 端点）。

```go
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
```

**参数：**
- `ctx` - 上下文
- `accessToken` - 访问令牌

**返回值：**
- `*UserInfo` - 用户信息
- `error` - 如果请求失败

**示例：**

```go
userInfo, err := client.GetUserInfo(ctx, accessToken)
if err != nil {
    log.Printf("获取用户信息失败: %v", err)
    return
}

log.Printf("用户 ID: %s", userInfo.Sub)
log.Printf("姓名: %s", userInfo.Name)
log.Printf("电子邮件: %s", userInfo.Email)
```

#### RevokeToken()

撤销令牌。

```go
func (c *Client) RevokeToken(ctx context.Context, token string, tokenType string) error
```

**参数：**
- `ctx` - 上下文
- `token` - 要撤销的令牌
- `tokenType` - 令牌类型（"access_token" 或 "refresh_token"）

**返回值：**
- `error` - 如果撤销失败

**示例：**

```go
err := client.RevokeToken(ctx, refreshToken, "refresh_token")
if err != nil {
    log.Printf("撤销令牌失败: %v", err)
}
```

#### HealthCheck()

检查 OAuth2 提供商的健康状态。

```go
func (c *Client) HealthCheck(ctx context.Context) error
```

**返回值：**
- `error` - 如果健康检查失败

**示例：**

```go
if err := client.HealthCheck(ctx); err != nil {
    log.Printf("OAuth2 提供商不健康: %v", err)
}
```

#### Close()

关闭客户端并清理资源。

```go
func (c *Client) Close(ctx context.Context) error
```

**示例：**

```go
defer client.Close(ctx)
```

---

## OAuth2 流程

### 授权码流程

最安全的流程，适用于 Web 应用。

```go
// 步骤 1: 重定向用户到授权 URL
func handleLogin(w http.ResponseWriter, r *http.Request) {
    state := generateRandomState() // 生成随机状态值
    authURL := client.GetAuthorizationURL(state)
    http.Redirect(w, r, authURL, http.StatusFound)
}

// 步骤 2: 处理回调
func handleCallback(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 验证状态
    state := r.URL.Query().Get("state")
    if !verifyState(state) {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }
    
    // 交换授权码
    code := r.URL.Query().Get("code")
    tokenResp, err := client.ExchangeAuthorizationCode(ctx, code)
    if err != nil {
        http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
        return
    }
    
    // 验证 ID 令牌
    idClaims, err := client.VerifyIDToken(ctx, tokenResp.IDToken)
    if err != nil {
        http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
        return
    }
    
    // 保存令牌到会话
    saveSession(w, tokenResp)
    
    log.Printf("用户 %s 登录成功", idClaims.Email)
    http.Redirect(w, r, "/dashboard", http.StatusFound)
}
```

### 客户端凭证流程

适用于服务对服务的通信。

```go
func getServiceToken(ctx context.Context) (string, error) {
    tokenResp, err := client.ClientCredentials(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get client credentials: %w", err)
    }
    return tokenResp.AccessToken, nil
}

func callProtectedAPI(ctx context.Context) error {
    token, err := getServiceToken(ctx)
    if err != nil {
        return err
    }
    
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 处理响应...
    return nil
}
```

### 刷新令牌流程

定期刷新访问令牌。

```go
func refreshAccessToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
    newTokenResp, err := client.RefreshToken(ctx, refreshToken)
    if err != nil {
        return nil, fmt.Errorf("failed to refresh token: %w", err)
    }
    return newTokenResp, nil
}

// 自动刷新令牌的 HTTP 客户端
type TokenRefreshTransport struct {
    Transport    http.RoundTripper
    TokenSource  func(ctx context.Context) (string, error)
}

func (t *TokenRefreshTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    token, err := t.TokenSource(req.Context())
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+token)
    return t.Transport.RoundTrip(req)
}
```

---

## 令牌管理

### TokenResponse 结构体

令牌响应。

```go
type TokenResponse struct {
    // AccessToken 是访问令牌
    AccessToken string
    
    // TokenType 是令牌类型（通常是 "Bearer"）
    TokenType string
    
    // RefreshToken 是刷新令牌（可选）
    RefreshToken string
    
    // IDToken 是 OIDC ID 令牌（可选）
    IDToken string
    
    // ExpiresIn 是令牌过期时间（秒）
    ExpiresIn int
    
    // Scope 是授予的作用域
    Scope string
}
```

### TokenClaims 结构体

访问令牌声明。

```go
type TokenClaims struct {
    // Subject 是主题标识符（用户 ID）
    Subject string `json:"sub"`
    
    // Issuer 是颁发者
    Issuer string `json:"iss"`
    
    // Audience 是受众
    Audience string `json:"aud"`
    
    // ExpiresAt 是过期时间
    ExpiresAt time.Time `json:"exp"`
    
    // IssuedAt 是颁发时间
    IssuedAt time.Time `json:"iat"`
    
    // NotBefore 是生效时间
    NotBefore time.Time `json:"nbf"`
    
    // Email 是电子邮件地址
    Email string `json:"email,omitempty"`
    
    // EmailVerified 表示电子邮件是否已验证
    EmailVerified bool `json:"email_verified,omitempty"`
    
    // Roles 是角色列表
    Roles []string `json:"roles,omitempty"`
    
    // CustomClaims 包含其他自定义声明
    CustomClaims map[string]interface{} `json:"-"`
}
```

### IDTokenClaims 结构体

OIDC ID 令牌声明。

```go
type IDTokenClaims struct {
    // Subject 是主题标识符（用户 ID）
    Subject string `json:"sub"`
    
    // Name 是用户全名
    Name string `json:"name,omitempty"`
    
    // GivenName 是名
    GivenName string `json:"given_name,omitempty"`
    
    // FamilyName 是姓
    FamilyName string `json:"family_name,omitempty"`
    
    // Email 是电子邮件地址
    Email string `json:"email,omitempty"`
    
    // EmailVerified 表示电子邮件是否已验证
    EmailVerified bool `json:"email_verified,omitempty"`
    
    // Picture 是用户头像 URL
    Picture string `json:"picture,omitempty"`
    
    // Locale 是用户区域设置
    Locale string `json:"locale,omitempty"`
}
```

---

## 提供商集成

### Keycloak

```yaml
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-app
  client_secret: secret
  issuer_url: https://auth.example.com/realms/myrealm
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

```go
config := &oauth2.Config{
    Provider:     "keycloak",
    ClientID:     "my-app",
    ClientSecret: "secret",
    IssuerURL:    "https://auth.example.com/realms/myrealm",
    RedirectURL:  "http://localhost:8080/callback",
    UseDiscovery: true,
}
```

### Auth0

```yaml
oauth2:
  enabled: true
  provider: auth0
  client_id: your-client-id
  client_secret: your-client-secret
  issuer_url: https://your-domain.auth0.com/
  redirect_url: http://localhost:8080/callback
  use_discovery: true
```

### Google

```yaml
oauth2:
  enabled: true
  provider: google
  client_id: your-client-id.apps.googleusercontent.com
  client_secret: your-client-secret
  issuer_url: https://accounts.google.com
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

### Microsoft/Azure AD

```yaml
oauth2:
  enabled: true
  provider: microsoft
  client_id: your-application-id
  client_secret: your-client-secret
  issuer_url: https://login.microsoftonline.com/{tenant}/v2.0
  redirect_url: http://localhost:8080/callback
  use_discovery: true
```

### 自定义提供商

```yaml
oauth2:
  enabled: true
  provider: custom
  client_id: your-client-id
  client_secret: your-client-secret
  auth_url: https://provider.example.com/oauth2/auth
  token_url: https://provider.example.com/oauth2/token
  user_info_url: https://provider.example.com/oauth2/userinfo
  jwks_url: https://provider.example.com/oauth2/jwks
  use_discovery: false
  scopes:
    - openid
    - profile
```

---

## 缓存管理

### 启用缓存

```go
config := &oauth2.Config{
    // ... 其他配置 ...
    CacheConfig: oauth2.TokenCacheConfig{
        Enabled:         true,
        MaxSize:         5000,
        TTL:             15 * time.Minute,
        CleanupInterval: 10 * time.Minute,
    },
}
```

### 缓存统计

```go
// 注意：需要通过客户端内部方法访问缓存统计
// 这是一个未来的改进点
```

---

## 错误处理

### 错误类型

```go
var (
    // ErrInvalidConfig 表示配置无效
    ErrInvalidConfig = errors.New("oauth2: invalid config")
    
    // ErrAuthorizationFailed 表示授权失败
    ErrAuthorizationFailed = errors.New("oauth2: authorization failed")
    
    // ErrTokenExchangeFailed 表示令牌交换失败
    ErrTokenExchangeFailed = errors.New("oauth2: token exchange failed")
    
    // ErrTokenValidationFailed 表示令牌验证失败
    ErrTokenValidationFailed = errors.New("oauth2: token validation failed")
    
    // ErrProviderUnavailable 表示提供商不可用
    ErrProviderUnavailable = errors.New("oauth2: provider unavailable")
)
```

### 错误处理示例

```go
tokenResp, err := client.ExchangeAuthorizationCode(ctx, code)
if err != nil {
    switch {
    case errors.Is(err, oauth2.ErrTokenExchangeFailed):
        log.Printf("令牌交换失败: %v", err)
        http.Error(w, "Authorization failed", http.StatusUnauthorized)
    case errors.Is(err, oauth2.ErrProviderUnavailable):
        log.Printf("OAuth2 提供商不可用: %v", err)
        http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
    default:
        log.Printf("未预期的错误: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
    return
}
```

---

## 代码示例

### 完整的 Web 应用示例

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/innovationmech/swit/pkg/security/oauth2"
)

var client *oauth2.Client

func main() {
    ctx := context.Background()

    // 创建 OAuth2 客户端
    config := &oauth2.Config{
        Provider:     "keycloak",
        ClientID:     "my-app",
        ClientSecret: "secret",
        IssuerURL:    "https://auth.example.com/realms/myrealm",
        RedirectURL:  "http://localhost:8080/callback",
        UseDiscovery: true,
        Scopes:       []string{"openid", "profile", "email"},
    }

    var err error
    client, err = oauth2.NewClient(ctx, config)
    if err != nil {
        log.Fatalf("创建 OAuth2 客户端失败: %v", err)
    }
    defer client.Close(ctx)

    // 注册路由
    http.HandleFunc("/login", handleLogin)
    http.HandleFunc("/callback", handleCallback)
    http.HandleFunc("/protected", handleProtected)
    http.HandleFunc("/logout", handleLogout)

    log.Println("服务器启动在 :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
    state := generateRandomState()
    // 在实际应用中，应该将 state 存储在会话中
    authURL := client.GetAuthorizationURL(state)
    http.Redirect(w, r, authURL, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 验证状态参数
    state := r.URL.Query().Get("state")
    if state == "" {
        http.Error(w, "Missing state parameter", http.StatusBadRequest)
        return
    }

    // 交换授权码
    code := r.URL.Query().Get("code")
    if code == "" {
        http.Error(w, "Missing code parameter", http.StatusBadRequest)
        return
    }

    tokenResp, err := client.ExchangeAuthorizationCode(ctx, code)
    if err != nil {
        log.Printf("交换授权码失败: %v", err)
        http.Error(w, "Authorization failed", http.StatusUnauthorized)
        return
    }

    // 验证 ID 令牌
    idClaims, err := client.VerifyIDToken(ctx, tokenResp.IDToken)
    if err != nil {
        log.Printf("验证 ID 令牌失败: %v", err)
        http.Error(w, "Invalid ID token", http.StatusUnauthorized)
        return
    }

    // 保存令牌到会话（简化示例，实际应使用安全的会话管理）
    // saveToSession(w, tokenResp)

    log.Printf("用户 %s (%s) 登录成功", idClaims.Name, idClaims.Email)
    http.Redirect(w, r, "/protected", http.StatusFound)
}

func handleProtected(w http.ResponseWriter, r *http.Request) {
    // 从会话获取访问令牌（简化示例）
    // accessToken := getFromSession(r)

    // 验证令牌
    // claims, err := client.ValidateToken(r.Context(), accessToken)
    // if err != nil {
    //     http.Error(w, "Unauthorized", http.StatusUnauthorized)
    //     return
    // }

    w.Write([]byte("受保护的内容"))
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
    // 从会话获取刷新令牌
    // refreshToken := getRefreshTokenFromSession(r)

    // 撤销令牌
    // _ = client.RevokeToken(r.Context(), refreshToken, "refresh_token")

    // 清除会话
    // clearSession(w)

    http.Redirect(w, r, "/", http.StatusFound)
}

func generateRandomState() string {
    // 实现生成随机状态值的逻辑
    return "random-state-value"
}
```

### API 服务示例（客户端凭证流程）

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/innovationmech/swit/pkg/security/oauth2"
)

func main() {
    ctx := context.Background()

    // 创建 OAuth2 客户端
    config := &oauth2.Config{
        Provider:     "keycloak",
        ClientID:     "api-service",
        ClientSecret: "service-secret",
        IssuerURL:    "https://auth.example.com/realms/myrealm",
        UseDiscovery: true,
    }

    client, err := oauth2.NewClient(ctx, config)
    if err != nil {
        log.Fatalf("创建 OAuth2 客户端失败: %v", err)
    }
    defer client.Close(ctx)

    // 使用客户端凭证获取令牌
    tokenResp, err := client.ClientCredentials(ctx)
    if err != nil {
        log.Fatalf("获取客户端凭证失败: %v", err)
    }

    log.Printf("访问令牌: %s", tokenResp.AccessToken)
    log.Printf("令牌类型: %s", tokenResp.TokenType)
    log.Printf("过期时间: %d 秒", tokenResp.ExpiresIn)

    // 使用令牌调用受保护的 API
    err = callProtectedAPI(ctx, tokenResp.AccessToken)
    if err != nil {
        log.Fatalf("调用 API 失败: %v", err)
    }
}

func callProtectedAPI(ctx context.Context, accessToken string) error {
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+accessToken)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    log.Printf("API 响应状态: %d", resp.StatusCode)
    return nil
}
```

---

## 最佳实践

### 1. 使用 OIDC 发现

优先使用 OIDC 发现来自动配置端点：

```go
config := &oauth2.Config{
    Provider:     "keycloak",
    ClientID:     "my-app",
    ClientSecret: "secret",
    IssuerURL:    "https://auth.example.com/realms/myrealm",
    UseDiscovery: true,  // ✅ 推荐
}
```

### 2. 启用令牌缓存

对于高并发场景，启用令牌缓存：

```go
config.CacheConfig = oauth2.TokenCacheConfig{
    Enabled:         true,
    MaxSize:         5000,
    TTL:             15 * time.Minute,
    CleanupInterval: 10 * time.Minute,
}
```

### 3. 安全地存储敏感信息

- 使用环境变量或密钥管理服务存储 `ClientSecret`
- 不要在代码中硬编码敏感信息
- 使用 `pkg/security/secrets` 包管理密钥

```go
// ✅ 推荐
config.ClientSecret = os.Getenv("OAUTH2_CLIENT_SECRET")

// ❌ 避免
config.ClientSecret = "hardcoded-secret"
```

### 4. 验证状态参数

始终验证授权码流程中的状态参数以防止 CSRF 攻击：

```go
func handleCallback(w http.ResponseWriter, r *http.Request) {
    state := r.URL.Query().Get("state")
    
    // ✅ 验证状态
    if !verifyState(state) {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }
    
    // 继续处理...
}
```

### 5. 实现令牌刷新

自动刷新过期的访问令牌：

```go
func getValidToken(ctx context.Context, currentToken *TokenResponse) (string, error) {
    // 检查令牌是否即将过期（提前 5 分钟）
    if time.Now().Add(5 * time.Minute).After(currentToken.ExpiresAt) {
        // 刷新令牌
        newToken, err := client.RefreshToken(ctx, currentToken.RefreshToken)
        if err != nil {
            return "", err
        }
        return newToken.AccessToken, nil
    }
    return currentToken.AccessToken, nil
}
```

### 6. 错误处理和重试

实现适当的错误处理和重试逻辑：

```go
func exchangeCodeWithRetry(ctx context.Context, code string, maxRetries int) (*TokenResponse, error) {
    var tokenResp *TokenResponse
    var err error
    
    for i := 0; i < maxRetries; i++ {
        tokenResp, err = client.ExchangeAuthorizationCode(ctx, code)
        if err == nil {
            return tokenResp, nil
        }
        
        // 如果是临时性错误，等待后重试
        if isTemporaryError(err) {
            time.Sleep(time.Duration(i+1) * time.Second)
            continue
        }
        
        // 永久性错误，立即返回
        return nil, err
    }
    
    return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}
```

### 7. 健康检查

定期检查 OAuth2 提供商的健康状态：

```go
func startHealthCheck(ctx context.Context, client *oauth2.Client) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := client.HealthCheck(ctx); err != nil {
                log.Printf("OAuth2 提供商健康检查失败: %v", err)
                // 触发告警...
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### 8. 使用 TLS

在生产环境中始终使用 TLS：

```yaml
oauth2:
  tls:
    enabled: true
    cert_file: /path/to/client-cert.pem
    key_file: /path/to/client-key.pem
    ca_file: /path/to/ca-cert.pem
```

### 9. 最小权限原则

只请求应用所需的最小作用域：

```go
config := &oauth2.Config{
    Scopes: []string{"openid", "profile", "email"},  // ✅ 只请求需要的作用域
    // 避免请求过多不必要的作用域
}
```

### 10. 安全的会话管理

使用安全的会话管理来存储令牌：

- 使用 HTTP-only cookies
- 启用 Secure 标志（HTTPS）
- 设置适当的 SameSite 属性
- 实现会话超时

```go
func saveSession(w http.ResponseWriter, token *TokenResponse) {
    http.SetCookie(w, &http.Cookie{
        Name:     "access_token",
        Value:    token.AccessToken,
        HttpOnly: true,      // ✅ 防止 XSS
        Secure:   true,      // ✅ 只通过 HTTPS
        SameSite: http.SameSiteStrictMode,  // ✅ 防止 CSRF
        MaxAge:   token.ExpiresIn,
    })
}
```

---

## 相关文档

- [OPA 策略 API 参考](./security-opa.md)
- [安全配置参考](./security-config.md)
- [JWT 令牌验证](../pkg/security/jwt/README.md)
- [OAuth2 集成指南](../guide/oauth2-integration.md)

## 许可证

Copyright (c) 2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.










