# OAuth2/OIDC Integration Guide

This guide will help you integrate OAuth2/OIDC authentication in the Swit framework, with support for major providers (Keycloak, Auth0, Google, Microsoft, Okta, and more).

## Overview

The Swit framework provides comprehensive OAuth2/OIDC authentication support with the following key features:

### Core Features

- ✅ **Multi-Provider Support** - Keycloak, Auth0, Google, Microsoft, Okta, and more
- ✅ **OIDC Auto-Discovery** - Automatically discover endpoints from issuer URL
- ✅ **Multiple Authorization Flows** - Authorization Code, PKCE, Client Credentials, Password, Refresh Token
- ✅ **JWT Token Validation** - Local JWT signature verification and remote token introspection
- ✅ **Token Caching** - Built-in token cache for improved performance
- ✅ **TLS/mTLS Support** - Complete transport layer security configuration
- ✅ **Gin Middleware** - Ready-to-use route protection middleware
- ✅ **Role-Based Access Control** - Integrated with OPA for RBAC/ABAC
- ✅ **Health Checks** - Built-in provider connection health checks

### Supported Providers

| Provider | Type ID | OIDC Discovery | Default Scopes |
|----------|---------|---------------|----------------|
| Keycloak | `keycloak` | ✅ | openid, profile, email |
| Auth0 | `auth0` | ✅ | openid, profile, email |
| Google | `google` | ✅ | openid, profile, email |
| Microsoft | `microsoft` | ✅ | openid, profile, email |
| Okta | `okta` | ✅ | openid, profile, email |
| Custom | `custom` | Optional | Custom |

---

## Quick Start

### Prerequisites

- Go 1.23+
- Running OAuth2 provider (this guide uses Keycloak)
- Docker and Docker Compose (optional, for quick demo)

### 1. Quick Demo with Docker Compose

The fastest way to get started is with our complete example:

```bash
# Clone the repository
git clone https://github.com/innovationmech/swit.git
cd swit/examples/oauth2-authentication

# Start all services (Keycloak + PostgreSQL + Example Service)
docker-compose up -d

# Wait for services to be ready (~60 seconds)
docker-compose ps
```

