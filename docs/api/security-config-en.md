# Security Configuration Reference

This document provides a complete configuration reference for Swit framework security features, including OAuth2/OIDC, OPA, TLS, secret management, and auditing.

## Table of Contents

- [Overview](#overview)
- [OAuth2/OIDC Configuration](#oauth2oidc-configuration)
- [OPA Policy Configuration](#opa-policy-configuration)
- [TLS/mTLS Configuration](#tlsmtls-configuration)
- [JWT Configuration](#jwt-configuration)
- [Secret Management Configuration](#secret-management-configuration)
- [Audit Log Configuration](#audit-log-configuration)
- [Security Metrics Configuration](#security-metrics-configuration)
- [Complete Configuration Examples](#complete-configuration-examples)
- [Environment Variables](#environment-variables)
- [Best Practices](#best-practices)

## Overview

The Swit framework provides multi-layered security configuration options, supporting:

- **Authentication** - OAuth2/OIDC, JWT, API Keys
- **Authorization** - OPA Policy Engine (RBAC/ABAC)
- **Transport Security** - TLS/mTLS encryption
- **Secret Management** - Environment variables, files, Vault integration
- **Auditing** - Security event logging and tracing

### Configuration Methods

1. **YAML Configuration Files** - Recommended for production
2. **Environment Variables** - For containerized deployments and sensitive information
3. **Code Configuration** - For dynamic configuration and testing

---

## OAuth2/OIDC Configuration

### Basic Configuration

```yaml
# swit.yaml or switauth.yaml
oauth2:
  enabled: true
  provider: keycloak
  client_id: my-app
  client_secret: ${OAUTH2_CLIENT_SECRET}  # Read from environment variable
  issuer_url: https://auth.example.com/realms/myrealm
  redirect_url: http://localhost:8080/callback
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
  http_timeout: 30s
```

### Complete Configuration Options

```yaml
oauth2:
  # Basic configuration
  enabled: true                    # Enable OAuth2 authentication
  provider: keycloak               # Provider: keycloak, auth0, google, microsoft, okta, custom
  client_id: my-app                # Client ID
  client_secret: secret            # Client secret (use environment variable recommended)
  redirect_url: http://localhost:8080/callback  # Redirect URL
  
  # OIDC discovery
  issuer_url: https://auth.example.com/realms/myrealm  # Issuer URL
  use_discovery: true              # Enable OIDC auto-discovery
  
  # Manual endpoint configuration (when use_discovery=false)
  auth_url: https://auth.example.com/oauth2/auth       # Authorization endpoint
  token_url: https://auth.example.com/oauth2/token     # Token endpoint
  user_info_url: https://auth.example.com/oauth2/userinfo  # UserInfo endpoint
  jwks_url: https://auth.example.com/oauth2/jwks       # JWKS endpoint
  
  # Scopes
  scopes:
    - openid
    - profile
    - email
    - offline_access           # For refresh tokens
  
  # Timeout and retry
  http_timeout: 30s              # HTTP request timeout
  
  # Testing options (don't use in production)
  skip_issuer_verification: false  # Skip issuer verification
  skip_expiry_check: false         # Skip expiry check
  
  # JWT configuration
  jwt:
    signing_method: RS256          # Signing method: RS256, RS384, RS512, HS256, ES256
    clock_skew: 5m                 # Clock skew tolerance
    skip_claims_validation: false   # Skip claims validation (testing only)
    audience: my-api               # Expected audience claim
    required_claims:               # Required claims
      - sub
      - email
  
  # Token caching
  cache:
    enabled: true                  # Enable token caching
    max_size: 1000                 # Maximum cache entries
    ttl: 10m                       # Cache expiration time
    cleanup_interval: 5m           # Cleanup interval
  
  # TLS configuration
  tls:
    enabled: true                  # Enable TLS
    cert_file: /path/to/client-cert.pem  # Client certificate
    key_file: /path/to/client-key.pem    # Client private key
    ca_file: /path/to/ca-cert.pem        # CA certificate
    insecure_skip_verify: false    # Skip certificate verification (not recommended)
```

### Provider-Specific Configuration

#### Keycloak

```yaml
oauth2:
  provider: keycloak
  issuer_url: https://auth.example.com/realms/myrealm
  use_discovery: true
  # Keycloak automatically sets openid scope
```

#### Auth0

```yaml
oauth2:
  provider: auth0
  issuer_url: https://your-domain.auth0.com/
  use_discovery: true
  # Auth0 requires audience specification
  jwt:
    audience: https://your-api.example.com
```

#### Google

```yaml
oauth2:
  provider: google
  issuer_url: https://accounts.google.com
  use_discovery: true
  scopes:
    - openid
    - profile
    - email
```

---

## OPA Policy Configuration

### Embedded Mode

```yaml
opa:
  mode: embedded                   # Operation mode: embedded, remote, sidecar
  default_decision_path: authz/allow  # Default decision path
  log_level: info                  # Log level: debug, info, warn, error
  
  # Embedded mode configuration
  embedded:
    policy_dir: ./pkg/security/opa/policies  # Policy directory
    data_dir: ./data                         # Data directory (optional)
    enable_logging: true                     # Enable OPA internal logging
    enable_decision_logs: false              # Enable decision logging
  
  # Decision caching
  cache:
    enabled: true                  # Enable caching
    max_size: 10000                # Maximum cache entries
    ttl: 5m                        # Cache expiration time
    enable_metrics: true           # Enable cache metrics
```

### Remote Mode

```yaml
opa:
  mode: remote                     # Remote mode
  default_decision_path: authz/allow
  log_level: info
  
  # Remote OPA server configuration
  remote:
    url: http://opa-server:8181    # OPA server URL
    timeout: 30s                   # Request timeout
    max_retries: 3                 # Maximum retries
    auth_token: ${OPA_AUTH_TOKEN}  # Authentication token (optional)
    
    # TLS configuration (optional)
    tls:
      enabled: true
      cert_file: /path/to/client-cert.pem
      key_file: /path/to/client-key.pem
      ca_file: /path/to/ca-cert.pem
  
  # Decision caching
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
    enable_metrics: true
```

### Sidecar Mode (Kubernetes)

```yaml
opa:
  mode: sidecar                    # Sidecar mode
  default_decision_path: authz/allow
  
  remote:
    url: http://localhost:8181     # Local sidecar
    timeout: 10s                   # Shorter timeout
    max_retries: 2
  
  cache:
    enabled: true
    max_size: 5000
    ttl: 3m
```

---

## TLS/mTLS Configuration

### Server TLS Configuration

```yaml
server:
  name: my-service
  
  # HTTP TLS configuration
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /path/to/server-cert.pem
      key_file: /path/to/server-key.pem
      # Client certificate verification (mTLS)
      client_auth: require_and_verify
      client_ca_file: /path/to/client-ca.pem
      # TLS version
      min_version: "1.2"           # 1.0, 1.1, 1.2, 1.3
      max_version: "1.3"
  
  # gRPC TLS configuration
  grpc:
    addr: :9090
    tls:
      enabled: true
      cert_file: /path/to/server-cert.pem
      key_file: /path/to/server-key.pem
      client_auth: require_and_verify
      client_ca_file: /path/to/client-ca.pem
```

### TLS Options

**client_auth options:**
- `no_client_cert` - Don't request client certificate
- `request_client_cert` - Request but don't verify
- `require_any_client_cert` - Require but don't verify
- `verify_client_cert_if_given` - Verify if provided
- `require_and_verify` - Require and verify (recommended for mTLS)

**TLS versions:**
- `1.0` - TLS 1.0 (not recommended)
- `1.1` - TLS 1.1 (not recommended)
- `1.2` - TLS 1.2 (recommended minimum)
- `1.3` - TLS 1.3 (recommended)

---

## JWT Configuration

```yaml
jwt:
  # Signature verification
  signing_method: RS256            # RS256, RS384, RS512, HS256, ES256
  
  # JWKS configuration (for RS256 and other public key algorithms)
  jwks:
    url: https://auth.example.com/.well-known/jwks.json
    cache_enabled: true
    cache_ttl: 1h
    refresh_interval: 15m
  
  # HMAC secret (for HS256 and other symmetric algorithms)
  hmac_secret: ${JWT_HMAC_SECRET}  # Read from environment variable
  
  # Claims validation
  issuer: https://auth.example.com    # Expected issuer
  audience: my-api                    # Expected audience
  clock_skew: 5m                      # Clock skew tolerance
  
  # Required claims
  required_claims:
    - sub
    - exp
    - iat
  
  # Blacklist configuration
  blacklist:
    enabled: true
    redis:
      addr: localhost:6379
      password: ${REDIS_PASSWORD}
      db: 0
```

---

## Secret Management Configuration

### Environment Variable Provider

```yaml
secrets:
  provider: env                      # Environment variable provider
  env:
    prefix: SWIT_SECRET_             # Secret prefix
```

Usage:
```bash
export SWIT_SECRET_DATABASE_PASSWORD=mypassword
export SWIT_SECRET_API_KEY=myapikey
```

### File Provider

```yaml
secrets:
  provider: file                     # File provider
  file:
    path: /etc/secrets               # Secrets directory
    # Secret files: /etc/secrets/database_password, /etc/secrets/api_key
```

### Vault Provider

```yaml
secrets:
  provider: vault                    # HashiCorp Vault provider
  vault:
    addr: https://vault.example.com:8200
    token: ${VAULT_TOKEN}            # Vault token
    # Or use AppRole authentication
    auth_method: approle
    role_id: ${VAULT_ROLE_ID}
    secret_id: ${VAULT_SECRET_ID}
    
    # KV store configuration
    kv_version: 2                    # KV v1 or v2
    mount_path: secret               # Mount path
    path: swit/production            # Secret path
    
    # TLS configuration
    tls:
      enabled: true
      ca_cert: /path/to/vault-ca.pem
```

---

## Audit Log Configuration

```yaml
security:
  audit:
    enabled: true                    # Enable auditing
    log_level: info                  # Log level
    
    # Audit event types
    events:
      - authentication_success
      - authentication_failure
      - authorization_denied
      - token_issued
      - token_revoked
      - policy_evaluated
    
    # Output configuration
    outputs:
      # File output
      - type: file
        path: /var/log/swit/audit.log
        format: json                 # json or text
        max_size: 100                # MB
        max_backups: 10
        max_age: 30                  # days
      
      # Webhook output
      - type: webhook
        url: https://audit-collector.example.com/events
        headers:
          Authorization: Bearer ${AUDIT_WEBHOOK_TOKEN}
    
    # Sensitive field filtering
    sensitive_fields:
      - password
      - client_secret
      - api_key
      - token
    
    # Redact policy
    redact_policy: mask              # mask, remove, hash
```

---

## Security Metrics Configuration

```yaml
security:
  metrics:
    enabled: true                    # Enable security metrics
    namespace: swit                  # Metrics namespace
    subsystem: security              # Metrics subsystem
    
    # Metric types
    collect:
      - authentication_total         # Total authentications
      - authentication_errors        # Authentication errors
      - authorization_total          # Total authorizations
      - authorization_denied         # Authorization denials
      - token_validation_duration    # Token validation duration
      - policy_evaluation_duration   # Policy evaluation duration
    
    # Labels
    labels:
      service: my-service
      environment: production
```

---

## Complete Configuration Examples

### Production Environment

```yaml
# swit.yaml - Production configuration
server:
  name: my-service
  http:
    addr: :8080
    tls:
      enabled: true
      cert_file: /etc/swit/certs/server-cert.pem
      key_file: /etc/swit/certs/server-key.pem
      client_auth: require_and_verify
      client_ca_file: /etc/swit/certs/client-ca.pem
      min_version: "1.2"

oauth2:
  enabled: true
  provider: keycloak
  client_id: my-service
  client_secret: ${OAUTH2_CLIENT_SECRET}
  issuer_url: https://auth.example.com/realms/production
  use_discovery: true
  cache:
    enabled: true
    max_size: 5000
    ttl: 15m

opa:
  mode: remote
  default_decision_path: authz/allow
  remote:
    url: https://opa.example.com
    timeout: 30s
    max_retries: 3
    auth_token: ${OPA_AUTH_TOKEN}
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m

security:
  audit:
    enabled: true
    events:
      - authentication_failure
      - authorization_denied
    outputs:
      - type: file
        path: /var/log/swit/audit.log
        format: json
```

### Development Environment

```yaml
# swit-dev.yaml - Development configuration
server:
  name: my-service-dev
  http:
    addr: :8080
    tls:
      enabled: false               # Disable TLS in dev

oauth2:
  enabled: true
  provider: keycloak
  client_id: my-service-dev
  client_secret: dev-secret        # Plain text OK in dev
  issuer_url: http://localhost:8180/realms/dev
  use_discovery: true
  skip_issuer_verification: true   # OK in dev

opa:
  mode: embedded                   # Use embedded mode in dev
  embedded:
    policy_dir: ./pkg/security/opa/policies
  cache:
    enabled: false                 # Disable cache for debugging
```

---

## Environment Variables

### OAuth2/OIDC

```bash
OAUTH2_ENABLED=true
OAUTH2_PROVIDER=keycloak
OAUTH2_CLIENT_ID=my-app
OAUTH2_CLIENT_SECRET=secret
OAUTH2_ISSUER_URL=https://auth.example.com/realms/myrealm
OAUTH2_REDIRECT_URL=http://localhost:8080/callback
OAUTH2_USE_DISCOVERY=true
```

### OPA

```bash
OPA_MODE=remote
OPA_REMOTE_URL=http://opa-server:8181
OPA_CACHE_ENABLED=true
OPA_CACHE_MAX_SIZE=10000
OPA_AUTH_TOKEN=secret-token
```

### JWT

```bash
JWT_JWKS_URL=https://auth.example.com/.well-known/jwks.json
JWT_ISSUER=https://auth.example.com
JWT_AUDIENCE=my-api
JWT_HMAC_SECRET=your-secret-key
```

---

## Best Practices

### 1. Use Environment Variables for Sensitive Information

❌ **Avoid:**
```yaml
oauth2:
  client_secret: hardcoded-secret    # Don't hardcode
```

✅ **Recommended:**
```yaml
oauth2:
  client_secret: ${OAUTH2_CLIENT_SECRET}
```

### 2. Enable TLS

Always enable TLS in production:

```yaml
server:
  http:
    tls:
      enabled: true
      min_version: "1.2"              # Minimum TLS 1.2
```

### 3. Use mTLS for Service-to-Service Communication

```yaml
server:
  grpc:
    tls:
      enabled: true
      client_auth: require_and_verify  # Require client certificates
```

### 4. Configure Audit Logging

Log all security-related events:

```yaml
security:
  audit:
    enabled: true
    events:
      - authentication_failure
      - authorization_denied
```

### 5. Enable Decision Caching

Improve performance:

```yaml
opa:
  cache:
    enabled: true
    max_size: 10000
    ttl: 5m
```

### 6. Monitor Security Metrics

```yaml
security:
  metrics:
    enabled: true
    collect:
      - authentication_errors
      - authorization_denied
```

### 7. Use Least Privilege Principle

Only request necessary scopes:

```yaml
oauth2:
  scopes:
    - openid
    - profile    # Only request what you need
```

### 8. Separate Development and Production Configurations

Use different configuration files:
- `swit.yaml` - Production
- `swit-dev.yaml` - Development
- `swit-test.yaml` - Testing

---

## Related Documentation

- [OAuth2/OIDC API Reference](./security-oauth2-en.md)
- [OPA Policy API Reference](./security-opa-en.md)
- [Server Configuration Reference](../configuration-reference.md)
- [Security Best Practices Guide](../../SECURITY.md)

## License

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.





