# OPA ABAC 策略使用指南

本文档介绍如何使用 OPA (Open Policy Agent) ABAC (基于属性的访问控制) 策略模板来实现细粒度的权限管理。

## 目录

- [概述](#概述)
- [策略架构](#策略架构)
- [快速开始](#快速开始)
- [属性类型](#属性类型)
- [访问控制规则](#访问控制规则)
- [动态属性评分](#动态属性评分)
- [实际应用场景](#实际应用场景)
- [测试策略](#测试策略)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)

## 概述

ABAC (Attribute-Based Access Control) 是一种高度灵活的访问控制模型，它基于主体、资源、操作和环境的属性来做出访问决策。与传统的 RBAC 相比，ABAC 提供了更细粒度的控制能力。

### ABAC 的优势

- ✅ **细粒度控制** - 基于多种属性组合做出决策
- ✅ **动态评估** - 根据实时环境条件调整访问权限
- ✅ **可扩展性** - 无需为每种场景创建新角色
- ✅ **合规性** - 更容易实现复杂的合规要求
- ✅ **上下文感知** - 考虑时间、地点、设备等环境因素

### ABAC vs RBAC

| 特性 | RBAC | ABAC |
|------|------|------|
| 决策基础 | 角色 | 多种属性 |
| 灵活性 | 中等 | 高 |
| 复杂度 | 低 | 中到高 |
| 维护成本 | 角色数量增长时增加 | 相对稳定 |
| 适用场景 | 组织结构明确 | 动态、复杂的访问需求 |

## 策略架构

### 核心决策流程

```
┌─────────────────────────────┐
│  请求输入 (Input)           │
│  ┌───────────────────────┐  │
│  │ Subject (主体)        │  │
│  │ - user, roles         │  │
│  │ - attributes          │  │
│  ├───────────────────────┤  │
│  │ Action (操作)         │  │
│  │ - create, read, etc.  │  │
│  ├───────────────────────┤  │
│  │ Resource (资源)       │  │
│  │ - type, id            │  │
│  │ - owner, attributes   │  │
│  ├───────────────────────┤  │
│  │ Environment (环境)    │  │
│  │ - time, location      │  │
│  │ - IP, device          │  │
│  └───────────────────────┘  │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│  ABAC 策略评估              │
│  ┌───────────────────────┐  │
│  │ 1. 主体属性检查       │  │
│  │ 2. 资源属性检查       │  │
│  │ 3. 操作验证           │  │
│  │ 4. 环境约束检查       │  │
│  │ 5. 动态属性评分       │  │
│  └───────────────────────┘  │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│  决策结果 + 审计日志        │
│  allow: true/false          │
│  audit_log: {...}           │
└─────────────────────────────┘
```

### 输入数据结构

策略期望以下输入结构：

```json
{
  "subject": {
    "user": "alice",
    "roles": ["employee", "manager"],
    "attributes": {
      "department": "engineering",
      "clearance_level": 3,
      "certifications": ["security", "compliance"]
    }
  },
  "action": "read",
  "resource": {
    "type": "document",
    "id": "doc-123",
    "owner": "bob",
    "attributes": {
      "department": "engineering",
      "security_level": 2,
      "sensitivity": "high",
      "classification": "internal"
    }
  },
  "environment": {
    "time": {
      "weekday": 3,
      "hour": 14
    },
    "location": "office",
    "ip_address": "10.0.1.100",
    "device_type": "laptop",
    "device_trusted": true
  }
}
```

## 快速开始

### 1. 加载 ABAC 策略

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
        DefaultDecisionPath: "abac/allow",
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
            "roles": []string{"employee"},
            "attributes": map[string]interface{}{
                "department": "engineering",
                "clearance_level": 3,
            },
        },
        "action": "read",
        "resource": map[string]interface{}{
            "type": "document",
            "id":   "doc-123",
            "attributes": map[string]interface{}{
                "department":     "engineering",
                "security_level": 2,
            },
        },
        "environment": map[string]interface{}{
            "time": map[string]interface{}{
                "weekday": 3,
                "hour":    14,
            },
            "location":    "office",
            "ip_address":  "10.0.1.100",
            "device_type": "laptop",
        },
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
        log.Println("Access granted")
        log.Printf("Score: %d/%d", result.Score, result.RequiredScore)
    }
}
```

## 属性类型

### 主体属性 (Subject Attributes)

主体属性描述请求访问的用户或服务：

| 属性 | 类型 | 描述 | 示例 |
|------|------|------|------|
| `user` | string | 用户标识符 | "alice" |
| `roles` | []string | 角色列表 | ["employee", "manager"] |
| `department` | string | 所属部门 | "engineering" |
| `clearance_level` | int | 安全级别 | 0-4 (0=无, 4=绝密) |
| `certifications` | []string | 认证资格 | ["security", "compliance"] |
| `groups` | []string | 用户组 | ["engineering", "leadership"] |

### 资源属性 (Resource Attributes)

资源属性描述被访问的对象：

| 属性 | 类型 | 描述 | 示例 |
|------|------|------|------|
| `type` | string | 资源类型 | "document", "project" |
| `id` | string | 资源标识符 | "doc-123" |
| `owner` | string | 资源所有者 | "bob" |
| `department` | string | 所属部门 | "engineering" |
| `security_level` | int | 安全级别 | 0-4 |
| `sensitivity` | string | 敏感度 | "low", "medium", "high", "critical" |
| `classification` | string | 数据分类 | "public", "internal", "confidential" |
| `members` | []string | 成员列表 | ["alice", "bob"] |

### 环境属性 (Environment Attributes)

环境属性描述访问请求的上下文：

| 属性 | 类型 | 描述 | 示例 |
|------|------|------|------|
| `time.weekday` | int | 星期几 | 1-7 (1=周一, 7=周日) |
| `time.hour` | int | 小时 | 0-23 |
| `time_window.start` | int64 | 时间窗口开始 | 纳秒时间戳 |
| `time_window.end` | int64 | 时间窗口结束 | 纳秒时间戳 |
| `location` | string | 地理位置 | "office", "home", "vpn" |
| `ip_address` | string | IP 地址 | "10.0.1.100" |
| `device_type` | string | 设备类型 | "desktop", "laptop", "mobile" |
| `device_trusted` | bool | 设备是否受信任 | true, false |

## 访问控制规则

### 1. 基于资源所有者的访问控制

资源所有者对自己的资源拥有完全控制权：

```rego
# 策略定义（已在 abac.rego 中）
allow if {
    input.resource.owner == input.subject.user
    input.action in ["read", "update", "delete"]
}
```

使用示例：

```go
// Alice 可以访问自己的文档
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{},
    },
    "action": "update",
    "resource": map[string]interface{}{
        "type":  "document",
        "id":    "doc-123",
        "owner": "alice",  // 所有者是 alice
    },
})
// result.Allowed == true
```

### 2. 基于部门的访问控制

同一部门的用户可以访问部门资源：

```rego
# 策略定义
allow if {
    input.subject.attributes.department == input.resource.attributes.department
    input.action == "read"
}
```

使用示例：

```go
// 工程部门的 Alice 可以读取工程部门的文档
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
        "attributes": map[string]interface{}{
            "department": "engineering",
        },
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "department": "engineering",
        },
    },
})
// result.Allowed == true
```

### 3. 基于安全级别的访问控制

用户的安全级别必须高于或等于资源的安全级别：

```rego
# 安全级别定义
# 0 = 无, 1 = 公开, 2 = 内部, 3 = 机密, 4 = 绝密

