# Saga DSL 完整参考文档

本文档提供 Saga DSL 的完整语法参考，包括所有配置项、参数类型、默认值、约束和示例。

## 目录

- [文档结构](#文档结构)
- [Saga 配置](#saga-配置)
- [全局配置](#全局配置)
- [步骤配置](#步骤配置)
- [动作配置](#动作配置)
- [补偿配置](#补偿配置)
- [重试策略](#重试策略)
- [条件执行](#条件执行)
- [钩子配置](#钩子配置)
- [数据类型](#数据类型)
- [模板语法](#模板语法)
- [约束和限制](#约束和限制)

## 文档结构

一个 Saga DSL 文件包含以下顶层字段：

```yaml
saga:                      # Saga 配置（必需）
  # ...

global_retry_policy:       # 全局重试策略（可选）
  # ...

global_compensation:       # 全局补偿配置（可选）
  # ...

steps:                     # 步骤列表（必需，至少1个）
  - # 步骤 1
  - # 步骤 2
  # ...
```

## Saga 配置

### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `id` | string | 是 | - | Saga 定义的唯一标识符 |
| `name` | string | 是 | - | Saga 的人类可读名称 |
| `description` | string | 否 | "" | Saga 的详细描述 |
| `version` | string | 否 | "" | Saga 定义的版本号（推荐使用语义化版本） |
| `timeout` | duration | 否 | 0（无限制） | 整个 Saga 执行的最大超时时间 |
| `mode` | ExecutionMode | 否 | "orchestration" | 执行模式 |
| `tags` | []string | 否 | [] | 用于分类和过滤的标签 |
| `metadata` | map[string]interface{} | 否 | {} | 自定义元数据 |

### 执行模式（ExecutionMode）

| 值 | 描述 | 使用场景 |
|---|------|----------|
| `orchestration` | 编排模式：中心化协调所有步骤 | 默认模式，适合大多数场景，流程控制集中 |
| `choreography` | 编舞模式：通过事件驱动，各服务自主协调 | 松耦合场景，服务间通过事件通信 |
| `hybrid` | 混合模式：结合编排和编舞 | 复杂场景，部分步骤编排，部分步骤事件驱动 |

### 示例

```yaml
saga:
  id: order-processing-v2
  name: Order Processing Saga
  description: |
    Processes customer orders including validation, payment,
    inventory reservation, and shipping notification.
  version: "2.1.0"
  timeout: 10m
  mode: orchestration
  tags:
    - orders
    - payment
    - inventory
    - critical
  metadata:
    owner: order-team
    sla: high
    business_unit: commerce
    cost_center: "CC-12345"
    environment: production
```

### 约束

- `id`: 必须是唯一的，建议使用 kebab-case（小写字母和连字符）
- `name`: 长度应在 3-100 个字符之间
- `version`: 推荐使用语义化版本格式（如 "1.2.3"）
- `timeout`: 必须 > 0 或为 0（表示无限制）

## 全局配置

### 全局重试策略（global_retry_policy）

为所有步骤定义默认的重试策略。单个步骤可以覆盖此配置。

#### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `type` | RetryType | 是 | - | 重试类型 |
| `max_attempts` | int | 是 | - | 最大重试次数（包括初始尝试） |
| `initial_delay` | duration | 是 | - | 首次重试前的延迟 |
| `max_delay` | duration | 否 | 0（无限制） | 最大重试延迟 |
| `multiplier` | float64 | 否 | 1.0 | 指数/线性退避的乘数 |
| `jitter` | bool | 否 | false | 是否添加随机抖动 |
| `increment` | duration | 否 | 0 | 线性退避的增量 |
| `retryable_errors` | []string | 否 | [] | 可重试的错误类型列表 |

#### 重试类型（RetryType）

| 值 | 描述 | 延迟计算公式 |
|---|------|--------------|
| `fixed_delay` | 固定延迟 | `delay = initial_delay` |
| `exponential_backoff` | 指数退避 | `delay = min(initial_delay * multiplier^attempt, max_delay)` |
| `linear_backoff` | 线性退避 | `delay = min(initial_delay + increment * attempt, max_delay)` |
| `no_retry` | 不重试 | - |

#### 示例

**固定延迟：**
```yaml
global_retry_policy:
  type: fixed_delay
  max_attempts: 3
  initial_delay: 5s
```

**指数退避（推荐）：**
```yaml
global_retry_policy:
  type: exponential_backoff
  max_attempts: 5
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true
  retryable_errors:
    - timeout
    - network
    - service_unavailable
    - temporary_error
```

**线性退避：**
```yaml
global_retry_policy:
  type: linear_backoff
  max_attempts: 5
  initial_delay: 1s
  increment: 2s
  max_delay: 30s
```

### 全局补偿配置（global_compensation）

定义全局补偿策略。

#### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `strategy` | CompensationStrategy | 否 | "sequential" | 补偿执行策略 |
| `timeout` | duration | 否 | 0（无限制） | 补偿操作的总超时时间 |
| `max_attempts` | int | 否 | 1 | 补偿失败时的最大重试次数 |

#### 补偿策略（CompensationStrategy）

| 值 | 描述 | 适用场景 |
|---|------|----------|
| `sequential` | 按反向顺序依次补偿 | 默认策略，补偿有依赖关系 |
| `parallel` | 并行补偿所有步骤 | 补偿操作相互独立 |
| `best_effort` | 尽力补偿，即使部分失败也继续 | 补偿失败可接受的场景 |

#### 示例

```yaml
global_compensation:
  strategy: sequential
  timeout: 5m
  max_attempts: 3
```

## 步骤配置

### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `id` | string | 是 | - | 步骤的唯一标识符 |
| `name` | string | 是 | - | 步骤的人类可读名称 |
| `description` | string | 否 | "" | 步骤的详细描述 |
| `type` | StepType | 是 | - | 步骤类型 |
| `action` | ActionConfig | 是 | - | 步骤要执行的动作 |
| `compensation` | CompensationAction | 否 | null | 补偿操作配置 |
| `retry_policy` | RetryPolicy | 否 | 使用全局配置 | 步骤特定的重试策略 |
| `timeout` | duration | 否 | 0（无限制） | 步骤执行的超时时间 |
| `condition` | Condition | 否 | null | 条件执行配置 |
| `dependencies` | []string | 否 | [] | 依赖的步骤 ID 列表 |
| `async` | bool | 否 | false | 是否异步执行 |
| `on_success` | HookConfig | 否 | null | 成功时的钩子配置 |
| `on_failure` | HookConfig | 否 | null | 失败时的钩子配置 |
| `metadata` | map[string]interface{} | 否 | {} | 自定义元数据 |

### 步骤类型（StepType）

| 值 | 描述 | 使用场景 |
|---|------|----------|
| `service` | 通用服务调用 | 适用于 HTTP 或 gRPC 服务 |
| `function` | 本地函数执行 | 执行本地 Go 函数 |
| `http` | HTTP 请求 | 显式 HTTP 调用 |
| `grpc` | gRPC 调用 | 显式 gRPC 调用 |
| `message` | 消息发布 | 发送消息到消息代理 |
| `custom` | 自定义实现 | 自定义步骤类型 |

### 示例

```yaml
steps:
  - id: validate-order
    name: Validate Order
    description: |
      Validates order data including:
      - Customer information
      - Product availability
      - Pricing accuracy
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
    metadata:
      criticality: high
      team: order-team
```

### 约束

- `id`: 在同一 Saga 中必须唯一，建议使用 kebab-case
- `dependencies`: 引用的步骤 ID 必须存在，不能形成循环依赖
- `timeout`: 应该小于 Saga 的总超时时间
- 步骤的 `type` 必须与 `action` 配置匹配

## 动作配置

### Service 动作

用于调用 HTTP 或 gRPC 服务。

#### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `name` | string | 是 | - | 服务名称 |
| `endpoint` | string | 否 | "" | 服务端点 URL |
| `method` | string | 是 | - | HTTP 方法或 RPC 方法名 |
| `path` | string | 否 | "" | URL 路径（HTTP） |
| `headers` | map[string]string | 否 | {} | HTTP 头或元数据 |
| `body` | map[string]interface{} | 否 | {} | 请求体或参数 |
| `timeout` | duration | 否 | 0 | 服务调用超时 |

#### 示例

```yaml
action:
  service:
    name: payment-service
    endpoint: http://payment-service:8080
    method: POST
    path: /api/v1/payments/process
    headers:
      Content-Type: application/json
      Authorization: "Bearer {{.context.auth_token}}"
      X-Request-ID: "{{.context.request_id}}"
      X-Idempotency-Key: "{{.saga_id}}-{{.step_id}}"
    body:
      amount: "{{.input.amount}}"
      currency: "USD"
      customer_id: "{{.output.validate-customer.customer_id}}"
      payment_method: "{{.input.payment_method}}"
      metadata:
        order_id: "{{.input.order_id}}"
    timeout: 30s
```

### Function 动作

用于执行本地 Go 函数。

#### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `name` | string | 是 | - | 函数名称 |
| `handler` | string | 否 | "" | 函数处理器路径 |
| `parameters` | map[string]interface{} | 否 | {} | 函数参数 |

#### 示例

```yaml
action:
  function:
    name: validateData
    handler: pkg.handlers.ValidateOrderData
    parameters:
      order_data: "{{.input.order}}"
      validation_rules: "{{.config.rules}}"
      strict_mode: true
```

### Message 动作

用于发布消息到消息代理。

#### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `topic` | string | 是 | - | 消息主题 |
| `broker` | string | 否 | "default" | 消息代理名称 |
| `payload` | map[string]interface{} | 是 | - | 消息负载 |
| `headers` | map[string]string | 否 | {} | 消息头 |
| `routing_key` | string | 否 | "" | 路由键（RabbitMQ） |

#### 示例

```yaml
action:
  message:
    topic: orders.created
    broker: rabbitmq-cluster
    payload:
      order_id: "{{.output.create-order.order_id}}"
      customer_id: "{{.input.customer_id}}"
      total_amount: "{{.output.calculate-total.amount}}"
      status: "created"
      timestamp: "{{.timestamp}}"
    headers:
      source: order-saga
      version: "1.0"
      content_type: application/json
    routing_key: orders.high-priority
```

### Custom 动作

用于自定义实现。配置格式取决于具体实现。

#### 示例

```yaml
action:
  custom:
    type: external-api
    config:
      provider: stripe
      api_version: "2023-10-16"
      endpoint: https://api.stripe.com/v1/charges
      credentials:
        api_key: "${STRIPE_API_KEY}"
      options:
        timeout: 30s
        retry_count: 3
```

## 补偿配置

### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `type` | CompensationType | 是 | - | 补偿类型 |
| `action` | ActionConfig | 条件必需 | null | 补偿动作（type=custom 时必需） |
| `strategy` | CompensationStrategy | 否 | 继承全局配置 | 补偿策略 |
| `timeout` | duration | 否 | 0 | 补偿超时 |
| `max_attempts` | int | 否 | 1 | 补偿失败的最大重试次数 |
| `on_failure` | OnFailureConfig | 否 | null | 补偿失败时的处理 |

### 补偿类型（CompensationType）

| 值 | 描述 | 适用场景 |
|---|------|----------|
| `automatic` | 自动补偿 | 使用默认逻辑自动回滚 |
| `custom` | 自定义补偿 | 需要特定的补偿逻辑 |
| `skip` | 跳过补偿 | 只读操作或无副作用的操作 |

### 示例

**自定义补偿：**
```yaml
compensation:
  type: custom
  action:
    service:
      name: inventory-service
      endpoint: http://inventory-service:8080
      method: POST
      path: /api/inventory/release
      body:
        reservation_id: "{{.output.reserve-inventory.reservation_id}}"
        reason: "saga_compensation"
        timestamp: "{{.timestamp}}"
  strategy: sequential
  timeout: 1m
  max_attempts: 5
  on_failure:
    action: alert
    retry_policy:
      type: exponential_backoff
      max_attempts: 10
      initial_delay: 5s
      max_delay: 5m
    notifications:
      - type: pagerduty
        target: compensation-failures
        message: "Critical: Compensation failed for reservation {{.output.reserve-inventory.reservation_id}}"
```

**跳过补偿：**
```yaml
compensation:
  type: skip  # 只读操作不需要补偿
```

### OnFailureConfig

补偿失败时的处理配置。

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `action` | string | 否 | "log" | 失败时的动作：log, alert, ignore |
| `retry_policy` | RetryPolicy | 否 | null | 补偿失败的重试策略 |
| `notifications` | []Notification | 否 | [] | 失败通知配置 |

## 重试策略

详细的重试策略配置，可用于全局配置或单个步骤。

### 常见重试场景配置

**网络调用（推荐）：**
```yaml
retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true
  retryable_errors:
    - timeout
    - network
    - connection_refused
    - service_unavailable
```

**数据库操作：**
```yaml
retry_policy:
  type: exponential_backoff
  max_attempts: 5
  initial_delay: 100ms
  max_delay: 5s
  multiplier: 2.0
  jitter: true
  retryable_errors:
    - deadlock
    - connection_timeout
    - temporary_error
```

**幂等性操作（可多次重试）：**
```yaml
retry_policy:
  type: exponential_backoff
  max_attempts: 10
  initial_delay: 500ms
  max_delay: 1m
  multiplier: 2.0
  jitter: true
```

**非幂等性操作（谨慎重试）：**
```yaml
retry_policy:
  type: fixed_delay
  max_attempts: 2
  initial_delay: 5s
  retryable_errors:
    - timeout  # 仅在超时时重试
```

### 可重试错误类型

常见的可重试错误类型包括：

- `timeout`: 请求超时
- `network`: 网络错误
- `connection_refused`: 连接被拒绝
- `connection_timeout`: 连接超时
- `service_unavailable`: 服务不可用（HTTP 503）
- `too_many_requests`: 请求过多（HTTP 429）
- `temporary_error`: 临时错误
- `deadlock`: 数据库死锁
- `resource_exhausted`: 资源耗尽

## 条件执行

步骤可以根据条件决定是否执行。

### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `expression` | string | 是 | - | 条件表达式 |
| `variables` | map[string]interface{} | 否 | {} | 表达式中使用的变量 |

### 表达式语法

条件表达式支持以下运算符和变量引用：

**比较运算符：**
- `>`: 大于
- `<`: 小于
- `>=`: 大于等于
- `<=`: 小于等于
- `==`: 等于
- `!=`: 不等于

**逻辑运算符：**
- `&&`: 逻辑与
- `||`: 逻辑或
- `!`: 逻辑非

**变量引用：**
- `$input.field`: 输入数据
- `$output.step-id.field`: 步骤输出
- `$context.field`: 执行上下文
- `$variables.name`: 自定义变量

### 示例

```yaml
condition:
  expression: "$input.amount > $variables.threshold && $input.currency == 'USD'"
  variables:
    threshold: 1000

# 更复杂的条件
condition:
  expression: |
    ($output.validate-customer.risk_score < $variables.max_risk) &&
    ($input.payment_method == 'credit_card' || $input.payment_method == 'debit_card') &&
    !$context.test_mode
  variables:
    max_risk: 75
```

## 钩子配置

钩子在步骤成功或失败时执行特定动作。

### 字段列表

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `actions` | []ActionConfig | 否 | [] | 要执行的动作列表 |
| `notifications` | []Notification | 否 | [] | 要发送的通知列表 |

### Notification

| 字段 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `type` | string | 是 | - | 通知类型 |
| `target` | string | 是 | - | 通知目标 |
| `message` | string | 是 | - | 通知消息 |

### 通知类型

- `email`: 电子邮件
- `slack`: Slack 消息
- `webhook`: Webhook 回调
- `sms`: 短信
- `pagerduty`: PagerDuty 告警

### 示例

```yaml
on_success:
  actions:
    - message:
        topic: orders.completed
        broker: default
        payload:
          order_id: "{{.output.create-order.order_id}}"
          status: "completed"
          completion_time: "{{.timestamp}}"
    - service:
        name: analytics-service
        method: POST
        path: /api/events/track
        body:
          event_type: order_completed
          order_id: "{{.output.create-order.order_id}}"
  notifications:
    - type: email
      target: "{{.input.customer_email}}"
      message: |
        Your order {{.output.create-order.order_id}} has been completed successfully.
        Thank you for your purchase!
    - type: webhook
      target: http://notification-service:8080/order-completed
      message: "Order completed: {{.output.create-order.order_id}}"

on_failure:
  notifications:
    - type: slack
      target: "#order-alerts"
      message: |
        ⚠️ Order processing failed
        Order ID: {{.input.order_id}}
        Error: {{.error.message}}
        Saga ID: {{.saga_id}}
    - type: pagerduty
      target: order-processing-team
      message: "Critical: Order {{.input.order_id}} processing failed"
```

## 数据类型

### Duration

表示时间长度，使用 Go duration 格式。

**格式：** `<数字><单位>`

**单位：**
- `ns`: 纳秒
- `us` 或 `µs`: 微秒
- `ms`: 毫秒
- `s`: 秒
- `m`: 分钟
- `h`: 小时

**示例：**
```yaml
timeout: 30s          # 30 秒
timeout: 5m           # 5 分钟
timeout: 1h30m45s     # 1 小时 30 分钟 45 秒
timeout: 500ms        # 500 毫秒
timeout: 100us        # 100 微秒
initial_delay: 1s
max_delay: 5m
```

### Map

键值对映射，支持嵌套。

**示例：**
```yaml
headers:
  Content-Type: application/json
  Authorization: "Bearer token"
  X-Custom-Header: value

metadata:
  owner: team-name
  priority: high
  tags:
    - production
    - critical
  nested:
    key1: value1
    key2: value2
```

## 模板语法

DSL 使用 Go template 语法来引用动态数据。

### 变量作用域

| 作用域 | 语法 | 描述 | 示例 |
|--------|------|------|------|
| Input | `{{.input.field}}` | Saga 输入参数 | `{{.input.order_id}}` |
| Output | `{{.output.step-id.field}}` | 步骤输出数据 | `{{.output.create-order.id}}` |
| Context | `{{.context.field}}` | 执行上下文 | `{{.context.auth_token}}` |
| Built-in | `{{.variable}}` | 内置变量 | `{{.timestamp}}` |

### 内置变量

| 变量 | 类型 | 描述 | 示例值 |
|------|------|------|--------|
| `{{.timestamp}}` | string | 当前时间戳（RFC3339） | "2024-01-15T10:30:00Z" |
| `{{.saga_id}}` | string | Saga 实例 ID | "saga-12345" |
| `{{.step_id}}` | string | 当前步骤 ID | "process-payment" |
| `{{.step_name}}` | string | 当前步骤名称 | "Process Payment" |

### 模板函数

支持的模板函数：

```yaml
# 字符串操作
message: "{{.input.name | upper}}"          # 转大写
message: "{{.input.name | lower}}"          # 转小写
message: "{{.input.text | trim}}"           # 去除空白

# 默认值
value: "{{.input.optional | default \"default-value\"}}"

# 条件
message: "{{if .input.is_premium}}Premium{{else}}Standard{{end}}"

# 格式化
timestamp: "{{.timestamp | date \"2006-01-02\"}}"
```

### 完整示例

```yaml
action:
  service:
    name: order-service
    endpoint: http://order-service:8080
    method: POST
    path: /api/orders
    headers:
      Content-Type: application/json
      Authorization: "Bearer {{.context.auth_token}}"
      X-Request-ID: "{{.context.request_id}}"
      X-Saga-ID: "{{.saga_id}}"
      X-Step-ID: "{{.step_id}}"
    body:
      order_id: "{{.input.order_id}}"
      customer:
        id: "{{.input.customer_id}}"
        name: "{{.input.customer_name | upper}}"
        email: "{{.input.customer_email | lower}}"
      items: "{{.input.items}}"
      total_amount: "{{.output.calculate-total.amount}}"
      payment_method: "{{.input.payment_method | default \"credit_card\"}}"
      metadata:
        created_at: "{{.timestamp}}"
        saga_id: "{{.saga_id}}"
        step: "{{.step_name}}"
```

## 约束和限制

### 命名约束

- **ID 字段（saga.id, step.id）：**
  - 必须唯一
  - 只能包含：小写字母、数字、连字符（-）
  - 长度：3-100 个字符
  - 推荐格式：kebab-case（如 `order-processing-saga`）

- **Name 字段：**
  - 长度：3-200 个字符
  - 可以包含任何 UTF-8 字符

### 数值约束

| 字段 | 最小值 | 最大值 | 推荐值 |
|------|--------|--------|--------|
| `max_attempts` | 1 | 100 | 3-5 |
| `timeout` | 1s | 24h | 根据业务需求 |
| `initial_delay` | 10ms | 1h | 100ms-5s |
| `max_delay` | 1s | 1h | 30s-5m |
| `multiplier` | 1.0 | 10.0 | 2.0-3.0 |
| `steps` 数量 | 1 | 1000 | < 50 |

### 依赖关系约束

- 步骤依赖不能形成循环
- 依赖的步骤必须存在
- 最大依赖深度：建议 < 10

### 超时约束

- 步骤超时 < Saga 超时
- 服务调用超时 < 步骤超时
- 补偿超时应大于步骤超时

### 大小限制

- 单个 YAML 文件：< 10 MB
- 步骤数量：< 1000（建议 < 50）
- 模板变量引用深度：< 10
- 嵌套 map 深度：< 10

### 性能考虑

- **解析性能：**
  - 小型配置（< 10 步骤）：< 1ms
  - 中型配置（10-50 步骤）：< 10ms
  - 大型配置（50-1000 步骤）：< 100ms

- **内存使用：**
  - 基础内存：< 1 MB
  - 每个步骤：约 1-5 KB

## 高级特性

### 环境变量替换

支持在配置中使用环境变量：

```yaml
action:
  service:
    name: payment-service
    endpoint: ${PAYMENT_SERVICE_URL}  # 环境变量
    headers:
      Authorization: "Bearer ${API_TOKEN}"
    body:
      api_key: ${PAYMENT_API_KEY}
```

格式：
- `${VAR_NAME}`: 标准格式
- `$VAR_NAME`: 简写格式

### 文件包含

支持包含其他 YAML 文件（需要启用）：

```yaml
# main.saga.yaml
saga:
  id: main-saga
  name: Main Saga

# 包含共享配置
!include shared/common-config.yaml

steps:
  - id: step-1
    # ...
```

### 条件步骤执行

使用条件表达式动态决定步骤执行：

```yaml
- id: premium-processing
  name: Premium Customer Processing
  type: service
  condition:
    expression: "$input.customer_tier == 'premium'"
  action:
    # ...
```

### 并行执行

多个没有依赖关系的步骤会自动并行执行：

```yaml
steps:
  # 这两个步骤并行执行
  - id: check-inventory
    name: Check Inventory
  
  - id: validate-payment
    name: Validate Payment
  
  # 这个步骤等待上面两个都完成
  - id: confirm-order
    name: Confirm Order
    dependencies:
      - check-inventory
      - validate-payment
```

### 异步步骤

标记为异步的步骤不会阻塞后续步骤：

```yaml
- id: send-email
  name: Send Confirmation Email
  type: message
  async: true  # 异步执行，不阻塞
  action:
    message:
      topic: emails.send
      payload:
        to: "{{.input.customer_email}}"
```

## 版本兼容性

### 当前版本：1.0

- **向后兼容：** 未来版本会保持向后兼容
- **弃用策略：** 弃用的特性会在至少 2 个主要版本中保留
- **迁移指南：** 重大变更会提供迁移指南

### 版本声明

在 Saga 定义中声明使用的 DSL 版本：

```yaml
saga:
  id: my-saga
  version: "1.0.0"  # Saga 定义版本
  # DSL 版本通过 parser 确定
```

## 最佳实践

### 1. 结构化命名

使用一致的命名约定：

```yaml
# 好的命名
id: order-processing-saga
steps:
  - id: validate-customer-credit
  - id: reserve-inventory-items
  - id: process-payment-transaction

# 避免的命名
id: saga1
steps:
  - id: step1
  - id: do_something
  - id: ProcessPayment  # 不要用驼峰命名
```

### 2. 适当的超时配置

为不同类型的操作设置合理的超时：

```yaml
# 快速操作（内存/缓存）
- id: cache-lookup
  timeout: 1s

# 数据库操作
- id: db-query
  timeout: 5s

# 网络调用
- id: api-call
  timeout: 30s

# 复杂处理
- id: complex-calculation
  timeout: 5m
```

### 3. 重试策略选择

根据操作类型选择合适的重试策略：

```yaml
# 幂等操作 - 可以积极重试
retry_policy:
  type: exponential_backoff
  max_attempts: 5

# 非幂等操作 - 谨慎重试
retry_policy:
  type: fixed_delay
  max_attempts: 2

# 只读操作 - 可以多次重试
retry_policy:
  type: exponential_backoff
  max_attempts: 10
```

### 4. 补偿设计

为有副作用的操作设计补偿：

```yaml
# 需要补偿的操作
- id: reserve-inventory
  compensation:
    type: custom
    action:
      service:
        method: POST
        path: /inventory/release

# 只读操作可以跳过
- id: check-inventory
  compensation:
    type: skip
```

### 5. 错误处理

使用钩子处理错误：

```yaml
on_failure:
  actions:
    - service:
        name: logging-service
        method: POST
        path: /logs/error
        body:
          saga_id: "{{.saga_id}}"
          error: "{{.error}}"
  notifications:
    - type: slack
      target: "#alerts"
      message: "Saga {{.saga_id}} failed: {{.error.message}}"
```

## 故障排查

### 常见错误

#### 1. 循环依赖

**错误信息：**
```
circular dependency detected in step dependencies involving step 'step-a'
```

**解决方法：**
检查步骤依赖关系，确保没有循环引用。

#### 2. 未知步骤引用

**错误信息：**
```
step 'step-b' depends on non-existent step 'step-a'
```

**解决方法：**
确保依赖的步骤 ID 存在且拼写正确。

#### 3. 配置验证失败

**错误信息：**
```
validation errors: field 'ID' validation failed on 'required' tag
```

**解决方法：**
检查必需字段是否已填写。

#### 4. 模板变量错误

**错误信息：**
```
template: undefined variable '.output.unknown-step.field'
```

**解决方法：**
确保引用的步骤和字段存在。

### 调试技巧

1. **启用详细日志：**
```go
parser := dsl.NewParser(
    dsl.WithLogger(myLogger),  // 自定义 logger
)
```

2. **验证配置：**
```bash
# 使用 validator 工具
swit-cli validate saga.yaml
```

3. **检查解析结果：**
```go
def, err := dsl.ParseFile("saga.yaml")
if err != nil {
    fmt.Printf("Parse error: %v\n", err)
    return
}
fmt.Printf("Parsed successfully: %+v\n", def)
```

## 相关文档

- [Saga DSL 快速入门](saga-dsl-guide.md)
- [Saga 用户指南](saga-user-guide.md)
- [示例库](../examples/saga-dsl/)
- [API 文档](https://pkg.go.dev/github.com/innovationmech/swit/pkg/saga/dsl)
- [Saga Pattern](https://microservices.io/patterns/data/saga.html)

## 更新历史

- **1.0.0** (2025-01): 初始版本
  - 基础 DSL 语法
  - 步骤类型和动作配置
  - 补偿和重试策略
  - 条件执行和钩子

