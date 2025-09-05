# Jaeger UI 演示场景和示例数据

本文档提供了 Jaeger UI 的实际演示场景和示例数据，配合 [Jaeger UI 使用指南](jaeger-ui-guide.md) 使用。

## 前置条件

确保您已经按照 [分布式追踪示例](../examples/distributed-tracing/README.md) 设置好了环境：

```bash
cd examples/distributed-tracing
./scripts/setup.sh
```

## 演示场景

### 场景1: 正常业务流程追踪

**目标**: 演示完整的订单创建流程

**执行步骤**:
1. 启动演示环境
2. 创建正常订单
3. 在 Jaeger UI 中查看追踪

**演示命令**:
```bash
# 启动服务
./scripts/start.sh

# 创建正常订单
curl -X POST http://localhost:8081/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "CUST-001",
    "product_id": "PROD-001", 
    "quantity": 2,
    "amount": 199.99
  }'
```

**Jaeger UI 查询**:
- Service: `order-service`
- Operation: `POST /api/orders`
- Time Range: `Last 5 minutes`

**预期结果**:
- 完整的端到端追踪链
- 包含订单、库存、支付服务调用
- 总耗时约 200-500ms
- 无错误标记

### 场景2: 错误处理演示

**目标**: 演示支付失败的处理过程

**执行步骤**:
```bash
# 模拟支付失败
curl -X POST http://localhost:8081/api/orders \
  -H "Content-Type: application/json" \
  -H "X-Simulate-Payment-Failure: true" \
  -d '{
    "customer_id": "CUST-002",
    "product_id": "PROD-002",
    "quantity": 1,
    "amount": 99.99
  }'
```

**Jaeger UI 查询**:
- Service: `order-service`
- Tags: `error:true`
- Time Range: `Last 5 minutes`

**预期结果**:
- 追踪显示错误标记（红色）
- 支付服务返回失败状态
- 包含错误详细信息和堆栈跟踪
- 显示补偿事务执行

### 场景3: 性能瓶颈分析

**目标**: 演示慢查询对系统性能的影响

**执行步骤**:
```bash
# 模拟慢查询
curl -X POST http://localhost:8081/api/orders \
  -H "Content-Type: application/json"  
  -H "X-Simulate-Slow-Database: true" \
  -d '{
    "customer_id": "CUST-003",
    "product_id": "PROD-003",
    "quantity": 3,
    "amount": 299.97
  }'
```

**Jaeger UI 查询**:
- Service: `order-service`
- Tags: `duration:>1s`
- Time Range: `Last 5 minutes`

**预期结果**:
- 追踪显示明显的性能问题
- 数据库操作占用大部分时间
- 时间线视图清晰显示瓶颈点

### 场景4: 高并发场景

**目标**: 模拟系统在高负载下的行为

**执行脚本**:
```bash
# 运行负载测试
./scripts/load-test.sh -c 10 -r 100
```

**Jaeger UI 分析**:
- Service: `All Services`
- Time Range: `Last 15 minutes`
- 查看系统整体性能表现

## 实际示例数据

### 示例1: 成功订单追踪数据

**Trace ID**: `1a2b3c4d5e6f7890`

**Span 结构**:
```sql
order-service: POST /api/orders (480ms)
├── inventory-service: check-inventory (120ms)
│   └── database: SELECT inventory (80ms)
├── payment-service: process-payment (280ms) 
│   ├── external-api: payment-gateway (200ms)
│   └── database: UPDATE payment (50ms)
└── database: INSERT order (80ms)
```

**关键标签**:
- `order.id`: "ORD-20241205-001"
- `customer.id`: "CUST-001"
- `product.id`: "PROD-001"
- `http.status_code`: 201

### 示例2: 错误追踪数据

**Trace ID**: `2b3c4d5e6f7g8901`

**错误信息**:
```text
payment-service: process-payment (150ms) [ERROR]
├── external-api: payment-gateway (100ms) [ERROR]
│   └── error.message: "Card declined"
│   └── error.code: "CARD_DECLINED"
└── database: ROLLBACK transaction (30ms)
```

**错误标签**:
- `error`: true
- `error.kind`: "payment_declined"
- `http.status_code`: 402
- `payment.failure_reason`: "insufficient_funds"

### 示例3: 性能问题数据

**Trace ID**: `3c4d5e6f7g8h9012`