allow if {
    user_clearance_level >= resource_security_level
    input.action in ["read", "list"]
}
```

使用示例：

```go
// Alice (clearance level 3) 可以访问 level 2 的文档
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
        "attributes": map[string]interface{}{
            "clearance_level": 3,  // Alice 的安全级别
        },
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "security_level": 2,  // 文档的安全级别
        },
    },
})
// result.Allowed == true

// Bob (clearance level 1) 不能访问 level 3 的文档
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "bob",
        "roles": []string{"employee"},
        "attributes": map[string]interface{}{
            "clearance_level": 1,
        },
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-456",
        "attributes": map[string]interface{}{
            "security_level": 3,
        },
    },
})
// result.Allowed == false
```

### 4. 基于时间的访问控制

#### 工作时间限制

```rego
# 策略定义
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

使用示例：

```go
// 周三下午 2 点可以访问
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "read",
    "resource": "documents",
    "environment": map[string]interface{}{
        "time": map[string]interface{}{
            "weekday": 3,   // 周三
            "hour":    14,  // 14:00
        },
    },
})
// result.Allowed == true

// 周六下午 2 点不能访问
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "read",
    "resource": "documents",
    "environment": map[string]interface{}{
        "time": map[string]interface{}{
            "weekday": 6,   // 周六
            "hour":    14,
        },
    },
})
// result.Allowed == false
```

