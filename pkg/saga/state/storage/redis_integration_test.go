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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// TestRedisIntegration_FullLifecycle tests the complete lifecycle of a saga in Redis storage
func TestRedisIntegration_FullLifecycle(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// 1. Create and save a new saga
	saga1 := createTestSagaInstance("saga-lifecycle-1", "def-lifecycle", saga.StateRunning)
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// 2. Retrieve the saga
	retrieved, err := storage.GetSaga(ctx, "saga-lifecycle-1")
	if err != nil {
		t.Fatalf("Failed to get saga: %v", err)
	}
	if retrieved.GetID() != "saga-lifecycle-1" {
		t.Errorf("Retrieved saga ID = %v, want saga-lifecycle-1", retrieved.GetID())
	}
	if retrieved.GetState() != saga.StateRunning {
		t.Errorf("Retrieved saga state = %v, want StateRunning", retrieved.GetState())
	}

	// 3. Save step states
	now := time.Now()
	step1 := &saga.StepState{
		ID:        "step-1",
		Name:      "Step1",
		State:     saga.StepStateCompleted,
		Attempts:  1,
		CreatedAt: now,
		StartedAt: &now,
	}
	if err := storage.SaveStepState(ctx, "saga-lifecycle-1", step1); err != nil {
		t.Fatalf("Failed to save step state: %v", err)
	}

	step2 := &saga.StepState{
		ID:        "step-2",
		Name:      "Step2",
		State:     saga.StepStateRunning,
		Attempts:  1,
		CreatedAt: now,
		StartedAt: &now,
	}
	if err := storage.SaveStepState(ctx, "saga-lifecycle-1", step2); err != nil {
		t.Fatalf("Failed to save step state: %v", err)
	}

	// 4. Retrieve step states
	steps, err := storage.GetStepStates(ctx, "saga-lifecycle-1")
	if err != nil {
		t.Fatalf("Failed to get step states: %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Got %d step states, want 2", len(steps))
	}

	// 5. Update saga state to completed
	if err := storage.UpdateSagaState(ctx, "saga-lifecycle-1", saga.StateCompleted, map[string]interface{}{
		"result": "success",
	}); err != nil {
		t.Fatalf("Failed to update saga state: %v", err)
	}

	// 6. Verify the update
	updated, err := storage.GetSaga(ctx, "saga-lifecycle-1")
	if err != nil {
		t.Fatalf("Failed to get updated saga: %v", err)
	}
	if updated.GetState() != saga.StateCompleted {
		t.Errorf("Updated saga state = %v, want StateCompleted", updated.GetState())
	}
	meta := updated.GetMetadata()
	if meta["result"] != "success" {
		t.Errorf("Updated saga metadata[result] = %v, want success", meta["result"])
	}

	// 7. Delete the saga
	if err := storage.DeleteSaga(ctx, "saga-lifecycle-1"); err != nil {
		t.Fatalf("Failed to delete saga: %v", err)
	}

	// 8. Verify deletion
	_, err = storage.GetSaga(ctx, "saga-lifecycle-1")
	if err == nil {
		t.Error("Saga should not exist after deletion")
	}
	if err != state.ErrSagaNotFound {
		t.Errorf("Expected ErrSagaNotFound, got %v", err)
	}
}

