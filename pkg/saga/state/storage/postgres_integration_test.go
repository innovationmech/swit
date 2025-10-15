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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// testPostgresAvailable checks if PostgreSQL is available for testing.
// Set POSTGRES_TEST_DSN environment variable to enable PostgreSQL integration tests.
// Example: export POSTGRES_TEST_DSN="host=localhost port=5432 user=postgres password=postgres dbname=swit_test sslmode=disable"
func testPostgresAvailable(t *testing.T) bool {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("Skipping PostgreSQL integration tests. Set POSTGRES_TEST_DSN to enable.")
		return false
	}
	return true
}

// getTestPostgresConfig returns a test PostgreSQL configuration.
func getTestPostgresConfig() *PostgresConfig {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=swit_test sslmode=disable"
	}

	return &PostgresConfig{
		DSN:               dsn,
		MaxOpenConns:      10,
		MaxIdleConns:      5,
		ConnMaxLifetime:   30 * time.Minute,
		ConnMaxIdleTime:   10 * time.Minute,
		ConnectionTimeout: 5 * time.Second,
		QueryTimeout:      10 * time.Second,
		AutoMigrate:       true,
		TablePrefix:       "test_",
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		MaxRetryBackoff:   5 * time.Second,
	}
}

// cleanupTestPostgresData cleans up all test data from PostgreSQL.
func cleanupTestPostgresData(t *testing.T, storage *PostgresStateStorage) {
	if storage == nil || storage.db == nil {
		return
	}

	ctx := context.Background()

	// Delete all test data from tables
	tables := []string{
		storage.eventsTable,
		storage.stepsTable,
		storage.instancesTable,
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := storage.db.ExecContext(ctx, query); err != nil {
			t.Logf("Warning: Failed to cleanup table %s: %v", table, err)
		}
	}
}

// ==========================
// Full Lifecycle Tests
// ==========================

// TestPostgresIntegration_FullLifecycle tests the complete lifecycle of a saga in PostgreSQL storage.
func TestPostgresIntegration_FullLifecycle(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

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
		ID:          "step-1",
		SagaID:      "saga-lifecycle-1",
		StepIndex:   0,
		Name:        "Step1",
		State:       saga.StepStateCompleted,
		Attempts:    1,
		MaxAttempts: 3,
		CreatedAt:   now,
		StartedAt:   &now,
		CompletedAt: &now,
	}
	if err := storage.SaveStepState(ctx, "saga-lifecycle-1", step1); err != nil {
		t.Fatalf("Failed to save step state: %v", err)
	}

	step2 := &saga.StepState{
		ID:          "step-2",
		SagaID:      "saga-lifecycle-1",
		StepIndex:   1,
		Name:        "Step2",
		State:       saga.StepStateRunning,
		Attempts:    1,
		MaxAttempts: 3,
		CreatedAt:   now,
		StartedAt:   &now,
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

	// 8. Verify deletion (steps should also be deleted due to CASCADE)
	_, err = storage.GetSaga(ctx, "saga-lifecycle-1")
	if err == nil {
		t.Error("Saga should not exist after deletion")
	}
	if err != state.ErrSagaNotFound {
		t.Errorf("Expected ErrSagaNotFound, got %v", err)
	}

	// Verify steps are also deleted
	steps, err = storage.GetStepStates(ctx, "saga-lifecycle-1")
	if err != nil {
		t.Fatalf("Failed to get step states after saga deletion: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Expected 0 steps after saga deletion, got %d", len(steps))
	}
}

// ==========================
// Multiple Saga Tests
// ==========================

// TestPostgresIntegration_MultipleActiveSagas tests managing multiple active sagas.
func TestPostgresIntegration_MultipleActiveSagas(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

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

	// Test 1: Get all active sagas (should be 3: Running + Compensating)
	activeSagas, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		States: []saga.SagaState{saga.StateRunning, saga.StateCompensating, saga.StateStepCompleted},
	})
	if err != nil {
		t.Fatalf("Failed to get active sagas: %v", err)
	}
	if len(activeSagas) != 4 {
		t.Errorf("Got %d active sagas, want 4", len(activeSagas))
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
	if len(defASagas) != 4 {
		t.Errorf("Got %d def-A sagas, want 4 (all def-A sagas)", len(defASagas))
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

	// Test 5: Test pagination
	page1, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		Limit:  2,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("Got %d sagas on page 1, want 2", len(page1))
	}

	page2, err := storage.GetActiveSagas(ctx, &saga.SagaFilter{
		Limit:  2,
		Offset: 2,
	})
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}
	if len(page2) > 2 {
		t.Errorf("Got %d sagas on page 2, want <= 2", len(page2))
	}

	// Test 6: Count sagas
	count, err := storage.CountSagas(ctx, &saga.SagaFilter{
		States: []saga.SagaState{saga.StateRunning},
	})
	if err != nil {
		t.Fatalf("Failed to count sagas: %v", err)
	}
	if count != 3 {
		t.Errorf("Got count %d, want 3", count)
	}
}

