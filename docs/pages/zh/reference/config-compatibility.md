---
title: 配置兼容性矩阵
description: 跨 Broker 的配置兼容性、约束与降级策略
---

# 配置兼容性矩阵

本文汇总 Kafka / RabbitMQ / NATS 在配置层面的兼容性、不变量约束以及当目标 Broker 不支持某选项时的替代方案。与“能力矩阵”配合阅读效果更佳。

## 概览表

| 领域 | Kafka | RabbitMQ | NATS |
|---|---|---|---|
| 端点格式 | `host:port`（无 scheme） | `amqp(s)://host[:port]` | `nats(s)://host[:port]` |
| 鉴权 | SASL（PLAIN、SCRAM-256/512）、none | 用户/密码；TLS；AMQP URL | Token/JWT；TLS；NATS creds |
| TLS 验证跳过 | 配置支持；默认建议校验 | 支持；优先校验 | 支持；启用时 SkipVerify=true 会警告 |
| DLQ | 无原生（需自实现） | 原生 DLX | 无原生（需自实现） |
| 延迟投递 | 自实现 | 原生（插件/参数） | 自实现 |
| 优先级 | 自实现 | 原生 | 自实现 |
| 有序性 | 分区 + key | 不保证 | Subject 级有序 |
| 流/回放 | 原生 | 无 | JetStream |

## 适配器强制不变量

- Kafka：
  - Endpoint 必须为 `host:port`（禁止带 URL scheme）。
  - `producer.idempotent=true` 时必须 `producer.acks=all`。
  - `acks=none` 会提示潜在数据丢失警告。
- RabbitMQ：
  - Endpoint 必须以 `amqp://` 或 `amqps://` 开头且为合法 URL。
  - 心跳 < 10s 会产生抖动风险警告。
  - 拓扑校验：交换机/队列名称与类型、TTL 正值、绑定引用存在等。
- NATS：
  - Endpoint 必须包含 scheme（如 `nats://`）。
  - 启用 TLS 且 `SkipVerify=true` 会给出安全性警告。
  - JetStream 校验：流/消费者/KV/ObjectStore 的字段与取值范围等。

## 常见配置映射

- 死信（DLQ）：
  - Kafka/NATS：将失败投递至 parking 主题/subject，并用 DLQ worker 处理。
  - RabbitMQ：使用 DLX/队列参数。
- 延迟投递：
  - Kafka/NATS：用消费者定时/重试退避调度。
  - RabbitMQ：使用延迟队列或 TTL + DLX。
- 优先级：
  - 使用多主题/队列/subject 分级，并按权重分配消费者。
- 有序性：
  - Kafka：按 key 分区；NATS：subject 内有序；RabbitMQ：设计幂等处理。

## 迁移建议

切换 Broker 时，可使用代码中的“兼容性检查器 + 切换计划”功能获得特性差异、检查清单与建议。当总体兼容性为中等或更低、或将失去关键保证（如事务、DLQ、有序性、消费组）时，建议引入双写与影子消费。

## 参考

- 能力矩阵：/zh/reference/capabilities
- 校验来源：`pkg/messaging/{kafka|rabbitmq|nats}` 适配器校验逻辑


