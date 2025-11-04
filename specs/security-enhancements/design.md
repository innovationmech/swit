# 安全增强 - 设计文档

## 架构概览

安全增强功能通过三个核心模块扩展 Swit 框架的安全能力：

```
┌─────────────────────────────────────────────────────────────┐
│                    Swit Application Layer                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┴───────────────┐
        │                               │
┌───────▼──────────┐          ┌────────▼─────────┐
│  Auth Middleware │          │  Policy Engine   │
│  (OAuth2/OIDC)   │          │      (OPA)       │
└───────┬──────────┘          └────────┬─────────┘
        │                               │
        ├───────────────┬───────────────┤
        │               │               │
┌───────▼──────┐  ┌────▼─────┐  ┌─────▼────────┐
│ HTTP/gRPC    │  │  Token   │  │   Policy     │
│ Interceptor  │  │ Validator│  │  Evaluator   │
└──────────────┘  └──────────┘  └──────────────┘
        │               │               │
        └───────────────┴───────────────┘
                        │
        ┌───────────────▼───────────────┐
        │   Security Audit & Logging     │
        └────────────────────────────────┘
```

## 组件设计

### 1. OAuth2/OIDC 认证模块

#### 1.1 OAuth2 配置管理

**位置**：`pkg/security/oauth2/config.go`

```go
// OAuth2Config OAuth2 配置
type OAuth2Config struct {
    // 基础配置
    Enabled      bool              `yaml:"enabled" json:"enabled"`
    Provider     string            `yaml:"provider" json:"provider"` // keycloak, auth0, custom
    
    // OAuth2 端点配置
    ClientID     string            `yaml:"client_id" json:"client_id"`
    ClientSecret string            `yaml:"client_secret" json:"client_secret"`
    RedirectURL  string            `yaml:"redirect_url" json:"redirect_url"`
    Scopes       []string          `yaml:"scopes" json:"scopes"`
    
    // OIDC 发现配置
    IssuerURL    string            `yaml:"issuer_url" json:"issuer_url"`
    
    // 或手动端点配置
    AuthURL      string            `yaml:"auth_url" json:"auth_url"`
    TokenURL     string            `yaml:"token_url" json:"token_url"`
    UserInfoURL  string            `yaml:"user_info_url" json:"user_info_url"`
    JWKSURL      string            `yaml:"jwks_url" json:"jwks_url"`
    
    // JWT 配置
    JWTConfig    JWTConfig         `yaml:"jwt" json:"jwt"`
    
    // 缓存配置
    CacheConfig  TokenCacheConfig  `yaml:"cache" json:"cache"`
    
    // TLS 配置
    TLSConfig    TLSConfig         `yaml:"tls" json:"tls"`
}

// JWTConfig JWT 令牌配置
type JWTConfig struct {
    SigningMethod   string        `yaml:"signing_method" json:"signing_method"` // RS256, HS256
    PublicKey       string        `yaml:"public_key" json:"public_key"`
    PrivateKey      string        `yaml:"private_key" json:"private_key"`
    Issuer          string        `yaml:"issuer" json:"issuer"`
    Audience        []string      `yaml:"audience" json:"audience"`
    ExpirationTime  time.Duration `yaml:"expiration_time" json:"expiration_time"`
    RefreshTime     time.Duration `yaml:"refresh_time" json:"refresh_time"`
    ClockSkew       time.Duration `yaml:"clock_skew" json:"clock_skew"`
}

// TokenCacheConfig 令牌缓存配置
type TokenCacheConfig struct {
    Enabled      bool          `yaml:"enabled" json:"enabled"`
    TTL          time.Duration `yaml:"ttl" json:"ttl"`
    MaxSize      int           `yaml:"max_size" json:"max_size"`
    CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}
```

#### 1.2 OAuth2 客户端

**位置**：`pkg/security/oauth2/client.go`