// ==========================
// Transaction Tests
// ==========================

// TestPostgresIntegration_TransactionCommit tests transaction commit behavior.
func TestPostgresIntegration_TransactionCommit(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()

	// Begin transaction
	tx, err := storage.BeginTransaction(ctx, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Save multiple sagas within transaction
	saga1 := createTestSagaInstance("saga-tx-1", "def-tx", saga.StateRunning)
	saga2 := createTestSagaInstance("saga-tx-2", "def-tx", saga.StateRunning)

	if err := tx.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga1 in transaction: %v", err)
	}

	if err := tx.SaveSaga(ctx, saga2); err != nil {
		t.Fatalf("Failed to save saga2 in transaction: %v", err)
	}

	// Verify sagas are not visible outside transaction yet
	_, err = storage.GetSaga(ctx, "saga-tx-1")
	if err != state.ErrSagaNotFound {
		t.Errorf("Saga should not be visible outside transaction before commit, got error: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify sagas are now visible
	retrieved1, err := storage.GetSaga(ctx, "saga-tx-1")
	if err != nil {
		t.Fatalf("Failed to get saga1 after commit: %v", err)
	}
	if retrieved1.GetID() != "saga-tx-1" {
		t.Errorf("Retrieved saga ID = %v, want saga-tx-1", retrieved1.GetID())
	}

	retrieved2, err := storage.GetSaga(ctx, "saga-tx-2")
	if err != nil {
		t.Fatalf("Failed to get saga2 after commit: %v", err)
	}
	if retrieved2.GetID() != "saga-tx-2" {
		t.Errorf("Retrieved saga ID = %v, want saga-tx-2", retrieved2.GetID())
	}
}

// TestPostgresIntegration_TransactionRollback tests transaction rollback behavior.
func TestPostgresIntegration_TransactionRollback(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()

	// Begin transaction
	tx, err := storage.BeginTransaction(ctx, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Save sagas within transaction
	saga1 := createTestSagaInstance("saga-rollback-1", "def-rollback", saga.StateRunning)
	saga2 := createTestSagaInstance("saga-rollback-2", "def-rollback", saga.StateRunning)

	if err := tx.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga1 in transaction: %v", err)
	}

	if err := tx.SaveSaga(ctx, saga2); err != nil {
		t.Fatalf("Failed to save saga2 in transaction: %v", err)
	}

	// Rollback transaction
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify sagas are not visible after rollback
	_, err = storage.GetSaga(ctx, "saga-rollback-1")
	if err != state.ErrSagaNotFound {
		t.Errorf("Saga should not exist after rollback, got error: %v", err)
	}

	_, err = storage.GetSaga(ctx, "saga-rollback-2")
	if err != state.ErrSagaNotFound {
		t.Errorf("Saga should not exist after rollback, got error: %v", err)
	}
}

// TestPostgresIntegration_BatchOperations tests batch save operations.
func TestPostgresIntegration_BatchOperations(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()

	// Create multiple sagas
	sagas := make([]saga.SagaInstance, 10)
	for i := 0; i < 10; i++ {
		sagas[i] = createTestSagaInstance(fmt.Sprintf("saga-batch-%d", i), "def-batch", saga.StateRunning)
	}

	// Batch save sagas
	if err := storage.BatchSaveSagas(ctx, sagas, 30*time.Second); err != nil {
		t.Fatalf("Failed to batch save sagas: %v", err)
	}

	// Verify all sagas were saved
	for i := 0; i < 10; i++ {
		sagaID := fmt.Sprintf("saga-batch-%d", i)
		_, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			t.Errorf("Failed to get saga %s: %v", sagaID, err)
		}
	}
}

// ==========================
// Concurrent Operations Tests
// ==========================

// TestPostgresIntegration_ConcurrentWrites tests concurrent writes to PostgreSQL storage.
func TestPostgresIntegration_ConcurrentWrites(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()
	const numGoroutines = 20
	const numOperationsPerGoroutine = 10

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
	count := 0
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOperationsPerGoroutine; j++ {
			sagaID := fmt.Sprintf("saga-concurrent-%d-%d", i, j)
			_, err := storage.GetSaga(ctx, sagaID)
			if err == nil {
				count++
			}
		}
	}

	expected := numGoroutines * numOperationsPerGoroutine
	if count != expected {
		t.Errorf("Expected %d sagas, got %d", expected, count)
	}
}

