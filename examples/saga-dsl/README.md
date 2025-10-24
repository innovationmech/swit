# Saga DSL 示例集

本目录包含多个完整的 Saga DSL 示例，演示了不同的业务场景和 DSL 特性。

## 📚 示例列表

### 1. 订单处理 Saga（order-processing.saga.yaml）

**业务场景**: 电商订单处理流程

**包含的步骤**:
- 订单验证
- 库存预留
- 支付处理
- 订单确认
- 发货通知
- 积分更新

**演示的特性**:
- ✅ 基本的步骤定义和依赖关系
- ✅ 服务调用（HTTP 和 gRPC）
- ✅ 自定义补偿操作
- ✅ 重试策略配置
- ✅ 成功/失败钩子
- ✅ 条件执行
- ✅ 异步步骤
- ✅ 消息发布

**适用于**: 初学者了解基本的 Saga 工作流

**运行方式**:
```go
package main

import (
    "log"
    "github.com/innovationmech/swit/pkg/saga/dsl"
)

func main() {
    // 解析 Saga 定义
    def, err := dsl.ParseFile("examples/saga-dsl/order-processing.saga.yaml")
    if err != nil {
        log.Fatalf("Failed to parse: %v", err)
    }
    
    log.Printf("Loaded Saga: %s (v%s)", def.Saga.Name, def.Saga.Version)
    log.Printf("Steps: %d", len(def.Steps))
}
```

---

### 2. 支付流程 Saga（payment-flow.saga.yaml）

**业务场景**: 完整的支付处理流程，包含风险评估和多种支付方式

**包含的步骤**:
- 支付请求验证
- 欺诈风险评估
- 3D Secure 认证
- 支付授权
- 支付捕获
- 账户余额更新
- 交易记录
- 支付确认通知
- 分析指标更新

**演示的特性**:
- ✅ 复杂的支付处理流程
- ✅ 条件步骤执行（3DS）
- ✅ 多层重试策略
- ✅ 关键步骤的补偿失败处理
- ✅ 幂等性保证（使用 Idempotency-Key）
- ✅ 实时通知集成
- ✅ 风险评估和欺诈检测
- ✅ PCI-DSS 合规性考虑

**适用于**: 
- 学习支付处理的最佳实践
- 理解金融交易的补偿机制
- 了解如何处理关键业务流程

**关键配置**:
```yaml
# 支付授权的补偿 - 支付失败时作废授权
compensation:
  type: custom
  action:
    service:
      name: payment-gateway-service
      method: POST
      path: /api/v1/void
      body:
        authorization_id: "{{.output.authorize-payment.authorization_id}}"
  timeout: 1m
  max_attempts: 5
  on_failure:
    action: alert
    notifications:
      - type: pagerduty
        message: "CRITICAL: Failed to void authorization"
```

---

### 3. 库存管理 Saga（inventory.saga.yaml）

**业务场景**: 跨仓库的分布式库存管理和分配

**包含的步骤**:
- 库存可用性检查
- 库存分配优化
- 多仓库库存预留
- 库存锁定
- 库存分配确认
- 释放库存锁
- 更新库存报告
- 触发补货检查
- 同步到 ERP 系统
- 生成拣货单
- 发送分配通知

**演示的特性**:
- ✅ 多仓库场景处理
- ✅ 资源锁定机制
- ✅ 分布式库存管理
- ✅ 优化算法集成
- ✅ 条件执行（备用仓库）
- ✅ ERP 系统集成
- ✅ 补货触发机制
- ✅ 完善的错误恢复

**适用于**:
- 学习分布式资源管理
- 理解库存锁定和释放机制
- 了解跨系统集成

**库存锁定示例**:
```yaml
- id: lock-inventory
  name: Lock Reserved Inventory
  action:
    service:
      name: inventory-lock-service
      method: POST
      path: /api/v1/locks/acquire
  compensation:
    type: custom
    action:
      service:
        name: inventory-lock-service
        method: POST
        path: /api/v1/locks/release
        body:
          lock_id: "{{.output.lock-inventory.lock_id}}"
          force: true
```

