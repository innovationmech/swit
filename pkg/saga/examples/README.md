# Saga 示例应用文档

本目录包含了 Swit 框架 Saga 分布式事务系统的完整示例应用，展示了在实际业务场景中如何使用 Saga 模式处理复杂的分布式事务。

## 目录结构

```
examples/
├── README.md                           # 本文档
├── order_saga.go                       # 订单处理示例
├── order_saga_test.go                  # 订单处理测试
├── payment_saga.go                     # 支付处理示例
├── payment_saga_test.go                # 支付处理测试
├── inventory_saga.go                   # 库存管理示例
├── inventory_saga_test.go              # 库存管理测试
├── user_registration_saga.go           # 用户注册示例
├── user_registration_saga_test.go      # 用户注册测试
├── e2e_test.go                         # 端到端测试
├── E2E_TESTING.md                      # 端到端测试文档
└── docs/                               # 详细文档目录
    ├── order_saga.md                   # 订单处理详细文档
    ├── payment_saga.md                 # 支付处理详细文档
    ├── inventory_saga.md               # 库存管理详细文档
    ├── user_registration_saga.md       # 用户注册详细文档
    └── architecture.md                 # 架构设计文档
```

## 示例概览

### 1. 订单处理 Saga（Order Processing）

**文件**: `order_saga.go`  
**文档**: [docs/order_saga.md](docs/order_saga.md)

演示了电商订单处理的完整流程，包括：
- 创建订单
- 预留库存
- 处理支付
- 确认订单
- 发送通知

**适用场景**：
- 电商下单流程
- 多步骤事务协调
- 资源预留与释放
- 订单状态管理

### 2. 支付处理 Saga（Payment Processing）

**文件**: `payment_saga.go`  
**文档**: [docs/payment_saga.md](docs/payment_saga.md)

演示了跨账户资金转账的完整流程，包括：
- 账户验证
- 资金冻结
- 执行转账
- 风险检测
- 生成凭证

**适用场景**：
- 金融转账业务
- 资金安全管理
- 风控系统集成
- 账户余额操作

### 3. 库存管理 Saga（Inventory Management）

**文件**: `inventory_saga.go`  
**文档**: [docs/inventory_saga.md](docs/inventory_saga.md)

演示了多仓库库存协调的完整流程，包括：
- 查询可用仓库
- 库存可用性检查
- 分配库存
- 锁定库存
- 生成提货单

**适用场景**：
- 多仓库管理
- 库存分配策略
- 供应链协调
- 库存预留机制

### 4. 用户注册 Saga（User Registration）

**文件**: `user_registration_saga.go`  
**文档**: [docs/user_registration_saga.md](docs/user_registration_saga.md)

演示了用户注册和初始化的完整流程，包括：
- 创建用户账户
- 发送验证邮件
- 初始化配置
- 分配资源配额
- 欢迎邮件

**适用场景**：
- 用户注册流程
- 账户初始化
- 资源分配
- 邮件验证

## 快速开始

### 使用 Docker Compose（推荐）

最简单的方式是使用 Docker Compose 启动所有依赖服务：

```bash
# 启动所有依赖服务
cd pkg/saga/examples
docker compose up -d

# 运行示例
go test -v -run TestOrderSaga
```

详细的 Docker 使用说明，请参考 [DOCKER.md](DOCKER.md)。

### 前置条件

- Go 1.23+
- 已安装 Swit 框架依赖
- （可选）Docker 和 Docker Compose - 用于运行依赖服务

### 运行示例

#### 1. 订单处理示例

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/examples"
)