#### 时间窗口限制

```go
import "time"

// 授予 24 小时的临时访问权限
expiresAt := time.Now().Add(24 * time.Hour).UnixNano()

result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "read",
    "resource": "sensitive-data",
    "environment": map[string]interface{}{
        "time_window": map[string]interface{}{
            "start": time.Now().UnixNano(),
            "end":   expiresAt,
        },
    },
})
```

### 5. 基于地理位置的访问控制

```rego
# 策略定义
allow if {
    location_allowed
    is_object(input.resource)
    input.resource.attributes.sensitivity == "high"
    input.action in ["read", "list"]
}

location_allowed if {
    input.environment.location in ["office", "vpn", "headquarters"]
}
```

使用示例：

```go
// 从办公室访问敏感资源
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "sensitivity": "high",
        },
    },
    "environment": map[string]interface{}{
        "location": "office",
    },
})
// result.Allowed == true

// 从公共 Wi-Fi 无法访问
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "sensitivity": "high",
        },
    },
    "environment": map[string]interface{}{
        "location": "public-wifi",
    },
})
// result.Allowed == false
```

### 6. 基于 IP 地址的访问控制

策略支持以下 IP 段：
- `10.x.x.x` - 内网 A 类
- `192.168.x.x` - 内网 C 类
- `172.16-31.x.x` - 内网 B 类
- 自定义公司 IP 段

```go
// 从内网访问
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "delete",
    "resource": "documents",
    "environment": map[string]interface{}{
        "ip_address": "10.0.1.100",
    },
})
// result.Allowed == true

// 从外网无法执行删除操作
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "delete",
    "resource": "documents",
    "environment": map[string]interface{}{
        "ip_address": "1.2.3.4",
    },
})
// result.Allowed == false
```

### 7. 基于设备类型的访问控制

```go
// 从受信任设备（desktop/laptop）执行删除操作
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "delete",
    "resource": "documents",
    "environment": map[string]interface{}{
        "device_type": "desktop",
    },
})
// result.Allowed == true

// 从移动设备无法执行删除操作
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "delete",
    "resource": "documents",
    "environment": map[string]interface{}{
        "device_type": "mobile",
    },
})
// result.Allowed == false
```

### 8. 基于数据分类的访问控制

支持三种数据分类级别：
- **public** - 任何人都可以访问
- **internal** - 只有员工可以访问
- **confidential** - 只有授权用户可以访问

```go
// 公开数据
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "guest",
        "roles": []string{"guest"},
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "classification": "public",
        },
    },
})
// result.Allowed == true

// 内部数据需要员工身份
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-456",
        "attributes": map[string]interface{}{
            "classification": "internal",
        },
    },
})
// result.Allowed == true

// 机密数据需要特殊授权
result, _ = client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
        "attributes": map[string]interface{}{
            "access_levels": []string{"authorized"},
        },
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-789",
        "attributes": map[string]interface{}{
            "classification": "confidential",
        },
    },
})
// result.Allowed == true
```

## 动态属性评分

当用户没有明确的角色权限时，可以通过累积分数来获得访问权限。系统会根据多个维度的属性计算总分，达到所需分数即可获得访问权限。

### 评分维度

| 维度 | 最高分 | 描述 |
|------|--------|------|
| 角色分数 | 30 | admin=30, manager=20, employee=10 |
| 部门匹配 | 20 | 用户和资源在同一部门 |
| 安全级别 | 30 | 用户安全级别 >= 资源安全级别 |
| 时间分数 | 10 | 工作时间=10, 时间窗口内=5 |
| 地理位置 | 10 | 公司网络=10, 允许位置=5 |
| 设备类型 | 10 | 受信任设备=10, 允许设备=5 |
| **总分** | **110** | 默认要求 60 分 |

### 评分示例

