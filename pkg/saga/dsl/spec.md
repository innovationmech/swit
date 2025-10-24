# Saga DSL 语法规范

## 概述

Saga DSL（Domain-Specific Language）是一种声明式的 YAML 配置语言，用于定义分布式事务的 Saga 工作流。通过 DSL，用户可以以人类可读的格式定义复杂的多服务事务，包括步骤执行、补偿操作、重试策略和错误处理。

## 设计目标

- **声明式**：使用 YAML 格式，易于阅读和编写
- **类型安全**：使用 Go 结构体进行验证和类型检查
- **灵活性**：支持多种执行模式和步骤类型
- **可扩展**：支持自定义步骤类型和动作
- **完整性**：覆盖 Saga 模式的所有核心概念

## 版本

当前版本：**1.0**

## 基础结构

Saga DSL 文件由两个顶层部分组成：

```yaml
saga:
  # Saga 配置
steps:
  # 步骤列表
```

### 完整示例

```yaml
# 订单处理 Saga 示例
saga:
  id: order-processing-saga
  name: Order Processing Saga
  description: Processes customer orders with payment and inventory
  version: "1.0.0"
  timeout: 5m
  mode: orchestration
  tags:
    - orders
    - payment
    - inventory
  metadata:
    owner: order-team
    priority: high

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true

global_compensation:
  strategy: sequential
  timeout: 2m
  max_attempts: 3

steps:
  - id: validate-order
    name: Validate Order
    description: Validates order data and inventory availability
    type: service
    action:
      service:
        name: order-service
        endpoint: http://order-service:8080
        method: POST
        path: /api/orders/validate
        headers:
          Content-Type: application/json
        body:
          order_id: "{{.input.order_id}}"
          items: "{{.input.items}}"
        timeout: 10s
    compensation:
      type: skip
    timeout: 30s

  - id: reserve-inventory
    name: Reserve Inventory
    description: Reserves inventory for order items
    type: grpc
    action:
      service:
        name: inventory-service
        endpoint: inventory-service:9090
        method: ReserveInventory
        body:
          order_id: "{{.output.validate-order.order_id}}"
          items: "{{.input.items}}"
    compensation:
      type: custom
      action:
        service:
          name: inventory-service
          endpoint: inventory-service:9090
          method: ReleaseInventory
          body:
            reservation_id: "{{.output.reserve-inventory.reservation_id}}"
      strategy: sequential
      timeout: 30s
      max_attempts: 3
      on_failure:
        action: alert
    retry_policy:
      type: exponential_backoff
      max_attempts: 5
      initial_delay: 500ms
      max_delay: 10s
      multiplier: 2.0
      jitter: true
      retryable_errors:
        - timeout
        - network
        - service_unavailable
    timeout: 1m
    dependencies:
      - validate-order

  - id: process-payment
    name: Process Payment
    description: Processes payment for the order
    type: service
    action:
      service:
        name: payment-service
        endpoint: http://payment-service:8080
        method: POST
        path: /api/payments/process
        headers:
          Content-Type: application/json
          Authorization: "Bearer {{.context.auth_token}}"
        body:
          order_id: "{{.output.validate-order.order_id}}"
          amount: "{{.input.amount}}"
          payment_method: "{{.input.payment_method}}"
    compensation:
      type: custom
      action:
        service:
          name: payment-service
          endpoint: http://payment-service:8080
          method: POST
          path: /api/payments/refund
          body:
            transaction_id: "{{.output.process-payment.transaction_id}}"
            amount: "{{.output.process-payment.amount}}"
      timeout: 1m
      max_attempts: 5
      on_failure:
        action: alert
        retry_policy:
          type: exponential_backoff
          max_attempts: 10
          initial_delay: 5s
          max_delay: 5m
    timeout: 2m
    dependencies:
      - validate-order
    on_success:
      notifications:
        - type: webhook
          target: http://notification-service:8080/payment-success
          message: "Payment successful for order {{.output.validate-order.order_id}}"
    on_failure:
      notifications:
        - type: slack
          target: "#orders-alerts"
          message: "Payment failed for order {{.output.validate-order.order_id}}"

  - id: confirm-order
    name: Confirm Order
    description: Confirms the order and notifies customer
    type: service
    action:
      service:
        name: order-service
        endpoint: http://order-service:8080
        method: POST
        path: /api/orders/confirm
        body:
          order_id: "{{.output.validate-order.order_id}}"
          transaction_id: "{{.output.process-payment.transaction_id}}"
          reservation_id: "{{.output.reserve-inventory.reservation_id}}"
    compensation:
      type: custom
      action:
        service:
          name: order-service
          endpoint: http://order-service:8080
          method: POST
          path: /api/orders/cancel
          body:
            order_id: "{{.output.validate-order.order_id}}"
    timeout: 30s
    dependencies:
      - reserve-inventory
      - process-payment
    on_success:
      actions:
        - message:
            topic: orders.confirmed
            broker: default
            payload:
              order_id: "{{.output.validate-order.order_id}}"
              timestamp: "{{.timestamp}}"
      notifications:
        - type: email
          target: "{{.input.customer_email}}"
          message: "Your order {{.output.validate-order.order_id}} has been confirmed"

  - id: notify-shipping
    name: Notify Shipping
    description: Sends notification to shipping service
    type: message
    action:
      message:
        topic: shipping.new-order
        broker: default
        payload:
          order_id: "{{.output.validate-order.order_id}}"
          items: "{{.input.items}}"
          address: "{{.input.shipping_address}}"
        routing_key: shipping.high-priority
    compensation:
      type: skip
    async: true
    dependencies:
      - confirm-order
```

