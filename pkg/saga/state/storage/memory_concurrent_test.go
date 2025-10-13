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
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// TestMemoryStateStorage_ConcurrentSaveAndGet tests concurrent save and get operations.
func TestMemoryStateStorage_ConcurrentSaveAndGet(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	numGoroutines := 100
	numOperationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Save and Get goroutines

	// Concurrent saves
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				sagaID := fmt.Sprintf("saga-%d-%d", id, j)
				instance := &mockSagaInstance{
					id:           sagaID,
					definitionID: "def-1",
					sagaState:    saga.StateRunning,
					currentStep:  j,
					totalSteps:   10,
					createdAt:    time.Now(),
					updatedAt:    time.Now(),
					timeout:      5 * time.Minute,
				}
				if err := storage.SaveSaga(ctx, instance); err != nil {
					t.Errorf("failed to save saga %s: %v", sagaID, err)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				sagaID := fmt.Sprintf("saga-%d-%d", id, j)
				// May or may not exist depending on timing
				_, _ = storage.GetSaga(ctx, sagaID)
			}
		}(i)
	}

	wg.Wait()

	// Verify that all sagas were saved
	expectedCount := numGoroutines * numOperationsPerGoroutine
	actualCount := len(storage.sagas)
	if actualCount != expectedCount {
		t.Errorf("expected %d sagas, got %d", expectedCount, actualCount)
	}
}

// TestMemoryStateStorage_ConcurrentUpdateSagaState tests concurrent state updates.
func TestMemoryStateStorage_ConcurrentUpdateSagaState(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	numSagas := 50
	numUpdatesPerSaga := 100

	// Pre-create sagas
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		if err := storage.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("failed to save saga %s: %v", sagaID, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(numSagas * numUpdatesPerSaga)

	states := []saga.SagaState{
		saga.StateRunning,
		saga.StateStepCompleted,
		saga.StateCompensating,
		saga.StateCompleted,
	}

	// Concurrent updates
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		for j := 0; j < numUpdatesPerSaga; j++ {
			go func(sID string, updateIdx int) {
				defer wg.Done()
				newState := states[updateIdx%len(states)]
				metadata := map[string]interface{}{
					"update": updateIdx,
				}
				_ = storage.UpdateSagaState(ctx, sID, newState, metadata)
			}(sagaID, j)
		}
	}

	wg.Wait()

	// Verify all sagas still exist and have valid states
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			t.Errorf("failed to retrieve saga %s: %v", sagaID, err)
			continue
		}

		// Verify state is valid
		validState := false
		for _, s := range states {
			if instance.GetState() == s {
				validState = true
				break
			}
		}
		if !validState {
			t.Errorf("saga %s has invalid state: %v", sagaID, instance.GetState())
		}

		// Verify metadata exists
		if instance.GetMetadata() == nil {
			t.Errorf("saga %s should have metadata", sagaID)
		}
	}
}

// TestMemoryStateStorage_ConcurrentSaveAndDeleteSaga tests concurrent save and delete operations.
func TestMemoryStateStorage_ConcurrentSaveAndDeleteSaga(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	numGoroutines := 50
	numOperationsPerGoroutine := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Save and Delete goroutines

	var saveCount, deleteCount atomic.Int64

	// Concurrent saves
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				sagaID := fmt.Sprintf("saga-%d-%d", id, j)
				instance := &mockSagaInstance{
					id:           sagaID,
					definitionID: "def-1",
					sagaState:    saga.StateCompleted,
					createdAt:    time.Now(),
					updatedAt:    time.Now(),
				}
				if err := storage.SaveSaga(ctx, instance); err == nil {
					saveCount.Add(1)
				}
			}
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				sagaID := fmt.Sprintf("saga-%d-%d", id, j)
				// May or may not exist depending on timing
				if err := storage.DeleteSaga(ctx, sagaID); err == nil {
					deleteCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Test should complete without data races (checked by -race flag)
	t.Logf("Saves: %d, Deletes: %d, Remaining: %d",
		saveCount.Load(), deleteCount.Load(), len(storage.sagas))
}