func main() {
    // 创建内存状态存储
    stateStorage := coordinator.NewInMemoryStateStorage()
    
    // 创建内存事件发布器
    eventPublisher := coordinator.NewInMemoryEventPublisher()
    
    // 配置 Coordinator
    config := &coordinator.OrchestratorConfig{
        StateStorage:   stateStorage,
        EventPublisher: eventPublisher,
        RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
    }
    
    // 创建 Coordinator
    sagaCoordinator, err := coordinator.NewOrchestratorCoordinator(config)
    if err != nil {
        log.Fatal(err)
    }
    defer sagaCoordinator.Close()
    
    // 创建订单数据
    orderData := &examples.OrderData{
        CustomerID:  "CUST-12345",
        TotalAmount: 299.99,
        Items: []examples.OrderItem{
            {
                ProductID: "PROD-001",
                SKU:       "SKU-001",
                Quantity:  2,
                UnitPrice: 149.99,
                Total:     299.98,
            },
        },
        Address: examples.ShippingAddress{
            RecipientName: "张三",
            Phone:         "13800138000",
            Province:      "北京市",
            City:          "北京市",
            District:      "朝阳区",
            Street:        "朝阳路100号",
            Zipcode:       "100000",
        },
        PaymentMethod: "credit_card",
        PaymentToken:  "tok_1234567890",
    }
    
    // 创建 Saga 定义
    sagaDef := examples.NewOrderSagaDefinition()
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, orderData)
    if err != nil {
        log.Fatalf("启动 Saga 失败: %v", err)
    }
    
    log.Printf("Saga 已启动，实例 ID: %s", instance.GetID())
    
    // 等待 Saga 完成
    time.Sleep(5 * time.Second)
    
    // 检查状态
    finalInstance, err := sagaCoordinator.GetSagaInstance(instance.GetID())
    if err != nil {
        log.Fatalf("获取 Saga 实例失败: %v", err)
    }
    
    log.Printf("Saga 最终状态: %s", finalInstance.GetStatus())
}
```

#### 2. 支付处理示例

```go
// 创建转账数据
transferData := &examples.TransferData{
    FromAccount: "ACC-001",
    ToAccount:   "ACC-002",
    Amount:      1000.00,
    Currency:    "CNY",
    Purpose:     "salary",
    Description: "工资转账",
    CustomerID:  "CUST-12345",
    CustomerName: "张三",
    RiskLevel:   "low",
}

// 创建 Saga 定义
sagaDef := examples.NewPaymentSagaDefinition()

// 启动 Saga
instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, transferData)
```

#### 3. 库存管理示例

```go
// 创建库存请求数据
inventoryData := &examples.InventoryRequestData{
    OrderID:     "ORD-12345",
    CustomerID:  "CUST-12345",
    RequestType: "reservation",
    Items: []examples.InventoryItem{
        {
            ProductID:   "PROD-001",
            SKU:         "SKU-001",
            Quantity:    10,
            Unit:        "piece",
            Description: "商品描述",
        },
    },
    PreferredWarehouses: []string{"WH-BJ-01", "WH-SH-01"},
    AllocationStrategy:  "priority",
    Priority:            5,
}

// 创建 Saga 定义
sagaDef := examples.NewInventorySagaDefinition()

// 启动 Saga
instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, inventoryData)
```

#### 4. 用户注册示例

```go
// 创建注册数据
registrationData := &examples.UserRegistrationData{
    Email:        "user@example.com",
    Username:     "newuser",
    Password:     "hashed_password_here",
    FirstName:    "三",
    LastName:     "张",
    Phone:        "13800138000",
    Source:       "web",
    Language:     "zh-CN",
    Timezone:     "Asia/Shanghai",
    Subscription: "free",
    AcceptedTOS:  true,
    AcceptedAt:   time.Now(),
}

// 创建 Saga 定义
sagaDef := examples.NewUserRegistrationSagaDefinition()

// 启动 Saga
instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, registrationData)
```

## 运行测试

### 运行所有测试

```bash
cd pkg/saga/examples
go test -v
```

### 运行特定示例的测试

```bash
# 订单处理测试
go test -v -run TestOrderSaga

# 支付处理测试
go test -v -run TestPaymentSaga

# 库存管理测试
go test -v -run TestInventorySaga