---

### 4. 复杂旅行预订 Saga（complex-travel-booking.saga.yaml）

**业务场景**: 多服务协调的完整旅行预订流程

**包含的步骤**:
- 预订请求验证
- 签证需求检查
- 总费用计算
- **并行预订**:
  - 往返航班预订
  - 酒店预订
  - 租车预订（条件）
- 旅游保险购买（条件）
- 签证申请（条件、异步）
- 最终金额计算
- 支付处理
- 所有服务确认
- 生成确认文档
- 发送通知
- 创建日历事件
- 更新积分

**演示的特性**:
- ✅ 复杂的步骤依赖关系
- ✅ 并行步骤执行
- ✅ 多个条件步骤
- ✅ 异步长时间运行步骤
- ✅ 多阶段工作流
- ✅ 综合性补偿策略
- ✅ 多服务协调
- ✅ 实时客户通知

**适用于**:
- 学习复杂工作流设计
- 理解并行和串行步骤的协调
- 掌握多服务 Saga 的最佳实践

**并行执行示例**:
```yaml
# 这三个步骤会并行执行（无相互依赖）
- id: reserve-outbound-flight
  dependencies:
    - calculate-total-cost

- id: reserve-return-flight
  dependencies:
    - calculate-total-cost

- id: reserve-hotel
  dependencies:
    - calculate-total-cost
```

---

## 🎯 特性对照表

| 特性 | 订单处理 | 支付流程 | 库存管理 | 旅行预订 |
|------|---------|---------|---------|---------|
| 基本步骤定义 | ✅ | ✅ | ✅ | ✅ |
| 依赖关系 | ✅ | ✅ | ✅ | ✅ |
| 并行执行 | ❌ | ❌ | ✅ | ✅ |
| 条件执行 | ✅ | ✅ | ✅ | ✅ |
| 异步步骤 | ✅ | ✅ | ✅ | ✅ |
| 自定义补偿 | ✅ | ✅ | ✅ | ✅ |
| 重试策略 | ✅ | ✅ | ✅ | ✅ |
| 钩子 (Hooks) | ✅ | ✅ | ✅ | ✅ |
| 消息发布 | ✅ | ✅ | ✅ | ✅ |
| HTTP 调用 | ✅ | ✅ | ✅ | ✅ |
| gRPC 调用 | ✅ | ❌ | ❌ | ❌ |
| 函数执行 | ❌ | ❌ | ❌ | ✅ |
| 资源锁定 | ❌ | ❌ | ✅ | ❌ |
| 多阶段流程 | ✅ | ✅ | ✅ | ✅ |
| 复杂度 | 中 | 高 | 高 | 很高 |

## 📖 使用指南

### 如何选择示例

1. **刚开始学习 Saga DSL？**
   - 从 `order-processing.saga.yaml` 开始
   - 理解基本概念和语法

2. **需要处理支付或金融交易？**
   - 查看 `payment-flow.saga.yaml`
   - 学习关键业务流程的最佳实践

3. **需要管理分布式资源？**
   - 参考 `inventory.saga.yaml`
   - 了解资源锁定和多系统协调

4. **需要构建复杂工作流？**
   - 研究 `complex-travel-booking.saga.yaml`
   - 学习如何设计多服务协调的 Saga

### 测试示例

#### 1. 验证语法

使用解析器验证 YAML 语法：

```go
package main

import (
    "fmt"
    "log"
    "github.com/innovationmech/swit/pkg/saga/dsl"
)

func main() {
    files := []string{
        "examples/saga-dsl/order-processing.saga.yaml",
        "examples/saga-dsl/payment-flow.saga.yaml",
        "examples/saga-dsl/inventory.saga.yaml",
        "examples/saga-dsl/complex-travel-booking.saga.yaml",
    }
    
    for _, file := range files {
        fmt.Printf("\nValidating: %s\n", file)
        def, err := dsl.ParseFile(file)
        if err != nil {
            log.Printf("❌ Validation failed: %v\n", err)
            continue
        }
        fmt.Printf("✅ Valid! Saga: %s (v%s), Steps: %d\n", 
            def.Saga.Name, def.Saga.Version, len(def.Steps))
    }
}
```

