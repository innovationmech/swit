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
)

// BenchmarkRedisStorage_SaveSaga benchmarks SaveSaga operation
func BenchmarkRedisStorage_SaveSaga(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-save-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}
}

// BenchmarkRedisStorage_GetSaga benchmarks GetSaga operation
func BenchmarkRedisStorage_GetSaga(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with sagas
	const numSagas = 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-get-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-get-%d", i%numSagas)
		_, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			b.Fatalf("Failed to get saga: %v", err)
		}
	}
}

// BenchmarkRedisStorage_UpdateSagaState benchmarks UpdateSagaState operation
func BenchmarkRedisStorage_UpdateSagaState(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with sagas
	const numSagas = 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-update-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-update-%d", i%numSagas)
		err := storage.UpdateSagaState(ctx, sagaID, saga.StateStepCompleted, nil)
		if err != nil {
			b.Fatalf("Failed to update saga state: %v", err)
		}
	}
}

// BenchmarkRedisStorage_DeleteSaga benchmarks DeleteSaga operation
func BenchmarkRedisStorage_DeleteSaga(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with sagas
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-delete-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateCompleted)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-delete-%d", i)
		if err := storage.DeleteSaga(ctx, sagaID); err != nil {
			b.Fatalf("Failed to delete saga: %v", err)
		}
	}
}

// BenchmarkRedisStorage_GetActiveSagas benchmarks GetActiveSagas operation
func BenchmarkRedisStorage_GetActiveSagas(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with active sagas
	const numSagas = 100
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-active-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetActiveSagas(ctx, nil)
		if err != nil {
			b.Fatalf("Failed to get active sagas: %v", err)
		}
	}
}

// BenchmarkRedisStorage_GetActiveSagas_WithFilter benchmarks GetActiveSagas with filter
func BenchmarkRedisStorage_GetActiveSagas_WithFilter(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with sagas
	const numSagas = 100
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-filter-%d", i)
		defID := fmt.Sprintf("def-%d", i%3) // 3 different definition IDs
		sagaInst := createTestSagaInstance(sagaID, defID, saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	filter := &saga.SagaFilter{
		DefinitionIDs: []string{"def-0"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetActiveSagas(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to get filtered sagas: %v", err)
		}
	}
}

// BenchmarkRedisStorage_GetTimeoutSagas benchmarks GetTimeoutSagas operation
func BenchmarkRedisStorage_GetTimeoutSagas(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()
	now := time.Now()

	// Pre-populate with sagas that should timeout
	const numSagas = 50
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-timeout-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		sagaInst.startTime = now.Add(-2 * time.Hour)
		sagaInst.timeout = 1 * time.Hour
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetTimeoutSagas(ctx, now)
		if err != nil {
			b.Fatalf("Failed to get timeout sagas: %v", err)
		}
	}
}

// BenchmarkRedisStorage_SaveStepState benchmarks SaveStepState operation
func BenchmarkRedisStorage_SaveStepState(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Create a saga first
	sagaInstance := createTestSagaInstance("saga-bench-step", "def-bench", saga.StateRunning)
	if err := storage.SaveSaga(ctx, sagaInstance); err != nil {
		b.Fatalf("Failed to save saga: %v", err)
	}

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step := &saga.StepState{
			ID:        fmt.Sprintf("step-%d", i),
			Name:      "BenchStep",
			State:     saga.StepStateCompleted,
			Attempts:  1,
			CreatedAt: now,
			StartedAt: &now,
		}
		if err := storage.SaveStepState(ctx, "saga-bench-step", step); err != nil {
			b.Fatalf("Failed to save step state: %v", err)
		}
	}
}

// BenchmarkRedisStorage_GetStepStates benchmarks GetStepStates operation
func BenchmarkRedisStorage_GetStepStates(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Create a saga first
	sagaInstance := createTestSagaInstance("saga-bench-getsteps", "def-bench", saga.StateRunning)
	if err := storage.SaveSaga(ctx, sagaInstance); err != nil {
		b.Fatalf("Failed to save saga: %v", err)
	}

	// Add steps
	now := time.Now()
	for i := 0; i < 10; i++ {
		step := &saga.StepState{
			ID:        fmt.Sprintf("step-%d", i),
			Name:      "BenchStep",
			State:     saga.StepStateCompleted,
			Attempts:  1,
			CreatedAt: now,
			StartedAt: &now,
		}
		if err := storage.SaveStepState(ctx, "saga-bench-getsteps", step); err != nil {
			b.Fatalf("Failed to save step state: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetStepStates(ctx, "saga-bench-getsteps")
		if err != nil {
			b.Fatalf("Failed to get step states: %v", err)
		}
	}
}

// BenchmarkRedisStorage_CleanupExpiredSagas benchmarks CleanupExpiredSagas operation
func BenchmarkRedisStorage_CleanupExpiredSagas(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Pre-populate with expired sagas
		const numSagas = 10
		for j := 0; j < numSagas; j++ {
			sagaID := fmt.Sprintf("saga-bench-cleanup-%d-%d", i, j)
			sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateCompleted)
			sagaInst.updatedAt = now.Add(-2 * time.Hour)
			if err := storage.SaveSaga(ctx, sagaInst); err != nil {
				b.Fatalf("Failed to save saga: %v", err)
			}
		}
		b.StartTimer()

		olderThan := now.Add(-1 * time.Hour)
		if err := storage.CleanupExpiredSagas(ctx, olderThan); err != nil {
			b.Fatalf("Failed to cleanup expired sagas: %v", err)
		}
	}
}