# 用户注册测试
go test -v -run TestUserRegistrationSaga
```

### 运行端到端测试

```bash
go test -v -run TestE2E
```

### 查看测试覆盖率

```bash
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 核心特性展示

### 1. 步骤执行和补偿

所有示例都展示了如何：
- 定义步骤的执行逻辑
- 实现相应的补偿操作
- 处理执行失败和回滚

### 2. 错误处理和重试

示例展示了多种错误处理策略：
- 可重试错误 vs 不可重试错误
- 自定义重试策略
- 指数退避算法
- 最大重试次数限制

### 3. 超时管理

演示了如何配置：
- Saga 级别的全局超时
- 步骤级别的独立超时
- 超时处理和恢复

### 4. 状态持久化

展示了状态管理的最佳实践：
- 步骤结果的保存和传递
- 补偿数据的准备
- 状态查询和恢复

### 5. 事件发布

演示了事件驱动的集成方式：
- Saga 生命周期事件
- 步骤执行事件
- 补偿事件
- 自定义业务事件

## 架构模式

所有示例都遵循以下架构模式：

### 数据结构组织

```
示例文件结构：
├── 数据结构定义（Data Structures）
│   ├── 输入数据类型
│   ├── 步骤结果类型
│   └── 辅助数据类型
├── 步骤实现（Step Implementations）
│   ├── 步骤结构体
│   ├── Execute 方法
│   ├── Compensate 方法
│   └── 接口方法实现
├── Saga 定义（Saga Definition）
│   ├── 定义结构体
│   ├── 步骤列表
│   ├── 配置参数
│   └── 接口方法实现
└── 辅助函数（Helper Functions）
    ├── 构造函数
    ├── 验证函数
    └── 工具函数
```

### 步骤设计原则

1. **单一职责**：每个步骤只负责一个明确的业务操作
2. **幂等性**：步骤和补偿操作都应该是幂等的
3. **原子性**：每个步骤应该是原子操作
4. **可观测性**：记录详细的日志和指标
5. **错误分类**：区分可重试和不可重试的错误

### 补偿策略

1. **向后补偿**：回滚已完成的步骤（默认）
2. **前向恢复**：继续执行后续步骤
3. **混合策略**：根据错误类型选择策略

## 最佳实践

### 1. 数据设计

- 使用结构体而非 map 存储数据
- 为每个步骤定义清晰的输入输出类型
- 在输入数据中包含必要的元数据
- 使用类型安全的数据传递

### 2. 错误处理

```go
// 定义业务错误类型
type BusinessError struct {
    Code       string
    Message    string
    Retryable  bool
    Details    map[string]interface{}
}

func (e *BusinessError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 在步骤中使用
func (s *Step) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // ... 业务逻辑
    
    if someCondition {
        return nil, &BusinessError{
            Code:      "INSUFFICIENT_BALANCE",
            Message:   "账户余额不足",
            Retryable: false,
            Details: map[string]interface{}{
                "required": 1000.00,
                "available": 500.00,
            },
        }
    }
    
    return result, nil
}
```

### 3. 日志记录

```go
func (s *Step) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 记录开始执行
    log.Printf("[%s] 开始执行步骤", s.GetID())
    
    // 执行业务逻辑
    result, err := s.doWork(ctx, data)
    
    if err != nil {
        // 记录错误详情
        log.Printf("[%s] 执行失败: %v", s.GetID(), err)
        return nil, err
    }
    
    // 记录成功结果
    log.Printf("[%s] 执行成功", s.GetID())
    return result, nil
}
```

### 4. 超时配置

```go
// Saga 级别的全局超时
sagaDef.SetTimeout(30 * time.Minute)

// 步骤级别的超时（覆盖全局配置）
step.SetTimeout(5 * time.Second)

// 在 Context 中使用超时
ctx, cancel := context.WithTimeout(parentCtx, step.GetTimeout())
defer cancel()
```

### 5. 重试策略

