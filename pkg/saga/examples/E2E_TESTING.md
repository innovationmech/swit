# Saga 示例应用端到端测试文档

本文档描述了 Saga 示例应用的端到端（E2E）测试框架和使用方法。

## 概述

E2E 测试用于验证完整的 Saga 业务流程，包括：

- **订单处理 Saga**：电商订单的完整流程（创建订单→预留库存→处理支付→确认订单）
- **支付处理 Saga**：跨账户资金转账（验证账户→冻结资金→转账→解冻→审计→通知）
- **库存管理 Saga**：多仓库库存协调（检查库存→预留→分配→释放→审计→通知）
- **用户注册 Saga**：新用户注册和初始化（创建用户→发送验证邮件→初始化配置→分配资源→跟踪事件）

## 目录结构

```
pkg/saga/examples/
├── e2e_test.go              # E2E 测试主文件
├── order_saga_test.go       # 订单处理单元测试（包含 Mock 服务）
├── payment_saga_test.go     # 支付处理单元测试（包含 Mock 服务）
├── inventory_saga_test.go   # 库存管理单元测试（包含 Mock 服务）
├── user_registration_saga_test.go  # 用户注册单元测试（包含 Mock 服务）
└── E2E_TESTING.md          # 本文档
```

## 快速开始

### 运行所有 E2E 测试

```bash
cd pkg/saga/examples
go test -v -run E2E
```

### 运行特定 Saga 的 E2E 测试

```bash
# 订单处理 Saga
go test -v -run TestOrderProcessingSaga_E2E

# 支付处理 Saga
go test -v -run TestPaymentProcessingSaga_E2E

# 库存管理 Saga
go test -v -run TestInventoryManagementSaga_E2E

# 用户注册 Saga
go test -v -run TestUserRegistrationSaga_E2E
```

### 运行性能基准测试

```bash
go test -bench=BenchmarkOrderProcessingSaga_E2E -benchmem
go test -bench=BenchmarkPaymentProcessingSaga_E2E -benchmem
```

## 测试场景

### 1. 订单处理 Saga 测试场景

#### 正常流程（Happy Path）
- **测试**：`TestOrderProcessingSaga_E2E_HappyPath`
- **场景**：完整的订单处理流程成功执行
- **验证**：
  - 所有步骤按顺序执行
  - 订单状态变为 `confirmed`
  - 没有执行补偿步骤

#### 库存不足场景
- **测试**：`TestOrderProcessingSaga_E2E_InventoryFailure`
- **场景**：库存预留步骤失败
- **验证**：
  - 订单被取消（补偿）
  - 支付步骤未执行
  - 补偿步骤正确执行

#### 支付失败场景
- **测试**：`TestOrderProcessingSaga_E2E_PaymentFailure`
- **场景**：支付处理步骤失败
- **验证**：
  - 库存被释放（补偿）
  - 订单被取消（补偿）
  - 确认订单步骤未执行

#### 超时场景
- **测试**：`TestOrderProcessingSaga_E2E_Timeout`
- **场景**：步骤执行超时
- **验证**：
  - Saga 因超时失败
  - 已执行步骤被补偿

### 2. 支付处理 Saga 测试场景

#### 正常流程
- **测试**：`TestPaymentProcessingSaga_E2E_HappyPath`
- **场景**：完整的转账流程成功执行
- **验证**：
  - 所有步骤执行成功
  - 通知状态为 `sent`
  - 审计记录已创建

#### 余额不足场景
- **测试**：`TestPaymentProcessingSaga_E2E_InsufficientBalance`
- **场景**：账户余额不足
- **验证**：
  - 冻结步骤失败
  - 转账步骤未执行

#### 转账失败场景
- **测试**：`TestPaymentProcessingSaga_E2E_TransferFailure`
- **场景**：转账执行失败
- **验证**：
  - 冻结金额被解冻（补偿）

### 3. 库存管理 Saga 测试场景

#### 正常流程
- **测试**：`TestInventoryManagementSaga_E2E_HappyPath`
- **场景**：多仓库库存协调成功
- **验证**：
  - 库存检查、预留、分配、释放全部成功
  - 审计记录和通知发送成功

#### 库存不足场景
- **测试**：`TestInventoryManagementSaga_E2E_InsufficientInventory`
- **场景**：所有仓库库存总和不足
- **验证**：
  - 检查步骤失败
  - 后续步骤未执行

#### 分配失败场景
- **测试**：`TestInventoryManagementSaga_E2E_AllocationFailure`
- **场景**：库存分配失败
- **验证**：
  - 预留被释放（补偿）

