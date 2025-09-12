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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockResource implements RecoverableResource for testing
type MockResource struct {
	name         string
	resourceType ResourceType
	isActive     bool
	isHealthy    bool
	lastActivity time.Time
	cleaned      bool
	metrics      *ResourceMetrics
}

func NewMockResource(name string, resourceType ResourceType) *MockResource {
	return &MockResource{
		name:         name,
		resourceType: resourceType,
		isActive:     true,
		isHealthy:    true,
		lastActivity: time.Now(),
		cleaned:      false,
		metrics: &ResourceMetrics{
			MemoryUsed: 1024,
		},
	}
}

func (m *MockResource) GetResourceName() string {
	return m.name
}

func (m *MockResource) GetResourceType() ResourceType {
	return m.resourceType
}

func (m *MockResource) IsHealthy(ctx context.Context) bool {
	return m.isHealthy
}

func (m *MockResource) IsActive() bool {
	return m.isActive
}

func (m *MockResource) GetLastActivity() time.Time {
	return m.lastActivity
}

func (m *MockResource) GetResourceMetrics() *ResourceMetrics {
	return m.metrics
}

func (m *MockResource) Recover(ctx context.Context) error {
	m.isHealthy = true
	m.isActive = true
	m.lastActivity = time.Now()
	return nil
}

func (m *MockResource) Cleanup(ctx context.Context) error {
	m.cleaned = true
	m.isActive = false
	m.lastActivity = time.Now()
	return nil
}

// MockRecoveryHandler implements RecoveryHandler for testing
type MockRecoveryHandler struct {
	name      string
	canHandle bool
	action    *RecoveryAction
	err       error
}

func NewMockRecoveryHandler(name string, canHandle bool) *MockRecoveryHandler {
	return &MockRecoveryHandler{
		name:      name,
		canHandle: canHandle,
		action: &RecoveryAction{
			ActionType:   RecoveryActionRetry,
			ResourceName: "test-resource",
			Description:  "Retry operation",
			Success:      true,
		},
	}
}

func (h *MockRecoveryHandler) CanHandle(err error) bool {
	return h.canHandle
}

func (h *MockRecoveryHandler) HandleError(ctx context.Context, err error, context *ErrorContext) (*RecoveryAction, error) {
	if h.err != nil {
		return nil, h.err
	}
	return h.action, nil
}

func (h *MockRecoveryHandler) GetHandlerName() string {
	return h.name
}

// createTestConfig creates a valid test configuration
func createTestConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Enabled:                 true,
		AutoCleanupInterval:     5 * time.Minute,
		LeakDetectionInterval:   10 * time.Minute,
		ResourceTimeout:         30 * time.Minute,
		MaxRecoveryAttempts:     10,
		CircuitBreakerThreshold: 5,
		RecoveryLogEnabled:      true,
	}
}

func TestNewRecoveryManager(t *testing.T) {
	manager := NewRecoveryManager()
	assert.NotNil(t, manager)
	assert.IsType(t, &recoveryManagerImpl{}, manager)
}

func TestRecoveryManagerInitialize(t *testing.T) {
	manager := NewRecoveryManager()
	config := &RecoveryConfig{
		Enabled:                 true,
		AutoCleanupInterval:     5 * time.Minute,
		LeakDetectionInterval:   10 * time.Minute,
		ResourceTimeout:         30 * time.Minute,
		MaxRecoveryAttempts:     10,
		CircuitBreakerThreshold: 5,
		RecoveryLogEnabled:      true,
	}

	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Test invalid config
	invalidConfig := &RecoveryConfig{
		Enabled:             true,
		AutoCleanupInterval: -1 * time.Minute, // Invalid negative duration
	}
	err = manager.Initialize(ctx, invalidConfig)
	assert.Error(t, err)
}

func TestRecoveryManagerRegisterResource(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Test valid resource registration
	resource := NewMockResource("test-resource", ResourceTypeConnection)
	err = manager.RegisterResource("test-resource", resource)
	require.NoError(t, err)

	// Test duplicate registration
	err = manager.RegisterResource("test-resource", resource)
	assert.Error(t, err)

	// Test empty name
	err = manager.RegisterResource("", resource)
	assert.Error(t, err)

	// Test nil resource
	err = manager.RegisterResource("nil-resource", nil)
	assert.Error(t, err)
}

func TestRecoveryManagerUnregisterResource(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	resource := NewMockResource("test-resource", ResourceTypeConnection)
	err = manager.RegisterResource("test-resource", resource)
	require.NoError(t, err)

	// Test successful unregistration
	err = manager.UnregisterResource("test-resource")
	require.NoError(t, err)

	// Test unregister non-existent resource
	err = manager.UnregisterResource("non-existent")
	assert.Error(t, err)
}

func TestRecoveryManagerRegisterRecoveryHandler(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	handler := NewMockRecoveryHandler("test-handler", true)

	// Test valid handler registration
	err = manager.RegisterRecoveryHandler(ErrorTypeConnection, handler)
	require.NoError(t, err)

	// Test duplicate registration for same error type - should not error as we allow multiple handlers
	err = manager.RegisterRecoveryHandler(ErrorTypeConnection, handler)
	assert.NoError(t, err)
}

