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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/redis/go-redis/v9"
)

// Redis key naming conventions
const (
	// sagaKeyPattern is the pattern for Saga instance keys: {prefix}saga:{sagaID}
	sagaKeyPattern = "%ssaga:%s"

	// stepKeyPattern is the pattern for step state keys: {prefix}step:{sagaID}:{stepID}
	stepKeyPattern = "%sstep:%s:%s"

	// stepsSetKeyPattern is the pattern for the set of step IDs: {prefix}steps:{sagaID}
	stepsSetKeyPattern = "%ssteps:%s"

	// sagaIndexKeyPattern is the pattern for indexing sagas by state: {prefix}index:state:{state}
	sagaIndexKeyPattern = "%sindex:state:%s"

	// sagaTimeoutKeyPattern is the pattern for timeout sorted set: {prefix}timeout
	sagaTimeoutKeyPattern = "%stimeout"
)

// RedisStateStorage provides a Redis-based implementation of saga.StateStorage.
// It stores Saga instances and their states in Redis, supporting standalone, cluster,
// and sentinel modes. This implementation is suitable for production environments
// requiring high availability and distributed state management.
//
// Key Design:
//   - Saga instances: {prefix}saga:{sagaID}
//   - Step states: {prefix}step:{sagaID}:{stepID}
//   - Step set: {prefix}steps:{sagaID} (set of step IDs)
//   - State index: {prefix}index:state:{state} (set of saga IDs)
//   - Timeout index: {prefix}timeout (sorted set with deadline as score)
//
// The storage is thread-safe and supports concurrent access from multiple processes.
type RedisStateStorage struct {
	// conn is the Redis connection manager
	conn *RedisConnection

	// config holds the Redis configuration
	config *RedisConfig

	// client is the underlying Redis client
	client RedisClient

	// closed indicates whether the storage has been closed
	closed bool
}

// NewRedisStateStorage creates a new Redis state storage with the specified configuration.
// It establishes a connection to Redis and verifies connectivity.
func NewRedisStateStorage(config *RedisConfig) (*RedisStateStorage, error) {
	if config == nil {
		return nil, ErrInvalidRedisConfig
	}

	// Create Redis connection
	conn, err := NewRedisConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis connection: %w", err)
	}

	// Get the client
	client, err := conn.GetClient()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get redis client: %w", err)
	}

	// Verify connectivity with a ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisStateStorage{
		conn:   conn,
		config: config,
		client: client,
		closed: false,
	}, nil
}

// NewRedisStateStorageWithRetry creates a new Redis state storage with retry logic.
// It will attempt to connect multiple times with exponential backoff before giving up.
func NewRedisStateStorageWithRetry(ctx context.Context, config *RedisConfig, maxAttempts int) (*RedisStateStorage, error) {
	if config == nil {
		return nil, ErrInvalidRedisConfig
	}

	// Create Redis connection with retry
	conn, err := NewRedisConnectionWithRetry(ctx, config, maxAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis connection with retry: %w", err)
	}

	// Get the client
	client, err := conn.GetClient()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get redis client: %w", err)
	}

	return &RedisStateStorage{
		conn:   conn,
		config: config,
		client: client,
		closed: false,
	}, nil
}

// SaveSaga persists a Saga instance to Redis storage.
// It serializes the Saga data to JSON and stores it with the configured TTL.
// It also updates state and timeout indexes for efficient querying.
func (r *RedisStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if instance == nil {
		return state.ErrInvalidSagaID
	}

	sagaID := instance.GetID()
	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Convert SagaInstance to SagaInstanceData
	data := r.convertToSagaInstanceData(instance)

	// Serialize to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize saga: %w", err)
	}

	// Prepare Redis commands using pipeline for atomicity
	pipe := r.client.Pipeline()

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

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save saga to redis: %w", err)
	}

	return nil
}