### 4. 用户注册 Saga 测试场景

#### 正常流程
- **测试**：`TestUserRegistrationSaga_E2E_HappyPath`
- **场景**：用户注册全流程成功
- **验证**：
  - 用户创建、邮件发送、配置初始化、资源分配、事件跟踪全部成功
  - 注册事件已记录

#### 邮箱已存在场景
- **测试**：`TestUserRegistrationSaga_E2E_EmailExists`
- **场景**：邮箱已被注册
- **验证**：
  - 用户创建步骤失败
  - 后续步骤未执行

#### 邮件发送失败场景
- **测试**：`TestUserRegistrationSaga_E2E_EmailSendFailure`
- **场景**：验证邮件发送失败
- **验证**：
  - 用户被删除（补偿）

#### 资源分配失败场景
- **测试**：`TestUserRegistrationSaga_E2E_ResourceAllocationFailure`
- **场景**：资源分配失败
- **验证**：
  - 配置被删除（补偿）
  - 邮件验证被取消（补偿）
  - 用户被删除（补偿）

### 5. 并发测试
- **测试**：`TestMultipleSagas_E2E_Concurrent`
- **场景**：多个 Saga 并发执行
- **验证**：
  - 所有 Saga 正确执行
  - 无竞态条件

## 测试实现说明

E2E 测试使用各示例文件中已有的 Mock 服务（如 `mockOrderService`、`mockPaymentService` 等），这些 Mock 服务在对应的单元测试文件中定义。

### E2E 测试执行流程

1. 创建 Saga 定义和模拟服务
2. 验证 Saga 定义的有效性
3. 按顺序执行所有 Saga 步骤
4. 验证每个步骤的结果
5. 在失败场景中测试补偿逻辑

### 示例代码

```go
// 创建模拟服务（使用已有的 mock）
orderService := &mockOrderService{}
inventoryService := &mockInventoryService{}
paymentService := &mockPaymentService{}

// 创建 Saga 定义
sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

// 执行所有步骤
var currentData interface{} = orderData
for _, step := range sagaDef.GetSteps() {
    result, err := step.Execute(ctx, currentData)
    if err != nil {
        // 处理错误并执行补偿
    }
    currentData = result
}
```

## Mock 服务

### 订单服务 (MockOrderService)

```go
orderService := testutil.NewMockOrderService()

// 设置延迟
orderService.SetDelay(100 * time.Millisecond)

// 模拟错误
orderService.SetCreateOrderError(errors.New("服务不可用"))
```

### 库存服务 (MockInventoryService)

```go
inventoryService := testutil.NewMockInventoryService()

// 模拟库存不足错误
inventoryService.SetReserveInventoryError(errors.New("库存不足"))
```

### 支付服务 (MockPaymentService)

```go
paymentService := testutil.NewMockPaymentService()

// 模拟支付失败
paymentService.SetProcessPaymentError(errors.New("余额不足"))
```

### 账户服务 (MockAccountService)

```go
accountService := testutil.NewMockAccountService()

// 设置账户余额
accountService.SetBalance("ACC-001", 1000.00)
```

### 用户服务 (MockUserService)

```go
userService := testutil.NewMockUserService()

// 添加已存在的邮箱
userService.AddExistingEmail("existing@example.com")

// 添加已存在的用户名
userService.AddExistingUsername("existinguser")
```

## 辅助函数

`testutil` 包提供了多个辅助函数：

```go
// 检查切片是否包含元素
testutil.Contains(slice, "item")

// 检查字符串是否包含子串
testutil.ContainsString(str, "substr")

// 格式化字符串
testutil.FormatString("Order-%d", 123)
```

## 测试最佳实践

### 1. 独立性

每个测试应该是独立的，不依赖其他测试的状态：

```go
func TestSaga_E2E_Scenario(t *testing.T) {
    // 每个测试创建自己的执行器和服务
    executor := testutil.NewTestSagaExecutor()
    defer executor.Close()

    orderService := testutil.NewMockOrderService()
    // ...
}
```

### 2. 清理资源

使用 `defer` 确保资源被正确清理：

```go
executor := testutil.NewTestSagaExecutor()
defer executor.Close()
```

### 3. 明确的验证

测试应该明确验证所有关键行为：

```go
// 验证结果类型
confirmResult, ok := result.(*ConfirmStepResult)
if !ok {
    t.Fatalf("期望 *ConfirmStepResult，实际 %T", result)
}

// 验证具体字段
if confirmResult.Status != "confirmed" {
    t.Errorf("期望状态 confirmed，实际 %s", confirmResult.Status)
}

// 验证步骤执行
executedSteps := executor.GetExecutedSteps()
if !testutil.Contains(executedSteps, "create-order") {
    t.Error("create-order 步骤应该被执行")
}
```