```go
// Client OAuth2 客户端
type Client struct {
    config       *OAuth2Config
    oauth2Config *oauth2.Config
    oidcProvider *oidc.Provider
    oidcVerifier *oidc.IDTokenVerifier
    cache        TokenCache
    metrics      *SecurityMetrics
    logger       *zap.Logger
}

// NewClient 创建 OAuth2 客户端
func NewClient(config *OAuth2Config) (*Client, error) {
    // 使用 OIDC 发现自动配置
    if config.IssuerURL != "" {
        return newOIDCClient(config)
    }
    // 手动配置端点
    return newManualClient(config)
}

// AuthCodeURL 获取授权码 URL
func (c *Client) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string

// Exchange 交换授权码为令牌
func (c *Client) Exchange(ctx context.Context, code string) (*Token, error)

// RefreshToken 刷新访问令牌
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*Token, error)

// ValidateToken 验证访问令牌
func (c *Client) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error)

// ValidateIDToken 验证 ID 令牌
func (c *Client) ValidateIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error)

// GetUserInfo 获取用户信息
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)

// IntrospectToken 内省令牌（检查有效性）
func (c *Client) IntrospectToken(ctx context.Context, token string) (*TokenIntrospection, error)
```

#### 1.3 JWT 令牌验证器

**位置**：`pkg/security/jwt/validator.go`

```go
// Validator JWT 验证器
type Validator struct {
    config     *JWTConfig
    keyFunc    jwt.Keyfunc
    cache      *jwks.Cache
    metrics    *SecurityMetrics
}

// NewValidator 创建 JWT 验证器
func NewValidator(config *JWTConfig) (*Validator, error)

// ValidateToken 验证 JWT 令牌
func (v *Validator) ValidateToken(tokenString string) (*Claims, error)

// ValidateWithContext 验证 JWT 令牌（带上下文）
func (v *Validator) ValidateWithContext(ctx context.Context, tokenString string) (*Claims, error)

// ExtractToken 从请求中提取令牌
func (v *Validator) ExtractToken(r *http.Request) (string, error)

// Claims JWT 声明
type Claims struct {
    jwt.RegisteredClaims
    Subject     string                 `json:"sub"`
    Email       string                 `json:"email,omitempty"`
    Name        string                 `json:"name,omitempty"`
    Groups      []string               `json:"groups,omitempty"`
    Roles       []string               `json:"roles,omitempty"`
    Permissions []string               `json:"permissions,omitempty"`
    CustomClaims map[string]interface{} `json:"custom,omitempty"`
}
```

#### 1.4 认证中间件

**位置**：`pkg/middleware/auth_oauth2.go`

```go
// OAuth2Middleware OAuth2 认证中间件（HTTP）
func OAuth2Middleware(client *oauth2.Client, options ...Option) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从请求中提取令牌
        token, err := extractToken(c.Request)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, 
                gin.H{"error": "missing or invalid token"})
            return
        }
        
        // 2. 验证令牌
        claims, err := client.ValidateToken(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, 
                gin.H{"error": "token validation failed"})
            return
        }
        
        // 3. 将认证信息注入上下文
        c.Set("auth_claims", claims)
        c.Set("user_id", claims.Subject)
        
        c.Next()
    }
}

// RequireScopes 要求特定作用域
func RequireScopes(scopes ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := getClaimsFromContext(c)
        if !hasScopes(claims, scopes) {
            c.AbortWithStatusJSON(http.StatusForbidden, 
                gin.H{"error": "insufficient scopes"})
            return
        }
        c.Next()
    }
}

// RequireRoles 要求特定角色
func RequireRoles(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := getClaimsFromContext(c)
        if !hasRoles(claims, roles) {
            c.AbortWithStatusJSON(http.StatusForbidden, 
                gin.H{"error": "insufficient roles"})
            return
        }
        c.Next()
    }
}
```

**位置**：`pkg/middleware/auth_grpc.go`

```go
// UnaryAuthInterceptor gRPC 一元认证拦截器
func UnaryAuthInterceptor(client *oauth2.Client) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, 
        info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        
        // 1. 从元数据中提取令牌
        token, err := extractTokenFromMetadata(ctx)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "missing token")
        }
        
        // 2. 验证令牌
        claims, err := client.ValidateToken(ctx, token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        
        // 3. 将认证信息注入上下文
        ctx = contextWithClaims(ctx, claims)
        
        return handler(ctx, req)
    }
}

// StreamAuthInterceptor gRPC 流认证拦截器
func StreamAuthInterceptor(client *oauth2.Client) grpc.StreamServerInterceptor {
    return func(srv interface{}, ss grpc.ServerStream, 
        info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
        
        ctx := ss.Context()
        
        // 验证逻辑同一元拦截器
        token, err := extractTokenFromMetadata(ctx)
        if err != nil {
            return status.Error(codes.Unauthenticated, "missing token")
        }
        
        claims, err := client.ValidateToken(ctx, token)
        if err != nil {
            return status.Error(codes.Unauthenticated, "invalid token")
        }
        
        ctx = contextWithClaims(ctx, claims)
        wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
        
        return handler(srv, wrapped)
    }
}
```

