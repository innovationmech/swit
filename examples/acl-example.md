# Saga ACL 使用示例

本文档展示如何使用 Saga 访问控制列表 (ACL) 系统。

## 基本概念

ACL 系统提供细粒度的资源级别访问控制，主要包含三个维度：

1. **Principal**: 主体（用户、角色等）
2. **Resource**: 资源（Saga 实例、配置等）
3. **Action**: 操作（execute、read、cancel 等）

## 快速开始

### 1. 创建 ACL 管理器

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/saga/security"
)

func main() {
    // 创建 ACL 管理器
    manager := security.NewACLManager(&security.ACLManagerConfig{
        DefaultEffect: security.ACLEffectDeny, // 默认拒绝
        EnableMetrics: true,                   // 启用性能指标
    })
}
```

### 2. 添加 ACL 规则

```go
// 允许用户 123 执行所有订单相关的 saga
rule1 := security.NewACLRule(
    "allow-user123-orders",  // 规则 ID
    100,                     // 优先级
    security.ACLEffectAllow, // 允许
    "user:123",             // 主体
    "saga:order-*",         // 资源（支持通配符）
    "execute",              // 操作
)
manager.AddRule(rule1)

// 拒绝用户 456 删除任何 saga
rule2 := security.NewACLRule(
    "deny-user456-delete",
    200,                     // 更高优先级
    security.ACLEffectDeny,  // 拒绝
    "user:456",
    "saga:*",               // 所有 saga
    "delete",
)
manager.AddRule(rule2)
```

### 3. 检查访问权限

```go
// 方式 1: 使用 Evaluate 获取详细决策
ctx := security.NewACLContext("user:123", "saga:order-456", "execute")
decision, err := manager.Evaluate(context.Background(), ctx)
if err != nil {
    // 处理错误
}

if decision.Allowed {
    fmt.Printf("访问允许: %s\n", decision.Reason)
    fmt.Printf("匹配规则: %s\n", decision.MatchedRule.ID)
    fmt.Printf("评估时间: %v\n", decision.EvaluationTime)
} else {
    fmt.Printf("访问拒绝: %s\n", decision.Reason)
}

// 方式 2: 使用 CheckAccess 简化检查
err = manager.CheckAccess(context.Background(), ctx)
if err != nil {
    // 访问被拒绝
    fmt.Printf("错误: %v\n", err)
}

// 方式 3: 使用便捷方法检查 Saga 访问
err = manager.CheckAccessForSaga(context.Background(), "user:123", "order-456", "execute")
if err != nil {
    // 访问被拒绝
}
```

## 高级特性

### 通配符匹配

```go
// 前缀匹配
rule := security.NewACLRule("rule1", 100, security.ACLEffectAllow,
    "user:*",        // 所有用户
    "saga:order-*",  // 所有订单 saga
    "read",
)

// 后缀匹配
rule2 := security.NewACLRule("rule2", 100, security.ACLEffectAllow,
    "*:admin",       // 所有管理员
    "saga:*",
    "*",             // 所有操作
)

// 完全通配
rule3 := security.NewACLRule("rule3", 100, security.ACLEffectAllow,
    "*",            // 所有主体
    "*",            // 所有资源
    "read",         // 仅读操作
)
```

### 条件匹配

```go
// 创建带条件的规则
rule := security.NewACLRule("rule-with-conditions", 100, security.ACLEffectAllow,
    "user:123", "saga:*", "execute")

// 添加 IP 限制
rule.Conditions["ip_address"] = "192.168.1.*"

// 添加时间限制
rule.Conditions["time"] = "business_hours"

manager.AddRule(rule)

// 评估时提供条件属性
ctx := security.NewACLContext("user:123", "saga:order-456", "execute").
    WithAttribute("ip_address", "192.168.1.100").
    WithAttribute("time", "business_hours")

decision, _ := manager.Evaluate(context.Background(), ctx)
```

### 规则过期

```go
import "time"

// 创建有效期为 1 小时的临时规则
rule := security.NewACLRule("temp-rule", 100, security.ACLEffectAllow,
    "user:temp", "saga:*", "read")

expiresAt := time.Now().Add(1 * time.Hour)
rule.ExpiresAt = &expiresAt

manager.AddRule(rule)

// 清理过期规则
removed := manager.CleanupExpiredRules()
fmt.Printf("清理了 %d 个过期规则\n", removed)
```

### RBAC 集成

```go
// 创建 RBAC 管理器
rbacManager := security.NewRBACManager(nil)
rbacManager.AssignRole("user:123", "operator")

// 创建集成 RBAC 的 ACL 管理器
aclManager := security.NewACLManager(&security.ACLManagerConfig{
    RBACManager: rbacManager,
})

// 为角色添加 ACL 规则
rule := security.NewACLRule("rule-for-operator", 100, security.ACLEffectAllow,
    "role:operator", // 针对角色
    "saga:*",
    "execute",
)
aclManager.AddRule(rule)

// 检查访问（会考虑用户的角色）
err := aclManager.CheckAccessWithRoles(context.Background(),
    "user:123", "saga:order-456", "execute")
