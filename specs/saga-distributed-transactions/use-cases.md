# Saga 分布式事务使用场景分析

## 概述

本文档详细分析了 Swit 框架中 Saga 分布式事务的典型使用场景，包括业务需求、技术实现方案和最佳实践建议。

## 核心使用场景

### 1. 电商订单处理

#### 场景描述
电商系统中最典型的分布式事务场景，涉及订单创建、库存扣减、支付处理、物流发货等多个服务。

#### 业务流程
```
用户下单 → 创建订单 → 扣减库存 → 处理支付 → 确认订单 → 发货通知
   ↓        ↓        ↓        ↓        ↓        ↓
失败补偿 ← 取消支付 ← 恢复库存 ← 取消订单 ← 通知失败
```

#### 技术实现

**Saga 定义示例**:
```go
type OrderProcessingSaga struct {
    orderService     OrderServiceClient
    inventoryService InventoryServiceClient
    paymentService   PaymentServiceClient
    shippingService  ShippingServiceClient
    notificationService NotificationServiceClient
}

func (ops *OrderProcessingSaga) GetSagaDefinition() *SagaDefinition {
    return &SagaDefinition{
        ID:   "order-processing",
        Name: "电商订单处理",
        Steps: []SagaStep{
            &CreateOrderStep{service: ops.orderService},
            &ReserveInventoryStep{service: ops.inventoryService},
            &ProcessPaymentStep{service: ops.paymentService},
            &ConfirmOrderStep{service: ops.orderService},
            &CreateShipmentStep{service: ops.shippingService},
            &SendNotificationStep{service: ops.notificationService},
        },
        Timeout: 30 * time.Minute,
        RetryPolicy: &ExponentialBackoffPolicy{
            InitialDelay: 1 * time.Second,
            MaxDelay:     30 * time.Second,
            MaxAttempts:  5,
        },
    }
}
```

**步骤实现示例**:
```go
// 创建订单步骤
type CreateOrderStep struct {
    service OrderServiceClient
}

func (cos *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    order, err := cos.service.CreateOrder(ctx, &CreateOrderRequest{
        CustomerID: orderData.CustomerID,
        Items:      orderData.Items,
        Total:      orderData.Total,
        Address:    orderData.Address,
    })
    if err != nil {
        return nil, fmt.Errorf("创建订单失败: %w", err)
    }
    
    // 将订单ID传递给下一步
    return &OrderStepResult{
        OrderID: order.ID,
        Order:   order,
    }, nil
}

func (cos *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*OrderStepResult)
    
    err := cos.service.CancelOrder(ctx, &CancelOrderRequest{
        OrderID: result.OrderID,
        Reason:  "saga_compensation",
    })
    if err != nil {
        // 补偿失败，记录日志并继续
        log.Printf("补偿取消订单失败: %v", err)
        return nil // 补偿失败不应该阻止整个补偿流程
    }
    
    return nil
}
```

#### 关键考虑因素

**库存处理策略**:
- 预扣库存 vs 实扣库存
- 超卖保护机制
- 库存释放策略

**支付处理**:
- 支付超时处理
- 支付失败重试
- 退款流程自动化

**异常处理**:
- 网络超时处理
- 服务不可用降级
- 数据不一致修复

---

### 2. 金融服务转账

#### 场景描述
银行或支付系统中的跨账户资金转账，需要确保资金转移的原子性和一致性。

#### 业务流程
```
发起转账 → 验证账户 → 冻结资金 → 执行转账 → 释放冻结 → 通知结果
    ↓        ↓        ↓        ↓        ↓        ↓
回滚操作 ← 解冻资金 ← 冻结失败 ← 验证失败 ← 通知失败
```

#### 技术实现

**资金转账 Saga**:
```go
type FundTransferSaga struct {
    accountService   AccountServiceClient
    transferService  TransferServiceClient
    auditService     AuditServiceClient
    notificationService NotificationServiceClient
}

func (fts *FundTransferSaga) GetSagaDefinition() *SagaDefinition {
    return &SagaDefinition{
        ID:   "fund-transfer",
        Name: "资金转账",
        Steps: []SagaStep{
            &ValidateAccountsStep{service: fts.accountService},
            &FreezeAmountStep{service: fts.accountService},
            &ExecuteTransferStep{service: fts.transferService},
            &ReleaseFreezeStep{service: fts.accountService},
            &CreateAuditRecordStep{service: fts.auditService},
            &SendTransferNotificationStep{service: fts.notificationService},
        },
        Timeout: 10 * time.Minute, // 金融操作需要较短超时
        RetryPolicy: &LinearBackoffPolicy{
            InitialDelay: 500 * time.Millisecond,
            MaxDelay:     5 * time.Second,
            MaxAttempts:  3, // 金融操作重试次数较少
        },
    }
}
```