### 2. OPA 策略引擎集成

#### 2.1 OPA 配置管理

**位置**：`pkg/security/opa/config.go`

```go
// OPAConfig OPA 配置
type OPAConfig struct {
    Enabled          bool              `yaml:"enabled" json:"enabled"`
    
    // 部署模式
    Mode             string            `yaml:"mode" json:"mode"` // embedded, remote, sidecar
    
    // 嵌入式配置
    PolicyDir        string            `yaml:"policy_dir" json:"policy_dir"`
    DataFile         string            `yaml:"data_file" json:"data_file"`
    
    // 远程 OPA 服务器配置
    ServerURL        string            `yaml:"server_url" json:"server_url"`
    
    // 策略配置
    DefaultPolicy    string            `yaml:"default_policy" json:"default_policy"`
    PolicyPath       string            `yaml:"policy_path" json:"policy_path"` // package 路径
    
    // 决策配置
    DecisionPath     string            `yaml:"decision_path" json:"decision_path"`
    AllowByDefault   bool              `yaml:"allow_by_default" json:"allow_by_default"`
    
    // 缓存配置
    CacheEnabled     bool              `yaml:"cache_enabled" json:"cache_enabled"`
    CacheTTL         time.Duration     `yaml:"cache_ttl" json:"cache_ttl"`
    
    // 性能配置
    Timeout          time.Duration     `yaml:"timeout" json:"timeout"`
    MaxConcurrent    int               `yaml:"max_concurrent" json:"max_concurrent"`
    
    // 审计配置
    AuditEnabled     bool              `yaml:"audit_enabled" json:"audit_enabled"`
    AuditLogPath     string            `yaml:"audit_log_path" json:"audit_log_path"`
}
```

#### 2.2 OPA 客户端

**位置**：`pkg/security/opa/client.go`

```go
// Client OPA 客户端
type Client struct {
    config    *OPAConfig
    opa       *sdk.OPA        // 嵌入式 OPA
    httpClient *http.Client   // 远程 OPA
    cache     PolicyCache
    metrics   *SecurityMetrics
    logger    *zap.Logger
}

// NewClient 创建 OPA 客户端
func NewClient(config *OPAConfig) (*Client, error)

// LoadPolicy 加载策略
func (c *Client) LoadPolicy(ctx context.Context, policyName string, policy string) error

// LoadPolicyFromFile 从文件加载策略
func (c *Client) LoadPolicyFromFile(ctx context.Context, filePath string) error

// EvaluatePolicy 评估策略决策
func (c *Client) EvaluatePolicy(ctx context.Context, input map[string]interface{}) (*Decision, error)

// EvaluateWithPath 使用指定路径评估策略
func (c *Client) EvaluateWithPath(ctx context.Context, path string, 
    input map[string]interface{}) (*Decision, error)

// Decision 策略决策结果
type Decision struct {
    Allowed    bool                   `json:"allowed"`
    Reason     string                 `json:"reason,omitempty"`
    Violations []string               `json:"violations,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyInput 策略输入
type PolicyInput struct {
    // 请求信息
    Request  RequestInfo   `json:"request"`
    // 用户信息
    User     UserInfo      `json:"user"`
    // 资源信息
    Resource ResourceInfo  `json:"resource"`
    // 自定义数据
    Custom   map[string]interface{} `json:"custom,omitempty"`
}

type RequestInfo struct {
    Method   string            `json:"method"`
    Path     string            `json:"path"`
    Headers  map[string]string `json:"headers"`
    Query    map[string]string `json:"query"`
}