```go
// 使用指数退避
retryPolicy := saga.NewExponentialBackoffRetryPolicy(
    3,                    // 最大重试次数
    1*time.Second,        // 初始延迟
    10*time.Second,       // 最大延迟
)

// 使用固定间隔
retryPolicy := saga.NewFixedIntervalRetryPolicy(
    5,                    // 最大重试次数
    2*time.Second,        // 重试间隔
)

// 自定义重试条件
func (s *Step) IsRetryable(err error) bool {
    if busErr, ok := err.(*BusinessError); ok {
        return busErr.Retryable
    }
    // 默认网络错误可重试
    return true
}
```

## 常见问题 (FAQ)

### Q1: 如何选择合适的示例作为起点？

**A**: 根据您的业务场景选择：
- **订单处理**：适合需要协调多个服务的场景
- **支付处理**：适合金融类、需要高可靠性的场景
- **库存管理**：适合需要资源分配和协调的场景
- **用户注册**：适合用户生命周期管理场景

### Q2: 步骤执行失败后会发生什么？

**A**: 
1. 系统会根据重试策略尝试重新执行
2. 如果重试仍然失败，会触发补偿流程
3. 已完成的步骤会按相反顺序执行补偿操作
4. 最终 Saga 状态会变为 `Compensated`

### Q3: 如何处理部分补偿失败的情况？

**A**: 
- 补偿操作会根据配置进行重试
- 可以配置补偿超时时间
- 系统会记录补偿失败的详细信息
- 可以通过监控面板查看和手动干预

### Q4: 如何在生产环境中使用这些示例？

**A**: 
1. 替换内存存储为持久化存储（Redis/PostgreSQL）
2. 集成真实的消息队列（NATS/RabbitMQ）
3. 添加监控和告警
4. 实现真实的业务逻辑而非模拟
5. 配置合适的超时和重试策略
6. 添加安全认证和授权

### Q5: 如何测试 Saga 的补偿逻辑？

**A**: 
参考各示例的测试文件：
- 使用 `FailAfterN` 模拟特定步骤失败
- 验证补偿操作的执行顺序
- 检查最终状态是否正确
- 验证数据是否正确回滚

### Q6: 性能优化建议？

**A**: 
1. 使用异步执行非关键步骤
2. 合理设置超时时间
3. 使用连接池管理外部资源
4. 批量处理相似操作
5. 缓存频繁访问的数据
6. 使用分布式追踪定位瓶颈

### Q7: 如何处理并发问题？

**A**: 
- 使用乐观锁处理并发更新
- 在关键资源上使用分布式锁
- 确保步骤和补偿操作的幂等性
- 使用版本号或时间戳检测冲突

## 相关文档

- [Docker 使用指南](DOCKER.md) - Docker Compose 配置和使用
- [脚本使用指南](scripts/README.md) - 运行脚本和开发工具
- [端到端测试](E2E_TESTING.md) - 端到端测试文档
- [架构设计](docs/architecture.md) - 架构设计文档
- [Saga 用户指南](../../../docs/saga-user-guide.md)
- [Saga DSL 指南](../../../docs/saga-dsl-guide.md)
- [Saga 监控指南](../../../docs/saga-monitoring-guide.md)

## 贡献指南

欢迎贡献新的示例或改进现有示例！

### 添加新示例

1. 创建 `your_saga.go` 和 `your_saga_test.go`
2. 遵循现有的代码结构和命名约定
3. 提供完整的文档（`docs/your_saga.md`）
4. 添加单元测试和集成测试
5. 更新本 README 文档

### 代码规范

- 遵循 Go 代码风格指南
- 添加清晰的注释
- 使用有意义的变量名
- 保持函数简洁
- 编写全面的测试

## 许可证

MIT License - 详见 [LICENSE](../../../LICENSE) 文件

## 联系方式

- Issue: https://github.com/innovationmech/swit/issues
- Email: dreamerlyj@gmail.com

---

**注意**: 这些示例仅用于演示目的。在生产环境中使用前，请根据实际业务需求进行适当的调整和优化。

