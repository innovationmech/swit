# Saga DSL 快速入门指南

## 简介

Saga DSL 是一种声明式的 YAML 配置语言，用于定义分布式事务的 Saga 工作流。通过 DSL，您可以用简洁易读的方式定义复杂的多服务事务，包括：

- 步骤执行顺序和依赖关系
- 服务调用和消息发布
- 补偿操作和错误处理
- 重试策略和超时配置
- 条件执行和异步处理

## 5 分钟快速上手

### 第一步：创建 Saga 定义文件

创建一个名为 `my-first-saga.yaml` 的文件：

```yaml
saga:
  id: my-first-saga
  name: My First Saga
  description: A simple Saga example
  timeout: 5m
  mode: orchestration

steps:
  - id: step-1
    name: First Step
    type: service
    action:
      service:
        name: my-service
        endpoint: http://my-service:8080
        method: POST
        path: /api/action
        body:
          message: "Hello from Saga!"
    timeout: 30s
```

### 第二步：解析 Saga 定义

在 Go 代码中加载和解析 Saga 定义：

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/innovationmech/swit/pkg/saga/dsl"
)

func main() {
    // 解析 YAML 文件
    def, err := dsl.ParseFile("my-first-saga.yaml")
    if err != nil {
        log.Fatalf("Failed to parse Saga: %v", err)
    }
    
    fmt.Printf("Loaded Saga: %s (%s)\n", def.Saga.Name, def.Saga.ID)
    fmt.Printf("Steps: %d\n", len(def.Steps))
}
```

### 第三步：添加补偿操作

为步骤添加补偿操作，以便在失败时回滚：

```yaml
steps:
  - id: step-1
    name: First Step
    type: service
    action:
      service:
        name: my-service
        endpoint: http://my-service:8080
        method: POST
        path: /api/action
        body:
          message: "Hello from Saga!"
    compensation:
      type: custom
      action:
        service:
          name: my-service
          endpoint: http://my-service:8080
          method: POST
          path: /api/compensate
          body:
            step_id: "step-1"
    timeout: 30s
```

恭喜！您已经创建了第一个带有补偿操作的 Saga。

## 基本概念

### Saga 配置

Saga 配置定义了整个工作流的基本信息：

- **id**: Saga 的唯一标识符
- **name**: 人类可读的名称
- **description**: Saga 的描述
- **timeout**: 整个 Saga 的执行超时时间
- **mode**: 执行模式（orchestration、choreography、hybrid）

### 步骤 (Steps)

步骤是 Saga 的基本执行单元，每个步骤包含：

- **id**: 步骤的唯一标识符
- **name**: 步骤名称
- **type**: 步骤类型（service、function、http、grpc、message、custom）
- **action**: 步骤要执行的动作
- **compensation**: 补偿操作（可选）
- **timeout**: 步骤超时时间

### 步骤类型

DSL 支持多种步骤类型：

1. **service**: 通用服务调用（HTTP/gRPC）
2. **function**: 本地函数执行
3. **http**: HTTP 请求
4. **grpc**: gRPC 调用
5. **message**: 消息发布
6. **custom**: 自定义实现

### 补偿操作

补偿操作定义了当 Saga 失败时如何撤销已完成步骤的效果。补偿类型包括：

- **automatic**: 自动补偿（使用默认逻辑）
- **custom**: 自定义补偿动作
- **skip**: 跳过补偿

## 实用示例

### 示例 1: 简单的 HTTP 服务调用

```yaml
saga:
  id: simple-http-saga
  name: Simple HTTP Saga
  timeout: 2m

steps:
  - id: call-api
    name: Call External API
    type: http
    action:
      service:
        name: external-api
        endpoint: https://api.example.com
        method: POST
        path: /v1/data
        headers:
          Content-Type: application/json
          Authorization: "Bearer ${API_TOKEN}"
        body:
          key: "value"
        timeout: 10s
    timeout: 30s
```

### 示例 2: 带依赖关系的多步骤 Saga

```yaml
saga:
  id: multi-step-saga
  name: Multi-Step Saga
  timeout: 5m

