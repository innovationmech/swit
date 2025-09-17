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
