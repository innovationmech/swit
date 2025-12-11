# OPA 策略 API 参考

本文档提供 Swit 框架中 OPA (Open Policy Agent) 策略引擎的完整 API 参考，包括配置、客户端、评估器、策略管理和最佳实践。

## 目录

- [概述](#概述)
- [配置 API](#配置-api)
- [客户端 API](#客户端-api)
- [评估器 API](#评估器-api)
- [策略输入构建](#策略输入构建)
- [RBAC 与 ABAC](#rbac-与-abac)
- [策略管理](#策略管理)
- [缓存管理](#缓存管理)
- [监控与指标](#监控与指标)
- [代码示例](#代码示例)
- [最佳实践](#最佳实践)

## 概述

OPA 包 (`pkg/security/opa`) 为 Swit 框架提供了基于策略的访问控制能力。主要特性包括：

- ✅ **多模式支持** - 嵌入式、远程和 Sidecar 模式
- ✅ **RBAC & ABAC** - 内置 RBAC 和 ABAC 策略模板
- ✅ **动态策略管理** - 运行时加载、更新和移除策略
- ✅ **决策缓存** - 内置高性能决策缓存
- ✅ **自动发现** - 支持 OIDC 风格的路径自动规范化
- ✅ **性能监控** - 内置评估指标和性能监控
- ✅ **易于集成** - 与 Gin/gRPC 无缝集成

### 运行模式

| 模式 | 常量 | 描述 | 适用场景 |
|------|------|------|---------|
| 嵌入式 | `ModeEmbedded` | OPA 引擎运行在同一进程 | 开发、小型部署 |
| 远程 | `ModeRemote` | 连接到外部 OPA 服务器 | 生产、集中管理 |
| Sidecar | `ModeSidecar` | 连接到本地 OPA sidecar 容器 | Kubernetes 部署 |

---

## 配置 API

### Config 结构体

`Config` 是 OPA 客户端的主要配置结构。

```go
type Config struct {
    // Mode 是 OPA 运行模式
    // 支持: "embedded", "remote", "sidecar"
    Mode string `json:"mode" yaml:"mode" mapstructure:"mode"`
    
    // EmbeddedConfig 是嵌入式模式配置
    EmbeddedConfig *EmbeddedConfig `json:"embedded" yaml:"embedded" mapstructure:"embedded"`
    
    // RemoteConfig 是远程模式配置
    RemoteConfig *RemoteConfig `json:"remote" yaml:"remote" mapstructure:"remote"`
    
    // CacheConfig 是决策缓存配置
    CacheConfig *CacheConfig `json:"cache" yaml:"cache" mapstructure:"cache"`
    
    // DefaultDecisionPath 是默认的决策路径
    // 例如: "authz/allow", "rbac.allow"
    DefaultDecisionPath string `json:"default_decision_path" yaml:"default_decision_path" mapstructure:"default_decision_path"`
    
    // LogLevel 是日志级别
    // 支持: "debug", "info", "warn", "error"
    LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
}
```

#### 配置方法

##### SetDefaults()

设置配置的默认值。

```go
func (c *Config) SetDefaults()
```

**默认值：**
- `Mode`: "embedded"
- `LogLevel`: "info"
- `DefaultDecisionPath`: "allow"

**示例：**

```go
config := &opa.Config{
    Mode: "embedded",
}
config.SetDefaults()
// 现在 LogLevel = "info", DefaultDecisionPath = "allow"
```

##### Validate()

验证配置的有效性。

```go
func (c *Config) Validate() error
```

**返回值：**
- `error` - 如果配置无效，返回描述性错误

**验证规则：**
- `Mode` 必须是 "embedded"、"remote" 或 "sidecar"
- 嵌入式模式需要 `EmbeddedConfig`
- 远程模式需要 `RemoteConfig` 和有效的 URL

**示例：**

```go
config := &opa.Config{
    Mode: "embedded",
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
}
if err := config.Validate(); err != nil {
    log.Fatalf("无效的配置: %v", err)
}
```

##### LoadFromEnv()

从环境变量加载配置。

```go
func (c *Config) LoadFromEnv()
```

**支持的环境变量：**
- `OPA_MODE` - 运行模式
- `OPA_REMOTE_URL` - 远程 OPA URL
- `OPA_CACHE_ENABLED` - 启用缓存
- `OPA_CACHE_MAX_SIZE` - 缓存最大条目数
- `OPA_CACHE_TTL` - 缓存过期时间
- `OPA_LOG_LEVEL` - 日志级别

**示例：**

```bash
export OPA_MODE=embedded
export OPA_CACHE_ENABLED=true
export OPA_CACHE_MAX_SIZE=10000
```

```go
config := &opa.Config{}
config.LoadFromEnv()
```

---

### EmbeddedConfig 结构体

嵌入式模式配置。

```go
type EmbeddedConfig struct {
    // PolicyDir 是策略文件目录路径
    PolicyDir string `json:"policy_dir" yaml:"policy_dir" mapstructure:"policy_dir"`
    
    // DataDir 是数据文件目录路径（可选）
    DataDir string `json:"data_dir,omitempty" yaml:"data_dir,omitempty" mapstructure:"data_dir"`
    
    // EnableLogging 启用 OPA 内部日志
    EnableLogging bool `json:"enable_logging" yaml:"enable_logging" mapstructure:"enable_logging"`
    
    // EnableDecisionLogs 启用决策日志
    EnableDecisionLogs bool `json:"enable_decision_logs" yaml:"enable_decision_logs" mapstructure:"enable_decision_logs"`
}
```

**示例：**

```yaml
mode: embedded
embedded:
  policy_dir: ./pkg/security/opa/policies
  enable_logging: true
  enable_decision_logs: false
```

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir:          "./policies",
        EnableLogging:      true,
        EnableDecisionLogs: false,
    },
}
```

---

### RemoteConfig 结构体

远程模式配置。

```go
type RemoteConfig struct {
    // URL 是远程 OPA 服务器 URL
    // 例如: "http://opa-server:8181"
    URL string `json:"url" yaml:"url" mapstructure:"url"`
    
    // Timeout 是请求超时时间
    Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
    
    // MaxRetries 是最大重试次数
    MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`
    
    // AuthToken 是认证令牌（可选）
    AuthToken string `json:"auth_token,omitempty" yaml:"auth_token,omitempty" mapstructure:"auth_token"`
    
    // TLS 是 TLS 配置（可选）
    TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}
```

**默认值：**
- `Timeout`: 30秒
- `MaxRetries`: 3

**示例：**

```yaml
mode: remote
remote:
  url: http://opa-server:8181
  timeout: 30s
  max_retries: 3
  auth_token: ${OPA_AUTH_TOKEN}
  tls:
    enabled: true
    ca_file: /path/to/ca.pem
```

```go
config := &opa.Config{
    Mode: opa.ModeRemote,
    RemoteConfig: &opa.RemoteConfig{
        URL:        "http://opa-server:8181",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
    },
}
```

---

### CacheConfig 结构体

决策缓存配置。

```go
type CacheConfig struct {
    // Enabled 启用决策缓存
    Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
    
    // MaxSize 是缓存最大条目数
    MaxSize int `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
    
    // TTL 是缓存过期时间
    TTL time.Duration `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
    
    // EnableMetrics 启用缓存指标
    EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics" mapstructure:"enable_metrics"`
}
```

**默认值：**
- `Enabled`: true
- `MaxSize`: 10000
- `TTL`: 5分钟
- `EnableMetrics`: true

**示例：**

```yaml
cache:
  enabled: true
  max_size: 10000
  ttl: 5m
  enable_metrics: true
```

```go
config := &opa.Config{
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       10000,
        TTL:           5 * time.Minute,
        EnableMetrics: true,
    },
}
```

---

## 客户端 API

### Client 接口

OPA 客户端接口。

```go
type Client interface {
    // Evaluate 评估策略决策
    Evaluate(ctx context.Context, path string, input interface{}) (*Result, error)
    
    // Query 执行 Rego 查询
    Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error)
    
    // PartialEvaluate 执行部分评估
    PartialEvaluate(ctx context.Context, query string, input interface{}) (*PartialResult, error)
    
    // LoadPolicy 加载策略（仅嵌入式模式支持）
    LoadPolicy(ctx context.Context, name string, policy string) error
    
    // LoadPolicyFromFile 从文件加载策略（仅嵌入式模式支持）
    LoadPolicyFromFile(ctx context.Context, path string) error
    
    // LoadData 加载数据到 OPA store（仅嵌入式模式支持）
    LoadData(ctx context.Context, path string, data interface{}) error
    
    // RemovePolicy 移除策略（仅嵌入式模式支持）
    RemovePolicy(ctx context.Context, name string) error
    
    // Close 关闭客户端
    Close(ctx context.Context) error
    
    // IsEmbedded 是否为嵌入式模式
    IsEmbedded() bool
}
```

### 创建客户端

#### NewClient()

创建新的 OPA 客户端。

```go
func NewClient(ctx context.Context, config *Config) (Client, error)
```

**参数：**
- `ctx` - 上下文
- `config` - OPA 配置

**返回值：**
- `Client` - OPA 客户端实例
- `error` - 如果初始化失败

**示例：**

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
    log.Fatalf("创建客户端失败: %v", err)
}
defer client.Close(ctx)
```

### 客户端方法

#### Evaluate()

评估策略决策。

```go
func (c Client) Evaluate(ctx context.Context, path string, input interface{}) (*Result, error)
```

**参数：**
- `ctx` - 上下文
- `path` - 决策路径（例如："authz/allow", "rbac.allow"）
- `input` - 策略输入数据

**返回值：**
- `*Result` - 评估结果
- `error` - 如果评估失败

**路径格式：**
- 斜杠格式：`"authz/allow"`, `"rbac/rules/allow"`
- 点分隔格式：`"authz.allow"`, `"rbac.rules.allow"`
- 两种格式可互换使用，自动规范化

**示例：**

```go
// 基本评估
result, err := client.Evaluate(ctx, "authz/allow", map[string]interface{}{
    "user":   "alice",
    "action": "read",
    "resource": "documents",
})
if err != nil {
    log.Printf("评估失败: %v", err)
    return
}

if result.Allowed {
    log.Println("访问允许")
} else {
    log.Println("访问拒绝")
}

// 使用默认路径（如果配置中设置了 DefaultDecisionPath）
result, err = client.Evaluate(ctx, "", input)
```

#### Query()

执行自定义 Rego 查询。

```go
func (c Client) Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error)
```

**参数：**
- `ctx` - 上下文
- `query` - Rego 查询字符串
- `input` - 查询输入数据

**返回值：**
- `rego.ResultSet` - 查询结果集
- `error` - 如果查询失败

**示例：**

```go
query := `data.rbac.user_roles`
input := map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"editor"},
    },
}

resultSet, err := client.Query(ctx, query, input)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

for _, result := range resultSet {
    log.Printf("查询结果: %+v", result.Expressions)
}
```

#### PartialEvaluate()

执行部分评估（用于优化和预计算）。

```go
func (c Client) PartialEvaluate(ctx context.Context, query string, input interface{}) (*PartialResult, error)
```

**参数：**
- `ctx` - 上下文
- `query` - Rego 查询字符串
- `input` - 部分输入数据

**返回值：**
- `*PartialResult` - 部分评估结果
- `error` - 如果评估失败

**示例：**

```go
query := `data.authz.allow`
input := map[string]interface{}{
    "user": "alice",
    // 部分输入，action 和 resource 未知
}

partialResult, err := client.PartialEvaluate(ctx, query, input)
if err != nil {
    log.Printf("部分评估失败: %v", err)
    return
}

log.Printf("部分查询: %+v", partialResult.Queries)
```

#### LoadPolicy()

加载策略（仅嵌入式模式）。

```go
func (c Client) LoadPolicy(ctx context.Context, name string, policy string) error
```

**参数：**
- `ctx` - 上下文
- `name` - 策略名称（例如："authz.rego"）
- `policy` - 策略内容（Rego 代码）

**返回值：**
- `error` - 如果加载失败

**示例：**

```go
policy := `
package authz

import rego.v1

default allow = false

allow if {
    input.user == "admin"
}
`

err := client.LoadPolicy(ctx, "admin.rego", policy)
if err != nil {
    log.Printf("加载策略失败: %v", err)
}
```

#### LoadPolicyFromFile()

从文件加载策略（仅嵌入式模式）。

```go
func (c Client) LoadPolicyFromFile(ctx context.Context, path string) error
```

**参数：**
- `ctx` - 上下文
- `path` - 策略文件路径

**返回值：**
- `error` - 如果加载失败

**示例：**

```go
err := client.LoadPolicyFromFile(ctx, "./policies/rbac.rego")
if err != nil {
    log.Printf("加载策略文件失败: %v", err)
}
```

#### LoadData()

加载数据到 OPA store（仅嵌入式模式）。

```go
func (c Client) LoadData(ctx context.Context, path string, data interface{}) error
```

**参数：**
- `ctx` - 上下文
- `path` - 数据路径（例如："users", "roles"）
- `data` - 数据内容

**返回值：**
- `error` - 如果加载失败

**示例：**

```go
users := map[string]interface{}{
    "alice": map[string]interface{}{
        "roles": []string{"admin", "editor"},
    },
    "bob": map[string]interface{}{
        "roles": []string{"viewer"},
    },
}

err := client.LoadData(ctx, "users", users)
if err != nil {
    log.Printf("加载数据失败: %v", err)
}
```

#### RemovePolicy()

移除策略（仅嵌入式模式）。

```go
func (c Client) RemovePolicy(ctx context.Context, name string) error
```

**参数：**
- `ctx` - 上下文
- `name` - 策略名称

**返回值：**
- `error` - 如果移除失败

**示例：**

```go
err := client.RemovePolicy(ctx, "old-policy.rego")
if err != nil {
    log.Printf("移除策略失败: %v", err)
}
```

#### Close()

关闭客户端并清理资源。

```go
func (c Client) Close(ctx context.Context) error
```

**示例：**

```go
defer client.Close(ctx)
```

---

### Result 结构体

策略评估结果。

```go
type Result struct {
    // Decision 决策结果（通常是 bool 或 map）
    Decision interface{} `json:"decision"`
    
    // DecisionID 决策 ID（用于审计和追踪）
    DecisionID string `json:"decision_id,omitempty"`
    
    // Allowed 便捷的布尔结果（当决策为 bool 时）
    Allowed bool `json:"allowed"`
    
    // Metrics 评估指标
    Metrics *Metrics `json:"metrics,omitempty"`
    
    // Bindings 变量绑定（包含策略中定义的其他变量）
    Bindings map[string]interface{} `json:"bindings,omitempty"`
}
```

**示例：**

```go
result, err := client.Evaluate(ctx, "rbac/allow", input)
if err != nil {
    return err
}

// 使用 Allowed 字段
if result.Allowed {
    log.Println("访问允许")
}

// 访问原始决策
log.Printf("决策: %v", result.Decision)

// 访问绑定的变量（如审计日志）
if auditLog, ok := result.Bindings["audit_log"].(map[string]interface{}); ok {
    log.Printf("审计日志: %+v", auditLog)
}

// 访问性能指标
if result.Metrics != nil {
    log.Printf("评估时间: %d ns", result.Metrics.TimerEvalNs)
}
```

---

## 评估器 API

### Evaluator 结构体

策略评估器提供了更高级的 API。

```go
type Evaluator struct {
    // 内部字段（不导出）
}
```

### 创建评估器

#### NewEvaluator()

从现有客户端创建评估器。

```go
func NewEvaluator(client Client) *Evaluator
```

**参数：**
- `client` - OPA 客户端

**返回值：**
- `*Evaluator` - 评估器实例

**示例：**

```go
client, err := opa.NewClient(ctx, config)
if err != nil {
    log.Fatal(err)
}

evaluator := opa.NewEvaluator(client)
```

#### NewEvaluatorWithConfig()

使用配置创建评估器。

```go
func NewEvaluatorWithConfig(ctx context.Context, config *Config) (*Evaluator, error)
```

**参数：**
- `ctx` - 上下文
- `config` - OPA 配置

**返回值：**
- `*Evaluator` - 评估器实例
- `error` - 如果创建失败

**示例：**

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
}

evaluator, err := opa.NewEvaluatorWithConfig(ctx, config)
if err != nil {
    log.Fatalf("创建评估器失败: %v", err)
}
defer evaluator.Close(ctx)
```

### 评估器方法

#### Evaluate()

评估策略。

```go
func (e *Evaluator) Evaluate(ctx context.Context, path string, input interface{}) (*Result, error)
```

**示例：**

```go
result, err := evaluator.Evaluate(ctx, "authz/allow", input)
```

#### IsAllowed()

评估是否允许（便捷方法）。

```go
func (e *Evaluator) IsAllowed(ctx context.Context, path string, input interface{}) (bool, error)
```

**参数：**
- `ctx` - 上下文
- `path` - 决策路径
- `input` - 策略输入

**返回值：**
- `bool` - 是否允许
- `error` - 如果评估失败

**示例：**

```go
allowed, err := evaluator.IsAllowed(ctx, "authz/allow", input)
if err != nil {
    log.Printf("评估失败: %v", err)
    return
}

if allowed {
    log.Println("访问允许")
}
```

#### EvaluateWithDefault()

评估策略，失败时返回默认值。

```go
func (e *Evaluator) EvaluateWithDefault(ctx context.Context, path string, input interface{}, defaultValue bool) bool
```

**参数：**
- `ctx` - 上下文
- `path` - 决策路径
- `input` - 策略输入
- `defaultValue` - 默认值（评估失败时返回）

**返回值：**
- `bool` - 评估结果或默认值

**示例：**

```go
// 评估失败时默认拒绝访问
allowed := evaluator.EvaluateWithDefault(ctx, "authz/allow", input, false)

if allowed {
    log.Println("访问允许")
} else {
    log.Println("访问拒绝或评估失败")
}
```

#### BatchEvaluate()

批量评估策略。

```go
func (e *Evaluator) BatchEvaluate(ctx context.Context, path string, inputs []interface{}) ([]*Result, error)
```

**参数：**
- `ctx` - 上下文
- `path` - 决策路径
- `inputs` - 输入列表

**返回值：**
- `[]*Result` - 结果列表
- `error` - 如果任何评估失败

**示例：**

```go
inputs := []interface{}{
    map[string]interface{}{"user": "alice", "action": "read"},
    map[string]interface{}{"user": "bob", "action": "write"},
    map[string]interface{}{"user": "charlie", "action": "delete"},
}

results, err := evaluator.BatchEvaluate(ctx, "authz/allow", inputs)
if err != nil {
    log.Printf("批量评估失败: %v", err)
    return
}

for i, result := range results {
    log.Printf("输入 %d: 允许=%v", i, result.Allowed)
}
```

#### EvaluateRBAC()

评估 RBAC 策略。

```go
func (e *Evaluator) EvaluateRBAC(ctx context.Context, input *RBACInput) (*Result, error)
```

**参数：**
- `ctx` - 上下文
- `input` - RBAC 输入

**返回值：**
- `*Result` - 评估结果
- `error` - 如果评估失败

**示例：**

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
    log.Printf("RBAC 评估失败: %v", err)
    return
}

if result.Allowed {
    log.Println("RBAC: 访问允许")
}
```

#### EvaluateABAC()

评估 ABAC 策略。

```go
func (e *Evaluator) EvaluateABAC(ctx context.Context, input *ABACInput) (*Result, error)
```

**参数：**
- `ctx` - 上下文
- `input` - ABAC 输入

**返回值：**
- `*Result` - 评估结果
- `error` - 如果评估失败

**示例：**

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
    Environment: &opa.Environment{
        Time: map[string]interface{}{
            "weekday": 3,
            "hour":    14,
        },
        Location:   "office",
        IPAddress:  "10.0.1.100",
        DeviceType: "laptop",
    },
}

result, err := evaluator.EvaluateABAC(ctx, input)
if err != nil {
    log.Printf("ABAC 评估失败: %v", err)
    return
}

if result.Allowed {
    log.Println("ABAC: 访问允许")
    log.Printf("评分: %d/%d", result.Score, result.RequiredScore)
}
```

#### Close()

关闭评估器。

```go
func (e *Evaluator) Close(ctx context.Context) error
```

---

## 策略输入构建

### PolicyInput 结构体

策略输入的标准化数据结构。

```go
type PolicyInput struct {
    // Request 请求信息
    Request RequestInfo `json:"request"`
    
    // User 用户信息
    User UserInfo `json:"user"`
    
    // Resource 资源信息
    Resource ResourceInfo `json:"resource"`
    
    // Custom 自定义数据
    Custom map[string]interface{} `json:"custom,omitempty"`
}
```

### PolicyInputBuilder

策略输入构建器提供了流式 API。

#### NewPolicyInputBuilder()

创建策略输入构建器。

```go
func NewPolicyInputBuilder() *PolicyInputBuilder
```

**返回值：**
- `*PolicyInputBuilder` - 构建器实例

**示例：**

```go
builder := opa.NewPolicyInputBuilder()
```

#### FromHTTPRequest()

从 Gin HTTP 请求构建输入。

```go
func (b *PolicyInputBuilder) FromHTTPRequest(c *gin.Context) *PolicyInputBuilder
```

**参数：**
- `c` - Gin 上下文

**返回值：**
- `*PolicyInputBuilder` - 构建器实例（链式调用）

**示例：**

```go
func authzMiddleware(evaluator *opa.Evaluator) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 HTTP 请求构建输入
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

#### FromGRPCContext()

从 gRPC 上下文构建输入。

```go
func (b *PolicyInputBuilder) FromGRPCContext(ctx context.Context, fullMethod string) *PolicyInputBuilder
```

**参数：**
- `ctx` - gRPC 上下文
- `fullMethod` - 完整的 gRPC 方法名

**返回值：**
- `*PolicyInputBuilder` - 构建器实例（链式调用）

**示例：**

```go
func UnaryAuthzInterceptor(evaluator *opa.Evaluator) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 从 gRPC 上下文构建输入
        input := opa.NewPolicyInputBuilder().
            FromGRPCContext(ctx, info.FullMethod).
            WithUserID("alice").
            WithUserRoles([]string{"editor"}).
            Build()
        
        result, err := evaluator.Evaluate(ctx, "authz/allow", input)
        if err != nil {
            return nil, status.Error(codes.Internal, "authorization check failed")
        }
        
        if !result.Allowed {
            return nil, status.Error(codes.PermissionDenied, "access denied")
        }
        
        return handler(ctx, req)
    }
}
```

#### 用户相关方法

设置用户信息的便捷方法。

```go
func (b *PolicyInputBuilder) WithUser(user UserInfo) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserID(id string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUsername(username string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserEmail(email string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserRoles(roles []string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserPermissions(permissions []string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserGroups(groups []string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithUserAttribute(key string, value interface{}) *PolicyInputBuilder
```

**示例：**

```go
input := opa.NewPolicyInputBuilder().
    WithUserID("alice").
    WithUsername("alice").
    WithUserEmail("alice@example.com").
    WithUserRoles([]string{"editor", "viewer"}).
    WithUserPermissions([]string{"read:docs", "write:docs"}).
    WithUserGroups([]string{"engineering"}).
    WithUserAttribute("department", "engineering").
    WithUserAttribute("clearance_level", 3).
    Build()
```

#### 资源相关方法

设置资源信息的便捷方法。

```go
func (b *PolicyInputBuilder) WithResource(resource ResourceInfo) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceType(resourceType string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceID(id string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceName(name string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceOwner(owner string) *PolicyInputBuilder
func (b *PolicyInputBuilder) WithResourceAttribute(key string, value interface{}) *PolicyInputBuilder
```

**示例：**

```go
input := opa.NewPolicyInputBuilder().
    WithResourceType("document").
    WithResourceID("doc-123").
    WithResourceName("Quarterly Report").
    WithResourceOwner("bob").
    WithResourceAttribute("department", "engineering").
    WithResourceAttribute("security_level", 2).
    Build()
```

#### 自定义数据方法

```go
func (b *PolicyInputBuilder) WithCustomData(key string, value interface{}) *PolicyInputBuilder
```

**示例：**

```go
input := opa.NewPolicyInputBuilder().
    WithCustomData("transaction_id", "txn-456").
    WithCustomData("source_ip", "10.0.1.100").
    Build()
```

#### Build()

构建最终的策略输入。

```go
func (b *PolicyInputBuilder) Build() *PolicyInput
```

**返回值：**
- `*PolicyInput` - 策略输入

---

### 便捷函数

#### BuildPolicyInputFromHTTP()

从 HTTP 请求构建策略输入（便捷函数）。

```go
func BuildPolicyInputFromHTTP(c *gin.Context) *PolicyInput
```

**参数：**
- `c` - Gin 上下文

**返回值：**
- `*PolicyInput` - 策略输入

**示例：**

```go
func authzHandler(c *gin.Context) {
    input := opa.BuildPolicyInputFromHTTP(c)
    // 自动提取请求信息和用户信息（从上下文）
    
    result, err := evaluator.Evaluate(c.Request.Context(), "authz/allow", input)
    // ...
}
```

#### BuildPolicyInputFromGRPC()

从 gRPC 上下文构建策略输入（便捷函数）。

```go
func BuildPolicyInputFromGRPC(ctx context.Context, fullMethod string) *PolicyInput
```

**参数：**
- `ctx` - gRPC 上下文
- `fullMethod` - 完整的 gRPC 方法名

**返回值：**
- `*PolicyInput` - 策略输入

**示例：**

```go
func grpcAuthzInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) error {
    input := opa.BuildPolicyInputFromGRPC(ctx, info.FullMethod)
    // 自动提取请求信息和用户信息（从上下文）
    
    result, err := evaluator.Evaluate(ctx, "authz/allow", input)
    // ...
}
```

#### ExtractUserFromContext()

从 Gin 上下文提取用户信息。

```go
func ExtractUserFromContext(c *gin.Context) UserInfo
```

**参数：**
- `c` - Gin 上下文

**返回值：**
- `UserInfo` - 用户信息

**提取来源：**
- 从上下文键提取：`user_id`, `username`, `email`, `roles`, `permissions`, `groups`
- 从 JWT claims 提取：`sub`, `email`, 其他自定义 claims

**示例：**

```go
func middleware(c *gin.Context) {
    // 假设 JWT 中间件已经设置了用户信息
    user := opa.ExtractUserFromContext(c)
    
    log.Printf("用户: %s, 角色: %v", user.Username, user.Roles)
}
```

#### ExtractUserFromGRPCContext()

从 gRPC 上下文提取用户信息。

```go
func ExtractUserFromGRPCContext(ctx context.Context) UserInfo
```

**参数：**
- `ctx` - gRPC 上下文

**返回值：**
- `UserInfo` - 用户信息

---

## RBAC 与 ABAC

### RBAC 输入结构

```go
type RBACInput struct {
    // Subject 主体（用户）
    Subject *Subject `json:"subject"`
    
    // Action 操作
    Action string `json:"action"`
    
    // Resource 资源
    Resource string `json:"resource"`
    
    // Context 上下文（可选）
    Context map[string]interface{} `json:"context,omitempty"`
}

type Subject struct {
    // User 用户标识符
    User string `json:"user"`
    
    // Roles 角色列表
    Roles []string `json:"roles"`
    
    // Groups 用户组列表（可选）
    Groups []string `json:"groups,omitempty"`
    
    // Attributes 其他属性（可选）
    Attributes map[string]interface{} `json:"attributes,omitempty"`
}
```

### ABAC 输入结构

```go
type ABACInput struct {
    // Subject 主体（用户）
    Subject *Subject `json:"subject"`
    
    // Action 操作
    Action string `json:"action"`
    
    // Resource 资源
    Resource *Resource `json:"resource"`
    
    // Environment 环境
    Environment *Environment `json:"environment,omitempty"`
}

type Resource struct {
    // Type 资源类型
    Type string `json:"type"`
    
    // ID 资源 ID
    ID string `json:"id,omitempty"`
    
    // Owner 资源所有者
    Owner string `json:"owner,omitempty"`
    
    // Attributes 资源属性
    Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type Environment struct {
    // Time 时间信息
    Time map[string]interface{} `json:"time,omitempty"`
    
    // Location 地理位置
    Location string `json:"location,omitempty"`
    
    // IPAddress IP 地址
    IPAddress string `json:"ip_address,omitempty"`
    
    // DeviceType 设备类型
    DeviceType string `json:"device_type,omitempty"`
    
    // Additional 其他环境属性
    Additional map[string]interface{} `json:"additional,omitempty"`
}
```

---

## 策略管理

### Manager 接口

策略管理器接口（用于高级策略管理）。

```go
type Manager interface {
    // LoadPolicy 加载策略
    LoadPolicy(ctx context.Context, name string, policy string) error
    
    // LoadPolicyFromFile 从文件加载策略
    LoadPolicyFromFile(ctx context.Context, name, path string) error
    
    // RemovePolicy 移除策略
    RemovePolicy(ctx context.Context, name string) error
    
    // ListPolicies 列出所有策略
    ListPolicies(ctx context.Context) ([]string, error)
    
    // Watch 监听策略变化
    Watch(ctx context.Context, callback func(event PolicyEvent)) error
    
    // Close 关闭管理器
    Close(ctx context.Context) error
}
```

### 创建管理器

#### NewManager()

从现有客户端创建管理器。

```go
func NewManager(client Client) (Manager, error)
```

#### NewManagerWithConfig()

使用配置创建管理器。

```go
func NewManagerWithConfig(ctx context.Context, config *Config) (Manager, error)
```

**示例：**

```go
manager, err := opa.NewManagerWithConfig(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer manager.Close(ctx)

// 加载策略
policy := `
package authz

import rego.v1

default allow = false

allow if {
    input.user == "admin"
}
`

err = manager.LoadPolicy(ctx, "admin.rego", policy)
if err != nil {
    log.Printf("加载策略失败: %v", err)
}
```

---

## 缓存管理

### 启用缓存

```go
config := &opa.Config{
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       10000,
        TTL:           5 * time.Minute,
        EnableMetrics: true,
    },
}
```

### 缓存统计

```go
// 通过客户端访问缓存（具体实现取决于客户端类型）
// 这是未来的改进点
```

---

## 监控与指标

### 评估指标

每次评估都会返回性能指标。

```go
result, err := client.Evaluate(ctx, "authz/allow", input)
if err != nil {
    return err
}

if result.Metrics != nil {
    log.Printf("评估时间: %d ns", result.Metrics.TimerEvalNs)
    log.Printf("查询时间: %d ns", result.Metrics.TimerRegoQueryEvalNs)
}
```

---

## 代码示例

### 完整的 HTTP 授权中间件

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/security/opa"
)

func main() {
    // 创建 OPA 评估器
    ctx := context.Background()
    config := &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./policies",
        },
        DefaultDecisionPath: "authz/allow",
    }

    evaluator, err := opa.NewEvaluatorWithConfig(ctx, config)
    if err != nil {
        log.Fatalf("创建评估器失败: %v", err)
    }
    defer evaluator.Close(ctx)

    // 创建 Gin 路由
    r := gin.Default()

    // 应用授权中间件
    r.Use(AuthzMiddleware(evaluator))

    r.GET("/api/documents/:id", handleGetDocument)
    r.POST("/api/documents", handleCreateDocument)
    r.PUT("/api/documents/:id", handleUpdateDocument)
    r.DELETE("/api/documents/:id", handleDeleteDocument)

    log.Println("服务器启动在 :8080")
    r.Run(":8080")
}

func AuthzMiddleware(evaluator *opa.Evaluator) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 构建策略输入
        input := opa.NewPolicyInputBuilder().
            FromHTTPRequest(c).
            WithUserID(getCurrentUserID(c)).
            WithUserRoles(getCurrentUserRoles(c)).
            WithResourceType("document").
            WithResourceID(c.Param("id")).
            Build()

        // 评估策略
        result, err := evaluator.Evaluate(c.Request.Context(), "", input)
        if err != nil {
            log.Printf("策略评估失败: %v", err)
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
                "error": "Authorization check failed",
            })
            return
        }

        if !result.Allowed {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "Access denied",
            })
            return
        }

        // 继续处理请求
        c.Next()
    }
}

