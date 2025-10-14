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

package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/redis/go-redis/v9"
)

// setupRedisTransactionTest sets up a Redis storage for transaction testing
func setupRedisTransactionTest(t *testing.T) (*RedisStateStorage, func()) {
	t.Helper()

	if !testRedisAvailable(t) {
		return nil, func() {}
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	cleanup := func() {
		cleanupTestData(t, storage)
		storage.Close()
	}

	return storage, cleanup
}

// TestWithTransaction tests basic transaction execution
func TestWithTransaction(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create a test saga
	instance := createTestSagaInstance("tx-test-1", "test-def", saga.StateRunning)

	// Save the saga within a transaction
	opts := DefaultTransactionOptions()
	err := storage.WithTransaction(ctx, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
		// Serialize the saga
		jsonData, err := storage.serializer.SerializeSagaInstance(instance)
		if err != nil {
			return err
		}

		// Store the saga
		sagaKey := storage.getSagaKey(instance.GetID())
		pipe.Set(ctx, sagaKey, jsonData, storage.config.TTL)

		// Add to state index
		stateIndexKey := storage.getStateIndexKey(instance.GetState().String())
		pipe.SAdd(ctx, stateIndexKey, instance.GetID())

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to execute transaction: %v", err)
	}

	// Verify the saga was saved
	retrieved, err := storage.GetSaga(ctx, instance.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga: %v", err)
	}

	if retrieved.GetID() != instance.GetID() {
		t.Errorf("Expected saga ID %s, got %s", instance.GetID(), retrieved.GetID())
	}
}

// TestWithTransactionRollback tests transaction rollback on error
func TestWithTransactionRollback(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create a test saga
	instance := createTestSagaInstance("tx-rollback-1", "test-def", saga.StateRunning)

	// Execute a transaction that fails
	opts := DefaultTransactionOptions()
	opts.EnableRetry = false // Disable retry for this test

	err := storage.WithTransaction(ctx, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
		// Serialize the saga
		jsonData, err := storage.serializer.SerializeSagaInstance(instance)
		if err != nil {
			return err
		}

		// Store the saga
		sagaKey := storage.getSagaKey(instance.GetID())
		pipe.Set(ctx, sagaKey, jsonData, storage.config.TTL)

		// Return an error to trigger rollback
		return errors.New("intentional error")
	})

	if err == nil {
		t.Fatal("Expected transaction to fail")
	}

	// Verify the saga was NOT saved
	_, err = storage.GetSaga(ctx, instance.GetID())
	if !errors.Is(err, state.ErrSagaNotFound) {
		t.Errorf("Expected ErrSagaNotFound, got %v", err)
	}
}

