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

package adapters

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

// MockMessageBrokerAdapter implements MessageBrokerAdapter for testing.
type MockMessageBrokerAdapter struct {
	info          *messaging.BrokerAdapterInfo
	capabilities  *messaging.BrokerCapabilities
	defaultConfig *messaging.BrokerConfig
	createError   error
	validateError error
}

func NewMockMessageBrokerAdapter(name string, brokerTypes []messaging.BrokerType) *MockMessageBrokerAdapter {
	return &MockMessageBrokerAdapter{
		info: &messaging.BrokerAdapterInfo{
			Name:                 name,
			Version:              "1.0.0",
			Description:          "Mock adapter for testing",
			SupportedBrokerTypes: brokerTypes,
		},
		capabilities: &messaging.BrokerCapabilities{
			SupportsConsumerGroups:      true,
			SupportedCompressionTypes:   []messaging.CompressionType{messaging.CompressionNone},
			SupportedSerializationTypes: []messaging.SerializationType{messaging.SerializationJSON},
		},
		defaultConfig: &messaging.BrokerConfig{
			Type:      messaging.BrokerTypeInMemory,
			Endpoints: []string{"localhost:9092"},
		},
	}
}

func (m *MockMessageBrokerAdapter) GetAdapterInfo() *messaging.BrokerAdapterInfo {
	return m.info
}

func (m *MockMessageBrokerAdapter) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	return &MockMessageBroker{}, nil
}

func (m *MockMessageBrokerAdapter) ValidateConfiguration(config *messaging.BrokerConfig) *messaging.AdapterValidationResult {
	result := &messaging.AdapterValidationResult{Valid: true}
	if m.validateError != nil {
		result.Valid = false
		result.Errors = []messaging.AdapterValidationError{{
			Message: m.validateError.Error(),
		}}
	}
	return result
}

func (m *MockMessageBrokerAdapter) GetCapabilities() *messaging.BrokerCapabilities {
	return m.capabilities
}

func (m *MockMessageBrokerAdapter) GetDefaultConfiguration() *messaging.BrokerConfig {
	return m.defaultConfig
}

func (m *MockMessageBrokerAdapter) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	return &messaging.HealthStatus{
		Status:  messaging.HealthStatusHealthy,
		Message: "Mock adapter is healthy",
	}, nil
}

// MockMessageBroker implements MessageBroker for testing.
type MockMessageBroker struct{}

func (m *MockMessageBroker) Connect(ctx context.Context) error    { return nil }
func (m *MockMessageBroker) Disconnect(ctx context.Context) error { return nil }
func (m *MockMessageBroker) Close() error                         { return nil }
func (m *MockMessageBroker) IsConnected() bool                    { return true }
func (m *MockMessageBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	return nil, nil
}
func (m *MockMessageBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	return nil, nil
}
func (m *MockMessageBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	return &messaging.HealthStatus{Status: messaging.HealthStatusHealthy}, nil
}
func (m *MockMessageBroker) GetMetrics() *messaging.BrokerMetrics {
	return &messaging.BrokerMetrics{}
}
func (m *MockMessageBroker) GetCapabilities() *messaging.BrokerCapabilities {
	return &messaging.BrokerCapabilities{}
}