func getCurrentUserID(c *gin.Context) string {
    // 从上下文或 JWT 获取用户 ID
    if userID, exists := c.Get("user_id"); exists {
        return userID.(string)
    }
    return ""
}

func getCurrentUserRoles(c *gin.Context) []string {
    // 从上下文或 JWT 获取用户角色
    if roles, exists := c.Get("roles"); exists {
        return roles.([]string)
    }
    return []string{}
}

func handleGetDocument(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Document retrieved"})
}

func handleCreateDocument(c *gin.Context) {
    c.JSON(http.StatusCreated, gin.H{"message": "Document created"})
}

func handleUpdateDocument(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Document updated"})
}

func handleDeleteDocument(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}
```

### gRPC 拦截器示例

```go
package main

import (
    "context"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "github.com/innovationmech/swit/pkg/security/opa"
)

func UnaryAuthzInterceptor(evaluator *opa.Evaluator) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 构建策略输入
        input := opa.NewPolicyInputBuilder().
            FromGRPCContext(ctx, info.FullMethod).
            WithUserID(getUserIDFromContext(ctx)).
            WithUserRoles(getUserRolesFromContext(ctx)).
            Build()

        // 评估策略
        result, err := evaluator.Evaluate(ctx, "authz/allow", input)
        if err != nil {
            log.Printf("策略评估失败: %v", err)
            return nil, status.Error(codes.Internal, "authorization check failed")
        }

        if !result.Allowed {
            return nil, status.Error(codes.PermissionDenied, "access denied")
        }

        // 继续处理请求
        return handler(ctx, req)
    }
}

