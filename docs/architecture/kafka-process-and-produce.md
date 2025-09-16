# Kafka Process-and-Produce Transaction Pattern

The process-and-produce pattern coordinates message consumption, domain processing, and new message publication inside a single transactional boundary. When transactions are enabled, Kafka guarantees that either all produced messages become visible together or none are published. This document explains how to configure the SWIT Kafka adapter for transactional publishing and how to apply the pattern in services.

## When to Use This Pattern

- You consume from a Kafka topic, perform side effects, and must publish follow-up events exactly once.
- You need transactional semantics between a consumer and a producer to prevent duplicate downstream processing.
- You want deterministic retries without re-emitting already-processed messages.

## Enabling Transactions

```yaml
broker:
  type: kafka
  endpoints:
    - localhost:9092
publisher:
  topic: processed-orders
  transactional: true
```

Set `publisher.transactional` to `true` so the Kafka adapter creates idempotent, transactional producers. Each transaction publishes metadata headers:

- `swit.transaction.id`
- `swit.transaction.sequence`
- `swit.transaction.producer`

Consumers can use these headers to deduplicate inbound messages when replaying.

## End-to-End Flow

1. Consume a message and perform local validation or enrichment.
2. Call `BeginTransaction` on the publisher to allocate a transactional session.
3. Publish all follow-up events via the transaction handle.
4. Commit the transaction to atomically persist all pending messages.
5. Roll back the transaction if processing fails; no messages become visible.

### Go Example

```go
func handleOrder(ctx context.Context, broker messaging.MessageBroker, msg *messaging.Message) error {
    publisher, err := broker.CreatePublisher(messaging.PublisherConfig{
        Topic:         "processed-orders",
        Transactional: true,
    })
    if err != nil {
        return err
    }
    defer publisher.Close()

    tx, err := publisher.BeginTransaction(ctx)
    if err != nil {
        return err
    }

    // Domain processing
    processed, err := processOrder(msg)
    if err != nil {
        _ = tx.Rollback(ctx)
        return err
    }

    if err := tx.Publish(ctx, processed); err != nil {
        _ = tx.Rollback(ctx)
        return err
    }

    if err := tx.Commit(ctx); err != nil {
        return err
    }
    return nil
}
```

## Operational Considerations

- **Idempotent Consumers**: Use the provided headers to de-duplicate when retries occur.
- **Timeouts**: Provide reasonable deadlines to transaction operations; the adapter propagates context cancellation.
- **Metrics**: Retrieve publisher metrics via `GetMetrics()` to monitor transaction counts and latency.
- **Fallback**: When the broker reports `ErrTransactionsNotSupported`, disable the `transactional` flag to fall back to at-least-once semantics.

This pattern keeps processing and emission in sync so downstream consumers observe a consistent event stream without duplicates.