## Saga 配置

### 必需字段

- `id` (string): Saga 定义的唯一标识符
- `name` (string): Saga 的人类可读名称

### 可选字段

- `description` (string): Saga 的描述
- `version` (string): Saga 定义的版本号
- `timeout` (duration): 整个 Saga 执行的超时时间
- `mode` (string): 执行模式，可选值：
  - `orchestration`：编排模式（默认）
  - `choreography`：编舞模式
  - `hybrid`：混合模式
- `tags` ([]string): 用于分类和过滤的标签
- `metadata` (map): 自定义元数据

### 示例

```yaml
saga:
  id: user-registration-saga
  name: User Registration
  description: Handles user registration with email verification and profile creation
  version: "2.1.0"
  timeout: 10m
  mode: orchestration
  tags:
    - user
    - registration
    - auth
  metadata:
    owner: auth-team
    sla: critical
    business_unit: customer-success
```

## 全局配置

### 全局重试策略

定义所有步骤的默认重试策略，单个步骤可以覆盖此策略。

```yaml
global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true
  retryable_errors:
    - timeout
    - network
```

### 全局补偿配置

定义全局补偿策略。

```yaml
global_compensation:
  strategy: sequential
  timeout: 5m
  max_attempts: 3
```

## 步骤配置

每个步骤定义一个执行单元及其补偿操作。

### 必需字段

- `id` (string): 步骤的唯一标识符
- `name` (string): 步骤的人类可读名称
- `type` (string): 步骤类型，可选值：
  - `service`: 服务调用（REST/RPC）
  - `function`: 本地函数执行
  - `http`: HTTP 请求
  - `grpc`: gRPC 调用
  - `message`: 消息发布
  - `custom`: 自定义实现
- `action`: 步骤执行的动作配置

### 可选字段

- `description` (string): 步骤描述
- `compensation`: 补偿配置
- `retry_policy`: 步骤特定的重试策略
- `timeout` (duration): 步骤执行超时
- `condition`: 条件执行配置
- `dependencies` ([]string): 依赖的步骤 ID 列表
- `async` (bool): 是否异步执行
- `on_success`: 成功时的钩子
- `on_failure`: 失败时的钩子
- `metadata` (map): 自定义元数据

### 示例

```yaml
steps:
  - id: create-user
    name: Create User Account
    description: Creates a new user account in the database
    type: service
    action:
      service:
        name: user-service
        endpoint: http://user-service:8080
        method: POST
        path: /api/users
        headers:
          Content-Type: application/json
        body:
          username: "{{.input.username}}"
          email: "{{.input.email}}"
          password: "{{.input.password}}"
    compensation:
      type: custom
      action:
        service:
          name: user-service
          method: DELETE
          path: /api/users/{{.output.create-user.user_id}}
    retry_policy:
      type: exponential_backoff
      max_attempts: 3
      initial_delay: 1s
      max_delay: 10s
    timeout: 30s
```

## 动作配置

### Service 动作

用于调用服务（HTTP 或 gRPC）。

```yaml
action:
  service:
    name: service-name
    endpoint: http://service:8080
    method: POST
    path: /api/resource
    headers:
      Content-Type: application/json
      Authorization: Bearer token
    body:
      key: value
    timeout: 10s
```