func TestBaseMessageBrokerAdapter(t *testing.T) {
	info := &messaging.BrokerAdapterInfo{
		Name:                 "test-adapter",
		Version:              "1.0.0",
		SupportedBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeInMemory},
	}

	capabilities := &messaging.BrokerCapabilities{
		SupportsConsumerGroups: true,
	}

	defaultConfig := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	adapter := NewBaseMessageBrokerAdapter(info, capabilities, defaultConfig)

	t.Run("GetAdapterInfo", func(t *testing.T) {
		result := adapter.GetAdapterInfo()
		if result.Name != "test-adapter" {
			t.Errorf("Expected adapter name 'test-adapter', got '%s'", result.Name)
		}
		if result.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", result.Version)
		}
	})

	t.Run("GetCapabilities", func(t *testing.T) {
		result := adapter.GetCapabilities()
		if !result.SupportsConsumerGroups {
			t.Error("Expected SupportsConsumerGroups to be true")
		}
	})

	t.Run("GetDefaultConfiguration", func(t *testing.T) {
		result := adapter.GetDefaultConfiguration()
		if result.Type != messaging.BrokerTypeInMemory {
			t.Errorf("Expected type InMemory, got %s", result.Type)
		}
	})

	t.Run("ValidateConfiguration", func(t *testing.T) {
		// Use the adapter's default config as a starting point for a valid config
		validConfig := adapter.GetDefaultConfiguration()
		validConfig.Type = messaging.BrokerTypeInMemory

		result := adapter.ValidateConfiguration(validConfig)
		if !result.Valid {
			t.Errorf("Expected valid configuration, got errors: %+v", result.Errors)
		}

		// Test nil config
		result = adapter.ValidateConfiguration(nil)
		if result.Valid {
			t.Error("Expected nil configuration to be invalid")
		}

		// Test unsupported type
		invalidConfig := adapter.GetDefaultConfiguration()
		invalidConfig.Type = messaging.BrokerTypeKafka // Not supported by test adapter

		result = adapter.ValidateConfiguration(invalidConfig)
		if result.Valid {
			t.Error("Expected invalid configuration for unsupported type")
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		status, err := adapter.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if status.Status != messaging.HealthStatusHealthy {
			t.Errorf("Expected healthy status, got %s", status.Status)
		}
	})

	t.Run("CreateBroker", func(t *testing.T) {
		config := &messaging.BrokerConfig{
			Type:      messaging.BrokerTypeKafka, // Unsupported type
			Endpoints: []string{"localhost:9092"},
		}
		_, err := adapter.CreateBroker(config)
		if err == nil {
			t.Error("Expected error for unsupported broker type")
		}
	})
}

