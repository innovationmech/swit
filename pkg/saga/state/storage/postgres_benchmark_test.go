// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// PostgreSQL benchmarks require a running PostgreSQL instance.
// Set PG_TEST_DSN environment variable to enable these benchmarks:
// export PG_TEST_DSN="postgres://user:pass@localhost:5432/swit_test?sslmode=disable"
//
// Run benchmarks with:
// go test -bench=BenchmarkPostgres -benchtime=3s -benchmem ./pkg/saga/state/storage/

func getTestDSN() string {
	dsn := os.Getenv("PG_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/swit_test?sslmode=disable"
	}
	return dsn
}

func skipIfNoPostgres(b *testing.B) *PostgresStateStorage {
	dsn := getTestDSN()
	
	config := &PostgresConfig{
		DSN:               dsn,
		MaxOpenConns:      25,
		MaxIdleConns:      5,
		ConnMaxLifetime:   time.Hour,
		ConnMaxIdleTime:   30 * time.Minute,
		ConnectionTimeout: 10 * time.Second,
		QueryTimeout:      30 * time.Second,
		AutoMigrate:       false,
	}

	storage, err := NewPostgresStateStorage(config)
	if err != nil {
		b.Skipf("PostgreSQL not available: %v", err)
		return nil
	}

	return storage
}