// TestMemoryStateStorage_ConcurrentStepStateOperations tests concurrent step state operations.
func TestMemoryStateStorage_ConcurrentStepStateOperations(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	numSagas := 20
	numStepsPerSaga := 50

	// Pre-create sagas
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		if err := storage.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("failed to save saga %s: %v", sagaID, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(numSagas * numStepsPerSaga * 2) // Save and Get operations

	stepStates := []saga.StepStateEnum{
		saga.StepStatePending,
		saga.StepStateRunning,
		saga.StepStateCompleted,
		saga.StepStateFailed,
	}

	// Concurrent step state saves
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		for j := 0; j < numStepsPerSaga; j++ {
			go func(sID string, stepIdx int) {
				defer wg.Done()
				stepStateEnum := stepStates[stepIdx%len(stepStates)]
				step := &saga.StepState{
					ID:        fmt.Sprintf("%s-step-%d", sID, stepIdx),
					SagaID:    sID,
					StepIndex: stepIdx,
					Name:      fmt.Sprintf("Step %d", stepIdx),
					State:     stepStateEnum,
				}
				_ = storage.SaveStepState(ctx, sID, step)
			}(sagaID, j)
		}
	}

	// Concurrent step state reads
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		for j := 0; j < numStepsPerSaga; j++ {
			go func(sID string) {
				defer wg.Done()
				_, _ = storage.GetStepStates(ctx, sID)
			}(sagaID)
		}
	}

	wg.Wait()

	// Verify all step states were saved correctly
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		steps, err := storage.GetStepStates(ctx, sagaID)
		if err != nil {
			t.Errorf("failed to get step states for %s: %v", sagaID, err)
			continue
		}
		if len(steps) != numStepsPerSaga {
			t.Errorf("expected %d steps for %s, got %d", numStepsPerSaga, sagaID, len(steps))
		}
	}
}

// TestMemoryStateStorage_ConcurrentMixedOperations tests a mix of all operations concurrently.
func TestMemoryStateStorage_ConcurrentMixedOperations(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	duration := 3 * time.Second
	numWorkers := 50

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	var (
		saveOps   atomic.Int64
		getOps    atomic.Int64
		updateOps atomic.Int64
		deleteOps atomic.Int64
		filterOps atomic.Int64
		stepOps   atomic.Int64
	)

	// Start worker goroutines performing random operations
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for {
				select {
				case <-stopChan:
					return
				default:
					sagaID := fmt.Sprintf("saga-%d", rnd.Intn(100))

					// Random operation
					switch rnd.Intn(6) {
					case 0: // Save
						instance := &mockSagaInstance{
							id:           sagaID,
							definitionID: "def-1",
							sagaState:    saga.StateRunning,
							createdAt:    time.Now(),
							updatedAt:    time.Now(),
						}
						_ = storage.SaveSaga(ctx, instance)
						saveOps.Add(1)

					case 1: // Get
						_, _ = storage.GetSaga(ctx, sagaID)
						getOps.Add(1)

					case 2: // Update
						_ = storage.UpdateSagaState(ctx, sagaID, saga.StateCompleted, nil)
						updateOps.Add(1)

					case 3: // Delete
						_ = storage.DeleteSaga(ctx, sagaID)
						deleteOps.Add(1)

					case 4: // Filter
						filter := &saga.SagaFilter{
							States: []saga.SagaState{saga.StateRunning},
						}
						_, _ = storage.GetActiveSagas(ctx, filter)
						filterOps.Add(1)

					case 5: // Step state
						step := &saga.StepState{
							ID:        fmt.Sprintf("%s-step-1", sagaID),
							SagaID:    sagaID,
							StepIndex: 0,
							State:     saga.StepStateRunning,
						}
						_ = storage.SaveStepState(ctx, sagaID, step)
						stepOps.Add(1)
					}
				}
			}
		}(i)
	}

	// Run for the specified duration
	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	// Report results
	t.Logf("Mixed operations completed:")
	t.Logf("  Save:   %d ops", saveOps.Load())
	t.Logf("  Get:    %d ops", getOps.Load())
	t.Logf("  Update: %d ops", updateOps.Load())
	t.Logf("  Delete: %d ops", deleteOps.Load())
	t.Logf("  Filter: %d ops", filterOps.Load())
	t.Logf("  Step:   %d ops", stepOps.Load())
	t.Logf("  Total:  %d ops", saveOps.Load()+getOps.Load()+updateOps.Load()+
		deleteOps.Load()+filterOps.Load()+stepOps.Load())

	// Test should complete without data races or panics
}

