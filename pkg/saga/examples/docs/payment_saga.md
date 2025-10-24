# 支付处理 Saga 示例文档

## 概述

支付处理 Saga（`payment_saga.go`）展示了如何使用 Swit 框架处理跨账户资金转账的复杂分布式事务。这是一个对数据一致性和安全性要求极高的场景，需要严格的错误处理和补偿机制。

## 业务场景

### 适用场景

- 银行转账业务
- 支付平台的资金划转
- 电商平台的商家结算
- 跨境汇款
- 企业财务系统的资金管理

### 业务流程

```
发起转账 → 验证账户 → 冻结资金 → 执行转账 → 风险检测 → 记录审计 → 发送通知
```

**关键特点**：
- 强一致性要求
- 资金安全保障
- 完整的审计跟踪
- 风险控制机制

## 架构设计

### 服务依赖关系

```
┌─────────────────┐
│  Saga           │
│  Coordinator    │
└────────┬────────┘
         │
    ┌────┴──────────┬──────────────┬──────────────┬──────────────┐
    │               │              │              │              │
┌───▼──────┐  ┌────▼─────┐  ┌─────▼──────┐  ┌───▼──────┐  ┌───▼──────┐
│ 账户服务 │  │ 转账服务 │  │ 风控服务   │  │ 审计服务 │  │ 通知服务 │
│(Account) │  │(Transfer)│  │(Risk)      │  │(Audit)   │  │(Notify)  │
└──────────┘  └──────────┘  └────────────┘  └──────────┘  └──────────┘
```

### 步骤流程图

```
┌──────────────────────────────────────────────────────────┐
│                 支付处理 Saga                             │
└──────────────────────────────────────────────────────────┘

步骤 1: ValidateAccountsStep (验证账户)
  ├─ Execute: 验证源账户和目标账户的有效性
  └─ Compensate: 无需补偿

步骤 2: FreezeAmountStep (冻结资金)
  ├─ Execute: 冻结源账户中的转账金额
  └─ Compensate: 解冻资金

步骤 3: ExecuteTransferStep (执行转账)
  ├─ Execute: 从源账户扣款，向目标账户入账
  └─ Compensate: 回滚转账（从目标账户扣款，退回源账户）

步骤 4: RiskDetectionStep (风险检测)
  ├─ Execute: 执行风控检查
  └─ Compensate: 标记为风险事件

步骤 5: CreateAuditRecordStep (创建审计记录)
  ├─ Execute: 记录转账审计信息
  └─ Compensate: 更新审计状态为回滚

步骤 6: SendNotificationStep (发送通知)
  ├─ Execute: 发送转账成功通知
  └─ Compensate: 发送转账取消通知
```

### 数据流转

```
TransferData (输入)
    │
    ├─> ValidateAccountsStep
    │     └─> ValidateAccountsResult
    │           │
    │           ├─> FreezeAmountStep
    │           │     └─> FreezeResult
    │           │           │
    │           │           ├─> ExecuteTransferStep
    │           │           │     └─> TransferResult
    │           │           │           │
    │           │           │           ├─> RiskDetectionStep
    │           │           │           │     └─> RiskDetectionResult
    │           │           │           │           │
    │           │           │           │           ├─> CreateAuditRecordStep
    │           │           │           │           │     └─> AuditRecordResult
    │           │           │           │           │           │
    │           │           │           │           │           └─> SendNotificationStep
    │           │           │           │           │                 └─> NotificationResult (输出)
    │           │           │           │           │
    │           │           │           │           └─ (失败) ─> 补偿...
    │           │           │           │
    │           │           │           └─ (失败) ─> 补偿 RiskDetectionStep
    │           │           │
    │           │           └─ (失败) ─> 补偿 ExecuteTransferStep + FreezeAmountStep
    │           │
    │           └─ (失败) ─> 补偿 FreezeAmountStep
    │
    └─ (失败) ─> 无需补偿
```

## 数据结构

### 输入数据

#### TransferData

```go
type TransferData struct {
    // 转账基本信息
    TransferID  string      // 转账ID（如果已生成）
    FromAccount string      // 源账户ID
    ToAccount   string      // 目标账户ID
    Amount      float64     // 转账金额
    Currency    string      // 货币类型（USD、CNY等）
    
    // 转账说明
    Purpose     string      // 转账用途
    Description string      // 转账描述
    
    // 客户信息
    CustomerID   string     // 客户ID
    CustomerName string     // 客户姓名
    
    // 风控信息
    RiskLevel     string    // 风险等级（low、medium、high）
    RiskFactors   []string  // 风险因素列表
    RequireReview bool      // 是否需要人工审核
    
    // 元数据
    Metadata map[string]interface{} // 额外元数据
}
```