func TestBrokerAdapterRegistry(t *testing.T) {
	registry := NewBrokerAdapterRegistry()

	// Create mock adapters
	kafkaAdapter := NewMockMessageBrokerAdapter("kafka-adapter", []messaging.BrokerType{messaging.BrokerTypeKafka})
	natsAdapter := NewMockMessageBrokerAdapter("nats-adapter", []messaging.BrokerType{messaging.BrokerTypeNATS})

	t.Run("RegisterAdapter", func(t *testing.T) {
		err := registry.RegisterAdapter(kafkaAdapter)
		if err != nil {
			t.Errorf("Unexpected error registering adapter: %v", err)
		}

		err = registry.RegisterAdapter(natsAdapter)
		if err != nil {
			t.Errorf("Unexpected error registering adapter: %v", err)
		}

		// Test duplicate registration
		err = registry.RegisterAdapter(kafkaAdapter)
		if err == nil {
			t.Error("Expected error for duplicate adapter registration")
		}
	})

	t.Run("GetAdapter", func(t *testing.T) {
		adapter, err := registry.GetAdapter("kafka-adapter")
		if err != nil {
			t.Errorf("Unexpected error getting adapter: %v", err)
		}
		if adapter.GetAdapterInfo().Name != "kafka-adapter" {
			t.Error("Got wrong adapter")
		}

		// Test non-existent adapter
		_, err = registry.GetAdapter("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent adapter")
		}
	})

	t.Run("GetAdapterForBrokerType", func(t *testing.T) {
		adapter, err := registry.GetAdapterForBrokerType(messaging.BrokerTypeKafka)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if adapter.GetAdapterInfo().Name != "kafka-adapter" {
			t.Error("Got wrong adapter for Kafka")
		}

		// Test unsupported broker type
		_, err = registry.GetAdapterForBrokerType(messaging.BrokerTypeRabbitMQ)
		if err == nil {
			t.Error("Expected error for unsupported broker type")
		}
	})

	t.Run("ListRegisteredAdapters", func(t *testing.T) {
		adapters := registry.ListRegisteredAdapters()
		if len(adapters) != 2 {
			t.Errorf("Expected 2 adapters, got %d", len(adapters))
		}
	})

	t.Run("GetSupportedBrokerTypes", func(t *testing.T) {
		types := registry.GetSupportedBrokerTypes()
		if len(types) != 2 {
			t.Errorf("Expected 2 broker types, got %d", len(types))
		}
	})

	t.Run("CreateBroker", func(t *testing.T) {
		config := &messaging.BrokerConfig{
			Type:      messaging.BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
		}
		broker, err := registry.CreateBroker(config)
		if err != nil {
			t.Errorf("Unexpected error creating broker: %v", err)
		}
		if broker == nil {
			t.Error("Expected broker to be created")
		}
	})

	t.Run("ValidateConfiguration", func(t *testing.T) {
		config := &messaging.BrokerConfig{
			Type:      messaging.BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
		}
		result, err := registry.ValidateConfiguration(config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !result.Valid {
			t.Error("Expected configuration to be valid")
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		health, err := registry.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(health) != 2 {
			t.Errorf("Expected health for 2 adapters, got %d", len(health))
		}
	})

	t.Run("UnregisterAdapter", func(t *testing.T) {
		err := registry.UnregisterAdapter("kafka-adapter")
		if err != nil {
			t.Errorf("Unexpected error unregistering adapter: %v", err)
		}

		// Verify adapter is gone
		_, err = registry.GetAdapter("kafka-adapter")
		if err == nil {
			t.Error("Expected error after unregistering adapter")
		}

		// Test unregistering non-existent adapter
		err = registry.UnregisterAdapter("nonexistent")
		if err == nil {
			t.Error("Expected error unregistering non-existent adapter")
		}
	})
}

func TestBrokerSelector(t *testing.T) {
	registry := NewBrokerAdapterRegistry()

	// Register test adapters
	kafkaAdapter := NewMockMessageBrokerAdapter("kafka-adapter", []messaging.BrokerType{messaging.BrokerTypeKafka})
	kafkaAdapter.capabilities.SupportsTransactions = true
	kafkaAdapter.capabilities.SupportsOrdering = true

	natsAdapter := NewMockMessageBrokerAdapter("nats-adapter", []messaging.BrokerType{messaging.BrokerTypeNATS})
	natsAdapter.capabilities.SupportsTransactions = false

	registry.RegisterAdapter(kafkaAdapter)
	registry.RegisterAdapter(natsAdapter)

	selector := NewBrokerSelector(registry)

	t.Run("SelectBroker_PreferredType", func(t *testing.T) {
		criteria := SelectionCriteria{
			PreferredBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeNATS, messaging.BrokerTypeKafka},
		}

		result, err := selector.SelectBroker(criteria)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.SelectedBrokerType != messaging.BrokerTypeNATS {
			t.Errorf("Expected NATS (preferred), got %s", result.SelectedBrokerType)
		}
	})

	t.Run("SelectBroker_RequiredCapability", func(t *testing.T) {
		criteria := SelectionCriteria{
			RequiredCapabilities: []CapabilityRequirement{
				{Name: "transactions", Required: true},
			},
		}

		result, err := selector.SelectBroker(criteria)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.SelectedBrokerType != messaging.BrokerTypeKafka {
			t.Errorf("Expected Kafka (supports transactions), got %s", result.SelectedBrokerType)
		}
	})

	t.Run("SelectBroker_ExcludedType", func(t *testing.T) {
		criteria := SelectionCriteria{
			ExcludedBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeKafka},
		}

		result, err := selector.SelectBroker(criteria)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.SelectedBrokerType != messaging.BrokerTypeNATS {
			t.Errorf("Expected NATS (Kafka excluded), got %s", result.SelectedBrokerType)
		}
	})

	t.Run("SelectBroker_NoMatch", func(t *testing.T) {
		criteria := SelectionCriteria{
			RequiredCapabilities: []CapabilityRequirement{
				{Name: "nonexistent-capability", Required: true},
			},
		}

		_, err := selector.SelectBroker(criteria)
		if err == nil {
			t.Error("Expected error for impossible criteria")
		}
	})
}

func TestGetDefaultSelectionCriteria(t *testing.T) {
	criteria := GetDefaultSelectionCriteria()

	if len(criteria.RequiredCapabilities) == 0 {
		t.Error("Expected default criteria to have required capabilities")
	}

	if len(criteria.PreferredBrokerTypes) == 0 {
		t.Error("Expected default criteria to have preferred broker types")
	}

	// Check that consumer groups are required
	found := false
	for _, req := range criteria.RequiredCapabilities {
		if req.Name == "consumer_groups" && req.Required {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected consumer groups to be required in default criteria")
	}
}