// GetSaga retrieves a Saga instance by its ID from Redis storage.
// Returns state.ErrSagaNotFound if the Saga does not exist.
func (r *RedisStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if r.closed {
		return nil, state.ErrStorageClosed
	}

	if sagaID == "" {
		return nil, state.ErrInvalidSagaID
	}

	// Get saga data from Redis
	sagaKey := r.getSagaKey(sagaID)
	jsonData, err := r.client.Get(ctx, sagaKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, state.ErrSagaNotFound
		}
		return nil, fmt.Errorf("failed to get saga from redis: %w", err)
	}

	// Deserialize from JSON
	var data saga.SagaInstanceData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to deserialize saga: %w", err)
	}

	// Return wrapped instance
	return &redisSagaInstance{data: &data}, nil
}

// UpdateSagaState updates only the state of a Saga instance.
// This is a lightweight operation that doesn't require loading the entire Saga.
func (r *RedisStateStorage) UpdateSagaState(ctx context.Context, sagaID string, sagaState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Get current saga data
	instance, err := r.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	// Convert to data structure
	data := r.convertToSagaInstanceData(instance)

	// Store old state for index updates
	oldState := data.State

	// Update state and metadata
	data.State = sagaState
	data.UpdatedAt = time.Now()

	if metadata != nil {
		if data.Metadata == nil {
			data.Metadata = make(map[string]interface{})
		}
		for key, value := range metadata {
			data.Metadata[key] = value
		}
	}

	// Serialize updated data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize updated saga: %w", err)
	}

	// Prepare Redis commands using pipeline
	pipe := r.client.Pipeline()

	// Update saga data
	sagaKey := r.getSagaKey(sagaID)
	pipe.Set(ctx, sagaKey, jsonData, r.config.TTL)

	// Update state indexes if state changed
	if oldState != sagaState {
		// Remove from old state index
		oldStateIndexKey := r.getStateIndexKey(oldState.String())
		pipe.SRem(ctx, oldStateIndexKey, sagaID)

		// Add to new state index
		newStateIndexKey := r.getStateIndexKey(sagaState.String())
		pipe.SAdd(ctx, newStateIndexKey, sagaID)

		// Remove from timeout index if saga is no longer active
		if !sagaState.IsActive() {
			timeoutKey := r.getTimeoutKey()
			pipe.ZRem(ctx, timeoutKey, sagaID)
		}
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update saga state in redis: %w", err)
	}

	return nil
}

// DeleteSaga removes a Saga instance from Redis storage.
// Returns state.ErrSagaNotFound if the Saga does not exist.
func (r *RedisStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Get current saga to determine its state for index cleanup
	instance, err := r.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	sagaState := instance.GetState()

	// Prepare Redis commands using pipeline
	pipe := r.client.Pipeline()

	// Delete saga data
	sagaKey := r.getSagaKey(sagaID)
	pipe.Del(ctx, sagaKey)

	// Remove from state index
	stateIndexKey := r.getStateIndexKey(sagaState.String())
	pipe.SRem(ctx, stateIndexKey, sagaID)

	// Remove from timeout index
	timeoutKey := r.getTimeoutKey()
	pipe.ZRem(ctx, timeoutKey, sagaID)

	// Get all step IDs and delete them
	stepsSetKey := r.getStepsSetKey(sagaID)
	stepIDs, err := r.client.SMembers(ctx, stepsSetKey).Result()
	if err == nil {
		for _, stepID := range stepIDs {
			stepKey := r.getStepKey(sagaID, stepID)
			pipe.Del(ctx, stepKey)
		}
		pipe.Del(ctx, stepsSetKey)
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete saga from redis: %w", err)
	}

	return nil
}