**重要字段**：
- `Amount`: 转账金额，必须大于0
- `FromAccount`/`ToAccount`: 必须是有效的账户ID
- `Currency`: 货币类型，用于汇率转换和验证
- `RiskLevel`: 风险等级，影响审核流程

### 步骤结果

#### AccountBalance

```go
type AccountBalance struct {
    AccountID      string    // 账户ID
    Available      float64   // 可用余额
    Frozen         float64   // 冻结金额
    Total          float64   // 总余额
    Currency       string    // 货币类型
    LastUpdated    time.Time // 最后更新时间
    MinimumBalance float64   // 最低余额要求
    OverdraftLimit float64   // 透支额度
}
```

#### ValidateAccountsResult

```go
type ValidateAccountsResult struct {
    TransferID      string          // 转账ID
    FromAccountInfo *AccountBalance // 源账户信息
    ToAccountInfo   *AccountBalance // 目标账户信息
    ValidationTime  time.Time       // 验证时间
    ValidationCode  string          // 验证码
    Metadata        map[string]interface{}
}
```

#### FreezeResult

```go
type FreezeResult struct {
    FreezeID   string    // 冻结ID
    AccountID  string    // 账户ID
    Amount     float64   // 冻结金额
    Currency   string    // 货币类型
    FrozenAt   time.Time // 冻结时间
    ExpiresAt  time.Time // 过期时间
    FreezeType string    // 冻结类型
    Metadata   map[string]interface{}
}
```

#### TransferResult

```go
type TransferResult struct {
    TransactionID   string    // 交易ID
    TransferID      string    // 转账ID
    FromAccount     string    // 源账户ID
    ToAccount       string    // 目标账户ID
    Amount          float64   // 转账金额
    Currency        string    // 货币类型
    Status          string    // 转账状态
    CompletedAt     time.Time // 完成时间
    TransactionHash string    // 交易哈希（防篡改）
    Metadata        map[string]interface{}
}
```

## 步骤详解

### 步骤 1: ValidateAccountsStep（验证账户）

#### 功能描述

验证源账户和目标账户的有效性，检查账户状态、余额和权限。

#### 验证规则

1. **账户存在性检查**：账户必须存在且处于激活状态
2. **余额检查**：源账户可用余额必须足够（考虑最低余额要求）
3. **账户类型检查**：账户类型是否支持转账操作
4. **货币类型检查**：账户货币类型是否匹配
5. **权限检查**：客户是否有权限操作源账户

#### 执行逻辑

```go
func (s *ValidateAccountsStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    transferData := data.(*TransferData)
    
    // 验证源账户
    fromAccount, err := s.service.ValidateAccount(ctx, transferData.FromAccount)
    if err != nil {
        return nil, fmt.Errorf("源账户验证失败: %w", err)
    }
    
    // 验证目标账户
    toAccount, err := s.service.ValidateAccount(ctx, transferData.ToAccount)
    if err != nil {
        return nil, fmt.Errorf("目标账户验证失败: %w", err)
    }
    
    // 检查余额充足性
    requiredBalance := transferData.Amount + fromAccount.MinimumBalance
    if fromAccount.Available < requiredBalance {
        return nil, &InsufficientBalanceError{
            Available: fromAccount.Available,
            Required:  requiredBalance,
        }
    }
    
    // 生成验证码（用于后续步骤验证）
    validationCode := generateValidationCode(transferData, fromAccount, toAccount)
    
    return &ValidateAccountsResult{
        TransferID:      generateTransferID(),
        FromAccountInfo: fromAccount,
        ToAccountInfo:   toAccount,
        ValidationTime:  time.Now(),
        ValidationCode:  validationCode,
    }, nil
}
```

#### 补偿逻辑

```go
func (s *ValidateAccountsStep) Compensate(ctx context.Context, data interface{}) error {
    // 验证步骤无需补偿，因为没有修改任何状态
    return nil
}
```

#### 配置参数

- **超时时间**: 5秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误可重试，验证失败不可重试

### 步骤 2: FreezeAmountStep（冻结资金）

#### 功能描述

在源账户中冻结转账金额，确保资金在转账完成前不会被使用。

#### 执行逻辑