steps:
  # 第一步：验证数据
  - id: validate
    name: Validate Data
    type: service
    action:
      service:
        name: validation-service
        method: POST
        path: /validate
    timeout: 30s

  # 第二步：处理数据（依赖验证步骤）
  - id: process
    name: Process Data
    type: service
    action:
      service:
        name: processing-service
        method: POST
        path: /process
        body:
          validation_result: "{{.output.validate.result}}"
    compensation:
      type: custom
      action:
        service:
          name: processing-service
          method: DELETE
          path: /process/{{.output.process.id}}
    dependencies:
      - validate
    timeout: 1m

  # 第三步：通知（依赖处理步骤）
  - id: notify
    name: Send Notification
    type: message
    action:
      message:
        topic: data.processed
        broker: default
        payload:
          process_id: "{{.output.process.id}}"
          status: "completed"
    compensation:
      type: skip
    dependencies:
      - process
    async: true
```

### 示例 3: 带重试策略的 Saga

```yaml
saga:
  id: retry-saga
  name: Saga with Retry Policy
  timeout: 10m

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true

steps:
  - id: unreliable-service
    name: Call Unreliable Service
    type: service
    action:
      service:
        name: unreliable-service
        method: POST
        path: /api/action
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
    timeout: 2m
```

### 示例 4: 带钩子的 Saga

```yaml
saga:
  id: hook-saga
  name: Saga with Hooks
  timeout: 5m

steps:
  - id: payment
    name: Process Payment
    type: service
    action:
      service:
        name: payment-service
        method: POST
        path: /charge
        body:
          amount: "{{.input.amount}}"
          customer_id: "{{.input.customer_id}}"
    compensation:
      type: custom
      action:
        service:
          name: payment-service
          method: POST
          path: /refund
          body:
            transaction_id: "{{.output.payment.transaction_id}}"
    on_success:
      notifications:
        - type: email
          target: "{{.input.customer_email}}"
          message: "Payment successful"
        - type: webhook
          target: http://notification-service:8080/payment-success
          message: "Payment processed for customer {{.input.customer_id}}"
    on_failure:
      notifications:
        - type: slack
          target: "#payment-alerts"
          message: "Payment failed for customer {{.input.customer_id}}"
    timeout: 2m
```

## 变量和模板

DSL 支持 Go 模板语法来引用动态数据：

### 输入变量

访问 Saga 的输入参数：

```yaml
body:
  customer_id: "{{.input.customer_id}}"
  amount: "{{.input.amount}}"
  order_items: "{{.input.items}}"
```

### 输出变量

引用前面步骤的输出：

```yaml
body:
  user_id: "{{.output.create-user.user_id}}"
  order_id: "{{.output.create-order.order_id}}"
  payment_id: "{{.output.process-payment.transaction_id}}"
```

### 上下文变量

访问执行上下文信息：

```yaml
headers:
  Authorization: "Bearer {{.context.auth_token}}"
  X-Request-ID: "{{.context.request_id}}"
  X-Trace-ID: "{{.context.trace_id}}"
```

### 内置变量

使用内置的系统变量：

```yaml
body:
  timestamp: "{{.timestamp}}"
  saga_id: "{{.saga_id}}"
  step_name: "{{.step_name}}"
```

## 时间格式

所有时间值使用 Go duration 格式：

- `ns`: 纳秒
- `us` 或 `µs`: 微秒
- `ms`: 毫秒
- `s`: 秒
- `m`: 分钟
- `h`: 小时

示例：

```yaml
timeout: 30s          # 30 秒
timeout: 5m           # 5 分钟
timeout: 1h30m        # 1 小时 30 分钟
timeout: 500ms        # 500 毫秒
initial_delay: 1s     # 1 秒
max_delay: 30s        # 30 秒
```

## 环境变量

DSL 支持环境变量替换，使用 `${VAR_NAME}` 或 `$VAR_NAME` 语法：

```yaml
action:
  service:
    name: payment-service
    endpoint: ${PAYMENT_SERVICE_URL}
    headers:
      Authorization: "Bearer ${API_TOKEN}"
      Content-Type: application/json
```

在解析时，这些占位符会被实际的环境变量值替换。

## 常见问题

### Q: 如何定义步骤的执行顺序？

A: 使用 `dependencies` 字段指定步骤依赖关系。没有依赖的步骤会首先执行，有依赖的步骤会在所有依赖步骤完成后执行。

```yaml
steps:
  - id: step-a
    # ...
  
  - id: step-b
    dependencies:
      - step-a  # step-b 在 step-a 完成后执行
