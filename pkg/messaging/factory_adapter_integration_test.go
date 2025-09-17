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

package messaging

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// testAdapterRegistry implements AdapterRegistryProvider for tests.
type testAdapterRegistry struct {
	// map of brokerType -> adapter
	typeToAdapter map[BrokerType]MessageBrokerAdapter
}

func (r *testAdapterRegistry) GetAdapterForBrokerType(brokerType BrokerType) (MessageBrokerAdapter, error) {
	if a, ok := r.typeToAdapter[brokerType]; ok {
		return a, nil
	}
	return nil, NewConfigError("no adapter for type", nil)
}

func (r *testAdapterRegistry) GetSupportedBrokerTypes() []BrokerType {
	types := make([]BrokerType, 0, len(r.typeToAdapter))
	for t := range r.typeToAdapter {
		types = append(types, t)
	}
	return types
}

func (r *testAdapterRegistry) ValidateConfiguration(config *BrokerConfig) (*AdapterValidationResult, error) {
	if config == nil {
		return &AdapterValidationResult{Valid: false, Errors: []AdapterValidationError{{Message: "configuration cannot be nil"}}}, nil
	}
	a, ok := r.typeToAdapter[config.Type]
	if !ok {
		return nil, NewConfigError("no adapter for type", nil)
	}
	return a.ValidateConfiguration(config), nil
}

// testAdapter implements MessageBrokerAdapter for tests.
type testAdapter struct {
	info   *BrokerAdapterInfo
	broker MessageBroker
	// validateOK toggles validation result
	validateOK bool
}

func (a *testAdapter) GetAdapterInfo() *BrokerAdapterInfo { return a.info }
func (a *testAdapter) CreateBroker(config *BrokerConfig) (MessageBroker, error) {
	if a.broker == nil {
		return nil, errors.New("no broker stub")
	}
	return a.broker, nil
}
func (a *testAdapter) ValidateConfiguration(config *BrokerConfig) *AdapterValidationResult {
	if a.validateOK {
		return &AdapterValidationResult{Valid: true}
	}
	return &AdapterValidationResult{
		Valid: false,
		Errors: []AdapterValidationError{{
			Field:   "Endpoints",
			Message: "at least one endpoint must be specified",
			Code:    "ENDPOINTS_EMPTY",
		}},
	}
}
func (a *testAdapter) GetCapabilities() *BrokerCapabilities {
	return &BrokerCapabilities{SupportsConsumerGroups: true}
}
func (a *testAdapter) GetDefaultConfiguration() *BrokerConfig {
	return &BrokerConfig{Type: BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
}
func (a *testAdapter) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	return &HealthStatus{Status: HealthStatusHealthy, LastChecked: time.Now()}, nil
}

// testBroker is a minimal MessageBroker stub
type testBroker struct{}

func (b *testBroker) Connect(ctx context.Context) error                              { return nil }
func (b *testBroker) Disconnect(ctx context.Context) error                           { return nil }
func (b *testBroker) Close() error                                                   { return nil }
func (b *testBroker) IsConnected() bool                                              { return true }
func (b *testBroker) CreatePublisher(cfg PublisherConfig) (EventPublisher, error)    { return nil, nil }
func (b *testBroker) CreateSubscriber(cfg SubscriberConfig) (EventSubscriber, error) { return nil, nil }
func (b *testBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	return &HealthStatus{Status: HealthStatusHealthy}, nil
}
func (b *testBroker) GetMetrics() *BrokerMetrics           { return &BrokerMetrics{} }
func (b *testBroker) GetCapabilities() *BrokerCapabilities { return &BrokerCapabilities{} }

func TestFactory_UsesAdapterRegistry_CreateBroker_Success(t *testing.T) {
	registry := &testAdapterRegistry{typeToAdapter: map[BrokerType]MessageBrokerAdapter{}}
	adapter := &testAdapter{
		info: &BrokerAdapterInfo{
			Name:                 "test",
			SupportedBrokerTypes: []BrokerType{BrokerTypeKafka},
		},
		broker:     &testBroker{},
		validateOK: true,
	}
	registry.typeToAdapter[BrokerTypeKafka] = adapter

	factory := &messageBrokerFactoryImpl{factories: make(map[BrokerType]func(*BrokerConfig) (MessageBroker, error)), adapters: registry}

	cfg := &BrokerConfig{Type: BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	b, err := factory.CreateBroker(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil broker")
	}
}

func TestFactory_ValidateConfig_AdapterErrorDetails(t *testing.T) {
	registry := &testAdapterRegistry{typeToAdapter: map[BrokerType]MessageBrokerAdapter{}}
	adapter := &testAdapter{
		info:       &BrokerAdapterInfo{Name: "test", SupportedBrokerTypes: []BrokerType{BrokerTypeKafka}},
		broker:     &testBroker{},
		validateOK: false,
	}
	registry.typeToAdapter[BrokerTypeKafka] = adapter

	factory := &messageBrokerFactoryImpl{factories: make(map[BrokerType]func(*BrokerConfig) (MessageBroker, error)), adapters: registry}

	cfg := &BrokerConfig{Type: BrokerTypeKafka, Endpoints: nil}
	err := factory.ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error from adapter validation")
	}
	if !strings.Contains(err.Error(), "adapter configuration validation failed") {
		t.Fatalf("expected adapter validation error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Endpoints") {
		t.Fatalf("expected field detail in error, got: %v", err)
	}
}

func TestFactory_ValidateConfig_UnsupportedTypeListsSupported(t *testing.T) {
	factory := &messageBrokerFactoryImpl{factories: make(map[BrokerType]func(*BrokerConfig) (MessageBroker, error))}

	// Register one legacy type to appear in supported list
	factory.RegisterBrokerFactory(BrokerTypeInMemory, func(config *BrokerConfig) (MessageBroker, error) { return &testBroker{}, nil })

	cfg := &BrokerConfig{Type: BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	err := factory.ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
	if !strings.Contains(err.Error(), "supported:") {
		t.Fatalf("expected supported types in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "inmemory") {
		t.Fatalf("expected 'inmemory' listed, got: %v", err)
	}
}
