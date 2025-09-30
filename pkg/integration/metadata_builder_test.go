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

package integration

import (
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestServiceMetadataBuilder(t *testing.T) {
	t.Run("build valid metadata", func(t *testing.T) {
		metadata, err := NewServiceMetadataBuilder("test-service", "1.0.0").
			WithDescription("Test service").
			WithTag("backend").
			WithDependency("auth-service").
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if metadata.Name != "test-service" {
			t.Errorf("expected name 'test-service', got %s", metadata.Name)
		}

		if metadata.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %s", metadata.Version)
		}

		if metadata.Description != "Test service" {
			t.Errorf("expected description 'Test service', got %s", metadata.Description)
		}

		if len(metadata.Tags) != 1 || metadata.Tags[0] != "backend" {
			t.Errorf("expected tag 'backend', got %v", metadata.Tags)
		}

		if len(metadata.Dependencies) != 1 || metadata.Dependencies[0] != "auth-service" {
			t.Errorf("expected dependency 'auth-service', got %v", metadata.Dependencies)
		}
	})

	t.Run("build with messaging capabilities", func(t *testing.T) {
		capabilities, err := NewMessagingCapabilitiesBuilder().
			WithBrokerType(messaging.BrokerTypeKafka).
			WithBatching(true).
			WithAsync(true).
			WithConcurrency(10).
			Build()

		if err != nil {
			t.Fatalf("expected no error building capabilities, got %v", err)
		}

		metadata, err := NewServiceMetadataBuilder("messaging-service", "2.0.0").
			WithMessagingCapabilities(capabilities).
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if metadata.MessagingCapabilities == nil {
			t.Fatal("expected messaging capabilities to be set")
		}

		if !metadata.MessagingCapabilities.SupportsBatching {
			t.Error("expected batching support")
		}

		if !metadata.MessagingCapabilities.SupportsAsync {
			t.Error("expected async support")
		}
	})

	t.Run("build with events", func(t *testing.T) {
		metadata, err := NewServiceMetadataBuilder("event-service", "1.0.0").
			WithPublishedEvent(EventMetadata{
				EventType: "order.created",
				Version:   "1.0.0",
			}).
			WithSubscribedEvent(EventMetadata{
				EventType: "payment.completed",
				Version:   "1.0.0",
			}).
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(metadata.PublishedEvents) != 1 {
			t.Errorf("expected 1 published event, got %d", len(metadata.PublishedEvents))
		}

		if len(metadata.SubscribedEvents) != 1 {
			t.Errorf("expected 1 subscribed event, got %d", len(metadata.SubscribedEvents))
		}
	})

	t.Run("build with pattern support", func(t *testing.T) {
		patterns := PatternSupport{
			Saga:           true,
			Outbox:         true,
			CircuitBreaker: true,
		}

		metadata, err := NewServiceMetadataBuilder("pattern-service", "1.0.0").
			WithPatternSupport(patterns).
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !metadata.PatternSupport.Saga {
			t.Error("expected saga pattern support")
		}

		if !metadata.PatternSupport.Outbox {
			t.Error("expected outbox pattern support")
		}
	})

	t.Run("build with invalid version", func(t *testing.T) {
		_, err := NewServiceMetadataBuilder("invalid-service", "not-semver").
			Build()

		if err == nil {
			t.Error("expected error for invalid version")
		}
	})

	t.Run("MustBuild panics on error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid metadata")
			}
		}()

		NewServiceMetadataBuilder("invalid", "bad-version").
			MustBuild()
	})
}

func TestMessagingCapabilitiesBuilder(t *testing.T) {
	t.Run("build valid capabilities", func(t *testing.T) {
		capabilities, err := NewMessagingCapabilitiesBuilder().
			WithBrokerType(messaging.BrokerTypeKafka).
			WithBrokerType(messaging.BrokerTypeRabbitMQ).
			WithBatching(true).
			WithAsync(true).
			WithTransactions(true).
			WithOrdering(true).
			WithCompression(messaging.CompressionSnappy).
			WithSerialization("json").
			WithConcurrency(20).
			WithBatchSize(100).
			WithProcessingTimeout(30 * time.Second).
			WithDeadLetterSupport(true).
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(capabilities.BrokerTypes) != 2 {
			t.Errorf("expected 2 broker types, got %d", len(capabilities.BrokerTypes))
		}

		if !capabilities.SupportsBatching {
			t.Error("expected batching support")
		}

		if capabilities.MaxConcurrency != 20 {
			t.Errorf("expected max concurrency 20, got %d", capabilities.MaxConcurrency)
		}

		if capabilities.MaxBatchSize != 100 {
			t.Errorf("expected max batch size 100, got %d", capabilities.MaxBatchSize)
		}
	})

	t.Run("build with retry policy", func(t *testing.T) {
		retryPolicy := &RetryPolicyMetadata{
			MaxRetries:      3,
			InitialInterval: time.Second,
			BackoffType:     BackoffTypeExponential,
			Multiplier:      2.0,
		}

		capabilities, err := NewMessagingCapabilitiesBuilder().
			WithBrokerType(messaging.BrokerTypeKafka).
			WithRetryPolicy(retryPolicy).
			Build()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if capabilities.RetryPolicy == nil {
			t.Fatal("expected retry policy to be set")
		}

		if capabilities.RetryPolicy.MaxRetries != 3 {
			t.Errorf("expected max retries 3, got %d", capabilities.RetryPolicy.MaxRetries)
		}
	})

	t.Run("build without broker types", func(t *testing.T) {
		_, err := NewMessagingCapabilitiesBuilder().
			WithBatching(true).
			Build()

		if err == nil {
			t.Error("expected error for missing broker types")
		}
	})

	t.Run("MustBuild panics on error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid capabilities")
			}
		}()

		NewMessagingCapabilitiesBuilder().
			MustBuild()
	})
}