This starts:
- **Keycloak** (http://localhost:8081) - OIDC Provider
- **PostgreSQL** (localhost:5432) - Keycloak Database
- **OAuth2 Example Service** (http://localhost:8080) - Demo Application

Keycloak comes pre-configured with:
- Realm: `swit`
- Client: `swit-example` (secret: `swit-example-secret`)
- Users: `testuser`/`password`, `admin`/`admin`

### 2. Test Authentication Flow

#### Step 1: Get Service Information

```bash
curl http://localhost:8080/api/v1/public/info
```

#### Step 2: Initiate Login

```bash
curl http://localhost:8080/api/v1/public/login
```

Copy the returned `authorization_url` and open it in your browser.

#### Step 3: Login with Keycloak

1. Enter credentials: `testuser` / `password`
2. After authorization, redirect to callback URL
3. Copy the returned `access_token`

#### Step 4: Access Protected Endpoint

```bash
export ACCESS_TOKEN="your-access-token-here"

curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/protected/profile
```

### 3. Basic Code Example

Here's a minimal integration example:

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
    // 1. Configure OAuth2 client
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
    
    // 2. Create OAuth2 client
    ctx := context.Background()
    oauth2Client, err := oauth2.NewClient(ctx, oauth2Config)
    if err != nil {
        log.Fatalf("Failed to create OAuth2 client: %v", err)
    }
    defer oauth2Client.Close()
    
    // 3. Create JWT validator (for local token validation)
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
    
    // 4. Create Gin router with middleware
    router := gin.Default()
    
    // Public endpoints
    router.GET("/login", func(c *gin.Context) {
        authURL := oauth2Client.AuthCodeURL("random-state")
        c.JSON(200, gin.H{"authorization_url": authURL})
    })
    
    // Protected endpoints
    protected := router.Group("/protected")
    protected.Use(middleware.OAuth2Middleware(oauth2Client, jwtValidator))
    {
        protected.GET("/profile", func(c *gin.Context) {
            userInfo, _ := middleware.GetUserInfo(c)
            c.JSON(200, userInfo)
        })
    }
    
    // Start server
    router.Run(":8080")
}
```

---

## Configuration Guide

### Complete Configuration Example

Create a `swit.yaml` configuration file:

```yaml
service_name: "my-oauth2-service"

# OAuth2/OIDC Configuration
oauth2:
  # Basic configuration
  enabled: true
  provider: keycloak  # keycloak, auth0, google, microsoft, okta, custom
  client_id: my-app
  client_secret: ${OAUTH2_CLIENT_SECRET}  # Use environment variable
  redirect_url: http://localhost:8080/callback
  
  # OIDC discovery configuration
  issuer_url: http://localhost:8081/realms/myrealm
  use_discovery: true  # Enable automatic endpoint discovery
  
  # OAuth2 scopes
  scopes:
    - openid
    - profile
    - email
    - offline_access  # For refresh tokens
  
  # HTTP timeout
  http_timeout: 30s
  
  # JWT token validation configuration
  jwt:
    signing_method: RS256  # RS256, RS384, RS512, HS256, ES256
    clock_skew: 30s        # Clock skew tolerance
    audience: my-app       # Expected audience claim
    issuer: http://localhost:8081/realms/myrealm
    required_claims:       # Required claims
      - sub
      - email
  
  # Token cache configuration
  cache:
    enabled: true
    type: memory
    ttl: 10m              # Cache expiration time
    max_size: 1000        # Maximum cache entries
    cleanup_interval: 5m  # Cleanup interval
  
  # TLS configuration (required for production)
  tls:
    enabled: true
    insecure_skip_verify: false  # Must be false in production
    min_version: TLS1.2
```

### Manual Endpoint Configuration (Without OIDC Discovery)

If your provider doesn't support OIDC discovery, configure endpoints manually:

```yaml
oauth2:
  enabled: true
  provider: custom
  client_id: my-app
  client_secret: ${OAUTH2_CLIENT_SECRET}
  redirect_url: http://localhost:8080/callback
  
  # Manual endpoint configuration
  use_discovery: false
  auth_url: https://auth.example.com/oauth2/authorize
  token_url: https://auth.example.com/oauth2/token
  user_info_url: https://auth.example.com/oauth2/userinfo
  jwks_url: https://auth.example.com/oauth2/jwks
  
  scopes:
    - openid
    - profile
```

### Environment Variable Override

All configuration can be overridden with environment variables:

```bash
# OAuth2 basic configuration
export OAUTH2_ENABLED=true
export OAUTH2_PROVIDER=keycloak
export OAUTH2_CLIENT_ID=my-app
export OAUTH2_CLIENT_SECRET=my-secret
export OAUTH2_ISSUER_URL=http://localhost:8081/realms/swit
export OAUTH2_REDIRECT_URL=http://localhost:8080/callback

# JWT configuration
export OAUTH2_JWT_SIGNING_METHOD=RS256
export OAUTH2_JWT_AUDIENCE=my-app

# Cache configuration
export OAUTH2_CACHE_ENABLED=true
export OAUTH2_CACHE_TTL=10m
```

---

## Multi-Provider Integration

### Keycloak Integration

Keycloak is the recommended open-source OAuth2/OIDC provider.

#### 1. Keycloak Server Setup

Quick start with Docker:

```bash
docker run -d \
  --name keycloak \
  -p 8081:8080 \
  -e KEYCLOAK_ADMIN=admin \
  -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:latest \
  start-dev
```

#### 2. Create Realm and Client

Visit http://localhost:8081/admin and login with `admin`/`admin`:

1. **Create Realm**
   - Click the dropdown menu in top-left corner, select "Create Realm"
   - Name: `swit`
   - Click "Create"

2. **Create Client**
   - Go to Clients page, click "Create client"
   - Client ID: `swit-app`
   - Client Protocol: `openid-connect`
   - Click "Next"

3. **Configure Client Settings**
   - Client authentication: `On` (confidential client)
   - Authorization: `Off`
   - Authentication flow: Check `Standard flow` and `Direct access grants`
   - Click "Next"

4. **Configure Redirect URIs**
   - Valid redirect URIs: `http://localhost:8080/*`
   - Web origins: `http://localhost:8080`
   - Click "Save"

5. **Get Client Secret**
   - Go to "Credentials" tab
   - Copy the "Client secret" value

#### 3. Configure Swit Application

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

### Auth0 Integration

#### 1. Create Auth0 Application

1. Login to [Auth0 Dashboard](https://manage.auth0.com/)
2. Applications → Create Application
3. Select "Regular Web Applications"
4. Note the Domain, Client ID, and Client Secret

#### 2. Configure Application Settings

- **Allowed Callback URLs**: `http://localhost:8080/callback`
- **Allowed Logout URLs**: `http://localhost:8080`
- **Allowed Web Origins**: `http://localhost:8080`

#### 3. Configure Swit Application

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

### Google Integration

#### 1. Create Google OAuth2 Credentials

1. Visit [Google Cloud Console](https://console.cloud.google.com/)
2. APIs & Services → Credentials
3. Create Credentials → OAuth client ID
4. Application type: Web application
5. Authorized redirect URIs: `http://localhost:8080/callback`

#### 2. Configure Swit Application

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

### Microsoft (Azure AD) Integration

#### 1. Register Azure AD Application

1. Visit [Azure Portal](https://portal.azure.com/)
2. Azure Active Directory → App registrations → New registration
3. Redirect URI: `http://localhost:8080/callback`
4. Certificates & secrets → New client secret

#### 2. Configure Swit Application

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

### Okta Integration

#### 1. Create Okta Application

1. Login to [Okta Admin Console](https://your-domain.okta.com/admin)
2. Applications → Create App Integration
3. Sign-in method: OIDC
4. Application type: Web Application
5. Sign-in redirect URIs: `http://localhost:8080/callback`

#### 2. Configure Swit Application

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

## Authorization Flows

The Swit framework supports multiple OAuth2 authorization flows.

### Authorization Code Flow

The most secure and commonly used flow for web applications.

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
        // Generate random state for CSRF protection
        state := generateRandomString(32)
        
        // Save state (use session or Redis in production)
        flowManager.SaveState(state, "user-session-id")
        
        // Generate authorization URL
        authURL := oauth2Client.AuthCodeURL(state)
        
        c.JSON(200, gin.H{
            "authorization_url": authURL,
            "state": state,
        })
    }
}

func handleCallback(oauth2Client *oauth2.Client, flowManager *oauth2.FlowManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get authorization code and state
        code := c.Query("code")
        state := c.Query("state")
        
        // Validate state
        if !flowManager.ValidateState(state) {
            c.JSON(400, gin.H{"error": "invalid_state"})
            return
        }
        
        // Exchange authorization code for tokens
        ctx := context.Background()
        token, err := oauth2Client.Exchange(ctx, code)
        if err != nil {
            c.JSON(500, gin.H{"error": "token_exchange_failed"})
            return
        }
        
        // Return tokens (save to session in production)
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

### Authorization Code Flow with PKCE

PKCE (Proof Key for Code Exchange) adds an extra security layer, recommended for public clients (SPAs, mobile apps).

```go
import (
    "crypto/sha256"
    "golang.org/x/oauth2"
)

func handleLoginWithPKCE(oauth2Client *oauth2.Client, flowManager *oauth2.FlowManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Generate PKCE parameters
        verifier := generateRandomString(64)
        challenge := generateCodeChallenge(verifier)
        
        state := generateRandomString(32)
        
        // Save state and verifier
        flowManager.SavePKCESession(state, verifier)
        
        // Generate authorization URL with PKCE
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

### Client Credentials Flow

Suitable for service-to-service communication (Machine-to-Machine).

```go
import (
    "context"
    "golang.org/x/oauth2/clientcredentials"
)

func getClientCredentialsToken(oauth2Client *oauth2.Client) (*oauth2.Token, error) {
    ctx := context.Background()
    
    // Configure client credentials flow
    config := &clientcredentials.Config{
        ClientID:     oauth2Client.GetConfig().ClientID,
        ClientSecret: oauth2Client.GetConfig().ClientSecret,
        TokenURL:     oauth2Client.GetConfig().Endpoint.TokenURL,
        Scopes:       []string{"api.read", "api.write"},
    }
    
    // Get token
    token, err := config.Token(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get client credentials token: %w", err)
    }
    
    return token, nil
}
```

---

## JWT Token Validation

Swit supports two token validation methods: local JWT validation and remote token introspection.

### Local JWT Validation (Recommended)

Local validation is fast and suitable for high-concurrency scenarios.

```go
import (
    "context"
    "github.com/innovationmech/swit/pkg/security/jwt"
)

func setupJWTValidator(oauth2Client *oauth2.Client) (*jwt.Validator, error) {
    // Configure JWT validator
    jwtConfig := &jwt.Config{
        // Signing method
        SigningMethod: "RS256",
        
        // JWKS URL (from OAuth2 client)
        JWKSURL: oauth2Client.GetJWKSURL(),
        
        // Issuer
        Issuer: oauth2Client.GetIssuerURL(),
        
        // Expected audience
        Audience: "your-client-id",
        
        // Clock skew tolerance
        ClockSkew: 30 * time.Second,
        
        // Required claims
        RequiredClaims: []string{"sub", "email"},
    }
    
    // Create validator
    validator, err := jwt.NewValidator(jwtConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create JWT validator: %w", err)
    }
    
    return validator, nil
}

func validateTokenLocally(validator *jwt.Validator, tokenString string) error {
    ctx := context.Background()
    
    // Validate token
    token, err := validator.ValidateToken(ctx, tokenString)
    if err != nil {
        return fmt.Errorf("token validation failed: %w", err)
    }
    
    // Extract claims
    claims, err := validator.ExtractClaims(token)
    if err != nil {
        return fmt.Errorf("failed to extract claims: %w", err)
    }
    
    // Access claims
    subject := claims["sub"].(string)
    email := claims["email"].(string)
    
    fmt.Printf("User: %s (%s)\n", subject, email)
    
    return nil
}
```

### Remote Token Introspection

Validate tokens through the provider's introspection endpoint.

```go
func validateTokenRemotely(oauth2Client *oauth2.Client, tokenString string) error {
    ctx := context.Background()
    
    // Call token introspection endpoint
    introspection, err := oauth2Client.IntrospectToken(ctx, tokenString)
    if err != nil {
        return fmt.Errorf("token introspection failed: %w", err)
    }
    
    // Check if token is active
    if !introspection.Active {
        return fmt.Errorf("token is not active")
    }
    
    // Access token metadata
    fmt.Printf("Token is valid. Subject: %s\n", introspection.Subject)
    fmt.Printf("Scopes: %v\n", introspection.Scope)
    
    return nil
}
```

---

## Middleware Protection

Swit provides ready-to-use Gin middleware for protecting HTTP endpoints.

### Basic Usage

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/middleware"
)

func setupRouter(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) *gin.Engine {
    router := gin.Default()
    
    // Public endpoints (no authentication required)
    router.GET("/public/info", handlePublicInfo)
    
    // Protected endpoints (authentication required)
    protected := router.Group("/protected")
    protected.Use(middleware.OAuth2Middleware(oauth2Client, jwtValidator))
    {
        protected.GET("/profile", handleProfile)
        protected.GET("/data", handleData)
    }
    
    return router
}

func handleProfile(c *gin.Context) {
    // Get user info from context
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

### Custom Middleware Configuration

```go
func setupAdvancedMiddleware(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) gin.HandlerFunc {
    config := &middleware.OAuth2MiddlewareConfig{
        OAuth2Client: oauth2Client,
        JWTValidator: jwtValidator,
        
        // Use remote token introspection instead of local validation
        UseIntrospection: true,
        
        // Skip authentication for specific paths
        SkipPaths: []string{
            "/health",
            "/metrics",
        },
        
        // Read token from cookie
        CookieName: "access_token",
        
        // Custom error handler
        ErrorHandler: func(c *gin.Context, err error) {
            c.JSON(401, gin.H{
                "error": "unauthorized",
                "message": err.Error(),
            })
            c.Abort()
        },
    }
    
    return middleware.OAuth2MiddlewareWithConfig(config)
}
```

### Optional Authentication

Allow both anonymous and authenticated users:

```go
func setupOptionalAuth(oauth2Client *oauth2.Client, jwtValidator *jwt.Validator) *gin.Engine {
    router := gin.Default()
    
    // Optional authentication endpoints
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
        // Personalized content for authenticated users
        c.JSON(200, gin.H{
            "message": fmt.Sprintf("Welcome back, %s!", userInfo.Username),
            "premium": true,
        })
    } else {
        // Basic content for anonymous users
        c.JSON(200, gin.H{
            "message": "Welcome! Please login for more features.",
            "premium": false,
        })
    }
}
```

---

## FAQ

### Q1: How to choose the right OAuth2 flow?

**A:** Based on your application type:

- **Web Application**: Authorization Code Flow
- **Single Page Application (SPA)**: Authorization Code Flow + PKCE
- **Mobile Application**: Authorization Code Flow + PKCE
- **Service-to-Service**: Client Credentials Flow
- **Trusted Application**: Password Flow (not recommended, use only when necessary)

### Q2: Local JWT validation vs Remote token introspection?

**A:** Consider these factors:

- **High Performance Requirements**: Local JWT validation
- **Need Immediate Revocation**: Remote token introspection
- **Short Token Lifetime (< 15 min)**: Local JWT validation
- **Sensitive Operations**: Remote token introspection

Recommendation: Use local validation for most requests, remote introspection for sensitive operations.

### Q3: How to securely store Client Secret in production?

**A:** Best practices:

1. **Environment Variables**:
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

### Q4: Where should access tokens be stored?

**A:** Secure storage recommendations:

- **Web Application (Server-side)**: HTTP-only Cookie
- **SPA**: In memory (don't use localStorage)
- **Mobile Application**: System secure storage (Keychain/KeyStore)
- **Server-side**: Memory or Redis (with encryption)

---

## Troubleshooting

### Issue 1: OIDC Discovery Failed

**Symptom**:
```
oauth2: discovery failed: failed to create OIDC provider: dial tcp: connection refused
```

**Solution**:

1. Verify provider is running:
```bash
curl http://localhost:8081/realms/swit/.well-known/openid-configuration
```

2. Check issuer_url configuration
3. Verify network connectivity and firewall settings
4. For Docker, check network configuration

### Issue 2: Token Validation Failed

**Symptom**:
```
oauth2: failed to verify ID token: unable to verify JWT signature
```

**Solution**:

1. Check JWKS URL accessibility:
```bash
curl http://localhost:8081/realms/swit/protocol/openid-connect/certs
```

2. Verify JWT signing method configuration
3. Check system clock synchronization
4. Increase clock skew tolerance:
```yaml
oauth2:
  jwt:
    clock_skew: 60s  # Increase to 60 seconds
```

### Issue 3: Invalid Redirect URI

**Symptom**:
```
error: invalid_request
error_description: Invalid redirect_uri
```

**Solution**:

1. Ensure callback URL matches exactly (including protocol, port, path)
2. Add redirect URI in provider admin console
3. Use wildcards (development only):
```
http://localhost:8080/*
```

---

## Related Documentation

- [OAuth2/OIDC API Reference](/api/security-oauth2-en)
- [Security Configuration Reference](/api/security-config-en)
- [Security Best Practices](/guide/security-best-practices)

## Example Code

- [OAuth2 Authentication Example](https://github.com/innovationmech/swit/tree/master/examples/oauth2-authentication)
- [Multi-Provider Integration Example](https://github.com/innovationmech/swit/tree/master/examples/multi-provider-oauth2)
- [RBAC Authorization Example](https://github.com/innovationmech/swit/tree/master/examples/oauth2-rbac)

## External Resources

- [OAuth 2.0 Specification (RFC 6749)](https://datatracker.ietf.org/doc/html/rfc6749)
- [OIDC Core Specification](https://openid.net/specs/openid-connect-core-1_0.html)
- [PKCE Specification (RFC 7636)](https://datatracker.ietf.org/doc/html/rfc7636)
- [JWT Specification (RFC 7519)](https://datatracker.ietf.org/doc/html/rfc7519)
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [Auth0 Documentation](https://auth0.com/docs)

---

**Last Updated**: 2025-11-24  
**Version**: 1.0.0  
**Maintainer**: Swit Framework Team