```go
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"manager"},  // +20分
        "attributes": map[string]interface{}{
            "department":      "engineering",
            "clearance_level": 3,
        },
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "department":     "engineering",  // 部门匹配 +20分
            "security_level": 2,              // 安全级别符合 +30分
        },
    },
    "environment": map[string]interface{}{
        "time": map[string]interface{}{
            "weekday": 3,   // 周三
            "hour":    14,  // 14:00，工作时间 +10分
        },
        "location":    "office",     // 办公室 +10分
        "ip_address":  "10.0.1.100", // 内网
        "device_type": "desktop",    // 受信任设备 +10分
    },
})
// 总分 = 20 + 20 + 30 + 10 + 10 + 10 = 100 分
// 默认要求 60 分，所以允许访问
// result.Allowed == true
```

### 自定义所需分数

可以根据资源的敏感性配置不同的所需分数：

```go
// 敏感资源需要更高的分数
result, _ := client.Evaluate(ctx, "", map[string]interface{}{
    "subject": map[string]interface{}{
        "user":  "alice",
        "roles": []string{"employee"},
        "attributes": map[string]interface{}{
            "clearance_level": 2,
        },
    },
    "action": "read",
    "resource": map[string]interface{}{
        "type": "document",
        "id":   "doc-123",
        "attributes": map[string]interface{}{
            "security_level":  3,
            "required_score":  80,  // 自定义所需分数
        },
    },
})
```

## 实际应用场景

本节提供了三个真实世界的 ABAC 策略示例，涵盖医疗保健、金融和多租户 SaaS 应用场景。

### 1. 医疗保健行业 (Healthcare)

详见 `pkg/security/opa/policies/examples/healthcare_abac.rego`

**关键特性**：
- 基于医患关系的访问控制
- 急诊情况下的临时访问
- 患者同意管理
- 去标识化数据用于研究
- 时间和位置限制
- 完整的审计追踪

**应用场景**：
```go
// 医生访问自己负责的患者记录
result, _ := evaluator.EvaluateABAC(ctx, &opa.ABACInput{
    Subject: &opa.Subject{
        User:  "dr-smith",
        Roles: []string{"doctor"},
    },
    Action: "read",
    Resource: &opa.Resource{
        Type: "patient_record",
        ID:   "patient-123",
        Attributes: map[string]interface{}{
            "attending_physicians": []string{"dr-smith", "dr-jones"},
        },
    },
})
```

### 2. 金融行业 (Financial)

详见 `pkg/security/opa/policies/examples/financial_abac.rego`

**关键特性**：
- 基于交易金额的分级授权
- 四眼原则（Maker-Checker）
- 基于风险评分的控制
- KYC/AML 合规检查
- 职责分离
- 时间窗口限制

**应用场景**：
```go
// 大额交易需要高级管理层批准
result, _ := evaluator.EvaluateABAC(ctx, &opa.ABACInput{
    Subject: &opa.Subject{
        User:  "senior-manager",
        Roles: []string{"senior_management"},
    },
    Action: "approve",
    Resource: &opa.Resource{
        Type: "transaction",
        ID:   "txn-123",
        Attributes: map[string]interface{}{
            "amount":     1500000,  // $1.5M
            "risk_score": 45,
            "created_by": "teller-01",  // 不能自己批准
        },
    },
    Environment: &opa.Environment{
        Time: map[string]interface{}{
            "weekday": 3,
            "hour":    14,  // 工作时间
        },
    },
})
```

### 3. 多租户 SaaS 应用 (Multi-Tenant SaaS)

详见 `pkg/security/opa/policies/examples/multi_tenant_abac.rego`

**关键特性**：
- 租户数据隔离
- 跨租户共享
- 基于订阅计划的功能限制
- API 配额管理
- 工作空间访问控制
- 数据驻留合规性

**应用场景**：
```go
// 租户用户访问自己租户的数据
result, _ := evaluator.EvaluateABAC(ctx, &opa.ABACInput{
    Subject: &opa.Subject{
        User:     "alice",
        TenantID: "tenant-123",
        Roles:    []string{"admin"},
        Attributes: map[string]interface{}{
            "subscription_plan":  "professional",
            "api_calls_today":   5000,
        },
    },
    Action: "update",
    Resource: &opa.Resource{
        Type:     "document",
        ID:       "doc-123",
        TenantID: "tenant-123",  // 同一租户
    },
})
```

## 测试策略

### 运行测试

使用 OPA CLI 运行测试：

```bash
# 运行所有 ABAC 测试
opa test pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego -v

# 运行特定测试
opa test pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego \
  -v -r test_sufficient_clearance_level

# 生成覆盖率报告
opa test --coverage \
  pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego
```

