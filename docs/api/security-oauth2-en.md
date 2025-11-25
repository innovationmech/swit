# OAuth2/OIDC API Reference

This document provides a complete API reference for OAuth2/OIDC security features in the Swit framework, including configuration, clients, flows, and best practices.

## Table of Contents

- [Overview](#overview)
- [Configuration API](#configuration-api)
- [Client API](#client-api)
- [OAuth2 Flows](#oauth2-flows)
- [Token Management](#token-management)
- [Provider Integration](#provider-integration)
- [Cache Management](#cache-management)
- [Error Handling](#error-handling)
- [Code Examples](#code-examples)
- [Best Practices](#best-practices)

## Overview

The OAuth2/OIDC package (`pkg/security/oauth2`) provides enterprise-grade identity authentication and authorization features for the Swit framework. Key features include:

- ✅ **Multi-Provider Support** - Keycloak, Auth0, Google, Microsoft, Okta, and more
- ✅ **OIDC Auto-Discovery** - Automatically discover endpoints from issuer URL
- ✅ **Multiple Authorization Flows** - Authorization code, client credentials, password, refresh token
- ✅ **JWT Token Validation** - Complete JWT signature and claims validation
- ✅ **Token Caching** - Built-in token caching for improved performance
- ✅ **TLS Support** - Full TLS/mTLS configuration
- ✅ **Health Checks** - Built-in provider connection health checks

### Supported Providers

| Provider | Type Constant | OIDC Discovery | Default Scopes |
|----------|--------------|----------------|----------------|
| Keycloak | `ProviderKeycloak` | ✅ | openid, profile, email |
| Auth0 | `ProviderAuth0` | ✅ | openid, profile, email |
| Google | `ProviderGoogle` | ✅ | openid, profile, email |
| Microsoft | `ProviderMicrosoft` | ✅ | openid, profile, email |
| Okta | `ProviderOkta` | ✅ | openid, profile, email |
| Custom | `ProviderCustom` | Optional | Custom |

---

## Configuration API

### Config Structure

`Config` is the main configuration structure for the OAuth2/OIDC client.

```go
type Config struct {
    // Enabled indicates whether OAuth2 authentication is enabled
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // Provider is the OAuth2 provider name
    // Supported: "keycloak", "auth0", "google", "microsoft", "okta", "custom"
    Provider string `json:"provider" yaml:"provider" mapstructure:"provider"`
    
    // ClientID is the OAuth2 client identifier
    ClientID string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`
    
    // ClientSecret is the OAuth2 client secret
    ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`
    
    // RedirectURL is the OAuth2 redirect URL for authorization code flow
    RedirectURL string `json:"redirect_url" yaml:"redirect_url" mapstructure:"redirect_url"`
    
    // Scopes is the list of OAuth2 scopes to request
    Scopes []string `json:"scopes" yaml:"scopes" mapstructure:"scopes"`
    
    // IssuerURL is the OIDC issuer URL (used for discovery)
    IssuerURL string `json:"issuer_url" yaml:"issuer_url" mapstructure:"issuer_url"`
    
    // AuthURL is the OAuth2 authorization endpoint URL (optional, discovered via OIDC)
    AuthURL string `json:"auth_url,omitempty" yaml:"auth_url,omitempty" mapstructure:"auth_url"`
    
    // TokenURL is the OAuth2 token endpoint URL (optional, discovered via OIDC)
    TokenURL string `json:"token_url,omitempty" yaml:"token_url,omitempty" mapstructure:"token_url"`
    
    // UserInfoURL is the OIDC user info endpoint URL (optional)
    UserInfoURL string `json:"user_info_url,omitempty" yaml:"user_info_url,omitempty" mapstructure:"user_info_url"`
    
    // JWKSURL is the URL of the JSON Web Key Set for token verification
    JWKSURL string `json:"jwks_url,omitempty" yaml:"jwks_url,omitempty" mapstructure:"jwks_url"`
    
    // UseDiscovery indicates whether to use OIDC discovery
    UseDiscovery bool `json:"use_discovery" yaml:"use_discovery" mapstructure:"use_discovery"`
    
    // HTTPTimeout is the HTTP client timeout for OAuth2 requests
    HTTPTimeout time.Duration `json:"http_timeout" yaml:"http_timeout" mapstructure:"http_timeout"`
    
    // SkipIssuerVerification skips the issuer verification (testing only)
    SkipIssuerVerification bool `json:"skip_issuer_verification,omitempty" yaml:"skip_issuer_verification,omitempty" mapstructure:"skip_issuer_verification"`
    
    // SkipExpiryCheck skips the token expiry check (testing only)
    SkipExpiryCheck bool `json:"skip_expiry_check,omitempty" yaml:"skip_expiry_check,omitempty" mapstructure:"skip_expiry_check"`
    
    // JWTConfig is the JWT token validation configuration
    JWTConfig JWTConfig `json:"jwt" yaml:"jwt" mapstructure:"jwt"`
    
    // CacheConfig is the token cache configuration
    CacheConfig TokenCacheConfig `json:"cache" yaml:"cache" mapstructure:"cache"`
    
    // TLSConfig is the TLS configuration for OAuth2 connections
    TLSConfig TLSConfig `json:"tls" yaml:"tls" mapstructure:"tls"`
}
```

#### Configuration Methods

##### SetDefaults()

Sets default values for the configuration.

```go
func (c *Config) SetDefaults()
```

**Example:**

```go
config := &oauth2.Config{
    Provider:  "keycloak",
    ClientID:  "my-client",
    IssuerURL: "https://auth.example.com/realms/myrealm",
}
config.SetDefaults()
// Now HTTPTimeout = 30s, UseDiscovery = true (for Keycloak)
```

##### Validate()

Validates the configuration.

```go
func (c *Config) Validate() error
```

**Returns:**
- `error` - Descriptive error if configuration is invalid

**Validation Rules:**
- `ClientID` cannot be empty
- `IssuerURL` cannot be empty (if `UseDiscovery` is true)
- If not using discovery, `AuthURL` and `TokenURL` must be set

**Example:**

```go
config := &oauth2.Config{
    Provider:  "keycloak",
    ClientID:  "my-client",
    IssuerURL: "https://auth.example.com/realms/myrealm",
}
if err := config.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

##### ApplyProviderDefaults()

Applies provider-specific defaults.

```go
func (c *Config) ApplyProviderDefaults() error
```

**Example:**

```go
config := &oauth2.Config{
    Provider: "keycloak",
    ClientID: "my-client",
}
config.ApplyProviderDefaults()
// Automatically sets: UseDiscovery=true, Scopes=["openid","profile","email"]
```

##### LoadFromEnv()

Loads configuration from environment variables.

```go
func (c *Config) LoadFromEnv()
```

**Supported Environment Variables:**
- `OAUTH2_ENABLED` - Enable OAuth2
- `OAUTH2_PROVIDER` - Provider type
- `OAUTH2_CLIENT_ID` - Client ID
- `OAUTH2_CLIENT_SECRET` - Client secret
- `OAUTH2_ISSUER_URL` - Issuer URL
- `OAUTH2_REDIRECT_URL` - Redirect URL
- `OAUTH2_SCOPES` - Scopes (comma-separated)
- `OAUTH2_USE_DISCOVERY` - Enable OIDC discovery

**Example:**

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

### JWTConfig Structure

JWT token validation configuration.

```go
type JWTConfig struct {
    // SigningMethod is the expected JWT signing method
    // Supported: "RS256", "RS384", "RS512", "HS256", "HS384", "HS512", "ES256", "ES384", "ES512"
    SigningMethod string `json:"signing_method" yaml:"signing_method" mapstructure:"signing_method"`
    
    // ClockSkew is the duration of time skew to tolerate when verifying time-based claims
    ClockSkew time.Duration `json:"clock_skew" yaml:"clock_skew" mapstructure:"clock_skew"`
    
    // SkipClaimsValidation skips standard claims validation (iss, aud, exp, nbf, iat)
    // Testing purposes only
    SkipClaimsValidation bool `json:"skip_claims_validation,omitempty" yaml:"skip_claims_validation,omitempty" mapstructure:"skip_claims_validation"`
    
    // RequiredClaims is a list of claims that must be present in the token
    RequiredClaims []string `json:"required_claims,omitempty" yaml:"required_claims,omitempty" mapstructure:"required_claims"`
    
    // Audience is the expected audience claim value
    Audience string `json:"audience,omitempty" yaml:"audience,omitempty" mapstructure:"audience"`
}
```

**Defaults:**
- `SigningMethod`: "RS256"
- `ClockSkew`: 5 minutes

**Example:**

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

### TokenCacheConfig Structure

Token cache configuration.

```go
type TokenCacheConfig struct {
    // Enabled indicates whether token caching is enabled
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // MaxSize is the maximum number of tokens in the cache
    MaxSize int `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
    
    // TTL is the time-to-live for cache entries
    TTL time.Duration `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
    
    // CleanupInterval is the cache cleanup interval
    CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" mapstructure:"cleanup_interval"`
}
```

**Defaults:**
- `Enabled`: true
- `MaxSize`: 1000
- `TTL`: 10 minutes
- `CleanupInterval`: 5 minutes

**Example:**

```yaml
cache:
  enabled: true
  max_size: 5000
  ttl: 15m
  cleanup_interval: 10m
```

---

### TLSConfig Structure

TLS configuration for OAuth2 connections.

```go
type TLSConfig struct {
    // Enabled indicates whether TLS is enabled
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // CertFile is the client certificate file path
    CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty" mapstructure:"cert_file"`
    
    // KeyFile is the client private key file path
    KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty" mapstructure:"key_file"`
    
    // CAFile is the CA certificate file path
    CAFile string `json:"ca_file,omitempty" yaml:"ca_file,omitempty" mapstructure:"ca_file"`
    
    // InsecureSkipVerify skips server certificate verification (not recommended for production)
    InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`
}
```

**Example:**

```yaml
tls:
  enabled: true
  cert_file: /path/to/client-cert.pem
  key_file: /path/to/client-key.pem
  ca_file: /path/to/ca-cert.pem
```

---

## Client API

### Client Structure

OAuth2/OIDC client.

```go
type Client struct {
    // Internal fields (not exported)
}
```

### Creating a Client

#### NewClient()

Creates a new OAuth2/OIDC client.

```go
func NewClient(ctx context.Context, cfg *Config) (*Client, error)
```

**Parameters:**
- `ctx` - Context
- `cfg` - OAuth2 configuration

**Returns:**
- `*Client` - OAuth2 client instance
- `error` - If initialization fails

**Example:**

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
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close(ctx)
```

### Client Methods

#### GetAuthorizationURL()

Gets the authorization URL (for authorization code flow).

```go
func (c *Client) GetAuthorizationURL(state string, opts ...oauth2.AuthCodeOption) string
```

**Parameters:**
- `state` - State parameter (for CSRF protection)
- `opts` - Optional authorization options

**Returns:**
- `string` - Authorization URL

**Example:**

```go
state := "random-state-value"
authURL := client.GetAuthorizationURL(state)
// Redirect user to authURL
http.Redirect(w, r, authURL, http.StatusFound)
```

#### ExchangeAuthorizationCode()

Exchanges authorization code for access token.

```go
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*TokenResponse, error)
```

**Parameters:**
- `ctx` - Context
- `code` - Authorization code
- `opts` - Optional exchange options

**Returns:**
- `*TokenResponse` - Token response
- `error` - If exchange fails

**Example:**

```go
code := r.URL.Query().Get("code")
tokenResp, err := client.ExchangeAuthorizationCode(ctx, code)
if err != nil {
    log.Printf("Failed to exchange code: %v", err)
    return
}

log.Printf("Access token: %s", tokenResp.AccessToken)
log.Printf("Refresh token: %s", tokenResp.RefreshToken)
log.Printf("ID token: %s", tokenResp.IDToken)
```

#### ClientCredentials()

Gets access token using client credentials flow.

```go
func (c *Client) ClientCredentials(ctx context.Context, scopes ...string) (*TokenResponse, error)
```

**Parameters:**
- `ctx` - Context
- `scopes` - Requested scopes (optional)

**Returns:**
- `*TokenResponse` - Token response
- `error` - If request fails

**Example:**

```go
tokenResp, err := client.ClientCredentials(ctx, "read:data", "write:data")
if err != nil {
    log.Fatalf("Failed to get client credentials: %v", err)
}

log.Printf("Access token: %s", tokenResp.AccessToken)
```

#### PasswordCredentials()

Gets access token using password credentials flow.

```go
func (c *Client) PasswordCredentials(ctx context.Context, username, password string, scopes ...string) (*TokenResponse, error)
```

**Parameters:**
- `ctx` - Context
- `username` - Username
- `password` - Password
- `scopes` - Requested scopes (optional)

**Returns:**
- `*TokenResponse` - Token response
- `error` - If authentication fails

**⚠️ Warning:** Password flow is insecure, use only in trusted first-party applications.

**Example:**

```go
tokenResp, err := client.PasswordCredentials(ctx, "alice", "password123")
if err != nil {
    log.Printf("Password authentication failed: %v", err)
    return
}

log.Printf("Access token: %s", tokenResp.AccessToken)
```

#### RefreshToken()

Gets new access token using refresh token.

```go
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
```

**Parameters:**
- `ctx` - Context
- `refreshToken` - Refresh token

**Returns:**
- `*TokenResponse` - New token response
- `error` - If refresh fails

**Example:**

```go
newTokenResp, err := client.RefreshToken(ctx, oldTokenResp.RefreshToken)
if err != nil {
    log.Printf("Failed to refresh token: %v", err)
    return
}

log.Printf("New access token: %s", newTokenResp.AccessToken)
```

#### ValidateToken()

Validates access token validity.

```go
func (c *Client) ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
```

**Parameters:**
- `ctx` - Context
- `token` - Access token

**Returns:**
- `*TokenClaims` - Token claims
- `error` - If validation fails

**Example:**

```go
claims, err := client.ValidateToken(ctx, accessToken)
if err != nil {
    log.Printf("Token validation failed: %v", err)
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}

log.Printf("User ID: %s", claims.Subject)
log.Printf("Email: %s", claims.Email)
```

For more client methods and complete examples, please refer to the Chinese version above or the [source code documentation](../../pkg/security/oauth2/).

---

## Best Practices

### 1. Use OIDC Discovery

Prefer OIDC discovery to automatically configure endpoints:

```go
config := &oauth2.Config{
    Provider:     "keycloak",
    ClientID:     "my-app",
    ClientSecret: "secret",
    IssuerURL:    "https://auth.example.com/realms/myrealm",
    UseDiscovery: true,  // ✅ Recommended
}
```

### 2. Enable Token Caching

For high-concurrency scenarios, enable token caching:

```go
config.CacheConfig = oauth2.TokenCacheConfig{
    Enabled:         true,
    MaxSize:         5000,
    TTL:             15 * time.Minute,
    CleanupInterval: 10 * time.Minute,
}
```

### 3. Secure Sensitive Information

- Use environment variables or secret management services for `ClientSecret`
- Don't hardcode sensitive information in code
- Use `pkg/security/secrets` package for secret management

```go
// ✅ Recommended
config.ClientSecret = os.Getenv("OAUTH2_CLIENT_SECRET")

// ❌ Avoid
config.ClientSecret = "hardcoded-secret"
```

### 4. Validate State Parameter

Always validate the state parameter in authorization code flow to prevent CSRF attacks:

```go
func handleCallback(w http.ResponseWriter, r *http.Request) {
    state := r.URL.Query().Get("state")
    
    // ✅ Validate state
    if !verifyState(state) {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }
    
    // Continue processing...
}
```

### 5. Implement Token Refresh

Automatically refresh expired access tokens:

```go
func getValidToken(ctx context.Context, currentToken *TokenResponse) (string, error) {
    // Check if token is about to expire (5 minutes early)
    if time.Now().Add(5 * time.Minute).After(currentToken.ExpiresAt) {
        // Refresh token
        newToken, err := client.RefreshToken(ctx, currentToken.RefreshToken)
        if err != nil {
            return "", err
        }
        return newToken.AccessToken, nil
    }
    return currentToken.AccessToken, nil
}
```

---

## Related Documentation

- [OPA Policy API Reference](./security-opa-en.md)
- [Security Configuration Reference](./security-config-en.md)
- [JWT Token Validation](../../pkg/security/jwt/README.md)
- [OAuth2 Integration Guide](../guide/oauth2-integration.md)

## License

Copyright (c) 2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.