func getUserIDFromContext(ctx context.Context) string {
    if userID := ctx.Value("user_id"); userID != nil {
        return userID.(string)
    }
    return ""
}

func getUserRolesFromContext(ctx context.Context) []string {
    if roles := ctx.Value("roles"); roles != nil {
        return roles.([]string)
    }
    return []string{}
}
```

---

## 最佳实践

### 1. 选择合适的运行模式

**嵌入式模式：**
- ✅ 适合：开发、测试、小型部署
- ✅ 优点：简单、快速、无外部依赖
- ❌ 缺点：策略更新需要重启

**远程模式：**
- ✅ 适合：生产环境、集中管理
- ✅ 优点：策略集中管理、动态更新
- ❌ 缺点：网络延迟、需要额外的 OPA 服务器

**Sidecar 模式：**
- ✅ 适合：Kubernetes 部署
- ✅ 优点：低延迟、易于扩展
- ❌ 缺点：增加 pod 资源消耗

### 2. 启用决策缓存

对于高并发场景，启用缓存可显著提升性能：

```go
config.CacheConfig = &opa.CacheConfig{
    Enabled:       true,
    MaxSize:       10000,
    TTL:           5 * time.Minute,
    EnableMetrics: true,
}
```

### 3. 使用合适的策略模板

- **RBAC** - 适合基于角色的简单场景
- **ABAC** - 适合复杂的属性based场景

### 4. 策略版本控制

将策略文件纳入版本控制：

```bash
git add pkg/security/opa/policies/rbac.rego
git commit -m "feat(opa): add new role for report admin"
git tag -a policy-v1.2.0 -m "Policy v1.2.0"
```

### 5. 测试策略

使用 OPA CLI 测试策略：

```bash
opa test pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego -v
```

### 6. 监控性能

启用指标并监控评估性能：

```go
result, err := evaluator.Evaluate(ctx, "authz/allow", input)
if err != nil {
    return err
}

