# Service Metadata Extensions for Messaging

## Overview

This package provides comprehensive service metadata extensions to support messaging capabilities in the SWIT framework. It enables services to declare their messaging features, event schemas, and supported patterns in a standardized way.

**Issue**: #437 - feat(integration): service metadata extensions for messaging  
**Dependencies**: #173, #174, #175  
**Parent**: #177 (Integration Patterns & Advanced Features)

## Features

- **Service Metadata**: Comprehensive service description with messaging capabilities
- **Messaging Capabilities**: Declare broker support, batching, async, transactions, ordering, etc.
- **Event Schemas**: Define published and subscribed event structures with validation
- **Pattern Support**: Indicate implementation of saga, outbox, inbox, CQRS, and other patterns
- **Validation**: Built-in validation for metadata, schemas, and configurations
- **Builder Pattern**: Fluent API for constructing metadata with validation
- **Integration**: Seamless integration with transport layer metadata

## Core Types

### ServiceMetadata

Complete service metadata including messaging capabilities and event definitions:

```go
type ServiceMetadata struct {
    Name                  string
    Version               string
    Description           string
    MessagingCapabilities *MessagingCapabilities
    PublishedEvents       []EventMetadata
    SubscribedEvents      []EventMetadata
    PatternSupport        PatternSupport
    Tags                  []string
    Dependencies          []string
    Extended              map[string]interface{}
}
```

### MessagingCapabilities

Defines messaging features supported by a service:

```go
type MessagingCapabilities struct {
    BrokerTypes          []messaging.BrokerType
    SupportsBatching     bool
    SupportsAsync        bool
    SupportsTransactions bool
    SupportsOrdering     bool
    CompressionFormats   []messaging.CompressionType
    SerializationFormats []string
    MaxConcurrency       int
    MaxBatchSize         int
    ProcessingTimeout    time.Duration
    DeadLetterSupport    bool
    RetryPolicy          *RetryPolicyMetadata
}
```

### EventMetadata

Describes an event that a service publishes or subscribes to:

```go
type EventMetadata struct {
    EventType          string
    Version            string
    Description        string
    Schema             *MessageSchema
    Topics             []string
    Required           bool
    Deprecated         bool
    DeprecationMessage string
    Examples           []string
}
```

### MessageSchema

Defines the structure and validation rules for messages:

```go
type MessageSchema struct {
    Type            SchemaType // json-schema, protobuf, avro, custom
    Format          string
    Definition      interface{}
    ContentType     string
    Encoding        string
    RequiredFields  []string
    ValidationRules map[string]interface{}
}
```

### PatternSupport

Indicates which messaging patterns are implemented:

```go
type PatternSupport struct {
    Saga           bool
    Outbox         bool
    Inbox          bool
    CQRS           bool
    EventSourcing  bool
    CircuitBreaker bool
    BulkOperations bool
}
```

## Usage Examples

### Basic Service Metadata

```go
import (
    "github.com/innovationmech/swit/pkg/integration"
    "github.com/innovationmech/swit/pkg/messaging"
)

// Create minimal service metadata
metadata, err := integration.NewServiceMetadataBuilder("my-service", "1.0.0").
    WithDescription("My awesome service").
    WithMessagingCapabilities(
        integration.NewMessagingCapabilitiesBuilder().
            WithBrokerType(messaging.BrokerTypeKafka).
            WithAsync(true).
            MustBuild(),
    ).
    Build()

if err != nil {
    log.Fatalf("Failed to build metadata: %v", err)
}
```

### Complete Service with Events

```go
// Define messaging capabilities
capabilities := integration.NewMessagingCapabilitiesBuilder().
    WithBrokerType(messaging.BrokerTypeKafka).
    WithBrokerType(messaging.BrokerTypeRabbitMQ).
    WithBatching(true).
    WithAsync(true).
    WithTransactions(true).
    WithConcurrency(20).
    WithBatchSize(100).
    WithProcessingTimeout(30 * time.Second).
    WithDeadLetterSupport(true).
    WithRetryPolicy(integration.NewRetryPolicy(
        3,
        time.Second,
        integration.BackoffTypeExponential,
    )).
    MustBuild()

// Define published event with schema
orderCreatedEvent := integration.EventMetadata{
    EventType:   "order.created",
    Version:     "1.0.0",
    Description: "Emitted when a new order is created",
    Schema: &integration.MessageSchema{
        Type:        integration.SchemaTypeJSONSchema,
        ContentType: "application/json",
        RequiredFields: []string{"orderId", "customerId", "items"},
    },
    Topics:   []string{"orders", "order.events"},
    Required: true,
}

// Define subscribed event
paymentEvent := integration.NewEventMetadata("payment.completed", "1.0.0")
paymentEvent.Description = "Payment confirmation from payment service"
paymentEvent.Topics = []string{"payment.events"}
paymentEvent.Required = true

// Build complete metadata
metadata, err := integration.NewServiceMetadataBuilder("order-service", "2.0.0").
    WithDescription("Order processing service").
    WithMessagingCapabilities(capabilities).
    WithPublishedEvent(orderCreatedEvent).
    WithSubscribedEvent(paymentEvent).
    WithPatternSupport(integration.PatternSupport{
        Saga:           true,
        Outbox:         true,
        Inbox:          true,
        CircuitBreaker: true,
    }).
    WithTag("core").
    WithTag("transactional").
    WithDependency("payment-service").
    WithDependency("inventory-service").
    Build()

if err != nil {
    log.Fatalf("Failed to build metadata: %v", err)
}
```