type UserInfo struct {
    ID          string   `json:"id"`
    Roles       []string `json:"roles"`
    Groups      []string `json:"groups"`
    Permissions []string `json:"permissions"`
    Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

type ResourceInfo struct {
    Type       string `json:"type"`
    ID         string `json:"id"`
    Owner      string `json:"owner"`
    Attributes map[string]interface{} `json:"attributes,omitempty"`
}
```

#### 2.3 策略管理器

**位置**：`pkg/security/opa/manager.go`

```go
// PolicyManager 策略管理器
type PolicyManager struct {
    client    *Client
    watcher   *PolicyWatcher
    validator *PolicyValidator
    logger    *zap.Logger
}

// NewPolicyManager 创建策略管理器
func NewPolicyManager(client *Client) *PolicyManager

// RegisterPolicy 注册策略
func (pm *PolicyManager) RegisterPolicy(name string, policy *Policy) error

// UpdatePolicy 更新策略
func (pm *PolicyManager) UpdatePolicy(name string, policy *Policy) error

// DeletePolicy 删除策略
func (pm *PolicyManager) DeletePolicy(name string) error

// ListPolicies 列出所有策略
func (pm *PolicyManager) ListPolicies() ([]*Policy, error)

// ValidatePolicy 验证策略语法
func (pm *PolicyManager) ValidatePolicy(policy string) error

// WatchPolicies 监听策略变化（从文件系统或远程）
func (pm *PolicyManager) WatchPolicies(ctx context.Context) error

// Policy 策略定义
type Policy struct {
    Name        string    `json:"name"`
    Version     string    `json:"version"`
    Description string    `json:"description"`
    Content     string    `json:"content"` // Rego 策略内容
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

#### 2.4 策略中间件

**位置**：`pkg/middleware/policy_opa.go`

```go
// OPAMiddleware OPA 策略中间件（HTTP）
func OPAMiddleware(client *opa.Client, options ...Option) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 构建策略输入
        input := buildPolicyInput(c)
        
        // 2. 评估策略
        decision, err := client.EvaluatePolicy(c.Request.Context(), input)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, 
                gin.H{"error": "policy evaluation failed"})
            return
        }
        
        // 3. 检查决策结果
        if !decision.Allowed {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error":  "access denied",
                "reason": decision.Reason,
            })
            return
        }
        
        // 4. 记录审计日志
        auditPolicyDecision(c, decision)
        
        c.Next()
    }
}

// ResourcePolicy 资源级策略中间件
func ResourcePolicy(resourceType string, getResourceID func(*gin.Context) string) gin.HandlerFunc {
    return func(c *gin.Context) {
        input := buildResourcePolicyInput(c, resourceType, getResourceID(c))
        // 评估逻辑同上
    }
}
```

**位置**：`pkg/middleware/policy_grpc.go`

```go
// UnaryPolicyInterceptor gRPC 一元策略拦截器
func UnaryPolicyInterceptor(client *opa.Client) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, 
        info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        
        // 1. 构建策略输入
        input := buildGRPCPolicyInput(ctx, info, req)
        
        // 2. 评估策略
        decision, err := client.EvaluatePolicy(ctx, input)
        if err != nil {
            return nil, status.Error(codes.Internal, "policy evaluation failed")
        }
        
        // 3. 检查决策
        if !decision.Allowed {
            return nil, status.Error(codes.PermissionDenied, decision.Reason)
        }
        
        return handler(ctx, req)
    }
}

// StreamPolicyInterceptor gRPC 流策略拦截器
func StreamPolicyInterceptor(client *opa.Client) grpc.StreamServerInterceptor {
    // 实现类似逻辑
}
```

#### 2.5 RBAC/ABAC 策略示例

**位置**：`pkg/security/opa/policies/rbac.rego`

```rego
package swit.authz.rbac

import future.keywords.if
import future.keywords.in

# 默认拒绝
default allow := false

# 基于角色的访问控制
allow if {
    # 检查用户角色
    some role in input.user.roles
    # 检查角色权限
    role_permissions[role][_] == required_permission
}

# 角色权限映射
role_permissions := {
    "admin": ["read", "write", "delete"],
    "editor": ["read", "write"],
    "viewer": ["read"],
}

# 提取所需权限
required_permission := concat(":", [
    input.request.method,
    input.resource.type
])

# 超级管理员总是允许
allow if {
    "superadmin" in input.user.roles
}

# 资源所有者检查
allow if {
    input.user.id == input.resource.owner
    "owner" in input.user.roles
}
```

**位置**：`pkg/security/opa/policies/abac.rego`

```rego
package swit.authz.abac

import future.keywords.if
import future.keywords.in

default allow := false

# 基于属性的访问控制
allow if {
    # 检查用户属性
    user_department_matches
    # 检查资源属性
    resource_sensitivity_allowed
    # 检查时间约束
    time_constraint_satisfied
}

# 部门匹配检查
user_department_matches if {
    input.user.attributes.department == input.resource.attributes.department
}

# 敏感性检查
resource_sensitivity_allowed if {
    resource_level := input.resource.attributes.sensitivity_level
    user_clearance := input.user.attributes.clearance_level
    clearance_levels[user_clearance] >= clearance_levels[resource_level]
}

clearance_levels := {
    "public": 0,
    "internal": 1,
    "confidential": 2,
    "secret": 3,
}

# 时间约束
time_constraint_satisfied if {
    # 工作时间检查
    is_business_hours
}

is_business_hours if {
    now := time.now_ns()
    hour := time.clock([now])[0]
    hour >= 9
    hour < 18
}
```

### 3. 安全最佳实践工具链

#### 3.1 安全扫描集成

**位置**：`pkg/security/scanner/scanner.go`

```go
// SecurityScanner 安全扫描器
type SecurityScanner struct {
    config  *ScannerConfig
    tools   []ScanTool
    logger  *zap.Logger
}

// ScannerConfig 扫描器配置
type ScannerConfig struct {
    Enabled       bool              `yaml:"enabled" json:"enabled"`
    Tools         []string          `yaml:"tools" json:"tools"` // gosec, govulncheck
    OutputDir     string            `yaml:"output_dir" json:"output_dir"`
    FailOnHigh    bool              `yaml:"fail_on_high" json:"fail_on_high"`
    FailOnMedium  bool              `yaml:"fail_on_medium" json:"fail_on_medium"`
    Excludes      []string          `yaml:"excludes" json:"excludes"`
}

// ScanTool 扫描工具接口
type ScanTool interface {
    Name() string
    Scan(ctx context.Context, target string) (*ScanResult, error)
}

// ScanResult 扫描结果
type ScanResult struct {
    Tool         string             `json:"tool"`
    Timestamp    time.Time          `json:"timestamp"`
    Findings     []SecurityFinding  `json:"findings"`
    Summary      ScanSummary        `json:"summary"`
}

// SecurityFinding 安全发现
type SecurityFinding struct {
    ID          string   `json:"id"`
    Severity    string   `json:"severity"` // critical, high, medium, low
    Category    string   `json:"category"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    File        string   `json:"file"`
    Line        int      `json:"line"`
    Code        string   `json:"code,omitempty"`
    Remediation string   `json:"remediation,omitempty"`
}

// Scan 执行安全扫描
func (ss *SecurityScanner) Scan(ctx context.Context, target string) ([]*ScanResult, error)

// GenerateReport 生成扫描报告
func (ss *SecurityScanner) GenerateReport(results []*ScanResult, format string) (string, error)
```

#### 3.2 依赖漏洞检查

**位置**：`pkg/security/vulnerability/checker.go`

```go
// VulnerabilityChecker 漏洞检查器
type VulnerabilityChecker struct {
    config    *CheckerConfig
    databases []VulnDatabase
    logger    *zap.Logger
}

// CheckerConfig 检查器配置
type CheckerConfig struct {
    Enabled       bool              `yaml:"enabled" json:"enabled"`
    Databases     []string          `yaml:"databases" json:"databases"` // osv, nvd
    UpdatePeriod  time.Duration     `yaml:"update_period" json:"update_period"`
}

// CheckDependencies 检查依赖漏洞
func (vc *VulnerabilityChecker) CheckDependencies(ctx context.Context) ([]*Vulnerability, error)

// Vulnerability 漏洞信息
type Vulnerability struct {
    ID          string    `json:"id"`
    Package     string    `json:"package"`
    Version     string    `json:"version"`
    FixedIn     string    `json:"fixed_in,omitempty"`
    Severity    string    `json:"severity"`
    CVSS        float64   `json:"cvss"`
    Description string    `json:"description"`
    References  []string  `json:"references"`
}
```

#### 3.3 TLS/mTLS 配置

**位置**：`pkg/security/tls/config.go`

```go
// TLSConfig TLS 配置
type TLSConfig struct {
    Enabled            bool     `yaml:"enabled" json:"enabled"`
    
    // 证书配置
    CertFile           string   `yaml:"cert_file" json:"cert_file"`
    KeyFile            string   `yaml:"key_file" json:"key_file"`
    CAFile             string   `yaml:"ca_file" json:"ca_file"`
    
    // mTLS 配置
    ClientAuth         string   `yaml:"client_auth" json:"client_auth"` // none, request, require
    ClientCAs          []string `yaml:"client_cas" json:"client_cas"`
    
    // 协议配置
    MinVersion         string   `yaml:"min_version" json:"min_version"` // TLS1.2, TLS1.3
    MaxVersion         string   `yaml:"max_version" json:"max_version"`
    CipherSuites       []string `yaml:"cipher_suites" json:"cipher_suites"`
    
    // 证书验证
    InsecureSkipVerify bool     `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
    ServerName         string   `yaml:"server_name" json:"server_name"`
}

// NewTLSConfig 创建 TLS 配置
func NewTLSConfig(config *TLSConfig) (*tls.Config, error)

// LoadCertificate 加载证书
func LoadCertificate(certFile, keyFile string) (tls.Certificate, error)

// LoadCAPool 加载 CA 证书池
func LoadCAPool(caFiles ...string) (*x509.CertPool, error)
```

#### 3.4 密钥管理

**位置**：`pkg/security/secrets/manager.go`

```go
// SecretsManager 密钥管理器
type SecretsManager struct {
    config   *SecretsConfig
    provider SecretsProvider
    cache    *secretsCache
    logger   *zap.Logger
}

// SecretsConfig 密钥配置
type SecretsConfig struct {
    Provider   string            `yaml:"provider" json:"provider"` // env, file, vault
    VaultURL   string            `yaml:"vault_url" json:"vault_url"`
    VaultToken string            `yaml:"vault_token" json:"vault_token"`
    FilePath   string            `yaml:"file_path" json:"file_path"`
}

// SecretsProvider 密钥提供者接口
type SecretsProvider interface {
    GetSecret(ctx context.Context, key string) (string, error)
    SetSecret(ctx context.Context, key string, value string) error
    DeleteSecret(ctx context.Context, key string) error
}

// GetSecret 获取密钥
func (sm *SecretsManager) GetSecret(ctx context.Context, key string) (string, error)

// RefreshSecrets 刷新密钥缓存
func (sm *SecretsManager) RefreshSecrets(ctx context.Context) error
```

## 集成设计

### 1. 服务器框架集成

**位置**：`pkg/server/security.go`

```go
// SecurityManager 安全管理器
type SecurityManager struct {
    oauth2Client     *oauth2.Client
    opaClient        *opa.Client
    scanner          *scanner.SecurityScanner
    vulnChecker      *vulnerability.VulnerabilityChecker
    secretsManager   *secrets.SecretsManager
    metrics          *SecurityMetrics
    logger           *zap.Logger
}

// SecurityConfig 安全配置
type SecurityConfig struct {
    OAuth2    oauth2.OAuth2Config           `yaml:"oauth2" json:"oauth2"`
    OPA       opa.OPAConfig                 `yaml:"opa" json:"opa"`
    TLS       tls.TLSConfig                 `yaml:"tls" json:"tls"`
    Scanner   scanner.ScannerConfig         `yaml:"scanner" json:"scanner"`
    Secrets   secrets.SecretsConfig         `yaml:"secrets" json:"secrets"`
}

// NewSecurityManager 创建安全管理器
func NewSecurityManager(config *SecurityConfig) (*SecurityManager, error)

// InitializeSecurity 初始化安全功能
func (sm *SecurityManager) InitializeSecurity(ctx context.Context) error

// RegisterSecurityMiddleware 注册安全中间件
func (sm *SecurityManager) RegisterSecurityMiddleware(
    router *gin.Engine, 
    grpcServer *grpc.Server) error
```

### 2. 中间件管理器集成

**位置**：`pkg/server/middleware.go` (扩展)

```go
// ConfigureSecurityMiddleware 配置安全中间件
func (m *MiddlewareManager) ConfigureSecurityMiddleware(
    router *gin.Engine, 
    securityMgr *SecurityManager) error {
    
    // OAuth2 认证
    if securityMgr.oauth2Client != nil {
        router.Use(middleware.OAuth2Middleware(securityMgr.oauth2Client))
    }
    
    // OPA 策略
    if securityMgr.opaClient != nil {
        router.Use(middleware.OPAMiddleware(securityMgr.opaClient))
    }
    
    return nil
}
```

## 安全指标

### 安全相关指标

**位置**：`pkg/security/metrics.go`

```go
// SecurityMetrics 安全指标
type SecurityMetrics struct {
    // 认证指标
    AuthAttempts      *prometheus.CounterVec   // 认证尝试次数
    AuthSuccesses     *prometheus.CounterVec   // 认证成功次数
    AuthFailures      *prometheus.CounterVec   // 认证失败次数
    TokenValidations  *prometheus.CounterVec   // 令牌验证次数
    
    // 授权指标
    PolicyEvaluations *prometheus.CounterVec   // 策略评估次数
    AccessDenied      *prometheus.CounterVec   // 访问拒绝次数
    AccessGranted     *prometheus.CounterVec   // 访问授予次数
    
    // 性能指标
    AuthDuration      *prometheus.HistogramVec // 认证耗时
    PolicyDuration    *prometheus.HistogramVec // 策略评估耗时
    
    // 安全事件
    SecurityEvents    *prometheus.CounterVec   // 安全事件计数
    VulnerabilitiesFound *prometheus.GaugeVec  // 发现的漏洞数量
}
```

## 性能考虑

### 1. 认证性能优化

- **令牌缓存**：缓存已验证的令牌，避免重复验证
- **JWKS 缓存**：缓存公钥，减少网络请求
- **连接池**：复用 HTTP 连接
- **并发控制**：限制并发验证请求

### 2. 策略评估优化

- **策略编译缓存**：预编译策略，避免重复解析
- **决策缓存**：缓存相同输入的决策结果
- **部分评估**：只评估需要的策略路径
- **嵌入式部署**：使用嵌入式 OPA 避免网络延迟

### 3. 性能目标

- OAuth2 令牌验证：< 10ms (P99)
- OPA 策略评估：< 5ms (P99)
- 总体认证授权开销：< 15ms (P99)

## 安全考虑

### 1. 密钥管理

- 不在代码中硬编码密钥
- 使用环境变量或密钥管理服务
- 定期轮换密钥
- 加密存储敏感配置

### 2. 令牌安全

- 使用短期访问令牌 + 长期刷新令牌
- 实施令牌撤销机制
- 使用安全的签名算法（RS256, ES256）
- 验证令牌的所有声明（iss, aud, exp 等）

### 3. 传输安全

- 强制使用 TLS 1.2+
- 禁用弱加密套件
- 支持 mTLS 双向认证
- 验证证书链

### 4. 审计日志

- 记录所有认证尝试
- 记录所有授权决策
- 记录安全事件和异常
- 定期审查日志

## 测试策略

### 1. 单元测试

- 令牌验证逻辑
- 策略评估逻辑
- 配置解析和验证
- 错误处理

### 2. 集成测试

- 完整认证流程
- 策略执行流程
- 中间件集成
- 端到端安全测试

### 3. 安全测试

- 渗透测试
- 漏洞扫描
- 模糊测试
- 负载测试

### 4. 合规测试

- OWASP Top 10 检查
- CWE 常见弱点检查
- 安全基线验证

## 迁移策略

### 1. 向后兼容

- 安全功能默认禁用
- 渐进式启用
- 不破坏现有 API

### 2. 迁移步骤

1. 部署新的安全组件（不启用）
2. 更新配置文档
3. 在测试环境启用和验证
4. 逐步在生产环境推广
5. 监控和优化

### 3. 回滚计划

- 保留配置开关
- 监控关键指标
- 准备快速回滚方案

## 文档规划

### 1. API 文档

- OAuth2/OIDC API 参考
- OPA 策略 API 参考
- 安全配置参考

### 2. 开发指南

- 安全最佳实践
- 认证授权集成指南
- 策略编写指南
- 安全测试指南

### 3. 运维指南

- 部署安全配置
- 证书管理
- 密钥轮换
- 事件响应

### 4. 示例代码

- OAuth2 集成示例
- OPA 策略示例
- mTLS 配置示例
- 安全扫描集成示例