```

### Q: 什么时候应该跳过补偿？

A: 对于只读操作或没有副作用的操作（如验证、查询），可以跳过补偿：

```yaml
compensation:
  type: skip
```

### Q: 如何处理超时？

A: 在三个层级设置超时：
1. Saga 级别：整个工作流的最大执行时间
2. 步骤级别：单个步骤的最大执行时间
3. 服务调用级别：单次服务调用的超时

```yaml
saga:
  timeout: 10m  # Saga 级别

steps:
  - id: step-1
    timeout: 1m  # 步骤级别
    action:
      service:
        timeout: 30s  # 服务调用级别
```

### Q: 如何配置重试？

A: 可以在全局或步骤级别配置重试策略：

```yaml
# 全局重试策略
global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s

steps:
  - id: step-1
    # 步骤级别重试策略（覆盖全局配置）
    retry_policy:
      type: exponential_backoff
      max_attempts: 5
      initial_delay: 500ms
```

### Q: 如何实现并行执行？

A: 多个没有相互依赖的步骤会自动并行执行。如果需要控制并行度，可以使用依赖关系：

```yaml
steps:
  # 这两个步骤会并行执行
  - id: step-a
  - id: step-b
  
  # 这个步骤会在 step-a 和 step-b 都完成后执行
  - id: step-c
    dependencies:
      - step-a
      - step-b
```

## 最佳实践

### 1. 使用描述性的 ID 和名称

**好的示例：**
```yaml
steps:
  - id: validate-customer-credit
    name: Validate Customer Credit Score
```

**不好的示例：**
```yaml
steps:
  - id: step1
    name: Step 1
```

### 2. 为有副作用的操作定义补偿

任何修改状态的操作都应该有补偿：

```yaml
- id: reserve-inventory
  action:
    service:
      method: POST
      path: /inventory/reserve
  compensation:
    type: custom
    action:
      service:
        method: POST
        path: /inventory/release
```

### 3. 使用合理的超时时间

- 网络调用：10-30 秒
- 数据库操作：5-10 秒
- 复杂计算：1-5 分钟
- 整个 Saga：根据业务需求，通常 5-15 分钟

### 4. 配置适当的重试策略

对于瞬时错误（网络抖动、服务暂时不可用），使用指数退避：

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
    - service_unavailable
```

### 5. 使用标签和元数据

添加标签和元数据便于监控和过滤：

```yaml
saga:
  tags:
    - payment
    - critical
    - customer-facing
  metadata:
    owner: payment-team
    sla: high
    business_unit: finance
```

### 6. 明确声明依赖关系

即使某些依赖关系看起来很明显，也应该明确声明：

```yaml
steps:
  - id: create-order
  - id: process-payment
  - id: confirm-order
    dependencies:
      - create-order
      - process-payment
```

### 7. 使用钩子进行通知和监控

在关键步骤添加成功/失败钩子：

```yaml
on_success:
  notifications:
    - type: webhook
      target: http://monitoring-service/success
on_failure:
  notifications:
    - type: slack
      target: "#alerts"
```

### 8. 版本控制

为 Saga 定义添加版本号，便于追踪变更：

```yaml
saga:
  id: order-processing-saga
  version: "2.1.0"
```

## 下一步

现在您已经掌握了 Saga DSL 的基础知识，可以：

1. 查看 [Saga DSL 完整参考](saga-dsl-reference.md) 了解所有配置选项
2. 浏览 [examples/saga-dsl/](../examples/saga-dsl/) 目录中的完整示例
3. 阅读 [Saga 用户指南](saga-user-guide.md) 了解 Saga 模式的概念
4. 查看 [API 文档](https://pkg.go.dev/github.com/innovationmech/swit/pkg/saga/dsl) 了解编程接口

## 相关资源

- [Saga Pattern 介绍](https://microservices.io/patterns/data/saga.html)
- [YAML 语法](https://yaml.org/spec/1.2/)
- [Go Template 语法](https://pkg.go.dev/text/template)
- [Saga 架构文档](architecture.md)

