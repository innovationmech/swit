---
title: Configuration Compatibility Matrix
description: Cross-broker configuration compatibility, constraints, and fallbacks
---

# Configuration Compatibility Matrix

This document summarizes cross-broker configuration compatibility, key invariants, and suggested fallbacks when an option is not supported by a target broker. It complements the feature view in the Capabilities Matrix.

## Summary Table

| Area | Kafka | RabbitMQ | NATS |
|---|---|---|---|
| Endpoint format | `host:port` (no scheme) | `amqp(s)://host[:port]` | `nats(s)://host[:port]` |
| Auth types | SASL (PLAIN, SCRAM-256/512), none | User/password; TLS; via AMQP URL | Token/JWT; TLS; NATS creds |
| TLS verify skip | Supported via config; secure by default | Supported via TLS; prefer verified | Supported; warns when SkipVerify=true |
| DLQ | No native; emulate | Native DLX | No native; emulate |
| Delayed delivery | Emulate | Native (plug-in/args) | Emulate |
| Priority | Emulate | Native | Emulate |
| Ordering | Partitions + keys | Not guaranteed | Subject ordering (per subject) |
| Streams/Replay | Native | No | JetStream |

## Invariants enforced by adapters

- Kafka:
  - Endpoints must be `host:port` (no URL scheme).
  - If `producer.idempotent=true`, then `producer.acks=all` is required.
  - `acks=none` emits a data loss warning.
- RabbitMQ:
  - Endpoints must start with `amqp://` or `amqps://` and be valid URLs.
  - Heartbeat < 10s emits a churn warning.
  - Topology validation enforces exchange/queue names, types, positive TTL, and valid bindings.
- NATS:
  - Endpoints must include a scheme like `nats://`.
  - TLS `SkipVerify=true` warns when TLS is enabled.
  - JetStream validation enforces required fields (stream subjects, consumer policies, KV/ObjectStore limits, etc.).

## Common option mappings

- Dead Letter:
  - Kafka/NATS: publish failures to a parking topic/subject and process with a DLQ worker.
  - RabbitMQ: use DLX/arguments on queues.
- Delayed Delivery:
  - Kafka/NATS: schedule via consumer timers/cron or retry backoff.
  - RabbitMQ: use delayed queues or TTL + DLX pattern.
- Priority:
  - Use separate topics/queues/subjects per priority and weight consumers when not natively supported.
- Ordering:
  - Kafka: partition by key; NATS: subject-level ordering; RabbitMQ: design idempotent handlers.

## Migration guidance

When switching adapters, use the Broker Switch Plan and Compatibility Checker in the codebase to enumerate feature deltas and produce a migration checklist. Prefer dual-writes for medium-or-lower overall compatibility or when losing critical guarantees like transactions, DLQ, ordering, or consumer groups.

## References

- Capabilities: /en/reference/capabilities
- Validation sources: adapter validators in `pkg/messaging/{kafka|rabbitmq|nats}`


