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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testService is a simple test service for dependency injection testing
type testService struct {
	name        string
	initialized bool
	shutdown    bool
}

func (s *testService) Initialize(ctx context.Context) error {
	s.initialized = true
	return nil
}

func (s *testService) Shutdown(ctx context.Context) error {
	s.shutdown = true
	return nil
}

// testServiceWithError is a test service that returns errors during lifecycle
type testServiceWithError struct {
	initError     error
	shutdownError error
}

func (s *testServiceWithError) Initialize(ctx context.Context) error {
	return s.initError
}

func (s *testServiceWithError) Shutdown(ctx context.Context) error {
	return s.shutdownError
}

// testServiceFactory creates a test service
func testServiceFactory(name string) DependencyFactory {
	return func(container BusinessDependencyContainer) (interface{}, error) {
		return &testService{name: name}, nil
	}
}

// errorFactory returns an error during creation
func errorFactory(container BusinessDependencyContainer) (interface{}, error) {
	return nil, errors.New("factory error")
}

func TestNewSimpleBusinessDependencyContainer(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	assert.NotNil(t, container)
	assert.False(t, container.IsInitialized())
	assert.False(t, container.IsClosed())
	assert.Empty(t, container.GetDependencyNames())
}

func TestSimpleBusinessDependencyContainer_RegisterSingleton(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	// Test successful registration
	err := container.RegisterSingleton("test-service", testServiceFactory("test"))
	assert.NoError(t, err)

	names := container.GetDependencyNames()
	assert.Contains(t, names, "test-service")

	// Test duplicate registration
	err = container.RegisterSingleton("test-service", testServiceFactory("test2"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test registration after close
	container.Close()
	err = container.RegisterSingleton("new-service", testServiceFactory("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is closed")
}

func TestSimpleBusinessDependencyContainer_RegisterTransient(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	// Test successful registration
	err := container.RegisterTransient("test-service", testServiceFactory("test"))
	assert.NoError(t, err)

	names := container.GetDependencyNames()
	assert.Contains(t, names, "test-service")

	// Test duplicate registration
	err = container.RegisterTransient("test-service", testServiceFactory("test2"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestSimpleBusinessDependencyContainer_RegisterInstance(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	instance := &testService{name: "instance"}

	// Test successful registration
	err := container.RegisterInstance("test-instance", instance)
	assert.NoError(t, err)

	names := container.GetDependencyNames()
	assert.Contains(t, names, "test-instance")

	// Verify instance is immediately available
	retrieved, err := container.GetService("test-instance")
	assert.NoError(t, err)
	assert.Same(t, instance, retrieved)
}

func TestSimpleBusinessDependencyContainer_GetService(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	// Test non-existent service
	_, err := container.GetService("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test singleton service
	err = container.RegisterSingleton("singleton", testServiceFactory("singleton"))
	require.NoError(t, err)

	service1, err := container.GetService("singleton")
	assert.NoError(t, err)
	assert.NotNil(t, service1)

	service2, err := container.GetService("singleton")
	assert.NoError(t, err)
	assert.Same(t, service1, service2) // Same instance for singleton

	// Test transient service
	err = container.RegisterTransient("transient", testServiceFactory("transient"))
	require.NoError(t, err)

	transient1, err := container.GetService("transient")
	assert.NoError(t, err)
	assert.NotNil(t, transient1)

	transient2, err := container.GetService("transient")
	assert.NoError(t, err)
	assert.NotSame(t, transient1, transient2) // Different instances for transient

	// Test factory error
	err = container.RegisterSingleton("error-service", errorFactory)
	require.NoError(t, err)

	_, err = container.GetService("error-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create dependency")

	// Test service retrieval after close
	container.Close()
	_, err = container.GetService("singleton")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is closed")
}

func TestSimpleBusinessDependencyContainer_Initialize(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	ctx := context.Background()

	// Register services
	err := container.RegisterSingleton("service1", testServiceFactory("service1"))
	require.NoError(t, err)

	err = container.RegisterTransient("service2", testServiceFactory("service2"))
	require.NoError(t, err)

	// Initialize container
	err = container.Initialize(ctx)
	assert.NoError(t, err)
	assert.True(t, container.IsInitialized())

	// Verify singleton was initialized
	service, err := container.GetService("service1")
	require.NoError(t, err)
	testSvc := service.(*testService)
	assert.True(t, testSvc.initialized)

	// Test double initialization (should be no-op)
	err = container.Initialize(ctx)
	assert.NoError(t, err)

	// Test initialization after close
	container.Close()
	err = container.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is closed")
}

func TestSimpleBusinessDependencyContainer_InitializeWithError(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	ctx := context.Background()

	// Register service that fails initialization
	err := container.RegisterSingleton("error-service", func(container BusinessDependencyContainer) (interface{}, error) {
		return &testServiceWithError{
			initError: errors.New("init error"),
		}, nil
	})
	require.NoError(t, err)

	// Initialize should fail
	err = container.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize dependency")
	assert.False(t, container.IsInitialized())
}

func TestSimpleBusinessDependencyContainer_Close(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	ctx := context.Background()

	// Register and initialize services
	err := container.RegisterSingleton("service1", testServiceFactory("service1"))
	require.NoError(t, err)

	err = container.Initialize(ctx)
	require.NoError(t, err)

	// Get service to ensure it's created
	service, err := container.GetService("service1")
	require.NoError(t, err)

	// Close container
	err = container.Close()
	assert.NoError(t, err)
	assert.True(t, container.IsClosed())
	assert.False(t, container.IsInitialized())

	// Verify service was shutdown
	testSvc := service.(*testService)
	assert.True(t, testSvc.shutdown)

	// Test double close (should be no-op)
	err = container.Close()
	assert.NoError(t, err)
}

func TestSimpleBusinessDependencyContainer_CloseWithError(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	ctx := context.Background()

	// Register service that fails shutdown
	err := container.RegisterSingleton("error-service", func(container BusinessDependencyContainer) (interface{}, error) {
		return &testServiceWithError{
			shutdownError: errors.New("shutdown error"),
		}, nil
	})
	require.NoError(t, err)

	err = container.Initialize(ctx)
	require.NoError(t, err)

	// Get service to ensure it's created
	_, err = container.GetService("error-service")
	require.NoError(t, err)

	// Close should report error but still close
	err = container.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "errors during container shutdown")
	assert.True(t, container.IsClosed())
}

func TestSimpleBusinessDependencyContainer_GetDependencyMetadata(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	// Test non-existent dependency
	_, err := container.GetDependencyMetadata("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Register and test metadata
	err = container.RegisterSingleton("test-service", testServiceFactory("test"))
	require.NoError(t, err)

	metadata, err := container.GetDependencyMetadata("test-service")
	assert.NoError(t, err)
	assert.Equal(t, "test-service", metadata.Name)
	assert.True(t, metadata.Singleton)
	assert.False(t, metadata.Initialized)
	assert.Nil(t, metadata.Instance)

	// Get service to create instance
	_, err = container.GetService("test-service")
	require.NoError(t, err)

	// Check metadata again
	metadata, err = container.GetDependencyMetadata("test-service")
	assert.NoError(t, err)
	assert.NotNil(t, metadata.Instance)
	assert.NotNil(t, metadata.Type)
}

func TestDependencyContainerBuilder(t *testing.T) {
	instance := &testService{name: "instance"}

	container := NewBusinessDependencyContainerBuilder().
		AddSingleton("singleton", testServiceFactory("singleton")).
		AddTransient("transient", testServiceFactory("transient")).
		AddInstance("instance", instance).
		Build()

	assert.NotNil(t, container)

	// Verify all dependencies are registered
	names := container.(*SimpleBusinessDependencyContainer).GetDependencyNames()
	assert.Contains(t, names, "singleton")
	assert.Contains(t, names, "transient")
	assert.Contains(t, names, "instance")

	// Test services work correctly
	singleton, err := container.GetService("singleton")
	assert.NoError(t, err)
	assert.NotNil(t, singleton)

	transient, err := container.GetService("transient")
	assert.NoError(t, err)
	assert.NotNil(t, transient)

	retrievedInstance, err := container.GetService("instance")
	assert.NoError(t, err)
	assert.Same(t, instance, retrievedInstance)
}

func TestDependencyContainerBuilder_ErrorHandling(t *testing.T) {
	// Builder should handle registration errors gracefully
	container := NewBusinessDependencyContainerBuilder().
		AddSingleton("service", testServiceFactory("service")).
		AddSingleton("service", testServiceFactory("duplicate")). // Duplicate should be logged but not fail
		Build()

	assert.NotNil(t, container)

	// Only the first registration should succeed
	names := container.(*SimpleBusinessDependencyContainer).GetDependencyNames()
	assert.Contains(t, names, "service")
	assert.Len(t, names, 1)
}

func TestDependencyLifecycleIntegration(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	ctx := context.Background()

	// Register service with lifecycle
	err := container.RegisterSingleton("lifecycle-service", testServiceFactory("lifecycle"))
	require.NoError(t, err)

	// Initialize container
	err = container.Initialize(ctx)
	require.NoError(t, err)

	// Get service
	service, err := container.GetService("lifecycle-service")
	require.NoError(t, err)

	testSvc := service.(*testService)
	assert.True(t, testSvc.initialized)
	assert.False(t, testSvc.shutdown)

	// Close container
	err = container.Close()
	require.NoError(t, err)

	assert.True(t, testSvc.shutdown)
}

func TestBusinessDependencyContainerConcurrency(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	// Register services
	err := container.RegisterSingleton("service", testServiceFactory("service"))
	require.NoError(t, err)

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			service, err := container.GetService("service")
			assert.NoError(t, err)
			assert.NotNil(t, service)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All should get the same singleton instance
	service1, err := container.GetService("service")
	require.NoError(t, err)

	service2, err := container.GetService("service")
	require.NoError(t, err)

	assert.Same(t, service1, service2)
}