func TestRecoveryManagerHandleError(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	handler := NewMockRecoveryHandler("connection-handler", true)
	manager.RegisterRecoveryHandler(ErrorTypeConnection, handler)

	// Test error handling with registered handler
	testErr := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("connection failed").
		Build()

	errorCtx := &ErrorContext{
		Component: "test-component",
		Operation: "test-operation",
	}

	result, err := manager.HandleError(ctx, testErr, errorCtx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.ActionsTaken, "RETRY")

	// Test error handling without handler
	unhandledErr := NewError(ErrorTypeValidation, ErrCodeInvalidConfig).
		Message("validation error").
		Build()

	result, err = manager.HandleError(ctx, unhandledErr, errorCtx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestRecoveryManagerPerformCleanup(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Register test resource
	resource := NewMockResource("cleanup-test", ResourceTypeConnection)
	err = manager.RegisterResource("cleanup-test", resource)
	require.NoError(t, err)

	// Perform cleanup
	result, err := manager.PerformCleanup(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, resource.cleaned)
	assert.Equal(t, 1, result.ResourcesCleaned)
}

func TestRecoveryManagerDetectLeaks(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	config.ResourceTimeout = 1 * time.Millisecond // Very short timeout for testing
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Register old resource that should be detected as leak
	oldResource := NewMockResource("old-resource", ResourceTypeConnection)
	oldResource.lastActivity = time.Now().Add(-1 * time.Hour)
	oldResource.isActive = true

	err = manager.RegisterResource("old-resource", oldResource)
	require.NoError(t, err)

	// Detect leaks
	result, err := manager.DetectLeaks(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.LeaksDetected, "Expected leaks to be detected")
}

func TestRecoveryManagerGetRecoveryMetrics(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Register test resource
	resource := NewMockResource("metrics-test", ResourceTypeConnection)
	err = manager.RegisterResource("metrics-test", resource)
	require.NoError(t, err)

	// Get metrics
	metrics := manager.GetRecoveryMetrics()
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.TotalRecoveryAttempts, uint64(0))
}

func TestRecoveryManagerClose(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Register test resource
	resource := NewMockResource("close-test", ResourceTypeConnection)
	err = manager.RegisterResource("close-test", resource)
	require.NoError(t, err)

	// Close manager
	err = manager.Close(ctx)
	require.NoError(t, err)

	// Verify resource was cleaned up
	assert.True(t, resource.cleaned)
}

func TestRecoveryManagerConcurrentOperations(t *testing.T) {
	manager := NewRecoveryManager()
	config := createTestConfig()
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	// Test concurrent resource registration
	const numGoroutines = 10
	const numResources = 5

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numResources; j++ {
				resourceName := fmt.Sprintf("resource-%d-%d", id, j)
				resource := NewMockResource(resourceName, ResourceTypeConnection)

				err := manager.RegisterResource(resourceName, resource)
				if err != nil {
					errors <- err
					return
				}

				// Immediately unregister to avoid conflicts
				err = manager.UnregisterResource(resourceName)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case err := <-errors:
			t.Errorf("Concurrent operation failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
	}
}

func TestRecoveryManagerErrorRecoveryWithCircuitBreaker(t *testing.T) {
	manager := NewRecoveryManager()
	config := &RecoveryConfig{
		Enabled:                 true,
		MaxRecoveryAttempts:     3,
		CircuitBreakerThreshold: 3,
	}
	ctx := context.Background()
	err := manager.Initialize(ctx, config)
	require.NoError(t, err)

	handler := NewMockRecoveryHandler("circuit-test-handler", true)
	manager.RegisterRecoveryHandler(ErrorTypeConnection, handler)

	// Create a test error
	testErr := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("connection failed").
		Build()

	errorCtx := &ErrorContext{
		Component: "circuit-test",
		Operation: "test-operation",
	}

	// Trigger circuit breaker by sending multiple errors
	for i := 0; i < 5; i++ {
		_, err := manager.HandleError(ctx, testErr, errorCtx)
		require.NoError(t, err)
	}

	// Verify that recovery still works even with circuit breaker
	result, err := manager.HandleError(ctx, testErr, errorCtx)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRecoveryConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *RecoveryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RecoveryConfig{
				Enabled:                 true,
				AutoCleanupInterval:     5 * time.Minute,
				LeakDetectionInterval:   10 * time.Minute,
				ResourceTimeout:         30 * time.Minute,
				CircuitBreakerThreshold: 5,
				MaxRecoveryAttempts:     3,
			},
			wantErr: false,
		},
		{
			name: "negative cleanup interval",
			config: &RecoveryConfig{
				Enabled:             true,
				AutoCleanupInterval: -1 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero circuit breaker threshold",
			config: &RecoveryConfig{
				Enabled:                 true,
				CircuitBreakerThreshold: 0,
			},
			wantErr: true,
		},
		{
			name: "zero max recovery attempts",
			config: &RecoveryConfig{
				Enabled:             true,
				MaxRecoveryAttempts: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewRecoveryManager()
			ctx := context.Background()
			err := manager.Initialize(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
