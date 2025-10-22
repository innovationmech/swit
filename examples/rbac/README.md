# Saga RBAC 使用示例

本目录包含 Saga RBAC (基于角色的访问控制) 系统的使用示例。

## 快速开始

### 1. 创建 RBAC 管理器

```go
import "github.com/innovationmech/swit/pkg/saga/security"

// 创建带缓存的 RBAC 管理器
manager := security.NewRBACManager(&security.RBACManagerConfig{
    CacheEnabled: true,
    CacheTTL:     5 * time.Minute,
    CacheMaxSize: 1000,
})
```

### 2. 使用预定义角色

系统提供了 4 个预定义角色：

- **admin**: 管理员，拥有所有权限
- **operator**: 操作员，可执行和管理 Saga
- **viewer**: 查看者，只读访问
- **auditor**: 审计员，可查看审计日志

```go
// 分配预定义角色
manager.AssignRole("user-123", "operator")
```

### 3. 创建自定义角色

```go
// 创建自定义角色
role := security.NewRole(
    "saga-executor",
    "Saga Executor - Can execute sagas",
    security.PermissionSagaExecute,
    security.PermissionSagaRead,
    security.PermissionStepExecute,
)

// 添加到管理器
err := manager.CreateRole(role)
if err != nil {
    log.Fatal(err)
}

// 分配给用户
manager.AssignRole("user-456", "saga-executor")
```

### 4. 权限继承

```go
// 创建父角色
baseRole := security.NewRole("base-reader", "Base Reader", 
    security.PermissionSagaRead,
    security.PermissionStepRead,
)
manager.CreateRole(baseRole)

// 创建子角色，继承父角色权限
advancedRole := security.NewRole("advanced-reader", "Advanced Reader",
    security.PermissionMonitorView,
    security.PermissionAuditView,
)
advancedRole.AddParent("base-reader")
manager.CreateRole(advancedRole)

// 用户将拥有父角色和子角色的所有权限
manager.AssignRole("user-789", "advanced-reader")
```

### 5. 使用权限检查中间件

```go
// 创建 RBAC 中间件
rbacMiddleware := security.NewRBACMiddleware(&security.RBACMiddlewareConfig{
    RBACManager: manager,
})

// 在处理函数中检查权限
func handleSagaExecution(ctx context.Context) error {
    // 检查单个权限
    if err := rbacMiddleware.RequirePermission(ctx, security.PermissionSagaExecute); err != nil {
        return fmt.Errorf("permission denied: %w", err)
    }
    
    // 执行 Saga 操作
    // ...
    return nil
}

// 检查多个权限（需全部满足）
func handleSagaUpdate(ctx context.Context) error {
    if err := rbacMiddleware.RequirePermissions(ctx,
        security.PermissionSagaUpdate,
        security.PermissionSagaRead,
    ); err != nil {
        return err
    }
    // ...
}

// 检查多个权限（满足任一即可）
func handleSagaView(ctx context.Context) error {
    if err := rbacMiddleware.RequireAnyPermission(ctx,
        security.PermissionSagaRead,
        security.PermissionMonitorView,
    ); err != nil {
        return err
    }
    // ...
}
```

### 6. 便捷的辅助方法

```go
// 使用布尔返回值的辅助方法
if rbacMiddleware.HasPermission(ctx, security.PermissionSagaExecute) {
    // 用户有执行权限
}

if rbacMiddleware.HasRole(ctx, "admin") {
    // 用户是管理员
}

if rbacMiddleware.HasAnyPermission(ctx, 
    security.PermissionSagaCreate,
    security.PermissionSagaUpdate,
) {
    // 用户有创建或更新权限
}
```

### 7. 获取用户权限

```go
// 获取用户的所有权限（包括继承的）
permissions, err := manager.GetUserPermissions("user-123")
if err != nil {
    log.Fatal(err)
}

// 检查特定权限
if permissions.Has(security.PermissionSagaExecute) {
    fmt.Println("User can execute sagas")
}

// 列出所有权限
for _, perm := range permissions.List() {
    fmt.Printf("Permission: %s\n", perm)
}
```

### 8. 集成到 Saga 执行流程

```go
// 在 Saga 协调器中集成 RBAC
type SagaCoordinator struct {
    rbacMiddleware *security.RBACMiddleware
    // ... other fields
}

func (c *SagaCoordinator) Execute(ctx context.Context, sagaID string) error {
    // 1. 验证身份（假设已有 AuthContext）
    authCtx, ok := security.AuthFromContext(ctx)
    if !ok {
        return errors.New("authentication required")
    }
    
    // 2. 检查权限
    if err := c.rbacMiddleware.RequirePermission(ctx, security.PermissionSagaExecute); err != nil {
        return fmt.Errorf("permission denied: %w", err)
    }
    
    // 3. 执行 Saga
    // ...
    
    return nil
}
```

## 权限列表

### Saga 执行权限
- `saga:execute` - 执行 Saga
- `saga:read` - 读取 Saga 信息
- `saga:cancel` - 取消 Saga
- `saga:retry` - 重试 Saga
- `saga:rollback` - 回滚 Saga
- `saga:compensate` - 补偿 Saga

### Saga 管理权限
- `saga:create` - 创建 Saga
- `saga:update` - 更新 Saga
- `saga:delete` - 删除 Saga
- `saga:list` - 列出 Saga

### 步骤级别权限
- `step:execute` - 执行步骤
- `step:read` - 读取步骤
- `step:update` - 更新步骤

### 监控和审计权限
- `monitor:view` - 查看监控信息
- `monitor:admin` - 管理监控配置
- `audit:view` - 查看审计日志
- `audit:export` - 导出审计日志

### 管理权限
- `admin:full` - 完全管理权限
- `role:manage` - 管理角色
- `user:manage` - 管理用户
- `config:write` - 写配置
- `config:read` - 读配置

## 性能优化

RBAC 系统使用了权限缓存来提升性能：

```go
// 获取缓存统计
stats := manager.GetCacheStats()
fmt.Printf("Cache hit rate: %.2f%%\n", stats["hit_rate"].(float64) * 100)
fmt.Printf("Cache size: %d\n", stats["size"].(int))
```

典型性能指标：
- 权限检查延迟（有缓存）: <5ms
- 权限检查延迟（无缓存）: <10ms
- 缓存命中率: >90%

## 最佳实践

1. **使用预定义角色**: 优先使用系统提供的预定义角色
2. **权限最小化**: 只授予必要的权限
3. **角色继承**: 利用继承减少重复配置
4. **启用缓存**: 在生产环境中启用权限缓存
5. **定期审计**: 定期检查用户权限分配

## 相关文档

- [RBAC 设计文档](../../docs/saga-rbac.md)
- [安全最佳实践](../../docs/security-best-practices.md)
- [认证中间件](../../pkg/saga/security/auth.go)

