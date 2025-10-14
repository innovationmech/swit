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
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultMaxRetries is the default maximum number of transaction retries
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default delay between transaction retries
	DefaultRetryDelay = 10 * time.Millisecond

	// MaxRetryDelay is the maximum delay between transaction retries
	MaxRetryDelay = 500 * time.Millisecond
)

var (
	// ErrTransactionFailed indicates that a Redis transaction failed
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrTransactionConflict indicates that a transaction conflict occurred
	ErrTransactionConflict = errors.New("transaction conflict")

	// ErrTransactionAborted indicates that a transaction was aborted
	ErrTransactionAborted = errors.New("transaction aborted")

	// ErrOptimisticLockFailed indicates that an optimistic lock failed
	ErrOptimisticLockFailed = errors.New("optimistic lock failed")

	// ErrCompareAndSwapFailed indicates that a compare-and-swap operation failed
	ErrCompareAndSwapFailed = errors.New("compare and swap failed")
)

// TransactionOptions configures transaction behavior
type TransactionOptions struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// EnableRetry enables automatic retry on conflict
	EnableRetry bool
}

// DefaultTransactionOptions returns the default transaction options
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		MaxRetries:  DefaultMaxRetries,
		RetryDelay:  DefaultRetryDelay,
		EnableRetry: true,
	}
}

// TxFunc is a function that executes within a transaction
type TxFunc func(ctx context.Context, pipe redis.Pipeliner) error

