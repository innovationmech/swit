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

func TestServiceMetadata_Validate(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ServiceMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata with all fields",
			metadata: &ServiceMetadata{
				Name:        "order-service",
				Version:     "1.0.0",
				Description: "Order processing service",
				MessagingCapabilities: &MessagingCapabilities{
					BrokerTypes:          []messaging.BrokerType{messaging.BrokerTypeKafka},
					SupportsBatching:     true,
					SupportsAsync:        true,
					SupportsTransactions: true,
					MaxConcurrency:       10,
					RetryPolicy: &RetryPolicyMetadata{
						MaxRetries:      3,
						InitialInterval: time.Second,
						BackoffType:     BackoffTypeExponential,
					},
				},
				PublishedEvents: []EventMetadata{
					{
						EventType:   "order.created",
						Version:     "1.0.0",
						Description: "Order created event",
						Topics:      []string{"orders"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid metadata minimal fields",
			metadata: &ServiceMetadata{
				Name:    "simple-service",
				Version: "2.1.0",
			},
			wantErr: false,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			wantErr:  true,
		},
		{
			name: "missing name",
			metadata: &ServiceMetadata{
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			metadata: &ServiceMetadata{
				Name: "test-service",
			},
			wantErr: true,
		},
		{
			name: "invalid version format",
			metadata: &ServiceMetadata{
				Name:    "test-service",
				Version: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid messaging capabilities",
			metadata: &ServiceMetadata{
				Name:    "test-service",
				Version: "1.0.0",
				MessagingCapabilities: &MessagingCapabilities{
					BrokerTypes: []messaging.BrokerType{}, // empty
				},
			},
			wantErr: true,
		},
		{
			name: "invalid published event",
			metadata: &ServiceMetadata{
				Name:    "test-service",
				Version: "1.0.0",
				PublishedEvents: []EventMetadata{
					{
						EventType: "", // empty
						Version:   "1.0.0",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ServiceMetadata.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessagingCapabilities_Validate(t *testing.T) {
	tests := []struct {
		name         string
		capabilities *MessagingCapabilities
		wantErr      bool
	}{
		{
			name: "valid capabilities",
			capabilities: &MessagingCapabilities{
				BrokerTypes:      []messaging.BrokerType{messaging.BrokerTypeKafka, messaging.BrokerTypeRabbitMQ},
				SupportsBatching: true,
				MaxConcurrency:   10,
				MaxBatchSize:     100,
			},
			wantErr: false,
		},
		{
			name:         "nil capabilities",
			capabilities: nil,
			wantErr:      true,
		},
		{
			name: "empty broker types",
			capabilities: &MessagingCapabilities{
				BrokerTypes: []messaging.BrokerType{},
			},
			wantErr: true,
		},
		{
			name: "negative max concurrency",
			capabilities: &MessagingCapabilities{
				BrokerTypes:    []messaging.BrokerType{messaging.BrokerTypeKafka},
				MaxConcurrency: -1,
			},
			wantErr: true,
		},
		{
			name: "negative max batch size",
			capabilities: &MessagingCapabilities{
				BrokerTypes:  []messaging.BrokerType{messaging.BrokerTypeKafka},
				MaxBatchSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid retry policy",
			capabilities: &MessagingCapabilities{
				BrokerTypes: []messaging.BrokerType{messaging.BrokerTypeKafka},
				RetryPolicy: &RetryPolicyMetadata{
					MaxRetries:      -1, // invalid
					InitialInterval: time.Second,
					BackoffType:     BackoffTypeExponential,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.capabilities.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MessagingCapabilities.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEventMetadata_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *EventMetadata
		wantErr bool
	}{
		{
			name: "valid event metadata",
			event: &EventMetadata{
				EventType:   "order.created",
				Version:     "1.0.0",
				Description: "Order created event",
				Topics:      []string{"orders"},
				Schema: &MessageSchema{
					Type:           SchemaTypeJSONSchema,
					ContentType:    "application/json",
					RequiredFields: []string{"orderId", "customerId"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid event minimal fields",
			event: &EventMetadata{
				EventType: "simple.event",
				Version:   "2.0.0",
			},
			wantErr: false,
		},
		{
			name:    "nil event",
			event:   nil,
			wantErr: true,
		},
		{
			name: "missing event type",
			event: &EventMetadata{
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			event: &EventMetadata{
				EventType: "test.event",
			},
			wantErr: true,
		},
		{
			name: "invalid version format",
			event: &EventMetadata{
				EventType: "test.event",
				Version:   "bad-version",
			},
			wantErr: true,
		},
		{
			name: "deprecated without message",
			event: &EventMetadata{
				EventType:  "test.event",
				Version:    "1.0.0",
				Deprecated: true,
			},
			wantErr: true,
		},
		{
			name: "deprecated with message",
			event: &EventMetadata{
				EventType:          "test.event",
				Version:            "1.0.0",
				Deprecated:         true,
				DeprecationMessage: "Use new.event instead",
			},
			wantErr: false,
		},
		{
			name: "invalid schema",
			event: &EventMetadata{
				EventType: "test.event",
				Version:   "1.0.0",
				Schema: &MessageSchema{
					Type: "", // empty type
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EventMetadata.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageSchema_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  *MessageSchema
		wantErr bool
	}{
		{
			name: "valid json schema",
			schema: &MessageSchema{
				Type:        SchemaTypeJSONSchema,
				Format:      "draft-07",
				ContentType: "application/json",
			},
			wantErr: false,
		},
		{
			name: "valid protobuf schema",
			schema: &MessageSchema{
				Type:        SchemaTypeProtobuf,
				ContentType: "application/protobuf",
			},
			wantErr: false,
		},
		{
			name:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			name: "missing type",
			schema: &MessageSchema{
				ContentType: "application/json",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			schema: &MessageSchema{
				Type: "invalid-type",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MessageSchema.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryPolicyMetadata_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  *RetryPolicyMetadata
		wantErr bool
	}{
		{
			name: "valid retry policy",
			policy: &RetryPolicyMetadata{
				MaxRetries:      3,
				InitialInterval: time.Second,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				BackoffType:     BackoffTypeExponential,
			},
			wantErr: false,
		},
		{
			name: "valid constant backoff",
			policy: &RetryPolicyMetadata{
				MaxRetries:      5,
				InitialInterval: time.Second,
				BackoffType:     BackoffTypeConstant,
			},
			wantErr: false,
		},
		{
			name:    "nil policy",
			policy:  nil,
			wantErr: true,
		},
		{
			name: "negative max retries",
			policy: &RetryPolicyMetadata{
				MaxRetries:      -1,
				InitialInterval: time.Second,
				BackoffType:     BackoffTypeExponential,
			},
			wantErr: true,
		},
		{
			name: "negative initial interval",
			policy: &RetryPolicyMetadata{
				MaxRetries:      3,
				InitialInterval: -time.Second,
				BackoffType:     BackoffTypeExponential,
			},
			wantErr: true,
		},
		{
			name: "max interval less than initial",
			policy: &RetryPolicyMetadata{
				MaxRetries:      3,
				InitialInterval: 10 * time.Second,
				MaxInterval:     5 * time.Second,
				BackoffType:     BackoffTypeExponential,
			},
			wantErr: true,
		},
		{
			name: "invalid multiplier",
			policy: &RetryPolicyMetadata{
				MaxRetries:      3,
				InitialInterval: time.Second,
				Multiplier:      0.5, // must be >= 1
				BackoffType:     BackoffTypeExponential,
			},
			wantErr: true,
		},
		{
			name: "invalid backoff type",
			policy: &RetryPolicyMetadata{
				MaxRetries:      3,
				InitialInterval: time.Second,
				BackoffType:     "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryPolicyMetadata.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSemver(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "valid major.minor.patch",
			version: "1.0.0",
			wantErr: false,
		},
		{
			name:    "valid with pre-release",
			version: "1.0.0-alpha",
			wantErr: false,
		},
		{
			name:    "valid with pre-release and number",
			version: "1.0.0-beta.1",
			wantErr: false,
		},
		{
			name:    "valid with build metadata",
			version: "1.0.0+20130313144700",
			wantErr: false,
		},
		{
			name:    "valid with pre-release and build",
			version: "1.0.0-rc.1+20130313144700",
			wantErr: false,
		},
		{
			name:    "valid large version numbers",
			version: "10.20.30",
			wantErr: false,
		},
		{
			name:    "invalid format",
			version: "invalid",
			wantErr: true,
		},
		{
			name:    "invalid missing patch",
			version: "1.0",
			wantErr: true,
		},
		{
			name:    "invalid leading zeros",
			version: "01.0.0",
			wantErr: true,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSemver(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSemver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceMetadata_HasCapability(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		MessagingCapabilities: &MessagingCapabilities{
			BrokerTypes:          []messaging.BrokerType{messaging.BrokerTypeKafka},
			SupportsBatching:     true,
			SupportsAsync:        true,
			SupportsTransactions: false,
			DeadLetterSupport:    true,
		},
	}

	tests := []struct {
		name       string
		capability string
		want       bool
	}{
		{
			name:       "has batching capability",
			capability: "batching",
			want:       true,
		},
		{
			name:       "has async capability",
			capability: "async",
			want:       true,
		},
		{
			name:       "does not have transactions capability",
			capability: "transactions",
			want:       false,
		},
		{
			name:       "has dead_letter capability",
			capability: "dead_letter",
			want:       true,
		},
		{
			name:       "unknown capability",
			capability: "unknown",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metadata.HasCapability(tt.capability); got != tt.want {
				t.Errorf("ServiceMetadata.HasCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMetadata_HasPattern(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		PatternSupport: PatternSupport{
			Saga:           true,
			Outbox:         true,
			Inbox:          false,
			CQRS:           true,
			EventSourcing:  false,
			CircuitBreaker: true,
		},
	}

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{
			name:    "has saga pattern",
			pattern: "saga",
			want:    true,
		},
		{
			name:    "has outbox pattern",
			pattern: "outbox",
			want:    true,
		},
		{
			name:    "does not have inbox pattern",
			pattern: "inbox",
			want:    false,
		},
		{
			name:    "has cqrs pattern",
			pattern: "cqrs",
			want:    true,
		},
		{
			name:    "unknown pattern",
			pattern: "unknown",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metadata.HasPattern(tt.pattern); got != tt.want {
				t.Errorf("ServiceMetadata.HasPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMetadata_GetPublishedEvent(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		PublishedEvents: []EventMetadata{
			{
				EventType: "order.created",
				Version:   "1.0.0",
			},
			{
				EventType: "order.updated",
				Version:   "1.0.0",
			},
		},
	}

	tests := []struct {
		name      string
		eventType string
		wantNil   bool
	}{
		{
			name:      "finds existing event",
			eventType: "order.created",
			wantNil:   false,
		},
		{
			name:      "does not find non-existent event",
			eventType: "order.deleted",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadata.GetPublishedEvent(tt.eventType)
			if (got == nil) != tt.wantNil {
				t.Errorf("ServiceMetadata.GetPublishedEvent() = %v, wantNil %v", got, tt.wantNil)
			}
			if !tt.wantNil && got.EventType != tt.eventType {
				t.Errorf("ServiceMetadata.GetPublishedEvent() eventType = %v, want %v", got.EventType, tt.eventType)
			}
		})
	}
}

func TestServiceMetadata_GetSubscribedEvent(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		SubscribedEvents: []EventMetadata{
			{
				EventType: "payment.completed",
				Version:   "1.0.0",
			},
			{
				EventType: "inventory.updated",
				Version:   "1.0.0",
			},
		},
	}

	tests := []struct {
		name      string
		eventType string
		wantNil   bool
	}{
		{
			name:      "finds existing event",
			eventType: "payment.completed",
			wantNil:   false,
		},
		{
			name:      "does not find non-existent event",
			eventType: "shipping.dispatched",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadata.GetSubscribedEvent(tt.eventType)
			if (got == nil) != tt.wantNil {
				t.Errorf("ServiceMetadata.GetSubscribedEvent() = %v, wantNil %v", got, tt.wantNil)
			}
			if !tt.wantNil && got.EventType != tt.eventType {
				t.Errorf("ServiceMetadata.GetSubscribedEvent() eventType = %v, want %v", got.EventType, tt.eventType)
			}
		})
	}
}

func TestServiceMetadata_SupportsBrokerType(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		MessagingCapabilities: &MessagingCapabilities{
			BrokerTypes: []messaging.BrokerType{
				messaging.BrokerTypeKafka,
				messaging.BrokerTypeRabbitMQ,
			},
		},
	}

	tests := []struct {
		name       string
		brokerType messaging.BrokerType
		want       bool
	}{
		{
			name:       "supports kafka",
			brokerType: messaging.BrokerTypeKafka,
			want:       true,
		},
		{
			name:       "supports rabbitmq",
			brokerType: messaging.BrokerTypeRabbitMQ,
			want:       true,
		},
		{
			name:       "does not support nats",
			brokerType: messaging.BrokerTypeNATS,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metadata.SupportsBrokerType(tt.brokerType); got != tt.want {
				t.Errorf("ServiceMetadata.SupportsBrokerType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMetadata_NilCapabilities(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:    "test-service",
		Version: "1.0.0",
		// No messaging capabilities
	}

	if metadata.HasCapability("batching") {
		t.Error("Expected false for nil capabilities")
	}

	if metadata.SupportsBrokerType(messaging.BrokerTypeKafka) {
		t.Error("Expected false for nil capabilities")
	}
}
