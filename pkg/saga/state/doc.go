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

// Package state provides state management and persistence for Saga instances.
//
// This package offers a complete state management solution for the Saga pattern,
// including state storage backends, state managers, and serialization support.
// It enables reliable Saga execution by persisting instance state and providing
// recovery capabilities.
//
// # Architecture
//
// The state package consists of three main components:
//
//   - Storage Backends: Persist Saga state to various storage systems
//   - State Manager: Coordinate state operations and provide transaction-like semantics
//   - Serializer: Convert Saga state to/from different formats for storage
//
// # Storage Backends
//
// The package provides multiple storage backend implementations:
//
//   - Memory Storage: Fast in-memory storage for development and testing
//   - Redis Storage: Distributed storage with automatic expiration (future)
//   - Database Storage: Durable storage with ACID guarantees (future)
//
// Example usage:
//
//	// Create a memory storage backend
//	storage := memory.NewMemoryStateStorage()
//
//	// Save a Saga instance
//	err := storage.SaveSaga(ctx, sagaInstance)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Retrieve a Saga instance
//	instance, err := storage.GetSaga(ctx, sagaID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # State Manager
//
// The StateManager coordinates state operations and ensures consistency:
//
//	manager := state.NewStateManager(storage)
//
//	// Update Saga state atomically
//	err := manager.UpdateSagaState(ctx, sagaID, newState)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Serialization
//
// The Serializer converts Saga state for storage:
//
//	serializer := state.NewJSONSerializer()
//
//	// Serialize Saga instance
//	data, err := serializer.Serialize(sagaInstance)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Deserialize Saga instance
//	instance, err := serializer.Deserialize(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Concurrency and Safety
//
// All storage implementations are designed to be thread-safe and support
// concurrent access. The StateManager provides transaction-like semantics
// to ensure consistency when multiple operations need to be performed atomically.
//
// # Performance Considerations
//
// Different storage backends have different performance characteristics:
//
//   - Memory Storage: Fastest, but not durable across restarts
//   - Redis Storage: Good performance with durability and distribution
//   - Database Storage: Best durability guarantees, suitable for critical workflows
//
// Choose the appropriate backend based on your requirements for performance,
// durability, and distribution.
//
// # Error Handling
//
// All storage operations return errors that can be checked against the
// standard error types defined in the saga package:
//
//	instance, err := storage.GetSaga(ctx, sagaID)
//	if errors.Is(err, saga.ErrSagaNotFound) {
//	    // Handle not found case
//	}
//
// # Best Practices
//
//   - Use Memory Storage for development and testing
//   - Use Redis Storage for distributed systems with high performance requirements
//   - Use Database Storage for critical workflows requiring ACID guarantees
//   - Always handle context cancellation in long-running operations
//   - Implement cleanup of expired Sagas to prevent storage growth
//   - Monitor storage performance and adjust backend configuration as needed
package state
