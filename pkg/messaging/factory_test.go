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
	"errors"
	"testing"
	"time"
)

// TestNewMessageBrokerFactory tests the factory constructor.
func TestNewMessageBrokerFactory(t *testing.T) {
	factory := NewMessageBrokerFactory()
	if factory == nil {
		t.Error("Expected non-nil factory")
	}

	// Should be empty initially
	types := factory.GetSupportedBrokerTypes()
	if len(types) != 0 {
		t.Errorf("Expected empty supported types, got: %v", types)
	}
}

// TestGetDefaultFactory tests the default factory.
func TestGetDefaultFactory(t *testing.T) {
	factory := GetDefaultFactory()
	if factory == nil {
		t.Error("Expected non-nil default factory")
	}

	// Should be the same instance
	factory2 := GetDefaultFactory()
	if factory != factory2 {
		t.Error("Expected same factory instance")
	}
}

// TestRegisterBrokerFactory tests broker factory registration.
func TestRegisterBrokerFactory(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Check supported types
	types := factory.GetSupportedBrokerTypes()
	if len(types) != 1 {
		t.Errorf("Expected 1 supported type, got: %d", len(types))
	}

	if types[0] != BrokerTypeInMemory {
		t.Errorf("Expected %v, got: %v", BrokerTypeInMemory, types[0])
	}
}

// TestCreateBroker tests broker creation.
func TestCreateBroker(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Test with nil config
	_, err := factory.CreateBroker(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Test with valid config
	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	broker, err := factory.CreateBroker(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if broker == nil {
		t.Error("Expected non-nil broker")
	}

	// Test with unsupported broker type
	config.Type = BrokerTypeKafka
	_, err = factory.CreateBroker(config)
	if err == nil {
		t.Error("Expected error for unsupported broker type")
	}

	if !IsConfigurationError(err) {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

// TestCreateBrokerWithFailingFactory tests broker creation with a failing factory.
func TestCreateBrokerWithFailingFactory(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Register a failing factory
	failingFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return nil, errors.New("factory failed")
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, failingFactory)

	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	_, err := factory.CreateBroker(config)
	if err == nil {
		t.Error("Expected error from failing factory")
	}
}

// TestValidateConfig tests configuration validation.
func TestValidateConfig(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Test with nil config
	err := factory.ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Test with valid config
	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	err = factory.ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Test with unsupported broker type
	config.Type = BrokerTypeKafka
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for unsupported broker type")
	}

	// Test with invalid config
	config.Type = BrokerTypeInMemory
	config.Endpoints = nil // Invalid: no endpoints
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

// TestGlobalFunctions tests the global convenience functions.
func TestGlobalFunctions(t *testing.T) {
	// Clean up the default factory for testing
	originalFactory := defaultFactory
	defer func() {
		defaultFactory = originalFactory
	}()

	// Replace with a test factory
	testFactory := NewMessageBrokerFactory()
	defaultFactory = testFactory.(*messageBrokerFactoryImpl)

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Test GetSupportedBrokerTypes
	types := GetSupportedBrokerTypes()
	if len(types) != 1 || types[0] != BrokerTypeInMemory {
		t.Errorf("Expected [%v], got: %v", BrokerTypeInMemory, types)
	}

	// Test NewMessageBroker
	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	broker, err := NewMessageBroker(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if broker == nil {
		t.Error("Expected non-nil broker")
	}

	// Test ValidateBrokerConfig
	err = ValidateBrokerConfig(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test with invalid config
	config.Endpoints = nil
	err = ValidateBrokerConfig(config)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

// TestConcurrentFactoryAccess tests concurrent access to the factory.
func TestConcurrentFactoryAccess(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Register multiple factories concurrently
	done := make(chan bool, 4)

	go func() {
		factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeKafka, func(config *BrokerConfig) (MessageBroker, error) {
			return &mockMessageBroker{}, nil
		})
		done <- true
	}()

	go func() {
		factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeNATS, func(config *BrokerConfig) (MessageBroker, error) {
			return &mockMessageBroker{}, nil
		})
		done <- true
	}()

	go func() {
		types := factory.GetSupportedBrokerTypes()
		// Should not panic
		_ = len(types)
		done <- true
	}()

	go func() {
		config := &BrokerConfig{
			Type:      BrokerTypeInMemory,
			Endpoints: []string{"localhost:8080"},
		}
		_ = factory.ValidateConfig(config)
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify both factories were registered
	types := factory.GetSupportedBrokerTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 registered factories, got: %d", len(types))
	}
}
