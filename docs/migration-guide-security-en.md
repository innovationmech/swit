# Swit Security Features Migration Guide

This document provides a complete migration guide for upgrading from older versions of Swit to v0.9.0 and enabling security features.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Upgrade Steps](#upgrade-steps)
- [OAuth2/OIDC Integration](#oauth2oidc-integration)
- [OPA Policy Engine Integration](#opa-policy-engine-integration)
- [Secrets Management Integration](#secrets-management-integration)
- [TLS/mTLS Configuration](#tlsmtls-configuration)
- [Security Scanning Integration](#security-scanning-integration)
- [Audit Logging Configuration](#audit-logging-configuration)
- [Security Metrics Configuration](#security-metrics-configuration)
- [Known Issues](#known-issues)
- [Troubleshooting](#troubleshooting)

---

## Overview

Swit v0.9.0 introduces comprehensive enterprise-grade security features, including:

- **OAuth2/OIDC Authentication** - Support for Keycloak, Auth0, and other identity providers
- **OPA Policy Engine** - Fine-grained policy-based access control (RBAC/ABAC)
- **Secrets Management** - Support for environment variables, files, HashiCorp Vault
- **TLS/mTLS** - Transport layer security and mutual authentication
- **Security Scanning** - gosec, Trivy, govulncheck integration
- **Audit Logging** - Complete security event tracking
- **Security Metrics** - Prometheus security metrics collection

### Compatibility Notes

- ✅ **Fully Backward Compatible** - Existing code works without modifications
- ✅ **Optional Features** - All security features are optional, enable as needed
- ✅ **Gradual Adoption** - Security components can be enabled incrementally

---

## Prerequisites

### Go Version

- **Minimum Version**: Go 1.23.12+
- **Recommended Version**: Go 1.23.12 or higher

### Dependency Updates

Before upgrading, ensure dependencies are updated:

```bash
go get -u github.com/innovationmech/swit@v0.9.0
go mod tidy
```

### New Dependencies

v0.9.0 introduces the following new dependencies (automatically installed):

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/open-policy-agent/opa` | v1.4.2+ | OPA Policy Engine |
| `github.com/coreos/go-oidc/v3` | v3.11.0+ | OIDC Support |
| `golang.org/x/oauth2` | v0.26.0+ | OAuth2 Client |
| `github.com/hashicorp/vault/api` | v1.16.0+ | Vault Integration |
| `github.com/golang-jwt/jwt/v5` | v5.2.1+ | JWT Processing |

---

## Upgrade Steps

### Step 1: Update Dependencies

```bash
# Update to v0.9.0
go get -u github.com/innovationmech/swit@v0.9.0

# Clean up dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Step 2: Run Security Scan (Optional but Recommended)

```bash
# Run complete security scan
make security

# Or run individual scanning tools
govulncheck ./...
```

### Step 3: Choose and Configure Security Features

Select and enable the following security features based on your needs:

- [OAuth2/OIDC Authentication](#oauth2oidc-integration)
- [OPA Policy Engine](#opa-policy-engine-integration)
- [Secrets Management](#secrets-management-integration)
- [TLS/mTLS](#tlsmtls-configuration)

### Step 4: Test and Verify

```bash
# Run tests
make test

# Run integration tests
make test-advanced TYPE=integration
```

---

## OAuth2/OIDC Integration

### Basic Configuration

Add OAuth2 configuration to `swit.yaml`:

```yaml
oauth2:
  enabled: true
  provider: keycloak  # or auth0, custom
  client_id: my-service
  client_secret: ${OAUTH2_CLIENT_SECRET}  # Read from environment variable
  issuer_url: https://auth.example.com/realms/production
  use_discovery: true
  
  scopes:
    - openid
    - profile
    - email
  
  jwt:
    signing_method: RS256
    clock_skew: 5m
    required_claims:
      - sub
      - email
  
  cache:
    enabled: true
    max_size: 5000
    ttl: 15m
  
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-bundle.crt
```

### Code Integration

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/gin-gonic/gin"
)

func main() {
    // Load configuration
    config := &oauth2.Config{
        Enabled:      true,
        Provider:     "keycloak",
        ClientID:     "my-service",
        ClientSecret: os.Getenv("OAUTH2_CLIENT_SECRET"),
        IssuerURL:    "https://auth.example.com/realms/production",
        UseDiscovery: true,
        Scopes:       []string{"openid", "profile", "email"},
    }
    config.SetDefaults()
    
    // Create OAuth2 client
    client, err := oauth2.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Create Gin router
    router := gin.Default()
    
    // Apply authentication middleware
    router.Use(oauth2AuthMiddleware(client))
    
    // Protected routes
    router.GET("/api/v1/protected", protectedHandler)
    
    router.Run(":8080")
}

func oauth2AuthMiddleware(client *oauth2.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get token from Authorization header
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing authorization header"})
            return
        }
        
        // Validate token
        claims, err := client.ValidateToken(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        
        // Store user info in context
        c.Set("user_id", claims.Subject)
        c.Set("email", claims.Email)
        c.Next()
    }
}
```

### Environment Variable Overrides

OAuth2 configuration supports environment variable overrides:

| Environment Variable | Configuration Item |
|---------------------|-------------------|
| `OAUTH2_ENABLED` | `oauth2.enabled` |
| `OAUTH2_PROVIDER` | `oauth2.provider` |
| `OAUTH2_CLIENT_ID` | `oauth2.client_id` |
| `OAUTH2_CLIENT_SECRET` | `oauth2.client_secret` |
| `OAUTH2_ISSUER_URL` | `oauth2.issuer_url` |
| `OAUTH2_USE_DISCOVERY` | `oauth2.use_discovery` |
| `OAUTH2_JWT_SIGNING_METHOD` | `oauth2.jwt.signing_method` |

---

## OPA Policy Engine Integration

### Embedded Mode Configuration

```yaml
opa:
  mode: embedded
  embedded:
    policy_dir: ./policies
    enable_logging: true
    enable_decision_logs: true
  default_decision_path: authz/allow
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
```

### Remote Mode Configuration

```yaml
opa:
  mode: remote
  remote:
    url: http://opa-server:8181
    timeout: 30s
    max_retries: 3
    tls:
      enabled: true
      ca_file: /etc/ssl/certs/ca.pem
  default_decision_path: authz/allow
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
```

### Create RBAC Policy

Create `policies/rbac.rego`:

```rego
package authz

import rego.v1

# Default deny
default allow := false

# Super admin has all permissions
allow if {
    "admin" in input.subject.roles
}

# Role permissions mapping
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

### Code Integration

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    
    // Create OPA client
    config := &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./policies",
        },
        DefaultDecisionPath: "authz/allow",
        CacheConfig: &opa.CacheConfig{
            Enabled: true,
            MaxSize: 10000,
            TTL:     5 * time.Minute,
        },
    }
    
    client, err := opa.NewClient(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close(ctx)
    
    // Create Gin router
    router := gin.Default()
    
    // Apply authorization middleware
    router.Use(opaAuthzMiddleware(client))
    
    router.Run(":8080")
}

func opaAuthzMiddleware(client opa.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Build OPA input
        input := map[string]interface{}{
            "subject": map[string]interface{}{
                "user":  c.GetString("user_id"),
                "roles": c.GetStringSlice("roles"),
            },
            "action":   c.Request.Method,
            "resource": c.Request.URL.Path,
        }
        
        // Evaluate policy
        result, err := client.Evaluate(c.Request.Context(), input)
        if err != nil {
            c.AbortWithStatusJSON(500, gin.H{"error": "policy evaluation failed"})
            return
        }
        
        if !result.Allow {
            c.AbortWithStatusJSON(403, gin.H{"error": "access denied"})
            return
        }
        
        c.Next()
    }
}
```

### Environment Variable Overrides

| Environment Variable | Configuration Item |
|---------------------|-------------------|
| `OPA_MODE` | `opa.mode` |
| `OPA_DEFAULT_DECISION_PATH` | `opa.default_decision_path` |
| `OPA_EMBEDDED_POLICY_DIR` | `opa.embedded.policy_dir` |
| `OPA_REMOTE_URL` | `opa.remote.url` |
| `OPA_CACHE_ENABLED` | `opa.cache.enabled` |
| `OPA_CACHE_MAX_SIZE` | `opa.cache.max_size` |
| `OPA_CACHE_TTL` | `opa.cache.ttl` |

---

## Secrets Management Integration

### Multi-Provider Configuration

```yaml
secrets:
  providers:
    # Environment variable provider (highest priority)
    - type: env
      enabled: true
      env:
        prefix: APP_SECRET_
    
    # File provider
    - type: file
      enabled: true
      file:
        path: /etc/secrets
        format: yaml
    
    # HashiCorp Vault provider
    - type: vault
      enabled: true
      vault:
        address: https://vault.example.com:8200
        auth_method: approle
        role_id: ${VAULT_ROLE_ID}
        secret_id: ${VAULT_SECRET_ID}
        mount_path: secret
        path: swit/production
  
  cache:
    enabled: true
    ttl: 5m
    max_size: 1000
  
  refresh:
    enabled: true
    interval: 10m
    keys:
      - database_password
      - api_key
```

### Code Integration

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/security/secrets"
)

func main() {
    // Create secrets manager
    config := &secrets.ManagerConfig{
        Providers: []*secrets.ProviderConfig{
            {
                Type:    secrets.ProviderTypeEnv,
                Enabled: true,
                Env: &secrets.EnvProviderConfig{
                    Prefix: "APP_SECRET_",
                },
            },
            {
                Type:    secrets.ProviderTypeVault,
                Enabled: true,
                Vault: &secrets.VaultProviderConfig{
                    Address:    "https://vault.example.com:8200",
                    AuthMethod: "approle",
                    RoleID:     os.Getenv("VAULT_ROLE_ID"),
                    SecretID:   os.Getenv("VAULT_SECRET_ID"),
                },
            },
        },
        Cache: &secrets.CacheConfig{
            Enabled: true,
            TTL:     5 * time.Minute,
        },
    }
    
    manager, err := secrets.NewManager(config)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()
    
    // Get secret
    ctx := context.Background()
    dbPassword, err := manager.GetSecretValue(ctx, "database_password")
    if err != nil {
        log.Fatal(err)
    }
    
    // Use secret
    connectDatabase(dbPassword)
}
```

---

## TLS/mTLS Configuration

### Server TLS Configuration

```yaml
server:
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "TLS1.2"
      max_version: "TLS1.3"
  
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      min_version: "TLS1.2"
```

### mTLS Configuration (Mutual Authentication)

```yaml
server:
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /etc/ssl/certs/server-cert.pem
      key_file: /etc/ssl/private/server-key.pem
      # Require and verify client certificate
      client_auth: require_and_verify
      client_ca_file: /etc/ssl/certs/client-ca.pem
      min_version: "TLS1.2"
```

### Certificate Generation

```bash
# Generate CA certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
  -subj "/CN=Swit CA"

# Generate server certificate
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=myservice.example.com"
openssl x509 -req -days 365 -in server.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -set_serial 01 \
  -out server-cert.pem

# Generate client certificate (for mTLS)
openssl genrsa -out client-key.pem 4096
openssl req -new -key client-key.pem -out client.csr \
  -subj "/CN=client"
openssl x509 -req -days 365 -in client.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -set_serial 02 \
  -out client-cert.pem
```

---

## Security Scanning Integration

### Makefile Integration

The project includes integrated security scanning commands:

```bash
# Run complete security scan
make security

# Run gosec only
make security-gosec

# Run Trivy only
make security-trivy

# Run govulncheck only
make security-govulncheck
```

### CI/CD Integration

Add security scanning to GitHub Actions:

```yaml
# .github/workflows/security.yml
name: Security Scan

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: ./...
      
      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
      
      - name: Run Trivy
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
```

---

## Audit Logging Configuration

### Configuration Example

```yaml
security:
  audit:
    enabled: true
    log_level: info
    
    events:
      - authentication_success
      - authentication_failure
      - authorization_denied
      - token_issued
      - token_revoked
      - sensitive_data_accessed
    
    outputs:
      - type: file
        path: /var/log/swit/audit.log
        format: json
        max_size: 100  # MB
        max_backups: 10
        max_age: 30    # days
    
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
    
    redact_policy: mask
```

### Code Integration

```go
import "github.com/innovationmech/swit/pkg/security/audit"

// Create audit logger
auditor := audit.NewLogger(auditConfig)

// Log authentication event
auditor.Log(audit.Event{
    Type:      "authentication_success",
    Actor:     userID,
    Action:    "login",
    Resource:  "system",
    Result:    "success",
    Timestamp: time.Now(),
    Metadata: map[string]interface{}{
        "ip_address": clientIP,
        "user_agent": userAgent,
    },
})
```

---

## Security Metrics Configuration

### Configuration Example

```yaml
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
```

### Prometheus Query Examples

```promql
# Authentication failure rate
rate(swit_security_authentication_errors_total[5m])

# Authorization denial rate
rate(swit_security_authorization_denied_total[5m])

# Token validation P95 latency
histogram_quantile(0.95, 
  rate(swit_security_token_validation_duration_bucket[5m]))

# Policy evaluation P99 latency
histogram_quantile(0.99, 
  rate(swit_security_policy_evaluation_duration_bucket[5m]))
```

### Alert Rules Example

```yaml
groups:
  - name: security
    rules:
      - alert: HighAuthenticationFailureRate
        expr: rate(swit_security_authentication_errors_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate"
      
      - alert: FrequentAuthorizationDenials
        expr: rate(swit_security_authorization_denied_total[5m]) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Frequent authorization denials"
```

---

## Known Issues

### 1. JWKS Cache Refresh

**Issue**: In high-concurrency scenarios, JWKS cache refresh may cause brief token validation delays.

**Solution**: Configure appropriate cache TTL and pre-refresh time:

```yaml
oauth2:
  cache:
    enabled: true
    ttl: 1h
    refresh_before: 10m  # Refresh 10 minutes early
```

### 2. OPA Policy Hot Reload

**Issue**: In embedded mode, policy file updates don't take effect automatically.

**Solution**: Use Bundle configuration or restart the service:

```yaml
opa:
  embedded:
    bundle:
      service_url: http://bundle-server:8080
      resource: policies/bundle.tar.gz
      polling_min_delay_seconds: 60
```

### 3. Vault Connection Timeout

**Issue**: Vault connections may timeout during network instability.

**Solution**: Configure retry and timeout:

```yaml
secrets:
  providers:
    - type: vault
      vault:
        timeout: 30s
        max_retries: 3
        retry_wait: 1s
```

---

## Troubleshooting

### Authentication Failures

1. **Check Token Format**
   ```bash
   # Decode JWT token
   echo $TOKEN | cut -d'.' -f2 | base64 -d | jq
   ```

2. **Verify OIDC Discovery Endpoint**
   ```bash
   curl https://auth.example.com/.well-known/openid-configuration
   ```

3. **Check Clock Synchronization**
   ```bash
   # Ensure server clock is synchronized with auth server
   ntpdate -q pool.ntp.org
   ```

### Authorization Failures

1. **Enable OPA Decision Logs**
   ```yaml
   opa:
     embedded:
       enable_decision_logs: true
   ```

2. **Test Policy**
   ```bash
   # Test policy using OPA CLI
   opa eval -i input.json -d policies/ "data.authz.allow"
   ```

### Secret Retrieval Failures

1. **Check Provider Status**
   ```bash
   # Check Vault status
   vault status
   
   # Check environment variables
   env | grep APP_SECRET_
   ```

2. **Verify Vault Permissions**
   ```bash
   vault token lookup
   vault kv get secret/swit/production
   ```

### TLS Handshake Failures

1. **Verify Certificate Chain**
   ```bash
   openssl verify -CAfile ca-cert.pem server-cert.pem
   ```

2. **Check Certificate Validity**
   ```bash
   openssl x509 -in server-cert.pem -noout -dates
   ```

3. **Test TLS Connection**
   ```bash
   openssl s_client -connect localhost:8080 -CAfile ca-cert.pem
   ```

---

## Related Documentation

- [Security Best Practices](./security-best-practices.md)
- [Security Checklist](./security-checklist.md)
- [Configuration Reference](./configuration-reference.md)
- [OPA RBAC Guide](./opa-rbac-guide.md)
- [OPA ABAC Guide](./opa-abac-guide.md)
- [OAuth2 Integration Guide](./oauth2-integration-guide.md)

---

## License

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.


