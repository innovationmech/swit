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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// BenchmarkMemoryStateStorage_SaveSaga benchmarks the SaveSaga operation.
func BenchmarkMemoryStateStorage_SaveSaga(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	instance := &mockSagaInstance{
		id:           "bench-saga",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		currentStep:  1,
		totalSteps:   5,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		timeout:      5 * time.Minute,
		metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instance.id = fmt.Sprintf("bench-saga-%d", i)
		_ = storage.SaveSaga(ctx, instance)
	}
}

// BenchmarkMemoryStateStorage_GetSaga benchmarks the GetSaga operation.
func BenchmarkMemoryStateStorage_GetSaga(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 10000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-%d", i%numSagas)
		_, _ = storage.GetSaga(ctx, sagaID)
	}
}

// BenchmarkMemoryStateStorage_UpdateSagaState benchmarks the UpdateSagaState operation.
func BenchmarkMemoryStateStorage_UpdateSagaState(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	metadata := map[string]interface{}{
		"updated":   true,
		"timestamp": time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-%d", i%numSagas)
		_ = storage.UpdateSagaState(ctx, sagaID, saga.StateCompleted, metadata)
	}
}

// BenchmarkMemoryStateStorage_DeleteSaga benchmarks the DeleteSaga operation.
func BenchmarkMemoryStateStorage_DeleteSaga(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		storage := NewMemoryStateStorage()
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateCompleted,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
		b.StartTimer()

		_ = storage.DeleteSaga(ctx, sagaID)
	}
}

// BenchmarkMemoryStateStorage_GetActiveSagas benchmarks the GetActiveSagas operation with filters.
func BenchmarkMemoryStateStorage_GetActiveSagas(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 5000
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
		_ = storage.SaveSaga(ctx, instance)
	}

	filter := &saga.SagaFilter{
		States: []saga.SagaState{saga.StateRunning},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetActiveSagas(ctx, filter)
	}
}

// BenchmarkMemoryStateStorage_GetActiveSagas_NoFilter benchmarks GetActiveSagas without filter.
func BenchmarkMemoryStateStorage_GetActiveSagas_NoFilter(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 5000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetActiveSagas(ctx, nil)
	}
}

// BenchmarkMemoryStateStorage_GetTimeoutSagas benchmarks the GetTimeoutSagas operation.
func BenchmarkMemoryStateStorage_GetTimeoutSagas(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-10 * time.Minute)

	// Pre-populate with sagas
	numSagas := 5000
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
		_ = storage.SaveSaga(ctx, instance)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetTimeoutSagas(ctx, now)
	}
}

// BenchmarkMemoryStateStorage_CleanupExpiredSagas benchmarks the CleanupExpiredSagas operation.
func BenchmarkMemoryStateStorage_CleanupExpiredSagas(b *testing.B) {
	ctx := context.Background()
	now := time.Now()
	old := now.Add(-2 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		storage := NewMemoryStateStorage()

		// Pre-populate with sagas
		numSagas := 1000
		for j := 0; j < numSagas; j++ {
			sagaID := fmt.Sprintf("saga-%d", j)
			state := saga.StateCompleted
			if j%2 == 0 {
				state = saga.StateRunning
			}
			instance := &mockSagaInstance{
				id:           sagaID,
				definitionID: "def-1",
				sagaState:    state,
				createdAt:    old,
				updatedAt:    old,
			}
			_ = storage.SaveSaga(ctx, instance)
		}
		b.StartTimer()

		cutoff := now.Add(-1 * time.Hour)
		_ = storage.CleanupExpiredSagas(ctx, cutoff)
	}
}

// BenchmarkMemoryStateStorage_SaveStepState benchmarks the SaveStepState operation.
func BenchmarkMemoryStateStorage_SaveStepState(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 100
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-%d", i%numSagas)
		step := &saga.StepState{
			ID:        fmt.Sprintf("%s-step-%d", sagaID, i),
			SagaID:    sagaID,
			StepIndex: i % 10,
			Name:      fmt.Sprintf("Step %d", i%10),
			State:     saga.StepStateRunning,
		}
		_ = storage.SaveStepState(ctx, sagaID, step)
	}
}

// BenchmarkMemoryStateStorage_GetStepStates benchmarks the GetStepStates operation.
func BenchmarkMemoryStateStorage_GetStepStates(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas and steps
	numSagas := 100
	numStepsPerSaga := 50
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)

		for j := 0; j < numStepsPerSaga; j++ {
			step := &saga.StepState{
				ID:        fmt.Sprintf("%s-step-%d", sagaID, j),
				SagaID:    sagaID,
				StepIndex: j,
				State:     saga.StepStateCompleted,
			}
			_ = storage.SaveStepState(ctx, sagaID, step)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-%d", i%numSagas)
		_, _ = storage.GetStepStates(ctx, sagaID)
	}
}

// BenchmarkMemoryStateStorage_ParallelSave benchmarks parallel SaveSaga operations.
func BenchmarkMemoryStateStorage_ParallelSave(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-%d", i)
			instance := &mockSagaInstance{
				id:           sagaID,
				definitionID: "def-1",
				sagaState:    saga.StateRunning,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
			}
			_ = storage.SaveSaga(ctx, instance)
			i++
		}
	})
}