### Function 动作

用于执行本地函数。

```yaml
action:
  function:
    name: validateData
    handler: pkg.handlers.ValidateData
    parameters:
      data: "{{.input.data}}"
      rules: "{{.config.validation_rules}}"
```

### Message 动作

用于发布消息到消息代理。

```yaml
action:
  message:
    topic: orders.created
    broker: default
    payload:
      order_id: "{{.output.create-order.id}}"
      timestamp: "{{.timestamp}}"
    headers:
      source: order-saga
    routing_key: orders.high-priority
```

### Custom 动作

用于自定义实现。

```yaml
action:
  custom:
    type: external-api
    config:
      url: https://external-api.com
      method: POST
      timeout: 30s
```

## 补偿配置

补偿定义了当 Saga 失败时如何撤销已完成步骤的效果。

### 补偿类型

- `automatic`: 自动补偿（使用默认逻辑）
- `custom`: 自定义补偿动作
- `skip`: 跳过补偿

### 补偿策略

- `sequential`: 按反向顺序依次补偿（默认）
- `parallel`: 并行补偿所有步骤
- `best_effort`: 尽力补偿所有步骤，即使某些失败

### 示例

```yaml
compensation:
  type: custom
  action:
    service:
      name: inventory-service
      method: POST
      path: /api/inventory/release
      body:
        reservation_id: "{{.output.reserve-inventory.id}}"
  strategy: sequential
  timeout: 1m
  max_attempts: 5
  on_failure:
    action: alert
    retry_policy:
      type: exponential_backoff
      max_attempts: 10
      initial_delay: 5s
```

## 重试策略

定义步骤失败时的重试行为。

### 重试类型

- `fixed_delay`: 固定延迟
- `exponential_backoff`: 指数退避
- `linear_backoff`: 线性退避
- `no_retry`: 不重试

### 固定延迟

```yaml
retry_policy:
  type: fixed_delay
  max_attempts: 3
  initial_delay: 5s
```

### 指数退避

```yaml
retry_policy:
  type: exponential_backoff
  max_attempts: 5
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true
```

### 线性退避

```yaml
retry_policy:
  type: linear_backoff
  max_attempts: 5
  initial_delay: 1s
  increment: 2s
  max_delay: 30s
```

### 可重试错误

可以指定哪些错误类型应该触发重试：

```yaml
retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 10s
  retryable_errors:
    - timeout
    - network
    - service_unavailable
    - temporary_error
```

## 条件执行

步骤可以根据条件决定是否执行。

```yaml
condition:
  expression: "$input.amount > 1000"
  variables:
    threshold: 1000
    currency: USD
```

条件表达式支持：
- 比较运算符：`>`, `<`, `>=`, `<=`, `==`, `!=`
- 逻辑运算符：`&&`, `||`, `!`
- 变量引用：`$input.field`, `$output.step-id.field`, `$context.field`

## 依赖关系

步骤可以声明对其他步骤的依赖：

```yaml
steps:
  - id: step-a
    name: Step A
    # ...
    
  - id: step-b
    name: Step B
    dependencies:
      - step-a
    # ...
    
  - id: step-c
    name: Step C
    dependencies:
      - step-a
      - step-b
    # ...
```

## 钩子

步骤可以定义成功或失败时的钩子。

### 成功钩子

```yaml
on_success:
  actions:
    - message:
        topic: orders.completed
        payload:
          order_id: "{{.output.order_id}}"
  notifications:
    - type: email
      target: customer@example.com
      message: "Order completed successfully"
```

### 失败钩子

```yaml
on_failure:
  actions:
    - service:
        name: alert-service
        method: POST
        path: /api/alerts
        body:
          severity: high
          message: "Step failed"
  notifications:
    - type: slack
      target: "#alerts"
      message: "ALERT: Step {{.step_name}} failed"
    - type: webhook
      target: http://monitoring:8080/webhook
      message: "Failure detected"
```

## 变量和模板

DSL 支持使用 Go 模板语法引用变量：

### 输入变量

```yaml
body:
  customer_id: "{{.input.customer_id}}"
  amount: "{{.input.amount}}"
```

### 输出变量

引用前面步骤的输出：

```yaml
body:
  user_id: "{{.output.create-user.user_id}}"
  order_id: "{{.output.create-order.order_id}}"
```