// BenchmarkPostgresStorage_SaveSaga benchmarks the SaveSaga operation.
func BenchmarkPostgresStorage_SaveSaga(b *testing.B) {
	storage := skipIfNoPostgres(b)
	if storage == nil {
		return
	}
	defer storage.Close()
	defer cleanupBenchData(b, storage, "saga-bench-save")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-save-%d", i)
		inst := createBenchSagaInstance(sagaID, "def-bench", saga.StateRunning)
		err := storage.SaveSaga(ctx, inst)
		if err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}
	b.StopTimer()

	// Report ops/sec
	elapsed := b.Elapsed()
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkPostgresStorage_GetSaga benchmarks the GetSaga operation.
func BenchmarkPostgresStorage_GetSaga(b *testing.B) {
	storage := skipIfNoPostgres(b)
	if storage == nil {
		return
	}
	defer storage.Close()
	defer cleanupBenchData(b, storage, "saga-bench-get")

	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 1000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-get-%d", i)
		instance := createBenchSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, instance); err != nil {
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
	b.StopTimer()

	elapsed := b.Elapsed()
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkPostgresStorage_UpdateSagaState benchmarks the UpdateSagaState operation.
func BenchmarkPostgresStorage_UpdateSagaState(b *testing.B) {
	storage := skipIfNoPostgres(b)
	if storage == nil {
		return
	}
	defer storage.Close()
	defer cleanupBenchData(b, storage, "saga-bench-update")

	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 500
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-update-%d", i)
		instance := createBenchSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, instance); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	metadata := map[string]interface{}{
		"updated":   true,
		"timestamp": time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sagaID := fmt.Sprintf("saga-bench-update-%d", i%numSagas)
		err := storage.UpdateSagaState(ctx, sagaID, saga.StateStepCompleted, metadata)
		if err != nil {
			b.Fatalf("Failed to update saga state: %v", err)
		}
	}
	b.StopTimer()

	elapsed := b.Elapsed()
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkPostgresStorage_GetActiveSagas benchmarks the GetActiveSagas operation.
func BenchmarkPostgresStorage_GetActiveSagas(b *testing.B) {
	storage := skipIfNoPostgres(b)
	if storage == nil {
		return
	}
	defer storage.Close()
	defer cleanupBenchData(b, storage, "saga-bench-filter")

	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 2000
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-filter-%d", i)
		state := saga.StateRunning
		if i%3 == 0 {
			state = saga.StateCompleted
		} else if i%3 == 1 {
			state = saga.StateFailed
		}
		instance := createBenchSagaInstance(sagaID, "def-bench", state)
		if err := storage.SaveSaga(ctx, instance); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	filter := &saga.SagaFilter{
		States: []saga.SagaState{saga.StateRunning, saga.StateStepCompleted},
		Limit:  100,
		Offset: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetActiveSagas(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to get active sagas: %v", err)
		}
	}
	b.StopTimer()

	elapsed := b.Elapsed()
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkPostgresStorage_ParallelGet benchmarks parallel GetSaga operations.
func BenchmarkPostgresStorage_ParallelGet(b *testing.B) {
	storage := skipIfNoPostgres(b)
	if storage == nil {
		return
	}
	defer storage.Close()
	defer cleanupBenchData(b, storage, "saga-bench-parallel")

	ctx := context.Background()

	// Pre-populate with sagas
	numSagas := 500
	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-bench-parallel-%d", i)
		instance := createBenchSagaInstance(sagaID, "def-bench", saga.StateRunning)
		if err := storage.SaveSaga(ctx, instance); err != nil {
			b.Fatalf("Failed to save saga: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sagaID := fmt.Sprintf("saga-bench-parallel-%d", i%numSagas)
			_, err := storage.GetSaga(ctx, sagaID)
			if err != nil {
				b.Fatalf("Failed to get saga: %v", err)
			}
			i++
		}
	})
}

// Helper functions

func createBenchSagaInstance(id, definitionID string, state saga.SagaState) saga.SagaInstance {
	return &benchSagaInstance{
		id:           id,
		definitionID: definitionID,
		name:         "Benchmark Saga",
		description:  "Saga for benchmark testing",
		sagaState:    state,
		currentStep:  0,
		totalSteps:   5,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		initialData:  map[string]interface{}{"test": "data"},
		currentData:  map[string]interface{}{"test": "data"},
		timeout:      time.Minute * 5,
		metadata:     map[string]interface{}{"env": "bench"},
		version:      1,
	}
}

func cleanupBenchData(b *testing.B, storage *PostgresStateStorage, prefix string) {
	ctx := context.Background()
	query := fmt.Sprintf("DELETE FROM %s WHERE id LIKE $1", storage.instancesTable)
	_, err := storage.db.ExecContext(ctx, query, prefix+"%")
	if err != nil {
		b.Logf("Warning: Failed to cleanup bench data: %v", err)
	}
}

// benchSagaInstance is a minimal implementation for benchmarking
type benchSagaInstance struct {
	id            string
	definitionID  string
	name          string
	description   string
	sagaState     saga.SagaState
	currentStep   int
	totalSteps    int
	createdAt     time.Time
	updatedAt     time.Time
	startedAt     *time.Time
	completedAt   *time.Time
	timedOutAt    *time.Time
	initialData   map[string]interface{}
	currentData   map[string]interface{}
	resultData    map[string]interface{}
	err           *saga.SagaError
	timeout       time.Duration
	retryPolicy   *saga.RetryPolicy
	metadata      map[string]interface{}
	traceID       string
	spanID        string
	version       int
}

func (t *benchSagaInstance) GetID() string                             { return t.id }
func (t *benchSagaInstance) GetDefinitionID() string                   { return t.definitionID }
func (t *benchSagaInstance) GetName() string                           { return t.name }
func (t *benchSagaInstance) GetDescription() string                    { return t.description }
func (t *benchSagaInstance) GetState() saga.SagaState                  { return t.sagaState }
func (t *benchSagaInstance) GetCurrentStep() int                       { return t.currentStep }
func (t *benchSagaInstance) GetTotalSteps() int                        { return t.totalSteps }
func (t *benchSagaInstance) GetCompletedSteps() int                    { return t.currentStep }
func (t *benchSagaInstance) GetCreatedAt() time.Time                   { return t.createdAt }
func (t *benchSagaInstance) GetUpdatedAt() time.Time                   { return t.updatedAt }
func (t *benchSagaInstance) GetStartTime() time.Time                   { return t.createdAt }
func (t *benchSagaInstance) GetEndTime() time.Time                     {
	if t.completedAt != nil {
		return *t.completedAt
	}
	return time.Time{}
}
func (t *benchSagaInstance) GetResult() interface{}                    { return t.resultData }
func (t *benchSagaInstance) GetStartedAt() *time.Time                  { return t.startedAt }
func (t *benchSagaInstance) GetCompletedAt() *time.Time                { return t.completedAt }
func (t *benchSagaInstance) GetTimedOutAt() *time.Time                 { return t.timedOutAt }
func (t *benchSagaInstance) GetInitialData() map[string]interface{}    { return t.initialData }
func (t *benchSagaInstance) GetCurrentData() map[string]interface{}    { return t.currentData }
func (t *benchSagaInstance) GetResultData() map[string]interface{}     { return t.resultData }
func (t *benchSagaInstance) GetError() *saga.SagaError                 { return t.err }
func (t *benchSagaInstance) GetTimeout() time.Duration                 { return t.timeout }
func (t *benchSagaInstance) GetRetryPolicy() *saga.RetryPolicy         { return t.retryPolicy }
func (t *benchSagaInstance) GetMetadata() map[string]interface{}       { return t.metadata }
func (t *benchSagaInstance) GetTraceID() string                        { return t.traceID }
func (t *benchSagaInstance) GetSpanID() string                         { return t.spanID }
func (t *benchSagaInstance) GetVersion() int                           { return t.version }
func (t *benchSagaInstance) IsTerminal() bool                          {
	return t.sagaState == saga.StateCompleted || t.sagaState == saga.StateFailed ||
		t.sagaState == saga.StateCancelled || t.sagaState == saga.StateTimedOut
}
func (t *benchSagaInstance) SetState(state saga.SagaState)             { t.sagaState = state }
func (t *benchSagaInstance) SetCurrentStep(step int)                   { t.currentStep = step }
func (t *benchSagaInstance) SetUpdatedAt(ts time.Time)                 { t.updatedAt = ts }
func (t *benchSagaInstance) SetStartedAt(ts *time.Time)                { t.startedAt = ts }
func (t *benchSagaInstance) SetCompletedAt(ts *time.Time)              { t.completedAt = ts }
func (t *benchSagaInstance) SetTimedOutAt(ts *time.Time)               { t.timedOutAt = ts }
func (t *benchSagaInstance) SetCurrentData(data map[string]interface{}) { t.currentData = data }
func (t *benchSagaInstance) SetResultData(data map[string]interface{}) { t.resultData = data }
func (t *benchSagaInstance) SetError(err *saga.SagaError)              { t.err = err }
func (t *benchSagaInstance) SetMetadata(m map[string]interface{})      { t.metadata = m }
func (t *benchSagaInstance) SetVersion(v int)                          { t.version = v }