### 测试示例场景

#### 示例 1: 测试安全级别

```bash
cd /Users/liyanjie/VSCodeProjects/swit
opa test pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego \
  -v -r test_sufficient_clearance_level
```

#### 示例 2: 测试时间约束

```bash
opa test pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego \
  -v -r test_business_hours_access
```

#### 示例 3: 测试动态评分

```bash
opa test pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego \
  -v -r test_high_score_grants_access
```

### 编写自定义测试

```rego
package abac_test

import rego.v1
import data.abac

# 测试自定义场景
test_custom_scenario if {
    abac.allow with input as {
        "subject": {
            "user": "custom-user",
            "roles": ["custom-role"],
            "attributes": {
                "custom-attr": "value",
            },
        },
        "action": "custom-action",
        "resource": {
            "type": "custom-resource",
            "id": "res-123",
        },
    }
}
```

### 性能基准测试

```bash
# 运行性能基准测试
opa test --bench pkg/security/opa/policies/abac.rego \
  pkg/security/opa/policies/abac_test.rego
```

## 最佳实践

### 1. 属性设计原则

**DO ✅**：
- 使用标准化的属性名称（如 `clearance_level` 而不是 `cl`）
- 属性值应该是明确且可验证的
- 使用枚举值限制可能的选项
- 为属性提供默认值

**DON'T ❌**：
- 避免在属性中存储敏感信息
- 不要使用过于复杂的嵌套结构
- 避免属性名称的歧义

### 2. 策略组织

将策略按功能模块组织：

```
policies/
├── abac.rego              # 基础 ABAC 策略
├── abac_test.rego         # 测试文件
└── examples/
    ├── healthcare_abac.rego
    ├── financial_abac.rego
    └── multi_tenant_abac.rego
```

### 3. 性能优化

#### 使用决策缓存

```go
config := &opa.Config{
    Mode: opa.ModeEmbedded,
    CacheConfig: &opa.CacheConfig{
        Enabled:       true,
        MaxSize:       10000,
        TTL:           5 * time.Minute,
        EnableMetrics: true,
    },
}
```

#### 优化策略规则

```rego
# 好的做法：提前退出
allow if {
    "admin" in input.subject.roles  # 快速检查
}

allow if {
    subject_attributes_valid
    resource_attributes_valid
    action_attributes_valid
    environment_attributes_valid  # 复杂检查放在最后
}

# 避免：过多的嵌套和计算
```

### 4. 安全考虑

#### 输入验证

始终验证输入数据的完整性和有效性：

```go
func ValidateABACInput(input *ABACInput) error {
    if input.Subject == nil || input.Subject.User == "" {
        return errors.New("subject is required")
    }
    if input.Action == "" {
        return errors.New("action is required")
    }
    if input.Resource == nil {
        return errors.New("resource is required")
    }
    return nil
}
```

#### 默认拒绝

始终使用默认拒绝策略：

```rego
# 在策略开头
default allow := false
```

#### 审计日志

记录所有访问决策：

```go
if auditLog, ok := result.Bindings["audit_log"].(map[string]interface{}); ok {
    logger.Info("Access decision",
        zap.String("user", auditLog["user"].(string)),
        zap.String("action", auditLog["action"].(string)),
        zap.Bool("allowed", auditLog["decision"].(bool)),
        zap.Int64("timestamp", auditLog["timestamp"].(int64)))
}
```

### 5. 测试策略

#### 测试覆盖率目标

- 目标：> 90% 覆盖率
- 测试所有主要规则
- 测试边界条件
- 测试拒绝场景

#### 测试分类

```bash
# 单元测试
opa test -v pkg/security/opa/policies/abac*.rego

# 集成测试
go test ./pkg/security/opa/... -v

# 性能测试
opa test --bench pkg/security/opa/policies/abac*.rego
```

### 6. 版本控制

#### 策略版本管理

```bash
# 提交策略变更
git add pkg/security/opa/policies/abac.rego
git commit -m "feat(abac): add device type validation"

# 使用标签标记版本
git tag -a abac-v1.1.0 -m "ABAC Policy v1.1.0"
```

#### 向后兼容性

- 添加新规则时保持现有规则不变
- 使用特性标志控制新功能
- 提供迁移指南

