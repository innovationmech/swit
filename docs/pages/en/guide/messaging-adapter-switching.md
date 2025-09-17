# Messaging Adapter Switching

Switching between messaging adapters should be a configuration exercise—not a rewrite. This guide outlines
how to evaluate compatibility, plan migrations, and roll out broker changes with minimal risk.

## 1. Evaluate the Migration Plan

Use `messaging.PlanBrokerSwitch` to understand semantic differences before updating the configuration.

```go
ctx := context.Background()
plan, err := messaging.PlanBrokerSwitch(ctx, currentConfig, targetConfig)
if err != nil {
    return fmt.Errorf("plan switch: %w", err)
}

fmt.Printf("Switching %s -> %s (score %d)\n", plan.CurrentType, plan.TargetType, plan.CompatibilityScore)
for _, item := range plan.Checklist {
    fmt.Printf("✔ %s\n", item)
}
```

The plan captures capability deltas (transactions, dead-letter handling, ordering, etc.), a migration checklist,
and dual-write recommendations when compatibility is medium or lower.

## 2. Drive Switching from Configuration

Keep broker definitions side-by-side and select the active adapter through configuration or environment variables.

```yaml
current: rabbitmq
migration_target: kafka
brokers:
  rabbitmq:
    type: rabbitmq
    endpoints: ["amqp://guest:guest@localhost:5672/"]
  kafka:
    type: kafka
    endpoints: ["localhost:9092"]
```

At deployment time set `current` (or a `SWIT_TARGET_BROKER` override) to the desired adapter. The
`examples/messaging/adapter-switch` sample loads this structure, prints the switch checklist, and
can be run as part of CI to confirm migrations remain safe.

## 3. Capture Semantic Differences

When the plan reports feature losses, implement the suggested mitigations:

- `transactions`: adopt the transactional outbox + idempotent consumers pattern.
- `ordering`: shard by key or accept reordering with idempotent handlers.
- `dead_letter`: publish failures to a parking queue with a DLQ worker.
- `delayed_delivery`: schedule retries through timers when native delays are missing.
- `priority`: split workloads into dedicated queues and scale consumers accordingly.
- `seek`: store offsets externally and replay from checkpoints as needed.
- `streaming`: augment with a streaming service or batch-pull loop if required.

These notes are also available programmatically through `plan.FeatureDeltas` and the updated
capability matrix.

## 4. Rollout Checklist

1. Validate connectivity to the target endpoints and credentials.
2. Enable dual writes and shadow consumers when recommended.
3. Execute the generated migration steps (staging validation → production rollout).
4. Remove dual writes once metrics confirm parity and update operational runbooks.

Following this workflow keeps adapter switching a repeatable, low-risk procedure.
