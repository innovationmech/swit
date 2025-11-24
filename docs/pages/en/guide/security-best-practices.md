# Security Best Practices

A comprehensive security guide for the Swit microservices framework, covering the complete security lifecycle from development to deployment.

## Table of Contents

- [Overview](#overview)
- [Security Development Lifecycle (SDLC)](#security-development-lifecycle-sdlc)
- [Authentication Best Practices](#authentication-best-practices)
- [Authorization Best Practices](#authorization-best-practices)
- [API Security Guidelines](#api-security-guidelines)
- [Common Vulnerability Protection (OWASP Top 10)](#common-vulnerability-protection-owasp-top-10)
- [Data Protection](#data-protection)
- [Transport Layer Security](#transport-layer-security)
- [Secret Management](#secret-management)
- [Security Configuration](#security-configuration)
- [Security Monitoring & Auditing](#security-monitoring--auditing)
- [Container Security](#container-security)
- [Dependency Management](#dependency-management)
- [Security Testing](#security-testing)
- [Related Resources](#related-resources)

---

## Overview

The Swit framework provides multi-layered security features, including:

- **Authentication** - OAuth2/OIDC, JWT, mTLS
- **Authorization** - OPA Policy Engine (RBAC/ABAC)
- **Transport Security** - TLS/mTLS encryption
- **Data Protection** - Encryption, masking, auditing
- **Security Monitoring** - Metrics, logs, tracing
- **Automated Scanning** - gosec, Trivy, govulncheck

### Security Principles

1. **Defense in Depth** - Multiple layers of security controls
2. **Least Privilege** - Grant only necessary permissions
3. **Deny by Default** - Access requires explicit allowance
4. **Fail Securely** - Default to deny on errors
5. **Complete Audit** - Log all security events

---

## Security Development Lifecycle (SDLC)

### 1. Planning Phase

**Threat Modeling**

Identify potential security threats:

```bash
# Use threat modeling tools
- Data flow diagram analysis
- STRIDE threat classification
- Attack surface analysis
```

**Security Requirements**

Define security requirements:
- Authentication and authorization requirements
- Data protection requirements
- Compliance requirements (GDPR, PCI-DSS, etc.)
- Audit logging requirements

### 2. Design Phase

**Architecture Security Review**

```go
// ✅ Recommended: Use framework security features
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

// ❌ Avoid: Custom insecure authentication schemes
// Don't reinvent the wheel
```

**Design Principles**
- Separate authentication and authorization logic
- Use mature security libraries
- Implement proper error handling
- Avoid leaking sensitive information

### 3. Development Phase

**Secure Coding Standards**

Follow Go secure coding best practices:

```go
// ✅ Input validation
func CreateUser(req *CreateUserRequest) error {
    if err := validateEmail(req.Email); err != nil {
        return fmt.Errorf("invalid email: %w", err)
    }
    if err := validatePassword(req.Password); err != nil {
        return fmt.Errorf("weak password: %w", err)
    }
    // Use parameterized queries to prevent SQL injection
    return db.Create(&User{Email: req.Email}).Error
}

// ❌ Avoid: SQL string concatenation
// query := "SELECT * FROM users WHERE email = '" + email + "'"
```

**Code Review Checklist**

Check before each commit:
- [ ] Complete input validation
- [ ] No hardcoded secrets
- [ ] Proper error handling
- [ ] Sensitive data encrypted
- [ ] Logs contain no sensitive info
- [ ] Dependencies updated

### 4. Testing Phase

**Security Testing Types**

```bash
# Static code analysis
make quality              # Includes gosec scan

# Dependency vulnerability scanning
make security            # Trivy + govulncheck

# Unit tests (including security tests)
make test

# Integration tests
make test-advanced TYPE=integration
```

### 5. Deployment Phase

**Pre-deployment Checks**

```bash
# 1. Scan container images
trivy image myservice:latest

# 2. Validate configuration
./myservice --validate-config

# 3. Check secret management
# Ensure no hardcoded secrets

# 4. Enable TLS
# Ensure TLS is enabled in production
```

### 6. Maintenance Phase

**Continuous Security Monitoring**

- Regular dependency updates
- Monitor security advisories
- Review audit logs
- Perform regular security scans
- Conduct penetration testing

---

## Authentication Best Practices

### OAuth2/OIDC Authentication

**Recommended Configuration**

```yaml
# swit.yaml
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-service
  client_secret: ${OAUTH2_CLIENT_SECRET}  # Read from environment variable
  issuer_url: https://auth.example.com/realms/production
  use_discovery: true
  
  # Security configuration
  scopes:
    - openid
    - profile
    - email
  
  jwt:
    signing_method: RS256  # Use asymmetric encryption
    clock_skew: 5m
    required_claims:
      - sub
      - email
  
  # Enable caching for performance
  cache:
    enabled: true
    max_size: 5000
    ttl: 15m
  
  # TLS configuration
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-bundle.crt
```

**Implementation Example**

```go
package main

import (
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/innovationmech/swit/pkg/middleware"
)

func setupOAuth2() error {
    // 1. Create OAuth2 client
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
    
    // 2. Create authentication middleware
    authMiddleware := middleware.NewOAuth2Middleware(client)
    
    // 3. Apply to routes
    router.Use(authMiddleware.Authenticate())
    
    return nil
}
```

### JWT Token Validation

**Best Practices**

```go
import "github.com/innovationmech/swit/pkg/security/jwt"

// JWT validator configuration
jwtConfig := &jwt.ValidatorConfig{
    // Use JWKS to automatically fetch public keys
    JWKSConfig: &jwt.JWKSConfig{
        URL:             "https://auth.example.com/.well-known/jwks.json",
        CacheEnabled:    true,
        CacheTTL:        time.Hour,
        RefreshInterval: 15 * time.Minute,
    },
    
    // Validate claims
    Issuer:   "https://auth.example.com",
    Audience: "my-service-api",
    ClockSkew: 5 * time.Minute,
    
    // Required claims
    RequiredClaims: []string{"sub", "exp", "iat"},
}

validator, err := jwt.NewValidator(jwtConfig)
if err != nil {
    log.Fatal(err)
}

// Validate token
token := "eyJhbGciOiJSUzI1NiIs..."
claims, err := validator.Validate(token)
if err != nil {
    // Token invalid
    return err
}

// Use claims
userID := claims.Subject
email := claims["email"].(string)
```

### Password Security

**Password Policy**

```go
import "github.com/innovationmech/swit/pkg/utils"

// Password validation rules
func ValidatePassword(password string) error {
    if len(password) < 12 {
        return errors.New("password must be at least 12 characters")
    }
    
    // Check complexity
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
    hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)
    
    if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
        return errors.New("password must contain uppercase, lowercase, number and special character")
    }
    
    return nil
}

// Password hashing
func HashPassword(password string) (string, error) {
    // Use bcrypt (recommended)
    hashedPassword, err := utils.HashPassword(password)
    if err != nil {
        return "", err
    }
    return hashedPassword, nil
}

// Password verification
func VerifyPassword(hashedPassword, password string) bool {
    return utils.CheckPassword(hashedPassword, password)
}
```

### mTLS Authentication

**Client Certificate Verification**

```yaml
# swit.yaml
server:
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      # Require and verify client certificates
      client_auth: require_and_verify
      client_ca_file: /etc/ssl/certs/client-ca.pem
      min_version: "1.2"
```

---

## Authorization Best Practices

### OPA Policy Engine

**RBAC Implementation**

```rego
# policies/rbac.rego
package authz

import future.keywords.if
import future.keywords.in

# Default deny
default allow := false

# Admins have all permissions
allow if {
    "admin" in input.subject.roles
}

# Role-permission mapping
role_permissions := {
    "editor": ["read", "write"],
    "viewer": ["read"],
    "manager": ["read", "write", "delete", "approve"],
}

# Check role permissions
allow if {
    some role in input.subject.roles
    permissions := role_permissions[role]
    input.action in permissions
}

# Resource owner can access their own resources
allow if {
    input.resource.owner == input.subject.user
}
```

**ABAC Implementation**

```rego
# policies/abac.rego
package authz

import future.keywords.if

# Attribute-based access control
allow if {
    # Check department match
    input.subject.attributes.department == input.resource.department
    
    # Check security clearance
    input.subject.attributes.clearance_level >= input.resource.security_level
    
    # Check time window
    is_business_hours
}

# Business hours check
is_business_hours if {
    hour := time.clock(time.now_ns())[0]
    hour >= 9
    hour < 18
}
```

**Go Integration**

```go
import "github.com/innovationmech/swit/pkg/security/opa"

// Create OPA client
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

// Evaluate policy
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

### Authorization Middleware

```go
import (
    "github.com/innovationmech/swit/pkg/middleware"
    "github.com/gin-gonic/gin"
)

// Create authorization middleware
func setupAuthorization(opaClient *opa.Client) gin.HandlerFunc {
    return middleware.NewOPAMiddleware(opaClient, middleware.OPAMiddlewareConfig{
        InputBuilder: func(c *gin.Context) map[string]interface{} {
            // Extract user info from context
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

// Apply to routes
router.GET("/api/v1/documents/:id", 
    authMiddleware,
    authorizationMiddleware,
    getDocumentHandler,
)
```

---

## API Security Guidelines

### Input Validation

**Validate All Inputs**

```go
import (
    "github.com/go-playground/validator/v10"
)

// Use struct tags for validation
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
    
    // Validate request
    if err := validate.Struct(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Additional business validation
    if err := ValidatePassword(req.Password); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Process request...
}
```

### Output Encoding

**Prevent XSS Attacks**

```go
import "html"

// HTML encoding
func SafeOutput(userInput string) string {
    return html.EscapeString(userInput)
}

// JSON response automatically encodes
c.JSON(200, gin.H{
    "message": userInput,  // Gin handles JSON encoding automatically
})
```

### Rate Limiting

**Prevent Abuse**

```go
import "github.com/innovationmech/swit/pkg/middleware"

// IP-based rate limiting
rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
    RequestsPerSecond: 10,
    Burst:            20,
    KeyFunc: func(c *gin.Context) string {
        return c.ClientIP()
    },
})

router.Use(rateLimiter)

// User-based rate limiting
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

### CORS Configuration

**Secure Cross-Origin Configuration**

```go
import "github.com/gin-contrib/cors"

// CORS configuration
corsConfig := cors.Config{
    // ❌ Avoid: Allow all origins
    // AllowOrigins: []string{"*"},
    
    // ✅ Recommended: Explicitly specify allowed origins
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

### API Versioning

**Backward Compatibility**

```go
// Use path versioning
v1 := router.Group("/api/v1")
{
    v1.GET("/users", getUsersV1)
}

v2 := router.Group("/api/v2")
{
    v2.GET("/users", getUsersV2)
}

// Or use header versioning
router.Use(func(c *gin.Context) {
    version := c.GetHeader("API-Version")
    if version == "" {
        version = "v1"  // Default version
    }
    c.Set("api_version", version)
    c.Next()
})
```

---

## Common Vulnerability Protection (OWASP Top 10)

### 1. Injection Attacks

**SQL Injection Protection**

```go
// ✅ Use GORM parameterized queries
db.Where("email = ?", email).First(&user)

// ✅ Use named parameters
db.Where("email = @email AND status = @status", 
    sql.Named("email", email),
    sql.Named("status", "active"),
).Find(&users)

// ❌ Avoid: String concatenation
// query := "SELECT * FROM users WHERE email = '" + email + "'"
```

**Command Injection Protection**

```go
import "os/exec"

// ✅ Use parameter arrays
cmd := exec.Command("ls", "-la", userDir)

// ❌ Avoid: Shell execution
// cmd := exec.Command("sh", "-c", "ls -la " + userDir)
```

### 2. Broken Authentication

**Session Management**

```go
// ✅ Use secure session configuration
sessionConfig := sessions.Config{
    CookieHTTPOnly: true,   // Prevent XSS
    CookieSecure:   true,   // HTTPS only
    CookieSameSite: http.SameSiteStrictMode,  // Prevent CSRF
    MaxAge:         3600,   // 1 hour expiry
}

// Regenerate session ID after login
func Login(c *gin.Context) {
    // Authenticate user...
    
    // Regenerate session ID to prevent session fixation
    session := sessions.Default(c)
    session.Clear()
    session.Set("user_id", user.ID)
    session.Save()
}
```

### 3. Sensitive Data Exposure

**Encrypt Sensitive Data**

```go
import "github.com/innovationmech/swit/pkg/utils"

// Encrypt sensitive fields
type User struct {
    ID       uint
    Email    string
    SSN      string  // Social Security Number - needs encryption
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

**Log Redaction**

```go
import "github.com/innovationmech/swit/pkg/logger"

// Configure log redaction
logger.SetRedactFields([]string{
    "password",
    "token",
    "api_key",
    "secret",
    "ssn",
    "credit_card",
})

// Logs automatically redacted
log.Info("User created", 
    zap.String("email", user.Email),
    zap.String("password", "********"),  // Automatically redacted
)
```

### 4. XML External Entities (XXE)

**Secure XML Parsing**

```go
import "encoding/xml"

// ✅ Disable external entities
decoder := xml.NewDecoder(reader)
decoder.Strict = true
// Go's xml package doesn't parse external entities by default
```

### 5. Broken Access Control

**Enforce Access Control**

```go
// ✅ Check permissions for each request
func GetDocument(c *gin.Context) {
    docID := c.Param("id")
    userID := c.GetString("user_id")
    
    // Get document
    var doc Document
    if err := db.First(&doc, docID).Error; err != nil {
        c.JSON(404, gin.H{"error": "not found"})
        return
    }
    
    // Check permissions
    if doc.OwnerID != userID && !hasAdminRole(c) {
        c.JSON(403, gin.H{"error": "access denied"})
        return
    }
    
    c.JSON(200, doc)
}

// ❌ Avoid: Controlling access via URL parameters
// func GetDocument(c *gin.Context) {
//     if c.Query("admin") == "true" {
//         // Dangerous!
//     }
// }
```

### 6. Security Misconfiguration

**Secure Default Configuration**

```yaml
# ✅ Production configuration
server:
  debug: false           # Disable debug mode
  expose_errors: false   # Don't expose error details
  
  http:
    tls:
      enabled: true
      min_version: "1.2"

# ❌ Avoid: Development config in production
# server:
#   debug: true
#   expose_errors: true
```

### 7. Cross-Site Scripting (XSS)

**Content Security Policy (CSP)**

```go
// Add CSP headers
router.Use(func(c *gin.Context) {
    c.Header("Content-Security-Policy", 
        "default-src 'self'; "+
        "script-src 'self' 'unsafe-inline'; "+
        "style-src 'self' 'unsafe-inline'; "+
        "img-src 'self' data: https:;")
    c.Next()
})

// Add other security headers
router.Use(func(c *gin.Context) {
    c.Header("X-Content-Type-Options", "nosniff")
    c.Header("X-Frame-Options", "DENY")
    c.Header("X-XSS-Protection", "1; mode=block")
    c.Next()
})
```

### 8. Insecure Deserialization

**Secure Deserialization**

```go
// ✅ Use type-safe JSON binding
var req CreateUserRequest
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(400, gin.H{"error": "invalid request"})
    return
}

// ❌ Avoid: Untrusted deserialization
// var data map[string]interface{}
// if err := json.Unmarshal(untrustedData, &data); err != nil {
//     // May execute malicious code
// }
```

### 9. Using Components with Known Vulnerabilities

**Dependency Management**

```bash
# Regularly scan dependencies
make security

# Update dependencies
go get -u ./...
go mod tidy

# Check for known vulnerabilities
govulncheck ./...
```

### 10. Insufficient Logging & Monitoring

**Complete Audit Logging**

```go
import "github.com/innovationmech/swit/pkg/security/audit"

// Record security events
auditor := audit.NewLogger(audit.Config{
    Enabled: true,
    Events: []string{
        "authentication_success",
        "authentication_failure",
        "authorization_denied",
        "sensitive_data_accessed",
    },
})

// Record authentication event
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

## Data Protection

### Data at Rest Encryption

**Database Encryption**

```go
import "github.com/innovationmech/swit/pkg/utils"

// Field-level encryption
type CreditCard struct {
    ID          uint
    UserID      uint
    Number      string  // Encrypted storage
    CVV         string  // Encrypted storage
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

**File Encryption**

```go
import "crypto/aes"

// Encrypt file
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

### Data in Transit Encryption

See [Transport Layer Security](#transport-layer-security) section.

### Data Masking

**PII Data Masking**

```go
// Mask phone number
func MaskPhone(phone string) string {
    if len(phone) < 7 {
        return "***"
    }
    return phone[:3] + "****" + phone[len(phone)-4:]
}

// Mask email
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

// API response masking
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

## Transport Layer Security

### TLS Configuration

**Production TLS**

```yaml
# swit.yaml
server:
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "1.2"        # Minimum TLS 1.2
      max_version: "1.3"        # Recommended TLS 1.3
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

### mTLS (Mutual TLS)

**Service-to-Service Communication**

```yaml
# Server-side
server:
  grpc:
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      # Require client certificates
      client_auth: require_and_verify
      client_ca_file: /etc/ssl/certs/client-ca.pem
```

**Client Configuration**

```go
import (
    "crypto/tls"
    "crypto/x509"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func createMTLSClient() (*grpc.ClientConn, error) {
    // Load client certificate
    cert, err := tls.LoadX509KeyPair(
        "/etc/ssl/certs/client-cert.pem",
        "/etc/ssl/private/client-key.pem",
    )
    if err != nil {
        return nil, err
    }
    
    // Load CA certificate
    caCert, err := os.ReadFile("/etc/ssl/certs/ca-cert.pem")
    if err != nil {
        return nil, err
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    // TLS configuration
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        MinVersion:   tls.VersionTLS12,
    }
    
    creds := credentials.NewTLS(tlsConfig)
    
    // Create connection
    return grpc.Dial("service.example.com:9090",
        grpc.WithTransportCredentials(creds),
    )
}
```

### Certificate Management

**Certificate Generation**

```bash
# Generate CA certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem

# Generate server certificate
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -in server.csr -CA ca-cert.pem -CAkey ca-key.pem -set_serial 01 -out server-cert.pem

# Generate client certificate
openssl genrsa -out client-key.pem 4096
openssl req -new -key client-key.pem -out client.csr
openssl x509 -req -days 365 -in client.csr -CA ca-cert.pem -CAkey ca-key.pem -set_serial 02 -out client-cert.pem
```

**Certificate Rotation**

```bash
# Use cert-manager (Kubernetes)
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: myservice-cert
spec:
  secretName: myservice-tls
  duration: 2160h    # 90 days
  renewBefore: 360h  # Renew 15 days before expiry
  issuerRef:
    name: ca-issuer
    kind: Issuer
  dnsNames:
    - myservice.example.com
```

---

## Secret Management

### Environment Variables

**Basic Usage**

```bash
# .env file (don't commit to Git)
OAUTH2_CLIENT_SECRET=your-secret-here
JWT_HMAC_SECRET=your-jwt-secret
DATABASE_PASSWORD=your-db-password
```

```go
import "github.com/joho/godotenv"

func init() {
    // Load .env file
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

**Create Secret**

```bash
# From files
kubectl create secret generic app-secrets \
  --from-file=oauth2-client-secret=./client-secret.txt \
  --from-file=db-password=./db-password.txt

# From literal values
kubectl create secret generic app-secrets \
  --from-literal=oauth2-client-secret='your-secret' \
  --from-literal=db-password='your-password'
```

**Use Secret**

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

**Configuration**

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

**Usage**

```go
import "github.com/innovationmech/swit/pkg/security/secrets"

// Create Vault client
secretsManager, err := secrets.NewManager(secretsConfig)
if err != nil {
    log.Fatal(err)
}

// Get secret
clientSecret, err := secretsManager.GetSecret("oauth2_client_secret")
if err != nil {
    log.Fatal(err)
}

// Set secret
err = secretsManager.SetSecret("api_key", "new-api-key")
if err != nil {
    log.Fatal(err)
}
```

### Secret Rotation

**Automatic Rotation Strategy**

```go
// Rotate secrets periodically
func rotateKeys(secretsManager *secrets.Manager) {
    ticker := time.NewTicker(30 * 24 * time.Hour)  // 30 days
    defer ticker.Stop()
    
    for range ticker.C {
        // Generate new key
        newKey := generateSecureKey()
        
        // Save new key
        if err := secretsManager.SetSecret("encryption_key_new", newKey); err != nil {
            log.Error("Failed to rotate key", zap.Error(err))
            continue
        }
        
        // Notify services to use new key
        notifyKeyRotation(newKey)
        
        // Keep old key for a while to decrypt old data
        time.Sleep(7 * 24 * time.Hour)
        
        // Delete old key
        secretsManager.DeleteSecret("encryption_key_old")
    }
}
```

---

## Security Configuration

### Configuration Validation

**Validate on Startup**

```go
import "github.com/innovationmech/swit/pkg/server"

func main() {
    // Load configuration
    config, err := server.LoadConfig("swit.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        log.Fatal("Invalid configuration:", err)
    }
    
    // Security checks
    if err := validateSecurityConfig(config); err != nil {
        log.Fatal("Security configuration error:", err)
    }
}

func validateSecurityConfig(config *server.ServerConfig) error {
    // TLS must be enabled in production
    if config.Environment == "production" {
        if !config.HTTP.TLS.Enabled {
            return errors.New("TLS must be enabled in production")
        }
        if config.HTTP.TLS.MinVersion < "1.2" {
            return errors.New("TLS 1.2 or higher required in production")
        }
    }
    
    // Check secret configuration
    if config.OAuth2.Enabled && config.OAuth2.ClientSecret == "" {
        return errors.New("OAuth2 client secret is required")
    }
    
    return nil
}
```

### Environment Separation

**Configuration File Structure**

```
config/
├── swit.yaml              # Base configuration
├── swit-dev.yaml          # Development environment
├── swit-staging.yaml      # Staging environment
└── swit-production.yaml   # Production environment
```

**Load Configuration**

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

### Secure Defaults

**Configuration Defaults**

```go
func NewDefaultSecurityConfig() *SecurityConfig {
    return &SecurityConfig{
        // TLS enabled by default
        TLS: &TLSConfig{
            Enabled:    true,
            MinVersion: "1.2",
        },
        
        // CORS default restrictions
        CORS: &CORSConfig{
            AllowOrigins:     []string{},  // Explicitly configure
            AllowCredentials: false,
        },
        
        // Rate limiting enabled by default
        RateLimit: &RateLimitConfig{
            Enabled:           true,
            RequestsPerSecond: 10,
            Burst:            20,
        },
        
        // Audit enabled by default
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

## Security Monitoring & Auditing

### Audit Logging

**Configure Auditing**

```yaml
# swit.yaml
security:
  audit:
    enabled: true
    log_level: info
    
    # Event types to record
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
    
    # Output configuration
    outputs:
      - type: file
        path: /var/log/swit/audit.log
        format: json
        max_size: 100      # MB
        max_backups: 10
        max_age: 30        # days
      
      - type: syslog
        network: udp
        addr: localhost:514
        tag: swit-audit
    
    # Sensitive field filtering
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

**Record Audit Events**

```go
import "github.com/innovationmech/swit/pkg/security/audit"

// Create audit logger
auditor := audit.NewLogger(auditConfig)

// Record authentication event
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

// Record authorization event
auditor.Log(audit.Event{
    Type:      "authorization_denied",
    Actor:     userID,
    Action:    "delete",
    Resource:  fmt.Sprintf("documents/%s", docID),
    Result:    "denied",
    Reason:    "insufficient permissions",
    Timestamp: time.Now(),
})

// Record sensitive data access
auditor.Log(audit.Event{
    Type:      "sensitive_data_accessed",
    Actor:     userID,
    Action:    "read",
    Resource:  "user/ssn",
    Result:    "success",
    Timestamp: time.Now(),
})
```

### Security Metrics

**Prometheus Metrics**

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

**Query Metrics**

```promql
# Authentication failure rate
rate(swit_security_authentication_errors_total[5m])

# Authorization denial rate
rate(swit_security_authorization_denied_total[5m])

# Token validation latency
histogram_quantile(0.95, 
  rate(swit_security_token_validation_duration_bucket[5m]))

# Policy evaluation latency
histogram_quantile(0.99, 
  rate(swit_security_policy_evaluation_duration_bucket[5m]))
```

**Alert Rules**

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

### Security Monitoring

**Real-time Monitoring**

```go
import "github.com/innovationmech/swit/pkg/security/metrics"

// Create security metrics collector
metricsCollector := metrics.NewCollector(metricsConfig)

// Middleware to record metrics
func securityMetricsMiddleware(collector *metrics.Collector) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start)
        
        // Record request metrics
        collector.RecordRequest(metrics.RequestMetrics{
            Path:       c.Request.URL.Path,
            Method:     c.Request.Method,
            StatusCode: c.Writer.Status(),
            Duration:   duration,
            UserID:     c.GetString("user_id"),
        })
        
        // Record authentication metrics
        if authenticated := c.GetBool("authenticated"); authenticated {
            collector.IncAuthentications("success")
        } else {
            collector.IncAuthentications("failure")
        }
    }
}
```

---

## Container Security

### Dockerfile Security

**Best Practices**

```dockerfile
# Use specific version of base image
FROM golang:1.21-alpine3.18 AS builder

# Run as non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" -o /app/myservice ./cmd/myservice

# Minimize runtime image
FROM alpine:3.18

# Install CA certificates
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=appuser:appgroup /app/myservice .

# Copy configuration files
COPY --chown=appuser:appgroup swit.yaml .

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/myservice", "health"]

# Run application
ENTRYPOINT ["/app/myservice"]
```

### Image Scanning

**CI/CD Integration**

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

**Local Scanning**

```bash
# Scan image
trivy image myservice:latest

# Scan Dockerfile
trivy config Dockerfile

# Scan and generate report
trivy image --format json --output results.json myservice:latest
```

### Kubernetes Security

**Pod Security Policy**

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

## Dependency Management

### Dependency Scanning

**Makefile Integration**

```makefile
# Security scanning
.PHONY: security
security: security-gosec security-trivy security-govulncheck

# gosec - Go security scanning
.PHONY: security-gosec
security-gosec:
	@echo "Running gosec..."
	@gosec -fmt=json -out=gosec-report.json ./...

# Trivy - Vulnerability scanning
.PHONY: security-trivy
security-trivy:
	@echo "Running trivy..."
	@trivy fs --format json --output trivy-report.json .

# govulncheck - Go vulnerability database checking
.PHONY: security-govulncheck
security-govulncheck:
	@echo "Running govulncheck..."
	@govulncheck ./...
```

**Regular Updates**

```bash
# Check updatable dependencies
go list -u -m all

# Update all dependencies to latest minor version
go get -u ./...

# Update specific dependency
go get github.com/gin-gonic/gin@latest

# Clean unused dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Dependency Locking

**Using go.sum**

```bash
# go.sum file locks dependency versions
# Ensure go.sum is committed to version control

# Verify go.sum
go mod verify

# If go.sum doesn't exist, regenerate
go mod tidy
```

### Private Dependencies

**Configure Private Repositories**

```bash
# Configure GOPRIVATE
export GOPRIVATE=github.com/myorg/*

# Configure Git credentials
git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
```

---

## Security Testing

### Unit Testing

**Security-Related Tests**

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
        // Parameterized queries should be safe
        var count int64
        err := db.Model(&User{}).Where("email = ?", input).Count(&count).Error
        assert.NoError(t, err)
        assert.Equal(t, int64(0), count)
    }
}
```

### Integration Testing

**Authentication Tests**

```go
func TestOAuth2Authentication(t *testing.T) {
    // Setup test server
    router := setupTestRouter()
    
    // Test unauthenticated access
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/protected", nil)
    router.ServeHTTP(w, req)
    assert.Equal(t, 401, w.Code)
    
    // Test invalid token
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("GET", "/api/v1/protected", nil)
    req.Header.Set("Authorization", "Bearer invalid-token")
    router.ServeHTTP(w, req)
    assert.Equal(t, 401, w.Code)
    
    // Test valid token
    validToken := generateTestToken()
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("GET", "/api/v1/protected", nil)
    req.Header.Set("Authorization", "Bearer "+validToken)
    router.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}
```

**Authorization Tests**

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

### Penetration Testing

**OWASP ZAP Integration**

```yaml
# zap-scan.yml
name: OWASP ZAP Scan

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM

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

## Related Resources

### Official Documentation

- [Configuration Reference](../api/security-config-en.md)
- [OAuth2/OIDC API](../api/security-oauth2-en.md)
- [OPA Policy API](../api/security-opa-en.md)
- [Security Checklist](../../security-checklist.md)
- [Incident Response](../../security-incident-response.md)

### External Resources

**OWASP Resources**
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Go Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Go_SCP_Cheat_Sheet.html)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)

**Go Security**
- [Go Security Policy](https://golang.org/security)
- [Go Vulnerability Database](https://pkg.go.dev/vuln/)
- [Effective Go - Security](https://golang.org/doc/effective_go)

**Standards & Compliance**
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [PCI DSS](https://www.pcisecuritystandards.org/)
- [GDPR](https://gdpr.eu/)

**Tool Documentation**
- [gosec Documentation](https://github.com/securego/gosec)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [OPA Documentation](https://www.openpolicyagent.org/docs/)
- [OAuth 2.0 RFC](https://oauth.net/2/)
- [OpenID Connect](https://openid.net/connect/)

---

## License

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.