#### 2. 检查步骤依赖

```go
func analyzeDependencies(def *dsl.SagaDefinition) {
    fmt.Printf("\n=== Dependency Analysis: %s ===\n", def.Saga.Name)
    
    for _, step := range def.Steps {
        if len(step.Dependencies) > 0 {
            fmt.Printf("Step '%s' depends on: %v\n", step.ID, step.Dependencies)
        } else {
            fmt.Printf("Step '%s' has no dependencies (can run first)\n", step.ID)
        }
    }
}
```

#### 3. 可视化 Saga 流程

```go
func visualizeSaga(def *dsl.SagaDefinition) {
    fmt.Printf("\n=== Saga Flow: %s ===\n", def.Saga.Name)
    fmt.Println("graph TD")
    
    for _, step := range def.Steps {
        nodeId := step.ID
        if len(step.Dependencies) == 0 {
            fmt.Printf("    START --> %s\n", nodeId)
        }
        for _, dep := range step.Dependencies {
            fmt.Printf("    %s --> %s\n", dep, nodeId)
        }
    }
}
```

### 修改示例以适应您的需求

#### 1. 更改服务端点

```yaml
# 原始
action:
  service:
    name: payment-service
    endpoint: http://payment-service:8080

# 修改为您的服务
action:
  service:
    name: my-payment-service
    endpoint: http://my-payment.example.com
```

#### 2. 调整超时时间

```yaml
# 根据您的服务响应时间调整
timeout: 30s  # 改为 5s, 1m, 等
```

#### 3. 修改重试策略

```yaml
retry_policy:
  type: exponential_backoff
  max_attempts: 5      # 根据需要调整
  initial_delay: 1s    # 调整初始延迟
  max_delay: 1m        # 调整最大延迟
```

#### 4. 添加自定义元数据

```yaml
metadata:
  owner: your-team
  project: your-project
  environment: production
  custom_field: your-value
```

## 🔍 常见模式

### 模式 1: 预留和确认

许多业务场景需要先预留资源，然后在确认支付后才最终分配：

```yaml
- id: reserve-resource
  name: Reserve Resource
  action:
    service:
      method: POST
      path: /api/reserve
  compensation:
    type: custom
    action:
      service:
        method: POST
        path: /api/release

- id: confirm-reservation
  name: Confirm Reservation
  action:
    service:
      method: POST
      path: /api/confirm
  dependencies:
    - reserve-resource
    - process-payment
```

**使用场景**: 订单处理、库存管理、座位预订等

### 模式 2: 验证-执行-确认

关键操作前进行验证，执行后确认：

```yaml
- id: validate
  name: Validate Request
  compensation:
    type: skip  # 验证无副作用

- id: execute
  name: Execute Action
  dependencies:
    - validate
  compensation:
    type: custom  # 执行需要补偿

- id: confirm
  name: Confirm Action
  dependencies:
    - execute
```

**使用场景**: 支付处理、数据修改、状态变更等

### 模式 3: 并行执行 + 汇总

多个独立操作并行执行，然后汇总结果：

```yaml
# 并行执行
- id: task-a
  name: Task A
  dependencies:
    - validation

- id: task-b
  name: Task B
  dependencies:
    - validation

- id: task-c
  name: Task C
  dependencies:
    - validation

# 等待所有任务完成后汇总
- id: aggregate-results
  name: Aggregate Results
  dependencies:
    - task-a
    - task-b
    - task-c
```

**使用场景**: 多服务协调、数据聚合、批量处理等

### 模式 4: 条件分支

根据条件决定执行路径：

```yaml
- id: check-condition
  name: Check Condition

- id: path-a
  name: Path A
  condition:
    expression: "$output.check-condition.result == 'A'"
  dependencies:
    - check-condition

- id: path-b
  name: Path B
  condition:
    expression: "$output.check-condition.result == 'B'"
  dependencies:
    - check-condition
```

