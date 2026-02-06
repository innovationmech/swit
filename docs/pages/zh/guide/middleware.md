# 中间件

本指南介绍 Swit 中的中间件系统，它为 HTTP 和 gRPC 传输提供可配置的请求处理管道，包括身份认证、授权、速率限制、CORS 和安全功能。

## 概述

中间件层（`pkg/middleware`）为 HTTP（Gin）和 gRPC 传输提供了一套全面的中间件组件。它内置支持安全、可观测性和流量管理。

## HTTP 中间件

### 认证中间件

#### OAuth2/OIDC 认证

```go
import (
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/innovationmech/swit/pkg/middleware"
)

// 创建 OAuth2 客户端
oauth2Client, err := oauth2.NewClient(&oauth2.Config{
    Provider:     "keycloak",
    ClientID:     "my-service",
    ClientSecret: os.Getenv("OAUTH2_CLIENT_SECRET"),
    IssuerURL:    "https://auth.example.com/realms/production",
    UseDiscovery: true,
})

// 创建认证中间件
authMiddleware := middleware.NewOAuth2Middleware(oauth2Client)

// 应用到路由
router.Use(authMiddleware.Authenticate())

// 或应用到特定路由组
protected := router.Group("/api/v1")
protected.Use(authMiddleware.Authenticate())
{
    protected.GET("/users", getUsers)
    protected.POST("/users", createUser)
}
```

#### JWT 认证

```go
import (
    "github.com/innovationmech/swit/pkg/security/jwt"
    "github.com/innovationmech/swit/pkg/middleware"
)

// 创建 JWT 验证器
validator, err := jwt.NewValidator(&jwt.ValidatorConfig{
    JWKSConfig: &jwt.JWKSConfig{
        URL:          "https://auth.example.com/.well-known/jwks.json",
        CacheEnabled: true,
        CacheTTL:     time.Hour,
    },
    Issuer:   "https://auth.example.com",
    Audience: "my-service-api",
})

// 创建 JWT 中间件
jwtMiddleware := middleware.NewJWTMiddleware(validator)

// 应用到路由
router.Use(jwtMiddleware.Authenticate())
```

### 授权中间件

#### OPA 策略授权

```go
import (
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/innovationmech/swit/pkg/middleware"
)

// 创建 OPA 客户端
opaClient, err := opa.NewClient(&opa.Config{
    Mode:                "embedded",
    DefaultDecisionPath: "authz/allow",
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
})

// 创建授权中间件
authzMiddleware := middleware.NewOPAMiddleware(opaClient, middleware.OPAMiddlewareConfig{
    InputBuilder: func(c *gin.Context) map[string]interface{} {
        return map[string]interface{}{
            "subject": map[string]interface{}{
                "user":  c.GetString("user_id"),
                "roles": c.GetStringSlice("roles"),
            },
            "action":   c.Request.Method,
            "resource": c.Request.URL.Path,
        }
    },
    OnDeny: func(c *gin.Context) {
        c.JSON(403, gin.H{"error": "访问被拒绝"})
        c.Abort()
    },
})

// 在认证之后应用
router.Use(authMiddleware.Authenticate())
router.Use(authzMiddleware.Authorize())
```

### 速率限制中间件

```go
import "github.com/innovationmech/swit/pkg/middleware"

// 基于 IP 的速率限制
rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 100,
    Burst:            200,
    KeyFunc: func(c *gin.Context) string {
        return c.ClientIP()
    },
})

router.Use(rateLimiter)

// 基于用户的速率限制
userRateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 1000,
    Burst:            2000,
    KeyFunc: func(c *gin.Context) string {
        return "user:" + c.GetString("user_id")
    },
})

// 在认证之后应用
router.Use(authMiddleware.Authenticate())
router.Use(userRateLimiter)
```

### CORS 中间件

```go
import "github.com/gin-contrib/cors"

corsConfig := cors.Config{
    AllowOrigins: []string{
        "https://app.example.com",
        "https://admin.example.com",
    },
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders: []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,
}

router.Use(cors.New(corsConfig))
```

### 超时中间件

```go
import "github.com/innovationmech/swit/pkg/middleware"

timeoutMiddleware := middleware.NewTimeoutMiddleware(middleware.TimeoutConfig{
    RequestTimeout: 30 * time.Second,
    HandlerTimeout: 25 * time.Second,
})

router.Use(timeoutMiddleware)
```

### 请求日志中间件

```go
import "github.com/innovationmech/swit/pkg/middleware"

loggingMiddleware := middleware.NewLoggingMiddleware(middleware.LoggingConfig{
    SkipPaths: []string{"/health", "/metrics"},
    LogHeaders: false,
    LogBody:    false,
})

router.Use(loggingMiddleware)
```

### 安全头部中间件

```go
import "github.com/innovationmech/swit/pkg/middleware"

securityHeaders := middleware.NewSecurityHeadersMiddleware(middleware.SecurityHeadersConfig{
    ContentSecurityPolicy: "default-src 'self'",
    XContentTypeOptions:   "nosniff",
    XFrameOptions:         "DENY",
    XSSProtection:         "1; mode=block",
})

router.Use(securityHeaders)
```