### 上下文变量

访问执行上下文：

```yaml
headers:
  Authorization: "Bearer {{.context.auth_token}}"
  X-Request-ID: "{{.context.request_id}}"
  X-Trace-ID: "{{.context.trace_id}}"
```

### 内置变量

- `{{.timestamp}}`: 当前时间戳
- `{{.saga_id}}`: Saga 实例 ID
- `{{.step_name}}`: 当前步骤名称

## 时间格式

所有时间值使用 Go duration 格式：

- `ns`: 纳秒
- `us` 或 `µs`: 微秒
- `ms`: 毫秒
- `s`: 秒
- `m`: 分钟
- `h`: 小时

示例：
- `500ms`: 500 毫秒
- `30s`: 30 秒
- `5m`: 5 分钟
- `1h30m`: 1 小时 30 分钟

## 验证规则

DSL 使用 `validate` 标签进行验证：

1. **必需字段**：使用 `required` 标签
2. **最小值**：使用 `min` 标签
3. **枚举值**：使用 `oneof` 标签
4. **嵌套验证**：使用 `dive` 标签

验证在解析 YAML 时自动执行。

## 完整的电商订单示例

```yaml
saga:
  id: ecommerce-order-saga
  name: E-Commerce Order Processing
  description: Complete order processing workflow
  version: "3.0.0"
  timeout: 10m
  mode: orchestration
  tags:
    - ecommerce
    - orders
    - payment

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true

global_compensation:
  strategy: sequential
  timeout: 5m
  max_attempts: 3

steps:
  - id: validate-order
    name: Validate Order
    type: service
    action:
      service:
        name: order-service
        method: POST
        path: /validate
    timeout: 30s

  - id: check-inventory
    name: Check Inventory
    type: grpc
    action:
      service:
        name: inventory-service
        method: CheckStock
    dependencies:
      - validate-order
    timeout: 1m

  - id: reserve-inventory
    name: Reserve Inventory
    type: service
    action:
      service:
        name: inventory-service
        method: POST
        path: /reserve
    compensation:
      type: custom
      action:
        service:
          name: inventory-service
          method: DELETE
          path: /reserve/{{.output.reserve-inventory.id}}
    dependencies:
      - check-inventory
    timeout: 1m

  - id: process-payment
    name: Process Payment
    type: service
    action:
      service:
        name: payment-service
        method: POST
        path: /charge
    compensation:
      type: custom
      action:
        service:
          name: payment-service
          method: POST
          path: /refund
      max_attempts: 5
    dependencies:
      - validate-order
    timeout: 2m

  - id: create-shipment
    name: Create Shipment
    type: service
    action:
      service:
        name: shipping-service
        method: POST
        path: /shipments
    compensation:
      type: custom
      action:
        service:
          name: shipping-service
          method: DELETE
          path: /shipments/{{.output.create-shipment.id}}
    dependencies:
      - reserve-inventory
      - process-payment
    timeout: 1m

  - id: send-confirmation
    name: Send Confirmation Email
    type: message
    action:
      message:
        topic: notifications.order-confirmed
        payload:
          order_id: "{{.output.validate-order.order_id}}"
          customer_email: "{{.input.customer_email}}"
    compensation:
      type: skip
    async: true
    dependencies:
      - create-shipment
```

## 最佳实践

1. **明确命名**：使用清晰、描述性的 ID 和名称
2. **适当的超时**：为每个步骤设置合理的超时时间
3. **补偿操作**：始终为有副作用的步骤定义补偿
4. **重试策略**：为网络调用配置重试，使用指数退避
5. **依赖管理**：明确声明步骤间的依赖关系
6. **错误处理**：使用钩子捕获和响应错误
7. **监控**：利用元数据和标签进行监控和过滤
8. **版本控制**：为 Saga 定义使用版本号
9. **文档化**：为 Saga 和步骤提供清晰的描述

## 未来扩展

计划在未来版本中添加的特性：

- 条件分支（if/else）
- 循环（for/while）
- 并行执行组
- 子 Saga 调用
- 动态步骤生成
- 高级表达式语言
- Schema 验证增强

## 参考

- [Saga Pattern](https://microservices.io/patterns/data/saga.html)
- [YAML 1.2 规范](https://yaml.org/spec/1.2/)
- [Go Template 语法](https://pkg.go.dev/text/template)
- [Go Validator](https://github.com/go-playground/validator)