// BenchmarkMemoryStateStorage_ParallelGet benchmarks parallel GetSaga operations.
func BenchmarkMemoryStateStorage_ParallelGet(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 10000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-%d", i%numSagas)
			_, _ = storage.GetSaga(ctx, sagaID)
			i++
		}
	})
}

// BenchmarkMemoryStateStorage_ParallelUpdate benchmarks parallel UpdateSagaState operations.
func BenchmarkMemoryStateStorage_ParallelUpdate(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	metadata := map[string]interface{}{
		"updated": true,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-%d", i%numSagas)
			_ = storage.UpdateSagaState(ctx, sagaID, saga.StateCompleted, metadata)
			i++
		}
	})
}

// BenchmarkMemoryStateStorage_ParallelMixed benchmarks parallel mixed operations.
func BenchmarkMemoryStateStorage_ParallelMixed(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with some sagas
	numSagas := 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-%d", i%numSagas)

			// Mix of operations (roughly 40% read, 40% write, 20% update)
			op := i % 5
			switch op {
			case 0, 1: // Read
				_, _ = storage.GetSaga(ctx, sagaID)
			case 2, 3: // Write
				instance := &mockSagaInstance{
					id:           sagaID,
					definitionID: "def-1",
					sagaState:    saga.StateRunning,
					createdAt:    time.Now(),
					updatedAt:    time.Now(),
				}
				_ = storage.SaveSaga(ctx, instance)
			case 4: // Update
				_ = storage.UpdateSagaState(ctx, sagaID, saga.StateCompleted, nil)
			}
			i++
		}
	})
}

// BenchmarkMemoryStateStorage_WithDifferentCapacities benchmarks performance with different initial capacities.
func BenchmarkMemoryStateStorage_WithDifferentCapacities(b *testing.B) {
	capacities := []int{10, 100, 1000, 10000}

	for _, capacity := range capacities {
		b.Run(fmt.Sprintf("Capacity-%d", capacity), func(b *testing.B) {
			config := &state.MemoryStorageConfig{
				InitialCapacity: capacity,
				MaxCapacity:     0,
				EnableMetrics:   true,
			}
			storage := NewMemoryStateStorageWithConfig(config)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sagaID := fmt.Sprintf("saga-%d", i)
				instance := &mockSagaInstance{
					id:           sagaID,
					definitionID: "def-1",
					sagaState:    saga.StateRunning,
					createdAt:    time.Now(),
					updatedAt:    time.Now(),
				}
				_ = storage.SaveSaga(ctx, instance)
			}
		})
	}
}

// BenchmarkMemoryStateStorage_LargeMetadata benchmarks performance with large metadata.
func BenchmarkMemoryStateStorage_LargeMetadata(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create large metadata
	largeMetadata := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeMetadata[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
			metadata:     largeMetadata,
		}
		_ = storage.SaveSaga(ctx, instance)
	}
}

// BenchmarkMemoryStateStorage_ManySteps benchmarks performance with many steps.
func BenchmarkMemoryStateStorage_ManySteps(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	numStepsPerSaga := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		_ = storage.SaveSaga(ctx, instance)
		b.StartTimer()

		for j := 0; j < numStepsPerSaga; j++ {
			step := &saga.StepState{
				ID:        fmt.Sprintf("%s-step-%d", sagaID, j),
				SagaID:    sagaID,
				StepIndex: j,
				State:     saga.StepStateRunning,
			}
			_ = storage.SaveStepState(ctx, sagaID, step)
		}
	}
}

// BenchmarkMemoryStateStorage_FilterComplexity benchmarks different filter complexities.
func BenchmarkMemoryStateStorage_FilterComplexity(b *testing.B) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 5000
	now := time.Now()
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		instance := &mockSagaInstance{
			id:           sagaID,
			definitionID: fmt.Sprintf("def-%d", i%5),
			sagaState:    saga.StateRunning,
			createdAt:    now.Add(-time.Duration(i) * time.Minute),
			updatedAt:    now,
			metadata: map[string]interface{}{
				"env":      fmt.Sprintf("env-%d", i%3),
				"priority": i % 5,
			},
		}
		_ = storage.SaveSaga(ctx, instance)
	}

	tests := []struct {
		name   string
		filter *saga.SagaFilter
	}{
		{
			name:   "NoFilter",
			filter: nil,
		},
		{
			name: "StateOnly",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning},
			},
		},
		{
			name: "DefinitionOnly",
			filter: &saga.SagaFilter{
				DefinitionIDs: []string{"def-0", "def-1"},
			},
		},
		{
			name: "MetadataOnly",
			filter: &saga.SagaFilter{
				Metadata: map[string]interface{}{
					"env": "env-0",
				},
			},
		},
		{
			name: "Combined",
			filter: &saga.SagaFilter{
				States:        []saga.SagaState{saga.StateRunning},
				DefinitionIDs: []string{"def-0"},
				Metadata: map[string]interface{}{
					"env": "env-0",
				},
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = storage.GetActiveSagas(ctx, tt.filter)
			}
		})
	}
}
