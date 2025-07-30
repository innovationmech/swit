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

package server

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types
type TestService struct {
	Name string
}

type TestDatabase struct {
	URL string
}

type TestCleanupService struct {
	Cleaned bool
}

func (t *TestCleanupService) Cleanup(ctx context.Context) error {
	t.Cleaned = true
	return nil
}

type TestFailingCleanupService struct{}

func (t *TestFailingCleanupService) Cleanup(ctx context.Context) error {
	return errors.New("cleanup failed")
}

func TestNewSimpleDependencyContainer(t *testing.T) {
	container := NewSimpleDependencyContainer()

	assert.NotNil(t, container)
	assert.NotNil(t, container.dependencies)
	assert.Equal(t, 0, container.Count())
	assert.False(t, container.IsInitialized())
}

func TestSimpleDependencyContainer_Register(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Test successful registration
	service := &TestService{Name: "test"}
	err := container.Register("service", service)
	assert.NoError(t, err)
	assert.True(t, container.Has("service"))
	assert.Equal(t, 1, container.Count())

	// Test duplicate registration
	err = container.Register("service", service)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test empty name
	err = container.Register("", service)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")

	// Test nil value
	err = container.Register("nil-service", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value cannot be nil")
}

func TestSimpleDependencyContainer_RegisterFactory(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Test successful factory registration
	factory := func() (interface{}, error) {
		return &TestService{Name: "factory"}, nil
	}

	err := container.RegisterFactory("factory-service", factory, true)
	assert.NoError(t, err)
	assert.True(t, container.Has("factory-service"))
	assert.Equal(t, 1, container.Count())

	// Test duplicate registration
	err = container.RegisterFactory("factory-service", factory, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test empty name
	err = container.RegisterFactory("", factory, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")

	// Test nil factory
	err = container.RegisterFactory("nil-factory", nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory function cannot be nil")
}

func TestSimpleDependencyContainer_Get(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Test getting non-existent dependency
	_, err := container.Get("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting direct value
	service := &TestService{Name: "test"}
	err = container.Register("service", service)
	require.NoError(t, err)

	result, err := container.Get("service")
	assert.NoError(t, err)
	assert.Equal(t, service, result)

	// Test getting singleton factory
	callCount := 0
	factory := func() (interface{}, error) {
		callCount++
		return &TestService{Name: "singleton"}, nil
	}

	err = container.RegisterFactory("singleton-service", factory, true)
	require.NoError(t, err)

	// First call
	result1, err := container.Get("singleton-service")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call should return same instance
	result2, err := container.Get("singleton-service")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount) // Factory should not be called again
	assert.Equal(t, result1, result2)

	// Test getting non-singleton factory
	callCount2 := 0
	factory2 := func() (interface{}, error) {
		callCount2++
		return &TestService{Name: fmt.Sprintf("non-singleton-%d", callCount2)}, nil
	}

	err = container.RegisterFactory("non-singleton-service", factory2, false)
	require.NoError(t, err)

	// First call
	result3, err := container.Get("non-singleton-service")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount2)

	// Second call should create new instance
	result4, err := container.Get("non-singleton-service")
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount2) // Factory should be called again
	assert.NotEqual(t, result3, result4)

	// Test factory error
	failingFactory := func() (interface{}, error) {
		return nil, errors.New("factory error")
	}

	err = container.RegisterFactory("failing-service", failingFactory, true)
	require.NoError(t, err)

	_, err = container.Get("failing-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create dependency")
}

func TestSimpleDependencyContainer_GetByType(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Test getting by non-existent type
	_, err := container.GetByType(reflect.TypeOf(&TestDatabase{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no dependency found for type")

	// Test getting by existing type
	service := &TestService{Name: "test"}
	err = container.Register("service", service)
	require.NoError(t, err)

	result, err := container.GetByType(reflect.TypeOf(service))
	assert.NoError(t, err)
	assert.Equal(t, service, result)
}

func TestSimpleDependencyContainer_Initialize(t *testing.T) {
	container := NewSimpleDependencyContainer()
	ctx := context.Background()

	// Test initializing empty container
	err := container.Initialize(ctx)
	assert.NoError(t, err)
	assert.True(t, container.IsInitialized())

	// Test double initialization
	err = container.Initialize(ctx)
	assert.NoError(t, err) // Should not error

	// Reset for next test
	container = NewSimpleDependencyContainer()

	// Test initializing with singleton factories
	callCount := 0
	factory := func() (interface{}, error) {
		callCount++
		return &TestService{Name: "initialized"}, nil
	}

	err = container.RegisterFactory("singleton-service", factory, true)
	require.NoError(t, err)

	// Non-singleton factory
	factory2 := func() (interface{}, error) {
		return &TestService{Name: "non-singleton"}, nil
	}

	err = container.RegisterFactory("non-singleton-service", factory2, false)
	require.NoError(t, err)

	// Direct value
	err = container.Register("direct-service", &TestService{Name: "direct"})
	require.NoError(t, err)

	err = container.Initialize(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount) // Only singleton factory should be called
	assert.True(t, container.IsInitialized())

	// Test initialization with failing factory
	container2 := NewSimpleDependencyContainer()
	failingFactory := func() (interface{}, error) {
		return nil, errors.New("initialization failed")
	}

	err = container2.RegisterFactory("failing-service", failingFactory, true)
	require.NoError(t, err)

	err = container2.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize dependency")
}

func TestSimpleDependencyContainer_Cleanup(t *testing.T) {
	container := NewSimpleDependencyContainer()
	ctx := context.Background()

	// Test cleanup with cleanable service
	cleanupService := &TestCleanupService{}
	err := container.Register("cleanup-service", cleanupService)
	require.NoError(t, err)

	// Regular service without cleanup
	regularService := &TestService{Name: "regular"}
	err = container.Register("regular-service", regularService)
	require.NoError(t, err)

	err = container.Cleanup(ctx)
	assert.NoError(t, err)
	assert.True(t, cleanupService.Cleaned)
	assert.False(t, container.IsInitialized())

	// Test cleanup with failing service
	container2 := NewSimpleDependencyContainer()
	failingService := &TestFailingCleanupService{}
	err = container2.Register("failing-service", failingService)
	require.NoError(t, err)

	err = container2.Cleanup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup completed with")
	assert.Contains(t, err.Error(), "errors")
}

func TestSimpleDependencyContainer_List(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Test empty container
	names := container.List()
	assert.Empty(t, names)

	// Test with dependencies
	err := container.Register("service1", &TestService{Name: "test1"})
	require.NoError(t, err)

	err = container.Register("service2", &TestService{Name: "test2"})
	require.NoError(t, err)

	names = container.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "service1")
	assert.Contains(t, names, "service2")
}

func TestSimpleDependencyContainer_Clear(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Add some dependencies
	err := container.Register("service1", &TestService{Name: "test1"})
	require.NoError(t, err)

	err = container.Register("service2", &TestService{Name: "test2"})
	require.NoError(t, err)

	err = container.Initialize(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, container.Count())
	assert.True(t, container.IsInitialized())

	// Clear container
	container.Clear()

	assert.Equal(t, 0, container.Count())
	assert.False(t, container.IsInitialized())
	assert.Empty(t, container.List())
}

func TestSimpleDependencyContainer_GetDependencyInfo(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Test non-existent dependency
	_, err := container.GetDependencyInfo("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test direct value dependency
	service := &TestService{Name: "test"}
	err = container.Register("service", service)
	require.NoError(t, err)

	info, err := container.GetDependencyInfo("service")
	assert.NoError(t, err)
	assert.Equal(t, "service", info.Name)
	assert.Equal(t, reflect.TypeOf(service), info.Type)
	assert.True(t, info.Singleton)
	assert.True(t, info.Initialized)

	// Test factory dependency
	factory := func() (interface{}, error) {
		return &TestService{Name: "factory"}, nil
	}

	err = container.RegisterFactory("factory-service", factory, false)
	require.NoError(t, err)

	info, err = container.GetDependencyInfo("factory-service")
	assert.NoError(t, err)
	assert.Equal(t, "factory-service", info.Name)
	assert.False(t, info.Singleton)
	assert.False(t, info.Initialized)
}

func TestSimpleDependencyContainer_ConcurrentAccess(t *testing.T) {
	container := NewSimpleDependencyContainer()

	// Register a factory
	callCount := 0
	factory := func() (interface{}, error) {
		callCount++
		time.Sleep(10 * time.Millisecond) // Simulate some work
		return &TestService{Name: "concurrent"}, nil
	}

	err := container.RegisterFactory("concurrent-service", factory, true)
	require.NoError(t, err)

	// Concurrent access
	const numGoroutines = 10
	results := make([]interface{}, numGoroutines)
	errors := make([]error, numGoroutines)
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			results[index], errors[index] = container.Get("concurrent-service")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check results
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i])
		assert.NotNil(t, results[i])
		// All results should be the same instance (singleton)
		if i > 0 {
			assert.Equal(t, results[0], results[i])
		}
	}

	// Factory should only be called once despite concurrent access
	assert.Equal(t, 1, callCount)
}