// TestRedisIntegration_MultipleActiveSagas tests managing multiple active sagas
func TestRedisIntegration_MultipleActiveSagas(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create multiple sagas with different states and definitions
	sagas := []saga.SagaInstance{
		createTestSagaInstance("saga-1", "def-A", saga.StateRunning),
		createTestSagaInstance("saga-2", "def-A", saga.StateRunning),
		createTestSagaInstance("saga-3", "def-B", saga.StateRunning),
		createTestSagaInstance("saga-4", "def-A", saga.StateCompleted),
		createTestSagaInstance("saga-5", "def-B", saga.StateCompensating),
		createTestSagaInstance("saga-6", "def-A", saga.StateFailed),
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("Failed to save saga %s: %v", s.GetID(), err)
		}
	}

	// Test 1: Get all active sagas (should be 3)
	activeSagas, err := storage.GetActiveSagas(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get active sagas: %v", err)
	}
	if len(activeSagas) != 3 {
		t.Errorf("Got %d active sagas, want 3", len(activeSagas))
	}

	// Test 2: Filter by state
	runningSagas, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		States: []saga.SagaState{saga.StateRunning},
	})
	if err != nil {
		t.Fatalf("Failed to get running sagas: %v", err)
	}
	if len(runningSagas) != 3 {
		t.Errorf("Got %d running sagas, want 3", len(runningSagas))
	}

	// Test 3: Filter by definition ID
	defASagas, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		DefinitionIDs: []string{"def-A"},
	})
	if err != nil {
		t.Fatalf("Failed to get def-A sagas: %v", err)
	}
	if len(defASagas) != 2 {
		t.Errorf("Got %d def-A active sagas, want 2", len(defASagas))
	}

	// Test 4: Filter by state and definition ID
	defARunning, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		States:        []saga.SagaState{saga.StateRunning},
		DefinitionIDs: []string{"def-A"},
	})
	if err != nil {
		t.Fatalf("Failed to get def-A running sagas: %v", err)
	}
	if len(defARunning) != 2 {
		t.Errorf("Got %d def-A running sagas, want 2", len(defARunning))
	}
}

// TestRedisIntegration_TimeoutDetection tests timeout detection functionality
func TestRedisIntegration_TimeoutDetection(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()
	now := time.Now()

	// Create sagas with different start times and timeouts
	sagas := []struct {
		id        string
		startTime time.Time
		timeout   time.Duration
	}{
		{"saga-timeout-1", now.Add(-2 * time.Hour), 1 * time.Hour},   // Should timeout
		{"saga-timeout-2", now.Add(-90 * time.Minute), 1 * time.Hour}, // Should timeout
		{"saga-timeout-3", now.Add(-30 * time.Minute), 1 * time.Hour}, // Should not timeout
		{"saga-timeout-4", now.Add(-10 * time.Minute), 2 * time.Hour}, // Should not timeout
	}

	for _, s := range sagas {
		sagaInst := createTestSagaInstance(s.id, "def-timeout", saga.StateRunning)
		sagaInst.startTime = s.startTime
		sagaInst.timeout = s.timeout
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			t.Fatalf("Failed to save saga %s: %v", s.id, err)
		}
	}

	// Query for timeouts before now
	timedOut, err := storage.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Fatalf("Failed to get timeout sagas: %v", err)
	}

	if len(timedOut) != 2 {
		t.Errorf("Got %d timed out sagas, want 2", len(timedOut))
	}

	// Verify the correct sagas are returned
	foundIDs := make(map[string]bool)
	for _, s := range timedOut {
		foundIDs[s.GetID()] = true
	}
	if !foundIDs["saga-timeout-1"] || !foundIDs["saga-timeout-2"] {
		t.Errorf("Expected saga-timeout-1 and saga-timeout-2, got %v", foundIDs)
	}
}

// TestRedisIntegration_Cleanup tests cleanup of expired sagas
func TestRedisIntegration_Cleanup(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()
	now := time.Now()

	// Create terminal sagas with different update times
	sagas := []struct {
		id        string
		state     saga.SagaState
		updatedAt time.Time
	}{
		{"saga-old-1", saga.StateCompleted, now.Add(-3 * time.Hour)},
		{"saga-old-2", saga.StateFailed, now.Add(-2 * time.Hour)},
		{"saga-recent-1", saga.StateCompleted, now.Add(-30 * time.Minute)},
		{"saga-recent-2", saga.StateCompensated, now.Add(-15 * time.Minute)},
	}

	for _, s := range sagas {
		sagaInst := createTestSagaInstance(s.id, "def-cleanup", s.state)
		sagaInst.updatedAt = s.updatedAt
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			t.Fatalf("Failed to save saga %s: %v", s.id, err)
		}
	}

	// Cleanup sagas older than 1 hour
	olderThan := now.Add(-1 * time.Hour)
	if err := storage.CleanupExpiredSagas(ctx, olderThan); err != nil {
		t.Fatalf("Failed to cleanup expired sagas: %v", err)
	}

	// Verify old sagas are deleted
	for _, s := range []string{"saga-old-1", "saga-old-2"} {
		_, err := storage.GetSaga(ctx, s)
		if err == nil {
			t.Errorf("Saga %s should be deleted", s)
		}
		if err != state.ErrSagaNotFound {
			t.Errorf("Expected ErrSagaNotFound for %s, got %v", s, err)
		}
	}

	// Verify recent sagas still exist
	for _, s := range []string{"saga-recent-1", "saga-recent-2"} {
		_, err := storage.GetSaga(ctx, s)
		if err != nil {
			t.Errorf("Saga %s should still exist: %v", s, err)
		}
	}
}