**资金冻结步骤**:
```go
type FreezeAmountStep struct {
    service AccountServiceClient
}

func (fas *FreezeAmountStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    transferData := data.(*TransferData)
    
    // 验证账户余额
    balance, err := fas.service.GetBalance(ctx, transferData.FromAccount)
    if err != nil {
        return nil, fmt.Errorf("获取账户余额失败: %w", err)
    }
    
    if balance.Available < transferData.Amount {
        return nil, fmt.Errorf("账户余额不足")
    }
    
    // 冻结资金
    freezeID, err := fas.service.FreezeAmount(ctx, &FreezeRequest{
        AccountID: transferData.FromAccount,
        Amount:    transferData.Amount,
        Reason:    fmt.Sprintf("转账到账户 %s", transferData.ToAccount),
    })
    if err != nil {
        return nil, fmt.Errorf("冻结资金失败: %w", err)
    }
    
    return &FreezeStepResult{
        FreezeID: freezeID,
        Amount:   transferData.Amount,
    }, nil
}

func (fas *FreezeAmountStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*FreezeStepResult)
    
    // 解冻资金
    err := fas.service.UnfreezeAmount(ctx, &UnfreezeRequest{
        FreezeID: result.FreezeID,
        Reason:   "saga_compensation",
    })
    if err != nil {
        log.Printf("解冻资金失败: %v", err)
        // 资金解冻失败需要人工介入
        return fmt.Errorf("资金解冻失败，需要人工处理: %w", err)
    }
    
    return nil
}
```

#### 关键考虑因素

**金融安全**:
- 幂等性保证
- 资金安全优先
- 审计跟踪完整性

**性能要求**:
- 低延迟处理
- 高并发支持
- 实时状态查询

**合规要求**:
- 监管数据记录
- 反洗钱检查
- 交易限制控制

---

### 3. 旅行预订系统

#### 场景描述
在线旅行预订涉及航班预订、酒店预订、租车服务等多个独立服务，需要确保整体预订的一致性。

#### 业务流程
```
查询产品 → 预订航班 → 预订酒店 → 预订租车 → 确认支付 → 生成订单
    ↓        ↓        ↓        ↓        ↓        ↓
取消操作 ← 取消租车 ← 取消酒店 ← 取消航班 ← 支付失败
```

#### 技术实现

**旅行预订 Saga**:
```go
type TravelBookingSaga struct {
    flightService    FlightServiceClient
    hotelService     HotelServiceClient
    carService       CarRentalServiceClient
    paymentService   PaymentServiceClient
    bookingService   BookingServiceClient
}

func (tbs *TravelBookingSaga) GetSagaDefinition() *SagaDefinition {
    return &SagaDefinition{
        ID:   "travel-booking",
        Name: "旅行预订",
        Steps: []SagaStep{
            &BookFlightStep{service: tbs.flightService},
            &BookHotelStep{service: tbs.hotelService},
            &BookCarRentalStep{service: tbs.carService},
            &ProcessPaymentStep{service: tbs.paymentService},
            &CreateBookingStep{service: tbs.bookingService},
        },
        Timeout: 20 * time.Minute,
        RetryPolicy: &ExponentialBackoffPolicy{
            InitialDelay: 2 * time.Second,
            MaxDelay:     60 * time.Second,
            MaxAttempts:  3,
        },
        CompensationStrategy: &ParallelCompensationStrategy{}, // 并行补偿以提高效率
    }
}
```