if result.Metrics != nil && result.Metrics.TimerEvalNs > 1000000 { // > 1ms
    log.Printf("WARNING: Slow policy evaluation: %d ns", result.Metrics.TimerEvalNs)
}
```

### 7. 错误处理

实现适当的错误处理和回退策略：

```go
allowed := evaluator.EvaluateWithDefault(ctx, "authz/allow", input, false) // 默认拒绝

if !allowed {
    c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
        "error": "Access denied",
    })
    return
}
```

### 8. 安全的策略输入

不要在策略输入中包含敏感信息：

```go
// ✅ 好的做法
builder.WithUserID("alice").
    WithUserRoles([]string{"editor"})

// ❌ 避免
builder.WithUserAttribute("password", "secret123")
```

### 9. 使用构建器模式

使用 `PolicyInputBuilder` 构建结构化的输入：

```go
input := opa.NewPolicyInputBuilder().
    FromHTTPRequest(c).
    WithUserRoles(roles).
    WithResourceType("document").
    Build()
```

### 10. 审计日志

利用策略返回的审计信息：

```go
result, err := evaluator.Evaluate(ctx, "authz/allow", input)
if err == nil && result.Bindings != nil {
    if auditLog, ok := result.Bindings["audit_log"]; ok {
        // 记录审计日志
        log.Printf("Audit: %+v", auditLog)
    }
}
```

---

## 相关文档

- [OAuth2/OIDC API 参考](./security-oauth2.md)
- [安全配置参考](./security-config.md)
- [OPA RBAC 策略指南](../opa-rbac-guide.md)
- [OPA ABAC 策略指南](../opa-abac-guide.md)
- [OPA 包 README](../../pkg/security/opa/README.md)

## 许可证

Copyright (c) 2024-2025 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.