// TestRedisIntegration_ConcurrentWrites tests concurrent writes to Redis storage
func TestRedisIntegration_ConcurrentWrites(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()
	const numGoroutines = 50
	const numOperationsPerGoroutine = 20

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*numOperationsPerGoroutine)

	// Launch concurrent writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				sagaID := fmt.Sprintf("saga-concurrent-%d-%d", goroutineID, j)
				sagaInst := createTestSagaInstance(sagaID, "def-concurrent", saga.StateRunning)
				if err := storage.SaveSaga(ctx, sagaInst); err != nil {
					errCh <- fmt.Errorf("goroutine %d operation %d: %w", goroutineID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	errorCount := 0
	for err := range errCh {
		t.Errorf("Concurrent write error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Got %d errors during concurrent writes", errorCount)
	}

	// Verify all sagas were saved
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOperationsPerGoroutine; j++ {
			sagaID := fmt.Sprintf("saga-concurrent-%d-%d", i, j)
			_, err := storage.GetSaga(ctx, sagaID)
			if err != nil {
				t.Errorf("Failed to retrieve saga %s: %v", sagaID, err)
			}
		}
	}
}

// TestRedisIntegration_ConcurrentReadsWrites tests concurrent reads and writes
func TestRedisIntegration_ConcurrentReadsWrites(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Pre-populate with some sagas
	for i := 0; i < 10; i++ {
		sagaID := fmt.Sprintf("saga-rw-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-rw", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			t.Fatalf("Failed to save initial saga: %v", err)
		}
	}

	const numReaders = 20
	const numWriters = 10
	const duration = 2 * time.Second

	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	errCh := make(chan error, numReaders+numWriters)

	// Launch readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					sagaID := fmt.Sprintf("saga-rw-%d", readerID%10)
					_, err := storage.GetSaga(ctx, sagaID)
					if err != nil && err != state.ErrSagaNotFound {
						errCh <- fmt.Errorf("reader %d: %w", readerID, err)
						return
					}
				}
			}
		}(i)
	}

	// Launch writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			opCount := 0
			for {
				select {
				case <-stopCh:
					return
				default:
					sagaID := fmt.Sprintf("saga-rw-%d", writerID%10)
					sagaInst := createTestSagaInstance(sagaID, "def-rw", saga.StateRunning)
					if err := storage.SaveSaga(ctx, sagaInst); err != nil {
						errCh <- fmt.Errorf("writer %d: %w", writerID, err)
						return
					}
					opCount++
				}
			}
		}(i)
	}

	// Run for duration
	time.Sleep(duration)
	close(stopCh)
	wg.Wait()
	close(errCh)

	// Check for errors
	errorCount := 0
	for err := range errCh {
		t.Errorf("Concurrent read/write error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Got %d errors during concurrent reads/writes", errorCount)
	}
}