// 用户 123 通过 operator 角色获得访问权限
```

### 规则管理

```go
// 列出所有规则
rules := manager.ListRules()
for _, rule := range rules {
    fmt.Printf("规则: %s, 优先级: %d\n", rule.ID, rule.Priority)
}

// 按 Principal 过滤
userRules := manager.ListRulesByPrincipal("user:123")

// 按 Resource 过滤
sagaRules := manager.ListRulesByResource("saga:order-123")

// 更新规则
rule, _ := manager.GetRule("rule-1")
rule.Priority = 500
manager.UpdateRule(rule)

// 删除规则
manager.DeleteRule("rule-1")

// 清除所有规则
manager.Clear()
```

### 规则导入导出

```go
// 导出规则（用于备份）
rules := manager.ExportRules()

// 保存到文件
data, _ := json.Marshal(rules)
os.WriteFile("acl-backup.json", data, 0644)

// 从文件恢复
data, _ := os.ReadFile("acl-backup.json")
var rules []*security.ACLRule
json.Unmarshal(data, &rules)

// 导入规则
newManager := security.NewACLManager(nil)
newManager.ImportRules(rules)
```

### 性能监控

```go
// 获取性能指标
metrics := manager.GetMetrics()

fmt.Printf("总评估次数: %v\n", metrics["total_evaluations"])
fmt.Printf("允许次数: %v\n", metrics["allowed_count"])
fmt.Printf("拒绝次数: %v\n", metrics["denied_count"])
fmt.Printf("平均评估时间: %v\n", metrics["average_eval_time"])
fmt.Printf("总规则数: %v\n", metrics["total_rules"])

// 重置指标
manager.ResetMetrics()
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/innovationmech/swit/pkg/saga/security"
)

func main() {
    // 1. 创建 ACL 管理器
    manager := security.NewACLManager(&security.ACLManagerConfig{
        DefaultEffect: security.ACLEffectDeny,
        EnableMetrics: true,
    })

    // 2. 添加规则
    // 允许管理员执行所有操作
    manager.AddRule(security.NewACLRule(
        "admin-full-access", 1000,
        security.ACLEffectAllow,
        "role:admin", "*", "*",
    ))

    // 允许操作员执行 saga
    manager.AddRule(security.NewACLRule(
        "operator-execute", 500,
        security.ACLEffectAllow,
        "role:operator", "saga:*", "execute",
    ))

    // 拒绝所有人删除生产环境 saga
    manager.AddRule(security.NewACLRule(
        "deny-prod-delete", 900,
        security.ACLEffectDeny,
        "*", "saga:prod-*", "delete",
    ))

    // 3. 检查访问
    testCases := []struct {
        user     string
        resource string
        action   string
    }{
        {"role:admin", "saga:prod-order-123", "delete"},     // 应该允许（管理员）
        {"role:operator", "saga:test-order-456", "execute"}, // 应该允许（操作员）
        {"role:operator", "saga:prod-order-789", "delete"},  // 应该拒绝（高优先级拒绝规则）
        {"user:guest", "saga:order-111", "read"},            // 应该拒绝（无规则，默认拒绝）
    }

    for _, tc := range testCases {
        ctx := security.NewACLContext(tc.user, tc.resource, tc.action)
        decision, _ := manager.Evaluate(context.Background(), ctx)

        status := "❌ 拒绝"
        if decision.Allowed {
            status = "✅ 允许"
        }

        fmt.Printf("%s - %s %s on %s: %s\n",
            status, tc.user, tc.action, tc.resource, decision.Reason)
    }

    // 4. 查看性能指标
    metrics := manager.GetMetrics()
    fmt.Printf("\n性能指标:\n")
    fmt.Printf("- 评估次数: %v\n", metrics["total_evaluations"])
    fmt.Printf("- 平均时间: %v\n", metrics["average_eval_time"])
}
```

## 最佳实践

1. **规则优先级规划**
   - 高优先级 (800-1000): 全局拒绝规则
   - 中优先级 (400-700): 角色和组规则
   - 低优先级 (100-300): 用户特定规则
   - 默认 (0): 回退规则

2. **通配符使用**
   - 优先使用具体规则，避免过度使用通配符
   - 使用前缀通配符来管理资源层级（如 `saga:order-*`）
   - 谨慎使用完全通配符 `*`

3. **性能优化**
   - 高频访问的规则使用较高优先级（减少评估次数）
   - 定期清理过期规则
   - 启用性能指标监控

4. **安全考虑**
   - 默认使用 `Deny` 效果（白名单模式）
   - 拒绝规则优先级高于允许规则
   - 定期审计和更新规则

5. **规则组织**
   - 使用有意义的规则 ID
   - 添加描述说明规则用途
   - 使用元数据标记规则类别

## 参考

- RBAC 实现: `pkg/saga/security/rbac.go`
- 权限定义: `pkg/saga/security/permission.go`
- 完整测试: `pkg/saga/security/acl_test.go`

