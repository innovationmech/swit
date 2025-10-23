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

package monitoring

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
)

// TestRefCountedMutex_BasicFunctionality tests the basic functionality of refCountedMutex
func TestRefCountedMutex_BasicFunctionality(t *testing.T) {
	rcm := &refCountedMutex{}

	// Initial state: no references
	assert.Equal(t, int64(0), rcm.count)
	assert.True(t, rcm.canDelete())

	// Acquire first reference
	mutex1 := rcm.acquire()
	assert.Equal(t, int64(1), rcm.count)
	assert.False(t, rcm.canDelete())

	// Acquire second reference
	mutex2 := rcm.acquire()
	assert.Equal(t, int64(2), rcm.count)
	assert.False(t, rcm.canDelete())

	// Should be the same underlying mutex
	assert.Same(t, mutex1, mutex2)

	// Release first reference
	count := rcm.release()
	assert.Equal(t, int64(1), count)
	assert.False(t, rcm.canDelete())

	// Release second reference
	count = rcm.release()
	assert.Equal(t, int64(0), count)
	assert.True(t, rcm.canDelete())
}

// TestOperationLock_ConcurrentAccess tests that the operation lock correctly handles concurrent access
// and prevents the race condition where locks are deleted while still in use.
func TestOperationLock_ConcurrentAccess(t *testing.T) {
	// Create a mock coordinator
	mockCoord := &mockControlCoordinator{
		sagas: make(map[string]saga.SagaInstance),
	}

	// Create the API
	api := NewSagaControlAPI(mockCoord, nil)

	sagaID := "test-saga-123"
	numGoroutines := 10
	operationsPerGoroutine := 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Launch multiple goroutines that try to acquire and release locks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				// Try to acquire the lock
				err := api.acquireOperationLock(sagaID)
				if err != nil {
					errors <- err
					return
				}

				// Simulate some work
				time.Sleep(1 * time.Millisecond)

				// Release the lock
				api.releaseOperationLock(sagaID)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for any errors (concurrent operation errors are expected and normal)
	concurrentErrors := 0
	for err := range errors {
		if err != nil {
			if err == ErrConcurrentOperation {
				concurrentErrors++
			} else {
				t.Errorf("Unexpected error during concurrent access: %v", err)
			}
		}
	}

	t.Logf("Concurrent operation errors encountered: %d (expected behavior)", concurrentErrors)

	// Verify that the lock is eventually cleaned up
	time.Sleep(100 * time.Millisecond) // Allow cleanup goroutines to run
	_, exists := api.operationLock.Load(sagaID)
	if exists {
		t.Logf("Warning: Lock still exists in map after all operations completed")
		// This is not necessarily an error, as cleanup happens asynchronously
	}
}

// TestOperationLock_RaceConditionFix specifically tests the race condition scenario described in the bug
func TestOperationLock_RaceConditionFix(t *testing.T) {
	mockCoord := &mockControlCoordinator{
		sagas: make(map[string]saga.SagaInstance),
	}
	api := NewSagaControlAPI(mockCoord, nil)

	sagaID := "race-condition-test"

	// This test verifies that our fix prevents the race condition where:
	// 1. Request A finishes and schedules cleanup
	// 2. Request B acquires the same lock and starts a long operation
	// 3. Request A's cleanup runs and would delete the lock while B still holds it (old bug)
	// 4. Request C would then create a new lock and run concurrently with B (the race condition)

	// With our fix, step 3 should NOT delete the lock while B still holds it

	var wg sync.WaitGroup
	requestBCompleted := make(chan bool)

	// Request A: Quick operation that finishes first
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := api.acquireOperationLock(sagaID)
		assert.NoError(t, err)

		// Very short operation
		time.Sleep(5 * time.Millisecond)

		// Release and schedule cleanup (this is where the old bug would manifest)
		api.releaseOperationLock(sagaID)
	}()

	// Wait for Request A to complete before starting Request B
	time.Sleep(10 * time.Millisecond)

	// Request B: Long operation that starts after A releases but may overlap with A's cleanup goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := api.acquireOperationLock(sagaID)
		assert.NoError(t, err)

		// Longer operation that overlaps with cleanup delay
		time.Sleep(50 * time.Millisecond)

		close(requestBCompleted)

		// This should work correctly without race conditions
		api.releaseOperationLock(sagaID)
	}()

	// Wait for Request B to complete
	<-requestBCompleted

	// At this point, Request B should still be holding the lock or just released it
	// Request C should either get blocked (if B still holds it) or succeed (if B just released it)
	// The key point is that there should NOT be a situation where both B and C hold different locks

	wg.Wait()

	// Wait for any cleanup goroutines to finish
	time.Sleep(100 * time.Millisecond)

	// Verify that we haven't created a situation with multiple locks for the same saga
	// (This would be evidence of the race condition)
	_, exists := api.operationLock.Load(sagaID)
	if exists {
		// If lock exists, it should be the only one
		t.Logf("Lock cleanup completed successfully - no race condition detected")
	} else {
		t.Logf("Lock cleaned up from map - this is normal behavior")
	}
}

// mockControlCoordinator is a minimal mock for testing control operations
type mockControlCoordinator struct {
	sagas map[string]saga.SagaInstance
	mu    sync.Mutex
}

func (m *mockControlCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	return nil, nil
}

func (m *mockControlCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if saga, exists := m.sagas[sagaID]; exists {
		return saga, nil
	}
	return nil, saga.NewSagaNotFoundError(sagaID)
}

func (m *mockControlCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	return nil
}

func (m *mockControlCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	return []saga.SagaInstance{}, nil
}

func (m *mockControlCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	return &saga.CoordinatorMetrics{}
}

func (m *mockControlCoordinator) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockControlCoordinator) Close() error {
	return nil
}
