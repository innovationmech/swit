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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// ErrCleanupInProgress indicates that a cleanup operation is already in progress
	ErrCleanupInProgress = errors.New("cleanup operation already in progress")

	// ErrInvalidCleanupPolicy indicates that the cleanup policy is invalid
	ErrInvalidCleanupPolicy = errors.New("invalid cleanup policy")
)

// CleanupPolicy defines the policy for cleaning up expired Saga states
type CleanupPolicy struct {
	// MaxAge is the maximum age for terminal Sagas before they are cleaned up.
	// Zero means no age-based cleanup.
	MaxAge time.Duration `json:"max_age" yaml:"max_age"`

	// MaxCount is the maximum number of terminal Sagas to keep per state.
	// Zero means no count-based cleanup.
	MaxCount int `json:"max_count" yaml:"max_count"`

	// BatchSize is the number of Sagas to process in each cleanup batch.
	// Default: 100
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// CleanupInterval is the interval between automatic cleanup runs.
	// Zero means no automatic cleanup.
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`

	// StatestoClean specifies which terminal states should be cleaned up.
	// Empty means all terminal states.
	StatesToClean []saga.SagaState `json:"states_to_clean" yaml:"states_to_clean"`

	// EnableMetrics enables cleanup metrics collection.
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
}

// DefaultCleanupPolicy returns the default cleanup policy
func DefaultCleanupPolicy() *CleanupPolicy {
	return &CleanupPolicy{
		MaxAge:          7 * 24 * time.Hour, // 7 days
		MaxCount:        0,                  // No count-based limit
		BatchSize:       100,
		CleanupInterval: 1 * time.Hour,
		StatesToClean:   nil, // All terminal states
		EnableMetrics:   true,
	}
}

// Validate validates the cleanup policy
func (p *CleanupPolicy) Validate() error {
	if p == nil {
		return ErrInvalidCleanupPolicy
	}

	if p.MaxAge < 0 {
		return fmt.Errorf("%w: max age must be >= 0", ErrInvalidCleanupPolicy)
	}

	if p.MaxCount < 0 {
		return fmt.Errorf("%w: max count must be >= 0", ErrInvalidCleanupPolicy)
	}

	if p.BatchSize <= 0 {
		return fmt.Errorf("%w: batch size must be > 0", ErrInvalidCleanupPolicy)
	}

	if p.CleanupInterval < 0 {
		return fmt.Errorf("%w: cleanup interval must be >= 0", ErrInvalidCleanupPolicy)
	}

	// Validate states to clean
	for _, state := range p.StatesToClean {
		if !state.IsTerminal() {
			return fmt.Errorf("%w: cannot clean non-terminal state %s", ErrInvalidCleanupPolicy, state)
		}
	}

	return nil
}

// CleanupStats holds statistics about cleanup operations
type CleanupStats struct {
	// StartTime is when the cleanup operation started
	StartTime time.Time `json:"start_time"`

	// EndTime is when the cleanup operation completed
	EndTime time.Time `json:"end_time"`

	// Duration is how long the cleanup took
	Duration time.Duration `json:"duration"`

	// ScannedCount is the number of Sagas scanned
	ScannedCount int `json:"scanned_count"`

	// DeletedCount is the number of Sagas deleted
	DeletedCount int `json:"deleted_count"`

	// ErrorCount is the number of errors encountered
	ErrorCount int `json:"error_count"`

	// StateBreakdown shows the count of deleted Sagas by state
	StateBreakdown map[string]int `json:"state_breakdown"`

	// LastError is the last error encountered
	LastError error `json:"last_error,omitempty"`
}

// NewCleanupStats creates a new CleanupStats instance
func NewCleanupStats() *CleanupStats {
	return &CleanupStats{
		StartTime:      time.Now(),
		StateBreakdown: make(map[string]int),
	}
}

// Complete marks the cleanup as complete and calculates duration
func (s *CleanupStats) Complete() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
}

// ExpirationManager manages the expiration and cleanup of Saga states
type ExpirationManager struct {
	storage      *RedisStateStorage
	policy       *CleanupPolicy
	ticker       *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
	cleanupMutex sync.Mutex
	lastCleanup  time.Time
	lastStats    *CleanupStats
	statsLock    sync.RWMutex
	running      bool
	runningMutex sync.RWMutex
}

// NewExpirationManager creates a new expiration manager
func NewExpirationManager(storage *RedisStateStorage, policy *CleanupPolicy) (*ExpirationManager, error) {
	if storage == nil {
		return nil, errors.New("storage cannot be nil")
	}

	if policy == nil {
		policy = DefaultCleanupPolicy()
	}

	if err := policy.Validate(); err != nil {
		return nil, err
	}

	return &ExpirationManager{
		storage: storage,
		policy:  policy,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start starts the automatic cleanup task
func (m *ExpirationManager) Start(ctx context.Context) error {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()

	if m.running {
		return errors.New("expiration manager already running")
	}

	// Only start ticker if cleanup interval is set
	if m.policy.CleanupInterval > 0 {
		m.ticker = time.NewTicker(m.policy.CleanupInterval)
		m.wg.Add(1)
		go m.cleanupLoop(ctx)
	}

	m.running = true
	logger.GetLogger().Info("expiration manager started",
		zap.Duration("cleanup_interval", m.policy.CleanupInterval),
		zap.Duration("max_age", m.policy.MaxAge),
		zap.Int("max_count", m.policy.MaxCount),
	)

	return nil
}

// Stop stops the automatic cleanup task
func (m *ExpirationManager) Stop() error {
	m.runningMutex.Lock()
	defer m.runningMutex.Unlock()

	if !m.running {
		return nil
	}

	close(m.stopCh)

	if m.ticker != nil {
		m.ticker.Stop()
	}

	m.wg.Wait()
	m.running = false

	logger.GetLogger().Info("expiration manager stopped")
	return nil
}

// cleanupLoop runs the cleanup task periodically
func (m *ExpirationManager) cleanupLoop(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		case <-m.ticker.C:
			if err := m.RunCleanup(ctx); err != nil {
				logger.GetLogger().Error("automatic cleanup failed", zap.Error(err))
			}
		}
	}
}

// RunCleanup performs a manual cleanup operation
func (m *ExpirationManager) RunCleanup(ctx context.Context) error {
	// Prevent concurrent cleanups
	if !m.cleanupMutex.TryLock() {
		return ErrCleanupInProgress
	}
	defer m.cleanupMutex.Unlock()

	stats := NewCleanupStats()
	defer func() {
		stats.Complete()
		m.saveStats(stats)
	}()

	logger.GetLogger().Info("starting cleanup operation",
		zap.Duration("max_age", m.policy.MaxAge),
		zap.Int("max_count", m.policy.MaxCount),
		zap.Int("batch_size", m.policy.BatchSize),
	)

	// Determine which states to clean
	statesToClean := m.getStatesToClean()

	// Perform cleanup for each state
	for _, sagaState := range statesToClean {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		deleted, scanned, err := m.cleanupState(ctx, sagaState, stats)
		stats.ScannedCount += scanned
		stats.DeletedCount += deleted

		if err != nil {
			stats.ErrorCount++
			stats.LastError = err
			logger.GetLogger().Error("cleanup failed for state",
				zap.String("state", sagaState.String()),
				zap.Error(err),
			)
			// Continue with other states
			continue
		}

		if deleted > 0 {
			logger.GetLogger().Info("cleanup completed for state",
				zap.String("state", sagaState.String()),
				zap.Int("deleted", deleted),
				zap.Int("scanned", scanned),
			)
		}
	}

	m.lastCleanup = time.Now()

	logger.GetLogger().Info("cleanup operation completed",
		zap.Duration("duration", stats.Duration),
		zap.Int("scanned", stats.ScannedCount),
		zap.Int("deleted", stats.DeletedCount),
		zap.Int("errors", stats.ErrorCount),
	)

	return nil
}

// cleanupState performs cleanup for a specific Saga state
func (m *ExpirationManager) cleanupState(ctx context.Context, sagaState saga.SagaState, stats *CleanupStats) (int, int, error) {
	stateIndexKey := m.storage.getStateIndexKey(sagaState.String())

	// Get all Sagas in this state
	sagaIDs, err := m.storage.client.SMembers(ctx, stateIndexKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("failed to get sagas for state %s: %w", sagaState, err)
	}

	scannedCount := len(sagaIDs)
	deletedCount := 0

	if len(sagaIDs) == 0 {
		return 0, 0, nil
	}

	// Prepare candidates for cleanup
	var candidates []cleanupCandidate

	for _, sagaID := range sagaIDs {
		if ctx.Err() != nil {
			return deletedCount, scannedCount, ctx.Err()
		}

		instance, err := m.storage.GetSaga(ctx, sagaID)
		if err != nil {
			if errors.Is(err, state.ErrSagaNotFound) {
				// Already deleted, skip
				continue
			}
			return deletedCount, scannedCount, err
		}

		candidates = append(candidates, cleanupCandidate{
			ID:        sagaID,
			UpdatedAt: instance.GetUpdatedAt(),
			State:     sagaState,
		})
	}

	// Apply cleanup policies
	toDelete := m.applyCleanupPolicies(candidates)

	// Delete in batches
	for i := 0; i < len(toDelete); i += m.policy.BatchSize {
		if ctx.Err() != nil {
			return deletedCount, scannedCount, ctx.Err()
		}

		end := i + m.policy.BatchSize
		if end > len(toDelete) {
			end = len(toDelete)
		}

		batch := toDelete[i:end]
		count, err := m.deleteBatch(ctx, batch)
		deletedCount += count

		if err != nil {
			return deletedCount, scannedCount, err
		}

		// Track by state
		stats.StateBreakdown[sagaState.String()] += count
	}

	return deletedCount, scannedCount, nil
}

// cleanupCandidate represents a Saga that is a candidate for cleanup
type cleanupCandidate struct {
	ID        string
	UpdatedAt time.Time
	State     saga.SagaState
}

// applyCleanupPolicies applies cleanup policies and returns Sagas to delete
func (m *ExpirationManager) applyCleanupPolicies(candidates []cleanupCandidate) []cleanupCandidate {
	var toDelete []cleanupCandidate

	// Apply age-based policy
	if m.policy.MaxAge > 0 {
		cutoffTime := time.Now().Add(-m.policy.MaxAge)
		for _, candidate := range candidates {
			if candidate.UpdatedAt.Before(cutoffTime) {
				toDelete = append(toDelete, candidate)
			}
		}
	}

	// Apply count-based policy
	if m.policy.MaxCount > 0 && len(candidates) > m.policy.MaxCount {
		// Sort by update time (oldest first)
		sortedCandidates := make([]cleanupCandidate, len(candidates))
		copy(sortedCandidates, candidates)

		// Simple bubble sort by UpdatedAt (good enough for this use case)
		for i := 0; i < len(sortedCandidates); i++ {
			for j := i + 1; j < len(sortedCandidates); j++ {
				if sortedCandidates[i].UpdatedAt.After(sortedCandidates[j].UpdatedAt) {
					sortedCandidates[i], sortedCandidates[j] = sortedCandidates[j], sortedCandidates[i]
				}
			}
		}

		// Keep only MaxCount newest, delete the rest
		excessCount := len(sortedCandidates) - m.policy.MaxCount
		if excessCount > 0 {
			for i := 0; i < excessCount; i++ {
				// Only add if not already in toDelete
				alreadyAdded := false
				for _, existing := range toDelete {
					if existing.ID == sortedCandidates[i].ID {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					toDelete = append(toDelete, sortedCandidates[i])
				}
			}
		}
	}

	return toDelete
}

// deleteBatch deletes a batch of Sagas
func (m *ExpirationManager) deleteBatch(ctx context.Context, batch []cleanupCandidate) (int, error) {
	deletedCount := 0

	for _, candidate := range batch {
		if ctx.Err() != nil {
			return deletedCount, ctx.Err()
		}

		if err := m.storage.DeleteSaga(ctx, candidate.ID); err != nil {
			if errors.Is(err, state.ErrSagaNotFound) {
				// Already deleted, continue
				continue
			}
			return deletedCount, fmt.Errorf("failed to delete saga %s: %w", candidate.ID, err)
		}

		deletedCount++
	}

	return deletedCount, nil
}

// getStatesToClean returns the list of states to clean
func (m *ExpirationManager) getStatesToClean() []saga.SagaState {
	if len(m.policy.StatesToClean) > 0 {
		return m.policy.StatesToClean
	}

	// Return all terminal states
	return []saga.SagaState{
		saga.StateCompleted,
		saga.StateCompensated,
		saga.StateFailed,
		saga.StateCancelled,
		saga.StateTimedOut,
	}
}

// saveStats saves the cleanup statistics
func (m *ExpirationManager) saveStats(stats *CleanupStats) {
	m.statsLock.Lock()
	defer m.statsLock.Unlock()
	m.lastStats = stats
}

// GetLastStats returns the statistics from the last cleanup operation
func (m *ExpirationManager) GetLastStats() *CleanupStats {
	m.statsLock.RLock()
	defer m.statsLock.RUnlock()
	return m.lastStats
}

// GetLastCleanupTime returns the time of the last cleanup operation
func (m *ExpirationManager) GetLastCleanupTime() time.Time {
	m.cleanupMutex.Lock()
	defer m.cleanupMutex.Unlock()
	return m.lastCleanup
}

// SetTTL sets the TTL for a specific Saga key
func (r *RedisStateStorage) SetTTL(ctx context.Context, sagaID string, ttl time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if ttl < 0 {
		return errors.New("TTL must be >= 0")
	}

	sagaKey := r.getSagaKey(sagaID)

	// Set TTL on the main saga key
	if err := r.client.Expire(ctx, sagaKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL for saga %s: %w", sagaID, err)
	}

	// Also set TTL on associated step keys
	stepsSetKey := r.getStepsSetKey(sagaID)
	stepIDs, err := r.client.SMembers(ctx, stepsSetKey).Result()
	if err != nil && err != redis.Nil {
		// Log but don't fail
		logger.GetLogger().Warn("failed to get step IDs for TTL update",
			zap.String("saga_id", sagaID),
			zap.Error(err))
		return nil
	}

	// Set TTL on each step key
	for _, stepID := range stepIDs {
		stepKey := r.getStepKey(sagaID, stepID)
		if err := r.client.Expire(ctx, stepKey, ttl).Err(); err != nil {
			// Log but don't fail
			logger.GetLogger().Warn("failed to set TTL for step",
				zap.String("saga_id", sagaID),
				zap.String("step_id", stepID),
				zap.Error(err))
		}
	}

	// Set TTL on steps set key
	if err := r.client.Expire(ctx, stepsSetKey, ttl).Err(); err != nil {
		logger.GetLogger().Warn("failed to set TTL for steps set",
			zap.String("saga_id", sagaID),
			zap.Error(err))
	}

	return nil
}

// GetTTL returns the remaining TTL for a Saga key
func (r *RedisStateStorage) GetTTL(ctx context.Context, sagaID string) (time.Duration, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	if r.closed {
		return 0, state.ErrStorageClosed
	}

	if sagaID == "" {
		return 0, state.ErrInvalidSagaID
	}

	sagaKey := r.getSagaKey(sagaID)

	ttl, err := r.client.TTL(ctx, sagaKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for saga %s: %w", sagaID, err)
	}

	// Redis returns -2 if key doesn't exist, -1 if no expiration
	if ttl < 0 {
		return ttl, nil
	}

	return ttl, nil
}

// RefreshTTL refreshes the TTL for a Saga to the configured default TTL
func (r *RedisStateStorage) RefreshTTL(ctx context.Context, sagaID string) error {
	return r.SetTTL(ctx, sagaID, r.config.TTL)
}

// CleanupExpiredByTTL removes Sagas that have naturally expired in Redis.
// This is primarily for cleanup of index entries, as Redis will automatically
// delete the keys when they expire.
func (r *RedisStateStorage) CleanupExpiredByTTL(ctx context.Context) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	if r.closed {
		return 0, state.ErrStorageClosed
	}

	cleanedCount := 0

	// Get all terminal states
	terminalStates := []saga.SagaState{
		saga.StateCompleted,
		saga.StateCompensated,
		saga.StateFailed,
		saga.StateCancelled,
		saga.StateTimedOut,
	}

	for _, sagaState := range terminalStates {
		stateIndexKey := r.getStateIndexKey(sagaState.String())
		sagaIDs, err := r.client.SMembers(ctx, stateIndexKey).Result()
		if err != nil && err != redis.Nil {
			return cleanedCount, fmt.Errorf("failed to get sagas for state %s: %w", sagaState, err)
		}

		// Check each saga ID to see if the key still exists
		for _, sagaID := range sagaIDs {
			sagaKey := r.getSagaKey(sagaID)
			exists, err := r.client.Exists(ctx, sagaKey).Result()
			if err != nil {
				continue
			}

			// If key doesn't exist (expired), remove from index
			if exists == 0 {
				// Remove from state index
				if err := r.client.SRem(ctx, stateIndexKey, sagaID).Err(); err != nil {
					logger.GetLogger().Warn("failed to remove expired saga from index",
						zap.String("saga_id", sagaID),
						zap.String("state", sagaState.String()),
						zap.Error(err),
					)
					continue
				}

				// Remove from timeout index
				timeoutKey := r.getTimeoutKey()
				r.client.ZRem(ctx, timeoutKey, sagaID)

				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		logger.GetLogger().Info("cleaned up expired saga index entries",
			zap.Int("count", cleanedCount))
	}

	return cleanedCount, nil
}
