# Swit 安全最佳实践指南

本文档提供 Swit 微服务框架的全面安全最佳实践指南，涵盖从开发到部署的完整安全生命周期。

## 目录

- [概述](#概述)
- [安全开发生命周期 (SDLC)](#安全开发生命周期-sdlc)
- [身份认证最佳实践](#身份认证最佳实践)
- [授权最佳实践](#授权最佳实践)
- [API 安全指南](#api-安全指南)
- [常见漏洞防护 (OWASP Top 10)](#常见漏洞防护-owasp-top-10)
- [数据保护](#数据保护)
- [传输层安全](#传输层安全)
- [密钥管理](#密钥管理)
- [安全配置](#安全配置)
- [安全监控与审计](#安全监控与审计)
- [容器安全](#容器安全)
- [依赖管理](#依赖管理)
- [安全测试](#安全测试)
- [相关资源](#相关资源)

---

## 概述

Swit 框架提供了多层次的安全功能，包括：

- **身份认证** - OAuth2/OIDC、JWT、mTLS
- **授权** - OPA 策略引擎（RBAC/ABAC）
- **传输安全** - TLS/mTLS 加密
- **数据保护** - 加密、脱敏、审计
- **安全监控** - 指标、日志、追踪
- **自动化扫描** - gosec、Trivy、govulncheck

### 安全原则

1. **纵深防御** - 多层安全控制
2. **最小权限** - 只授予必需的权限
3. **默认拒绝** - 明确允许才能访问
4. **安全失败** - 错误时默认拒绝访问
5. **完整日志** - 记录所有安全事件

---

## 安全开发生命周期 (SDLC)

### 1. 规划阶段

**威胁建模**

识别潜在的安全威胁：

```bash
# 使用威胁建模工具
- 数据流图分析
- STRIDE 威胁分类
- 攻击面分析
```

**安全需求**

定义安全需求：
- 认证和授权需求
- 数据保护需求
- 合规性要求（GDPR、PCI-DSS 等）
- 审计日志需求

### 2. 设计阶段

**架构安全审查**

```go
// ✅ 推荐：使用框架的安全功能
config := &server.ServerConfig{
    Name: "my-service",
    Security: &SecurityConfig{
        OAuth2: &OAuth2Config{
            Enabled: true,
            Provider: "keycloak",
        },
        OPA: &OPAConfig{
            Mode: "embedded",
            PolicyDir: "./policies",
        },
    },
}

// ❌ 避免：自定义不安全的认证方案
// 不要重新发明轮子
```

**设计原则**
- 分离认证和授权逻辑
- 使用成熟的安全库
- 实现适当的错误处理
- 避免安全敏感信息泄露

### 3. 开发阶段

**安全编码标准**

遵循 Go 安全编码最佳实践：

```go
// ✅ 输入验证
func CreateUser(req *CreateUserRequest) error {
    if err := validateEmail(req.Email); err != nil {
        return fmt.Errorf("invalid email: %w", err)
    }
    if err := validatePassword(req.Password); err != nil {
        return fmt.Errorf("weak password: %w", err)
    }
    // 使用参数化查询防止 SQL 注入
    return db.Create(&User{Email: req.Email}).Error
}

// ❌ 避免：字符串拼接 SQL
// query := "SELECT * FROM users WHERE email = '" + email + "'"
```

**代码审查清单**

每次提交前检查：
- [ ] 输入验证完整
- [ ] 无硬编码密钥
- [ ] 错误处理得当
- [ ] 敏感数据已加密
- [ ] 日志无敏感信息
- [ ] 依赖项已更新

### 4. 测试阶段

**安全测试类型**

```bash
# 静态代码分析
make quality              # 包含 gosec 扫描

# 依赖漏洞扫描
make security            # Trivy + govulncheck

# 单元测试（包括安全测试）
make test

# 集成测试
make test-advanced TYPE=integration
```

### 5. 部署阶段

**部署前检查**

```bash
# 1. 扫描容器镜像
trivy image myservice:latest

# 2. 验证配置
./myservice --validate-config

# 3. 检查密钥管理
# 确保没有硬编码密钥

# 4. 启用 TLS
# 确保生产环境启用 TLS
```

### 6. 维护阶段

**持续安全监控**

- 定期更新依赖
- 监控安全公告
- 审查审计日志
- 执行定期安全扫描
- 进行渗透测试

---

## 身份认证最佳实践

### OAuth2/OIDC 认证

**推荐配置**

```yaml
# swit.yaml
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-service
  client_secret: ${OAUTH2_CLIENT_SECRET}  # 从环境变量读取
  issuer_url: https://auth.example.com/realms/production
  use_discovery: true
  
  # 安全配置
  scopes:
    - openid
    - profile
    - email
  
  jwt:
    signing_method: RS256  # 使用非对称加密
    clock_skew: 5m
    required_claims:
      - sub
      - email
  
  # 启用缓存提升性能
  cache:
    enabled: true
    max_size: 5000
    ttl: 15m
  
  # TLS 配置
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-bundle.crt
```

**实现示例**

```go
package main

import (
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/innovationmech/swit/pkg/middleware"
)

func setupOAuth2() error {
    // 1. 创建 OAuth2 客户端
    config := &oauth2.Config{
        Provider:    "keycloak",
        ClientID:    "my-service",
        ClientSecret: os.Getenv("OAUTH2_CLIENT_SECRET"),
        IssuerURL:   "https://auth.example.com/realms/production",
        UseDiscovery: true,
    }
    
    client, err := oauth2.NewClient(config)
    if err != nil {
        return err
    }
    
    // 2. 创建认证中间件
    authMiddleware := middleware.NewOAuth2Middleware(client)
    
    // 3. 应用到路由
    router.Use(authMiddleware.Authenticate())
    
    return nil
}
```

### JWT 令牌验证

**最佳实践**

```go
import "github.com/innovationmech/swit/pkg/security/jwt"

// JWT 验证器配置
jwtConfig := &jwt.ValidatorConfig{
    // 使用 JWKS 自动获取公钥
    JWKSConfig: &jwt.JWKSConfig{
        URL:             "https://auth.example.com/.well-known/jwks.json",
        CacheEnabled:    true,
        CacheTTL:        time.Hour,
        RefreshInterval: 15 * time.Minute,
    },
    
    // 验证声明
    Issuer:   "https://auth.example.com",
    Audience: "my-service-api",
    ClockSkew: 5 * time.Minute,
    
    // 必需的声明
    RequiredClaims: []string{"sub", "exp", "iat"},
}

validator, err := jwt.NewValidator(jwtConfig)
if err != nil {
    log.Fatal(err)
}

// 验证令牌
token := "eyJhbGciOiJSUzI1NiIs..."
claims, err := validator.Validate(token)
if err != nil {
    // 令牌无效
    return err
}

// 使用声明
userID := claims.Subject
email := claims["email"].(string)
```

### 密码安全

**密码策略**

```go
import "github.com/innovationmech/swit/pkg/utils"

// 密码验证规则
func ValidatePassword(password string) error {
    if len(password) < 12 {
        return errors.New("password must be at least 12 characters")
    }
    
    // 检查复杂度
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
    hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)
    
    if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
        return errors.New("password must contain uppercase, lowercase, number and special character")
    }
    
    return nil
}

// 密码哈希
func HashPassword(password string) (string, error) {
    // 使用 bcrypt（推荐）
    hashedPassword, err := utils.HashPassword(password)
    if err != nil {
        return "", err
    }
    return hashedPassword, nil
}

// 密码验证
func VerifyPassword(hashedPassword, password string) bool {
    return utils.CheckPassword(hashedPassword, password)
}
```

### mTLS 认证

**客户端证书验证**

```yaml
# swit.yaml
server:
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      # 要求并验证客户端证书
      client_auth: require_and_verify
      client_ca_file: /etc/ssl/certs/client-ca.pem
      min_version: "1.2"
```

---

## 授权最佳实践

### OPA 策略引擎

**RBAC 实现**

```rego
# policies/rbac.rego
package authz

import future.keywords.if
import future.keywords.in

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

**ABAC 实现**

```rego
# policies/abac.rego
package authz

import future.keywords.if

# 基于属性的访问控制
allow if {
    # 检查部门匹配
    input.subject.attributes.department == input.resource.department
    
    # 检查安全级别
    input.subject.attributes.clearance_level >= input.resource.security_level
    
    # 检查时间窗口
    is_business_hours
}

# 工作时间检查
is_business_hours if {
    hour := time.clock(time.now_ns())[0]
    hour >= 9
    hour < 18
}
```

**Go 集成**

```go
import "github.com/innovationmech/swit/pkg/security/opa"

// 创建 OPA 客户端
opaConfig := &opa.Config{
    Mode:                "embedded",
    DefaultDecisionPath: "authz/allow",
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./pkg/security/opa/policies",
    },
    Cache: &opa.CacheConfig{
        Enabled:  true,
        MaxSize:  10000,
        TTL:      5 * time.Minute,
    },
}

client, err := opa.NewClient(opaConfig)
if err != nil {
    log.Fatal(err)
}

// 评估策略
input := map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"editor"},
    },
    "action":   "write",
    "resource": "documents/project-plan.pdf",
}

result, err := client.Evaluate(ctx, input)
if err != nil {
    return err
}

if !result.Allow {
    return errors.New("access denied")
}
```

### 权限检查中间件

```go
import (
    "github.com/innovationmech/swit/pkg/middleware"
    "github.com/gin-gonic/gin"
)

// 创建授权中间件
func setupAuthorization(opaClient *opa.Client) gin.HandlerFunc {
    return middleware.NewOPAMiddleware(opaClient, middleware.OPAMiddlewareConfig{
        InputBuilder: func(c *gin.Context) map[string]interface{} {
            // 从上下文提取用户信息
            user := c.GetString("user")
            roles := c.GetStringSlice("roles")
            
            return map[string]interface{}{
                "subject": map[string]interface{}{
                    "user":  user,
                    "roles": roles,
                },
                "action":   c.Request.Method,
                "resource": c.Request.URL.Path,
            }
        },
        OnDeny: func(c *gin.Context) {
            c.JSON(403, gin.H{"error": "access denied"})
            c.Abort()
        },
    })
}

// 应用到路由
router.GET("/api/v1/documents/:id", 
    authMiddleware,
    authorizationMiddleware,
    getDocumentHandler,
)
```

---

## API 安全指南

### 输入验证

**验证所有输入**

```go
import (
    "github.com/go-playground/validator/v10"
)

// 使用结构体标签验证
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,alphanum,min=3,max=20"`
    Password string `json:"password" validate:"required,min=12"`
    Age      int    `json:"age" validate:"required,gte=18,lte=120"`
}

var validate = validator.New()

func CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    
    // 验证请求
    if err := validate.Struct(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 额外的业务验证
    if err := ValidatePassword(req.Password); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 处理请求...
}
```

### 输出编码

**防止 XSS 攻击**

```go
import "html"

// HTML 编码
func SafeOutput(userInput string) string {
    return html.EscapeString(userInput)
}

// JSON 响应自动编码
c.JSON(200, gin.H{
    "message": userInput,  // Gin 自动处理 JSON 编码
})
```

### 速率限制

**防止滥用**

```go
import "github.com/innovationmech/swit/pkg/middleware"

// 基于 IP 的速率限制
rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 10,
    Burst:            20,
    KeyFunc: func(c *gin.Context) string {
        return c.ClientIP()
    },
})

router.Use(rateLimiter)

// 基于用户的速率限制
userRateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 100,
    Burst:            200,
    KeyFunc: func(c *gin.Context) string {
        user := c.GetString("user")
        return "user:" + user
    },
})

router.Use(authMiddleware, userRateLimiter)
```

### CORS 配置

**安全的跨域配置**

```go
import "github.com/gin-contrib/cors"

// CORS 配置
corsConfig := cors.Config{
    // ❌ 避免：允许所有来源
    // AllowOrigins: []string{"*"},
    
    // ✅ 推荐：明确指定允许的来源
    AllowOrigins: []string{
        "https://app.example.com",
        "https://admin.example.com",
    },
    
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders: []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,
}

router.Use(cors.New(corsConfig))
```

### API 版本控制

**向后兼容**

```go
// 使用路径版本控制
v1 := router.Group("/api/v1")
{
    v1.GET("/users", getUsersV1)
}

v2 := router.Group("/api/v2")
{
    v2.GET("/users", getUsersV2)
}

// 或使用头部版本控制
router.Use(func(c *gin.Context) {
    version := c.GetHeader("API-Version")
    if version == "" {
        version = "v1"  // 默认版本
    }
    c.Set("api_version", version)
    c.Next()
})
```

---

## 常见漏洞防护 (OWASP Top 10)

### 1. 注入攻击

**SQL 注入防护**

```go
// ✅ 使用 GORM 参数化查询
db.Where("email = ?", email).First(&user)

// ✅ 使用命名参数
db.Where("email = @email AND status = @status", 
    sql.Named("email", email),
    sql.Named("status", "active"),
).Find(&users)

// ❌ 避免：字符串拼接
// query := "SELECT * FROM users WHERE email = '" + email + "'"
```

**命令注入防护**

```go
import "os/exec"

// ✅ 使用参数数组
cmd := exec.Command("ls", "-la", userDir)

// ❌ 避免：shell 执行
// cmd := exec.Command("sh", "-c", "ls -la " + userDir)
```

### 2. 身份认证失效

**会话管理**

```go
// ✅ 使用安全的会话配置
sessionConfig := sessions.Config{
    CookieHTTPOnly: true,   // 防止 XSS
    CookieSecure:   true,   // 仅 HTTPS
    CookieSameSite: http.SameSiteStrictMode,  // 防止 CSRF
    MaxAge:         3600,   // 1 小时过期
}

// 登录后重新生成会话 ID
func Login(c *gin.Context) {
    // 验证用户...
    
    // 重新生成会话 ID 防止会话固定攻击
    session := sessions.Default(c)
    session.Clear()
    session.Set("user_id", user.ID)
    session.Save()
}
```

### 3. 敏感数据泄露

**加密敏感数据**

```go
import "github.com/innovationmech/swit/pkg/utils"

// 加密敏感字段
type User struct {
    ID       uint
    Email    string
    SSN      string  // 社保号码 - 需要加密
}

func (u *User) BeforeSave(tx *gorm.DB) error {
    if u.SSN != "" {
        encrypted, err := utils.Encrypt(u.SSN, encryptionKey)
        if err != nil {
            return err
        }
        u.SSN = encrypted
    }
    return nil
}

func (u *User) AfterFind(tx *gorm.DB) error {
    if u.SSN != "" {
        decrypted, err := utils.Decrypt(u.SSN, encryptionKey)
        if err != nil {
            return err
        }
        u.SSN = decrypted
    }
    return nil
}
```

**日志脱敏**

```go
import "github.com/innovationmech/swit/pkg/logger"

// 配置日志脱敏
logger.SetRedactFields([]string{
    "password",
    "token",
    "api_key",
    "secret",
    "ssn",
    "credit_card",
})

// 日志会自动脱敏
log.Info("User created", 
    zap.String("email", user.Email),
    zap.String("password", "********"),  // 自动脱敏
)
```

### 4. XML 外部实体 (XXE)

**安全的 XML 解析**

```go
import "encoding/xml"

// ✅ 禁用外部实体
decoder := xml.NewDecoder(reader)
decoder.Strict = true
// Go 的 xml 包默认不解析外部实体
```

### 5. 访问控制失效

**强制访问控制**

```go
// ✅ 每个请求都检查权限
func GetDocument(c *gin.Context) {
    docID := c.Param("id")
    userID := c.GetString("user_id")
    
    // 获取文档
    var doc Document
    if err := db.First(&doc, docID).Error; err != nil {
        c.JSON(404, gin.H{"error": "not found"})
        return
    }
    
    // 检查权限
    if doc.OwnerID != userID && !hasAdminRole(c) {
        c.JSON(403, gin.H{"error": "access denied"})
        return
    }
    
    c.JSON(200, doc)
}

// ❌ 避免：通过 URL 参数控制访问
// func GetDocument(c *gin.Context) {
//     if c.Query("admin") == "true" {
//         // 危险！
//     }
// }
```

### 6. 安全配置错误

**安全的默认配置**

```yaml
# ✅ 生产环境配置
server:
  debug: false           # 禁用调试模式
  expose_errors: false   # 不暴露错误详情
  
  http:
    tls:
      enabled: true
      min_version: "1.2"

# ❌ 避免：开发配置用于生产
# server:
#   debug: true
#   expose_errors: true
```

### 7. 跨站脚本 (XSS)

**内容安全策略 (CSP)**

```go
// 添加 CSP 头部
router.Use(func(c *gin.Context) {
    c.Header("Content-Security-Policy", 
        "default-src 'self'; "+
        "script-src 'self' 'unsafe-inline'; "+
        "style-src 'self' 'unsafe-inline'; "+
        "img-src 'self' data: https:;")
    c.Next()
})

// 添加其他安全头部
router.Use(func(c *gin.Context) {
    c.Header("X-Content-Type-Options", "nosniff")
    c.Header("X-Frame-Options", "DENY")
    c.Header("X-XSS-Protection", "1; mode=block")
    c.Next()
})
```

### 8. 不安全的反序列化

**安全的反序列化**

```go
// ✅ 使用类型安全的 JSON 绑定
var req CreateUserRequest
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(400, gin.H{"error": "invalid request"})
    return
}

// ❌ 避免：不受信任的反序列化
// var data map[string]interface{}
// if err := json.Unmarshal(untrustedData, &data); err != nil {
//     // 可能执行恶意代码
// }
```

### 9. 使用含有已知漏洞的组件

**依赖管理**

```bash
# 定期扫描依赖
make security

# 更新依赖
go get -u ./...
go mod tidy

# 检查已知漏洞
govulncheck ./...
```

### 10. 日志和监控不足

**完整的审计日志**

```go
import "github.com/innovationmech/swit/pkg/security/audit"

// 记录安全事件
auditor := audit.NewLogger(audit.Config{
    Enabled: true,
    Events: []string{
        "authentication_success",
        "authentication_failure",
        "authorization_denied",
        "sensitive_data_accessed",
    },
})

// 记录认证事件
auditor.Log(audit.Event{
    Type:      "authentication_success",
    Actor:     userID,
    Action:    "login",
    Resource:  "system",
    Timestamp: time.Now(),
    Result:    "success",
})
```

---

## 数据保护

### 静态数据加密

**数据库加密**

```go
import "github.com/innovationmech/swit/pkg/utils"

// 字段级加密
type CreditCard struct {
    ID          uint
    UserID      uint
    Number      string  // 加密存储
    CVV         string  // 加密存储
    ExpiryMonth int
    ExpiryYear  int
}

func (c *CreditCard) BeforeSave(tx *gorm.DB) error {
    if c.Number != "" {
        encrypted, err := utils.EncryptAES(c.Number, encryptionKey)
        if err != nil {
            return err
        }
        c.Number = encrypted
    }
    
    if c.CVV != "" {
        encrypted, err := utils.EncryptAES(c.CVV, encryptionKey)
        if err != nil {
            return err
        }
        c.CVV = encrypted
    }
    
    return nil
}

func (c *CreditCard) AfterFind(tx *gorm.DB) error {
    if c.Number != "" {
        decrypted, err := utils.DecryptAES(c.Number, encryptionKey)
        if err != nil {
            return err
        }
        c.Number = decrypted
    }
    
    if c.CVV != "" {
        decrypted, err := utils.DecryptAES(c.CVV, encryptionKey)
        if err != nil {
            return err
        }
        c.CVV = decrypted
    }
    
    return nil
}
```

**文件加密**

```go
import "crypto/aes"

// 加密文件
func EncryptFile(inputPath, outputPath string, key []byte) error {
    plaintext, err := os.ReadFile(inputPath)
    if err != nil {
        return err
    }
    
    ciphertext, err := utils.EncryptAES(plaintext, key)
    if err != nil {
        return err
    }
    
    return os.WriteFile(outputPath, ciphertext, 0600)
}
```

### 传输中的数据加密

见 [传输层安全](#传输层安全) 章节。

### 数据脱敏

**PII 数据脱敏**

```go
// 脱敏电话号码
func MaskPhone(phone string) string {
    if len(phone) < 7 {
        return "***"
    }
    return phone[:3] + "****" + phone[len(phone)-4:]
}

// 脱敏邮箱
func MaskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return "***"
    }
    username := parts[0]
    if len(username) > 3 {
        username = username[:3] + "***"
    }
    return username + "@" + parts[1]
}

// API 响应脱敏
type UserResponse struct {
    ID    uint   `json:"id"`
    Email string `json:"email"`
    Phone string `json:"phone"`
}

func (u *User) ToResponse() UserResponse {
    return UserResponse{
        ID:    u.ID,
        Email: MaskEmail(u.Email),
        Phone: MaskPhone(u.Phone),
    }
}
```

---

## 传输层安全

### TLS 配置

**生产环境 TLS**

```yaml
# swit.yaml
server:
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "1.2"        # 最低 TLS 1.2
      max_version: "1.3"        # 推荐 TLS 1.3
      cipher_suites:
        - TLS_AES_128_GCM_SHA256
        - TLS_AES_256_GCM_SHA384
        - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
        - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "1.2"
```

### mTLS（双向 TLS）

**服务间通信**

```yaml
# 服务器端
server:
  grpc:
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      # 要求客户端证书
      client_auth: require_and_verify
      client_ca_file: /etc/ssl/certs/client-ca.pem
```

**客户端配置**

```go
import (
    "crypto/tls"
    "crypto/x509"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func createMTLSClient() (*grpc.ClientConn, error) {
    // 加载客户端证书
    cert, err := tls.LoadX509KeyPair(
        "/etc/ssl/certs/client-cert.pem",
        "/etc/ssl/private/client-key.pem",
    )
    if err != nil {
        return nil, err
    }
    
    // 加载 CA 证书
    caCert, err := os.ReadFile("/etc/ssl/certs/ca-cert.pem")
    if err != nil {
        return nil, err
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    // TLS 配置
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        MinVersion:   tls.VersionTLS12,
    }
    
    creds := credentials.NewTLS(tlsConfig)
    
    // 创建连接
    return grpc.Dial("service.example.com:9090",
        grpc.WithTransportCredentials(creds),
    )
}
```

### 证书管理

**证书生成**

```bash
# 生成 CA 证书
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem

# 生成服务器证书
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -in server.csr -CA ca-cert.pem -CAkey ca-key.pem -set_serial 01 -out server-cert.pem

# 生成客户端证书
openssl genrsa -out client-key.pem 4096
openssl req -new -key client-key.pem -out client.csr
openssl x509 -req -days 365 -in client.csr -CA ca-cert.pem -CAkey ca-key.pem -set_serial 02 -out client-cert.pem
```

**证书轮换**

```bash
# 使用 cert-manager（Kubernetes）
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: myservice-cert
spec:
  secretName: myservice-tls
  duration: 2160h    # 90 天
  renewBefore: 360h  # 提前 15 天续期
  issuerRef:
    name: ca-issuer
    kind: Issuer
  dnsNames:
    - myservice.example.com
```

---

## 密钥管理

### 环境变量

**基本用法**

```bash
# .env 文件（不要提交到 Git）
OAUTH2_CLIENT_SECRET=your-secret-here
JWT_HMAC_SECRET=your-jwt-secret
DATABASE_PASSWORD=your-db-password
```

```go
import "github.com/joho/godotenv"

func init() {
    // 加载 .env 文件
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
}

func main() {
    clientSecret := os.Getenv("OAUTH2_CLIENT_SECRET")
    if clientSecret == "" {
        log.Fatal("OAUTH2_CLIENT_SECRET is required")
    }
}
```

### Kubernetes Secrets

**创建 Secret**

```bash
# 从文件创建
kubectl create secret generic app-secrets \
  --from-file=oauth2-client-secret=./client-secret.txt \
  --from-file=db-password=./db-password.txt

# 从字面值创建
kubectl create secret generic app-secrets \
  --from-literal=oauth2-client-secret='your-secret' \
  --from-literal=db-password='your-password'
```

**使用 Secret**

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myservice
spec:
  template:
    spec:
      containers:
      - name: myservice
        image: myservice:latest
        env:
        - name: OAUTH2_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: oauth2-client-secret
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: db-password
```

### HashiCorp Vault

**配置**

```yaml
# swit.yaml
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
      ca_cert: /etc/ssl/certs/vault-ca.pem
```

**使用**

```go
import "github.com/innovationmech/swit/pkg/security/secrets"

// 创建 Vault 客户端
secretsManager, err := secrets.NewManager(secretsConfig)
if err != nil {
    log.Fatal(err)
}

// 获取密钥
clientSecret, err := secretsManager.GetSecret("oauth2_client_secret")
if err != nil {
    log.Fatal(err)
}

// 设置密钥
err = secretsManager.SetSecret("api_key", "new-api-key")
if err != nil {
    log.Fatal(err)
}
```

### 密钥轮换

**自动轮换策略**

```go
// 定期轮换密钥
func rotateKeys(secretsManager *secrets.Manager) {
    ticker := time.NewTicker(30 * 24 * time.Hour)  // 30 天
    defer ticker.Stop()
    
    for range ticker.C {
        // 生成新密钥
        newKey := generateSecureKey()
        
        // 保存新密钥
        if err := secretsManager.SetSecret("encryption_key_new", newKey); err != nil {
            log.Error("Failed to rotate key", zap.Error(err))
            continue
        }
        
        // 通知服务使用新密钥
        notifyKeyRotation(newKey)
        
        // 保留旧密钥一段时间以便解密旧数据
        time.Sleep(7 * 24 * time.Hour)
        
        // 删除旧密钥
        secretsManager.DeleteSecret("encryption_key_old")
    }
}
```

---

## 安全配置

### 配置验证

**启动时验证**

```go
import "github.com/innovationmech/swit/pkg/server"

func main() {
    // 加载配置
    config, err := server.LoadConfig("swit.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 验证配置
    if err := config.Validate(); err != nil {
        log.Fatal("Invalid configuration:", err)
    }
    
    // 安全检查
    if err := validateSecurityConfig(config); err != nil {
        log.Fatal("Security configuration error:", err)
    }
}

func validateSecurityConfig(config *server.ServerConfig) error {
    // 生产环境必须启用 TLS
    if config.Environment == "production" {
        if !config.HTTP.TLS.Enabled {
            return errors.New("TLS must be enabled in production")
        }
        if config.HTTP.TLS.MinVersion < "1.2" {
            return errors.New("TLS 1.2 or higher required in production")
        }
    }
    
    // 检查密钥配置
    if config.OAuth2.Enabled && config.OAuth2.ClientSecret == "" {
        return errors.New("OAuth2 client secret is required")
    }
    
    return nil
}
```

### 环境分离

**配置文件结构**

```
config/
├── swit.yaml              # 基础配置
├── swit-dev.yaml          # 开发环境
├── swit-staging.yaml      # 预发布环境
└── swit-production.yaml   # 生产环境
```

**加载配置**

```go
func LoadEnvironmentConfig() (*server.ServerConfig, error) {
    env := os.Getenv("ENVIRONMENT")
    if env == "" {
        env = "dev"
    }
    
    configFile := fmt.Sprintf("config/swit-%s.yaml", env)
    return server.LoadConfig(configFile)
}
```

### 安全默认值

**配置默认值**

```go
func NewDefaultSecurityConfig() *SecurityConfig {
    return &SecurityConfig{
        // TLS 默认启用
        TLS: &TLSConfig{
            Enabled:    true,
            MinVersion: "1.2",
        },
        
        // CORS 默认限制
        CORS: &CORSConfig{
            AllowOrigins:     []string{},  // 明确配置
            AllowCredentials: false,
        },
        
        // 速率限制默认启用
        RateLimit: &RateLimitConfig{
            Enabled:           true,
            RequestsPerSecond: 10,
            Burst:            20,
        },
        
        // 审计默认启用
        Audit: &AuditConfig{
            Enabled: true,
            Events: []string{
                "authentication_failure",
                "authorization_denied",
            },
        },
    }
}
```

---

## 安全监控与审计

### 审计日志

**配置审计**

```yaml
# swit.yaml
security:
  audit:
    enabled: true
    log_level: info
    
    # 记录的事件类型
    events:
      - authentication_success
      - authentication_failure
      - authorization_denied
      - token_issued
      - token_revoked
      - policy_evaluated
      - sensitive_data_accessed
      - password_changed
      - permission_changed
    
    # 输出配置
    outputs:
      - type: file
        path: /var/log/swit/audit.log
        format: json
        max_size: 100      # MB
        max_backups: 10
        max_age: 30        # 天
      
      - type: syslog
        network: udp
        addr: localhost:514
        tag: swit-audit
    
    # 敏感字段过滤
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
      - ssn
      - credit_card
    
    redact_policy: mask
    mask_char: "*"
```

**记录审计事件**

```go
import "github.com/innovationmech/swit/pkg/security/audit"

// 创建审计日志记录器
auditor := audit.NewLogger(auditConfig)

// 记录认证事件
auditor.Log(audit.Event{
    Type:      "authentication_failure",
    Actor:     email,
    Action:    "login",
    Resource:  "system",
    Result:    "failure",
    Reason:    "invalid password",
    Timestamp: time.Now(),
    Metadata: map[string]interface{}{
        "ip_address": clientIP,
        "user_agent": userAgent,
    },
})

// 记录授权事件
auditor.Log(audit.Event{
    Type:      "authorization_denied",
    Actor:     userID,
    Action:    "delete",
    Resource:  fmt.Sprintf("documents/%s", docID),
    Result:    "denied",
    Reason:    "insufficient permissions",
    Timestamp: time.Now(),
})

// 记录敏感数据访问
auditor.Log(audit.Event{
    Type:      "sensitive_data_accessed",
    Actor:     userID,
    Action:    "read",
    Resource:  "user/ssn",
    Result:    "success",
    Timestamp: time.Now(),
})
```

### 安全指标

**Prometheus 指标**

```yaml
# swit.yaml
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
      - jwt_cache_hits
      - jwt_cache_misses
    
    labels:
      service: my-service
      environment: production
```

**查询指标**

```promql
# 认证失败率
rate(swit_security_authentication_errors_total[5m])

# 授权拒绝率
rate(swit_security_authorization_denied_total[5m])

# 令牌验证延迟
histogram_quantile(0.95, 
  rate(swit_security_token_validation_duration_bucket[5m]))

# 策略评估延迟
histogram_quantile(0.99, 
  rate(swit_security_policy_evaluation_duration_bucket[5m]))
```

**告警规则**

```yaml
# prometheus-alerts.yaml
groups:
  - name: security
    rules:
      - alert: HighAuthenticationFailureRate
        expr: |
          rate(swit_security_authentication_errors_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate"
          description: "More than 10 authentication failures per second"
      
      - alert: FrequentAuthorizationDenials
        expr: |
          rate(swit_security_authorization_denied_total[5m]) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Frequent authorization denials"
          description: "Possible unauthorized access attempts"
      
      - alert: SlowPolicyEvaluation
        expr: |
          histogram_quantile(0.95,
            rate(swit_security_policy_evaluation_duration_bucket[5m])
          ) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Slow policy evaluation"
          description: "95th percentile policy evaluation > 1s"
```

### 安全监控

**实时监控**

```go
import "github.com/innovationmech/swit/pkg/security/metrics"

// 创建安全指标收集器
metricsCollector := metrics.NewCollector(metricsConfig)

// 中间件记录指标
func securityMetricsMiddleware(collector *metrics.Collector) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start)
        
        // 记录请求指标
        collector.RecordRequest(metrics.RequestMetrics{
            Path:       c.Request.URL.Path,
            Method:     c.Request.Method,
            StatusCode: c.Writer.Status(),
            Duration:   duration,
            UserID:     c.GetString("user_id"),
        })
        
        // 记录认证指标
        if authenticated := c.GetBool("authenticated"); authenticated {
            collector.IncAuthentications("success")
        } else {
            collector.IncAuthentications("failure")
        }
    }
}
```

---

## 容器安全

### Dockerfile 安全

**最佳实践**

```dockerfile
# 使用特定版本的基础镜像
FROM golang:1.21-alpine3.18 AS builder

# 以非 root 用户运行
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" -o /app/myservice ./cmd/myservice

# 最小化运行时镜像
FROM alpine:3.18

# 安装 CA 证书
RUN apk --no-cache add ca-certificates

# 创建非 root 用户
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder --chown=appuser:appgroup /app/myservice .

# 复制配置文件
COPY --chown=appuser:appgroup swit.yaml .

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/myservice", "health"]

# 运行应用
ENTRYPOINT ["/app/myservice"]
```

### 镜像扫描

**CI/CD 集成**

```yaml
# .github/workflows/security.yml
name: Security Scan

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: docker build -t myservice:${{ github.sha }} .
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: myservice:${{ github.sha }}
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'
      
      - name: Upload Trivy results to GitHub Security
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'
```

**本地扫描**

```bash
# 扫描镜像
trivy image myservice:latest

# 扫描 Dockerfile
trivy config Dockerfile

# 扫描并生成报告
trivy image --format json --output results.json myservice:latest
```

### Kubernetes 安全

**Pod 安全策略**

```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
```

**Security Context**

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myservice
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      
      containers:
      - name: myservice
        image: myservice:latest
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          capabilities:
            drop:
              - ALL
        
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
          requests:
            cpu: "100m"
            memory: "128Mi"
```

**Network Policy**

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: myservice-netpol
spec:
  podSelector:
    matchLabels:
      app: myservice
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            name: production
      ports:
      - protocol: TCP
        port: 8080
  egress:
    - to:
      - podSelector:
          matchLabels:
            app: database
      ports:
      - protocol: TCP
        port: 5432
```

---

## 依赖管理

### 依赖扫描

**Makefile 集成**

```makefile
# 安全扫描
.PHONY: security
security: security-gosec security-trivy security-govulncheck

# gosec - Go 安全扫描
.PHONY: security-gosec
security-gosec:
	@echo "Running gosec..."
	@gosec -fmt=json -out=gosec-report.json ./...

# Trivy - 漏洞扫描
.PHONY: security-trivy
security-trivy:
	@echo "Running trivy..."
	@trivy fs --format json --output trivy-report.json .

# govulncheck - Go 漏洞数据库检查
.PHONY: security-govulncheck
security-govulncheck:
	@echo "Running govulncheck..."
	@govulncheck ./...
```

**定期更新**

```bash
# 检查可更新的依赖
go list -u -m all

# 更新所有依赖到最新次要版本
go get -u ./...

# 更新特定依赖
go get github.com/gin-gonic/gin@latest

# 清理未使用的依赖
go mod tidy

# 验证依赖
go mod verify
```

### 依赖锁定

**使用 go.sum**

```bash
# go.sum 文件锁定依赖版本
# 确保 go.sum 提交到版本控制

# 验证 go.sum
go mod verify

# 如果 go.sum 不存在，重新生成
go mod tidy
```

### 私有依赖

**配置私有仓库**

```bash
# 配置 GOPRIVATE
export GOPRIVATE=github.com/myorg/*

# 配置 Git 凭据
git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
```

---

## 安全测试

### 单元测试

**安全相关测试**

```go
package security_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPasswordValidation(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {
            name:     "valid password",
            password: "SecureP@ssw0rd!",
            wantErr:  false,
        },
        {
            name:     "too short",
            password: "Short1!",
            wantErr:  true,
        },
        {
            name:     "no special char",
            password: "LongPassword123",
            wantErr:  true,
        },
        {
            name:     "no number",
            password: "LongPassword!@#",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestXSSPrevention(t *testing.T) {
    maliciousInputs := []string{
        "<script>alert('XSS')</script>",
        "<img src=x onerror=alert('XSS')>",
        "javascript:alert('XSS')",
    }
    
    for _, input := range maliciousInputs {
        output := SanitizeInput(input)
        assert.NotContains(t, output, "<script>")
        assert.NotContains(t, output, "onerror")
        assert.NotContains(t, output, "javascript:")
    }
}

func TestSQLInjectionPrevention(t *testing.T) {
    maliciousInputs := []string{
        "' OR '1'='1",
        "'; DROP TABLE users--",
        "1' UNION SELECT * FROM users--",
    }
    
    for _, input := range maliciousInputs {
        // 使用参数化查询应该安全
        var count int64
        err := db.Model(&User{}).Where("email = ?", input).Count(&count).Error
        assert.NoError(t, err)
        assert.Equal(t, int64(0), count)
    }
}
```

### 集成测试

**认证测试**

```go
func TestOAuth2Authentication(t *testing.T) {
    // 设置测试服务器
    router := setupTestRouter()
    
    // 测试未认证访问
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/protected", nil)
    router.ServeHTTP(w, req)
    assert.Equal(t, 401, w.Code)
    
    // 测试无效令牌
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("GET", "/api/v1/protected", nil)
    req.Header.Set("Authorization", "Bearer invalid-token")
    router.ServeHTTP(w, req)
    assert.Equal(t, 401, w.Code)
    
    // 测试有效令牌
    validToken := generateTestToken()
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("GET", "/api/v1/protected", nil)
    req.Header.Set("Authorization", "Bearer "+validToken)
    router.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}
```

**授权测试**

```go
func TestOPAAuthorization(t *testing.T) {
    tests := []struct {
        name       string
        user       string
        roles      []string
        action     string
        resource   string
        wantAllow  bool
    }{
        {
            name:      "admin can delete",
            user:      "admin",
            roles:     []string{"admin"},
            action:    "delete",
            resource:  "documents/1",
            wantAllow: true,
        },
        {
            name:      "viewer cannot write",
            user:      "user1",
            roles:     []string{"viewer"},
            action:    "write",
            resource:  "documents/1",
            wantAllow: false,
        },
        {
            name:      "editor can write",
            user:      "user2",
            roles:     []string{"editor"},
            action:    "write",
            resource:  "documents/1",
            wantAllow: true,
        },
    }
    
    opaClient := setupTestOPA()
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            input := map[string]interface{}{
                "subject": map[string]interface{}{
                    "user":  tt.user,
                    "roles": tt.roles,
                },
                "action":   tt.action,
                "resource": tt.resource,
            }
            
            result, err := opaClient.Evaluate(context.Background(), input)
            assert.NoError(t, err)
            assert.Equal(t, tt.wantAllow, result.Allow)
        })
    }
}
```

### 渗透测试

**OWASP ZAP 集成**

```yaml
# zap-scan.yml
name: OWASP ZAP Scan

on:
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨2点

jobs:
  zap_scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Start application
        run: |
          docker-compose up -d
          sleep 30
      
      - name: ZAP Baseline Scan
        uses: zaproxy/action-baseline@v0.7.0
        with:
          target: 'http://localhost:8080'
          rules_file_name: '.zap/rules.tsv'
          cmd_options: '-a'
      
      - name: ZAP Full Scan
        uses: zaproxy/action-full-scan@v0.4.0
        with:
          target: 'http://localhost:8080'
          rules_file_name: '.zap/rules.tsv'
```

---

## 相关资源

### 官方文档

- [配置参考](./api/security-config.md)
- [OAuth2/OIDC API](./api/security-oauth2.md)
- [OPA 策略 API](./api/security-opa.md)
- [OPA RBAC 指南](./opa-rbac-guide.md)
- [OPA ABAC 指南](./opa-abac-guide.md)
- [安全检查清单](./security-checklist.md)
- [事件响应流程](./security-incident-response.md)

### 外部资源

**OWASP 资源**
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Go Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Go_SCP_Cheat_Sheet.html)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)

**Go 安全**
- [Go Security Policy](https://golang.org/security)
- [Go Vulnerability Database](https://pkg.go.dev/vuln/)
- [Effective Go - Security](https://golang.org/doc/effective_go)

**标准与合规**
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [PCI DSS](https://www.pcisecuritystandards.org/)
- [GDPR](https://gdpr.eu/)

**工具文档**
- [gosec Documentation](https://github.com/securego/gosec)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [OPA Documentation](https://www.openpolicyagent.org/docs/)
- [OAuth 2.0 RFC](https://oauth.net/2/)
- [OpenID Connect](https://openid.net/connect/)

---

## 许可证

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.










