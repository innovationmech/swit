# Choreography-Based Saga Example

This example demonstrates how to implement choreography-based Saga coordination using the SWIT framework. In choreography mode, services react to events autonomously without central orchestration.

## Overview

This example implements a complete order processing workflow using event-driven choreography:

1. **Order Service** - Receives and validates orders
2. **Inventory Service** - Checks and reserves inventory
3. **Payment Service** - Processes payments
4. **Shipping Service** - Creates shipments
5. **Notification Service** - Sends customer notifications

## Architecture

```
┌─────────────────┐
│  Order Service  │
└────────┬────────┘
         │ Publishes: SagaStarted
         ▼
┌─────────────────────┐
│ Inventory Service   │
└────────┬────────────┘
         │ Publishes: StepCompleted (inventory-reserved)
         ▼
┌─────────────────────┐
│  Payment Service    │
└────────┬────────────┘
         │ Publishes: StepCompleted (payment-processed)
         ▼
┌─────────────────────┐
│  Shipping Service   │
└────────┬────────────┘
         │ Publishes: StepCompleted (shipping-created)
         ▼
┌─────────────────────┐
│ Notification Service│
└─────────────────────┘
```

## Key Features

### Event-Driven Coordination
- Services react to events autonomously
- No central orchestrator controlling the flow
- Each service knows what to do based on event types

### Handler Priority
Services are assigned priorities to control execution order:
- Order Service: Priority 100
- Inventory Service: Priority 90
- Payment Service: Priority 80
- Shipping Service: Priority 70
- Notification Service: Priority 50

### Decoupled Services
- Services don't directly call each other
- Communication happens only through events
- Easy to add/remove services without changing others

### Fault Tolerance
- Each handler can have its own retry policy
- Failed events can trigger compensation flows
- Timeouts prevent indefinite waiting

## Running the Example

### Prerequisites

```bash
# Ensure you have Go 1.23+ installed
go version

# Navigate to the example directory
cd examples/saga-choreography
```

### Run the Example

```bash
# Run directly
go run main.go

# Or build and run
go build -o saga-choreography
./saga-choreography
```

### Expected Output

```
{"level":"info","msg":"Starting choreography-based saga example"}
{"level":"info","msg":"All service handlers registered successfully"}
{"level":"info","msg":"Choreography coordinator started successfully"}
{"level":"info","msg":"Starting order saga demonstration"}
{"level":"info","msg":"Order service: Received new order","saga_id":"order-saga-001","order_id":"ORD-2025-001"}
{"level":"info","msg":"Order service: Order validated, requesting inventory check"}
{"level":"info","msg":"Inventory service: Checking inventory","saga_id":"order-saga-001"}
{"level":"info","msg":"Inventory service: Inventory available, reserving stock"}
{"level":"info","msg":"Payment service: Processing payment","saga_id":"order-saga-001"}
{"level":"info","msg":"Payment service: Payment processed successfully","amount":200.0}
{"level":"info","msg":"Shipping service: Creating shipment","saga_id":"order-saga-001"}
{"level":"info","msg":"Shipping service: Shipment created, tracking number generated"}

=== ORDER CONFIRMATION ===
Your order has been successfully processed!
==========================

{"level":"info","msg":"Order saga demonstration completed"}
```

## Understanding the Code

### Service Handler Interface

Each service implements the `ChoreographyEventHandler` interface:

```go
type ChoreographyEventHandler interface {
    HandleEvent(ctx context.Context, event *saga.SagaEvent) error
    GetHandlerID() string
    GetSupportedEventTypes() []saga.SagaEventType
    GetPriority() int
}
```

### Event Flow Example

```go
// Order Service publishes initial event
orderEvent := &saga.SagaEvent{
    ID:        "order-saga-event-001",
    SagaID:    "order-saga-001",
    Type:      saga.EventSagaStarted,
    Data: map[string]interface{}{
        "order_id": "ORD-2025-001",
        "total_amount": 200.0,
    },
}
publisher.PublishEvent(ctx, orderEvent)

// Inventory Service reacts to the event
func (h *InventoryServiceHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
    if event.Type == saga.EventStepStarted {
        // Check and reserve inventory
        // ...
        // Publish completion event for next service
    }
    return nil
}
```

### Coordinator Configuration

```go
config := &coordinator.ChoreographyConfig{
    EventPublisher:        publisher,
    StateStorage:          storage,
    MaxConcurrentHandlers: 20,
    EventTimeout:          30 * time.Second,
    EnableMetrics:         true,
    HandlerRetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Second),
}
```

## Extending the Example

### Adding a New Service

1. Create a new handler struct:

```go
type NewServiceHandler struct {
    log logger.Logger
}
```

2. Implement the interface methods:

```go
func (h *NewServiceHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
    // Handle events
    return nil
}

func (h *NewServiceHandler) GetHandlerID() string {
    return "new-service"
}

func (h *NewServiceHandler) GetSupportedEventTypes() []saga.SagaEventType {
    return []saga.SagaEventType{saga.EventStepCompleted}
}

func (h *NewServiceHandler) GetPriority() int {
    return 60 // Adjust priority as needed
}
```

3. Register the handler:

```go
newHandler := &NewServiceHandler{log: log}
if err := coord.RegisterHandler(newHandler); err != nil {
    log.Fatal("Failed to register new handler", "error", err)
}
```

### Implementing Compensation

To handle failures, implement compensation logic:

```go
func (h *ServiceHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
    if event.Type == saga.EventSagaFailed {
        // Compensate: undo previous actions
        log.Info("Compensating service actions")
        
        // Release reserved resources
        // Refund payments
        // etc.
        
        return nil
    }
    // Normal processing
    return nil
}
```

## Comparison with Orchestration

| Aspect | Choreography | Orchestration |
|--------|--------------|---------------|
| Control | Decentralized | Centralized |
| Coupling | Loose | Tight |
| Complexity | Distributed | Centralized |
| Scalability | High | Medium |
| Debugging | Harder | Easier |
| Best for | Complex, long-running workflows | Simple, sequential workflows |

## Best Practices

1. **Event Naming**: Use clear, descriptive event names
2. **Idempotency**: Ensure handlers can safely process the same event multiple times
3. **Error Handling**: Always implement proper error handling and logging
4. **Monitoring**: Enable metrics to track event processing
5. **Timeouts**: Set appropriate timeouts for event handlers
6. **Compensation**: Always plan for failure scenarios

## Related Examples

- [Saga Orchestrator](../saga-orchestrator/) - Orchestration-based saga
- [Hybrid Mode](../saga-hybrid/) - Combining orchestration and choreography
- [Messaging Integration](../messaging/) - Message broker integration

## References

- [Saga Pattern](../../docs/saga-user-guide.md)
- [Choreography Coordinator API](../../pkg/saga/coordinator/choreography.go)
- [Event Publisher Interface](../../pkg/saga/types.go)