func TestConvertToHandlerMetadata(t *testing.T) {
	t.Run("convert valid service metadata", func(t *testing.T) {
		serviceMetadata := &ServiceMetadata{
			Name:         "test-service",
			Version:      "1.0.0",
			Description:  "Test service",
			Tags:         []string{"backend"},
			Dependencies: []string{"auth-service"},
		}

		handlerMetadata := ConvertToHandlerMetadata(serviceMetadata, "/health")

		if handlerMetadata == nil {
			t.Fatal("expected handler metadata, got nil")
		}

		if handlerMetadata.Name != "test-service" {
			t.Errorf("expected name 'test-service', got %s", handlerMetadata.Name)
		}

		if handlerMetadata.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %s", handlerMetadata.Version)
		}

		if handlerMetadata.HealthEndpoint != "/health" {
			t.Errorf("expected health endpoint '/health', got %s", handlerMetadata.HealthEndpoint)
		}

		if handlerMetadata.MessagingMetadata == nil {
			t.Error("expected messaging metadata to be set")
		}
	})

	t.Run("convert nil metadata", func(t *testing.T) {
		handlerMetadata := ConvertToHandlerMetadata(nil, "/health")

		if handlerMetadata != nil {
			t.Error("expected nil for nil input")
		}
	})
}

func TestExtractServiceMetadata(t *testing.T) {
	t.Run("extract valid service metadata", func(t *testing.T) {
		originalMetadata := &ServiceMetadata{
			Name:    "test-service",
			Version: "1.0.0",
		}

		handlerMetadata := ConvertToHandlerMetadata(originalMetadata, "/health")
		extractedMetadata := ExtractServiceMetadata(handlerMetadata)

		if extractedMetadata == nil {
			t.Fatal("expected service metadata, got nil")
		}

		if extractedMetadata.Name != originalMetadata.Name {
			t.Errorf("expected name %s, got %s", originalMetadata.Name, extractedMetadata.Name)
		}

		if extractedMetadata.Version != originalMetadata.Version {
			t.Errorf("expected version %s, got %s", originalMetadata.Version, extractedMetadata.Version)
		}
	})

	t.Run("extract from nil handler metadata", func(t *testing.T) {
		extractedMetadata := ExtractServiceMetadata(nil)

		if extractedMetadata != nil {
			t.Error("expected nil for nil input")
		}
	})
}

func TestNewRetryPolicy(t *testing.T) {
	tests := []struct {
		name            string
		maxRetries      int
		initialInterval time.Duration
		backoffType     BackoffType
	}{
		{
			name:            "exponential backoff",
			maxRetries:      3,
			initialInterval: time.Second,
			backoffType:     BackoffTypeExponential,
		},
		{
			name:            "linear backoff",
			maxRetries:      5,
			initialInterval: 2 * time.Second,
			backoffType:     BackoffTypeLinear,
		},
		{
			name:            "constant backoff",
			maxRetries:      10,
			initialInterval: 500 * time.Millisecond,
			backoffType:     BackoffTypeConstant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := NewRetryPolicy(tt.maxRetries, tt.initialInterval, tt.backoffType)

			if policy == nil {
				t.Fatal("expected retry policy, got nil")
			}

			if policy.MaxRetries != tt.maxRetries {
				t.Errorf("expected max retries %d, got %d", tt.maxRetries, policy.MaxRetries)
			}

			if policy.InitialInterval != tt.initialInterval {
				t.Errorf("expected initial interval %v, got %v", tt.initialInterval, policy.InitialInterval)
			}

			if policy.BackoffType != tt.backoffType {
				t.Errorf("expected backoff type %v, got %v", tt.backoffType, policy.BackoffType)
			}

			if policy.MaxInterval <= 0 {
				t.Error("expected max interval to be set")
			}
		})
	}
}

func TestNewEventMetadata(t *testing.T) {
	event := NewEventMetadata("order.created", "1.0.0")

	if event.EventType != "order.created" {
		t.Errorf("expected event type 'order.created', got %s", event.EventType)
	}

	if event.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", event.Version)
	}

	if event.Topics == nil {
		t.Error("expected topics to be initialized")
	}

	if event.Examples == nil {
		t.Error("expected examples to be initialized")
	}
}

func TestNewMessageSchema(t *testing.T) {
	tests := []struct {
		name       string
		schemaType SchemaType
	}{
		{
			name:       "json schema",
			schemaType: SchemaTypeJSONSchema,
		},
		{
			name:       "protobuf schema",
			schemaType: SchemaTypeProtobuf,
		},
		{
			name:       "avro schema",
			schemaType: SchemaTypeAvro,
		},
		{
			name:       "custom schema",
			schemaType: SchemaTypeCustom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewMessageSchema(tt.schemaType)

			if schema == nil {
				t.Fatal("expected message schema, got nil")
			}

			if schema.Type != tt.schemaType {
				t.Errorf("expected schema type %v, got %v", tt.schemaType, schema.Type)
			}

			if schema.RequiredFields == nil {
				t.Error("expected required fields to be initialized")
			}

			if schema.ValidationRules == nil {
				t.Error("expected validation rules to be initialized")
			}
		})
	}
}
