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

package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/adapters"
)

// KafkaAdapter implements messaging.MessageBrokerAdapter based on segmentio/kafka-go.
type KafkaAdapter struct {
	*adapters.BaseMessageBrokerAdapter
}

// newAdapter constructs a Kafka adapter with metadata, capabilities and default config.
func newAdapter() *KafkaAdapter {
	info := &messaging.BrokerAdapterInfo{
		Name:                 "kafka-go",
		Version:              "0.1.0",
		Description:          "Kafka adapter built on segmentio/kafka-go",
		SupportedBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeKafka},
		Author:               "SWIT",
		License:              "MIT",
	}

	// Use built-in capability profile for Kafka
	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeKafka)

	// Provide a safe default configuration template
	defaultCfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}

	return &KafkaAdapter{BaseMessageBrokerAdapter: adapters.NewBaseMessageBrokerAdapter(info, caps, defaultCfg)}
}

// CreateBroker implements MessageBrokerAdapter.CreateBroker.
func (a *KafkaAdapter) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	// Base validation (type/endpoints) first
	if res := a.ValidateConfiguration(config); !res.Valid {
		return nil, messaging.NewConfigError("kafka adapter configuration validation failed", nil)
	}

	if config.Type != messaging.BrokerTypeKafka {
		return nil, messaging.NewConfigError(fmt.Sprintf("unsupported broker type for kafka adapter: %s", config.Type), nil)
	}

	broker := newKafkaBroker(config)
	return broker, nil
}

// ValidateConfiguration augments base validation with Kafka specifics (timeouts defaults etc.).
func (a *KafkaAdapter) ValidateConfiguration(config *messaging.BrokerConfig) *messaging.AdapterValidationResult {
	res := a.BaseMessageBrokerAdapter.ValidateConfiguration(config)
	if !res.Valid {
		return res
	}
	// Suggest sensible defaults if missing
	if config != nil {
		if config.Connection.Timeout <= 0 {
			res.Suggestions = append(res.Suggestions, messaging.AdapterValidationSuggestion{
				Field:          "Connection.Timeout",
				Message:        "use a positive timeout",
				SuggestedValue: 10 * time.Second,
			})
		}
	}
	return res
}

// HealthCheck validates adapter wiring (not broker connectivity).
func (a *KafkaAdapter) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status, err := a.BaseMessageBrokerAdapter.HealthCheck(ctx)
	if status != nil {
		status.Details["library"] = "segmentio/kafka-go"
	}
	return status, err
}

// init registers the Kafka adapter and wires the adapter registry into factory.
func init() {
	// Ensure adapter registry is connected (handled in adapters package init). Then register adapter.
	_ = adapters.RegisterGlobalAdapter(newAdapter())
}