```go
func (s *FreezeAmountStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    validateResult := data.(*ValidateAccountsResult)
    
    // 从元数据获取转账信息
    amount := validateResult.Metadata["amount"].(float64)
    fromAccount := validateResult.FromAccountInfo.AccountID
    
    // 调用账户服务冻结资金
    freezeResult, err := s.service.FreezeAmount(
        ctx,
        fromAccount,
        amount,
        fmt.Sprintf("transfer-%s", validateResult.TransferID),
    )
    if err != nil {
        return nil, fmt.Errorf("冻结资金失败: %w", err)
    }
    
    // 设置30分钟自动过期
    freezeResult.ExpiresAt = time.Now().Add(30 * time.Minute)
    
    return freezeResult, nil
}
```

#### 补偿逻辑

```go
func (s *FreezeAmountStep) Compensate(ctx context.Context, data interface{}) error {
    freezeResult := data.(*FreezeResult)
    
    // 解冻资金
    err := s.service.UnfreezeAmount(ctx, freezeResult.FreezeID, "saga_compensation")
    if err != nil {
        return fmt.Errorf("解冻资金失败: %w", err)
    }
    
    return nil
}
```

**重要提示**：
- 冻结操作必须是幂等的
- 设置合理的过期时间，防止资金永久冻结
- 补偿失败时应触发告警

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误和临时锁定可重试

### 步骤 3: ExecuteTransferStep（执行转账）

#### 功能描述

执行实际的资金转移：从源账户扣除已冻结的金额，向目标账户入账。

#### 执行逻辑

```go
func (s *ExecuteTransferStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    freezeResult := data.(*FreezeResult)
    
    // 从元数据获取转账信息
    transferID := freezeResult.Metadata["transfer_id"].(string)
    
    // 执行转账
    result, err := s.service.ExecuteTransfer(ctx, transferID, freezeResult.FreezeID)
    if err != nil {
        return nil, fmt.Errorf("执行转账失败: %w", err)
    }
    
    // 生成交易哈希（用于防篡改验证）
    result.TransactionHash = generateTransactionHash(result)
    
    return result, nil
}
```

#### 转账原子性保证

```go
// 在账户服务中，转账操作必须是事务性的
func (s *AccountService) ExecuteTransfer(ctx context.Context, transferID, freezeID string) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 1. 扣除源账户的冻结金额
    if err := s.deductFrozenAmount(tx, freezeID); err != nil {
        return err
    }
    
    // 2. 向目标账户入账
    if err := s.addAmount(tx, toAccount, amount, transferID); err != nil {
        return err
    }
    
    // 3. 提交事务
    return tx.Commit()
}
```

#### 补偿逻辑

```go
func (s *ExecuteTransferStep) Compensate(ctx context.Context, data interface{}) error {
    transferResult := data.(*TransferResult)
    
    // 执行反向转账
    // 注意：这是一个新的转账操作，而不是简单的回滚
    reverseTransfer := &TransferData{
        FromAccount: transferResult.ToAccount,   // 反向
        ToAccount:   transferResult.FromAccount, // 反向
        Amount:      transferResult.Amount,
        Currency:    transferResult.Currency,
        Purpose:     "compensation",
        Description: fmt.Sprintf("补偿原转账 %s", transferResult.TransferID),
    }
    
    _, err := s.service.CreateTransfer(ctx, reverseTransfer)
    if err != nil {
        // 补偿失败是严重问题，需要人工介入
        return fmt.Errorf("转账补偿失败，需要人工处理: %w", err)
    }
    
    return nil
}
```

**关键注意事项**：
- 转账操作必须在数据库事务中完成
- 生成交易哈希用于防篡改
- 补偿操作实际上是一个反向转账
- 补偿失败必须触发告警和人工介入

#### 配置参数

- **超时时间**: 30秒（转账操作可能较慢）
- **重试策略**: 指数退避，最多重试2次
- **可重试**: 非常保守，只有明确的网络超时才重试

### 步骤 4: RiskDetectionStep（风险检测）

#### 功能描述

对已完成的转账进行风险检测，识别可疑交易。

#### 检测规则

1. **金额异常检测**：是否超过用户日常转账金额
2. **频率检测**：短时间内多次转账
3. **地理位置检测**：异地登录转账
4. **黑名单检测**：目标账户是否在黑名单中
5. **行为模式检测**：是否符合用户历史行为模式

#### 执行逻辑