// BenchmarkRedisStorage_HealthCheck benchmarks HealthCheck operation
func BenchmarkRedisStorage_HealthCheck(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := storage.HealthCheck(ctx); err != nil {
			b.Fatalf("HealthCheck failed: %v", err)
		}
	}
}

// BenchmarkRedisStorage_ConcurrentSaves benchmarks concurrent save operations
func BenchmarkRedisStorage_ConcurrentSaves(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-bench-concurrent-%d", i)
			sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
			if err := storage.SaveSaga(ctx, sagaInst); err != nil {
				b.Fatalf("Failed to save saga: %v", err)
			}
			i++
		}
	})
}

// BenchmarkRedisStorage_ConcurrentReads benchmarks concurrent read operations
func BenchmarkRedisStorage_ConcurrentReads(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with sagas
	const numSagas = 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-read-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-bench-read-%d", i%numSagas)
			_, err := storage.GetSaga(ctx, sagaID)
			if err != nil {
				b.Fatalf("Failed to get saga: %v", err)
			}
			i++
		}
	})
}

// BenchmarkRedisStorage_ConcurrentMixed benchmarks mixed concurrent operations
func BenchmarkRedisStorage_ConcurrentMixed(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Pre-populate with sagas
	const numSagas = 100
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-mixed-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 3 {
			case 0: // Read
				sagaID := fmt.Sprintf("saga-bench-mixed-%d", i%numSagas)
				storage.GetSaga(ctx, sagaID)
			case 1: // Update
				sagaID := fmt.Sprintf("saga-bench-mixed-%d", i%numSagas)
				storage.UpdateSagaState(ctx, sagaID, saga.StateStepCompleted, nil)
			case 2: // Save new
				sagaID := fmt.Sprintf("saga-bench-mixed-new-%d", i)
				sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
				storage.SaveSaga(ctx, sagaInst)
			}
			i++
		}
	})
}

// BenchmarkRedisStorage_LargePayload benchmarks operations with large payloads
func BenchmarkRedisStorage_LargePayload(b *testing.B) {
	if !testRedisAvailable(&testing.T{}) {
		b.Skip("Redis is not available for benchmarking")
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(&testing.T{}, storage)

	ctx := context.Background()

	// Create large metadata
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d-with-some-additional-data", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-large-%d", i)
		sagaInst := createTestSagaInstance(sagaID, "def-bench", saga.StateRunning)
		sagaInst.metadata = largeData
		if err := storage.SaveSaga(ctx, sagaInst); err != nil {
			b.Fatalf("Failed to save large saga: %v", err)
		}
	}
}

// BenchmarkRedisStorage_Serialization benchmarks serialization performance
func BenchmarkRedisStorage_Serialization(b *testing.B) {
	serializerOpts := &SerializationOptions{
		EnableCompression: true,
		CompressionLevel:  -1,
		PrettyPrint:       false,
	}
	serializer := NewSagaSerializer(serializerOpts)

	sagaInst := createTestSagaInstance("saga-bench-ser", "def-bench", saga.StateRunning)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.SerializeSagaInstance(sagaInst)
		if err != nil {
			b.Fatalf("Failed to serialize: %v", err)
		}
	}
}

// BenchmarkRedisStorage_Deserialization benchmarks deserialization performance
func BenchmarkRedisStorage_Deserialization(b *testing.B) {
	serializerOpts := &SerializationOptions{
		EnableCompression: true,
		CompressionLevel:  -1,
		PrettyPrint:       false,
	}
	serializer := NewSagaSerializer(serializerOpts)

	sagaInst := createTestSagaInstance("saga-bench-deser", "def-bench", saga.StateRunning)
	data, err := serializer.SerializeSagaInstance(sagaInst)
	if err != nil {
		b.Fatalf("Failed to serialize: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.DeserializeSagaInstance(data)
		if err != nil {
			b.Fatalf("Failed to deserialize: %v", err)
		}
	}
}