// TestPostgresIntegration_ConcurrentTransactions tests concurrent transaction handling.
func TestPostgresIntegration_ConcurrentTransactions(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()
	const numTransactions = 10

	var wg sync.WaitGroup
	errCh := make(chan error, numTransactions)

	// Launch concurrent transactions
	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go func(txID int) {
			defer wg.Done()

			tx, err := storage.BeginTransaction(ctx, 30*time.Second)
			if err != nil {
				errCh <- fmt.Errorf("tx %d: failed to begin: %w", txID, err)
				return
			}

			// Save multiple sagas in transaction
			for j := 0; j < 5; j++ {
				sagaID := fmt.Sprintf("saga-tx-%d-%d", txID, j)
				saga := createTestSagaInstance(sagaID, "def-tx", saga.StateRunning)
				if err := tx.SaveSaga(ctx, saga); err != nil {
					tx.Rollback(ctx)
					errCh <- fmt.Errorf("tx %d: failed to save saga: %w", txID, err)
					return
				}
			}

			// Commit transaction
			if err := tx.Commit(ctx); err != nil {
				errCh <- fmt.Errorf("tx %d: failed to commit: %w", txID, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		t.Errorf("Concurrent transaction error: %v", err)
	}

	// Verify all sagas were saved
	count := 0
	for i := 0; i < numTransactions; i++ {
		for j := 0; j < 5; j++ {
			sagaID := fmt.Sprintf("saga-tx-%d-%d", i, j)
			_, err := storage.GetSaga(ctx, sagaID)
			if err == nil {
				count++
			}
		}
	}

	expected := numTransactions * 5
	if count != expected {
		t.Errorf("Expected %d sagas, got %d", expected, count)
	}
}

// ==========================
// Optimistic Locking Tests
// ==========================

// TestPostgresIntegration_OptimisticLocking tests optimistic locking with version.
func TestPostgresIntegration_OptimisticLocking(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()

	// Create and save a saga
	saga1 := createTestSagaInstance("saga-lock-1", "def-lock", saga.StateRunning)
	if err := storage.SaveSaga(ctx, saga1); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Try to update with correct version (should succeed)
	if err := storage.UpdateSagaStateWithOptimisticLock(ctx, "saga-lock-1", saga.StateCompleted, 1, nil); err != nil {
		t.Fatalf("Failed to update with correct version: %v", err)
	}

	// Try to update with wrong version (should fail)
	err = storage.UpdateSagaStateWithOptimisticLock(ctx, "saga-lock-1", saga.StateFailed, 1, nil)
	if err == nil {
		t.Error("Expected error when updating with wrong version")
	}
	if err != ErrOptimisticLockFailed {
		t.Errorf("Expected ErrOptimisticLockFailed, got %v", err)
	}

	// Verify state is still Completed
	updated, err := storage.GetSaga(ctx, "saga-lock-1")
	if err != nil {
		t.Fatalf("Failed to get updated saga: %v", err)
	}
	if updated.GetState() != saga.StateCompleted {
		t.Errorf("State should be Completed, got %v", updated.GetState())
	}
}

// ==========================
// Timeout and Cleanup Tests
// ==========================

// TestPostgresIntegration_TimeoutDetection tests timeout detection functionality.
func TestPostgresIntegration_TimeoutDetection(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()
	now := time.Now()

	// Create sagas with different start times and timeouts
	sagas := []struct {
		id        string
		startTime time.Time
		timeout   time.Duration
	}{
		{"saga-timeout-1", now.Add(-2 * time.Hour), 1 * time.Hour},    // Should timeout
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
		for _, s := range timedOut {
			t.Logf("Timed out saga: %s", s.GetID())
		}
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

// ==========================
// Health Check and Monitoring Tests
// ==========================

// TestPostgresIntegration_HealthCheck tests health check functionality.
func TestPostgresIntegration_HealthCheck(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
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

// TestPostgresIntegration_PoolStats tests getting connection pool statistics.
func TestPostgresIntegration_PoolStats(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Get pool stats
	stats, err := storage.GetPoolStats()
	if err != nil {
		t.Fatalf("Failed to get pool stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Pool stats should not be nil")
	}

	t.Logf("Pool stats: MaxOpenConnections=%d, OpenConnections=%d, InUse=%d, Idle=%d",
		stats.MaxOpenConnections, stats.OpenConnections, stats.InUse, stats.Idle)
}

// ==========================
// Data Consistency Tests
// ==========================

// TestPostgresIntegration_DataConsistency tests data consistency across operations.
func TestPostgresIntegration_DataConsistency(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

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

// ==========================
// Database Migration Tests
// ==========================

// TestPostgresIntegration_SchemaCreation tests that schema is created correctly.
func TestPostgresIntegration_SchemaCreation(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	config.AutoMigrate = true

	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage with auto-migrate: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()

	// Verify tables exist by querying them
	tables := []string{
		storage.instancesTable,
		storage.stepsTable,
		storage.eventsTable,
	}

	for _, table := range tables {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		var count int
		err := storage.db.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query table %s: %v", table, err)
		}
	}
}

// ==========================
// Performance Benchmark Tests
// ==========================

// BenchmarkPostgresIntegration_SaveSaga benchmarks SaveSaga operation.
func BenchmarkPostgresIntegration_SaveSaga(b *testing.B) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		b.Skip("Skipping benchmark. Set POSTGRES_TEST_DSN to enable.")
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer func() {
		// Cleanup - convert B to T-like for cleanup function
		t := &testing.T{}
		cleanupTestPostgresData(t, storage)
	}()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-%d", i)
		saga := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, saga); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}
}

// BenchmarkPostgresIntegration_GetSaga benchmarks GetSaga operation.
func BenchmarkPostgresIntegration_GetSaga(b *testing.B) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		b.Skip("Skipping benchmark. Set POSTGRES_TEST_DSN to enable.")
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer func() {
		t := &testing.T{}
		cleanupTestPostgresData(t, storage)
	}()

	ctx := context.Background()

	// Pre-populate with sagas
	for i := 0; i < 100; i++ {
		sagaID := fmt.Sprintf("saga-bench-%d", i)
		saga := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, saga); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-%d", i%100)
		_, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			b.Fatalf("Failed to get saga: %v", err)
		}
	}
}