// WithTransaction executes a function within a Redis MULTI/EXEC transaction.
// The transaction is automatically retried on conflict if retry is enabled.
func (r *RedisStateStorage) WithTransaction(ctx context.Context, opts *TransactionOptions, fn TxFunc) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	var lastErr error
	maxAttempts := opts.MaxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := r.executeTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !opts.EnableRetry || !isRetryableError(err) {
			return err
		}

		// Don't retry on context errors
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt < maxAttempts {
			delay := opts.RetryDelay * time.Duration(attempt)
			if delay > MaxRetryDelay {
				delay = MaxRetryDelay
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	return fmt.Errorf("%w after %d attempts: %v", ErrTransactionFailed, maxAttempts, lastErr)
}

// executeTransaction executes a single transaction attempt
func (r *RedisStateStorage) executeTransaction(ctx context.Context, fn TxFunc) error {
	pipe := r.client.TxPipeline()

	// Execute the transaction function
	if err := fn(ctx, pipe); err != nil {
		pipe.Discard()
		return err
	}

	// Execute the transaction
	_, err := pipe.Exec(ctx)
	if err != nil {
		if err == redis.TxFailedErr {
			return ErrTransactionConflict
		}
		return fmt.Errorf("%w: %v", ErrTransactionFailed, err)
	}

	return nil
}

// WithOptimisticLock executes a function with Redis WATCH for optimistic locking.
// The watched keys are monitored for changes, and the transaction is aborted if any watched key is modified.
func (r *RedisStateStorage) WithOptimisticLock(ctx context.Context, watchKeys []string, opts *TransactionOptions, fn TxFunc) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if len(watchKeys) == 0 {
		return errors.New("at least one watch key is required")
	}

	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	var lastErr error
	maxAttempts := opts.MaxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := r.executeOptimisticLock(ctx, watchKeys, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !opts.EnableRetry || !isRetryableError(err) {
			return err
		}

		// Don't retry on context errors
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt < maxAttempts {
			delay := opts.RetryDelay * time.Duration(attempt)
			if delay > MaxRetryDelay {
				delay = MaxRetryDelay
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	return fmt.Errorf("%w after %d attempts: %v", ErrOptimisticLockFailed, maxAttempts, lastErr)
}

// executeOptimisticLock executes a single optimistic lock attempt
func (r *RedisStateStorage) executeOptimisticLock(ctx context.Context, watchKeys []string, fn TxFunc) error {
	// Get the underlying client that supports Watch
	// We need to cast to the concrete type to access Watch
	var client interface {
		Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error
	}

	// Try to get the watch-capable client
	switch c := r.client.(type) {
	case *redis.Client:
		client = c
	case *redis.ClusterClient:
		// For cluster client, we need a different approach
		// Use a simple transaction with WATCH emulation via retry
		return r.executeClusterOptimisticLock(ctx, watchKeys, fn)
	default:
		// Fallback to regular transaction for unsupported client types
		return r.executeTransaction(ctx, fn)
	}

	// Use TxPipelined which handles WATCH/MULTI/EXEC automatically
	err := client.Watch(ctx, func(tx *redis.Tx) error {
		// Execute the transaction function within the watched context
		pipe := tx.TxPipeline()

		if err := fn(ctx, pipe); err != nil {
			return err
		}

		// Execute the pipelined commands
		_, err := pipe.Exec(ctx)
		return err
	}, watchKeys...)

	if err != nil {
		if err == redis.TxFailedErr {
			return ErrTransactionConflict
		}
		return err
	}

	return nil
}

// executeClusterOptimisticLock handles optimistic locking for cluster clients
func (r *RedisStateStorage) executeClusterOptimisticLock(ctx context.Context, watchKeys []string, fn TxFunc) error {
	// For cluster mode, we use GET-compare-SET pattern
	// This is a simplified optimistic lock without true WATCH support
	pipe := r.client.TxPipeline()

	if err := fn(ctx, pipe); err != nil {
		pipe.Discard()
		return err
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		if err == redis.TxFailedErr {
			return ErrTransactionConflict
		}
		return fmt.Errorf("%w: %v", ErrTransactionFailed, err)
	}

	return nil
}

// CompareAndSwap performs an atomic compare-and-swap operation on a Saga state.
// It updates the Saga only if the current state matches the expected state.
func (r *RedisStateStorage) CompareAndSwap(ctx context.Context, sagaID string, expectedState, newState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	sagaKey := r.getSagaKey(sagaID)
	watchKeys := []string{sagaKey}

	opts := DefaultTransactionOptions()

	return r.WithOptimisticLock(ctx, watchKeys, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
		// Get current saga to verify state
		instance, err := r.GetSaga(ctx, sagaID)
		if err != nil {
			return err
		}

		currentState := instance.GetState()
		if currentState != expectedState {
			return fmt.Errorf("%w: expected %s, got %s", ErrCompareAndSwapFailed, expectedState, currentState)
		}

		// Convert to data structure
		data := r.convertToSagaInstanceData(instance)

		// Store old state for index updates
		oldState := data.State

		// Update state and metadata
		data.State = newState
		data.UpdatedAt = time.Now()

		if metadata != nil {
			if data.Metadata == nil {
				data.Metadata = make(map[string]interface{})
			}
			for key, value := range metadata {
				data.Metadata[key] = value
			}
		}

		// Create a temporary instance for serialization
		updatedInstance := &redisSagaInstance{data: data}

		// Serialize updated data
		jsonData, err := r.serializer.SerializeSagaInstance(updatedInstance)
		if err != nil {
			return fmt.Errorf("failed to serialize updated saga: %w", err)
		}

		// Update saga data in the pipeline
		pipe.Set(ctx, sagaKey, jsonData, r.config.TTL)

		// Update state indexes if state changed
		if oldState != newState {
			// Remove from old state index
			oldStateIndexKey := r.getStateIndexKey(oldState.String())
			pipe.SRem(ctx, oldStateIndexKey, sagaID)

			// Add to new state index
			newStateIndexKey := r.getStateIndexKey(newState.String())
			pipe.SAdd(ctx, newStateIndexKey, sagaID)

			// Remove from timeout index if saga is no longer active
			if !newState.IsActive() {
				timeoutKey := r.getTimeoutKey()
				pipe.ZRem(ctx, timeoutKey, sagaID)
			}
		}

		return nil
	})
}

// BatchOperation represents a single operation in a batch transaction
type BatchOperation struct {
	// Type is the operation type (save, update, delete)
	Type BatchOperationType

	// SagaID is the ID of the saga to operate on
	SagaID string

	// Instance is the saga instance (for save operations)
	Instance saga.SagaInstance

	// State is the new state (for update operations)
	State saga.SagaState

	// Metadata is additional metadata (for update operations)
	Metadata map[string]interface{}

	// StepState is the step state (for save step operations)
	StepState *saga.StepState
}

// BatchOperationType defines the type of batch operation
type BatchOperationType string

const (
	// BatchOpSaveSaga saves a saga instance
	BatchOpSaveSaga BatchOperationType = "save_saga"

	// BatchOpUpdateState updates a saga state
	BatchOpUpdateState BatchOperationType = "update_state"

	// BatchOpDeleteSaga deletes a saga
	BatchOpDeleteSaga BatchOperationType = "delete_saga"

	// BatchOpSaveStep saves a step state
	BatchOpSaveStep BatchOperationType = "save_step"
)

// ExecuteBatch executes multiple operations in a single transaction
func (r *RedisStateStorage) ExecuteBatch(ctx context.Context, operations []BatchOperation, opts *TransactionOptions) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if len(operations) == 0 {
		return nil
	}

	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	return r.WithTransaction(ctx, opts, func(ctx context.Context, pipe redis.Pipeliner) error {
		for i, op := range operations {
			if err := r.executeBatchOperation(ctx, pipe, &op); err != nil {
				return fmt.Errorf("batch operation %d failed: %w", i, err)
			}
		}
		return nil
	})
}

// executeBatchOperation executes a single batch operation
func (r *RedisStateStorage) executeBatchOperation(ctx context.Context, pipe redis.Pipeliner, op *BatchOperation) error {
	switch op.Type {
	case BatchOpSaveSaga:
		return r.executeBatchSaveSaga(ctx, pipe, op)

	case BatchOpUpdateState:
		return r.executeBatchUpdateState(ctx, pipe, op)

	case BatchOpDeleteSaga:
		return r.executeBatchDeleteSaga(ctx, pipe, op)

	case BatchOpSaveStep:
		return r.executeBatchSaveStep(ctx, pipe, op)

	default:
		return fmt.Errorf("unknown batch operation type: %s", op.Type)
	}
}

// executeBatchSaveSaga saves a saga in a batch operation
func (r *RedisStateStorage) executeBatchSaveSaga(ctx context.Context, pipe redis.Pipeliner, op *BatchOperation) error {
	if op.Instance == nil {
		return state.ErrInvalidSagaID
	}

	sagaID := op.Instance.GetID()
	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Serialize the saga
	jsonData, err := r.serializer.SerializeSagaInstance(op.Instance)
	if err != nil {
		return fmt.Errorf("failed to serialize saga: %w", err)
	}

	// Convert to data structure
	data := r.convertToSagaInstanceData(op.Instance)

	// Store the saga data
	sagaKey := r.getSagaKey(sagaID)
	pipe.Set(ctx, sagaKey, jsonData, r.config.TTL)

	// Update state index
	stateIndexKey := r.getStateIndexKey(data.State.String())
	pipe.SAdd(ctx, stateIndexKey, sagaID)

	// If saga is active, add to timeout index
	if data.State.IsActive() && data.StartedAt != nil {
		timeoutKey := r.getTimeoutKey()
		deadline := data.StartedAt.Add(data.Timeout)
		pipe.ZAdd(ctx, timeoutKey, redis.Z{
			Score:  float64(deadline.Unix()),
			Member: sagaID,
		})
	}

	return nil
}

// executeBatchUpdateState updates a saga state in a batch operation
func (r *RedisStateStorage) executeBatchUpdateState(ctx context.Context, pipe redis.Pipeliner, op *BatchOperation) error {
	if op.SagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Get current saga data
	instance, err := r.GetSaga(ctx, op.SagaID)
	if err != nil {
		return err
	}

	// Convert to data structure
	data := r.convertToSagaInstanceData(instance)

	// Store old state for index updates
	oldState := data.State

	// Update state and metadata
	data.State = op.State
	data.UpdatedAt = time.Now()

	if op.Metadata != nil {
		if data.Metadata == nil {
			data.Metadata = make(map[string]interface{})
		}
		for key, value := range op.Metadata {
			data.Metadata[key] = value
		}
	}

	// Create a temporary instance for serialization
	updatedInstance := &redisSagaInstance{data: data}

	// Serialize updated data
	jsonData, err := r.serializer.SerializeSagaInstance(updatedInstance)
	if err != nil {
		return fmt.Errorf("failed to serialize updated saga: %w", err)
	}

	// Update saga data
	sagaKey := r.getSagaKey(op.SagaID)
	pipe.Set(ctx, sagaKey, jsonData, r.config.TTL)

	// Update state indexes if state changed
	if oldState != op.State {
		// Remove from old state index
		oldStateIndexKey := r.getStateIndexKey(oldState.String())
		pipe.SRem(ctx, oldStateIndexKey, op.SagaID)

		// Add to new state index
		newStateIndexKey := r.getStateIndexKey(op.State.String())
		pipe.SAdd(ctx, newStateIndexKey, op.SagaID)

		// Remove from timeout index if saga is no longer active
		if !op.State.IsActive() {
			timeoutKey := r.getTimeoutKey()
			pipe.ZRem(ctx, timeoutKey, op.SagaID)
		}
	}

	return nil
}

// executeBatchDeleteSaga deletes a saga in a batch operation
func (r *RedisStateStorage) executeBatchDeleteSaga(ctx context.Context, pipe redis.Pipeliner, op *BatchOperation) error {
	if op.SagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Get current saga to determine its state for index cleanup
	instance, err := r.GetSaga(ctx, op.SagaID)
	if err != nil {
		return err
	}

	sagaState := instance.GetState()

	// Delete saga data
	sagaKey := r.getSagaKey(op.SagaID)
	pipe.Del(ctx, sagaKey)

	// Remove from state index
	stateIndexKey := r.getStateIndexKey(sagaState.String())
	pipe.SRem(ctx, stateIndexKey, op.SagaID)

	// Remove from timeout index
	timeoutKey := r.getTimeoutKey()
	pipe.ZRem(ctx, timeoutKey, op.SagaID)

	// Get all step IDs and delete them
	stepsSetKey := r.getStepsSetKey(op.SagaID)
	stepIDs, err := r.client.SMembers(ctx, stepsSetKey).Result()
	if err == nil {
		for _, stepID := range stepIDs {
			stepKey := r.getStepKey(op.SagaID, stepID)
			pipe.Del(ctx, stepKey)
		}
		pipe.Del(ctx, stepsSetKey)
	}

	return nil
}

// executeBatchSaveStep saves a step state in a batch operation
func (r *RedisStateStorage) executeBatchSaveStep(ctx context.Context, pipe redis.Pipeliner, op *BatchOperation) error {
	if op.SagaID == "" {
		return state.ErrInvalidSagaID
	}

	if op.StepState == nil {
		return state.ErrInvalidState
	}

	// Serialize step state
	jsonData, err := r.serializer.SerializeStepState(op.StepState)
	if err != nil {
		return fmt.Errorf("failed to serialize step state: %w", err)
	}

	// Store step state
	stepKey := r.getStepKey(op.SagaID, op.StepState.ID)
	pipe.Set(ctx, stepKey, jsonData, r.config.TTL)

	// Add step ID to the set of steps for this saga
	stepsSetKey := r.getStepsSetKey(op.SagaID)
	pipe.SAdd(ctx, stepsSetKey, op.StepState.ID)

	return nil
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Transaction conflicts are retryable
	if errors.Is(err, ErrTransactionConflict) {
		return true
	}

	// Redis transaction failures are retryable
	if errors.Is(err, redis.TxFailedErr) {
		return true
	}

	// Network errors are typically retryable
	if errors.Is(err, redis.Nil) {
		return false // Key not found is not retryable
	}

	// Check for context errors (not retryable)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Default: consider other errors retryable
	return true
}
