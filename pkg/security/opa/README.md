# OPA Security Package

本包提供 Open Policy Agent (OPA) 策略引擎集成，用于实现细粒度的基于策略的访问控制。

## 功能特性

- **双模式支持**：支持嵌入式模式和远程模式
  - 嵌入式模式：OPA 引擎运行在同一进程中
  - 远程模式：连接到外部 OPA 服务器
- **策略管理**：动态加载、更新和移除策略
- **决策缓存**：内置决策缓存机制，提升性能
- **策略模板**：提供 RBAC 和 ABAC 策略模板
- **评估器 API**：高层抽象的策略评估接口

## 目录结构

```
pkg/security/opa/
├── config.go           # OPA 配置
├── client.go           # 客户端接口
├── embedded_client.go  # 嵌入式客户端实现
├── remote_client.go    # 远程客户端实现
├── manager.go          # 策略管理器
├── evaluator.go        # 策略评估器
├── cache.go            # 决策缓存
└── policies/           # 策略模板
    ├── rbac.rego       # RBAC 策略模板
    ├── abac.rego       # ABAC 策略模板
    └── examples/       # 示例策略
```

## 快速开始

### 嵌入式模式

```go
package main

import (
    "context"
    "log"
    
    "github.com/innovationmech/swit/pkg/security/opa"
)

func main() {
    ctx := context.Background()
    
    // 创建配置
    config := &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./policies",
        },
        DefaultDecisionPath: "authz/allow",
    }
    
    // 创建客户端
    client, err := opa.NewClient(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close(ctx)
    
    // 评估策略
    result, err := client.Evaluate(ctx, "", map[string]interface{}{
        "user":   "alice",
        "action": "read",
        "resource": "documents",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    if result.Allowed {
        log.Println("Access granted")
    } else {
        log.Println("Access denied")
    }
}
```

### 远程模式

```go
config := &opa.Config{
    Mode: opa.ModeRemote,
    RemoteConfig: &opa.RemoteConfig{
        URL: "http://localhost:8181",
        Timeout: 30 * time.Second,
    },
    DefaultDecisionPath: "authz/allow",
}

client, err := opa.NewClient(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer client.Close(ctx)
```

### 使用评估器

```go
// 创建评估器
evaluator, err := opa.NewEvaluatorWithConfig(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer evaluator.Close(ctx)

// RBAC 评估
input := &opa.RBACInput{
    Subject: &opa.Subject{
        User:  "alice",
        Roles: []string{"editor"},
    },
    Action:   "write",
    Resource: "documents",
}

result, err := evaluator.EvaluateRBAC(ctx, input)
if err != nil {
    log.Fatal(err)
}

if result.Allowed {
    log.Println("RBAC: Access granted")
}
```

### 使用策略管理器

```go
// 创建管理器
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

allow = true if {
    input.role == "admin"
}
`

err = manager.LoadPolicy(ctx, "admin.rego", policy)
if err != nil {
    log.Fatal(err)
}

// 从文件加载
err = manager.LoadPolicyFromFile(ctx, "rbac.rego", "./policies/rbac.rego")
if err != nil {
    log.Fatal(err)
}
```

## 配置选项

### 嵌入式模式配置

```yaml
mode: embedded
embedded:
  policy_dir: "./policies"
  enable_logging: true
  enable_decision_logs: false
cache:
  enabled: true
  max_size: 10000
  ttl: 5m
```

### 远程模式配置

```yaml
mode: remote
remote:
  url: "http://localhost:8181"
  timeout: 30s
  max_retries: 3
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    ca_file: "/path/to/ca.pem"
  auth:
    type: bearer
    token: "your-token"
cache:
  enabled: true
  max_size: 10000
  ttl: 5m
```

## 策略模板

### RBAC 策略

基于角色的访问控制策略模板位于 `policies/rbac.rego`。

特性：
- 角色权限映射
- 角色资源映射
- 用户组支持
- 资源所有者检查
- 临时权限支持

### ABAC 策略

基于属性的访问控制策略模板位于 `policies/abac.rego`。

特性：
- 主体属性验证
- 资源属性验证
- 环境属性验证
- 安全级别检查
- 地理位置限制
- 时间窗口限制
- 动态属性评分

## 性能优化

### 缓存配置

启用缓存可显著提升策略评估性能：

```go
config := &opa.Config{
    CacheConfig: &opa.CacheConfig{
        Enabled: true,
        MaxSize: 10000,
        TTL:     5 * time.Minute,
        EnableMetrics: true,
    },
}

// 获取缓存统计
stats := cache.Stats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate * 100)
```

### 预编译策略

在生产环境中，建议预编译策略以减少启动时间。

## 测试

运行单元测试：

```bash
go test ./pkg/security/opa/...
```

运行覆盖率测试：

```bash
go test ./pkg/security/opa/... -cover
```

## 依赖

- `github.com/open-policy-agent/opa` v1.10.1+

## 注意事项

1. **策略语法**：策略必须使用 Rego v1 语法（`import rego.v1`）
2. **性能**：对于高并发场景，建议启用缓存
3. **安全**：远程模式应启用 TLS 和认证
4. **测试**：建议为自定义策略编写完善的测试

## 未来计划

- [ ] 支持 Bundle 动态更新
- [ ] 完善 OPA 查询变量绑定问题
- [ ] 添加更多策略示例
- [ ] 支持策略版本管理
- [ ] 集成 Prometheus 指标
- [ ] 添加策略测试工具

## 相关链接

- [Open Policy Agent 官方文档](https://www.openpolicyagent.org/docs/latest/)
- [Rego 语言参考](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [Epic #776: OPA 策略引擎集成](https://github.com/innovationmech/swit/issues/776)
- [Task #791: OPA 依赖和基础结构](https://github.com/innovationmech/swit/issues/791)

