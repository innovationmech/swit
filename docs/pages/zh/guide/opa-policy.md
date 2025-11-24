# OPA 策略编写指南

本指南介绍如何使用 Open Policy Agent (OPA) 编写和集成策略，包括 Rego 语言基础、最佳实践、RBAC/ABAC 示例和性能优化。

## 快速导航

- [概述](#概述) - OPA 核心概念
- [Rego 语言基础](#rego-语言基础) - 语法和数据类型
- [最佳实践](#最佳实践) - 策略编写规范
- [RBAC 策略](#rbac-策略) - 基于角色的访问控制
- [ABAC 策略](#abac-策略) - 基于属性的访问控制
- [测试](#测试) - 策略测试方法
- [性能优化](#性能优化) - 提升策略性能
- [FAQ](#常见问题) - 常见问题解答

## 概述

Open Policy Agent (OPA) 是一个开源的通用策略引擎，使用声明式语言 Rego 定义策略，支持 API 授权、数据过滤、Kubernetes 准入控制等场景。

### 核心优势

| 特性 | 传统方案 | OPA 方案 |
|------|---------|---------|
| 策略位置 | 分散在代码中 | 集中管理 |
| 策略变更 | 需要重新部署 | 动态加载 |
| 策略测试 | 难以测试 | 独立测试 |
| 策略复用 | 重复编写 | 模块化复用 |

### 工作原理

```
应用程序 ──查询──> OPA 引擎 ──评估──> 决策结果
                    ↑
                加载策略
                    │
                Rego 策略文件
```

## Rego 语言基础

### 基本语法

```rego
package example

import rego.v1

# 默认拒绝
default allow := false

# 简单规则
allow if {
    input.user == "alice"
}

# 带条件的规则
allow if {
    input.user == "bob"
    input.action == "read"
}

# 超级管理员
allow if {
    "admin" in input.user.roles
}
```

### 数据类型

```rego
# 字符串
name := "alice"

# 数字
count := 42

# 布尔值
is_admin := true

# 数组
users := ["alice", "bob"]

# 对象
user := {"name": "alice", "role": "admin"}

# 集合
admins := {"alice", "bob"}
```

### 常用操作

```rego
# 比较
x == y    # 等于
x != y    # 不等于
x > y     # 大于

# 成员检查
"alice" in ["alice", "bob"]
"admin" in user.roles

# 迭代
some role in input.user.roles
every item in [1, 2, 3] { item > 0 }

# 否定
not condition
```

### 内置函数

```rego
# 字符串函数
startswith("hello world", "hello")  # true
contains("hello world", "world")     # true
lower("HELLO")                       # "hello"
split("a,b,c", ",")                  # ["a", "b", "c"]

# 数组函数
count([1, 2, 3])                    # 3
concat(",", ["a", "b"])             # "a,b"

# 对象函数
object.get(obj, "key", "default")

# 时间函数
time.now_ns()
```

## 最佳实践

### 1. 默认拒绝原则

```rego
# ✅ 推荐
default allow := false

allow if {
    # 明确的允许条件
}

# ❌ 避免
default allow := true  # 不安全！
```

### 2. 模块化设计

```rego
# ✅ 推荐：拆分为小规则
allow if {
    user_authenticated
    user_authorized
    resource_accessible
}

user_authenticated if {
    input.user.token != ""
}

# ❌ 避免：单体规则
allow if {
    input.user.token != ""
    some role in input.user.roles
    # ... 所有逻辑混在一起
}
```

### 3. 有意义的命名

```rego
# ✅ 推荐
user_is_admin if {
    "admin" in input.user.roles
}

# ❌ 避免
check1 if {
    "admin" in input.user.roles
}
```

### 4. 输入验证

```rego
# ✅ 推荐：验证输入
allow if {
    input.subject
    input.subject.user
    input.action
    input.resource
    # 然后进行授权检查
    user_authorized
}
```

### 5. 添加文档

```rego
# RBAC 策略
#
# 输入格式：
#   {"subject": {"user": "alice", "roles": ["editor"]},
#    "action": "read", "resource": "documents"}
#
# 返回：allow: true/false
#
package rbac

import rego.v1

default allow := false
```

### 6. 性能优化

```rego
# ✅ 快速路径优先
allow if {
    "admin" in input.subject.roles  # 快
}

allow if {
    complex_calculation  # 慢
}

# ❌ 复杂检查优先
allow if {
    complex_calculation  # 总是执行
}
```

## RBAC 策略

基于角色的访问控制 (RBAC) 是最常用的访问控制模型。

### 基础策略

```rego
package rbac

import rego.v1

default allow := false

# 主决策规则
allow if {
    some role in user_roles
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}

# 超级管理员
allow if {
    "admin" in user_roles
}

# 获取用户角色
user_roles contains role if {
    some role in input.subject.roles
}

# 角色权限映射
role_permissions := {
    "admin": {
        "create": true,
        "read": true,
        "update": true,
        "delete": true,
    },
    "editor": {
        "create": true,
        "read": true,
        "update": true,
    },
    "viewer": {
        "read": true,
    },
}
```

### 使用示例

```go
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
    log.Println("✅ 访问允许")
}
```

### 高级特性

#### 资源所有者权限

```rego
# 用户可以访问自己的资源
allow if {
    is_owner
    input.action in ["read", "update", "delete"]
}

is_owner if {
    input.resource == sprintf("users/%s", [input.subject.user])
}
```

#### 用户组角色继承

```rego
# 通过用户组继承角色
user_roles contains role if {
    some group in input.subject.groups
    some role in group_roles[group]
}

group_roles := {
    "engineering": ["editor", "contributor"],
    "management": ["admin"],
}
```

#### 临时权限

```rego
# 支持临时权限
allow if {
    has_temporary_permission
}

has_temporary_permission if {
    some temp_perm in input.context.temporary_permissions
    temp_perm.action == input.action
    temp_perm.resource == input.resource
    temp_perm.expires_at > time.now_ns()
}
```

## ABAC 策略

基于属性的访问控制 (ABAC) 提供更细粒度的控制。

### 基础策略

```rego
package abac

import rego.v1

default allow := false

# 主决策规则
allow if {
    subject_attributes_valid
    resource_attributes_valid
    action_attributes_valid
    environment_attributes_valid
}

# 超级管理员
allow if {
    "admin" in input.subject.roles
}

# 主体属性验证
subject_attributes_valid if {
    input.subject.user
    count(input.subject.roles) > 0
}

# 资源属性验证
resource_attributes_valid if {
    input.resource.type
    input.resource.id
}
```

### 访问控制规则

#### 基于部门

```rego
allow if {
    input.subject.attributes.department == input.resource.attributes.department
    input.action == "read"
}
```

#### 基于安全级别

```rego
allow if {
    user_clearance_level >= resource_security_level
    input.action in ["read", "list"]
}

user_clearance_level := object.get(
    input.subject.attributes,
    "clearance_level",
    0
)
```

#### 基于时间

```rego
# 工作时间限制
is_business_hours if {
    input.environment.time
    current_time := input.environment.time
    current_time.weekday >= 1
    current_time.weekday <= 5
    current_time.hour >= 9
    current_time.hour < 18
}

allow if {
    is_business_hours
    "employee" in input.subject.roles
}
```

#### 基于地理位置

```rego
allow if {
    location_allowed
    input.resource.attributes.sensitivity == "high"
}

location_allowed if {
    input.environment.location in ["office", "vpn"]
}
```

#### 动态属性评分

```rego
# 多维度评分
allow if {
    dynamic_attribute_score >= required_score
}

dynamic_attribute_score := score if {
    score := role_score + 
             department_match_score + 
             clearance_score + 
             time_score + 
             location_score
}

required_score := 60

# 角色分数
role_score := 30 if {
    "admin" in input.subject.roles
} else := 20 if {
    "manager" in input.subject.roles
} else := 10
```

### 使用示例

```go
evaluator, err := opa.NewEvaluatorWithConfig(ctx, &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./pkg/security/opa/policies",
    },
})
if err != nil {
    log.Fatal(err)
}

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
    log.Fatal(err)
}

if result.Allowed {
    log.Printf("✅ 访问允许 (分数: %d/%d)", 
        result.Score, result.RequiredScore)
}
```

## 测试

### 测试文件结构

```rego
package rbac_test

import rego.v1
import data.rbac

# 测试成功场景
test_admin_has_all_permissions if {
    rbac.allow with input as {
        "subject": {"user": "alice", "roles": ["admin"]},
        "action": "delete",
        "resource": "users"
    }
}

# 测试失败场景
test_viewer_cannot_delete if {
    not rbac.allow with input as {
        "subject": {"user": "bob", "roles": ["viewer"]},
        "action": "delete",
        "resource": "documents"
    }
}

# 测试边界情况
test_empty_roles_denied if {
    not rbac.allow with input as {
        "subject": {"user": "charlie", "roles": []},
        "action": "read",
        "resource": "documents"
    }
}
```

### 运行测试

#### 使用 OPA CLI

```bash
# 运行所有测试
opa test pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego -v

# 运行特定测试
opa test -v -r test_admin_has_all_permissions \
  pkg/security/opa/policies/rbac*.rego

# 生成覆盖率报告
opa test --coverage pkg/security/opa/policies/rbac*.rego
```

#### 使用 Go 测试

```go
func TestRBACPolicy(t *testing.T) {
    ctx := context.Background()
    
    evaluator, err := opa.NewEvaluatorWithConfig(ctx, &opa.Config{
        Mode: opa.ModeEmbedded,
        EmbeddedConfig: &opa.EmbeddedConfig{
            PolicyDir: "./policies",
        },
    })
    require.NoError(t, err)
    defer evaluator.Close(ctx)
    
    tests := []struct {
        name     string
        input    *opa.RBACInput
        expected bool
    }{
        {
            name: "管理员可以删除用户",
            input: &opa.RBACInput{
                Subject:  &opa.Subject{User: "alice", Roles: []string{"admin"}},
                Action:   "delete",
                Resource: "users",
            },
            expected: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := evaluator.EvaluateRBAC(ctx, tt.input)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result.Allowed)
        })
    }
}
```

### 测试最佳实践

- **全面覆盖**: 测试所有主要规则、边界条件、否定场景
- **表驱动**: 使用表驱动测试处理多个场景
- **辅助函数**: 创建测试辅助函数简化测试代码
- **覆盖率目标**: > 90% 覆盖率

## 性能优化

### 1. 启用决策缓存

```go
config := &opa.Config{
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       10000,         // 最多缓存 10000 个决策
        TTL:           5 * time.Minute, // 缓存有效期
        EnableMetrics: true,          // 启用缓存指标
    },
}
```

### 2. 策略优化

#### 提前退出

```rego
# ✅ 快速路径优先
allow if {
    "admin" in input.subject.roles  # 快速检查
}

allow if {
    complex_evaluation  # 复杂评估
}
```

#### 避免不必要的迭代

```rego
# ✅ 使用集合操作
allow if {
    some role in {"admin", "editor"}
    role in input.subject.roles
}

# ❌ 遍历整个列表
allow if {
    some role in input.subject.roles
    role in ["admin", "editor", "viewer", ...]
}
```

#### 索引优化

```rego
# ✅ 使用对象作为索引 (O(1))
role_permissions := {
    "admin": {...},
    "editor": {...},
}

# ❌ 使用数组遍历 (O(n))
role_permissions := [
    {"role": "admin", "permissions": {...}},
]
```

### 3. 性能监控

```go
config := &opa.Config{
    MetricsConfig: &opa.MetricsConfig{
        Enabled:     true,
        Port:        9090,
        Path:        "/metrics",
        EnableCache: true,
    },
}
```

### 4. 生产环境配置

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       50000,
        TTL:           10 * time.Minute,
        EnableMetrics: true,
    },
    MetricsConfig: &opa.MetricsConfig{
        Enabled:     true,
        Port:        9090,
        EnableCache: true,
    },
    LogLevel: "info",
}
```

### 性能基准

| 场景 | 平均响应时间 | QPS | 缓存命中率 |
|------|-------------|-----|-----------|
| RBAC 简单检查 | < 1ms | > 10,000 | 95% |
| RBAC 复杂检查 | < 3ms | > 5,000 | 90% |
| ABAC 简单检查 | < 2ms | > 8,000 | 90% |
| ABAC 复杂评分 | < 5ms | > 3,000 | 85% |

## 常见问题

### Q: 为什么使用 OPA 而不是硬编码权限？

**A**: OPA 提供：
- 策略与代码解耦，可独立更新
- 集中管理所有策略
- 独立测试和审计
- 跨服务复用策略

### Q: RBAC 还是 ABAC？

**A**: 根据场景选择：

| 场景 | 推荐 |
|------|------|
| 组织结构明确、角色稳定 | RBAC |
| 需要细粒度、多条件控制 | ABAC |
| 简单权限管理 | RBAC |
| 动态、上下文相关控制 | ABAC |

也可以结合使用。

### Q: 嵌入式还是服务器模式？

**A**:
- **嵌入式**: 低延迟、简单部署（适合单体应用）
- **服务器**: 动态更新、跨服务共享（适合微服务）

### Q: 如何调试策略？

**A**:
1. 使用 OPA REPL 交互式测试
2. 添加 debug_info 规则输出中间结果
3. 启用详细日志（LogLevel: "debug"）

### Q: 策略评估太慢怎么办？

**A**:
1. 启用决策缓存
2. 优化策略规则（提前退出、避免不必要计算）
3. 使用索引优化（对象而非数组）
4. 使用 profiling 工具找出瓶颈

### Q: 如何在生产环境更新策略？

**A**:
- **嵌入式**: 调用 `evaluator.ReloadPolicies(ctx)`
- **服务器**: 通过 API 更新策略
- **Bundle**: 配置从远程加载策略包

### Q: 如何集成到现有系统？

**A**:
1. 识别授权检查点
2. 将授权逻辑转换为 Rego 策略
3. 逐步迁移（从非关键路径开始）
4. 双轨运行验证
5. 监控和调优

```go
// 在中间件中集成 OPA
func AuthorizationMiddleware(evaluator *opa.Evaluator) gin.HandlerFunc {
    return func(c *gin.Context) {
        input := &opa.RBACInput{
            Subject: &opa.Subject{
                User:  c.GetString("user"),
                Roles: c.GetStringSlice("roles"),
            },
            Action:   c.Request.Method,
            Resource: c.Request.URL.Path,
        }
        
        result, err := evaluator.EvaluateRBAC(c.Request.Context(), input)
        if err != nil || !result.Allowed {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        
        c.Next()
    }
}
```

## 相关资源

### 官方文档
- [OPA 官方文档](https://www.openpolicyagent.org/docs/latest/)
- [Rego 语言参考](https://www.openpolicyagent.org/docs/latest/policy-language/)

### 项目文档
- [OPA 包文档](../../../pkg/security/opa/README.md)
- [RBAC 策略指南](../../../opa-rbac-guide.md)
- [ABAC 策略指南](../../../opa-abac-guide.md)
- [OPA API 参考](../../api/security-opa.md)

### 示例代码
- [RBAC 策略实现](../../../pkg/security/opa/policies/rbac.rego)
- [ABAC 策略实现](../../../pkg/security/opa/policies/abac.rego)
- [行业示例](../../../pkg/security/opa/policies/examples/)

## 许可证

Copyright (c) 2024 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.