**航班预订步骤**:
```go
type BookFlightStep struct {
    service FlightServiceClient
}

func (bfs *BookFlightStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    bookingData := data.(*TravelBookingData)
    
    // 检查航班可用性
    availability, err := bfs.service.CheckAvailability(ctx, &AvailabilityRequest{
        FlightID:    bookingData.FlightID,
        Date:        bookingData.TravelDate,
        Passengers:  len(bookingData.Passengers),
    })
    if err != nil {
        return nil, fmt.Errorf("检查航班可用性失败: %w", err)
    }
    
    if !availability.Available {
        return nil, fmt.Errorf("航班不可用")
    }
    
    // 预订航班
    booking, err := bfs.service.BookFlight(ctx, &BookFlightRequest{
        FlightID:    bookingData.FlightID,
        Date:        bookingData.TravelDate,
        Passengers:  bookingData.Passengers,
        HoldTime:    15 * time.Minute, // 临时预订15分钟
    })
    if err != nil {
        return nil, fmt.Errorf("预订航班失败: %w", err)
    }
    
    return &FlightBookingResult{
        BookingID:   booking.ID,
        Confirmation: booking.ConfirmationCode,
        Price:       booking.TotalPrice,
    }, nil
}

func (bfs *BookFlightStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*FlightBookingResult)
    
    // 取消航班预订
    err := bfs.service.CancelFlight(ctx, &CancelFlightRequest{
        BookingID: result.BookingID,
        Reason:    "saga_compensation",
    })
    if err != nil {
        log.Printf("取消航班预订失败: %v", err)
        return nil // 航班预订通常可以自动过期
    }
    
    return nil
}
```

#### 关键考虑因素

**库存管理**:
- 临时预订机制
- 超售控制
- 库存释放策略

**用户体验**:
- 实时价格查询
- 预订状态通知
- 灵活的取消政策

**供应商集成**:
- 多供应商支持
- 标准化接口
- 错误处理差异

---

### 4. 用户注册和初始化

#### 场景描述
新用户注册涉及创建用户账户、发送验证邮件、初始化用户配置、分配初始资源等多个步骤。

#### 业务流程
```
注册信息 → 创建账户 → 发送验证邮件 → 初始化配置 → 分配资源 → 完成注册
    ↓        ↓         ↓          ↓        ↓        ↓
清理操作 ← 释放资源 ← 清理配置 ← 删除账户 ← 邮件失败
```

#### 技术实现

**用户注册 Saga**:
```go
type UserRegistrationSaga struct {
    userService        UserServiceClient
    emailService       EmailServiceClient
    configService      ConfigServiceClient
    resourceService    ResourceServiceClient
    analyticsService   AnalyticsServiceClient
}

func (urs *UserRegistrationSaga) GetSagaDefinition() *SagaDefinition {
    return &SagaDefinition{
        ID:   "user-registration",
        Name: "用户注册",
        Steps: []SagaStep{
            &CreateUserAccountStep{service: urs.userService},
            &SendVerificationEmailStep{service: urs.emailService},
            &InitializeUserConfigStep{service: urs.configService},
            &AllocateResourcesStep{service: urs.resourceService},
            &TrackRegistrationEventStep{service: urs.analyticsService},
        },
        Timeout: 10 * time.Minute,
        RetryPolicy: &ExponentialBackoffPolicy{
            InitialDelay: 1 * time.Second,
            MaxDelay:     30 * time.Second,
            MaxAttempts:  3,
        },
    }
}
```

**账户创建步骤**:
```go
type CreateUserAccountStep struct {
    service UserServiceClient
}

func (cuas *CreateUserAccountStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    registrationData := data.(*UserRegistrationData)
    
    // 验证用户输入
    if err := validateRegistrationData(registrationData); err != nil {
        return nil, fmt.Errorf("用户数据验证失败: %w", err)
    }
    
    // 检查用户是否已存在
    existing, err := cuas.service.GetUserByEmail(ctx, registrationData.Email)
    if err != nil && !errors.Is(err, ErrUserNotFound) {
        return nil, fmt.Errorf("检查用户是否存在失败: %w", err)
    }
    if existing != nil {
        return nil, fmt.Errorf("用户已存在")
    }
    
    // 创建用户账户
    user, err := cuas.service.CreateUser(ctx, &CreateUserRequest{
        Email:     registrationData.Email,
        Username:  registrationData.Username,
        Password:  registrationData.Password,
        FirstName: registrationData.FirstName,
        LastName:  registrationData.LastName,
        Status:    UserStatusPending, // 待验证状态
    })
    if err != nil {
        return nil, fmt.Errorf("创建用户账户失败: %w", err)
    }
    
    return &UserCreationResult{
        UserID:   user.ID,
        Username: user.Username,
        Email:    user.Email,
    }, nil
}

func (cuas *CreateUserAccountStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*UserCreationResult)
    
    // 删除用户账户
    err := cuas.service.DeleteUser(ctx, &DeleteUserRequest{
        UserID: result.UserID,
        Reason: "registration_failed",
    })
    if err != nil {
        log.Printf("删除用户账户失败: %v", err)
        // 用户删除失败可能需要数据清理
        return fmt.Errorf("用户账户删除失败，需要手动清理: %w", err)
    }
    
    return nil
}
```