## gRPC 拦截器

### 认证拦截器

```go
import (
    "github.com/innovationmech/swit/pkg/middleware"
    "google.golang.org/grpc"
)

// 创建认证拦截器
authInterceptor := middleware.NewGRPCAuthInterceptor(oauth2Client)

// 应用到 gRPC 服务器
server := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor.UnaryInterceptor()),
    grpc.StreamInterceptor(authInterceptor.StreamInterceptor()),
)
```

### 授权拦截器

```go
authzInterceptor := middleware.NewGRPCOPAInterceptor(opaClient)

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        authInterceptor.UnaryInterceptor(),
        authzInterceptor.UnaryInterceptor(),
    ),
)
```

### 日志拦截器

```go
loggingInterceptor := middleware.NewGRPCLoggingInterceptor()

server := grpc.NewServer(
    grpc.UnaryInterceptor(loggingInterceptor.UnaryInterceptor()),
)
```

### 恢复拦截器

```go
recoveryInterceptor := middleware.NewGRPCRecoveryInterceptor()

server := grpc.NewServer(
    grpc.UnaryInterceptor(recoveryInterceptor.UnaryInterceptor()),
)
```

## 中间件链

### HTTP 中间件链

```go
// 推荐的中间件顺序
router := gin.New()

// 1. 恢复（捕获 panic）
router.Use(gin.Recovery())

// 2. 请求 ID
router.Use(middleware.NewRequestIDMiddleware())

// 3. 日志
router.Use(middleware.NewLoggingMiddleware(loggingConfig))

// 4. 安全头部
router.Use(middleware.NewSecurityHeadersMiddleware(securityConfig))

// 5. CORS
router.Use(cors.New(corsConfig))

// 6. 速率限制
router.Use(middleware.NewRateLimiter(rateLimitConfig))

// 7. 超时
router.Use(middleware.NewTimeoutMiddleware(timeoutConfig))

// 8. 认证
router.Use(authMiddleware.Authenticate())

// 9. 授权
router.Use(authzMiddleware.Authorize())
```

### gRPC 拦截器链

```go
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        recoveryInterceptor.UnaryInterceptor(),
        loggingInterceptor.UnaryInterceptor(),
        rateLimitInterceptor.UnaryInterceptor(),
        authInterceptor.UnaryInterceptor(),
        authzInterceptor.UnaryInterceptor(),
    ),
    grpc.ChainStreamInterceptor(
        recoveryInterceptor.StreamInterceptor(),
        loggingInterceptor.StreamInterceptor(),
        authInterceptor.StreamInterceptor(),
        authzInterceptor.StreamInterceptor(),
    ),
)
```

## 配置

### YAML 配置

```yaml
http:
  middleware:
    enable_cors: true
    enable_auth: true
    enable_rate_limit: true
    enable_logging: true
    enable_timeout: true
    
    cors:
      allow_origins:
        - "https://app.example.com"
      allow_methods:
        - "GET"
        - "POST"
        - "PUT"
        - "DELETE"
      allow_headers:
        - "Origin"
        - "Content-Type"
        - "Authorization"
      allow_credentials: true
      max_age: 86400
    
    rate_limit:
      requests_per_second: 100
      burst_size: 200
      key_func: "ip"
    
    timeout:
      request_timeout: 30s
      handler_timeout: 25s

grpc:
  interceptors:
    enable_auth: true
    enable_logging: true
    enable_recovery: true
    enable_rate_limit: false
```

## 自定义中间件

### 创建自定义 HTTP 中间件

```go
func CustomMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 请求前
        startTime := time.Now()
        
        // 处理请求
        c.Next()
        
        // 请求后
        duration := time.Since(startTime)
        log.Printf("请求耗时 %v", duration)
    }
}

router.Use(CustomMiddleware())
```

### 创建自定义 gRPC 拦截器

```go
func CustomUnaryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // RPC 前
        startTime := time.Now()
        
        // 处理 RPC
        resp, err := handler(ctx, req)
        
        // RPC 后
        duration := time.Since(startTime)
        log.Printf("RPC %s 耗时 %v", info.FullMethod, duration)
        
        return resp, err
    }
}
```

## 最佳实践

1. **顺序很重要**：按正确顺序应用中间件（恢复 → 日志 → 认证 → 业务逻辑）
2. **使用超时**：始终设置请求超时以防止资源耗尽
3. **提前限流**：在昂贵操作之前应用速率限制
4. **先认证后授权**：始终在授权检查之前进行认证
5. **适当记录日志**：不要记录敏感数据（密码、令牌）
6. **优雅处理错误**：使用恢复中间件捕获 panic
7. **正确配置 CORS**：生产环境不要使用 `*` 作为来源

## 相关文档

- [安全最佳实践](/zh/guide/security-best-practices)
- [OAuth2 集成](/zh/guide/oauth2-integration)
- [OPA 策略指南](/zh/guide/opa-policy)
- [传输层](/zh/guide/transport-layer)
- [配置参考](/zh/guide/configuration-reference)