### 7. 文档化

为每个自定义规则提供文档：

```rego
# 规则：基于项目成员的访问控制
# 
# 描述：
#   项目成员可以读取和更新项目资源
# 
# 输入要求：
#   - input.subject.user: 用户标识符
#   - input.resource.type: 必须是 "project"
#   - input.resource.attributes.members: 成员列表
# 
# 示例：
#   {
#     "subject": {"user": "alice"},
#     "action": "read",
#     "resource": {
#       "type": "project",
#       "attributes": {"members": ["alice", "bob"]}
#     }
#   }
allow if {
    input.subject.user in project_members
    input.action in ["read", "update", "list"]
    input.resource.type == "project"
}
```

## 故障排查

### 问题 1: 策略评估返回 false，但预期应该通过

**可能原因**：
- 输入数据格式不正确
- 缺少必需的属性
- 属性值不匹配
- 评分未达到要求

**解决方法**：

```bash
# 使用 OPA REPL 调试
opa run pkg/security/opa/policies/abac.rego

# 在 REPL 中测试
> import data.abac
> abac.allow with input as {
    "subject": {"user": "alice", "roles": ["employee"]},
    "action": "read",
    "resource": "documents"
  }
true

# 检查评分
> abac.dynamic_attribute_score with input as {...}
45

# 检查各项分数
> abac.role_score with input as {...}
10
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

2. 分析策略性能：

```bash
# 运行性能基准测试
opa test --bench pkg/security/opa/policies/abac*.rego

# 使用 profiling
opa eval -d pkg/security/opa/policies/abac.rego \
  --profile --input input.json "data.abac.allow"
```

3. 优化策略规则：

```rego
# 优化前：每次都计算所有分数
allow if {
    dynamic_attribute_score >= required_score
}

# 优化后：提前检查简单条件
allow if {
    # 先检查是否为管理员（快）
    "admin" in input.subject.roles
}

allow if {
    # 再检查复杂评分（慢）
    dynamic_attribute_score >= required_score
}
```

### 问题 3: 测试失败

**常见原因**：
- 策略逻辑变更
- 测试数据过期
- 输入格式不匹配

**解决方法**：

```bash
# 运行详细测试输出
opa test -v pkg/security/opa/policies/abac*.rego

# 只运行失败的测试
opa test -v -r test_name pkg/security/opa/policies/abac*.rego

# 检查覆盖率
opa test --coverage pkg/security/opa/policies/abac*.rego
```

### 问题 4: 属性未正确提取

**可能原因**：
- 属性路径错误
- 属性不存在
- 数据类型不匹配

**调试方法**：

```rego
# 添加调试规则
debug_info := {
    "user_clearance": user_clearance_level,
    "resource_security": resource_security_level,
    "role_score": role_score,
    "clearance_score": clearance_score,
}
```

在应用中获取调试信息：

```go
result, err := client.Evaluate(ctx, "", input)
if err != nil {
    log.Fatal(err)
}

if debugInfo, ok := result.Bindings["debug_info"].(map[string]interface{}); ok {
    log.Printf("Debug: %+v", debugInfo)
}
```

### 问题 5: 跨租户或跨部门访问问题

**检查清单**：
- ✅ 确认租户/部门 ID 正确
- ✅ 检查共享配置是否生效
- ✅ 验证时间窗口未过期
- ✅ 确认权限列表正确

**调试示例**：

```bash
# 使用 OPA 命令行评估
opa eval -d pkg/security/opa/policies/abac.rego \
  --input input.json \
  --format pretty \
  "data.abac"
```

## 相关资源

- [OPA 官方文档](https://www.openpolicyagent.org/docs/latest/)
- [Rego 语言参考](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [ABAC vs RBAC 对比](https://www.openpolicyagent.org/docs/latest/comparison-to-other-systems/)
- [pkg/security/opa/README.md](../pkg/security/opa/README.md) - OPA 包文档
- [docs/opa-rbac-guide.md](./opa-rbac-guide.md) - RBAC 策略指南
- [Epic #776: OPA 策略引擎集成](https://github.com/innovationmech/swit/issues/776)
- [Task #800: ABAC 策略模板](https://github.com/innovationmech/swit/issues/800)

## 许可证

Copyright (c) 2024 Six-Thirty Labs, Inc.

Licensed under the Apache License, Version 2.0.

