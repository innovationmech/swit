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

package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareRegistry_Creation(t *testing.T) {
	registry := NewMiddlewareRegistry()

	assert.NotNil(t, registry)

	// Should have built-in factories registered
	factories := registry.ListFactories()
	assert.Contains(t, factories, "timeout")
	assert.Contains(t, factories, "retry")
	assert.Contains(t, factories, "recovery")
}

func TestMiddlewareRegistry_RegisterMiddleware(t *testing.T) {
	registry := NewMiddlewareRegistry()

	middleware := &MockMiddleware{name: "test-middleware"}

	// Test successful registration
	err := registry.RegisterMiddleware("test", middleware)
	assert.NoError(t, err)

	// Test duplicate registration
	err = registry.RegisterMiddleware("test", middleware)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestMiddlewareRegistry_GetMiddleware(t *testing.T) {
	registry := NewMiddlewareRegistry()

	middleware := &MockMiddleware{name: "test-middleware"}
	err := registry.RegisterMiddleware("test", middleware)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := registry.GetMiddleware("test")
	assert.NoError(t, err)
	assert.Equal(t, middleware, retrieved)

	// Test non-existent middleware
	_, err = registry.GetMiddleware("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMiddlewareRegistry_RegisterFactory(t *testing.T) {
	registry := NewMiddlewareRegistry()

	factory := func(config map[string]interface{}) (Middleware, error) {
		return &MockMiddleware{name: "custom-middleware"}, nil
	}

	// Test successful registration
	err := registry.RegisterFactory("custom", factory)
	assert.NoError(t, err)

	// Test duplicate registration
	err = registry.RegisterFactory("custom", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestMiddlewareRegistry_CreateMiddleware(t *testing.T) {
	registry := NewMiddlewareRegistry()

	// Test built-in timeout factory
	config := map[string]interface{}{
		"timeout": "5s",
	}

	middleware, err := registry.CreateMiddleware("timeout", config)
	assert.NoError(t, err)
	assert.NotNil(t, middleware)
	assert.Equal(t, "timeout", middleware.Name())

	// Test built-in retry factory
	retryConfig := map[string]interface{}{
		"max_attempts": 5,
	}

	retryMiddleware, err := registry.CreateMiddleware("retry", retryConfig)
	assert.NoError(t, err)
	assert.NotNil(t, retryMiddleware)
	assert.Equal(t, "retry", retryMiddleware.Name())

	// Test built-in recovery factory
	recoveryMiddleware, err := registry.CreateMiddleware("recovery", map[string]interface{}{})
	assert.NoError(t, err)
	assert.NotNil(t, recoveryMiddleware)
	assert.Equal(t, "recovery", recoveryMiddleware.Name())

	// Test non-existent factory
	_, err = registry.CreateMiddleware("non-existent", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMiddlewareRegistry_CustomFactory(t *testing.T) {
	registry := NewMiddlewareRegistry()

	// Register custom factory
	factory := func(config map[string]interface{}) (Middleware, error) {
		name := "custom"
		if n, ok := config["name"]; ok {
			if nameStr, ok := n.(string); ok {
				name = nameStr
			}
		}
		return &MockMiddleware{name: name}, nil
	}

	err := registry.RegisterFactory("custom", factory)
	require.NoError(t, err)

	// Create middleware using custom factory
	config := map[string]interface{}{
		"name": "my-custom-middleware",
	}

	middleware, err := registry.CreateMiddleware("custom", config)
	assert.NoError(t, err)
	assert.Equal(t, "my-custom-middleware", middleware.Name())
}

func TestMiddlewareRegistry_ListOperations(t *testing.T) {
	registry := NewMiddlewareRegistry()

	// Register some middleware
	middleware1 := &MockMiddleware{name: "middleware1"}
	middleware2 := &MockMiddleware{name: "middleware2"}

	err := registry.RegisterMiddleware("mw1", middleware1)
	require.NoError(t, err)
	err = registry.RegisterMiddleware("mw2", middleware2)
	require.NoError(t, err)

	// Test ListMiddleware
	middleware := registry.ListMiddleware()
	assert.Len(t, middleware, 2)
	assert.Contains(t, middleware, "mw1")
	assert.Contains(t, middleware, "mw2")

	// Test ListFactories (should include built-ins)
	factories := registry.ListFactories()
	assert.GreaterOrEqual(t, len(factories), 3) // At least timeout, retry, recovery
	assert.Contains(t, factories, "timeout")
	assert.Contains(t, factories, "retry")
	assert.Contains(t, factories, "recovery")
}

func TestMiddlewareRegistry_UnregisterOperations(t *testing.T) {
	registry := NewMiddlewareRegistry()

	// Register middleware and factory
	middleware := &MockMiddleware{name: "test-middleware"}
	err := registry.RegisterMiddleware("test", middleware)
	require.NoError(t, err)

	factory := func(config map[string]interface{}) (Middleware, error) {
		return middleware, nil
	}
	err = registry.RegisterFactory("test-factory", factory)
	require.NoError(t, err)

	// Test UnregisterMiddleware
	removed := registry.UnregisterMiddleware("test")
	assert.True(t, removed)

	_, err = registry.GetMiddleware("test")
	assert.Error(t, err)

	// Test unregistering non-existent middleware
	removed = registry.UnregisterMiddleware("non-existent")
	assert.False(t, removed)

	// Test UnregisterFactory
	removed = registry.UnregisterFactory("test-factory")
	assert.True(t, removed)

	_, err = registry.CreateMiddleware("test-factory", map[string]interface{}{})
	assert.Error(t, err)

	// Test unregistering non-existent factory
	removed = registry.UnregisterFactory("non-existent")
	assert.False(t, removed)
}

func TestMiddlewareRegistry_TimeoutFactory(t *testing.T) {
	registry := NewMiddlewareRegistry()

	t.Run("default timeout", func(t *testing.T) {
		middleware, err := registry.CreateMiddleware("timeout", map[string]interface{}{})
		assert.NoError(t, err)
		assert.NotNil(t, middleware)

		// Test that it wraps correctly
		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		wrapped := middleware.Wrap(handler)
		err = wrapped.Handle(context.Background(), &Message{ID: "test"})
		assert.NoError(t, err)
	})

	t.Run("custom timeout duration", func(t *testing.T) {
		config := map[string]interface{}{
			"timeout": 50 * time.Millisecond,
		}

		middleware, err := registry.CreateMiddleware("timeout", config)
		assert.NoError(t, err)

		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		wrapped := middleware.Wrap(handler)
		err = wrapped.Handle(context.Background(), &Message{ID: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("custom timeout string", func(t *testing.T) {
		config := map[string]interface{}{
			"timeout": "2s",
		}

		middleware, err := registry.CreateMiddleware("timeout", config)
		assert.NoError(t, err)
		assert.NotNil(t, middleware)
	})
}

func TestMiddlewareRegistry_RetryFactory(t *testing.T) {
	registry := NewMiddlewareRegistry()

	t.Run("default retry config", func(t *testing.T) {
		middleware, err := registry.CreateMiddleware("retry", map[string]interface{}{})
		assert.NoError(t, err)
		assert.NotNil(t, middleware)

		attempts := 0
		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			attempts++
			if attempts < 2 {
				return fmt.Errorf("temporary error")
			}
			return nil
		})

		wrapped := middleware.Wrap(handler)
		err = wrapped.Handle(context.Background(), &Message{ID: "test"})
		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("custom max attempts", func(t *testing.T) {
		config := map[string]interface{}{
			"max_attempts": 5,
		}

		middleware, err := registry.CreateMiddleware("retry", config)
		assert.NoError(t, err)

		attempts := 0
		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			attempts++
			if attempts < 4 {
				return fmt.Errorf("temporary error")
			}
			return nil
		})

		wrapped := middleware.Wrap(handler)
		err = wrapped.Handle(context.Background(), &Message{ID: "test"})
		assert.NoError(t, err)
		assert.Equal(t, 4, attempts)
	})
}

func TestMiddlewareRegistry_RecoveryFactory(t *testing.T) {
	registry := NewMiddlewareRegistry()

	t.Run("default recovery", func(t *testing.T) {
		middleware, err := registry.CreateMiddleware("recovery", map[string]interface{}{})
		assert.NoError(t, err)
		assert.NotNil(t, middleware)

		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			panic("test panic")
		})

		wrapped := middleware.Wrap(handler)
		err = wrapped.Handle(context.Background(), &Message{ID: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "panic recovered")
	})

	t.Run("recovery with custom logger", func(t *testing.T) {
		logger := &MockLogger{}
		config := map[string]interface{}{
			"logger": logger,
		}

		middleware, err := registry.CreateMiddleware("recovery", config)
		assert.NoError(t, err)

		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			panic("test panic")
		})

		wrapped := middleware.Wrap(handler)
		err = wrapped.Handle(context.Background(), &Message{ID: "test"})
		assert.Error(t, err)
		assert.True(t, logger.ErrorCalled)
	})
}

func TestGlobalMiddlewareRegistry(t *testing.T) {
	// Test that global registry exists and has built-in factories
	assert.NotNil(t, GlobalMiddlewareRegistry)

	factories := GlobalMiddlewareRegistry.ListFactories()
	assert.Contains(t, factories, "timeout")
	assert.Contains(t, factories, "retry")
	assert.Contains(t, factories, "recovery")
}

// MockLogger for testing
type MockLogger struct {
	DebugCalled bool
	InfoCalled  bool
	WarnCalled  bool
	ErrorCalled bool
	LastMessage string
	LastFields  []interface{}
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.DebugCalled = true
	m.LastMessage = msg
	m.LastFields = fields
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.InfoCalled = true
	m.LastMessage = msg
	m.LastFields = fields
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.WarnCalled = true
	m.LastMessage = msg
	m.LastFields = fields
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.ErrorCalled = true
	m.LastMessage = msg
	m.LastFields = fields
}