```go
func (s *RiskDetectionStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    transferResult := data.(*TransferResult)
    
    // 执行风险检测
    riskScore, riskFactors, err := s.service.DetectRisk(ctx, transferResult)
    if err != nil {
        return nil, fmt.Errorf("风险检测失败: %w", err)
    }
    
    // 判断风险等级
    riskLevel := calculateRiskLevel(riskScore)
    
    result := &RiskDetectionResult{
        TransferID:  transferResult.TransferID,
        RiskScore:   riskScore,
        RiskLevel:   riskLevel,
        RiskFactors: riskFactors,
        DetectedAt:  time.Now(),
    }
    
    // 高风险交易可能需要人工审核或直接拒绝
    if riskLevel == "high" {
        result.RequireReview = true
        // 可以选择在这里失败，触发补偿
        // return nil, &HighRiskError{TransferID: transferResult.TransferID}
    }
    
    return result, nil
}
```

#### 补偿逻辑

```go
func (s *RiskDetectionStep) Compensate(ctx context.Context, data interface{}) error {
    // 标记为风险事件
    result := data.(*RiskDetectionResult)
    return s.service.MarkAsRiskEvent(ctx, result.TransferID)
}
```

#### 配置参数

- **超时时间**: 15秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 检测服务暂时不可用时可重试

### 步骤 5: CreateAuditRecordStep（创建审计记录）

#### 功能描述

记录转账的完整审计信息，用于合规和问题排查。

#### 审计内容

- 转账ID和交易ID
- 源账户和目标账户
- 转账金额和货币
- 发起人信息
- 时间戳
- 风险评分
- 操作结果

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试5次（审计必须成功）
- **可重试**: 几乎所有错误都可重试

### 步骤 6: SendNotificationStep（发送通知）

#### 功能描述

向转账相关方发送通知，告知转账结果。

#### 通知渠道

- 短信通知
- 邮件通知
- App 推送
- 微信/钉钉通知

#### 配置参数

- **超时时间**: 20秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 通知失败可重试，但不应阻止 Saga 完成

## 使用示例

### 基本用法

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
    // 创建 Coordinator
    sagaCoordinator := setupCoordinator()
    defer sagaCoordinator.Close()
    
    // 创建服务客户端
    accountService := NewAccountService()
    transferService := NewTransferService()
    riskService := NewRiskService()
    auditService := NewAuditService()
    notificationService := NewNotificationService()
    
    // 创建 Saga 定义
    sagaDef := examples.NewPaymentProcessingSaga(
        accountService,
        transferService,
        riskService,
        auditService,
        notificationService,
    )
    
    // 准备转账数据
    transferData := &examples.TransferData{
        FromAccount:  "ACC-001",
        ToAccount:    "ACC-002",
        Amount:       1000.00,
        Currency:     "CNY",
        Purpose:      "salary",
        Description:  "工资转账",
        CustomerID:   "CUST-12345",
        CustomerName: "张三",
        RiskLevel:    "low",
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, transferData)
    if err != nil {
        log.Fatalf("启动 Saga 失败: %v", err)
    }
    
    log.Printf("转账 Saga 已启动，实例 ID: %s", instance.GetID())
    
    // 监控执行结果
    time.Sleep(10 * time.Second)
    
    finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())
    log.Printf("转账最终状态: %s", finalInstance.GetStatus())
}
```

### 处理余额不足

```go
transferData := &examples.TransferData{
    FromAccount: "ACC-001",
    ToAccount:   "ACC-002",
    Amount:      1000000.00, // 远超账户余额
    Currency:    "CNY",
}

instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, transferData)

// 监控结果
time.Sleep(5 * time.Second)
finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())

if finalInstance.GetStatus() == saga.StatusFailed {
    log.Println("转账失败：余额不足")
}
```

### 处理高风险交易

```go
transferData := &examples.TransferData{
    FromAccount:   "ACC-001",
    ToAccount:     "ACC-SUSPICIOUS", // 可疑账户
    Amount:        50000.00,         // 大额转账
    RiskLevel:     "high",
    RequireReview: true,
}

// 启动 Saga
instance, _ := sagaCoordinator.StartSaga(ctx, sagaDef, transferData)