// GetActiveSagas retrieves all active Saga instances based on the filter.
func (r *RedisStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if r.closed {
		return nil, state.ErrStorageClosed
	}

	var result []saga.SagaInstance
	var sagaIDs []string

	// If filter specifies states, query by state indexes
	if filter != nil && len(filter.States) > 0 {
		for _, filterState := range filter.States {
			stateIndexKey := r.getStateIndexKey(filterState.String())
			ids, err := r.client.SMembers(ctx, stateIndexKey).Result()
			if err != nil && err != redis.Nil {
				return nil, fmt.Errorf("failed to get sagas by state: %w", err)
			}
			sagaIDs = append(sagaIDs, ids...)
		}
	} else {
		// Get all active states
		activeStates := []saga.SagaState{saga.StateRunning, saga.StateStepCompleted, saga.StateCompensating}
		for _, activeState := range activeStates {
			stateIndexKey := r.getStateIndexKey(activeState.String())
			ids, err := r.client.SMembers(ctx, stateIndexKey).Result()
			if err != nil && err != redis.Nil {
				return nil, fmt.Errorf("failed to get active sagas: %w", err)
			}
			sagaIDs = append(sagaIDs, ids...)
		}
	}

	// Fetch each saga and apply additional filters
	for _, sagaID := range sagaIDs {
		instance, err := r.GetSaga(ctx, sagaID)
		if err != nil {
			if errors.Is(err, state.ErrSagaNotFound) {
				// Saga was deleted between index query and fetch, skip it
				continue
			}
			return nil, err
		}

		if r.matchesFilter(instance, filter) {
			result = append(result, instance)
		}
	}

	return result, nil
}

// GetTimeoutSagas retrieves Saga instances that have timed out before the specified time.
func (r *RedisStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if r.closed {
		return nil, state.ErrStorageClosed
	}

	// Query timeout sorted set
	timeoutKey := r.getTimeoutKey()
	sagaIDs, err := r.client.ZRangeByScore(ctx, timeoutKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", before.Unix()),
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get timeout sagas: %w", err)
	}

	var result []saga.SagaInstance

	// Fetch each saga
	for _, sagaID := range sagaIDs {
		instance, err := r.GetSaga(ctx, sagaID)
		if err != nil {
			if errors.Is(err, state.ErrSagaNotFound) {
				// Saga was deleted, remove from timeout index
				r.client.ZRem(ctx, timeoutKey, sagaID)
				continue
			}
			return nil, err
		}

		result = append(result, instance)
	}

	return result, nil
}

// SaveStepState persists the state of a specific step within a Saga.
func (r *RedisStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if step == nil {
		return state.ErrInvalidState
	}

	// Verify that the Saga exists
	_, err := r.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	// Serialize step state
	jsonData, err := json.Marshal(step)
	if err != nil {
		return fmt.Errorf("failed to serialize step state: %w", err)
	}

	// Prepare Redis commands using pipeline
	pipe := r.client.Pipeline()

	// Store step state
	stepKey := r.getStepKey(sagaID, step.ID)
	pipe.Set(ctx, stepKey, jsonData, r.config.TTL)

	// Add step ID to the set of steps for this saga
	stepsSetKey := r.getStepsSetKey(sagaID)
	pipe.SAdd(ctx, stepsSetKey, step.ID)

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save step state to redis: %w", err)
	}

	return nil
}

// GetStepStates retrieves all step states for a Saga instance.
func (r *RedisStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if r.closed {
		return nil, state.ErrStorageClosed
	}

	if sagaID == "" {
		return nil, state.ErrInvalidSagaID
	}

	// Verify that the Saga exists
	_, err := r.GetSaga(ctx, sagaID)
	if err != nil {
		return nil, err
	}

	// Get all step IDs for this saga
	stepsSetKey := r.getStepsSetKey(sagaID)
	stepIDs, err := r.client.SMembers(ctx, stepsSetKey).Result()
	if err != nil {
		if err == redis.Nil {
			return []*saga.StepState{}, nil
		}
		return nil, fmt.Errorf("failed to get step IDs: %w", err)
	}

	if len(stepIDs) == 0 {
		return []*saga.StepState{}, nil
	}

	// Fetch each step state
	var result []*saga.StepState

	for _, stepID := range stepIDs {
		stepKey := r.getStepKey(sagaID, stepID)
		jsonData, err := r.client.Get(ctx, stepKey).Result()
		if err != nil {
			if err == redis.Nil {
				// Step was deleted, skip it
				continue
			}
			return nil, fmt.Errorf("failed to get step state: %w", err)
		}

		var step saga.StepState
		if err := json.Unmarshal([]byte(jsonData), &step); err != nil {
			return nil, fmt.Errorf("failed to deserialize step state: %w", err)
		}

		result = append(result, &step)
	}

	return result, nil
}

