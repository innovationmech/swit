# Message Broker Capabilities Matrix

This document summarizes supported capabilities across major brokers and suggests graceful degradation strategies.

## Core Features

| Feature | Kafka | RabbitMQ | NATS |
|---|---:|---:|---:|
| Transactions | ✅ | ✅ | ❌ |
| Ordering | ✅ | ❌ | ✅ |
| Partitioning | ✅ | ❌ | ❌ |
| Dead Letter | ❌ | ✅ | ❌ |
| Delayed Delivery | ❌ | ✅ | ❌ |
| Priority | ❌ | ✅ | ❌ |
| Streaming | ✅ | ❌ | ✅ |
| Seek / Replay | ✅ | ❌ | ✅ |
| Consumer Groups | ✅ | ✅ | ✅ |

## Limits & Formats

| Property | Kafka | RabbitMQ | NATS |
|---|---:|---:|---:|
| Max Message Size | 1048576 | 134217728 | 67108864 |
| Max Batch Size | 1000 | 0 | 0 |
| Compression | gzip, lz4, none, snappy, zstd | gzip, none | none |
| Serialization | avro, json, protobuf | json, msgpack, protobuf | json, msgpack, protobuf |

## Graceful Degradation

- Transactions: Use transactional outbox + idempotent consumers when unsupported.
- Ordering: Partition by key or design idempotent handlers to tolerate reordering.
- Partitioning: Emulate via routing keys/subjects; shard by naming convention.
- Dead Letter: Publish to error/parking topics and process with DLQ worker if no native DLQ.
- Delayed Delivery: Use delayed queues (RabbitMQ) or schedule via consumer timers/cron.
- Priority: Separate topics/queues by priority and weight consumers accordingly.
- Seek/Replay: Persist offsets and re-consume from stored positions when broker lacks seek.

## Extended Capabilities

Some adapters expose extended keys in the capabilities map (e.g., `nats.jetstream`, `nats.key_value_store`, `rabbitmq.exchanges`).

## Migration Notes

When switching adapters, pay attention to semantic differences:

- `transactions`: use an outbox + idempotent consumers if the target lacks native support.
- `ordering`: shard by key or design handlers that tolerate reordering.
- `partitioning`: emulate with routing keys or subject naming conventions.
- `dead_letter`: publish failures to a parking queue with a DLQ processor.
- `delayed_delivery`: schedule retries through timers or cron-style workers.
- `priority`: split workloads across queues and weight consumer capacity accordingly.
- `seek`: persist offsets externally and replay from checkpoints.
- `streaming`: augment with streaming pipelines or batch pull loops if the target is queue-oriented.

The `messaging.PlanBrokerSwitch` helper surfaces these notes programmatically through `FeatureDeltas` so migrations can be validated in tests or CI pipelines.