// TestRedisIntegration_NetworkFailure tests behavior during network failures
func TestRedisIntegration_NetworkFailure(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	config.DialTimeout = 1 * time.Second
	config.ReadTimeout = 1 * time.Second
	config.WriteTimeout = 1 * time.Second
	config.MaxRetries = 0 // Disable retries for faster failure

	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Save a saga successfully
	saga1 := createTestSagaInstance("saga-network-1", "def-network", saga.StateRunning)
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Simulate network timeout with a short context
	shortCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure context is expired

	// This should fail due to context cancellation
	err = storage.SaveSaga(shortCtx, createTestSagaInstance("saga-network-2", "def-network", saga.StateRunning))
	if err == nil {
		t.Error("Expected error due to context timeout, got nil")
	}
}

// TestRedisIntegration_DataConsistency tests data consistency across operations
func TestRedisIntegration_DataConsistency(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create a saga with complex metadata
	saga1 := createTestSagaInstance("saga-consistency-1", "def-consistency", saga.StateRunning)
	saga1.metadata = map[string]interface{}{
		"string": "value",
		"number": 42,
		"float":  3.14,
		"bool":   true,
		"nested": map[string]interface{}{
			"key": "value",
		},
		"array": []interface{}{"a", "b", "c"},
	}

	// Save the saga
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Retrieve and verify all fields
	retrieved, err := storage.GetSaga(ctx, "saga-consistency-1")
	if err != nil {
		t.Fatalf("Failed to get saga: %v", err)
	}

	// Verify basic fields
	if retrieved.GetID() != saga1.GetID() {
		t.Errorf("ID mismatch: got %v, want %v", retrieved.GetID(), saga1.GetID())
	}
	if retrieved.GetDefinitionID() != saga1.GetDefinitionID() {
		t.Errorf("DefinitionID mismatch: got %v, want %v", retrieved.GetDefinitionID(), saga1.GetDefinitionID())
	}
	if retrieved.GetState() != saga1.GetState() {
		t.Errorf("State mismatch: got %v, want %v", retrieved.GetState(), saga1.GetState())
	}
	if retrieved.GetCurrentStep() != saga1.GetCurrentStep() {
		t.Errorf("CurrentStep mismatch: got %v, want %v", retrieved.GetCurrentStep(), saga1.GetCurrentStep())
	}
	if retrieved.GetTotalSteps() != saga1.GetTotalSteps() {
		t.Errorf("TotalSteps mismatch: got %v, want %v", retrieved.GetTotalSteps(), saga1.GetTotalSteps())
	}
	if retrieved.GetTimeout() != saga1.GetTimeout() {
		t.Errorf("Timeout mismatch: got %v, want %v", retrieved.GetTimeout(), saga1.GetTimeout())
	}
	if retrieved.GetTraceID() != saga1.GetTraceID() {
		t.Errorf("TraceID mismatch: got %v, want %v", retrieved.GetTraceID(), saga1.GetTraceID())
	}

	// Verify metadata
	meta := retrieved.GetMetadata()
	if meta["string"] != "value" {
		t.Errorf("Metadata[string] = %v, want value", meta["string"])
	}
	// Note: JSON unmarshaling converts numbers to float64
	if meta["number"] != float64(42) {
		t.Errorf("Metadata[number] = %v, want 42", meta["number"])
	}
}

// TestRedisIntegration_HealthCheck tests health check functionality
func TestRedisIntegration_HealthCheck(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Health check should pass
	if err := storage.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}

	// Close storage and verify health check fails
	storage.Close()
	if err := storage.HealthCheck(ctx); err == nil {
		t.Error("HealthCheck should fail after Close()")
	}
}

// TestRedisIntegration_MetricsCollection tests metrics collection
func TestRedisIntegration_MetricsCollection(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	config.EnableMetrics = true
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Verify metrics collector is initialized
	metrics := storage.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics collector should be initialized")
	}

	// Perform some operations
	saga1 := createTestSagaInstance("saga-metrics-1", "def-metrics", saga.StateRunning)
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	_, err = storage.GetSaga(ctx, "saga-metrics-1")
	if err != nil {
		t.Fatalf("Failed to get saga: %v", err)
	}

	// Note: We can't easily verify metric values without accessing internal state,
	// but we can verify the metrics collector exists and operations complete successfully
}