**使用场景**: 业务规则处理、A/B 测试、个性化流程等

### 模式 5: 异步后处理

主流程完成后异步执行非关键任务：

```yaml
- id: main-task
  name: Main Business Logic

- id: async-notification
  name: Send Notification
  async: true
  dependencies:
    - main-task

- id: async-analytics
  name: Update Analytics
  async: true
  dependencies:
    - main-task
```

**使用场景**: 通知发送、日志记录、分析更新等

## 🚀 最佳实践

### 1. 步骤设计

- ✅ 每个步骤应该是原子操作
- ✅ 步骤之间通过输出/输入传递数据
- ✅ 避免步骤间的副作用
- ✅ 使用清晰描述性的 ID 和名称

### 2. 补偿设计

- ✅ 所有有副作用的步骤必须有补偿
- ✅ 补偿操作应该是幂等的
- ✅ 关键补偿应该有失败告警
- ✅ 只读操作可以跳过补偿

### 3. 超时配置

- ✅ 为每个步骤设置合理的超时
- ✅ Saga 超时应大于所有步骤超时之和
- ✅ 考虑网络延迟和服务响应时间
- ✅ 长时间运行的步骤使用 async

### 4. 重试策略

- ✅ 使用指数退避避免服务过载
- ✅ 添加 jitter 避免重试风暴
- ✅ 明确指定可重试的错误类型
- ✅ 非幂等操作谨慎配置重试

### 5. 错误处理

- ✅ 使用钩子捕获和记录错误
- ✅ 关键错误应该触发告警
- ✅ 为客户可见的操作提供友好的错误消息
- ✅ 记录足够的上下文用于故障排查

### 6. 监控和可观测性

- ✅ 使用标签和元数据进行分类
- ✅ 在关键步骤添加指标收集
- ✅ 使用结构化日志
- ✅ 集成分布式追踪

## 🔧 故障排查

### 问题 1: 解析失败

**错误**: `parse error: YAML syntax error`

**解决方法**:
1. 检查 YAML 语法（缩进、引号等）
2. 验证字段名称拼写
3. 确保必需字段都已填写

```bash
# 使用 YAML 验证工具
yamllint examples/saga-dsl/your-saga.yaml
```

### 问题 2: 循环依赖

**错误**: `circular dependency detected`

**解决方法**:
1. 检查步骤依赖关系
2. 使用图形化工具可视化依赖
3. 确保依赖是有向无环图（DAG）

### 问题 3: 补偿失败

**错误**: `compensation failed for step X`

**解决方法**:
1. 检查补偿操作的幂等性
2. 增加补偿重试次数
3. 添加告警通知
4. 考虑人工介入机制

### 问题 4: 超时

**错误**: `step timeout exceeded`

**解决方法**:
1. 增加步骤超时时间
2. 优化服务响应时间
3. 将长时间操作标记为 async
4. 检查网络连接和服务健康状态

## 📚 相关文档

- [Saga DSL 快速入门](../../docs/saga-dsl-guide.md)
- [Saga DSL 完整参考](../../docs/saga-dsl-reference.md)
- [Saga 用户指南](../../docs/saga-user-guide.md)
- [Saga 架构设计](../../docs/architecture.md)
- [API 文档](https://pkg.go.dev/github.com/innovationmech/swit/pkg/saga/dsl)

## 💡 贡献示例

欢迎贡献新的示例！请确保：

1. 示例代表真实的业务场景
2. 包含详细的注释说明
3. 演示特定的 DSL 特性或模式
4. 遵循命名约定和最佳实践
5. 在 README 中添加示例说明

提交 PR 时请包含：
- 示例 YAML 文件
- 业务场景说明
- 演示的特性列表
- 使用说明

## 🙋 获取帮助

- 查看 [常见问题](../../docs/saga-dsl-guide.md#常见问题)
- 提交 [GitHub Issue](https://github.com/innovationmech/swit/issues)
- 加入社区讨论

---

**最后更新**: 2025-01
**版本**: 2.0.0

