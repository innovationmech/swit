// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package examples

import (
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/integration"
	"github.com/innovationmech/swit/pkg/messaging"
)

// ExampleOrderServiceMetadata demonstrates how to create comprehensive
// service metadata for an order processing service with messaging capabilities.
func ExampleOrderServiceMetadata() *integration.ServiceMetadata {
	// Define messaging capabilities
	messagingCapabilities := integration.NewMessagingCapabilitiesBuilder().
		WithBrokerType(messaging.BrokerTypeKafka).
		WithBrokerType(messaging.BrokerTypeRabbitMQ).
		WithBatching(true).
		WithAsync(true).
		WithTransactions(true).
		WithOrdering(true).
		WithCompression(messaging.CompressionSnappy).
		WithCompression(messaging.CompressionGZIP).
		WithSerialization("json").
		WithSerialization("protobuf").
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

	// Define published events
	orderCreatedEvent := integration.EventMetadata{
		EventType:   "order.created",
		Version:     "1.0.0",
		Description: "Emitted when a new order is created",
		Schema: &integration.MessageSchema{
			Type:        integration.SchemaTypeJSONSchema,
			ContentType: "application/json",
			RequiredFields: []string{
				"orderId",
				"customerId",
				"items",
				"totalAmount",
				"createdAt",
			},
			ValidationRules: map[string]interface{}{
				"orderId":     map[string]string{"type": "uuid"},
				"customerId":  map[string]string{"type": "uuid"},
				"totalAmount": map[string]interface{}{"type": "number", "minimum": 0},
			},
		},
		Topics:   []string{"orders", "order.events"},
		Required: true,
		Examples: []string{
			`{"orderId": "123e4567-e89b-12d3-a456-426614174000", "customerId": "uuid", "items": [...], "totalAmount": 99.99}`,
		},
	}

	orderUpdatedEvent := integration.EventMetadata{
		EventType:   "order.updated",
		Version:     "1.0.0",
		Description: "Emitted when an order is updated",
		Schema: &integration.MessageSchema{
			Type:           integration.SchemaTypeJSONSchema,
			ContentType:    "application/json",
			RequiredFields: []string{"orderId", "updatedFields", "updatedAt"},
		},
		Topics:   []string{"orders", "order.events"},
		Required: false,
	}

	// Define subscribed events
	paymentCompletedEvent := integration.EventMetadata{
		EventType:   "payment.completed",
		Version:     "1.0.0",
		Description: "Payment confirmation from payment service",
		Topics:      []string{"payment.events"},
		Required:    true,
	}

	inventoryReservedEvent := integration.EventMetadata{
		EventType:   "inventory.reserved",
		Version:     "1.0.0",
		Description: "Inventory reservation confirmation",
		Topics:      []string{"inventory.events"},
		Required:    true,
	}

	// Define pattern support
	patternSupport := integration.PatternSupport{
		Saga:           true, // Implements saga for distributed transactions
		Outbox:         true, // Uses outbox pattern for reliable publishing
		Inbox:          true, // Uses inbox pattern for idempotent processing
		CQRS:           false,
		EventSourcing:  false,
		CircuitBreaker: true, // Implements circuit breaker for resilience
		BulkOperations: true, // Supports bulk order processing
	}

	// Build complete service metadata
	metadata, err := integration.NewServiceMetadataBuilder("order-service", "2.1.0").
		WithDescription("Order processing and management service").
		WithMessagingCapabilities(messagingCapabilities).
		WithPublishedEvent(orderCreatedEvent).
		WithPublishedEvent(orderUpdatedEvent).
		WithSubscribedEvent(paymentCompletedEvent).
		WithSubscribedEvent(inventoryReservedEvent).
		WithPatternSupport(patternSupport).
		WithTag("core").
		WithTag("transactional").
		WithTag("event-driven").
		WithDependency("payment-service").
		WithDependency("inventory-service").
		WithDependency("notification-service").
		WithExtended("region", "us-east-1").
		WithExtended("team", "checkout").
		Build()

	if err != nil {
		panic(fmt.Sprintf("failed to build service metadata: %v", err))
	}

	return metadata
}