// BenchmarkPostgresIntegration_Transaction benchmarks transaction operations.
func BenchmarkPostgresIntegration_Transaction(b *testing.B) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		b.Skip("Skipping benchmark. Set POSTGRES_TEST_DSN to enable.")
	}

	config := getTestPostgresConfig()
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer func() {
		t := &testing.T{}
		cleanupTestPostgresData(t, storage)
	}()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := storage.BeginTransaction(ctx, 30*time.Second)
		if err != nil {
			b.Fatalf("Failed to begin transaction: %v", err)
		}

		sagaID := fmt.Sprintf("saga-tx-bench-%d", i)
		saga := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := tx.SaveSaga(ctx, saga); err != nil {
			b.Fatalf("Failed to save saga in transaction: %v", err)
		}

		if err := tx.Commit(ctx); err != nil {
			b.Fatalf("Failed to commit transaction: %v", err)
		}
	}
}

// ==========================
// Error Handling Tests
// ==========================

// TestPostgresIntegration_ConnectionFailure tests behavior when connection fails.
func TestPostgresIntegration_ConnectionFailure(t *testing.T) {
	// This test intentionally uses an invalid DSN
	config := &PostgresConfig{
		DSN:               "host=invalid-host port=5432 user=postgres password=postgres dbname=test sslmode=disable",
		ConnectionTimeout: 1 * time.Second,
		AutoMigrate:       false,
	}

	// Creation should fail
	_, err := NewPostgresStateStorage(config)
	if err == nil {
		t.Error("Expected error when creating storage with invalid DSN, got nil")
	}
}

// TestPostgresIntegration_QueryTimeout tests query timeout handling.
func TestPostgresIntegration_QueryTimeout(t *testing.T) {
	if !testPostgresAvailable(t) {
		return
	}

	config := getTestPostgresConfig()
	config.QueryTimeout = 1 * time.Nanosecond // Very short timeout
	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestPostgresData(t, storage)

	ctx := context.Background()

	// This should likely timeout or complete very quickly
	saga1 := createTestSagaInstance("saga-timeout-test", "def-test", saga.StateRunning)
	err = storage.SaveSaga(ctx, saga1)
	// We expect either success (if it's fast enough) or timeout error
	if err != nil {
		t.Logf("Got expected error due to short timeout: %v", err)
	}
}

// ==========================
// Helper Functions (if needed - can reuse from test_helpers.go)
// ==========================