// CleanupExpiredSagas removes Saga instances that are older than the specified time.
func (r *RedisStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	// Get all terminal state Sagas
	terminalStates := []saga.SagaState{
		saga.StateCompleted,
		saga.StateCompensated,
		saga.StateFailed,
		saga.StateCancelled,
		saga.StateTimedOut,
	}

	var toDelete []string

	for _, terminalState := range terminalStates {
		stateIndexKey := r.getStateIndexKey(terminalState.String())
		sagaIDs, err := r.client.SMembers(ctx, stateIndexKey).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to get terminal sagas: %w", err)
		}

		// Check each saga's update time
		for _, sagaID := range sagaIDs {
			instance, err := r.GetSaga(ctx, sagaID)
			if err != nil {
				if errors.Is(err, state.ErrSagaNotFound) {
					continue
				}
				return err
			}

			if instance.GetUpdatedAt().Before(olderThan) {
				toDelete = append(toDelete, sagaID)
			}
		}
	}

	// Delete expired sagas
	for _, sagaID := range toDelete {
		if err := r.DeleteSaga(ctx, sagaID); err != nil {
			// Log error but continue cleanup
			continue
		}
	}

	return nil
}

// Close closes the storage and releases resources.
func (r *RedisStateStorage) Close() error {
	if r.closed {
		return nil
	}

	r.closed = true

	if r.conn != nil {
		return r.conn.Close()
	}

	return nil
}

// HealthCheck performs a health check on the Redis connection.
func (r *RedisStateStorage) HealthCheck(ctx context.Context) error {
	if r.closed {
		return state.ErrStorageClosed
	}

	return r.conn.HealthCheck(ctx)
}

// getSagaKey returns the Redis key for a Saga instance.
func (r *RedisStateStorage) getSagaKey(sagaID string) string {
	return fmt.Sprintf(sagaKeyPattern, r.config.KeyPrefix, sagaID)
}

// getStepKey returns the Redis key for a step state.
func (r *RedisStateStorage) getStepKey(sagaID, stepID string) string {
	return fmt.Sprintf(stepKeyPattern, r.config.KeyPrefix, sagaID, stepID)
}

// getStepsSetKey returns the Redis key for the set of step IDs.
func (r *RedisStateStorage) getStepsSetKey(sagaID string) string {
	return fmt.Sprintf(stepsSetKeyPattern, r.config.KeyPrefix, sagaID)
}

// getStateIndexKey returns the Redis key for a state index.
func (r *RedisStateStorage) getStateIndexKey(stateName string) string {
	return fmt.Sprintf(sagaIndexKeyPattern, r.config.KeyPrefix, stateName)
}

// getTimeoutKey returns the Redis key for the timeout sorted set.
func (r *RedisStateStorage) getTimeoutKey() string {
	return fmt.Sprintf(sagaTimeoutKeyPattern, r.config.KeyPrefix)
}

// convertToSagaInstanceData converts a SagaInstance to SagaInstanceData for storage.
func (r *RedisStateStorage) convertToSagaInstanceData(instance saga.SagaInstance) *saga.SagaInstanceData {
	now := time.Now()
	startTime := instance.GetStartTime()
	endTime := instance.GetEndTime()

	data := &saga.SagaInstanceData{
		ID:           instance.GetID(),
		DefinitionID: instance.GetDefinitionID(),
		State:        instance.GetState(),
		CurrentStep:  instance.GetCurrentStep(),
		TotalSteps:   instance.GetTotalSteps(),
		CreatedAt:    instance.GetCreatedAt(),
		UpdatedAt:    instance.GetUpdatedAt(),
		Timeout:      instance.GetTimeout(),
		Metadata:     instance.GetMetadata(),
		TraceID:      instance.GetTraceID(),
		Error:        instance.GetError(),
		ResultData:   instance.GetResult(),
	}

	// Set started time if available
	if !startTime.IsZero() {
		data.StartedAt = &startTime
	}

	// Set completed time if available
	if !endTime.IsZero() {
		if instance.GetState() == saga.StateTimedOut {
			data.TimedOutAt = &endTime
		} else {
			data.CompletedAt = &endTime
		}
	}

	// Update UpdatedAt if it's zero
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = now
	}

	return data
}

