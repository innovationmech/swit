# Dead Letter Queue (DLQ) Management Example

This example demonstrates how to use the DLQ management system in SWIT for handling failed message processing with retry policies and metrics.

## Overview

The DLQ (Dead Letter Queue) management system provides a robust way to handle messages that fail processing. It includes:

- Multiple handling policies (store, discard, retry, notify)
- Configurable retry strategies with backoff
- Automatic message expiration based on TTL
- Queue size limits with automatic eviction
- Comprehensive metrics collection
- Prometheus integration for monitoring

## Features

### Policies

1. **Store**: Store failed messages in the DLQ for later inspection
2. **Discard**: Discard failed messages without storing
3. **Retry**: Retry failed messages up to a configured maximum
4. **Notify**: Store messages and trigger notification callbacks

### Configuration

```go
config := resilience.DLQManagerConfig{
    Enabled:       true,
    Policy:        resilience.DeadLetterPolicyStore,
    MaxRetries:    3,
    RetryInterval: 5 * time.Second,
    TTL:           7 * 24 * time.Hour, // 7 days
    MaxQueueSize:  10000,
    NotifyCallback: func(msg *resilience.DeadLetterMessage) {
        // Handle notification
    },
}
```

## Running the Example

```bash
cd examples/dlq-management
go run main.go
```

## Integration with Messaging

The DLQ manager can be integrated with RabbitMQ, NATS, or Kafka subscribers:

```go
// In your message handler
func (h *MyHandler) OnError(ctx context.Context, msg *messaging.Message, err error) messaging.ErrorAction {
    // Route to DLQ
    dlqMsg := &resilience.DeadLetterMessage{
        ID:            msg.ID,
        OriginalTopic: msg.Topic,
        Payload:       msg.Payload,
        Headers:       msg.Headers,
        RetryCount:    msg.DeliveryAttempt,
        FailureReason: err.Error(),
    }
    
    if err := h.dlqManager.Route(ctx, dlqMsg); err != nil {
        log.Printf("failed to route to DLQ: %v", err)
    }
    
    return messaging.ErrorActionDeadLetter
}
```

## Metrics

The DLQ manager provides the following metrics:

- `dlq_messages_total` - Total messages received
- `dlq_messages_stored_total` - Messages stored in DLQ
- `dlq_messages_discarded_total` - Messages discarded
- `dlq_messages_retried_total` - Messages retried
- `dlq_messages_removed_total` - Messages manually removed
- `dlq_messages_expired_total` - Messages expired by TTL
- `dlq_messages_evicted_total` - Messages evicted by capacity limit
- `dlq_notifications_sent_total` - Notifications sent
- `dlq_queue_size` - Current queue size (gauge)

## Prometheus Integration

```go
// Create Prometheus metrics
promMetrics := resilience.NewDLQPrometheusMetrics("myapp", "dlq")

// Update metrics after operations
promMetrics.RecordMessage()
promMetrics.RecordStore(mgr.GetQueueSize())
```

## YAML Configuration

```yaml
dead_letter:
  enabled: true
  policy: store  # store, discard, retry, notify
  max_retries: 3
  retry_interval: 5s
  ttl: 168h  # 7 days
  max_queue_size: 10000
  enable_metrics: true
```

## Best Practices

1. **Set appropriate TTL**: Balance storage requirements with debugging needs
2. **Monitor metrics**: Set up alerts for high DLQ rates
3. **Regular cleanup**: Periodically review and remove processed DLQ messages
4. **Capacity planning**: Set `max_queue_size` based on your memory constraints
5. **Use notifications**: Implement notification callbacks for critical failures
6. **Structured logging**: Include sufficient context in failure reasons

## Related Documentation

- [Resilience Package](../../pkg/resilience/README.md)
- [Messaging Configuration](../../docs/configuration-reference.md)
- [Error Handling Best Practices](../../docs/operations-guide.md)