// TestWithOptimisticLock tests optimistic locking with WATCH
func TestWithOptimisticLock(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("lock-test-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	sagaKey := storage.getSagaKey(instance.GetID())
	watchKeys := []string{sagaKey}

	// Execute with optimistic lock
	opts := DefaultTransactionOptions()
	err = storage.WithOptimisticLock(ctx, watchKeys, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
		// Get current saga
		retrieved, err := storage.GetSaga(ctx, instance.GetID())
		if err != nil {
			return err
		}

		// Modify state
		data := storage.convertToSagaInstanceData(retrieved)
		data.State = saga.StateCompleted
		data.UpdatedAt = time.Now()

		updatedInstance := &redisSagaInstance{data: data}
		jsonData, err := storage.serializer.SerializeSagaInstance(updatedInstance)
		if err != nil {
			return err
		}

		// Update in pipeline
		pipe.Set(ctx, sagaKey, jsonData, storage.config.TTL)

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to execute optimistic lock: %v", err)
	}

	// Verify the saga was updated
	retrieved, err := storage.GetSaga(ctx, instance.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga: %v", err)
	}

	if retrieved.GetState() != saga.StateCompleted {
		t.Errorf("Expected state %s, got %s", saga.StateCompleted, retrieved.GetState())
	}
}

// TestOptimisticLockConflict tests concurrent modification detection
func TestOptimisticLockConflict(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("conflict-test-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	sagaKey := storage.getSagaKey(instance.GetID())
	watchKeys := []string{sagaKey}

	// Use a channel to coordinate the conflict
	started := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(2)

	var txErr1, txErr2 error

	// First transaction - will be slower
	go func() {
		defer wg.Done()

		opts := DefaultTransactionOptions()
		opts.EnableRetry = false // Disable retry to test conflict detection

		txErr1 = storage.WithOptimisticLock(ctx, watchKeys, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
			// Signal that we've started
			started <- true

			// Wait a bit to ensure the second transaction modifies the key
			time.Sleep(100 * time.Millisecond)

			// Try to update
			retrieved, err := storage.GetSaga(ctx, instance.GetID())
			if err != nil {
				return err
			}

			data := storage.convertToSagaInstanceData(retrieved)
			data.State = saga.StateCompleted

			updatedInstance := &redisSagaInstance{data: data}
			jsonData, err := storage.serializer.SerializeSagaInstance(updatedInstance)
			if err != nil {
				return err
			}

			pipe.Set(ctx, sagaKey, jsonData, storage.config.TTL)
			return nil
		})
	}()

	// Second transaction - will be faster
	go func() {
		defer wg.Done()

		// Wait for the first transaction to start and set up WATCH
		<-started
		time.Sleep(50 * time.Millisecond)

		// Directly update the saga to cause a conflict
		err := storage.UpdateSagaState(ctx, instance.GetID(), saga.StateStepCompleted, nil)
		txErr2 = err
	}()

	wg.Wait()

	// The second transaction should succeed
	if txErr2 != nil {
		t.Errorf("Second transaction should succeed: %v", txErr2)
	}

	// The first transaction should detect the conflict
	if txErr1 == nil {
		t.Error("First transaction should fail due to conflict")
	} else if !errors.Is(txErr1, ErrTransactionConflict) && !errors.Is(txErr1, redis.TxFailedErr) {
		t.Logf("Got error: %v (type: %T)", txErr1, txErr1)
	}
}

// TestCompareAndSwap tests atomic compare-and-swap operations
func TestCompareAndSwap(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("cas-test-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Test successful CAS
	metadata := map[string]interface{}{"updated": true}
	err = storage.CompareAndSwap(ctx, instance.GetID(), saga.StateRunning, saga.StateCompleted, metadata)
	if err != nil {
		t.Fatalf("Failed to compare and swap: %v", err)
	}

	// Verify the state was updated
	retrieved, err := storage.GetSaga(ctx, instance.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga: %v", err)
	}

	if retrieved.GetState() != saga.StateCompleted {
		t.Errorf("Expected state %s, got %s", saga.StateCompleted, retrieved.GetState())
	}

	if retrievedMetadata := retrieved.GetMetadata(); retrievedMetadata["updated"] != true {
		t.Errorf("Expected metadata to be updated")
	}
}

// TestCompareAndSwapFailure tests CAS failure when state doesn't match
func TestCompareAndSwapFailure(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("cas-fail-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Try to CAS with wrong expected state
	err = storage.CompareAndSwap(ctx, instance.GetID(), saga.StateCompleted, saga.StateFailed, nil)
	if err == nil {
		t.Fatal("Expected CAS to fail")
	}

	if !errors.Is(err, ErrCompareAndSwapFailed) {
		t.Errorf("Expected ErrCompareAndSwapFailed, got %v", err)
	}

	// Verify the state was NOT changed
	retrieved, err := storage.GetSaga(ctx, instance.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga: %v", err)
	}

	if retrieved.GetState() != saga.StateRunning {
		t.Errorf("Expected state %s, got %s", saga.StateRunning, retrieved.GetState())
	}
}

// TestExecuteBatch tests batch transaction operations
func TestExecuteBatch(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create test sagas
	instance1 := createTestSagaInstance("batch-1", "test-def", saga.StateRunning)
	instance2 := createTestSagaInstance("batch-2", "test-def", saga.StateRunning)
	instance3 := createTestSagaInstance("batch-3", "test-def", saga.StateRunning)

	// Save instance1 first (for update test)
	err := storage.SaveSaga(ctx, instance1)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Create batch operations
	operations := []BatchOperation{
		{
			Type:     BatchOpSaveSaga,
			Instance: instance2,
		},
		{
			Type:     BatchOpSaveSaga,
			Instance: instance3,
		},
		{
			Type:     BatchOpUpdateState,
			SagaID:   instance1.GetID(),
			State:    saga.StateCompleted,
			Metadata: map[string]interface{}{"batch": true},
		},
	}

	// Execute batch
	opts := DefaultTransactionOptions()
	err = storage.ExecuteBatch(ctx, operations, opts)
	if err != nil {
		t.Fatalf("Failed to execute batch: %v", err)
	}

	// Verify all operations succeeded
	retrieved1, err := storage.GetSaga(ctx, instance1.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga 1: %v", err)
	}
	if retrieved1.GetState() != saga.StateCompleted {
		t.Errorf("Expected saga 1 state %s, got %s", saga.StateCompleted, retrieved1.GetState())
	}

	retrieved2, err := storage.GetSaga(ctx, instance2.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga 2: %v", err)
	}
	if retrieved2.GetID() != instance2.GetID() {
		t.Errorf("Saga 2 ID mismatch")
	}

	retrieved3, err := storage.GetSaga(ctx, instance3.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga 3: %v", err)
	}
	if retrieved3.GetID() != instance3.GetID() {
		t.Errorf("Saga 3 ID mismatch")
	}
}

// TestExecuteBatchWithSteps tests batch operations including step states
func TestExecuteBatchWithSteps(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("batch-steps-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Create step states
	now := time.Now()
	completedAt := now
	step1 := &saga.StepState{
		ID:          "step-1",
		SagaID:      instance.GetID(),
		StepIndex:   0,
		Name:        "step-1",
		State:       saga.StepStateCompleted,
		StartedAt:   &now,
		CompletedAt: &completedAt,
	}

	step2 := &saga.StepState{
		ID:        "step-2",
		SagaID:    instance.GetID(),
		StepIndex: 1,
		Name:      "step-2",
		State:     saga.StepStateRunning,
		StartedAt: &now,
	}

	// Create batch operations
	operations := []BatchOperation{
		{
			Type:      BatchOpSaveStep,
			SagaID:    instance.GetID(),
			StepState: step1,
		},
		{
			Type:      BatchOpSaveStep,
			SagaID:    instance.GetID(),
			StepState: step2,
		},
	}

	// Execute batch
	opts := DefaultTransactionOptions()
	err = storage.ExecuteBatch(ctx, operations, opts)
	if err != nil {
		t.Fatalf("Failed to execute batch: %v", err)
	}

	// Verify steps were saved
	steps, err := storage.GetStepStates(ctx, instance.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve steps: %v", err)
	}

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Verify step states
	stepMap := make(map[string]*saga.StepState)
	for _, step := range steps {
		stepMap[step.ID] = step
	}

	if stepMap["step-1"].State != saga.StepStateCompleted {
		t.Errorf("Expected step-1 state %s, got %s", saga.StepStateCompleted, stepMap["step-1"].State)
	}

	if stepMap["step-2"].State != saga.StepStateRunning {
		t.Errorf("Expected step-2 state %s, got %s", saga.StepStateRunning, stepMap["step-2"].State)
	}
}

// TestExecuteBatchDelete tests batch delete operations
func TestExecuteBatchDelete(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save test sagas
	instance1 := createTestSagaInstance("batch-del-1", "test-def", saga.StateRunning)
	instance2 := createTestSagaInstance("batch-del-2", "test-def", saga.StateRunning)

	err := storage.SaveSaga(ctx, instance1)
	if err != nil {
		t.Fatalf("Failed to save saga 1: %v", err)
	}

	err = storage.SaveSaga(ctx, instance2)
	if err != nil {
		t.Fatalf("Failed to save saga 2: %v", err)
	}

	// Create batch delete operations
	operations := []BatchOperation{
		{
			Type:   BatchOpDeleteSaga,
			SagaID: instance1.GetID(),
		},
		{
			Type:   BatchOpDeleteSaga,
			SagaID: instance2.GetID(),
		},
	}

	// Execute batch
	opts := DefaultTransactionOptions()
	err = storage.ExecuteBatch(ctx, operations, opts)
	if err != nil {
		t.Fatalf("Failed to execute batch: %v", err)
	}

	// Verify sagas were deleted
	_, err = storage.GetSaga(ctx, instance1.GetID())
	if !errors.Is(err, state.ErrSagaNotFound) {
		t.Errorf("Expected saga 1 to be deleted, got error: %v", err)
	}

	_, err = storage.GetSaga(ctx, instance2.GetID())
	if !errors.Is(err, state.ErrSagaNotFound) {
		t.Errorf("Expected saga 2 to be deleted, got error: %v", err)
	}
}

// TestTransactionRetry tests automatic retry on conflict
func TestTransactionRetry(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create a test saga
	instance := createTestSagaInstance("retry-test-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Track retry attempts
	attempts := 0

	// Execute with retry enabled
	opts := &TransactionOptions{
		MaxRetries:  3,
		RetryDelay:  10 * time.Millisecond,
		EnableRetry: true,
	}

	sagaKey := storage.getSagaKey(instance.GetID())

	err = storage.WithTransaction(ctx, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
		attempts++

		// Simulate a transient error on first attempt
		if attempts == 1 {
			return ErrTransactionConflict
		}

		// Succeed on retry
		retrieved, err := storage.GetSaga(ctx, instance.GetID())
		if err != nil {
			return err
		}

		data := storage.convertToSagaInstanceData(retrieved)
		data.State = saga.StateCompleted

		updatedInstance := &redisSagaInstance{data: data}
		jsonData, err := storage.serializer.SerializeSagaInstance(updatedInstance)
		if err != nil {
			return err
		}

		pipe.Set(ctx, sagaKey, jsonData, storage.config.TTL)
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction should succeed after retry: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

// TestConcurrentCompareAndSwap tests concurrent CAS operations
func TestConcurrentCompareAndSwap(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("concurrent-cas-1", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Run concurrent CAS operations
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			// Try to transition from Running to Completed
			err := storage.CompareAndSwap(
				ctx,
				instance.GetID(),
				saga.StateRunning,
				saga.StateCompleted,
				map[string]interface{}{"goroutine": index},
			)

			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Only one CAS should succeed
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful CAS, got %d", successCount)
	}

	// Verify final state
	retrieved, err := storage.GetSaga(ctx, instance.GetID())
	if err != nil {
		t.Fatalf("Failed to retrieve saga: %v", err)
	}

	if retrieved.GetState() != saga.StateCompleted {
		t.Errorf("Expected final state %s, got %s", saga.StateCompleted, retrieved.GetState())
	}
}

// TestConcurrentBatchOperations tests concurrent batch operations
func TestConcurrentBatchOperations(t *testing.T) {
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		t.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	const numGoroutines = 5
	const sagasPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Create batch operations for this goroutine
			operations := make([]BatchOperation, sagasPerGoroutine)
			for j := 0; j < sagasPerGoroutine; j++ {
				sagaID := fmt.Sprintf("concurrent-batch-%d-%d", goroutineID, j)
				instance := createTestSagaInstance(sagaID, "test-def", saga.StateRunning)
				operations[j] = BatchOperation{
					Type:     BatchOpSaveSaga,
					Instance: instance,
				}
			}

			// Execute batch
			opts := DefaultTransactionOptions()
			if err := storage.ExecuteBatch(ctx, operations, opts); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Batch operation failed: %v", err)
	}

	// Verify all sagas were saved
	expectedCount := numGoroutines * sagasPerGoroutine
	actualCount := 0

	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < sagasPerGoroutine; j++ {
			sagaID := fmt.Sprintf("concurrent-batch-%d-%d", i, j)
			_, err := storage.GetSaga(ctx, sagaID)
			if err == nil {
				actualCount++
			}
		}
	}

	if actualCount != expectedCount {
		t.Errorf("Expected %d sagas, got %d", expectedCount, actualCount)
	}
}

// TestIsRetryableError tests the retry error detection logic
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "transaction conflict",
			err:      ErrTransactionConflict,
			expected: true,
		},
		{
			name:     "redis tx failed",
			err:      redis.TxFailedErr,
			expected: true,
		},
		{
			name:     "redis nil",
			err:      redis.Nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// Benchmark tests

// BenchmarkCompareAndSwap benchmarks CAS operations
func BenchmarkCompareAndSwap(b *testing.B) {
	// Create a testing.T wrapper for setup
	t := &testing.T{}
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		b.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	// Create and save a test saga
	instance := createTestSagaInstance("bench-cas", "test-def", saga.StateRunning)
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		b.Fatalf("Failed to save saga: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Alternate between states
		if i%2 == 0 {
			storage.CompareAndSwap(ctx, instance.GetID(), saga.StateRunning, saga.StateCompleted, nil)
		} else {
			storage.CompareAndSwap(ctx, instance.GetID(), saga.StateCompleted, saga.StateRunning, nil)
		}
	}
}

// BenchmarkExecuteBatch benchmarks batch operations
func BenchmarkExecuteBatch(b *testing.B) {
	// Create a testing.T wrapper for setup
	t := &testing.T{}
	storage, cleanup := setupRedisTransactionTest(t)
	if storage == nil {
		b.Skip("Redis not available")
	}
	defer cleanup()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create batch operations
		operations := make([]BatchOperation, 10)
		for j := 0; j < 10; j++ {
			sagaID := fmt.Sprintf("bench-batch-%d-%d", i, j)
			instance := createTestSagaInstance(sagaID, "test-def", saga.StateRunning)
			operations[j] = BatchOperation{
				Type:     BatchOpSaveSaga,
				Instance: instance,
			}
		}

		// Execute batch
		opts := DefaultTransactionOptions()
		storage.ExecuteBatch(ctx, operations, opts)
	}
}