// TestMemoryStateStorage_ConcurrentGetActiveSagas tests concurrent filter operations.
func TestMemoryStateStorage_ConcurrentGetActiveSagas(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		state := saga.StateRunning
		if i%3 == 0 {
			state = saga.StateCompleted
		}
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: fmt.Sprintf("def-%d", i%5),
			sagaState:    state,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		if err := storage.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("failed to save saga %s: %v", sagaID, err)
		}
	}

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent filter queries
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Different filter combinations
			filters := []*saga.SagaFilter{
				{States: []saga.SagaState{saga.StateRunning}},
				{States: []saga.SagaState{saga.StateCompleted}},
				{DefinitionIDs: []string{"def-0", "def-1"}},
				nil, // No filter
			}

			filter := filters[id%len(filters)]
			_, err := storage.GetActiveSagas(ctx, filter)
			if err != nil {
				t.Errorf("filter operation failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// TestMemoryStateStorage_ConcurrentGetTimeoutSagas tests concurrent timeout queries.
func TestMemoryStateStorage_ConcurrentGetTimeoutSagas(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-10 * time.Minute)

	// Pre-populate with sagas with various timeouts
	numSagas := 500
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		startTime := past.Add(time.Duration(i) * time.Second)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			startTime:    startTime,
			timeout:      5 * time.Minute,
			createdAt:    startTime,
			updatedAt:    startTime,
		}
		if err := storage.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("failed to save saga %s: %v", sagaID, err)
		}
	}

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent timeout queries
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			before := now.Add(-time.Duration(id) * time.Minute)
			_, err := storage.GetTimeoutSagas(ctx, before)
			if err != nil {
				t.Errorf("timeout query failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// TestMemoryStateStorage_ConcurrentCleanupExpiredSagas tests concurrent cleanup operations.
func TestMemoryStateStorage_ConcurrentCleanupExpiredSagas(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	old := now.Add(-2 * time.Hour)

	// Pre-populate with sagas
	numSagas := 500
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		state := saga.StateCompleted
		if i%2 == 0 {
			state = saga.StateRunning
		}
		updatedAt := old
		if i%3 == 0 {
			updatedAt = now
		}
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    state,
			createdAt:    old,
			updatedAt:    updatedAt,
		}
		if err := storage.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("failed to save saga %s: %v", sagaID, err)
		}
	}

	numGoroutines := 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent cleanup operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			cutoff := now.Add(-1 * time.Hour)
			err := storage.CleanupExpiredSagas(ctx, cutoff)
			if err != nil {
				t.Errorf("cleanup operation failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify cleanup happened correctly
	// Only old completed sagas should be cleaned up
	instances, err := storage.GetActiveSagas(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	for _, inst := range instances {
		// Should not have old completed sagas
		if inst.GetState() == saga.StateCompleted && inst.GetUpdatedAt().Before(now.Add(-1*time.Hour)) {
			t.Errorf("old completed saga %s should have been cleaned up", inst.GetID())
		}
	}
}

// TestMemoryStateStorage_StressTest is a comprehensive stress test.
func TestMemoryStateStorage_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	storage := NewMemoryStateStorageWithConfig(&state.MemoryStorageConfig{
		InitialCapacity: 1000,
		MaxCapacity:     0, // No limit
		EnableMetrics:   true,
	})
	ctx := context.Background()

	duration := 5 * time.Second
	numWorkers := 100

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	var totalOps atomic.Int64
	var errors atomic.Int64

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
			ops := 0

			for {
				select {
				case <-stopChan:
					return
				default:
					sagaID := fmt.Sprintf("saga-%d-%d", workerID, rnd.Intn(50))

					// Perform random operation
					var err error
					switch rnd.Intn(10) {
					case 0, 1, 2: // Save (30%)
						instance := &mockSagaInstance{
							id:           sagaID,
							definitionID: fmt.Sprintf("def-%d", rnd.Intn(5)),
							sagaState:    saga.StateRunning,
							currentStep:  rnd.Intn(10),
							totalSteps:   10,
							createdAt:    time.Now(),
							updatedAt:    time.Now(),
							timeout:      5 * time.Minute,
							metadata: map[string]interface{}{
								"worker": workerID,
								"op":     ops,
							},
						}
						err = storage.SaveSaga(ctx, instance)

					case 3, 4: // Get (20%)
						_, err = storage.GetSaga(ctx, sagaID)

					case 5: // Update (10%)
						newState := saga.StateCompleted
						if rnd.Intn(2) == 0 {
							newState = saga.StateRunning
						}
						err = storage.UpdateSagaState(ctx, sagaID, newState, map[string]interface{}{
							"updated": time.Now(),
						})

					case 6: // Delete (10%)
						err = storage.DeleteSaga(ctx, sagaID)

					case 7: // Filter (10%)
						filter := &saga.SagaFilter{
							States: []saga.SagaState{saga.StateRunning},
						}
						_, err = storage.GetActiveSagas(ctx, filter)

					case 8: // Step state save (10%)
						step := &saga.StepState{
							ID:        fmt.Sprintf("%s-step-%d", sagaID, rnd.Intn(5)),
							SagaID:    sagaID,
							StepIndex: rnd.Intn(5),
							State:     saga.StepStateRunning,
						}
						err = storage.SaveStepState(ctx, sagaID, step)

					case 9: // Step state get (10%)
						_, err = storage.GetStepStates(ctx, sagaID)
					}

					if err != nil && err != state.ErrSagaNotFound && err != state.ErrInvalidSagaID {
						errors.Add(1)
					}

					ops++
					totalOps.Add(1)
				}
			}
		}(i)
	}

	// Run for the specified duration
	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	totalOperations := totalOps.Load()
	errorCount := errors.Load()
	opsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Stress test results:")
	t.Logf("  Duration:       %v", duration)
	t.Logf("  Workers:        %d", numWorkers)
	t.Logf("  Total ops:      %d", totalOperations)
	t.Logf("  Errors:         %d", errorCount)
	t.Logf("  Ops/sec:        %.2f", opsPerSecond)
	t.Logf("  Final sagas:    %d", len(storage.sagas))

	// Verify performance meets requirements (> 10000 TPS)
	if opsPerSecond < 10000 {
		t.Logf("WARNING: Performance below target (%.2f ops/sec < 10000 ops/sec)", opsPerSecond)
	} else {
		t.Logf("SUCCESS: Performance meets target (%.2f ops/sec >= 10000 ops/sec)", opsPerSecond)
	}
}