// 如果配置了高风险自动拒绝，Saga 会在风险检测步骤失败并回滚
```

## 安全考虑

### 1. 防止重复转账

```go
// 使用幂等性密钥
transferData.Metadata["idempotency_key"] = generateIdempotencyKey(
    transferData.FromAccount,
    transferData.ToAccount,
    transferData.Amount,
    time.Now(),
)
```

### 2. 交易签名验证

```go
// 生成交易哈希
func generateTransactionHash(result *TransferResult) string {
    data := fmt.Sprintf("%s|%s|%s|%f|%s",
        result.TransferID,
        result.FromAccount,
        result.ToAccount,
        result.Amount,
        result.CompletedAt.Format(time.RFC3339),
    )
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

### 3. 敏感数据加密

```go
// 加密账户信息
encryptedData := encrypt(transferData, encryptionKey)
```

### 4. 审计日志

所有操作都应记录详细的审计日志：

```go
auditLog := &AuditLog{
    Timestamp:   time.Now(),
    Operation:   "transfer",
    UserID:      transferData.CustomerID,
    FromAccount: transferData.FromAccount,
    ToAccount:   transferData.ToAccount,
    Amount:      transferData.Amount,
    Status:      "success",
    IPAddress:   ctx.Value("ip_address").(string),
}
```

## 最佳实践

### 1. 金额精度处理

```go
// 使用整数表示金额（单位：分）
type Amount struct {
    Value    int64  // 金额值（分）
    Currency string // 货币类型
}

// 转换
func toAmount(value float64, currency string) Amount {
    return Amount{
        Value:    int64(value * 100),
        Currency: currency,
    }
}
```

### 2. 并发控制

```go
// 使用分布式锁防止并发转账
lockKey := fmt.Sprintf("account:lock:%s", accountID)
lock := acquireLock(lockKey, 30*time.Second)
defer lock.Release()
```

### 3. 补偿策略配置

```go
compensationStrategy := saga.NewSequentialCompensationStrategy(
    5 * time.Minute, // 补偿总超时
    saga.WithCompensationRetry(5, 2*time.Second),
    saga.WithCompensationAlertOnFailure(true),
    saga.WithManualInterventionOnFailure(true),
)
```

### 4. 监控告警

```go
// 配置关键指标监控
metrics := &PaymentMetrics{
    TransferDuration:      prometheus.NewHistogram(...),
    FailureRate:           prometheus.NewCounter(...),
    CompensationFailures:  prometheus.NewCounter(...),
    HighRiskTransactions:  prometheus.NewCounter(...),
}
```

## 测试

### 运行测试

```bash
# 运行支付处理测试
go test -v -run TestPaymentSaga

# 测试余额不足场景
go test -v -run TestPaymentSaga/InsufficientBalance

# 测试补偿场景
go test -v -run TestPaymentSaga/CompensationScenario
```

### 测试覆盖场景

1. ✅ 正常转账流程
2. ✅ 余额不足
3. ✅ 账户不存在
4. ✅ 冻结失败
5. ✅ 转账执行失败
6. ✅ 高风险交易
7. ✅ 补偿失败处理
8. ✅ 超时场景

## 生产环境配置

```yaml
saga:
  payment_processing:
    timeout: 3m
    
    retry:
      max_attempts: 3
      initial_interval: 2s
      max_interval: 30s
    
    compensation:
      timeout: 5m
      retry_attempts: 5
      alert_on_failure: true
      manual_intervention: true
    
    steps:
      validate_accounts:
        timeout: 5s
      freeze_amount:
        timeout: 10s
      execute_transfer:
        timeout: 30s
        retry_attempts: 2
      risk_detection:
        timeout: 15s
      create_audit_record:
        timeout: 10s
        retry_attempts: 5
      send_notification:
        timeout: 20s
        retry_attempts: 3
    
    # 告警配置
    alerts:
      - type: compensation_failure
        severity: critical
        channels: [email, sms, pagerduty]
      - type: high_risk_transaction
        severity: warning
        channels: [email]
```

## 常见问题

### Q1: 如何处理跨货币转账？

**A**: 在 ValidateAccountsStep 中添加汇率转换逻辑，并在 ExecuteTransferStep 中执行实际转换。

### Q2: 补偿失败后资金状态不一致怎么办？

**A**: 
1. 配置多次重试
2. 触发人工介入告警
3. 使用对账系统定期检查
4. 提供手动补偿工具

### Q3: 如何防止资金永久冻结？

**A**: 设置冻结过期时间，后台定期扫描并自动解冻过期的冻结记录。

## 相关资源

- [Saga 用户指南](../../../../docs/saga-user-guide.md)
- [架构设计文档](architecture.md)
- [订单处理 Saga](order_saga.md)
- [Saga 安全指南](../../../../docs/saga-security-guide.md)