### 4. 测试失败场景

不仅测试成功路径，还要测试各种失败场景：

```go
// 测试库存不足
inventoryService.SetReserveInventoryError(errors.New("库存不足"))

// 测试支付失败
paymentService.SetProcessPaymentError(errors.New("余额不足"))

// 测试超时
executor.SetTimeout(100 * time.Millisecond)
orderService.SetDelay(500 * time.Millisecond)
```

### 5. 并发安全

确保测试在并发环境下也能正常工作：

```go
func TestMultipleSagas_E2E_Concurrent(t *testing.T) {
    const concurrentSagas = 10
    results := make(chan error, concurrentSagas)

    for i := 0; i < concurrentSagas; i++ {
        go func(index int) {
            executor := testutil.NewTestSagaExecutor()
            defer executor.Close()
            
            // 执行 Saga...
            results <- err
        }(i)
    }

    // 收集结果...
}
```

## 覆盖率目标

E2E 测试的覆盖率目标：

- **步骤执行**：100% 步骤执行路径覆盖
- **补偿逻辑**：100% 补偿路径覆盖
- **错误处理**：覆盖所有主要错误场景
- **边界条件**：超时、并发、资源不足等

## 运行测试并查看覆盖率

```bash
# 生成覆盖率报告
cd pkg/saga/examples
go test -v -run E2E -coverprofile=coverage_e2e.out
go tool cover -html=coverage_e2e.out -o coverage_e2e.html

# 查看覆盖率统计
go tool cover -func=coverage_e2e.out
```

## 性能基准

E2E 测试包含性能基准测试：

```bash
# 运行基准测试
go test -bench=. -benchmem -run=^$

# 示例输出
BenchmarkOrderProcessingSaga_E2E-8        100    10523478 ns/op    2345 B/op    45 allocs/op
BenchmarkPaymentProcessingSaga_E2E-8       50    23451289 ns/op    5678 B/op    89 allocs/op
```

## 常见问题

### 1. 测试超时

**问题**：测试执行超时

**解决**：
```go
// 增加超时时间
executor.SetTimeout(10 * time.Minute)

// 或在测试中设置
go test -timeout 30m
```

### 2. 并发测试失败

**问题**：并发测试时出现竞态条件

**解决**：
```bash
# 使用竞态检测器运行测试
go test -race -run TestMultipleSagas_E2E_Concurrent
```

### 3. Mock 服务行为不符合预期

**问题**：Mock 服务没有返回预期的错误或数据

**解决**：
```go
// 确保在执行 Saga 之前设置 Mock 行为
inventoryService := testutil.NewMockInventoryService()
inventoryService.SetReserveInventoryError(errors.New("库存不足"))

// 然后创建 Saga 定义
sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)
```

## 扩展测试

### 添加新的测试场景

1. 在 `e2e_test.go` 中添加新的测试函数
2. 使用 `testutil` 中的执行器和 Mock 服务
3. 验证预期行为

```go
func TestOrderProcessingSaga_E2E_NewScenario(t *testing.T) {
    executor := testutil.NewTestSagaExecutor()
    defer executor.Close()

    // 设置场景...
    // 执行 Saga...
    // 验证结果...
}
```

### 添加新的 Mock 服务

1. 在 `testutil/` 目录下创建新的 Mock 服务文件
2. 实现服务接口
3. 提供错误注入和状态查询方法

```go
type MockNewService struct {
    mu    sync.Mutex
    error error
}

func NewMockNewService() *MockNewService {
    return &MockNewService{}
}

func (m *MockNewService) SetError(err error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.error = err
}

func (m *MockNewService) SomeOperation(ctx context.Context, data interface{}) (interface{}, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.error != nil {
        return nil, m.error
    }

    // 正常逻辑...
    return result, nil
}
```

## 参考资料

- [Saga 模式文档](../../docs/saga-user-guide.md)
- [单元测试文档](../README.md)
- [Go 测试最佳实践](https://go.dev/doc/tutorial/add-a-test)

## 贡献

欢迎贡献新的测试场景和改进。请确保：

1. 测试能够独立运行
2. 包含清晰的注释
3. 遵循现有的命名约定
4. 更新本文档以反映新增的测试

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 Issue: [GitHub Issues](https://github.com/innovationmech/swit/issues)
- 邮件: dreamerlyj@gmail.com