#### 关键考虑因素

**数据隐私**:
- GDPR 合规
- 数据清理策略
- 隐私保护措施

**安全性**:
- 密码强度验证
- 邮箱验证机制
- 防垃圾注册

**用户体验**:
- 快速注册流程
- 清晰的状态反馈
- 友好的错误提示

---

## 高级使用场景

### 5. 供应链管理

#### 场景描述
制造业供应链中的采购、生产、库存、物流等环节的协调管理。

#### 技术特点
- 长时间运行的事务
- 复杂的依赖关系
- 多种补偿策略
- 实时状态跟踪

### 6. 医疗预约系统

#### 场景描述
医院或诊所的预约系统，涉及医生排班、科室资源、患者记录等多个方面。

#### 技术特点
- 严格的并发控制
- 实时资源状态
- 复杂的调度算法
- 高可靠性要求

### 7. 教育课程注册

#### 场景描述
在线教育平台的课程注册，涉及课程容量、付费、学习资源分配等。

#### 技术特点
- 容量管理
- 学习进度跟踪
- 资源动态分配
- 灵活的退课策略

## 最佳实践建议

### 1. Saga 设计原则

**步骤设计**:
- 每个步骤应该是幂等的
- 补偿操作必须能够撤销原操作
- 步骤间的依赖关系应该明确
- 避免长时间阻塞操作

**错误处理**:
- 区分可重试和不可重试的错误
- 实现合适的重试策略
- 提供手动干预机制
- 记录详细的错误信息

**性能优化**:
- 使用异步处理提高并发性
- 实现批量操作减少开销
- 优化状态存储查询
- 监控关键性能指标

### 2. 补偿策略

**补偿时机**:
- 立即补偿 vs 延迟补偿
- 批量补偿 vs 单个补偿
- 自动补偿 vs 手动补偿
- 部分补偿 vs 完全补偿

**补偿实现**:
- 补偿操作必须是幂等的
- 补偿失败的处理策略
- 补偿超时的处理机制
- 补偿结果的验证

### 3. 监控和运维

**关键指标**:
- Saga 成功率
- 平均执行时间
- 补偿成功率
- 系统资源使用

**告警规则**:
- Saga 失败率过高
- 执行时间异常
- 补偿失败率过高
- 系统资源不足

**故障处理**:
- 自动恢复机制
- 手动干预流程
- 故障排查工具
- 数据修复脚本

### 4. 测试策略

**单元测试**:
- 每个步骤的独立测试
- 补偿操作的测试
- 错误场景的测试
- 边界条件的测试

**集成测试**:
- 端到端流程测试
- 多服务协作测试
- 故障恢复测试
- 性能压力测试

**混沌测试**:
- 网络分区模拟
- 服务故障模拟
- 数据不一致测试
- 高并发场景测试

## 技术选型建议

### 1. 存储选择

**Redis**:
- 适合高并发、低延迟场景
- 数据结构灵活
- 支持集群部署
- 内存成本较高

**PostgreSQL**:
- 适合强一致性要求
- 丰富的查询功能
- 成熟的生态系统
- 扩展性相对有限

**混合存储**:
- Redis 作为缓存层
- PostgreSQL 作为持久层
- 根据数据特性选择合适存储

### 2. 消息系统选择

**Kafka**:
- 高吞吐量
- 持久化存储
- 分布式架构
- 学习成本较高

**RabbitMQ**:
- 功能丰富
- 易于使用
- 灵活的路由
- 性能相对较低

**NATS**:
- 轻量级
- 高性能
- 简单部署
- 功能相对简单

### 3. 监控工具选择

**Prometheus**:
- 时间序列数据库
- 强大的查询语言
- 丰富的生态系统
- 配置相对复杂

**Jaeger**:
- 分布式追踪
- 与 OpenTelemetry 兼容
- 可视化界面
- 存储成本较高

**Grafana**:
- 数据可视化
- 多数据源支持
- 丰富的面板
- 学习成本适中

---

**文档版本**: 1.0  
**创建日期**: 2025-01-01  
**最后更新**: 2025-01-01  
**负责人**: Swit 开发团队