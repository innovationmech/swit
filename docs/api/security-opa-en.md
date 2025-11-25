# OPA Policy API Reference

This document provides a complete API reference for OPA (Open Policy Agent) policy engine in the Swit framework, including configuration, clients, evaluators, policy management, and best practices.

## Table of Contents

- [Overview](#overview)
- [Configuration API](#configuration-api)
- [Client API](#client-api)
- [Evaluator API](#evaluator-api)
- [Policy Input Building](#policy-input-building)
- [RBAC & ABAC](#rbac--abac)
- [Policy Management](#policy-management)
- [Cache Management](#cache-management)
- [Monitoring & Metrics](#monitoring--metrics)
- [Code Examples](#code-examples)
- [Best Practices](#best-practices)

## Overview

The OPA package (`pkg/security/opa`) provides policy-based access control capabilities for the Swit framework. Key features include:

- ✅ **Multi-Mode Support** - Embedded, Remote, and Sidecar modes
- ✅ **RBAC & ABAC** - Built-in RBAC and ABAC policy templates
- ✅ **Dynamic Policy Management** - Runtime policy loading, updating, and removal
- ✅ **Decision Caching** - Built-in high-performance decision caching
- ✅ **Auto-Discovery** - OIDC-style path normalization support
- ✅ **Performance Monitoring** - Built-in evaluation metrics and performance monitoring
- ✅ **Easy Integration** - Seamless integration with Gin/gRPC

### Operation Modes

| Mode | Constant | Description | Use Case |
|------|----------|-------------|----------|
| Embedded | `ModeEmbedded` | OPA engine runs in same process | Development, small deployments |
| Remote | `ModeRemote` | Connect to external OPA server | Production, centralized management |
| Sidecar | `ModeSidecar` | Connect to local OPA sidecar container | Kubernetes deployments |

---

## Configuration API

### Config Structure

`Config` is the main configuration structure for the OPA client.

```go
type Config struct {
    // Mode is the OPA operation mode
    // Supported: "embedded", "remote", "sidecar"
    Mode string `json:"mode" yaml:"mode" mapstructure:"mode"`
    
    // EmbeddedConfig is the embedded mode configuration
    EmbeddedConfig *EmbeddedConfig `json:"embedded" yaml:"embedded" mapstructure:"embedded"`
    
    // RemoteConfig is the remote mode configuration
    RemoteConfig *RemoteConfig `json:"remote" yaml:"remote" mapstructure:"remote"`
    
    // CacheConfig is the decision cache configuration
    CacheConfig *CacheConfig `json:"cache" yaml:"cache" mapstructure:"cache"`
    
    // DefaultDecisionPath is the default decision path
    // Example: "authz/allow", "rbac.allow"
    DefaultDecisionPath string `json:"default_decision_path" yaml:"default_decision_path" mapstructure:"default_decision_path"`
    
    // LogLevel is the log level
    // Supported: "debug", "info", "warn", "error"
    LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
}
```

#### Configuration Methods

##### SetDefaults()

Sets default values for the configuration.

```go
func (c *Config) SetDefaults()
```

**Defaults:**
- `Mode`: "embedded"
- `LogLevel`: "info"
- `DefaultDecisionPath`: "allow"

##### Validate()

Validates the configuration.

```go
func (c *Config) Validate() error
```

**Returns:**
- `error` - Descriptive error if configuration is invalid

**Validation Rules:**
- `Mode` must be "embedded", "remote", or "sidecar"
- Embedded mode requires `EmbeddedConfig`
- Remote mode requires `RemoteConfig` with valid URL

---

### EmbeddedConfig Structure

Embedded mode configuration.

```go
type EmbeddedConfig struct {
    // PolicyDir is the policy file directory path
    PolicyDir string `json:"policy_dir" yaml:"policy_dir" mapstructure:"policy_dir"`
    
    // DataDir is the data file directory path (optional)
    DataDir string `json:"data_dir,omitempty" yaml:"data_dir,omitempty" mapstructure:"data_dir"`
    
    // EnableLogging enables OPA internal logging
    EnableLogging bool `json:"enable_logging" yaml:"enable_logging" mapstructure:"enable_logging"`
    
    // EnableDecisionLogs enables decision logging
    EnableDecisionLogs bool `json:"enable_decision_logs" yaml:"enable_decision_logs" mapstructure:"enable_decision_logs"`
}
```

**Example:**

```yaml
mode: embedded
embedded:
  policy_dir: ./pkg/security/opa/policies
  enable_logging: true
  enable_decision_logs: false
```

---

### RemoteConfig Structure

Remote mode configuration.

```go
type RemoteConfig struct {
    // URL is the remote OPA server URL
    // Example: "http://opa-server:8181"
    URL string `json:"url" yaml:"url" mapstructure:"url"`
    
    // Timeout is the request timeout
    Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
    
    // MaxRetries is the maximum number of retries
    MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`
    
    // AuthToken is the authentication token (optional)
    AuthToken string `json:"auth_token,omitempty" yaml:"auth_token,omitempty" mapstructure:"auth_token"`
    
    // TLS is the TLS configuration (optional)
    TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}
```

**Defaults:**
- `Timeout`: 30 seconds
- `MaxRetries`: 3

**Example:**

```yaml
mode: remote
remote:
  url: http://opa-server:8181
  timeout: 30s
  max_retries: 3
```

---

### CacheConfig Structure

Decision cache configuration.

```go
type CacheConfig struct {
    // Enabled enables decision caching
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // MaxSize is the maximum number of cache entries
    MaxSize int `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
    
    // TTL is the cache expiration time
    TTL time.Duration `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
    
    // EnableMetrics enables cache metrics
    EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics" mapstructure:"enable_metrics"`
}
```

**Defaults:**
- `Enabled`: true
- `MaxSize`: 10000
- `TTL`: 5 minutes
- `EnableMetrics`: true

---

## Client API

### Client Interface

OPA client interface.

```go
type Client interface {
    // Evaluate evaluates a policy decision
    Evaluate(ctx context.Context, path string, input interface{}) (*Result, error)
    
    // Query executes a Rego query
    Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error)
    
    // PartialEvaluate executes partial evaluation
    PartialEvaluate(ctx context.Context, query string, input interface{}) (*PartialResult, error)
    
    // LoadPolicy loads a policy (embedded mode only)
    LoadPolicy(ctx context.Context, name string, policy string) error
    
    // LoadPolicyFromFile loads a policy from file (embedded mode only)
    LoadPolicyFromFile(ctx context.Context, path string) error
    
    // LoadData loads data to OPA store (embedded mode only)
    LoadData(ctx context.Context, path string, data interface{}) error
    
    // RemovePolicy removes a policy (embedded mode only)
    RemovePolicy(ctx context.Context, name string) error
    
    // Close closes the client
    Close(ctx context.Context) error
    
    // IsEmbedded returns whether this is an embedded mode client
    IsEmbedded() bool
}
```

### Creating a Client

#### NewClient()

Creates a new OPA client.

```go
func NewClient(ctx context.Context, config *Config) (Client, error)
```

**Parameters:**
- `ctx` - Context
- `config` - OPA configuration

**Returns:**
- `Client` - OPA client instance
- `error` - If initialization fails

**Example:**

```go
ctx := context.Background()
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
    DefaultDecisionPath: "authz/allow",
}

client, err := opa.NewClient(ctx, config)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close(ctx)
```

### Client Methods

#### Evaluate()

Evaluates a policy decision.

```go
func (c Client) Evaluate(ctx context.Context, path string, input interface{}) (*Result, error)
```

**Parameters:**
- `ctx` - Context
- `path` - Decision path (e.g., "authz/allow", "rbac.allow")
- `input` - Policy input data

**Returns:**
- `*Result` - Evaluation result
- `error` - If evaluation fails

**Path Formats:**
- Slash format: `"authz/allow"`, `"rbac/rules/allow"`
- Dot-separated format: `"authz.allow"`, `"rbac.rules.allow"`
- Both formats are interchangeable and automatically normalized

**Example:**

```go
// Basic evaluation
result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
    "user":     "alice",
    "action":   "read",
    "resource": "documents",
})
if err != nil {
    log.Printf("Evaluation failed: %v", err)
    return
}

if result.Allowed {
    log.Println("Access allowed")
} else {
    log.Println("Access denied")
}
```

---

### Result Structure

Policy evaluation result.

```go
type Result struct {
    // Decision is the decision result (usually bool or map)
    Decision interface{} `json:"decision"`
    
    // DecisionID is the decision ID (for auditing and tracing)
    DecisionID string `json:"decision_id,omitempty"`
    
    // Allowed is a convenient boolean result (when decision is bool)
    Allowed bool `json:"allowed"`
    
    // Metrics are the evaluation metrics
    Metrics *Metrics `json:"metrics,omitempty"`
    
    // Bindings are variable bindings (contains other variables defined in policy)
    Bindings map[string]interface{} `json:"bindings,omitempty"`
}
```

**Example:**

```go
result, err := client.Evaluate(ctx, "rbac/allow", input)
if err != nil {
    return err
}

// Use Allowed field
if result.Allowed {
    log.Println("Access allowed")
}

// Access raw decision
log.Printf("Decision: %v", result.Decision)

// Access bound variables (like audit logs)
if auditLog, ok := result.Bindings["audit_log"].(map[string]interface{}); ok {
    log.Printf("Audit log: %+v", auditLog)
}

// Access performance metrics
if result.Metrics != nil {
    log.Printf("Evaluation time: %d ns", result.Metrics.TimerEvalNs)
}
```

---

## Evaluator API

### Evaluator Structure

Policy evaluator provides higher-level API.

```go
type Evaluator struct {
    // Internal fields (not exported)
}
```

### Creating an Evaluator

#### NewEvaluator()

Creates an evaluator from an existing client.

```go
func NewEvaluator(client Client) *Evaluator
```

#### NewEvaluatorWithConfig()

Creates an evaluator with configuration.

```go
func NewEvaluatorWithConfig(ctx context.Context, config *Config) (*Evaluator, error)
```

**Example:**

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
}

evaluator, err := opa.NewEvaluatorWithConfig(ctx, config)
if err != nil {
    log.Fatalf("Failed to create evaluator: %v", err)
}
defer evaluator.Close(ctx)
```

### Evaluator Methods

#### IsAllowed()

Evaluates whether access is allowed (convenience method).

```go
func (e *Evaluator) IsAllowed(ctx context.Context, path string, input interface{}) (bool, error)
```

**Example:**

```go
allowed, err := evaluator.IsAllowed(ctx, "authz/allow", input)
if err != nil {
    log.Printf("Evaluation failed: %v", err)
    return
}

if allowed {
    log.Println("Access allowed")
}
```

#### EvaluateWithDefault()

Evaluates policy, returns default value on failure.

```go
func (e *Evaluator) EvaluateWithDefault(ctx context.Context, path string, input interface{}, defaultValue bool) bool
```

**Example:****

```go
// Default deny on evaluation failure
allowed := evaluator.EvaluateWithDefault(ctx, "authz/allow", input, false)

if allowed {
    log.Println("Access allowed")
} else {
    log.Println("Access denied or evaluation failed")
}
```

#### EvaluateRBAC()

Evaluates RBAC policy.

```go
func (e *Evaluator) EvaluateRBAC(ctx context.Context, input *RBACInput) (*Result, error)
```

**Example:**

```go
input := &opa.RBACInput{
    Subject: &opa.Subject{
        User:  "alice",
        Roles: []string{"editor"},
    },
    Action:   "update",
    Resource: "documents",
}

result, err := evaluator.EvaluateRBAC(ctx, input)
if err != nil {
    log.Printf("RBAC evaluation failed: %v", err)
    return
}

if result.Allowed {
    log.Println("RBAC: Access allowed")
}
```

#### EvaluateABAC()

Evaluates ABAC policy.

```go
func (e *Evaluator) EvaluateABAC(ctx context.Context, input *ABACInput) (*Result, error)
```

**Example:**

```go
input := &opa.ABACInput{
    Subject: &opa.Subject{
        User:  "alice",
        Roles: []string{"employee"},
        Attributes: map[string]interface{}{
            "department":      "engineering",
            "clearance_level": 3,
        },
    },
    Action: "read",
    Resource: &opa.Resource{
        Type: "document",
        ID:   "doc-123",
        Attributes: map[string]interface{}{
            "department":     "engineering",
            "security_level": 2,
        },
    },
}

result, err := evaluator.EvaluateABAC(ctx, input)
if err != nil {
    log.Printf("ABAC evaluation failed: %v", err)
    return
}

if result.Allowed {
    log.Println("ABAC: Access allowed")
}
```

---

## Policy Input Building

### PolicyInput Structure

Standardized data structure for policy input.

```go
type PolicyInput struct {
    // Request information
    Request RequestInfo `json:"request"`
    
    // User information
    User UserInfo `json:"user"`
    
    // Resource information
    Resource ResourceInfo `json:"resource"`
    
    // Custom data
    Custom map[string]interface{} `json:"custom,omitempty"`
}
```

### PolicyInputBuilder

Policy input builder provides a fluent API.

#### NewPolicyInputBuilder()

Creates a policy input builder.

```go
func NewPolicyInputBuilder() *PolicyInputBuilder
```

#### FromHTTPRequest()

Builds input from Gin HTTP request.

```go
func (b *PolicyInputBuilder) FromHTTPRequest(c *gin.Context) *PolicyInputBuilder
```

**Example:**

```go
func authzMiddleware(evaluator *opa.Evaluator) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Build input from HTTP request
        input := opa.NewPolicyInputBuilder().
            FromHTTPRequest(c).
            WithUserID("alice").
            WithUserRoles([]string{"editor"}).
            WithResourceType("documents").
            Build()
        
        result, err := evaluator.Evaluate(c.Request.Context(), "authz/allow", input)
        if err != nil {
            c.AbortWithStatus(http.StatusInternalServerError)
            return
        }
        
        if !result.Allowed {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        
        c.Next()
    }
}
```

#### Builder Methods

User-related methods:

```go
func (b *PolicyInputBuilder) WithUserID(id string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUsername(username string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserEmail(email string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserRoles(roles []string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserPermissions(permissions []string) *PolicyInputBuilder
```

Resource-related methods:

```go
func (b *PolicyInputBuilder) WithResourceType(resourceType string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceID(id string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceName(name string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceOwner(owner string) *PolicyInputBuilder
```

---

## Best Practices

### 1. Choose the Right Mode

**Embedded Mode:**
- ✅ Good for: Development, testing, small deployments
- ✅ Pros: Simple, fast, no external dependencies
- ❌ Cons: Policy updates require restart

**Remote Mode:**
- ✅ Good for: Production, centralized management
- ✅ Pros: Centralized policy management, dynamic updates
- ❌ Cons: Network latency, requires additional OPA server

### 2. Enable Decision Caching

For high-concurrency scenarios, enable caching to significantly improve performance:

```go
config.CacheConfig = &opa.CacheConfig{
    Enabled:       true,
    MaxSize:       10000,
    TTL:           5 * time.Minute,
    EnableMetrics: true,
}
```

### 3. Monitor Performance

Enable metrics and monitor evaluation performance:

```go
result, err := evaluator.Evaluate(ctx, "authz/allow", input)
if err != nil {
    return err
}

if result.Metrics != nil && result.Metrics.TimerEvalNs > 1000000 { // > 1ms
    log.Printf("WARNING: Slow policy evaluation: %d ns", result.Metrics.TimerEvalNs)
}
```

### 4. Implement Error Handling

Implement proper error handling and fallback strategies:

```go
allowed := evaluator.EvaluateWithDefault(ctx, "authz/allow", input, false) // Default deny

if !allowed {
    c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
        "error": "Access denied",
    })
    return
}
```

### 5. Use Builder Pattern

Use `PolicyInputBuilder` to build structured input:

```go
input := opa.NewPolicyInputBuilder().
    FromHTTPRequest(c).
    WithUserRoles(roles).
    WithResourceType("document").
    Build()
```

---

## Related Documentation

- [OAuth2/OIDC API Reference](./security-oauth2-en.md)
- [Security Configuration Reference](./security-config-en.md)
- [OPA RBAC Policy Guide](../opa-rbac-guide.md)
- [OPA ABAC Policy Guide](../opa-abac-guide.md)
- [OPA Package README](../../pkg/security/opa/README.md)

## License

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.




