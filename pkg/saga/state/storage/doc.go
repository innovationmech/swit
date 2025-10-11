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

// Package storage provides concrete implementations of state storage backends
// for Saga state persistence.
//
// This package contains implementations of the saga.StateStorage interface,
// allowing Saga instances to be persisted to different storage systems based
// on the requirements of the application.
//
// # Available Backends
//
// Memory Storage:
//
//	storage := storage.NewMemoryStateStorage()
//
// Memory storage provides fast, in-memory persistence suitable for:
//   - Development and testing environments
//   - Short-lived Sagas that don't require durability
//   - High-performance scenarios where durability is not critical
//
// # Storage Interface
//
// All storage backends implement the saga.StateStorage interface:
//
//	type StateStorage interface {
//	    SaveSaga(ctx context.Context, saga SagaInstance) error
//	    GetSaga(ctx context.Context, sagaID string) (SagaInstance, error)
//	    UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error
//	    DeleteSaga(ctx context.Context, sagaID string) error
//	    GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]SagaInstance, error)
//	    GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error)
//	    SaveStepState(ctx context.Context, sagaID string, step *StepState) error
//	    GetStepStates(ctx context.Context, sagaID string) ([]*StepState, error)
//	    CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error
//	}
//
// # Concurrency
//
// All storage implementations are designed to be thread-safe and support
// concurrent access from multiple goroutines. Each implementation uses
// appropriate synchronization mechanisms to ensure data consistency.
//
// # Error Handling
//
// Storage operations return standard errors from the saga package:
//
//   - saga.ErrSagaNotFound: When a Saga instance cannot be found
//   - saga.ErrInvalidSagaID: When the provided Saga ID is invalid
//   - saga.ErrStorageFailure: When a storage operation fails
//
// # Best Practices
//
// 1. Choose the Right Backend:
//   - Memory: Development, testing, non-critical workloads
//   - Redis: Distributed systems, high performance, moderate durability
//   - Database: Critical workflows, ACID guarantees, long-term storage
//
// 2. Handle Context Cancellation:
//
//	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
//	defer cancel()
//	err := storage.SaveSaga(ctx, saga)
//
// 3. Implement Cleanup:
//
//	// Periodically clean up old Sagas
//	ticker := time.NewTicker(1 * time.Hour)
//	go func() {
//	    for range ticker.C {
//	        storage.CleanupExpiredSagas(ctx, time.Now().Add(-24*time.Hour))
//	    }
//	}()
//
// 4. Monitor Performance:
//   - Track storage operation latency
//   - Monitor storage capacity and growth
//   - Set up alerts for storage failures
//
// # Future Extensions
//
// The package is designed to be extensible. Future storage backends may include:
//   - Redis storage with automatic expiration and clustering
//   - PostgreSQL storage with full ACID guarantees
//   - MongoDB storage for document-based persistence
//   - Kafka storage for event-sourced Sagas
package storage