### Integration with Transport Layer

```go
// Convert service metadata to transport handler metadata
handlerMetadata := integration.ConvertToHandlerMetadata(
    serviceMetadata,
    "/health",
)

// Use in transport service handler
type MyService struct {
    metadata *transport.HandlerMetadata
}

func (s *MyService) GetMetadata() *transport.HandlerMetadata {
    return s.metadata
}

// Extract service metadata from handler metadata
if extracted := integration.ExtractServiceMetadata(handlerMetadata); extracted != nil {
    fmt.Printf("Service has %d published events\n", len(extracted.PublishedEvents))
}
```

### Querying Capabilities

```go
// Check if service supports specific capabilities
if metadata.HasCapability("batching") {
    fmt.Println("Service supports batching")
}

if metadata.HasCapability("async") {
    fmt.Println("Service supports async publishing")
}

// Check pattern implementations
if metadata.HasPattern("saga") {
    fmt.Println("Service implements saga pattern")
}

if metadata.HasPattern("outbox") {
    fmt.Println("Service implements outbox pattern")
}

// Check broker support
if metadata.SupportsBrokerType(messaging.BrokerTypeKafka) {
    fmt.Println("Service supports Kafka")
}

// Get specific event metadata
if event := metadata.GetPublishedEvent("order.created"); event != nil {
    fmt.Printf("Event: %s (version %s)\n", event.EventType, event.Version)
    fmt.Printf("Topics: %v\n", event.Topics)
    if event.Schema != nil {
        fmt.Printf("Schema type: %s\n", event.Schema.Type)
        fmt.Printf("Required fields: %v\n", event.Schema.RequiredFields)
    }
}
```

### Validation

```go
// Metadata is automatically validated during build
metadata, err := integration.NewServiceMetadataBuilder("my-service", "invalid-version").
    Build()

if err != nil {
    // Will fail: "invalid service version: version invalid-version does not match semantic versioning format"
    fmt.Printf("Validation error: %v\n", err)
}

// Explicit validation
metadata := &integration.ServiceMetadata{
    Name:    "test-service",
    Version: "1.0.0",
}

if err := metadata.Validate(); err != nil {
    fmt.Printf("Metadata is invalid: %v\n", err)
}
```

## Schema Types

The following schema types are supported:

- `SchemaTypeJSONSchema`: JSON Schema (recommended for JSON payloads)
- `SchemaTypeProtobuf`: Protocol Buffers schema
- `SchemaTypeAvro`: Apache Avro schema
- `SchemaTypeCustom`: Custom schema format

## Backoff Types

The following backoff strategies are supported for retry policies:

- `BackoffTypeConstant`: Fixed delay between retries
- `BackoffTypeLinear`: Linear increase in delay
- `BackoffTypeExponential`: Exponential backoff (recommended)

## Validation Rules

All metadata types include comprehensive validation:

### ServiceMetadata
- Name and version are required
- Version must follow semantic versioning (e.g., 1.0.0, 2.1.3-alpha)
- Messaging capabilities must be valid if present
- All events must be valid

### MessagingCapabilities
- At least one broker type is required
- Concurrency and batch size must be positive
- Retry policy must be valid if present

### EventMetadata
- Event type and version are required
- Version must follow semantic versioning
- Deprecated events must include deprecation message
- Schema must be valid if present

### RetryPolicyMetadata
- Max retries cannot be negative
- Initial interval must be positive
- Max interval must be >= initial interval
- Multiplier must be >= 1.0
- Backoff type must be valid

## Integration with Existing Code

This package integrates seamlessly with existing SWIT components:

1. **Transport Layer**: `HandlerMetadata` now includes `MessagingMetadata` field
2. **Service Registration**: Use `ConvertToHandlerMetadata()` to integrate with transport handlers
3. **Discovery**: Metadata can be used for service discovery and capability matching
4. **Validation**: All validation follows SWIT framework patterns

## Examples

Complete examples are available in:
- `pkg/integration/examples/service_metadata_example.go`

These examples demonstrate:
- Creating comprehensive service metadata
- Defining event schemas
- Configuring retry policies
- Integrating with transport layer
- Querying capabilities and patterns

## Testing

Run tests with:

```bash
# Run all integration tests
make test PACKAGE=./pkg/integration

# Run specific metadata tests
go test ./pkg/integration -v -run TestServiceMetadata

# Run with coverage
make test-coverage PACKAGE=./pkg/integration
```

## References

- Issue #437: [Service metadata extensions for messaging](https://github.com/innovationmech/swit/issues/437)
- Parent Issue #177: [Integration Patterns & Advanced Features](https://github.com/innovationmech/swit/issues/177)
- Specification: `specs/event-driven-messaging/integration-patterns.md`

## License

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

See LICENSE file for details.