// TestRedisIntegration_ConnectionResilience tests connection resilience
func TestRedisIntegration_ConnectionResilience(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	config.MaxRetries = 3
	config.MinRetryBackoff = 10 * time.Millisecond
	config.MaxRetryBackoff = 100 * time.Millisecond

	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Perform operations to verify connection resilience
	for i := 0; i < 10; i++ {
		sagaID := fmt.Sprintf("saga-resilience-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-resilience", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			t.Errorf("Failed to save saga %s: %v", sagaID, err)
		}

		retrieved, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			t.Errorf("Failed to get saga %s: %v", sagaID, err)
		}
		if retrieved.GetID() != sagaID {
			t.Errorf("Retrieved saga ID = %v, want %s", retrieved.GetID(), sagaID)
		}
	}
}

// TestRedisIntegration_LargePayload tests handling of large payloads
func TestRedisIntegration_LargePayload(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create a saga with large metadata
	saga1 := createTestSagaInstance("saga-large-1", "def-large", saga.StateRunning)
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}
	saga1.metadata = largeData

	// Save the saga
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga with large payload: %v", err)
	}

	// Retrieve and verify
	retrieved, err := storage.GetSaga(ctx, "saga-large-1")
	if err != nil {
		t.Fatalf("Failed to get saga with large payload: %v", err)
	}

	meta := retrieved.GetMetadata()
	if len(meta) != 1001 { // 1000 + 1 original "test" key
		t.Errorf("Retrieved metadata size = %d, want 1001", len(meta))
	}
}

// TestRedisIntegration_ConnectionFailure tests behavior when Redis is unavailable
func TestRedisIntegration_ConnectionFailure(t *testing.T) {
	// This test intentionally uses an invalid address
	config := &RedisConfig{
		Mode:        RedisModeStandalone,
		Addr:        "localhost:99999", // Invalid port
		DB:          0,
		KeyPrefix:   "test:",
		TTL:         1 * time.Hour,
		DialTimeout: 1 * time.Second,
		MaxRetries:  0,
	}

	ctx := context.Background()

	// Creation should fail
	_, err := NewRedisStateStorage(config)
	if err == nil {
		t.Error("Expected error when creating storage with invalid address, got nil")
	}

	// Test with retry
	_, err = NewRedisStateStorageWithRetry(ctx, config, 2)
	if err == nil {
		t.Error("Expected error when creating storage with retry with invalid address, got nil")
	}
}

// TestRedisIntegration_TTLExpiration tests that TTL is properly set
func TestRedisIntegration_TTLExpiration(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	config.TTL = 2 * time.Second // Short TTL for testing
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a saga
	saga1 := createTestSagaInstance("saga-ttl-1", "def-ttl", saga.StateRunning)
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Verify saga exists
	_, err = storage.GetSaga(ctx, "saga-ttl-1")
	if err != nil {
		t.Fatalf("Failed to get saga: %v", err)
	}

	// Check TTL is set on the key
	sagaKey := storage.getSagaKey("saga-ttl-1")
	ttl := storage.client.TTL(ctx, sagaKey).Val()
	if ttl <= 0 {
		t.Errorf("TTL should be positive, got %v", ttl)
	}
	if ttl > 2*time.Second {
		t.Errorf("TTL should be <= 2s, got %v", ttl)
	}
}

