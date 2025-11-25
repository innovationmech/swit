# Middleware

This guide covers the middleware system in Swit, which provides configurable request processing pipelines for HTTP and gRPC transports, including authentication, authorization, rate limiting, CORS, and security features.

## Overview

The middleware layer (`pkg/middleware`) provides a comprehensive set of middleware components for both HTTP (Gin) and gRPC transports. It includes built-in support for security, observability, and traffic management.

## HTTP Middleware

### Authentication Middleware

#### OAuth2/OIDC Authentication

```go
import (
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/innovationmech/swit/pkg/middleware"
)

// Create OAuth2 client
oauth2Client, err := oauth2.NewClient(&oauth2.Config{
    Provider:     "keycloak",
    ClientID:     "my-service",
    ClientSecret: os.Getenv("OAUTH2_CLIENT_SECRET"),
    IssuerURL:    "https://auth.example.com/realms/production",
    UseDiscovery: true,
})

// Create authentication middleware
authMiddleware := middleware.NewOAuth2Middleware(oauth2Client)

// Apply to routes
router.Use(authMiddleware.Authenticate())

// Or apply to specific route groups
protected := router.Group("/api/v1")
protected.Use(authMiddleware.Authenticate())
{
    protected.GET("/users", getUsers)
    protected.POST("/users", createUser)
}
```

#### JWT Authentication

```go
import (
    "github.com/innovationmech/swit/pkg/security/jwt"
    "github.com/innovationmech/swit/pkg/middleware"
)

// Create JWT validator
validator, err := jwt.NewValidator(&jwt.ValidatorConfig{
    JWKSConfig: &jwt.JWKSConfig{
        URL:          "https://auth.example.com/.well-known/jwks.json",
        CacheEnabled: true,
        CacheTTL:     time.Hour,
    },
    Issuer:   "https://auth.example.com",
    Audience: "my-service-api",
})

// Create JWT middleware
jwtMiddleware := middleware.NewJWTMiddleware(validator)

// Apply to routes
router.Use(jwtMiddleware.Authenticate())
```

### Authorization Middleware

#### OPA Policy Authorization

```go
import (
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/innovationmech/swit/pkg/middleware"
)

// Create OPA client
opaClient, err := opa.NewClient(&opa.Config{
    Mode:                "embedded",
    DefaultDecisionPath: "authz/allow",
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
})

// Create authorization middleware
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
        c.JSON(403, gin.H{"error": "access denied"})
        c.Abort()
    },
})

// Apply after authentication
router.Use(authMiddleware.Authenticate())
router.Use(authzMiddleware.Authorize())
```

### Rate Limiting Middleware

```go
import "github.com/innovationmech/swit/pkg/middleware"

// IP-based rate limiting
rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 100,
    Burst:            200,
    KeyFunc: func(c *gin.Context) string {
        return c.ClientIP()
    },
})

router.Use(rateLimiter)

// User-based rate limiting
userRateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 1000,
    Burst:            2000,
    KeyFunc: func(c *gin.Context) string {
        return "user:" + c.GetString("user_id")
    },
})

// Apply after authentication
router.Use(authMiddleware.Authenticate())
router.Use(userRateLimiter)
```

### CORS Middleware

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

### Timeout Middleware

```go
import "github.com/innovationmech/swit/pkg/middleware"

timeoutMiddleware := middleware.NewTimeoutMiddleware(middleware.TimeoutConfig{
    RequestTimeout: 30 * time.Second,
    HandlerTimeout: 25 * time.Second,
})

router.Use(timeoutMiddleware)
```

### Request Logging Middleware

```go
import "github.com/innovationmech/swit/pkg/middleware"

loggingMiddleware := middleware.NewLoggingMiddleware(middleware.LoggingConfig{
    SkipPaths: []string{"/health", "/metrics"},
    LogHeaders: false,
    LogBody:    false,
})

router.Use(loggingMiddleware)
```

### Security Headers Middleware

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

## gRPC Interceptors

### Authentication Interceptor

```go
import (
    "github.com/innovationmech/swit/pkg/middleware"
    "google.golang.org/grpc"
)

// Create authentication interceptor
authInterceptor := middleware.NewGRPCAuthInterceptor(oauth2Client)

// Apply to gRPC server
server := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor.UnaryInterceptor()),
    grpc.StreamInterceptor(authInterceptor.StreamInterceptor()),
)
```

### Authorization Interceptor

```go
authzInterceptor := middleware.NewGRPCOPAInterceptor(opaClient)

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        authInterceptor.UnaryInterceptor(),
        authzInterceptor.UnaryInterceptor(),
    ),
)
```

### Logging Interceptor

```go
loggingInterceptor := middleware.NewGRPCLoggingInterceptor()

server := grpc.NewServer(
    grpc.UnaryInterceptor(loggingInterceptor.UnaryInterceptor()),
)
```

### Recovery Interceptor

```go
recoveryInterceptor := middleware.NewGRPCRecoveryInterceptor()

server := grpc.NewServer(
    grpc.UnaryInterceptor(recoveryInterceptor.UnaryInterceptor()),
)
```

## Middleware Chain

### HTTP Middleware Chain

```go
// Recommended middleware order
router := gin.New()

// 1. Recovery (catch panics)
router.Use(gin.Recovery())

// 2. Request ID
router.Use(middleware.NewRequestIDMiddleware())

// 3. Logging
router.Use(middleware.NewLoggingMiddleware(loggingConfig))

// 4. Security headers
router.Use(middleware.NewSecurityHeadersMiddleware(securityConfig))

// 5. CORS
router.Use(cors.New(corsConfig))

// 6. Rate limiting
router.Use(middleware.NewRateLimiter(rateLimitConfig))

// 7. Timeout
router.Use(middleware.NewTimeoutMiddleware(timeoutConfig))

// 8. Authentication
router.Use(authMiddleware.Authenticate())

// 9. Authorization
router.Use(authzMiddleware.Authorize())
```

### gRPC Interceptor Chain

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

## Configuration

### YAML Configuration

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

## Custom Middleware

### Creating Custom HTTP Middleware

```go
func CustomMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Before request
        startTime := time.Now()
        
        // Process request
        c.Next()
        
        // After request
        duration := time.Since(startTime)
        log.Printf("Request took %v", duration)
    }
}

router.Use(CustomMiddleware())
```

### Creating Custom gRPC Interceptor

```go
func CustomUnaryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // Before RPC
        startTime := time.Now()
        
        // Process RPC
        resp, err := handler(ctx, req)
        
        // After RPC
        duration := time.Since(startTime)
        log.Printf("RPC %s took %v", info.FullMethod, duration)
        
        return resp, err
    }
}
```

## Best Practices

1. **Order Matters**: Apply middleware in the correct order (recovery → logging → auth → business logic)
2. **Use Timeout**: Always set request timeouts to prevent resource exhaustion
3. **Rate Limit Early**: Apply rate limiting before expensive operations
4. **Authenticate Before Authorize**: Always authenticate before authorization checks
5. **Log Appropriately**: Don't log sensitive data (passwords, tokens)
6. **Handle Errors Gracefully**: Use recovery middleware to catch panics
7. **Configure CORS Properly**: Don't use `*` for origins in production

## Related Documentation

- [Security Best Practices](/guide/security-best-practices)
- [OAuth2 Integration](/guide/oauth2-integration)
- [OPA Policy Guide](/guide/opa-policy)
- [Transport Layer](/guide/transport-layer)
- [Configuration Reference](/guide/configuration-reference)