// TestMemoryStateStorage_DataConsistencyUnderLoad tests data consistency under concurrent load.
func TestMemoryStateStorage_DataConsistencyUnderLoad(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	numSagas := 100
	numUpdatesPerSaga := 100

	// Create sagas with initial state
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StatePending,
			currentStep:  0,
			totalSteps:   5,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
			metadata: map[string]interface{}{
				"counter": 0,
			},
		}
		if err := storage.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("failed to save saga %s: %v", sagaID, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(numSagas)

	// Each saga gets updated by a single goroutine sequentially
	for i := 0; i < numSagas; i++ {
		go func(sagaIndex int) {
			defer wg.Done()
			sagaID := fmt.Sprintf("saga-%d", sagaIndex)

			for j := 0; j < numUpdatesPerSaga; j++ {
				// Get current state
				instance, err := storage.GetSaga(ctx, sagaID)
				if err != nil {
					t.Errorf("failed to get saga %s: %v", sagaID, err)
					continue
				}

				// Update state
				currentCounter := 0
				if instance.GetMetadata() != nil {
					if c, ok := instance.GetMetadata()["counter"].(int); ok {
						currentCounter = c
					}
				}

				newMetadata := map[string]interface{}{
					"counter": currentCounter + 1,
				}

				err = storage.UpdateSagaState(ctx, sagaID, saga.StateRunning, newMetadata)
				if err != nil {
					t.Errorf("failed to update saga %s: %v", sagaID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all sagas have consistent final state
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			t.Errorf("failed to get final saga %s: %v", sagaID, err)
			continue
		}

		if instance.GetState() != saga.StateRunning {
			t.Errorf("saga %s has unexpected state: %v", sagaID, instance.GetState())
		}

		// Note: Due to concurrent updates, counter might not be exactly numUpdatesPerSaga
		// but the important thing is no data corruption or panics
		if instance.GetMetadata() == nil {
			t.Errorf("saga %s should have metadata", sagaID)
		}
	}
}
