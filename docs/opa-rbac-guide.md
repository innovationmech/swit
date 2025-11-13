# OPA RBAC 策略使用指南

本文档介绍如何使用 OPA (Open Policy Agent) RBAC (基于角色的访问控制) 策略模板来实现细粒度的权限管理。

## 目录

- [概述](#概述)
- [策略架构](#策略架构)
- [快速开始](#快速开始)
- [角色定义](#角色定义)
- [权限映射](#权限映射)
- [高级特性](#高级特性)
- [测试策略](#测试策略)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)

## 概述

RBAC 策略模板提供了一个完整的基于角色的访问控制解决方案，支持：

- ✅ **角色权限映射** - 将操作权限分配给不同角色
- ✅ **角色资源映射** - 控制角色可以访问的资源类型
- ✅ **超级管理员支持** - 管理员拥有所有权限
- ✅ **资源所有者检查** - 用户可以访问自己的资源
- ✅ **用户组支持** - 基于组的角色继承
- ✅ **临时权限** - 支持时限性的临时访问权限
- ✅ **审计日志** - 记录所有访问决策

## 策略架构

### 核心决策流程

```
┌─────────────────┐
│  请求输入 (Input)│
│  - subject     │
│  - action      │
│  - resource    │
│  - context     │
└────────┬────────┘
         │
         v
┌────────────────────┐
│  RBAC 策略评估     │
│  1. 超级管理员?    │
│  2. 角色权限匹配?  │
│  3. 资源所有者?    │
│  4. 组角色继承?    │
│  5. 临时权限?      │
└────────┬───────────┘
         │
         v
┌────────────────────┐
│   决策结果         │
│   allow: true/false│
└────────────────────┘
```

### 输入数据结构

策略期望以下输入结构：

```json
{
  "subject": {
    "user": "alice",
    "roles": ["editor", "viewer"],
    "groups": ["engineering"],
    "attributes": {}
  },
  "action": "read",
  "resource": "documents",
  "context": {
    "time": true,
    "temporary_permissions": [
      {
        "action": "write",
        "resource": "project/secret",
        "expires_at": 1735689600000000000
      }
    ]
  }
}
```

## 快速开始

### 1. 加载 RBAC 策略

使用嵌入式模式加载策略：

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
            PolicyDir: "./pkg/security/opa/policies",
        },
        DefaultDecisionPath: "rbac/allow",
    }
    
    // 创建客户端
    client, err := opa.NewClient(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close(ctx)
    
    // 评估策略
    result, err := client.Evaluate(ctx, "", map[string]interface{}{
        "subject": map[string]interface{}{
            "user":  "alice",
            "roles": []string{"editor"},
        },
        "action":   "update",
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

### 2. 使用评估器（推荐）

使用高级评估器 API：

```go
package main

import (
    "context"
    "log"
    
    "github.com/innovationmech/swit/pkg/security/opa"
)

func main() {
    ctx := context.Background()
    
    // 创建评估器
    evaluator, err := opa.NewEvaluatorWithConfig(ctx, &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./pkg/security/opa/policies",
        },
    })
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
        Action:   "update",
        Resource: "documents",
    }
    
    result, err := evaluator.EvaluateRBAC(ctx, input)
    if err != nil {
        log.Fatal(err)
    }
    
    if result.Allowed {
        log.Println("Access granted")
    }
}
```

## 角色定义

### 预定义角色

RBAC 策略模板包含以下预定义角色：

#### 1. Admin (超级管理员)
- **描述**: 拥有所有权限的超级管理员
- **权限**: create, read, update, delete, list
- **资源**: users, documents, settings, reports
- **使用场景**: 系统管理员、运维人员

```rego
"admin": {
    "create": true,
    "read": true,
    "update": true,
    "delete": true,
    "list": true,
}
```

#### 2. Editor (编辑者)
- **描述**: 可以创建、读取和更新资源
- **权限**: create, read, update, list
- **资源**: documents, reports
- **使用场景**: 内容编辑、文档管理

```rego
"editor": {
    "create": true,
    "read": true,
    "update": true,
    "list": true,
}
```

#### 3. Viewer (查看者)
- **描述**: 只读访问权限
- **权限**: read, list
- **资源**: documents, reports
- **使用场景**: 普通用户、审核人员

```rego
"viewer": {
    "read": true,
    "list": true,
}
```

#### 4. Contributor (贡献者)
- **描述**: 可以创建和读取资源
- **权限**: create, read, list
- **资源**: documents
- **使用场景**: 内容创建者、提交者

```rego
"contributor": {
    "create": true,
    "read": true,
    "list": true,
}
```

### 自定义角色

你可以通过修改策略文件来添加自定义角色：

```rego
role_permissions := {
    # ... 现有角色 ...
    
    # 自定义角色: 审核员
    "auditor": {
        "read": true,
        "list": true,
        "audit": true,  # 自定义操作
    },
    
    # 自定义角色: 报表管理员
    "report_admin": {
        "create": true,
        "read": true,
        "update": true,
        "delete": true,
        "list": true,
        "export": true,  # 自定义操作
    },
}

role_resources := {
    # ... 现有映射 ...
    
    "auditor": {
        "audit_logs": true,
        "reports": true,
    },
    
    "report_admin": {
        "reports": true,
    },
}
```

## 权限映射

### 操作权限

策略支持以下标准操作：

| 操作 | 描述 | 适用角色 |
|-----|------|---------|
| `create` | 创建资源 | admin, editor, contributor |
| `read` | 读取资源 | admin, editor, viewer, contributor |
| `update` | 更新资源 | admin, editor |
| `delete` | 删除资源 | admin |
| `list` | 列出资源 | admin, editor, viewer, contributor |

### 资源类型

策略支持以下资源类型：

| 资源类型 | 描述 | 可访问角色 |
|---------|------|----------|
| `users` | 用户管理 | admin |
| `documents` | 文档管理 | admin, editor, viewer, contributor |
| `settings` | 系统设置 | admin |
| `reports` | 报表管理 | admin, editor, viewer |

### 添加自定义操作

```rego
# 在策略中添加新操作
role_permissions := {
    "custom_role": {
        "read": true,
        "export": true,      # 自定义操作
        "share": true,       # 自定义操作
        "analyze": true,     # 自定义操作
    },
}
```

## 高级特性

### 1. 资源所有者权限

用户自动获得对自己资源的访问权限：

```go
// Alice 可以访问 users/alice
input := map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{},  // 即使没有角色
    },
    "action":   "read",
    "resource": "users/alice",  // 资源路径包含用户名
}
// 结果: allow = true
```

策略实现：

```rego
# 检查是否为资源所有者
is_owner if {
    input.resource == sprintf("users/%s", [input.subject.user])
}

# 资源所有者拥有完全访问权限
allow if {
    is_owner
    input.action in ["read", "update", "delete"]
}
```

### 2. 用户组角色继承

通过用户组自动继承角色权限：

```go
input := map[string]interface{}{
    "subject": map[string]interface{}{
        "user":   "alice",
        "roles":  []string{},
        "groups": []string{"engineering"},  // Alice 属于 engineering 组
    },
    "action":   "update",
    "resource": "documents",
}
// engineering 组继承了 editor 和 contributor 角色
// 结果: allow = true
```

预定义组映射：

```rego
group_roles := {
    "engineering": ["editor", "contributor"],
    "management": ["admin"],
    "support": ["viewer"],
    "sales": ["viewer", "contributor"],
}
```

### 3. 临时权限

授予有时间限制的临时访问权限：

```go
input := map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{},
    },
    "action":   "write",
    "resource": "project/secret",
    "context": map[string]interface{}{
        "time": true,
        "temporary_permissions": []map[string]interface{}{
            {
                "action":     "write",
                "resource":   "project/secret",
                "expires_at": time.Now().Add(24 * time.Hour).UnixNano(),
            },
        },
    },
}
// 在过期时间前: allow = true
// 过期后: allow = false
```

策略实现：

```rego
# 临时权限检查
allow if {
    is_business_hours
    has_temporary_permission
}

# 检查临时权限
has_temporary_permission if {
    some temp_perm in input.context.temporary_permissions
    temp_perm.action == input.action
    temp_perm.resource == input.resource
    temp_perm.expires_at > time.now_ns()
}
```

### 4. 工作时间限制

策略支持基于时间的访问控制：

```rego
# 判断是否在工作时间（示例）
is_business_hours if {
    input.context.time
    # 实现实际的时间范围检查
    # 例如：周一到周五 9:00-18:00
}
```

### 5. 审计日志

策略自动生成审计日志：

```rego
audit_log := {
    "user": input.subject.user,
    "action": input.action,
    "resource": input.resource,
    "roles": user_roles,
    "decision": allow,
    "timestamp": time.now_ns(),
}
```

在应用中获取审计信息：

```go
result, err := client.Evaluate(ctx, "", input)
if err != nil {
    log.Fatal(err)
}

// 访问审计日志
if auditLog, ok := result.Bindings["audit_log"].(map[string]interface{}); ok {
    log.Printf("Audit: user=%s, action=%s, decision=%v",
        auditLog["user"],
        auditLog["action"],
        auditLog["decision"])
}
```

## 测试策略

### 运行测试

使用 OPA CLI 运行测试：

```bash
# 运行所有测试
opa test pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego -v

# 运行特定测试
opa test pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego -v -r test_admin_has_all_permissions
```

### 测试覆盖率

```bash
# 生成覆盖率报告
opa test --coverage pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego

# 生成 HTML 覆盖率报告
opa test --coverage --format=json pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego > coverage.json
```

### 编写自定义测试

```rego
package rbac_test

import rego.v1
import data.rbac

# 测试自定义角色
test_custom_role_has_export_permission if {
    rbac.allow with input as {
        "subject": {
            "user": "reporter",
            "roles": ["report_admin"],
        },
        "action": "export",
        "resource": "reports",
    }
}

# 测试拒绝场景
test_custom_role_cannot_delete_users if {
    not rbac.allow with input as {
        "subject": {
            "user": "reporter",
            "roles": ["report_admin"],
        },
        "action": "delete",
        "resource": "users",
    }
}
```

### 基准测试

评估策略性能：

```bash
# 运行基准测试
opa test --bench pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego
```

## 最佳实践

### 1. 最小权限原则

始终遵循最小权限原则，只授予用户完成任务所需的最小权限：

```go
// 好的做法：为特定任务分配特定角色
user.Roles = []string{"viewer"}  // 只需要读权限

// 避免：过度授权
user.Roles = []string{"admin"}   // 不要轻易授予管理员权限
```

### 2. 使用用户组管理角色

对于大型团队，使用用户组来管理角色更加高效：

```go
// 推荐：通过组管理
user := User{
    ID:     "alice",
    Groups: []string{"engineering"},  // engineering 组自动继承相关角色
}

// 避免：为每个用户单独分配角色
user := User{
    ID:    "alice",
    Roles: []string{"editor", "contributor", "viewer"},  // 难以维护
}
```

### 3. 临时权限用于特殊场景

只在需要时授予临时权限，并设置合理的过期时间：

```go
// 授予 24 小时的临时访问权限
tempPermission := TempPermission{
    Action:    "write",
    Resource:  "sensitive/data",
    ExpiresAt: time.Now().Add(24 * time.Hour),
}
```

### 4. 定期审计访问日志

利用审计日志定期检查访问模式：

```go
// 记录所有访问决策
logger.Info("Access decision",
    zap.String("user", auditLog.User),
    zap.String("action", auditLog.Action),
    zap.Bool("allowed", auditLog.Decision),
    zap.Time("timestamp", auditLog.Timestamp))
```

### 5. 测试驱动策略开发

在修改策略前编写测试：

```rego
# 1. 先写测试
test_new_feature if {
    rbac.allow with input as {...}
}

# 2. 再实现策略
allow if {
    # 新功能实现
}

# 3. 运行测试验证
```

### 6. 版本控制策略文件

将策略文件纳入版本控制：

```bash
# 提交策略变更
git add pkg/security/opa/policies/rbac.rego
git commit -m "feat(rbac): add new role for report administrators"

# 使用标签标记策略版本
git tag -a v1.2.0 -m "RBAC Policy v1.2.0"
```

### 7. 分离策略和数据

将策略逻辑和数据分离，便于维护：

```rego
# rbac.rego - 策略逻辑
allow if {
    some role in user_roles
    role_permissions[role][input.action]
}

# data.json - 角色数据（可以从外部加载）
{
    "role_permissions": {
        "admin": {...},
        "editor": {...}
    }
}
```

## 故障排查

### 问题 1: 策略评估返回 false，但预期应该通过

**可能原因**：
- 输入数据格式不正确
- 角色或资源未正确定义
- 缺少必需的输入字段

**解决方法**：

```bash
# 使用 OPA REPL 调试
opa run pkg/security/opa/policies/rbac.rego

# 在 REPL 中测试
> import data.rbac
> rbac.allow with input as {"subject": {"user": "alice", "roles": ["editor"]}, "action": "read", "resource": "documents"}
true

# 检查中间结果
> rbac.user_roles with input as {"subject": {"user": "alice", "roles": ["editor"]}}
{"editor"}
```

### 问题 2: 性能问题，策略评估慢

**解决方法**：

1. 启用决策缓存：

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

2. 使用基准测试分析：

```bash
opa test --bench pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego
```

### 问题 3: 测试失败

**常见原因**：
- 策略逻辑变更
- 测试数据过期
- 输入格式不匹配

**解决方法**：

```bash
# 运行详细测试输出
opa test -v pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego

# 只运行失败的测试
opa test -v -r test_name pkg/security/opa/policies/rbac.rego pkg/security/opa/policies/rbac_test.rego
```

### 问题 4: 无法加载策略文件

**可能原因**：
- 文件路径不正确
- 策略语法错误
- 缺少 `import rego.v1`

**解决方法**：

```bash
# 验证策略语法
opa check pkg/security/opa/policies/rbac.rego

# 格式化策略文件
opa fmt -w pkg/security/opa/policies/rbac.rego
```

### 问题 5: 远程 OPA 服务器连接失败

**解决方法**：

```go
// 添加详细的错误处理
config := &opa.Config{
    Mode: opa.ModeRemote,
    RemoteConfig: &opa.RemoteConfig{
        URL:        "http://opa-server:8181",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
    },
}

client, err := opa.NewClient(ctx, config)
if err != nil {
    log.Printf("Failed to connect to OPA server: %v", err)
    // 回退到嵌入式模式
    config.Mode = opa.ModeEmbedded
    client, err = opa.NewClient(ctx, config)
}
```

### 调试技巧

1. **启用详细日志**：

```go
config := &opa.Config{
    LogLevel: "debug",
}
```

2. **使用 OPA 决策日志**：

```rego
# 在策略中添加调试信息
debug_info := {
    "user_roles": user_roles,
    "has_permission": has_permission("read"),
    "can_access": can_access_resource("documents"),
}
```

3. **逐步验证**：

```bash
# 验证单个规则
opa eval -d pkg/security/opa/policies/rbac.rego "data.rbac.user_roles" \
  --input <(echo '{"subject": {"user": "alice", "roles": ["editor"]}}')
```

## 相关资源

- [OPA 官方文档](https://www.openpolicyagent.org/docs/latest/)
- [Rego 语言参考](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [pkg/security/opa/README.md](../pkg/security/opa/README.md) - OPA 包文档
- [Epic #776: OPA 策略引擎集成](https://github.com/innovationmech/swit/issues/776)
- [Task #799: RBAC 策略模板](https://github.com/innovationmech/swit/issues/799)

## 许可证

Copyright (c) 2024 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.