// ExampleSimpleServiceMetadata demonstrates minimal service metadata
// for a simple messaging service.
func ExampleSimpleServiceMetadata() *integration.ServiceMetadata {
	// Minimal capabilities
	capabilities := integration.NewMessagingCapabilitiesBuilder().
		WithBrokerType(messaging.BrokerTypeInMemory).
		WithAsync(true).
		MustBuild()

	// Simple event definition
	simpleEvent := integration.NewEventMetadata("notification.sent", "1.0.0")
	simpleEvent.Description = "Notification sent to user"
	simpleEvent.Topics = []string{"notifications"}

	// Build minimal metadata
	metadata, err := integration.NewServiceMetadataBuilder("notification-service", "1.0.0").
		WithDescription("Simple notification service").
		WithMessagingCapabilities(capabilities).
		WithPublishedEvent(simpleEvent).
		Build()

	if err != nil {
		panic(fmt.Sprintf("failed to build service metadata: %v", err))
	}

	return metadata
}

// ExampleValidateServiceMetadata demonstrates how to validate service metadata
// and handle validation errors.
func ExampleValidateServiceMetadata() {
	// Create metadata with potential issues
	metadata := &integration.ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		MessagingCapabilities: &integration.MessagingCapabilities{
			BrokerTypes: []messaging.BrokerType{messaging.BrokerTypeKafka},
		},
	}

	// Validate metadata
	if err := metadata.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		return
	}

	fmt.Println("Service metadata is valid")

	// Check specific capabilities
	if metadata.HasCapability("batching") {
		fmt.Println("Service supports batching")
	}

	if metadata.HasPattern("saga") {
		fmt.Println("Service implements saga pattern")
	}

	// Query event metadata
	if event := metadata.GetPublishedEvent("order.created"); event != nil {
		fmt.Printf("Found published event: %s (version %s)\n", event.EventType, event.Version)
	}
}

// ExampleIntegrationWithHandlerMetadata demonstrates converting between
// ServiceMetadata and transport.HandlerMetadata for framework integration.
func ExampleIntegrationWithHandlerMetadata() {
	// Create service metadata
	serviceMetadata := ExampleSimpleServiceMetadata()

	// Convert to handler metadata for transport layer
	handlerMetadata := integration.ConvertToHandlerMetadata(serviceMetadata, "/health")

	fmt.Printf("Service: %s (version %s)\n", handlerMetadata.Name, handlerMetadata.Version)
	fmt.Printf("Health endpoint: %s\n", handlerMetadata.HealthEndpoint)

	// Extract service metadata from handler metadata
	extractedMetadata := integration.ExtractServiceMetadata(handlerMetadata)
	if extractedMetadata != nil {
		fmt.Printf("Messaging capabilities: %v\n", extractedMetadata.MessagingCapabilities != nil)
		fmt.Printf("Published events: %d\n", len(extractedMetadata.PublishedEvents))
	}
}

// ExampleQueryServiceCapabilities demonstrates querying service capabilities
// and patterns for runtime decisions.
func ExampleQueryServiceCapabilities() {
	metadata := ExampleOrderServiceMetadata()

	// Check broker support
	if metadata.SupportsBrokerType(messaging.BrokerTypeKafka) {
		fmt.Println("Service supports Kafka")
	}

	if metadata.SupportsBrokerType(messaging.BrokerTypeNATS) {
		fmt.Println("Service supports NATS")
	} else {
		fmt.Println("Service does not support NATS")
	}

	// Check capabilities
	capabilities := []string{"batching", "async", "transactions", "ordering"}
	for _, cap := range capabilities {
		if metadata.HasCapability(cap) {
			fmt.Printf("✓ Capability: %s\n", cap)
		} else {
			fmt.Printf("✗ Capability: %s\n", cap)
		}
	}

	// Check patterns
	patterns := []string{"saga", "outbox", "inbox", "cqrs"}
	for _, pattern := range patterns {
		if metadata.HasPattern(pattern) {
			fmt.Printf("✓ Pattern: %s\n", pattern)
		} else {
			fmt.Printf("✗ Pattern: %s\n", pattern)
		}
	}
}