// TestRedisIntegration_StateIndexConsistency tests that state indexes are consistent
func TestRedisIntegration_StateIndexConsistency(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a saga in running state
	saga1 := createTestSagaInstance("saga-index-1", "def-index", saga.StateRunning)
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Verify it's in the running index
	runningKey := storage.getStateIndexKey(saga.StateRunning.String())
	isMember := storage.client.SIsMember(ctx, runningKey, "saga-index-1").Val()
	if !isMember {
		t.Error("Saga should be in running state index")
	}

	// Update to completed state
	if err := storage.UpdateSagaState(ctx, "saga-index-1", saga.StateCompleted, nil); err != nil {
		t.Fatalf("Failed to update saga state: %v", err)
	}

	// Verify it's removed from running index
	isMember = storage.client.SIsMember(ctx, runningKey, "saga-index-1").Val()
	if isMember {
		t.Error("Saga should not be in running state index after update")
	}

	// Verify it's in the completed index
	completedKey := storage.getStateIndexKey(saga.StateCompleted.String())
	isMember = storage.client.SIsMember(ctx, completedKey, "saga-index-1").Val()
	if !isMember {
		t.Error("Saga should be in completed state index")
	}

	// Delete saga
	if err := storage.DeleteSaga(ctx, "saga-index-1"); err != nil {
		t.Fatalf("Failed to delete saga: %v", err)
	}

	// Verify it's removed from completed index
	isMember = storage.client.SIsMember(ctx, completedKey, "saga-index-1").Val()
	if isMember {
		t.Error("Saga should not be in completed state index after deletion")
	}
}

// TestRedisIntegration_FilterByCreatedTime tests filtering by creation time
func TestRedisIntegration_FilterByCreatedTime(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()
	now := time.Now()

	// Create sagas with different creation times
	sagas := []struct {
		id        string
		createdAt time.Time
	}{
		{"saga-time-1", now.Add(-2 * time.Hour)},
		{"saga-time-2", now.Add(-1 * time.Hour)},
		{"saga-time-3", now.Add(-30 * time.Minute)},
	}

	for _, s := range sagas {
		sagaInst := createTestSagaInstance(s.id, "def-time", saga.StateRunning)
		sagaInst.createdAt = s.createdAt
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			t.Fatalf("Failed to save saga %s: %v", s.id, err)
		}
	}

	// Filter by created after 1.5 hours ago
	createdAfter := now.Add(-90 * time.Minute)
	filtered, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		CreatedAfter: &createdAfter,
	})
	if err != nil {
		t.Fatalf("Failed to get filtered sagas: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Got %d filtered sagas, want 2", len(filtered))
	}

	// Filter by created before 45 minutes ago
	createdBefore := now.Add(-45 * time.Minute)
	filtered, err = storage.GetActiveSagas(ctx, &saga.SagaFilter{
		CreatedBefore: &createdBefore,
	})
	if err != nil {
		t.Fatalf("Failed to get filtered sagas: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Got %d filtered sagas, want 2", len(filtered))
	}
}

// TestRedisIntegration_FilterByMetadata tests filtering by metadata
func TestRedisIntegration_FilterByMetadata(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create sagas with different metadata
	saga1 := createTestSagaInstance("saga-meta-1", "def-meta", saga.StateRunning)
	saga1.metadata = map[string]interface{}{
		"type":     "A",
		"priority": "high",
	}

	saga2 := createTestSagaInstance("saga-meta-2", "def-meta", saga.StateRunning)
	saga2.metadata = map[string]interface{}{
		"type":     "A",
		"priority": "low",
	}

	saga3 := createTestSagaInstance("saga-meta-3", "def-meta", saga.StateRunning)
	saga3.metadata = map[string]interface{}{
		"type":     "B",
		"priority": "high",
	}

	for _, s := range []saga.SagaInstance{saga1, saga2, saga3} {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("Failed to save saga: %v", err)
		}
	}

	// Filter by type=A
	filtered, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		Metadata: map[string]interface{}{
			"type": "A",
		},
	})
	if err != nil {
		t.Fatalf("Failed to get filtered sagas: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Got %d sagas with type=A, want 2", len(filtered))
	}

	// Filter by type=A and priority=high
	filtered, err = storage.GetActiveSagas(ctx, &saga.SagaFilter{
		Metadata: map[string]interface{}{
			"type":     "A",
			"priority": "high",
		},
	})
	if err != nil {
		t.Fatalf("Failed to get filtered sagas: %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("Got %d sagas with type=A and priority=high, want 1", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].GetID() != "saga-meta-1" {
		t.Errorf("Expected saga-meta-1, got %s", filtered[0].GetID())
	}
}