**性能分析**:
```sql
order-service: POST /api/orders (2.8s) [SLOW]
└── database: complex-query (2.5s) [BOTTLENECK]
    ├── db.statement: "SELECT * FROM orders JOIN customers..."
    ├── db.rows_affected: 15420
    └── performance.issue: "missing_index"
```

## 常见查询示例

### 1. 业务查询

```bash
# 查看特定用户的所有操作
user.id:CUST-001

# 查看特定订单的处理流程
order.id:ORD-20241205-001

# 查看特定商品的相关操作
product.id:PROD-001
```

### 2. 技术查询

```bash
# 查看数据库相关操作
component:database

# 查看外部API调用
component:http-client

# 查看缓存相关操作
component:redis
```

### 3. 性能查询

```bash
# 查看慢操作
duration:>1s

# 查看特定服务的性能
service:payment-service duration:>500ms

# 查看高内存使用
memory.usage:>100MB
```

### 4. 错误查询

```bash
# 查看所有错误
error:true

# 查看特定类型错误
error.kind:timeout

# 查看HTTP错误
http.status_code:>=400
```

## UI 操作演示

### 搜索界面

1. **服务选择**: 下拉菜单显示所有可用服务
2. **操作选择**: 根据所选服务显示相关操作
3. **时间选择**: 预设和自定义时间范围
4. **标签过滤**: 支持键值对和复杂查询

### 追踪详情界面

1. **时间线视图**: 水平条显示各 Span 执行时间
2. **标签面板**: 显示所有相关标签和元数据
3. **日志面板**: 显示 Span 内的结构化日志
4. **进程信息**: 显示服务实例信息

### 比较功能

1. **多追踪比较**: 选择 2-4 个追踪进行对比
2. **差异高亮**: 自动标记时间和结构差异
3. **并排显示**: 方便直观对比分析

## 故障排查演练

### 演练1: API 响应时间突然增加

**症状**: 用户反馈系统响应变慢

**排查步骤**:
1. 查询最近15分钟的所有追踪
2. 按响应时间排序，查看最慢的请求
3. 分析慢请求的共同特征
4. 定位具体的性能瓶颈

**Jaeger 查询**:
```bash
Service: api-gateway
Time: Last 15 minutes
Sort: Duration (Desc)
```

### 演练2: 间歇性服务错误

**症状**: 偶尔出现服务调用失败

**排查步骤**:
1. 使用错误标签过滤
2. 分析错误发生的时间模式
3. 检查错误相关的服务和操作
4. 查看错误详细信息和上下文

**Jaeger 查询**:
```bash
Service: All
Tags: error:true
Time: Last 1 hour
```

### 演练3: 新功能性能验证

**场景**: 验证新部署功能的性能表现

**验证步骤**:
1. 对比部署前后的性能数据
2. 使用比较功能分析差异
3. 验证性能是否满足预期
4. 识别潜在的性能风险

**Jaeger 操作**:
- 使用时间范围比较部署前后
- 选择相同的业务操作进行对比
- 关注关键性能指标变化

## 最佳实践示例

### 1. 监控大盘设置

**关键指标**:
- 服务可用性: 99.9%
- 平均响应时间: < 200ms
- 95th百分位: < 500ms
- 错误率: < 0.1%

### 2. 告警规则配置

**性能告警**:
```yaml
- alert: SlowAPIResponse
  expr: jaeger_query_latency > 1s
  for: 5m
  labels:
    severity: warning
```

**错误告警**:
```yaml
- alert: HighErrorRate
  expr: error_rate > 0.05
  for: 2m
  labels:
    severity: critical
```

## 团队协作建议

### 1. 故障响应流程

1. **接到告警**: 立即查看 Jaeger UI
2. **初步分析**: 使用预设查询快速定位
3. **深入调查**: 分析具体追踪记录
4. **协作沟通**: 分享追踪链接给相关团队
5. **问题解决**: 验证修复效果

### 2. 性能优化流程

1. **基准测试**: 收集当前性能数据
2. **瓶颈识别**: 使用时间线分析
3. **优化实施**: 基于分析结果优化
4. **效果验证**: 对比优化前后数据
5. **持续监控**: 建立长期监控机制

---

通过这些实际的演示场景和示例，您可以更好地理解和掌握 Jaeger UI 的强大功能。建议结合实际业务场景进行练习，逐步建立起有效的分布式追踪分析能力。