// matchesFilter checks if a Saga instance matches the given filter criteria.
func (r *RedisStateStorage) matchesFilter(instance saga.SagaInstance, filter *saga.SagaFilter) bool {
	if filter == nil {
		return true
	}

	// Filter by state
	if len(filter.States) > 0 {
		matched := false
		for _, state := range filter.States {
			if instance.GetState() == state {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by definition ID
	if len(filter.DefinitionIDs) > 0 {
		matched := false
		for _, defID := range filter.DefinitionIDs {
			if instance.GetDefinitionID() == defID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by creation time
	if filter.CreatedAfter != nil && instance.GetCreatedAt().Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && instance.GetCreatedAt().After(*filter.CreatedBefore) {
		return false
	}

	// Filter by metadata
	if len(filter.Metadata) > 0 {
		instanceMetadata := instance.GetMetadata()
		if instanceMetadata == nil {
			return false
		}
		for key, value := range filter.Metadata {
			if instanceMetadata[key] != value {
				return false
			}
		}
	}

	return true
}

// redisSagaInstance is a Redis-backed implementation of saga.SagaInstance.
type redisSagaInstance struct {
	data *saga.SagaInstanceData
}

// GetID returns the unique identifier of this Saga instance.
func (r *redisSagaInstance) GetID() string {
	return r.data.ID
}

// GetDefinitionID returns the identifier of the Saga definition this instance follows.
func (r *redisSagaInstance) GetDefinitionID() string {
	return r.data.DefinitionID
}

// GetState returns the current state of the Saga instance.
func (r *redisSagaInstance) GetState() saga.SagaState {
	return r.data.State
}

// GetCurrentStep returns the index of the currently executing step.
func (r *redisSagaInstance) GetCurrentStep() int {
	return r.data.CurrentStep
}

// GetStartTime returns the time when the Saga instance was started.
func (r *redisSagaInstance) GetStartTime() time.Time {
	if r.data.StartedAt != nil {
		return *r.data.StartedAt
	}
	return time.Time{}
}

// GetEndTime returns the time when the Saga instance reached a terminal state.
func (r *redisSagaInstance) GetEndTime() time.Time {
	if r.data.CompletedAt != nil {
		return *r.data.CompletedAt
	}
	if r.data.TimedOutAt != nil {
		return *r.data.TimedOutAt
	}
	return time.Time{}
}

// GetResult returns the final result data of the Saga execution.
func (r *redisSagaInstance) GetResult() interface{} {
	return r.data.ResultData
}

// GetError returns the error that caused the Saga to fail.
func (r *redisSagaInstance) GetError() *saga.SagaError {
	return r.data.Error
}

// GetTotalSteps returns the total number of steps in the Saga definition.
func (r *redisSagaInstance) GetTotalSteps() int {
	return r.data.TotalSteps
}

// GetCompletedSteps returns the number of steps that have completed successfully.
func (r *redisSagaInstance) GetCompletedSteps() int {
	count := 0
	for _, step := range r.data.StepStates {
		if step.State == saga.StepStateCompleted {
			count++
		}
	}
	return count
}

// GetCreatedAt returns the creation time of the Saga instance.
func (r *redisSagaInstance) GetCreatedAt() time.Time {
	return r.data.CreatedAt
}

// GetUpdatedAt returns the last update time of the Saga instance.
func (r *redisSagaInstance) GetUpdatedAt() time.Time {
	return r.data.UpdatedAt
}

// GetTimeout returns the timeout duration for this Saga instance.
func (r *redisSagaInstance) GetTimeout() time.Duration {
	return r.data.Timeout
}

// GetMetadata returns the metadata associated with this Saga instance.
func (r *redisSagaInstance) GetMetadata() map[string]interface{} {
	return r.data.Metadata
}

// GetTraceID returns the distributed tracing identifier for this Saga.
func (r *redisSagaInstance) GetTraceID() string {
	return r.data.TraceID
}

// IsTerminal returns true if the Saga is in a terminal state.
func (r *redisSagaInstance) IsTerminal() bool {
	return r.data.State.IsTerminal()
}

// IsActive returns true if the Saga is currently active.
func (r *redisSagaInstance) IsActive() bool {
	return r.data.State.IsActive()
}
