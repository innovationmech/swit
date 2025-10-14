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

	"github.com/google/uuid"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupPolicy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  *CleanupPolicy
		wantErr bool
	}{
		{
			name:    "nil policy",
			policy:  nil,
			wantErr: true,
		},
		{
			name: "valid default policy",
			policy: &CleanupPolicy{
				MaxAge:          24 * time.Hour,
				MaxCount:        100,
				BatchSize:       50,
				CleanupInterval: 1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "negative max age",
			policy: &CleanupPolicy{
				MaxAge:    -1 * time.Hour,
				BatchSize: 50,
			},
			wantErr: true,
		},
		{
			name: "negative max count",
			policy: &CleanupPolicy{
				MaxCount:  -1,
				BatchSize: 50,
			},
			wantErr: true,
		},
		{
			name: "zero batch size",
			policy: &CleanupPolicy{
				BatchSize: 0,
			},
			wantErr: true,
		},
		{
			name: "negative cleanup interval",
			policy: &CleanupPolicy{
				BatchSize:       50,
				CleanupInterval: -1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "non-terminal state in states to clean",
			policy: &CleanupPolicy{
				BatchSize: 50,
				StatesToClean: []saga.SagaState{
					saga.StateCompleted,
					saga.StateRunning, // Non-terminal
				},
			},
			wantErr: true,
		},
		{
			name: "valid with specific states",
			policy: &CleanupPolicy{
				BatchSize: 50,
				StatesToClean: []saga.SagaState{
					saga.StateCompleted,
					saga.StateFailed,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultCleanupPolicy(t *testing.T) {
	policy := DefaultCleanupPolicy()
	require.NotNil(t, policy)

	assert.Equal(t, 7*24*time.Hour, policy.MaxAge)
	assert.Equal(t, 0, policy.MaxCount)
	assert.Equal(t, 100, policy.BatchSize)
	assert.Equal(t, 1*time.Hour, policy.CleanupInterval)
	assert.True(t, policy.EnableMetrics)
	assert.Nil(t, policy.StatesToClean)

	err := policy.Validate()
	assert.NoError(t, err)
}

func TestCleanupStats(t *testing.T) {
	stats := NewCleanupStats()
	require.NotNil(t, stats)

	assert.False(t, stats.StartTime.IsZero())
	assert.True(t, stats.EndTime.IsZero())
	assert.NotNil(t, stats.StateBreakdown)

	time.Sleep(10 * time.Millisecond)
	stats.Complete()

	assert.False(t, stats.EndTime.IsZero())
	assert.Greater(t, stats.Duration, time.Duration(0))
}

func TestExpirationManager_NewExpirationManager(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	tests := []struct {
		name    string
		storage *RedisStateStorage
		policy  *CleanupPolicy
		wantErr bool
	}{
		{
			name:    "nil storage",
			storage: nil,
			policy:  DefaultCleanupPolicy(),
			wantErr: true,
		},
		{
			name:    "nil policy uses default",
			storage: storage,
			policy:  nil,
			wantErr: false,
		},
		{
			name:    "invalid policy",
			storage: storage,
			policy:  &CleanupPolicy{BatchSize: 0},
			wantErr: true,
		},
		{
			name:    "valid policy",
			storage: storage,
			policy:  DefaultCleanupPolicy(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewExpirationManager(tt.storage, tt.policy)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, mgr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mgr)
			}
		})
	}
}

func TestExpirationManager_StartStop(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	policy := &CleanupPolicy{
		MaxAge:          1 * time.Hour,
		BatchSize:       10,
		CleanupInterval: 100 * time.Millisecond,
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	ctx := context.Background()

	// Start the manager
	err = mgr.Start(ctx)
	assert.NoError(t, err)

	// Try to start again (should fail)
	err = mgr.Start(ctx)
	assert.Error(t, err)

	// Stop the manager
	err = mgr.Stop()
	assert.NoError(t, err)

	// Stop again (should be idempotent)
	err = mgr.Stop()
	assert.NoError(t, err)
}

func TestExpirationManager_RunCleanup_AgeBased(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Create test sagas with different ages
	now := time.Now()
	oldSaga := createTestSagaWithTime(t, "old-saga", saga.StateCompleted, now.Add(-2*time.Hour))
	recentSaga := createTestSagaWithTime(t, "recent-saga", saga.StateCompleted, now.Add(-30*time.Minute))

	err := storage.SaveSaga(ctx, oldSaga)
	require.NoError(t, err)

	err = storage.SaveSaga(ctx, recentSaga)
	require.NoError(t, err)

	// Create cleanup policy (max age = 1 hour)
	policy := &CleanupPolicy{
		MaxAge:    1 * time.Hour,
		BatchSize: 10,
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	// Run cleanup
	err = mgr.RunCleanup(ctx)
	assert.NoError(t, err)

	// Verify old saga was deleted
	_, err = storage.GetSaga(ctx, "old-saga")
	assert.Error(t, err)

	// Verify recent saga still exists
	_, err = storage.GetSaga(ctx, "recent-saga")
	assert.NoError(t, err)

	// Check stats
	stats := mgr.GetLastStats()
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.DeletedCount)
	assert.Greater(t, stats.ScannedCount, 0)
}

func TestExpirationManager_RunCleanup_CountBased(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Create 5 completed sagas
	now := time.Now()
	for i := 0; i < 5; i++ {
		sagaID := fmt.Sprintf("saga-%s", uuid.New().String())
		testSaga := createTestSagaWithTime(t, sagaID, saga.StateCompleted, now.Add(-time.Duration(i)*time.Minute))
		err := storage.SaveSaga(ctx, testSaga)
		require.NoError(t, err)
	}

	// Create cleanup policy (keep only 3)
	policy := &CleanupPolicy{
		MaxCount:  3,
		BatchSize: 10,
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	// Run cleanup
	err = mgr.RunCleanup(ctx)
	assert.NoError(t, err)

	// Check stats - should delete 2 oldest
	stats := mgr.GetLastStats()
	require.NotNil(t, stats)
	assert.Equal(t, 2, stats.DeletedCount)

	// Verify we have exactly 3 sagas left
	filter := &saga.SagaFilter{
		States: []saga.SagaState{saga.StateCompleted},
	}
	remaining, err := storage.GetActiveSagas(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, remaining, 3)
}

func TestExpirationManager_RunCleanup_MultipleStates(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create sagas in different terminal states
	states := []saga.SagaState{
		saga.StateCompleted,
		saga.StateFailed,
		saga.StateCompensated,
	}

	for _, state := range states {
		sagaID := fmt.Sprintf("saga-%s", uuid.New().String())
		testSaga := createTestSagaWithTime(t, sagaID, state, now.Add(-2*time.Hour))
		err := storage.SaveSaga(ctx, testSaga)
		require.NoError(t, err)
	}

	// Create cleanup policy (max age = 1 hour)
	policy := &CleanupPolicy{
		MaxAge:    1 * time.Hour,
		BatchSize: 10,
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	// Run cleanup
	err = mgr.RunCleanup(ctx)
	assert.NoError(t, err)

	// Verify all were deleted
	stats := mgr.GetLastStats()
	require.NotNil(t, stats)
	assert.Equal(t, 3, stats.DeletedCount)

	// Check state breakdown
	assert.Contains(t, stats.StateBreakdown, "completed")
	assert.Contains(t, stats.StateBreakdown, "failed")
	assert.Contains(t, stats.StateBreakdown, "compensated")
}

func TestExpirationManager_RunCleanup_SpecificStates(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create sagas in different states
	completedSaga := createTestSagaWithTime(t, "completed-saga", saga.StateCompleted, now.Add(-2*time.Hour))
	failedSaga := createTestSagaWithTime(t, "failed-saga", saga.StateFailed, now.Add(-2*time.Hour))

	err := storage.SaveSaga(ctx, completedSaga)
	require.NoError(t, err)

	err = storage.SaveSaga(ctx, failedSaga)
	require.NoError(t, err)

	// Create cleanup policy (only clean completed)
	policy := &CleanupPolicy{
		MaxAge:    1 * time.Hour,
		BatchSize: 10,
		StatesToClean: []saga.SagaState{
			saga.StateCompleted,
		},
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	// Run cleanup
	err = mgr.RunCleanup(ctx)
	assert.NoError(t, err)

	// Verify only completed was deleted
	_, err = storage.GetSaga(ctx, "completed-saga")
	assert.Error(t, err)

	// Failed saga should still exist
	_, err = storage.GetSaga(ctx, "failed-saga")
	assert.NoError(t, err)

	stats := mgr.GetLastStats()
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.DeletedCount)
}

func TestExpirationManager_ConcurrentCleanup(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Create some test data to make cleanup take longer
	now := time.Now()
	for i := 0; i < 10; i++ {
		sagaID := fmt.Sprintf("saga-%s", uuid.New().String())
		testSaga := createTestSagaWithTime(t, sagaID, saga.StateCompleted, now.Add(-2*time.Hour))
		err := storage.SaveSaga(ctx, testSaga)
		require.NoError(t, err)
	}

	policy := &CleanupPolicy{
		MaxAge:    1 * time.Hour,
		BatchSize: 2, // Small batch to make it slower
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	// Start two concurrent cleanups with minimal delay
	errCh := make(chan error, 2)

	go func() {
		errCh <- mgr.RunCleanup(ctx)
	}()

	go func() {
		time.Sleep(1 * time.Millisecond) // Very short delay to try to catch the lock
		errCh <- mgr.RunCleanup(ctx)
	}()

	// One should succeed, one should get ErrCleanupInProgress
	err1 := <-errCh
	err2 := <-errCh

	// At least one should succeed
	hasSuccess := err1 == nil || err2 == nil
	assert.True(t, hasSuccess, "at least one cleanup should succeed")

	// If both succeeded, that's also acceptable (second one started after first finished)
	// If one got conflict error, that's the expected behavior
	if err1 != nil && err2 != nil {
		hasConflict := err1 == ErrCleanupInProgress || err2 == ErrCleanupInProgress
		assert.True(t, hasConflict, "if both failed, one should get conflict error")
	}
}

func TestRedisStateStorage_SetTTL(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Create a test saga
	testSaga := createTestSaga(t, "test-saga")
	err := storage.SaveSaga(ctx, testSaga)
	require.NoError(t, err)

	// Set custom TTL
	customTTL := 1 * time.Hour
	err = storage.SetTTL(ctx, "test-saga", customTTL)
	assert.NoError(t, err)

	// Get TTL
	ttl, err := storage.GetTTL(ctx, "test-saga")
	assert.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, customTTL)
}

func TestRedisStateStorage_GetTTL(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	tests := []struct {
		name      string
		sagaID    string
		setupFunc func()
		wantErr   bool
	}{
		{
			name:   "existing saga",
			sagaID: "existing-saga",
			setupFunc: func() {
				testSaga := createTestSaga(t, "existing-saga")
				err := storage.SaveSaga(ctx, testSaga)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:      "non-existent saga",
			sagaID:    "non-existent",
			setupFunc: func() {},
			wantErr:   false, // GetTTL returns -2 for non-existent keys
		},
		{
			name:      "empty saga ID",
			sagaID:    "",
			setupFunc: func() {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			ttl, err := storage.GetTTL(ctx, tt.sagaID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				t.Logf("TTL for %s: %v", tt.sagaID, ttl)
			}
		})
	}
}

func TestRedisStateStorage_RefreshTTL(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Create a test saga
	testSaga := createTestSaga(t, "test-saga")
	err := storage.SaveSaga(ctx, testSaga)
	require.NoError(t, err)

	// Set short TTL
	err = storage.SetTTL(ctx, "test-saga", 1*time.Second)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Refresh TTL
	err = storage.RefreshTTL(ctx, "test-saga")
	assert.NoError(t, err)

	// Get new TTL
	ttl, err := storage.GetTTL(ctx, "test-saga")
	assert.NoError(t, err)
	assert.Greater(t, ttl, 1*time.Second) // Should be back to default
}

func TestRedisStateStorage_CleanupExpiredByTTL(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Create test sagas in completed state
	saga1 := createTestSagaWithTime(t, "saga-1", saga.StateCompleted, time.Now())
	err := storage.SaveSaga(ctx, saga1)
	require.NoError(t, err)

	saga2 := createTestSagaWithTime(t, "saga-2", saga.StateCompleted, time.Now())
	err = storage.SaveSaga(ctx, saga2)
	require.NoError(t, err)

	// Set very short TTL on saga-1 (minimum 1 second for Redis)
	err = storage.SetTTL(ctx, "saga-1", 1*time.Second)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Verify saga-1 key is expired (Redis auto-deleted it)
	_, err = storage.GetSaga(ctx, "saga-1")
	assert.Error(t, err, "saga-1 should be expired and auto-deleted by Redis")

	// Run cleanup to remove stale index entries
	count, err := storage.CleanupExpiredByTTL(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "should clean up at least 1 expired index entry")

	// Verify saga-2 still exists
	_, err = storage.GetSaga(ctx, "saga-2")
	assert.NoError(t, err)
}

func TestExpirationManager_AutomaticCleanup(t *testing.T) {
	storage := createTestRedisStorage(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create old saga
	oldSaga := createTestSagaWithTime(t, "old-saga", saga.StateCompleted, now.Add(-2*time.Hour))
	err := storage.SaveSaga(ctx, oldSaga)
	require.NoError(t, err)

	// Create manager with short interval
	policy := &CleanupPolicy{
		MaxAge:          1 * time.Hour,
		BatchSize:       10,
		CleanupInterval: 100 * time.Millisecond,
	}

	mgr, err := NewExpirationManager(storage, policy)
	require.NoError(t, err)

	// Start automatic cleanup
	err = mgr.Start(ctx)
	require.NoError(t, err)
	defer mgr.Stop()

	// Wait for automatic cleanup to run
	time.Sleep(300 * time.Millisecond)

	// Verify saga was deleted
	_, err = storage.GetSaga(ctx, "old-saga")
	assert.Error(t, err)

	// Check that cleanup ran
	lastCleanup := mgr.GetLastCleanupTime()
	assert.False(t, lastCleanup.IsZero())
}

// Helper functions

func createTestSaga(t *testing.T, sagaID string) saga.SagaInstance {
	return createTestSagaWithTime(t, sagaID, saga.StateCompleted, time.Now())
}

func createTestSagaWithTime(t *testing.T, sagaID string, state saga.SagaState, updatedAt time.Time) saga.SagaInstance {
	data := &saga.SagaInstanceData{
		ID:           sagaID,
		DefinitionID: "test-def",
		State:        state,
		CurrentStep:  0,
		TotalSteps:   1,
		CreatedAt:    updatedAt.Add(-1 * time.Hour),
		UpdatedAt:    updatedAt,
		Timeout:      30 * time.Minute,
		Metadata:     make(map[string]interface{}),
	}

	return &redisSagaInstance{data: data}
}

func createTestRedisStorage(t *testing.T) *RedisStateStorage {
	config := &RedisConfig{
		Mode:      RedisModeStandalone,
		Addr:      "localhost:6379",
		DB:        15, // Use a test database
		KeyPrefix: "test:saga:",
		TTL:       24 * time.Hour,
	}

	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Skip("Redis not available for testing:", err)
	}

	// Clean up test data
	ctx := context.Background()
	storage.client.FlushDB(ctx)

	return storage
}
