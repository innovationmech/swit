# OPA 策略编写和集成指南

本指南介绍如何使用 Open Policy Agent (OPA) 编写和集成策略，包括 Rego 语言入门、策略编写最佳实践、RBAC 和 ABAC 策略示例、测试和性能优化。

## 目录

- [概述](#概述)
- [Rego 语言入门](#rego-语言入门)
- [策略编写最佳实践](#策略编写最佳实践)
- [RBAC 策略示例](#rbac-策略示例)
- [ABAC 策略示例](#abac-策略示例)
- [策略测试](#策略测试)
- [性能优化](#性能优化)
- [常见问题解答](#常见问题解答)

## 概述

Open Policy Agent (OPA) 是一个开源的通用策略引擎，它使用声明式语言 Rego 来定义策略。OPA 可以统一整个技术栈的策略执行，支持 API 授权、数据过滤、Kubernetes 准入控制等多种场景。

### OPA 的核心概念

#### 1. 策略即代码 (Policy as Code)

OPA 允许你将策略定义为代码，具有以下优势：

- **版本控制**: 策略可以像代码一样进行版本管理
- **测试**: 可以编写单元测试和集成测试
- **复用**: 策略可以被模块化和复用
- **审计**: 所有策略变更都有完整的审计追踪

#### 2. 解耦决策与执行

OPA 将策略决策与策略执行分离：

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│  应用程序   │───────>│     OPA     │───────>│   策略决策  │
│   (执行)    │  查询  │  (决策引擎)  │  结果  │  (allow/deny)│
└─────────────┘        └─────────────┘        └─────────────┘
                              │
                              │ 加载
                              ▼
                        ┌─────────────┐
                        │  Rego 策略  │
                        └─────────────┘
```

#### 3. 数据驱动决策

OPA 基于以下三个输入做出决策：

- **Input**: 请求的上下文信息（用户、资源、操作等）
- **Data**: 外部数据（用户列表、资源属性等）
- **Policy**: Rego 策略规则

### 为什么使用 OPA？

| 特性 | 传统方案 | OPA 方案 |
|------|---------|---------|
| 策略位置 | 分散在代码中 | 集中管理 |
| 策略变更 | 需要重新部署 | 动态加载 |
| 策略测试 | 难以测试 | 独立测试 |
| 策略复用 | 重复编写 | 模块化复用 |
| 审计追踪 | 难以追踪 | 完整审计 |

## Rego 语言入门

Rego 是 OPA 使用的声明式查询语言，专门用于编写策略。

### 基本语法

#### 1. 规则 (Rules)

规则是 Rego 的基本构建块：

```rego
package example

import rego.v1

# 简单规则：返回布尔值
allow if {
    input.user == "alice"
}

# 带条件的规则
allow if {
    input.user == "bob"
    input.action == "read"
}

# 多个规则（OR 关系）
allow if {
    input.user == "admin"
}

# 定义值的规则
user_name := input.user

# 生成集合
admins contains user if {
    some user in data.users
    user.role == "admin"
}
```

#### 2. 数据类型

Rego 支持以下数据类型：

```rego
# 字符串
name := "alice"

# 数字
count := 42
price := 19.99

# 布尔值
is_admin := true

# 数组
users := ["alice", "bob", "charlie"]

# 对象
user := {
    "name": "alice",
    "role": "admin",
    "age": 30
}

# 集合 (Set)
admin_set := {"alice", "bob"}
```

#### 3. 运算符

```rego
# 比较运算符
x == y  # 等于
x != y  # 不等于
x < y   # 小于
x <= y  # 小于等于
x > y   # 大于
x >= y  # 大于等于

# 逻辑运算符（隐式 AND）
rule if {
    condition1
    condition2  # AND
}

# 成员运算符
"alice" in ["alice", "bob"]  # true
"admin" in user.roles        # 检查数组成员

# 否定
not condition
```

#### 4. 内置函数

Rego 提供了丰富的内置函数：

```rego
# 字符串函数
startswith("hello world", "hello")  # true
contains("hello world", "world")     # true
lower("HELLO")                       # "hello"
upper("hello")                       # "HELLO"
split("a,b,c", ",")                  # ["a", "b", "c"]

# 数组函数
count([1, 2, 3])                    # 3
concat(",", ["a", "b", "c"])        # "a,b,c"

# 对象函数
object.get(obj, "key", "default")   # 获取对象值，带默认值
object.keys({"a": 1, "b": 2})       # ["a", "b"]

# 集合函数
count({"a", "b", "c"})              # 3

# 时间函数
time.now_ns()                       # 当前时间（纳秒）
time.parse_ns("2006-01-02", "2024-01-01")  # 解析时间

# 类型检查
is_string("hello")                  # true
is_number(42)                       # true
is_boolean(true)                    # true
is_array([1, 2, 3])                 # true
is_object({"a": 1})                 # true
```

#### 5. 迭代和量词

```rego
# some: 存在量词
allow if {
    some role in input.user.roles
    role == "admin"
}

# every: 全称量词
all_positive if {
    every item in [1, 2, 3] {
        item > 0
    }
}

# 迭代集合
user_names contains name if {
    some user in data.users
    name := user.name
}

# 多重迭代
pairs contains pair if {
    some x in [1, 2, 3]
    some y in ["a", "b", "c"]
    pair := {"x": x, "y": y}
}
```

### 策略结构

一个完整的 Rego 策略文件通常包含以下部分：

```rego
# 1. 包声明（必需）
package example.authz

# 2. 导入语句
import rego.v1
import data.users
import data.roles

# 3. 默认规则
default allow := false

# 4. 主决策规则
allow if {
    user_is_admin
}

allow if {
    user_has_permission
}

# 5. 辅助规则
user_is_admin if {
    "admin" in input.user.roles
}

user_has_permission if {
    some role in input.user.roles
    some permission in role_permissions[role]
    permission == input.action
}

# 6. 数据定义
role_permissions := {
    "admin": ["read", "write", "delete"],
    "editor": ["read", "write"],
    "viewer": ["read"],
}

# 7. 审计规则
audit_log := {
    "user": input.user.name,
    "action": input.action,
    "resource": input.resource,
    "allowed": allow,
    "timestamp": time.now_ns(),
}
```

### Rego 高级特性

#### 1. 理解 (Comprehensions)

```rego
# 数组理解
numbers := [x | x := [1, 2, 3, 4, 5][_]; x > 2]
# 结果: [3, 4, 5]

# 对象理解
admin_ages := {name: age |
    some user in data.users
    user.role == "admin"
    name := user.name
    age := user.age
}

# 集合理解
admin_names := {name |
    some user in data.users
    user.role == "admin"
    name := user.name
}
```

#### 2. 函数 (Functions)

```rego
# 定义函数
max_value(a, b) := a if {
    a >= b
} else := b

# 带默认值的函数
get_role(user) := role if {
    role := user.role
} else := "viewer"

# 使用函数
result := max_value(10, 20)  # 20
```

#### 3. 部分规则 (Partial Rules)

```rego
# 生成对象
user_permissions[user] := permissions if {
    some user in data.users
    permissions := user.permissions
}

# 结果是一个映射：{"alice": [...], "bob": [...]}
```

#### 4. With 关键字

`with` 关键字用于在测试或评估时覆盖输入或数据：

```rego
# 策略
allow if {
    input.user == "alice"
}

# 测试
test_allow_alice if {
    allow with input as {"user": "alice"}
}

test_deny_bob if {
    not allow with input as {"user": "bob"}
}
```

## 策略编写最佳实践

### 1. 组织策略文件

#### 目录结构

```
policies/
├── common/
│   ├── helpers.rego          # 通用辅助函数
│   └── constants.rego        # 常量定义
├── authz/
│   ├── rbac.rego             # RBAC 策略
│   ├── rbac_test.rego        # RBAC 测试
│   ├── abac.rego             # ABAC 策略
│   └── abac_test.rego        # ABAC 测试
├── examples/
│   ├── healthcare.rego       # 医疗行业示例
│   ├── financial.rego        # 金融行业示例
│   └── multi_tenant.rego     # 多租户示例
└── data/
    ├── users.json            # 用户数据
    └── roles.json            # 角色数据
```

#### 包命名规范

```rego
# 使用层级化的包名
package example.authz.rbac
package example.authz.abac
package example.api.validation
```

### 2. 默认拒绝原则

始终使用默认拒绝策略，这是安全最佳实践：

```rego
# ✅ 推荐：默认拒绝
default allow := false

allow if {
    # 明确的允许条件
}

# ❌ 避免：默认允许
default allow := true  # 不安全！

deny if {
    # 这种方式容易出错
}
```

### 3. 策略模块化

将复杂的策略分解为小的、可复用的规则：

```rego
# ✅ 推荐：模块化
allow if {
    user_authenticated
    user_authorized
    resource_accessible
}

user_authenticated if {
    # 认证逻辑
}

user_authorized if {
    # 授权逻辑
}

resource_accessible if {
    # 资源访问检查
}

# ❌ 避免：单体规则
allow if {
    # 所有逻辑都在一个规则中
    input.user.token != ""
    some role in input.user.roles
    role_permissions[role][input.action]
    input.resource.owner == input.user.name
    # ... 更多条件
}
```

### 4. 使用有意义的命名

```rego
# ✅ 推荐：清晰的命名
user_is_admin if {
    "admin" in input.user.roles
}

user_has_permission(action) if {
    some role in input.user.roles
    role_permissions[role][action]
}

# ❌ 避免：模糊的命名
check1 if {
    "admin" in input.user.roles
}

x(a) if {
    some r in input.user.roles
    y[r][a]
}
```

### 5. 添加文档注释

为策略添加清晰的文档：

```rego
# RBAC (Role-Based Access Control) 策略
#
# 此策略实现基于角色的访问控制，支持：
# - 角色权限映射
# - 角色资源映射
# - 超级管理员支持
# - 审计日志
#
# 输入格式：
#   {
#     "subject": {"user": "alice", "roles": ["editor"]},
#     "action": "read",
#     "resource": "documents"
#   }
#
# 返回：
#   allow: true/false
#
package rbac

import rego.v1

# 默认拒绝所有访问
default allow := false

# 主决策规则
# 允许访问当用户具有适当的角色权限
allow if {
    some role in user_roles
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}
```

### 6. 输入验证

始终验证输入数据的完整性：

```rego
# ✅ 推荐：验证输入
allow if {
    # 验证必需字段存在
    input.subject
    input.subject.user
    input.action
    input.resource
    
    # 进行实际的授权检查
    user_authorized
}

# ❌ 避免：不验证输入
allow if {
    # 直接使用可能不存在的字段
    input.subject.user == "admin"  # 如果 subject 不存在会报错
}
```

### 7. 错误处理

使用默认值和条件检查来处理缺失数据：

```rego
# 使用 object.get 提供默认值
clearance_level := object.get(
    input.subject.attributes,
    "clearance_level",
    0  # 默认值
)

# 检查字段是否存在
allow if {
    input.subject.attributes
    input.subject.attributes.clearance_level
    input.subject.attributes.clearance_level >= 3
}

# 或者使用 else 子句
user_role := input.subject.role if {
    input.subject.role
} else := "guest"
```

### 8. 性能考虑

#### 提前退出

```rego
# ✅ 推荐：快速路径优先
allow if {
    "admin" in input.subject.roles  # 快速检查
}

allow if {
    user_has_permission  # 复杂检查放后面
}

# ❌ 避免：复杂检查优先
allow if {
    complex_calculation_result >= threshold
}

allow if {
    "admin" in input.subject.roles
}
```

#### 避免不必要的计算

```rego
# ✅ 推荐：条件计算
allow if {
    user_authenticated
    # 只有当用户认证后才计算复杂的分数
    dynamic_score >= required_score
}

# ❌ 避免：总是计算
allow if {
    dynamic_score >= required_score  # 即使用户未认证也会计算
}
```

### 9. 测试驱动开发

为每个策略编写测试：

```rego
package rbac_test

import rego.v1
import data.rbac

# 测试管理员权限
test_admin_has_all_permissions if {
    rbac.allow with input as {
        "subject": {"user": "alice", "roles": ["admin"]},
        "action": "delete",
        "resource": "users"
    }
}

# 测试权限拒绝
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

### 10. 版本控制策略

```rego
# 在策略中包含版本信息
metadata := {
    "version": "1.0.0",
    "description": "RBAC policy for document management",
    "author": "security-team",
    "last_updated": "2024-01-15"
}
```

## RBAC 策略示例

基于角色的访问控制 (RBAC) 是最常用的访问控制模型。以下是完整的 RBAC 实现示例。

### 基础 RBAC 策略

```rego
package rbac

import rego.v1

# 默认拒绝
default allow := false

# 主决策规则
allow if {
    some role in user_roles
    role_permissions[role][input.action]
    role_resources[role][input.resource]
}

# 超级管理员拥有所有权限
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
        "list": true,
    },
    "editor": {
        "create": true,
        "read": true,
        "update": true,
        "list": true,
    },
    "viewer": {
        "read": true,
        "list": true,
    },
}

# 角色资源映射
role_resources := {
    "admin": {
        "users": true,
        "documents": true,
        "settings": true,
        "reports": true,
    },
    "editor": {
        "documents": true,
        "reports": true,
    },
    "viewer": {
        "documents": true,
        "reports": true,
    },
}
```

### 使用 RBAC 策略

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
        log.Println("✅ Access granted")
    } else {
        log.Println("❌ Access denied")
    }
}
```

### 高级 RBAC 特性

#### 1. 资源所有者权限

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

#### 2. 用户组角色继承

```rego
# 通过用户组继承角色
user_roles contains role if {
    some group in input.subject.groups
    some role in group_roles[group]
}

group_roles := {
    "engineering": ["editor", "contributor"],
    "management": ["admin"],
    "support": ["viewer"],
}
```

#### 3. 临时权限

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

### 完整示例

详见项目中的 RBAC 实现：
- 策略文件：`pkg/security/opa/policies/rbac.rego`
- 测试文件：`pkg/security/opa/policies/rbac_test.rego`
- 使用指南：`docs/opa-rbac-guide.md`

## ABAC 策略示例

基于属性的访问控制 (ABAC) 提供了比 RBAC 更细粒度的控制。

### 基础 ABAC 策略

```rego
package abac

import rego.v1

# 默认拒绝
default allow := false

# 主决策规则
allow if {
    subject_attributes_valid
    resource_attributes_valid
    action_attributes_valid
    environment_attributes_valid
}

# 超级管理员绕过所有检查
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
    is_string(input.resource)
}

resource_attributes_valid if {
    input.resource.type
    input.resource.id
}

# 操作属性验证
action_attributes_valid if {
    input.action in allowed_actions
}

allowed_actions := [
    "create", "read", "update", "delete", "list",
    "approve", "reject", "export", "share"
]

# 环境属性验证
environment_attributes_valid if {
    not input.environment
}

environment_attributes_valid if {
    input.environment
    ip_address_valid
    time_valid
    device_type_valid
}
```

### ABAC 访问控制规则

#### 1. 基于部门的访问控制

```rego
allow if {
    input.subject.attributes.department == input.resource.attributes.department
    input.action == "read"
}
```

#### 2. 基于安全级别的访问控制

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

resource_security_level := object.get(
    input.resource.attributes,
    "security_level",
    0
)
```

#### 3. 基于时间的访问控制

```rego
# 工作时间限制
is_business_hours if {
    input.environment.time
    current_time := input.environment.time
    # 周一到周五
    current_time.weekday >= 1
    current_time.weekday <= 5
    # 9:00-18:00
    current_time.hour >= 9
    current_time.hour < 18
}

allow if {
    is_business_hours
    "employee" in input.subject.roles
    input.action in ["read", "list"]
}
```

#### 4. 基于地理位置的访问控制

```rego
allow if {
    location_allowed
    input.resource.attributes.sensitivity == "high"
    input.action in ["read", "list"]
}

location_allowed if {
    input.environment.location in ["office", "vpn", "headquarters"]
}
```

#### 5. 动态属性评分

```rego
# 基于多维度属性的动态评分
allow if {
    dynamic_attribute_score >= required_score
}

dynamic_attribute_score := score if {
    score := role_score + 
             department_match_score + 
             clearance_score + 
             time_score + 
             location_score + 
             device_score
}

required_score := 60  # 默认要求 60 分

# 角色分数
role_score := 30 if {
    "admin" in input.subject.roles
} else := 20 if {
    "manager" in input.subject.roles
} else := 10 if {
    "employee" in input.subject.roles
} else := 0

# 部门匹配分数
department_match_score := 20 if {
    input.subject.attributes.department == input.resource.attributes.department
} else := 0

# 安全级别分数
clearance_score := 30 if {
    user_clearance_level >= resource_security_level
} else := 0
```

### 使用 ABAC 策略

```go
package main

import (
    "context"
    "log"
    
    "github.com/innovationmech/swit/pkg/security/opa"
)

func main() {
    ctx := context.Background()
    
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
    
    // ABAC 评估
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
        log.Printf("✅ Access granted (Score: %d/%d)", 
            result.Score, result.RequiredScore)
    } else {
        log.Println("❌ Access denied")
    }
}
```

### 完整示例

详见项目中的 ABAC 实现：
- 策略文件：`pkg/security/opa/policies/abac.rego`
- 测试文件：`pkg/security/opa/policies/abac_test.rego`
- 使用指南：`docs/opa-abac-guide.md`
- 行业示例：`pkg/security/opa/policies/examples/`

## 策略测试

测试是策略开发的重要组成部分，确保策略按预期工作。

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

# 测试辅助规则
test_user_roles if {
    rbac.user_roles == {"editor", "viewer"} with input as {
        "subject": {"user": "dave", "roles": ["editor", "viewer"]}
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
opa test pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego \
  -v -r test_admin_has_all_permissions

# 生成覆盖率报告
opa test --coverage \
  pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego
```

#### 使用 Go 测试

```go
package opa_test

import (
    "context"
    "testing"
    
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

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
            name: "admin can delete users",
            input: &opa.RBACInput{
                Subject:  &opa.Subject{User: "alice", Roles: []string{"admin"}},
                Action:   "delete",
                Resource: "users",
            },
            expected: true,
        },
        {
            name: "viewer cannot delete documents",
            input: &opa.RBACInput{
                Subject:  &opa.Subject{User: "bob", Roles: []string{"viewer"}},
                Action:   "delete",
                Resource: "documents",
            },
            expected: false,
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

#### 1. 全面的测试覆盖

```rego
# 测试所有主要规则
test_rule_1 if { ... }
test_rule_2 if { ... }

# 测试边界条件
test_empty_input if { ... }
test_missing_fields if { ... }
test_invalid_data if { ... }

# 测试否定场景
test_deny_unauthorized if { ... }
test_deny_insufficient_permissions if { ... }
```

#### 2. 使用表驱动测试

```rego
test_role_permissions if {
    test_cases := [
        {"role": "admin", "action": "delete", "expected": true},
        {"role": "editor", "action": "delete", "expected": false},
        {"role": "viewer", "action": "read", "expected": true},
        {"role": "viewer", "action": "update", "expected": false},
    ]
    
    every test_case in test_cases {
        result := rbac.allow with input as {
            "subject": {"user": "test", "roles": [test_case.role]},
            "action": test_case.action,
            "resource": "documents"
        }
        result == test_case.expected
    }
}
```

#### 3. 测试辅助函数

```rego
# 创建测试辅助函数
make_input(user, roles, action, resource) := {
    "subject": {"user": user, "roles": roles},
    "action": action,
    "resource": resource
}

# 在测试中使用
test_admin_access if {
    rbac.allow with input as make_input("alice", ["admin"], "delete", "users")
}
```

#### 4. 性能基准测试

```bash
# 运行基准测试
opa test --bench \
  pkg/security/opa/policies/rbac.rego \
  pkg/security/opa/policies/rbac_test.rego

# 输出示例：
# PASS: 15/15
# Benchmark results:
# test_admin_access: 0.05ms
# test_viewer_access: 0.03ms
```

### 测试覆盖率目标

- **目标**: > 90% 覆盖率
- 测试所有主要决策规则
- 测试所有辅助规则
- 测试边界条件和错误情况
- 测试性能关键路径

## 性能优化

OPA 策略的性能对于生产环境至关重要。以下是一些优化技巧。

### 1. 决策缓存

启用决策缓存可以显著提高性能：

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       10000,         // 最多缓存 10000 个决策
        TTL:           5 * time.Minute, // 缓存有效期 5 分钟
        EnableMetrics: true,          // 启用缓存指标
    },
}
```

### 2. 策略优化

#### 提前退出

```rego
# ✅ 推荐：快速路径优先
allow if {
    "admin" in input.subject.roles  # 快速检查
}

allow if {
    complex_evaluation  # 复杂评估放后面
}

# ❌ 避免：复杂评估优先
allow if {
    score := calculate_score  # 总是计算
    score >= 60
}
```

#### 避免不必要的迭代

```rego
# ✅ 推荐：使用集合操作
allow if {
    some role in {"admin", "editor"}
    role in input.subject.roles
}

# ❌ 避免：遍历整个列表
allow if {
    some role in input.subject.roles
    role in ["admin", "editor", "viewer", "contributor", ...]
}
```

#### 索引优化

```rego
# ✅ 推荐：使用对象作为索引
role_permissions := {
    "admin": {...},
    "editor": {...},
}

allow if {
    some role in input.subject.roles
    role_permissions[role][input.action]  # O(1) 查找
}

# ❌ 避免：使用数组遍历
role_permissions := [
    {"role": "admin", "permissions": {...}},
    {"role": "editor", "permissions": {...}},
]

allow if {
    some perm in role_permissions  # O(n) 遍历
    perm.role in input.subject.roles
    perm.permissions[input.action]
}
```

### 3. 数据优化

#### 预加载静态数据

```rego
# 将静态数据定义在策略中
role_permissions := {
    "admin": {"read": true, "write": true, "delete": true},
    "editor": {"read": true, "write": true},
}

# 或者从外部文件加载
import data.permissions.role_permissions
```

#### 限制数据大小

```rego
# ✅ 推荐：只加载需要的数据
user_roles := input.subject.roles

# ❌ 避免：加载大量不必要的数据
all_users := data.users  # 可能包含数千个用户
```

### 4. 性能监控

#### 启用性能指标

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

#### 使用 Profiling

```bash
# 分析策略性能
opa eval -d policies/rbac.rego \
  --profile \
  --input input.json \
  "data.rbac.allow"

# 输出包含：
# - 每个规则的执行时间
# - 调用次数
# - 总执行时间
```

### 5. 基准测试

```bash
# 运行基准测试
opa test --bench policies/

# 比较不同版本的性能
opa test --bench policies/ > benchmark-v1.txt
# 修改策略后
opa test --bench policies/ > benchmark-v2.txt
diff benchmark-v1.txt benchmark-v2.txt
```

### 6. 生产环境配置

```go
// 生产环境推荐配置
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    EmbeddedConfig: &opa.EmbeddedConfig{
        PolicyDir: "./policies",
    },
    // 启用缓存
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       50000,  // 根据实际需求调整
        TTL:           10 * time.Minute,
        EnableMetrics: true,
    },
    // 性能监控
    MetricsConfig: &opa.MetricsConfig{
        Enabled:     true,
        Port:        9090,
        EnableCache: true,
    },
    // 日志级别
    LogLevel: "info",  // 生产环境使用 info 或 warn
}
```

### 性能基准

基于项目中的测试，以下是一些性能参考：

| 场景 | 平均响应时间 | QPS | 缓存命中率 |
|------|-------------|-----|-----------|
| RBAC 简单检查 | < 1ms | > 10,000 | 95% |
| RBAC 复杂检查 | < 3ms | > 5,000 | 90% |
| ABAC 简单检查 | < 2ms | > 8,000 | 90% |
| ABAC 复杂评分 | < 5ms | > 3,000 | 85% |

## 常见问题解答

### 1. OPA vs 硬编码权限检查

**问**: 为什么要使用 OPA 而不是在代码中直接检查权限？

**答**: OPA 提供了以下优势：
- **策略与代码解耦**: 可以在不修改代码的情况下更新策略
- **集中管理**: 所有策略在一个地方定义和管理
- **可测试性**: 策略可以独立测试
- **可审计性**: 策略变更有完整的审计追踪
- **复用性**: 策略可以在多个服务间共享

### 2. RBAC vs ABAC

**问**: 应该使用 RBAC 还是 ABAC？

**答**: 选择取决于你的需求：

| 场景 | 推荐 |
|------|------|
| 组织结构明确，角色稳定 | RBAC |
| 需要基于多种条件的细粒度控制 | ABAC |
| 简单的权限管理 | RBAC |
| 动态的、上下文相关的访问控制 | ABAC |
| 快速实现 | RBAC |
| 灵活性和可扩展性 | ABAC |

你也可以结合使用：先用 RBAC 进行粗粒度控制，再用 ABAC 进行细粒度控制。

### 3. 嵌入式 vs 服务器模式

**问**: 应该使用嵌入式模式还是独立的 OPA 服务器？

**答**: 
- **嵌入式模式** (推荐用于单体应用):
  - 优点：低延迟、简单部署、无网络开销
  - 缺点：策略更新需要重启应用
  
- **服务器模式** (推荐用于微服务):
  - 优点：策略动态更新、跨服务共享、集中管理
  - 缺点：网络延迟、额外的部署复杂度

### 4. 策略更新

**问**: 如何在生产环境中更新策略？

**答**: 有几种方法：

1. **嵌入式模式**：
```go
// 重新加载策略
evaluator.ReloadPolicies(ctx)
```

2. **服务器模式**：
```bash
# 通过 API 更新策略
curl -X PUT http://opa-server:8181/v1/policies/rbac \
  --data-binary @rbac.rego
```

3. **Bundle 模式**：
```yaml
# 配置 OPA 从远程位置加载策略包
services:
  - name: bundle-server
    url: https://policies.example.com

bundles:
  - name: authz
    service: bundle-server
    resource: bundles/authz.tar.gz
```

### 5. 性能优化

**问**: OPA 策略评估太慢怎么办？

**答**: 尝试以下优化：
1. 启用决策缓存
2. 优化策略规则（提前退出、避免不必要的计算）
3. 使用索引优化（对象而非数组）
4. 减少数据加载量
5. 使用性能分析工具找出瓶颈

参见 [性能优化](#性能优化) 章节。

### 6. 调试策略

**问**: 如何调试 OPA 策略？

**答**: 
1. **使用 OPA REPL**:
```bash
opa run policies/rbac.rego

# 在 REPL 中测试
> import data.rbac
> rbac.allow with input as {"subject": {...}, "action": "read"}
```

2. **添加调试输出**:
```rego
debug_info := {
    "user_roles": user_roles,
    "permissions": user_permissions,
    "decision": allow,
}
```

3. **启用详细日志**:
```go
config := &opa.Config{
    LogLevel: "debug",
}
```

### 7. 错误处理

**问**: 如何处理策略评估错误？

**答**: 
```go
result, err := evaluator.EvaluateRBAC(ctx, input)
if err != nil {
    // 记录错误
    log.Error("Policy evaluation failed", zap.Error(err))
    
    // 采取回退策略：默认拒绝
    return false, err
}

if !result.Allowed {
    // 记录拒绝原因
    log.Info("Access denied",
        zap.String("user", input.Subject.User),
        zap.String("action", input.Action))
}

return result.Allowed, nil
```

### 8. 测试覆盖率

**问**: 如何确保策略有足够的测试覆盖率？

**答**: 
```bash
# 生成覆盖率报告
opa test --coverage policies/

# 目标：> 90% 覆盖率
# 确保测试：
# - 所有主要决策规则
# - 所有辅助规则
# - 边界条件
# - 错误情况
```

### 9. 策略版本控制

**问**: 如何管理策略的版本？

**答**: 
1. 将策略文件纳入 Git 版本控制
2. 在策略中包含版本元数据
3. 使用 Git 标签标记策略版本
4. 在 PR 中审查策略变更
5. 使用 CI/CD 自动测试策略

```bash
git add policies/rbac.rego
git commit -m "feat(rbac): add new role for auditors"
git tag -a policy-v1.2.0 -m "RBAC Policy v1.2.0"
```

### 10. 集成到现有系统

**问**: 如何将 OPA 集成到现有系统？

**答**: 
1. **识别授权点**: 找出代码中需要授权检查的地方
2. **定义策略**: 将现有的授权逻辑转换为 Rego 策略
3. **逐步迁移**: 先在非关键路径上使用 OPA，逐步扩展
4. **双轨运行**: 同时运行旧的和新的授权逻辑，比对结果
5. **监控和调优**: 监控性能和正确性，及时调整

```go
// 示例：在 HTTP 中间件中集成 OPA
func AuthorizationMiddleware(evaluator *opa.Evaluator) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 构建 OPA 输入
        input := &opa.RBACInput{
            Subject: &opa.Subject{
                User:  c.GetString("user"),
                Roles: c.GetStringSlice("roles"),
            },
            Action:   c.Request.Method,
            Resource: c.Request.URL.Path,
        }
        
        // 评估策略
        result, err := evaluator.EvaluateRBAC(c.Request.Context(), input)
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

## 相关资源

### 官方文档
- [OPA 官方文档](https://www.openpolicyagent.org/docs/latest/)
- [Rego 语言参考](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA 策略测试](https://www.openpolicyagent.org/docs/latest/policy-testing/)
- [OPA 性能优化](https://www.openpolicyagent.org/docs/latest/policy-performance/)

### 项目文档
- [pkg/security/opa/README.md](../pkg/security/opa/README.md) - OPA 包文档
- [docs/opa-rbac-guide.md](./opa-rbac-guide.md) - RBAC 策略使用指南
- [docs/opa-abac-guide.md](./opa-abac-guide.md) - ABAC 策略使用指南
- [docs/api/security-opa.md](./api/security-opa.md) - OPA API 参考

### 示例代码
- [pkg/security/opa/policies/rbac.rego](../pkg/security/opa/policies/rbac.rego) - RBAC 策略实现
- [pkg/security/opa/policies/abac.rego](../pkg/security/opa/policies/abac.rego) - ABAC 策略实现
- [pkg/security/opa/policies/examples/](../pkg/security/opa/policies/examples/) - 行业示例

### 相关 Issues
- [Epic #776: OPA 策略引擎集成](https://github.com/innovationmech/swit/issues/776)
- [Task #799: RBAC 策略模板](https://github.com/innovationmech/swit/issues/799)
- [Task #800: ABAC 策略模板](https://github.com/innovationmech/swit/issues/800)
- [Task #815: API 文档](https://github.com/innovationmech/swit/issues/815)

## 许可证

Copyright (c) 2024 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.

